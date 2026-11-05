# Layered photographic parallax

Parallax is the apparent relative movement of near and far content when the
viewpoint changes. A depth map is an input, not an animation. Native deterministic
`ops composite` now renders one bounded nearby view; `ops parallax` renders a
lossless RGBA PNG-sequence ZIP and its manifest. These use the same pinhole
projection and prepared-layer compositor. They do not reconstruct a complete
scene from a single photograph or articulate photographed devices.

## Scene preparation and qualification

### Approved delivery contract, 2026-10-22

The full delivery includes three source routes: geometry-first using existing
scene assets; an empty photographic scene paired with a known device; and
photographed-device removal/replacement using a reviewed clean plate. It also
includes reusable prop reconstruction, normal API/CLI/UI access, durable
orchestration, resource qualification and an improved Aquila film. The earlier
foundation receipt does not close this destination. Progress and outstanding checks are retained in
`docs/internal/PROBLEMS.md` under Hybrid scene preparation and inference lifetime.

The exchange contract is `hybrid-device-scene/v1`. Its typed photographic DTO
lives in `packages/proto/schemas/image-tools/v1/preparation/preparation.proto`.
Geometry-first uses Device Studio's existing immutable scene/device/setup
revisions directly, so it does not invent a photograph or estimated camera.
Image Tools prepares immutable
assets; Device Studio validates and renders them through its existing physical
asset and job owners. The program owns stage references and retry identity, not
another inference queue. The PreparationService assembles and retains the exchange through its normal
API/CLI and exposes source/clean-plate comparison and immutable reviews in the
Library UI. Device Studio conversion still requires its independent inner
geometry, placement and render checks; retaining a bundle does not qualify a shot.

After a preparation receives a completed visual review, Device Studio exposes
`ScenesService.ImportPreparedPhotograph` through both `scenes from-preparation`
and the Library intake form. The caller supplies the exact preparation SHA-256
and explicit license, attribution, source-label and source-use claims. The
default Library rights remain edit-only. Local rendering is added only when
`allow_local_render` is explicitly selected with a bounded rationale; this is
source-use permission, not motion qualification, publication or redistribution
approval. The typed
`SceneCandidate.photographic_background` retains the source review, geometry,
camera envelope and estimated-scale lineage across save/get; the returned scene
and Library revisions remain unqualified. The immutable artifact adaptation
records the current
`photographic-scene-assembly-v5-gltf-top-origin-uv-row-major-affine-identity-emission-estimated-light-shadow-source-camera-plate-no-parallax`
profile. Its typed `PhotographicBackground.render_mode` is
`SOURCE_CAMERA_PLATE_V1`: it composites the exact digest-verified staged source
raster behind the known device and analytic support-plane catcher. The camera
must remain fixed; no parallax or camera-motion claim is made. The older
depth-reprojected beauty-mesh path remains a separate experimental mode because
single-view depth discontinuities can leave visible holes. The profile binds
top-origin glTF texture V, the canonical row-major affine UV identity, baked
source appearance, an explicitly estimated device light and support shadow.
Changes to conversion semantics create a distinct candidate identity without
changing the source-use claim digest. A pending
`EMPTY_SCENE` can be inspected first with the read-only `scenes geometry-qa`
command, which emits source/ROI/depth/plane overlays and never changes review
state.

The `image-tools.scene-preparation-stage` program also exposes a bounded
`import-scene` action. It forwards an accepted child preparation, exact bundle
SHA and declared source-use claims to Device Studio's typed scene owner; it does
not accept a caller-supplied assembly profile or claim that the resulting scene
is qualified. The same import remains available through Device Studio's typed
RPC, CLI and Library form. The source-camera plate compositor preserves the
original raster without reprojecting it, adds only an explicitly estimated
source-window light for the known device, and composites a transparent device
pass with a support-plane shadow catcher over the plate. Conversion provenance records source-camera
pixel-center UVs in glTF's top-origin convention; Device Studio's importer
performs the single V flip. Affine UV arrays use `[a,b,tx,c,d,ty]`, evaluated
as `u'=a*u+b*v+tx` and `v'=c*u+d*v+ty`. Native visual qualification of this
photographic composition and its bounded motion remains separate from
geometry-first scene qualification.

Current native samples on 2026-10-22 close two bounded photographic paths.
The empty-desk preparation uses `SOURCE_CAMERA_PLATE_V1`; its reviewed 960×540
static frame and 48-frame, 24 fps, fixed-camera device motion both preserve the
source plate and contact shadow. The motion is a 6 mm slide and 2° yaw of the
known device. This is a stationary-viewpoint composition: it does not qualify
camera parallax or measured scene scale. Its 1920×1080 MP4 and render receipt are
retained under the hybrid-device-scenes effort evidence.

