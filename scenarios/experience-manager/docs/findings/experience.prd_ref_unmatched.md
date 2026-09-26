# `experience.prd_ref_unmatched`

> **Severity default:** ERROR · **Capability:** intent_linkage · **Fix class:** manual

## What it means

An experience page or claim points at a PRD operational target that cannot be found.

## How to fix it

Update the `prd_ref` to an existing OT id or add the missing operational target to the scenario PRD when the UX claim represents real intended product value.

## Provenance

Emitted by experience-manager's intent-linkage checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
