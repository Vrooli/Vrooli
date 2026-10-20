# Device Studio renderer integration evidence

Investigation handoff, 2026-10-19. This is a bounded implementation design for the E2 photographic compositor and E3 native renderer. It records source and host evidence; it does not claim either renderer is implemented or qualified in Device Studio.

## Findings

Device Studio is currently a generated scaffold. Its API wires the example `notes` domain, and its docs explicitly say the product domains are designed but not implemented (`docs/internal/SEAMS.md`, `api/internal/modules/registry.go`). Product contracts already live in `docs/domains/rendering.md`, `docs/domains/presentations.md`, `docs/concepts/DATA.md`, `docs/internal/SECURITY.md`, and `docs/internal/PERFORMANCE.md`; implementation should realize those contracts rather than add another competing domain model.

The v6 archive contains two successful, but standalone, compositor experiments: `v6/phone/compose.py` and `v6/laptop/compose.py`. The phone path has a pinned plate, manually calibrated source-to-plate homography, an active-screen mask distinct from hardware/pill pixels, 2x supersampling, premultiplied-alpha resolve, and a pre-encode exact outside-mask pixel check. It renders a four-second, 96-frame 24 fps sequence from retained source frames and encodes H.264. The laptop path uses a pinned quad and uniform `contain` transform, preserving its photographic plate outside the screen. The phone's `source-clock.json` binds each output index to the actual retained capture digest and source time. These scripts are evidence and algorithm references only: paths, inputs, dependency versions, process ownership, job persistence, and UI/API integration are experiment-specific. The phone uses OpenCV and NumPy plus host FFmpeg; the recipe does not support moving device/camera, parallax, or authentic operating-system chrome.

The current v8 host inventory reports Blender 4.0.2 and FFmpeg 6.1.1. The host has an RTX 4070 Ti SUPER and NVIDIA driver 580.173.02, but that does not qualify Blender GPU rendering. The read-only Blender probe reports CUDA compiled in, OptiX unavailable, and only CPU devices enumerated for Cycles CUDA/OptiX; the installed Blender data package has no `.cubin`, `.ptx`, or `.fatbin` kernels and `nvcc` is absent. GPU capacity/render performance is therefore unqualified. Blender CPU is available as a research route, but no Device Studio scene or render is certified. The retained v8 reports are `v8/research/desktop-rendering-evidence/toolchain-probe.json` and `gpu-probe.json`; the scripts disclose their exact read-only scope. FFmpeg's observed host build enables GPL and libx264; do not ship/copy it as an assumed permissive or portable dependency.

## E2 contract and pipeline

The first complete render should accept one immutable `ShotRevision` and one `JobAttempt`. The shot pins source artifact ID and digest, presentation/profile revision, plate ID and digest, source clock, screen homography, active display mask and hardware occlusion mask, output frame rate/dimensions/color profile, chrome configuration, compositor version, and encoder profile. The attempt pins the resolved tool profile and exact closure digest, plus durable state and checkpoint/output receipts. Existing `DATA.md` already defines source media timestamps, presentation profiles, shot dependency pinning, and jobs/attempts; use those owners and add only the fields needed to support this contract.

A compact presentation mapping is:

```json
{
  "kind": "photographic_plate",
  "plate": {"artifact_id": "…", "sha256": "…", "width": 1672, "height": 941},
  "screen": {
    "source_size": [1920, 1243],
    "source_to_plate_homography": [[1,0,0],[0,1,0],[0,0,1]],
    "fit": "contain",
    "active_mask_artifact_id": "…",
    "hardware_occlusion_mask_artifact_id": "…",
    "mask_space": "plate_pixels"
  },
  "clock": {
    "source_kind": "video_pts",
    "time_base": [1, 90000],
    "trim_start_pts": 0,
    "trim_end_pts": 360000,
    "playback_rate": [1, 1],
    "out_of_range": "hold_last",
    "output_rate": [24, 1]
  },
  "chrome": {
    "mode": "phone_status",
    "clock_text": "9:41",
    "clock_policy": "fixed",
    "signal": "full",
    "wifi": "on",
    "battery_percent": 84,
    "theme": "light",
    "home_indicator": true,
    "edge_background": {"mode": "sample_fixed", "source": "app_top_edge"}
  }
}
```

