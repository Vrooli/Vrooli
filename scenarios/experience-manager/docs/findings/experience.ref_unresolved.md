# `experience.ref_unresolved`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** manual

## What it means

A claim, sketch, binding, state, or journey step references an experience object that does not exist.

## How to fix it

Correct the reference target or add the missing declared object. Do not silently delete the claim unless the UX intent is no longer true.

## Provenance

Emitted by experience-manager's spec parser checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
