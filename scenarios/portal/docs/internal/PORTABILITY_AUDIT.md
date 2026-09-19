# Portal Cross-Platform Readiness Audit

## Last Updated

2026-09-07

## Target Tiers

- [x] Tier 1 local stack
- [ ] Tier 2 desktop release certification
- [ ] Tier 3 mobile release certification
- [ ] Tier 4 cloud/SaaS certification
- [ ] Tier 5 enterprise/appliance certification

## Current Posture

- Portal scenario dependencies are declared optional, including Device Control,
  Bridge, Web Console, and account surfaces.
- Context-capture storage resolves through `api-core/storage` and production
  database access uses `database.RoutedDB`.
- Native helper and package claims remain subject to the active plan's live
  Windows, macOS arm64, GNOME Wayland, and KDE Wayland prerequisites.

## Evidence

Portal storage validation passed at `L3` on
`20260907-185947-6613476c`. Portal UI and API focused tests also passed in this
execution. These are local portability checks, not platform release receipts.

## Remaining Changes

1. Run live native acceptance and signed package lifecycle checks on each
   required external host.
2. Keep optional-provider absence visible as an actionable unavailable state.
3. Reconcile the final support matrix only from immutable platform receipts.
