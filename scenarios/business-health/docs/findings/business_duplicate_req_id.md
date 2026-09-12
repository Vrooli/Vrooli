# `business_duplicate_req_id`

> **Severity default:** ERROR · **Capability:** requirements_registry (L1) · **Fix class:** manual

## What it means

Multiple requirements share the same ID (case-insensitive). IDs anchor evidence history, prd_refs, and [REQ:ID] test tags, so a duplicate makes every reference ambiguous.

## How to fix it

Decide which requirement keeps the ID (usually the one with evidence history), assign the other a new unique ID, and update any `[REQ:ID]` test tags and children/depends_on refs that pointed at the renamed one.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
