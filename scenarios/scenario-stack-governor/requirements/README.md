# Requirements

This scenario uses the modular requirements registry:
- `requirements/index.json` imports per-operational-target modules.
- Each module has one or more requirement IDs that tests should eventually validate.

## Workflow
- Tag tests with `[REQ:<ID>]`.
- Run `make test` (or `vrooli scenario test scenario-stack-governor`) to execute phased tests once they exist.
- Use `vrooli scenario requirements report scenario-stack-governor` to view coverage.

