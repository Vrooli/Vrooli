# Docs Audit

This file records the outcome of the project-level docs restructuring.

## Current State

The canonical project docs now center on:

- `docs/README.md`
- `docs/QUICKSTART.md`
- `docs/concepts/`
- `docs/guides/`
- `docs/reference/`
- `docs/operations/`
- `docs/deployment/`
- `docs/scenarios/`
- `docs/resources/`
- `docs/strategy/`
- `docs/plans/`

The old catch-all `docs/devops/` shape has been dissolved in practice. The remaining project-level `docs/devops/` material should be treated as migration leftovers unless promoted back into a canonical section.

## What Changed

- strategy docs moved under `docs/strategy/`
- active workflow docs moved under `docs/guides/`, `docs/operations/`, `docs/reference/`, and `docs/deployment/reference/`
- scenario-specific project docs were relocated into the owning scenarios
- historical wrappers and the temporary historical staging layer were removed once callers were updated

## Remaining Cleanup Themes

- continue auditing `docs/scenarios/` for leaves that should be canonical vs. reference
- continue auditing `docs/resources/` for historical or over-specialized leaves
- keep `docs/manifest.json` aligned with the actual canonical tree
- avoid reintroducing top-level compatibility wrappers unless a real caller requires one

## Guidance

- new project-level docs should land in the canonical sections above
- historical context should be folded into maintained docs or retained only when it still has active maintenance value
- scenario-specific and resource-specific design docs should live with the owning subsystem, not at project root
