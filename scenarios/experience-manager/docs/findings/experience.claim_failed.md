# `experience.claim_failed`

> **Severity default:** ERROR · **Capability:** structure_reconciliation · **Fix class:** manual

## What it means

A machine-tier experience claim was checked against captured structure and did not hold.

## How to fix it

Change the UI to satisfy the declared claim, or revise the claim if the spec no longer represents intended behavior.

## Provenance

Emitted by experience-manager's AX-tree reconciliation checks (dimension `experience`, advisory cap at ERROR). Active pages may gate on this when strict gating is enabled.
