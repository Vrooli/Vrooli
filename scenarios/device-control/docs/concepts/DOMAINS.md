# Domains — Device Control

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

The domain split follows one rule: **each domain owns exactly one question,
and no domain answers two.** The test is deletion — removing a strategy
should touch only `strategies`; adding a new phone should touch only
`devices`; changing how a target is located should touch only `flows`.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/device-control/v1/shared/health.proto` |
| devices | Answer "what devices do I control and what can each actually do right now?" | Turn the bridge fleet into a probed, per-device capability inventory that never claims an unproven capability. | Device records, capability snapshots, health history. | reporting | query, integration | Device, AttachedDevice, CapabilitySnapshot, Reachability | `api/internal/devices/`, `api/handlers/devices/`, `cli/domains/devices/`, `ui/src/features/devices/` |
| strategies | Answer "how can this device be driven?" | Own the pluggable adapter registry, the capability floor contract, and the conformance suite that verifies a strategy's declaration. | Strategy registrations, conformance results. | service | registry | Strategy, CapabilityFloor, CapabilityTier, ConformanceResult | `api/internal/strategies/`, `api/internal/strategies/<adapter>/`, `api/handlers/strategies/`, `cli/domains/strategies/`, `ui/src/features/strategies/` |
| sessions | Answer "who is driving this device right now, and may they?" | Own leases, the live control channel, and the raw verb surface. Enforce single-session safety and immediate kill. | Lease records, session state, verb audit. | service | lifecycle, workflow | Session, Lease, Verb, KillSwitch | `api/internal/sessions/`, `api/handlers/sessions/`, `cli/domains/sessions/`, `ui/src/features/sessions/` |
| flows | Answer "what should be done, and can this device do it?" | Own the flow envelope, the step vocabularies, target resolution, the pre-execution capability gap report, execution, and chaptered evidence. | Flow definitions, runs, steps, evidence references, visual anchors. | workflow | crud, service | Flow, Step, ResolutionRung, CapabilityGap, Run, Chapter, Anchor | `api/internal/flows/`, `api/handlers/flows/`, `cli/domains/flows/`, `ui/src/features/flows/` |
| agent | Answer "can something work this out for me?" | Own goal-directed runs: spawn an agent-manager run driven by the prompt-manager skill, bound it, audit it, and promote a successful run into a deterministic flow. | Agent run records, goal state, promotion provenance. | workflow | integration | AgentRun, Goal, Bound, Promotion | `api/internal/agent/`, `api/handlers/agent/`, `cli/domains/agent/`, `ui/src/features/agent/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/device-control/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

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

### devices

- Purpose: turn the `vrooli-bridge` fleet into a per-device capability
  inventory that reports only what has been probed.
- Primary archetype: reporting / query.
- Secondary traits: integration (bridge), periodic probing.
- Owns: device records, capability snapshots, reachability and health
  history, and the probe schedule.
- Does not own: device identity, pairing, trust, or scopes — `vrooli-bridge`
  owns all of it. This domain reads the fleet and adds device-specific
  probing on top.
- Key invariant: a capability that cannot be proven at probe time is
  reported `unavailable` with the exact missing prerequisite and a next
  action. It is never reported as present because the device *kind*
  usually has it.
- Reachability rule: an attached device whose host node is offline reports
  `unreachable` carrying that reason. It never silently disappears from the
  inventory, because a vanishing device reads as "not mine" rather than
  "temporarily unavailable."
- Requirements: `DVC-P0-003`.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md).

### strategies

- Purpose: own the pluggable adapter contract so that adding a way to drive
  a device never requires changing the executor.
- Primary archetype: service / registry.
- Secondary traits: conformance verification.
- Owns: the capability floor interface, strategy registration, the declared
  capability manifest, and the conformance suite.
- Does not own: what to do with a device (`flows`), or permission to do it
  (`sessions`).
- The capability floor — every strategy implements exactly these three:
  - `observe() -> Frame` — a screen image plus size, scale, and timestamp.
  - `actuate(PointerEvent | KeyEvent)` — press, release, move, key input.
  - `describe() -> CapabilityDeclaration` — what else this strategy offers.
- Everything else is optional and declared: semantic tree, app lifecycle,
  permission control, network control, orientation, clipboard, file
  transfer, video recording, device logs.
- Why the floor is this small: it is the largest set that
  `ios-mirror` — pixels and synthetic HID events only — can satisfy. Making
  the floor any richer would exclude a strategy we need, and shared
  capabilities are built against the floor so they work everywhere.
- Launch strategies: `android-adb` (P0), then `ios-simctl`,
  `ios-xcuitest`, `ios-mirror`, `host-desktop` (P1).
