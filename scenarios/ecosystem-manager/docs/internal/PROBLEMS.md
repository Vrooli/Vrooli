# Problems — Ecosystem Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-05-30 — Auto-steer is an open-loop schedule, not a state-aware controller

**Symptom:** When ecosystem-manager auto-steers a target toward a maturity
profile, it does not adapt to the target's actual failing dimensions. The
same skill order runs regardless of what is actually broken; ineffective
skills are re-selected; two skills can undo each other's work across
iterations and only the max-iterations backstop stops it. The system
reaches profile exit gates by brute-force iteration rather than by
reasoning about which skill is most likely to move a failing dimension.

**Root cause:** Auto-steer selects skills by a **fixed per-phase order
drawn from the profile**. It is an open-loop schedule with metric-based
exit gates, not a closed-loop controller. Specifically it does **not**:
(a) diagnose the target's current failing dimensions and pick the most
relevant skill for them; (b) learn per-`(skill, dimension)` effectiveness
over time; (c) detect inter-skill oscillation / thrashing at runtime —
only the blunt max-iterations backstop exists; (d) consume
development-toolchain-validator (DTV) signals as skill trust/cost priors
or as an eligibility gate. There is no findings-set state, no
skill→dimension capability map, and no effectiveness table.

**Workaround:** Tune the per-phase profile order by hand and rely on the
max-iterations cap to terminate runs that thrash. Operators watch
[`SystemLogsModal`](../../ui/src/components/modals/SystemLogsModal.tsx)
to spot oscillation manually.

**Real fix:** Implement the closed-loop controller specified in
[`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md): profiles
as objective functions, a findings-set as controller state, a
skill→dimension capability map, a learned per-`(skill, dimension)`
effectiveness table, DTV signals as trust/cost priors and an eligibility
gate, and a three-layer thrashing defense (replacing the lone
max-iterations backstop). This is the **motivating next phase** after the
2026-05-30 docs overhaul. Implementation is **not started**.

**Owner:** unassigned

**Refs:** `api/pkg/autosteer/phase_coordinator.go`,
`api/pkg/autosteer/execution_state_manager.go`,
`api/pkg/autosteer/profile_repository_fs.go`,
[`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md)

### 2026-05-30 — Metrics collection re-runs full target builds/tests each iteration

**Symptom:** Each auto-steer iteration is expensive; wall-clock per
iteration is dominated by re-building and re-testing the entire target
even when a skill only touched a narrow slice of it.

**Root cause:** Metrics collection re-runs the target's builds/tests
wholesale every iteration. There is no targeted or incremental re-audit
that scopes work to the dimensions a skill could have affected.

**Workaround:** Keep max-iterations conservative so cost stays bounded;
prefer profiles with fewer phases for quick passes.

**Real fix:** Add incremental / targeted re-audit so only the affected
dimensions are re-measured between iterations. This also feeds the
closed-loop controller (it needs per-dimension deltas, not whole-target
verdicts).

**Owner:** unassigned

**Refs:** `api/pkg/queue/execution_manager.go`,
`api/pkg/autosteer/phase_coordinator.go`

### 2026-05-30 — No external metrics/tracing export; alerting is manual

**Symptom:** Operators cannot see auto-steer health from outside the
scenario. There is no metrics endpoint, no distributed tracing, and no
automatic alerting on stuck or thrashing runs.

**Root cause:** Observability is file-based logging only
(`api/pkg/systemlog/log.go`). No external metrics/tracing exporter is
wired.

**Workaround:** Tail logs via the CLI or the
[`SystemLogsModal`](../../ui/src/components/modals/SystemLogsModal.tsx);
operators alert each other manually.

**Real fix:** Export metrics/traces to a standard collector and add
threshold-based alerting on iteration count, oscillation, and run
duration.

**Owner:** unassigned

**Refs:** `api/pkg/systemlog/log.go`

### 2026-05-30 — Operator context-switching under failure is high (UX friction)

**Symptom:** During failure scenarios operators bounce between logs,
settings, and task details; dense task cards can hide key blockers
without opening the details modal.

**Root cause:** Failure-relevant context is spread across separate
modals rather than surfaced inline on the board.

**Workaround:** Open the task details modal early; keep the system-logs
modal open alongside the board.

**Real fix:** Surface blockers inline on task cards and consolidate the
failure-investigation surfaces.

**Owner:** unassigned

**Refs:** `ui/src/components/kanban/TaskCard.tsx`,
`ui/src/components/modals/SystemLogsModal.tsx`,
`ui/src/components/modals/TaskDetailsModal.tsx`

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Transport | The scenario's own HTTP API is **REST/JSON over gorilla/mux** (`api/pkg/server/server.go` uses `mux.NewRouter()`; no `connect-go`/`connectrpc` dependency), not proto/Connect-RPC — the current Vrooli transport standard. Proto schemas under `packages/proto/schemas/ecosystem-manager/v1/` **do** exist, but they back the agent-manager client serialization and UI response validation only — they do not serve Connect-RPC. Most Go handlers still return `map[string]any`/embedded structs rather than proto response types. | Cross-scenario anti-drift guarantees from proto+Connect are absent; UI relies on fallback normalization at the boundary. | Migrate the API surface to Connect-RPC and return proto response types from handlers. Migration deferred — significant surface (8 domains, ~30 handlers). See [`DECISIONS.md`](DECISIONS.md). |
| Docs contract | Internal docs predated the v2 scenario-docs contract (ASSUMPTIONS/COHERENCE-NOTES/ERROR-SEMANTICS/EXPERIENCE-AUDIT/INTEROP_AUDIT/INVARIANTS/SECURITY-POSTURE/TEMPORAL-FLOWS). | Stale docs misled agents (e.g., the old INTEROP audit overstated proto adoption). | Overhauled to the v2 contract on 2026-05-30 (this work); stale files folded into PROBLEMS/PROGRESS and retired, and `CONTROL-MODEL.md` authored. |
| Auto-steer control | Auto-steer is an open-loop fixed-order schedule with metric exit gates, not a closed-loop, state-aware controller (see Entry 2026-05-30). | Reaches goals by brute-force iteration; no thrashing detection beyond max-iterations; no DTV priors. | Implement the controller in [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md). Not started — next phase. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs (incl. deferred Connect-RPC migration)
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller mental model (target design for auto-steer)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
