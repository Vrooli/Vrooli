# Scenes, motion, shot compilation and output

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

Motion has separate device root, joint and camera channels with explicit easing/interpolation and time ranges. Bound tracks to supported targets/ranges. Repeated playback does not accumulate transforms. Preview and native worker evaluate the same channel/time semantics; their lighting is visibly different until separately matched. Camera screen-entry ends on the active app surface and transitions to the actual capture, preserving crop/time continuity. It must not merely enlarge the whole device and stop on a bezel.

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

Output access resolves an opaque job/output identity under its project and requires verified completion. Read the immutable output through the routed blob owner, check stored byte count/digest and image metadata, and return native-size PNG with a safe download filename. Return actual dimensions; no source or placeholder image may be substituted when output is missing. Private execution and cleanup receipts remain available for internal validation but are not arbitrary downloadable filesystem paths.

### Private still-worker contract

Use a fixed `render-worker --manifest <path>` mode in the shipped Go API executable for the first durable PNG job. It is a child operation launched by the shared platform owner, not a second API service or scheduler. Recognize this exact mode before development service preflight/database/listener setup, so a pinned job never silently rebuilds its worker. Normal service invocation retains its existing lifecycle path. Unknown worker flags fail closed; there is no command-string, script or user-supplied codec option.

The private versioned manifest contains the job/attempt/input fingerprint, compositor profile, project-resolved image bindings and calibration. Bindings carry relative workspace names, SHA-256 and byte counts. The parent stages verified immutable input bytes in an attempt-owned directory under routed storage; the child has no database or fleet dependency and never resolves media IDs or developer home paths. Require bounded manifest/image sizes, regular files, path containment and no symlink traversal. The manifest pins the complete composition inputs; verify every digest before decoding and before publishing an output. Separate algorithm correctness from filesystem/network sandboxing: this data-only worker executes shipped Go code, and supplies no arbitrary-code sandbox guarantee.

Write fixed output names in that attempt workspace using exclusive creation and atomic publication, without overwriting prior attempts. Retain a machine-readable receipt binding job/attempt/input identity, input byte digests, algorithm/profile, image dimensions, protected-pixel counts, output PNG digest and size. Successful worker exit is necessary but insufficient: the parent checks exact identity, full output decode and digest plus native-tree quiescence before persisting job success, then explicitly releases the retained process-manager unit. On failure preserve a bounded diagnostic and no success receipt. Workspaces and eventual blob publication remain the job adapter's responsibility; the worker must not invent a private retention policy.
