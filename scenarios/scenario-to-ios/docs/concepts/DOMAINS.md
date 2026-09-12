# Domains — Scenario to iOS

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `template-manager detemplate <scenario>` removes every
fenced example once your real domains are green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/scenario-to-ios/v1/shared/health.proto` |
| projects | Answer "what generated iOS app exists for which scenario, at which revision?" | Reserved for persisted project records and additional shell families; the P0 Capacitor generation boundary is owned by `builds`. | Deferred until project records or a second shell family exists. | service | crud | Project, ShellTemplate, BundleIdentity | — (reserved; P0 generation is `api/internal/builds/`) |
| targets | Answer "which iOS targets can I prove right now, and where must the work run?" | Implement the spine `Prober` seam, distinguish terminal `unsupported` from actionable `unavailable`, and resolve the build-host role. | Capability snapshots, build-host role state, probe history. | reporting | query, integration | Target, CapabilitySnapshot, BuildHostRole, ToolchainVersion | `api/internal/targets/`, `api/handlers/targets/`, `cli/domains/targets/`, `ui/src/features/targets/` |
| builds | Answer "how does a project become a signed artifact, on a machine that is not this one?" | Implement the spine `Builder` seam: remote `xcodebuild` invocation, SDK-floor assertion, and signing by secret reference. | Build records, artifact references, assertion results. | workflow | service, integration | Build, Artifact, SdkFloorAssertion, SigningIdentity | `api/internal/builds/`, `api/handlers/builds/`, `cli/domains/builds/`, `ui/src/features/builds/` |
| journeys | Answer "does the generated app behave like a Vrooli app on iOS?" | Implement the spine `Driver` seam by composing device-control verbs and browser-automation-studio flows onto one chaptered timeline. | Journey plans, run chapters, evidence references, strategy attribution. | workflow | integration | JourneyPlan, Chapter, Assertion, Lease, StrategyTier | `api/internal/journeys/`, `api/handlers/journeys/`, `cli/domains/journeys/`, `ui/src/features/journeys/` |
| releases | Answer "may this artifact ship?" | Drive the spine's immutable validation matrix and compute the fail-closed release gate, emitting reference-only verdicts. | Matrix run selections, gate verdicts, promotability records. | workflow | reporting | MatrixRun, Cell, Disposition, TargetVerdict, Promotability | `api/internal/releases/`, `api/handlers/releases/`, `cli/domains/releases/`, `ui/src/features/releases/` |
| distribution | Answer "where may this artifact legitimately go?" | Implement the spine `Distributor` seam over channels with genuinely different signing and review requirements. | Channel availability records, upload receipts. | service | integration | Channel, TestFlightTrack, AdHocRoute, UploadReceipt | `api/internal/distribution/`, `api/handlers/distribution/`, `cli/domains/distribution/`, `ui/src/features/distribution/` |
| readiness | Answer "what stands between me and shipping this?" | Own the probed Apple release-readiness ladder that backs the CLI, the UI walkthrough, and the generated documentation. | Rung state, probe results, next actions. | reporting | query | Rung, LadderState, NextAction | `api/internal/readiness/`, `api/handlers/readiness/`, `cli/domains/readiness/`, `ui/src/features/readiness/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### projects

- Purpose: reserve the future project-record and shell-family boundary.
  P0 Capacitor Xcode project generation belongs to the `Builder` seam in
  `builds`, so this domain remains deliberately unimplemented until it
  owns persisted project records or a second shell technology.
- Primary archetype: service / crud.
- Owns when activated: generated-project records and additional shell
  template selection. It does not own the current P0 generation path or
  the target scenario's UI.
- Key invariant: bundle identity is derived, not assigned by hand, so the
  App Store Connect record a build uploads to is reproducible.
- Requirements when activated: `S2I-P2-002`, `S2I-P2-003`.

### targets

- Purpose: report what can actually be proven for iOS, and resolve which
  node will do the work.
- Primary archetype: reporting / query.
- Secondary traits: integration (bridge, device-control).
- Owns: the spine `Prober` implementation, capability snapshots, the
  build-host role resolution, and recorded toolchain versions.
- Does not own: device identity or trust (`vrooli-bridge`), device verbs
  (`device-control`), or the Apple toolchain (a host capability).
- **The distinction this domain exists to get right**: a Linux host
  reports native iOS targets as `unsupported` — terminal, correct forever,
  never a gap to close — while a macOS node missing a prerequisite reports
  `unavailable` with the exact missing item. Collapsing those two into one
  state is the failure mode that would make this ramp dishonest.
- Probes Xcode version, installed simulator runtimes including the x86_64
  universal architecture variant required on Intel hosts, GUI session
  availability, and signing identity presence.
- **Build-host role**: the iOS build and signing host is a role satisfied
  by any qualifying node, never a named machine. The fulfilling node's
  macOS and Xcode versions are recorded in evidence, and a host that can
  still validate but no longer produce submittable builds reports that as
  a distinct state rather than a generic failure.
- Requirements: `S2I-P0-003`, `S2I-P0-004`.

### builds

- Purpose: produce a signed, distributable artifact from a generated
  project — on a remote macOS node, because no other kind of node can.
- Primary archetype: workflow / service, with integration as a first-class
  trait rather than an afterthought.
- Owns: the spine `Builder` implementation, remote `xcodebuild`
  invocation, simulator `.app` and device IPA outputs, the SDK-floor
  assertion, and signing by secret reference.
- Does not own: key material — `secrets-manager` holds certificates,
  provisioning profiles, and App Store Connect keys, and this domain never
  reads them into the repository, a generated project, or evidence.
