<template>
  <el-dialog
    v-model="visible"
    title="项目管理"
    width="80%"
    :close-on-click-modal="false"
  >
    <!-- Header -->
    <div class="dialog-header">
      <el-input v-model="search" placeholder="搜索项目..." clearable style="flex:1" />
      <el-button type="primary" @click="showAddDialog = true">添加项目</el-button>
    </div>

    <!-- Project Grid -->
    <div class="project-grid" v-loading="loading">
      <template v-if="filteredProjects.length > 0">
        <div
          v-for="project in filteredProjects"
          :key="project.id"
          class="project-card"
          @click="openDetail(project)"
        >
          <div class="card-name">{{ project.name }}</div>
          <div class="card-path">{{ project.path }}</div>
          <div class="card-tech" v-if="project.techStack">
            <el-tag size="small">{{ project.techStack }}</el-tag>
          </div>
          <div class="card-git" v-if="project.gitBranch">
            <span>🌿 {{ project.gitBranch }}</span>
            <span v-if="project.gitDirty" class="dirty"> ⚠️ dirty</span>
          </div>
          <div class="card-actions">
            <el-button size="small" link @click.stop="openInFinder(project)">Finder</el-button>
            <el-button size="small" link @click.stop="openInTerminal(project)">终端</el-button>
            <el-button size="small" link @click.stop="refreshProject(project)">刷新</el-button>
            <el-button size="small" link type="danger" @click.stop="deleteProject(project)">删除</el-button>
          </div>
        </div>
      </template>
      <div v-else class="empty-state">
        <div class="empty-icon">📁</div>
        <p>暂无项目</p>
        <p>点击右上角「添加项目」开始</p>
      </div>
    </div>

    <!-- Add Dialog -->
    <el-dialog v-model="showAddDialog" title="添加项目" width="400px" append-to-body>
      <el-input v-model="newPath" placeholder="/path/to/project" />
      <template #footer>
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" @click="handleAdd">添加</el-button>
      </template>
    </el-dialog>
  </el-dialog>

  <!-- Project Detail Drawer -->
  <ProjectDetail
    v-model="drawerVisible"
    :project="selectedProject"
    @deleted="handleDeleted"
    @refresh="loadProjects"
  />
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import ProjectDetail from './ProjectDetail.vue'

const API_BASE = 'http://localhost:18788'

const visible = defineModel<boolean>()
const loading = ref(false)
const projects = ref<any[]>([])
const search = ref('')
const showAddDialog = ref(false)
const newPath = ref('')
const drawerVisible = ref(false)
const selectedProject = ref<any>(null)

const filteredProjects = computed(() => {
  if (!search.value) return projects.value
  const q = search.value.toLowerCase()
  return projects.value.filter(p =>
    p.name.toLowerCase().includes(q) || p.path.toLowerCase().includes(q)
  )
})

async function loadProjects() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/api/projects`)
    const data = await res.json()
    projects.value = data.projects || []
  } finally {
    loading.value = false
  }
}

function openDetail(project: any) {
  selectedProject.value = project
  drawerVisible.value = true
}

async function handleAdd() {
  if (!newPath.value.trim()) return
  try {
    await fetch(`${API_BASE}/api/projects`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: newPath.value })
    })
    ElMessage.success('项目添加成功')
    showAddDialog.value = false
    newPath.value = ''
    loadProjects()
  } catch {
    ElMessage.error('添加失败')
  }
}

async function deleteProject(project: any) {
  try {
    await fetch(`${API_BASE}/api/projects/${project.id}`, { method: 'DELETE' })
    ElMessage.success('项目已删除')
    loadProjects()
  } catch {
    ElMessage.error('删除失败')
  }
}

async function openInFinder(project: any) {
  await fetch(`${API_BASE}/api/projects/open`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: project.path, action: 'finder' })
  })
}

async function openInTerminal(project: any) {
  await fetch(`${API_BASE}/api/projects/open`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: project.path, action: 'terminal' })
  })
}

async function refreshProject(project: any) {
  try {
    await fetch(`${API_BASE}/api/projects/${project.id}/scan`, { method: 'POST' })
    ElMessage.success('扫描完成')
    loadProjects()
  } catch {
    ElMessage.error('扫描失败')
  }
}

function handleDeleted() {
  drawerVisible.value = false
  loadProjects()
}

onMounted(loadProjects)
</script>

<style scoped>
.dialog-header {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  align-items: center;
}
.project-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  max-height: 60vh;
  overflow-y: auto;
  padding-right: 8px;
}
.project-card {
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.project-card:hover {
  background: rgba(30, 41, 59, 0.9);
  border-color: var(--neon-cyan, #00fff9);
  box-shadow: 0 0 10px rgba(0, 255, 249, 0.2);
}
.card-name { font-weight: 600; font-size: 14px; }
.card-path { font-size: 11px; color: #888; word-break: break-all; }
.card-tech { margin-top: 4px; }
.card-git { font-size: 12px; color: #aaa; margin-top: auto; }
.card-git .dirty { color: #e6a23c; }
.card-actions { display: flex; gap: 4px; margin-top: 8px; border-top: 1px solid rgba(255,255,255,0.05); padding-top: 8px; }
.empty-state { text-align: center; padding: 60px 20px; color: #888; grid-column: 1 / -1; }
.empty-icon { font-size: 64px; margin-bottom: 20px; }
</style>
