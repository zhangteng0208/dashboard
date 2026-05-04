<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

interface PresetCommand {
  id: string;
  label: string;
  command: string;
}

const STORAGE_KEY = 'dashboard_preset_commands';

const defaultCommands: PresetCommand[] = [
  { id: '1', label: '系统信息', command: 'uname -a && sw_vers' },
  { id: '2', label: '磁盘使用', command: 'df -h' },
  { id: '3', label: '内存状态', command: 'vm_stat' },
  { id: '4', label: 'CPU信息', command: 'sysctl -n machdep.cpu.brand_string' },
  { id: '5', label: '网络连接', command: 'netstat -an | head -20' },
  { id: '6', label: '进程TOP10', command: 'ps aux | head -11' },
  { id: '7', label: '端口占用', command: 'lsof -iTCP -sTCP:LISTEN | head -15' },
  { id: '8', label: '负载情况', command: 'uptime' },
  { id: '9', label: '环境变量', command: 'echo $PATH | tr ":" "\\n"' },
  { id: '10', label: '已安装应用', command: 'ls /Applications | head -20' },
];

const commands = ref<PresetCommand[]>([]);
const editingId = ref<string | null>(null);
const editLabel = ref('');
const editCommand = ref('');
const showAddForm = ref(false);
const newLabel = ref('');
const newCommand = ref('');
const searchQuery = ref('');
const listRef = ref<HTMLElement | null>(null);

function loadCommands() {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) {
    try {
      commands.value = JSON.parse(stored);
    } catch {
      commands.value = [...defaultCommands];
    }
  } else {
    commands.value = [...defaultCommands];
  }
}

function saveCommands() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(commands.value));
}

function filteredCommands() {
  if (!searchQuery.value.trim()) return commands.value;
  const q = searchQuery.value.toLowerCase();
  return commands.value.filter(
    cmd => cmd.label.toLowerCase().includes(q) || cmd.command.toLowerCase().includes(q)
  );
}

function startEdit(cmd: PresetCommand) {
  editingId.value = cmd.id;
  editLabel.value = cmd.label;
  editCommand.value = cmd.command;
  showAddForm.value = false;
}

function cancelEdit() {
  editingId.value = null;
  editLabel.value = '';
  editCommand.value = '';
}

function saveEdit() {
  if (!editLabel.value.trim() || !editCommand.value.trim()) return;

  const idx = commands.value.findIndex(c => c.id === editingId.value);
  if (idx !== -1) {
    commands.value[idx] = {
      id: editingId.value!,
      label: editLabel.value.trim(),
      command: editCommand.value.trim(),
    };
    saveCommands();
  }
  cancelEdit();
}

function deleteCommand(id: string) {
  commands.value = commands.value.filter(c => c.id !== id);
  saveCommands();
}

function startAdd() {
  showAddForm.value = true;
  newLabel.value = '';
  newCommand.value = '';
  editingId.value = null;
  searchQuery.value = '';
  nextTick(() => {
    const input = listRef.value?.querySelector('.new-label-input') as HTMLInputElement;
    input?.focus();
  });
}

function cancelAdd() {
  showAddForm.value = false;
  newLabel.value = '';
  newCommand.value = '';
}

function addCommand() {
  if (!newLabel.value.trim() || !newCommand.value.trim()) return;

  const id = Date.now().toString();
  commands.value.push({
    id,
    label: newLabel.value.trim(),
    command: newCommand.value.trim(),
  });
  saveCommands();
  cancelAdd();
}

async function executeInGhostty(cmd: PresetCommand) {
  try {
    await fetch('http://127.0.0.1:18788/api/terminal/ghostty', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ command: cmd.command }),
    });
    emit('close');
  } catch (e) {
    console.error('Failed to execute:', e);
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close');
  }
}

