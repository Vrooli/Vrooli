# Subscription Entitlements System

This document describes a scenario-specific subsystem spanning `landing-page-business-suite` and `browser-automation-studio`. It is not project-level architecture for Vrooli as a whole.

## Status

- scope: subsystem architecture
- authority: limited to the scenarios named above
- role: analysis and design reference

If the implementation has changed since the last detailed audit, the code for those scenarios is the source of truth.

## What This Document Is For

Use it when you need to understand:

- how subscription state, entitlements, usage, or limits are split across those scenarios
- known design gaps or model mismatches in that subsystem
- where future consolidation or cleanup work may be needed

## What This Document Is Not For

Do not use this file as:

- the canonical architecture guide for Vrooli
- a billing or commercial roadmap
- a claim that every implementation detail here is still current without code verification

For project-level architecture, start with:

- [../concepts/ARCHITECTURE.md](../concepts/ARCHITECTURE.md)
- [../context.md](../context.md)
- [../decisions.md](../decisions.md)

## Maintenance Guidance

If you update the underlying scenarios:

- keep this document tightly scoped to the subsystem
- prefer durable models and current gaps over exhaustive code dumps
- link to the relevant scenario docs or code paths when possible
- remove stale status claims that are no longer verified
