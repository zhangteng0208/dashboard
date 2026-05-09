# Project Management Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有项目管理基础上扩展：标签系统、别名/简介编辑、Git 远程/分支/diff 查看、NPM/Maven Scripts 展示与执行、README 解析。

**Architecture:** 数据库新增字段和表，后端扩展 scan 逻辑新增独立 API，前端 ProjectDetail 添加 Tab 页。

**Tech Stack:** Go + SQLite, Vue3 + TypeScript + Element Plus

---

## File Structure

```
go_backend/main.go                  # 修改：数据库 Schema + 所有新 API handlers
src/components/ProjectDetail.vue    # 修改：新增 Tabs（标签/Git/Scripts/README）
src/components/TagSelector.vue      # 新建：标签选择器组件
```

---

## Task 1: 后端 - 数据库 Schema 变更

**Files:**
- Modify: `go_backend/main.go:120-180` (initDB 函数附近)

- [ ] **Step 1: 添加 projects 表字段变更**

在 `initDB()` 中 projects 表创建之后添加：

```go
// projects 表新增 alias 和 description 字段
_, err = db.Exec("ALTER TABLE projects ADD COLUMN alias TEXT")
if err != nil && err.Error() != "UNIQUE constraint failed: projects.alias" {
    log.Printf("projects alias column may already exist: %v", err)
}

_, err = db.Exec("ALTER TABLE projects ADD COLUMN description TEXT")
if err != nil && err.Error() != "UNIQUE constraint failed: projects.description" {
    log.Printf("projects description column may already exist: %v", err)
}
```

- [ ] **Step 2: 创建 project_tags 表**

```go
_, err = db.Exec(`
  CREATE TABLE IF NOT EXISTS project_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#0a84ff'
  )
`)
if err != nil {
    log.Printf("Failed to create project_tags table: %v", err)
}
```

- [ ] **Step 3: 创建 project_tag_relations 表**

```go
_, err = db.Exec(`
  CREATE TABLE IF NOT EXISTS project_tag_relations (
    project_id INTEGER,
    tag_id INTEGER,
    PRIMARY KEY (project_id, tag_id),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES project_tags(id) ON DELETE CASCADE
  )
`)
if err != nil {
    log.Printf("Failed to create project_tag_relations table: %v", err)
}
```

- [ ] **Step 4: 验证**

启动后端后：
```bash
curl http://localhost:18788/api/projects
# 应正常返回（字段兼容）
```

---

## Task 2: 后端 - GET /api/tags 和 POST /api/tags

**Files:**
- Modify: `go_backend/main.go` (在 projectsHandler 之后添加)

- [ ] **Step 1: 添加 tags handler**

在 `projectsHandler` 函数后面添加：

```go
func tagsHandler(w http.ResponseWriter, r *http.Request) {
  // GET: 列出所有标签
  if r.Method == "GET" {
    rows, err := DB.Query("SELECT id, name, color FROM project_tags ORDER BY name")
    if err != nil {
      http.Error(w, err.Error(), 500)
      return
    }
    defer rows.Close()

    var results []map[string]interface{}
    for rows.Next() {
      var id int
      var name, color string
      rows.Scan(&id, &name, &color)
      results = append(results, map[string]interface{}{"id": id, "name": name, "color": color})
    }
    if results == nil {
      results = []map[string]interface{}{}
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"tags": results})
    return
  }

  // POST: 创建标签
  if r.Method == "POST" {
    var req struct {
      Name  string `json:"name"`
      Color string `json:"color"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
      http.Error(w, `{"error":"无效的请求体"}`, 400)
      return
    }
    if req.Name == "" {
      http.Error(w, `{"error":"标签名不能为空"}`, 400)
      return
    }
    if req.Color == "" {
      req.Color = "#0a84ff"
    }

    result, err := DB.Exec(
      "INSERT INTO project_tags (name, color) VALUES (?, ?)",
      req.Name, req.Color,
    )
    if err != nil {
      if strings.Contains(err.Error(), "UNIQUE") {
        http.Error(w, `{"error":"标签名已存在"}`, 409)
        return
      }
      http.Error(w, err.Error(), 500)
      return
    }

    id, _ := result.LastInsertId()
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
      "id": id, "name": req.Name, "color": req.Color,
    })
  }
}
```

- [ ] **Step 2: 注册路由**

找到路由注册位置，添加：
```go
http.HandleFunc("/api/tags", tagsHandler)
```

- [ ] **Step 3: 测试**

```bash
curl http://localhost:18788/api/tags
# 应返回: {"tags":[]}

