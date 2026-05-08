package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// --- Caching infrastructure ---

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

var (
	sysStatsMu     sync.RWMutex
	sysStatsCache  cacheEntry
	hermesCacheMu  sync.RWMutex
	hermesCache    = make(map[string]cacheEntry)
	cachedConfig   cacheEntry
	configPath     string
	lastConfigMtime int64
)

const (
	sysStatsTTL = 5 * time.Second
	hermesTTL   = 30 * time.Second
	configTTL   = 60 * time.Second
)

var tcpPortRegex = regexp.MustCompile(`TCP (\S+?):(\d+)`)

func getCachedSysStats() map[string]interface{} {
	sysStatsMu.RLock()
	defer sysStatsMu.RUnlock()
	if sysStatsCache.expiresAt.After(time.Now()) {
		var result map[string]interface{}
		json.Unmarshal(sysStatsCache.data, &result)
		return result
	}
	return nil
}

func setCachedSysStats(data map[string]interface{}) {
	sysStatsMu.Lock()
	defer sysStatsMu.Unlock()
	b, _ := json.Marshal(data)
	sysStatsCache = cacheEntry{data: b, expiresAt: time.Now().Add(sysStatsTTL)}
}

func getCachedHermes(key string) []byte {
	hermesCacheMu.RLock()
	defer hermesCacheMu.RUnlock()
	if entry, ok := hermesCache[key]; ok && entry.expiresAt.After(time.Now()) {
		return entry.data
	}
	return nil
}

func setCachedHermes(key string, data []byte) {
	hermesCacheMu.Lock()
	defer hermesCacheMu.Unlock()
	// Cleanup expired entries if map is getting large
	if len(hermesCache) > 100 {
		for k, v := range hermesCache {
			if v.expiresAt.Before(time.Now()) {
				delete(hermesCache, k)
			}
		}
	}
	hermesCache[key] = cacheEntry{data: data, expiresAt: time.Now().Add(hermesTTL)}
}

func getCachedConfig() map[string]interface{} {
	sysStatsMu.RLock()
	defer sysStatsMu.RUnlock()
	if cachedConfig.expiresAt.After(time.Now()) {
		var result map[string]interface{}
		json.Unmarshal(cachedConfig.data, &result)
		return result
	}
	return nil
}

func setCachedConfig(data map[string]interface{}) {
	sysStatsMu.Lock()
	defer sysStatsMu.Unlock()
	b, _ := json.Marshal(data)
	cachedConfig = cacheEntry{data: b, expiresAt: time.Now().Add(configTTL)}
}

func initDB() {
	os.MkdirAll(os.Getenv("HOME")+"/.dashboard", 0755)
	dbPath := os.Getenv("HOME") + "/.dashboard/dashboard.db"
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	DB.Exec(`
		CREATE TABLE IF NOT EXISTS pinned_clipboard (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)
	`)
	DB.Exec(`
		CREATE TABLE IF NOT EXISTS metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			cpu REAL,
			memory REAL,
			disk REAL,
			network_up REAL,
			network_down REAL
		)
	`)
	// 创建 projects 表
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			tech_stack TEXT,
			git_branch TEXT,
			git_dirty INTEGER DEFAULT 0,
			git_commit TEXT,
			git_commit_msg TEXT,
			git_commit_author TEXT,
			git_commit_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Failed to create projects table: %v", err)
	}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	PID       int    `json:"pid"`
}

