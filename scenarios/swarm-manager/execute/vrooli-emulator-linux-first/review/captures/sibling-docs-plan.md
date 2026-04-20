# Implementation Plan — `chore/vrooli-emulator-documentation`

> **Status:** Scope-only stub. A full plan must be authored before execution; this file currently records the deliverables this item owns (including absorbed scope) so that planning work is not lost.

## Scope

### Original scope (from spec.json description)

- Document the `/api/v1/sessions/` API contract, including the `headless: true` path and the required `app_path` field.
- Document the iframe embed / `/embedded/emulator/external-url` protocol for future consumers (deployment-manager visual validation, additional scenarios).
- Document the operator CLI surface.

### Absorbed scope (rehomed from retired `execute/vrooli-emulator-linux-first`)

The following deliverables were moved here on 2026-04-18 when `execute/vrooli-emulator-linux-first` was archived (see that item's plan.md §6 and §8 Phase A for rationale; round 2 decision `d1=A`):

- **PRD fill-out at `scenarios/vrooli-emulator/PRD.md`** — complete the Overview, Operational Targets, Tech Snapshot, Dependencies, and UX/Branding stubs.
- **Operator runbook at `scenarios/vrooli-emulator/docs/runbook.md`** — day-to-day operator procedures: starting/stopping the service, creating and destroying sessions, inspecting metrics, tailing logs, cleaning up stale sessions.
- **Operational baseline values-as-docs** — record the agreed defaults for:
  - max concurrent sessions
  - Xvfb display number range
  - session TTL
  - per-session resource ceilings (CPU, memory, disk)
  - stale-session janitor cadence

  These are *documentation of the defaults*, not the code that enforces them; enforcement lives in the code-owning sibling items.

## Source of the absorbed scope

`scenarios/swarm-manager/execute/vrooli-emulator-linux-first/plan.md` §6 Current Technical Context and §8 Phase A. Preserved here so the rehoming is auditable after the source item is archived.
