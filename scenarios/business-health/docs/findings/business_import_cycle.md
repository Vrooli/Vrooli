# `business_import_cycle`

> **Severity default:** ERROR · **Capability:** requirements_registry (L1) · **Fix class:** manual

## What it means

Requirements form a cycle through `children`/`depends_on` references, so the dependency graph has no valid order.

## How to fix it

The finding lists the cycle path. Remove or redirect the reference that misstates the relationship — usually a child pointing back at an ancestor.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
