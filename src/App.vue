<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from "vue";
import DateWeatherCard from "./components/DateWeatherCard.vue";
import TerminalDialog from "./components/TerminalDialog.vue";
import ClipboardDialog from "./components/ClipboardDialog.vue";
import ProjectDialog from "./components/ProjectDialog.vue";
import TagSelector from "./components/TagSelector.vue";

// ===================================
//   PHASE 2a: CHART HOVER STATE
// ===================================
const chartHover = ref<{ x: number; cpu: number; memory: number } | null>(null);
const chartSvgRef = ref<SVGSVGElement | null>(null);

function handleChartHover(event: MouseEvent) {
  const svg = chartSvgRef.value;
  if (!svg || !historyData.value.length) return;
  const rect = svg.getBoundingClientRect();
  const x = (event.clientX - rect.left) / rect.width;
  const dataIndex = Math.round(x * (historyData.value.length - 1));
  const dataPoint = historyData.value[Math.max(0, Math.min(dataIndex, historyData.value.length - 1))];
  if (dataPoint) {
    chartHover.value = {
      x: x * 400,
      cpu: dataPoint.cpu,
      memory: dataPoint.memory
    };
  }
}

function handleChartLeave() {
  chartHover.value = null;
}

// ===================================
//   PHASE 2b: PROGRESS BAR COMPUTED
// ===================================
function getProgressColor(percent: number): string {
  if (percent < 60) return '#00fff9';
  if (percent < 85) return '#ff00ff';
  return '#ff3366';
}

// ===================================
//   PHASE 2c: SORT ANIMATION STATE
// ===================================
const sortAnimationKey = ref(0);
let lastSortKey = '';
function triggerSortAnimation(key: 'cpu' | 'memory' | 'name') {
  if (key !== lastSortKey) {
    sortAnimationKey.value++;
    lastSortKey = key;
  }
}

const backendStatus = ref<any>(null);
const systemInfo = ref<any>(null);
const systemDetails = ref<any>(null);
const processes = ref<any[]>([]);
const ports = ref<any[]>([]);
const selectedPort = ref<any>(null);
const loading = ref(true);
const activeTab = ref<'processes' | 'ports'>('processes');
const processSortKey = ref<'cpu' | 'memory' | 'name'>('cpu');
const hermesStatus = ref<any>(null);
const hermesConfig = ref<any>(null);
const hermesToolsets = ref<any[]>([]);
const hermesCron = ref<any[]>([]);
const hermesGateway = ref<any>(null);
const hermesProfiles = ref<any[]>([]);
const hermesVersion = ref<any>(null);
const selectedSession = ref<any>(null);
const hermesInsights = ref<any>(null);
const hermesSkills = ref<any[]>([]);
const hermesQuota = ref<any>(null);
const hermesPersonalities = ref<string[]>(['helpful', 'concise', 'technical', 'creative', 'teacher', 'kawaii', 'catgirl', 'pirate', 'shakespeare', 'surfer', 'noir', 'uwu', 'philosopher', 'hype']);
const personalityLabels: Record<string, string> = {
  helpful: '助手',
  concise: '简洁',
  technical: '技术',
  creative: '创意',
  teacher: '教师',
  kawaii: '可爱',
  catgirl: '猫娘',
  pirate: '海盗',
  shakespeare: '莎士比亚',
  surfer: '冲浪',
  noir: 'Noir',
  uwu: 'uwu',
  philosopher: '哲人',
  hype: '热情',
};
const selectedPersonality = ref('kawaii');
const historyData = ref<any[]>([]);
const historyTimeRange = ref<'1h' | '6h' | '24h'>('1h');
const processSearch = ref('');
const processDetail = ref<any>(null);
const wsConnected = ref(false);

// ===================================
//   PHASE 3b: VALUE UPDATE FLASH
// ===================================
const cpuUpdated = ref(false);
const memUpdated = ref(false);

watch(() => systemInfo.value?.cpu_percent, (newVal, oldVal) => {
  if (oldVal !== null && newVal !== oldVal) {
    cpuUpdated.value = true;
    setTimeout(() => { cpuUpdated.value = false; }, 400);
  }
});

watch(() => systemInfo.value?.memory_percent, (newVal, oldVal) => {
  if (oldVal !== null && newVal !== oldVal) {
    memUpdated.value = true;
    setTimeout(() => { memUpdated.value = false; }, 400);
  }
});
let ws: WebSocket | null = null;
let wsReconnectTimer: number | null = null;
const personalityDescriptions: Record<string, string> = {
  helpful: '乐于助人，友好亲切的AI助手，始终提供帮助和支持。',
  concise: '简洁直接，抓住重点，避免冗余，让回答简短有力。',
  technical: '专业技术顾问，提供精准、详细的技术信息和解决方案。',
  creative: '跳出思维定式，提供创新想法和富有想象力的解决方案。',
  teacher: '耐心教导，用清晰的方式解释概念，配合实例帮助理解。',
  kawaii: '超级可爱！使用(◕‿◕)、✧、ヮ这样的可爱表情，超级热情洋溢！',
  catgirl: '喵~ 喵星人小娜！说话带"喵"，用 (=^ω^=) 这样的颜文字，活泼好奇！',
  pirate: '嘿吼！你是和海盗船长Hermes在说话！用海盗的方式交谈，记得说"Yo ho ho"！',
  shakespeare: '听令！汝正与William Shakespeare风格的助手对话，优雅、华丽、富有戏剧性！',
  surfer: 'Dude！你在和最酷的AI聊天，超棒的！我们会超级 Chill，冲浪般流畅！',
  noir: '雨点敲打着终端，就像悔恨敲打良心。他们叫我Hermes——在代码的海洋里寻找真相。',
  uwu: 'hewwo！我是你可爱的小帮手uwu~ *蹭蹭你的代码* OwO 什么是这个？让我看看！',
  philosopher: '致敬，智慧追求者。吾乃探讨万物深层意义的助手。让我们审视问题，而不仅是how，更是why。',
  hype: 'YOOO 冲鸭！！！我超级兴奋想帮你！每个问题都超棒，我们一起搞定它！💪🔥',
};
const error = ref("");
const backendUrl = ref("");
const showTerminal = ref(false);
const showClipboard = ref(false);
const showProjectDialog = ref(false);
const showTagDialog = ref(false);
const currentProjectId = ref<number | null>(null);

function handleTagChange() {
  // 刷新项目详情
}

let pollTimer: number | null = null;
let processPollTimer: number | null = null;
let hermesPollTimer: number | null = null;
let historyPollTimer: number | null = null;

const BACKEND_URL = 'http://127.0.0.1:18788';

function clamp(value: number, max: number): number {
  return Math.min(Math.max(value, 0), max);
}

// ===================================
//   PHASE 2c: HEAT MAP COLORS
// ===================================
function getHeatColor(value: number): string {
  if (value < 30) return '#22c55e'; // green - low usage
  if (value < 60) return '#84cc16'; // lime
  if (value < 80) return '#eab308'; // yellow
  if (value < 90) return '#f97316'; // orange
  return '#ef4444'; // red - high usage
}

function getHeatGradient(value: number, type: 'cpu' | 'memory'): string {
  const color = getHeatColor(value);
  if (type === 'cpu') {
    return `linear-gradient(90deg, ${color}cc, ${color}88)`;
  }
  return `linear-gradient(90deg, ${color}cc, ${color}88)`;
}

function getChartPath(data: any[], key: 'cpu' | 'memory', maxVal: number): string {
  if (!data || data.length < 2) return '';
  const w = 400;
  const h = 80;
  const step = w / (data.length - 1);
  const points = data.map((d, i) => {
    const val = key === 'cpu' ? d.cpu : d.memory;
    const y = h - (Math.min(val, maxVal) / maxVal) * h;
    return `${i * step},${y}`;
  });
  return `M0,${h} L${points.join(' L')} L${w},${h} Z`;
}

function formatTokens(tokens: number): string {
  if (tokens >= 1000000) {
    return (tokens / 1000000).toFixed(1) + 'M';
  }
  if (tokens >= 1000) {
    return (tokens / 1000).toFixed(1) + 'K';
  }
  return tokens.toString();
}

const cpuChartPath = computed(() => getChartPath(historyData.value, 'cpu', 100));
const memChartPath = computed(() => getChartPath(historyData.value, 'memory', 100));

const sortedProcesses = computed(() => {
  let sorted = [...processes.value];
  if (processSearch.value) {
    const q = processSearch.value.toLowerCase();
    sorted = sorted.filter(p => (p.name || '').toLowerCase().includes(q));
  }
  // Trigger sort animation
  triggerSortAnimation(processSortKey.value);
  switch (processSortKey.value) {
    case 'cpu':
      return sorted.sort((a, b) => (b.cpu || 0) - (a.cpu || 0));
    case 'memory':
      return sorted.sort((a, b) => (b.memory || 0) - (a.memory || 0));
    case 'name':
      return sorted.sort((a, b) => (a.name || '').localeCompare(b.name || ''));
    default:
      return sorted;
  }
});

function changePersonality() {
  console.log('Personality:', selectedPersonality.value);
}

async function fetchBackendStatus() {
  try {
    // Direct HTTP call to backend instead of Tauri invoke
    const res = await fetch(`${BACKEND_URL}/health`);
    if (res.ok) {
      const data = await res.json();
      backendStatus.value = {
        running: true,
        pid: data.pid,
        port: 18788,
        healthy: true
      };
      backendUrl.value = `${BACKEND_URL}/`;
      return backendStatus.value;
    }
    throw new Error('Backend not healthy');
  } catch (e) {
    backendStatus.value = {
      running: false,
      pid: 0,
      port: 18788,
      healthy: false
    };
    error.value = '';
    return null;
  }
}

async function fetchSystemInfo() {
  if (!backendUrl.value) return;
  try {
    const res = await fetch(`${backendUrl.value}api/dashboard`);
    if (res.ok) {
      systemInfo.value = await res.json();
    }
  } catch (e) {
    console.error("Failed to fetch system info:", e);
  }
}

async function fetchSystemDetails() {
  if (!backendUrl.value) return;
  try {
    const res = await fetch(`${backendUrl.value}api/system`);
    if (res.ok) {
      systemDetails.value = await res.json();
    }
  } catch (e) {
    console.error("Failed to fetch system details:", e);
  }
}

async function fetchProcesses() {
  if (!backendUrl.value) {
    console.log("fetchProcesses: backendUrl is empty, skipping");
    return;
  }
  try {
    const url = `${backendUrl.value}api/processes`;
    console.log("fetchProcesses: fetching", url);
    const res = await fetch(url);
    console.log("fetchProcesses: response status", res.status);
    if (res.ok) {
      const data = await res.json();
      console.log("fetchProcesses: got", data.processes?.length, "processes");
      processes.value = data.processes || [];
    }
  } catch (e) {
    console.error("Failed to fetch processes:", e);
  }
}

async function fetchPorts() {
  if (!backendUrl.value) return;
  try {
    const res = await fetch(`${backendUrl.value}api/ports`);
    if (res.ok) {
      const data = await res.json();
      ports.value = data.ports || [];
    }
  } catch (e) {
    console.error("Failed to fetch ports:", e);
  }
}

function showPortDetail(port: any) {
  selectedPort.value = port;
}

async function showSessionDetail(sessionId: string) {
  try {
    const res = await fetch(`${backendUrl.value}api/hermes/session/${sessionId}`);
    if (res.ok) {
      selectedSession.value = await res.json();
    }
  } catch (e) {
    console.error("Failed to fetch session detail:", e);
  }
}

async function fetchHermesData() {
  if (!backendUrl.value) return;
  try {
    const [statusRes, configRes, toolsetsRes, cronRes, gatewayRes, profilesRes, versionRes, insightsRes, skillsRes, quotaRes] = await Promise.all([
      fetch(`${backendUrl.value}api/hermes/status`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/config`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/toolsets`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/cron`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/gateway-state`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/profiles`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/version`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/insights`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/skills`).catch(() => null),
      fetch(`${backendUrl.value}api/hermes/quota`).catch(() => null),
    ]);

    if (statusRes?.ok) hermesStatus.value = await statusRes.json();
    if (configRes?.ok) {
      hermesConfig.value = await configRes.json();
      // Sync selectedPersonality with display.personality from config
      if (hermesConfig.value?.display?.personality) {
        selectedPersonality.value = hermesConfig.value.display.personality;
      }
    }
    if (toolsetsRes?.ok) {
      const data = await toolsetsRes.json();
      hermesToolsets.value = data.toolsets || [];
    }
    if (cronRes?.ok) {
      const data = await cronRes.json();
      hermesCron.value = data.jobs || [];
    }
    if (gatewayRes?.ok) hermesGateway.value = await gatewayRes.json();
    if (profilesRes?.ok) {
      const data = await profilesRes.json();
      hermesProfiles.value = data.profiles || [];
    }
    if (versionRes?.ok) {
      const data = await versionRes.json();
      hermesVersion.value = data.version || null;
    }
    if (insightsRes?.ok) {
      const data = await insightsRes.json();
      hermesInsights.value = data.insights || null;
    }
    if (skillsRes?.ok) {
      const data = await skillsRes.json();
      hermesSkills.value = data.skills || [];
    }
    if (quotaRes?.ok) {
      const data = await quotaRes.json();
      hermesQuota.value = data.quota || null;
    }
  } catch (e) {
    console.error("Failed to fetch Hermes data:", e);
  }
}

async function restartBackend() {
  loading.value = true;
  error.value = "";
  try {
    // Can't restart via HTTP, but we can try to restart the process
    // For now, just reload the status
    await new Promise(r => setTimeout(r, 1000));
    await fetchBackendStatus();
    await fetchSystemInfo();
    await fetchSystemDetails();
  } catch (e) {
    error.value = `Restart failed: ${e}`;
  } finally {
    loading.value = false;
  }
}

async function poll() {
  const status = await fetchBackendStatus();
  if (status?.healthy) {
    await fetchSystemInfo();
    await fetchSystemDetails();
  }
}

