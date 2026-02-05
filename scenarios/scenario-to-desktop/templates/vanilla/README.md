# Vanilla Desktop App Template

This template provides a production-ready Electron application foundation with a modular, testable architecture. It transforms any Vrooli scenario into a cross-platform desktop application.

## Template Structure

```
vanilla/
├── main.ts                 # Main Electron process (orchestrator)
├── preload.ts              # Secure IPC bridge for renderer
├── splash.html             # Loading screen with error display
├── splash-preload.ts       # Splash window IPC bridge
├── package.json.template   # Build configuration template
├── tsconfig.json           # TypeScript settings
├── vitest.config.ts        # Test configuration
├── eslint.config.js        # Linting rules
│
├── auth/                   # Authentication module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Interfaces and types
│   ├── manager.ts          # AuthManager implementation
│   └── __tests__/          # Unit tests
│
├── bundle/                 # Bundle validation module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Manifest types
│   ├── validator.ts        # Manifest validation
│   └── __tests__/          # Unit tests
│
├── ipc/                    # Inter-process communication module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Channel definitions
│   ├── handlers.ts         # IPC handler registration
│   ├── channels.ts         # Channel constants
│   └── __tests__/          # Unit tests
│
├── runtime/                # Bundled runtime management module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Runtime types
│   ├── control-client.ts   # HTTP client for runtime API
│   ├── exit-tracker.ts     # Process exit monitoring
│   └── __tests__/          # Unit tests
│
├── splash/                 # Splash screen management module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Splash status types
│   ├── manager.ts          # SplashWindowManager
│   ├── server-readiness.ts # Server health checking
│   └── __tests__/          # Unit tests
│
├── storage/                # App data storage module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Storage interfaces
│   ├── app-storage.ts      # File operations
│   └── __tests__/          # Unit tests
│
├── telemetry/              # Deployment analytics module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # Event types
│   ├── recorder.ts         # Event recording
│   ├── uploader.ts         # Telemetry upload
│   └── __tests__/          # Unit tests
│
├── window-state/           # Window persistence module
│   ├── index.ts            # Public API exports
│   ├── types.ts            # State interfaces
│   ├── manager.ts          # WindowStateManager
│   ├── storage.ts          # State file I/O
│   ├── display.ts          # Display enumeration
│   ├── validator.ts        # Pure validation functions
│   └── __tests__/          # Unit tests
│
├── test-utils/             # Shared testing utilities
│   ├── index.ts            # Public API exports
│   ├── electron-mocks.ts   # Electron API mocks
│   ├── fs-mocks.ts         # Filesystem mocks
│   ├── async-helpers.ts    # Test timing utilities
│   └── setup.ts            # Test setup
│
├── examples/               # Feature implementation examples
│   ├── file-operations.ts  # Native file dialogs
│   ├── native-menus.ts     # Application menus
│   ├── notifications.ts    # System notifications
│   └── README.md           # Examples documentation
│
└── scripts/                # Build helper scripts
    ├── fix-rcedit.js       # Windows resource editor fix
    ├── post-build-windows.js
    ├── setup-dmg-license.js
    └── setup-wine.js
```

---

## Core Modules

### auth/ - Authentication

Secure authentication with magic links and encrypted token storage.

**Key Features:**
- Magic link authentication (opens browser for login)
- Token encryption using Electron's `safeStorage`
- Automatic token refresh scheduling
- CSRF protection with state validation
- Protocol URL handling for custom schemes

**Public API:**
```typescript
import { createAuthManager, IAuthManager } from './auth';

const auth = createAuthManager({
  storage: authStorage,
  safeStorage: electron.safeStorage,
  http: httpClient,
  shell: electron.shell,
  timer: timerImpl,
  uuid: uuidGenerator,
  config: { protocol: 'myapp', lpbsUrl: 'https://...' },
  onAuthChange: (event) => { /* handle auth state */ },
  onTokenRefresh: (tokens) => { /* handle refresh */ },
});

await auth.startLogin();       // Initiate magic link flow
await auth.handleProtocolUrl(url);  // Process callback
const user = auth.getCurrentUser();
await auth.logout();
```

