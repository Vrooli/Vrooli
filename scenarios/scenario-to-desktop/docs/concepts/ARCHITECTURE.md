# Architecture Overview

This document provides a visual guide to the scenario-to-desktop system architecture, template structure, and key workflows.

## System Purpose

**scenario-to-desktop** transforms any Vrooli scenario into a cross-platform desktop application (Windows/macOS/Linux). Desktop apps are **superset wrappers**, not reimplementations—your web application runs unchanged inside Electron, enhanced with native OS capabilities.

**Key Implementation Files:**
- [CODE: api/main.go] - API server entry point
- [CODE: templates/vanilla/main.ts] - Electron main process template
- [CODE: templates/vanilla/preload.ts] - Secure IPC bridge template
- [CODE: templates/build-tools/template-generator.ts] - Template generation logic

```
┌─────────────────────────────────────────────────────────────────┐
│                    scenario-to-desktop                          │
│  "Desktop apps are superset wrappers, not reimplementations"    │
└─────────────────────────────────────────────────────────────────┘
                              │
     ┌────────────────────────┼────────────────────────┐
     ▼                        ▼                        ▼
┌─────────┐            ┌─────────────┐           ┌──────────┐
│   API   │            │  Templates  │           │    UI    │
│  (Go)   │            │  (Vanilla)  │           │ (React)  │
└─────────┘            └─────────────┘           └──────────┘
```

---

## Directory Structure

```
scenario-to-desktop/
├── api/                          # Go API server (port 15000-19999)
│   ├── generation/               # Desktop wrapper generation logic
│   ├── build/                    # Build orchestration
│   ├── bundle/                   # Runtime bundling for offline apps
│   ├── pipeline/                 # Multi-stage build pipeline
│   ├── telemetry/                # Event tracking & analytics
│   ├── preflight/                # Pre-build validation
│   └── domain/                   # Core business logic
│
├── cli/                          # Command-line interface
│
├── ui/                           # React web management interface (port 35000-39999)
│   └── src/
│       ├── components/           # UI elements
│       ├── hooks/                # State management
│       ├── domain/               # Business logic
│       ├── store/                # Zustand stores
│       └── lib/                  # Utilities
│
├── templates/                    # Template system
│   ├── vanilla/                  # Base Electron implementation
│   │   ├── main.ts               # Electron main process
│   │   ├── preload.ts            # Secure IPC bridge
│   │   ├── splash.html           # Professional splash screen
│   │   ├── splash-preload.ts     # Splash IPC bridge
│   │   │
│   │   ├── auth/                 # Authentication module
│   │   ├── bundle/               # Bundle validation module
│   │   ├── ipc/                  # IPC handler module
│   │   ├── runtime/              # Bundled runtime module
│   │   ├── splash/               # Splash management module
│   │   ├── storage/              # App storage module
│   │   ├── telemetry/            # Analytics module
│   │   ├── window-state/         # Window persistence module
│   │   ├── test-utils/           # Shared testing utilities
│   │   │
│   │   ├── examples/             # Feature implementation examples
│   │   └── scripts/              # Build helper scripts
│   │
│   ├── advanced/                 # Template configuration files
│   │   ├── universal-app.json    # Default template (95% of use cases)
│   │   ├── advanced-app.json     # Professional features
│   │   ├── multi-window.json     # Complex workflows
│   │   └── kiosk-mode.json       # Full-screen deployments
│   │
│   └── build-tools/              # Template generator system
│
├── runtime/                      # Bundled runtime for offline apps (future)
│
└── docs/                         # Documentation
    ├── OVERVIEW.md               # Current vs roadmap
    ├── QUICKSTART.md             # Getting started
    ├── concepts/                 # Architecture, glossary
    ├── reference/                # API, pipeline docs
    └── internal/                 # SEAMS.md, ASSUMPTIONS.md
```

---

## Template Module Architecture

