# `prd_template_content`

> **Severity default:** WARNING · **Capability:** prd_contract (L2) · **Fix class:** manual

## What it means

A required section exists but its content fails the substance checks: it is empty, still carries generation placeholders, lacks the expected anchor content (e.g. Overview without a Purpose line), or a priority tier has no valid `- [ ] OT-…` checklist lines.

## How to fix it

Write the real content. The finding message names the section and the specific issue (`empty_section`, `missing_content`, or `invalid_checklist`). Placeholders like "[Outcome title]" must be replaced, not deleted.

## Provenance

Emitted by business-health's PRD contract checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
