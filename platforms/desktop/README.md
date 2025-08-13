# Electron Desktop Application Template

## 🤖 AI Agent Instructions

This template provides a foundation for converting any web application into a native desktop application using Electron. As an AI agent, you should adapt this template based on the specific app's architecture and requirements.

## 📋 Quick Adaptation Checklist

1. [ ] Update `APP_CONFIG` in `src/main.ts` with app-specific values
2. [ ] Determine server type (node/static/external/executable)
3. [ ] Customize splash screen (`src/splash.html`)
4. [ ] Configure preload APIs (`src/preload.ts`)
5. [ ] Add IPC handlers in main process for needed APIs
6. [ ] Update build configuration
7. [ ] Test on target platforms

## 🏗️ Template Structure

```
desktop/
├── src/
│   ├── main.ts        # Main Electron process (app lifecycle, window management)
│   ├── preload.ts     # Bridge between web app and native APIs
│   └── splash.html    # Loading screen shown during startup
├── assets/
│   └── icon.ico       # Application icon (Windows/Linux)
├── package.json       # Dependencies and build scripts
└── tsconfig.json      # TypeScript configuration
```

## 🎯 Common Adaptation Scenarios

### Scenario 1: Node.js Backend Application
**Example:** Express server with React frontend

```typescript
// In main.ts, update APP_CONFIG:
const APP_CONFIG = {
    APP_NAME: "MyExpressApp",
    SERVER_TYPE: "node",
    SERVER_PORT: 3001,
    SERVER_PATH: "server/index.js",
    // ... other config
};
```

### Scenario 2: Static Single-Page Application  
**Example:** Vite-built React app with no backend

```typescript
// In main.ts, update APP_CONFIG:
const APP_CONFIG = {
    APP_NAME: "MyReactApp",
    SERVER_TYPE: "static",
    SERVER_PATH: "dist/index.html",  // Built files location
    // SERVER_PORT not needed for static
    // ... other config
};

// Also update window loading:
// Replace: await mainWindow.loadURL(SERVER_URL);
// With: await mainWindow.loadFile(path.join(app.getAppPath(), APP_CONFIG.SERVER_PATH));
```

### Scenario 3: Python/Go/Rust Backend
**Example:** FastAPI server with compiled executable

```typescript
// In main.ts, update APP_CONFIG:
const APP_CONFIG = {
    APP_NAME: "MyPythonApp",
    SERVER_TYPE: "executable",
    SERVER_PORT: 8000,
    SERVER_PATH: "backend/server.exe",  // or "backend/server" on Unix
    // ... other config
};
```

### Scenario 4: External/Cloud Backend
**Example:** App that connects to remote API

```typescript
// In main.ts, update APP_CONFIG:
const APP_CONFIG = {
    APP_NAME: "CloudApp",
    SERVER_TYPE: "external",
    SERVER_PATH: "https://api.myapp.com",
    // ... other config
};
```

## 🔧 Customization Points

### 1. Window Configuration
Modify in `main.ts`:
- `WINDOW_WIDTH` / `WINDOW_HEIGHT` - Initial window size
- `WINDOW_BACKGROUND` - Background color while loading
- Add `minWidth`, `minHeight`, `maxWidth`, `maxHeight` for constraints
- Set `frame: false` for custom window chrome
- Add `icon: path.join(...)` for app icon

### 2. Native Features
Enable features in `preload.ts` based on app needs:

| Feature | Use Case | Implementation |
|---------|----------|----------------|
| File System | Save/load local files | Uncomment file operations |
| System Tray | Background operation | Add tray code to main.ts |
| Global Shortcuts | Hotkeys | Use `globalShortcut` API |
| Native Menus | Context menus | Use `Menu.buildFromTemplate()` |
| Notifications | System alerts | Use Notification API |
| Auto-update | App updates | Integrate electron-updater |
| Deep Linking | URL protocol handling | Register protocol |

### 3. Splash Screen
Customize `splash.html`:
- Replace spinner with app logo
- Update colors to match branding
- Add loading progress indicators
- Include version information

### 4. Security Considerations
- Keep `nodeIntegration: false` and `contextIsolation: true`
- Whitelist IPC channels in preload
- Validate all user inputs
- Use Content Security Policy
- Sign executables for distribution

## 📦 Build Configuration

