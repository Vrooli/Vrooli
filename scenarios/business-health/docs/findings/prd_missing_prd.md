# `prd_missing_prd`

> **Severity default:** ERROR · **Capability:** prd_contract (L0) · **Fix class:** manual

## What it means

The scenario has no readable `PRD.md` at its root. Without it the scenario states no product intent: no operational targets exist for requirements to link to, and every downstream intent check is unevaluable.

## How to fix it

Author a PRD. Run `business-health wizard <scenario>` for an interview-driven scaffold (or `--answers file.json` non-interactively); the output conforms to the canonical template by construction. Authoring the actual intent — users, value promise, targets — is the calling agent's or operator's judgment.

## Provenance

Emitted by business-health's PRD contract checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