The photographed-device sample uses a separately reviewed authored clean plate,
cropped from 1536×1024 to 1536×864 at source offset (0,80), with matching source,
plate, masks and freshly inferred MoGe geometry. A 960×540 native static preview
shows the known phone manually placed on the estimated support plane with a
composited contact shadow. The crop clips the original phone's upper edge; the
placement is an authored composition, not an original-pose fit, and monocular
scale remains unmeasured. The preview does not qualify photographed-device
motion or a local repair model. Source-camera compositing accepts only source and
output aspect ratios within 0.2%; other aspect ratios must be explicitly cropped,
re-reviewed and re-prepared rather than stretched.

- Preserve oriented source width/height, original bytes/hash and the explicit
  EXIF-to-oriented transform. Masks and geometry refer to this oriented raster.
  Pixel coordinates use centers; normalized intrinsics divide the first matrix
  row by width and the second by height. Store dimensions alongside intrinsics.
- Geometry estimates use OpenCV camera axes (+x right, +y down, +z forward).
  Store finite float32 depth/point/normal arrays and a separate validity mask;
  invalid samples do not become zero-depth surfaces. Distinguish learned metric
  estimates from measured scale. Retain model/checkpoint and postprocess settings.
- Device Studio's existing world basis remains authoritative:
  `device-studio-right-handed-meters-x-right-y-up-z-front-v1`. Camera-local
  conversion is `(x,y,z) -> (x,-y,-z)`, followed by the explicit camera-to-world
  pose. Matrices are column-major, active column-vector transforms; composition
  is parent × local. No implicit camera pose or inferred conversion from names.
- Support planes carry normal, offset, inlier support and residual in the same
  declared basis/units. Fitting excludes invalid and removed-device pixels.
  Camera, rigid object pose and hinge angle remain independently inspectable.
- Keep the original photograph, removal mask (including superseded shadows and
  reflections), repaired background, device mask, occluder masks and optional
  refined alpha as separate hashed assets. A segmentation region is not a matte.
  Record repair crop/overscan transforms and generated-region masks.
- LaMa `object_removal` accepts explicit `backgroundRepair.maskExpansionPixels`
  (0–64 oriented source pixels) and up to 32 normalized `removalRegions`.
  Unset controls preserve the supplied mask unchanged; additions union with it
  and never infer other removal areas. PNG provenance records the uploaded mask
  digest, effective mask as an embedded PNG payload, added pixel count, oriented
  crop and letterbox transform, elapsed inference time and sidecar process peak
  RSS. Pixels outside the effective mask retain their original decoded RGB
  values. The fill remains an estimate and needs visual review, especially when
  the removal spans most of the fixed model crop.
- An experimental fixed-1024 Big-LaMa export is now built through the managed
  model owner from pinned Apache-2.0 source and generator-only safetensors. The
  native build used a 16 GiB RAM and 2000m CPU no-reclaim claim, a 16 GiB cgroup
  ceiling and two PyTorch/BLAS threads; it completed in 145.8 seconds with a
  13,639,499,776-byte exporter-process high-water mark. It passed strict tensor
  parity, a 512 reference probe (max absolute difference 0.001114), and 1024
  ONNX parity (max absolute difference 0.01458). This qualifies the export
  artifact, not photographic repair quality. A native 1536×1024 phone removal
  using the existing full-device mask took 30.8 seconds of repair time and
  6.2 seconds to load the model under an 8 GiB shared no-reclaim RAM claim; the
  claim released and the sidecar process high-water mark was 3,302,875,136
  bytes. Every decoded source pixel outside the 546,274-pixel effective mask
  remained exact, but visual review rejected the result because a broad dark
  device/shadow footprint remained. The candidate stays disabled. Preserve this
  rejected evidence at
  `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/`;
  `lama-1024-phone-repaired-native-v1.provenance.json` records the source, mask,
  output, model, resource and transform details. CPU ONNX sessions now use at
  most eight intra-op threads, one sequential inter-op thread, and record those
  settings; this bounds worker fan-out but does not reserve CPUs, set affinity,
  or impose a quota.
