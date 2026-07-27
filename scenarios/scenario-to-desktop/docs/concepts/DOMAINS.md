# Scenario-to-Desktop Domain Inventory

This is the canonical ownership map for the desktop deployment ramp.  A
domain owns a cohesive capability and its durable state; transports, process
composition, and generic substrate are deliberately listed as non-domains.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| generation | Analyzes a scenario and compiles its Electron wrapper deterministically. | Turn a Vrooli scenario into a platform-neutral desktop project. | Generated project configuration and build inputs. | service | orchestration | DesktopConfig, ScenarioMetadata, TemplateGenerator | `api/generation/`, `templates/` |
| pipeline | Orchestrates validated desktop generation from preflight through artifacts. | Provide replay-safe operator workflows and stage state. | Pipeline runs and stage outcomes. | orchestration | mutation | Pipeline, Stage, IdempotencyKey | `api/pipeline/` |
| preflight | Establishes the admissibility and deployment shape of a requested desktop build. | Prevent invalid deployment requests before work begins. | Preflight report and resolved deployment configuration. | classification | query | Preflight, BundleManifest, DeploymentMode | `api/preflight/` |
| build | Builds and packages generated desktop applications. | Produce installable artifacts for supported operating systems. | Build state and artifact metadata. | orchestration | mutation | Build, Artifact, TargetPlatform | `api/build/` |
| signing | Manages platform signing configuration, readiness, and certificate operations. | Make distributable desktop artifacts trustworthy. | Signing configuration and certificate references. | service | mutation | SigningConfig, Certificate, Readiness | `api/signing/` |
| deploy | Executes deployment-target and release operations for finished artifacts. | Deliver desktop artifacts to the selected target. | Deployment state and target configuration. | orchestration | mutation | Deployment, DeploymentTarget | `api/deploy/` |
| livedesktop | Runs a generated desktop application and captures runtime evidence. | Validate the locally native path and remote desktop targets through a uniform seam. | Runtime sessions and observation metadata. | service | orchestration | LiveDesktop, RuntimeSession, EvidenceCapture | `api/livedesktop/`, `api/screenrecording/`, `api/evidence/` |
| smoketest | Probes generated applications for executable desktop behavior. | Supply automated runtime acceptance evidence. | Smoke-test reports. | service | query | SmokeTest, ProbeResult | `api/smoketest/` |
| records | Persists the audit history of generated desktop applications. | Preserve operator-visible deployment provenance. | Desktop generation records. | service | mutation | DesktopRecord, Provenance | `api/records/` |
| bundle | Resolves bundle manifests and desktop dependency substitutions. | Keep bundled deployment policy explicit and reproducible. | Bundle manifests. | service | query | Bundle, Manifest, DependencySwap | `api/bundle/` |
| scenario | Discovers and validates source-scenario metadata. | Give desktop generation a reliable source-model boundary. | Scenario inventory snapshots. | query | classification | ScenarioInfo, ServiceJSON | `api/scenario/` |
| telemetry | Records desktop runtime telemetry and process measurements. | Provide operational visibility without coupling it to generation. | Telemetry events and process metrics. | service | reporting | TelemetryEvent, ProcessMetric | `api/telemetry/`, `api/procmetrics/` |
| tasks | Coordinates operator investigation and remediation tasks. | Turn detected issues into accountable work. | Task state and task outcomes. | orchestration | mutation | Task, Investigation, Fix | `api/tasks/`, `api/agentmanager/` |
| storage | Provides scenario-local persistence and storage migration behavior. | Keep durable state portable and schema-safe. | Database files, migrations, and state snapshots. | service | mutation | Persistence, Migration, State | `api/persistence/`, `api/storagemigrate/`, `api/storagepaths/`, `api/state/` |
| system | Reports host and platform capability required for desktop builds. | Make platform constraints observable before execution. | None. | query | classification | SystemInfo, PlatformCapability | `api/system/` |

## Non-Domains

- `api/main.go` — composition root.
- `api/connect.go` — Connect transport registration.
- `api/handlers_*.go` — HTTP/Connect transport adapters.
- `api/domain/` — shared API contracts and boundary types.
- `api/shared/` — generic substrate used across product domains.
- `api/captures/` — compatibility-free transport composition for evidence capture.
- `cli/` — command transport and presentation surface.
- `ui/` — operator presentation surface and generated Connect client consumer.
