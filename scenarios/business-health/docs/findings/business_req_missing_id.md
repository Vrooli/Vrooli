# `business_req_missing_id`

> **Severity default:** ERROR · **Capability:** requirements_registry (L1) · **Fix class:** manual

## What it means

A requirement entry has an empty `id`. Without an ID it cannot be referenced, tagged in tests, or accrue evidence.

## How to fix it

Mint a unique, stable ID following the module's convention (e.g. `BH-VAL-007`). Never reuse a retired ID.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
