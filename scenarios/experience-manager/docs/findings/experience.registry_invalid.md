# `experience.registry_invalid`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** manual

## What it means

The fleet capability registry does not conform to its schema or references an undeclared axis, evidence kind, state, facet, or duplicate capability.

## How to fix it

Repair the owning capability-registry document and rerun the experience phase. Status is derived from the registry and reconciler support; it must not be authored as a green flag.

## Provenance

Emitted by experience-manager's Go registry validator during the experience phase.
