# 🖥️ Desktop Wrapper Guide

## The Universal Wrapper Concept

The scenario-to-desktop system creates a **universal desktop wrapper** that transforms any Vrooli scenario into a native desktop application. The key insight: **the desktop version is a superset of the web app**, not a reimplementation.

```
┌─────────────────────────────────────┐
│     Desktop Application             │
│  ┌───────────────────────────────┐  │
│  │    Electron Wrapper Layer     │  │ ← Native OS Features
│  │  (File access, menus, etc.)   │  │
│  ├───────────────────────────────┤  │
│  │                               │  │
│  │     Web Application UI        │  │ ← Unchanged from web
│  │    (React, Vite, etc.)        │  │
│  │                               │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Core Principles

1. **Zero Reimplementation**: The web app runs unchanged inside Electron
2. **Progressive Enhancement**: Desktop features add capabilities, don't replace them
3. **Server Flexibility**: Works with static files, external APIs, or embedded servers
4. **Standard Structure**: All desktop versions live in `scenarios/<name>/platforms/electron/`

## Architecture Patterns

### Pattern 1: Static SPA Wrapper

**When to Use**: Scenario has a static UI with no backend

**Structure**:
```
scenarios/picker-wheel/
├── ui/
│   └── dist/           # Built web app
└── platforms/
    └── electron/       # Desktop wrapper
        ├── src/
        │   ├── main.ts      # Loads ui/dist/index.html
        │   └── preload.ts   # Minimal IPC bridge
        └── renderer/        # Symlink to ../../ui/dist/
```

**Configuration**:
```typescript
{
  server_type: "static",
  server_path: "../../ui/dist",  // Path to built UI
  api_endpoint: ""                // No API needed
}
```

### Pattern 2: External API Connection

**When to Use**: Scenario has a separate API server that runs independently

**Structure**:
```
scenarios/system-monitor/
├── api/                # Go API server (runs separately)
├── ui/
│   └── dist/          # Built web app
└── platforms/
    └── electron/
        ├── src/
        │   ├── main.ts      # Connects to localhost:API_PORT
        │   └── preload.ts   # IPC bridge
        └── renderer/        # UI files
```

**Configuration**:
```typescript
{
  server_type: "external",
  server_path: "../../ui/dist",
  api_endpoint: "http://localhost:15200"  // API must be running
}
```

**Usage Flow**:
1. User starts API: `vrooli scenario start system-monitor`
2. User launches desktop app
3. Desktop app connects to running API on configured port

### Pattern 3: Embedded Node Server

**When to Use**: Scenario has a Node.js/Express backend that can be embedded

**Structure**:
```
scenarios/document-manager/
├── server/
│   └── index.js       # Express server
├── ui/
│   └── dist/
└── platforms/
    └── electron/
        ├── src/
        │   ├── main.ts      # Forks server/index.js
        │   └── preload.ts
        └── server/          # Copy of ../../server/
```

**Configuration**:
```typescript
{
  server_type: "node",
  server_path: "./server/index.js",
  server_port: 3000
}
```

**Main Process Code**:
```typescript
// main.ts - Start embedded server
import { fork } from 'child_process';

const serverProcess = fork(path.join(__dirname, '../server/index.js'), {
  env: { PORT: '3000' }
});

// Load UI pointing to localhost:3000
mainWindow.loadURL('http://localhost:3000');
```

### Pattern 4: Embedded Executable

**When to Use**: Scenario has a compiled binary backend (Go, Rust, etc.)

**Structure**:
```
scenarios/analytics-dashboard/
├── api/
│   └── analytics-api  # Go binary
├── ui/
│   └── dist/
└── platforms/
    └── electron/
        ├── src/
        │   ├── main.ts      # Spawns analytics-api
        │   └── preload.ts
        └── bin/
            └── analytics-api  # Bundled binary
```

**Configuration**:
```typescript
{
  server_type: "executable",
  server_path: "./bin/analytics-api",
  server_port: 8080
}
```

**Main Process Code**:
```typescript
// main.ts - Start embedded executable
import { spawn } from 'child_process';

const serverProcess = spawn(path.join(__dirname, '../bin/analytics-api'), {
  env: { PORT: '8080' }
});

// Wait for server to be ready, then load UI
await waitForServer('http://localhost:8080/health');
mainWindow.loadURL('http://localhost:8080');
```

## Adding Desktop-Specific Features

### File System Access

Desktop apps can access the file system directly. Here's how to add this capability:

#### 1. Define IPC Channels in Preload

```typescript
// preload.ts
import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // File operations
  saveFile: (filename: string, content: string) =>
    ipcRenderer.invoke('save-file', filename, content),

  openFile: () =>
    ipcRenderer.invoke('open-file'),

  selectDirectory: () =>
    ipcRenderer.invoke('select-directory'),

  // OS information
  getPlatform: () =>
    ipcRenderer.invoke('get-platform'),
});
```

#### 2. Implement Handlers in Main Process

```typescript
// main.ts
import { dialog, app } from 'electron';
import fs from 'fs/promises';

