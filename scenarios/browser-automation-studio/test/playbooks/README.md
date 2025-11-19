# BAS Playbook Registry

Automation workflows live here for both **capabilities** (feature surfaces) and **journeys** (multi-surface flows).

## Directory Layout
- `capabilities/` – mirrors PRD operational targets
  - `foundation/` – project CRUD, workflow lists, version history
  - `builder/` – canvas, palette, toolbar, header, demo seeds
  - `execution/` – telemetry + control UIs and automation fixtures
  - `ai/` – AI modal + generation smoke flows
  - `replay/` – replay UI and export automation
  - `nonfunctional/` – performance/error-handling checks
- `journeys/` – composite flows (e.g., happy-path new user)
- `__subflows/` – reusable fixtures (`metadata.fixture_id` required)
- `__seeds/` – deterministic seed/cleanup scripts supporting flows

Numbers in the requirements folders determine **what** each target means; this layout explains **where** to place the validations.

## Authoring Checklist
1. Pick the correct capability folder + surface (e.g., `builder/toolbar`). If the folder doesn’t exist yet, create it and update this README.
2. File naming: `feature-action.json` (kebab-case, include verb).
3. Add metadata with `description`, `requirement` ID, and `version`.
4. Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts` (never hard-code CSS). Add new selector keys there when needed.
5. Reuse fixtures by setting `"workflowId": "@fixture/<slug>"` rather than duplicating setup steps.
6. Update the matching requirement module to reference the new playbook path; auto-sync will keep statuses fresh when tests run.

## Running Locally
```
browser-automation-studio execution create \
  --file test/playbooks/capabilities/foundation/projects/new-project-create.json \
  --wait
```

## Notes
- Keep this README under 100 lines; link to shared docs for deep dives (`docs/testing/guides/ui-automation-with-bas.md`).
- No compatibility shims: move files directly into this layout even if it causes short-term failures.
- Upcoming automation ideas referenced in requirements should point to `test/playbooks/journeys/...` using the final desired path.
