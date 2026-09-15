# BAS Automation

Store automation workflows here. Keep it short:

- `cases/<operational-target>/<surface>/` mirrors operational targets (rename folders as needed).
- `flows/` contains multi-surface user flows.
- `actions/` hosts fixtures referenced via `@fixture/<slug>`.
- `seeds/` includes optional setup scripts when deterministic data is required.

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
Use the Test Genie registry command documented by `test-genie registry --help` to regenerate this file; the exact subcommand is owned by Test Genie and changes independently of RCL.
```

This regenerates `bas/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on.
