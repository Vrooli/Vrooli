# `prd_template_unexpected_sections`

> **Severity default:** INFO · **Capability:** prd_contract (L2) · **Fix class:** manual

## What it means

`PRD.md` contains a top-level heading the canonical template does not define. It may be a misnamed canonical section or intentional scenario-specific content.

## How to fix it

If it is a misspelled/renamed canonical section, rename it to the exact canonical heading. If it is genuinely scenario-specific material, consider moving it under `## 📎 Appendix` (the designated free-form section) so parsers keep a closed vocabulary.

## Provenance

Emitted by business-health's PRD contract checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
