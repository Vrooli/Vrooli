# Scenes, motion, shot compilation and output

Hybrid photographic scenes follow the source-dependent preparation contract in
`path:scenarios/image-tools/docs/reference/layered-photography.md`. Known-camera
geometry-first scenes, device-free plates and photographed-device replacement
are distinct routes. Image Tools owns reusable inference/repair/reprojection;
Device Studio owns known-device pose/hinge fitting, physical screen projection,
contact shadows and composition. A qualified mask or prepared parallax sequence
does not establish a calibrated physical scene. Qualify a short source-matched
render and screen handoff before replacing an accepted film's device shot.

## Scope

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

A shot is the combining type. Scene/device/motion/presentation libraries are independently selectable, importable and authorable; the compiler accepts only compatible combinations.

## Strategy eligibility

| Requested behavior | Eligible starting route | Required proof |
|---|---|---|
| Fixed device/camera, changing screen, bounded crop push | Pinned photographic plate | Calibration/masks, pixel footprint and approved appearance |
| Exact hardware geometry or controlled placement | Qualified 3D | Dimensions, materials, support and actual renderer look |
| Phone spin, laptop hinge, physical parallax | Qualified 3D | Parts/tracks, continuous interactions and temporal quality |
| Moving 3D device on a fixed photographed background | Optional hybrid research | Camera/light match, shadow/reflection catchers and occlusion over full motion |

A manual strategy override cannot waive incompatibility. Explain the violating motion/surface requirement and show valid alternatives. A still can need 3D precision; a video can use a fixed plate. The output container does not decide strategy.

## Scene and motion model

Scene slots expose measured support plane/mesh, normal, local frame, mount interfaces, exclusion zones and clearance. Device orientation/scale is physical, not fitted to a silhouette. Props, materials, texture scale, lights, camera/lens, focus and depth of field are independent and versioned. Art direction favors restrained realistic surfaces and coherent illumination; primitive props and more samples do not meet the photographic benchmark.

Motion has separate device root, joint and camera channels with explicit easing/interpolation and time ranges. Bound tracks to supported targets/ranges. Repeated playback does not accumulate transforms. Preview and native worker evaluate the same channel/time semantics; their lighting is visibly different until separately matched. Camera screen-entry ends on the active app surface and transitions to the actual capture, preserving crop/time continuity. It must not merely enlarge the whole device and stop on a bezel. The direct-app transition surface must contain source pixels and the presentation-owned flat letterbox fill only. Browser window outlines, rounded corners, toolbar, dock and phone system chrome must not leak into that surface; preserve the source mapping rather than cropping away source pixels to hide chrome.

### Photographic editorial handoff destination

A photographic screen-entry camera may name an optional destination rectangle in normalized output coordinates (x/y/width/height in integer basis 10000, positive size, entirely inside output). Omitting it preserves full-output contain. The renderer contains the entire active app viewport within that rectangle, without stretch or source crop, and fades the surrounding device/chrome to the pinned matte. The saved revision and worker manifest pin the destination. Output orientation changes resolve the same normalized intent against that output raster. This is a bounded photographic camera endpoint, not arbitrary moving hardware or transparent output.

New photographic app-only plain video revisions use the existing v2 app-raster presentation adapter, matching the video worker. Plain still revisions retain their historical v1 semantics. A historical v1 plain video must be saved as a new revision before rendering; compilation must not silently reinterpret its saved identity.

The following film shot must use the same source, source-time continuation, aspect, matte and computed contained viewport bounds. A destination rectangle alone does not prove editorial continuity. Clients may introduce surrounding titles/background after the screen reaches its endpoint; they must not replace the screen with another take at that join.

## Compilation and revision

Validate source availability/digests/rights, physical state, display masks/UVs, scene support, attachment compatibility, swept clearance, motion capabilities, renderer features, resolution/color, project privacy and total budget. Compilation is not a lasting authorization grant; dispatch/resume/children recheck current consent, account, scope and aggregate limits per [flows](../concepts/FLOWS.md). Compilation returns an immutable execution description and typed issues with repair paths. A changed dependency/profile/tool invalidates affected cached compile/review/render evidence. A saved draft may be incomplete, but render admission cannot silently supply unknown physical measurements or replace unavailable assets.