curl -X POST http://localhost:18788/api/tags \
  -H "Content-Type: application/json" \
  -d '{"name":"java","color":"#ff6b6b"}'
# 应返回新标签
```

---

## Task 3: 后端 - PUT /api/projects/:id/tags

**Files:**
- Modify: `go_backend/main.go` (在 projectsHandler 中 /api/projects/ DELETE 逻辑后添加)

- [ ] **Step 1: 添加更新项目标签的 handler**

在 projectsHandler 的 `/api/projects/` 路由分支中添加 PUT 分支：

```go
// PUT: 更新项目标签
if r.Method == "PUT" && strings.Contains(r.URL.Path, "/tags") {
  // 提取 id: /api/projects/1/tags -> id=1
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/tags"), "/")
  id := pathParts[len(pathParts)-1]

  var req struct {
    TagIDs []int `json:"tagIds"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, `{"error":"无效的请求体"}`, 400)
    return
  }

  // 删除旧关联
  DB.Exec("DELETE FROM project_tag_relations WHERE project_id = ?", id)

  // 插入新关联
  for _, tagID := range req.TagIDs {
    DB.Exec(
      "INSERT OR IGNORE INTO project_tag_relations (project_id, tag_id) VALUES (?, ?)",
      id, tagID,
    )
  }

  w.WriteHeader(204)
  return
}
```

- [ ] **Step 2: 测试**

```bash
# 先获取项目列表中的某个 id，假设是 1
curl -X PUT http://localhost:18788/api/projects/1/tags \
  -H "Content-Type: application/json" \
  -d '{"tagIds":[1]}'
# 应返回 204
```

---

## Task 4: 后端 - GET /api/projects/:id/tags (获取项目标签)

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加获取项目标签的 handler**

在 projectsHandler 的 `/api/projects/` 路由分支中添加 GET /tags 分支（与 PUT 同一个代码块）：

```go
// GET: 获取项目标签
if r.Method == "GET" && strings.Contains(r.URL.Path, "/tags") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/tags"), "/")
  id := pathParts[len(pathParts)-1]

  rows, err := DB.Query(`
    SELECT t.id, t.name, t.color
    FROM project_tags t
    INNER JOIN project_tag_relations r ON t.id = r.tag_id
    WHERE r.project_id = ?
  `, id)
  if err != nil {
    http.Error(w, err.Error(), 500)
    return
  }
  defer rows.Close()

  var tags []map[string]interface{}
  for rows.Next() {
    var tagID int
    var name, color string
    rows.Scan(&tagID, &name, &color)
    tags = append(tags, map[string]interface{}{"id": tagID, "name": name, "color": color})
  }
  if tags == nil {
    tags = []map[string]interface{}{}
  }
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]interface{}{"tags": tags})
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl http://localhost:18788/api/projects/1/tags
# 应返回: {"tags":[]}
```

---

## Task 5: 后端 - PATCH /api/projects/:id (更新别名/简介)

**Files:**
- Modify: `go_backend/main.go` (在 projectsHandler 中)

- [ ] **Step 1: 在 projectsHandler 的 /api/projects/ 分支中添加 PATCH 处理**

在 DELETE 分支之后、POST 分支之前添加：

```go
// PATCH: 更新项目别名/简介 (全部字段必填)
if r.Method == "PATCH" {
  pathParts := strings.Split(r.URL.Path, "/")
  id := pathParts[len(pathParts)-1]

  var req struct {
    Name        string `json:"name"`
    Alias       string `json:"alias"`
    Description string `json:"description"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, `{"error":"无效的请求体"}`, 400)
    return
  }

  // 全部字段必填校验
  if req.Name == "" || req.Alias == "" || req.Description == "" {
    http.Error(w, `{"error":"name, alias, description 全部字段必填"}`, 400)
    return
  }

  _, err := DB.Exec(
    "UPDATE projects SET name=?, alias=?, description=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
    req.Name, req.Alias, req.Description, id,
  )
  if err != nil {
    http.Error(w, err.Error(), 500)
    return
  }

  w.WriteHeader(204)
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl -X PATCH http://localhost:18788/api/projects/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"dashboard","alias":"主面板","description":"个人仪表盘项目"}'
# 应返回 204
```

---

## Task 6: 后端 - scanGitRemote() 函数

**Files:**
- Modify: `go_backend/main.go` (在 scanGitInfo 函数后添加)

- [ ] **Step 1: 添加 scanGitRemote 函数**

```go
// scanGitRemote 获取 Git 远程地址
func scanGitRemote(projPath string) map[string]interface{} {
  result := map[string]interface{}{
    "remoteUrl": "", "remoteType": "",
  }

  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  // git remote -v
  cmd := exec.CommandContext(ctx, "git", "-C", projPath, "remote", "-v")
  out, err := cmd.Output()
  if err != nil {
    return result
  }

  lines := strings.Split(strings.TrimSpace(string(out)), "\n")
  for _, line := range lines {
    parts := strings.Fields(line)
    if len(parts) >= 2 {
      result["remoteUrl"] = parts[1]
      // 判断类型
      url := parts[1]
      if strings.HasPrefix(url, "git@") {
        result["remoteType"] = "ssh"
      } else if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
        result["remoteType"] = "https"
      } else if strings.Contains(url, "github.com") {
        result["remoteType"] = "github"
      }
      break
    }
  }

  return result
}
```

- [ ] **Step 2: 测试**

```bash
# 在一个 git 项目目录下测试
cd /tmp && git clone https://github.com/torvalds/linux.git && \
curl -X POST http://localhost:18788/api/projects/1/remote
```

---

## Task 7: 后端 - GET /api/projects/:id/remote

**Files:**
- Modify: `go_backend/main.go` (在 projectsHandler 中添加)

- [ ] **Step 1: 在 /api/projects/ 路由中添加 remote handler**

在 PATCH 分支之后添加：

```go
// GET /api/projects/:id/remote - 获取 Git 远程信息
if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/remote") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/remote"), "/")
  id := pathParts[len(pathParts)-1]

  var projPath string
  err := DB.QueryRow("SELECT path FROM projects WHERE id = ?", id).Scan(&projPath)
  if err == sql.ErrNoRows {
    http.Error(w, `{"error":"项目不存在"}`, 404)
    return
  }

  remoteInfo := scanGitRemote(projPath)
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(remoteInfo)
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl http://localhost:18788/api/projects/1/remote
```

---

## Task 8: 后端 - scanGitBranches() 函数

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 scanGitBranches 函数**

在 scanGitRemote 函数后添加：

```go
// scanGitBranches 获取所有分支及最后提交信息
func scanGitBranches(projPath string) []map[string]interface{} {
  var branches []map[string]interface{}

  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()

  // git branch -a --format='%(refname:short)|%(objectname:short)|%(subject)|%(committerdate:iso)'
  cmd := exec.CommandContext(ctx, "git", "-C", projPath, "branch", "-a",
    "--format=%(refname:short)|%(objectname:short)|%(subject)|%(committerdate:iso)")
  out, err := cmd.Output()
  if err != nil {
    return branches
  }

  lines := strings.Split(strings.TrimSpace(string(out)), "\n")
  for _, line := range lines {
    parts := strings.Split(line, "|")
    if len(parts) >= 4 {
      name := parts[0]
      commit := parts[1]
      msg := parts[2]
      date := parts[3]
      if name == "" {
        name = "HEAD"
      }
      branches = append(branches, map[string]interface{}{
        "name":           name,
        "lastCommit":     commit,
        "lastCommitMsg":  msg,
        "lastCommitDate": date,
      })
    }
  }

  return branches
}
```

---

## Task 9: 后端 - GET /api/projects/:id/branches

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 branches handler**

在 remote handler 之后添加：

```go
// GET /api/projects/:id/branches - 获取所有分支
if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/branches") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/branches"), "/")
  id := pathParts[len(pathParts)-1]

  var projPath string
  err := DB.QueryRow("SELECT path FROM projects WHERE id = ?", id).Scan(&projPath)
  if err == sql.ErrNoRows {
    http.Error(w, `{"error":"项目不存在"}`, 404)
    return
  }

  branches := scanGitBranches(projPath)
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]interface{}{"branches": branches})
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl http://localhost:18788/api/projects/1/branches
```

---

## Task 10: 后端 - GET /api/projects/:id/diff

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 diff handler**

在 branches handler 之后添加：

```go
// GET /api/projects/:id/diff - 获取 git diff HEAD vs working tree
if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/diff") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/diff"), "/")
  id := pathParts[len(pathParts)-1]

  var projPath string
  err := DB.QueryRow("SELECT path FROM projects WHERE id = ?", id).Scan(&projPath)
  if err == sql.ErrNoRows {
    http.Error(w, `{"error":"项目不存在"}`, 404)
    return
  }

  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  // git diff --stat HEAD
  cmd := exec.CommandContext(ctx, "git", "-C", projPath, "diff", "--stat", "HEAD")
  statOut, _ := cmd.Output()

  // git diff --name-only HEAD
  cmd = exec.CommandContext(ctx, "git", "-C", projPath, "diff", "--name-only", "HEAD")
  nameOut, _ := cmd.Output()

  var changed []map[string]interface{}
  names := strings.Split(strings.TrimSpace(string(nameOut)), "\n")
  statLines := strings.Split(strings.TrimSpace(string(statOut)), "\n")

  for i, name := range names {
    if name == "" {
      continue
    }
    additions := 0
    deletions := 0
    if i < len(statLines)-1 && i > 0 {
      // 解析 stat 行: " file | 2 +++ 1 ---"
      statLine := strings.TrimSpace(statLines[i])
      parts := strings.Split(statLine, "|")
      if len(parts) >= 2 {
        nums := strings.Fields(parts[1])
        for _, p := range nums {
          if strings.HasSuffix(p, "+") {
            n, _ := strconv.Atoi(strings.TrimSuffix(p, "+"))
            additions += n
          }
          if strings.HasSuffix(p, "-") {
            n, _ := strconv.Atoi(strings.TrimSuffix(p, "-"))
            deletions += n
          }
        }
      }
    }
    changed = append(changed, map[string]interface{}{
      "file":      name,
      "additions": additions,
      "deletions": deletions,
    })
  }

  // 统计总计
  var totalAdd, totalDel int
  for _, c := range changed {
    totalAdd += c["additions"].(int)
    totalDel += c["deletions"].(int)
  }

  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]interface{}{
    "changed": changed,
    "stats": map[string]interface{}{
      "additions": totalAdd,
      "deletions": totalDel,
    },
  })
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl http://localhost:18788/api/projects/1/diff
```

---

## Task 11: 后端 - scanProjectScripts() 函数

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 scanProjectScripts 函数**

```go
// scanProjectScripts 扫描项目中的 NPM/Maven/Gradle scripts
func scanProjectScripts(projPath string) map[string]interface{} {
  result := map[string]interface{}{
    "framework": "", "scripts": []map[string]interface{}{},
  }

  // 检查 package.json (npm)
  pkgFile := projPath + "/package.json"
  if data, err := os.ReadFile(pkgFile); err == nil {
    var pkg struct {
      Scripts map[string]string `json:"scripts"`
    }
    if json.Unmarshal(data, &pkg) == nil && pkg.Scripts != nil {
      result["framework"] = "npm"
      var scripts []map[string]interface{}
      for name, cmd := range pkg.Scripts {
        scripts = append(scripts, map[string]interface{}{
          "name":    name,
          "command": cmd,
          "type":    "npm",
        })
      }
      result["scripts"] = scripts
      return result
    }
  }

  // 检查 pom.xml (Maven)
  pomFile := projPath + "/pom.xml"
  if _, err := os.Stat(pomFile); err == nil {
    result["framework"] = "maven"
    // Maven goals 简化为固定列表
    result["scripts"] = []map[string]interface{}{
      {"name": "compile", "command": "mvn compile", "type": "maven"},
      {"name": "test", "command": "mvn test", "type": "maven"},
      {"name": "package", "command": "mvn package", "type": "maven"},
      {"name": "clean", "command": "mvn clean", "type": "maven"},
      {"name": "spring-boot:run", "command": "mvn spring-boot:run", "type": "maven"},
    }
    return result
  }

  // 检查 build.gradle (Gradle)
  gradleFile := projPath + "/build.gradle"
  if _, err := os.Stat(gradleFile); err == nil {
    result["framework"] = "gradle"
    result["scripts"] = []map[string]interface{}{
      {"name": "build", "command": "./gradlew build", "type": "gradle"},
      {"name": "test", "command": "./gradlew test", "type": "gradle"},
      {"name": "bootRun", "command": "./gradlew bootRun", "type": "gradle"},
      {"name": "clean", "command": "./gradlew clean", "type": "gradle"},
    }
    return result
  }

  // 检查 go.mod (Go)
  goModFile := projPath + "/go.mod"
  if _, err := os.Stat(goModFile); err == nil {
    result["framework"] = "go"
    result["scripts"] = []map[string]interface{}{
      {"name": "build", "command": "go build ./...", "type": "go"},
      {"name": "test", "command": "go test ./...", "type": "go"},
      {"name": "run", "command": "go run .", "type": "go"},
    }
    return result
  }

  return result
}
```

---

## Task 12: 后端 - GET /api/projects/:id/scripts

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 scripts handler (扫描)**

在 diff handler 之后添加：

```go
// GET /api/projects/:id/scripts - 扫描项目 scripts
if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/scripts") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/scripts"), "/")
  id := pathParts[len(pathParts)-1]

  var projPath string
  err := DB.QueryRow("SELECT path FROM projects WHERE id = ?", id).Scan(&projPath)
  if err == sql.ErrNoRows {
    http.Error(w, `{"error":"项目不存在"}`, 404)
    return
  }

  scriptsInfo := scanProjectScripts(projPath)
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(scriptsInfo)
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl http://localhost:18788/api/projects/1/scripts
```

---

## Task 13: 后端 - POST /api/projects/:id/scripts/exec

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加脚本执行 handler**

在 scripts handler 之后添加：

```go
// 允许执行的命令白名单
var allowedCommands = map[string]bool{
  "npm":          true,
  "mvn":          true,
  "gradle":       true,
  "./gradlew":    true,
  "go":           true,
  "git":          true,
}

