# Renderer core E2 preparation evidence

This directory contains task-local baseline and replay evidence for the bounded `api/internal/composition` package. It does not constitute Device Studio E2 acceptance: no job owner, public wire/API, chrome generator, video decoder, encoder, packaged tool profile, durable cancellation/recovery, or qualification workflow is connected yet.

## Source baseline

The worker-card local exact-path snapshot was captured before source writes at `2026-10-20T01:53:36Z`. The capture records explicit absence for all six planned source files. `baseline-original/manifest.json` and `prewrite.paths` are the durable byte/absence baseline. Baseline manifest SHA-256: `d004a34bae0ff9a4cfd9d26ed9d0e3a5568727f1f7ae7fe362887908612e5f56`. All six source targets were absent before the task and are now present; no concurrent source edit was observed. `postwrite-manifest.json` records each source hash/size/line count and the baseline digest; its SHA-256 at capture was `1c12dbf736864843bd09f09c26d63b4b794ff55cb0a0bf0679b6d47e94a6d280`. Source revision was not recorded because this bounded worker-card route did not inspect git. The baseline bytes remain immutable.

## Package contract

`api/internal/composition` has no non-standard-library imports and no IO. `Compose(ctx, Inputs)` takes resolved plate, authentic precomposed screen image, surface size, ordered plate-raster-boundary quad (TL/TR/BR/BL), active and optional hardware masks, plus a bounded sample count. Masks must be `image.Gray`/`image.Gray16` grayscale coverage or `image.Alpha`/`image.Alpha16` alpha-only images; RGBA/NRGBA masks are rejected so black color cannot silently erase alpha semantics. It uniformly contains the screen within the declared surface, computes a homography, samples the inverse transform at plate subpixels with premultiplied bilinear filtering, and returns an RGBA64 image and protected-pixel receipt. Dimensions and pixel complexity are bounded; invalid geometry, masks, or fit inputs fail before output allocation. Cancellation is checked before work, while checking for an empty effective mask, and between row batches. Hardware coverage removes screen coverage. The literal active-mask center value 0 or hardware-mask center value 1 defines the protected pixel set before antialiasing. Those pixels are copied from the plate and read back for exact equality; only other pixels use antialiased subpixel coverage.

The in-memory compose receipt sets `replay_identity_bound:false`; a job adapter must add immutable artifact refs/digests before making a replayability claim. The v6 fixture receipt records its actual input/output file digests separately.

`PlanFrames(ctx, FrameClock)` independently maps a rational output rate to rational source PTS targets. Each output sample selects the latest source PTS at or before its target and records that sample’s source-frame index (the last duplicate PTS wins deterministically). Trim is `[start,end)`; source underflow and EOF require explicit hold policies. Frame count is bounded and intermediate math uses `math/big` before checked conversion to the exported int64 rational shape. This is a planning seam only; it does not decode video.

## Verification commands and evidence

From `scenarios/device-studio/api`:

```text
gofmt -w internal/composition
go test ./internal/composition -count=1 -v
```

Result: pass (the optional v6 fixture was skipped in this run). Regressions cover identity mapping; perspective corner and grid mapping; active/hardware mask priority and literal-mask boundary protection; mixed-alpha transparent-color fringe; contain fit without crop/stretch; nonzero image bounds; malformed/huge inputs; cancel before and during work; rational frame timing, latest-PTS selection, VFR/repeated PTS, explicit underflow/EOF, invalid clocks, overflow and bounds. The optional v6 replay fixture is intentionally skipped in this general command unless its two environment variables are set.

Read-only archived v6 replay, with outputs constrained to this directory:

```text
DEVICE_STUDIO_V6_PHONE_DIR=/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v6/phone \
DEVICE_STUDIO_RENDERER_CORE_DIR=/home/matthalloran8/Vrooli/scenarios/device-studio/docs/internal/goal/evidence/renderer-core \
go test ./internal/composition -run '^TestV6PhonePlateFixture$' -count=1 -v
```

Result: pass. An initial gofmt invocation used a repository-relative path from the API directory and failed to locate it; the documented command above is the corrected invocation. This replay uses the archived phone plate, source-screen raster, active mask and calibration without modifying those originals. The generated [v6 replay image](v6-phone-replay.png) is 1536×1024; after the boundary-protection correction its SHA-256 is `903fb7a8454800dc21bd42e6525352bbddacbddeaf9bbcdcf4d095f602acd13e`, with input digests in [v6 replay receipt](v6-phone-replay.json). It visually reconstructs the accepted fixed photographic phone presentation. It is evidence of this package's still composition only; it does not reproduce v6's 96-frame MP4 or claim higher-resolution detail.

## Remaining adapter work

The parent job/worker owner can consume `Composition.Inputs` and `Result` directly after resolving bytes and adapting its compiled presentation quad/masks. The contract's coordinate frame is top-left with +u right/+v down; quad coordinates are raster boundaries while the algorithm samples pixel centers. v6's saved homography is in its OpenCV pixel-center convention, so the opt-in replay converts each source boundary through `H(x-0.5,y-0.5)+0.5` before calling this package.

The next safe integration scope is a job adapter that resolves and digest-checks artifacts, invokes `Compose`, and atomically stores PNG plus receipt. Video decode/PTS extraction, FFmpeg ownership, configurable chrome, persistent schemas, UI surfaces, worker process containment, cross-platform packaging, and the PRD's 2560px still-detail source requirement remain outside this task.