function connectWebSocket() {
  if (!backendUrl.value) return;
  try {
    const wsUrl = backendUrl.value.replace('http', 'ws') + 'ws';
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket connected');
      wsConnected.value = true;
      if (wsReconnectTimer) {
        clearTimeout(wsReconnectTimer);
        wsReconnectTimer = null;
      }
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'dashboard' && msg.data) {
          systemInfo.value = msg.data;
        } else if (msg.type === 'hermes') {
          // Update Hermes state from WebSocket
          if (msg.active_sessions !== undefined) {
            hermesStatus.value = { ...hermesStatus.value, active_sessions: msg.active_sessions };
          }
          if (msg.sessions !== undefined) {
            hermesStatus.value = { ...hermesStatus.value, sessions: msg.sessions };
          }
          if (msg.platforms) {
            hermesGateway.value = { ...hermesGateway.value, platforms: msg.platforms };
          }
          if (msg.toolsets) {
            hermesToolsets.value = msg.toolsets;
          }
          if (msg.cron) {
            hermesCron.value = msg.cron;
          }
        }
      } catch (e) {
        console.error('WebSocket parse error:', e);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      wsConnected.value = false;
      // Reconnect after 3 seconds
      if (!wsReconnectTimer) {
        wsReconnectTimer = window.setTimeout(() => {
          wsReconnectTimer = null;
          connectWebSocket();
        }, 3000);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      ws?.close();
    };
  } catch (e) {
    console.error('WebSocket connection failed:', e);
  }
}

function disconnectWebSocket() {
  if (ws) {
    ws.close();
    ws = null;
  }
  if (wsReconnectTimer) {
    clearTimeout(wsReconnectTimer);
    wsReconnectTimer = null;
  }
}

async function pollProcesses() {
  await fetchProcesses();
  await fetchPorts();
}

async function pollHermes() {
  await fetchHermesData();
}

async function fetchHistory() {
  if (!backendUrl.value) return;
  try {
    const limitMap = { '1h': 60, '6h': 360, '24h': 1440 };
    const limit = limitMap[historyTimeRange.value] || 60;
    const res = await fetch(`${backendUrl.value}api/history?limit=${limit}`);
    if (res.ok) {
      const data = await res.json();
      historyData.value = Array.isArray(data) ? data.reverse() : [];
    }
  } catch (e) {
    console.error("Failed to fetch history:", e);
  }
}

function killProcess(pid: number) {
  if (!backendUrl.value) return;
  fetch(`${backendUrl.value}api/process/kill/${pid}`, { method: 'POST' })
    .then(res => res.json())
    .then(() => fetchProcesses())
    .catch(console.error);
}

function showProcessDetail(proc: any) {
  processDetail.value = proc;
}

onMounted(async () => {
  await poll();
  await pollProcesses();
  await pollHermes();
  await fetchHistory();
  loading.value = false;
  // Start WebSocket for real-time updates
  connectWebSocket();
  // Fallback polling (also provides initial data before WS connects)
  pollTimer = window.setInterval(poll, 30000);
  processPollTimer = window.setInterval(pollProcesses, 30000);
  hermesPollTimer = window.setInterval(pollHermes, 30000);
  historyPollTimer = window.setInterval(fetchHistory, 30000);
});

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer);
  if (processPollTimer) clearInterval(processPollTimer);
  if (hermesPollTimer) clearInterval(hermesPollTimer);
  if (historyPollTimer) clearInterval(historyPollTimer);
  disconnectWebSocket();
});
</script>

