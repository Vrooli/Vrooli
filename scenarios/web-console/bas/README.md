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
test-genie registry build
```

This regenerates `bas/registry.json`, which is tracked so other agents can see which files exist, which requirements they validate, and what fixtures they depend on.

## Desktop evidence authoring rules

- A case that needs a terminal session MUST compose `actions/open-workspace.json`
  followed by `actions/ensure-session.json`. The shared action owns the empty
  workspace transition; individual cases must not duplicate session probing.
- A case that depends on a layout MUST select the product's display-mode control
  through the settings surface, assert the workspace `data-display-mode` value,
  and restore the prior mode during teardown.
- Prefer `[data-testid="terminal-pane"]` for pane assertions. The
  `terminal-pane-container` selector is grid-specific and is not a stable
  assertion across tabs and sidebar modes.
- Do not inline session-probing JavaScript expressions in a case. Put reusable
  behavior in an action or selector so it has one owner and one repair point.
- Provider-owned desktop evidence proves only the selected case against the
  selected packaged artifact, target, and profile. It does not by itself prove
  every Web Console capability, offline support, signing, or release authority.
- BAS does not provide an implicit full reset between cases. Cases must establish
  their own preconditions and use `capture-session-baseline` plus
  `teardown-test-sessions` when they create sessions.
