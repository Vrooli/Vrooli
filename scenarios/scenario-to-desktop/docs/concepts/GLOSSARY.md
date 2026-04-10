# Glossary

Key terms and concepts used throughout the scenario-to-desktop documentation.

## Core Concepts

### Desktop Wrapper
An Electron application that wraps an existing web application, providing native OS integration without reimplementing the UI. The web app runs unchanged inside the wrapper.

### Template
A pre-configured set of Electron source files (main.ts, preload.ts, modules/) that define the desktop application's structure and capabilities. Templates are customized via variable substitution during generation.

### Generation
The process of creating a desktop wrapper from a scenario's web UI. Takes the scenario's `ui/dist/` and configuration to produce a complete `platforms/electron/` directory.

### Deployment Mode
How the desktop app connects to its backend:
- **external-server**: Thin client connecting to a remote Vrooli server
- **bundled**: Standalone app with embedded API and runtime (recommended default)
- **cloud-api**: Future mode for cloud-native APIs

### Server Type
How the desktop app loads its UI content:
- **external**: UI and API from remote URLs
- **static**: UI from local files, no API
- **node**: Embedded Node.js server
- **executable**: Embedded compiled binary (Go, Rust, etc.)

---

## Template Types

### Universal Template
Default template suitable for 95% of scenarios. Provides native menus, file dialogs, notifications, auto-updater, and window state persistence. Window size: 1200x800.

### Advanced Template
Extends Universal with system tray integration, global keyboard shortcuts, deep linking support, and platform-specific features (Windows Jump Lists, macOS Dock menus).

### Multi-Window Template
For complex applications requiring multiple windows. Adds inter-window communication, floating tool panels, and multi-monitor support.

### Kiosk Template
For dedicated hardware deployments. Provides full-screen lock mode, security hardening, auto-restart on crash, and remote monitoring capabilities.

---

## Template Modules

The vanilla template includes 8 core modules, each with clear responsibilities and testable interfaces.

### Auth Module (`auth/`)
Handles secure user authentication with magic link flows. Encrypts tokens using Electron's `safeStorage`, manages automatic refresh scheduling, and handles protocol URL callbacks for the login flow.

### Bundle Module (`bundle/`)
Validates bundled deployment manifests for offline applications. Parses `bundle.json`, verifies platform-specific binaries exist, and extracts health check configurations.

### IPC Module (`ipc/`)
Manages type-safe inter-process communication between the main Electron process and renderer (web app). Defines channel constants, handler registration, and message types for file operations, storage, authentication, and system commands.

### Runtime Module (`runtime/`)
Controls the bundled API binary lifecycle for offline applications. Spawns the Go/Rust binary, monitors for unexpected exits, and provides an HTTP client for runtime control endpoints (`/readyz`, `/ports`, `/validate`).

### Splash Module (`splash/`)
Manages the splash screen shown during application startup. Creates the splash window, sends status updates via IPC, checks server readiness with retry logic, and displays error information with diagnostic logs.

### Storage Module (`storage/`)
Provides sandboxed file storage for application data. Abstracts OS-specific data directories, handles file read/write operations, and provides storage statistics.

### Telemetry Module (`telemetry/`)
Records deployment analytics events to a JSONL file and uploads them to the telemetry API. Tracks standard events like `app_start`, `server_ready`, `app_shutdown`, and errors.

### Window-State Module (`window-state/`)
Persists window position, size, and state (maximized/fullscreen) across application restarts. Handles multi-monitor scenarios and gracefully recovers when displays are disconnected.

### Test-Utils Module (`test-utils/`)
Shared testing utilities including mock factories for Electron APIs (BrowserWindow, ipcMain, dialog, shell, screen, safeStorage), filesystem mocks, and async test helpers.

---

## Architecture Components

### Main Process
The Node.js process that manages the Electron application lifecycle, creates windows, handles native OS integration, and communicates with renderer processes via IPC. Orchestrates all template modules.

### Renderer Process
The sandboxed browser process that runs your web application. Has no direct access to Node.js APIs—must communicate through the preload script.

### Preload Script
A secure bridge between the main process and renderer. Exposes a controlled set of APIs via `window.desktop` while maintaining security through context isolation.

### IPC (Inter-Process Communication)
The mechanism for communication between main and renderer processes. Uses named channels with whitelisted messages for security.

---

## Module Interfaces (Seams)

Interfaces that abstract dependencies for testing. Each module defines seams that can be mocked in unit tests.

### IAuthManager
Interface for the authentication module. Methods: `startLogin()`, `handleProtocolUrl()`, `getCurrentUser()`, `logout()`, `isAuthenticated()`.

