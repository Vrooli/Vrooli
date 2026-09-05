# Cross-ramp delivery spine: phase 1 evidence and seam record

This document is the phase-1 record for plan
`cross-ramp-delivery-spine-extract-the-shared-validation-and`. It freezes the
pre-extraction oracle, records the current consumers, and pins the contract for
`packages/delivery-ramp-go`. It is not a second evidence contract: the
canonical wire contracts remain `journey-evidence.v2` and the deployment
manager evidence API.

## Frozen oracle

The oracle was captured before source movement and remains outside `/tmp` at:

`/home/matthalloran8/.vrooli/plan-artifacts/cross-ramp-delivery-spine/oracle/`

| Artifact | SHA-256 | Evidence |
|---|---|---|
| `journey-evidence.v2.json` | `859d5cd41d16ee500aa8c92d12983150c27448c459c824826c5eaab05ee7c3de` | persisted sidecar |
| `evidence.manifest.json` | `15cb726b42b7c7d34895319c88d9bec3e02cbced3128d70bd21f74ba2ce73a68` | persisted producer manifest |
| `journey-evidence.mp4` | `62a91d525be8e63ca7d9d17504c6ff030b85fb1c18571d6442c08e774024d24c` | persisted recording |

The unchanged native run was pipeline
`a89fd9d4-a380-5653-da1a-54a71a7efaf4`, smoke test
`smoke-hello-desktop-1786366803041`, with capability `hello-desktop`, profile
`normal-review`, platform `linux-amd64`, display `:99`, and Openbox. The
resolved local target was `local-linux-amd64`, with advertised capability
values `[6, 1, 2]`, and was available and healthy. The journey disposition was
`pass`; it contained eight passing chapters and 33 events. The manifest was in
state `passed`, with `journey-evidence.v2`, eight ordered chapter IDs,
`event_count: 33`, and `redaction_status: verified` in its timeline.

The recording was independently decoded with `ffprobe`: H.264, 1920x1080,
22.733333 seconds, 341 frames, and 1,189,203 bytes. It contains useful
application frames. The sidecar and manifest were re-read from the durable
directory after the capture shell completed.

The first unchanged attempt encountered an unregistered stale child process on
port 23100. The listener was identified with `vrooli diagnose-port --json`,
then the exact stale PID was terminated. The successful rerun above is the
oracle; this remediation does not change the oracle inputs or the source
baseline.

The immutable plan baseline is
`cross-ramp-delivery-spine-baseline`, synchronized with required coverage for
`deployment-manager` and `scenario-to-desktop` at Git SHA
`f0e66634aee65324a21ce0a1edc6a024f642927c`. The baseline's scenario-to-desktop
documentation phase reported pre-existing knowledge-observatory snippet
failures; Git Control Tower still marked behavioral coverage ready, and no
source was edited to alter that baseline.

## Consumer inventory

The inventory below names the package boundaries and the symbols that must be
re-homed or replaced. Handler and CLI/UI wire projections remain consumer
surfaces; they are not evidence that transport code belongs in the spine.

