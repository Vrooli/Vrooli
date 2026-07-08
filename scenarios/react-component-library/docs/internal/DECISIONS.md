# Decisions — React Component Library

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-05-12 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-07-08 | Treat SQLite as an additive projection of Git-tracked component files. | Slot, headers, dependency declarations, preview import maps, and style affinities all derive from `component.json` and version source headers. | Schema changes use one-shot migrations for existing dev DBs; re-index rebuilds projection rows without recreating the database. | Revisit only if the component library stops using Git-tracked files as source of truth. |
| 2026-07-08 | Use `templates/design/*/metadata.json` IDs as the canonical design-style vocabulary. | Component affinity declarations need a stable key that scenario generation and UX validation can share. | `component.json designStyles[]` rejects unknown style IDs, `components styles` exposes the registry, and search/UI/adoption workflows treat affinities as advisory signals. | Revisit when design styles move out of templates or become versioned cross-scenario resources. |
| 2026-07-08 | Store dependency declarations by component version and kind. | Adopters can pin older component versions, and peer dependencies have different compatibility semantics than runtime/dev dependencies. | `component_dep_declarations` is keyed by `(component_id, version, dep_name)` and carries `kind`; adoption validation resolves against the requested adopted version. | Revisit if component artifacts gain a package-lock-style dependency manifest. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
