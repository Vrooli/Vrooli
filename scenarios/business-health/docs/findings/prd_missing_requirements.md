# `prd_missing_requirements`

> **Severity default:** ERROR · **Capability:** requirements_registry (L0) · **Fix class:** auto (fixer pending)

## What it means

The scenario has no parseable `requirements/index.json`. Operational targets have nowhere to link and no requirement claims exist.

## How to fix it

Run `business-health fix <scenario> --apply` to create a minimal valid registry (index + one module skeleton), or `business-health wizard <scenario>` to scaffold a registry derived from the PRD's operational targets.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
