# BAS Playbook Registry

Automation workflows live here for both **cases** (assertive tests) and **flows** (multi-surface journeys).

## Directory Layout
- `cases/` – mirrors PRD operational targets
  - `foundation/` – project CRUD, workflow lists, version history
  - `builder/` – canvas, palette, toolbar, header, demo seeds
  - `execution/` – telemetry + control UIs and automation fixtures
  - `ai/` – AI modal + generation smoke flows
  - `replay/` – replay UI and export automation
  - `nonfunctional/` – performance/error-handling checks
- `flows/` – composite user journeys (e.g., happy-path new user)
- `actions/` – reusable fixtures (`metadata.fixture_id` required; no assertions)
- `seeds/` – deterministic seed entrypoint (seed.go or seed.sh) supporting flows

Numbers in the requirements folders determine **what** each target means; this layout explains **where** to place the validations. When adding a new top-level case/flow folder, prefix the segment with a two-digit ordinal (e.g. `01-foundation`, `02-builder`) so execution order is obvious.

## Authoring Checklist
1. Pick the correct case folder + surface (e.g., `builder/toolbar`). If the folder doesn’t exist yet, create it and update this README.
2. File naming: `feature-action.json` (kebab-case, include verb).
3. Add metadata with `description`, `version`, and `reset` (optionally `name`). Requirement linkage now lives in `requirements/*.json` via `validation.ref` entries; do **not** set `metadata.requirement`. Use `"reset"` to tell the runner when to reseed:
   - `"reset": "full"` when the workflow mutates state and needs a fresh seed on the next test
   - `"reset": "none"` when the workflow is read-only and can share state with the previous test
4. For faster failure detection, you can set `settings.entrySelector` (and optional `settings.entrySelectorTimeoutMs`) to declare the first element that proves the page is ready. The harness will probe this selector before running steps; otherwise it falls back to the first selector it finds.
4. Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts` (never hard-code CSS). Add new selector keys there when needed.
5. Reuse fixtures by setting `"workflowPath": "actions/<slug>.json"` with optional `"params"` object rather than duplicating setup steps. Use `${@store/key}` for runtime values stored earlier in the workflow, or `${@params/key}` for input parameters passed to the workflow.
6. If a fixture guarantees requirement coverage (e.g., demo workflow seeding), list those requirement IDs under `metadata.requirements`. The resolver propagates them to parent workflows via `metadata.requirementsFromFixtures` so coverage reports know which requirements the run exercised.
7. Update the matching requirement module to reference the new playbook path; auto-sync will keep statuses fresh when tests run.
8. Regenerate `registry.json` after any edits and verify the `order` field lists your workflow where you expect it.
9. Use the CLI helpers to stay consistent:
   - `browser-automation-studio playbooks scaffold <folder> <name>` creates a stub workflow JSON with the correct metadata/reset fields under `bas/`
   - `browser-automation-studio playbooks verify` flags folders that are missing the two-digit prefixes required for deterministic ordering


### Subflow Reference

Subflows (reusable action workflows) are called using `workflowPath` with optional `params`:

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

### Variable Namespaces

BAS supports three namespaces for variable interpolation:

- `${@store/key}` – Mutable runtime state, writable via `storeResult` in evaluate steps
- `${@params/key}` – Read-only input parameters passed to the workflow or subflow
- `${@env/key}` – Read-only environment/project configuration

**Fallback chains** are supported: `${@params/x|@env/x|"default"}` tries each option in order.

**Type preservation**: Single-interpolation values like `"${@params/count}"` preserve the original type (number, boolean, etc.). Mixed content becomes a string.

### Fixture Metadata Reference

- `fixture_id` – slug identifying the fixture (used in legacy references)
- `parameters` – optional array describing accepted arguments. Each entry supports `name`, `type` (`string|number|boolean|enum`), `required`, `default`, `enumValues`, and `description`.
- `requirements` – optional array of requirement IDs the fixture covers during execution. These IDs propagate to every playbook that calls the fixture.

## Running Locally
```
browser-automation-studio execution create \
  --file bas/cases/01-foundation/01-projects/new-project-create.json \
  --wait
```

## Notes
- Keep this README under 100 lines; link to shared docs for deep dives (`docs/testing/guides/ui-automation-with-bas.md`).
- No compatibility shims: move files directly into this layout even if it causes short-term failures.
- Upcoming automation ideas referenced in requirements should point to `bas/flows/...` using the final desired path.
- Regenerate the registry after edits: `test-genie registry build` (run from scenario directory). The registry records each workflow's `order`, `reset` behavior, fixtures, and covered requirement IDs so the runner can execute them deterministically. Use `browser-automation-studio playbooks order` to quickly inspect the rendered story without opening `registry.json` manually.
