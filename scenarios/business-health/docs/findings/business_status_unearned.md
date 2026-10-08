# `business_status_unearned`

> **Severity default:** ERROR · **Capability:** evidence_traceability (L1) · **Fix class:** manual

## What it means

A requirement status contradicts what the evidence sync writer last recorded — someone hand-edited progress that was never earned.

## How to fix it

Revert the status to the synced value, then earn the change: tag tests with `[REQ:ID]`, run the comprehensive suite, and let sync move it. Hand-editing sync-owned statuses is the one move this provider treats as dishonest rather than merely stale.

## Provenance

Emitted by business-health's evidence traceability checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
