# BAS Playbook Registry

BAS workflows live under `bas/` and are hierarchical: **actions** are the smallest reusable steps, **flows** compose actions into journeys, and **cases** assert requirements by calling flows. The fastest path is to validate actions first, then flows, then cases.

## Directory Layout
- `actions/` - reusable fixtures (no assertions, must set `metadata.fixture_id`)
- `flows/` - multi-step journeys that compose actions
- `cases/` - requirement-focused validations
  - `01-foundation/` project CRUD, workflow lists, version history
  - `02-builder/` canvas, palette, toolbar, header, demo seeds
  - `03-execution/` telemetry + control UIs and automation fixtures
  - `04-ai/` AI modal + generation smoke flows
  - `05-replay/` replay UI and export automation
  - `06-nonfunctional/` performance/error-handling checks
- `seeds/` - deterministic seed entrypoint (seed.go or seed.sh) supporting flows

## Setup (for a scenario that does not yet have `bas/`)
1. Create `bas/actions`, `bas/flows`, `bas/cases`, and `bas/seeds`.
2. Add at least one action and one flow (see scaffold below).
3. Generate the registry: `test-genie registry build` (run from the scenario directory).

## Authoring Workflow (recommended order)
1. **Actions**: atomic, reusable steps; no assertions.
2. **Flows**: compose actions into journeys; still no assertions.
3. **Cases**: call flows and assert requirements.

## Authoring Checklist
1. Pick the correct case folder + surface (e.g., `02-builder/toolbar`). Add a two-digit prefix for new top-level folders so order is explicit.
2. File naming: `feature-action.json` (kebab-case, include verb).
3. Add metadata with `description`, `version`, `reset` (optionally `name`). Requirement linkage lives in `requirements/*.json` via `validation.ref`; do **not** set `metadata.requirement`.
4. Use `"reset": "full"` if the workflow mutates state; `"reset": "none"` for read-only validation.
5. Use `settings.entrySelector` (and optional `settings.entrySelectorTimeoutMs`) so the runner can fail fast when the page never stabilizes.
6. Reference selectors via `@selector/<key>` from `ui/src/constants/selectors.ts`. Add new `data-testid` entries there (never hard-code CSS).
7. Reuse fixtures by setting `"workflowPath": "actions/<slug>.json"` with optional `"params"` instead of duplicating steps.
8. If a fixture guarantees requirement coverage, list those requirement IDs under `metadata.requirements` so coverage propagates.
9. Regenerate `registry.json` after edits and confirm ordering with `browser-automation-studio playbooks order`.

## CLI Helpers
- `browser-automation-studio playbooks scaffold <folder> <name>` creates a stub workflow under `bas/`.
- `browser-automation-studio playbooks verify` flags folders missing two-digit prefixes.

## Subflow Reference
```json
{
  "type": "subflow",
  "data": {
    "workflowPath": "actions/open-demo-project.json",
    "params": {
      "projectId": "${@params/projectId}",
      "projectName": "${@params/projectName}"
    }
  }
}
```

## Variables
- `${@store/key}` mutable runtime state (set via `storeResult`)
- `${@params/key}` input parameters to the workflow or subflow
- `${@env/key}` environment or project configuration
- Fallbacks: `${@params/x|@env/x|"default"}` tries in order

## Run + Debug
Run actions/flows/cases directly with the BAS CLI (test-genie only orchestrates them as part of scenario testing):
```bash
browser-automation-studio execution create \
  --file bas/cases/01-foundation/01-projects/new-project-create.json \
  --wait
```
Use the BAS UI execution viewer for logs, screenshots, and timeline replay when a run fails.

## Notes
- No compatibility shims: move files directly into this layout even if it causes short-term failures.
- Upcoming automation ideas in requirements should point to `bas/flows/...` using the final desired path.
- For deeper BAS authoring guidance: `scenarios/test-genie/docs/phases/playbooks/ui-automation-with-bas.md`.
