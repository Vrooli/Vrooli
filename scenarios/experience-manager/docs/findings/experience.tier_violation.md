# `experience.tier_violation`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** manual

## What it means

Claim tier semantics are invalid: custom checks are not machine-tier, or an unknown type was assigned a stricter tier than aspirational.

## How to fix it

Correct the tier/type pair. Unknown or custom claim types must stay manual or aspirational until a deterministic checker exists.

## Provenance

Emitted by experience-manager's spec parser checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