**Testing Seams:** `ISafeStorage`, `IAuthHttpClient`, `IShell`, `IAuthTimer`, `IUuidGenerator`

---

### bundle/ - Bundle Validation

Validates bundled deployment manifests for offline applications.

**Key Features:**
- Bundle manifest parsing and validation
- Platform-specific binary verification
- Health check configuration extraction
- Preflight validation support

**Public API:**
```typescript
import { validateBundleManifest, BundleManifest } from './bundle';

const result = validateBundleManifest({
  manifestPath: '/path/to/bundle.json',
  fs: fileSystem,
  path: pathUtils,
  platform: process.platform,
});

if (result.valid) {
  console.log('Services:', result.manifest.services);
}
```

**Testing Seams:** `IBundleFileSystem`, `IBundlePathUtils`, `IPlatformInfo`

---

### ipc/ - Inter-Process Communication

Type-safe communication between Electron main and renderer processes.

**Key Features:**
- Structured channel definitions (FILE, SYSTEM, APP, STORAGE, AUTH)
- Handler registration for all channels
- Request/response type safety
- Dialog integration (file save/open)

**Public API:**
```typescript
import { registerAllHandlers, IPC_CHANNELS } from './ipc';

registerAllHandlers(ipcMain, {
  storage: appStorage,
  auth: authManager,
  dialog: electron.dialog,
  fs: fileSystem,
  // ... other dependencies
});
```

**Channels:**
| Channel | Direction | Purpose |
|---------|-----------|---------|
| `file:save` | Renderer → Main | Save file with native dialog |
| `file:open` | Renderer → Main | Open file with native dialog |
| `app:info` | Renderer → Main | Get app/platform info |
| `storage:read` | Renderer → Main | Read from app storage |
| `storage:write` | Renderer → Main | Write to app storage |
| `auth:status` | Renderer → Main | Get authentication status |

**Testing Seams:** `IIpcMain`, `IDialog`, `IFileFs`

---

### runtime/ - Bundled Runtime Management

Manages the bundled Go/Rust API binary for offline applications.

**Key Features:**
- Process spawning and lifecycle management
- Exit tracking with stderr capture
- HTTP client for control API (/readyz, /ports, /validate)
- Error pattern matching for crash detection

**Public API:**
```typescript
import { createRuntimeControlClient, RuntimeExitTracker } from './runtime';

const runtime = createRuntimeControlClient({
  baseUrl: 'http://localhost:18700',
  http: httpClient,
  timer: timerImpl,
});

const ready = await runtime.checkReady();
const ports = await runtime.getPorts();
const diagnostics = await runtime.getDiagnostics();

// Exit tracking
const tracker = new RuntimeExitTracker();
tracker.trackProcess(childProcess);
if (tracker.hasExitedUnexpectedly()) {
  console.error('Runtime crashed:', tracker.getExitInfo());
}
```

**Testing Seams:** `IRuntimeHttpClient`, `IProcessSpawner`, `IRuntimeFileSystem`, `ITimer`

---

### splash/ - Splash Screen Management

Professional splash screen with status updates and error display.

**Key Features:**
- Splash window creation and lifecycle
- IPC-based status communication
- Server readiness polling with retry logic
- Error display with diagnostic logs
- Copy logs and retry functionality

**Public API:**
```typescript
import { createSplashManager, checkServerReadiness } from './splash';

const splash = createSplashManager({
  windowFactory: browserWindowFactory,
  pathResolver: pathResolver,
  ipcMain: electron.ipcMain,
  clipboard: electron.clipboard,
});

await splash.create();
splash.updateStatus({ phase: 'loading', message: 'Starting services...' });

// Check server readiness
const result = await checkServerReadiness(httpClient, {
  url: 'http://localhost:3000/readyz',
  timeoutMs: 30000,
  intervalMs: 500,
  acceptableStatusCodes: [200, 204],
});

if (result.ready) {
  await splash.close();
}
```