func isCommandAllowed(cmd string) bool {
  parts := strings.Fields(cmd)
  if len(parts) == 0 {
    return false
  }
  base := parts[0]
  // 允许 npm run xxx 和 mvn xxx 格式
  if base == "npm" && len(parts) >= 2 && parts[1] == "run" {
    return true
  }
  if base == "mvn" || base == "gradle" || base == "./gradlew" || base == "go" || base == "git" {
    return true
  }
  return false
}

// POST /api/projects/:id/scripts/exec - 执行脚本
if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/scripts/exec") {
  pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/scripts/exec"), "/")
  id := pathParts[len(pathParts)-1]

  var projPath string
  err := DB.QueryRow("SELECT path FROM projects WHERE id = ?", id).Scan(&projPath)
  if err == sql.ErrNoRows {
    http.Error(w, `{"error":"项目不存在"}`, 404)
    return
  }

  var req struct {
    Command string `json:"command"`
  }
  if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, `{"error":"无效的请求体"}`, 400)
    return
  }

  if !isCommandAllowed(req.Command) {
    http.Error(w, `{"error":"命令不在白名单中"}`, 403)
    return
  }

  // 生成 job ID
  jobID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)

  // 异步执行
  go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    args := strings.Fields(req.Command)
    cmd := exec.CommandContext(ctx, args[0], args[1:]...)
    cmd.Dir = projPath
    output, err := cmd.CombinedOutput()

    // 输出结果存入临时文件（简化，实际可用 Redis）
    result := map[string]interface{}{
      "jobId":   jobID,
      "status":  "completed",
      "output":  string(output),
    }
    if err != nil {
      result["status"] = "failed"
    }
    // 实际应通过 GET /api/projects/:id/scripts/executions/:jobId 查询
    log.Printf("Script exec [%s] finished: %s", jobID, result["status"])
  }()

  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(map[string]interface{}{"jobId": jobID})
  return
}
```

- [ ] **Step 2: 测试**

```bash
curl -X POST http://localhost:18788/api/projects/1/scripts/exec \
  -H "Content-Type: application/json" \
  -d '{"command":"npm run build"}'
