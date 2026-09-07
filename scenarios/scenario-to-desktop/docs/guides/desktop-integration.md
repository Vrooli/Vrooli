# Desktop Integration Guide

## Overview

This guide is a feature cookbook for enhancing the generated Electron wrapper. Use it after completing the thin-client quickstart in `docs/QUICKSTART.md`.

## File Structure (generated)

```
scenarios/<scenario>/
└── platforms/electron/
    ├── main.ts             # Electron main process
    ├── preload.ts          # Secure IPC bridge
    ├── package.json        # Desktop dependencies
    ├── tsconfig.json
    ├── splash.html
    ├── assets/             # Icons for all platforms
    ├── dist/               # Compiled TypeScript (git-ignored)
    └── dist-electron/      # Built packages (git-ignored)
```

## Desktop Features

## Monetization journey identifiers

The desktop readiness journey uses these stable step identifiers:

- `signin_shared_session`
- `second_app_resolves`
- `tampered_class_a`
- `class_b_local`
- `offline_class_b`
- `offline_gate_degrades`
- `outbox_drains_once`
- `expired_lease_falls_back`

Paid bundle guidance must also identify an account-surface path and a
journey-probe path. Those paths are checked by monetization conformance.

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

## Deploy and Signing

### Code Signing

If you do not have certificates yet, you can build unsigned installers for local testing (expect OS warnings). When you
are ready to ship:

1) Capture cert details in the Signing tab or CLI (`scenario-to-desktop signing set <scenario>`).  
2) Make sure platform tools are installed (`signtool` on Windows, `codesign`/`notarytool` on macOS, `gpg` on Linux).  
3) Re-run the build with signing enabled.

Minimum inputs per platform:
- **Windows**: `.pfx/.p12` path + password env var (e.g., `WIN_CERT_PASSWORD`), or a certificate thumbprint from the Windows cert store.
- **macOS**: Developer ID identity (`security find-identity -v -p codesigning`) and Team ID. Notarization is optional until store submission.
- **Linux**: GPG key ID or fingerprint from `gpg --list-secret-keys`; optional custom keyring path.

For detailed, up-to-date steps and troubleshooting, see the dedicated signing guide:
[scenarios/scenario-to-desktop/docs/guides/code-signing.md](code-signing.md).

Sample minimal `signing.json` (generated by the Signing tab/CLI):
```json
{
  "enabled": true,
  "windows": {
    "certificate_source": "file",
    "certificate_file": "/path/to/cert.pfx",
    "certificate_password_env": "WIN_CERT_PASSWORD",
    "timestamp_server": "http://timestamp.digicert.com",
    "sign_algorithm": "sha256"
  },
  "macos": {
    "identity": "Developer ID Application: Your Name (TEAMID)",
    "team_id": "TEAMID",
    "hardened_runtime": true,
    "notarize": false
  },
  "linux": {
    "gpg_key_id": "YOURKEYID"
  }
}
```
For sandbox testing, you can use a self-signed PFX on Windows and a local GPG key on Linux; expect OS warnings until you use trusted certificates.

Quick self-signed helpers (for testing only, not production):
```bash
# Windows (on any OS): create PFX (prompts for password)
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=Test Cert"
openssl pkcs12 -export -out test-desktop.pfx -inkey key.pem -in cert.pem

# Linux/macOS GPG: create a local signing key
gpg --quick-generate-key "Test Desktop <test@example.com>" default default 1y
gpg --list-secret-keys
```
Point `certificate_file` to `test-desktop.pfx` and set the password env var; use the GPG key ID from `--list-secret-keys`. Expect OS warnings until you swap to trusted certs.

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
export CSC_KEY_PASSWORD="$WINDOWS_CERTIFICATE_PASSWORD"

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

## Built-in presentation extension (contract versions 1 and 2)

Use `pipeline run <scenario> --native-extension-file <declaration.json>` or
set `native_extension` in the generation or pipeline configuration to select the
Scenario-to-Desktop-owned presentation module:

```json
{
  "version": 1,
  "module": "presentation",
  "permissions": ["window.presentation"],
  "platforms": ["linux", "mac", "win"]
}
```

The declared platforms must cover every requested build target (or the current
host when targets are omitted). Unknown versions, modules, permissions, duplicate
platforms, and extra JSON fields are rejected. Extension configuration cannot
supply imports, entrypoints, helper binaries, or template patches. A
`helper_providers` declaration may select only the governed
`device-control` / `desktop.session` provider; it carries ownership metadata
and never supplies a path or executable. The generator writes
`native-extension.json` with the selected source entrypoint, bridge, and
helper-provider metadata; native helper bytes remain ramp-owned artifacts.