onMounted(() => {
  loadCommands();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

watch(() => props.visible, (val) => {
  if (val) {
    loadCommands();
    editingId.value = null;
    showAddForm.value = false;
    searchQuery.value = '';
  }
});
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="visible" class="cmd-overlay" @click.self="emit('close')">
        <div class="cmd-dialog">
          <!-- Header -->
          <div class="cmd-header">
            <div class="cmd-title-group">
              <svg class="cmd-title-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <polyline points="4 17 10 11 4 5"/>
                <line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              <span class="cmd-title">快捷命令</span>
            </div>
            <button class="cmd-close" @click="emit('close')" aria-label="关闭">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <!-- Search Bar -->
          <div class="cmd-search">
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="searchQuery"
              class="search-input"
              placeholder="搜索命令..."
              type="text"
            />
            <kbd class="search-kbd">ESC</kbd>
          </div>

          <div class="cmd-body">
            <!-- Command List -->
            <div class="cmd-list" ref="listRef">
              <!-- Add Form -->
              <Transition name="slide">
                <div v-if="showAddForm" class="cmd-item add-form">
                  <div class="form-row">
                    <input
                      v-model="newLabel"
                      class="edit-input new-label-input"
                      placeholder="命令名称"
                      @keydown.enter="addCommand"
                      @keydown.esc="cancelAdd"
                    />
                  </div>
                  <div class="form-row">
                    <textarea
                      v-model="newCommand"
                      class="edit-textarea"
                      placeholder="shell 命令..."
                      rows="2"
                      @keydown.enter.ctrl="addCommand"
                      @keydown.esc="cancelAdd"
                    ></textarea>
                  </div>
                  <div class="form-actions">
                    <span class="form-hint">Ctrl+Enter 保存</span>
                    <div class="form-btns">
                      <button class="btn btn-cancel" @click="cancelAdd">取消</button>
                      <button class="btn btn-save" @click="addCommand">保存</button>
                    </div>
                  </div>
                </div>
              </Transition>

              <!-- Edit Form -->
              <Transition name="slide">
                <div v-if="editingId" class="cmd-item edit-form">
                  <div class="form-row">
                    <input
                      v-model="editLabel"
                      class="edit-input"
                      placeholder="命令名称"
                      @keydown.enter="saveEdit"
                      @keydown.esc="cancelEdit"
                    />
                  </div>
                  <div class="form-row">
                    <textarea
                      v-model="editCommand"
                      class="edit-textarea"
                      placeholder="shell 命令..."
                      rows="2"
                      @keydown.enter.ctrl="saveEdit"
                      @keydown.esc="cancelEdit"
                    ></textarea>
                  </div>
                  <div class="form-actions">
                    <span class="form-hint">Ctrl+Enter 保存</span>
                    <div class="form-btns">
                      <button class="btn btn-cancel" @click="cancelEdit">取消</button>
                      <button class="btn btn-save" @click="saveEdit">保存</button>
                    </div>
                  </div>
                </div>
              </Transition>

              <!-- Command Items -->
              <TransitionGroup name="list" tag="div" class="cmd-items">
                <div
                  v-for="cmd in filteredCommands()"
                  :key="cmd.id"
                  class="cmd-item"
                  :class="{ editing: editingId === cmd.id }"
                  @click="executeInGhostty(cmd)"
                >
                  <template v-if="editingId === cmd.id">
                    <!-- Edit mode handled above -->
                  </template>
                  <template v-else>
                    <div class="cmd-main">
                      <div class="cmd-label">{{ cmd.label }}</div>
                      <div class="cmd-preview">{{ cmd.command }}</div>
                    </div>
                    <div class="cmd-meta">
                      <span class="cmd-enter">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="9 10 4 15 9 20"/>
                          <path d="M20 4v7a4 4 0 0 1-4 4H4"/>
                        </svg>
                        运行
                      </span>
                    </div>
                    <div class="cmd-actions" @click.stop>
                      <button class="action-btn edit-btn" @click="startEdit(cmd)" aria-label="编辑">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                        </svg>
                      </button>
                      <button class="action-btn delete-btn" @click="deleteCommand(cmd.id)" aria-label="删除">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="3 6 5 6 21 6"/>
                          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                      </button>
                    </div>
                  </template>
                </div>
              </TransitionGroup>

              <div v-if="filteredCommands().length === 0 && !showAddForm && !editingId" class="empty-state">
                <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                <p>暂无{{ searchQuery ? '匹配' : '' }}命令</p>
                <button v-if="!searchQuery" class="btn btn-primary" @click="startAdd">添加命令</button>
              </div>
            </div>

            <!-- Sidebar -->
            <div class="cmd-sidebar">
              <button class="add-command-btn" @click="startAdd">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                新增命令
              </button>

              <div class="sidebar-divider"></div>

              <div class="sidebar-tips">
                <h4>使用提示</h4>
                <ul>
                  <li>
                    <kbd>Enter</kbd>
                    <span>运行命令</span>
                  </li>
                  <li>
                    <kbd>Esc</kbd>
                    <span>关闭弹窗</span>
                  </li>
                  <li>
                    <kbd>Ctrl+Enter</kbd>
                    <span>保存编辑</span>
                  </li>
                </ul>
              </div>

              <div class="sidebar-ghostty">
                <div class="ghostty-badge">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <polyline points="4 17 10 11 4 5"/>
                    <line x1="12" y1="19" x2="20" y2="19"/>
                  </svg>
                  Ghostty
                </div>
                <p>命令将在 Ghostty 终端中执行</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600&display=swap');

/* Transitions */
.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.2s ease;
}
.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}

