# Data — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

These are semantic design constraints. Public schemas will be authored in proto and generated through their owner during implementation.

## Storage Overview

Use per-domain repository interfaces over SQLite and api-core's per-user storage roots. Keep immutable binary inputs/results in blob/artifact storage with verified digests; use managed state for checkpoints, bounded cache for decoded frames/previews and logs for diagnostics. Do not store mutable content beside a desktop executable or inside ASAR. A remote worker resolves explicit transferred artifacts, never a developer home path. Pinned policy references are historical context; current authority and consent are rechecked before each external dispatch as specified in FLOWS.md.

## Data Ownership

| Record | Required semantics | Owner |
|---|---|---|
| Project | Identity, owner/membership, revision, selected funding source, privacy/budget policy revisions | projects |
| Source media | Original digest, container/streams, dimensions, orientation, color/range, timestamps, duration, audio policy and capture provenance | projects; byte owner external |
| Device revision | Exact/generic identity, variant, measured/estimated dimensions with sources/uncertainty, units/axes, parts/joints, displays, attachment interfaces, original/adapted assets | devices |
| Scene revision | Physical units, environment assets, support frames/geometry, mounts/clearance/exclusions, lights/cameras, available strategy | scenes |
| Presentation revision | App viewport, fit/crop, OS/browser layers, hardware masks, theme/clock/battery, edge-color policy, display/screen mapping | presentations |
| Motion revision | Time units, transform/joint channels, interpolation, limits and compatible targets | motions |
| Shot revision | Pinned dependencies, source frame schedule, strategy, frame rate, output/color profile, privacy/budget context and compiler version | shots |
| Job/attempt | Idempotency key, actor, input revision, state, attempts/children, leases/heartbeats, output refs, cancellation, resource/cost receipts and reconciliation | jobs |
| Candidate/review | Parent source, edits, reference comparisons, exact scope, reviewer identity, findings and invalidation inputs | creations/reviews |

## Schema Map

Future domain schemas live beside the code interpreting them and register through SchemaProvider/EnsureSchemas. No product tables are implemented by this design. Use stable opaque IDs and immutable revision IDs. Version updates use expected-revision checks and return typed conflicts; do not overwrite another editor's work. Store funding amounts in authoritative integer/minor units or exact decimal representation with unit/rate version, never unqualified float dollars. Unknown is an explicit state, not zero.

Persist versioned coordinate conventions and transforms. Canonical convention v1 uses right-handed meters: +X right, +Y up, +Z toward the reference front viewer; a default front camera looks along −Z and a horizontal support plane spans X/Z. Use active column-vector transforms with local TRS composed T·R·S, normalized quaternion [x,y,z,w] rotations, radians for persisted joint angles and degrees only as a labeled UI conversion. No implicit Euler order. Record per-asset source-to-canonical transforms and named local frames; adapters explicitly convert Blender/other source conventions. Display coordinates have top-left origin, +u right and +v down in normalized [0,1], with pixel-center sampling and explicit UV conversion; do not infer raster origin from mesh UVs. Validate finite values and nondegenerate transforms. Distinguish physical panel outline, active display, software-safe/app viewport and hardware occlusions. A closed laptop envelope is not its open pose. A circular mask is not the rectangular raster. Native recorded dimensions and source timestamps remain independent of output frame size.

## Migrations And Compatibility

Greenfield schemas use the shared declarative mechanism. Use owner-guided out-of-tree conversion for development data when appropriate; production upgrades earn versioned migrations with backup/recovery. Project/package schema versions and renderer profiles version separately. Reject unsupported required features; preserve originals and explain migration choices. Do not silently downgrade a fold, curved display or motion into a flat approximation.

## Import / Export

A portable project contains a manifest of relative dependency names, digests, versions, source/attribution records and permitted editable bytes. Validate archive traversal, symlinks, decompressed size, image/mesh/texture complexity and external references before admission. Disable imported scripts; generated/native code runs only in the isolated execution path. Downloads use authorized public/provider routes with redirects/SSRF restrictions and limits.

Export PNG, MP4, qualified portable GLB and native packages as applicable; a GLB with a static image is not video playback. Exclude secrets and private runtime paths. Rights can permit rendering while forbidding editable redistribution; export must enforce the selected operation's permissions. Rendered outputs carry an adjacent attribution/provenance receipt; preserve source permissions through downstream delivery. The research originals are never import scratch space.

## Retention And Deletion

Keep original imported bytes, pinned shot inputs, published outputs, reviews and unresolved accounting records until an explicit retention decision permits deletion. Delete reconstructible cache independently. Reference-aware garbage collection must not remove assets needed by saved shots, active jobs or reviews. Project deletion explains scope, supports cancellation before commitment, stops/settles owned jobs and reports retained shared/published/accounting references. Privacy deletion and financial retention are distinct owner policies. Shared assets are not removed merely because one project is deleted. Default projects are retained; no background age-based deletion is introduced here.

## Privacy Notes

Real app media remains local unless the project explicitly opts into external processing. The uploaded bytes—not just the original filename—must satisfy the current consent revision. Redacted references have their own digests and evidence. Thumbnails/contact sheets/prompt text/agent vision inherit sensitivity. Provider secrets stay in credential authority; immutable receipts retain safe references rather than tokens or sensitive prompt payloads.

## Cross-References

[DOMAINS.md](DOMAINS.md), [FLOWS.md](FLOWS.md), [security](../internal/SECURITY.md), [presentations](../domains/presentations.md), [research retention](../research/README.md).

## First media intake implementation

E2 begins with project-owned PNG/JPEG originals through a bounded multipart endpoint and typed metadata/list/get operations. The detailed admission and test contract is retained in [the device-definition handoff](../internal/goal/evidence/device-definitions/README.md#e2-handoff-media-admission-and-immutable-photographic-shots). The source filename is display metadata, never a path. Return opaque IDs, byte digest/count, decoded dimensions, actual format, orientation and color-profile status; do not return internal blob keys. The original bytes remain unchanged.

Use the existing projects repository/domain and shared routed blobstore. A small shared scenario adapter may replace the Notes-local routed blob adapter when both product and example use it; keep ownership in the existing shared blob API, not a new storage framework. Resolve roots per request through PickRequired. Import must verify project existence before consuming/storing bytes. Bounded request envelope, file bytes, decoded dimensions and full decode are separate checks. Explicitly reject unsupported orientation/profile until a pinned normalization path exists.

Concurrent identical imports must converge without deleting a winning reference or overwriting an existing blob. Prefer unique staging keys and an atomic metadata uniqueness decision over pretending BlobStore.Put is conditional creation. Clean only this operation's unreferenced staging object on failure or loss of the metadata race. No generic background garbage collector is needed for this slice. Persist project/media ID and digest in future shot revisions; uploaded bytes are not a qualified public asset and trigger no provider call.

PNG/JPEG intake is an implementation stage, not a reduction of the accepted video/shot/export scope. The endpoint and CLI must be usable before the UI import integration; no render availability is implied by an imported source alone.
