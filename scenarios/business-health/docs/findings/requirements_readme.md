# `requirements_readme`

> **Severity default:** WARNING · **Capability:** requirements_registry (L2) · **Fix class:** auto (fixer pending)

## What it means

`requirements/README.md` is missing or no longer explains the registry contract (operational-target linkage, auto-sync, validation refs).

## How to fix it

Run `business-health fix <scenario> --apply` to restore the canonical README, then re-add any scenario-specific notes it carried.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
