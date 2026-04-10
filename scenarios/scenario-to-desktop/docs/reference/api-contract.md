# API Contract & Data Models

> Relocated from PRD.md during documentation restructuring (2026-04-06).
> For the live endpoint reference and handler details, see [api-architecture.md](api-architecture.md).

## Endpoints

### POST /api/v1/desktop/generate

Generate a desktop application from a scenario configuration.

**Input**:
```yaml
scenario_name: string
framework: "electron" | "tauri" | "neutralino"
template_type: "basic" | "advanced" | "kiosk" | "multi_window"
config:
  app_name: string
  app_id: string
  description: string
  version: string
  author: string
  company: string
  api_endpoint: string
  target_platforms: string[]
  features:
    system_tray: boolean
    auto_updater: boolean
    native_menus: boolean
    file_associations: string[]
```

**Output**:
```yaml
build_id: string
desktop_path: string
install_instructions: string
test_command: string
```

**SLA**: Response time < 60s, availability 99%

### GET /api/v1/desktop/status/{build_id}

Check desktop build status and get deployment info.

**Input**: `build_id` (path parameter)

**Output**:
```yaml
status: "building" | "ready" | "failed"
desktop_paths?: Record<string, string>
error_log?: string[]
test_results?: TestResult[]
```

**SLA**: Response time < 100ms, availability 99%

## Data Models

### DesktopTemplate

Stored on the filesystem.

```yaml
id: string
name: string
framework: "electron" | "tauri" | "neutralino"
type: "basic" | "advanced" | "kiosk" | "multi_window"
template_files:
  package_json: string
  main: string
  preload: string
  renderer: string
  build_config: string
  icons: Record<string, string>
variables: Record<string, any>
created_at: timestamp
```

Relationships: References scenario configurations.

### DesktopBuild

Stored on the filesystem.

```yaml
id: string
scenario_name: string
template_id: string
framework: string
build_config: DesktopConfig
output_paths:
  windows: string
  macos: string
  linux: string
status: "building" | "ready" | "failed"
build_log: string[]
created_at: timestamp
```

Relationships: Tracks builds per scenario.

## Event Interface

### Published Events

| Event | Payload | Subscribers |
|-------|---------|-------------|
| `desktop.build.completed` | `{ build_id, scenario_name, success, platforms }` | ecosystem-manager, notification-hub |
| `desktop.deployed` | `{ build_id, deployment_urls, store_urls }` | deployment-manager, analytics-hub |

### Consumed Events

| Event | Action |
|-------|--------|
| `scenario.updated` | Regenerate desktop applications for updated scenarios |
| `resource.browserless.ready` | Start desktop app testing pipeline |

## Cross-Scenario Interactions

### Provides To

| Scenario | Capability | Interface |
|----------|-----------|-----------|
| system-monitor | Native desktop system monitoring application | API/CLI |
| personal-digital-twin | Offline-capable desktop assistant application | API/CLI |
| document-manager | Desktop document management with native file integration | API/CLI |

### Consumes From

| Scenario | Capability | Fallback |
|----------|-----------|----------|
| browserless | Desktop application UI testing and screenshot capture | Manual testing instructions |
| notification-hub | Build completion notifications | CLI output only |