The compiled revision pins the frame schedule, all source revisions, renderer/tool/profile version, assets, camera/lights/tracks, screen/audio policies and output settings. Cache keys include those inputs plus transformation/code versions. Cache reuse does not repeat a provider charge. Record whether a cache hit or fresh execution produced each artifact.

## Worker execution and capability discovery

Probe actual worker capabilities in order: hardware detected, backend compiled, device enumerated, capacity admitted, representative render qualified. `auto` chooses a qualified local profile only. Explicit unavailable GPU selection produces a reason and an explicit compatible CPU option; it never switches to paid cloud. Pin native renderer pack and included Python/kernels/dependencies. Do not copy Linux ambient Python paths into desktop workers.

Use owned processes, bounded memory/tasks/wall time and explicit workspace paths. No raw package-manager installation from the scenario. Generated Python and imported files need a separately validated filesystem/network isolation boundary. Limits alone do not supply it. Capture stdout/stderr safely, checkpoint completed frame batches, write output atomically, validate full decode and retain terminal cleanup receipt. A UI disconnect does not destroy the job.

## Output and visual qualification

Before a long render, approve a final-renderer look frame and a short continuous sample at the promised profile. Inspect specular changes, temporal aliasing/flicker, joints, contacts, seams and screen legibility. Current v5 scene finish is a negative benchmark; v6 establishes photographic direction but supplies no 3D craft certification. Record reviewer, exact inputs, scope, observations and limitations. Higher samples or successful MP4 decoding cannot substitute for this judgment.

Export direct PNG/MP4 plus receipt, optional qualified GLB, and native project package when permitted. Provide a poster/contact sheet and inspectable source recipe without requiring an HTML viewer. Bit-identical cross-GPU path tracing is not promised; deterministic timeline/inputs and declared visual tolerance are the portability contract.

## Cross-References

[presentations](presentations.md), [devices](devices.md), [flows](../concepts/FLOWS.md), [performance](../internal/PERFORMANCE.md), [deployment](../operations/DEPLOYMENT.md).

## First photographic implementation seam

The approved implementation uses a small domain-specific Go compositor over resolved images, with no ambient OpenCV/Python dependency. The algorithm owns calibrated screen projection and protected hardware pixels; shared blob storage owns bytes, the job domain owns persistence, and platform/tool owners own native decoding/encoding execution. It does not become a generic image editor or process supervisor.

Quad points are ordered top-left, top-right, bottom-right, bottom-left in the declared plate raster space. The implementation must document pixel-boundary versus center conventions, reject nonfinite/degenerate/self-crossing transforms and bound image/sample complexity. Contain preserves source aspect; any supported crop needs explicit intent. Alpha sampling is premultiplied to prevent dark fringes. Zero-active-mask and hardware-occluded pixels retain the original plate exactly before lossy encode.

Output timing is rational. Source frame selection uses recorded presentation timestamps and a declared trim/rate/EOF policy; decoder frame indices alone do not prove source time. Preview and final output use the same schedule. This core initially accepts already composed authentic screen layers; configurable OS/browser chrome and decoder/job integration remain separate implementation obligations.

Retained v6 plates are valid private algorithm/reference fixtures but do not meet the final high-resolution still detail requirement. New higher-resolution source/native outputs are required; upscaling cannot be passed off as recovered detail. Bounded compositor tests and historical-sample replay do not constitute a usable saved-shot/render application or visual approval.

### First saved photographic shot

The first saved-shot implementation combines project-owned source media with a photographic scene, presentation and fixed motion. `shots` owns shot identity, head revision and immutable revisions through its own schema/repository; `projects` continues owning media. The public contract uses typed messages rather than arbitrary recipe JSON. Create establishes revision 1. Editing requires the expected head revision and atomically appends a new revision; stale edits return a typed conflict. Old revisions remain addressable and executable. Server-generated recipe hashes include all validated values and dependency digests, not client assertions or display names alone.

