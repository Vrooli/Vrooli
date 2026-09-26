# `intent.prd_ref_unmatched`

> **Severity default:** WARNING · **Capability:** intent_linkage (L1) · **Fix class:** manual

## What it means

A requirement's `prd_ref` names an `OT-…` ID that does not exist in `PRD.md`. The requirement claims to serve an outcome the PRD never states.

## How to fix it

Decide which side is wrong: if the outcome is real but undeclared, add the OT line to the PRD (product decision); if the ref is stale, repoint it at the correct OT. Pure ID-format drift is covered by `prd_ot_id_format` and its fixer instead.

## Provenance

Emitted by business-health's intent linkage checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