```

---

## Task 14: 后端 - README 扫描

**Files:**
- Modify: `go_backend/main.go`

- [ ] **Step 1: 添加 README 扫描函数并扩展现有 scan 接口**

在 scanGitInfo 函数之后添加：

```go
// scanReadme 扫描 README 文件
func scanReadme(projPath string) string {
  for _, name := range []string{"README.md", "README.en.md", "README.cn.md"} {
    filePath := projPath + "/" + name
    if data, err := os.ReadFile(filePath); err == nil {
      content := string(data)
      // 简单转纯文本：去掉 markdown 语法
      content = regexp.MustCompile(`#{1,6}\s*`).ReplaceAllString(content, "")
      content = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(content, "$1")
      content = regexp.MustCompile(`[*_`]+`).ReplaceAllString(content, "")
      return strings.TrimSpace(content)
    }
  }
  return ""
}
```

- [ ] **Step 2: 扩展现有的 GET /api/projects/:id/scan 返回 README 和 frameworkType**

找到现有的项目列表查询，添加 README 和 frameworkType 字段（修改 Query 和 Scan 部分）：

```go
// 在 projectsHandler 的 GET 分支中，projects 表查询添加：
// 1. SELECT 增加 readme_content 字段（新增列默认为空）
// 2. 或者在 scanGitInfo 返回中添加 readme 和 framework 信息

// 实际方案：在 POST /api/projects/:id/scan 的响应中返回 README 和 framework
// 修改 scanGitInfo 调用处，返回额外信息
```

**注意：** README 内容和 frameworkType 通过现有 scan API 的响应返回，不需要改数据库 schema。

---

## Task 15: 前端 - ProjectDetail.vue 添加 Tabs

**Files:**
- Modify: `src/components/ProjectDetail.vue`

- [ ] **Step 1: 替换为 Tab 布局**

将现有的简单 Git 展示替换为 Element Plus el-tabs 结构：

```vue
<template>
  <el-drawer
    v-model="visible"
    title="项目详情"
    direction="rtl"
    size="400px"
    class="project-detail-drawer el-drawer--dark"
    :body-class="'drawer-dark-body'"
  >
    <div v-if="project" class="detail-content">
      <!-- 保持原有的 Project Header 和 Actions 不变 -->

      <!-- 原有 Git Info Section 替换为 Tabs -->
      <el-tabs v-model="activeTab" class="detail-tabs">
        <!-- Tab 1: 基本信息 -->
        <el-tab-pane label="基本信息" name="basic">
          <div class="tab-content">
            <div v-if="project.techStack" class="info-row">
              <span class="info-label">技术栈</span>
              <span class="tech-badge">{{ project.techStack }}</span>
            </div>
            <!-- Git 信息卡片 -->
            <div v-if="project.gitBranch" class="git-section">
              <h3 class="section-title">Git</h3>
              <div class="git-grid">
                <div class="git-item">
                  <span class="git-label">分支</span>
                  <span class="git-value">{{ project.gitBranch }}</span>
                </div>
                <div class="git-item" v-if="project.gitCommit">
                  <span class="git-label">Commit</span>
                  <span class="git-value commit">{{ project.gitCommit }}</span>
                </div>
                <div class="git-item">
                  <span class="git-label">状态</span>
                  <span class="git-value" :class="{ dirty: project.gitDirty }">
                    <span class="status-dot" :class="{ dirty: project.gitDirty }"></span>
                    {{ project.gitDirty ? '有更改' : '干净' }}
                  </span>
                </div>
              </div>
            </div>

            <!-- 别名/简介编辑表单 -->
            <div class="alias-section">
              <h3 class="section-title">项目信息</h3>
              <div class="alias-form">
                <div class="form-field">
                  <label>别名</label>
                  <input v-model="projectForm.alias" placeholder="项目别名" />
                </div>
                <div class="form-field">
                  <label>简介</label>
                  <textarea v-model="projectForm.description" placeholder="项目简介" rows="2"></textarea>
                </div>
                <button class="btn-save" @click="saveProjectInfo">保存</button>
              </div>
            </div>

            <!-- 标签选择 -->
            <div class="tags-section">
              <h3 class="section-title">标签</h3>
              <div class="tags-list">
                <span
                  v-for="tag in projectTags"
                  :key="tag.id"
                  class="tag-badge"
                  :style="{ backgroundColor: tag.color + '30', color: tag.color, borderColor: tag.color }"
                >
                  {{ tag.name }}
                </span>
                <button class="btn-add-tag" @click="showTagDialog = true">+ 添加标签</button>
              </div>
            </div>

            <!-- Scripts 列表 -->
            <div v-if="projectScripts.length > 0" class="scripts-section">
              <h3 class="section-title">Scripts ({{ projectFramework }})</h3>
              <div class="scripts-list">
                <div v-for="script in projectScripts" :key="script.name" class="script-item">
                  <span class="script-name">{{ script.name }}</span>
                  <button class="btn-run" @click="runScript(script)">运行</button>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Tab 2: Git 分支 -->
        <el-tab-pane label="分支" name="branches">
          <div class="tab-content">
            <div v-if="branches.length > 0" class="branches-list">
              <div
                v-for="branch in branches"
                :key="branch.name"
                class="branch-item"
              >
                <div class="branch-name">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="git-icon">
                    <line x1="6" y1="3" x2="6" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>
                  </svg>
                  {{ branch.name }}
                </div>
                <div class="branch-info">
                  <span class="branch-commit">{{ branch.lastCommit }}</span>
                  <span class="branch-msg">{{ branch.lastCommitMsg }}</span>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">暂无分支信息</div>
          </div>
        </el-tab-pane>

        <!-- Tab 3: Diff -->
        <el-tab-pane label="Diff" name="diff">
          <div class="tab-content">
            <div v-if="diffInfo.changed && diffInfo.changed.length > 0">
              <div class="diff-stats">
                <span class="stat-add">+{{ diffInfo.stats.additions }}</span>
                <span class="stat-del">-{{ diffInfo.stats.deletions }}</span>
              </div>
              <div v-for="file in diffInfo.changed" :key="file.file" class="diff-file">
                <span class="diff-file-name">{{ file.file }}</span>
                <span class="diff-nums">
                  <span class="add">+{{ file.additions }}</span>
                  <span class="del">-{{ file.deletions }}</span>
                </span>
              </div>
            </div>
            <div v-else class="empty-state">工作区干净，无更改</div>
          </div>
        </el-tab-pane>

        <!-- Tab 4: README -->
        <el-tab-pane label="README" name="readme">
          <div class="tab-content">
            <div v-if="readmeContent" class="readme-content">
              {{ readmeContent }}
            </div>
            <div v-else class="empty-state">未找到 README 文件</div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- Actions 保持不变 -->
      <div class="actions-section">...</div>
    </div>
  </el-drawer>
</template>
```

- [ ] **Step 2: 添加新的 data 和 methods**

```ts
const activeTab = ref('basic')
const showTagDialog = ref(false)
const projectTags = ref<any[]>([])
const projectScripts = ref<any[]>([])
const projectFramework = ref('')
const branches = ref<any[]>([])
const diffInfo = ref<any>({ changed: [], stats: { additions: 0, deletions: 0 } })
const readmeContent = ref('')
const projectForm = ref({
  alias: '',
  description: '',
})

async function loadProjectDetail() {
  if (!props.project) return

  // 加载标签
  const tagsRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/tags`)
  const tagsData = await tagsRes.json()
  projectTags.value = tagsData.tags || []

  // 加载 Scripts
  const scriptsRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/scripts`)
  const scriptsData = await scriptsRes.json()
  projectScripts.value = scriptsData.scripts || []
  projectFramework.value = scriptsData.framework || ''

  // 加载分支
  const branchesRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/branches`)
  const branchesData = await branchesRes.json()
  branches.value = branchesData.branches || []

  // 加载 Diff
  const diffRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/diff`)
  diffInfo.value = await diffRes.json()

  // 加载 README (通过 scan 接口或单独字段)
  readmeContent.value = props.project.readmeContent || ''

  // 初始化表单
  projectForm.value.alias = props.project.alias || ''
  projectForm.value.description = props.project.description || ''
}

async function saveProjectInfo() {
  // 调用 PATCH /api/projects/:id
  await fetch(`${API_BASE}/api/projects/${props.project.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: props.project.name,
      alias: projectForm.value.alias,
      description: projectForm.value.description,
    }),
  })
  ElMessage.success('保存成功')
  emit('refresh')
}