The photographic scene component pins plate and active-mask media IDs, an optional hardware-mask media ID, dimensions, calibrated quad, surface size and a versioned coordinate convention. All bytes must belong to the same project and be reverified when resolved. Mask encodings/dimensions and calibration follow the composition core. A depiction label describes the photographic device; it does not claim exact qualified hardware or usable physical joints. Initial fixed photographic motion rejects rotation, hinge and parallax with an explicit compatible-route issue. Native 3D will be a distinct typed strategy, not an ignored field on this recipe.

The presentation component pins source media, app-only versus already-composed capture, fit, viewport and any simulated chrome parameters. Initial app-only/contain and already-composed-screen modes must remain distinguishable. Saved values identify the exact presentation implementation version. Scene and presentation component values are embedded immutable snapshots with independent canonical digests; later library selection resolves a reusable component revision to the same snapshot shape. This avoids needing an entire asset-library service before one saved shot, without making mutable library pointers part of replay identity.

Initial still output uses the plate's native raster size and a pinned PNG encoder/compositor profile. Preview may display a scaled copy, but the API returns actual output dimensions and source resolution; it never claims the release still threshold for the historical small plates. The compiler returns actionable issues before dispatch and verifies hashes again at execution. Public content/export routes resolve opaque project/shot/revision/output IDs through their owners. Neither raw paths nor a caller-supplied output digest can authorize output publication.

The durable job owner remains the execution authority. A still worker can use the Go compositor without FFmpeg or Blender. Native video adds qualified decoder/encoder profiles later; this does not allow a synchronous request to masquerade as a durable job, nor a source capture to masquerade as a final render. Execution integration must pin its executable/profile, persist intent before dispatch, verify output before completion and retain cancellation/recovery evidence. The Linux platform operation owner has passed focused and parent review; product integration and non-Linux adapters remain pending.

### Public still jobs and lifecycle progress

Expose typed submit/get/list/cancel through the public API and CLI. Submit accepts an owning project, saved shot/revision, bounded idempotency key and qualified output profile; it delegates to the existing render coordinator. It cannot supply an executable, raw path, arbitrary recipe or caller-authoritative digest. Get and project-scoped paginated list are read-only. Public status distinguishes queued work, observed execution, unresolved operation outcome, cancellation requested, and verified terminal outcome. Return bounded user-facing failure/recovery information without private workspace paths or command details.

A lifecycle-owned bounded progress loop advances the existing jobs ledger without requiring browser polling. Persist release-pending work on execution claims and clear it only after the exact operation is released (or a previously terminal claim is confirmed absent). Include this work in bounded restart discovery, so a crash after terminal commit cannot leak an undiscoverable retained operation. The jobs domain owns this bookkeeping; do not add a competing queue. Stop the progress loop before closing its database. Stopping the API does not imply that native jobs have stopped or were cancelled; startup reconciles persisted exact identities.

Primary progress starts with the API lifecycle. Test progress requires an installed test route and explicit submission/cancellation under that route. Each step rechecks the original route binding; browser disconnect does not remove it, while expired/missing routing cannot fall back to primary storage. A service restart does not imply restoration of a prior test lease. Validate restart semantics separately from leased interaction evidence.

Background test operations must bind the exact database pool and expected file-root lease through the shared routing owners. A boolean test-mode flag or lease check followed by a fresh route lookup is insufficient: lease replacement between checking and querying must never access the replacement pool or primary storage. Expiry, removal, same-ID pool replacement, and a mismatched DB/file lease fail closed. Do not restore expired leases or implement a private routing layer. Preserve existing unbound caller semantics; the stronger bound operation is explicit. Validate with deterministic replacement barriers at owner and pump boundaries.

Output access resolves an opaque job/output identity under its project and requires verified completion. Read the immutable output through the routed blob owner, check stored byte count/digest and image metadata, and return native-size PNG with a safe download filename. Return actual dimensions; no source or placeholder image may be substituted when output is missing. Private execution and cleanup receipts remain available for internal validation but are not arbitrary downloadable filesystem paths.