The vanilla template uses a **modular, seam-based architecture** where each module has:
- Clear single responsibility
- Explicit interfaces for dependencies (seams)
- Co-located unit tests
- Factory functions for dependency injection

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           templates/vanilla/                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐                 │
│    │   main.ts    │───▶│  preload.ts  │───▶│  splash.html │                 │
│    │ (Orchestrator)    │  (IPC Bridge) │    │ (Loading UI) │                 │
│    └───────┬──────┘    └──────────────┘    └──────────────┘                 │
│            │                                                                 │
│            │  Uses 8 Core Modules:                                           │
│            ▼                                                                 │
│    ┌───────────────────────────────────────────────────────────────────┐    │
│    │                        CORE MODULES                                │    │
│    ├───────────────────────────────────────────────────────────────────┤    │
│    │                                                                    │    │
│    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐│    │
│    │  │    auth/    │  │   bundle/   │  │    ipc/     │  │  runtime/ ││    │
│    │  │             │  │             │  │             │  │           ││    │
│    │  │ Magic link  │  │ Manifest    │  │ Type-safe   │  │ Process   ││    │
│    │  │ Token mgmt  │  │ validation  │  │ channels    │  │ lifecycle ││    │
│    │  │ Auto-refresh│  │ Platform    │  │ Handlers    │  │ Exit track││    │
│    │  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘│    │
│    │                                                                    │    │
│    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐│    │
│    │  │   splash/   │  │  storage/   │  │ telemetry/  │  │window-    ││    │
│    │  │             │  │             │  │             │  │state/     ││    │
│    │  │ Window mgmt │  │ App data    │  │ Events      │  │           ││    │
│    │  │ Status IPC  │  │ Sandboxed   │  │ Sessions    │  │ Persist   ││    │
│    │  │ Readiness   │  │ Cross-plat  │  │ Upload      │  │ Multi-mon ││    │
│    │  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘│    │
│    │                                                                    │    │
│    │  ┌─────────────┐                                                   │    │
│    │  │ test-utils/ │  Shared mocks: electron, fs, async helpers       │    │
│    │  └─────────────┘                                                   │    │
│    │                                                                    │    │
│    └───────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Module Responsibilities

