# `experience.claim_unverifiable`

> **Severity default:** WARNING · **Capability:** structure_reconciliation · **Fix class:** manual

## What it means

A machine-tier claim has no Tier 0-1 deterministic checker.

## How to fix it

Move the claim to manual or aspirational tier, or implement a deterministic checker before treating it as machine-verifiable.

## Provenance

Emitted by experience-manager's reconciliation checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