Values are illustrative; all geometry, colors and clocks are persisted with explicit units/space and validated by the compiler. A video source uses rational time base and presentation timestamps, not frame-index-as-time; image/sequences use a separately typed clock. The phone status background is its own layer extending under the Dynamic Island, while the physical pill/camera remains an occlusion. Validate that masks are nonempty, bounded, correctly sized, nonoverlapping where forbidden, and bound to the pinned plate/source dimensions. Preserve fit/crop decisions without stretching app pixels.

The worker pipeline is: resolve and digest-check pinned blobs; validate/compile the immutable shot; decode the source by PTS into a bounded immutable frame cache or use a pinned deterministic decoder; evaluate source frame and chrome state at each rational output time; fit/warp with the pinned homography; composite with premultiplied alpha and supersampling; restore all pixels outside the active screen mask from the plate; optionally add profile chrome only in the declared app/system regions; write each frame batch to staging with its schedule and digest; encode PNG directly or pipe frames to the resolved FFmpeg binary for MP4; fully decode and inspect output metadata; atomically publish blob plus receipt. PNG still and short continuous MP4 are E2 acceptance. For photo shots, unchanged plate pixels should be exactly verified before lossy encoding. For MP4, only decoded visual checks and source/screen timing claims apply; H.264 is lossy.

Status chrome can be generated as a vector/SVG or deterministic raster layer without another installed native tool, provided fonts/assets are explicitly bundled and licensed. Keep browser/desktop chrome as independently configurable layers and expose app-only, browser-with-desktop, and declared host-profile modes. Never infer native OS behavior from this simulated chrome. Source audio is removed by default; any future pass-through must be explicit, timestamp-aligned, and tested.

E2 job state belongs to the existing Device Studio `jobs` domain, not a second job system: queued → admitted → running → checkpointing/encoding → validating → completed, with terminal failed/cancelled states and truthful partial-output references. Persist an idempotency key, attempt number, lease/heartbeat, completed frame batches, log references, cancellation request, child process identity, output staging refs, terminal reason, and cleanup receipt. Cancellation flows through the existing process owner to the whole descendant scope, closes encoder input, removes partial staging or retains it as explicitly incomplete, records exit/result facts, and never promotes partial output. Resume only from digest-verified completed frame batches and the same resolved tool/shot closure. A UI disconnect does not cancel a job. Disk-full, worker crash, process timeout, encoder failure, corrupt checkpoint, and shutdown/restart each need explicit recovery behavior.

## E3 native route and minimum scene proof

E3 is a separate strategy selected only for actual articulation, parallax, exact model geometry, or a qualified 3D look. Keep the domain shot/motion schema renderer-neutral; translate canonical right-handed meter coordinates, explicit local frames and normalized quaternions to Blender at a narrow adapter boundary. Bind the authentic app raster to a measured display mesh UV/surface, preserve physical bezel/occlusion geometry, and evaluate source clock, device/joint transforms, and camera at the same canonical output times. The first acceptance is one professionally art-directed phone scene with a real rigid-body silhouette, accurately measured flat display and pill/camera cutout, restrained material roughness/specular response, contact shadow, coherent key/fill/rim lighting, a practical support surface, one fixed art-directed camera, and a short controlled three-quarter orbit or handoff ending on readable app pixels. Include one approved still and a continuous 2–4 second temporal sample. The source/artifacts, scale/axes, support/contact, UV/material and screen crop must be inspectable. A geometry-only turntable, primitive phone, default HDRI, noisy/glossy plastic, or sample-count increase is not the professional proof. Do not expand device catalog until this scene receives visual review.

The local Blender build is not a qualified GPU pack. E3 needs an owner-provisioned, versioned package manifest covering the Blender executable, runtime libraries, Python modules/extensions, Cycles backend kernels/denoisers, color config, fonts and scene assets, hashes, OS/architecture and licenses. Use the control-plane host-tool/dependency provisioning route (Scenario Dependency Analyzer / host-tool owner) for managed acquisition, and the scenario-to-desktop release/bundle owner for signed target artifacts; Device Studio must not implement a parallel installer or copy ambient paths. `auto` means a render-qualified local profile with admitted capacity. Explicit unavailable GPU requests report why and offer an explicit CPU profile when supported; no silent quality change or cloud/paid fallback. CPU remains a valid initial E3 route if performance/quality meets the declared scope. GPU compile flag, detected hardware, enumerated device, capacity, successful representative render, and visual qualification are separate evidence states.

