# Flows — Secrets Manager

## Purpose Of This Document

This document records flows whose ordering or resource authority matters.

## Flow Inventory

| Flow | Trigger | Outcome |
|---|---|---|
| Vault validation | API or CLI request | Coverage metadata and missing-secret posture |
| Deployment manifest | deployment consumer request | Tier-specific strategies without secret values |
| Override mutation | authorized operator request | Effective strategy recomputation |
| Desktop launch | bundled lifecycle start | Private metadata storage and an authorized resource binding |

## Flow Details

Vault validation resolves the resource boundary, reads coverage metadata, and records the result. Deployment manifest generation resolves resources and strategies before emitting a bundle-safe result.

## State Machines

Validation and scan runs move from requested to running to completed or failed. A failed external resource check must preserve an actionable reason; it must not be rewritten as success.

## Maturity Ladder

Current flows have handler and persistence coverage. Brokered shared-resource reuse and bundle admission have dedicated runtime tests; full live Tier 1 evidence depends on host Secret Service availability.

## Production Shape

The control plane selects a shared Vault only under its authority. Desktop bundles select private Vault by default and require explicit consent for shared reuse.

## Deferred / Unmodeled Flows

Scheduled stale-manifest refresh and automated remediation require future product decisions.

## Cross-References

- [Architecture](ARCHITECTURE.md)
- [Data](DATA.md)
- [Integrations](INTEGRATIONS.md)
