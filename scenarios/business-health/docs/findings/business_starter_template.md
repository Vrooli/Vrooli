# `business_starter_template`

> **Severity default:** WARNING · **Capability:** requirements_registry (L1) · **Fix class:** manual

## What it means

The registry still contains module(s) tagged `template-starter` — it describes the generated scaffold, not this scenario. A starter registry is a placeholder, not a claim.

## How to fix it

Replace starter modules with scenario-specific requirements: one falsifiable behavioral claim per requirement, each with a `prd_ref` to a real operational target. `business-health wizard` scaffolds one requirement stub per OT as a starting point.

## Provenance

Emitted by business-health's requirements registry checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