- If LaMa's fill is rejected on inspection, the existing local `inpaint` operation
  can regenerate the same retained effective mask with an explicit scene prompt.
  This is a separate durable job with a selected inpainting model and recorded
  prompt/seed/checkpoint provenance; no provider silently escalates a repair.
  The typed `InpaintParams` crop option bounds inference to the non-empty mask
  plus explicit source-pixel context, optionally resizing that crop to a 512 or
  1024 maximum edge. The sidecar maps the generated crop back to the oriented
  source and composites only through the original full-resolution binary mask;
  pixels outside it remain exact. Provenance records the source crop box and
  source-to-inference transform. This does not make generated texture factual:
  review the full frame for broad low-detail areas, shadows and texture changes.
  An explicit `neutralize_masked_pixels` option replaces masked pixels in the
  diffusion init crop with a smoothed nearest-unmasked-context fill before inference.
  This avoids conditioning the generator on the object being removed; it does
  not change the output mask or the exact outside-mask compositing contract.
- The SDXL Inpainting 0.1 candidate uses a pinned Diffusers repository revision
  and fp16-only component weights. It remains disabled until managed installation,
  GPU-bound execution, outside-mask verification and a photographic quality review
  pass together; the model card warns about degraded sharpness at strength 1.0.
- Reuse Device Studio's immutable device, scene, motion, setup and recording
  revisions. A device fit refers to exact mesh landmarks and supplied 2D points,
  with reprojection residuals and a visible correction overlay. Do not invent
  automatically detected landmarks or accept a poor fit silently.
- Store lighting/material settings, screen source hash/clock, bounded camera
  envelope, output profile and all source/model/settings hashes. A changed input
  invalidates dependent stages; a lost response reattaches to the same stage.
  Relative references must remain inside retained artifact roots.

Qualification uses known-camera laptop and phone scenes, an empty photographic
plate and a photographed device. Each route must retain actual input/output
identities and an inspected sample, not only mocks or catalog rows. Review
perspective, thin edges, table contact, occlusion, attached screen and temporal
stability at final playback scale. The Aquila sample must end on the exact
off-center rectangle, source recording and source time of the following shot;
other shots keep their independent layouts. Required host proof includes stage
RAM/VRAM and latency, contention, cancellation, crash recovery and released
capacity. Unavailable targets remain explicitly unverified.

Resource policy is independent of job priority: `allow_reclaim=false` must
prohibit reclaim in the control-plane decision itself, including idle-yield
targets. Lowering priority is insufficient. Release failures must be observable
with claim identity and retain the native TTL recovery boundary. Never release
a claim before its inference process is reaped.

### Authored clean plates

A reviewed external image edit can supply a clean plate when local repair is
unsatisfactory. Its declared repair JSON uses the separate schema
`vrooli.image-tools.authored-clean-plate/v1`; it is caller-retained evidence,
not server-attested inference or a local model qualification. Required fields:

| Field | Meaning |
|---|---|
| `source_sha256` | Exact oriented source artifact hash |
| `mask_sha256` | Exact declared removal-mask hash |
| `output_sha256` | Exact clean-plate artifact hash |
| `tool` | Tool name, at most 128 bytes |
| `tool_output_reference` | Retained tool output reference, at most 1024 bytes |
| `prompt` / `prompt_sha256` | Exact UTF-8 prompt, at most 16 KiB, and its SHA-256 |
| `pixel_scope` | Exactly `full_frame_generated` |

The protected preparation bundle retains this JSON alongside its hashed source,
mask, geometry and clean plate. Device Studio checks these bindings during normal
conversion. This variant does not accept `model_id` or `model_revision`: an
undisclosed tool checkpoint must not be invented or presented as local inference.
Full-frame generation does not promise unchanged pixels outside the removal
mask. Inspect framing, registration, texture and newly generated regions before
acceptance; an accepted repair still needs independent scene/motion review.

### Durable preparation and visual review

`PreparationService.CreatePhotograph` accepts a typed pending manifest and exact
bindings to Image Tools-owned source, mask, geometry and repair outputs. The
server verifies every member's actual size/hash and raster dimensions, packages
only declared artifacts, and retains the content-addressed ZIP plus source and
clean-plate previews independently of disposable job outputs. The declared
repair-provenance JSON can be supplied inline (at most 64 KiB); its artifact
hash pins the exact bytes. This is caller-supplied evidence, not a server
attestation of model execution.

Use `image-tools preparation create --file request.json`, then `get <sha256>` or
`list`. `--request` accepts the same bounded JSON for governed programs. Source
changes create pending assets; an old accepted review cannot be copied into a
new source admission. `ReviewPhotograph` / `preparation review --file review.json`
creates an immutable child. Include a caller-generated `operation_id` UUID and
reuse it unchanged when retrying after an uncertain response; the owner binds
that key to the canonical review-request fingerprint and returns the same
child, while rejecting key reuse for changed review content. Original bytes
and decisions remain readable.
Metadata lives in the owning `prepared_photograph` domain schema; all pixel and
geometry bytes stay outside SQLite in protected data blobs.

