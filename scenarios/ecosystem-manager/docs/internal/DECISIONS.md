# Decisions — Ecosystem Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Rationale | Status |
|---|---|---|---|
| 2026-01 | Consolidate four legacy tools (resource/scenario × generator/improver) into one unified Ecosystem Manager. | A single control plane for generating and improving both scenarios and resources eliminates duplicated loop logic and gives one queue, one profile system, and one metrics store. | ACCEPTED |
| 2026-01 | Store auto-steer profiles on the filesystem (`profiles/*/profile.json` + `metadata.json`); keep only execution state and history in a relational store. | Profiles are human-authored, version-controlled configuration (objective functions). They belong in git alongside code, not in a database. The database holds only the mutable runtime state. `[CODE: profiles/]` | ACCEPTED (store was Postgres; cut over to embedded SQLite 2026-06 — see below) |
| 2026-01 | Persist the task queue as YAML under `queue/<status>/`, using the directory name as the status and atomic file-moves for transitions. | Directory-as-status makes state legible and transitions atomic at the filesystem level (move = commit), avoiding a separate transactional store for the queue. `[CODE: queue/]` | ACCEPTED |
| 2026-01 | Use REST/JSON transport for the API. | Predates the proto + Connect-RPC standard now used across Vrooli; was the available option at the time. | SUPERSEDED-IN-PRINCIPLE / drift — see Superseded Decisions and [`PROBLEMS.md`](PROBLEMS.md) |
| 2026-05-30 | Reframe auto-steer as a **closed-loop controller** and profiles as **objective functions** (rather than fixed ordered scripts). | The "profile = fixed ordered script" mental model undersells the system. Treating it as a controller — profiles define the objective, the loop adaptively selects skills against measured findings — is the correct frame and unlocks adaptive skill selection, findings-based state, an effectiveness table, DTV priors, and thrashing defense. Docs are written first to anchor the model; implementation is future work in [`PROBLEMS.md`](PROBLEMS.md). See [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md). | ACCEPTED / DIRECTIONAL |
| 2026-05-31 | Aggregate per-skill DTV fitness **inside DTV** via a new `GetSkillFitness` RPC, rather than crunching raw validation records EM-side. | DTV owns the append-only records; deriving one skill's trust/cost/convergence client-side would mean N×M round-trips (enumerate every golden, paginate every tuple) and re-implementing aggregation in every consumer. One server-side fold is the "fix the substrate, design to the ideal" choice. EM consumes the aggregate through a Connect client. `[CODE: pkg/dtv/, scenarios/development-toolchain-validator/api/internal/report]` | ACCEPTED |
| 2026-05-31 | DTV is a **gate and a prior, never a source of efficacy**; the eligibility gate denies only RED, UNKNOWN fails open, and any DTV outage degrades to exact P1 (uniform prior, allow-all). | Goldens are pristine (zero findings), so DTV can never observe "skill closed N findings" — efficacy stays learned live by the bandit. Gating UNKNOWN would break first-run scenarios, and a synchronous DTV dependency would risk stalling the control loop; fail-open keeps P2 a strict enhancement of P1. The prior is blended via Bayesian shrinkage so live evidence washes it out regardless. `[CODE: pkg/autosteer/dtv_selection.go]` | ACCEPTED |
| 2026-06 | Hard cut-over from PostgreSQL to an **embedded SQLite** file, opened through the `api-core/storage` seam. No `vrooli-postgres-main` container, no `vrooli_ecosystem_manager` DB, no `POSTGRES_*` env. Schema is **domain-owned** (`api/pkg/{autosteer,effectiveness,steering}/schema.sql`, each with a `Schema()` func registered in `api/pkg/dbschema`'s `AllSchemas()`) and applied at boot by `database.EnsureSchemas`. The dead `task_executions` / `operation_metrics` tables and the central `initialization/postgres/schema.sql` were dropped. | EM is an internal single-operator control plane; an embedded file removes a heavyweight shared resource, makes storage variant-isolatable (Baseline Modes shadow), and lets each domain own its own schema. Backup is via data-backup-manager coverage of the storage root. `[CODE: api/pkg/storagepaths, api/pkg/dbschema]` | ACCEPTED |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| 2026-01 | REST/JSON API transport. | proto + Connect-RPC (current Vrooli standard). | Superseded in principle; migration deferred. The REST surface remains in service. Tracked in [`PROBLEMS.md`](PROBLEMS.md). |
| 2026-01 | PostgreSQL (`vrooli_ecosystem_manager` in `vrooli-postgres-main`) as the runtime store, schema bootstrapped from `initialization/postgres/schema.sql`. | Embedded SQLite via `api-core/storage` with domain-owned schemas applied by `database.EnsureSchemas` (2026-06 cut-over). | Hard cut-over — Postgres is fully removed (no fallback). See the 2026-06 row in the active log above. |
| (implicit) | "Profile = fixed ordered script" mental model. | Closed-loop controller / objective-function model (2026-05-30). | Reframe is directional; the script-style execution still runs until the controller implementation lands. See [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md). |

## Cross-References

- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller model and objective functions
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system structure and decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt (transport migration, controller implementation)
- [`PROGRESS.md`](PROGRESS.md) — completed work history
