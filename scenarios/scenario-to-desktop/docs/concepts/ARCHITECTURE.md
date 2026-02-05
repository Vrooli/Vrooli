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
├── cli/                          # Command-line interface
├── ui/                           # React web management interface (port 35000-39999)
├── templates/                    # Template system
│   ├── vanilla/                  # Base Electron implementation
│   │   ├── main.ts              # Electron main process
│   │   ├── preload.ts           # Secure IPC bridge
│   │   ├── splash.html          # Professional splash screen
│   │   ├── window-state/        # Window state persistence module
│   │   └── splash/              # Splash screen management
│   ├── advanced/                # Template configuration files
│   │   ├── universal-app.json   # Default template (95% of use cases)
│   │   ├── advanced-app.json    # Professional features
│   │   ├── multi-window.json    # Complex workflows
│   │   └── kiosk-mode.json      # Full-screen deployments
│   └── build-tools/             # Template generator system
├── runtime/                      # Bundled runtime for offline apps (future)
└── docs/                         # Documentation
```

---

## Template Architecture

### Template Types

```
templates/
├── vanilla/                    ◄── BASE IMPLEMENTATION (Electron)
│   ├── main.ts                     Main process (window creation, menus)
│   ├── preload.ts                  IPC bridge (web ↔ native)
│   ├── splash.html                 Professional loading screen
│   ├── window-state/               Remembers window size/position
│   └── splash/                     Splash screen management
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
        │   └── splash.html       # Loading screen
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

## Window State Persistence

**Implementation:** [CODE: templates/vanilla/window-state/manager.ts], [CODE: templates/vanilla/window-state/types.ts]

The template includes a module to remember window position across restarts:

```
window-state/
├── types.ts       # Type definitions and interfaces
├── storage.ts     # File I/O abstraction (IStateStorage seam)
├── display.ts     # Display/screen abstraction (IDisplayProvider seam)
├── validator.ts   # Pure validation functions (no mocks needed)
├── manager.ts     # Orchestration (WindowStateManager)
└── __tests__/     # Co-located tests
```

### State Persistence Flow

```
App Launch                                  App Close
    │                                           │
    ▼                                           ▼
┌─────────────┐                         ┌─────────────┐
│ Load State  │                         │ Save State  │
│ from Disk   │                         │  to Disk    │
└─────────────┘                         └─────────────┘
    │                                           ▲
    ▼                                           │
┌─────────────┐     Window Lifecycle     ┌─────────────┐
│  Validate   │ ───────────────────────▶ │  Capture    │
│  Position   │                          │   Bounds    │
└─────────────┘                          └─────────────┘
    │
    ▼
┌─────────────┐
│ Apply to    │
│  Window     │
└─────────────┘
```

### Multi-Monitor Support

The validator handles disconnected displays gracefully:

1. Load saved state with display ID
2. Check if original display still connected
3. If not, find best alternative (same position or primary)
4. Ensure window is visible (not off-screen)

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

## Related Documentation

- [DOC: docs/QUICKSTART.md] - Getting started with thin client generation
- [DOC: docs/deployment-modes.md] - Choosing deployment_mode and server_type
- [DOC: docs/desktop-integration-guide.md] - Feature cookbook (file system, menus, tray)
- [DOC: docs/internal/SEAMS.md] - Integration boundaries and testability
- [DOC: templates/README.md] - Template system details
