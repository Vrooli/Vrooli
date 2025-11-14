# Desktop Integration Guide

## Overview

This guide shows how to add desktop-specific features to your scenario after generating a desktop app with scenario-to-desktop. The desktop wrapper provides a universal Electron template that works with any Vrooli scenario built with React/Vite/TypeScript.

## Quick Start

### 1. Generate Desktop Wrapper

```bash
# Method 1: Using the UI (recommended)
# - Visit http://localhost:<UI_PORT>
# - Go to "Scenario Inventory" tab
# - Find your scenario and click "Generate Desktop"

# Method 2: Using the CLI
scenario-to-desktop generate <your-scenario-name>

# Method 3: Using the API directly
curl -X POST http://localhost:<API_PORT>/api/v1/desktop/generate \
  -H "Content-Type: application/json" \
  -d '{
    "app_name": "your-scenario-name",
    "app_display_name": "Your App Name",
    "framework": "electron",
    "template_type": "basic"
  }'
```

### 2. File Structure

After generation, your scenario will have this structure:

```
scenarios/<your-scenario>/
├── api/                    # Your Go API (existing)
├── cli/                    # Your CLI tool (existing)
├── ui/                     # Your React web app (existing)
└── platforms/              # NEW: Deployment targets
    └── electron/           # Desktop wrapper
        ├── main.ts        # Electron main process
        ├── preload.ts     # Bridge between Electron and your UI
        ├── package.json   # Desktop dependencies
        ├── tsconfig.json  # TypeScript config
        ├── splash.html    # Splash screen
        ├── assets/        # Icons for all platforms
        │   ├── icon.icns  # macOS icon
        │   ├── icon.ico   # Windows icon
        │   └── icon.png   # Linux icon
        ├── dist/          # Compiled TypeScript (git-ignored)
        └── dist-electron/ # Built packages (git-ignored)
```

### 3. Build Your Desktop App

```bash
# 1. Build your web UI first (required)
cd ui
npm run build

# 2. Navigate to desktop wrapper
cd ../platforms/electron

# 3. Install dependencies
npm install

# 4. Development mode (with hot reload)
npm run dev

# 5. Build for distribution
npm run dist              # Current platform only
npm run dist:win          # Windows
npm run dist:mac          # macOS
npm run dist:linux        # Linux
npm run dist:all          # All platforms
```

## Desktop Features

### File System Access

The Electron wrapper exposes file system APIs to your React app through the `window.electron` bridge.

**Example: File Picker**

```typescript
// In your React component
import { useState } from 'react';

function MyComponent() {
  const [fileContent, setFileContent] = useState('');

  const handleOpenFile = async () => {
    // Check if running in desktop mode
    if (typeof window.electron === 'undefined') {
      alert('File system access only available in desktop version');
      return;
    }

    const result = await window.electron.dialog.showOpenDialog({
      properties: ['openFile'],
      filters: [
        { name: 'JSON Files', extensions: ['json'] },
        { name: 'Text Files', extensions: ['txt'] },
        { name: 'All Files', extensions: ['*'] }
      ]
    });

    if (result.filePaths.length > 0) {
      const content = await window.electron.fs.readFile(result.filePaths[0]);
      setFileContent(content);
    }
  };

  return (
    <button onClick={handleOpenFile}>
      Open File
    </button>
  );
}
```

**Example: Save File**

```typescript
const handleSaveFile = async (data: string) => {
  if (typeof window.electron === 'undefined') return;

  const result = await window.electron.dialog.showSaveDialog({
    defaultPath: 'export.json',
    filters: [{ name: 'JSON', extensions: ['json'] }]
  });

  if (result.filePath) {
    await window.electron.fs.writeFile(result.filePath, data);
    alert('File saved successfully!');
  }
};
```

### System Notifications

```typescript
const showNotification = async () => {
  if (typeof window.electron === 'undefined') return;

  await window.electron.notification.show({
    title: 'Task Complete',
    body: 'Your export is ready!',
    icon: '/path/to/icon.png'
  });
};
```

### Native Menus

Customize the application menu in `platforms/electron/main.ts`:

```typescript
// In main.ts
import { Menu, MenuItem } from 'electron';

function createApplicationMenu() {
  const menuTemplate = [
    {
      label: 'File',
      submenu: [
        {
          label: 'Import',
          accelerator: 'CmdOrCtrl+I',
          click: () => {
            mainWindow?.webContents.send('menu-import');
          }
        },
        {
          label: 'Export',
          accelerator: 'CmdOrCtrl+E',
          click: () => {
            mainWindow?.webContents.send('menu-export');
          }
        },
        { type: 'separator' },
        { label: 'Exit', role: 'quit' }
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
    }
  ];

  const menu = Menu.buildFromTemplate(menuTemplate);
  Menu.setApplicationMenu(menu);
}
```

**Listen for menu events in your React app:**

```typescript
import { useEffect } from 'react';

function useMenuHandlers() {
  useEffect(() => {
    if (typeof window.electron === 'undefined') return;

    const unsubscribeImport = window.electron.ipc.on('menu-import', () => {
      handleImport();
    });

    const unsubscribeExport = window.electron.ipc.on('menu-export', () => {
      handleExport();
    });

    return () => {
      unsubscribeImport();
      unsubscribeExport();
    };
  }, []);
}
```

### System Tray

Enable system tray in your template configuration, then customize in `main.ts`:

