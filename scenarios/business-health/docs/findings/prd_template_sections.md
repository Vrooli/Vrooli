# `prd_template_sections`

> **Severity default:** ERROR · **Capability:** prd_contract (L1) · **Fix class:** auto (fixer pending)

## What it means

A required canonical-template section (or a required priority-tier subsection under Operational Targets) is missing from `PRD.md`. Tooling that parses the PRD — linkage checks, the wizard, search indexing — depends on the canonical shape.

## How to fix it

Run `business-health fix <scenario> --apply` to scaffold the missing section(s) with TODO-marked bodies, then replace the TODO content with real product intent. Preview first (fix without `--apply` is a dry-run diff).

## Provenance

Emitted by business-health's PRD contract checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
