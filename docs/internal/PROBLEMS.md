# Documentation Problems

This file tracks project-level documentation debt that still needs follow-up after the first-pass rewrite.

## Current Known Problems

- Many `docs/` files still reflect shell-era, Python-era, or other transitional project-level workflows.
- Some strategic docs are conceptually useful but operationally stale.
- Several older docs describe commands or deployment surfaces that are no longer canonical.
- Project-level docs and subsystem docs still overlap in ways that create duplication and drift.
- Not all canonical project docs are yet organized under the new `concepts/`, `guides/`, and `reference/` taxonomy.

## Highest-Priority Second Pass

- continue curating `docs/scenarios/` and `docs/resources/` so they stay small, canonical, and cross-cutting
- decide which compatibility wrappers can be removed after inbound links are updated
- decide whether any remaining archive material should be deleted rather than preserved
- keep `docs/manifest.json` aligned with the actual canonical tree
