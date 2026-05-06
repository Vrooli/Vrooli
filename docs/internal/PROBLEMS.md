# Documentation Problems

This file tracks the highest-value remaining documentation debt after the main project-level rewrite.

## Current Known Problems

- Some specialized leaf docs are still intentionally thin and may need deeper rewrites later.
- The docs will drift again unless command surfaces and canonical pages are updated together.
- Scenario-level and resource-level docs can still accumulate overlap with project-level canon if not curated.
- Plan documents remain numerous and need discipline so they do not get mistaken for current truth.

## Highest-Value Remaining Follow-Up

- continue curating specialized leaves under `path:docs/deployment/`, `path:docs/strategy/`, and other focused sections
- keep `docs/manifest.json` aligned with the actual canonical tree
- keep project-level command reference pages aligned with `go run ./cmd/vrooli ... --help`
- keep scenario and resource system docs small, cross-cutting, and clearly distinct from owning subsystem docs
