# `business_orphaned_ref`

> **Severity default:** ERROR · **Capability:** requirements_registry (L1) · **Fix class:** manual

## What it means

A `children` or `depends_on` entry references a requirement ID that does not exist in the registry (children: ERROR; depends_on: WARNING).

## How to fix it

Either the ref is stale (delete or repoint it) or the referenced requirement was removed prematurely (restore it). Check git history to see which side moved.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
