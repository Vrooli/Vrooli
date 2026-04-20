# Implementation Plan — `execute/emulator-acceptance-tests-phase-1`

> **Status:** Scope-only stub. A full plan must be authored before execution; this file currently records the deliverables this item owns (including absorbed scope) so that planning work is not lost.

## Scope

### Original scope (from spec.json description)

- Integration tests against the emulator service in isolation (no scenario-to-desktop dependency).
- Cover: create VNC session, create headless session with DISPLAY returned, launch an app from explicit `app_path`, take screenshot, tail metrics, destroy session.
- CLI smoke test exercising `session list/create/destroy/exec/logs`.
- See `research/emulator-extraction-and-service-plan` conclusion Findings 11 and 13 for the source analysis.

### Absorbed scope (rehomed from retired `execute/vrooli-emulator-linux-first`)

The following deliverables were moved here on 2026-04-18 when `execute/vrooli-emulator-linux-first` was archived (see that item's plan.md §6 and §8 Phase A for rationale; round 2 decision `d1=A`):

- **Phase 1 integration smoke harness** — a scripted end-to-end exercise that stands up the emulator, creates both a VNC session and a headless session, takes a capture, tails metrics, and tears everything down. This is the runnable equivalent of the originally-envisioned `make phase1-ready` target.
- **Per-distro validation matrix** — at minimum Ubuntu LTS 22.04 and 24.04. Other distros are deferred until a consumer requests them, to avoid unbounded validation scope.

## Source of the absorbed scope

`scenarios/swarm-manager/execute/vrooli-emulator-linux-first/plan.md` §6 Current Technical Context and §8 Phase A. Preserved here so the rehoming is auditable after the source item is archived.
