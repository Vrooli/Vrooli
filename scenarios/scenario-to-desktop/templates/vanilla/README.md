# Vanilla Desktop App Template

This template provides a complete Electron-based desktop application foundation that can be customized for any Vrooli scenario. It includes modern desktop features, cross-platform compatibility, and secure communication patterns.

## 🏗️ Template Structure

```
vanilla/
├── main.ts              # Main Electron process (app lifecycle, window management)
├── preload.ts          # Secure bridge between main and renderer processes  
├── splash.html         # Loading screen shown during startup
├── package.json.template # Dependencies and build configuration
├── tsconfig.json       # TypeScript compilation settings
└── README.md          # This documentation
```

## 🎯 Template Variables

The template uses the following variables that are replaced during generation:

### Application Identity
- `{{APP_NAME}}` - Technical app name (lowercase, no spaces)
- `{{APP_DISPLAY_NAME}}` - User-facing app name
- `{{APP_DESCRIPTION}}` - Brief app description
- `{{VERSION}}` - App version (e.g., "1.0.0")
- `{{AUTHOR}}` - Author/company name
- `{{LICENSE}}` - Software license (e.g., "MIT")
- `{{APP_ID}}` - Unique app identifier (e.g., "com.company.app")
- `{{APP_URL}}` - App website/help URL

### Server Configuration
- `{{SERVER_TYPE}}` - Server type: "node", "static", "external", "executable"
- `{{DEPLOYMENT_MODE}}` - Deployment intent: "external-server", "cloud-api", or "bundled"
- `{{SERVER_PORT}}` - Port for local server (if applicable)
- `{{SERVER_PATH}}` - Path to server entry point or static files
- `{{API_ENDPOINT}}` - Scenario API endpoint for communication
- `{{SCENARIO_DIST_PATH}}` - Path to scenario's built files
- `{{SCENARIO_NAME}}` - Scenario identifier for telemetry + optional server automation
- `{{AUTO_MANAGE_TIER1}}` - `true` to let the desktop wrapper run `vrooli` commands (defaults to `false` so thin clients can connect to hosted Vrooli servers without the CLI)
- `{{VROOLI_BINARY_PATH}}` - Override for locating the `vrooli` CLI

### Window Configuration
- `{{WINDOW_WIDTH}}` - Initial window width (default: 1200)
- `{{WINDOW_HEIGHT}}` - Initial window height (default: 800)
- `{{WINDOW_BACKGROUND}}` - Background color while loading

### Features (boolean values)
- `{{ENABLE_SPLASH}}` - Show splash screen during startup
- `{{ENABLE_MENU}}` - Show native application menu
- `{{ENABLE_SYSTEM_TRAY}}` - Add system tray icon
- `{{ENABLE_AUTO_UPDATER}}` - Enable automatic updates
- `{{ENABLE_SINGLE_INSTANCE}}` - Prevent multiple app instances
- `{{ENABLE_DEV_TOOLS}}` - Enable developer tools in development

### Styling (for splash screen)
- `{{SPLASH_BACKGROUND_START}}` - Splash gradient start color
- `{{SPLASH_BACKGROUND_END}}` - Splash gradient end color
- `{{SPLASH_TEXT_COLOR}}` - Splash text color
- `{{SPLASH_ACCENT_COLOR}}` - Splash accent/loading color
- `{{APP_ICON_PATH}}` - Path to app icon for splash

### Distribution
- `{{PUBLISHER_NAME}}` - Publisher name for Windows
- `{{YEAR}}` - Current year for copyright
- `{{PUBLISH_CONFIG}}` - Publication configuration (JSON)

## 🔧 Server Integration Patterns

The template supports multiple server architectures:

### 1. Node.js Server (`SERVER_TYPE: "node"`)
- Forks a Node.js process for the backend
- Suitable for Express, Fastify, or other Node.js servers
- Server process is managed by Electron