### Private still-worker contract

Use a fixed `render-worker --manifest <path>` mode in the shipped Go API executable for the first durable PNG job. It is a child operation launched by the shared platform owner, not a second API service or scheduler. Recognize this exact mode before development service preflight/database/listener setup, so a pinned job never silently rebuilds its worker. Normal service invocation retains its existing lifecycle path. Unknown worker flags fail closed; there is no command-string, script or user-supplied codec option.

The private versioned manifest contains the job/attempt/input fingerprint, compositor profile, project-resolved image bindings and calibration. Bindings carry relative workspace names, SHA-256 and byte counts. The parent stages verified immutable input bytes in an attempt-owned directory under routed storage; the child has no database or fleet dependency and never resolves media IDs or developer home paths. Require bounded manifest/image sizes, regular files, path containment and no symlink traversal. The manifest pins the complete composition inputs; verify every digest before decoding and before publishing an output. Separate algorithm correctness from filesystem/network sandboxing: this data-only worker executes shipped Go code, and supplies no arbitrary-code sandbox guarantee.

Write fixed output names in that attempt workspace using exclusive creation and atomic publication, without overwriting prior attempts. Retain a machine-readable receipt binding job/attempt/input identity, input byte digests, algorithm/profile, image dimensions, protected-pixel counts, output PNG digest and size. Successful worker exit is necessary but insufficient: the parent checks exact identity, full output decode and digest plus native-tree quiescence before persisting job success, then explicitly releases the retained process-manager unit. On failure preserve a bounded diagnostic and no success receipt. Workspaces and eventual blob publication remain the job adapter's responsibility; the worker must not invent a private retention policy.

### Paired video tool profile — selected E2V contract

The control-plane host/tool owner exposes the fixed read-only command `vrooli host renderer-profile ffmpeg-video --json`. It observes a governed pack; it does not install a tool, accept user argv or search ambient PATH for a substitute. A versioned cli/v1 artifact identifies platform/architecture, the archive and member digests/sizes, exact FFmpeg/FFprobe builds, explicit runtime prerequisites and resolved local executable paths. Keep content identity independent of install-directory names and observation time so a packaged relocation can retain the same artifact identity. Environment qualification has its own identity when OS/runtime prerequisites differ. Current workflow qualification remains separate from artifact presence or a successful version probe.

A dedicated `ffmpeg-video` managed tool declaration may stage the inspected paired archive through existing hostreq/binaryfetch; it must not replace the generic host FFmpeg command used by other scenarios. The resolver verifies both declared regular executable members and rejects drift, partial pairs, unavailable prerequisites and unsupported platforms. Known hash-verified binaries may receive bounded fixed version/config probes. No installing or probing unknown caller-selected executables from this read operation. Application workers consume explicit verified paths through the portable profile contract; packaged desktop resolves the same relative artifact set from bundle roots without control-plane imports or a monorepo dependency.

The first candidate is the exact dated BtbN9.0 codec-static Linux pack in the E2V inspection receipt. Actual ELF imports and loader still require an OS baseline; codec-static is not hermetic. Preserve explicit prerequisite/qualification limits and included license bytes, and keep distribution qualification separate. Do not clone an OS library tree or implement a private dynamic loader to satisfy the profile. Unknown prerequisites or an unqualified render path remain explicit.


### Fixed photographic video — implementation contract

One versioned video worker shall produce both a trim-start PNG sample and a
final MP4 from the same saved source schedule, presentation layout and plate
composition. The existing jobs coordinator and platform-owned operation shall
own dispatch, cancellation, reconciliation and release. No request-path decoder,
second timeline, queue, or process supervisor is introduced.

The immutable closure shall bind source bytes, accepted inspection receipt and
clock, source timebase/EOF, rational trim/playback/output rate, plate/masks/quad,
presentation, renderer/tool identities and output mode. Full sequential decoding
shall deliver fixed-size RGB24 frames through a bounded reusable buffer; a
concurrent bounded stats stream shall match every ordinal and PTS to the saved
clock before an output is accepted. `composition.PlanFrames` remains the schedule
authority. Raw frames shall not be staged as an entire video on disk.