An enabled renderer receives `window.desktopPresentation.get()` and
`window.desktopPresentation.set("expanded" | "palette" | "pill" | "hidden")`. Each promise
returns `{ version: 1, mode, revision, shortcut }`.
`subscribe(listener)` receives authoritative snapshots after native changes and
returns an unsubscribe function. Renderers should ignore older revisions. The shell resizes the existing BrowserWindow,
retains its renderer, restores the expanded bounds, and clamps bounds after
display removal. Only the application's main frame on its initial document origin
(or initial file path) may call this bridge. Exit fullscreen before changing modes.
Vanilla builds do not expose this bridge or register presentation IPC handlers.

This is a window presentation primitive. The application must provide compact
layouts and preserve conversation, branch, attachment and active-run state.
Version 2 adds global activation with this declaration:

```json
{
  "version": 2,
  "module": "presentation",
  "permissions": ["window.presentation", "global-shortcut"],
  "platforms": ["linux", "mac", "win"],
  "activation_shortcut": "CommandOrControl+Shift+Space"
}
```

The snapshot advertises `canHide` only while the activation shortcut is registered.
`set("hidden")` hides the native window without destroying its renderer; hiding is
refused when no shortcut is registered. Applications should offer Hide only when
`canHide` is true. Activation restores hidden or minimized windows. This preserves
mounted application state, but does not by itself guarantee background task
execution or define close-button and quit behavior.

Activation restores a minimized window, opens its palette, and focuses the same
renderer. A fullscreen window is focused without resizing. The shortcut snapshot
reports `registered`, `unavailable`, or `disabled`; registration refusal leaves
manual presentation controls available. Unavailability may reflect an existing
binding or desktop policy. Closing the window releases only its own registered
shortcut. Version 1 does not register a shortcut.

Closing a compact view saves its last expanded bounds and maximized state through
the existing window-state manager. Startup opens the expanded view and validates
those bounds against the available displays. Compact geometry does not become the
next expanded window size. This geometry persistence does not persist application
conversation or task state.

Presentation-enabled shells also attempt to create the existing system tray when
an application icon is available. Its Show, Palette, Pill, and Expanded actions
use the same presentation controller; double-click opens the palette. Application
activation and second-instance activation reveal the expanded view. Tray support
depends on the desktop environment. Hidden-mode admission still requires the
registered shortcut, so tray creation alone is not treated as proof of recovery.

The tray and Window menus offer **Keep running when window closes (this session)**
with `CommandOrControl+Shift+B`. It starts disabled and requires a registered
activation shortcut. When enabled, window close hides the existing renderer;
explicit Quit still requests exit through any registered guard. Disabling the
option restores normal close behavior.
Fullscreen close retains fullscreen state for recovery. The option does not
persist across process restarts.

Renderers may opt into guarded Quit through `onQuit(listener)` and answer its
request ID with `decideQuit(id, "quit" | "cancel" | "background")`. Native Quit
and ordinary window close wait after registration; background-close preference
still hides without requesting exit. Stale decisions and untrusted frames are
rejected. The shell resumes exit only after the matching quit decision. A repeated
Quit request reissues the pending ID, and renderer reload registration receives
any pending request.

Portal offers Cancel, Keep running in background, and Stop tasks and quit. It
authorizes exit only when the task registry is empty; callback completion alone
is insufficient. The app-shell agent task provider survives route changes and
recovers durable admissions after renderer restart. Enumeration must finish before
Quit can proceed. Owner reads reconcile every recovered admission; unknown state
remains registered, and a failed page offers a read-only retry. Lost Stop responses
are reconciled without repeating Stop. Desktop lease recovery after renderer restart
is still incomplete, so this does not establish all-task crash recovery.

Persistent background preferences and prior-application focus capture/restoration
remain outside these versions. The three declared
platforms select portable Electron APIs; they do not constitute physical
cross-platform acceptance evidence. Existing signing and update configuration
still applies to generated applications.

## Next Steps

1. **Customize your desktop app** - Edit `platforms/electron/main.ts` and `splash.html`
2. **Add desktop-specific features** - File system, notifications, menus, tray
3. **Test across platforms** - Windows, macOS, Linux
4. **Setup auto-updates** - Configure GitHub releases or custom update server
5. **Submit to app stores** - Follow platform-specific guidelines

## Resources

- [Electron Documentation](https://www.electronjs.org/docs)
- [Electron Builder](https://www.electron.build/)
- [Vrooli Documentation](../../../../docs/README.md)
- [scenario-to-desktop README](../../README.md)

---

**Need help?** Open an issue at https://github.com/vrooli/vrooli/issues