The process boundary should reuse `packages/platform-go` operation-owned containment and the desktop supervisor/job lifecycle rather than hand-rolling descendant tracking. The existing containment APIs are platform-specific: review the Linux cgroup, Windows Job Object, and macOS rlimit/process-group implementation before claiming equivalent resource enforcement. Import/generated scene code remains untrusted; disabling Blender autoexec and setting limits do not provide filesystem/network isolation. Until the worker owner proves a qualified sandbox boundary, accept only the controlled, reviewed scene recipe path and disable arbitrary imported scripts. On crash/reboot, durable jobs reconcile from leases/heartbeats and verified frame batches; on cancellation or app shutdown, terminate the owned tree and keep terminal cleanup evidence.

## Reuse boundaries

- **Device Studio owns:** project/shot/presentation/motion/job schemas and compiler; rendering strategy eligibility; source clock and mask/homography semantics; job state/reconciliation; visual acceptance and output provenance.
- **api-core storage/blobstore owns:** managed artifact bytes and opaque references. Follow the existing notes upload and Image Tools storage examples (`api/handlers/notes/attachments_handler.go`, `image-tools/api/internal/storage`). Use byte digests in Device Studio metadata; do not put media in SQLite or alongside the executable.
- **Scenario Dependency Analyzer/control plane owns:** governed dependency and host-tool resolution/install/upgrade. Device Studio resolves a pinned profile/receipt; it does not run raw package managers or build an installer.
- **platform-go + scenario-to-desktop own:** operation containment and desktop lifecycle/bundling/signing/update path. The renderer adapter should request an owned bounded process operation and return typed progress/result evidence.
- **Image Tools owns:** existing image operations, managed model/backend providers and its tool provisioning. It is not a general video renderer. Device Studio should not call Image Tools to emulate a deterministic compositor.
- **BAS owns authentic browser capture; Launch Video owns final film assembly/audio/CTA.** A composed shot is a direct PNG/MP4 artifact that can be consumed by Launch Video.

## Existing versus missing

Existing and evidenced: v6 phone/laptop algorithm prototypes and artifacts; v8 read-only host/tool probes; Device Studio domain requirements/design/data/security/performance contracts; notes multipart upload seam to api-core blobstore; Image Tools managed storage and provider/host-tool patterns; `platform-go` native containment primitives; scenario-to-desktop packaging and lifecycle code.

Missing in Device Studio: product schemas for shots/presentations/jobs; artifact source abstraction and digest enforcement; decoder/source-clock implementation; plate calibration/mask authoring UI and validation; configurable deterministic chrome; a bounded compositor worker; owned cancellation/recovery/checkpoint processing; PNG/MP4 export and output decode/receipt; packaged cross-platform FFmpeg/OpenCV or a narrower deterministic closure; Blender profile resolution/qualification, an approved 3D scene, and runtime security isolation; worker integration tests and target-native packaging evidence. The local `device-studio.db` and generated `notes` handlers do not constitute these features.

## Proposed code boundaries

Add only after public contracts are settled in proto and schemas are registered beside their interpreting domain code:

- `api/internal/presentations`: versioned profile repository, geometry/mask validation, source-clock types/evaluator, chrome profile.
- `api/internal/shots`: immutable shot revisions, compile/admission results, dependency digest calculation and strategy eligibility.
- `api/internal/rendering`: a small `Renderer` interface (`Capabilities`, `Render`), deterministic photographic implementation, frame compositor, and bounded PNG/MP4 encoder adapter. Keep the OpenCV/Python prototype as reference; choose implementation against package/bundle support before adding dependencies.
- `api/internal/jobs`: durable job/attempt repository, transitions, heartbeat/checkpoint/cancel/reconcile; launch children through the platform/desktop operation owner.
- `api/internal/artifacts`: thin consumer interface over api-core blobstore/storage, with digest/size/MIME checks and cleanup reference rules; no duplicate filesystem/blob database.
- `api/internal/renderers/blender`: later narrow native adapter and capability receipt, invoked only through the same job/process boundary.