**Testing Seams:** `IWindowFactory`, `IPathResolver`, `IHttpClient`, `ITimer`, `IClipboard`

---

### storage/ - App Data Storage

Sandboxed file storage for application data.

**Key Features:**
- OS-appropriate data directories
- Directory/file operations (read, write, delete, list)
- Storage statistics (size, file count)
- Text and binary file support

**Public API:**
```typescript
import { createAppStorage } from './storage';

const storage = createAppStorage({
  appName: 'my-app',
  fs: fileSystem,
  path: pathUtils,
});

await storage.ensureDir('config');
await storage.writeFile('config/settings.json', JSON.stringify(settings));
const data = await storage.readFile('config/settings.json');
const stats = await storage.getStats();
```

**Storage Locations:**
- **Linux:** `~/.config/<app-name>/`
- **macOS:** `~/Library/Application Support/<app-name>/`
- **Windows:** `%APPDATA%\<app-name>\`

**Testing Seams:** `IStorageFileSystem`, `IStoragePathUtils`

---

### telemetry/ - Deployment Analytics

Event recording and upload for deployment monitoring.

**Key Features:**
- JSONL event recording to user data directory
- Standard event types (app_start, server_ready, app_shutdown, etc.)
- Session tracking with unique IDs
- Telemetry upload with retry logic

**Public API:**
```typescript
import { createTelemetryRecorder, createTelemetryUploader } from './telemetry';

const recorder = createTelemetryRecorder({
  appName: 'my-app',
  fs: fileSystem,
  path: pathUtils,
});

await recorder.record({
  event: 'app_start',
  timestamp: new Date().toISOString(),
  session_id: sessionId,
  app_version: '1.0.0',
});

const uploader = createTelemetryUploader({
  endpoint: 'https://api.example.com/telemetry',
  http: httpClient,
});
await uploader.upload(recorder.getFilePath());
```

**Standard Events:**
| Event | Description |
|-------|-------------|
| `app_start` | Application launched |
| `server_ready` | Backend server reachable |
| `dependency_unreachable` | Connection to service failed |
| `app_ready` | Fully initialized and ready |
| `startup_error` | Error during startup |
| `app_shutdown` | Clean application exit |

**Testing Seams:** `IFileSystem`, `IHttpClient`, `IPathUtils`

---

### window-state/ - Window Persistence

Remembers window position, size, and state across restarts.

**Key Features:**
- State persistence to app data directory
- Multi-monitor support with display tracking
- Graceful handling of disconnected displays
- Validation and clamping to visible area

**Public API:**
```typescript
import { WindowStateManager, createWindowStateStorage, createDisplayProvider } from './window-state';

const storage = createWindowStateStorage(app, fs, path);
const displayProvider = createDisplayProvider(screen);

const manager = new WindowStateManager(
  { storage, displayProvider },
  { defaultWidth: 1200, defaultHeight: 800, minWidth: 400, minHeight: 300 }
);

const state = await manager.getInitialState();
const window = new BrowserWindow({ x: state.x, y: state.y, width: state.width, height: state.height });

manager.manage(window);

