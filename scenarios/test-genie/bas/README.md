# Scenario Playbooks

Store BAS workflows here. Keep it short.

## Separation Of Concerns (Important)

- **`bas/cases/**`**: the Playbooks phase test cases that test-genie executes.
- **Everything else under `bas/`** (`actions/`, `flows/`, etc.): reusable workflows/subflows that cases can reference, and that you can run outside test-genie for automations.

test-genie does **not** rewrite workflows. BAS resolves scenario navigation, tokens, and subflow paths at runtime. test-genie’s job is orchestration: provision isolated DB/Redis, run seeds once, restart the scenario against those resources, then execute the `bas/cases/**` workflows.

## Workflow Contract

Workflows should be runnable by BAS as-is. The Playbooks phase provides BAS:

- `project_root`: absolute path to the scenario’s `bas/` directory (for resolving `workflowPath` relative to `bas/`)
- `initial_params`: the contents of `coverage/runtime/seed-state.json` (seeded data)

Each case workflow should include `metadata.description` for humans and for registry generation:

```json
{
  "metadata": {
    "description": "What the workflow validates",
    "version": 1
  }
}
```

After adding or moving a case under `bas/cases/`, regenerate the registry from the scenario directory:

```bash
test-genie registry build
```

This regenerates `bas/registry.json` (tracked), which test-genie uses to determine which `bas/cases/**` files to execute.

## Playbooks isolation quickstart

- Playbooks automatically start the scenario against temporary Postgres/Redis for this phase. Seeds run once and write `coverage/runtime/seed-state.json`.
- Retain for debugging: `TEST_GENIE_PLAYBOOKS_RETAIN=1 test-genie execute my-scenario --preset comprehensive`
  - Observations will include ready-to-run `psql`/`redis-cli` commands to inspect the retained DB/Redis.
- Normal runs drop the temp resources and restart the scenario on its usual resources after Playbooks finishes.

See [Directory Structure](../../docs/phases/playbooks/directory-structure.md) for complete documentation on playbooks layout, fixtures, and naming conventions.
