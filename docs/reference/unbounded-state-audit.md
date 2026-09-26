# Unbounded-state audit

This document records the repository-level storage decisions for generated and
temporary state. The authoritative recovery implementation and provider
declarations are in [Storage Manager's unbounded-state audit](../../scenarios/storage-manager/docs/reference/unbounded-state-audit.md),
[root specification](../../scenarios/storage-manager/docs/reference/root-spec.md),
and [recovery ledger](../../scenarios/storage-manager/docs/reference/recovery-ledger.md).

## 2026-09-03 addendum

Storage Manager now separates retention from pressure recovery. Shared runtime
artifacts, scenario binaries, resource installations, databases, models, and
other durable state are not eligible for a generic age or size sweep. Recovery
may act autonomously only on a safe root or on a regenerable owner declaration
with an explicit budget; conditional actions require a host-local standing
approval. Each applied batch is recorded with bytes reclaimed, files removed,
duration, free-space readings, rung, and authority.

The host proof is deliberately not overstated. The safe candidates available
after the incident reclaimed 5.87 GB in the applied run, and the governed rate
chaos proof reclaimed 311 MB. The 250 GB / 50%-used acceptance gate therefore
remains unproven; no durable data was deleted to manufacture that number.

Two follow-up limits remain explicit:

- Multi-mount recovery targets need a separate target model and live proof.
- macOS and Windows watchdog and log-bound behavior has build-tagged coverage,
  but no live hosts were available for validation.

The 48-hour growth gate also remains open until the required observation window
includes ordinary development activity and a full test-genie run.