- Requirements: `DVC-P0-001`, `DVC-P0-002`, `DVC-P0-011`,
  `DVC-P1-001`..`DVC-P1-004`.

### sessions

- Purpose: make device access exclusive, observable, and revocable.
- Primary archetype: service / lifecycle.
- Secondary traits: audit, real-time channel.
- Owns: lease issue/renew/expire/release, the live control channel, the raw
  verb surface, per-verb audit records, and the kill switch.
- Does not own: trust or scope grants — `vrooli-bridge` owns those; this
  domain enforces exclusivity *within* an already-authorized reach.
- Key invariant: no verb reaches a strategy without a held, unexpired lease.
- Why leases are P0 rather than a later refinement: several strategies are
  physically single-session (iPhone Mirroring holds one connection; a
  WebDriverAgent session owns the device). Without a lease, two consumers do
  not collide loudly — they interleave and quietly corrupt each other's
  evidence, which is the hardest class of failure to diagnose after the fact.
- Requirements: `DVC-P0-004`, `DVC-P0-009`.

### flows

- Purpose: express what should happen on a device, prove up front that this
  device can do it, execute it, and record chaptered evidence.
- Primary archetype: workflow.
- Secondary traits: CRUD (definitions), service (execution).
- Owns: the flow envelope, step vocabularies, the target resolution ladder,
  the capability gap report, run execution, bounded wait policies, evidence
  chapters, and the visual-anchor library.
- Does not own: web-content automation semantics — a `bas.*` step delegates
  to `browser-automation-studio` and merges its result into one timeline.
- Step vocabularies in one envelope:
  - `device.*` — tap, swipe, type, key, launch, install, stop, uninstall,
    permission, observe, record.
  - `ai.*` — see, decide, extract, verify. Always through `ai-gateway`.
  - `flow.*` — conditional, loop, subflow, parallel.
  - `wait.*` — named bounded readiness and settle policies with upper
    bounds. Never a raw sleep; an exceeded bound is an evidence-visible
    failure.
  - `bas.*` — attach to a WebView and run a named BAS flow.
- The target resolution ladder — a step names its target by intent, and the
  resolver satisfies it at the highest rung the strategy supports:
  1. `semantic` — match against an accessibility or view tree.
     Deterministic, fast, no inference cost.
  2. `visual-anchor` — match a captured reference image.
     Deterministic, cheap, no semantic tree required.
  3. `vision` — ask `ai-gateway` to locate the target in the frame.
     Works on any strategy that satisfies the floor; costs tokens and is
     not deterministic.
  The chosen rung and its confidence are recorded in evidence so a reviewer
  can tell a proven result from an inferred one.
- Why this ladder is the design: it is what makes "shared capabilities work
  on any strategy" true rather than aspirational. Vision is the universal
  fallback; semantics are the fast path. The same flow runs on both, and
  the strategy — not the flow author — determines which rung is used.
- Requirements: `DVC-P0-005`, `DVC-P0-006`, `DVC-P0-007`, `DVC-P0-008`,
  `DVC-P1-007`, `DVC-P1-009`.

### agent

- Purpose: complete a goal on a device when no flow exists yet.
- Primary archetype: workflow.
- Secondary traits: integration (agent-manager, prompt-manager).
- Owns: goal-directed run lifecycle, bounds (step count, cost ceiling,
  lease scope), abort, and promotion of a successful run into a flow.
- Does not own: the agent runtime (`agent-manager`) or the operator skill
  content (`prompt-manager`).
- Why the CLI is the contract: the skill drives this scenario through its
  CLI, so any capability missing from the CLI is invisible to the agent.
  That is why CLI completeness is a P0 target rather than a convenience.
- The compounding loop: every agent action is recorded as a flow step, so a
  successful run exports as a deterministic flow. Infer once, replay free.
- Requirements: `DVC-P1-005`, `DVC-P1-006`.

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
| `evidence` | Captures, checksums, and redaction are currently owned by `flows` and `sessions` because both produce them and neither consumes the other's. Splitting now would create a domain with two callers and no independent vocabulary. | Split when a third producer appears, or when evidence retention/lifecycle policy grows rules of its own. |
| `scheduling` | Scheduled and event-triggered runs are P2. Until then a run is always initiated by an operator, an agent, or a delivery ramp. | Split when `OT-P2-002` starts. |
| `anchors` | The visual-anchor library is a table and a matcher inside `flows`. It has no lifecycle of its own yet. | Split if anchors gain sharing, versioning, or cross-scenario reuse. |

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
