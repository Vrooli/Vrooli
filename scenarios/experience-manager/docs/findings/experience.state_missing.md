# `experience.state_missing`

> **Severity default:** WARNING · **Capability:** state_contract · **Fix class:** manual

## What it means

A UX state required by `DESIGN.md` is missing from the page spec.

## How to fix it

Declare the state on the affected page or update `DESIGN.md` if the required state contract has changed.

## Provenance

Emitted by experience-manager's state-contract checks (dimension `experience`, advisory cap at ERROR). Missing `DESIGN.md` seeds this as advisory info rather than an error.
