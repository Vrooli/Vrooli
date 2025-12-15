# Scenario Playbooks

Store automation workflows here. Keep it short:

- `capabilities/<operational-target>/<surface>/` mirrors the PRD targets (rename folders as needed).
- `journeys/` contains multi-surface flows (new user onboarding, AI happy path, etc.).
- `__subflows/` hosts fixtures referenced via `@fixture/<slug>`.
- `__seeds/` includes a seed entrypoint (`seed.go` preferred; `seed.sh` allowed) that writes `coverage/runtime/seed-state.json`.

Each workflow JSON must include:

```json
{
  "metadata": {
    "description": "What the workflow validates",
    "requirement": "REQ-ID",
    "version": 1
  }
}
```

Reference selectors via `@selector/<key>` from `ui/src/consts/selectors.ts`. After adding or moving a workflow, run from the scenario directory:

```bash
test-genie registry build
```

This regenerates `bas/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on.

## Playbooks isolation quickstart

- Playbooks automatically start the scenario against temporary Postgres/Redis for this phase. Seeds run once and write `coverage/runtime/seed-state.json`.
- Retain for debugging: `TEST_GENIE_PLAYBOOKS_RETAIN=1 test-genie execute my-scenario --preset comprehensive`
  - Observations will include ready-to-run `psql`/`redis-cli` commands to inspect the retained DB/Redis.
- Normal runs drop the temp resources and restart the scenario on its usual resources after Playbooks finishes.

See [Directory Structure](../../docs/phases/playbooks/directory-structure.md) for complete documentation on playbooks layout, fixtures, and naming conventions.
