# Deployment — Network Manager

## Purpose

This document describes deployment expectations for Network Manager.

## Supported Deployment Modes

| Mode | Status | Notes |
|---|---|---|
| Tier 1 local Vrooli stack | Planned P0 | Primary development and first real deployment target. |
| Desktop bundle | Deferred | Needs adapter capability review and resolver-resource packaging. |
| Small-office appliance | Deferred | Natural fit after audit mode and router adapter support. |
| Cloud/SaaS | Not primary | Network control is local-first; hosted dashboards may come later. |

## Runtime Dependencies

P0 requires the generated API/UI/CLI and an AdGuard Home resolver resource or externally managed AdGuard Home endpoint. Router writes are not required for P0.

## Deployment Steps

1. Install or connect AdGuard Home through the eventual governed resource path.
2. Start Network Manager with the Vrooli lifecycle.
3. Confirm API and UI health.
4. Connect AdGuard Home credentials/configuration.
5. Run a read-only network health snapshot.
6. Enable conservative filtering only after preview.

## Safety Gates

- Persistent DNS changes require approval.
- Router writes are unavailable in P0.
- Rollback instructions or handles must be shown for applied changes.
- Query-log visibility must be explicit.

## Known Gaps

- No AdGuard Home resource implementation has been added yet.
- Router platform adapter is deferred.
- Desktop packaging needs cross-platform host adapter review.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md)
- [`OBSERVABILITY.md`](OBSERVABILITY.md)
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)