The Library's Prepared scenes section compares the retained source and clean
plate, shows prior observations, downloads the bundle, and retains acceptance or
repair requests with reviewer, rationale and evidence references. Camera-motion
coverage requires its own explicit inspected-overscan acknowledgement. A review
is an observation; Device Studio still validates geometric and motion contracts.

Derived `out/` references resolve to the scenario cache class. Uploaded inputs,
masks and content-addressed preparation ZIPs/previews stay in durable data;
assembling a preparation copies exact verified artifacts out of job caches.
The private lock-pinned Python runtime, materialized sidecar and requirements
lock also use cache; installed model/adapters remain in data. Existing output
references require a stopped-owner Storage Manager placement migration before
the new writer can start. Startup refuses a leftover legacy output tree instead
of silently losing references or reading two ambiguous roots. Rebuild a runtime
at its new path through the native provisioning owner: virtualenv entrypoints
can contain absolute paths and must not be treated as relocatable.

The `image-tools.scene-preparation-stage` governed program handles one bounded
owner boundary: submit/observe/cancel an inference stage, assemble a pending
preparation, inspect its identity, or retain a review. Native jobs own inference
and cancellation; immutable preparation identities own the assembled checkpoint.
The caller retains exact stage UUIDs, result refs and preparation IDs between
invocations. It does not add a second job queue or silently retry paid providers.

### Full scene geometry: implemented output

The model-backed `scene_geometry` operation runs pinned MoGe-2 ViT-S ONNX
through the native CPU provider and the durable AI job owner. It restores MoGe's
focal/shift camera fit and pinhole reprojection omitted by the raw ONNX graph.
The resulting ZIP contains `points.npy` and `depth.npy` as float32, `normals.npy`
as unit float32 camera-frame vectors, `validity.npy` as a separate uint8 mask,
`intrinsics.npy` as normalized K, and `manifest.json` with hashes, raster,
camera, model and uncertainty provenance. `np.load(..., allow_pickle=False)` can
read each NPY member from the ZIP. Invalid geometry is NaN in point/depth arrays
and is always identified by validity; it is never encoded as zero depth.

Support-plane fitting is opt-in and uses only a caller-selected normalized image
rectangle. The fitted normal, offset, threshold, inlier count/fraction and
residual quantiles are retained in the manifest. `support_plane_rejected.npy`
marks invalid or rejected samples inside that ROI; zero means either an inlier
or a pixel outside the ROI. Plane distances and offsets are in MoGe's estimated
metric scale and do not establish physical measurements. For example:

```bash
image-tools ai geometry device-on-table.png \
  --support-roi '{"supportRoi":{"x":0,"y":0.868,"width":1,"height":0.132}}' \
  --wait --out device-geometry.zip
```

The known-camera analytic fixture validates focal/shift recovery and projection
independently from learned scale. A durable-job qualification on a photographed
laptop plate is retained in
`~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/moge-device-table-qualification.json`.
It demonstrates a stable selected table-region fit, but there are no measured
intrinsics or scene distances for that photograph, so those residuals remain
model-consistency evidence only.

### Prompted device masks: implemented foundation

The model-backed `segment` operation uses the existing AI durable-job owner,
model selector, installed-weight checks and serialized local-model lane. It is
distinct from heuristic `select segment`. Submit typed
`AIParams.segmentation` on `POST /api/v1/ai/segment`, then wait once through
JobsService. The CLI exposes the same path:

```bash
image-tools ai segment source.png --model mobilesam \
  --selection '{"box":{"x":0.06,"y":0.02,"width":0.8,"height":0.96}}' \
  --wait --out device-mask.png
```

Prompts use normalized source-image coordinates. `points` supports positive and
negative hints (`negative: true`), optionally alongside a box. At least one
positive point or a positive-area box is required; auto-subject selection is not
silently replaced with a center click. The qualified MobileSAM export accepts
SAM1 encoder/decoder tensors. SAM 2.1 Tiny uses a separate Transformers adapter
with its own pinned config and checkpoint plus explicit upstream-matched
processor defaults; it selects the
highest-IoU prompt mask and uses Transformers source-size postprocessing. Both
return binary region masks, not feathered transparency mattes. The sidecar
honors EXIF orientation and bounds input to 16 million pixels; the native job has
a 180-second execution deadline. The local-model queue serializes CPU model
work too; queue name does not imply GPU execution.

