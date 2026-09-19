# `experience.binding_orphan`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** manual

## What it means

Declared elements and bindings are not closed in both directions, or a machine-tier claim references an unbound element.

## How to fix it

Add the missing binding, add the missing declared element, or retarget the claim to the correct bound element.

## Provenance

Emitted by experience-manager's spec parser checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