Do not create a generic plugin SDK, private process supervisor, host repair, tool installer, sandbox framework, provider router, or cross-renderer abstraction beyond typed common shot inputs and receipts. Revisit a broader abstraction only after a second real backend proves the shared behavior.

## Validation evidence to collect

1. Contract tests for rational clock mapping, VFR/repeated PTS, trim/rate/hold/reject policies, image sequences, crop/contain, pixel-center homography, mask bounds/occlusions and Dynamic Island/background separation.
2. A deterministic fixture shot: fixed plate and actual captured app media; approved still PNG plus 2–4 second MP4; unchanged pre-encode outside-mask pixel assertion; decoded MP4 dimensions/fps/duration/color/audio-stream policy; digest receipt and portable project reopen.
3. Job tests for idempotent submission, lease expiry, cancel before/during encoding, descendant cleanup, full disk, crash after a committed frame batch, corrupt/mismatched checkpoint, shutdown, retry and atomic publication.
4. E3 capability probes separately record binary/profile digest, backend compiled, device enumerated, admitted capacity and one representative render; then human review records the scene inputs and look/temporal findings. Include CPU output, unavailable GPU, missing kernel, missing library/denoiser, and no-silent-fallback behavior.
5. Bundle/target evidence for clean-user Linux x64, Windows x64, macOS arm64 app and renderer pack; no repo/home paths, no untracked downloads or implicit libraries. Verify process cancel/restart and artifact full decode on each claimed target. Cross-GPU bit identity is not promised.

Current host probe results are historical evidence from v8, not fresh qualification. No binaries were run as a scenario launch, no packages were installed, and no paid provider calls or edits outside this evidence directory were made during this investigation.

## Source pointers

- Scenario contract: `scenarios/device-studio/docs/domains/rendering.md`, `presentations.md`, `concepts/DATA.md`, `internal/SECURITY.md`, `internal/PERFORMANCE.md`, `internal/SEAMS.md`.
- Prototype: `/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v6/phone/compose.py`, `v6/laptop/compose.py`, `v6/phone/source-clock.json`.
- Tool proof: `/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v8/research/desktop-rendering-evidence/toolchain-probe.json`, `gpu-probe.json`, `probe_toolchain.py`, `probe_gpu.py`.
- Cross-platform/package and lifecycle research: `/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v8/research/desktop-rendering.md`, `implementation-approach.md`.
- Reuse source: `scenarios/device-studio/api/handlers/notes/attachments_handler.go`; `scenarios/image-tools/api/internal/storage`; `packages/platform-go/containment*.go`; `scenarios/scenario-to-desktop`.

## Read-only local probe record (2026-10-19)

Commands executed from the repository root; these inspect package/driver metadata only and do not launch Device Studio or render a scene:

```text
file /usr/bin/blender
/usr/bin/blender: ELF 64-bit LSB pie executable, x86-64, version 1 (GNU/Linux), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, for GNU/Linux 3.2.0, BuildID[sha1]=d378742bf77ec65f723f585b9c1e8209cc0f611b, stripped

dpkg-query -W -f='${Package} ${Version}\n' blender blender-data
blender 4.0.2+dfsg-1ubuntu8
blender-data 4.0.2+dfsg-1ubuntu8

nvidia-smi --query-gpu=name,driver_version,memory.total --format=csv,noheader,nounits
NVIDIA GeForce RTX 4070 Ti SUPER, 580.173.02, 16376

ffmpeg -version | head -n 3
ffmpeg version 6.1.1-3ubuntu5 Copyright (c) 2000-2023 the FFmpeg developers
built with gcc 13 (Ubuntu 13.2.0-23ubuntu3)
configuration: --prefix=/usr --extra-version=3ubuntu5 --toolchain=hardened --libdir=/usr/lib/x86_64-linux-gnu --incdir=/usr/include/x86_64-linux-gnu --enable-gpl ... --enable-libx264 ... --enable-shared

command -v nvcc
(no output; absent from PATH)

find /usr/share/blender -type f \\\n  \\( -name '*.cubin' -o -name '*.ptx' -o -name '*.fatbin' \\) -print -quit
(no output; no matching kernel payload found)
```