Qualification on the existing 1672×941 Aquila device plate produced an inspected
laptop silhouette through an isolated HTTP → durable job → installed MobileSAM
→ output-blob run in 2.83 seconds. The request disconnected after admission and
the job completed independently. This example does not qualify transparent
edges, every device type, SAM 2.1, or complete scene extraction. Evidence lives in
`~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/sam-native-job.log`.

### Interactive mask correction

The existing Select workspace now offers **Object masks** alongside the
explicit colour-selection tool. Place include/exclude points by pointer or
coordinate inputs, undo/clear hints, choose an available local model, and update
the binary region. The editor retains the actual serving model and downloads
the native result mask. It does not invent a classification or transparency
matte from segmentation confidence.

Each changed prompt gets a new operation identity; retrying a lost admission
response reattaches to the same durable job. Progress uses WatchJob and terminal
completion uses one WaitJob call. Cancellation reaches the job owner, including
an admission arriving after the user cancelled; cancellation failures remain
visible. Replacing the source unmounts the prior editor and cancels its work.
Focused interaction, late-admission cancellation, retry-identity, typed multipart
and accessibility checks pass. A native BAS browser capture on 2026-10-22 also
verified source attachment, switching to Object masks, and adding a coordinate
point at desktop size, with no captured console errors. This browser pass did
not submit inference or qualify a completed overlay; native model jobs provide
separate inference evidence.

### Recoverable preparation stages

`AIService.SubmitStoredAI` and `image-tools ai submit-stored` admit retained
Image Tools `input_ref`/`mask_ref` artifacts without uploading the same bytes
again. Both transports share typed validation, deployment/consent gates, model
planning and the existing durable job owner. A required UUID `operation_id`
reattaches to an existing admission before current model selection. Identity
includes typed parameters and actual input/mask bytes, so another key containing
the same bytes reattaches; changed bytes or parameters conflict. A retry can
recover after a model is unloaded. References are opaque owned blob keys, never
arbitrary filesystem paths or remote URLs.

The governed `image-tools.scene-preparation-stage` program performs one bounded
owner action per invocation: `submit`, `observe`, `cancel`, `assemble`,
`inspect-preparation`, `review`, `assess-repair` or `import-scene`. Inference
actions pin the selected model, force local-only execution with no resource
reclaim, bound their report and retain native job/operation identity. Call
`image-tools jobs wait <job-id>` once for completion. A lost submission response
is reconciled with the exact original inputs and UUID. Cancellation readback
distinguishes an accepted request from a terminal job; neither is a claim that
capacity release was independently verified. PreparationService owns immutable
source bundles and review children; the program forwards accepted child IDs,
exact bundle identity and declared source-use choices to Device Studio's typed
import owner. It does not own inference or create a second job queue. Every
prepared result still requires visual review, and scene/motion qualification
remains a separate Device Studio decision.

### Research choices and remaining delivery gates