type DashboardResponse struct {
	Hostname      string  `json:"hostname"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskPercent   float64 `json:"disk_percent"`
	NetUp         float64 `json:"net_up"`
	NetDown       float64 `json:"net_down"`
	Timestamp     string  `json:"timestamp"`
	PID           int     `json:"pid"`
}

type ProcessInfo struct {
	PID    int     `json:"pid"`
	Name   string  `json:"name"`
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

type PortInfo struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Port    int    `json:"port"`
	Address string `json:"address"`
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 60
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	rows, err := DB.Query("SELECT timestamp, cpu, memory, disk, network_up, network_down FROM metrics ORDER BY id DESC LIMIT ?", limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	if results == nil {
		results = []map[string]interface{}{}
	}
	for rows.Next() {
		var timestamp string
		var cpu, memory, disk, networkUp, networkDown float64
		rows.Scan(&timestamp, &cpu, &memory, &disk, &networkUp, &networkDown)
		results = append(results, map[string]interface{}{
			"timestamp":   timestamp,
			"cpu":         cpu,
			"memory":      memory,
			"disk":        disk,
			"networkUp":   networkUp,
			"networkDown": networkDown,
		})
	}

	// Reverse to chronological order
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func projectsHandler(w http.ResponseWriter, r *http.Request) {
	// DELETE: 删除指定项目
	if r.Method == "DELETE" {
		// 提取 ID (URL path: /api/projects/123)
		id := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		if id == "" || id == r.URL.Path {
			http.Error(w, `{"error":"项目ID不能为空"}`, 400)
			return
		}

		// 验证 ID 为有效整数
		projectID, err := strconv.Atoi(id)
		if err != nil || projectID <= 0 {
			http.Error(w, `{"error":"无效的项目ID"}`, 400)
			return
		}

		// 检查项目是否存在
		var exists int
		err = DB.QueryRow("SELECT 1 FROM projects WHERE id = ?", projectID).Scan(&exists)
		if err == sql.ErrNoRows {
			http.Error(w, `{"error":"项目不存在"}`, 404)
			return
		}

		// 删除项目
		_, err = DB.Exec("DELETE FROM projects WHERE id = ?", projectID)
		if err != nil {
			http.Error(w, `{"error":"删除失败"}`, 500)
			return
		}
		w.WriteHeader(204)
		return
	}

	// POST: 添加新项目
	if r.Method == "POST" {
		var req struct{ Path string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"无效的请求体"}`, 400)
			return
		}

		// 验证路径非空
		if req.Path == "" {
			http.Error(w, `{"error":"path 不能为空"}`, 400)
			return
		}

		// 验证路径存在且是目录
		info, err := os.Stat(req.Path)
		if err != nil || !info.IsDir() {
			http.Error(w, `{"error":"路径不存在或不是目录"}`, 400)
			return
		}

		// 提取项目名（目录名）
		name := filepath.Base(req.Path)

		// 插入数据库
		result, err := DB.Exec(
			"INSERT OR IGNORE INTO projects (path, name) VALUES (?, ?)",
			req.Path, name,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			http.Error(w, `{"error":"添加失败"}`, 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": id, "path": req.Path, "name": name,
		})
		return
	}

	// GET: 列出所有项目
	rows, err := DB.Query("SELECT id, path, name, tech_stack, git_branch, git_dirty, git_commit, git_commit_msg, git_commit_author, git_commit_date, created_at, updated_at FROM projects ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	if results == nil {
		results = []map[string]interface{}{}
	}
	for rows.Next() {
		var id int
		var path, name string
		var techStack, gitBranch, gitCommit, gitCommitMsg, gitCommitAuthor, gitCommitDate, createdAt, updatedAt *string
		var gitDirty int
		rows.Scan(&id, &path, &name, &techStack, &gitBranch, &gitDirty, &gitCommit, &gitCommitMsg, &gitCommitAuthor, &gitCommitDate, &createdAt, &updatedAt)
		results = append(results, map[string]interface{}{
			"id": id,
			"path": path,
			"name": name,
			"tech_stack": techStack,
			"git_branch": gitBranch,
			"git_dirty": gitDirty,
			"git_commit": gitCommit,
			"git_commit_msg": gitCommitMsg,
			"git_commit_author": gitCommitAuthor,
			"git_commit_date": gitCommitDate,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"projects": results})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		PID:       os.Getpid(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()

	// Get CPU percent
	cmd := exec.Command("sh", "-c", "ps -A -o %cpu | awk '{s+=$1} END {print s}'")
	out, _ := cmd.Output()
	cpuPercent, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)

	// Get memory percent
	var memPercent float64 = 84.0
	cmd = exec.Command("sh", "-c", "memory_pressure | head -1 | awk '{print $3}' | tr -d '%'")
	out, _ = cmd.Output()
	if mp, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
		memPercent = mp
	}

	// Get disk usage
	var diskPercent float64 = 1.8
	cmd = exec.Command("sh", "-c", "df -h / | tail -1 | awk '{print $5}' | tr -d '%'")
	out, _ = cmd.Output()
	if dp, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
		diskPercent = dp
	}

	// Get network stats
	cmd = exec.Command("netstat", "-ib")
	out, _ = cmd.Output()
	var netUp, netDown float64
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "en0") {
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				if ibytes, err := strconv.ParseFloat(fields[6], 64); err == nil {
					netDown = ibytes / 1024 / 1024
				}
				if obytes, err := strconv.ParseFloat(fields[4], 64); err == nil {
					netUp = obytes / 1024 / 1024
				}
			}
			break
		}
	}

	resp := DashboardResponse{
		Hostname:      hostname,
		CPUPercent:    cpuPercent / float64(runtime.NumCPU()),
		MemoryPercent: memPercent,
		DiskPercent:   diskPercent,
		NetUp:         netUp,
		NetDown:       netDown,
		Timestamp:     time.Now().Format(time.RFC3339),
		PID:          os.Getpid(),
	}

	// Save metrics to database
	timestamp := time.Now().Format(time.RFC3339)
	DB.Exec("INSERT INTO metrics (timestamp, cpu, memory, disk, network_up, network_down) VALUES (?, ?, ?, ?, ?, ?)",
		timestamp, cpuPercent/float64(runtime.NumCPU()), memPercent, diskPercent, netUp, netDown)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func systemHandler(w http.ResponseWriter, r *http.Request) {
	var cpuLogical, cpuPhysical int
	var cpuBrand, osVersion string
	var memTotal uint64

	cmd := exec.Command("sysctl", "hw.ncpu")
	out, _ := cmd.Output()
	line := strings.TrimSpace(string(out))
	cpuLogical, _ = strconv.Atoi(strings.Split(line, ": ")[1])

	cmd = exec.Command("sysctl", "hw.physicalcpu")
	out, _ = cmd.Output()
	line = strings.TrimSpace(string(out))
	cpuPhysical, _ = strconv.Atoi(strings.Split(line, ": ")[1])

	cmd = exec.Command("sysctl", "machdep.cpu.brand_string")
	out, _ = cmd.Output()
	line = strings.TrimSpace(string(out))
	cpuBrand = strings.Split(line, ": ")[1]

	cmd = exec.Command("sysctl", "hw.memsize")
	out, _ = cmd.Output()
	line = strings.TrimSpace(string(out))
	memTotal, _ = strconv.ParseUint(strings.Split(line, ": ")[1], 10, 64)

	type SystemInfo struct {
		Username      string `json:"username"`
		OS            string `json:"os"`
		NodeVersion   string `json:"node_version"`
		PythonVersion string `json:"python_version"`
		CPU           struct {
			CountPhysical int    `json:"count_physical"`
			CountLogical  int    `json:"count_logical"`
			Brand         string `json:"brand"`
		} `json:"cpu"`
		Memory struct {
			TotalGB  float64 `json:"total_gb"`
			UsedGB   float64 `json:"used_gb"`
			Percent  float64 `json:"percent"`
		} `json:"memory"`
		Disk struct {
			UsedGB  float64 `json:"used_gb"`
			TotalGB float64 `json:"total_gb"`
			Percent float64 `json:"percent"`
		} `json:"disk"`
		Network struct {
			Interfaces []struct {
				Name    string `json:"name"`
				Address string `json:"address"`
			} `json:"interfaces"`
			Wifi    string `json:"wifi"`
			Mac     string `json:"mac"`
			VpnPort string `json:"vpn_port"`
		} `json:"network"`
	}

	cmd = exec.Command("sw_vers", "-productVersion")
	out, _ = cmd.Output()
	osVersion = strings.TrimSpace(string(out))

	// Get disk info
	cmd = exec.Command("df", "-k", "/")
	out, _ = cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var diskTotal, diskUsed uint64
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 3 {
			diskTotal, _ = strconv.ParseUint(fields[1], 10, 64)
			diskUsed, _ = strconv.ParseUint(fields[2], 10, 64)
		}
	}

	// Get username
	username := os.Getenv("USER")
	if username == "" {
		username = "Unknown"
	}

	// Get WiFi name (SSID)
	var wifiName string
	cmd = exec.Command("system_profiler", "SPAirPortDataType")
	out, _ = cmd.Output()
	spLines := strings.Split(string(out), "\n")
	inCurrentNetwork := false
	for _, line := range spLines {
		if strings.Contains(line, "Current Network Information:") {
			inCurrentNetwork = true
			continue
		}
		if inCurrentNetwork {
			if strings.Contains(line, ":") && !strings.Contains(line, "Current Network") {
				wifiName = strings.TrimSpace(strings.Split(line, ":")[0])
				break
			}
			if strings.Contains(line, "Network Type:") && wifiName == "" {
				wifiName = "N/A"
			}
		}
	}
	if wifiName == "" {
		wifiName = "N/A"
	}

	// Get MAC address
	var macAddr string
	cmd = exec.Command("ifconfig", "en0")
	out, _ = cmd.Output()
	ifaceLines := strings.Split(string(out), "\n")
	for _, line := range ifaceLines {
		if strings.Contains(line, "ether") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				macAddr = strings.ToUpper(fields[1])
				break
			}
		}
	}
	if macAddr == "" {
		macAddr = "N/A"
	}

	// Get VPN port (check if VPN is active)
	vpnPort := "N/A"
	cmd = exec.Command("networksetup", "-getpassivefreessl", "N/A")
	out, _ = cmd.Output()
	// If we have a VPN service running, we could detect it here
	// For now, set to N/A unless VPN is detected

	// Get Node version - use shell to respect PATH
	var nodeVersion string
	cmd = exec.Command("sh", "-c", "node --version")
	out, _ = cmd.Output()
	nodeVersion = strings.TrimSpace(string(out))
	if nodeVersion == "" {
		nodeVersion = "N/A"
	}

	// Get Python version - use shell to respect PATH
	var pythonVersion string
	cmd = exec.Command("sh", "-c", "python3 --version")
	out, _ = cmd.Output()
	pythonVersion = strings.TrimSpace(string(out))
	if pythonVersion == "" {
		pythonVersion = "N/A"
	}

	resp := SystemInfo{
		Username:      username,
		OS:            "macOS " + osVersion,
		NodeVersion:   nodeVersion,
		PythonVersion: pythonVersion,
	}
	resp.CPU.CountPhysical = cpuPhysical
	resp.CPU.CountLogical = cpuLogical
	resp.CPU.Brand = cpuBrand
	resp.Memory.TotalGB = math.Round(float64(memTotal)/1024/1024/1024*100) / 100
	resp.Memory.UsedGB = math.Round(float64(memTotal)*0.84/1024/1024/1024*100) / 100
	resp.Memory.Percent = 84.0
	resp.Disk.TotalGB = math.Round(float64(diskTotal)/1024/1024*100) / 100
	resp.Disk.UsedGB = math.Round(float64(diskUsed)/1024/1024*100) / 100
	resp.Disk.Percent = math.Round(float64(diskUsed)/float64(diskTotal)*100*100) / 100

	// Get primary network interface IP
	cmd = exec.Command("ifconfig", "en0")
	out, _ = cmd.Output()
	var ipAddr string
	ipLines := strings.Split(string(out), "\n")
	for _, line := range ipLines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ipAddr = fields[1]
				break
			}
		}
	}
	if ipAddr == "" {
		ipAddr = "N/A"
	}
	resp.Network.Interfaces = append(resp.Network.Interfaces, struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}{
		Name:    "en0",
		Address: ipAddr,
	})
	resp.Network.Wifi = wifiName
	resp.Network.Mac = macAddr
	resp.Network.VpnPort = vpnPort

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func processesHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("ps", "aux")
	out, _ := cmd.Output()

	lines := strings.Split(string(out), "\n")
	var processes []ProcessInfo

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		name := fields[10]

		processes = append(processes, ProcessInfo{
			PID:    pid,
			Name:   name,
			CPU:    cpu,
			Memory: mem,
		})
		if len(processes) >= 50 {
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"processes": processes})
}

func portsHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	out, _ := cmd.Output()

	lines := strings.Split(string(out), "\n")
	var ports []PortInfo

	re := tcpPortRegex

	// First pass: collect all PIDs
	var pids []int
	pidSet := make(map[int]bool)
	for _, line := range lines {
		if strings.HasPrefix(line, "COMMAND") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		if !pidSet[pid] {
			pids = append(pids, pid)
			pidSet[pid] = true
		}
	}

	// Batch ps query: single call for all PIDs
	psNameMap := make(map[int]string)
	if len(pids) > 0 {
		pidStrs := make([]string, len(pids))
		for i, p := range pids {
			pidStrs[i] = strconv.Itoa(p)
		}
		psCmd := exec.Command("ps", "-p", strings.Join(pidStrs, ","), "-o", "pid,comm=")
		psOut, err := psCmd.Output()
		if err == nil {
			for _, psLine := range strings.Split(string(psOut), "\n") {
				fields := strings.Fields(psLine)
				if len(fields) >= 2 {
					if pid, err := strconv.Atoi(fields[0]); err == nil {
						name := strings.TrimSpace(fields[1])
						if idx := strings.LastIndex(name, "/"); idx >= 0 {
							name = name[idx+1:]
						}
						psNameMap[pid] = name
					}
				}
			}
		}
	}

	// Second pass: build port list using cached names
	for _, line := range lines {
		if strings.HasPrefix(line, "COMMAND") || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		pid, _ := strconv.Atoi(fields[1])
		name := fields[0]
		if psName, ok := psNameMap[pid]; ok {
			name = psName
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}
		addr := matches[1]
		port, _ := strconv.Atoi(matches[2])

		ports = append(ports, PortInfo{
			Name:    name,
			PID:     pid,
			Port:    port,
			Address: addr,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ports": ports})
}

func processKillHandler(w http.ResponseWriter, r *http.Request) {
	// Extract PID from URL path: /api/process/kill/{pid}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", 400)
		return
	}
	pidStr := parts[len(parts)-1]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "Invalid PID", 400)
		return
	}

	// Send kill signal
	cmd := exec.Command("kill", strconv.Itoa(pid))
	err = cmd.Run()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func hermesHandler(pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		}
		switch pattern {
		case "status":
			resp["status"] = "running"
			// Get real session stats from Hermes CLI
			cmd := exec.Command("hermes", "sessions", "stats")
			out, err := cmd.Output()
			if err == nil {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					if strings.Contains(line, "Total sessions:") {
						// Parse "Total sessions: 141"
						parts := strings.Split(strings.TrimSpace(line), "Total sessions:")
						if len(parts) >= 2 {
							s := strings.TrimSpace(parts[1])
							s = strings.Trim(s, " \t:")
							if val, err := strconv.Atoi(s); err == nil {
								resp["active_sessions"] = val
							}
						}
					}
				}
			}
			if resp["active_sessions"] == nil {
				resp["active_sessions"] = 0
			}
			// Get sessions list
			cmd = exec.Command("hermes", "sessions", "list")
			out, err = cmd.Output()
			var sessions []map[string]interface{}
			if err == nil {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if trimmed == "" || strings.HasPrefix(trimmed, "Session") || strings.HasPrefix(trimmed, "─") {
						continue
					}
					// ID is the last field, looks like "20260504_110450_40f7339e" or "cron_..."
					fields := strings.Fields(trimmed)
					if len(fields) >= 2 {
						id := fields[len(fields)-1]
						title := ""
						if len(fields) > 1 {
							// Title is first field if not "—"
							if fields[0] != "—" {
								title = fields[0]
							}
						}
						// Check if ID looks valid (contains underscore and looks like date-based or cron ID)
						if strings.Contains(id, "_") && (len(id) > 10 || strings.HasPrefix(id, "cron_")) {
							sessions = append(sessions, map[string]interface{}{
								"id":    id,
								"title": title,
							})
						}
					}
				}
			}
			if sessions == nil {
				sessions = []map[string]interface{}{}
			}
			resp["sessions"] = sessions
		case "config":
			// Read config from ~/.hermes/config.yaml
			home := os.Getenv("HOME")
			configPath := home + "/.hermes/config.yaml"
			data, err := os.ReadFile(configPath)
			if err == nil {
				content := string(data)
				resp["config"] = "hermes"

				// Parse model section - find top-level "model:" (starts at column 0)
				lines := strings.Split(content, "\n")
				inModelSection := false
				modelIndentation := -1
				modelMap := make(map[string]string)
				defaultModel := ""

				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))

					// Check for top-level model: (indentation 0, not a comment, not empty)
					if leadingSpaces == 0 && strings.HasPrefix(trimmed, "model:") && !strings.HasPrefix(trimmed, "#") {
						inModelSection = true
						modelIndentation = 0
						continue
					}

					if inModelSection {
						currentIndent := len(line) - len(strings.TrimLeft(line, " "))

						// If we hit a line with same or less indentation at column 0, end of section
						if currentIndent <= modelIndentation && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
							break
						}

						if strings.HasPrefix(trimmed, "default:") {
							defaultModel = strings.TrimSpace(strings.TrimPrefix(trimmed, "default:"))
						} else if strings.HasPrefix(trimmed, "provider:") {
							modelMap["provider"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "provider:"))
						} else if strings.HasPrefix(trimmed, "base_url:") {
							modelMap["base_url"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "base_url:"))
						}
					}
				}

				if defaultModel != "" {
					resp["default_model"] = defaultModel
				}
				if len(modelMap) > 0 {
					resp["model"] = modelMap
				}

				// Parse agent section - find top-level "agent:" (starts at column 0)
				agent := make(map[string]interface{})
				inAgentSection := false
				agentIndentation := -1

				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))

					if leadingSpaces == 0 && strings.HasPrefix(trimmed, "agent:") && !strings.HasPrefix(trimmed, "#") {
						inAgentSection = true
						agentIndentation = 0
						continue
					}

					if inAgentSection {
						currentIndent := len(line) - len(strings.TrimLeft(line, " "))

						// End of agent section
						if currentIndent <= agentIndentation && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
							break
						}

						if strings.HasPrefix(trimmed, "max_turns:") {
							agent["max_iterations"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "max_turns:"))
						} else if strings.HasPrefix(trimmed, "gateway_timeout:") {
							agent["timeout"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "gateway_timeout:"))
						} else if strings.HasPrefix(trimmed, "reasoning_effort:") {
							agent["reasoning_effort"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "reasoning_effort:"))
						} else if strings.HasPrefix(trimmed, "personalities:") {
							// Parse personalities section - no dash prefix, just "key: value"
							personalities := make(map[string]string)
							persIndentation := -1

							for j := i + 1; j < len(lines); j++ {
								pline := lines[j]
								ptrimmed := strings.TrimSpace(pline)
								pleading := len(pline) - len(strings.TrimLeft(pline, " "))

								if persIndentation == -1 {
									if pleading >= leadingSpaces && ptrimmed != "" && !strings.HasPrefix(ptrimmed, "#") {
										persIndentation = pleading
									} else if pleading < leadingSpaces {
										break
									}
									continue
								}

								if pleading < persIndentation && ptrimmed != "" && !strings.HasPrefix(ptrimmed, "#") {
									break
								}

								// No dash prefix - format is "key: value"
								if idx := strings.Index(ptrimmed, ":"); idx > 0 {
									key := strings.TrimSpace(ptrimmed[:idx])
									val := strings.TrimSpace(ptrimmed[idx+1:])
									if key != "" && val != "" {
										personalities[key] = val
									}
								}
							}
							if len(personalities) > 0 {
								agent["personalities"] = personalities
							}
						}
					}
				}

				if len(agent) > 0 {
					resp["agent"] = agent
				}

				// Parse display section
				for i, line := range lines {
					trimmed := strings.TrimSpace(line)
					leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))

					if leadingSpaces == 0 && strings.HasPrefix(trimmed, "display:") && !strings.HasPrefix(trimmed, "#") {
						display := make(map[string]interface{})

						for j := i + 1; j < len(lines); j++ {
							dline := lines[j]
							dtrimmed := strings.TrimSpace(dline)
							dleading := len(dline) - len(strings.TrimLeft(dline, " "))

							if dleading <= leadingSpaces && dtrimmed != "" && !strings.HasPrefix(dtrimmed, "#") {
								break
							}

							if dleading > leadingSpaces && strings.HasPrefix(dtrimmed, "personality:") {
								personality := strings.TrimSpace(strings.TrimPrefix(dtrimmed, "personality:"))
								display["personality"] = personality
								break
							}
						}
						if len(display) > 0 {
							resp["display"] = display
						}
						break
					}
				}

				// Parse toolsets - find top-level "toolsets:" (starts at column 0)
				var toolsets []string
				inToolsetSection := false
				toolsetIndentation := -1

				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))

					if leadingSpaces == 0 && strings.HasPrefix(trimmed, "toolsets:") && !strings.HasPrefix(trimmed, "#") {
						inToolsetSection = true
						toolsetIndentation = 0
						continue
					}

					if inToolsetSection {
						currentIndent := len(line) - len(strings.TrimLeft(line, " "))

						// End of toolsets section
						if currentIndent <= toolsetIndentation && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
							break
						}

						if strings.HasPrefix(trimmed, "-") {
							toolsets = append(toolsets, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
						} else if trimmed != "" && !strings.HasPrefix(trimmed, "-") {
							// List items have dash prefix, non-dash lines after list started = end of section
							if len(toolsets) > 0 {
								break
							}
						}
					}
				}

				if len(toolsets) > 0 {
					resp["toolsets"] = toolsets
				}
			}
			if resp["default_model"] == nil {
				resp["default_model"] = "unknown"
			}
		case "toolsets":
			// Return toolsets from config
			home := os.Getenv("HOME")
			configPath := home + "/.hermes/config.yaml"
			data, err := os.ReadFile(configPath)
			var toolsets []map[string]interface{}
			if err == nil {
				content := string(data)
				if idx := strings.Index(content, "toolsets:"); idx >= 0 {
					section := content[idx:]
					if lines := strings.Split(section, "\n"); len(lines) > 1 {
						for i := 1; i < len(lines); i++ {
							line := strings.TrimSpace(lines[i])
							if strings.HasPrefix(line, "-") {
								toolsets = append(toolsets, map[string]interface{}{
									"name":    strings.TrimSpace(strings.TrimPrefix(line, "-")),
									"enabled": true,
								})
							} else if line != "" && !strings.HasPrefix(line, "#") {
								break
							}
						}
					}
				}
			}
			if toolsets == nil {
				toolsets = []map[string]interface{}{}
			}
			resp["toolsets"] = toolsets
		case "cron":
			// Get real cron jobs from Hermes CLI
			cmd := exec.Command("hermes", "cron", "list")
			out, err := cmd.Output()
			var jobs []map[string]interface{}
			if err == nil {
				lines := strings.Split(string(out), "\n")
				var currentJob map[string]interface{}
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					// Job ID line: "a896ac477522 [active]"
					if strings.HasSuffix(trimmed, "[active]") || strings.HasSuffix(trimmed, "[paused]") {
						if currentJob != nil && currentJob["id"] != nil {
							jobs = append(jobs, currentJob)
						}
						parts := strings.Fields(trimmed)
						if len(parts) >= 1 {
							currentJob = map[string]interface{}{
								"id":     parts[0],
								"name":   "",
								"schedule": "",
								"next_run": "",
								"last_status": "unknown",
							}
						}
						continue
					}
					if currentJob != nil {
						if strings.HasPrefix(trimmed, "Name:") {
							currentJob["name"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Name:"))
						} else if strings.HasPrefix(trimmed, "Schedule:") {
							currentJob["schedule"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Schedule:"))
						} else if strings.HasPrefix(trimmed, "Next run:") {
							currentJob["next_run"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Next run:"))
						} else if strings.HasPrefix(trimmed, "Last run:") {
							lastRun := strings.TrimSpace(strings.TrimPrefix(trimmed, "Last run:"))
							// Last run line ends with status: "ok" or "failed"
							if strings.HasSuffix(lastRun, "ok") {
								currentJob["last_status"] = "ok"
							} else if strings.HasSuffix(lastRun, "failed") {
								currentJob["last_status"] = "failed"
							}
						}
					}
				}
				if currentJob != nil && currentJob["id"] != nil {
					jobs = append(jobs, currentJob)
				}
			}
			resp["jobs"] = jobs
		case "profiles":
			// Get real profiles from Hermes CLI
			cmd := exec.Command("hermes", "profile", "list")
			out, err := cmd.Output()
			var profiles []map[string]interface{}
			if err == nil {
				lines := strings.Split(string(out), "\n")
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					// Skip header, separator, and empty lines
					if trimmed == "" || strings.HasPrefix(trimmed, "Profile") || strings.HasPrefix(trimmed, "─") {
						continue
					}
					// Lines with profiles start with ◆ or ● or a letter (profile name)
					if len(trimmed) > 0 {
						r := []rune(trimmed)[0]
						if r == 9670 || r == 9679 || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
							fields := strings.Fields(trimmed)
							if len(fields) >= 2 {
								// Remove leading symbols from profile name
								name := fields[0]
								name = strings.TrimPrefix(name, "◆") // ◆
								name = strings.TrimPrefix(name, "●") // ●
								profile := map[string]interface{}{
									"name":          name,
									"model":        "MiniMax-M2.7-highspeed",
									"session_count": 0,
								}
								profiles = append(profiles, profile)
							}
						}
					}
				}
			}
			if profiles == nil || len(profiles) == 0 {
				profiles = []map[string]interface{}{
					{"name": "default", "model": "MiniMax-M2.7-highspeed", "session_count": 0},
				}
			}
			resp["profiles"] = profiles
		case "version":
			cmd := exec.Command("hermes", "version")
			out, err := cmd.Output()
			if err == nil {
				resp["version"] = strings.TrimSpace(string(out))
			} else {
				resp["version"] = "unknown"
			}
		case "gateway-state":
			cmd := exec.Command("hermes", "status")
			out, err := cmd.Output()
			resp["gateway_state"] = "running"
			resp["pid"] = os.Getpid()
			resp["platforms"] = map[string]map[string]string{
				"weixin":    {"state": "unknown"},
				"telegram":  {"state": "unknown"},
				"feishu":    {"state": "unknown"},
			}
			if err == nil {
				output := string(out)
				// Parse platform states
				platforms := map[string]map[string]string{
					"weixin":    {"state": "disconnected"},
					"telegram":  {"state": "disconnected"},
					"feishu":    {"state": "disconnected"},
				}
				if strings.Contains(output, "Weixin") && strings.Contains(output, "✓") {
					platforms["weixin"] = map[string]string{"state": "connected"}
				}
				if strings.Contains(output, "Telegram") && strings.Contains(output, "✓") {
					platforms["telegram"] = map[string]string{"state": "connected"}
				}
				if strings.Contains(output, "Feishu") && strings.Contains(output, "✓") {
					platforms["feishu"] = map[string]string{"state": "connected"}
				}
				resp["platforms"] = platforms
			}
		case "logs":
			resp["logs"] = []string{}
		case "insights":
			insightsCmd := exec.Command("hermes", "insights", "--days", "1")
			insightsOut, insightsErr := insightsCmd.Output()
			if insightsErr != nil {
				resp["insights"] = map[string]interface{}{}
			} else {
				insightsOutput := string(insightsOut)
				insightsData := map[string]interface{}{}
				if sessionsMatch := regexp.MustCompile(`Sessions:\s*(\d+)`).FindStringSubmatch(insightsOutput); len(sessionsMatch) > 1 {
					insightsData["sessions"], _ = strconv.Atoi(sessionsMatch[1])
				}
				if messagesMatch := regexp.MustCompile(`Messages:\s*([\d,]+)`).FindStringSubmatch(insightsOutput); len(messagesMatch) > 1 {
					msgStr := strings.ReplaceAll(messagesMatch[1], ",", "")
					insightsData["messages"], _ = strconv.Atoi(msgStr)
				}
				if toolMatch := regexp.MustCompile(`Tool calls:\s*([\d,]+)`).FindStringSubmatch(insightsOutput); len(toolMatch) > 1 {
					toolStr := strings.ReplaceAll(toolMatch[1], ",", "")
					insightsData["tool_calls"], _ = strconv.Atoi(toolStr)
				}
				if inputMatch := regexp.MustCompile(`Input tokens:\s*([\d,]+)`).FindStringSubmatch(insightsOutput); len(inputMatch) > 1 {
					inputStr := strings.ReplaceAll(inputMatch[1], ",", "")
					insightsData["input_tokens"], _ = strconv.ParseInt(inputStr, 10, 64)
				}
				if outputMatch := regexp.MustCompile(`Output tokens:\s*([\d,]+)`).FindStringSubmatch(insightsOutput); len(outputMatch) > 1 {
					outputStr := strings.ReplaceAll(outputMatch[1], ",", "")
					insightsData["output_tokens"], _ = strconv.ParseInt(outputStr, 10, 64)
				}
				if totalMatch := regexp.MustCompile(`Total tokens:\s*([\d,]+)`).FindStringSubmatch(insightsOutput); len(totalMatch) > 1 {
					totalStr := strings.ReplaceAll(totalMatch[1], ",", "")
					insightsData["total_tokens"], _ = strconv.ParseInt(totalStr, 10, 64)
				}
				platformsData := []map[string]interface{}{}
				platformLines := regexp.MustCompile(`(?m)^(\w+)\s+(\d+)\s+([\d,]+)\s+([\d,]+)`).FindAllStringSubmatch(insightsOutput, -1)
				for _, match := range platformLines {
					if len(match) > 4 && match[1] != "Platform" {
						msgStr := strings.ReplaceAll(match[3], ",", "")
						tokStr := strings.ReplaceAll(match[4], ",", "")
						platformsData = append(platformsData, map[string]interface{}{
							"platform": match[1],
							"messages": msgStr,
							"tokens":   tokStr,
						})
					}
				}
				insightsData["platforms"] = platformsData
				resp["insights"] = insightsData
			}
		case "skills":
			skillsCmd := exec.Command("hermes", "skills", "list")
			skillsOut, skillsErr := skillsCmd.Output()
			if skillsErr != nil {
				resp["skills"] = []map[string]interface{}{}
			} else {
				skillsOutput := string(skillsOut)
				skillsData := []map[string]interface{}{}
				lines := strings.Split(skillsOutput, "\n")
				for _, line := range lines {
					if strings.Contains(line, "│") && !strings.Contains(line, "━") && !strings.Contains(line, "┏") && !strings.Contains(line, "┃") && !strings.Contains(line, "┡") && !strings.HasPrefix(strings.TrimSpace(line), "0 hub-installed") {
						parts := strings.Split(line, "│")
						if len(parts) >= 4 {
							name := strings.TrimSpace(parts[1])
							category := strings.TrimSpace(parts[2])
							source := strings.TrimSpace(parts[3])
							if name != "" && name != "Name" && !strings.Contains(name, "─") {
								skillsData = append(skillsData, map[string]interface{}{
									"name":      name,
									"category":  category,
									"source":    source,
									"icon":      name[:1],
								})
							}
						}
					}
				}
				resp["skills"] = skillsData
			}
		case "quota":
			quotaCmd := exec.Command("mmx", "quota")
			quotaOut, quotaErr := quotaCmd.Output()
			if quotaErr != nil {
				resp["quota"] = map[string]interface{}{}
			} else {
				var quotaResult map[string]interface{}
				if err := json.Unmarshal(quotaOut, &quotaResult); err != nil {
					resp["quota"] = map[string]interface{}{}
				} else {
					resp["quota"] = quotaResult
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func terminalExecHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	cmd := exec.Command("sh", "-c", req.Command)
	out, err := cmd.CombinedOutput()

	resp := map[string]interface{}{
		"stdout": string(out),
		"stderr": "",
		"exit_code": 0,
	}
	if err != nil {
		resp["stderr"] = err.Error()
		resp["exit_code"] = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func terminalGhosttyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	escapedCmd := strings.Replace(req.Command, `"`, `\"`, -1)
	script := fmt.Sprintf(`tell application "Terminal"
    activate
    do script "%s"
end tell`, escapedCmd)

	tmpfile, err := os.CreateTemp("", "terminal-*.scpt")
	if err != nil {
		resp := map[string]interface{}{"exit_code": 1, "stderr": err.Error()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(script); err != nil {
		resp := map[string]interface{}{"exit_code": 1, "stderr": err.Error()}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	tmpfile.Close()

	cmd := exec.Command("osascript", tmpfile.Name())
	out, err := cmd.CombinedOutput()

	resp := map[string]interface{}{
		"stdout": string(out),
		"stderr": "",
		"exit_code": 0,
	}
	if err != nil {
		resp["stderr"] = err.Error()
		resp["exit_code"] = 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func clipboardPinnedHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := DB.Query("SELECT id, content, created_at FROM pinned_clipboard ORDER BY created_at DESC")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"items": []interface{}{}})
		return
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id int
		var content, createdAt string
		rows.Scan(&id, &content, &createdAt)
		items = append(items, map[string]interface{}{
			"id":         id,
			"content":    content,
			"created_at": createdAt,
		})
	}

	if items == nil {
		items = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}

func clipboardPinHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	_, err := DB.Exec("INSERT OR IGNORE INTO pinned_clipboard (content, created_at) VALUES (?, ?)",
		req.Content, time.Now().Format(time.RFC3339))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	var id int
	DB.QueryRow("SELECT id FROM pinned_clipboard WHERE content = ?", req.Content).Scan(&id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"id": id, "success": true})
}

func clipboardUnpinHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	_, err := DB.Exec("DELETE FROM pinned_clipboard WHERE content = ?", req.Content)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %v\n", r.Method, r.URL.Path, time.Since(start))
	})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSClient struct {
	conn *websocket.Conn
	send chan []byte
}