| Current owner | Consumer / boundary | Symbols or data crossing it | Classification | Extraction disposition |
|---|---|---|---|---|
| `validationmatrix` | `api/main.go`, `api/validation_matrix_executor.go`, pipeline stages | `Service`, `NewService`, `Executors`, `LocalExecutor`, `BridgeExecutor`, `CellRequest`, `CellResult`, `ComputeApplicability`, `EvaluateReleaseGate`, `MatrixRun`, `MatrixComparison` | internal orchestration plus persisted matrix state | Move domain model, service, gate, and executor seams to spine; leave HTTP handler wiring and desktop constructors at the scenario edge. |
| `validationmatrix` | `api/validationcatalog` and `api/validationcatalog/client.go` | `CatalogResolver`, `CatalogSnapshot`, `MatrixSelection`, profile contracts | public provider/catalog boundary | Move neutral selection and profile contracts; keep deployment-manager catalog transport in the consumer adapter. |
| `validationmatrix` | `api/bridgevalidation/client.go` | `BridgeExecutor`, `CellResult`, target identity, run disposition | internal transport adapter | Replace scenario package coupling with the spine `Transport` and `Driver`/cell result types. |
| `validationmatrix` | Connect/HTTP handler and CLI/UI generated clients | validation matrix JSON/proto fields, release gate and cell dispositions | rendered/public API | Preserve wire names and enum semantics; handlers remain scenario-owned. |
| `targetinventory` | `api/main.go`, `api/targetinventory/handler.go`, `/api/v1/validation/targets` | `Inventory`, `Target`, `TargetHealth`, `BridgeTrust`, `Discover` | public API and internal probe result | Move neutral target/capability types and `Prober` contract; keep HTTP serialization and host command probing in the desktop adapter. |
| `targetinventory` | `api/bridgevalidation/client.go` | `BridgeSource`, bridge identity, capabilities, trust, availability | bridge transport boundary | Model as spine transport inventory; bridge client supplies the implementation. |
| `targetinventory` | UI target-selection surface | target IDs, labels, capabilities, availability, reason | rendered UI | Keep response compatibility; consume spine-owned types through scenario handlers. |
| `bridgevalidation` | `api/main.go`, validation matrix executors | `Registry`, `Dispatcher`, `Runs`, `Client`, `NewClient`, bridge discovery and execution mapping | internal transport adapter | Retain Connect clients in scenario-to-desktop; return spine `Target`, `Transport`, and `CellResult` values. |
| `bridgevalidation` | validation matrix | bridge dispatch is not target evidence; failed/aborted/unavailable mapping | release-gate semantics | Centralize fail-closed disposition semantics in spine; adapter reports observed remote run state only. |
| `internal/capabilities` | `api/main.go`, capability registry handler | `Metadata`, `Known`, `Registry`, `ScenarioChecker` | internal scenario registry | Move only provider-neutral capability definitions and checks that belong to delivery-ramp contracts; scenario-specific catalog metadata stays at the edge. |
| `evidence` | `api/main.go`, pipeline smoke-test stage | `Manifest`, `ManifestWriter`, `Manifest.Validate`, gate/disposition constants | persisted data and release-gate contract | Move contract, validation, and manifest assembly inputs to spine; keep capture-store and screenrecording adapters in scenario-to-desktop. |
| `evidence` | `api/evidence/producer.go`, deployment-manager Connect client | `EvidenceClient`, `ConnectReporter`, `CaptureToEvidenceRef`, `ReportJourney` | public cross-scenario API | Keep producer implementation transport-specific; spine owns reference/checksum-only report values and fail-closed rules. |
| `evidence` | evidence Connect handler and CLI/UI | capture list, summary, desktop session and control responses | rendered/public API | Retain scenario transport and desktop session endpoints; do not move desktop control into the spine. |
| `evidence` | `captures`, `screenrecording`, `smoketest` | capture IDs, immutable refs, checksums, timeline/chapter data | internal persisted boundary | Use adapter interfaces; no capture bytes, credentials, bearer tokens, or endpoint details enter shared JSON. |
| `smoketest` | `api/pipeline/stage_smoketest.go`, `api/main.go` | `Service`, `PerformSmokeTest`, `JourneyResult`, `EvidenceReportInput`, `Status` | internal orchestration and persisted evidence | Move journey contract, runner orchestration, disposition, and evidence input types; retain desktop process, X11, Electron, and capture implementations as adapters. |
| `smoketest` | Connect handler, CLI, and generated UI client | smoke status, journey review, recording review, error/recovery fields | public/rendered API | Preserve JSON/proto compatibility; scenario handlers translate spine values. |
| `smoketest` | `api/smoketest/mocks` and test suites | `DesktopDriver`, `JourneyCapture`, process/platform seams | test-only seam | Move neutral fakes with the contract; keep desktop fakes for the desktop adapter package. |

The inventory explicitly distinguishes public API, persisted data, rendered UI,
and internal-only consumers so an extraction cannot accidentally treat a route
or generated client as permission to move platform technology into the spine.

## File classification

The classification applies recursively, including test and mock files. Tests
follow the classification of the production seam they exercise.

### `api/smoketest`

| Files | Classification | Reason |
|---|---|---|
| `journey_contract.go`, `types.go`, `config.go`, `errors.go`, `report.go`, `service_results.go` | provider-neutral | journey schema, dispositions, lifecycle/result vocabulary, bounded policy, and recovery semantics can be used by every ramp |
| `journey.go`, `service_run.go`, `service.go`, `service.go` tests | mixed; split during extraction | runner orchestration is neutral, while current `procmetrics`, X11/window, semantic HTTP, and capture calls are desktop implementations that must become adapter calls |
| `interfaces.go`, `mocks/mocks.go`, `mocks/mocks_test.go` | provider-neutral seam declarations plus desktop test doubles | the interfaces are reusable; concrete xdotool/process/display fakes remain desktop-side |
| `environment_reader.go`, `file_system.go`, `store.go` | provider-neutral substrate | environment, filesystem, persistence, and cancellation seams do not identify a ramp |
| `launchtrace.go`, `performance.go`, `telemetry_chain.go`, `telemetry_error_extractor.go`, `telemetry_resolver.go`, `output_parser.go` | provider-neutral | lifecycle trace, bounded telemetry parsing, performance vocabulary, and error recovery are evidence mechanics; concrete artifact paths and upload clients stay at the edge |
| `platform_resolver.go`, `prerequisites.go`, `prerequisites_unix.go`, `prerequisites_windows.go`, `process_executor.go`, `process_executor_unix.go`, `process_executor_windows.go` | desktop-specific | these select host commands, Xvfb, xdotool, process execution, and OS-specific smoke commands |
| `connect_handler.go`, `connect_handler_test.go` | scenario-specific transport edge | Connect/proto route translation is not shared package logic |
| `integration_test.go`, `cmd/test-fixture/main.go`, `cmd/test-fixture/main_test.go` | desktop-specific test fixture | the fixture exercises executable/process and desktop smoke behavior |
| `*_test.go` not otherwise listed | follows the exercised production seam | tests move with neutral contracts or remain with desktop adapters; no test-only compatibility package is retained |