| Module | Responsibility | Key Interfaces |
|--------|----------------|----------------|
| **auth/** | Secure authentication with magic links and encrypted tokens | `IAuthManager`, `ISafeStorage` |
| **bundle/** | Validate bundled deployment manifests for offline apps | `IBundleValidator`, `BundleManifest` |
| **ipc/** | Type-safe main↔renderer communication | `IpcHandlers`, `IPC_CHANNELS` |
| **runtime/** | Manage bundled API binary lifecycle and health | `IRuntimeControlClient`, `RuntimeExitTracker` |
| **splash/** | Splash window with status updates and error display | `ISplashWindowManager`, `ReadinessChecker` |
| **storage/** | Sandboxed app data persistence | `IAppStorage`, `IStorageFileSystem` |
| **telemetry/** | Deployment event recording and upload | `ITelemetryRecorder`, `ITelemetryUploader` |
| **window-state/** | Window position/size persistence across restarts | `IWindowStateManager`, `IStateStorage` |
| **test-utils/** | Shared testing mocks and helpers | Mock factories for all modules |

---

## Template Types

```
templates/
├── vanilla/                    ◄── BASE IMPLEMENTATION (Electron)
│   ├── main.ts                     Main process (window creation, menus)
│   ├── preload.ts                  IPC bridge (web ↔ native)
│   ├── splash.html                 Professional loading screen
│   └── [8 core modules]            Modular, testable architecture
│
└── advanced/                   ◄── FEATURE CONFIGURATIONS
    ├── universal-app.json          Default (95% of cases)
    ├── advanced-app.json           System tray, global shortcuts
    ├── multi-window.json           Multiple windows (IDEs)
    └── kiosk-mode.json             Full-screen locked mode
```

### Template Types Visual Comparison

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           UNIVERSAL (Default)                           │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ ┌─ File  Edit  View ─────────────────────────────── _ □ X ─────┐ │  │
│  │ │                                                               │ │  │
│  │ │              Your Web App Runs Unchanged                      │ │  │
│  │ │                                                               │ │  │
│  │ │    + Native file dialogs                                      │ │  │
│  │ │    + System notifications                                     │ │  │
│  │ │    + Auto-updater                                             │ │  │
│  │ │    + Window state persistence                                 │ │  │
│  │ │                                                               │ │  │
│  │ └───────────────────────────────────────────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│  Use for: picker-wheel, qr-code-generator, notes, nutrition-tracker   │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                              ADVANCED                                   │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Same as Universal, PLUS:                                         │  │
│  │                                                                  │  │
│  │   [Tray] System tray (minimize to tray, background operation)    │  │
│  │   [Key]  Global keyboard shortcuts (work when app minimized)     │  │
│  │   [Link] Deep linking (myapp://action URLs)                      │  │
│  │   [Menu] Platform-specific (Windows Jump Lists, macOS Dock)      │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│  Use for: system-monitor, document-manager, research-assistant         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                           MULTI-WINDOW                                  │
│  ┌────────────────────┐  ┌────────────────────┐  ┌─────────────────┐   │
│  │   Main Window      │  │  Secondary Window  │  │ Floating Panel  │   │
│  │   (Primary)        │  │  (Tool/Inspector)  │  │ (Always on top) │   │
│  │                    │  │                    │  │                 │   │
│  └────────────────────┘  └────────────────────┘  └─────────────────┘   │
│                                                                         │
│  + Inter-window communication                                           │
│  + Multi-monitor support                                                │
│  + Window state persistence for ALL windows                             │
│                                                                         │
│  Use for: agent-dashboard, mind-maps, IDEs                              │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                              KIOSK                                      │
│  ╔══════════════════════════════════════════════════════════════════╗  │
│  ║                                                                  ║  │
│  ║                    FULL SCREEN LOCKED                            ║  │
│  ║                                                                  ║  │
│  ║               (No escape to desktop)                             ║  │
│  ║                                                                  ║  │
│  ╚══════════════════════════════════════════════════════════════════╝  │
│                                                                         │
│  + Security hardening    + Auto-restart on crash                        │
│  + Remote monitoring     + Unattended operation                         │
│                                                                         │
│  Use for: Public displays, point-of-sale, industrial panels             │
└─────────────────────────────────────────────────────────────────────────┘
```

### Feature Matrix

| Feature | Universal | Advanced | Multi-Window | Kiosk |
|---------|:---------:|:--------:|:------------:|:-----:|
| Native menus | Yes | Yes | Yes | - |
| File dialogs | Yes | Yes | Yes | - |
| Notifications | Yes | Yes | Yes | - |
| Auto-updater | Yes | Yes | Yes | Yes |
| Window persistence | Yes | Yes | Yes | - |
| System tray | - | Yes | Yes | - |
| Global shortcuts | - | Yes | Yes | - |
| Deep linking | - | Yes | Yes | - |
| Multiple windows | - | - | Yes | - |
| Full-screen lock | - | - | - | Yes |

---

## Generation Flow

```
┌──────────────────┐      ┌───────────────────┐      ┌─────────────────────┐
│  Your Scenario   │      │  Template Engine  │      │   Desktop Wrapper   │
│  (picker-wheel)  │ ──▶  │  (build-tools/)   │ ──▶  │  platforms/electron │
└──────────────────┘      └───────────────────┘      └─────────────────────┘
        │                          │                          │
        │  service.json            │  Feature flags           │  Ready to build
        │  ui/dist/                │  Variable substitution   │  npm run dist
```

### Variable Substitution

Templates use Mustache-style variables that are replaced during generation:

```typescript
// Template (vanilla/main.ts):
const APP_NAME = "{{APP_NAME}}";
const WINDOW_WIDTH = {{WINDOW_WIDTH}};

// Generated (platforms/electron/src/main.ts):
const APP_NAME = "picker-wheel";
const WINDOW_WIDTH = 1200;
```

### Key Template Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `{{APP_NAME}}` | Scenario name | `picker-wheel` |
| `{{APP_DISPLAY_NAME}}` | User-facing name | `Picker Wheel` |
| `{{VERSION}}` | App version | `1.0.0` |
| `{{SERVER_TYPE}}` | How to load UI | `static/node/external` |
| `{{DEPLOYMENT_MODE}}` | Deployment strategy | `external-server/bundled` |
| `{{WINDOW_WIDTH}}` | Default window width | `1200` |
| `{{WINDOW_HEIGHT}}` | Default window height | `800` |
| `{{ENABLE_SPLASH}}` | Show splash screen | `true/false` |
| `{{ENABLE_SYSTEM_TRAY}}` | Enable tray icon | `true/false` |
| `{{PORTS_CONFIG}}` | Port environment config | `{"api":{"envVar":"API_PORT",...}}` |

---

## Deployment Modes

### Mode 1: Thin Client (Production Ready)

```
┌─────────────────┐         ┌────────────────────────────┐
│  Desktop App    │ ──HTTP──▶  Remote Vrooli Server      │
│  (UI only)      │         │  (API + Database + etc)    │
└─────────────────┘         └────────────────────────────┘
```

**How It Works:**
1. Desktop app connects to a running Vrooli scenario (LAN or Cloudflare tunnel)
2. UI files copied from scenario's `ui/dist/`
3. All API calls proxy to remote server
4. Zero offline capability

**Use When:**
- You have a Vrooli server already running
- You want to distribute UI-only thin clients
- Users need network connectivity

**Status:** Production-ready, recommended for immediate use

### Mode 2: Bundled App (In Development)

```
┌───────────────────────────────────────────┐
│              Desktop App                  │
│  ┌─────────────────────────────────────┐ │
│  │  UI  │  API Binary  │  Runtime      │ │
│  └─────────────────────────────────────┘ │
│         Everything runs locally           │
└───────────────────────────────────────────┘
```

**How It Works:**
1. Complete offline package (UI + API + runtime)
2. No server required—everything runs locally
3. Ideal for offline-first applications
4. Uses deployment-manager orchestration

**Use When:**
- Users need offline operation
- Full independence from central server
- Portable applications

**Status:** Under development in `runtime/` directory

---

## IPC Bridge Architecture

The preload script creates a secure bridge between your web app (sandboxed) and native Electron APIs:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        ELECTRON PROCESS                             │
│                                                                     │
│  ┌─────────────────┐          ┌─────────────────────────────────┐  │
│  │  Main Process   │◄─ IPC ──▶│      Renderer (Your Web App)    │  │
│  │  (Node.js)      │          │      (Sandboxed Browser)        │  │
│  │                 │          │                                 │  │
│  │  - File system  │          │  window.desktop.save(...)       │  │
│  │  - OS dialogs   │          │  window.desktop.open()          │  │
│  │  - Menus        │          │  window.desktop.notify(...)     │  │
│  │  - Tray         │          │  window.desktop.getInfo()       │  │
│  └─────────────────┘          └─────────────────────────────────┘  │
│                   ▲                           │                     │
│                   │                           │                     │
│                   └───── preload.ts ──────────┘                     │
│                        (Secure bridge)                              │
└─────────────────────────────────────────────────────────────────────┘
```

### Exposed Desktop APIs

Your web app can detect desktop mode and use native features:

```typescript
// Check if running in desktop
if (window.desktop) {
  // Use native features
  await window.desktop.save(content, "file.txt");
  await window.desktop.notify("Title", "Body");
  const info = await window.desktop.getInfo();
} else {
  // Fallback for browser
}
```

| API | Description |
|-----|-------------|
| `window.desktop.save(content, filename)` | Show save dialog |
| `window.desktop.open()` | Show open dialog |
| `window.desktop.saveJSON(data, filename)` | Save JSON file |
| `window.desktop.loadJSON()` | Load JSON file |
| `window.desktop.getInfo()` | Get platform/version info |
| `window.desktop.minimize()` | Minimize window |
| `window.desktop.maximize()` | Maximize window |
| `window.desktop.close()` | Close window |
| `window.desktop.notify(title, body)` | Show notification |
| `window.desktop.onMenuAction(cb)` | Listen for menu commands |
| `window.desktop.onProtocolUrl(cb)` | Handle deep links |
| `window.desktop.features.fileSystem` | Feature detection |

---

## Generated Output Structure

After generation, each scenario gets a `platforms/electron/` directory:

```
scenarios/picker-wheel/
├── api/                          # Your Go API
├── ui/                           # Your React app
│   └── dist/                     # Built web assets (required)
│
└── platforms/                    # NEW - Generated by scenario-to-desktop
    └── electron/
        ├── src/
        │   ├── main.ts           # Electron main process
        │   ├── preload.ts        # IPC bridge
        │   ├── splash.html       # Loading screen
        │   └── [modules]/        # All 8 core modules
        ├── assets/               # Platform icons
        ├── package.json
        │
        └── dist-electron/        # Build outputs
            ├── picker-wheel-1.0.0.AppImage   (Linux)
            ├── picker-wheel-1.0.0.deb        (Debian)
            ├── picker-wheel-1.0.0-setup.exe  (Windows)
            └── picker-wheel-1.0.0.zip        (macOS)
```

---

## Pipeline Architecture

**Implementation:** [CODE: api/pipeline/orchestrator.go], [CODE: api/pipeline/interfaces.go]

The build pipeline processes desktop apps through multiple stages:

```
┌─────────┐    ┌───────────┐    ┌──────────┐    ┌─────────┐    ┌────────────┐
│ Bundle  │───▶│ Preflight │───▶│ Generate │───▶│  Build  │───▶│ SmokeTest  │
│ Stage   │    │   Stage   │    │  Stage   │    │  Stage  │    │   Stage    │
└─────────┘    └───────────┘    └──────────┘    └─────────┘    └────────────┘
     │              │                │               │               │
     ▼              ▼                ▼               ▼               ▼
  Manifest      Validate         Generate        npm install     Launch &
  Generation    Services         Templates       npm run dist    Verify
```

### Pipeline Seams

The pipeline is built on clean interfaces for testability:

```go
// Orchestrator coordinates pipeline execution
type Orchestrator interface {
    RunPipeline(ctx context.Context, config *Config) (*Status, error)
    ResumePipeline(ctx context.Context, pipelineID string, config *Config) (*Status, error)
    GetStatus(pipelineID string) (*Status, bool)
    CancelPipeline(pipelineID string) bool
}

// Stage abstracts individual pipeline stages
type Stage interface {
    Name() string
    Execute(ctx context.Context, input *StageInput) *StageResult
    CanSkip(input *StageInput) bool
    Dependencies() []string
}
```

---

## Startup Sequence with Modules

```
App Launch
    │
    ▼
┌─────────────────┐
│ window-state/   │──▶ Load saved position from disk
│ getInitialState │    Validate against current displays
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ splash/manager  │──▶ Create splash window
│ create()        │    Display "Starting..."
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ telemetry/      │──▶ Record app_start event
│ record()        │    Generate session ID
└────────┬────────┘
         │
         ▼ (if bundled mode)
┌─────────────────┐
│ runtime/        │──▶ Spawn API binary
│ spawn()         │    Set port environment variables
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ splash/         │──▶ Poll /readyz endpoint
│ checkReadiness  │    Update splash status
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ splash/manager  │──▶ Close splash window
│ close()         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Main Window     │──▶ Load web application
│ show()          │    Apply maximized/fullscreen if saved
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ telemetry/      │──▶ Record app_ready event
│ record()        │
└─────────────────┘
```

---

## Port Environment Injection

**Implementation:** [CODE: api/generation/analyzer.go#readServiceJSON], [CODE: api/generation/types.go#PortConfig]

For bundled apps, port environment variables flow from `service.json` to the runtime:

```
service.json
    │ ports.api.env_var = "API_PORT", ports.api.port = 18700
    │ ports.ui.env_var = "UI_PORT", ports.ui.port = 36400
    ▼
analyzer.go extracts ALL ports dynamically
    │ metadata.Ports["api"] = {EnvVar: "API_PORT", Port: 18700}
    ▼
DesktopConfig.Ports populated
    │ config.Ports = metadata.Ports
    ▼
template-generator.ts
    │ PORTS_CONFIG = {"api":{"envVar":"API_PORT","port":18700},...}
    ▼
main.ts at startup
    │ const PORTS = {"api":{"envVar":"API_PORT","port":18700},...}
    ▼
main.ts at spawn (startBundledRuntime)
    │ runtimeEnv["API_PORT"] = "18700"
    │ runtimeEnv["UI_PORT"] = "36400"
    ▼
Go binary receives ALL env vars
    │ requireEnv("API_PORT") returns "18700"
```

---

## Telemetry & Monitoring

Desktop apps write telemetry to track deployment health:

**Location:** OS-specific user data directory
- **Linux:** `~/.config/<app-name>/deployment-telemetry.jsonl`
- **macOS:** `~/Library/Application Support/<app-name>/deployment-telemetry.jsonl`
- **Windows:** `%APPDATA%\<app-name>\deployment-telemetry.jsonl`

**Events Tracked:**
| Event | Description |
|-------|-------------|
| `app_start` | App launched |
| `external_server_mode` | Mode detected |
| `server_ready` | Server reachable |
| `dependency_unreachable` | Connection failed |
| `app_ready` | Fully initialized |
| `startup_error` | Errors during startup |
| `app_shutdown` | Clean exit |

---

## Related Documentation

- [DOC: templates/vanilla/README.md] - Complete module documentation
- [DOC: docs/QUICKSTART.md] - Getting started with thin client generation
- [DOC: docs/deployment-modes.md] - Choosing deployment_mode and server_type
- [DOC: docs/desktop-integration-guide.md] - Feature cookbook (file system, menus, tray)
- [DOC: docs/internal/SEAMS.md] - Integration boundaries and testability
- [DOC: docs/concepts/GLOSSARY.md] - Term definitions
