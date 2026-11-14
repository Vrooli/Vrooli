# `business_invalid_status`

> **Severity default:** WARNING · **Capability:** requirements_registry (L2) · **Fix class:** auto (fixer pending)

## What it means

A requirement's `status` is outside the vocabulary {pending, planned, in_progress, complete, not_implemented}.

## How to fix it

Run `business-health fix <scenario> --apply` to normalize recognizable variants (e.g. `done` → `complete`). Unrecognizable statuses are left with the finding — pick the honest value by hand. Remember statuses are earned by evidence sync, not asserted.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
