# `experience.attestation_expired`

> **Severity default:** WARNING · **Capability:** manual_attestation · **Fix class:** manual

## What it means

A manual-tier claim depends on an attestation whose expiry has passed.

## How to fix it

Re-run the manual check, record a fresh attestation with an honest expiry, or downgrade/remove the claim if it is no longer maintained.

## Provenance

Emitted by experience-manager's manual attestation ledger checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
