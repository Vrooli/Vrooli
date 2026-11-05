# Device Control and TV Launcher Maturity Effort

## Destination

Deliver the user-discussed TV and Android phone workflows as reliable, reusable Device Control capabilities, and complete the related Portal TV launcher work. Reduce duplicated paths and complexity while improving capture fidelity, performance, observability, and truthful capability reporting. Every completion claim must point to reproducible evidence and the owning implementation boundary.

- Destination: accepted maturity surfaces >= 9

## Scope

- Shared device lifecycle language and capabilities: wake, sleep, unlock/authenticate, readiness, and navigation to the device's home surface. Preserve distinctions between these states and compose sequences as reusable flows where appropriate.
- Android connectivity and transport parity, including reusable wireless-debugging setup flows only on compatible OS/device combinations; verified identity, USB/Wi-Fi state, and repeated reachability observations.
- Consistent capture commands for screenshots, recordings (including optional audio), device/app logs, network requests, and semantic/accessibility screen representations where each transport supports them.
- Screenshot fidelity, recording source/output frame evidence, audio presence/audibility/synchronization, playback impact, capture resource cost, and performance-state measurement.
- AI-powered control: semantic control discovery, action selection, decision-model/Gateway correctness, LLM fallback, reusable flow promotion, and measured success/latency/cost.
- Verified immutable Android artifact resolution and live install under the proper owner/lease boundary; no arbitrary local APK path as the trusted product workflow.
- Portal TV launcher crash/ANR/fallback diagnostics and root-cause fix; own-app long-press lifecycle actions expressed as app start/restart/stop, plus start for an app whose scenario is not running.
- Fix infrastructure failures that materially block these outcomes, and simplify relevant architecture, contracts, docs, tests, and repeated workflow friction.

## Authority and safety

The user authorized use of the TV and Android phone for Device Control work, including restart when needed. The user is currently away from the phone and cannot accept on-device prompts. Do not initiate a MediaProjection/audio consent flow until the phone owner can respond. Continue implementation and non-interactive validation meanwhile. The user supplied test-device PIN 5555 in this conversation; use only through the existing credential authority, never hard-code or log it. Physical actions require fresh device identity and availability verification.

Do not interpret one wireless timeout as proof that screen sleep disables Wi-Fi. Collect repeated, timestamped, identity-verified observations with transport, USB, screen, and reachability states. Never call a capability unavailable solely because discovery/infra timed out.

Use each owning scenario/control-plane boundary. Artifact resolution belongs to the artifact producer; host lifecycle repair belongs to the control plane; Portal UI lifecycle orchestration belongs to Portal/Device Control contracts. Do not bypass owners with raw ADB, private repair code, arbitrary APKs, or hand-edited generated output.

## Non-goals

Unrelated iOS/macOS/desktop/cloud expansion, full product roadmap completion, deployment/publication, new paid services, and broad refactors without a demonstrated dependency on the destination are excluded. Do not trigger consent while the user is away. Do not claim physical outcomes without fresh evidence.

## Canonical sources

Start with `STATUS.md`, then follow the scoped epoch files. Current evidence and problem history live in `../../PROBLEMS.md` and `../../PROGRESS.md`; current user-facing contracts in `../../../reference/capabilities.md`, `../../../reference/cli-commands.md`, `../../../PRD.md`, `../../../DESIGN.md`, and `../../../concepts/FLOWS.md`. Portal ownership/evidence is in `scenarios/portal/docs/internal/PROGRESS.md`; read its PRD and design before changing Portal. Testing protocol is `docs/TESTING.md`.
