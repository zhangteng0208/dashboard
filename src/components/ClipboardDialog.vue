<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
}>();

interface ClipboardItem {
  id: string;
  content: string;
  timestamp: number;
  preview: string;
  pinned?: boolean;
  dbId?: number;
}

const API_BASE = 'http://127.0.0.1:18788/api';
const STORAGE_KEY = 'dashboard_clipboard_history';
const MAX_ITEMS = 50;

const clipboardHistory = ref<ClipboardItem[]>([]);
const pinnedItems = ref<ClipboardItem[]>([]);
const searchQuery = ref('');
const copiedId = ref<string | null>(null);

async function loadPinnedItems() {
  try {
    const res = await fetch(`${API_BASE}/clipboard/pinned`);
    if (res.ok) {
      const data = await res.json();
      pinnedItems.value = (data.items || []).map((item: any) => ({
        id: `pinned-${item.id}`,
        content: item.content,
        timestamp: new Date(item.created_at).getTime(),
        preview: truncate(item.content),
        pinned: true,
        dbId: item.id,
      }));
    }
  } catch (e) {
    console.error('Failed to load pinned items:', e);
  }
}

function loadHistory() {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored) {
    try {
      clipboardHistory.value = JSON.parse(stored);
    } catch {
      clipboardHistory.value = [];
    }
  }
}

function saveHistory() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(clipboardHistory.value));
}

function truncate(text: string, maxLen = 60): string {
  const singleLine = text.replace(/\s+/g, ' ').trim();
  if (singleLine.length <= maxLen) return singleLine;
  return singleLine.slice(0, maxLen) + '...';
}

function formatTime(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}天前`;
  if (hours > 0) return `${hours}小时前`;
  if (minutes > 0) return `${minutes}分钟前`;
  return '刚刚';
}

const filteredHistory = computed(() => {
  if (!searchQuery.value.trim()) return clipboardHistory.value;
  const q = searchQuery.value.toLowerCase();
  return clipboardHistory.value.filter(
    item => item.content.toLowerCase().includes(q)
  );
});

const displayedItems = computed(() => {
  return [...pinnedItems.value, ...filteredHistory.value];
});

async function copyToClipboard(item: ClipboardItem) {
  try {
    await navigator.clipboard.writeText(item.content);
    copiedId.value = item.id;
    setTimeout(() => {
      copiedId.value = null;
    }, 1500);
  } catch (e) {
    console.error('Failed to copy:', e);
  }
}

async function togglePin(item: ClipboardItem) {
  if (item.pinned) {
    // Unpin - use content-based DELETE
    try {
      const res = await fetch(`${API_BASE}/clipboard/pin`, {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: item.content }),
      });
      if (res.ok) {
        pinnedItems.value = pinnedItems.value.filter(i => i.content !== item.content);
      }
    } catch (e) {
      console.error('Failed to unpin:', e);
    }
  } else {
    // Pin
    try {
      const res = await fetch(`${API_BASE}/clipboard/pin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: item.content }),
      });
      if (res.ok) {
        // Reload pinned items to get the correct dbId
        await loadPinnedItems();
      }
    } catch (e) {
      console.error('Failed to pin:', e);
    }
  }
}

async function clearHistory() {
  clipboardHistory.value = [];
  saveHistory();
}

async function pasteFromClipboard() {
  try {
    const text = await navigator.clipboard.readText();
    if (text && text.trim()) {
      const newItem: ClipboardItem = {
        id: Date.now().toString(),
        content: text,
        timestamp: Date.now(),
        preview: truncate(text),
      };
      clipboardHistory.value = [newItem, ...clipboardHistory.value.filter(i => i.content !== text)].slice(0, MAX_ITEMS);
      saveHistory();
    }
  } catch (e) {
    console.error('Failed to read clipboard:', e);
  }
}

function deleteItem(id: string) {
  clipboardHistory.value = clipboardHistory.value.filter(i => i.id !== id);
  saveHistory();
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    emit('close');
  }
}

onMounted(() => {
  loadHistory();
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown);
});

