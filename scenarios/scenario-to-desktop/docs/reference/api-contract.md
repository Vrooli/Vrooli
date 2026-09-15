# API Contract & Data Models

> Relocated from PRD.md during documentation restructuring (2026-04-06).
> For the live endpoint reference and handler details, see [api-architecture.md](api-architecture.md).

## Endpoints

## Contract migration inventory

The authoritative long-term transport is generated Connect RPC. This inventory
records every transport exception and its owning service group so that REST
retirement is measurable rather than inferred from file deletion.
`GET /docs/{docPath}`, smoke-test video, artifact download, capture file, and
live-desktop session-file endpoints remain plain HTTP because they stream a
file or WebSocket payload; their metadata and lifecycle operations are RPCs.

| Group | Transport boundary | Connect service target |
|---|---|---|
| Health | `GET /health`, `GET /api/v1/health` | `shared.HealthService.Check` |
| Pipeline | Connect: run, get, resume, cancel, list; active/create/reset/history/start per scenario. Plain HTTP remains only for bundle cleanup while its file-oriented lifecycle is redesigned. | `pipeline.PipelineService` |
| System | No REST administration routes. Status, template discovery, and Wine checks/install/status are unary RPCs. | `domain.SystemService` |
| Build | desktop package download; build-complete webhook | `domain.BuildService`; download remains HTTP |
| Signing | prerequisites, certificate discovery, Linux key generation, config CRUD, platform patch/delete, validate, readiness | `domain.SigningService` |
| Preflight/config | bundle preflight, manifest generation, scenario metadata/config generation | `domain.PreflightService`, `domain.ConfigService` |
| Deployment integration | No REST administration routes. Deploy-target configuration, connectivity checks, and readiness evidence are unary RPCs; pipeline release work uses the same durable repository internally. | `domain.DeployTargetService`; deployment-manager owns cross-ramp distribution and promotion |
| Scenario/state | No REST state administration routes. Desktop status and state load/save/delete/check/log/invalidate operations are unary RPCs. | `domain.OperationsService.ListDesktopScenarioStatus`, `StateService` |
| Telemetry | No REST administration routes. Ingest, summary, insights, tail, and deletion are unary RPCs; raw JSONL download remains HTTP streaming. | `TelemetryService` |
| Evidence | local desktop session start/list/get/heartbeat/launch/artifact/control/stop and capture list/summary/delete | `domain.EvidenceService`; capture file/download and VNC WebSocket remain HTTP streams |
| Records | No REST administration routes. Record listing, wrapper relocation, and desktop deletion are unary RPCs. | `DesktopRecordsService` |
| Tasks | create/list/get/stop pipeline task; agent-manager status | `TaskService` |
| Utilities | probe, proxy hints, scenario port lookup, docs manifest/content, icon preview | `ScenarioService` or `SystemService`; docs file remains HTTP |

The WebSocket route `/api/v1/livedesktop/sessions/{id}/ws` is intentionally
kept HTTP/WebSocket. Connect covers typed session control and evidence metadata;
the API proxy remains the browser's loopback-only VNC transport.

`OperationsService.ListDesktopScenarioStatus` is the typed inventory contract
for desktop readiness. It returns explicit scenario, connection, build-artifact,
record-location, and aggregate-stat fields; clients must not depend on a generic
JSON payload or the retired REST desktop-status response shape.

Pipeline status is a typed Connect contract. Each `StageResult.details` value
uses the `pipeline.StageDetails` oneof: resource-deployment resolution, bundle,
preflight, generation, build, smoke-test, or deploy. Clients must select the
oneof case; the API does not expose stage output as untyped JSON or require
clients to infer its shape from a stage name. The resource-deployment case
contains the immutable target/resource decision used by bundle staging, so a
desktop release can be audited from its pipeline record.

`EvidenceService` accepts an `EvidenceTarget`. `KIND_LOCAL` executes on the
scenario-to-desktop host and is the supported implementation today. A
`KIND_BRIDGE_NODE` request is rejected with `unimplemented` until the separate
remote-desktop capability owns the node-side streaming and execution protocol.
Vrooli Bridge remains the typed, allowlisted reach and job-dispatch layer; it
does not proxy VNC/WebSocket traffic or act as a remote-desktop server.

### POST /api/v1/desktop/generate

Generate a desktop application from a scenario configuration.

**Input**:
```yaml
scenario_name: string
framework: "electron"
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
framework: "electron"
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
| `browser-automation-studio.ready` | Start desktop app testing pipeline |

## Release Authority

scenario-to-desktop holds no release authority. It builds, packages, signs, and publishes; `deployment-manager` decides whether publishing is allowed and records what shipped. See [ADR-005](../../../deployment-manager/docs/decisions/005-governance-plane-boundary.md).

This scenario calls deployment-manager. deployment-manager does not drive this pipeline.

### Outbound Calls To deployment-manager

| Endpoint | When | Purpose |
|----------|------|---------|
| `POST /api/v1/profiles/{id}/approvals` | After a platform build is validated | Record an approval decision |
| `GET /api/v1/profiles/{id}/release-gate?commit=` | Before publishing | Ask whether publishing is allowed for this commit |
| `POST /api/v1/bundles/export` | During manifest generation | Generate a bundle manifest for the target tier |
| `EvidenceService.ReportTargetVerdict` (Connect) | After a desktop journey settles, when reporting is configured | Send the target disposition and producer-held capture references; no artifact bytes or local paths cross the boundary |

The gate response is authoritative. When the gate refuses, the pipeline stops and reports the named missing gate. Do not publish on a gate error, and do not treat an unreachable deployment-manager as an implicit allow.

Any new ramp scenario implements this same caller side. Target-specific behavior belongs here; gate and record logic belongs in deployment-manager.

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
| browser-automation-studio | Desktop application UI testing and screenshot capture | Manual testing instructions |
| notification-hub | Build completion notifications | CLI output only |