async function runScript(script: any) {
  await fetch(`${API_BASE}/api/projects/${props.project.id}/scripts/exec`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ command: script.command }),
  })
  ElMessage.success('脚本已提交执行')
}

onMounted(loadProjectDetail)
watch(() => props.project, loadProjectDetail)
```

---

## Task 16: 前端 - TagSelector.vue 组件

**Files:**
- Create: `src/components/TagSelector.vue`

- [ ] **Step 1: 创建标签选择器组件**

```vue
<template>
  <el-dialog v-model="visible" title="选择标签" width="400px">
    <div class="tag-grid">
      <div
        v-for="tag in allTags"
        :key="tag.id"
        class="tag-option"
        :class="{ selected: isSelected(tag.id) }"
        @click="toggleTag(tag)"
      >
        <span
          class="tag-color"
          :style="{ backgroundColor: tag.color }"
        ></span>
        <span class="tag-name">{{ tag.name }}</span>
        <svg v-if="isSelected(tag.id)" viewBox="0 0 24 24" fill="none" stroke="#22c55e" stroke-width="3" class="check-icon">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
      </div>
    </div>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="confirmSelection">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

const props = defineProps<{
  modelValue: boolean
  projectId: number
}>()
const emit = defineEmits(['update:modelValue', 'change'])

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const allTags = ref<any[]>([])
const selectedIds = ref<number[]>([])

