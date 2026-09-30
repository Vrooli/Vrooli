# `business_manual_expired`

> **Severity default:** WARNING · **Capability:** evidence_traceability (L2) · **Fix class:** manual

## What it means

A manual validation's most recent attestation in `coverage/manual-validations/log.jsonl` is older than the expiry policy allows; an attended check ages out of trustworthiness.

## How to fix it

Re-perform the manual procedure and record it with `business-health manual-log <scenario> <req-id>`. If the procedure is now automatable, replace the manual validation with a test-typed one instead.

## Provenance

Emitted by business-health's evidence traceability checks (dimension `business`, advisory cap at ERROR — this finding never blocks a suite on its own). Mapped in `.vrooli/maturity.json`; severity and ladder impact live there, not here.