// File save handler
ipcMain.handle('save-file', async (event, filename, content) => {
  const { filePath } = await dialog.showSaveDialog({
    defaultPath: filename,
    filters: [
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'All Files', extensions: ['*'] }
    ]
  });

  if (filePath) {
    await fs.writeFile(filePath, content, 'utf-8');
    return { success: true, path: filePath };
  }
  return { success: false };
});

// File open handler
ipcMain.handle('open-file', async () => {
  const { filePaths } = await dialog.showOpenDialog({
    properties: ['openFile'],
    filters: [
      { name: 'JSON Files', extensions: ['json'] },
      { name: 'All Files', extensions: ['*'] }
    ]
  });

  if (filePaths.length > 0) {
    const content = await fs.readFile(filePaths[0], 'utf-8');
    return { success: true, content, path: filePaths[0] };
  }
  return { success: false };
});

// Directory selector
ipcMain.handle('select-directory', async () => {
  const { filePaths } = await dialog.showOpenDialog({
    properties: ['openDirectory']
  });

  if (filePaths.length > 0) {
    return { success: true, path: filePaths[0] };
  }
  return { success: false };
});

// Platform info
ipcMain.handle('get-platform', () => {
  return {
    platform: process.platform,
    arch: process.arch,
    version: app.getVersion()
  };
});
```

#### 3. Use in React Components

```typescript
// React component (runs in renderer process)
declare global {
  interface Window {
    electronAPI: {
      saveFile: (filename: string, content: string) => Promise<any>;
      openFile: () => Promise<any>;
      selectDirectory: () => Promise<any>;
      getPlatform: () => Promise<any>;
    };
  }
}

function ExportButton() {
  const handleExport = async () => {
    const data = JSON.stringify({ /* your data */ }, null, 2);
    const result = await window.electronAPI.saveFile('export.json', data);

    if (result.success) {
      alert(`File saved to ${result.path}`);
    }
  };

  return <button onClick={handleExport}>Export Data</button>;
}
```

### Native Menus

Create application-specific menu bars:

```typescript
// main.ts
import { Menu, app } from 'electron';

function createApplicationMenu() {
  const template: any[] = [
    {
      label: 'File',
      submenu: [
        {
          label: 'New Project',
          accelerator: 'CmdOrCtrl+N',
          click: () => {
            mainWindow.webContents.send('menu-new-project');
          }
        },
        {
          label: 'Open...',
          accelerator: 'CmdOrCtrl+O',
          click: () => {
            mainWindow.webContents.send('menu-open');
          }
        },
        { type: 'separator' },
        {
          label: 'Exit',
          accelerator: 'CmdOrCtrl+Q',
          click: () => app.quit()
        }
      ]
    },
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' }
      ]
    },
    {
      label: 'View',
      submenu: [
        { role: 'reload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' }
      ]
    }
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// Call after creating window
createApplicationMenu();
```

**React Side - Listen for Menu Events**:
```typescript
useEffect(() => {
  // Listen for menu commands
  const handleNewProject = () => {
    // Create new project logic
  };

  const handleOpen = () => {
    window.electronAPI.openFile();
  };

  // Register listeners
  window.electronAPI?.on?.('menu-new-project', handleNewProject);
  window.electronAPI?.on?.('menu-open', handleOpen);

  return () => {
    window.electronAPI?.off?.('menu-new-project', handleNewProject);
    window.electronAPI?.off?.('menu-open', handleOpen);
  };
}, []);
```

### System Notifications

Show native OS notifications:

```typescript
// preload.ts
contextBridge.exposeInMainWorld('electronAPI', {
  showNotification: (title: string, body: string) =>
    ipcRenderer.invoke('show-notification', title, body),
});

// main.ts
import { Notification } from 'electron';

ipcMain.handle('show-notification', async (event, title, body) => {
  new Notification({
    title,
    body,
    icon: path.join(__dirname, '../assets/icon.png')
  }).show();

  return { success: true };
});

// React component
function TaskCompleteNotifier() {
  const notifyComplete = async () => {
    await window.electronAPI.showNotification(
      'Task Complete',
      'Your export has finished successfully!'
    );
  };

  return <button onClick={notifyComplete}>Complete Task</button>;
}
```

### System Tray Integration

Add a system tray icon for background operation:

```typescript
// main.ts
import { Tray, Menu, nativeImage } from 'electron';

let tray: Tray | null = null;

