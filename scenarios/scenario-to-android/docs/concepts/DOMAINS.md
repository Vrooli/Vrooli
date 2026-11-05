# Domains — Scenario to Android

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/scenario-to-android/v1/shared/health.proto` |
| projects | Answer "what generated Android app exists for which scenario, at which revision?" | Own Capacitor shell generation so a scenario becomes a buildable Android project without its UI being rewritten. | Generated-project records, shell template selection, bundle identity. | service | crud | Project, ShellTemplate, BundleIdentity | `api/internal/projects/` |
| targets | Answer "which Android targets can I prove right now, and what is missing?" | Implement the spine `Prober` seam and own emulator provisioning, so availability is probed rather than inferred. | Capability snapshots, AVD records, probe history. | reporting | query, integration | Target, CapabilitySnapshot, AVD, Transport | `api/internal/targets/` |
| builds | Answer "how does a project become a signed artifact?" | Implement the spine `Builder` seam: Gradle invocation, target-API assertion, and signing by secret reference. | Build records, artifact references, assertion results. | workflow | service | Build, Artifact, TargetApiAssertion, SigningIdentity | `api/internal/builds/` |
| journeys | Answer "does the generated app behave like a Vrooli app on Android?" | Implement the spine `Driver` seam by composing device-control verbs and browser-automation-studio flows onto one chaptered timeline. | Journey plans, run chapters, evidence references. | workflow | integration | JourneyPlan, Chapter, Assertion, Lease | `api/internal/journeys/` |
| releases | Answer "may this artifact ship?" | Compute the fail-closed release gate over the matrix adapter in the shared ramp composition layer, emitting reference-only verdicts. | Matrix run selections, gate verdicts. | workflow | reporting | MatrixRun, Cell, Disposition, TargetVerdict | `api/internal/releases/`, `api/handlers/releases/` |
| distribution | Answer "where may this artifact legitimately go?" | Implement the spine `Distributor` seam over channels with genuinely different identity requirements. | Channel availability records, upload receipts. | service | integration | Channel, PlayTrack, SideloadRoute, UploadReceipt | `api/internal/distribution/` |
| readiness | Answer "what stands between me and shipping this?" | Own the probed Google release-readiness ladder that backs the CLI, the UI walkthrough, and the generated documentation. | Rung state, probe results, next actions. | reporting | query | Rung, LadderState, NextAction | `api/internal/readiness/` |

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

- Purpose: turn a target scenario into a buildable Capacitor Android
  project without modifying that scenario's UI source.
- Primary archetype: service / crud.
- Owns: generated-project records, shell template selection, bundle
  identity derivation, and generation reproducibility.
- Does not own: the build itself (`builds`), or the target scenario's UI.
- Key invariant: generation is reproducible — the same scenario at the
  same revision produces the same project, so a rebuild is comparable.
- Shell templates are a family, not a hard-coded choice. Only the
  Capacitor template ships at P0; adding a second must not require
  changing any other domain.
- Requirements: `S2A-P0-001`, `S2A-P2-003`, `S2A-P2-004`.

### targets

- Purpose: report what this host — or a reachable bridge node — can
  actually prove for Android, and provision the emulator when it can.
- Primary archetype: reporting / query.
- Secondary traits: integration (bridge, device-control), provisioning.
- Owns: the spine `Prober` implementation, capability snapshots, AVD
  lifecycle for a validation run, and probe history.
- Does not own: device identity or trust (`vrooli-bridge`), device verbs
  (`device-control`), or toolchain acquisition (the `android-sdk`
  resource).
- Key invariant: availability is decided by probing, never by
  `runtime.GOOS`. `/dev/kvm` presence and writability is a probed
  capability because an unaccelerated emulator cannot produce usable
  video evidence — an absent one yields `unavailable`, never a slow pass.
- Requirements: `S2A-P0-003`, `S2A-P0-006`, `S2A-P1-002`.

### builds

- Purpose: produce a signed, distributable artifact from a generated
  project.
- Primary archetype: workflow / service.
- Owns: the spine `Builder` implementation, Gradle invocation, APK and
  AAB outputs, the target-API assertion, and signing by secret reference.
- Does not own: key material — `secrets-manager` holds it and this domain
  never reads it into the repository, a generated project, or evidence.
- Key invariant: the target-API floor is a machine assertion recorded in
  evidence with the observed value. A build below the Play floor fails
  closed naming the required level.
- Requirements: `S2A-P0-002`, `S2A-P0-007`.

### journeys

- Purpose: prove the generated app satisfies the Android platform
  contract, and let a scenario's own flows replay unchanged inside it.
- Primary archetype: workflow.
- Secondary traits: integration (device-control, browser-automation-studio).
- Owns: the spine `Driver` implementation, the registered conformance
  capability plan, chapter definitions and their bounded policies, and the
  composition of native and web steps onto one timeline.
- Does not own: device verbs (`device-control`), web-content automation
  (`browser-automation-studio`), or the evidence schema (the spine).
- Key invariant: the runner has no scenario-name branch. The conformance
  plan is registered and scenario-agnostic, so it is identical for every
  scenario and differs only by platform.
- Key invariant: a run holds a `device-control` lease for its full
  duration. Without it a concurrent task silently corrupts the evidence.
- Requirements: `S2A-P0-004`, `S2A-P0-005`, `S2A-P0-012`, `S2A-P1-001`,
  `S2A-P1-005`, `S2A-P1-006`.

### releases

- Purpose: decide, fail-closed, whether an artifact may ship.
- Primary archetype: workflow / reporting.
- Owns: matrix selection for Android profiles, gate computation, and
  emission of `common.v1.TargetVerdict` to deployment-manager.
- Does not own: matrix mechanics — the spine owns selection immutability,
  rerun, compare, wait, and abort. This domain chooses the cells and
  composes the verdict.
- Key invariant: `unavailable` and `unsupported` are terminal and can
  never be promoted to a pass; a missing required cell fails the gate
  rather than being omitted from it.
- Key invariant: verdicts are reference-only. Capture bytes, recordings,
  keys, and endpoints never cross the governance boundary.
- Requirements: `S2A-P0-010`.

### distribution

- Purpose: state where an artifact may legitimately go, per channel.
- Primary archetype: service / integration.
- Owns: the spine `Distributor` implementation, the channel model (Play,
  verified sideload, ADB internal), per-channel availability, and upload
  receipts.
- Does not own: the release gate (`releases`) or account state
  (`readiness`).
- Key invariant: each channel reports its own availability and reason. A
  channel is never presented as available because a sibling channel is —
  they have genuinely different identity requirements, and Google's
  developer-verification rollout makes that divergence grow over time.
- Requirements: `S2A-P0-008`, `S2A-P1-003`, `S2A-P1-007`, `S2A-P2-002`.

### readiness

- Purpose: make the path from no account to production listing a probed,
  ordered, actionable model rather than tribal knowledge.
- Primary archetype: reporting / query.
- Owns: the Google rung ladder, per-rung probes, automation
  classification, and next actions.
- Does not own: performing the human steps. Registration, identity
  verification, and payment are irreducibly manual and the model says so.
- Key invariant: one model backs the CLI, the UI walkthrough, and the
  generated documentation, so documentation cannot drift from code.
- Requirements: `S2A-P0-009`.

## Android P0 ownership register

Each P0 requirement has one product-domain owner. The `Builder` seam owns
both project generation and artifact production because generation and build
share one reproducible project-preparation boundary; the `projects` domain is
reserved for persisted project records and additional shell families.

| Requirement | Owning domain | Primary implementation evidence |
|---|---|---|
| `S2A-P0-001` | builds | `api/internal/builds/builder_test.go` |
| `S2A-P0-002` | builds | `api/internal/builds/builder_test.go` |
| `S2A-P0-003` | targets | `api/internal/targets/prober_test.go` |
| `S2A-P0-004` | journeys | `api/internal/journeys/driver_test.go` |
| `S2A-P0-005` | journeys | `api/internal/journeys/plan_test.go` |
| `S2A-P0-006` | targets | `../../resources/android-sdk/cli/main_test.go` |
| `S2A-P0-007` | builds | `api/internal/builds/builder_test.go` |
| `S2A-P0-008` | distribution | `api/internal/distribution/distributor_test.go` |
| `S2A-P0-009` | readiness | `api/internal/readiness/readiness_test.go` |
| `S2A-P0-010` | releases | `api/internal/releases/gate_test.go` |
| `S2A-P0-011` | releases | `cli/domains/android/register_test.go` |
| `S2A-P0-012` | journeys | `api/internal/journeys/driver_test.go` |
| `S2A-P0-013` | builds | `api/internal/builds/builder_test.go` |

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
| `formfactors` | Wear OS, Android TV, and Android XR carry their own Play target-API floors and their own conformance chapters. Modelling them before the phone path is proven would generalise against one example. | `S2A-P2-001` is picked up, or a second form factor is actually required. |
| `attestation` | Play Integrity verdicts are only meaningful once real distribution exists; recording them now would produce evidence nobody consumes. | `S2A-P2-002`, or a release gate needs to distinguish attested from unverified runs. |
| `bundles` | Multi-scenario applications change bundle identity, routing, and readiness at once. It is a different product, not a bigger version of this one. | `S2A-P2-004`. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- `cli/domains/` — CLI registration and transport adapters.
- `ui/src/features/delivery/` — shared delivery-ramp operator walkthrough.
- `../../resources/android-sdk/cli/` — governed resource-owned SDK lifecycle.
- `../../hello-mobile/ui/` — fixture-owned end-to-end test surface.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