async function loadTags() {
  const res = await fetch('http://localhost:18788/api/tags')
  const data = await res.json()
  allTags.value = data.tags || []

  // 加载项目当前标签
  const projRes = await fetch(`http://localhost:18788/api/projects/${props.projectId}/tags`)
  const projData = await projRes.json()
  selectedIds.value = projData.tags.map((t: any) => t.id)
}

function isSelected(id: number) {
  return selectedIds.value.includes(id)
}

function toggleTag(tag: any) {
  const idx = selectedIds.value.indexOf(tag.id)
  if (idx >= 0) {
    selectedIds.value.splice(idx, 1)
  } else {
    selectedIds.value.push(tag.id)
  }
}

async function confirmSelection() {
  await fetch(`http://localhost:18788/api/projects/${props.projectId}/tags`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tagIds: selectedIds.value }),
  })
  emit('change', selectedIds.value)
  visible.value = false
}

onMounted(loadTags)
</script>

<style scoped>
.tag-grid { display: flex; flex-direction: column; gap: 8px; }
.tag-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: rgba(30, 41, 59, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.12s ease;
}
.tag-option:hover { background: rgba(30, 41, 59, 0.8); }
.tag-option.selected { border-color: #22c55e; background: rgba(34, 197, 94, 0.1); }
.tag-color { width: 12px; height: 12px; border-radius: 50%; }
.tag-name { flex: 1; font-size: 14px; }
.check-icon { width: 16px; height: 16px; }
</style>
```

---

## Task 17: 前端 - 更新 App.vue (引入新组件)

**Files:**
- Modify: `src/App.vue`

- [ ] **Step 1: 在 App.vue 中引入 TagSelector**

在 ProjectDialog import 之后添加：
```ts
import TagSelector from './components/TagSelector.vue'
```

在 components 注册对象中添加：
```ts
TagSelector
```

- [ ] **Step 2: 在 ProjectDetail 旁边添加 TagSelector**

在 ProjectDialog 组件下方添加：
```vue
<TagSelector
  v-model="showTagDialog"
  :projectId="currentProjectId"
  @change="handleTagChange"
/>
```

添加状态：
```ts
const showTagDialog = ref(false)
const currentProjectId = ref<number | null>(null)

function handleTagChange() {
  // 刷新项目详情
}
```

---

## Verification Checklist

- [ ] `curl http://localhost:18788/api/tags` 返回标签列表
- [ ] `curl -X POST http://localhost:18788/api/tags -d '{"name":"java","color":"#ff6b6b"}'` 创建标签成功
- [ ] `curl -X PUT http://localhost:18788/api/projects/1/tags -d '{"tagIds":[1]}'` 更新标签成功
- [ ] `curl -X PATCH http://localhost:18788/api/projects/1 -d '{"name":"x","alias":"y","description":"z"}'` 更新项目信息成功
- [ ] `curl http://localhost:18788/api/projects/1/remote` 返回 Git 远程信息
- [ ] `curl http://localhost:18788/api/projects/1/branches` 返回分支列表
- [ ] `curl http://localhost:18788/api/projects/1/diff` 返回 diff 信息
- [ ] `curl http://localhost:18788/api/projects/1/scripts` 返回 scripts
- [ ] 前端 ProjectDetail 显示 4 个 Tab（基本信息/分支/Diff/README）
- [ ] 标签选择器可以创建和分配标签
- [ ] 脚本执行能提交任务并显示成功提示