function createSystemTray() {
  const icon = nativeImage.createFromPath(
    path.join(__dirname, '../assets/tray-icon.png')
  );

  tray = new Tray(icon);

  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show App',
      click: () => {
        mainWindow.show();
      }
    },
    {
      label: 'Settings',
      click: () => {
        mainWindow.show();
        mainWindow.webContents.send('navigate-to', '/settings');
      }
    },
    { type: 'separator' },
    {
      label: 'Quit',
      click: () => {
        app.quit();
      }
    }
  ]);

  tray.setContextMenu(contextMenu);
  tray.setToolTip('My Desktop App');

  // Double-click to show window
  tray.on('double-click', () => {
    mainWindow.show();
  });
}

// Allow window to minimize to tray
mainWindow.on('close', (event) => {
  if (!app.isQuitting) {
    event.preventDefault();
    mainWindow.hide();
  }
});
```

### Global Keyboard Shortcuts

Register shortcuts that work even when app is in background:

```typescript
// main.ts
import { globalShortcut } from 'electron';

function registerGlobalShortcuts() {
  // Show/hide app
  globalShortcut.register('CommandOrControl+Shift+Space', () => {
    if (mainWindow.isVisible()) {
      mainWindow.hide();
    } else {
      mainWindow.show();
      mainWindow.focus();
    }
  });

  // Quick capture
  globalShortcut.register('CommandOrControl+Shift+C', () => {
    mainWindow.show();
    mainWindow.webContents.send('quick-capture');
  });
}

// Clean up on quit
app.on('will-quit', () => {
  globalShortcut.unregisterAll();
});
```

### Deep Linking

Allow opening app from custom URLs (e.g., `myapp://open/project/123`):

```typescript
// main.ts
const PROTOCOL = 'myapp';

if (process.defaultApp) {
  if (process.argv.length >= 2) {
    app.setAsDefaultProtocolClient(PROTOCOL, process.execPath, [
      path.resolve(process.argv[1])
    ]);
  }
} else {
  app.setAsDefaultProtocolClient(PROTOCOL);
}

// Handle the protocol on macOS
app.on('open-url', (event, url) => {
  event.preventDefault();
  handleDeepLink(url);
});

// Handle the protocol on Windows
const gotTheLock = app.requestSingleInstanceLock();
if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', (event, commandLine) => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.focus();

      // Windows: the last argument is the deep link
      const url = commandLine.pop();
      if (url?.startsWith('myapp://')) {
        handleDeepLink(url);
      }
    }
  });
}

function handleDeepLink(url: string) {
  // Parse URL: myapp://open/project/123
  const parsed = new URL(url);
  mainWindow.show();
  mainWindow.webContents.send('deep-link', {
    protocol: parsed.protocol,
    host: parsed.host,
    pathname: parsed.pathname
  });
}
```

## Template Selection Guide

### Basic Template
**Best for**: Simple utilities, calculators, viewers

**Includes**:
- Native menus (File/Edit/View)
- File save/open dialogs
- Auto-updater integration
- Single window

**Use when**:
- Scenario doesn't need background operation
- Single-window interface is sufficient
- Minimal OS integration needed

### Advanced Template
**Best for**: Professional tools, system utilities, productivity apps

**Includes**:
- Everything from Basic template
- System tray integration
- Global keyboard shortcuts
- Rich notifications
- Multi-monitor support

**Use when**:
- App should run in background
- Need global hotkeys
- Complex OS integration required

### Multi-Window Template
**Best for**: IDEs, dashboards, complex workflows

**Includes**:
- Everything from Advanced template
- Multiple window management
- Inter-window communication
- Floating tool panels
- Window state persistence

**Use when**:
- Multiple independent views needed
- Tool palettes or inspectors
- Dashboard with detachable panels

### Kiosk Template
**Best for**: Public displays, point-of-sale, industrial controls

**Includes**:
- Full-screen lock mode
- Security hardening
- Remote monitoring
- Auto-restart
- Unattended operation

**Use when**:
- Dedicated hardware deployment
- Public-facing display
- No user access to OS needed

## Best Practices

### 1. Separation of Concerns

**Keep web logic in the web app**:
```typescript
// ❌ Bad: Business logic in main process
ipcMain.handle('calculate-total', (event, items) => {
  return items.reduce((sum, item) => sum + item.price, 0);
});

// ✅ Good: Business logic in renderer (web app)
// Main process only handles OS interactions
ipcMain.handle('save-invoice', async (event, data) => {
  const { filePath } = await dialog.showSaveDialog({...});
  await fs.writeFile(filePath, JSON.stringify(data));
  return { success: true, path: filePath };
});
```

### 2. Graceful Degradation