The initial profiles use1920×1080 or1080×1920 at24/30fps. A plate shall have the
exact target aspect and at least target dimensions; one uniform transform applies
to the complete composed image. Unsupported plates receive a typed issue. Existing
lower-resolution still recipes retain their current output semantics.

Decode shall preserve observed matrix/range tags and use the named BT.709 defaults
only for missing primaries/transfer, then convert to full-range RGB with sRGB
transfer for the image compositor. Encode shall explicitly convert that raster to
BT.709 limited-range H.264/yuv420p. Unknown/HDR policies remain unsupported. The
selected tool build and fixed filter graph are part of the versioned profile;
qualification evidence is under `goal/evidence/E2V/video-protocol`. Export-quality
preset and clip duration require a complete composition/encode/verification
benchmark; the synthetic ultrafast/CRF24 experiment is not export quality evidence.

AUTO fills and status/home inks shall be resolved from the exact trim-start frame
by a sample job, then accepted by an explicit save of a new immutable revision.
The sample receipt binds the base recipe and resolved values; final submission
pins that verified receipt. Manual fills may bypass sampling. Sample and final
shall use identical pre-encode pixels at output frame zero. The canonical pixel
digest covers raster dimensions, pixel/color profile and row-major RGB bytes;
job, revision and layout identities are separate receipt bindings. Lossy output
comparison shall decode back into the same sRGB-transfer space first.
The same trim-start sample operation also supports explicit manual and browser
layouts; these reuse their pinned layout on the same revision and claim no
sample-derived color provenance. Only AUTO phone sampling resolves fills/inks
and requires saving a new immutable revision before final render.

The worker shall stream encoded fragmented MP4 through a limiting writer with a
160MiB ceiling and verify final metadata, frame count, timestamps, full decode,
size and digest before publication. Attempt-owned writes shall be bounded by the
sum of source512MiB, staged worker256MiB, scene inputs64MiB, bounded manifest/
receipt/clock sidecars and output160MiB. This is a bound on owned writes, not a
filesystem quota. Memory2GiB, tasks128 and wall300seconds remain enforced by the
existing operation owner. The output-frame limit is separate from accepted source
duration; it shall be measured with margin before public final-render admission.
Audio is off in the first profile; explicit passthrough remains an open obligation.

A trim-start PNG preview is available for every supported presentation, including
browser/desktop and explicit manual phone settings. Only source-derived phone
colors require accepting values into a new revision. A manual/browser preview
uses its already pinned layout and does not invent sampled-color provenance.

#### Publication and sampled-color integrity

Video publication shall stream through the existing routed immutable artifact
store with an independent count and SHA-256 check, without materializing the
whole MP4 in API memory. A successful terminal job commit makes the artifact
visible. A crash after writing the artifact but before committing the job leaves
an unreferenced immutable artifact; reconciliation may reuse a verified exact
artifact or retention may collect it only after proving it has no references.
The budget shall include the workspace output and its durable published copy
(two copies of at most160MiB), plus verification buffers and bounded sidecars.

The sample's base-recipe canonicalization shall have an explicit version and
cover all recipe fields except the sample claim itself. Saving a sampled revision
shall verify the complete base recipe against the trusted sample receipt and
require stored fills and inks to match its resolved values. Explicit manual
colors are a separate policy and shall not carry a claim of sample provenance.
Old still recipes without ink overrides shall retain their serialized identities
and rendered behavior. Independent status/home ink overrides are profile-owned;
they shall not silently change older presentation versions.

#### Recording intake in the workbench

The media panel shall support recordings alongside images and display server-owned
inspection readiness, typed repair guidance, and explicit retry/cancel actions.
Only ready recordings may enter the shot compiler. Inspection job refreshes shall
be reads; importing and explicit Ensure/retry are mutations. Unknown outcomes
shall not become an automatic new attempt.

