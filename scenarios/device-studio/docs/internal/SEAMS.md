# Seams — Device Studio

## Wire contracts live in proto, not seams

Public messages/services belong in the proto owner. The following interfaces are planned substitution points for real implementations, not hand-maintained wire mirrors.

## Workflow transitions are not seams

[FLOWS.md](../concepts/FLOWS.md) owns transition rules. A fake process/provider can trigger a timeout, but must not own the domain decision that follows.

## Current seams

Projects, source media, immutable still shots and the shared jobs/native-operation adapter are implemented for the accepted E1/E2 slices. Recording inspection is live-qualified on the current host. Video rendering integration is in progress; the remaining creation, commerce and asset qualification seams below are planned.

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

Proto, domain endpoint descriptors and the generated endpoint manifest own the implemented project/media/shot/render surface. [api-endpoints.md](../reference/api-endpoints.md) also retains planned operations; those are not implementation evidence.

## Cross-references

[ARCHITECTURE.md](../concepts/ARCHITECTURE.md), [TESTING.md](TESTING.md), [PROBLEMS.md](PROBLEMS.md).

## Video CLI transfer boundary

The render CLI uses generated Connect clients for job metadata, modes and trusted
sample bindings, and the existing cli-core HTTP client for the documented raw
content route. That client supplies API-base selection, authentication and routed
test headers. Exports stream at most the declared160MiB limit into a private
same-directory temporary file; byte count, SHA256 and format are checked before
publication. PNG additionally checks dimensions and complete decoding. MP4 codec
and full-decode facts remain the server worker's responsibility. Atomic hard-link
publication refuses an existing destination, including a destination created during
transfer; unsupported filesystems fail explicitly. No video-sized memory buffer,
private HTTP client or fallback overwrite is introduced. Temporary files are
removed on normal success/failure; abrupt process termination can leave a clearly
named unpublished temporary file.

The `media clock` command reads digest-bound clock pages, and `renders sample-binding` retrieves a trusted AUTO phone claim for an explicit new revision.
Manual/browser samples are previews and do not claim sampled-color provenance.