**Make desktop features optional**:
```typescript
// React component
function ExportButton() {
  const isDesktop = typeof window.electronAPI !== 'undefined';

  const handleExport = async () => {
    const data = JSON.stringify({/* data */});

    if (isDesktop) {
      // Use native save dialog
      const result = await window.electronAPI.saveFile('export.json', data);
      if (result.success) {
        alert(`Saved to ${result.path}`);
      }
    } else {
      // Fall back to browser download
      const blob = new Blob([data], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'export.json';
      a.click();
    }
  };

  return <button onClick={handleExport}>Export</button>;
}
```

### 3. Security

**Always validate IPC inputs**:
```typescript
// ❌ Bad: No validation
ipcMain.handle('read-file', async (event, filePath) => {
  return await fs.readFile(filePath, 'utf-8');
});

// ✅ Good: Validate paths
ipcMain.handle('read-file', async (event, filePath) => {
  // Only allow reading from app's data directory
  const appData = app.getPath('userData');
  const resolvedPath = path.resolve(appData, filePath);

  if (!resolvedPath.startsWith(appData)) {
    throw new Error('Access denied: Invalid path');
  }

  return await fs.readFile(resolvedPath, 'utf-8');
});
```

### 4. Performance

**Minimize IPC calls**:
```typescript
// ❌ Bad: Many small IPC calls
for (const file of files) {
  await window.electronAPI.saveFile(file.name, file.content);
}

// ✅ Good: Batch operations
await window.electronAPI.saveFiles(
  files.map(f => ({ name: f.name, content: f.content }))
);
```

## Common Scenarios

### Adding "Save As" to Any Scenario

1. **Add IPC handler** (main.ts):
```typescript
ipcMain.handle('save-as', async (event, defaultName, content) => {
  const { filePath } = await dialog.showSaveDialog({
    defaultPath: defaultName
  });
  if (filePath) {
    await fs.writeFile(filePath, content);
    return { success: true, path: filePath };
  }
  return { success: false };
});
```

2. **Expose in preload** (preload.ts):
```typescript
contextBridge.exposeInMainWorld('electronAPI', {
  saveAs: (name: string, content: string) =>
    ipcRenderer.invoke('save-as', name, content),
});
```

3. **Use in React**:
```typescript
const handleSave = async () => {
  if (window.electronAPI) {
    const result = await window.electronAPI.saveAs(
      'document.txt',
      documentContent
    );
    if (result.success) {
      console.log('Saved to:', result.path);
    }
  }
};
```

### Making Any Scenario Installable

The desktop wrapper automatically makes the scenario installable:

1. **Windows**: Generates `.msi` installer with Start Menu integration
2. **macOS**: Creates `.pkg` installer for drag-to-Applications flow
3. **Linux**: Builds `.AppImage` and `.deb` packages

No additional code needed - electron-builder handles this based on `package.json`:

```json
{
  "build": {
    "appId": "com.vrooli.picker-wheel",
    "productName": "Picker Wheel",
    "win": {
      "target": "nsis",
      "icon": "assets/icon.ico"
    },
    "mac": {
      "target": "dmg",
      "icon": "assets/icon.icns"
    },
    "linux": {
      "target": ["AppImage", "deb"],
      "icon": "assets/icon.png"
    }
  }
}
```

## Troubleshooting

### UI Doesn't Load

**Symptom**: Blank window or "Cannot GET /" error

**Solution**: Check server_type and paths
```typescript
// For static SPAs
server_type: "static",
server_path: "../../ui/dist"  // Must point to built UI

// For external APIs
server_type: "external",
api_endpoint: "http://localhost:15200"  // API must be running
```

### IPC Not Working

**Symptom**: `window.electronAPI is undefined`

**Solution**: Ensure preload script is loaded
```typescript
// main.ts
const mainWindow = new BrowserWindow({
  webPreferences: {
    preload: path.join(__dirname, 'preload.js'),  // ← Must be set
    contextIsolation: true,
    nodeIntegration: false
  }
});
```

### File Paths Not Resolving

**Symptom**: "File not found" errors in packaged app

**Solution**: Use `__dirname` and `app.getAppPath()`
```typescript
// ❌ Bad: Relative paths break when packaged
const icon = './assets/icon.png';

// ✅ Good: Absolute paths work everywhere
const icon = path.join(__dirname, '../assets/icon.png');
```

## Resources

- [Electron Documentation](https://www.electronjs.org/docs/latest/)
- [Electron Security Best Practices](https://www.electronjs.org/docs/latest/tutorial/security)
- [IPC Pattern Examples](https://www.electronjs.org/docs/latest/tutorial/ipc)
- [electron-builder Docs](https://www.electron.build/)

---

**Remember**: The desktop wrapper enhances your web app with OS-level capabilities. Keep the core logic in the web app, and use the wrapper purely for desktop-specific features. This maintains the "superset" principle and ensures your web app continues to work standalone.