- Key invariant: the Xcode and iOS SDK floor is a machine assertion with
  the observed version recorded. A toolchain below the App Store Connect
  floor fails closed naming the required version, rather than producing an
  artifact that will be rejected at upload.
- Requirements: `S2I-P0-002`, `S2I-P0-007`.

### journeys

- Purpose: prove the generated app satisfies the iOS platform contract,
  and let a scenario's own flows replay unchanged inside it.
- Primary archetype: workflow.
- Secondary traits: integration (device-control, browser-automation-studio).
- Owns: the spine `Driver` implementation, the registered conformance
  capability plan, chapter definitions and their bounded policies, and the
  composition of native and web steps onto one timeline.
- Does not own: device verbs (`device-control`), web-content automation
  (`browser-automation-studio`), or the evidence schema (the spine).
- **Strategy selection is this domain's judgement call**: a cell is
  executed through `ios-simctl`, `ios-xcuitest`, or `ios-mirror` depending
  on what it must prove. The selected strategy and its capability tier are
  recorded in evidence, because a reviewer must be able to tell a semantic
  assertion from a pixel-and-OCR one.
- Key invariant: the runner has no scenario-name branch.
- Key invariant: a run holds a `device-control` lease for its full
  duration. The mirroring strategy is physically single-session, which
  makes a missing lease a certainty of corruption rather than a risk.
- Requirements: `S2I-P0-005`, `S2I-P0-006`, `S2I-P0-012`, `S2I-P1-001`,
  `S2I-P1-002`, `S2I-P1-005`, `S2I-P1-006`.

### releases

- Purpose: decide, fail-closed, whether an artifact may ship.
- Primary archetype: workflow / reporting.
- Owns: matrix selection for iOS profiles, gate computation, promotability
  rules, and emission of `common.v1.TargetVerdict`.
- Does not own: matrix mechanics — the spine owns selection immutability,
  rerun, compare, wait, and abort.
- Key invariant: `unavailable` and `unsupported` are terminal and can
  never be promoted to a pass; a missing required cell fails the gate.
- **Promotability rule**: evidence produced through the `ios-mirror`
  strategy is recorded as non-promotable and can never contribute to a
  passing gate. OCR reads pixels rather than meaning and cannot
  distinguish a correct rendering from a convincing image of one.
- Requirements: `S2I-P0-010`.

### distribution

- Purpose: state where an artifact may legitimately go, per channel.
- Primary archetype: service / integration.
- Owns: the spine `Distributor` implementation, the channel model
  (simulator-only, development device, TestFlight, App Store), per-channel
  availability, and upload receipts.
- Does not own: the release gate (`releases`) or account state
  (`readiness`).
- Key invariant: each channel reports its own availability and reason. A
  channel is never presented as available because a sibling channel is —
  simulator-only needs no signing at all, while App Store needs enrollment,
  signing, a Connect record, and review.
- Requirements: `S2I-P0-008`, `S2I-P1-003`, `S2I-P1-007`, `S2I-P2-004`.

### readiness

- Purpose: make Apple's path — the most opaque part of shipping an iOS
  app — a probed, ordered, actionable model.
- Primary archetype: reporting / query.
- Owns: the Apple rung ladder, per-rung probes, automation
  classification, and next actions.
- Does not own: performing the human steps. Enrollment, identity
  verification, and payment are irreducibly manual, and the surface says
  so plainly rather than implying automation that cannot exist.
- Key invariant: one model backs the CLI, the UI walkthrough, and the
  generated documentation, so documentation cannot drift from code.
- Requirements: `S2I-P0-009`.

## iOS P0 ownership register

Each P0 requirement has one product-domain owner. Capacitor project
generation is intentionally owned by `builds` because it shares the
reproducible project-preparation boundary with artifact production; the
`projects` domain is reserved for future persisted records and alternate
shell families.

| Requirement | Owning domain | Primary implementation evidence |
|---|---|---|
| `S2I-P0-001` | builds | `api/internal/builds/builder_test.go` |
| `S2I-P0-002` | builds | `api/internal/builds/builder_test.go` |
| `S2I-P0-003` | targets | `api/internal/targets/prober_test.go` |
| `S2I-P0-004` | targets | `api/internal/targets/prober_test.go` |
| `S2I-P0-005` | journeys | `api/internal/journeys/driver_test.go` |
| `S2I-P0-006` | journeys | `api/internal/journeys/plan_test.go` |
| `S2I-P0-007` | builds | `api/internal/builds/builder_test.go` |
| `S2I-P0-008` | distribution | `api/internal/distribution/distributor_test.go` |
| `S2I-P0-009` | readiness | `api/internal/readiness/readiness_test.go` |
| `S2I-P0-010` | releases | `api/internal/releases/gate_test.go` |
| `S2I-P0-011` | releases | `cli/domains/ios/register_test.go` |
| `S2I-P0-012` | journeys | `api/internal/journeys/driver_test.go` |

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `formfactors` | iPadOS layout specifics, visionOS, and watchOS each carry their own SDK floor and conformance chapters. Modelling them before the iPhone path is proven would generalise against one example. | `S2I-P2-001` is picked up, or a second form factor is actually required. |
| `review` | App Review submission and rejection tracking only becomes meaningful once TestFlight distribution works end to end. | `S2I-P2-004`. |
| `bundles` | Multi-scenario applications change bundle identity, routing, and readiness at once. It is a different product, not a bigger version of this one. | `S2I-P2-003`. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
