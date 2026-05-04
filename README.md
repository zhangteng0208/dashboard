# Dashboard

基于 Tauri + Vue 3 的 macOS 桌面 Dashboard，集成了 Go 后端提供实时系统监控。

## 功能特性

### 系统监控
- **CPU** — 实时 CPU 使用率，支持历史趋势图（1小时/6小时/24小时）
- **内存** — 已用/总量监控
- **磁盘** — 存储使用情况
- **网络** — 上传/下载速度（KB/s，精确到小数点后两位）

### 天气与日期
- 当前位置天气（温度、体感、风速、气压等）
- 公历/农历日期显示
- 传统八字、风水、节气、纳音五行
- 明日天气预报预览

### Hermes CLI 集成
- 查看 Hermes 活跃会话
- Gateway 连接状态
- 配置管理
- Toolset 列表
- Cron 任务状态
- Profile 管理

### 进程与端口管理
- 进程列表（CPU/内存热力图）
- 端口占用查看
- 支持终止进程

### 终端命令快捷方式
- 预设常用命令（系统信息、磁盘、内存、网络连接等）
- 支持添加/编辑/删除自定义命令
- 命令通过 Ghostty 终端执行

### 剪贴板历史
- 记录剪贴板历史
- 搜索、复制、置顶功能

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端框架 | Vue 3 + TypeScript |
| 桌面框架 | Tauri 2 |
| 后端 | Go (REST API + WebSocket) |
| 终端 | xterm.js + Ghostty |
| UI 风格 | 赛博朋克（Cyberpunk）暗色主题 |

## 项目结构

```
tauri_verify/
├── src/                    # Vue 前端源码
│   ├── App.vue            # 主应用组件
│   ├── main.ts            # Vue 入口
│   └── components/        # UI 组件
│       ├── DateWeatherCard.vue   # 日期天气卡片
│       ├── TerminalDialog.vue    # 终端命令弹窗
│       └── ClipboardDialog.vue   # 剪贴板弹窗
├── go_backend/            # Go 后端
│   └── main.go           # REST API + WebSocket 服务
├── src-tauri/            # Tauri 桌面应用配置
└── public/               # 静态资源
```

## 开发

### 前端开发

```bash
# 安装依赖
pnpm install

# 开发模式（热重载）
pnpm tauri dev
```

### 后端开发

```bash
cd go_backend

# 构建
go build -o dashboard-backend

# 运行（默认端口 18788）
./dashboard-backend
```

### 构建桌面应用

```bash
pnpm tauri build
```

构建产物位于 `src-tauri/target/release/bundle/`。

## 设计风格

采用赛博朋克视觉风格：
- 背景色：#0a0a0f（深黑）
- 霓虹青色：#00fff9（低负载指示）
- 霓虹品红：#ff00ff（中负载）
- 霓虹红：#ff3366（高负载）
- 扫描线背景动效
- 卡片入场动画（stagger）
- 支持 `prefers-reduced-motion`

## 注意事项

- 后端默认监听 `http://127.0.0.1:18788`
- 前端运行时后端需保持启动
- 天气数据来自 wttr.in
- 终端命令通过 Ghostty (`http://127.0.0.1:18788/api/terminal/ghostty`) 执行