### ISafeStorage
Abstracts Electron's `safeStorage` for encrypted token storage. Methods: `encryptString()`, `decryptString()`, `isEncryptionAvailable()`.

### IBundleValidator
Interface for bundle manifest validation. Methods: `validate()` returning validation results with errors and warnings.

### IRuntimeControlClient
HTTP client interface for runtime control API. Methods: `checkReady()`, `getPorts()`, `getDiagnostics()`, `validate()`.

### RuntimeExitTracker
Tracks runtime process exit status. Methods: `trackProcess()`, `hasExitedUnexpectedly()`, `getExitInfo()`.

### ISplashWindowManager
Interface for splash window lifecycle. Methods: `create()`, `updateStatus()`, `close()`, `isVisible()`, `onEscapePressed()`.

### IHttpClient
Abstracts HTTP requests for server readiness checks. Methods: `get(url, timeoutMs)` returning status code and body.

### IAppStorage
Interface for app data storage operations. Methods: `readFile()`, `writeFile()`, `deleteFile()`, `listDir()`, `ensureDir()`, `getStats()`.

### IStorageFileSystem
Low-level filesystem abstraction for storage module. Methods: `readFile()`, `writeFile()`, `mkdir()`, `stat()`, `readdir()`.

### ITelemetryRecorder
Interface for event recording. Methods: `record(event)`, `getFilePath()`, `getEvents()`.

### ITelemetryUploader
Interface for telemetry upload. Methods: `upload(filePath)`, `setEndpoint()`.

### IWindowStateManager
Interface for window state persistence. Methods: `getInitialState()`, `manage(window)`, `wasMaximized()`, `wasFullScreen()`.

### IStateStorage
Abstracts window state file persistence. Methods: `load()`, `save(state)`.

### IDisplayProvider
Abstracts screen/display enumeration. Methods: `getAllDisplays()`, `getPrimaryDisplay()`, `getDisplayAtPoint()`.

### IManagedWindow
Abstracts BrowserWindow for window state tracking. Methods: `getNormalBounds()`, `isMaximized()`, `isFullScreen()`, `isDestroyed()`, `on()`, `removeListener()`.

### ITimer
Time abstraction for deterministic testing. Methods: `now()`, `sleep(ms)`.

---

## Build System

### Electron Builder
The tool used to package Electron applications into distributable installers (MSI, PKG, AppImage, DEB, etc.).

### Template Generator
TypeScript tool (`templates/build-tools/template-generator.ts`) that performs variable substitution and creates the `platforms/electron/` directory from templates.

### Pipeline
The multi-stage build process: Bundle → Preflight → Generate → Build → SmokeTest → Deploy.

### Preflight
Validation stage that checks prerequisites before building: UI dist exists, dependencies available, services reachable (for thin clients).

### Smoke Test
Automated verification that the built desktop app launches correctly, connects to its backend, and reaches a ready state.

---

## Deployment Terms

### Thin Client
A desktop app that contains only the UI and connects to a remote Vrooli server for all API operations. Requires network connectivity.

### Bundled App
A self-contained desktop app with embedded UI, API binary, and runtime. Operates completely offline. (Recommended default)

### App Monitor
Vrooli's Cloudflare tunnel service that provides public HTTPS URLs for locally-running scenarios. Used to expose thin client backends.

### Telemetry
Event tracking written to `deployment-telemetry.jsonl` in the app's user data directory. Captures startup events, errors, and shutdown for deployment monitoring.

---

## File Locations

### platforms/electron/
The generated desktop wrapper directory within a scenario. Contains all Electron source code, modules, and configuration.

### dist-electron/
Build output directory containing compiled installers and unpacked builds for each platform.

### ui/dist/
The built web application assets (HTML, JS, CSS) that the desktop wrapper loads. Must exist before desktop generation.

### .vrooli/service.json
Scenario configuration file containing port definitions, lifecycle settings, and metadata used during desktop generation.

---

## Events

Standard telemetry events recorded by the telemetry module:

### app_start
Recorded when the application launches, before any other initialization.

### server_ready
Recorded when the backend server (remote or bundled) responds successfully to health checks.

### dependency_unreachable
Recorded when connection to a required service fails.

### app_ready
Recorded when the application is fully initialized and the main window is shown.

### startup_error
Recorded when an error occurs during startup, with error details.

### app_shutdown
Recorded when the application exits cleanly.

### bundled_runtime_exit
Recorded when the bundled runtime process exits unexpectedly, with exit code and stderr.

---

## Related Documentation

- [DOC: docs/concepts/ARCHITECTURE.md] - Visual architecture diagrams
- [DOC: deployment-modes.md] - Deployment mode details
- [DOC: docs/internal/SEAMS.md] - Complete seam documentation
- [DOC: templates/vanilla/README.md] - Module implementation details
