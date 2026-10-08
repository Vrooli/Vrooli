# `business_unproven_claim`

> **Severity default:** WARNING · **Capability:** evidence_traceability (L1) · **Fix class:** manual

## What it means

An operational target is checked `[x]` but no linked requirement has passing evidence — or a requirement's only evidence is an expired manual attestation. The contract claims more than the evidence supports.

## How to fix it

Either produce the evidence (run the comprehensive suite so sync earns the statuses; re-attest via `business-health manual-log`) or uncheck the OT / downgrade the status to what is actually proven. Honesty beats optics.

## Provenance

Emitted by business-health's evidence traceability checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
