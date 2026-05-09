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
