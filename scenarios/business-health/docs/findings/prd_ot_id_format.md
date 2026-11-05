# `prd_ot_id_format`

> **Severity default:** WARNING · **Capability:** prd_contract (L2) · **Fix class:** auto (fixer pending)

## What it means

An operational-target line has an ID that deviates from the canonical `OT-P<tier>-NNN` format (wrong prefix, missing zero-padding, or wrong tier for its section). Non-canonical IDs silently break the prd_ref join.

## How to fix it

Run `business-health fix <scenario> --apply` to normalize IDs to canonical form. The fixer also rewrites matching `prd_ref` values in requirements/ so the join stays closed.

## Provenance

Emitted by business-health's PRD contract checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