<template>
  <div class="dashboard dark">
    <div class="backend-status floating" :class="{ healthy: backendStatus?.healthy, unhealthy: backendStatus && !backendStatus.healthy }">
      <span class="status-dot"></span>
      <span class="status-text">{{ backendStatus?.healthy ? '已连接' : '离线' }}</span>
      <span class="status-pid" v-if="backendStatus?.pid">PID: {{ backendStatus.pid }}</span>
      <button class="status-restart" @click="restartBackend" :disabled="loading" title="重启后端">
        <svg class="restart-icon" :class="{ spinning: loading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M23 4v6h-6"/>
          <path d="M1 20v-6h6"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10"/>
          <path d="M20.49 15a9 9 0 0 1-14.85 3.36L1 14"/>
        </svg>
      </button>
    </div>

    <main class="main">
      <div v-if="error" class="error-banner">
        <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        {{ error }}
      </div>

      <!-- Four Column Layout -->
      <div class="columns">
        <!-- Column 1: Date & Weather & Lunar -->
        <div class="column column-1" :style="{ '--stagger-delay': '0ms', animationDelay: 'var(--stagger-delay)' }">
          <DateWeatherCard />
        </div>

        <!-- Column 2: System Info -->
        <div class="column column-2" :style="{ '--stagger-delay': '100ms', animationDelay: 'var(--stagger-delay)' }">
          <div class="panel">
            <div class="sys-hero-card" v-if="systemDetails">
              <div class="sys-hero-header">
                <div class="sys-hero-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="2" y="3" width="20" height="14" rx="2"/>
                    <path d="M8 21h8M12 17v4"/>
                  </svg>
                </div>
                <div class="sys-hero-title">
                  <div class="sys-hero-name">{{ systemDetails.username }}</div>
                  <div class="sys-hero-sub">{{ systemDetails.os }}</div>
                </div>
                <div class="sys-hero-badge">
                  <span class="badge-dot"></span>
                  <span>在线</span>
                </div>
              </div>
              <div class="sys-hero-stats">
                <div class="stat-item">
                  <div class="stat-icon cpu-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <rect x="4" y="4" width="16" height="16" rx="2"/>
                      <rect x="9" y="9" width="6" height="6"/>
                      <path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>
                    </svg>
                  </div>
                  <div class="stat-info">
                    <div class="stat-label">处理器</div>
                    <div class="stat-value">{{ systemDetails.cpu?.count_physical }} 核 {{ systemDetails.cpu?.count_logical }} 线程</div>
                  </div>
                </div>
                <div class="stat-item">
                  <div class="stat-icon mem-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <rect x="2" y="6" width="20" height="12" rx="2"/>
                      <path d="M6 10v4M10 10v4M14 10v4M18 10v4"/>
                    </svg>
                  </div>
                  <div class="stat-info">
                    <div class="stat-label">内存</div>
                    <div class="stat-value">{{ systemDetails.memory?.used_gb }} / {{ systemDetails.memory?.total_gb }} GB</div>
                  </div>
                  <div class="stat-bar">
                    <div class="stat-bar-fill" :style="{ width: systemDetails.memory?.percent + '%' }"></div>
                  </div>
                </div>
                <div class="stat-item">
                  <div class="stat-icon disk-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <ellipse cx="12" cy="5" rx="9" ry="3"/>
                      <path d="M3 5v6c0 5.5 4 10 9 10s9-4.5 9-10V5"/>
                      <path d="M3 11v6c0 5.5 4 10 9 10s9-4.5 9-10v-6"/>
                    </svg>
                  </div>
                  <div class="stat-info">
                    <div class="stat-label">存储</div>
                    <div class="stat-value">{{ systemDetails.disk?.used_gb }} / {{ systemDetails.disk?.total_gb }} GB</div>
                  </div>
                  <div class="stat-bar">
                    <div class="stat-bar-fill disk" :style="{ width: systemDetails.disk?.percent + '%' }"></div>
                  </div>
                </div>
                <div class="stat-item">
                  <div class="stat-icon net-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <circle cx="12" cy="12" r="10"/>
                      <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
                    </svg>
                  </div>
                  <div class="stat-info">
                    <div class="stat-label">网络</div>
                    <div class="stat-value mono">{{ systemDetails.network?.interfaces?.[0]?.address || 'N/A' }}</div>
                  </div>
                </div>
              </div>
              <div class="sys-hero-metrics" v-if="systemInfo">
                <!-- CPU Ring Gauge -->
                <div class="metric-row">
                  <div class="metric-row-label">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                      <rect x="4" y="4" width="16" height="16" rx="2"/>
                      <rect x="9" y="9" width="6" height="6"/>
                      <path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>
                    </svg>
                    <span>CPU</span>
                  </div>
                  <div class="progress-bar-container">
                    <div class="progress-bar">
                      <div class="progress-bar-fill" :style="{
                        width: systemInfo.cpu_percent + '%',
                        background: getProgressColor(systemInfo.cpu_percent),
                        boxShadow: '0 0 8px ' + getProgressColor(systemInfo.cpu_percent)
                      }"></div>
                    </div>
                    <span class="progress-bar-value" :class="{ 'value-updated': cpuUpdated }" :style="{ color: getProgressColor(systemInfo.cpu_percent) }">{{ systemInfo.cpu_percent }}%</span>
                  </div>
                </div>
                <!-- Memory Ring Gauge -->
                <div class="metric-row">
                  <div class="metric-row-label">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                      <rect x="2" y="6" width="20" height="12" rx="2"/>
                      <path d="M6 10v4M10 10v4M14 10v4M18 10v4"/>
                    </svg>
                    <span>内存</span>
                  </div>
                  <div class="progress-bar-container">
                    <div class="progress-bar">
                      <div class="progress-bar-fill" :style="{
                        width: systemInfo.memory_percent + '%',
                        background: getProgressColor(systemInfo.memory_percent),
                        boxShadow: '0 0 8px ' + getProgressColor(systemInfo.memory_percent)
                      }"></div>
                    </div>
                    <span class="progress-bar-value" :class="{ 'value-updated': memUpdated }" :style="{ color: getProgressColor(systemInfo.memory_percent) }">{{ systemInfo.memory_percent }}%</span>
                  </div>
                </div>
                <!-- Disk Ring Gauge -->
                <div class="metric-row">
                  <div class="metric-row-label">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                      <ellipse cx="12" cy="5" rx="9" ry="3"/>
                      <path d="M3 5v6c0 5.5 4 10 9 10s9-4.5 9-10V5"/>
                      <path d="M3 11v6c0 5.5 4 10 9 10s9-4.5 9-10v-6"/>
                    </svg>
                    <span>硬盘</span>
                  </div>
                  <div class="progress-bar-container">
                    <div class="progress-bar">
                      <div class="progress-bar-fill" :style="{
                        width: systemInfo.disk_percent + '%',
                        background: getProgressColor(systemInfo.disk_percent),
                        boxShadow: '0 0 8px ' + getProgressColor(systemInfo.disk_percent)
                      }"></div>
                    </div>
                    <span class="progress-bar-value" :style="{ color: getProgressColor(systemInfo.disk_percent) }">{{ systemInfo.disk_percent }}%</span>
                  </div>
                </div>
                <div class="metric-row net-row">
                  <div class="net-item up">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12">
                      <path d="M12 19V5M5 12l7-7 7 7"/>
                    </svg>
                    <span class="net-label">上传</span>
                    <div class="net-bar">
                      <div class="net-bar-fill up" :style="{ width: Math.min(systemInfo.net_up / 100, 100) + '%' }"></div>
                    </div>
                    <span class="net-val">{{ Number(systemInfo.net_up).toFixed(2) }} KB/s</span>
                  </div>
                  <div class="net-item down">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="12" height="12">
                      <path d="M12 5v14M19 12l-7 7-7-7"/>
                    </svg>
                    <span class="net-label">下载</span>
                    <div class="net-bar">
                      <div class="net-bar-fill down" :style="{ width: Math.min(systemInfo.net_down / 100, 100) + '%' }"></div>
                    </div>
                    <span class="net-val">{{ Number(systemInfo.net_down).toFixed(2) }} KB/s</span>
                  </div>
                </div>
              </div>
              <!-- Historical Metrics Chart -->
              <div class="sys-hero-history">
                <div class="history-header">
                  <span class="history-title">历史趋势</span>
                  <div class="time-range-btns">
                    <button class="time-btn" :class="{ active: historyTimeRange === '1h' }" @click="historyTimeRange = '1h'; fetchHistory()">1小时</button>
                    <button class="time-btn" :class="{ active: historyTimeRange === '6h' }" @click="historyTimeRange = '6h'; fetchHistory()">6小时</button>
                    <button class="time-btn" :class="{ active: historyTimeRange === '24h' }" @click="historyTimeRange = '24h'; fetchHistory()">24小时</button>
                  </div>
                </div>
                <div class="history-chart" v-if="historyData.length > 0">
                  <div class="chart-svg"
                    @mousemove="handleChartHover"
                    @mouseleave="handleChartLeave">
                    <svg ref="chartSvgRef" viewBox="0 0 400 80" preserveAspectRatio="none">
                      <defs>
                        <!-- CPU Gradient: neon orange to transparent -->
                        <linearGradient id="cpuGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stop-color="#ff6b35" stop-opacity="0.6"/>
                          <stop offset="60%" stop-color="#ff6b35" stop-opacity="0.2"/>
                          <stop offset="100%" stop-color="#ff6b35" stop-opacity="0"/>
                        </linearGradient>
                        <!-- Memory Gradient: neon cyan to transparent -->
                        <linearGradient id="memGrad" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stop-color="#00fff9" stop-opacity="0.5"/>
                          <stop offset="60%" stop-color="#00fff9" stop-opacity="0.15"/>
                          <stop offset="100%" stop-color="#00fff9" stop-opacity="0"/>
                        </linearGradient>
                        <!-- Neon glow filter for CPU line -->
                        <filter id="cpuGlow" x="-50%" y="-50%" width="200%" height="200%">
                          <feGaussianBlur stdDeviation="2" result="blur"/>
                          <feMerge>
                            <feMergeNode in="blur"/>
                            <feMergeNode in="SourceGraphic"/>
                          </feMerge>
                        </filter>
                        <!-- Neon glow filter for Memory line -->
                        <filter id="memGlow" x="-50%" y="-50%" width="200%" height="200%">
                          <feGaussianBlur stdDeviation="2" result="blur"/>
                          <feMerge>
                            <feMergeNode in="blur"/>
                            <feMergeNode in="SourceGraphic"/>
                          </feMerge>
                        </filter>
                      </defs>
                      <!-- CPU fill and line with glow -->
                      <path :d="cpuChartPath" fill="url(#cpuGrad)" stroke="none"/>
                      <path :d="cpuChartPath" fill="none" stroke="#ff6b35" stroke-width="2" opacity="0.9" filter="url(#cpuGlow)"/>
                      <!-- Memory fill and line with glow -->
                      <path :d="memChartPath" fill="url(#memGrad)" stroke="none"/>
                      <path :d="memChartPath" fill="none" stroke="#00fff9" stroke-width="2" opacity="0.9" filter="url(#memGlow)"/>
                      <!-- Hover cursor -->
                      <g v-if="chartHover" class="chart-cursor">
                        <line :x1="chartHover.x" y1="0" :x2="chartHover.x" y2="80" stroke="#ffffff" stroke-width="1" stroke-dasharray="2,2" opacity="0.5"/>
                        <circle :cx="chartHover.x" cy="40" r="4" fill="#ff6b35" filter="url(#cpuGlow)"/>
                        <circle :cx="chartHover.x" cy="40" r="4" fill="#00fff9"/>
                      </g>
                    </svg>
                    <!-- Hover tooltip -->
                    <div v-if="chartHover" class="chart-tooltip" :style="{ left: (chartHover.x / 400 * 100) + '%' }">
                      <div class="tooltip-row cpu">
                        <span class="tooltip-label">CPU</span>
                        <span class="tooltip-value">{{ chartHover.cpu.toFixed(1) }}%</span>
                      </div>
                      <div class="tooltip-row mem">
                        <span class="tooltip-label">内存</span>
                        <span class="tooltip-value">{{ chartHover.memory.toFixed(1) }}%</span>
                      </div>
                    </div>
                  </div>
                  <div class="chart-legend">
                    <span class="legend-item cpu"><span class="legend-dot"></span>CPU</span>
                    <span class="legend-item mem"><span class="legend-dot"></span>内存</span>
                  </div>
                </div>
                <div class="history-empty" v-else>
                  <span>加载历史数据...</span>
                </div>
              </div>
              <div class="sys-hero-footer">
                <div class="footer-item hostname">
                  <span class="footer-label">主机名</span>
                  <span class="footer-value">{{ systemInfo?.hostname || systemDetails?.hostname }}</span>
                </div>
                <div class="footer-divider"></div>
                <div class="footer-item wifi">
                  <span class="footer-label">WiFi</span>
                  <span class="footer-value">{{ systemDetails?.network?.wifi || 'N/A' }}</span>
                </div>
                <div class="footer-divider"></div>
                <div class="footer-item mac">
                  <span class="footer-label">MAC</span>
                  <span class="footer-value">{{ systemDetails?.network?.mac || 'N/A' }}</span>
                </div>
                <div class="footer-divider"></div>
                <div class="footer-item vpn">
                  <span class="footer-label">VPN</span>
                  <span class="footer-value">{{ systemDetails?.network?.vpn_port || 'N/A' }}</span>
                </div>
                <div class="footer-divider"></div>
                <div class="footer-item node">
                  <span class="footer-label">Node</span>
                  <span class="footer-value">{{ systemDetails?.node_version || 'N/A' }}</span>
                </div>
                <div class="footer-divider"></div>
                <div class="footer-item python">
                  <span class="footer-label">Python</span>
                  <span class="footer-value">{{ systemDetails?.python_version || 'N/A' }}</span>
                </div>
              </div>
            </div>
            <div class="panel-loading" v-else>
              <div class="loading-spinner"></div>
              <span>等待后端连接...</span>
            </div>
          </div>
          <!-- Process & Port Panel with Tabs -->
          <div class="panel tab-panel">
            <div class="panel-header">
              <div class="tab-buttons">
                <button class="tab-btn" :class="{ active: activeTab === 'processes' }" @click="activeTab = 'processes'">
                  进程列表
                </button>
                <button class="tab-btn" :class="{ active: activeTab === 'ports' }" @click="activeTab = 'ports'">
                  端口占用
                </button>
              </div>
            </div>
            <div class="panel-content">
              <!-- Process List -->
              <div v-show="activeTab === 'processes'" v-if="processes.length > 0">
                <div class="process-search">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                    <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>
                  </svg>
                  <input type="text" v-model="processSearch" placeholder="搜索进程..." class="search-input"/>
                </div>
                <div class="process-list">
                  <div class="process-header">
                    <button class="sort-btn" :class="{ active: processSortKey === 'name' }" @click="processSortKey = 'name'">进程</button>
                    <span class="proc-pid">PID</span>
                    <button class="sort-btn" :class="{ active: processSortKey === 'cpu' }" @click="processSortKey = 'cpu'">CPU</button>
                    <button class="sort-btn" :class="{ active: processSortKey === 'memory' }" @click="processSortKey = 'memory'">内存</button>
                    <span class="proc-net">网络</span>
                  </div>
                  <div class="process-row" v-for="proc in sortedProcesses.slice(0, 15)" :key="proc.pid + '-' + sortAnimationKey"
                    @contextmenu.prevent="showProcessDetail(proc)"
                    @dblclick="showProcessDetail(proc)"
                    :style="{ animation: 'sortPop 0.3s ease-out' }">
                    <span class="proc-name" :title="proc.name">{{ proc.name }}</span>
                    <span class="proc-pid">{{ proc.pid }}</span>
                    <span class="proc-cpu">
                      <span class="proc-bar-container">
                        <span class="proc-bar proc-bar-cpu" :style="{ width: clamp(proc.cpu, 100) + '%', background: getHeatGradient(proc.cpu, 'cpu') }"></span>
                      </span>
                      <span class="proc-value" :style="{ color: getHeatColor(proc.cpu) }">{{ (proc.cpu || 0).toFixed(1) }}%</span>
                    </span>
                    <span class="proc-mem">
                      <span class="proc-bar-container">
                        <span class="proc-bar proc-bar-mem" :style="{ width: Math.min(proc.memory || 0, 100) + '%', background: getHeatGradient(proc.memory, 'memory') }"></span>
                      </span>
                      <span class="proc-value" :style="{ color: getHeatColor(proc.memory) }">{{ (proc.memory || 0).toFixed(1) }}%</span>
                    </span>
                  </div>
                </div>
              </div>
              <!-- Port List -->
              <div v-show="activeTab === 'ports'" v-if="ports.length > 0">
                <div class="port-flow">
                  <div class="port-row" v-for="p in ports" :key="p.port" @click="showPortDetail(p)">
                    <span class="port-indicator"></span>
                    <span class="port-num">{{ p.port }}</span>
                    <span class="port-name">{{ p.name }}</span>
                    <span class="port-pid">{{ p.pid }}</span>
                  </div>
                </div>
              </div>
              <!-- Port Detail Popup -->
              <div class="port-detail-popup" v-if="selectedPort" @click.self="selectedPort = null">
                <div class="port-detail-content">
                  <div class="detail-header">
                    <span class="detail-port">{{ selectedPort.port }}</span>
                    <span class="detail-name">{{ selectedPort.name }}</span>
                    <button class="detail-close" @click="selectedPort = null">×</button>
                  </div>
                  <div class="detail-grid">
                    <div class="detail-item">
                      <span class="detail-label">PID</span>
                      <span class="detail-value neon-cyan">{{ selectedPort.pid }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">协议</span>
                      <span class="detail-value">{{ selectedPort.protocol || 'TCP' }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">状态</span>
                      <span class="detail-value status">{{ selectedPort.status || 'LISTEN' }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">地址</span>
                      <span class="detail-value">{{ selectedPort.address }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">用户</span>
                      <span class="detail-value">{{ selectedPort.user }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">CPU</span>
                      <span class="detail-value neon-magenta">{{ selectedPort.cpu }}%</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">内存</span>
                      <span class="detail-value neon-purple">{{ selectedPort.mem }}%</span>
                    </div>
                  </div>
                  <div class="detail-path" v-if="selectedPort.path">
                    <span class="detail-label">进程路径</span>
                    <span class="detail-path-value">{{ selectedPort.path }}</span>
                  </div>
                </div>
              </div>
              <div class="panel-loading" v-if="(activeTab === 'processes' && processes.length === 0) || (activeTab === 'ports' && ports.length === 0)">
                <div class="loading-spinner"></div>
                <span>加载中...</span>
              </div>
            </div>
            <!-- Process Detail Popup -->
            <div class="process-detail-popup" v-if="processDetail" @click.self="processDetail = null">
              <div class="popup-content">
                <div class="popup-header">
                  <span class="popup-title">进程详情</span>
                  <button class="popup-close" @click="processDetail = null">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>
                  </button>
                </div>
                <div class="popup-body">
                  <div class="detail-row">
                    <span class="detail-label">名称</span>
                    <span class="detail-value">{{ processDetail.name }}</span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">PID</span>
                    <span class="detail-value mono">{{ processDetail.pid }}</span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">CPU</span>
                    <span class="detail-value">{{ (processDetail.cpu || 0).toFixed(1) }}%</span>
                  </div>
                  <div class="detail-row">
                    <span class="detail-label">内存</span>
                    <span class="detail-value">{{ (processDetail.memory || 0).toFixed(1) }}%</span>
                  </div>
                  <div class="detail-row" v-if="processDetail.username">
                    <span class="detail-label">用户</span>
                    <span class="detail-value">{{ processDetail.username }}</span>
                  </div>
                </div>
                <div class="popup-actions">
                  <button class="popup-btn kill" @click="killProcess(processDetail.pid); processDetail = null">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                    结束进程
                  </button>
                  <button class="popup-btn close-btn" @click="processDetail = null">关闭</button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Column 3: Hermes -->
        <div class="column column-3" :style="{ '--stagger-delay': '200ms', animationDelay: 'var(--stagger-delay)' }">
          <div class="hermes-card">
            <!-- Header -->
            <div class="hermes-header">
              <div class="hermes-title-row">
                <div class="hermes-logo">
                  <img src="https://ts2.tc.mm.bing.net/th/id/OIP-C.NwwgQsf0tXI10nxqyBTBFAHaHh?rs=1&pid=ImgDetMain&o=7&rm=3" alt="Hermes" />
                </div>
                <div class="hermes-title-text">
                  <h2>HERMES</h2>
                  <span class="hermes-subtitle">{{ hermesVersion || 'Agent Gateway' }}</span>
                </div>
              </div>
              <div class="hermes-status-badge" :class="{ running: hermesGateway?.gateway_state === 'running' }">
                <span class="status-ring"></span>
                <span class="status-text">{{ hermesGateway?.gateway_state === 'running' ? 'ONLINE' : 'OFFLINE' }}</span>
              </div>
            </div>

            <!-- Quick Stats Bar -->
            <div class="hermes-stats-bar">
              <div class="stat-block">
                <span class="stat-val neon-cyan">{{ hermesStatus?.active_sessions || 0 }}</span>
                <span class="stat-key">会话</span>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-block">
                <span class="stat-val neon-magenta">{{ hermesProfiles.length || 0 }}</span>
                <span class="stat-key">配置</span>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-block">
                <span class="stat-val neon-purple">{{ hermesToolsets.length || 0 }}</span>
                <span class="stat-key">工具</span>
              </div>
              <div class="stat-divider"></div>
              <div class="stat-block">
                <span class="stat-val neon-blue">{{ hermesCron.length || 0 }}</span>
                <span class="stat-key">定时</span>
              </div>
            </div>

            <div class="hermes-body">
              <!-- Platforms -->
              <div class="hermes-block platforms-block">
                <div class="block-header">
                  <span class="block-title">平台连接</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="platforms-grid">
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.weixin?.state === 'connected' }">
                    <div class="platform-icon weixin">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M8.5 11a1.5 1.5 0 100-3 1.5 1.5 0 000 3zm5 0a1.5 1.5 0 100-3 1.5 1.5 0 000 3zM12 2C6.477 2 2 6.477 2 12c0 1.89.525 3.66 1.438 5.168L2 22l4.832-1.438A9.955 9.955 0 0012 22c5.523 0 10-4.477 10-10S17.523 2 12 2z"/></svg>
                    </div>
                    <span class="platform-name">微信</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.weixin?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.telegram?.state === 'connected' }">
                    <div class="platform-icon telegram">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69a.2.2 0 00-.05-.18c-.06-.05-.14-.03-.21-.02-.09.02-1.49.95-4.22 2.79-.4.27-.76.41-1.08.4-.36-.01-1.04-.2-1.55-.37-.63-.2-1.12-.31-1.08-.66.02-.18.27-.36.74-.55 2.92-1.27 4.86-2.11 5.83-2.51 2.78-1.16 3.35-1.36 3.73-1.36.08 0 .27.02.39.12.1.08.13.19.14.27-.01.06.01.24 0 .38z"/></svg>
                    </div>
                    <span class="platform-name">Telegram</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.telegram?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.feishu?.state === 'connected' }">
                    <div class="platform-icon feishu">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M9.5 3C6.46 3 4 5.46 4 8.5v7C4 18.54 6.46 21 9.5 21h5c3.04 0 5.5-2.46 5.5-5.5v-7c0-3.04-2.46-5.5-5.5-5.5h-5zm.62 4.06c.96-.27 2-.43 3.12-.43 1.12 0 2.16.16 3.12.43l-3.12 3.67-3.12-3.67zm-1.62 5.56c0 .39.08.76.22 1.1L10 15.28l2.28-1.56c.14-.34.22-.71.22-1.1V9.06L9.5 6.28 6.5 9.06v3.56z"/></svg>
                    </div>
                    <span class="platform-name">飞书</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.feishu?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.discord?.state === 'connected' }">
                    <div class="platform-icon discord">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/></svg>
                    </div>
                    <span class="platform-name">Discord</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.discord?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.slack?.state === 'connected' }">
                    <div class="platform-icon slack">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M5.042 15.165a2.528 2.528 0 01-2.52 2.523A2.528 2.528 0 010 15.165a2.527 2.527 0 012.522-2.52h2.52v2.52zm1.271 0a2.527 2.527 0 012.521-2.52 2.527 2.527 0 012.521 2.52v6.313A2.528 2.528 0 018.834 24a2.528 2.528 0 01-2.521-2.522v-6.313zM8.834 5.042a2.528 2.528 0 01-2.521-2.52A2.528 2.528 0 018.834 0a2.528 2.528 0 012.521 2.522v2.52H8.834zm0 1.271a2.528 2.528 0 012.521 2.521 2.528 2.528 0 01-2.521 2.521H2.522A2.528 2.528 0 010 8.834a2.528 2.528 0 012.522-2.521h6.312zm10.122 2.521a2.528 2.528 0 012.522-2.521A2.528 2.528 0 0124 8.834a2.528 2.528 0 01-2.522 2.521h-2.522V8.834zm-1.268 0a2.528 2.528 0 01-2.523 2.521 2.527 2.527 0 01-2.52-2.521V2.522A2.527 2.527 0 0115.165 0a2.528 2.528 0 012.523 2.522v6.312zm-2.523 10.122a2.528 2.528 0 012.523 2.522A2.528 2.528 0 0115.165 24a2.527 2.527 0 01-2.52-2.522v-2.522h2.52zm0-1.268a2.527 2.527 0 01-2.52-2.523 2.526 2.526 0 012.52-2.52h6.313A2.527 2.527 0 0124 15.165a2.528 2.528 0 01-2.522 2.523h-6.313z"/></svg>
                    </div>
                    <span class="platform-name">Slack</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.slack?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.whatsapp?.state === 'connected' }">
                    <div class="platform-icon whatsapp">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/></svg>
                    </div>
                    <span class="platform-name">WhatsApp</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.whatsapp?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                  <div class="platform-item" :class="{ connected: hermesGateway?.platforms?.matrix?.state === 'connected' }">
                    <div class="platform-icon matrix">
                      <svg viewBox="0 0 24 24" fill="currentColor"><path d="M1.22 8.36c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm21.56 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm-10.78 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm5.39 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm-5.39 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zM12.76 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74V.74c0-.41.33-.74.74-.74zm5.39 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74V.74c0-.41.33-.74.74-.74zm-10.78 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74V.74c0-.41.33-.74.74-.74zm-5.39 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74V.74c0-.41.33-.74.74-.74zm16.17 15.24c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm-10.78 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74zm-5.39 0c.41 0 .74.33.74.74v5.8c0 .41-.33.74-.74.74s-.74-.33-.74-.74v-5.8c0-.41.33-.74.74-.74z"/></svg>
                    </div>
                    <span class="platform-name">Matrix</span>
                    <span class="platform-status">{{ hermesGateway?.platforms?.matrix?.state === 'connected' ? '已连接' : '离线' }}</span>
                  </div>
                </div>
              </div>

              <!-- Insights Block -->
              <div class="hermes-block insights-block" v-if="hermesInsights && hermesInsights.total_tokens">
                <div class="block-header">
                  <span class="block-title">使用统计</span>
                  <span class="block-count">今日</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="insights-grid">
                  <div class="insight-item">
                    <span class="insight-val neon-cyan">{{ formatTokens(hermesInsights.total_tokens) }}</span>
                    <span class="insight-label">Tokens</span>
                  </div>
                  <div class="insight-item">
                    <span class="insight-val neon-magenta">{{ hermesInsights.messages || 0 }}</span>
                    <span class="insight-label">消息</span>
                  </div>
                  <div class="insight-item">
                    <span class="insight-val neon-purple">{{ hermesInsights.tool_calls || 0 }}</span>
                    <span class="insight-label">工具调用</span>
                  </div>
                </div>
              </div>

              <!-- Skills Block -->
              <div class="hermes-block skills-block" v-if="hermesSkills && hermesSkills.length > 0">
                <div class="block-header">
                  <span class="block-title">Skills</span>
                  <span class="block-count">{{ hermesSkills.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="skills-grid">
                  <div class="skill-item" v-for="skill in hermesSkills.slice(0, 12)" :key="skill.name">
                    <div class="skill-icon">{{ skill.icon || skill.name?.charAt(0).toUpperCase() }}</div>
                    <span class="skill-name">{{ skill.name }}</span>
                  </div>
                </div>
              </div>

              <!-- Profiles -->
              <div class="hermes-block profiles-block" v-if="hermesProfiles.length > 0">
                <div class="block-header">
                  <span class="block-title">Profiles</span>
                  <span class="block-count">{{ hermesProfiles.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="profiles-grid">
                  <div class="profile-card" v-for="profile in hermesProfiles" :key="profile.name">
                    <div class="profile-header">
                      <div class="profile-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                          <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                          <circle cx="12" cy="7" r="4"/>
                        </svg>
                      </div>
                      <div class="profile-info">
                        <div class="profile-name">{{ profile.name }}</div>
                        <div class="profile-model">{{ profile.model?.split('/').pop() || 'N/A' }}</div>
                      </div>
                    </div>
                    <div class="profile-stats">
                      <div class="profile-stat">
                        <span class="stat-num">{{ profile.session_count }}</span>
                        <span class="stat-label">会话</span>
                      </div>
                      <div class="profile-stat">
                        <span class="stat-num">{{ profile.toolsets?.length || 0 }}</span>
                        <span class="stat-label">工具</span>
                      </div>
                    </div>
                    <div class="profile-soul" v-if="profile.soul">{{ profile.soul.slice(0, 80) }}...</div>
                  </div>
                </div>
              </div>

              <!-- Active Sessions -->
              <div class="hermes-block sessions-block" v-if="hermesStatus?.sessions?.length > 0">
                <div class="block-header">
                  <span class="block-title">活跃会话</span>
                  <span class="block-count">{{ hermesStatus.sessions.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="sessions-list">
                  <div class="session-item" v-for="session in hermesStatus.sessions.slice(0, 4)" :key="session.id" @click="showSessionDetail(session.id)">
                    <div class="session-avatar">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                        <circle cx="12" cy="7" r="4"/>
                      </svg>
                    </div>
                    <div class="session-info">
                      <span class="session-title">{{ session.title || '会话' }}</span>
                      <span class="session-id">{{ session.id?.slice(0, 12) }}...</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Session Detail Popup -->
              <div class="session-detail-popup" v-if="selectedSession" @click.self="selectedSession = null">
                <div class="session-detail-content">
                  <div class="session-detail-header">
                    <div class="session-detail-title">{{ selectedSession.title || '会话详情' }}</div>
                    <button class="session-detail-close" @click="selectedSession = null">×</button>
                  </div>
                  <div class="session-messages" v-if="selectedSession.messages?.length > 0">
                    <div class="message-item" v-for="(msg, idx) in selectedSession.messages" :key="idx" :class="msg.role">
                      <div class="message-role">{{ msg.role === 'user' ? '用户' : '助手' }}</div>
                      <div class="message-content">{{ msg.content }}</div>
                    </div>
                  </div>
                  <div class="session-empty" v-else>
                    <span>暂无消息记录</span>
                  </div>
                </div>
              </div>

              <!-- Model Config -->
              <div class="hermes-block model-block" v-if="hermesConfig?.default_model">
                <div class="block-header">
                  <span class="block-title">模型配置</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="model-main">
                  <div class="model-icon">
                    <img src="https://avatars.githubusercontent.com/u/194880281?v=4" alt="MiniMax" />
                  </div>
                  <div class="model-info">
                    <div class="model-name">{{ hermesConfig?.default_model?.split('/').pop() || 'N/A' }}</div>
                    <div class="model-provider">
                      <span class="provider-badge">{{ hermesConfig?.model?.provider || 'N/A' }}</span>
                    </div>
                  </div>
                </div>
                <div class="model-details">
                  <div class="model-detail-item">
                    <span class="detail-label">Endpoint</span>
                    <span class="detail-value url">{{ hermesConfig?.model?.base_url || 'N/A' }}</span>
                  </div>
                  <div class="model-detail-item">
                    <span class="detail-label">Temperature</span>
                    <span class="detail-value">{{ hermesConfig?.model?.temperature ?? 'default' }}</span>
                  </div>
                  <div class="model-detail-item">
                    <span class="detail-label">Max Tokens</span>
                    <span class="detail-value">{{ hermesConfig?.model?.max_tokens || '8K' }}</span>
                  </div>
                  <div class="model-detail-item">
                    <span class="detail-label">Reasoning</span>
                    <span class="detail-value" :class="hermesConfig?.agent?.reasoning_effort">
                      {{ hermesConfig?.agent?.reasoning_effort || 'medium' }}
                    </span>
                  </div>
                </div>
                <!-- Quota Usage -->
                <div class="quota-section" v-if="hermesQuota?.model_remains && hermesQuota.model_remains.length > 0">
                  <div class="quota-header">用量配额</div>
                  <div class="quota-list">
                    <div class="quota-item" v-for="item in hermesQuota.model_remains.slice(0, 6)" :key="item.model_name">
                      <div class="quota-model">
                        <span class="quota-name">{{ item.model_name }}</span>
                        <span class="quota-usage">{{ item.current_interval_usage_count }} / {{ item.current_interval_total_count }}</span>
                      </div>
                      <div class="quota-bar">
                        <div class="quota-fill" :style="{ width: Math.min((item.current_interval_usage_count / item.current_interval_total_count) * 100, 100) + '%' }"></div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Personality Selector -->
              <div class="hermes-block personality-block">
                <div class="block-header">
                  <span class="block-title">人格</span>
                  <span class="block-count">{{ hermesPersonalities.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="personality-grid">
                  <button
                    v-for="p in hermesPersonalities"
                    :key="p"
                    class="personality-chip"
                    :class="{ active: selectedPersonality === p }"
                    @click="selectedPersonality = p"
                  >
                    <span class="chip-icon">
                      <svg v-if="p === 'kawaii'" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z"/><circle cx="8" cy="9" r="1.5"/><circle cx="16" cy="9" r="1.5"/><path d="M12 17c-1.5 0-2.5-.5-3-1.5l1-1 1 1c-.5 1-1.5 1.5-3 1.5"/></svg>
                      <svg v-else-if="p === 'catgirl'" viewBox="0 0 24 24" fill="currentColor"><path d="M12 3L4 9v12h16V9l-8-6zm0 2.5L18 10v9H6v-9l6-4.5z"/><circle cx="9" cy="10" r="1.5"/><circle cx="15" cy="10" r="1.5"/><path d="M12 17c-1 0-2-.5-2.5-1.5l.5-.5.5.5c-.5 1-1.5 1.5-2.5 1.5"/></svg>
                      <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/></svg>
                    </span>
                    <span class="chip-name">{{ personalityLabels[p] || p }}</span>
                  </button>
                </div>
                <div class="personality-preview-card">
                  <div class="preview-header">
                    <span class="preview-label">当前人格</span>
                    <span class="preview-name">{{ personalityLabels[selectedPersonality] || selectedPersonality }}</span>
                  </div>
                  <div class="preview-quote">
                    "{{ personalityDescriptions[selectedPersonality] }}"
                  </div>
                  <div class="preview-actions">
                    <button class="preview-btn activate" @click="changePersonality">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v4m0 12v4M4.93 4.93l2.83 2.83m8.48 8.48l2.83 2.83M2 12h4m12 0h4M4.93 19.07l2.83-2.83m8.48-8.48l2.83-2.83"/></svg>
                      激活
                    </button>
                  </div>
                </div>
              </div>

              <!-- Agent Settings -->
              <div class="hermes-block settings-block" v-if="hermesConfig?.agent">
                <div class="block-header">
                  <span class="block-title">Agent</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="settings-list">
                  <div class="setting-row">
                    <span class="setting-key">max_iterations</span>
                    <span class="setting-val">{{ hermesConfig?.agent?.max_iterations || 50 }}</span>
                  </div>
                  <div class="setting-row">
                    <span class="setting-key">timeout</span>
                    <span class="setting-val">{{ hermesConfig?.agent?.timeout || 180 }}s</span>
                  </div>
                  <div class="setting-row">
                    <span class="setting-key">chain_of_thought</span>
                    <span class="setting-val toggle" :class="{ on: hermesConfig?.agent?.enable_chain_of_thought }">
                      {{ hermesConfig?.agent?.enable_chain_of_thought ? '[ON]' : '[OFF]' }}
                    </span>
                  </div>
                  <div class="setting-row">
                    <span class="setting-key">reflection</span>
                    <span class="setting-val toggle" :class="{ on: hermesConfig?.agent?.enable_reflection }">
                      {{ hermesConfig?.agent?.enable_reflection ? '[ON]' : '[OFF]' }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Safety -->
              <div class="hermes-block safety-block" v-if="hermesConfig?.approvals">
                <div class="block-header">
                  <span class="block-title">安全审批</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="safety-grid">
                  <div class="safety-item" :class="hermesConfig?.approvals?.dangerous_commands || 'ask'">
                    <span class="safety-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
                    </span>
                    <span class="safety-label">危险命令</span>
                    <span class="safety-mode">{{ hermesConfig?.approvals?.dangerous_commands || 'manual' }}</span>
                  </div>
                  <div class="safety-item" :class="hermesConfig?.approvals?.file_write || 'ask'">
                    <span class="safety-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
                    </span>
                    <span class="safety-label">文件写入</span>
                    <span class="safety-mode">{{ hermesConfig?.approvals?.file_write || 'manual' }}</span>
                  </div>
                  <div class="safety-item" :class="hermesConfig?.approvals?.network_request || 'ask'">
                    <span class="safety-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                    </span>
                    <span class="safety-label">网络请求</span>
                    <span class="safety-mode">{{ hermesConfig?.approvals?.network_request || 'manual' }}</span>
                  </div>
                  <div class="safety-item" :class="hermesConfig?.approvals?.execute_command || 'ask'">
                    <span class="safety-icon">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
                    </span>
                    <span class="safety-label">命令执行</span>
                    <span class="safety-mode">{{ hermesConfig?.approvals?.execute_command || 'manual' }}</span>
                  </div>
                </div>
              </div>

              <!-- Toolsets -->
              <div class="hermes-block toolsets-block" v-if="hermesToolsets.length > 0">
                <div class="block-header">
                  <span class="block-title">工具集</span>
                  <span class="block-count">{{ hermesToolsets.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="toolsets-scroll">
                  <span class="toolset-tag" v-for="ts in hermesToolsets.slice(0, 12)" :key="ts.name">
                    {{ ts.name }}
                  </span>
                </div>
              </div>

              <!-- Cron Jobs -->
              <div class="hermes-block cron-block" v-if="hermesCron.length > 0">
                <div class="block-header">
                  <span class="block-title">定时任务</span>
                  <span class="block-count">{{ hermesCron.length }}</span>
                  <div class="block-decoration"></div>
                </div>
                <div class="cron-list-compact">
                  <div class="cron-item-compact" v-for="job in hermesCron.slice(0, 3)" :key="job.id">
                    <span class="cron-status-dot" :class="job.last_status"></span>
                    <span class="cron-name-compact">{{ job.name }}</span>
                    <span class="cron-schedule-compact">{{ job.schedule }}</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Footer -->
            <div class="hermes-footer">
              <span class="footer-timestamp">updated: {{ new Date().toLocaleTimeString() }}</span>
              <span class="footer-cmd">hermes@gateway:~$</span>
            </div>
          </div>
        </div>

        <!-- Column 4: Quick Actions -->
        <div class="column column-4" :style="{ '--stagger-delay': '300ms', animationDelay: 'var(--stagger-delay)' }">
          <div class="quick-actions-panel">
            <button class="quick-action-btn" @click="showTerminal = true" title="终端">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="4 17 10 11 4 5"/>
                <line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              <span class="quick-action-label">终端</span>
            </button>
            <button class="quick-action-btn" @click="showClipboard = true" title="粘贴板历史">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>
                <rect x="8" y="2" width="8" height="4" rx="1" ry="1"/>
              </svg>
              <span class="quick-action-label">粘贴</span>
            </button>
            <button class="quick-action-btn" @click="restartBackend()" :disabled="loading" title="重启后端">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M23 4v6h-6"/>
                <path d="M1 20v-6h6"/>
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10"/>
                <path d="M20.49 15a9 9 0 0 1-14.85 3.36L1 14"/>
              </svg>
              <span class="quick-action-label">重启</span>
            </button>
            <button class="quick-action-btn" @click="showProjectDialog = true" title="项目管理">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
              </svg>
              <span class="quick-action-label">项目</span>
            </button>
          </div>
        </div>
      </div>
    </main>

    <TerminalDialog
      :visible="showTerminal"
      @close="showTerminal = false"
    />

    <ClipboardDialog
      :visible="showClipboard"
      @close="showClipboard = false"
    />

    <ProjectDialog
      v-model="showProjectDialog"
    />

    <TagSelector
      v-if="currentProjectId !== null"
      v-model="showTagDialog"
      :projectId="currentProjectId"
      @change="handleTagChange"
    />

  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600;700&display=swap');

/* ===================================
   CYBERPUNK PHASE 1: GLOBAL CSS VARIABLES
   =================================== */
:root {
  /* Neon Palette */
  --neon-cyan: #00fff9;
  --neon-magenta: #ff00ff;
  --neon-purple: #b967ff;
  --neon-blue: #00d4ff;

  /* Text Colors */
  --text-primary: #e0e0e0;
  --text-secondary: #808080;
  --text-muted: #505050;

  /* Background */
  --bg-dark: #0a0a0f;
  --bg-card: rgba(20, 20, 30, 0.85);
  --bg-card-hover: rgba(30, 30, 45, 0.9);

  /* Neon Glows */
  --glow-cyan: 0 0 10px rgba(0, 255, 249, 0.5), 0 0 20px rgba(0, 255, 249, 0.3), 0 0 30px rgba(0, 255, 249, 0.1);
  --glow-magenta: 0 0 10px rgba(255, 0, 255, 0.5), 0 0 20px rgba(255, 0, 255, 0.3), 0 0 30px rgba(255, 0, 255, 0.1);
  --glow-purple: 0 0 10px rgba(185, 103, 255, 0.5), 0 0 20px rgba(185, 103, 255, 0.3), 0 0 30px rgba(185, 103, 255, 0.1);

  /* Fonts */
  --font-mono: 'JetBrains Mono', 'Fira Code', 'SF Mono', 'Monaco', monospace;
  --font-sans: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: var(--font-sans);
  background: var(--bg-dark);
  color: var(--text-primary);
  min-height: 100vh;
  -webkit-font-smoothing: antialiased;
}

::-webkit-scrollbar { width: 4px; height: 4px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: rgba(59, 130, 246, 0.1); border-radius: 2px; }
::-webkit-scrollbar-thumb:hover { background: #93c5fd; }

.dashboard {
  min-height: 100vh;
  padding: 24px;
}

/* Floating Status */
.backend-status.floating {
  position: fixed;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 18px;
  background: rgba(30, 41, 59, 0.85);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 24px;
  font-size: 13px;
  z-index: 100;
  backdrop-filter: blur(12px);
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

/* Quick Actions Panel */
.quick-actions-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
}

.quick-action-btn {
  width: 64px;
  height: 64px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s;
}

.quick-action-btn:hover {
  background: rgba(59, 130, 246, 0.2);
  border-color: rgba(59, 130, 246, 0.4);
  color: #f8fafc;
}

.quick-action-btn:active {
  transform: scale(0.95);
}

.quick-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.quick-action-btn svg {
  width: 22px;
  height: 22px;
}

.quick-action-label {
  font-size: 10px;
  font-weight: 500;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.backend-status.healthy .status-dot {
  background: #22c55e;
  box-shadow: 0 0 8px #22c55e;
}

.backend-status.unhealthy .status-dot {
  background: #ef4444;
  box-shadow: 0 0 8px #ef4444;
}

.backend-status.healthy .status-text { color: #22c55e; font-weight: 500; }
.backend-status.unhealthy .status-text { color: #ef4444; font-weight: 500; }
.status-pid { color: #64748b; }

.status-restart {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 6px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s;
  margin-left: 4px;
}

.status-restart:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #22d3ee;
  border-color: rgba(34, 211, 238, 0.3);
}

.status-restart:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.status-restart .restart-icon {
  width: 14px;
  height: 14px;
}

.status-restart .restart-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* Main */
.main {
  height: calc(100vh - 48px);
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 12px;
  margin-bottom: 20px;
  color: #fca5a5;
  font-size: 14px;
}

.error-icon { width: 20px; height: 20px; flex-shrink: 0; }

/* Columns */
.columns {
  display: grid;
  grid-template-columns: 1fr 1.2fr 1fr 80px;
  gap: 24px;
  height: calc(100vh - 48px);
}

.column {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.column-1 {
  background: var(--bg-card);
  border: 1px solid rgba(0, 255, 249, 0.15);
  border-radius: 20px;
  padding: 16px;
  transition: border-color 0.3s, box-shadow 0.3s;
}

.column-1:hover {
  border-color: rgba(0, 255, 249, 0.4);
  box-shadow: var(--glow-cyan), inset 0 1px 0 rgba(0, 255, 249, 0.1);
}

.column-2, .column-3 {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.column-3 {
  flex: 1;
  overflow: hidden;
}

.column-4 {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* Panel - Cyberpunk Card Base */
.panel {
  background: var(--bg-card);
  border: 1px solid rgba(185, 103, 255, 0.2);
  border-radius: 20px;
  padding: 24px;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: var(--glow-purple);
  transition: border-color 0.3s, box-shadow 0.3s;
}

.panel:hover {
  border-color: rgba(185, 103, 255, 0.4);
  box-shadow: var(--glow-purple), inset 0 1px 0 rgba(185, 103, 255, 0.05);
}

.panel-header {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  justify-content: space-between;
  align-items: flex-start;
}

.panel-header h2 {
  font-family: var(--font-mono);
  font-size: 16px;
  font-weight: 600;
  color: var(--neon-cyan);
  text-shadow: 0 0 10px rgba(0, 255, 249, 0.3);
}

.panel-subtitle {
  font-size: 11px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.sys-header-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sys-header-info {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.sys-info-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 8px;
  border: 1px solid rgba(255, 255, 255, 0.06);
}

.sys-info-icon {
  font-size: 12px;
}

.sys-info-val {
  font-size: 12px;
  color: #e2e8f0;
  font-weight: 500;
}

/* System Hero Card - Cyberpunk Style */
.sys-hero-card {
  background: linear-gradient(145deg, rgba(20, 20, 35, 0.95) 0%, rgba(10, 10, 20, 0.98) 100%);
  border: 1px solid rgba(0, 255, 249, 0.2);
  border-radius: 16px;
  padding: 16px;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: border-color 0.3s, box-shadow 0.3s;
}

.sys-hero-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, var(--neon-cyan), var(--neon-purple), var(--neon-magenta), var(--neon-cyan));
  background-size: 200% 100%;
}

@keyframes gradient-shift {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

.sys-hero-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.sys-hero-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(245, 176, 65, 0.2), rgba(245, 176, 65, 0.08));
  border: 1px solid rgba(245, 176, 65, 0.25);
  border-radius: 10px;
  color: #F5B041;
}

.sys-hero-icon svg {
  width: 22px;
  height: 22px;
}

.sys-hero-title {
  flex: 1;
  min-width: 0;
}

.sys-hero-name {
  font-size: 17px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: 0.3px;
}

.sys-hero-sub {
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sys-hero-badge {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  background: rgba(34, 197, 94, 0.12);
  border: 1px solid rgba(34, 197, 94, 0.25);
  border-radius: 16px;
  font-size: 11px;
  color: #34d399;
  font-weight: 500;
}

.badge-dot {
  width: 5px;
  height: 5px;
  background: #34d399;
  border-radius: 50%;
  box-shadow: 0 0 6px #34d399;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.sys-hero-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  margin-bottom: 12px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: rgba(0, 0, 0, 0.4);
  border-radius: 10px;
  border: 1px solid rgba(0, 255, 249, 0.1);
  transition: border-color 0.3s, box-shadow 0.3s;
}

.stat-item:hover {
  border-color: rgba(0, 255, 249, 0.25);
  box-shadow: inset 0 0 15px rgba(0, 255, 249, 0.05);
}

.stat-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  flex-shrink: 0;
}

.stat-icon svg {
  width: 16px;
  height: 16px;
}

.cpu-icon {
  background: rgba(251, 146, 60, 0.15);
  color: #fb923c;
}

.mem-icon {
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
}

.disk-icon {
  background: rgba(192, 132, 252, 0.15);
  color: #c084fc;
}

.net-icon {
  background: rgba(52, 211, 153, 0.15);
  color: #34d399;
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-label {
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--neon-cyan);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 2px;
}

.stat-value {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-primary);
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-value.mono {
  font-family: var(--font-mono);
  font-size: 11px;
}

.stat-bar {
  height: 3px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 2px;
  margin-top: 4px;
  overflow: hidden;
}

.stat-bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #38bdf8, #34d399);
  border-radius: 2px;
  transition: width 0.5s ease;
}

.stat-bar-fill.disk {
  background: linear-gradient(90deg, #c084fc, #a855f7);
}

.sys-hero-footer {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  flex-wrap: nowrap;
  overflow-x: auto;
}

.sys-hero-footer::-webkit-scrollbar {
  height: 0;
}

.footer-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex-shrink: 0;
}

.footer-label {
  font-size: 8px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

.footer-value {
  font-size: 11px;
  font-weight: 500;
  color: #94a3b8;
  white-space: nowrap;
}

/* Color themes per item */
.footer-item.hostname .footer-value { color: #3b82f6; }
.footer-item.wifi .footer-value { color: #22d3ee; }
.footer-item.mac .footer-value { color: #a855f7; }
.footer-item.vpn .footer-value { color: #fbbf24; }
.footer-item.node .footer-value { color: #22c55e; }
.footer-item.python .footer-value { color: #f5b041; }

.footer-divider {
  width: 1px;
  height: 20px;
  background: rgba(255, 255, 255, 0.06);
}

.sys-hero-metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
  margin-bottom: 12px;
}

.metric-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.metric-row-label {
  display: flex;
  align-items: center;
  gap: 5px;
  width: 60px;
  font-size: 10px;
  color: #64748b;
}

.metric-row-label svg {
  color: #fb923c;
  width: 12px;
  height: 12px;
}

.metric-row-bar {
  flex: 1;
  height: 5px;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 3px;
  overflow: hidden;
}

.metric-row-fill {
  height: 100%;
  background: linear-gradient(90deg, #fb923c, #f97316);
  border-radius: 3px;
  transition: width 0.5s ease;
}

.metric-row-fill.memory {
  background: linear-gradient(90deg, #38bdf8, #3b82f6);
}

.metric-row-fill.disk {
  background: linear-gradient(90deg, #c084fc, #a855f7);
}

.metric-row-fill.upload {
  background: linear-gradient(90deg, #34d399, #10b981);
}

.metric-row-fill.download {
  background: linear-gradient(90deg, #38bdf8, #06b6d4);
}

.metric-row-value {
  font-size: 11px;
  color: #e2e8f0;
  font-weight: 600;
  text-align: right;
  white-space: nowrap;
  min-width: 45px;
}

.metric-row-value.net-up {
  color: #34d399;
}

.metric-row-value.net-down {
  color: #38bdf8;
}

/* ===================================
   PHASE 2b: PROGRESS BARS
   =================================== */
.progress-bar-container {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: #1a1a2e;
  border-radius: 4px;
  overflow: hidden;
}

.progress-bar-fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.4s ease-out, background 0.3s ease, box-shadow 0.3s ease;
}

.progress-bar-value {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  text-shadow: 0 0 8px currentColor;
  min-width: 36px;
  text-align: right;
}

/* History Chart */
.sys-hero-history {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
  padding: 12px;
  margin-top: 8px;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.history-title {
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.time-range-btns {
  display: flex;
  gap: 4px;
}

.time-btn {
  padding: 3px 8px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 4px;
  color: #64748b;
  font-size: 9px;
  cursor: pointer;
  transition: all 0.15s;
}

.time-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #94a3b8;
}

.time-btn.active {
  background: rgba(34, 211, 238, 0.15);
  border-color: rgba(34, 211, 238, 0.3);
  color: #22d3ee;
}

.history-chart {
  height: 80px;
  position: relative;
}

.chart-svg {
  width: 100%;
  height: 60px;
  position: relative;
  cursor: crosshair;
}

.chart-svg svg {
  width: 100%;
  height: 100%;
}

/* Chart Tooltip - Phase 2a */
.chart-tooltip {
  position: absolute;
  top: -40px;
  transform: translateX(-50%);
  background: rgba(10, 10, 20, 0.95);
  border: 1px solid rgba(0, 255, 249, 0.3);
  border-radius: 6px;
  padding: 6px 10px;
  pointer-events: none;
  z-index: 10;
  box-shadow: 0 0 10px rgba(0, 255, 249, 0.2);
}

.tooltip-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
}

.tooltip-row.cpu .tooltip-label { color: #ff6b35; }
.tooltip-row.cpu .tooltip-value { color: #ff6b35; font-weight: 600; }
.tooltip-row.mem .tooltip-label { color: #00fff9; }
.tooltip-row.mem .tooltip-value { color: #00fff9; font-weight: 600; }

.chart-legend {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-top: 6px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 9px;
  color: #64748b;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}

.legend-item.cpu .legend-dot {
  background: #ff6b35;
  box-shadow: 0 0 6px #ff6b35;
}

.legend-item.mem .legend-dot {
  background: #00fff9;
  box-shadow: 0 0 6px #00fff9;
}

.history-empty {
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #475569;
  font-size: 11px;
}

/* Process Search */
.process-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 6px;
  margin-bottom: 10px;
}

.process-search svg {
  color: #475569;
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  background: none;
  border: none;
  color: #e2e8f0;
  font-size: 12px;
  outline: none;
  font-family: inherit;
}

.search-input::placeholder {
  color: #475569;
}

/* Process Detail Popup */
.process-detail-popup {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.popup-content {
  background: linear-gradient(145deg, rgba(30, 41, 59, 0.95), rgba(15, 23, 42, 0.98));
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  width: 380px;
  overflow: hidden;
}

.popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.popup-title {
  font-size: 12px;
  font-weight: 600;
  color: #f8fafc;
}

.popup-close {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.05);
  border: none;
  border-radius: 4px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.15s;
}

.popup-close:hover {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.popup-close svg {
  width: 14px;
  height: 14px;
}

.popup-body {
  padding: 20px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.detail-label {
  font-size: 12px;
  color: #64748b;
}

.detail-value {
  font-size: 13px;
  color: #e2e8f0;
  font-weight: 500;
}

.detail-value.mono {
  font-family: 'SF Mono', Monaco, monospace;
}

.detail-value.cpu-val { color: #f97316; }
.detail-value.mem-val { color: #38bdf8; }
.detail-value.status-val { color: #22c55e; }

.popup-actions {
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.1);
  border-top: 1px solid rgba(255, 255, 255, 0.04);
  display: flex;
  gap: 12px;
}

.popup-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 16px;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  font-family: inherit;
}

.popup-btn.kill {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.popup-btn.kill:hover {
  background: rgba(239, 68, 68, 0.25);
}

.popup-btn.close-btn {
  background: rgba(255, 255, 255, 0.05);
  color: #94a3b8;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.popup-btn.close-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #e2e8f0;
}

.popup-btn svg {
  width: 14px;
  height: 14px;
}

.net-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  padding: 0;
}

.net-item {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 10px;
  font-weight: 600;
}

.net-item.up {
  color: #34d399;
}

.net-item.down {
  color: #38bdf8;
}

.net-label {
  font-size: 9px;
  color: #64748b;
  width: 24px;
}

.net-val {
  font-family: 'SF Mono', 'Monaco', monospace;
  font-size: 10px;
}

.net-bar {
  flex: 1;
  height: 3px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 2px;
  overflow: hidden;
}

.net-bar-fill {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
}

.net-bar-fill.up {
  background: linear-gradient(90deg, #34d399, #10b981);
}

.net-bar-fill.down {
  background: linear-gradient(90deg, #38bdf8, #06b6d4);
}

.panel-content { flex: 1; overflow-y: auto; min-height: 0; }

.panel-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #64748b;
}

/* Metric Cards */
.metric-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-bottom: 24px;
}

.metric-card {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 14px;
  padding: 16px;
  display: grid;
  grid-template-columns: auto 1fr auto;
  gap: 14px;
  align-items: center;
}

.metric-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.metric-icon svg { width: 22px; height: 22px; }

.metric-icon.cpu {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
}

.metric-icon.memory {
  background: rgba(168, 85, 247, 0.15);
  color: #a855f7;
}

.metric-info { }

.metric-label {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 4px;
}

.metric-value {
  font-size: 22px;
  font-weight: 600;
  color: #f8fafc;
}

.metric-bar {
  width: 80px;
  height: 6px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
  overflow: hidden;
}

.metric-fill {
  height: 100%;
  background: #3b82f6;
  border-radius: 3px;
  transition: width 0.5s ease;
}

.metric-fill.memory { background: #a855f7; }

/* Info List */
.info-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 14px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
}

.info-label { font-size: 13px; color: #64748b; }
.info-value { font-size: 13px; color: #e2e8f0; font-weight: 500; }

/* Service List */
.service-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.service-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 12px;
}

.service-name { font-size: 14px; font-weight: 600; color: #f8fafc; margin-bottom: 4px; }
.service-desc { font-size: 12px; color: #64748b; }

.service-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  padding: 6px 12px;
  border-radius: 20px;
  background: rgba(239, 68, 68, 0.1);
  color: #fca5a5;
}

.service-status.running {
  background: rgba(34, 197, 94, 0.1);
  color: #86efac;
}

.status-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

/* Restart Button */
.restart-btn {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 14px 20px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 12px;
  color: #60a5fa;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: inherit;
}

.restart-btn:hover:not(:disabled) {
  background: rgba(59, 130, 246, 0.2);
  border-color: rgba(59, 130, 246, 0.3);
}

.restart-btn:disabled { opacity: 0.6; cursor: not-allowed; }

.restart-icon { width: 18px; height: 18px; }
.restart-icon.spinning { animation: spin 1s linear infinite; }

@keyframes spin { to { transform: rotate(360deg); } }

/* Quick Info */
.quick-info { flex: 0 0 auto; }

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.info-box {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 12px;
  padding: 14px;
  text-align: center;
}

.info-box-label { font-size: 11px; color: #64748b; margin-bottom: 6px; }
.info-box-value { font-size: 14px; font-weight: 600; color: #f8fafc; }

/* Loading Spinner */
.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid rgba(255, 255, 255, 0.1);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Tab Panel */
.tab-panel {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tab-panel .panel-header {
  margin-bottom: 12px;
  padding-bottom: 12px;
}

.tab-buttons {
  display: flex;
  gap: 4px;
  background: rgba(0, 0, 0, 0.3);
  padding: 4px;
  border-radius: 10px;
  border: 1px solid rgba(34, 211, 238, 0.1);
}

.tab-btn {
  flex: 1;
  background: transparent;
  border: none;
  border-radius: 6px;
  padding: 8px 16px;
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.25s ease;
  position: relative;
  text-transform: uppercase;
  letter-spacing: 1px;
}

.tab-btn::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 6px;
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.15), rgba(34, 211, 238, 0.05));
  opacity: 0;
  transition: opacity 0.25s;
}

.tab-btn:hover {
  color: #22d3ee;
}

.tab-btn:hover::before {
  opacity: 0.5;
}

.tab-btn.active {
  color: #22d3ee;
  text-shadow: 0 0 20px rgba(34, 211, 238, 0.5);
}

.tab-btn.active::before {
  opacity: 1;
}

.tab-btn.active::after {
  content: '';
  position: absolute;
  bottom: 4px;
  left: 50%;
  transform: translateX(-50%);
  width: 20px;
  height: 2px;
  background: #22d3ee;
  border-radius: 1px;
  box-shadow: 0 0 8px rgba(34, 211, 238, 0.8);
}

/* Process List */
.process-panel { }

.process-list {
  display: flex;
  flex-direction: column;
  gap: 3px;
  overflow-y: auto;
  flex: 1;
}

.process-list::-webkit-scrollbar { width: 3px; }
.process-list::-webkit-scrollbar-track { background: rgba(255,255,255,0.05); border-radius: 2px; }
.process-list::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.12); border-radius: 2px; }

.process-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  gap: 6px;
  padding: 4px 6px;
  font-size: 9px;
  color: #94a3b8;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.sort-btn {
  background: none;
  border: none;
  color: #94a3b8;
  font-size: 9px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  padding: 0;
  transition: color 0.15s;
}

.sort-btn:hover {
  color: #e2e8f0;
}

.sort-btn.active {
  color: #38bdf8;
}

.process-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1fr;
  gap: 6px;
  padding: 4px 6px;
  background: rgba(255,255,255,0.03);
  border-radius: 4px;
  align-items: center;
  font-size: 11px;
  transition: background 0.15s;
}

.process-row:hover { background: rgba(255,255,255,0.08); }

/* Sort animation - Phase 2c */
@keyframes sortPop {
  0% { opacity: 0; transform: translateY(-4px); }
  50% { transform: translateY(1px); }
  100% { opacity: 1; transform: translateY(0); }
}

.proc-name {
  color: #e0e0e0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 140px;
}

.proc-pid { color: #94a3b8; font-family: monospace; font-size: 10px; }

.proc-cpu, .proc-mem {
  display: flex;
  align-items: center;
  gap: 4px;
}

.proc-bar-container {
  flex: 1;
  height: 3px;
  background: rgba(255,255,255,0.1);
  border-radius: 2px;
  overflow: hidden;
  min-width: 40px;
}

.proc-bar {
  height: 100%;
  border-radius: 2px;
  transition: width 0.3s ease;
  display: block;
}

.proc-bar-cpu { background: linear-gradient(90deg, #f97316, #ef4444); }
.proc-bar-mem { background: linear-gradient(90deg, #06b6d4, #3b82f6); }

.proc-value {
  font-size: 10px;
  color: #94a3b8;
  min-width: 42px;
  text-align: right;
  font-family: monospace;
}

.proc-value.net-up { color: #34d399; }
.proc-value.net-down { color: #38bdf8; }
.proc-net {
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 9px;
  color: #94a3b8;
}

/* Port Flow - Data Stream Style */
.port-flow {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.port-row {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: linear-gradient(90deg, rgba(0, 255, 249, 0.08) 0%, rgba(0, 212, 255, 0.04) 50%, transparent 100%);
  border-left: 2px solid var(--neon-cyan);
  border-radius: 0 4px 4px 0;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  transition: all 0.2s ease;
  position: relative;
  overflow: hidden;
  cursor: pointer;
}

.port-row::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  height: 100%;
  width: 0;
  background: linear-gradient(90deg, rgba(0, 255, 249, 0.15), transparent);
  transition: width 0.3s ease;
}

.port-row:hover {
  background: linear-gradient(90deg, rgba(0, 255, 249, 0.15) 0%, rgba(0, 212, 255, 0.08) 50%, transparent 100%);
  transform: translateX(4px);
}

.port-row:hover::before {
  width: 100%;
}

.port-indicator {
  width: 6px;
  height: 6px;
  background: var(--neon-cyan);
  border-radius: 50%;
  margin-right: 12px;
  box-shadow: 0 0 8px var(--neon-cyan);
  animation: flow-pulse 1.5s ease-in-out infinite;
}

@keyframes flow-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 8px var(--neon-cyan); }
  50% { opacity: 0.5; box-shadow: 0 0 4px var(--neon-cyan); }
}

.port-row:hover .port-indicator {
  box-shadow: 0 0 12px var(--neon-cyan), 0 0 20px var(--neon-cyan);
}

.port-num {
  color: var(--neon-cyan);
  font-weight: 700;
  font-size: 14px;
  min-width: 50px;
  margin-right: 16px;
}

.port-name {
  color: #e2e8f0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.port-pid {
  color: var(--neon-purple);
  font-size: 11px;
  margin-left: 16px;
  opacity: 0.8;
}

/* Port Detail Popup */
.port-detail-popup {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.port-detail-content {
  background: #0a0a0f;
  border: 1px solid var(--neon-cyan);
  border-radius: 12px;
  padding: 20px;
  min-width: 350px;
  max-width: 450px;
  box-shadow: 0 0 30px rgba(0, 255, 249, 0.3), 0 0 60px rgba(0, 255, 249, 0.1);
  animation: popup-appear 0.2s ease-out;
}

@keyframes popup-appear {
  from { opacity: 0; transform: scale(0.95); }
  to { opacity: 1; transform: scale(1); }
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(0, 255, 249, 0.2);
}

.detail-port {
  font-size: 28px;
  font-weight: 800;
  color: var(--neon-cyan);
  text-shadow: 0 0 10px var(--neon-cyan);
}

.detail-name {
  flex: 1;
  font-size: 16px;
  color: #e2e8f0;
}

.detail-close {
  background: none;
  border: none;
  color: #64748b;
  font-size: 24px;
  cursor: pointer;
  padding: 0 8px;
  transition: color 0.15s;
}

.detail-close:hover { color: var(--neon-magenta); }

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 16px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-size: 14px;
  color: #e2e8f0;
}

.detail-value.neon-cyan { color: var(--neon-cyan); text-shadow: 0 0 8px var(--neon-cyan); }
.detail-value.neon-magenta { color: var(--neon-magenta); text-shadow: 0 0 8px var(--neon-magenta); }
.detail-value.neon-purple { color: var(--neon-purple); text-shadow: 0 0 8px var(--neon-purple); }

.detail-value.status {
  color: #22c55e;
  text-shadow: 0 0 8px rgba(34, 197, 94, 0.5);
}

.detail-path {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.detail-path-value {
  display: block;
  font-size: 11px;
  color: #94a3b8;
  margin-top: 4px;
  word-break: break-all;
  font-family: 'JetBrains Mono', monospace;
}

/* Session Detail Popup */
.session-detail-popup {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(6px);
}

.session-detail-content {
  background: #0a0a0f;
  border: 1px solid var(--neon-purple);
  border-radius: 12px;
  width: 600px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 0 40px rgba(185, 103, 255, 0.3), 0 0 80px rgba(185, 103, 255, 0.1);
  animation: popup-appear 0.2s ease-out;
}

.session-detail-header {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(185, 103, 255, 0.2);
}

.session-detail-title {
  flex: 1;
  font-size: 16px;
  font-weight: 600;
  color: #e2e8f0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-detail-close {
  background: none;
  border: none;
  color: #64748b;
  font-size: 24px;
  cursor: pointer;
  padding: 0 8px;
  transition: color 0.15s;
}

.session-detail-close:hover { color: var(--neon-magenta); }

.session-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.message-item {
  margin-bottom: 20px;
  padding: 14px 18px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.03);
  border-left: 3px solid #64748b;
}

.message-item.user {
  background: rgba(0, 255, 249, 0.08);
  border-left-color: var(--neon-cyan);
}

.message-item.assistant {
  background: rgba(185, 103, 255, 0.08);
  border-left-color: var(--neon-purple);
}

.message-role {
  font-size: 10px;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 10px;
  opacity: 0.8;
}

.message-item.user .message-role { color: var(--neon-cyan); }
.message-item.assistant .message-role { color: var(--neon-purple); }

.message-content {
  font-size: 14px;
  color: #f1f5f9;
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, sans-serif;
}

.session-empty {
  padding: 40px;
  text-align: center;
  color: #64748b;
}

/* Responsive */
@media (max-width: 1100px) {
  .columns { grid-template-columns: 1fr 1fr; height: auto; }
  .column-1 { grid-column: span 2; }
}

@media (max-width: 700px) {
  .columns { grid-template-columns: 1fr; }
  .column-1 { grid-column: span 1; }
  .dashboard { padding: 16px; }
}

/* Hermes Card - Cyberpunk Style */
.hermes-card {
  background: linear-gradient(145deg, rgba(20, 20, 35, 0.95) 0%, rgba(10, 10, 20, 0.98) 100%);
  border: 1px solid rgba(255, 0, 255, 0.25);
  border-radius: 16px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
  position: relative;
  box-shadow: var(--glow-magenta);
  transition: border-color 0.3s, box-shadow 0.3s;
}

.hermes-card:hover {
  border-color: rgba(255, 0, 255, 0.5);
}

.hermes-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.2);
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  position: relative;
  z-index: 1;
  flex-shrink: 0;
}

.hermes-title-row {
  display: flex;
  align-items: center;
  gap: 14px;
}

.hermes-logo {
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  color: #94a3b8;
}

.hermes-logo img {
  width: 28px;
  height: 28px;
  object-fit: contain;
}

.hermes-title-text h2 {
  font-family: var(--font-mono);
  font-size: 18px;
  font-weight: 700;
  color: var(--neon-magenta);
  letter-spacing: 2px;
  margin: 0;
  text-shadow: 0 0 15px rgba(255, 0, 255, 0.5);
}

.hermes-subtitle {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-secondary);
  letter-spacing: 0;
  text-transform: none;
}

.hermes-status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 6px;
}

.hermes-status-badge.running {
  background: rgba(34, 197, 94, 0.1);
  border-color: rgba(34, 197, 94, 0.3);
}

.status-ring {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
  /* Phase 4: dim glow for disconnected state */
  box-shadow: 0 0 4px rgba(239, 68, 68, 0.4);
}

.hermes-status-badge.running .status-ring {
  background: #22c55e;
  /* Phase 4: green glow for connected state */
  box-shadow:
    0 0 6px #22c55e,
    0 0 12px rgba(34, 197, 94, 0.6),
    0 0 20px rgba(34, 197, 94, 0.3);
  animation: gateway-pulse 1.5s ease-in-out infinite alternate;
}

@keyframes gateway-pulse {
  from { box-shadow: 0 0 6px #22c55e, 0 0 12px rgba(34, 197, 94, 0.6), 0 0 20px rgba(34, 197, 94, 0.3); }
  to   { box-shadow: 0 0 8px #22c55e, 0 0 18px rgba(34, 197, 94, 0.8), 0 0 30px rgba(34, 197, 94, 0.5); }
}

.status-text {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 11px;
  font-weight: 600;
  color: #f87171;
}

.hermes-status-badge.running .status-text {
  color: #22c55e;
}

.hermes-stats-bar {
  display: flex;
  align-items: center;
  justify-content: space-around;
  padding: 12px 20px;
  background: rgba(0, 0, 0, 0.15);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  z-index: 1;
  flex-shrink: 0;
}

.stat-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-val {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 18px;
  font-weight: 700;
  color: #f8fafc;
}

/* Hermes neon glow - cyan for sessions */
.stat-val.neon-cyan {
  color: var(--neon-cyan);
  font-weight: 800;
  font-size: 20px;
  text-shadow:
    0 0 5px var(--neon-cyan),
    0 0 10px var(--neon-cyan),
    0 0 20px var(--neon-cyan),
    0 0 40px rgba(0, 255, 249, 0.4);
  animation: neon-pulse-cyan 2s ease-in-out infinite alternate;
}

@keyframes neon-pulse-cyan {
  from { text-shadow: 0 0 5px var(--neon-cyan), 0 0 10px var(--neon-cyan), 0 0 20px var(--neon-cyan), 0 0 40px rgba(0, 255, 249, 0.4); }
  to   { text-shadow: 0 0 8px var(--neon-cyan), 0 0 15px var(--neon-cyan), 0 0 30px var(--neon-cyan), 0 0 60px rgba(0, 255, 249, 0.6); }
}

/* Hermes neon glow - magenta for PID */
.stat-val.neon-magenta {
  color: var(--neon-magenta);
  font-weight: 800;
  font-size: 20px;
  text-shadow:
    0 0 5px var(--neon-magenta),
    0 0 10px var(--neon-magenta),
    0 0 20px var(--neon-magenta),
    0 0 40px rgba(255, 0, 255, 0.4);
  animation: neon-pulse-magenta 2s ease-in-out infinite alternate;
}

@keyframes neon-pulse-magenta {
  from { text-shadow: 0 0 5px var(--neon-magenta), 0 0 10px var(--neon-magenta), 0 0 20px var(--neon-magenta), 0 0 40px rgba(255, 0, 255, 0.4); }
  to   { text-shadow: 0 0 8px var(--neon-magenta), 0 0 15px var(--neon-magenta), 0 0 30px var(--neon-magenta), 0 0 60px rgba(255, 0, 255, 0.6); }
}

/* Hermes neon glow - purple for tools */
.stat-val.neon-purple {
  color: var(--neon-purple);
  font-weight: 800;
  font-size: 20px;
  text-shadow:
    0 0 5px var(--neon-purple),
    0 0 10px var(--neon-purple),
    0 0 20px var(--neon-purple),
    0 0 40px rgba(185, 103, 255, 0.4);
  animation: neon-pulse-purple 2s ease-in-out infinite alternate;
}

@keyframes neon-pulse-purple {
  from { text-shadow: 0 0 5px var(--neon-purple), 0 0 10px var(--neon-purple), 0 0 20px var(--neon-purple), 0 0 40px rgba(185, 103, 255, 0.4); }
  to   { text-shadow: 0 0 8px var(--neon-purple), 0 0 15px var(--neon-purple), 0 0 30px var(--neon-purple), 0 0 60px rgba(185, 103, 255, 0.6); }
}

/* Hermes neon glow - blue for cron */
.stat-val.neon-blue {
  color: var(--neon-blue);
  font-weight: 800;
  font-size: 20px;
  text-shadow:
    0 0 5px var(--neon-blue),
    0 0 10px var(--neon-blue),
    0 0 20px var(--neon-blue),
    0 0 40px rgba(0, 212, 255, 0.4);
  animation: neon-pulse-blue 2s ease-in-out infinite alternate;
}

@keyframes neon-pulse-blue {
  from { text-shadow: 0 0 5px var(--neon-blue), 0 0 10px var(--neon-blue), 0 0 20px var(--neon-blue), 0 0 40px rgba(0, 212, 255, 0.4); }
  to   { text-shadow: 0 0 8px var(--neon-blue), 0 0 15px var(--neon-blue), 0 0 30px var(--neon-blue), 0 0 60px rgba(0, 212, 255, 0.6); }
}

.stat-key {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stat-divider {
  width: 1px;
  height: 30px;
  background: rgba(255, 255, 255, 0.08);
}

.hermes-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  position: relative;
  z-index: 1;
}

.hermes-block {
  background: rgba(0, 0, 0, 0.35);
  border: 1px solid rgba(185, 103, 255, 0.15);
  border-radius: 12px;
  padding: 14px;
  position: relative;
  transition: border-color 0.3s, box-shadow 0.3s;
}

.hermes-block:hover {
  border-color: rgba(185, 103, 255, 0.35);
  box-shadow: inset 0 0 20px rgba(185, 103, 255, 0.05);
}

.hermes-block::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 3px;
  height: 100%;
  background: linear-gradient(180deg, var(--neon-purple), rgba(185, 103, 255, 0.3), var(--neon-purple));
  border-radius: 3px 0 0 3px;
}

/* Block accent colors */
.hermes-block.platforms-block::before {
  background: linear-gradient(180deg, rgba(34, 197, 94, 0.7), rgba(34, 197, 94, 0.3), rgba(34, 197, 94, 0.7));
}
.hermes-block.model-block::before {
  background: linear-gradient(180deg, rgba(168, 85, 247, 0.7), rgba(168, 85, 247, 0.3), rgba(168, 85, 247, 0.7));
}
.hermes-block.settings-block::before {
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.7), rgba(59, 130, 246, 0.3), rgba(59, 130, 246, 0.7));
}
.hermes-block.safety-block::before {
  background: linear-gradient(180deg, rgba(251, 191, 36, 0.7), rgba(251, 191, 36, 0.3), rgba(251, 191, 36, 0.7));
}
.hermes-block.toolsets-block::before {
  background: linear-gradient(180deg, rgba(34, 211, 238, 0.7), rgba(34, 211, 238, 0.3), rgba(34, 211, 238, 0.7));
}
.hermes-block.cron-block::before {
  background: linear-gradient(180deg, rgba(236, 72, 153, 0.7), rgba(236, 72, 153, 0.3), rgba(236, 72, 153, 0.7));
}
.hermes-block.profiles-block::before {
  background: linear-gradient(180deg, rgba(99, 102, 241, 0.7), rgba(99, 102, 241, 0.3), rgba(99, 102, 241, 0.7));
}
.hermes-block.sessions-block::before {
  background: linear-gradient(180deg, rgba(14, 165, 233, 0.7), rgba(14, 165, 233, 0.3), rgba(14, 165, 233, 0.7));
}
.hermes-block.sessions-block .block-title { color: #0ea5e9; }

.sessions-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  background: rgba(14, 165, 233, 0.08);
  border-radius: 6px;
  border: 1px solid rgba(14, 165, 233, 0.15);
  position: relative;
  overflow: hidden;
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease;
}

/* Phase 4: Scanline hover effect for session items */
.session-item::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(14, 165, 233, 0.25), transparent);
  transition: left 0.4s ease;
  pointer-events: none;
}

.session-item:hover {
  background: rgba(14, 165, 233, 0.15);
  border-color: rgba(14, 165, 233, 0.4);
}

.session-item:hover::before {
  left: 100%;
}

.session-avatar {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(14, 165, 233, 0.15);
  border-radius: 6px;
  color: #0ea5e9;
}

.session-avatar svg {
  width: 14px;
  height: 14px;
}

.session-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.session-title {
  font-size: 11px;
  color: #f8fafc;
  font-weight: 500;
}

.session-id {
  font-size: 9px;
  color: rgba(14, 165, 233, 0.6);
  font-family: monospace;
}

.hermes-block.personality-block::before {
  background: linear-gradient(180deg, rgba(236, 72, 153, 0.7), rgba(236, 72, 153, 0.3), rgba(236, 72, 153, 0.7));
}
.hermes-block.personality-block .block-title { color: #ec4899; }

.personality-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  gap: 8px;
  margin-bottom: 14px;
}

.personality-chip {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 10px 6px;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.personality-chip:hover {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.15);
  transform: translateY(-1px);
}

.personality-chip.active {
  background: rgba(236, 72, 153, 0.15);
  border-color: rgba(236, 72, 153, 0.5);
  box-shadow: 0 0 12px rgba(236, 72, 153, 0.2);
}

.chip-icon {
  width: 20px;
  height: 20px;
  color: #94a3b8;
}

.personality-chip.active .chip-icon {
  color: #ec4899;
}

.chip-icon svg {
  width: 100%;
  height: 100%;
}

.chip-name {
  font-size: 10px;
  font-weight: 500;
  color: #94a3b8;
  text-transform: lowercase;
}

.personality-chip.active .chip-name {
  color: #f472b6;
}

.personality-preview-card {
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 10px;
  padding: 14px;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.preview-label {
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.preview-name {
  font-size: 14px;
  font-weight: 600;
  color: #f472b6;
  text-transform: capitalize;
}

.preview-quote {
  font-size: 12px;
  color: #cbd5e1;
  line-height: 1.6;
  font-style: italic;
  margin-bottom: 12px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 6px;
  border-left: 2px solid rgba(236, 72, 153, 0.4);
}

.preview-actions {
  display: flex;
  gap: 8px;
}

.preview-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  border: none;
}

.preview-btn.activate {
  background: linear-gradient(135deg, #ec4899, #f472b6);
  color: white;
}

.preview-btn.activate:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(236, 72, 153, 0.3);
}

.preview-btn svg {
  width: 14px;
  height: 14px;
}

.block-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.block-title {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--neon-purple);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.hermes-block.platforms-block .block-title { color: #22c55e; }
.hermes-block.model-block .block-title { color: #a855f7; }
.hermes-block.settings-block .block-title { color: #3b82f6; }
.hermes-block.safety-block .block-title { color: #fbbf24; }
.hermes-block.toolsets-block .block-title { color: #22d3ee; }
.hermes-block.cron-block .block-title { color: #ec4899; }
.hermes-block.profiles-block .block-title { color: #818cf8; }

.block-decoration {
  flex: 1;
  height: 1px;
  background: rgba(255, 255, 255, 0.06);
}

.block-count {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  padding: 2px 8px;
  background: rgba(148, 163, 184, 0.15);
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 4px;
  color: #94a3b8;
}

.hermes-block.platforms-block .block-count { background: rgba(34, 197, 94, 0.15); border-color: rgba(34, 197, 94, 0.3); color: #22c55e; }
.hermes-block.model-block .block-count { background: rgba(168, 85, 247, 0.15); border-color: rgba(168, 85, 247, 0.3); color: #a855f7; }
.hermes-block.settings-block .block-count { background: rgba(59, 130, 246, 0.15); border-color: rgba(59, 130, 246, 0.3); color: #3b82f6; }
.hermes-block.safety-block .block-count { background: rgba(251, 191, 36, 0.15); border-color: rgba(251, 191, 36, 0.3); color: #fbbf24; }
.hermes-block.toolsets-block .block-count { background: rgba(34, 211, 238, 0.15); border-color: rgba(34, 211, 238, 0.3); color: #22d3ee; }
.hermes-block.cron-block .block-count { background: rgba(236, 72, 153, 0.15); border-color: rgba(236, 72, 153, 0.3); color: #ec4899; }
.hermes-block.profiles-block .block-count { background: rgba(99, 102, 241, 0.15); border-color: rgba(99, 102, 241, 0.3); color: #818cf8; }

/* Platforms - Green accent */
.platforms-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.platform-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px;
  background: rgba(0, 0, 0, 0.3);
  border: 1px solid rgba(255, 255, 255, 0.05);
  border-radius: 6px;
  transition: all 0.2s ease;
}

.platform-item.connected {
  border-color: rgba(34, 197, 94, 0.3);
  background: rgba(34, 197, 94, 0.05);
}

.platform-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  opacity: 0.4;
  transition: all 0.2s ease;
}

.platform-item.connected .platform-icon {
  opacity: 1;
}

.platform-icon svg {
  width: 20px;
  height: 20px;
}

.platform-icon.weixin {
  background: rgba(7, 193, 96, 0.15);
  color: #07c160;
}

.platform-icon.telegram {
  background: rgba(0, 136, 204, 0.15);
  color: #0088cc;
}

.platform-icon.feishu {
  background: rgba(30, 136, 229, 0.15);
  color: #1e88e5;
}

.platform-icon.discord {
  background: rgba(88, 101, 242, 0.15);
  color: #5865f2;
}

.platform-icon.slack {
  background: rgba(97, 31, 105, 0.15);
  color: #611f69;
}

.platform-icon.whatsapp {
  background: rgba(37, 211, 102, 0.15);
  color: #25d366;
}

.platform-icon.matrix {
  background: rgba(0, 188, 212, 0.15);
  color: #00bcd4;
}

.platform-name {
  font-size: 11px;
  font-weight: 600;
  color: rgba(255, 255, 255, 0.5);
}

.platform-item.connected .platform-name {
  color: #f8fafc;
}

.platform-status {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 9px;
  color: #ef4444;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.platform-item.connected .platform-status {
  color: #22c55e;
}

/* Insights Block */
.insights-block {
  margin-top: 16px;
}

.insights-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  margin-bottom: 12px;
}

.insight-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  background: rgba(0, 255, 249, 0.05);
  border: 1px solid rgba(0, 255, 249, 0.1);
  border-radius: 8px;
  padding: 10px 6px;
}

.insight-val {
  font-family: 'JetBrains Mono', monospace;
  font-size: 16px;
  font-weight: 700;
}

.insight-label {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.4);
  margin-top: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.insight-platforms {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.insight-platform {
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
  padding: 4px 8px;
}

.platform-tag {
  font-size: 10px;
  color: rgba(255, 255, 255, 0.6);
  text-transform: capitalize;
}

.platform-tokens {
  font-family: 'JetBrains Mono', monospace;
  font-size: 10px;
  color: rgba(0, 255, 249, 0.8);
}

/* Skills - Green accent */
.skills-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.skill-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 4px;
  background: rgba(0, 255, 136, 0.05);
  border: 1px solid rgba(0, 255, 136, 0.1);
  border-radius: 8px;
  text-align: center;
}

.skill-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, rgba(0, 255, 136, 0.2), rgba(0, 255, 136, 0.05));
  border-radius: 6px;
  font-size: 12px;
  font-weight: 700;
  color: #00ff88;
  text-transform: uppercase;
}

.skill-name {
  font-size: 9px;
  color: rgba(255, 255, 255, 0.7);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

/* Profiles - Indigo accent */
.profiles-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.profile-card {
  background: rgba(99, 102, 241, 0.08);
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 10px;
  padding: 12px;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.profile-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(99, 102, 241, 0.15);
  border-radius: 8px;
  color: #818cf8;
}

.profile-icon svg {
  width: 18px;
  height: 18px;
}

.profile-info {
  flex: 1;
}

.profile-name {
  font-size: 13px;
  font-weight: 600;
  color: #f8fafc;
}

.profile-model {
  font-size: 10px;
  color: #818cf8;
  margin-top: 2px;
}

.profile-stats {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
}

.profile-stat {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.stat-num {
  font-size: 14px;
  font-weight: 600;
  color: #f8fafc;
}

.stat-label {
  font-size: 9px;
  color: rgba(99, 102, 241, 0.7);
}

.profile-soul {
  font-size: 9px;
  color: rgba(99, 102, 241, 0.6);
  line-height: 1.4;
  border-top: 1px solid rgba(99, 102, 241, 0.15);
  padding-top: 8px;
}

/* Model - Purple accent */
.model-main {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(168, 85, 247, 0.15);
}

.model-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(168, 85, 247, 0.12);
  border: 1px solid rgba(168, 85, 247, 0.25);
  border-radius: 10px;
  color: #a855f7;
  overflow: hidden;
}

.model-icon img {
  width: 32px;
  height: 32px;
  object-fit: contain;
}

.model-info {
  flex: 1;
}

.model-name {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 16px;
  font-weight: 700;
  color: #f8fafc;
}

.model-provider {
  margin-top: 4px;
}

.provider-badge {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  padding: 2px 8px;
  background: rgba(168, 85, 247, 0.15);
  border: 1px solid rgba(168, 85, 247, 0.25);
  border-radius: 4px;
  color: #c084fc;
}

.model-details {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.model-detail-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  background: rgba(168, 85, 247, 0.08);
  border-radius: 6px;
}

.detail-label {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 9px;
  color: rgba(168, 85, 247, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 11px;
  font-weight: 600;
  color: #f8fafc;
}

.detail-value.url {
  font-size: 9px;
  color: #a855f7;
  word-break: break-all;
}

.detail-value.high {
  color: #22c55e;
}

.detail-value.medium {
  color: #fbbf24;
}

.detail-value.low {
  color: #94a3b8;
}

/* Quota Usage */
.quota-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(168, 85, 247, 0.2);
}

.quota-header {
  font-size: 10px;
  color: rgba(168, 85, 247, 0.7);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}

.quota-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.quota-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.quota-model {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.quota-name {
  font-size: 10px;
  font-weight: 500;
  color: #e2e8f0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 140px;
}

.quota-usage {
  font-size: 9px;
  color: rgba(168, 85, 247, 0.8);
  font-family: 'JetBrains Mono', monospace;
}

.quota-bar {
  height: 4px;
  background: rgba(168, 85, 247, 0.15);
  border-radius: 2px;
  overflow: hidden;
}

.quota-fill {
  height: 100%;
  background: linear-gradient(90deg, #a855f7, #c084fc);
  border-radius: 2px;
  transition: width 0.3s ease;
}

/* Settings - Blue accent */
.settings-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.setting-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  background: rgba(59, 130, 246, 0.08);
  border-radius: 4px;
}

.setting-key {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  color: rgba(59, 130, 246, 0.8);
}

.setting-val {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 11px;
  font-weight: 600;
  color: #f8fafc;
}

.setting-val.toggle {
  padding: 2px 8px;
  border-radius: 3px;
  background: rgba(59, 130, 246, 0.1);
  color: #60a5fa;
}

.setting-val.toggle.on {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
}

/* Safety - Amber accent */
.safety-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.safety-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  background: rgba(251, 191, 36, 0.06);
  border-radius: 6px;
  border: 1px solid rgba(251, 191, 36, 0.15);
}

.safety-item.ask {
  border-color: rgba(251, 191, 36, 0.3);
}

.safety-item.yes {
  border-color: rgba(34, 197, 94, 0.3);
  background: rgba(34, 197, 94, 0.06);
}

.safety-item.yolo {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.06);
}

.safety-icon {
  width: 18px;
  height: 18px;
  color: rgba(251, 191, 36, 0.7);
}

.safety-item.ask .safety-icon { color: #fbbf24; }
.safety-item.yes .safety-icon { color: #34d399; }
.safety-item.yolo .safety-icon { color: #f87171; }

.safety-label {
  font-size: 9px;
  color: rgba(255, 255, 255, 0.5);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.safety-mode {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  font-weight: 600;
  color: #fbbf24;
}

.safety-item.yes .safety-mode { color: #34d399; }
.safety-item.yolo .safety-mode { color: #f87171; }

/* Toolsets - Cyan accent */
.toolsets-scroll {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.toolset-tag {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  padding: 4px 10px;
  background: rgba(34, 211, 238, 0.08);
  border: 1px solid rgba(34, 211, 238, 0.2);
  border-radius: 4px;
  color: #22d3ee;
  transition: all 0.2s ease;
}

.toolset-tag:hover {
  background: rgba(34, 211, 238, 0.15);
  border-color: rgba(34, 211, 238, 0.4);
  transform: translateY(-1px);
}

/* Cron - Pink accent */
.cron-list-compact {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.cron-item-compact {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  background: rgba(236, 72, 153, 0.08);
  border-radius: 4px;
}

.cron-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #64748b;
}

.cron-status-dot.ok {
  background: #ec4899;
}

.cron-status-dot.error {
  background: #f87171;
}

.cron-name-compact {
  flex: 1;
  font-size: 11px;
  color: #f8fafc;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cron-schedule-compact {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  color: rgba(236, 72, 153, 0.7);
}

/* Footer */
.hermes-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 20px;
  background: rgba(0, 0, 0, 0.2);
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  position: relative;
  z-index: 1;
  flex-shrink: 0;
}

.footer-timestamp {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 9px;
  color: #475569;
}

.footer-cmd {
  font-family: 'Plus Jakarta Sans', sans-serif;
  font-size: 10px;
  color: #64748b;
}

.footer-cmd::after {
  content: '_';
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ===================================
   PHASE 3a: ENTRY ANIMATION (STAGGER)
   =================================== */
@keyframes fadeSlideIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.card, .panel, .stat-card, .weather-row, .column {
  animation: fadeSlideIn 0.5s ease-out both;
  animation-delay: var(--stagger-delay, 0ms);
}

/* ===================================
   PHASE 3b: INTERACTION ANIMATIONS
   =================================== */
/* Card Hover - Neon border glow intensifies + translateY(-2px) */
.card:hover, .panel:hover, .stat-card:hover {
  transform: translateY(-2px);
  box-shadow:
    0 0 15px rgba(0, 255, 249, 0.4),
    0 0 30px rgba(0, 255, 249, 0.2),
    0 0 45px rgba(0, 255, 249, 0.1),
    inset 0 1px 0 rgba(0, 255, 249, 0.1);
  border-color: rgba(0, 255, 249, 0.5);
  transition: all 0.2s ease;
}

/* Button Press - Scale down on active */
button:active {
  transform: scale(0.97);
  transition: transform 0.1s ease;
}

/* Data Update Flash */
@keyframes valueFlash {
  0% {
    text-shadow: 0 0 20px currentColor, 0 0 40px currentColor;
    background: transparent;
  }
  50% {
    text-shadow: 0 0 30px currentColor, 0 0 60px currentColor, 0 0 80px currentColor;
  }
  100% {
    text-shadow: 0 0 8px currentColor;
    background: transparent;
  }
}

.value-updated {
  animation: valueFlash 0.4s ease-out;
}

/* ===================================
   PHASE 3c: BACKGROUND ATMOSPHERE
   =================================== */
/* Scanline Overlay */
.dashboard::before,
body::before {
  content: '';
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: repeating-linear-gradient(
    0deg,
    transparent,
    transparent 2px,
    rgba(0, 255, 249, 0.015) 2px,
    rgba(0, 255, 249, 0.015) 4px
  );
  pointer-events: none;
  z-index: 9999;
}

/* Neon Border Pulse on Key Cards */
@keyframes glowPulse {
  0%, 100% {
    box-shadow:
      0 0 5px rgba(0, 255, 249, 0.3),
      0 0 10px rgba(0, 255, 249, 0.2),
      inset 0 1px 0 rgba(0, 255, 249, 0.05);
    opacity: 0.8;
  }
  50% {
    box-shadow:
      0 0 15px rgba(0, 255, 249, 0.5),
      0 0 30px rgba(0, 255, 249, 0.3),
      0 0 45px rgba(0, 255, 249, 0.15),
      inset 0 1px 0 rgba(0, 255, 249, 0.1);
    opacity: 1;
  }
}

/* Hermes card glow pulse in magenta */
.hermes-card {
  animation: glowPulse 3s ease-in-out infinite;
  animation-name: glowPulseMagenta;
}

@keyframes glowPulseMagenta {
  0%, 100% {
    box-shadow:
      0 0 5px rgba(255, 0, 255, 0.3),
      0 0 10px rgba(255, 0, 255, 0.2),
      inset 0 1px 0 rgba(255, 0, 255, 0.05);
    opacity: 0.8;
  }
  50% {
    box-shadow:
      0 0 15px rgba(255, 0, 255, 0.5),
      0 0 30px rgba(255, 0, 255, 0.3),
      0 0 45px rgba(255, 0, 255, 0.15),
      inset 0 1px 0 rgba(255, 0, 255, 0.1);
    opacity: 1;
  }
}

/* ===================================
   ACCESSIBILITY: REDUCED MOTION
   =================================== */
@media (prefers-reduced-motion: reduce) {
  .card, .panel, .stat-card, .weather-row, .column {
    animation: none;
    opacity: 1;
    transform: none;
  }

  .sys-hero-card, .hermes-card {
    animation: none;
  }

  .value-updated {
    animation: none;
  }

  /* Keep hover interactions but remove transitions */
  .card:hover, .panel:hover, .stat-card:hover {
    transform: none;
    box-shadow: inherit;
    border-color: inherit;
    transition: none;
  }

  button:active {
    transform: none;
  }

  /* Disable scanlines for reduced motion */
  .app-container::before, body::before {
    display: none;
  }
}

/* ===================================
   ELEMENT PLUS DARK MODE OVERRIDES
   =================================== */
.dark {
  --el-bg-color: #1e1e1e;
  --el-bg-color-overlay: #2d2d2d;
  --el-text-color-primary: #ffffff;
  --el-text-color-regular: #a1a1a6;
  --el-text-color-secondary: #86868b;
  --el-text-color-placeholder: #6b6b6b;
  --el-border-color: #444444;
  --el-border-color-light: #333333;
  --el-border-color-lighter: #2d2d2d;
  --el-fill-color: #2d2d2d;
  --el-fill-color-light: #3d3d3d;
  --el-fill-color-lighter: #2d2d2d;
  --el-fill-color-blank: #1e1e1e;
  --el-color-primary: #0a84ff;
  --el-color-success: #30d158;
  --el-color-warning: #ff9f0a;
  --el-color-danger: #ff453a;
  --el-mask-color: rgba(0, 0, 0, 0.6);
}

</style>