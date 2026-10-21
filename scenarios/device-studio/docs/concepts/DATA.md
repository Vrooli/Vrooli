# Data — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19, updated during implementation. The E1 project workspace and E2 photographic still workflow implement a bounded subset; the broader contract below remains the release target. Recording intake is under development. Research demonstrates individual techniques, not a released scenario.

These are semantic design constraints. Public schemas will be authored in proto and generated through their owner during implementation.

## Storage Overview

Use per-domain repository interfaces over SQLite and api-core's per-user storage roots. Keep immutable binary inputs/results in blob/artifact storage with verified digests; use managed state for checkpoints, bounded cache for decoded frames/previews and logs for diagnostics. Do not store mutable content beside a desktop executable or inside ASAR. A remote worker resolves explicit transferred artifacts, never a developer home path. Pinned policy references are historical context; current authority and consent are rechecked before each external dispatch as specified in FLOWS.md.

## Data Ownership

| Record | Required semantics | Owner |
|---|---|---|
| Project | Identity, owner/membership, revision, selected funding source, privacy/budget policy revisions | projects |
| Source media | Original digest, container/streams, dimensions, orientation, color/range, timestamps, duration, audio policy and capture provenance | projects; byte owner external |
| Reviewed asset bundle | Immutable manifest/revision, byte digests and sizes, original/adapted provenance, per-use rights and exact review binding; routed bytes resolved through opaque refs | library |
| Device revision | Exact/generic identity, variant, measured/estimated dimensions with sources/uncertainty, units/axes, parts/joints, displays, attachment interfaces, original/adapted assets | devices |
| Scene revision | Physical units, environment assets, support frames/geometry, mounts/clearance/exclusions, lights/cameras, available strategy | scenes |
| Presentation revision | App viewport, fit/crop, OS/browser layers, hardware masks, theme/clock/battery, edge-color policy, display/screen mapping | presentations |
| Motion revision | Time units, transform/joint channels, interpolation, limits and compatible targets | motions |
| Shot revision | Pinned dependencies, source frame schedule, strategy, frame rate, output/color profile, privacy/budget context and compiler version | shots |
| Job/attempt | Idempotency key, actor, input revision, state, attempts/children, leases/heartbeats, output refs, cancellation, resource/cost receipts and reconciliation | jobs |
| Candidate/review | Parent source, edits, reference comparisons, exact scope, reviewer identity, findings and invalidation inputs | creations/reviews |

## Schema Map

Future domain schemas live beside the code interpreting them and register through SchemaProvider/EnsureSchemas. The narrow E1/E2 implementation now owns project and source-media tables; the broader data contract below remains design intent until its domains ship. Use stable opaque IDs and immutable revision IDs. Version updates use expected-revision checks and return typed conflicts; do not overwrite another editor's work. Store funding amounts in authoritative integer/minor units or exact decimal representation with unit/rate version, never unqualified float dollars. Unknown is an explicit state, not zero.

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