```typescript
// Example configuration
SERVER_TYPE: "node"
SERVER_PORT: 3000
SERVER_PATH: "backend/dist/server.js"
```

### 2. Static Files (`SERVER_TYPE: "static"`)
- Loads static HTML/JS files directly
- No server process needed
- Suitable for pre-built SPA applications

```typescript
// Example configuration
SERVER_TYPE: "static"
SERVER_PATH: "dist/index.html"
```

### 3. External Server (`SERVER_TYPE: "external"`)
- Connects to an external/cloud-based server
- No local server management
- Suitable for cloud-native scenarios

```typescript
// Example configuration
SERVER_TYPE: "external"
SERVER_PATH: "https://api.example.com"
```

### 4. Executable Server (`SERVER_TYPE: "executable"`)
- Spawns a compiled binary (Go, Rust, Python, etc.)
- Server binary is bundled with the app
- Cross-platform executable management

```typescript
// Example configuration
SERVER_TYPE: "executable"
SERVER_PORT: 8000
SERVER_PATH: "backend/server.exe"
```

## 🖥️ Desktop Features

### Native OS Integration
- **File System Access**: Save/open files with native dialogs
- **System Tray**: Persistent system tray icon with context menu
- **Native Menus**: Standard application menus with keyboard shortcuts
- **Notifications**: System notifications using native APIs
- **Auto-updater**: Seamless application updates

### Window Management
- **Single Instance**: Prevents multiple app instances by default
- **Splash Screen**: Professional loading screen during startup
- **Window State**: Remembers window size/position across sessions
- **Full Screen**: Native full-screen mode support

### Security
- **Context Isolation**: Secure communication between processes
- **IPC Validation**: Whitelisted channels for inter-process communication
- **Sandboxing**: Renderer process sandboxing for security
- **Code Signing**: Support for code signing (requires certificates)

## ⚙️ Local Server Bootstrapper (Opt-In)

Keep `AUTO_MANAGE_TIER1=false` for traditional thin clients. When you explicitly set it to `true` **and** keep `DEPLOYMENT_MODE=external-server`, the template ships with an automation layer that:

1. Searches for the `vrooli` CLI (prompts the user to locate it if missing, persisting the choice under the app's user data directory).
2. Runs `vrooli setup --yes yes --skip-sudo yes` once per machine to install required resources without elevated privileges.
3. Executes `vrooli scenario start <SCENARIO_NAME>` on app launch and waits for the scenario to report healthy.
4. Calls `vrooli scenario stop <SCENARIO_NAME>` when the desktop app exits—only if it started the scenario.

If you distribute the app to users who do not have `vrooli` installed, leave this disabled; the wrapper will simply load the remote UI/API you configured.

## 🚧 Bundled Mode Placeholder

`DEPLOYMENT_MODE="bundled"` currently shows a dialog explaining that offline/bundled builds are unavailable until Tier 2 shipping lands. Use this mode only for testing the UX copy; regenerate with `external-server` for production thin clients.

All steps log telemetry to `deployment-telemetry.jsonl` so deployment-manager can see how often bootstrapping succeeds versus when the user had to intervene manually.

## 🌐 Web Application Integration

The template provides a secure bridge between your web application and desktop features:

### Desktop API Usage

```javascript
// Available in your web application's JavaScript

// File operations
const filePath = await window.desktop.save(content, 'data.json');
const fileData = await window.desktop.open();
const jsonPath = await window.desktop.saveJSON(data, 'config.json');
const jsonData = await window.desktop.loadJSON();

// System information
const info = await window.desktop.getInfo();
console.log(`Running on ${info.platform} v${info.appVersion}`);

// Application control
await window.desktop.minimize();
await window.desktop.maximize();
await window.desktop.close();

// Notifications
window.desktop.notify('Process completed!', 'Success');

// Menu actions
window.desktop.onMenuAction((action, data) => {
    if (action === 'new') {
        // Handle new file/document
    }
});

// Protocol URLs (deep linking)
window.desktop.onProtocolUrl((url) => {
    // Handle custom protocol URLs like myapp://action
});

// Feature detection
if (window.desktop.features.fileSystem) {
    // File system features are available
}
```

### Environment Detection

```javascript
// Check if running in desktop mode
if (window.desktopUtils.isDesktop) {
    // Desktop-specific UI/behavior
} else {
    // Web browser fallbacks
}
```

## 🚀 Development Workflow

### Initial Setup
```bash
# Install dependencies
npm install

# Development build
npm run build

# Start in development mode
npm run dev

# Development with auto-rebuild
npm run dev:watch
```

### Production Build
```bash
# Build for current platform
npm run dist

# Build for specific platforms
npm run dist:win    # Windows
npm run dist:mac    # macOS
npm run dist:linux  # Linux

# Build for all platforms
npm run dist:all
```

### Testing
```bash
# Package without distributing
npm run pack

# Clean build artifacts
npm run clean
```

## 📦 Customization Guide

### Adding Custom Features

1. **Custom IPC Handlers** (in `main.ts`):
```typescript
ipcMain.handle("custom:action", async (event, data) => {
    // Handle custom functionality
    return result;
});
```

2. **Preload API Extensions** (in `preload.ts`):
```typescript
// Add to desktopAPI object
custom: {
    action: async (data: any) => {
        return ipcRenderer.invoke("custom:action", data);
    }
}
```

3. **Menu Customization** (in `main.ts`):
```typescript
const template: Electron.MenuItemConstructorOptions[] = [
    // Add custom menu items
    {
        label: "Custom",
        submenu: [
            // Custom menu items
        ]
    }
];
```

### Styling the Splash Screen

Modify the CSS variables in `splash.html`:
```css
:root {
    --primary-color: #your-color;
    --background-gradient: linear-gradient(135deg, #start, #end);
}
```

### Adding Native Dependencies

1. Install native modules:
```bash
npm install sqlite3 sharp  # Example native modules
```

2. Ensure proper rebuilding for Electron:
```bash
npm run postinstall
```

3. Use in main process (not renderer):
```typescript
// In main.ts only
const sqlite3 = require('sqlite3');
```

## 🔒 Security Best Practices

### Process Isolation
- Main process handles all native APIs
- Renderer process (web app) is sandboxed
- Communication only through whitelisted IPC channels

### Content Security Policy
- Strict CSP prevents code injection
- Only trusted domains allowed for external resources
- No inline scripts in production

### Code Signing
- Sign all executables for distribution
- Use proper certificates for each platform
- Enable auto-updater signature verification

## 🐛 Troubleshooting

### Common Issues

**App won't start:**
- Check server path configuration
- Verify all dependencies are installed
- Look for errors in development console

**IPC communication fails:**
- Ensure preload script is loaded correctly
- Verify channel names in whitelist
- Check TypeScript compilation errors

**Build fails:**
- Update electron-builder to latest version
- Check native module compatibility
- Verify build configuration in package.json

**Performance issues:**
- Enable hardware acceleration
- Optimize renderer process code
- Consider lazy loading for large applications

### Debug Mode

Enable verbose logging:
```bash
DEBUG=electron* npm run dev
```

Access developer tools:
```javascript
// In renderer process
if (!app.isPackaged) {
    mainWindow.webContents.openDevTools();
}
```

## 📚 Resources

- [Electron Documentation](https://www.electronjs.org/docs)
- [Electron Builder Configuration](https://www.electron.build/)
- [Electron Security Checklist](https://www.electronjs.org/docs/latest/tutorial/security)
- [IPC Communication Guide](https://www.electronjs.org/docs/latest/tutorial/ipc)

## 🔄 Template Updates

This template is regularly updated to include:
- Latest Electron security practices
- New desktop integration features
- Performance optimizations
- Cross-platform compatibility improvements

Generated applications inherit all template improvements through scenario-to-desktop regeneration.
