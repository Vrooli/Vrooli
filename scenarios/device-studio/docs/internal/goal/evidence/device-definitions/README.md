# Bounded device-definition foundation

This directory records the prewrite baseline and scope limits for the first device registry. The registry is source-backed scaffolding, not the qualified library, and it does not implement any UI/API/proto surface. No files were copied out of or written into `/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18`.

## Layout and checks

- Go domain and conservative validation: `scenarios/device-studio/api/internal/devices/definition.go`
- Desired-behavior tests: `scenarios/device-studio/api/internal/devices/definition_test.go`
- JSON definitions: `scenarios/device-studio/assets/devices/*.json`
- Original prewrite absence record: [prewrite-baseline.json](prewrite-baseline.json)
- Focused verification: `cd scenarios/device-studio/api && go test ./internal/devices`

Each seed keeps manufacturer measurements separate from unknown screen apertures, safe areas, cutouts, joints, poses, motion and supports. Lengths use the `mm` unit with X=width, Y=up/height and Z=depth as the named front-view axes; mechanical pivots use meters and joint angles use radians. Display regions use normalized top-left UV coordinates with an evidenced physical width/height metric for each surface. This metric can be anisotropic, so UV distances are not physical Euclidean distances unless both spans have equal scale. Non-rectangular qualified regions (including circles and rounded rectangles) require a concrete physical-mm geometry reference and shape evidence; a UV bounding box alone never proves the shape. Qualified non-rectangular containment or overlapping occlusion decisions also require an evidenced geometry relation. If the referenced geometry or relation is unavailable, keep the region/relation candidate-only or unknown so physical qualification remains needs-review. These conventions follow [`DATA.md`](../../../../concepts/DATA.md#coordinate-conventions).

## Seed coverage

The 11 JSON fixtures cover all eight accepted groups, with additional explicit examples for a sourced Samsung foldable and a foldable-iPhone concept:

| Fixture | Sourced facts and explicit unknowns |
|---|---|
| `iphone-16-pro.json` | Apple nominal body dimensions; active screen, Dynamic Island mask, support and physical rotation remain unknown. The concept fixture below is not used as its spec. |
| `macbook-pro-16-2021.json` | Apple nominal closed envelope. Open envelope, active display, hinge pivot/limits and motion remain unknown. |
| `ipad-pro-13-m4-2024.json` | Apple bare-tablet envelope. Display bounds, cover/keyboard and attachment geometry remain unknown. |
| `galaxy-z-fold6.json` | Samsung nominal folded/unfolded body envelopes and distinct inner/cover display identities. Screen boundaries, bend radius, pivot and hinge range remain unknown. |
| `foldable-iphone-concept.json` | Explicit concept identity only. It makes no Apple product claim and contains no invented dimensions, display geometry or fold limits. |
| `apple-watch-series-9-45mm.json` | Apple 45 mm case dimensions; active mask is explicitly unknown (not a rectangular screen assumption); strap fit remains unknown. |
| `dell-ultrasharp-u2723qe.json` | Dell no-stand, low-stand and high-stand envelope configurations. Active panel bounds and wall-mount interface remain unknown. |
| `samsung-du8000-55-ca.json` | Canadian SKU with/without-stand envelope and published VESA interface. Active screen bounds and complete bracket/load compatibility remain unknown. |
| `google-nest-hub-2nd-gen.json` | Manufacturer complete-tabletop envelope for second generation. Active display bounds are unknown; generic model listings do not prove generation. |
| `generic-digital-picture-frame-concept.json` | Generic adaptation, not a sourced digital-frame product. Body, active display and supports remain unknown; passive frame listing is only a mismatch-marked lead. |
| `bmw-i4-2022-curved-display-host.json` | BMW host model and separate 12.3/14.9-inch display diagonals. Assembly dimensions, active bounds, curvature, mounts and per-option CarPlay/Android Auto compatibility remain unknown. |

Manufacturer sources and archive record locations are embedded per field in each JSON file. Core measurements are nominal published dimensions; quantified tolerances, protrusions and measured mesh conformance are absent, so uncertainty is represented as `source_unspecified`, never fabricated. A null dimension is accepted only when the method and uncertainty are explicitly `unknown`.

## Candidate models and rights

Only locally retained v3/v4 candidates receive archive paths and SHA-256 values. The imported v4 MacBook GLB is creator-labeled M3/2024 and CC BY 4.0; its source metadata says it was scaled to the 355.7 mm width of the 2021 Apple catalog device. The fixture keeps it a family candidate and explicitly forbids applying 2021 dimensions to it. Credit jackbaeten and retain required attribution/change notices if reused; exact dependency rights still require closure. The imported iPhone model is an iPhone 15 Pro Max, CC BY 4.0, credited to MajdyModels/MG990 (embedded credit MpPower™); it is not the measured iPhone 16 Pro definition. Its local GLB digest is retained. The locally authored iPhone 16 Pro candidate has no identified third-party model license; Apple source images are copyrighted/reference-only and the author's redistribution authority remains unresolved.

The free iPhone 16 Pro, Galaxy Z Fold6, Apple Watch, iPad and exact-title laptop listings are metadata-only leads; no model bytes were acquired or inspected for them. Some listing terms are platform-reported CC BY 4.0 or royalty-free metadata, not proof of file-level rights or redistributed dependencies. The monitor lead is an Apple Studio Display, not a Dell U2723QE match. The passive CC0 picture-frame lead is not a digital display assembly. These remain `not_acquired`, never qualified assets.

## Qualification and compatibility behavior

`Definition.Validate` validates both incomplete and review-qualified domain values. It does not treat an unqualified seed as an error. A value of `qualification: qualified` requires a passed `physical_geometry` review record with a named reviewer and evidence; `PhysicalQualification` independently checks source-backed finite dimensions, uncertainty status, display boundary/active/app-safe shape and any declared joint geometry. The registry loader uses strict JSON decoding, rejects unknown fields and trailing content, and rejects duplicate IDs or duplicate group/kind/manufacturer/model/variant identities. It calls `ValidateSeed`, which rejects qualification claims or embedded review passes in seed files.

Compatibility is conservative. `CheckSupport` requires an exact support kind and checks the requested named dimension state (and optional named pose) rather than using whichever configuration happens to have dimensions. `CheckMotion` requires an explicit motion kind; articulated motion names each required joint, and only joints with finite axis, pivot and radian limits can pass. Unknown/candidate claims return `needs-review`; explicit incompatible claims reject. Surface bounds separate physical boundary, active display, app-safe area and occlusions; qualified app-safe pixels cannot overlap a qualified hardware occlusion. Non-rectangular geometry is represented by a UV bounding box for indexing plus an explicit physical-mm geometry reference and evidence; the bounding box never substitutes for physical shape.

Every seed remains `unqualified`. No model is copied or claimed as matching the sourced device, and there are no visual, motion, rights-closure, mount-safety, runtime, or library-release approvals in this foundation.

## E2 handoff: media admission and immutable photographic shots

This is an implementation recommendation only. It adds no runtime or API behavior and is not an acceptance receipt. Reuse the existing per-domain pattern: `api/main.go` resolves `api-core/storage` roots; `handlers/notes` owns a `blobstore.BlobStore` rooted below the `data` class and injects `MemoryBlobStore` in tests; `internal/projects` owns project validation/revisioned SQLite state; `internal/jobs` already persists render-job inputs and output receipts. Keep these owners: projects owns project/source-media metadata and source revisions; `internal/shots` owns immutable shot rows, compiler inputs and shot revision services; blobstore owns opaque source/derived bytes; jobs owns attempts and receipts; rendering owns composition. Avoid a new storage abstraction/framework.

### Public contract shape

This contract is provisional: do not freeze message fields until E2 defines the compiler/input definition and exact persistence boundary. Put source-media RPC/messages in the existing projects proto domain, and define a separate `ShotsService` under `packages/proto/schemas/device-studio/v1/shots/` for immutable shot create/get/list behavior; do not grow `ProjectsService` to own every behavior. Generate only through the proto owner when implementation is admitted. Match the existing Notes multipart exception for upload bytes: a bounded `multipart/form-data` REST upload endpoint can return a proto-typed `SourceMedia` response, while source listing/get and shot operations use Connect RPCs. Declare the upload REST exception explicitly in endpoint descriptors; do not base64 image bytes through Connect JSON.

Suggested proto-first shape only (names and fields are illustrative, and provisional):

```proto
service ProjectsService { // existing service; append source-media operations only
  rpc ListSourceMedia(ListSourceMediaRequest) returns (ListSourceMediaResponse);
  rpc GetSourceMedia(GetSourceMediaRequest) returns (GetSourceMediaResponse);
}
service ShotsService { // separate shots proto domain
  rpc CreateShotRevision(CreateShotRevisionRequest) returns (CreateShotRevisionResponse);
  rpc GetShotRevision(GetShotRevisionRequest) returns (GetShotRevisionResponse);
}
message SourceMedia {
  string id = 1; string project_id = 2; string sha256 = 3; int64 size_bytes = 4;
  string media_type = 5; uint32 width = 6; uint32 height = 7;
  string orientation = 8; string color_model = 9; string color_profile = 10;
  string original_filename = 11; google.protobuf.Timestamp imported_at = 12;
}
message ShotRevision {
  string id = 1; string project_id = 2; int64 revision = 3;
  string source_media_id = 4; string source_sha256 = 5;
  string device_revision_id = 6; string scene_revision_id = 7;
  string presentation_revision_id = 8; string motion_revision_id = 9;
  string recipe_sha256 = 10; string renderer_version = 11;
  string output_profile = 12; string compile_status = 13;
}
```

The upload response carries no host path or blob key. Source metadata and bytes stay project-owned; shot metadata, immutable revision storage and shot service logic belong to `internal/shots`. Use an opaque media ID in public requests and responses; store an internal data-plane key such as `projects/<project-id>/media/sha256/<digest>` only in the private repository. Exports contain project-relative dependency IDs/digests and permitted bytes, never absolute filesystem paths or local blob keys. Shot creation pins exact source digest, all dependency revision IDs/digests, recipe/compiler version and output profile into a new immutable revision. Editing creates a new revision; rerendering an old revision resolves its pinned inputs and verifies their digests before using cache/job state. A source filename is display metadata only, never a path or authority.

### Still-image admission boundary

Start with PNG and JPEG stills only. Ignore client `Content-Type` as authority. For multipart, cap the entire HTTP request (including boundaries and fields) before parsing, then independently cap and count the file-part stream; never compare request `Content-Length` with file bytes. Compare multipart parser file-part size metadata with the bytes actually read where available. For a raw-body upload only, the request `Content-Length` describes the file body and may be compared with bytes read. Inspect signature, whitelist the two formats, decode dimensions from bytes, enforce nonzero width/height and an explicit maximum pixel count before full decode, then fully decode to reject corrupt/truncated bodies. Hash the exact original bytes with SHA-256 while staging; record actual byte count, detected MIME, width/height and original filename. Use a digest-derived key and never expose it. The shared `BlobStore.Put` interface permits overwrite and does not provide conditional creation, so code must ensure duplicate digest writes contain verified identical bytes; metadata insert failure must clean up only a newly-created unreferenced blob, with concurrency handled deliberately rather than deleting a shared digest.

Record decoded color model and whether ICC/EXIF orientation metadata is present. Go's standard PNG/JPEG decoders do not provide a complete color-management contract; do not claim color preservation or silently apply an unspecified transform. For the initial SDR still slice, reject or mark needs-review for unsupported embedded ICC/HDR profiles and non-identity orientation until explicit normalization behavior and receipts exist; preserve source bytes unchanged. Any normalized derivative gets its own digest and transform/tool receipt. Validate by full decoding, dimension/pixel bounds and source/digest checks; never trust extension, header MIME, multipart size or an image's advertised dimensions alone.

### Minimal seams and tests

Implement one project-owned `MediaImportService` with injected `BlobStore`, project/source repository, clock/ID source and policy limits. Keep transport parsing in the handler, source-media SQLite metadata in `internal/projects`, and immutable shot rows/service in `internal/shots`; use the existing shared `api-core/storage` resolver only to configure the filesystem blobstore. Do not add a dependency for PNG/JPEG or media probing.

Required focused integration tests:

1. PNG and JPEG happy paths: spoofed MIME/extension ignored; response metadata matches byte digest, size, dimensions, detected MIME; stored bytes round-trip exactly.
2. Admission rejects empty, oversized request envelope, oversized file part, truncated/corrupt content, unsupported format, dimension/pixel bomb, file-part size mismatch (or raw-body Content-Length mismatch), unsupported orientation/color metadata; no metadata row or orphan bytes remain.
3. Duplicate upload of identical bytes is idempotent by digest; different content cannot overwrite an existing digest; concurrent same-digest imports remain correct.
4. Repository failure after blob put performs safe cleanup; duplicate references prevent cleanup of shared bytes.
5. Public JSON/proto/export surfaces contain no absolute path, private blob key, credential or host-specific root; export dependencies resolve only through package-relative ID/digest.
6. Create a shot from imported source plus pinned revisions; mutate working project selections and create another shot; first revision remains byte-for-byte stable and `GetShotRevision` resolves original source/recipe digests after API restart.
7. Renderer/job integration reopens the shot by revision, verifies every digest before reading, writes a deterministic receipt with input/output digests, and survives cancel/failure without mutating the source or prior shot.

### Video continuation without a second contract

Keep the same `SourceMedia` identity, exact-byte digest, repository reference, immutable shot pinning, provenance, rights and export rules for later MP4. Extend metadata with container/streams, rational timebase, per-frame presentation timestamps, audio policy, duration, color primaries/transfer/range and decoder identity; do not reinterpret `width/height` or source time. Admit MP4 only after bounded container probing, stream/resource limits, full decode, timestamp validation and explicit audio stripping/passthrough behavior. Decoder work belongs to the platform/tool owner; preview and render must share the pinned frame schedule. Do not introduce a parallel “video asset” model or infer timing from frame index/FPS.