Original recording playback shall use the existing routed content URL through
the browser media element, without materializing the entire recording as a JS
Blob or array buffer. This is an original-source preview, not proof of the exact
render schedule or successful whole-file digest verification. The current full
response can require complete download before metadata/seeking works; the UI
shall explain buffering and provide a playback failure state. Accurate trim and
sample/export verification use saved clock facts, not the media element clock.
Reliable byte-range seeking remains a separate delivery obligation; it shall not
weaken existing full-stream integrity or introduce product test-mode tokens.

The first fixed photographic video profile selects x264 medium/CRF18 with
B-frames disabled (`-bf 0`) and explicit two codec/filter threads. The bounded
synthetic experiment in `goal/evidence/E2V/video-protocol/medium-timing` found the
default fragmented MP4 started at1024ticks of1/12288 (two24fps frames), while the
same command with B-frames disabled started at0. Native output verification shall
observe actual zero-based timestamps, not normalize away an unreported offset.
This codec experiment does not qualify whole-render speed or photographic craft.

#### Recording workbench controls

The source player assists navigation; saved trim boundaries come from paged,
digest-bound inspected frame clocks and retain rational timestamps. The shot
editor invalidates prior sample acceptance on source, timing, scene or presentation
edits. Returning to AUTO clears only colors marked source-derived by the previous
sample; explicit manual choices remain. Late responses and stale clock pages cannot
replace the current draft. Final submission targets the displayed saved revision.
The CLI streams exports into bounded temporary files and publishes a destination
only after exact size/digest and format checks, without overwriting an existing
file. Browser MP4 preview uses a job-scoped content URL rather than a video-sized
JavaScript Blob. PNG previews retain pixel/metadata verification.

#### Reusable physical shot setups

The shot domain owns immutable physical setup presets as typed joins over exact
device, scene, and motion candidate revisions. A preset stores only the three
revision references and their canonical digests, selected display/support/
camera IDs, explicit placement transforms, backend choice, and a finite list
of supported still and motion output profiles. It does not copy candidate
definitions or assets, and it never stores caller-asserted qualification.
Saving a preset resolves every exact revision, checks the selected semantic
bindings and placement against those candidates, and checks current `UseRender`
rights for every referenced artifact through the library owner. A stale or
revoked reference makes the preset unusable until the user saves a new preset
revision against valid component revisions and rights.

Preset revisions use expected-head compare-and-swap and remain immutable. Get,
head, and bounded revision-list operations are read-only. Candidate consistency,
asset rights, renderer compatibility, and visual qualification are separate
facts: the initial preset response reports candidate-only status and never
promotes a combination because it was saved or rendered. The existing reviewed
phone-study setup remains an explicit starter preset with its verified
placement; other combinations are selectable only when a matching saved setup
exists. The editor may retain an incompatible local draft, but it must explain
which setup is missing and keep save/compile/render disabled instead of
inventing a placement or reusing a different device's transforms.

Every new physical setup selects an immutable scene support-slot ID in addition
to its device and scene support-frame IDs. The slot declares the allowed support
kind/interface and fit constraints; setup validation resolves it from the exact
scene revision and calls the canonical device support, pose, and motion
evaluators. Validation and compile responses expose a typed compatibility result
(`compatible`, `incompatible`, or `needs_review`) with bounded reasons. A known
contradiction is rejected and never render-eligible. Unknown envelope, mount,
pose, orientation, mask, or fit facts remain `needs_review` and cannot be
promoted by saving or rendering. Historical `scene-candidate/v1` slot-less revisions use an explicit
legacy interface-only result of `needs_review`; their immutable bytes and prior
review scopes remain unchanged, and the compatibility result does not widen those
reviews.

## Photographic screen-entry implementation contract

