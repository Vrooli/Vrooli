# Documentation Problems

This file tracks project-level documentation debt that still needs follow-up after the first-pass rewrite.

## Current Known Problems

- Many `docs/` files still reflect shell-era, Python-era, or other transitional project-level workflows.
- Some strategic docs are conceptually useful but operationally stale.
- Several older docs describe commands or deployment surfaces that are no longer canonical.
- Project-level docs and subsystem docs still overlap in ways that create duplication and drift.
- Not all canonical project docs are yet organized under the new `concepts/`, `guides/`, and `reference/` taxonomy.

## Highest-Priority Second Pass

- rewrite `docs/devops/README.md` and associated devops pages to match current supported flows
- rewrite `docs/roadmap.md`, `docs/risks.md`, and `docs/decisions.md` for current strategic truth
- audit `docs/scenarios/` and `docs/resources/` for the same taxonomy and stale examples
- mark historical plans and historical docs more explicitly where they are still useful but no longer canonical