watch(() => props.visible, (val) => {
  if (val) {
    loadHistory();
    loadPinnedItems();
    searchQuery.value = '';
    // Auto refresh clipboard when opening
    pasteFromClipboard();
  }
});
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div v-if="visible" class="clip-overlay" @click.self="emit('close')">
        <div class="clip-dialog">
          <!-- Header -->
          <div class="clip-header">
            <div class="clip-title-group">
              <svg class="clip-title-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>
                <rect x="8" y="2" width="8" height="4" rx="1" ry="1"/>
              </svg>
              <span class="clip-title">粘贴板历史</span>
              <span class="clip-count">{{ pinnedItems.length + clipboardHistory.length }}</span>
            </div>
            <button class="clip-close" @click="emit('close')" aria-label="关闭">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <!-- Search Bar -->
          <div class="clip-search">
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
            <input
              v-model="searchQuery"
              class="search-input"
              placeholder="搜索粘贴历史..."
              type="text"
            />
            <kbd class="search-kbd">ESC</kbd>
          </div>

          <!-- Content -->
          <div class="clip-content">
            <div v-if="displayedItems.length === 0" class="empty-state">
              <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>
                <rect x="8" y="2" width="8" height="4" rx="1" ry="1"/>
              </svg>
              <p>{{ searchQuery ? '没有匹配的历史' : '暂无粘贴历史' }}</p>
              <p class="empty-hint">复制内容后自动记录</p>
            </div>

            <div v-else class="clip-list">
              <!-- Pinned Section -->
              <div v-if="pinnedItems.length > 0 && !searchQuery" class="pinned-section">
                <div class="pinned-header">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M12 2L12 22M12 2L8 6M12 2L16 6"/>
                  </svg>
                  <span>置顶</span>
                </div>
                <TransitionGroup name="list">
                  <div
                    v-for="item in pinnedItems"
                    :key="item.id"
                    class="clip-item pinned"
                    :class="{ copied: copiedId === item.id }"
                    @click="copyToClipboard(item)"
                  >
                    <div class="clip-main">
                      <div class="clip-preview">{{ item.preview }}</div>
                      <div class="clip-meta">
                        <span class="clip-time">{{ formatTime(item.timestamp) }}</span>
                        <span class="clip-length">{{ item.content.length }} 字符</span>
                      </div>
                    </div>
                    <div class="clip-actions" @click.stop>
                      <button
                        class="action-btn pin-btn pinned"
                        @click="togglePin(item)"
                        aria-label="取消置顶"
                      >
                        <svg viewBox="0 0 24 24" fill="currentColor">
                          <path d="M12 2L12 22M12 2L8 6M12 2L16 6"/>
                        </svg>
                      </button>
                      <button
                        class="action-btn copy-btn"
                        :class="{ success: copiedId === item.id }"
                        @click="copyToClipboard(item)"
                        aria-label="复制"
                      >
                        <svg v-if="copiedId !== item.id" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                        </svg>
                        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <polyline points="20 6 9 17 4 12"/>
                        </svg>
                      </button>
                    </div>
                  </div>
                </TransitionGroup>
              </div>

              <!-- History Section -->
              <TransitionGroup name="list">
                <div
                  v-for="item in filteredHistory"
                  :key="item.id"
                  class="clip-item"
                  :class="{ copied: copiedId === item.id }"
                  @click="copyToClipboard(item)"
                >
                  <div class="clip-main">
                    <div class="clip-preview">{{ item.preview }}</div>
                    <div class="clip-meta">
                      <span class="clip-time">{{ formatTime(item.timestamp) }}</span>
                      <span class="clip-length">{{ item.content.length }} 字符</span>
                    </div>
                  </div>
                  <div class="clip-actions" @click.stop>
                    <button
                      class="action-btn pin-btn"
                      @click="togglePin(item)"
                      aria-label="置顶"
                    >
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M12 2L12 22M12 2L8 6M12 2L16 6"/>
                      </svg>
                    </button>
                    <button
                      class="action-btn copy-btn"
                      :class="{ success: copiedId === item.id }"
                      @click="copyToClipboard(item)"
                      aria-label="复制"
                    >
                      <svg v-if="copiedId !== item.id" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
                        <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
                      </svg>
                      <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="20 6 9 17 4 12"/>
                      </svg>
                    </button>
                    <button class="action-btn delete-btn" @click="deleteItem(item.id)" aria-label="删除">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="3 6 5 6 21 6"/>
                        <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                      </svg>
                    </button>
                  </div>
                </div>
              </TransitionGroup>
            </div>
          </div>

          <!-- Footer -->
          <div class="clip-footer">
            <button class="clear-btn" @click="clearHistory" :disabled="clipboardHistory.length === 0">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"/>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              </svg>
              清空历史
            </button>
            <span class="footer-hint">点击内容复制到粘贴板</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500&display=swap');

/* Transitions */
.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.2s ease;
}
.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}