| Stage | Candidate and reason | Evidence required before promotion |
| --- | --- | --- |
| Low-resource masks | Repaired existing CPU MobileSAM | Varied devices, thin edges, multiple prompts and refinement; keep segmentation distinct from matting |
| Higher-quality masks | [SAM 2.1 Tiny](https://github.com/facebookresearch/sam2) through the separate [Transformers adapter](https://huggingface.co/docs/transformers/model_doc/sam2), exercised with the scenario lock's Transformers 4.57.6 and Torch 2.12.1+cu130; pinned HF revision `de431c4043854a71d8101e17995dfe596bf101a5`, Apache-2.0, 155,908,064-byte weights, SHA-256 `48c14467e5cf9e51870511feb72c89688e82dd74523142c0538b663e193ac2a7`; the 5,695-byte `config.json` is separately pinned to `860aff9751b139d83a4ad7df1e5535416fded533e0ead02625edbefcb9953cce` | Managed install and durable HTTP-job CPU qualification passed on the 512×512 Aquila line-art fixture (10,265 foreground pixels, 3.96s inference, 1.18GiB peak RSS) and 1672×941 laptop plate (787,856 pixels, 0.98s inference, 1.14GiB peak RSS). On an RTX 4070 Ti SUPER, the same laptop mask ran in 0.90s with 455,361,536-byte CUDA peak allocation; CPU/GPU mask IoU was 0.999996. A fresh clean managed install confirmed both per-file hashes and reran the laptop job (0.997s inference, 1.14GiB peak RSS). A 9GiB temporary no-reclaim broker reservation made a GPU-planned job fall back to CPU; job claim and test reservation were both released. Inspected overlays follow the laptop silhouette and exclude the outside negative point. Broader photographic boundary quality remains unqualified; no SAM1 adapter reuse. Evidence: `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/sam2-{cpu,device-cpu,device-gpu,device-contention}-qualification.log` and masks/overlays |
| Larger mask comparison | [SAM 2.1 Base+](https://huggingface.co/facebook/sam2.1-hiera-base-plus/tree/47b9e36ec942f6ff9900555727478a1c96f63225), same Transformers 4.57.6 adapter; Apache-2.0, 323,476,296-byte pinned safetensors (`2012733a…9f780`) | Same laptop image/prompts via managed install and durable job: mask IoU with Tiny 0.9963, pixel disagreement 0.185%, 1.99s CPU inference and 1.35GiB peak RSS versus Tiny 0.98s / 1.14GiB. No meaningful visible boundary uplift in this fixture, so Base+ remains disabled. Evidence: `sam2-baseplus-device-cpu-qualification.log`, `sam2-baseplus-device-overlay.png` |
| Visible-scene geometry | [MoGe-2 ViT-S ONNX](https://github.com/microsoft/MoGe), pinned export revision `e50ffda41565591092adea54c6ac83d6212e1e23`, [MIT checkpoint metadata](https://huggingface.co/Ruicheng/moge-2-vits-normal) | Managed install and durable CPU job now produce full float geometry, camera recovery, normalized intrinsics, and an optional robust support-plane fit. Analytic known-camera tests pass; monocular scene scale remains unmeasured. Evidence: `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/moge-device-table-qualification.json` |
| Device-free background | Existing [LaMa](https://github.com/advimman/lama) quality backend, explicitly selected diffusion repair when needed | Remove superseded device/shadow/reflection; inspect hole quality at final scale and overscan throughout motion |
| Novel props | [TripoSR](https://github.com/VAST-AI-Research/TripoSR) as a permissive initial candidate; compare [SF3D](https://github.com/Stability-AI/stable-fast-3d) if access/license is suitable | Governed dependencies/weights, checkpoint license, mesh/material/UV inspection, hidden-surface limits and measured peak memory |
| Device fitting | Known mesh plus independently supplied landmarks/camera; hinge angle separate from rigid pose | Reprojection and silhouette/occlusion checks; no fabricated automatic landmark detection |
| Physical composition | Device Studio's device/render owners | Screen UV mapping, support plane, beauty/alpha/depth/normal/shadow passes, stable contact and source-continuous handoff in a motion sample |

Upstream TripoSR and SF3D each describe roughly 6 GB VRAM for their default
single-image configurations. Those figures are not local guarantees. No new
heavyweight runtime or weights were installed in this foundation repair. The
observed host has a 16 GB RTX 4070 Ti SUPER and about 64 GB RAM. Scheduling must
use current available capacity and measured stage headroom, not nominal VRAM or
the sum of advertised model sizes.

The geometry-first route now has a bounded native known-device render. The
device-free photograph source and estimated support geometry have been reviewed,
but their separate static composite and bounded-motion proof is still pending;
the photograph does not inherit geometry-first qualification. Automatic
landmark proposals, stronger masks, device-removal repair and prop
reconstruction need their own evidence. Cache stage outputs; inference prepares
assets once and does not run for each animation frame. Keep the accepted Aquila
film until a source-continuous photographic/physical sample is inspected and
clears its own gates.

### Hybrid device scenes: target and current limits

The operator's 2026-10-21 direction extends this capability to separately
controlled scene and device geometry. Choose from the actual starting asset:

| Input | Preparation route | Device motion |
| --- | --- | --- |
| Known camera, support plane and device mesh | Geometry first; create the environment around the controlled layout | Render the known device with the same camera and support plane |
| Device-free photograph | Recover camera/support geometry; fit the known mesh and light it against the plate | Render the device separately; retain background occlusion and contact shadows |
| Photograph containing a device | Prompted segmentation; camera/landmark pose fit including hinge angle; remove device, old shadow and reflection; repair background | Replace with the known mesh, then re-render with shadow/occlusion passes |
| Photograph needing only subtle nearby viewpoint change | Prepared layers or reconstructed visible background geometry, repaired disocclusions | Bounded reprojection; no claim of independent rigid-device rotation |
| New prop without an existing mesh | Optional image-to-object reconstruction, followed by mesh/material review | Qualified geometry only; generated hidden surfaces remain inventions |

Image Tools owns reusable masks, geometry estimation, background repair and
layered reprojection. Device Studio owns device assets, landmark/hinge fitting,
physical rendering, support/contact shadows and composition with live screens.
An orchestration program should retain their immutable outputs and identities;
neither scenario should duplicate the other's inference or job queue.

Prepared-plane parallax and full camera-relative scene geometry are implemented.
Device Studio now exposes a known-camera pose-fit RPC and `scenes
fit-known-camera-pose` CLI command. The operator supplies each named device-local
3D landmark and its observed source-raster pixel center. The fitter resolves the
camera and intrinsics from the exact immutable scene revision and resolves the
device revision through Device Studio's device owner, verifying its candidate
digest and geometry artifact ID/SHA-256. Landmark coordinates are still
operator-supplied in device-local meters; the endpoint records the verified
device and scene references in the result manifest. It estimates only a
unit-scale rigid transform from the supplied initial pose, and returns the
per-landmark reprojection residuals, RMS/maximum error and a vector SVG overlay
with transparent observed/projected markers, ready to stack over the exact source
raster using the manifest's raster dimensions and camera revision. An explicit
hinge joint, angle and application
rule are retained as a fixed assumption; the fitter does not optimize hinge
angle or detect landmarks. Degenerate point geometry, points behind the camera,
and caller threshold violations are rejected. This establishes the callable
fit path and known-camera synthetic recovery, not a photographed-device fit:
there is not yet a manually reviewed photo-to-mesh correspondence set or
silhouette/occlusion qualification. Do not treat the synthetic fixture as photo
evidence. A pinned fixture verifies the immutable Device Studio
`jack-baeten-laptop-neutral-rig-glb` (`laptop-neutral-rig.glb`, SHA-256
`3005e1e8b12e2017565292712aaad20dfb35cda2c42519a937d2c74afb6f5925`) and six
named vertices from its `LaptopScreenSurface` mesh under an analytic camera.
That fixture proves asset-bound projection and fit recovery only; its pixels
are generated from the known camera and are not photograph observations.

Submit a strict `FitKnownCameraPoseRequest` proto-JSON document through the
normal CLI path; retain the full response JSON as the fit manifest:

```bash
device-studio scenes fit-known-camera-pose --request laptop-fit.json --json
```

The request names exact `scene_id`/`scene_revision`/`camera_frame_id` and
`device_id`/`device_revision`/`device_revision_sha256`; its geometry artifact ID
and SHA-256 must match that immutable device revision. `landmarks[]` entries
carry a stable ID, `device_local_m` in canonical meters and source `pixel_x` /
`pixel_y` centers. Set `max_rms_pixels` and `max_point_pixels` to reject a fit
whose observed residual exceeds the review tolerance. The response retains these
references, each observed/projected pair and the overlay SVG.

Automatic scene decomposition, device replacement and image-to-object
reconstruction remain **targets, not qualified capabilities**. A grayscale depth PNG
cannot substitute for floating-point point maps, validity masks or calibrated
intrinsics. SAM regions also require edge inspection; they are not automatically
transparency mattes. Preserve original and repaired plates separately.

Each model stage must use the existing durable jobs and host capacity owner,
persist its output before the next stage, and release model memory before its
reservation ends. CPU fallbacks must actually select CPU execution. Never infer
availability from a catalog row, auto-download weights while rendering, or
advertise estimated VRAM as a measurement. Qualification includes cancellation,
worker exit/reaping, model switches, capacity return, missing dependencies,
license/checkpoint identity, and real visual outputs on the target hardware.

Initial research candidates are SAM 2.1 Tiny (mask quality escalation measured
against Base+), MoGe-2 geometry with camera/normal output, LaMa then an explicitly
selected diffusion repair backend, and existing device meshes. TripoSR is an
optional prop candidate, not a replacement for known device geometry. Upstream
MoGe ONNX exports omit focal/shift recovery and reprojection; integrating the raw
outputs alone would not deliver its full geometry contract. See
[MoGe ONNX notes](https://github.com/microsoft/MoGe/blob/main/docs/onnx.md),
[SAM 2](https://github.com/facebookresearch/sam2), and
[TripoSR](https://github.com/VAST-AI-Research/TripoSR).

## Prepare, render, inspect

1. Start from licensed/local source bytes. Estimate depth through the existing
   `depth_map` operation when useful. Depth Anything V2 Small is the current
   commercial-compatible option; installing its model/runtime is opt-in.
2. Separate foreground/background through the existing segment or mask workflow.
   Keep each layer full canvas with alpha and assign a positive relative distance.
   Keep a device and the active screen attached to the same plane when preparing
   a device composite. A guessed depth map is not measured device geometry.
3. Fill occluded background and required overscan through the existing inpainting
   operation, or provide a separately authored complete background. Inspect edges
   and hidden-region invention. These are explicit preparation operations, not
   hidden AI calls in a deterministic render.
4. Supply a typed `OpParams` document. For a still view use `composite.camera`.
   For a sequence use `parallax.scene`, `startCamera`, `endCamera`, `frames` and
   `framesPerSecond`. `scene.camera` must be absent in a sequence. List plates
   far-to-near by stack `depth`, with decreasing `distance` (0.25..16).
5. Render through the native API/CLI. Use the same plate order in JSON and
   multipart `plate0`, `plate1` parts / repeated `--plate` arguments.
6. Inspect the whole camera path at actual player scale. `requireOpaque` rejects
   uncovered or translucent output, including at intermediate frames. Explicit
   background color is permitted but does not prove reconstructed scenery.

```bash
image-tools ops composite source.png --params-file view.json \
  --plate filled-room.png --plate foreground.png --out view.png
image-tools ops parallax source.png --params-file movement.json \
  --plate filled-room.png --plate foreground.png --operation-id my-parallax-v1 --out movement.zip
```

```json
{"parallax":{"scene":{"width":1280,"height":720,"plates":[
  {"name":"filled room","depth":0,"opacity":1,"distance":3},
  {"name":"foreground","depth":1,"opacity":1,"distance":1}
]},"startCamera":{"z":0.02,"requireOpaque":true},
"endCamera":{"x":0.015,"z":0.04,"yawDegrees":0.4,"requireOpaque":true},
"frames":90,"framesPerSecond":30}}
```

The ZIP contains `frames/000000.png` onward and `manifest.json`: profile,
geometry, fps, smoothstep camera at each output ordinal, exact PNG hashes and
source-layer hashes/distances. It is a master frame sequence, not a silent claim
of MP4 or alpha-video support. Pass those frames to the film's existing managed
composition/encoder. No extra native video daemon or private FFmpeg worker is
introduced by this image operation.

## Bounds and failure policy

The nearby-view camera permits x/y/z translation of at most0.08 and yaw/pitch of
at most5°. Focal length defaults to canvas width and may be0.5..4 times that
width. All values must be finite. Planes, camera order, alpha and missing coverage
are validated. Identity camera preserves the source view; bilinear sampling uses
premultiplied alpha to avoid colored fringes. Motion uses smoothstep over output
ordinals and includes both exact endpoints. A sequence is2..180 frames at24/30fps,
at most240 million raster pixels, and at most128MiB of ZIP bytes. At most8 layers,96MiB of encoded layer bytes and24 million decoded source
pixels are admitted. Cancellation is checked during projection and between
frames. `parallax` submits to the existing durable CPU job lane with a180-second
execution deadline. Multipart submission requires an `Idempotency-Key`; the CLI
uses `--operation-id`. A retry with the same bytes returns the same job; changed
inputs conflict. The source/layers are retained before queue acceptance. The CLI
prints the job identity, waits once through JobsService, verifies the output hash
and downloads the ZIP. A lost wait does not cancel the job. Recover through
`jobs get`, `jobs wait` and `jobs download`. No private queue or render daemon is
added. The existing long-wait transport preserves waits beyond ordinary HTTP
write timeouts.

## Research and choice, 2026-10-21

[3D Photography using Context-aware Layered Depth Inpainting](https://shihmengli.github.io/3D-Photo-Inpainting/)
uses layered depth plus synthesized color/depth behind occlusions, then renders
nearby views with motion parallax. This explains why depth estimation alone was
insufficient for the requested effect.

[DepthFlow](https://github.com/BrokenSource/DepthFlow) is a purpose-built
image-to-parallax-video implementation with depth-driven ray marching. Its
repository identifies AGPL-3.0; no third-party code or runtime was installed for
this change. The native prepared-plane implementation has a smaller, explicit
geometry model and does not claim equivalent automatic reconstruction or speed.

[Depth Anything V2](https://github.com/DepthAnything/Depth-Anything-V2) supplies
depth estimation, not disocclusion repair or final camera motion. Its Small
weights are Apache-2.0; larger weights have different terms. Existing Image Tools
model governance remains authoritative.

[SHARP](https://github.com/apple-aiml-research/ml-sharp) reconstructs a Gaussian
representation for nearby views. That is a different, heavier reconstruction
route with its own model license and renderer requirements; it was researched,
not installed or claimed as a native capability here.

Use true physical geometry for hinge motion, large orbits or newly visible sides.
Prepared 2.5D layers are useful for subtle translations/rotations of photographic
scenes. They cannot establish exact hidden geometry or realistic articulated
hardware.