.dialog-enter-active .cmd-dialog,
.dialog-leave-active .cmd-dialog {
  transition: transform 0.25s ease, opacity 0.2s ease;
}
.dialog-enter-from .cmd-dialog,
.dialog-leave-to .cmd-dialog {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.2s ease;
}
.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}

.list-enter-active,
.list-leave-active {
  transition: all 0.15s ease;
}
.list-enter-from,
.list-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}
.list-move {
  transition: transform 0.2s ease;
}

/* Overlay */
.cmd-overlay {
  position: fixed;
  inset: 0;
  background: rgba(2, 6, 23, 0.85);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 20px;
}

/* Dialog */
.cmd-dialog {
  width: 100%;
  max-width: 820px;
  height: 70vh;
  max-height: 600px;
  background: #0f172a;
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.05) inset,
    0 25px 50px -12px rgba(0, 0, 0, 0.5),
    0 0 100px -50px rgba(34, 197, 94, 0.15);
}

/* Header */
.cmd-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
}

.cmd-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cmd-title-icon {
  width: 20px;
  height: 20px;
  color: #22c55e;
}

.cmd-title {
  font-size: 15px;
  font-weight: 600;
  color: #f8fafc;
  letter-spacing: -0.01em;
}

.cmd-close {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(148, 163, 184, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 8px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.15s ease;
}

.cmd-close:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.cmd-close svg {
  width: 16px;
  height: 16px;
}

/* Search */
.cmd-search {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  background: rgba(15, 23, 42, 0.5);
  border-bottom: 1px solid rgba(148, 163, 184, 0.08);
}

.search-icon {
  width: 18px;
  height: 18px;
  color: #475569;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  color: #f8fafc;
  font-size: 14px;
  outline: none;
}

.search-input::placeholder {
  color: #475569;
}

.search-kbd {
  padding: 3px 6px;
  background: rgba(148, 163, 184, 0.08);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 4px;
  color: #64748b;
  font-size: 10px;
  font-family: 'Fira Code', monospace;
}

/* Body */
.cmd-body {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

/* Command List */
.cmd-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px;
}

.cmd-items {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* Command Item */
.cmd-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.12s ease;
}

.cmd-item:hover {
  background: rgba(34, 197, 94, 0.06);
  border-color: rgba(34, 197, 94, 0.15);
}

.cmd-item:hover .cmd-actions {
  opacity: 1;
}

.cmd-item.editing {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.2);
  cursor: default;
}

.cmd-main {
  flex: 1;
  min-width: 0;
}

.cmd-label {
  font-size: 13px;
  font-weight: 500;
  color: #e2e8f0;
  margin-bottom: 3px;
}

.cmd-preview {
  font-size: 11px;
  font-family: 'Fira Code', monospace;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cmd-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.cmd-item:hover .cmd-meta {
  opacity: 1;
}

.cmd-enter {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.2);
  border-radius: 4px;
  color: #22c55e;
  font-size: 10px;
  font-weight: 500;
}

.cmd-enter svg {
  width: 12px;
  height: 12px;
}