E2 begins with project-owned PNG/JPEG originals through a bounded multipart endpoint and typed metadata/list/get operations. `POST /api/v1/projects/{project_id}/media` imports one file; `ProjectsService.ListSourceMedia` and `GetSourceMedia` return bounded typed metadata pages and single records; `GET /api/v1/projects/{project_id}/media/{media_id}/content` serves the digest-verified bytes by opaque IDs. The current decoder accepts PNG/JPEG up to 24 MiB, 24 million pixels, and 20,000 pixels per side. It fully decodes before persistence, rejects non-identity EXIF orientation, ICC profiles, PNG gamma/chromaticity/HDR declarations, and animated PNG. PNG sRGB declaration is recorded; absent profile metadata is `unspecified`. This records decoder evidence, not full color management. The detailed admission and test contract is retained in [the device-definition handoff](../internal/goal/evidence/device-definitions/README.md#e2-handoff-media-admission-and-immutable-photographic-shots). The source filename is sanitized display metadata, never a path. Return opaque IDs, byte digest/count, decoded dimensions, actual format, orientation and color-profile status; do not return internal blob keys. The original bytes remain unchanged.

Use the existing projects repository/domain and shared routed blobstore. A small shared scenario adapter may replace the Notes-local routed blob adapter when both product and example use it; keep ownership in the existing shared blob API, not a new storage framework. Resolve roots per request through PickRequired. Import must verify project existence before consuming/storing bytes. Bounded request envelope, file bytes, decoded dimensions and full decode are separate checks. Explicitly reject unsupported orientation/profile until a pinned normalization path exists.

Concurrent identical imports must converge without deleting a winning reference or overwriting an existing blob. Prefer unique staging keys and an atomic metadata uniqueness decision over pretending BlobStore.Put is conditional creation. Clean only this operation's unreferenced staging object on failure or loss of the metadata race. No generic background garbage collector is needed for this slice. Persist project/media ID and digest in future shot revisions; uploaded bytes are not a qualified public asset and trigger no provider call.

PNG/JPEG intake is an implementation stage, not a reduction of the accepted video/shot/export scope. The endpoint and CLI must be usable before the UI import integration; no render availability is implied by an imported source alone.

## Recording intake extension — selected design, implementation pending

Extend the existing source-media identity, original-byte ownership, deduplication and pagination for recordings. Keep one desired projects-owned media schema; use kind/readiness plus presence-aware inspected facts. Do not create a duplicate video-original table merely to avoid changing the initial image-only constraints. This pre-release development dataset follows the out-of-tree conversion route above, preserving original rows and blob identities with explicit before/after checks. Production upgrade machinery remains a later earned obligation.

Bounded streaming applies end-to-end, including multipart intake, duplicate verification, staging and export; raising the image byte limit while retaining whole-file reads is insufficient. Inspection runs as an explicit typed subject in the existing jobs owner. Media stores immutable receipt references and readiness; jobs owns attempts, cancellation, unknown outcomes and retryable terminal-result finalization. Ready requires verified source/tool/clock correspondence. Failed or cancelled inspection must also settle the media projection truthfully. Preserve old still job canonical JSON/hash and stored presentation-version dispatch.

### Initial recording envelope and integration boundary

The first local recording profile admits at most512MiB original bytes,10minutes,36,000 video frames,8 streams and3840×2160 (or2160×3840) raster pixels. PNG/JPEG retain their24MiB and existing decode bounds. Sniff a bounded prefix for MP4/QuickTime or WebM/Matroska; admission of a container is provisional until the pinned inspector verifies actual streams, full decode, rotation/color facts and exact PTS. A sniff result cannot mark video ready. Unsupported/HDR/ambiguous timing must return a typed rejection unless the implemented pinned profile explicitly handles it. These are initial operational limits, not claims of support for every codec in those containers.

Multipart intake has exactly one binary file part. Stream directly to one unique routed staging object and validate the multipart terminator before publishing metadata, so an extra field or truncated envelope cannot leave an apparently successful import. Unknown declared length is valid for streaming; distinguish it from an explicitly empty file. Check actual bytes and any supplied declared length. Preserve bounded image full-decode checks. Read/export verifies exact size/digest while streaming; a mismatch after headers aborts the response and clients must reject the incomplete/digest-mismatched transfer rather than receive a success-shaped error body.

One projects-owned inspection projection retains the job ID, state, typed issue and immutable receipt reference. Inspection receipt facts and clock metadata use versioned typed structures; original bytes and clock vectors remain private digest-bound artifacts. Public widths/heights for uninspected media must be absent, not fabricated zero dimensions. Existing image consumers keep their ready image behavior. Public bytes never expose storage paths. The inspector adapter is injected at the application composition boundary; projects does not start processes or acquire a second job ledger. Explicit inspection ensure/retry will use the existing coordinator and deterministic job input. Source import remains truthfully pending if inspection submission is interrupted; read methods do not silently enqueue.


## Reviewed physical asset bundles — E3b first implementation boundary

The library owns immutable binary bundles; devices, scenes and motions retain their
own domain revisions. A bundle manifest enumerates opaque artifact IDs, roles,
byte counts/digests, provenance, original/adapted relationships, attribution and
per-use permissions. A review binds the exact canonical manifest digest and allowed
uses. Integrity alone does not confer rights or visual approval. Initial publishing
is an internal trusted reviewed-bundle operation; a public arbitrary native project
or generated-code import is not silently enabled by this byte owner. The later
creation/import pipeline supplies validated candidates and explicit reviews.

Publishing stages verified bounded streams to unique routed keys and atomically
publishes immutable metadata. A retry of the exact revision converges; changing the
same revision conflicts. A failed or ambiguous metadata outcome must not delete a
possibly referenced object. Original research bytes are preserved outside import
scratch. Readers resolve opaque bundle/revision/artifact IDs and enforce exact size
and digest; public shapes never contain private blob keys or filesystem paths.

Use the existing routed blobstore, database SchemaProvider and repository pattern.
New library tables belong beside the library domain and are independent of device,
scene, motion and project table lifecycles. No hard cross-domain foreign keys,
second queue, downloader, process owner or generic migration framework is introduced.


## Physical component implementation contract

The internal physical foundation now has three typed domains: devices, scenes and motions. Shared `internal/physical` owns canonical meter/TRS/quaternion values and fully pinned artifact references. `internal/library` owns immutable reviewed bundle bytes, provenance, per-use rights and digest-verified readers. Device candidate revisions have immutable expected-head persistence; scene and motion candidate values are implemented, and their revision/public integration is in progress. Candidate validity does not confer geometric, mechanical, visual or rights qualification.

Device geometry binds exact GLB node identities to semantic parts, source units/basis, displays and support frames. Joint axis/pivot are expressed in the parent-part local frame; the child rest frame is parent-relative. Scene candidates bind geometry/texture artifacts, parts, material slots, support/region frames, camera optics/focus, area lights and environment. Material v1 fixes linear image filtering; emission strength is1.0. MASK requires an explicit cutoff. Motion candidates use exact rational timebases, integer key ticks, typed root/joint/camera targets, bounded STEP/LINEAR/SMOOTHSTEP interpolation and shortest-arc quaternion SLERP. The sampler returns keyed values; placement/rest/pivot composition belongs to the physical compiler.

A physical shot will pin exact device/scene/motion revisions and canonical digests, a versioned renderer profile, screen presentation and an exact output schedule. Compilation must freeze a complete immutable job closure, recheck bundle rights/review for actual use, and reject unresolved/unsupported geometry or capabilities before native dispatch. Existing photographic snapshots retain their original serialization/hash behavior when no physical binding is present. The existing jobs/coordinator and platform-owned operation remain the sole process lifecycle owners. No host path, mutable component head, arbitrary Python, or imported native project may enter the public closure.

The current phone study establishes an approved still and a verified48-frame rotation with a stationary app screenshot; it does not yet establish this public physical-shot path. Laptop overview studies preserve all source faces but do not qualify a mechanical hinge. Exact studies and design exploration remain in ignored local delivery evidence; their current production contracts belong here and in the owning domain source.
