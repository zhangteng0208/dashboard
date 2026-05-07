# Terminal Dialog Optimization Design

## Overview

Optimize the TerminalDialog component for tighter UI layout and add parameter placeholder support. Users can define commands with `{{paramName}}` placeholders and fill in values when executing.

## Design

### Main Dialog — Command Grid

- **Layout**: Compact grid (3-4 columns) instead of list
- **Command Card**: Icon + Label + Parameter Badge (if has params)
- **Parameter Badge**: Purple `{{...}}` tag on commands with placeholders
- **Execution**:
  - Commands without parameters: Click to execute directly
  - Commands with parameters: Click opens parameter popup

### Parameter Popup

- **Trigger**: Click on command with `{{paramName}}` placeholders
- **Size**: Centered modal, 360px wide
- **Content**:
  - Title: Command label
  - Subtitle: Original command template
  - Input fields: One per unique parameter
  - Buttons: Cancel / Execute
- **Execution**: Replace all `{{paramName}}` with user input, send to backend

### Data Model

```typescript
interface PresetCommand {
  id: string;
  label: string;       // Display name
  command: string;     // Template with {{paramName}} placeholders
  icon?: string;       // Optional icon
}
```

**Parameter Extraction**: Regex `/\{\{(\w+)\}\}/g` to extract parameter names.

### Default Commands

Replace existing defaults with mixed static/parametric commands:

```typescript
const defaultCommands: PresetCommand[] = [
  // Static commands (no params)
  { id: '1', label: '系统信息', command: 'uname -a && sw_vers' },
  { id: '2', label: '磁盘使用', command: 'df -h' },
  { id: '3', label: '内存状态', command: 'vm_stat' },
  { id: '4', label: 'CPU信息', command: 'sysctl -n machdep.cpu.brand_string' },
  { id: '5', label: '网络连接', command: 'netstat -an | head -20' },
  { id: '6', label: '进程TOP10', command: 'ps aux | head -11' },
  { id: '7', label: '端口占用', command: 'lsof -iTCP -sTCP:LISTEN | head -15' },
  { id: '8', label: '负载情况', command: 'uptime' },

  // Parametric commands
  { id: '9', label: 'Ping', command: 'ping {{host}}' },
  { id: '10', label: 'Git 最近提交', command: 'git log --oneline -n {{count}}' },
  { id: '11', label: '文件大小', command: 'du -sh {{path}}' },
  { id: '12', label: '端口检查', command: 'lsof -iTCP:{{port}}' },
];
```

### UI States

1. **Hover on static command**: Green "Run" badge appears
2. **Hover on parametric command**: Purple `{{...}}` badge + cursor pointer
3. **Parameter popup open**: Main dialog behind stays visible (dimmed)
4. **Execute with params**: Replace placeholders → POST to backend → close popup

## Implementation Notes

- Use `localStorage` key `dashboard_preset_commands` for persistence
- Parameter regex: `/\{\{(\w+)\}\}/g`
- Backend API: `POST /api/terminal/ghostty` with `{ command: string }`
- Keep backward compatibility with existing commands (no breaking changes)

## Scope

- Frontend only: TerminalDialog.vue modifications
- No backend changes required (uses existing `/api/terminal/ghostty` endpoint)