The bounded screen-entry route uses `MOTION_MODE_CROP_PUSH` with a typed, immutable `PhotographicCameraMotion` on the shot snapshot, separate from source `VideoTiming`. Its first profile is `screen-entry-projective-contain-v1`: profile version1, transition duration in output frames (2..120, no longer than the clip), and explicit camera matte RGB. The initial camera profile limits the complete output clip to120 frames (five seconds at24fps or four seconds at30fps), separately from the fixed profile’s300-frame limit, until representative runtime is measured. Reject longer camera clips before render with trim/rate guidance. Fixed shots forbid camera config; crop-push requires it. Unsupported rotation/hinge/parallax remain rejected by the photographic renderer. This route changes framing and rectifies photographed screen perspective; it does not simulate physical device motion.

Use a distinct photographic screen-entry video profile. Preserve existing fixed manifests, canonical recipe/timing digests and replay bytes when the optional camera field is absent. Pin camera configuration in the shot recipe, closure, manifest and sample verification. Changing camera values invalidates the accepted sample. `composition.PlanFrames` remains the sole source-time authority. Camera progress follows output ordinal, including when two output frames select the same source frame.

Derive the active app quad using the exact surface contain transform and homography used by the compositor. The endpoint is a centered rectangle containing `presentations.Layout.AppViewport` at uniform scale in the output raster. Preserve its aspect and existing presentation letterbox; fill remaining output area with the pinned matte. A portrait app in landscape output therefore has side bars. Never independently scale width and height or silently crop content to fill the frame. Any future cover/crop profile requires explicit crop semantics.

Interpolate corresponding viewport corners from their initial output positions to the endpoint rectangle using profile-pinned smoothstep over output frames0..N−1. Validate finite/convex/nondegenerate geometry for the bounded full path before dispatch; reject folding or singular paths with a repair explanation. Frame0 is an exact identity operation on the existing fixed composite. For intermediate frames, inverse-warp the composed scene using the corresponding projective transform. During the final quarter, smoothly fade scenery outside the moving viewport to matte and blend inside it toward the direct presented AppViewport from the same scheduled source frame. Antialias viewport boundaries. At and after the endpoint, render the direct AppViewport into the contained rectangle with matte outside: no bezel, scene or hardware occlusion pixels may remain. Never obtain the endpoint solely by magnifying the previously projected plate.

One queue, native execution owner, decoder and output verifier remain authoritative. Composition owns the shared projection/warp math; rendering owns ordinal evaluation and direct handoff; shots owns validation/immutable recipe; UI exposes duration/matte and explains containment. Required evidence includes fixed-byte compatibility, frame-zero sample parity, aspect/matte/hardware-free endpoint, perspective and singular-path cases, continuous source PTS including repeated VFR selections, actual output decode, cancellation/resource bounds and observed added runtime. Fixed-video run6 evidence remains valid only for its unchanged profile.


### Stationary photographic screen entry

Operator-approved extension (Aquila craft feedback,2026-10-21): `screen-entry-stationary-scene-v1` is an editorial reveal. Preserve the photograph's orientation and pixels while its screen lifts into `target_rect`; fade the stationary photograph to matte. An optional `scene_rect` contains the full photograph in a separate normalized output rectangle. Default scene placement is full-output contain. Derive the initial active screen projection from that exact contain placement. The entry begins with the native fixed composite and ends with the direct full viewport at the requested destination; source PTS must remain continuous. Do not label this as physical camera movement, depth parallax or laptop articulation. Keep legacy projective-contain saved recipes unchanged; reject scene placement on that legacy profile rather than ignoring it.

The destination is local to the following shot. A side destination permits large text alongside it; it does not require all later shots to use that placement. Review the room for warping and the screen for source-time/geometry continuity in the actual encoded handoff.


### Video source color at a screen handoff

The decoded active viewport is sRGB RGB24. Decode inspected SDR YUV with explicit
matrix, range, primaries and transfer through zscale, requesting planar GBR
before packing RGB24. Do not let a later implicit YUV-to-packed-RGB conversion
reinterpret a full-range planar result: it lifts dark UI values. A dark-sRGB
encode/decode regression exercises the actual FFmpeg filter and bounds the
round-trip error. The worker digest pins this repair; retained old artifacts are
not rewritten. Camera geometry and source clocks are unchanged by this fix.