### `api/evidence`

| Files | Classification | Reason |
|---|---|---|
| `contract.go`, `contract_test.go` | provider-neutral | manifest schema, artifact references, gate requirements, state transitions, and fail-closed validation are cross-ramp contracts |
| `manifest.go`, `manifest_performance_test.go` | mixed; split during extraction | manifest assembly is neutral, but capture-store and screenrecording inspection are desktop producer dependencies |
| `performance.go`, `performance_test.go` | provider-neutral | timeline/performance summaries and comparisons are not tied to Electron or a host OS |
| `producer.go`, `producer_test.go` | split boundary adapter | reference/checksum-only mapping is neutral; Connect client construction and deployment-manager transport stay scenario-side |
| `connect_handler.go`, `connect_handler_test.go` | scenario-specific transport edge | desktop session/control and capture endpoints are scenario-owned and must not enter the spine |

No file is classified as neutral merely because it has a neutral name: direct
X11, Electron, xdotool, ffmpeg, host process, deployment-manager transport,
Connect, HTTP-handler, and scenario-module dependencies are adapter or edge
signals.

## Pinned spine seam

The implementation module is:

```text
path: packages/delivery-ramp-go
module: github.com/vrooli/vrooli/packages/delivery-ramp-go
```

The following is the pinned contract. Names may be refined for idiomatic Go
during implementation, but the information carried by these types and the
four method boundaries may not be reduced.

```go
type TransportKind string

const (
	TransportLocal  TransportKind = "local"
	TransportBridge TransportKind = "bridge"
)

type Transport struct {
	Kind       TransportKind
	ID         string
	Trust      string
	Endpoint   string // never serialized into evidence or manifests
	Available  bool
	Reason     string
}

type Target struct {
	ID          string
	Label       string
	Platform    string
	OS          string
	Architecture string
	DeviceKind  string
	Transport   Transport
	Capabilities []string
	Available   bool
	Reason      string
}

type Cell struct {
	ID          string
	Target      Target
	ProfileID   string
	Capability  string
	Required    bool
}

type Prober interface {
	Probe(context.Context, ProbeRequest) (Inventory, error)
}

type Builder interface {
	Build(context.Context, BuildRequest) (Artifact, error)
}

type Driver interface {
	Execute(context.Context, DriverRequest) (JourneyResult, error)
}

type Distributor interface {
	Distribute(context.Context, DistributionRequest) (DistributionResult, error)
}
```

`ProbeRequest` carries the selected profile/capability requirements and may
resolve a local host or bridge inventory. `BuildRequest` carries a ramp-neutral
artifact request and target cell; it does not expose Electron, Capacitor, or
Xcode types. `DriverRequest` carries a cell, immutable artifact reference,
journey plan, bounded context, and evidence sink. `DistributionRequest` carries
an immutable artifact and target destinations. Results carry target identity,
disposition, missing capability, next action, checksums, and evidence
references; they never carry capture bytes or endpoint credentials.

The contract must express these adapter rows without a new interface or a
platform branch:

| Adapter | Desktop row | Android row | iOS row |
|---|---|---|---|
| Prober | host/Xvfb/display/window prerequisites and local capabilities | adb-visible device/emulator, API level, package/install and WebView/bridge capabilities | simulator/device, runtime/inspector availability, signing/install capabilities |
| Builder | Electron/AppImage/package artifact for a selected host target | Capacitor/Android package or bundle for a device/API cell | Capacitor/iOS archive/app for a simulator/device cell |
| Driver | process launch, X11/window/input actions, semantic assertions, journey captures | install/launch, WebView/bridge actions, permission/background/rotation/offline/deep-link chapters | install/launch, WebKit inspector/actions, permission/background/rotation/offline/deep-link chapters |
| Distributor | immutable artifact reference to the deployment manager | immutable package/bundle reference to the deployment manager | immutable archive/app reference to the deployment manager |

The shared package must not branch on ramp name, scenario name, or
`runtime.GOOS`. Unsupported and unavailable are terminal dispositions, never
implicit pass. Every unavailable result must include both a missing-capability
description and a next action. A bridge dispatch is never target evidence until
the Driver returns an observed result with evidence references.

## Phase-1 validation checklist

- Baseline: complete, two required members ready.
- Native oracle: pipeline and smoke test passed on `linux-amd64` using
  `hello-desktop`/`normal-review`.
- Sidecar: schema version 2, eight chapters, each passing with before/after
  capture IDs, assertions where required, and capture references.
- Manifest: passed; ordered `journey-evidence.v2` timeline, eight chapter IDs,
  33 events, verified redaction status, and SHA-256 references.
- Recording: independently decoded and useful.
- Durable reread: all three files survive outside `/tmp`.
- Module path and adapter information: pinned above before source movement.

Later phases must preserve this record and add implementation/conformance
evidence; they must not silently revise the oracle.
