# Decisions — Scenario to iOS

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-11 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-13 | Xcode is a probed host capability, not a Vrooli resource; the fulfilling macOS bridge node owns build and signing. | Apple licensing and host toolchain installation cannot be governed as a portable resource. | iOS discovery reports the exact Xcode/macOS gap, and bridge-node role is part of target identity. | Revisit when Apple permits a governable distribution mechanism for the required toolchain. |
| 2026-08-13 | Reuse the shared delivery spine and the hello-mobile fixture rather than creating iOS-specific orchestration contracts. | Android and iOS need the same source-bundle, target-inventory, journey-evidence, and verdict shape. | iOS adds only Apple toolchain and signing adapters while preserving provider-neutral matrix semantics. | Revisit when an Apple surface cannot be represented by the shared spine's exported seams. |
| 2026-08-13 | The iOS build and signing host is a bridge-node role. | Linux cannot run Xcode, simulator, or physical-device signing workflows. | A remote macOS node is selected by capability and trust; local Linux remains unsupported rather than unavailable for Apple-only work. | Revisit when a supported Apple runner is local or the bridge trust model changes. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
