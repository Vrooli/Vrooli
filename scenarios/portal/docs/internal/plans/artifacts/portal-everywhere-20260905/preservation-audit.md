# Preservation and executability audit

Authoring review date: 2026-09-05. This review checks the plan, not the product.

| Material request or decision | Phases | Acceptance / artifact |
|---|---|---|
| Agent orientation and speed, not only backend throughput | 26 | LEARN-01..05; fixed paired benchmark |
| Skills / programs / authoritative owner operations | 16..19 | FLOW-09..10 and skill/program fixtures |
| Normal Portal UI and native expanded parity | 14,21 | UI-01..02 |
| Pill, palette, shortcut, tray, background lifecycle | 20..21 | UI-01,04,05,09,10 |
| Pointer context before focus, selected text, region and annotation | 11,22 | UI-03,06,12; NAT-12 |
| Current machine equals a target, not necessarily API host | 3..4,21 | CAT-01 |
| Machines, attached devices, scenarios, browsers, terminals | 14,25 | CAT-04,06,08; UI-07 |
| Programmatic, semantic, and visual control | 8..11,17..18 | NAT-03..08; FLOW-07..09 |
| Windows, macOS, Linux portability | 7..10,29 | Five mandatory primary support rows |
| Bridge installed nodes and compute-managed machines | 12,25 | REM-01..08; CAT-06 |
| Remote screens and interaction | 12..14 | REM-03,06,07; latency matrix |
| Avoid duplicate device/desktop engines | 2,5,17,27 | Ownership invariants and replacement journey |
| Shared Portal/Web Console target UX | 3..4,14 | CAT-02,08 and terminal parity |
| Scenario widgets without native privilege inheritance | 15,28 | EMB-01..06; PKG-08 |
| Optional providers and bundles | 4,16,24 | OPT-01..10 |
| Smart fallback with same task meaning | 16 | OPT-06..07; REM-05 |
| Exactly one outcome despite disconnect uncertainty | 6,12,16 | AUTH-05..07; REM-04..05 |
| Workflow creation, validation, promotion, repair, replay | 17..19 | FLOW-01..10 |
| Voice through Audio Tools | 23 | UI-11; OPT-02; live audio receipts |
| Scenario-to-Desktop extension rather than template fork | 20,24,29 | PKG-01..08 |
| Old Assistant migration and issue capture | 27 | MIG-01..02 |
| Open-source/project research and UX inspiration | 8,13,30 | research-notes.md and bounded selection ADRs |
| Monetization and go-to-market | 30 | MIG-03; DEL-15; claims/cost package |
| Clean maintainable architecture | 2..3,5,15,20 | Owner zone maps, seam tests, contracts, vanilla regression |
| Extremely detailed goal-ready plan with absolute artifacts | All | Plan Manager structure receipt and source-manifest.json |

## Findings addressed before finalization

- Replaced the initial separate-desktop-engine recommendation with source-backed Device Control ownership.
- Preserved the native Portal directory placement and normal expanded UI implementation.
- Distinguished companion host, API host, Bridge node, desktop user session, surface, and attached device.
- Added explicit validation scopes because the current Plan Manager runtime requires them for multi-scenario phases.
- Corrected reference classification for JSON source artifacts.
- Corrected decision input syntax to match the owner parser.
- Removed a circular ledger dependency on its future Plan Manager completion receipt.
- Preserved terminal session behavior and old Assistant data through positive replacement tests.
- Added the existing Linux live-desktop harness boundary, preventing duplicate production control ownership.
- Defined primary platform families without pretending exact OS versions were inspected or live-tested.
- Separated current commands from proposed API/configuration examples.
- Marked source snapshots and prior reports as context rather than regression evidence.
- Required signed-release evidence for the declared release class without authorizing public publishing.
- Added bounded exploratory and deterministic replay performance measures with separate provenance.
- Preserved unknown effects after disconnection; a durable command ID does not guarantee transactional GUI execution.

## Unresolved implementation decisions

The helper language/dependency selection, screen transport choice, exact primary host versions, and extension schema details are explicit decision gates. They cannot change the owner architecture or silently remove mandatory outcomes. Missing credentials/hosts remain unresolved acceptance, not permission to claim completion.

## Artifact validation

The source-manifest snapshots have matching SHA-256 hashes. All absolute CODE/DOC references in the rendered plan resolve at authoring review. The pending ledger correctly fails its completeness check. The plan contains six architecture/state/sequence Mermaid diagrams before the phase dependency diagram is added. Owner structure validation is recorded separately from live product validation.

## Prior-plan reuse

Portal v0 describes the implemented chat, branch, readiness, and passive search foundation, despite its stored phases remaining todo. Current source therefore takes precedence over that status. The completed Web Console multi-device sizing plan contributes authoritative size lease, follower framing, viewer presence, and takeover behavior that this plan must preserve. Bridge acknowledged delivery and Device Control resolution-ladder plans provide related contracts, not blanket proof of this new product. Desktop maturity and release-trust plans contribute packaging/provenance context.

## Final authoring validation

Plan Manager reports valid with no violations. Git Control Tower source-scope advisory completed; ignored files were excluded by default. Six structural ledger probes passed, including pending, duplicate, missing-artifact, skipped-platform, unknown-regression, and noncircular pre-completion checks. These are authoring-tool checks only.
