# Seams — Device Studio

## Wire contracts live in proto, not seams

Public messages/services belong in the proto owner. The following interfaces are planned substitution points for real implementations, not hand-maintained wire mirrors.

## Workflow transitions are not seams

[FLOWS.md](../concepts/FLOWS.md) owns transition rules. A fake process/provider can trigger a timeout, but must not own the domain decision that follows.

## Current seams

The scaffold contains its worked notes/database/clock/server seams. Product seams below are designed and not implemented.

| Seam | Planned owner | Substitute | Essential adversarial cases |
|---|---|---|---|
| Artifact source/store | projects and shared storage adapter | Immutable small fixture bytes | Missing/corrupt digest, external URI, archive escape |
| Media decoder/clock | presentation adapter | Timestamped test raster/stream | VFR, rotated media, end-of-range, stale seek |
| Compiler | shots | Concrete domain service with fixture libraries | Unsupported motion, mask, support, tool/profile |
| Renderer process | jobs | Owned worker fake/replay | Exit failure, disk full, lost heartbeat, cancellation descendants |
| Provider/agent execution | creations adapter | Recorded typed receipts | Unknown model/cost, scope expansion, child budget, uncertain acknowledgment |
| Commerce/credential client | integration adapter | Owner contract fixtures | Wrong account/binding, expired/revoked lease, insufficient credit, replay |
| Qualification/publication | reviews/Asset Studio adapter | Exact-revision receipt fake | Approved old revision, missing rights, publication unavailable |

## Adding a new seam

Define the domain need first. Reuse shared packages for generic behavior. Co-locate fakes with the domain; only cross-domain test harnesses belong in common test utilities. Preserve provider error and unavailable semantics rather than returning success-shaped zero values.

## Architecture Alignment Notes

Data ownership is in [DOMAINS.md](../concepts/DOMAINS.md). Scene/motion libraries are pinned input readers, not clients of the job owner. Reviews reference job artifacts; jobs do not mutate past reviews.

## UI-side seams

Planned browser stage adapter, file-picker bridge, async job subscription, account adapter and locale/accessibility hooks. Test stale-result suppression, numeric alternatives to dragging and draft persistence. No fabricated selector binding is added to experience until the control exists.

## What is NOT a seam

A private wallet, private host installer, copied provider router or second generic asset release ledger is an ownership defect, not a test seam.

## API contract manifest

The scaffold endpoint manifest describes actual reference services only. Planned product operations live in [api-endpoints.md](../reference/api-endpoints.md) until proto/handlers/CLI bindings are implemented and generated.

## Cross-references

[ARCHITECTURE.md](../concepts/ARCHITECTURE.md), [TESTING.md](TESTING.md), [PROBLEMS.md](PROBLEMS.md).
