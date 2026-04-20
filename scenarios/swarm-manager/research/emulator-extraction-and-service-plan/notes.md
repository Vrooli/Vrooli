# Completion Summary

Processed the research conclusion for `research/emulator-extraction-and-service-plan`. All nine actions from the `## Actions` section were executed as specified: they were all "Create backlog item" actions, so the output is nine new backlog items all belonging to the existing `emulator-platform` initiative.

## Actions Taken

Single `swarm-manager backlog batch-create` call produced nine items:

### Phase 1 — Emulator stand-up + scenario-to-desktop livedesktop removal

- `execute/scaffold-vrooli-emulator-scenario` (priority 2, effort L) — scaffolds `scenarios/vrooli-emulator/` (API + UI + CLI + service.json), moves livedesktop and procmetrics in as internal packages, switches route prefix to `/api/v1/sessions/`, adds the `headless: true` session flag, drops `GET artifact` in favor of caller-owned `app_path`, and ships the minimal operator CLI (`session list/create/destroy/exec/logs`, `metrics tail`). Depends on this research item.
- `execute/emulator-acceptance-tests-phase-1` (priority 2, effort M) — integration tests against the emulator in isolation (VNC + headless sessions, launch from `app_path`, screenshot, metrics, destroy) plus a CLI smoke test. Depends on `execute/scaffold-vrooli-emulator-scenario`.
- `chore/scenario-to-desktop-remove-livedesktop` (priority 2, effort S) — the hard-cut removal of `api/livedesktop/`, `api/procmetrics/`, `registerLiveDesktopRoutes()`, and leftover UI/CLI wiring from scenario-to-desktop. Leaves smoketest's xvfb-run path in place for phase 3. Depends on the scaffold + phase 1 tests.

### Phase 2 — UI + iframe embed (restores the desktop viewer)

- `execute/vrooli-emulator-standalone-ui` (priority 2, effort L) — moves the livedesktop UI components and API client into the emulator UI, builds the standalone management page, integrates `@vrooli/iframe-bridge` as the child side. Depends on the scaffold.
- `execute/vrooli-emulator-external-url-endpoint` (priority 2, effort S) — `GET /embedded/emulator/external-url` with optional `session_id` deep-link param, mirroring the deployment-manager pattern. Depends on the standalone UI.
- `execute/scenario-to-desktop-emulator-iframe-embed` (priority 2, effort M) — restores the desktop-viewer in scenario-to-desktop via iframe embed with parent-side iframe-bridge. Closes the hard-cut downtime window. Depends on the external-url endpoint and the livedesktop removal chore.

### Phase 3 — smoketest migration

- `execute/smoketest-delegate-display-to-emulator` (priority 3, effort M) — replaces `RequiresHeadlessWrapper()` and `xvfb-run -a` with emulator client calls so smoketest becomes an emulator consumer. Depends on the scaffold and phase 1 tests.

### Cross-cutting

- `chore/vrooli-emulator-documentation` (priority 3, effort S) — documents the `/api/v1/sessions/` API, iframe embed protocol, and operator CLI surface. Depends on the scaffold and standalone UI.
- `research/vrooli-emulator-remote-backend-spike` (priority 5, effort M) — future-facing feasibility spike on macOS/iOS remote backends, explicitly not on the critical path. Includes a note to compare against the already-existing `execute/vrooli-emulator-remote-node-backend` plan.

All nine items were assigned to the `emulator-platform` initiative (REUSE — the initiative already exists) as required by the conclusion.

## Pre-existing Items Not Duplicated

Three older items in the `emulator-platform` / `trusted-node-bridge` space were left untouched, since the conclusion's actions use new names and target narrower scope:

- `execute/vrooli-emulator-linux-first` — older XL umbrella item; the newly-created `scaffold-vrooli-emulator-scenario` is a narrower phase-1 slice of the same work. The umbrella should probably be retired or reshaped by a human; doing so is beyond this processing run's scope.
- `execute/adopt-vrooli-emulator-in-deployment-flows` — older integration item covering deployment-manager consumption; complementary to the new phase-2 items, not duplicated by them.
- `execute/vrooli-emulator-remote-node-backend` — older implementation item under the `trusted-node-bridge` initiative; the new `research/vrooli-emulator-remote-backend-spike` is an upstream feasibility spike, not a duplicate. The spike's description references comparing to this existing plan.

## Files Affected

- Created 9 backlog item folders under `scenarios/swarm-manager/{execute,chore,research}/<name>/` with `spec.json` each.
- Reused existing initiative `emulator-platform` (no initiative changes).

## Deviations

None that change the substance. Two minor naming choices the conclusion left implicit:

1. Action 6's item name is `execute/scenario-to-desktop-emulator-iframe-embed` (derived from the title "Restore desktop viewer in scenario-to-desktop via emulator iframe embed"). The conclusion never referenced this item as a dependency, so the name was not pre-specified.
2. Action 7's item name is `execute/smoketest-delegate-display-to-emulator`, action 8 is `chore/vrooli-emulator-documentation`, action 9 is `research/vrooli-emulator-remote-backend-spike` — all derived from titles for the same reason.

Priority and effort mappings used: high→2, medium→3, low→5; small→S, medium→M, large→L. These follow the existing backlog conventions observed on sibling items under the same initiative.

## Verification

- [x] All 9 actions in `conclusion.md` executed
- [x] Follow-up backlog items created (batch-create reported "Created 9 backlog item(s)")
- [x] Initiative `emulator-platform` preserved on every item
- [x] Dependency graph accepted by validation + cycle check
- [x] `notes.md` written (this file)

## Follow-up

- A human should decide the fate of the older `execute/vrooli-emulator-linux-first` umbrella item now that it has been superseded by the narrower phase-1 items. Options: close as redundant, retarget at phase-2/3 integration, or keep as a coordination shell.
- Phase 1 and phase 2 should be scheduled close together to minimize the scenario-to-desktop desktop-viewer downtime window flagged as the hard-cut trade-off in the conclusion.
- The phase 1 acceptance-test item captures intent but deliberately leaves specific assertions/fixtures to the execute item's own plan; that refinement happens when the item is workshopped.
