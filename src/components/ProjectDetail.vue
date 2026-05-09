<script setup lang="ts">
/**
 * ProjectDetail.vue - 项目详情抽屉组件 (macOS Native Style)
 */
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const API_BASE = 'http://localhost:18788'

interface Project {
  id: number
  name: string
  path: string
  techStack?: string
  gitBranch?: string
  gitDirty?: boolean
  gitCommit?: string
  gitCommitMsg?: string
  gitCommitAuthor?: string
  gitCommitDate?: string
  alias?: string
  description?: string
  readmeContent?: string
}

const props = defineProps<{
  modelValue: boolean
  project: Project | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'deleted'): void
  (e: 'refresh'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

// Tab 相关的数据
const activeTab = ref('basic')
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
// @ts-ignore showTagDialog 暂时未使用，后续标签功能会用到
const showTagDialog = ref(false)

async function loadProjectDetail() {
  if (!props.project) return
  // 加载标签
  try {
    const tagsRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/tags`)
    const tagsData = await tagsRes.json()
    projectTags.value = tagsData.tags || []
  } catch (e) {
    console.error('加载标签失败:', e)
  }
  // 加载 Scripts
  try {
    const scriptsRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/scripts`)
    const scriptsData = await scriptsRes.json()
    projectScripts.value = scriptsData.scripts || []
    projectFramework.value = scriptsData.framework || ''
  } catch (e) {
    console.error('加载 Scripts 失败:', e)
  }
  // 加载分支
  try {
    const branchesRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/branches`)
    const branchesData = await branchesRes.json()
    branches.value = branchesData.branches || []
  } catch (e) {
    console.error('加载分支失败:', e)
  }
  // 加载 Diff
  try {
    const diffRes = await fetch(`${API_BASE}/api/projects/${props.project.id}/diff`)
    diffInfo.value = await diffRes.json()
  } catch (e) {
    console.error('加载 Diff 失败:', e)
  }
  // 初始化表单
  projectForm.value.alias = props.project.alias || ''
  projectForm.value.description = props.project.description || ''
  // 从 project 中获取 readmeContent（通过 scan API）
  readmeContent.value = props.project.readmeContent || ''
}

async function saveProjectInfo() {
  if (!props.project) return
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
  if (!props.project) return
  await fetch(`${API_BASE}/api/projects/${props.project.id}/scripts/exec`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ command: script.command }),
  })
  ElMessage.success('脚本已提交执行')
}

watch(() => props.project, loadProjectDetail, { immediate: true })

async function openInFinder() {
  if (!props.project) return
  try {
    const res = await fetch(`${API_BASE}/api/projects/open`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: props.project.path, action: 'finder' })
    })
    if (!res.ok) throw new Error('Failed to open in Finder')
  } catch (error) {
    console.error('打开 Finder 失败:', error)
    ElMessage.error('打开失败')
  }
}

async function openInTerminal() {
  if (!props.project) return
  try {
    const res = await fetch(`${API_BASE}/api/projects/open`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: props.project.path, action: 'terminal' })
    })
    if (!res.ok) throw new Error('Failed to open in Terminal')
  } catch (error) {
    console.error('打开终端失败:', error)
    ElMessage.error('打开失败')
  }
}

async function handleScan() {
  if (!props.project) return
  try {
    const res = await fetch(`${API_BASE}/api/projects/${props.project.id}/scan`, {
      method: 'POST'
    })
    if (!res.ok) throw new Error('Failed to scan project')
    ElMessage.success('刷新成功')
    emit('refresh')
  } catch (error) {
    console.error('扫描项目失败:', error)
    ElMessage.error('刷新失败')
  }
}

async function handleDelete() {
  if (!props.project) return
  try {
    await ElMessageBox.confirm(
      `确定要删除项目 "${props.project.name}" 吗？此操作不可恢复。`,
      '删除确认',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const res = await fetch(`${API_BASE}/api/projects/${props.project.id}`, {
      method: 'DELETE'
    })
    if (!res.ok) throw new Error('Failed to delete project')

    ElMessage.success('删除成功')
    emit('deleted')
    visible.value = false
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除项目失败:', error)
      ElMessage.error('删除失败')
    }
  }
}
</script>

<template>
  <el-drawer
    v-model="visible"
    title="项目详情"
    direction="rtl"
    size="420px"
    class="project-detail-drawer el-drawer--dark"
    :body-class="'drawer-dark-body'"
  >
    <div v-if="project" class="detail-content">
      <!-- Project Header -->
      <div class="project-header">
        <div class="project-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </div>
        <div class="project-title">
          <h2 class="project-name">{{ project.name }}</h2>
          <p class="project-path">{{ project.path }}</p>
        </div>
      </div>

      <!-- Tabs Section -->
      <el-tabs v-model="activeTab" class="detail-tabs">
        <!-- Tab 1: 基本信息 -->
        <el-tab-pane label="基本信息" name="basic">
          <div class="tab-content">
            <!-- Tech Stack -->
            <div v-if="project.techStack" class="info-row">
              <span class="info-label">技术栈</span>
              <span class="tech-badge">{{ project.techStack }}</span>
            </div>

            <!-- Git Info Card -->
            <div v-if="project.gitBranch" class="git-card">
              <h3 class="section-title">Git</h3>
              <div class="git-grid">
                <div class="git-item">
                  <span class="git-label">分支</span>
                  <span class="git-value">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="git-icon" aria-hidden="true">
                      <line x1="6" y1="3" x2="6" y2="15"/>
                      <circle cx="18" cy="6" r="3"/>
                      <circle cx="6" cy="18" r="3"/>
                      <path d="M18 9a9 9 0 0 1-9 9"/>
                    </svg>
                    {{ project.gitBranch }}
                  </span>
                </div>
                <div class="git-item" v-if="project.gitCommit">
                  <span class="git-label">Commit</span>
                  <span class="git-value commit">{{ project.gitCommit }}</span>
                </div>
                <div class="git-item" v-if="project.gitCommitMsg">
                  <span class="git-label">信息</span>
                  <span class="git-value message">{{ project.gitCommitMsg }}</span>
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

            <!-- Alias/Description 编辑表单 -->
            <div class="edit-form">
              <h3 class="section-title">项目信息</h3>
              <div class="form-item">
                <label class="form-label">别名</label>
                <el-input
                  v-model="projectForm.alias"
                  placeholder="输入项目别名"
                  class="form-input"
                />
              </div>
              <div class="form-item">
                <label class="form-label">描述</label>
                <el-input
                  v-model="projectForm.description"
                  type="textarea"
                  :rows="2"
                  placeholder="输入项目描述"
                  class="form-input"
                />
              </div>
              <button class="action-btn primary save-btn" @click.stop="saveProjectInfo">
                保存信息
              </button>
            </div>

            <!-- Tags 列表 -->
            <div v-if="projectTags.length > 0" class="tags-section">
              <h3 class="section-title">标签</h3>
              <div class="tags-list">
                <span v-for="tag in projectTags" :key="tag.id" class="tag-item">
                  {{ tag.name }}
                </span>
              </div>
            </div>

            <!-- Scripts 列表 -->
            <div v-if="projectScripts.length > 0" class="scripts-section">
              <h3 class="section-title">脚本</h3>
              <div class="scripts-list">
                <div v-for="script in projectScripts" :key="script.name" class="script-item">
                  <div class="script-info">
                    <span class="script-name">{{ script.name }}</span>
                    <span class="script-command">{{ script.command }}</span>
                  </div>
                  <button class="script-run-btn" @click.stop="runScript(script)">
                    运行
                  </button>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- Tab 2: 分支 -->
        <el-tab-pane label="分支" name="branches">
          <div class="tab-content">
            <div v-if="branches.length > 0" class="branches-list">
              <div v-for="branch in branches" :key="branch.name" class="branch-item">
                <div class="branch-info">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="git-icon" aria-hidden="true">
                    <line x1="6" y1="3" x2="6" y2="15"/>
                    <circle cx="18" cy="6" r="3"/>
                    <circle cx="6" cy="18" r="3"/>
                    <path d="M18 9a9 9 0 0 1-9 9"/>
                  </svg>
                  <span class="branch-name">{{ branch.name }}</span>
                  <span v-if="branch.current" class="branch-current">当前</span>
                </div>
                <span v-if="branch.commit" class="branch-commit">{{ branch.commit }}</span>
              </div>
            </div>
            <div v-else class="empty-state">
              <p>暂无分支信息</p>
            </div>
          </div>
        </el-tab-pane>

        <!-- Tab 3: Diff -->
        <el-tab-pane label="Diff" name="diff">
          <div class="tab-content">
            <div v-if="diffInfo.stats" class="diff-stats">
              <span class="stat additions">+{{ diffInfo.stats.additions }}</span>
              <span class="stat deletions">-{{ diffInfo.stats.deletions }}</span>
            </div>
            <div v-if="diffInfo.changed && diffInfo.changed.length > 0" class="diff-list">
              <div v-for="file in diffInfo.changed" :key="file.path" class="diff-file">
                <div class="diff-file-header">
                  <span class="diff-file-path">{{ file.path }}</span>
                  <span class="diff-file-stats">
                    <span class="additions">+{{ file.additions || 0 }}</span>
                    <span class="deletions">-{{ file.deletions || 0 }}</span>
                  </span>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <p>暂无更改</p>
            </div>
          </div>
        </el-tab-pane>

        <!-- Tab 4: README -->
        <el-tab-pane label="README" name="readme">
          <div class="tab-content">
            <div v-if="readmeContent" class="readme-content">
              <pre class="readme-text">{{ readmeContent }}</pre>
            </div>
            <div v-else class="empty-state">
              <p>暂无 README 内容</p>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>

      <!-- Actions Section -->
      <div class="actions-section">
        <button class="action-btn primary" @click.stop="openInFinder" aria-label="在 Finder 中打开">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M9 3H5a2 2 0 0 0-2 2v4m6-6h10a2 2 0 0 1 2 2v4M9 3v18m0 0h10a2 2 0 0 0 2-2V9M9 21H5a2 2 0 0 1-2-2V9m0 0h18"/>
          </svg>
          在 Finder 打开
        </button>
        <button class="action-btn primary" @click.stop="openInTerminal" aria-label="在终端中打开">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M4 17l6-6-6-6m8 14h8"/>
          </svg>
          在终端打开
        </button>
        <button class="action-btn" @click.stop="handleScan" aria-label="刷新项目信息">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M23 4v6h-6M1 20v-6h6"/>
            <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
          </svg>
          刷新信息
        </button>
        <button class="action-btn danger" @click.stop="handleDelete" aria-label="删除项目">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
            <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
          </svg>
          删除项目
        </button>
      </div>
    </div>
  </el-drawer>
</template>

<style scoped>
.project-detail-drawer {
  --md-bg: #1e1e1e;
  --md-surface: #2d2d2d;
  --md-surface-hover: #3d3d3d;
  --md-border: #444444;
  --md-text: #ffffff;
  --md-text-secondary: #a1a1a6;
  --md-accent: #0a84ff;
  --md-accent-hover: #409cff;
  --md-danger: #ff453a;
  --md-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
  --md-shadow-sm: 0 4px 16px rgba(0, 0, 0, 0.3);
  --md-radius: 12px;
  --md-radius-sm: 8px;
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", "Helvetica Neue", sans-serif;
  /* 覆盖 Element Plus 内部变量 */
  --el-bg-color: #1e1e1e;
  --el-bg-color-overlay: #2d2d2d;
  --el-text-color-primary: #ffffff;
  --el-text-color-regular: #a1a1a6;
  --el-border-color: #444444;
  --el-fill-color-light: #2d2d2d;
}

/* Override Element Plus Drawer */
:deep(.el-drawer) {
  background-color: #1e1e1e !important;
}

:deep(.el-drawer__wrapper) {
  background-color: #1e1e1e !important;
}

:deep(.el-overlay) {
  background-color: rgba(0, 0, 0, 0.6) !important;
}

:deep(.el-drawer__header) {
  padding: 20px 24px 16px !important;
  margin: 0;
  border-bottom: 1px solid #444444 !important;
  background-color: #1e1e1e !important;
}

:deep(.el-drawer__title) {
  font-size: 17px;
  font-weight: 600;
  color: #ffffff !important;
  letter-spacing: -0.02em;
}

:deep(.el-drawer__body) {
  padding: 0 !important;
  background-color: #1e1e1e !important;
  color: #ffffff !important;
}

/* Target body-class prop */
:deep(.drawer-dark-body) {
  background-color: #1e1e1e !important;
  color: #ffffff !important;
}

:deep(.el-drawer.drawer-dark-body) {
  background-color: #1e1e1e !important;
}

/* Content */
.detail-content {
  padding: 24px;
  background: var(--md-bg);
  color: var(--md-text);
  min-height: 100%;
}

/* Project Header */
.project-header {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}

.project-icon {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #5856d6 0%, #007aff 100%);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.project-icon svg {
  width: 28px;
  height: 28px;
  color: white;
}

.project-title {
  flex: 1;
  min-width: 0;
}

.project-name {
  font-size: 20px;
  font-weight: 700;
  color: var(--md-text);
  margin: 0 0 4px;
  letter-spacing: -0.02em;
}

.project-path {
  font-size: 13px;
  color: var(--md-text-secondary);
  margin: 0;
  word-break: break-all;
  line-height: 1.4;
}

/* Tabs Styles */
.detail-tabs {
  margin-top: 8px;
}

:deep(.el-tabs__header) {
  margin: 0;
}

:deep(.el-tabs__nav-wrap) {
  background-color: var(--md-bg);
}

:deep(.el-tabs__nav-wrap::after) {
  background-color: var(--md-border);
}

:deep(.el-tabs__item) {
  color: var(--md-text-secondary);
  font-size: 14px;
  height: 40px;
  line-height: 40px;
  padding: 0 16px;
}

:deep(.el-tabs__item:hover) {
  color: var(--md-text);
}

:deep(.el-tabs__item.is-active) {
  color: var(--md-accent);
}

:deep(.el-tabs__active-bar) {
  background-color: var(--md-accent);
}

:deep(.el-tabs__content) {
  padding: 0;
}

/* Tab Content */
.tab-content {
  padding: 16px 0;
}

/* Info Row */
.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid var(--md-border);
}

.info-label {
  font-size: 14px;
  color: var(--md-text-secondary);
}

.tech-badge {
  font-size: 12px;
  font-weight: 500;
  color: var(--md-accent);
  background: rgba(10, 132, 255, 0.2);
  padding: 4px 10px;
  border-radius: 6px;
}

/* Git Section */
.git-card {
  margin-top: 16px;
}

.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--md-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 12px;
}

.git-grid {
  background: var(--md-surface);
  border-radius: var(--md-radius-sm);
  padding: 4px 0;
}

.git-item {
  display: flex;
  align-items: flex-start;
  padding: 10px 14px;
}

.git-item:not(:last-child) {
  border-bottom: 1px solid var(--md-border);
}

.git-label {
  font-size: 13px;
  color: var(--md-text-secondary);
  width: 50px;
  flex-shrink: 0;
}

.git-value {
  font-size: 13px;
  color: var(--md-text);
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  word-break: break-all;
}

.git-value.commit {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  font-size: 12px;
  color: var(--md-text-secondary);
}

.git-value.message {
  color: var(--md-text-secondary);
}

.git-value.dirty {
  color: #ff9500;
}

.git-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.status-dot {
  width: 8px;
  height: 8px;
  background: #34c759;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.dirty {
  background: #ff9500;
}

/* Edit Form */
.edit-form {
  margin-top: 20px;
}

.form-item {
  margin-bottom: 12px;
}

.form-label {
  display: block;
  font-size: 13px;
  color: var(--md-text-secondary);
  margin-bottom: 6px;
}

.form-input {
  width: 100%;
}

:deep(.form-input .el-input__wrapper) {
  background-color: var(--md-surface);
  box-shadow: none;
  border: 1px solid var(--md-border);
}

:deep(.form-input .el-input__inner) {
  color: var(--md-text);
}

:deep(.form-input .el-textarea__inner) {
  background-color: var(--md-surface);
  box-shadow: none;
  border: 1px solid var(--md-border);
  color: var(--md-text);
  resize: none;
}

.save-btn {
  margin-top: 8px;
  width: 100%;
}

/* Tags Section */
.tags-section {
  margin-top: 20px;
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  font-size: 12px;
  color: var(--md-accent);
  background: rgba(10, 132, 255, 0.15);
  padding: 4px 10px;
  border-radius: 6px;
}

/* Scripts Section */
.scripts-section {
  margin-top: 20px;
}

.scripts-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.script-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--md-surface);
  border-radius: var(--md-radius-sm);
}

.script-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.script-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--md-text);
}

.script-command {
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  color: var(--md-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.script-run-btn {
  padding: 4px 12px;
  font-size: 12px;
  font-weight: 500;
  color: var(--md-accent);
  background: rgba(10, 132, 255, 0.15);
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.15s ease;
  flex-shrink: 0;
}

.script-run-btn:hover {
  background: rgba(10, 132, 255, 0.25);
}

/* Branches List */
.branches-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.branch-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--md-surface);
  border-radius: var(--md-radius-sm);
}

.branch-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.branch-name {
  font-size: 13px;
  color: var(--md-text);
}

.branch-current {
  font-size: 10px;
  color: var(--md-accent);
  background: rgba(10, 132, 255, 0.2);
  padding: 2px 6px;
  border-radius: 4px;
}

.branch-commit {
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  color: var(--md-text-secondary);
}

/* Diff Stats */
.diff-stats {
  display: flex;
  gap: 12px;
  margin-bottom: 12px;
}

.stat {
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  font-weight: 500;
}

.stat.additions {
  color: #34c759;
}

.stat.deletions {
  color: var(--md-danger);
}

/* Diff List */
.diff-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.diff-file {
  background: var(--md-surface);
  border-radius: var(--md-radius-sm);
  overflow: hidden;
}

.diff-file-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
}

.diff-file-path {
  font-size: 12px;
  color: var(--md-text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.diff-file-stats {
  display: flex;
  gap: 8px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  flex-shrink: 0;
}

.diff-file-stats .additions {
  color: #34c759;
}

.diff-file-stats .deletions {
  color: var(--md-danger);
}

/* README Content */
.readme-content {
  background: var(--md-surface);
  border-radius: var(--md-radius-sm);
  padding: 12px;
}

.readme-text {
  font-size: 12px;
  color: var(--md-text);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", monospace;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

/* Empty State */
.empty-state {
  padding: 32px 16px;
  text-align: center;
}

.empty-state p {
  color: var(--md-text-secondary);
  font-size: 13px;
  margin: 0;
}

/* Actions */
.actions-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--md-border);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 16px;
  background: var(--md-surface);
  border: 1px solid var(--md-border);
  border-radius: var(--md-radius-sm);
  font-size: 14px;
  font-weight: 500;
  color: var(--md-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-btn:focus-visible {
  outline: 2px solid var(--md-accent);
  outline-offset: 2px;
}

.action-btn svg {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.action-btn:hover {
  background: var(--md-bg);
  border-color: var(--md-text-secondary);
}

.action-btn:active {
  transform: scale(0.98);
}

.action-btn.primary {
  background: var(--md-accent);
  border-color: var(--md-accent);
  color: white;
}

.action-btn.primary:hover {
  background: var(--md-accent-hover);
  border-color: var(--md-accent-hover);
}

.action-btn.danger {
  color: var(--md-danger);
}

.action-btn.danger:hover {
  background: rgba(255, 59, 48, 0.1);
  border-color: var(--md-danger);
}
</style>

<!-- 非 scoped 样式强制覆盖 Element Plus -->
<style>
.el-drawer.project-detail-drawer {
  background-color: #1e1e1e !important;
}

.el-drawer.project-detail-drawer .el-drawer__header {
  background-color: #1e1e1e !important;
  border-bottom-color: #444444 !important;
}

.el-drawer.project-detail-drawer .el-drawer__title {
  color: #ffffff !important;
}

.el-drawer.project-detail-drawer .el-drawer__body {
  background-color: #1e1e1e !important;
  color: #ffffff !important;
}

/* Element Plus Tabs Dark Mode */
.el-tabs.project-detail-drawer .el-tabs__header {
  background-color: #1e1e1e;
}

.el-tabs.project-detail-drawer .el-tabs__nav-wrap::after {
  background-color: #444444;
}

.el-tabs.project-detail-drawer .el-tabs__item {
  color: #a1a1a6;
}

.el-tabs.project-detail-drawer .el-tabs__item:hover {
  color: #ffffff;
}

.el-tabs.project-detail-drawer .el-tabs__item.is-active {
  color: #0a84ff;
}

.el-tabs.project-detail-drawer .el-tabs__active-bar {
  background-color: #0a84ff;
}

/* Input styles in dark mode */
.el-input.project-detail-drawer .el-input__wrapper {
  background-color: #2d2d2d !important;
  box-shadow: none !important;
  border: 1px solid #444444 !important;
}

.el-input.project-detail-drawer .el-input__inner {
  color: #ffffff !important;
}

.el-textarea.project-detail-drawer .el-textarea__inner {
  background-color: #2d2d2d !important;
  box-shadow: none !important;
  border: 1px solid #444444 !important;
  color: #ffffff !important;
}
</style>