### Package.json Template
```json
{
  "name": "{{app-name}}-desktop",
  "version": "1.0.0",
  "main": "dist/main.js",
  "scripts": {
    "build": "tsc",
    "start": "electron .",
    "dev": "tsc -w & electron .",
    "pack": "electron-builder --dir",
    "dist": "electron-builder",
    "dist:all": "electron-builder -mwl"
  },
  "devDependencies": {
    "electron": "^28.0.0",
    "electron-builder": "^24.0.0",
    "typescript": "^5.0.0"
  },
  "build": {
    "appId": "com.{{company}}.{{app}}",
    "productName": "{{AppName}}",
    "directories": {
      "output": "dist-electron"
    },
    "files": [
      "dist/**/*",
      "assets/**/*",
      "{{app-files}}/**/*"
    ],
    "mac": {
      "category": "public.app-category.productivity"
    },
    "win": {
      "target": "nsis"
    },
    "linux": {
      "target": "AppImage"
    }
  }
}
```

## 🚀 Implementation Steps

### Step 1: Analyze App Structure
```bash
# Identify entry points
find . -name "index.html" -o -name "server.js" -o -name "main.py"

# Check for existing build outputs
ls -la dist/ build/ out/

# Identify static assets
find . -type f \( -name "*.png" -o -name "*.jpg" -o -name "*.svg" \)
```

### Step 2: Configure Template
1. Copy this template to app's `desktop/` folder
2. Update `APP_CONFIG` values
3. Adjust paths based on app structure
4. Add necessary dependencies

### Step 3: Implement IPC Handlers
In `main.ts`, add handlers for preload APIs:

```typescript
// Example: File operations
ipcMain.handle('fs:readFile', async (event, path) => {
  return fs.promises.readFile(path, 'utf-8');
});

// Example: System info
ipcMain.handle('system:getPlatform', () => {
  return process.platform;
});
```

### Step 4: Test Integration
```bash
# Build TypeScript
npm run build

# Test in development
npm run dev

# Build for distribution
npm run dist
```

## 🎨 Platform-Specific Enhancements

### Windows
- Add Windows installer with NSIS
- Implement jump list tasks
- Add thumbnail toolbar buttons
- Support Windows notifications

### macOS
- Add dock menu
- Implement Touch Bar support
- Support Handoff
- Add macOS-specific keyboard shortcuts

### Linux
- Create .desktop file
- Add Unity launcher integration
- Support system tray indicators
- Package as Snap/Flatpak/AppImage

## 🐛 Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| "Cannot find module" | Check SERVER_PATH is relative to app root |
| White screen on launch | Verify server is running and SERVER_URL is correct |
| App won't quit | Ensure server process is killed in 'quit' event |
| High memory usage | Implement lazy loading, limit window count |
| Security warnings | Sign code, notarize for macOS |

## 📚 Advanced Features

### Multi-Window Support
```typescript
const windows = new Map();

function createWindow(route = '/') {
  const win = new BrowserWindow({...});
  windows.set(win.id, win);
  win.loadURL(`${SERVER_URL}${route}`);
  win.on('closed', () => windows.delete(win.id));
}
```

### Protocol Handling
```typescript
app.setAsDefaultProtocolClient('myapp');
app.on('open-url', (event, url) => {
  event.preventDefault();
  // Handle myapp:// URLs
});
```

### Native Module Integration
```typescript
// For native Node modules
const sqlite3 = require('sqlite3');
const sharp = require('sharp');
// Ensure these are properly built for Electron
```

## 🔍 Debugging Tips

1. **Enable DevTools**: Press `Ctrl+Shift+I` or add to main.ts:
   ```typescript
   mainWindow.webContents.openDevTools();
   ```

2. **Log Locations**:
   - Windows: `%APPDATA%/{{app-name}}/logs`
   - macOS: `~/Library/Logs/{{app-name}}`
   - Linux: `~/.config/{{app-name}}/logs`

3. **Debug Server Issues**:
   ```typescript
   serverProcess.stdout.on('data', (data) => {
     console.log(`Server: ${data}`);
   });
   ```

## 📝 Final Notes for AI Agents

When adapting this template:

1. **Preserve Comments**: Keep AI AGENT NOTE comments for future agents
2. **Document Changes**: Add comments explaining app-specific modifications
3. **Test Thoroughly**: Verify on all target platforms
4. **Security First**: Never expose sensitive APIs without validation
5. **Performance**: Monitor memory usage and optimize as needed
6. **User Experience**: Ensure smooth transitions and provide feedback

Remember: This template is a starting point. Each app will have unique requirements that may need creative solutions beyond what's provided here.

## 🔗 Resources

- [Electron Documentation](https://www.electronjs.org/docs)
- [Electron Builder](https://www.electron.build/)
- [Electron Security Best Practices](https://www.electronjs.org/docs/latest/tutorial/security)
- [Node.js Child Process](https://nodejs.org/api/child_process.html)