```typescript
import { Tray, Menu, nativeImage } from 'electron';

let tray: Tray | null = null;

function createTray() {
  const icon = nativeImage.createFromPath(
    path.join(__dirname, '../assets/icon.png')
  );

  tray = new Tray(icon);

  const trayMenu = Menu.buildFromTemplate([
    {
      label: 'Show App',
      click: () => {
        mainWindow?.show();
      }
    },
    {
      label: 'Custom Action',
      click: () => {
        // Your custom action
        mainWindow?.webContents.send('tray-action');
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

  tray.setContextMenu(trayMenu);
  tray.setToolTip('Your App Name');
}
```

## Progressive Enhancement Pattern

Write code that works in both web and desktop environments:

```typescript
// Create a hook for desktop features
function useDesktop() {
  const isDesktop = typeof window.electron !== 'undefined';

  const openFile = async () => {
    if (!isDesktop) {
      // Web fallback: use HTML file input
      return new Promise((resolve) => {
        const input = document.createElement('input');
        input.type = 'file';
        input.onchange = (e) => {
          const file = (e.target as HTMLInputElement).files?.[0];
          if (file) {
            const reader = new FileReader();
            reader.onload = (e) => resolve(e.target?.result);
            reader.readAsText(file);
          }
        };
        input.click();
      });
    }

    // Desktop: use native file picker
    const result = await window.electron.dialog.showOpenDialog({
      properties: ['openFile']
    });

    if (result.filePaths.length > 0) {
      return await window.electron.fs.readFile(result.filePaths[0]);
    }
  };

  return { isDesktop, openFile };
}

// Usage
function MyComponent() {
  const { isDesktop, openFile } = useDesktop();

  return (
    <div>
      {isDesktop && <div className="badge">Desktop Mode</div>}
      <button onClick={openFile}>Open File</button>
    </div>
  );
}
```

## TypeScript Type Definitions

Add type definitions for the Electron bridge in your UI:

```typescript
// ui/src/types/electron.d.ts
export interface ElectronAPI {
  dialog: {
    showOpenDialog: (options: any) => Promise<{ filePaths: string[] }>;
    showSaveDialog: (options: any) => Promise<{ filePath?: string }>;
  };
  fs: {
    readFile: (path: string) => Promise<string>;
    writeFile: (path: string, data: string) => Promise<void>;
  };
  notification: {
    show: (options: { title: string; body: string; icon?: string }) => Promise<void>;
  };
  ipc: {
    on: (channel: string, callback: () => void) => () => void;
    send: (channel: string, ...args: any[]) => void;
  };
}

declare global {
  interface Window {
    electron?: ElectronAPI;
  }
}
```

## Auto-Updates

The desktop wrapper includes electron-updater by default. Configure it in `platforms/electron/package.json`:

```json
{
  "build": {
    "publish": {
      "provider": "github",
      "owner": "your-org",
      "repo": "your-repo"
    }
  }
}
```

The auto-updater will automatically check for updates on app startup and notify users when updates are available.

## Distribution

### Code Signing

**macOS:**
```bash
# Set environment variables
export APPLE_ID="your-apple-id@example.com"
export APPLE_ID_PASSWORD="app-specific-password"
export CSC_NAME="Developer ID Application: Your Name"

# Build with signing
npm run dist:mac
```

**Windows:**
```bash
# Set certificate path
export CSC_LINK="/path/to/certificate.pfx"
export CSC_KEY_PASSWORD="your-password"

# Build with signing
npm run dist:win
```

### App Store Submission

The generated desktop app can be submitted to:
- **Microsoft Store** (Windows)
- **Mac App Store** (macOS)
- **Snap Store** (Linux)

See [Electron Builder documentation](https://www.electron.build/) for platform-specific requirements.

## Testing

```bash
# Run in development mode
cd platforms/electron
npm run dev

# Test production build without packaging
npm run pack

# Full end-to-end test
npm run dist
# Then install and run the generated app
```

## Common Patterns

### Offline Mode

Desktop apps can work completely offline:

```typescript
// Check if online
const [isOnline, setIsOnline] = useState(navigator.onLine);

useEffect(() => {
  const handleOnline = () => setIsOnline(true);
  const handleOffline = () => setIsOnline(false);

  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);

  return () => {
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
  };
}, []);

// Use local storage or IndexedDB for offline data
```

### Deep Links

Register custom URL schemes in `package.json`:

```json
{
  "build": {
    "protocols": [
      {
        "name": "your-app",
        "schemes": ["yourapp"]
      }
    ]
  }
}
```

Then handle deep links in `main.ts`:

```typescript
app.on('open-url', (event, url) => {
  event.preventDefault();
  // Handle yourapp://some-action URLs
  mainWindow?.webContents.send('deep-link', url);
});
```

## Troubleshooting

### UI Not Loading
- Ensure `ui/dist/` exists (run `npm run build` in ui/)
- Check `SERVER_PATH` in `platforms/electron/main.ts`
- Verify `scenario_dist_path` in package.json points to correct location

### Native Modules Not Working
```bash
cd platforms/electron
npm run postinstall
```

### Permission Errors (macOS)
Add entitlements in `assets/entitlements.mac.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
  <true/>
  <key>com.apple.security.files.user-selected.read-write</key>
  <true/>
</dict>
</plist>
```

## Next Steps

1. **Customize your desktop app** - Edit `platforms/electron/main.ts` and `splash.html`
2. **Add desktop-specific features** - File system, notifications, menus, tray
3. **Test across platforms** - Windows, macOS, Linux
4. **Setup auto-updates** - Configure GitHub releases or custom update server
5. **Submit to app stores** - Follow platform-specific guidelines

## Resources

- [Electron Documentation](https://www.electronjs.org/docs)
- [Electron Builder](https://www.electron.build/)
- [Vrooli Documentation](https://docs.vrooli.com)
- [scenario-to-desktop README](../README.md)

---

**Need help?** Open an issue at https://github.com/vrooli/vrooli/issues
