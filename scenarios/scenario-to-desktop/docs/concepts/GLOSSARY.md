# Glossary

Key terms and concepts used throughout the scenario-to-desktop documentation.

## Core Concepts

### Desktop Wrapper
An Electron application that wraps an existing web application, providing native OS integration without reimplementing the UI. The web app runs unchanged inside the wrapper.

### Template
A pre-configured set of Electron source files (main.ts, preload.ts, etc.) that define the desktop application's structure and capabilities. Templates are customized via variable substitution during generation.

### Generation
The process of creating a desktop wrapper from a scenario's web UI. Takes the scenario's `ui/dist/` and configuration to produce a complete `platforms/electron/` directory.

### Deployment Mode
How the desktop app connects to its backend:
- **external-server**: Thin client connecting to a remote Vrooli server
- **bundled**: Standalone app with embedded API and runtime (in development)
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

## Architecture Components

### Main Process
The Node.js process that manages the Electron application lifecycle, creates windows, handles native OS integration, and communicates with renderer processes via IPC.

### Renderer Process
The sandboxed browser process that runs your web application. Has no direct access to Node.js APIs—must communicate through the preload script.

### Preload Script
A secure bridge between the main process and renderer. Exposes a controlled set of APIs via `window.desktop` while maintaining security through context isolation.

### IPC (Inter-Process Communication)
The mechanism for communication between main and renderer processes. Uses named channels with whitelisted messages for security.

### Window State Manager
A module that persists window position, size, and state (maximized/fullscreen) across application restarts. Handles multi-monitor scenarios gracefully.

### Splash Manager
Controls the splash screen shown during application startup. Displays status updates and handles error states with diagnostic information.

---

## Build System

### Electron Builder
The tool used to package Electron applications into distributable installers (MSI, PKG, AppImage, DEB, etc.).

### Template Generator
TypeScript tool (`templates/build-tools/template-generator.ts`) that performs variable substitution and creates the `platforms/electron/` directory from templates.

### Pipeline
The multi-stage build process: Bundle → Preflight → Generate → Build → SmokeTest → Distribution.

### Preflight
Validation stage that checks prerequisites before building: UI dist exists, dependencies available, services reachable (for thin clients).

### Smoke Test
Automated verification that the built desktop app launches correctly, connects to its backend, and reaches a ready state.

---

## Deployment Terms

### Thin Client
A desktop app that contains only the UI and connects to a remote Vrooli server for all API operations. Requires network connectivity.

### Bundled App
A self-contained desktop app with embedded UI, API binary, and runtime. Operates completely offline. (In development)

### App Monitor
Vrooli's Cloudflare tunnel service that provides public HTTPS URLs for locally-running scenarios. Used to expose thin client backends.

### Telemetry
Event tracking written to `deployment-telemetry.jsonl` in the app's user data directory. Captures startup events, errors, and shutdown for deployment monitoring.

---

## File Locations

### platforms/electron/
The generated desktop wrapper directory within a scenario. Contains all Electron source code and configuration.

### dist-electron/
Build output directory containing compiled installers and unpacked builds for each platform.

### ui/dist/
The built web application assets (HTML, JS, CSS) that the desktop wrapper loads. Must exist before desktop generation.

### .vrooli/service.json
Scenario configuration file containing port definitions, lifecycle settings, and metadata used during desktop generation.

---

## Seams (Testing Boundaries)

### IStateStorage
Interface abstracting window state persistence for testing without filesystem access.

### IDisplayProvider
Interface abstracting screen/display enumeration for testing multi-monitor scenarios.

### IManagedWindow
Interface abstracting BrowserWindow for testing window management without Electron.

### IHttpClient
Interface abstracting HTTP requests for testing server readiness checks.

---

## Related Documentation

- [DOC: docs/concepts/ARCHITECTURE.md] - Visual architecture diagrams
- [DOC: docs/deployment-modes.md] - Deployment mode details
- [DOC: docs/internal/SEAMS.md] - Complete seam documentation
