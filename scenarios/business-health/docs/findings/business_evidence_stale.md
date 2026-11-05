# `business_evidence_stale`

> **Severity default:** WARNING · **Capability:** evidence_traceability (L2) · **Fix class:** manual

## What it means

The requirements-sync snapshot is older than the scenario's latest comprehensive suite run (or missing while runs exist), so statuses reflect an outdated reality.

## How to fix it

Run the full suite (`vrooli scenario test <scenario>`); sync fires inside suite execution and refreshes the snapshot. business-health never writes evidence itself — this finding is advisory by design.

## Provenance

Emitted by business-health's evidence traceability checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
