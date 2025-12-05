# Scenario Playbooks

Store automation workflows here. Keep it short:

- `capabilities/<operational-target>/<surface>/` mirrors the PRD targets (rename folders as needed).
- `journeys/` contains multi-surface flows (new user onboarding, AI happy path, etc.).
- `__subflows/` hosts fixtures referenced via `@fixture/<slug>`.
- `__seeds/` includes optional setup/cleanup scripts when deterministic data is required.

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

This regenerates `test/playbooks/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on.

See [Directory Structure](../../docs/phases/playbooks/directory-structure.md) for complete documentation on playbooks layout, fixtures, and naming conventions.
