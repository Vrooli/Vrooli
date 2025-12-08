# Requirements Directory (Vrooli Ascension)

This folder mirrors the PRD operational targets so agents can reason about **what** must be true before worrying about **how** to implement it.

## Structure
- `01-foundation/` – Project CRUD, workflow lists, and version history guardrails.
- `02-builder/` – Canvas, toolbar, palette, and builder-specific UI (header/composite flows).
- `03-execution/` – Telemetry, controls, CLI mirrors.
- `04-ai/` – AI generation and related UI flows.
- `05-replay/` – Replay persistence/export.
- `06-nonfunctional/` – Performance, accessibility, graceful degradation.

Numbers preserve ordering/insertion; they do **not** indicate priority. Each target folder may contain nested subfolders (`ui`, `workflow-builder`, etc.) to keep modules focused.

## Lifecycle
1. PRD defines operational targets → mapped one-to-one with the folders above.
2. `requirements/index.json` imports each module; auto-sync propagates validation status from tests to requirements to PRD targets.
3. Validation types (`unit`, `integration`, `automation`, `manual`, etc.) reference concrete test files or playbooks.
4. Coverage summaries are emitted by the requirements reporter (`vrooli scenario requirements report browser-automation-studio`) and stored in `coverage/phase-results/`.

## How to Work Here
- When adding/changing a feature, first decide which operational target owns it; add/update the module JSON inside that folder.
- Keep requirement IDs stable; if you must rename/move them, update every validation reference and rerun the requirements sync script.
- New modules should include `_metadata`, `requirements` array, and complete validation references (tests and playbooks).
- Do **not** add compatibility shims (e.g., duplicate modules or alias directories) during migrations; short-term breakage is preferable to lingering cruft.

## Validation & Tooling
- Run `make test` (or `vrooli test <phase>`) to execute phases and sync requirement statuses.
- `vrooli scenario requirements report browser-automation-studio` generates machine- and human-readable coverage snapshots.
- Testing helpers fail when a requirement references a missing playbook (or vice versa) so drift is caught quickly.

## References
- Shared standards: `docs/testing/requirements.md` (see repo docs) for schema/details.
- Scenario authoring guides: `docs/scenarios/scenario-generator.md`, `scenario-improver.md`.
- Selector + playbook linkage lives in `test/playbooks/README.md` (mirrors these operational targets).
