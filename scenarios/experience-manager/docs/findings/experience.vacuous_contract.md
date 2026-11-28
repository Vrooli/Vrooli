# `experience.vacuous_contract`

> **Severity default:** ERROR · **Capability:** spec_contract · **Fix class:** authoring

## What it means

A portable component contract contains no claim backed by an implemented evaluator. A placeholder such as `contract-present` records existence, but it does not prove behavior.

## How to fix it

Add at least one claim using a supported evaluator, or leave the contract explicitly scaffolded until the component has a checkable behavior.

## Provenance

Emitted by experience-manager when parsing `library/**/experience-contract.json` files.