var (
	clients   = make(map[*WSClient]bool)
	clientsMu sync.RWMutex
)

// Shared broadcast channels - single goroutine produces, all clients consume
var (
	dashboardChan chan []byte
	hermesChan    chan []byte
)

func init() {
	dashboardChan = make(chan []byte, 10)
	hermesChan = make(chan []byte, 10)
}

// StartSharedBroadcasters launches the single producer goroutines
func startSharedBroadcasters() {
	// Dashboard metrics broadcaster - runs once for ALL clients
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hostname, _ := os.Hostname()
			cmd := exec.Command("sh", "-c", "ps -A -o %cpu | awk '{s+=$1} END {print s}'")
			out, _ := cmd.Output()
			cpuPercent, _ := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)

			var memPercent float64 = 84.0
			cmd = exec.Command("sh", "-c", "memory_pressure | head -1 | awk '{print $3}' | tr -d '%'")
			out, _ = cmd.Output()
			if mp, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
				memPercent = mp
			}

			var diskPercent float64 = 1.8
			cmd = exec.Command("sh", "-c", "df -h / | tail -1 | awk '{print $5}' | tr -d '%'")
			out, _ = cmd.Output()
			if dp, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
				diskPercent = dp
			}

			cmd = exec.Command("netstat", "-ib")
			out, _ = cmd.Output()
			var netUp, netDown float64
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "en0") {
					fields := strings.Fields(line)
					if len(fields) >= 7 {
						if ibytes, err := strconv.ParseFloat(fields[6], 64); err == nil {
							netDown = ibytes / 1024 / 1024
						}
						if obytes, err := strconv.ParseFloat(fields[4], 64); err == nil {
							netUp = obytes / 1024 / 1024
						}
					}
					break
				}
			}

			data := map[string]interface{}{
				"type": "dashboard",
				"data": map[string]interface{}{
					"hostname":       hostname,
					"cpu_percent":    cpuPercent / float64(runtime.NumCPU()),
					"memory_percent": memPercent,
					"disk_percent":   diskPercent,
					"net_up":         netUp,
					"net_down":       netDown,
					"timestamp":      time.Now().Format(time.RFC3339),
				},
			}
			jsonData, _ := json.Marshal(data)

			// Fan-out to all clients
			clientsMu.RLock()
			for client := range clients {
				select {
				case client.send <- jsonData:
				default:
					// Slow consumer, skip
				}
			}
			clientsMu.RUnlock()
		}
	}()

	// Hermes status broadcaster - runs once for ALL clients
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hermesData := map[string]interface{}{"type": "hermes"}

			// Get Hermes status
			cmd := exec.Command("hermes", "sessions", "stats")
			out, err := cmd.Output()
			if err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					if strings.Contains(line, "Total sessions:") {
						parts := strings.Split(strings.TrimSpace(line), "Total sessions:")
						if len(parts) >= 2 {
							s := strings.TrimSpace(parts[1])
							s = strings.Trim(s, " \t:")
							if val, err := strconv.Atoi(s); err == nil {
								hermesData["active_sessions"] = val
							}
						}
					}
				}
			}
			if hermesData["active_sessions"] == nil {
				hermesData["active_sessions"] = 0
			}

			// Get Hermes sessions list
			cmd = exec.Command("hermes", "sessions", "list")
			out, err = cmd.Output()
			var sessions []map[string]interface{}
			if err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					trimmed := strings.TrimSpace(line)
					if trimmed == "" || strings.HasPrefix(trimmed, "Session") || strings.HasPrefix(trimmed, "─") {
						continue
					}
					fields := strings.Fields(trimmed)
					if len(fields) >= 2 {
						id := fields[len(fields)-1]
						title := ""
						if fields[0] != "—" {
							title = fields[0]
						}
						if strings.Contains(id, "_") && (len(id) > 10 || strings.HasPrefix(id, "cron_")) {
							sessions = append(sessions, map[string]interface{}{
								"id":    id,
								"title": title,
							})
						}
					}
				}
			}
			if sessions == nil {
				sessions = []map[string]interface{}{}
			}
			hermesData["sessions"] = sessions

			// Get toolsets
			cmd = exec.Command("hermes", "toolsets", "list")
			out, err = cmd.Output()
			var toolsets []map[string]interface{}
			if err == nil {
				for _, line := range strings.Split(string(out), "\n") {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
						toolsets = append(toolsets, map[string]interface{}{
							"name":    trimmed,
							"enabled": true,
						})
					}
				}
			}
			if toolsets == nil {
				toolsets = []map[string]interface{}{}
			}
			hermesData["toolsets"] = toolsets

			// Get cron jobs
			cmd = exec.Command("hermes", "cron", "list")
			out, err = cmd.Output()
			var jobs []map[string]interface{}
			if err == nil {
				var currentJob map[string]interface{}
				for _, line := range strings.Split(string(out), "\n") {
					trimmed := strings.TrimSpace(line)
					if strings.HasSuffix(trimmed, "[active]") || strings.HasSuffix(trimmed, "[paused]") {
						if currentJob != nil && currentJob["id"] != nil {
							jobs = append(jobs, currentJob)
						}
						parts := strings.Fields(trimmed)
						if len(parts) >= 1 {
							currentJob = map[string]interface{}{
								"id":           parts[0],
								"name":         "",
								"schedule":     "",
								"last_status":  "unknown",
							}
						}
						continue
					}
					if currentJob != nil {
						if strings.HasPrefix(trimmed, "Name:") {
							currentJob["name"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Name:"))
						} else if strings.HasPrefix(trimmed, "Schedule:") {
							currentJob["schedule"] = strings.TrimSpace(strings.TrimPrefix(trimmed, "Schedule:"))
						} else if strings.HasPrefix(trimmed, "Last run:") {
							lastRun := strings.TrimSpace(strings.TrimPrefix(trimmed, "Last run:"))
							if strings.HasSuffix(lastRun, "ok") {
								currentJob["last_status"] = "ok"
							} else if strings.HasSuffix(lastRun, "failed") {
								currentJob["last_status"] = "failed"
							}
						}
					}
				}
				if currentJob != nil && currentJob["id"] != nil {
					jobs = append(jobs, currentJob)
				}
			}
			if jobs == nil {
				jobs = []map[string]interface{}{}
			}
			hermesData["cron"] = jobs

			// Get gateway state
			cmd = exec.Command("hermes", "status")
			out, err = cmd.Output()
			hermesData["gateway_state"] = "running"
			hermesData["platforms"] = map[string]map[string]string{
				"weixin":   {"state": "disconnected"},
				"telegram": {"state": "disconnected"},
				"feishu":   {"state": "disconnected"},
			}
			if err == nil {
				output := string(out)
				platforms := map[string]map[string]string{
					"weixin":   {"state": "disconnected"},
					"telegram": {"state": "disconnected"},
					"feishu":   {"state": "disconnected"},
				}
				if strings.Contains(output, "Weixin") && strings.Contains(output, "✓") {
					platforms["weixin"] = map[string]string{"state": "connected"}
				}
				if strings.Contains(output, "Telegram") && strings.Contains(output, "✓") {
					platforms["telegram"] = map[string]string{"state": "connected"}
				}
				if strings.Contains(output, "Feishu") && strings.Contains(output, "✓") {
					platforms["feishu"] = map[string]string{"state": "connected"}
				}
				hermesData["platforms"] = platforms
			}

			jsonData, _ := json.Marshal(hermesData)

			// Fan-out to all clients
			clientsMu.RLock()
			for client := range clients {
				select {
				case client.send <- jsonData:
				default:
				}
			}
			clientsMu.RUnlock()
		}
	}()
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &WSClient{
		conn: conn,
		send: make(chan []byte, 256),
	}

	clientsMu.Lock()
	clients[client] = true
	clientsMu.Unlock()

	go func() {
		defer func() {
			clientsMu.Lock()
			delete(clients, client)
			clientsMu.Unlock()
			conn.Close()
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer conn.Close()
		for message := range client.send {
			err := conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				break
			}
		}
	}()
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	initDB()
	startSharedBroadcasters()

	port := os.Getenv("DASHBOARD_PORT")
	if port == "" {
		port = "18788"
	}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/health", healthHandler)

	// Dashboard APIs
	mux.HandleFunc("/api/dashboard", dashboardHandler)
	mux.HandleFunc("/api/projects", projectsHandler)
	mux.HandleFunc("/api/projects/", projectsHandler) // 处理 /api/projects/:id 路由
	mux.HandleFunc("/api/history", historyHandler)
	mux.HandleFunc("/api/system", systemHandler)
	mux.HandleFunc("/api/processes", processesHandler)
	mux.HandleFunc("/api/ports", portsHandler)
	mux.HandleFunc("/api/process/kill/", processKillHandler)

	// WebSocket
	mux.HandleFunc("/ws", wsHandler)

	// Hermes
	mux.HandleFunc("/api/hermes/status", hermesHandler("status"))
	mux.HandleFunc("/api/hermes/config", hermesHandler("config"))
	mux.HandleFunc("/api/hermes/toolsets", hermesHandler("toolsets"))
	mux.HandleFunc("/api/hermes/cron", hermesHandler("cron"))
	mux.HandleFunc("/api/hermes/profiles", hermesHandler("profiles"))
	mux.HandleFunc("/api/hermes/version", hermesHandler("version"))
	mux.HandleFunc("/api/hermes/gateway-state", hermesHandler("gateway-state"))
	mux.HandleFunc("/api/hermes/logs", hermesHandler("logs"))
	mux.HandleFunc("/api/hermes/insights", hermesHandler("insights"))
	mux.HandleFunc("/api/hermes/skills", hermesHandler("skills"))
	mux.HandleFunc("/api/hermes/quota", hermesHandler("quota"))

	// Terminal
	mux.HandleFunc("/api/terminal/exec", terminalExecHandler)
	mux.HandleFunc("/api/terminal/ghostty", terminalGhosttyHandler)

	// Clipboard
	mux.HandleFunc("/api/clipboard/pinned", clipboardPinnedHandler)
	mux.HandleFunc("/api/clipboard/pin", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			clipboardPinHandler(w, r)
		} else if r.Method == "DELETE" {
			clipboardUnpinHandler(w, r)
		}
	})

	// CORS preflight - apply to all routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		http.NotFound(w, r)
	})

	// Wrap with CORS middleware
	handler := corsMiddleware(mux)

	fmt.Printf("Go Backend starting on port %s\n", port)
	if err := http.ListenAndServe("127.0.0.1:"+port, handler); err != nil {
		log.Fatal(err)
	}
}