// After window.show():
if (manager.wasMaximized()) window.maximize();
if (manager.wasFullScreen()) window.setFullScreen(true);
```

**State Schema:**
```typescript
interface WindowState {
  x?: number;
  y?: number;
  width: number;
  height: number;
  isMaximized: boolean;
  isFullScreen: boolean;
  displayId?: number;
}
```

**Testing Seams:** `IStateStorage`, `IDisplayProvider`, `IManagedWindow`, `IFileSystem`

---

### test-utils/ - Testing Utilities

Shared mocks and helpers for unit testing.

**Exports:**
```typescript
import {
  // Electron mocks
  createMockBrowserWindow,
  createMockIpcMain,
  createMockApp,
  createMockDialog,
  createMockShell,
  createMockScreen,
  createMockSafeStorage,

  // Filesystem mocks
  createMockFs,
  createMockPath,

  // Async helpers
  waitFor,
  delay,
} from './test-utils';
```

---

## Template Variables

Variables replaced during generation (Mustache syntax):

### Application Identity
| Variable | Description | Example |
|----------|-------------|---------|
| `{{APP_NAME}}` | Technical name (lowercase) | `picker-wheel` |
| `{{APP_DISPLAY_NAME}}` | User-facing name | `Picker Wheel` |
| `{{VERSION}}` | Semantic version | `1.0.0` |
| `{{APP_ID}}` | Unique identifier | `com.vrooli.picker-wheel` |

### Server Configuration
| Variable | Description | Example |
|----------|-------------|---------|
| `{{DEPLOYMENT_MODE}}` | Connection strategy | `external-server` / `bundled` |
| `{{SERVER_TYPE}}` | UI loading method | `external` / `static` / `node` |
| `{{SERVER_PATH}}` | Server URL or path | `https://api.example.com` |
| `{{PORTS_CONFIG}}` | Port configuration JSON | `{"api":{"envVar":"API_PORT","port":18700}}` |

### Window Configuration
| Variable | Description | Default |
|----------|-------------|---------|
| `{{WINDOW_WIDTH}}` | Initial width | `1200` |
| `{{WINDOW_HEIGHT}}` | Initial height | `800` |
| `{{WINDOW_BACKGROUND}}` | Background color | `#ffffff` |

### Features (boolean)
| Variable | Description | Default |
|----------|-------------|---------|
| `{{ENABLE_SPLASH}}` | Show splash screen | `true` |
| `{{ENABLE_MENU}}` | Native menus | `true` |
| `{{ENABLE_SYSTEM_TRAY}}` | System tray icon | `false` |
| `{{ENABLE_DEV_TOOLS}}` | Developer tools | `false` |

---

## Integration Patterns

### Dependency Injection

All modules use factory functions with explicit dependencies:

```typescript
// Production
const storage = createAppStorage({ fs: require('fs'), path: require('path'), appName });
const auth = createAuthManager({ storage, safeStorage: electron.safeStorage, ... });

// Testing
const mockStorage = createMockStorage();
const auth = createAuthManager({ storage: mockStorage, safeStorage: mockSafeStorage, ... });
```

### Module Composition in main.ts

```typescript
// 1. Create foundational modules
const storage = createAppStorage(config);
const telemetry = createTelemetryRecorder(telemetryConfig);

// 2. Create modules with dependencies
const runtime = createRuntimeControlClient(runtimeConfig);
const auth = createAuthManager({ storage, ... });
const windowState = new WindowStateManager({ storage: stateStorage, displayProvider });

// 3. Register IPC handlers with all modules
registerAllHandlers(ipcMain, { storage, auth, runtime, telemetry, ... });

// 4. Create windows with state
const state = await windowState.getInitialState();
mainWindow = new BrowserWindow({ ...state });
windowState.manage(mainWindow);
```

### Startup Sequence

```
App Launch
    ↓
WindowStateManager.getInitialState()
    ↓
SplashManager.create()
    ↓
TelemetryRecorder.record('app_start')
    ↓
RuntimeControlClient.spawn() [if bundled]
    ↓
checkServerReadiness()
    ↓
SplashManager.close()
    ↓
MainWindow.show()
    ↓
TelemetryRecorder.record('app_ready')
```

---

## Development

### Running Tests

```bash
# Run all tests
npm test

# Run with coverage
npm run test:coverage

# Run specific module tests
npm test -- --filter auth
npm test -- --filter splash
```

### Building

```bash
# Development build
npm run build

# Production build for current platform
npm run dist

# Platform-specific builds
npm run dist:win
npm run dist:mac
npm run dist:linux
```

---

## Related Documentation

- [DOC: docs/concepts/ARCHITECTURE.md] - System architecture diagrams
- [DOC: docs/internal/SEAMS.md] - Complete seam documentation
- [DOC: docs/concepts/GLOSSARY.md] - Term definitions
- [DOC: docs/desktop-integration-guide.md] - Feature cookbook