/* Actions */
.cmd-actions {
  display: flex;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.action-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(148, 163, 184, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 6px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.12s ease;
}

.action-btn svg {
  width: 14px;
  height: 14px;
}

.action-btn.edit-btn:hover {
  background: rgba(59, 130, 246, 0.15);
  border-color: rgba(59, 130, 246, 0.3);
  color: #60a5fa;
}

.action-btn.delete-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

/* Add/Edit Form */
.add-form,
.edit-form {
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 10px;
  cursor: default;
}

.form-row {
  width: 100%;
}

.edit-input,
.edit-textarea {
  width: 100%;
  padding: 10px 12px;
  background: rgba(2, 6, 23, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 8px;
  color: #f8fafc;
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s ease;
}

.edit-input:focus,
.edit-textarea:focus {
  border-color: rgba(59, 130, 246, 0.5);
}

.edit-input::placeholder,
.edit-textarea::placeholder {
  color: #475569;
}

.edit-textarea {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  resize: vertical;
  min-height: 50px;
}

.form-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.form-hint {
  font-size: 11px;
  color: #475569;
}

.form-btns {
  display: flex;
  gap: 8px;
}

/* Buttons */
.btn {
  padding: 7px 14px;
  border-radius: 7px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.12s ease;
}

.btn-save {
  background: rgba(34, 197, 94, 0.15);
  border: 1px solid rgba(34, 197, 94, 0.3);
  color: #22c55e;
}

.btn-save:hover {
  background: rgba(34, 197, 94, 0.25);
  border-color: rgba(34, 197, 94, 0.5);
}

.btn-cancel {
  background: rgba(148, 163, 184, 0.05);
  border: 1px solid rgba(148, 163, 184, 0.1);
  color: #94a3b8;
}

.btn-cancel:hover {
  background: rgba(148, 163, 184, 0.1);
  color: #e2e8f0;
}

.btn-primary {
  background: rgba(34, 197, 94, 0.15);
  border: 1px solid rgba(34, 197, 94, 0.3);
  color: #22c55e;
}

.btn-primary:hover {
  background: rgba(34, 197, 94, 0.25);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  width: 48px;
  height: 48px;
  color: #334155;
  margin-bottom: 16px;
}

.empty-state p {
  color: #64748b;
  font-size: 13px;
  margin-bottom: 16px;
}

/* Sidebar */
.cmd-sidebar {
  width: 220px;
  padding: 20px 16px;
  background: rgba(15, 23, 42, 0.5);
  border-left: 1px solid rgba(148, 163, 184, 0.08);
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.add-command-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 16px;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.2);
  border-radius: 8px;
  color: #22c55e;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.add-command-btn:hover {
  background: rgba(34, 197, 94, 0.2);
  border-color: rgba(34, 197, 94, 0.4);
}

.add-command-btn svg {
  width: 16px;
  height: 16px;
}

.sidebar-divider {
  height: 1px;
  background: rgba(148, 163, 184, 0.08);
}

.sidebar-tips h4 {
  font-size: 11px;
  font-weight: 600;
  color: #475569;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 12px;
}

.sidebar-tips ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sidebar-tips li {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: #64748b;
}

.sidebar-tips kbd {
  padding: 2px 6px;
  background: rgba(2, 6, 23, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 4px;
  font-family: 'Fira Code', monospace;
  font-size: 10px;
  color: #94a3b8;
}

.sidebar-ghostty {
  margin-top: auto;
  padding: 14px;
  background: rgba(2, 6, 23, 0.5);
  border: 1px solid rgba(148, 163, 184, 0.08);
  border-radius: 10px;
}

.ghostty-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: rgba(34, 197, 94, 0.1);
  border-radius: 6px;
  color: #22c55e;
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
}

.ghostty-badge svg {
  width: 14px;
  height: 14px;
}

.sidebar-ghostty p {
  font-size: 11px;
  color: #475569;
  line-height: 1.4;
}

/* Scrollbar */
.cmd-list::-webkit-scrollbar {
  width: 6px;
}

.cmd-list::-webkit-scrollbar-track {
  background: transparent;
}

.cmd-list::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.15);
  border-radius: 3px;
}

.cmd-list::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.25);
}
</style>