The package/driver command results agree with the retained v8 `toolchain-probe.json`. Blender backend enumeration comes from the retained `probe_gpu.py`/`gpu-probe.json` read-only probe: CUDA backend flag true, OptiX false, CUDA driver init succeeds and sees one RTX 4070 Ti SUPER; Cycles CUDA and OptiX enumeration each returns only the CPU. It intentionally did not render, create a CUDA render context, benchmark, or save Blender preferences. Thus the precise status is: GPU hardware and driver visible to the host; installed Blender GPU route not qualified, with missing precompiled kernels/compiler evidence; render capacity and performance unknown.

## E2 implementation choice and scope

For the first portable photographic renderer, use a small Device Studio Go composition package built on the standard `image` / `image/draw` primitives, with a domain-specific inverse homography sampler and mask compositor. Do not add OpenCV/Python to the desktop dependency closure for this operation: the archive proves the math and appearance but packages an unnecessarily broad runtime and its host paths are not portable. No existing shared renderer owner was found that provides pinned plate + calibrated screen homography/masks + source-clock-preserving composition; Image Tools owns image-editing/provider operations and managed bytes, not this deterministic presentation contract. Device Studio therefore owns this thin compositor, while reusing api-core blobstore for bytes and Scenario Dependency Analyzer/host-tool provisioning plus desktop packaging for FFmpeg.

Keep the Go seam deliberately narrow: `ComposeStill(ctx, PlateRef, ScreenFrame, PresentationMapping) -> PNG bytes + ComposeReceipt`; later `ComposeFrames(ctx, ShotRevision, FrameSink) -> frame receipts`. Decode inputs via Go's registered PNG/JPEG codecs for the initial raster slice. For video, use the governed FFmpeg profile as a decoder/encoder child owned by the job/desktop lifecycle; keep it outside the homography algorithm. Ensure decoder output retains an explicit PTS side channel or an immutable `PTS -> decoded-frame digest/ref` index. Never allow rawvideo's implicit sequential frame order to masquerade as authentic time. Before supporting VFR, qualify one FFmpeg invocation that emits/retains exact frame timestamps and tests duplicates, gaps, trim, and rate conversion. Native FFmpeg invocation should be argv-based with fixed argument construction, bounded pipes/temp storage, explicit input/output paths, no shell interpolation, and an exact version/build/profile receipt. `png` still export stays Go-native; `mp4` uses the same staged composed frames piped to the resolved FFmpeg encoder and full decode verification. This means FFmpeg is required only for video decode/encode, not for the compositor or still output.

The inverse warp maps each output plate pixel back through the inverse 3x3 homography into source pixel-center coordinates. Reject singular/ill-conditioned transforms, non-finite coefficients, degenerate quads, out-of-range masks and size mismatches before pixel work. Preserve the original plate pixel when the active mask is zero; blend only inside the active display mask, with hardware occlusion applied after the screen layer. Use a documented deterministic filter (bilinear first, with a bounded higher-quality option only if a dependency-free implementation and fixtures justify it), fixed color conversion, and explicit edge behavior. Record algorithm/version, source and plate digests, mapping, masks, sample filter, output dimensions and pre-encode outside-mask digest in the receipt. The `FrameSink` abstraction exists only to let the same one-frame algorithm feed a PNG file/blob, a bounded MP4 pipe, or a test collector; do not grow a renderer plugin API.

This slice promises a calibrated **fixed** photograph and screen replacement, configurable simulated chrome, and PNG/MP4 exports at the actual plate dimensions (or a smaller output). It does not promise upscaling detail. The retained v6 phone plate is 1536x1024 and laptop is 1672x941; neither establishes the PRD's 2560 px still-detail target. Those plates are still useful fixtures/reference renders. A 2560 px deliverable needs a separately approved higher-resolution plate and mask/calibration or qualified native 3D source, not interpolation of v6. E2 acceptance should state the actual source/output dimensions in the UI and receipt; compiler should reject a requested minimum-detail profile when the pinned source plate cannot meet it.