.dialog-enter-active .clip-dialog,
.dialog-leave-active .clip-dialog {
  transition: transform 0.25s ease, opacity 0.2s ease;
}
.dialog-enter-from .clip-dialog,
.dialog-leave-to .clip-dialog {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
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

/* Overlay */
.clip-overlay {
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
.clip-dialog {
  width: 100%;
  max-width: 600px;
  max-height: 70vh;
  background: #0f172a;
  border: 1px solid rgba(148, 163, 184, 0.1);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.05) inset,
    0 25px 50px -12px rgba(0, 0, 0, 0.5),
    0 0 100px -50px rgba(168, 85, 247, 0.15);
}

/* Header */
.clip-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
}

.clip-title-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.clip-title-icon {
  width: 20px;
  height: 20px;
  color: #a855f7;
}

.clip-title {
  font-size: 15px;
  font-weight: 600;
  color: #f8fafc;
  letter-spacing: -0.01em;
}

.clip-count {
  padding: 2px 8px;
  background: rgba(168, 85, 247, 0.15);
  border: 1px solid rgba(168, 85, 247, 0.3);
  border-radius: 10px;
  font-size: 11px;
  font-weight: 600;
  color: #a855f7;
}

.clip-close {
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

.clip-close:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.clip-close svg {
  width: 16px;
  height: 16px;
}

/* Search */
.clip-search {
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

/* Content */
.clip-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.clip-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

/* Pinned Section */
.pinned-section {
  margin-bottom: 8px;
}

.pinned-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  font-size: 11px;
  font-weight: 600;
  color: #a855f7;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pinned-header svg {
  width: 14px;
  height: 14px;
}

/* Clip Item */
.clip-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: rgba(168, 85, 247, 0.03);
  border: 1px solid transparent;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.12s ease;
}

.clip-item.pinned {
  background: rgba(168, 85, 247, 0.08);
  border-color: rgba(168, 85, 247, 0.25);
}

.clip-item:hover {
  background: rgba(168, 85, 247, 0.08);
  border-color: rgba(168, 85, 247, 0.2);
}

.clip-item:hover .clip-actions {
  opacity: 1;
}

.clip-item.copied {
  background: rgba(34, 197, 94, 0.1);
  border-color: rgba(34, 197, 94, 0.3);
}

.clip-main {
  flex: 1;
  min-width: 0;
}

.clip-preview {
  font-size: 13px;
  font-family: 'Fira Code', monospace;
  color: #e2e8f0;
  margin-bottom: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.clip-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 11px;
}

.clip-time {
  color: #64748b;
}

.clip-length {
  color: #475569;
}

/* Actions */
.clip-actions {
  display: flex;
  gap: 6px;
  opacity: 0;
  transition: opacity 0.12s ease;
}

.action-btn {
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
  transition: all 0.12s ease;
}

.action-btn svg {
  width: 16px;
  height: 16px;
}

.copy-btn:hover {
  background: rgba(168, 85, 247, 0.15);
  border-color: rgba(168, 85, 247, 0.3);
  color: #a855f7;
}

.copy-btn.success {
  background: rgba(34, 197, 94, 0.15);
  border-color: rgba(34, 197, 94, 0.3);
  color: #22c55e;
}

.delete-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.3);
  color: #ef4444;
}

.pin-btn:hover {
  background: rgba(168, 85, 247, 0.15);
  border-color: rgba(168, 85, 247, 0.3);
  color: #a855f7;
}

.pin-btn.pinned {
  background: rgba(168, 85, 247, 0.2);
  border-color: rgba(168, 85, 247, 0.4);
  color: #a855f7;
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
  margin-bottom: 4px;
}

.empty-hint {
  font-size: 11px !important;
  color: #475569 !important;
}

/* Footer */
.clip-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: rgba(15, 23, 42, 0.5);
  border-top: 1px solid rgba(148, 163, 184, 0.08);
}

.clear-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 8px;
  color: #ef4444;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.clear-btn:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.2);
  border-color: rgba(239, 68, 68, 0.4);
}

.clear-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.clear-btn svg {
  width: 14px;
  height: 14px;
}

.footer-hint {
  font-size: 11px;
  color: #475569;
}

/* Scrollbar */
.clip-content::-webkit-scrollbar {
  width: 6px;
}

.clip-content::-webkit-scrollbar-track {
  background: transparent;
}

.clip-content::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.15);
  border-radius: 3px;
}

.clip-content::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.25);
}
</style>
