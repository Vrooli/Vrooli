# Deployment — Plan Manager

This document records supported delivery tiers, packaging assumptions,
runtime dependencies, and deployment readiness for plan-manager.

## Purpose Of This Document

Use this document to answer:

- Where does plan-manager run, and on which tier?
- What does it need at runtime?
- How is it packaged and released?
- How is a bad release rolled back?

## Supported Tiers

- Local developer / agent runtime: the primary and only supported tier.
  plan-manager runs like any other Vrooli scenario on a developer's or
  agent's machine via the standard scenario lifecycle.
- Hosted / cloud tier: not applicable in v1. There is no special cloud
  deployment because velocity and handoff data are local data and the
  scenario is consumed in-ecosystem. A hosted tier is deferred until an
  external-demand trigger appears (see the business docs).

## Runtime Requirements

- Go binaries: the api server and the `vrooli plans` CLI.
- UI: a React + Vite + Tailwind bundle served by the scenario.
- Transport: Connect-RPC over proto.
- Storage: SQLite via api-core/storage, rooted at the scenario-independent
  `~/.vrooli` home store. This is deliberately NOT a scenario-private DB,
  so plans remain readable (via the CLI thin client) when the server is
  down.
- Resources: no heavy resources (no Postgres, Redis, Qdrant, Vault, etc.).
  plan-manager's integrations (code-facts, git-control-tower, test-genie /
  scenario-validation, prompt-manager, meta-optimization-manager,
  agent-manager) are all soft and degrade gracefully when absent.

## Packaging

- Packaged as a standard Vrooli scenario: Go binaries plus the built Vite
  UI bundle, started through `make start` / `vrooli scenario start
  plan-manager`. No container image or external artifact registry is
  required for the local tier.
- Per-binary distribution outside the Vrooli ecosystem is deferred — not
  applicable while plan-manager is internal-only.

## Release Checklist

This scenario is pre-implementation (documentation-first), so the
checklist below is the intended shape and will be tightened once the first
vertical slice exists:

- Build Go binaries and the Vite UI bundle cleanly.
- Run the scenario test suite via `vrooli scenario test plan-manager` and
  confirm it is green.
- Confirm the SQLite store at `~/.vrooli` migrates forward without data
  loss and that `vrooli plans` can still read existing plans.
- Confirm soft integrations degrade gracefully when their backing
  scenarios are unavailable.
- Confirm the standard scenario health endpoint reports healthy after
  start.

## Rollback

- Process rollback: stop the scenario (`vrooli scenario stop plan-manager`
  / `make stop`) and restart the previous known-good binary set through the
  standard lifecycle.
- Data rollback: plan data lives in the shared `~/.vrooli` SQLite store and
  is recoverable through Vrooli's standard home-store backup mechanisms;
  detailed restore steps are in [`RUNBOOK.md`](RUNBOOK.md). Because the
  store is shared and server-independent, a code rollback does not by
  itself discard plans.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operator procedures and restore steps
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — health and signals to check post-release
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
