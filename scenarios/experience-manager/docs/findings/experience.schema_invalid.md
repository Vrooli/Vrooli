# `experience.schema_invalid`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** manual

## What it means

An `experience/` document does not conform to `scenario-experience-spec/v1`.

## How to fix it

Validate the document shape against `.vrooli/schemas/scenario-experience-spec.schema.json`, then repair the invalid fields without changing the declared UX intent.

## Provenance

Emitted by experience-manager's spec parser checks (dimension `experience`, advisory cap at ERROR). Mapped by the experience phase maturity descriptor once the provider phase lands.
