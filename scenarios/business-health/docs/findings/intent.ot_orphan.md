# `intent.ot_orphan`

> **Severity default:** WARNING · **Capability:** intent_linkage (L1) · **Fix class:** auto (fixer pending)

## What it means

An operational target has no requirement pointing at it via `prd_ref`. The PRD promises an outcome nothing claims to implement or validate. P0/P1 orphans matter most; the severity reflects the tier.

## How to fix it

Run `business-health fix <scenario> --apply` to scaffold a stub requirement carrying the `prd_ref` (status `planned`, no fake validation), then flesh the stub into real falsifiable claims. Or, if the outcome is dead, remove the OT line — a product decision.

## Provenance

Emitted by business-health's intent linkage checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
