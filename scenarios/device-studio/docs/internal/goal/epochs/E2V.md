# E2V — Source-continuous photographic video

- Outcome: A saved fixed photographic shot accepts a real recording, previews the pinned source schedule, renders a bounded local MP4 job and exports/reopens the exact revision through UI/API/CLI.
- Kind: feature
- Started: 2026-10-20 (bounded prerequisite work)
- Estimate: revised planning forecast10650 net runtime lines; whole video slice remains unqualified
- Expected size: aggregate planning cap12500; only individually admitted boundaries may write
- Metric: 24/30 fps landscape/portrait source-clock, full decode, cancellation, restart and output-profile assertions
- Targets: current-host required; other native runtime validation excluded
- Workers: native Luna Materials (independent review/integration), Workspace (execution/media), Renderer (tool profile/native inspection)
- Status: bounded fixed-video, screen-entry, six-case matrix, browser endpoint and silent-audio checkpoints passed; photographic 1080p and full video release remain unqualified

## Integration design decision — before admission

The first revised estimate is approximately3300 net runtime lines, with a provisional4500 cap for the full tool/media/worker/API/CLI/UI slice. This is a planning forecast, not admission. The earlier650–950 estimate covered only a narrow video backend and omitted required media lifecycle and public surfaces. Component ranges: tool-profile400–700; media schema/streaming intake/inspection integration600–950; typed shot timing/closure250–400; decode/compose/encode/verification700–1100; job/output integration250–400; preview/UI/CLI450–750. Reconcile ownership overlap and lock the actual budget before writes. The first profile must satisfy DS-SCREEN-AUDIO: silent default, intact source/audio provenance and explained unsupported-policy rejection. Enabling passthrough is conditional on separate qualification; it is not promised by this profile. This corrects the earlier broader epoch wording after the canonical contract review retained in p8-matrix/audio-contract-review.md.

Video import requires an actual schema/domain extension. The current media table allows only PNG/JPEG and positive dimensions; an uninspected video cannot masquerade as ready image media. Store original bytes through bounded streaming, expose pending/ready/rejected or reconciliation state with typed reasons, and retain the canonical inspection receipt (selected stream, raster/color/rotation facts, exact rational timestamps, source and tool identities). Only qualified ready media is render-eligible. Do not raise the byte limit while retaining the current whole-file ReadAll path for large recordings.

Native inspection is owned execution. The parent selected a bounded generalization of the existing jobs subject/kind and lifecycle dispatcher, rather than a second media-specific intent/attempt/reconciliation ledger. Media owns source facts/readiness and its job reference; the existing jobs owner keeps attempt, native claim, cancellation, recovery and release state. Existing render API wrappers remain scoped. New media inspection uses an explicit subject kind, never a fake shot revision. Preserve the exact legacy canonical Input JSON and hash for old still jobs; optional tagged new fields need versioned handling and a retained legacy replay test. No second queue, process supervisor or polling loop is implied.

Inspect source timing once through the owned native inspection job and retain its bounded PTS vector/timebase/EOF as a digest-bound artifact. Submission pins that receipt and the selected trim-start frame used for automatic fills. Dispatch consumes the frozen source/tool/clock closure. A pinned decode invocation emits raw frames and corresponding timestamp evidence; compare every decoded frame's count/order/PTS with the retained schedule and fail on mismatch. This is decoder correspondence validation, not resolving current media metadata or substituting a fresh source clock. Reuse composition.PlanFrames for preview and final selection. Decoder seeking or an implicit FFmpeg fps filter cannot define a competing timeline.

## Scope

Preserve every video obligation from the original E2, ACCEPTANCE.md and presentations/rendering domain docs. Extend project media admission with qualified native probing/decoding, bounded duration/streams/dimensions/rotation/color handling, original-byte preservation and rational presentation timestamps. No guessed frame-index timing. Use the E2 immutable shot/job/output owners and the existing composition clock. Pin source trim/rate/EOF policy and implement preview/final parity. Simulated phone/browser regions remain outside the app viewport; captured chrome is never duplicated. Source audio is absent by default and explicit passthrough remains separately qualified.

Resolve FFmpeg through its actual governed tool owner, pin binary/build/dependency/license/profile evidence and use owned execution with bounded resources. No ambient PATH fallback, private installer or paid fallback. Implement typed fixed argv, safe staging and output verification; UI disconnection cannot cancel or invent the authoritative job outcome. Local tool failure or lost outcome remains explicit. Do not treat the current Linux operation owner as arbitrary-code sandboxing or foreign-platform certification.

## Exit gate

Lock exact scoped Test Genie phases and current-host leased API/CLI/BAS journeys before admission. Retain actual 1080p landscape/portrait SDR clips at 24/30 fps, full decode/metadata evidence, pinned input/output hashes, timing within one output frame, deterministic protected regions before lossy encoding, and genuine cancel/recovery cleanup. Short final-render samples precede long jobs. Camera entry must end on the active app viewport at continuous source time; it cannot stop on an enlarged bezel. The original E2 video gate is not accepted until this evidence exists.

## Non-goals

No physical phone spin/laptop articulation, AI generation, paid calls, library expansion or release-completion claim. Preserve original research. No direct API-service binary launches or cross-platform runtime testing.

## Retained authentic recording candidates

The parent read `inputs/source-manifest.json` in the original research archive and verified both original/derived file hashes. The unconverted phone capture is `/home/matthalloran8/.vrooli/plan-artifacts/efforts/product-video-research-2026-10-15/revision-05/captures/mobile-conversation.mp4`, SHA-256 `4bf3e1663ce60116e85e4172f359ac9ce7a2b39a99bf4fa5d8b2e0fe040db857`; the retained manifest records430×880,30fps,488frames. The desktop source is `/home/matthalloran8/.vrooli/plan-artifacts/efforts/video-toolchain-reliability-2026-10-16/production/edits/aquila-organization-edit-gop1s.mp4`, SHA-256 `ed9eb8a3c530ed9824bb8ee7369447983d4d261cfe68d34ceea8d98031471cc9`; it is already an editorial1440×800 derivative, not a real-time performance recording. Fresh import/PTS/decode qualification remains required.

Archive `inputs/phone.mp4` and `inputs/desktop.mp4` are four-second24fps derivatives with hashes `761a4dc2842dbdce6c3bb38c81a741f2740f8a5ae171d0abef1c55a6f174efd4` and `cea6ad0392e45d31e17c85fec1b51c3bd7c3e239aa3616f9f4b6847e809dc084`. Use the original30fps phone input to exercise actual resampling instead of importing an already24fps conversion as source-clock proof. These are private fixture candidates; historical sampled privacy review does not establish new distribution rights or exhaustive review. No source file was changed.

## Tool-owner investigation (2026-10-20; read-only)

This investigation made no tool installation, dependency, service, or runtime-source changes. The v8 research remains the source for the product requirement and licensing cautions: it calls for both FFmpeg and FFprobe, records that the observed host FFmpeg enables GPL/libx264, and does not treat that host build as a distributable profile ([v8 desktop-rendering research](../../../research/snapshots/v8__research__desktop-rendering.md.txt), especially its tool and licensing sections).

The current host-tool declaration is [`internal/tools/ffmpeg/tool.json`](../../../../../../internal/tools/ffmpeg/tool.json): it names only `ffmpeg`, has no exact version/acquisition pin, says `host-required`, and leaves Linux/macOS/Windows unqualified. No `internal/tools/ffprobe/tool.json` exists. The shared host path is `internal/hostreq/resolve.go` → `internal/runtime/tool_handler.go`, with declaration shapes in `internal/hostreqkit/manifest.go` and artifact acquisition in `packages/binaryfetch`. This is the right owner path for development setup; a Device Studio script or ambient `PATH` lookup is not.

There is a concrete paired-tool readiness gap in the generic owner. `hostreqkit.ResolveCommand` and runtime `resolveFetchCommand` return the first available candidate, so a list containing `ffmpeg` and `ffprobe` does not require both. The package install path can run `VerificationCheck`, but fetched-artifact installation chooses `values.FirstNonEmpty(Commands...)` as its launcher and, after fetch, probes that selected command's version; it does not run the manifest verification check on that fetch-success path (`internal/runtime/tool_handler.go`, `inspectFetch`, `applyFetch`, and `resolveFetchCommand`). Therefore a verified development profile needs an owner-level paired executable contract that resolves and probes each binary, rejects a partial pair, and records both exact build/config identities. Do not mark the existing manifest qualified by merely adding `ffprobe` to `commands`.

The standalone owner provides useful packaging primitives, not a completed tool-profile bridge. `scenario-to-desktop/runtime/manifest` models per-service `Binaries`, `Assets` with SHA-256/size, and `AssetDirs`; `api/bundle` stages declared assets, and `runtime/assets.Verifier` checks declared file hashes and optional size budgets. Its runtime bundle environment prepends bundle-local executable directories to `PATH`. No inspected path resolves a host-tool acquisition profile into a standalone renderer pack automatically. A standalone implementation must stage an explicitly selected OS/architecture pair (including any shared-library closure), bind every staged artifact to trusted digest and size metadata, then resolve the bundle-local paths deterministically. It must not copy whichever host binary happens to be installed. The current `host-required` declaration makes no standalone availability promise.

Smallest credible implementation seam: (1) resolve governed FFmpeg and FFprobe declarations together, with per-platform pinned artifact/build metadata and one renderer-profile identity; keep install/provisioning in hostreq/binaryfetch; (2) have Device Studio request that resolved profile and pass its explicit executable paths/build identity to its owned process adapter; (3) for desktop packaging, consume the same reviewed per-platform artifact set through scenario-to-desktop's explicit `Service.Assets`/`Binaries` contract, requiring SHA-256 and size and carrying dependency/license/source notices. Extend shared declaration syntax only if existing acquisition target fields prove insufficient. The actual version/build and supported platform set remain decisions for a later admitted qualification task; no version is inferred here.

Independent refinement (2026-10-20): a generic paired-tool schema is not a demonstrated prerequisite. Existing `binaryfetch.AcquisitionTarget` can express two tool declarations pointing to the same archive digest with different `BinPath` values. The current handler may duplicate extraction into per-tool directories; that is a possible later efficiency issue, not a reason to invent another installer. `Commands` remains fallback candidates, not a required set. The required behavior is a profile resolver that verifies both exact canonical executables, records their individual digest/version/build/dependency identities, and refuses an incomplete pair. No ambient runtime lookup follows qualification. A current-host profile and a standalone redistribution profile are separate qualifications. This narrower interpretation supersedes the earlier implication that paired manifest syntax must be added before video work.

Qualification must retain the exact `ffmpeg -version` and `ffprobe -version` output/build configuration, digests, dependency closure, source and license notices. For execution, probe bounded inputs before decode, use fixed argv without a shell, cap duration/streams/dimensions/frame count/output/workspace and process time, preserve source-clock timestamps and explicit rotation/color behavior, verify full decode and final output with FFprobe, and retain cancellation/quiescence evidence. Exercise malformed/truncated input, unusual timestamps, rotation/color metadata, output bounds, interrupted execution, and a short known fixture. Qualification of one host build does not qualify other OS/architecture artifacts or redistribution rights.

Estimate for the owner/profile seam plus development verification and focused tests is approximately 400–700 runtime lines, excluding generated artifacts, platform release assets, packaging notices, and evidence. Standalone qualification is separately dependent on selecting and reviewing exact redistributable artifacts and their dependency/license closure; it is not included in that estimate. E2V remains parked and unadmitted pending an explicit platform/profile decision and budget.

### Acquisition and timestamp follow-up

The parent checked the [official FFmpeg CLI documentation](https://ffmpeg.org/ffmpeg.html) and [framehash muxer documentation](https://ffmpeg.org/ffmpeg-formats.html). Passthrough frame policy and copied input timestamps do not alone guarantee unchanged timestamps: encoder timebase and muxer handling also matter. Framehash reports its output stream timebase, so compare rational times after explicit timebase handling, not bare integer equality across probe and decode outputs. Pin and qualify the actual command against this host version with VFR, duplicate PTS, nonzero/negative start and trim-boundary fixtures. Current upstream documentation describes the latest revision; it is not evidence that every option exists in the installed6.1.1 build. No native decode qualification has occurred yet.

The parent checked the upstream [BtbN build documentation](https://github.com/BtbN/FFmpeg-Builds/blob/master/README.md) and [release inventory](https://github.com/BtbN/FFmpeg-Builds/releases) on 2026-10-20. Linux/Windows static and shared variants are available, but the floating `latest` URL is unsuitable as a replay identity. Daily builds are retained for 14 days and month-end builds for 2 years. Selecting a dated URL plus digest is therefore necessary but does not itself guarantee long-term availability: the governed asset/distribution owner must retain the exact selected archive and notices. No artifact has been downloaded, installed or qualified by this inspection. macOS requires a separately selected acquisition source; absence from this builder is not an architectural limitation.

The existing `composition.PlanFrames` already schedules exact rational output times over recorded source PTS, including repeated timestamps and explicit underflow/EOF policies. Reuse it. The missing seam is verified native decode order and timestamps, not a second timeline calculator. FFprobe exposes per-frame records and program/library versions; bounded `show_entries` should extract only fields required by the pinned profile. Its documentation cautions that interval seeking is not exact, so `read_intervals` alone cannot prove the source frame chosen at a trim boundary. Retain actual decoded-frame PTS and full-output verification. [FFprobe documentation](https://ffmpeg.org/ffprobe.html)

The shared host inspection currently returns a command name/version, not a digest-bound paired execution profile. The application cannot import the repository's `internal/hostreq*` packages. The next bounded design must expose an owner-produced, portable profile artifact through a supported boundary and consume explicit canonical paths, rather than embedding a private host installer or importing control-plane internals. Standalone bundle resolution consumes the same profile identity with bundle-relative paths.

Boundary refinement: the current `vrooli host` command handler executes local host inventory through `internal/cli/hosthandlers` and `internal/app/host`; an always-running Onboarding RPC is not a demonstrated prerequisite. Prefer a fixed read-only local command returning the typed profile artifact, consumed through a narrow Device Studio adapter; desktop consumes its packaged equivalent without the checkout or fleet. The exact public command/schema is locked during admission. `internal/tools/hyperframes/profile.go` provides fixed render-policy precedent, but it is not a reusable paired binary/build resolver and its test-slice execution policy must not be copied into product renders. Search Hub's command provider was degraded during this inspection; Prompt Manager fallback returned no suitable paired-profile capability, and direct owner source/help supplied the execution boundary.

Reuse inspection also found `audio-tools/api/internal/audio/ffprobe.go` and `scenario-to-desktop/api/screenrecording/native_ffmpeg.go`. These provide audio probing and X11 recording behavior through command-name execution; neither supplies a pinned paired profile or this application's immutable frame schedule. They are references for their own bounded operations, not a substitute for the missing profile contract. Keep the existing Device Studio jobs/composition owners rather than importing a second recorder/session manager.

For the first video presentation profile, resolve automatic edge fills from the actual decoded frame selected at the saved trim start, then pin those colors and the resolved status/home inks for the shot. Manual fills override them. This implements the existing fixed-sample policy and avoids introducing frame-to-frame chrome flicker. Changing source or trim invalidates this sample identity. Per-frame adaptive/smoothed chrome needs its own explicit future policy rather than silently rerunning still defaults on every frame. The moving app remains timestamp-driven, and source audio remains absent by default. No additional settings or runtime implementation are admitted by this preparation note.

## UI integration review (read-only, 2026-10-20)

Materials reviewed the approved Light-table implementation without changing runtime files. Extend SourceMediaPanel for upload/inspection readiness and typed rejection/reconciliation; unknown video dimensions stay unknown until the canonical receipt exists. Distinguish original recording bytes from a generated poster. Only ready media can be selected. ShotWorkspace retains immutable source identity and rational trim/output-time/EOF policy in the saved revision; a compact guided Source timing section and advanced exact-time details expose the same owner facts. Preview must consume the pinned schedule rather than infer authority from HTMLVideoElement progress.

RenderControls continues to own sample/final submission, idempotency, actual status, cancellation and selected revision/output provenance. A short sample is a real job with the same composition and tool identity plus an explicit bounded output interval. Extend PNG-only download verification for MP4 with content hash/size and canonical probed raster/codec/rate/duration metadata. CLI media import/get/list exposes inspection readiness; renders submit accepts typed profile/sample choices; shot timing stays in the existing snapshot JSON surface. No private FFmpeg invocation or duplicate job ledger. UI/CLI forecast500–750 net runtime lines, inside the provisional whole-slice budget; this is preparation, not admission. Audio stays off until independently qualified passthrough is available. Disclose AI when an action actually uses it; do not add unnecessary AI disclaimers to local rendering.

## Bounded timestamp experiment admission (2026-10-20)

While D10 finishes, Materials may run a finite current-host FFmpeg/FFprobe timestamp experiment under `evidence/E2V/timing-probe/`, with exact `/usr/bin` digests checked against the retained candidate. This admits experimental files and bounded direct tool probes, not product runtime implementation or a qualified renderer profile. Use fixed argv, subprocess timeouts, bounded tiny synthetic timing fixtures and at most a short authentic recording segment; retain command/version/byte/hash/PTS evidence and failures. Verify whether one rawvideo encoding multiplexed through the tee muxer to raw bytes plus framehash supplies matching frame count/order/rational PTS without an implicit competing fps conversion. Include VFR and nonzero/negative or duplicate timestamps where actually representable; report unsupported cases rather than forcing success. Do not overwrite original sources, install tools, open a live app lease, alter services, or claim desktop redistribution qualification. Maximum60seconds per command and512MiB retained output; remove only experiment-owned raw scratch after retaining verified receipts. This is the only newly admitted E2V work.

### Timestamp experiment result

Materials completed the bounded host6.1.1 experiment; parent read the receipts/commands and verified every retained-file digest in `evidence/E2V/timing-probe/manifest.json`. The authentic first60 frames yielded matching raw RGB24/framehash hashes and exact rational source/output PTS correspondence. Unique synthetic VFR with negative offset preserved timestamps only with an explicit adequate encoder timebase. Duplicate input PTS were changed by the tee/framehash muxer despite preserved image frames. A follow-up `{n} {pts} {tb}` `-stats_enc_pre` sidecar retained duplicates and paired with raw byte hashes by ordinal in the tiny no-reorder/no-temporal-filter case. This proves neither a muxed-file duplicate fixture nor the complete product pipeline. Qualify the exact eventual tool/decoder mapping; reject unsupported correspondence explicitly. Retained raw scratch was removed after checks, with hashes and commands kept; no product or live app state changed.

### Tool-profile decision refinement

The proposed external contract is `vrooli host renderer-profile ffmpeg-video --json`, backed by control-plane host/tool owners and a versioned `cli/v1` profile artifact. No runtime write is yet admitted. Existing binaryfetch directory acquisition/tree-digest and desktop explicit asset contracts are reusable; no existing owner captures a complete relocatable dynamic loader/library closure. Do not build a private loader or require fully hermetic OS independence merely to advance video. A qualified profile can bind exact tools, included codec/runtime artifacts and explicit host/OS prerequisites, rejecting drift or unavailable dependencies. Claim reproducibility within that qualified environment; never claim the Ubuntu host pair has no ambient libraries or supplies a distributable pack.

Prefer a reviewed codec-static FFmpeg/FFprobe pack acquired through the existing owner over copying the Ubuntu dynamic installation. Inspect actual ELF/import dependencies and record any OS prerequisite boundary; the label static is not evidence of zero dependencies. The parent checked BtbN's current README, which declares a glibc/Linux floor even for its static build family. A dated archive/digest and actual build/notice evidence remain required, and no archive download/install or redistribution approval is established by this research. [Upstream build requirements](https://github.com/BtbN/FFmpeg-Builds/blob/master/README.md)

## Preparatory owner admission P1 — tar.xz acquisition (2026-10-20)

The selected concrete BtbN9.0 dated candidate is documented in evidence/E2V/tool-profile/btb-n-linux64-candidate.md. Its tar.xz format is unsupported by the governed acquisition owner. Admit Materials to extend packages/binaryfetch format validation/schema and its existing registered-decompressor path for tar.xz, with registration at internal/artifactcodecs. Preserve binaryfetch's standard-library-only boundary; obtain any required pure-Go xz dependency through narrowly scoped Scenario Dependency Analyzer, never raw package managers or broad recursive reconciliation. Reuse the registered compressed-tar implementation rather than copy another extractor. Bound decoder dictionary memory and retain existing archive path/entry/expanded-size guards. Cover unknown codec, corrupt/truncated xz, single-binary and directory extraction, traversal/links and existing zstd/gzip behavior. Capture exact prewrite bytes. Expected100 runtime lines, cap180, generated schema and tests separate; stop before exceeding the cap and propose measured change. Regenerate acquisition schema only through the package's documented owner command and verify it. Run focused package/root codec tests; independent review precedes any download/install or tool-manifest change. This admits the portable acquisition prerequisite only; no native profile, archive download, lifecycle mutation or E2V product video admission follows. Full-slice4500 forecast includes this preparation rather than hiding shared-owner cost.

### Source-owner proposal review

Parent reviewed evidence/E2V/integration/source-owner-proposal.md and selected the unified-media refinement recorded in DATA/ARCHITECTURE. The original companion-video-table proposal duplicates original identity, paging and deduplication to avoid a pre-release schema conversion; it is not selected. Use greenfield-with-data preservation at a coordinated boundary. Terminal inspection finalization must cover failure and cancellation as well as success. Reuse common execution lifecycle behavior with typed subject staging/result adapters rather than copying the still coordinator into a second native supervisor. Exact implementation ownership and growth are under refinement before runtime admission.

## Preparatory owner admission P2 — one execution lifecycle (2026-10-20)

Workspace may implement the shared jobs/executor prerequisite from the revised source-owner proposal. Extend jobs.Input with explicit optional typed subject identity and versioned validation while preserving exact canonical bytes/hash for legacy still inputs. Keep one durable attempts/native-claim/cancel/quiescence/release authority and one bounded pump. Extract the existing still-specific staging/completion behavior behind narrow typed subject adapters; do not duplicate the coordinator or introduce a second generic workflow framework. Persist retryable terminal-result finalization in the existing jobs owner for subjects that need it, covering success, known failure and cancellation including queued cancellation. Unknown native outcome stays unresolved. Legacy still jobs need no new finalization work. Reject unsupported subjects without dispatch.

Scope: api/internal/jobs, api/internal/renderjobs and minimal existing module/composition-root wiring; tests and evidence/E2V/integration. No projects/media schema/API/UI/proto edits, tool execution, lifecycle, leases or downloads yet. The still adapter must remain fully operational throughout; tests use injected alternate adapters only to establish reusable subject behavior, never product-ready inspection claims. Exact prewrite bytes required. Expected350 net runtime lines, cap500, moved existing code counted separately; stop before exceeding. Deletion: replace hard-coded still-only dispatch/staging branches with one lifecycle plus still adapter; remove superseded duplicates, preserve all existing protection. Validate exact legacy canonical JSON/hash from retained D10 and v2 replay plus focused jobs/renderjobs tests/race and adapter/finalization retry cases. Parent independent source review precedes any live use. The full video outcome remains open and overall4500 forecast will be reconciled against actual shared preparation before admitting media/renderer writes.

P1 dependency-owner extension (2026-10-20): source inspection confirmed SDA has no repository-root Go install surface, while xz registration belongs in root internal/artifactcodecs and binaryfetch must stay stdlib-only. Admit Materials to add the explicit canonical `--scenario vrooli --surface control-plane` root-Go target in SDA installgateway and CLI help, with positive resolution and negative wrong-scenario/ecosystem/traversal tests. The target resolves only the known repository root/go.mod, never a caller-selected path, and retains all existing approval/security/install governance. Expected30 runtime lines, cap80, separate from archive180 cap. No policy bypass, registry hand edit or broad recursive reconciliation. Focused owner tests and parent source review precede supported owner build/start and actual scoped dependency install. If the exact owner policy requires human approval, retain its actual response; do not invent one from ordinary unrecorded discovery. This shared boundary addition is accounted within E2V preparation.

Preparatory candidate inspection (2026-10-20): after archive-code review, Renderer may download the exact151219048-byte BtbN9.0 candidate to a private goal-owned cache as research data, verify publisher SHA6668d9fda6278375751f3272f5a6799829b424eedf1934f32d4ee784e646a66e, and inspect its tar members without executing or installing them. Cap download160MiB, expanded inspection1GiB, individual members300MiB, member count2000 and walltime180seconds; retain exact paths/bytes/hashes and bounded license/build-notice text. Read archive executable members as bytes for per-tool digest and ELF dependency inspection only; no runtime launch or global launcher. This supplies exact bin paths and pack facts for the governed profile declaration, not an alternate installation path. Parent authorizes this free research download. Record cache path/size and cleanup in INSTALLS.md; retain the verified archive until the governed acquisition consumes it or the effort finishes. No distribution/redistribution approval follows.

Candidate inspection result (2026-10-20): archive size/hash matched the pinned publisher values. Bounded streaming inspection found73 members,539,732,165 declared expanded bytes, no unsafe paths or links, and exact `bin/ffmpeg`, `bin/ffprobe`, and `bin/ffplay` hashes. ELF byte inspection shows the candidate still depends on the x86-64 glibc loader and system libraries. The only license/notice-named member was GPL-3 `LICENSE.txt`; embedded build configuration, codec behavior, OS compatibility and redistribution remain unverified. See `evidence/E2V/tool-profile/archive-inspection/README.md` and `manifest.json`. No candidate executable was run or installed, and no xz decoder dependency has been registered.

## Preparatory owner admission P3 — paired profile boundary (2026-10-20)

Renderer owns the paired tool-profile implementation specified in rendering.md: internal/tools/ffmpeg profile policy, a dedicated ffmpeg-video acquisition declaration using the inspected archive member paths/digests, internal/app/host plus hosthandlers/cli manifest, and a versioned cli/v1 protobuf artifact generated through its owner. Exact prewrite bytes; expected650 net runtime lines, cap850 including proto source/manifest, generated/tests separate. The declaration must not overwrite generic ffmpeg or ffprobe commands. Scope is deterministic resolution, identity/required-artifact checks and explicit readiness/unsupported facts, with bounded version probes only on hash-verified known binaries. No ambient fallback, caller argv/path, private installer, OS library cloning, scenario code or native media job yet. Separate artifact integrity, runtime prerequisites and actual workflow qualification. Reuse current owner acquisition/portable asset primitives; no generic registry framework.

Focused policy/CLI and contract tests cover missing/partial pair, tamper, wrongplatform/version, safe paths/relocation identity, bounded probe failures and absence of install effects. Parent reviews exact source before actual managed acquisition and qualification, which will follow real xz decoder review. Other platforms remain explicitly unqualified; no foreign runtime tests. The research cache is not an executable installation. This locks source work while acquisition preparation proceeds. Revised aggregate planning forecast4200/cap5500 before these writes accounts for archive/SDA260, executor500, profile850, media950, timing400, video1100 and UI/CLI750 ceilings (4810 total); it is not whole-epoch acceptance or license to cross individual caps.

### P1 independent review follow-up

The first governed decoder candidate was installed before its source-level dictionary behavior was fully checked. It is not imported: the selected implementation is `github.com/mikelolasagasti/xz@v1.0.1` with a real64MiB allocation bound. Remove the unused earlier direct requirement through SDA; an exact-root-file PreviewFix proposed inappropriate module graph expansion from evidence copies, so it was not applied. Do not use broad topology reconciliation or manual module edits.

Parent identified that testing decoder EOF alone did not prove archive-owner rejection: tar iteration stops before the compressed footer. Materials reproduced acceptance of both corrupt and truncated xz footers through actual bounded inspect/extract APIs. Admit shared registered-tar finalization within P1's180-line cap: honor the existing deadline, permit at most1MiB zero tar padding, reject nonzero/excess trailing decompressed content and codec-footer errors. Reuse the bounded walker; single-binary extraction must not introduce an unbounded drain. Retain valid xz/zstd regression evidence and actual-owner malformed archive failures. Report before exceeding the admitted cap.

P2 initial source review: parent independently passed `go test -race ./internal/jobs ./internal/renderjobs -count=1`. The353-net-line implementation preserves claim-before-workspace writes and one native lifecycle. Review requested explicit bound-but-corrupt/size-mismatched streamed inputs, route loss around finalization, and mismatched subject mode through dispatch; worker is closing these test gaps before acceptance. No live use is admitted by this source result.

## P2 source acceptance and P4a media admission (2026-10-20)

P2 accepted for source integration after parent source review, independent race pass and worker test-only follow-up covering corrupt/truncated/oversized bound streams, wrong/expired routing around projection/ack, and mismatched mode before dispatch claim.353 net runtime lines; old still canonical/hash behavior retained. This is not live inspection/video qualification.

Workspace owns P4a unified media intake/readiness: existing projects media/schema/repository; projects HTTP/Connect adapters; projects.proto and owner-generated outputs; minimal main/module wiring and necessary compile-preserving consumer adjustments. Capture exact prewrite bytes. Expected800 net runtime, cap950 including authored proto, generated/tests separate. Follow DATA recording envelope and source-owner-proposal. Replace unbounded recording materialization with streaming intake/open/duplicate verification, preserve one project/digest identity, validate the full single-part multipart envelope before persistence and clean only operation-owned staging. Keep ready image behavior and presence-aware pending video facts. Implement typed, idempotent inspection projection with source/job/receipt identity checks; no native execution or simulated ready receipt. A narrow injected inspection boundary may be defined, but actual adapter/worker registration belongs to P4b and the video-tool owner after their qualification. Do not invent a second coordinator or expose a silently successful no-op inspector.

P4a changes the desired pre-release schema. Do not start/rebuild the live scenario or mutate existing data until the separate backed-up conversion boundary is reviewed. Use fresh local test databases. Validate media service/repository, HTTP streaming contract, unknown facts and image regression behavior with focused tests; proto owner verification/generation precedes consumer checks. Preserve existing scope guarantees and public API semantics. If950 is insufficient, report concrete file deltas before crossing; do not compress behavior or tests to meet it.

P4b (not yet admitted) adds the actual inspection adapter, immutable native receipt verification and shared pump registration. Its80–120 estimated lines are additional cost, not hidden inside accepted P2. The aggregate5500 planning cap currently has headroom over the4810 component ceilings, but all subsequent additions remain explicit. Full video decode/encode and saved timing/UI slices remain open.

### P1 acceptance and dependency cleanup admission

Parent independently passed root artifactcodecs tests and all binaryfetch package tests after reviewing bounded compressed-stream finalization. Archive implementation remains bounded by the admitted slice; bounded directory acquisition validates codec footers and 1 MiB zero padding. The historical single-binary API still has no aggregate entry-count/expanded cap; it scans the remainder rather than skipping footer verification, retaining per-entry 2 GiB protection. The actual FetchDir managed-directory route now also finalizes registered compressed-tar streams after its shared walk; the corrupt-footer regression exercises FetchDir. It still uses ExtractOptions{} and retains only the historical per-entry cap, not the bounded inspection/extraction aggregate limits. Qualification is restricted to the exact independently inspected archive, with a finite installation command deadline, not a general archive resource-policy claim. No private extractor or generic policy expansion is admitted.

Admit Materials to add the minimal SDA-owned explicit root Go requirement removal, expected120/cap160 additional authored runtime lines. Use `deps remove --scenario vrooli --surface control-plane --ecosystem go --module <module>`, dry-run default, explicit `--apply`; exact canonical root only, no arbitrary path, shell, topology discovery, tidy or lockfile expansion. Existing x/mod is available. Use the module editor through this owner, verify the exact named existing requirement, retain before/after digests, and ensure all other semantic declarations and go.sum remain unchanged. An indirect comment does not mean the requirement is absent from this root declaration. Reject wrong surface/ecosystem/scenario, malformed module arguments, unsafe symlink targets and concurrent source changes. A before-digest precondition for apply is appropriate; no registry edits are needed to stop using a module. Register the command in existing CLI surfaces, with focused dry-run/apply/negative/race-preservation tests. Capture exact prewrite bytes. Parent source review precedes applying to root; coordinate root-writing stability. Remove only unused ulikunitz/xz; keep approved fork and all unrelated workspace state. This is additional shared-owner cost inside E2V forecast, not archive70 accounting.

### P3 source acceptance and current-host acquisition qualification

Parent independently passed `make verify-committed-gen` and `go test ./internal/app/host ./internal/cli/hosthandlers ./internal/tools/ffmpeg-video -count=1` on the frozen P3 source. Source growth518<850. Read-only profile distinguishes verified content from unqualified runtime/workflow facts; license wording now records inspected GPLv3 notice rather than asserting an unsupported exact expression. Actual native `-L` output is bounded and retained after hash checks.

Admit Renderer to supported `vrooli host install ffmpeg-video` after read-only/dry-run collision inspection, with sudo unavailable/error and a finite180second acquisition command bound. Only the dedicated new managed tool and its owner record/launcher may change; generic ffmpeg/ffprobe remain untouched. Preserve the independently verified research archive until final owner acquisition evidence is retained. The exact pack has known73 members/539732165 expanded bytes and no links; generic FetchDir does not offer aggregate/count/deadline options. Its real registered-tar path now uses the shared bounded-padding/footer finalizer. Do not represent the candidate-specific inspection as generic extraction policy.

After installation, retain the read-only typed profile, exact version/build/license probes and member hashes; inspect actual OS prerequisites on this host. Then a finite qualification experiment may run the installed digest-verified pair with fixed argv over a short authentic phone segment and tiny synthetic CFR/VFR/negative/duplicate PTS cases, reusing the earlier timing-probe protocol. At most60seconds per tool probe and512MiB goal-owned scratch; no scenario lifecycle/lease or original-file writes. Verify raw byte count/order, sidecar rational timestamps, decode errors and encoder availability. Record unsupported cases instead of changing expected schedules. This is tool-protocol qualification, not product job/UI acceptance or redistribution approval. Remove only experiment-owned raw scratch after retaining receipts. Archive exact install/profile source/build identities and observed limitations in E2V/tool-profile.

SDA removal accounting correction: the worker counted nonblank lines; parent exact prewrite physical-line diff is170 net (167 source+2 registration+1 manifest), exceeding the160 cap by10. Preserve this historical breach; revised closeout cap190 admits source review and actual narrow cleanup, not further features. Parent reviewed source and independently passed CLI depsapproved tests. Real removal of only the unused first xz requirement is authorized through the new owner command with expected before digest; retain exact module before/after and unchanged go.sum proof.

### Managed acquisition and P4a independent review (2026-10-20)

Supported root `make install` succeeded after the narrowly governed nested autoheal checksum repair. The dedicated `vrooli host install ffmpeg-video --sudo-mode error --json` completed in12.671seconds; generic FFmpeg commands were not replaced. The typed profile reports `ARTIFACT_PRESENT`, artifact identity `2ad074849ce8635ce03d623865a26fdb1636a9a44346d21d4af08c1b191a9789`. Renderer retains the actual install/profile evidence and is running the admitted finite timestamp protocol. Artifact presence is not full workflow, standalone packaging, or redistribution qualification. Install ledger records have empty attribution metadata; do not infer installer attribution from those fields.

P4a source is frozen for review. Parent independently passed projects/handlers race tests (1.787s/1.071s) and protobuf verifier. Source review confirms uncertain-write readback retains referenced/uncertain objects and early reader close no longer drains a recording. Materials performs a separate adversarial pass. Check same-length corrupt-content HTTP behavior before accepting: a hash error discovered only after all response bytes are written may still appear as a successful download. The growth receipt must include authored SQL, not only Go/proto. P4b must distinguish verified source rejection from tool execution failure and cannot require valid in-envelope raster/timing facts in a rejection receipt. No live schema conversion or Device Studio lifecycle action is admitted yet.

The concrete read-only P4b proposal is `evidence/E2V/integration/p4b-proposal.md`. Its370–670 estimate replaces the unsupported80–120 integrated estimate, but native worker and retry details may raise it. Workspace is refining explicit retry versus idempotent Ensure using the jobs owner; Renderer is refining the exact receipt/clock protocol after the selected-tool experiment. Preserve source/job/profile/worker identity, one lifecycle/pump, and no GET-triggered progress. No runtime P4b write admission follows from this proposal alone.

## P4b inspection implementation admission (2026-10-20)

Parent verified all43 retained protocol evidence hashes, three installed member hashes and the authentic source hash. Qualification manifest SHA-256 is `5a1904c5e7649245acf30e715139ade7b25788c11a85dff57fff7508ce3fa118`. Accept the bounded current-host protocol experiment only:60 authentic frames, unique-PTS synthetic VFR, tiny24fps H.264 round-trip; duplicate/nonmonotonic/ambiguous PTS are explicitly unsupported in v1. Native full-source inspection and real product lifecycle remain unqualified.

Select the same-job explicit retry design in `evidence/E2V/integration/p4b-workspace-contract.md`, subject to the race refinement below. Ensure is deterministic initial submission/recovery and never implicitly retries a terminal job. Retry takes an expected job version and appends one jobs-owned attempt only after known failed/cancelled state, exact execution release/quiescence as applicable, settled accounting and terminal subject-finalization acknowledgement. Persist retry-from version in the same transaction; a repeated old request cannot create another attempt. Input/profile/worker/source identity stays immutable. Unknown native outcome remains unresolved. Typed source rejection is a successful inspection result, so it is not retryable under this transition.

Media's current attempt is a projection fence, not a second lifecycle ledger. The retry transaction can make a queued job visible before media resets to pending. The shared adapter must idempotently repair that projection before dispatch/finalization using the exact current jobs attempt and source binding; it must not require a client retry to make progress. An old finalizer cannot reset or overwrite a later attempt. Initial submit-before-bind has the same pump race and needs deterministic bind recovery in the shared adapter. No hidden sweeper or second queue.

Admit three non-overlapping bounded source slices with exact prewrite snapshots and focused tests:

- Materials owns only `api/internal/jobs` typed retry transition, repository transaction and tests (expected120/cap180 physical net authored runtime). Preserve legacy input JSON/hash, existing still transition behavior and pending-finalization/release semantics. Interface compile adaptations outside this owner are test-only and coordinated. No public endpoint, media schema, worker, lifecycle or tool invocation.
- Renderer owns the native inspection protocol under the existing `api/internal/rendering` worker boundary (a narrow private package is allowed if it prevents cycles), portable typed profile resolution/validation adapter, fixed worker command dispatch and tests (expected500/cap700 physical net authored runtime). The API process must never own a private inspector supervisor. Existing native operation owns worker and its FFmpeg children. Do not edit projects/jobs/renderjobs/handlers/proto or `main.go`; expose the worker entry seam to Workspace. Reuse existing platform owner CLI resolution or explicit trusted local/bundle configuration for the host profile; no ambient FFmpeg lookup, caller argv/path, private install, or control-plane internal imports. Worker uses bounded full metadata/frame parsing and full decode with exact rational ordinal/PTS correspondence; explicit source/probe/decode/time/memory/output limits, no silent truncation at frame limit. Retain actual decoded EOF; reject unavailable/ambiguous timing, unsupported SDR/rotation/SAR facts and limits truthfully. No scratch holding entire raw video. All source/tool hashes are rechecked before use. Manifest and receipt bind source, job/attempt, worker/profile/protocol and immutable clock size/hash. Ready receipt requires full accepted facts; rejected receipt requires identity/classification without fabricated accepted facts. Coordinate one versioned wire schema with Workspace before writing its consumer. No scenario lifecycle or live worker execution until source review; focused injected/unit tests are admitted.
- Workspace owns remaining shared inspection adapter/Ensure/Retry/public composition and media projection contracts; exact runtime admission follows closure of P4a review and final wire agreement. Forecast350/cap550, reserved here rather than authorization for concurrent schema writes. Existing coordinator/pump is reused, not copied. Public scalars preserve missing-versus-zero and private paths remain private. Include receipt verification before media ready, all-terminal replay and explicit retry repair.

P4a independent review found two additional HTTP gaps and the image-only eager-read regression; fix within its remaining cap before source acceptance. A same-length corrupt content response must not appear successful; the canonical DATA contract requires transfer failure on hash mismatch. Validate the bounded whole multipart request, including epilogue, before Commit. Preserve finite streaming behavior and cancellation; no512MiB ReadAll.

Revised aggregate forecast5200/cap6500 replaces4200/5500 before P4b writes. Prior estimates omitted actual inspection verification/retry cost. New component reservations total1430 (700+180+550) for P4b; SQL and all authored runtime count, generated/tests separate. Preserve historical estimates/breaches. Stop before exceeding a component bound and report concrete deltas; never compress source or reduce validation to satisfy a line target. This admits implementation, not completed scenario, paid/provider calls, schema conversion, lifecycle, lease or distribution.

P4b jobs retry source accepted:103 physical net lines (<180), one atomic jobs transition with persisted retry-from-version, exact terminal/release/finalization/quiescence/settlement fences and unchanged input/fingerprint. Parent reviewed source and independently passed jobs/renderjobs race tests (13.542s/13.086s). Receipt: `evidence/E2V/integration/p4b-jobs-retry-parent-review.json`. This accepts the local state transition, not media projection repair or live native retry qualification.

P4a source accepted after HTTP/envelope/consumer follow-ups:743 physical net lines including SQL and the separately baselined shots consumer (<950). Parent reviewed final source, independently passed projects/shots/handlers races before the final transient-EOF fix, and passed focused reader/HTTP tests after that fix; worker reran relevant focused/race checks. Exact chronology and hashes are in `evidence/E2V/integration/p4a-parent-review.json`. Final-byte withholding plus response abort prevents a full successful corrupted download without a second pass or whole-file buffering. Multipart epilogue is included in the capped stream before commit. No live conversion or inspection qualification follows.

Workspace P4b integration admission now active (expected350/cap550), after wire agreement with Renderer. Own `api/internal/renderjobs` inspection subject adapter/submission/finalization, existing projects receipt/storage/projection and current-attempt fencing, projects handlers and projects.proto owner-generated outputs, minimal main/module composition and consumer compile fixes. Materials owns jobs retry source and Renderer owns rendering/profile worker source; coordinate rather than editing their files. Capture exact baseline first. Use the same coordinator/pump instance in upload/Ensure/Retry and existing render operations. Register one actual adapter; source status remains pending if submission fails, with truthful retryable operation error. Ensure, explicit expected-version Retry, bound source/tool/worker closure, verified receipt/clock publication, all-terminal replay, and pump-repair races are required. Public responses use IDs/presence-aware facts only. Endpoint names and exact private wire use owner agreement, not an independent JSON schema. If550 is insufficient, report concrete file deltas/remaining work before crossing. No lifecycle/live conversion until separate review.

### Native inspection estimate correction and profile-provider owner

Renderer crossed the700-line source cap before reporting: current new runtime940 lines (`inspection.go`736, `inspection_types.go`204), with only compile verification. Preserve that240-line historical breach; do not compress safety code or treat compilation as source acceptance. The omitted costs are concrete typed receipt/clock contracts, profile rehashing, full bounded parsing/decode, and independent output verification. Revised native slice expected1100/cap1250, including the remaining profile provider and verifier hardening; tests/generated/evidence remain separate. Aggregate forecast5600, cap6500 retained using other components' measured headroom. Pause before the revised bound if further work cannot fit; retain exact prewrite accounting.

Search Hub capability discovery returned degraded CLI-command provider but direct package discovery found the existing `github.com/vrooli/repo-contract-go/cliinvoke` owner, already available in the API module. Renderer owns a narrow `ResolveInspectionProfile(context.Context)` provider using that resolver/invoker and the fixed read-only `host renderer-profile ffmpeg-video --json` command. Use trusted explicit `VROOLI_BIN` or the existing contract home resolution, disable PATH fallback through the resolver's injected LookPath seam for this provider, require an absolute clean resolved executable and bound JSON/stdout/stderr plus timeout. `ResolveOptions.RuntimeHome` is passed the OS user home as required by its current call into `RuntimeHomeEntryPath` (despite ambiguous comment); do not hand-code `.vrooli/bin` or import root internals. No new generic invocation library or shared installer. Workspace injects this provider into the same coordinator and stages the typed profile privately; public clients supply no executable path. A trusted standalone bundle provider can implement the same function seam later without a checkout. The native worker independently checks profile content and binary hashes against the immutable manifest; a profile sidecar is execution configuration, not a public artifact or alternate identity.

Native color-policy refinement: parent bounded read-only inspection of the exact authentic phone found yuv420p, square pixels, limited range and bt470bg matrix, with absent primaries/transfer. `tool-profile/current-host/phone-color-parent-observation.json` retains facts. A bt709-only gate rejects this required practical source. Admit a named pinned SDR conversion/default policy supporting known SDR matrices and explicit missing primaries/transfer while preserving those source facts as absent/unknown; never relabel them measured sRGB. Explicit HDR or unsupported gamut stays rejected. Receipt/projection must name the policy so later rendering uses the same declared transform/defaults. Actual output color conversion remains separately tested; this observation is not a color-fidelity certificate.

Native review completion extension: at1093 lines, Renderer reported before crossing revised1250 that live timestamp-output bounding, descriptor-bound executable launch, and the practical SDR policy need another bounded increment. Admit all three correctness fixes; do not drop them to meet an estimate. Revised native expected1300/cap1450, aggregate forecast5800/cap6500 retained. Stats must be bounded while FFmpeg writes, not merely when reopening a file after exit. Pin each verified executable descriptor through launch on the qualified Linux profile and recheck source bytes after use; atomic replacement must not silently change the executed tool. Use a bounded stats pipe and existing operation ownership, not a filesystem quota subsystem or second supervisor. A tiny fixed-argv pipe protocol probe is covered by the previous60second/512MiB qualification envelope; retain failures and results, no app lifecycle/live job. Tests must establish these failures independently; source remains unaccepted.

### P4b integration prospective budget correction and binding review

The Workspace integration worker reported +606 physical authored runtime lines against its +550 cap before finishing public handler/module wiring. Preserve the historical +56 overage: inspection adapter397, projects/media and streaming139, coordinator32, proto37 and SQL1. Tests and generated output are separate. Remaining wiring, immutable binding repairs and meaningful regressions are required scope. The parent prospectively admits expected850/cap1000 for this same bounded integration surface; no extra queue or feature expansion is admitted. Aggregate planning forecast becomes6300/cap7500, with timing/video/UI still unqualified.

Before source acceptance, bind completion to the persisted manifest digest and exact immutable job/attempt/source/worker/profile/protocol facts; fix source-size binding role and strict parsing; reconstruct manifests from pinned source facts; repair attempt bindings before terminal projection including queued cancellation; keep private inspection artifact keys out of public render surfaces; and recover submit-before-bind jobs before resolving a fresh executable/profile. Add focused regressions covering these failure windows. Current partial source is not compile-ready and no live lifecycle/conversion is admitted.

### Development conversion tooling admission (synthetic only)

Admit Materials to implement the previously reviewed one-shot out-of-tree conversion tool and synthetic tests under `/tmp/device-studio-p4b-conversion/`, preserving source/evidence copies under `integration/development-conversion/`. Expected350/cap500 physical tool lines, tests/evidence separate; no product migration framework, dependency changes, primary mutation, or lifecycle action. Use the existing API module's SQLite driver and exact captured old/current canonical schema. Preserve all old row values, IDs, other tables/indexes and original blobs; explicit check/apply, single transaction, complete pre/post verification and rollback-injection tests. Unknown schema or source integrity refuses. Saved queued jobs are allowed; the later live gate requires no active native execution or concurrent writes, not an empty job history.

Parent resolved the primary database from the lifecycle-managed API's actual file descriptors and `SCENARIO_DATA_DIR`, registered an explicit SQLite backup target through Data Backup Manager, and read back durable identity. Backup run `c8259aad-1eb6-427e-a426-d877e12471bc` completed with snapshot `5f8dbf915cd7b13f66b0e34bdb4713e0`; scratch restore verification is pending. This local safety destination protects the development conversion, not host-loss disaster recovery. No database conversion or Device Studio rebuild/restart has occurred.

Backup follow-through: Data Backup Manager scratch restore `05c42d45-0007-4c4e-830f-54ce7b29d8ec` returned `RESTORE_STATUS_VERIFIED`, checksum `8983f68a55a161f08a74cb8263ff85c62c26b762ddfd145a23d02366002478f3`. Evidence and parent hash receipt are in `integration/development-conversion/`. Recheck source freshness at actual conversion; this does not admit primary mutation or establish off-host recovery.

### P4b final integration correction reservation

Workspace reported999 physical runtime lines before the1000 cap, with concrete remaining routed read pinning and public lifecycle integration checks. Prospectively revise expected1150/cap1300 (aggregate forecast6600/cap7500 retained). This is completion of the admitted inspection boundary, not new features: preserve one pump, activate the exact test route after successful inspection mutations, wire upload/Ensure/retry/cancel consistently, and pin route before read-only media/status queries. Concurrent Ensure calls must converge even when independently resolved profile observations carry different observed_at timestamps; a losing submission recovers and verifies the durable winner without changing its immutable closure. Parent identified these review obligations before further writes. Historical606/550 breach remains;999/1000 is a timely report, not an overage.

The new native stdout-overflow regression found anonymous bytes.Buffer embedding promoted ReadFrom and bypassed Write limits. Named buffers corrected both native and profile-call sinks; exact subprocess regressions pass. The same pattern exists in the task's P3 host-profile probe. Admit a narrow owner correction there, expected15/cap40 additional runtime lines inside original850 reservation, with prewrite snapshot and real subprocess regression. No root rebuild/install is implied.

### Conversion-tool estimate correction

Materials wrote798 physical lines in the out-of-tree draft against500 before reporting, a historical298-line overage; the draft is not compile-ready and no data operation ran. Preserve that breach. Parent review found required fixes to the table-swap sequence, truthful receipt fields, index verification and already-current behavior. Prospectively admit expected900/cap1050 tool lines with separate tests/evidence; product runtime budget is unchanged. Avoid a migration framework or new backup engine. The later live preflight owns process quiescence; sidecar presence alone is not a reliable active-process test and files must never be deleted manually. Use SQLite's consistent transaction/lock semantics and retained backup identity, preserving saved jobs and original blobs.

### Native source acceptance after actual overflow regression

Accept P4b native source1170/1450 with updated independent review and parent tests. Parent full rendering race passed before the correction; focused race then passed actual Probe/Decode/profile stdout-overflow and malformed clock header/ordinal/order cases after the named-buffer fix. The selected-tool authentic/tail-moov experiments establish all488 exact frame times and duration-derived EOF. `p4b-rendering/parent-source-acceptance.json` binds the precise review/test receipts. This accepts source, not the live coordinator, cgroup enforcement, cancellation/recovery, full video rendering or cross-platform runtime.

The same task-owned P3 host profile sink had the ReaderFrom bypass. Its +8line named-buffer correction passed independent parent actual-subprocess overflow race tests (`tool-profile/parent-output-bound-regression-receipt.json`). No root rebuild/reinstall occurred; the installed owner binary predates this correction, so refresh through supported owner lifecycle before claiming the new sink in live command execution.

### P4c media CLI completion admission

Parent owns a narrow existing CLI extension: media Ensure/Retry/Cancel commands through the typed ProjectsService, positive expected-version validation, and truthful media kind/readiness/status with absent dimensions shown unknown. Update existing import wording for recordings; no private process calls or job progress on GET. Expected220/cap300 physical authored runtime including manifest JSON; tests/generated separate. Exact baseline is `integration/p4c-cli-prewrite/`. This consumes the reserved UI/CLI scope and closes required RPC coverage before the scoped unit phase. UI video playback/timing remains a later boundary.

### P4c CLI source gate and P4b independent findings

Parent media CLI source accepted at186 physical runtime lines against300 cap. All31 CLI tests passed under `go test -race -json ./... -count=1`; generated proto verification passed and CLI primitive evidence was refreshed through its owning test. The commands preserve exact int64 expected versions, report the returned durable state, and show unknown video dimensions without fabricated values. Evidence: `integration/p4c-cli-source-gate.json`; first failed test output is retained (new test used wrong uint32 type, old image-only wording expectation, stale primitive inventory), then corrected through source/owner generation. No live CLI mutation occurred.

Independent P4b integration review found two material gaps after the focused source gate: rejected native receipts intentionally lack accepted-only raster/clock fields but media validation required them, preventing durable finalization; inspection jobs omitted the immutable storage-route binding used by still jobs. Workspace is repairing both with regression coverage within1300 cap. Source acceptance remains pending.

Scoped Test Genie unit run `20261020-085354-82b0bca9` completed FAIL in34seconds: test execution clean,43 warnings, one Unit Health architecture error on `trackingMediaRPCStub`. Parent read the complete run artifact bytes, compared the call-counting decorator to generic `databasetest.SliceRepo`, and found no CRUD/collection implementation to replace. Retained as probable shared-detector false positive, reported through report-bug as `knw-1792486584923841356`; no test weakening or scanner evasion. Full catalog/findings/log and hashes are retained in `integration/p4b-unit-*`. Shared RCL preflight also reported existing candidate compilation failure and selected its last valid artifact; no claim of a green current RCL rebuild.

Independent converter review withheld live acceptance for existing-file open race, dependent views, blob bounds and write-mode WAL/current-video tests. Materials is correcting the synthetic-only tool within1050 cap; no primary conversion, lease or Device Studio lifecycle has occurred. P5 read-only worker proposal exists in `integration/p5-exact-frame-video-render-proposal.md`; shot/API proposal and exact output-framing contract are being refined, not implemented.

### Converter synthetic gate and owner refresh

Parent synthetic gate passed23 test/subtest results for the968/1050 converter, running retained evidence source directly; `integration/development-conversion/parent-source-gate.json` records exact hashes and limits. Mode=rw prevents creation if the database disappears at open; the current missing-path test does not inject the specific Lstat/open race, so that timing is not claimed experimentally. Existing-file semantics are source-reviewed. Original independent findings must remain distinct from the author's correction report.

Supported root `make install` completed after the task-owned P3 stdout sink fix. The subsequent read-only `vrooli host renderer-profile ffmpeg-video --json` reports the same selected artifact identity; install/source/profile hashes are retained in `tool-profile/owner-install-boundfix-receipt.json`. No Device Studio lifecycle or additional tool installation occurred.

### Primary inventory and final converter readiness

Parent opened the actual primary SQLite read-only and verified integrity/FKs: one retained project, zero source media, zero shots/revisions, zero jobs/attempts/release rows. The owner's canonical `data/media` directory is absent because no primary media exists. A final7-line converter correction opens that root only for the first referenced blob; parent focused regressions prove emptyoldDB check/apply preserves missingroot and nonempty missingroot refuses. Final975/1050 source SHA91bf572b...c58c38 accepted with retained synthetic and live-readonly evidence. No primary data was changed.

Readonly converter check returns oldschema logical snapshot95bce5cfbfb577050b4da330fc6bd95b23c35e5f9224efdb78b705d99d94a189. Refreshed Data Backup Manager run e87d4058-eac6-4397-9010-76a19ab84afe completed, snapshotbbd5ef99be43d9b4dbe055a9842fb9bd; scratch restore789c12fe-ea03-4f28-9523-fdc5c244313e verified. There are no referenced original blobs to back up. Current databasefileSHA differs from restoredsnapshotchecksum despiteWAL0; no byte-equivalence claim is made (SQLite backup can change physical representation). Exact logical snapshot is rechecked after supportedstop before apply.

HistoricalD10 pairedclear+TestGeniecleanup receipts match the still-running API process and no later manualseedtoken was issued in this task. Formalunitrun terminal; current nativeunit list empty. The preflightrunbook correctly notes no universalmanualsessionlistRPC. Parent will use these concrete receipts plus fullreadonlyjobinventory and poststopdescriptor/nativeunitchecks for this boundedtask, not infer a new authorizationgrant or build a universalleaseinventory system. A test-tagged old ListProjects read returnedprimary200 and is retained only as a limitation, not proof of test routing or leaseabsence.

### P4b source acceptance and bounded live admission

Accept finalP4b integration1157/1300 after parent finalrace81 tests/subtests, exact5 changedsource/testhashes, positiveEnsure→nativeadapter dispatch and wrongroute/rejectedreceipt/knowncancel/failure→retry regressions. `integration/p4b-parent-source-acceptance.json` retains command/names/hashes. Native worker and CLI gates remain accepted separately. Scopedunit falsepositive remains explicit, not relabeledPASS.

Parent admits the reviewed current-host development conversion and supported Device Studio stop/start under the existing full-implementation authority: recheck exactlogicalpreflight and zerojob/nativeexecution state; use verifiedrefreshedDABMsnapshot; stop via make, confirm oldprocesses gone/no nativeunit, apply frozen975-line converter only if logicalSHA still95bce5cf...d94a189; reopencheck current schema and exactproject/allotherrows preserved; start via make. No blobtree creation is needed for0references. On mismatch beforecommit, transactionrefuses/rollsback; on confirmedbadcommit, retainstoppedstate and usebackupowner restoration beforestart. Then parent may create one paired TestGenie manuallease and execute boundedauthenticinspection/invalidsource/cancel/retry through routedpublicclients, retain nativeoutcome/release evidence, clearownerpair beforeTestGeniecleanup and verify primaryprojectunchanged. No P5 implementation, libraryexpansion or cross-platform/livepaid tests admitted here.

### P4b source recount and live conversion outcome

Workspace corrected the finalintegrationcount to1165/1300 (inspection.go delta14, previouslycounted6). Source/testhashes and passing parentrace apply unchanged; prior1157 was an8-line accountingerror, retained here.

Supported `make stop` completed; oldAPI1788994/UI1789025 exited and no ownednativeunits remained. Stoppedreadonlycheck matchedexactpreoperationlogicaldigest95bce5cf...d94a189. Frozenconverter applied successfully inonetransaction, then reopenedcheck reportedcurrent schema logicalSHAaf487ae2017935773f0e143a16f737f74f25eccda7fc528d84724441217d7707. Parent independently compared completeprojectrows/alltablecounts, integrity/FKs and absentblobroot: preserved. Supported `make start` completed healthy on ports19349/21427. Logs/checks/quiescence/preservation receipts retained under development-conversion. No manual SQL edits or liveworker/directservice launches were used. Parent now prepares oneTestGeniepairedlease for boundedpublicinspection validation.

### First live inspection failure and entrypoint correction

Paired Test Genie lease `6a8880cb-bff1-4bfc-a7db-f603d1ff352e` admitted one isolated workspace. Tagged installed CLI read the fixture while the untagged request returned not found; the media read also pinned the SQL and file route. The public harness imported the authentic phone recording, then observed job `d9c41d11-bc29-4634-8d11-d7b2ccbc4e20` fail with `inspection_execution_failed`. The failure is retained in `integration/p4b-live-inspection/run-1`; malformed-source/retry checks were not reached.

Independent native and integration review found that `main.go` dispatches only the still worker before development preflight. The new `inspect-worker` invocation fell through to the lifecycle guard and exited before any probe/decode. The exact native operation reports known exit status1, dead root and empty process tree; no cancellation or successful inspection is claimed. The source digest and size remain correct. Admit Workspace's narrow entrypoint correction and a regression exercising dispatch before preflight, expected20/cap40 extra runtime lines within the existing1300 integration reservation. Renderer native source remains frozen. Preserve per-attempt release evidence before clearing the old route, then rebuild through lifecycle and use a fresh paired lease for replay.

The harness was launched when its recorded hash was f9753eccff4b0eb4f4af4b9794e31ddb95de10f122a50914fb34d1cc6ef913fc. Renderer subsequently added a final retry-attempt assertion (current hash db9a4e41ea22396c39165402559b1c2ee673bdb141f5171a137bca51a30aec0a). That assertion was not reached by this failed run; do not claim the later complete source was executed. Journal2f742596-62af-4b18-8409-afd2f54a0153 captures the conversion/source gates and this finding; managed identity remains unverified.

### P5 bounded protocol experiment admission

Admit Renderer to add an opt-in same-package test-only qualification harness, at most500 test lines; no product runtime, schema, dependency, lease or lifecycle edits. Reuse the exact pinned tool pair and authentic phone,60second experiment deadline and160MiB output bound. Qualify raw RGB fixed-size frame reads with concurrently bounded stats and full ordinal/PTS/count reconciliation; compare against the next successful P4b receipt/clock before acceptance. Test encoder fragmented MP4 output through a hard limiting writer, then independent metadata and full decode verification. Qualify explicit observed matrix/range and declared missing-tag defaults through a pinned transfer/primaries conversion to compositor sRGB, then explicit output conversion; scale-only matrix conversion must not be called transfer-correct. Canonical pre-encode pixel digest covers raster/color/pixel representation and bytes; closure/layout identities belong in receipt fields, not a job-dependent digest that defeats parity. Benchmark the existing 1080p compositor separately from fixture setup. No full raw-video scratch or new process/resource framework. Numeric output duration and production implementation remain unadmitted until measured evidence.

### Dispatch correction deployed and fresh live replay

Parent accepted the12-line main dispatch correction (integration1177/1300). Focused race regressions invoke actual main in bounded test subprocesses and verify both worker parsers run before lifecycle preflight. The new main_test baseline was not byte-retained; that omission is explicit and does not alter runtime accounting. Retained native receipt, exact raw intent/profile/manifest/journal and released attempt prove the old failure is settled. Direct paired routing clear succeeded with870 test-pool requests and14 test-root writes; primary counters omitted/default0. Test Genie cleanup returned cleaned. Primary original project and all other table counts remained unchanged.

Supported make restart completed healthy with buildffbfd4b87e42f1c52108e060a81fc00b704660dd1ad7fdcf9662975c7777a973. A fresh paired lease0cc48733-67ad-4271-a29a-f5158a8e3cdd exposes fixture15f720e0-d9e2-4962-8eeb-dbfa192f6527 only to the tagged installedCLI. Parent launched exact frozen harness-run-2.py SHA db9a4e41ea22396c39165402559b1c2ee673bdb141f5171a137bca51a30aec0a against that fixture. The old failure is preserved; this replay must establish inspection and malformed/retry outcomes before P4b live acceptance.

### Second live replay: accepted source and two transition defects

The authentic phone passed the real shared coordinator/worker path: source preserved,488 frames,430×880, timebase1/15360, EOF249856, duration244/15. Exact accepted receipt SHA15209a9fe1d78303146e206f7ab35481c4f8c2e2a2f5f55f825037ce37666824 and clock5868bytes/SHA78983498a079d73ed317e32fc6d5f4cdbd4964d7f72776ef70c8990a5b98fbca are retained under run-2. Native success/release evidence is retained separately; this is a real inspection success, not a full video render.

The negative flow revealed that a128-byte MP4 prefix produces a successful native `stream_count_unsupported` rejection receipt, intentionally without a clock. The integration reader swallowed os.ErrNotExist and misclassified missing optional clock as an execution failure. Its explicit retry then remained queued because jobs.Start required exactly one attempt, despite Retry appending a second. The harness timed out; no full journey pass is claimed. Admit narrow fixes within existing jobs180/integration1300 reservations, with actual Retry→Start/dispatch and successful-rejection-without-clock regressions. Parent qualified32/64/128byte prefixes through the pinned FFprobe:32bytes exit1,64/128 exit0. The corrected401-line harness retains both classified128byte rejection and retryable32byte execution failure; its previous versions remain frozen.

Parent preserved readonly job/attempt/release evidence, cancelled the stuck queued retry through the public expected-version transition, and verified every claimed native attempt released with no pending release/finalization. Direct paired ownerclear succeeded (5462testpoolrequests,436testrootwrites, primary counters omitted/default0); Test Genie cleanup returned cleaned for lease0cc48733-67ad-4271-a29a-f5158a8e3cdd. The high write count includes repeated persisted intents from the failed Start loop; preserve this observation for recovery/retention review. No active lease remains. Source fixes and a fresh replay are pending.

P5 test experiment's first517line revision exceeded500 before reporting; preserve that17line breach. A subsequent650line prospective reservation admits replacing the redundant inspection recheck with actual rawRGB+PTS transport and frame-boundary tests. No production video implementation is admitted by the experiment. Projection-only157ms/frame is not a full-render performance promise; the final codec preset, clip cap and full composition/encode/verification benchmark remain open.

### P4b bounded live acceptance and P5 implementation admission

Parent accepts the bounded inspection journey after frozen run3 and native-cancel
harness both exited0. Four jobs/six attempts reached known terminal states; every
claimed attempt has a durable release receipt, with no pending release or subject
finalization. Native cancellation observed the exact pinned worker in a populated
cgroup and actual2GiB/128tasks/300second limits, then verified quiescence and explicit
same-job retry to a ready tail-moov recording. It does not prove an FFmpeg child
was already active. Direct ownerclear returned926testrequests/71testwrites and
primary counters omitted/default0; Test Genie cleaned leaseb224171c-9580-44a1-bfd7-8ba7527a0fcb.
Parent reopened primary read-only: original project/alltablecounts/integrity/FKs
unchanged; API healthy. No active lease. Evidence gate: integration/p4b-live-parent-acceptance.json.

Parent independently qualified the641line P5 test under race:488 rawRGB frames and
PTS rows match the saved clock, fixed color conversion runs, and bounded synthetic
fragmentedMP4 encodes and fully decodes. Separate parent evidence preserves the
worker's results. First parent run failed only because its separate output folder
was absent; that failed log remains. No sample/final composition claim follows.

Admit P5a Renderer runtime work under api/internal/rendering: one VideoManifest/
VideoReceipt and strict private worker entry, shared decode/clock/color/composition
for samplePNG and finalMP4, hard encoded-output writer, native verification inside
the owned worker, pure-Go streaming artifact verification for API consumption.
Expected1650/cap2000 authored runtime lines; tests/evidence separate. Reuse existing
inspection/source/profile/image/presentation/composition primitives; no duplicated
lifecycle or ambient tool resolution. Capture exact baselines before writes.

Admit P5b Workspace shot/API integration under existing projects/shots/renderjobs/
handlers/main and canonical projects/shots proto: immutable timing and receipt/
clock bindings; sample result verified through a narrow injected owner seam and
explicit new shot revision; same jobs/pump/nativeoperation; streamed video output,
public mode/profile facts and bounded clock pages. Expected1100/cap1500 including
proto source, generated/tests separate. Workspace owns main dispatch wiring, new
native operation mode limits and proto owner generation. Coordinate exported types
with Renderer before dependent compilation; no API-side native tool execution.

Initial private development profile caps output at10seconds/300frames,1080p24/30,
exact-aspect adequate plate, audiooff, fixedcamera. Use medium/CRF18 for the first
export-quality benchmark, then measure a complete5second sample/final journey with
actual presentation/projection/encode/verification before public profile acceptance
or increasing duration. Sourcefullclock/decode bounds remain separate. Explicit
audio, continuousscreenentry, higher-detailstills/native3D and release scope remain.

Revise aggregate forecast8800/cap10500 before these writes: real inspection,
immutable staging/retry/finalization and independent output verification exceeded
the original narrow estimates. Conservative component ceilings (sharedacquisition/
SDA260, executor500, profile850, intake950, jobs180, nativeinspection1450,
integration1300, CLI300, P5worker2000, P5integration1500, remainingUI/CLI1100)
total10390 and include the later UI reservation; maintain actual disjoint delta
accounting. P5combined3500 cap and the later UI gate must fit the aggregate actual
census; report before crossing any admitted boundary. No code
compression, duplicated pipelines, or dropped verification to meet line counts.

### P5 presentation seam and publication review

Admit Renderer to make the narrowly required presentations owner change within
its existing2000-line P5 runtime cap: optional independent status/home ink
configuration, omitted from serialization when absent, consumed only by the
current supported profile. Capture exact baselines before edits; retain old
still hashes/behavior and add regressions for frozen opposite-contrast inks.
No second rasterizer or layout pipeline. Workspace owns receipt trust and recipe
validation; it shall distinguish explicit manual colors from sample provenance.

Independent Materials review found no retained complete source bundle meeting
strict1080p exact-aspect rules: accepted phone1536x1024 and laptop1672x941 remain
native-resolution references. Vendor previews have neither usable separated
sources nor qualified redistribution rights. Do not upscale, stretch or silently
reframe them to claim1080p quality. Native-resolution evidence or synthetic
protocol fixtures do not qualify the final photographic1080p look.

The publication contract now explicitly includes bounded stream/count/hash into
the routed immutable store, commit-gated visibility, unreferenced-artifact crash
handling, and the second durable160MiB output copy. Sample base-recipe
canonicalization is versioned and verified colors must match the trusted receipt.
These tighten existing P5 obligations without introducing another lifecycle.

### P6a recording-intake UI admission

P4 is accepted independently of the unfinished P5 renderer. Admit Materials to
extend the existing SourceMediaPanel and narrow API URL helper, selectors and
en/ar/ja strings with supported recording uploads, server-owned readiness and
inspection job state, expected-version explicit retry/cancel, read-only polling
while active, and original-source native media playback. Preserve approved
Light-table presentation and existing image verification. No512MiB JSBlob or
arrayBuffer path for video; no product testmode workaround. Use the existing
API-base URL resolver and BAS workflow extra_headers when later validated live.

Expected300/cap450 authored UI runtime lines including locale changes, charged
to the reserved1100 UI/CLI component (remaining650). Tests and governed generated
outputs accounted separately. Capture exact prewrite before edits. No shot/render
controls, API/storage/proto edits, lifecycle, leases, dependency changes or native
execution. Test image behavior, pending/ready/rejected/failure/cancel and conflict,
videoURL/noBlob path, bounded active refetch and playback failure. Native player
may buffer a full source before seek; no range-seeking qualification claim.
Report before crossing450. P5 source/live gates and later shot UI gate remain.

P6a type-check exposed intentional optional source dimensions in the current
verified proto. Admit Materials' narrow ShotWorkspace consumer correction within
the same450-line cap, after exact file prewrite: eligible plate/image controls
require ready image kind and present positive dimensions before bounds/overlay
or image-byte fetching. Do not default absent dimensions to invented zeros or
route newly imported videos through the old imageBlob path. Actual video shot
editing remains in the later gate. Latest proto owner verifier passed after
shared lock contention; no stale-generated consumer workaround is introduced.

Parent admits one bounded test-only medium-preset timestamp experiment (up to160
harness lines, no product edits): exact hash-verified managed pair, synthetic
1080p RGB24 frames through the previously qualified zscale graph, at most24frames
per case,30seconds per native command and8MiB per output/log via RLIMIT_FSIZE.
Compare default B-frame timing with a no-B-frame candidate only if an offset is
observed. Pin2codec/filter threads for the test; this is protocol evidence, not
an owned-cgroup performance or photographic-quality claim. Preserve exact argv,
tool digests, output/probe bytes and failed outcomes. Do not modify source videos
or fake original inspection receipts.

Parent's101-line medium timing harness SHA5586398f4e3abf022615b17411cfe0112248d39d2889cf706ce16f5a45a3d9ad
completed both bounded cases. Exact managed executable hashes matched. Default
medium/CRF18 fragmented output24frames had17Bframes and firstPTS1024 at1/12288
(twoframes offset); no-B-frame case started0 with last11776 and24frames. Select
explicit-bf0 with2codec/filterthreads for the first profile; Renderer notified.
Retain both outputs, argv and raw probe under video-protocol/medium-timing. No
full-render performance, owned-cgroup or visual acceptance is inferred. Two CC0
high-resolution laptop photo candidates were also downloaded/viewed and rejected
for representative craft (soft display/foreshortened reflection); review and
provenance retained in plate-candidates. No paid calls or asset qualification.

Parent owns an additional opt-in same-package native integration test
`video_native_parent_test.go`, at most500 test lines, to validate actual sample
and5second final execution through RunVideoManifest on the preserved authentic
phone inspection. Stage exact original manifest/receipt/clock/tool profile into
isolated evidence workspaces and hash current test worker. Use a synthetic1080p
plate/mask to isolate functional correctness and keep clear that this is not a
photographic look. Sample on revision1, accept derived fills/inks into revision2,
require120frames24fps, zero-start verifiedfull decode, equal canonicalfirstpixels,
and unchanged protected plate region in the sample. Preserve manifests/outputs/
receipts/elapsed times; no service lifecycle or live SQL/file lease. Each worker
uses its300second internal deadline; test deadline12minutes permits both jobs.
Run only after source freeze and required focused regressions; source hashes and
measured scope accompany results. This supplies code-path timing, not native
cgroup limits or end-user performance acceptance.

### P5b measured budget adjustment

Workspace reports current authored delta687 (existingfiles/proto292 plus three
new files395), before a complete immutable closure/adapter. Realistic remaining
650–850 gives1337–1537. Prospectively raise its runtime cap1500→1800 and expected
1100→1450 to complete staging, original-inspection provenance, sample verification
and streamed publication without omitting endpoints or compressing code. P5a
remains2000; combinedP5cap3800. Aggregate forecast9100/cap11000; component ceilings
now10690 including the full1100 UI/CLI reservation. This is source-work admission,
not acceptance of temporary adapter errors or a render result. Final disjoint
runtime census and tests remain required before lifecycle/live work.

### Parent native result and additional bounded matrix

The exact frozen P5a source in p5a-worker/parent-native-run-1-source.json passed
actual sample and final RunVideoManifest execution on the authentic phone. Sample
2.571seconds; final75.507seconds for five output seconds at1920x1080/24fps,
120frames, firstPTS0, last60928 at1/12288, full decode verified. Revision1 sample
and revision2 final share canonical firstRGB6103002c...689548. Protected corner
pixels are unchanged. Retain all input/manifest/output/receipt bytes and result.
This is a synthetic-plate code-path measurement, not a photographic look, owned
cgroup or end-user performance acceptance. P5a runtime1824/2000 is frozen pending
independent review; P5b remains incomplete.

Admit Renderer to extend the same opt-in test harness (parent originally authored
213lines, total cap500 test lines) for one additional bounded case:1080x1920 at
30fps, one output second, nonzero trim and2x playback, from an exact-aspect larger
synthetic plate, using the preserved original phone receipt/clock/profile. Reuse
helpers rather than copy the whole pipeline; preserve the original five-second
case and its result. Require sample/final parity, native zeroPTS/count/rate,
uniform downsample and protected pixels. Use fresh evidence directory and pin
source hashes before execution. No product runtime edits, service lifecycle,
leases, real-scene qualification or source alteration. Run only the added case;
do not repeat the passed75second run unless a relevant runtime defect changes it.

Protocol evidence clarification: commands.json profile_artifact_sha2563737acd3...
is the SHA256 of the retained profile.json bytes (independently rechecked), while
its artifact_identity2ad07484... identifies the paired executable profile. These
are deliberately different identities; no artifact replacement was discovered.

P6a source accepted by parent: exact enumerated runtime hashes matched accounting,
158netlines/450cap, all38focused UI tests independently passed. Agent type-check
and changed-file ESLint passed. Parent source review confirms videoURL/native
playback, explicit versioned mutations, target-scoped conflicts and read-only
pending projection refresh. No live UI proof yet. Parent renderer/presentations
race also passed (3.921s/9.645s); independent adversarial review remains active.

### Native/API boundary correction before integration acceptance

Parent found a real mismatch hidden by the synthetic fixture: the worker compared
sample full RecipeSHA256 to final BaseRecipeSHA256, but shots deliberately computes
these with different canonical schemes. Compare sample and final base identities;
retain a separate binding to the sample full recipe. Preserve the prior native
pass as its actual narrower case. Update fixture to use unequal full/base hashes
and rerun it after this relevant runtime change. Workspace must bind sample's old
revision, complete verified color/ink/source-derived fields and actual shot ID,
and accept an unchanged already-pinned revision without treating matching fills
as an edit. Output orientation follows the scene plate, not the device display.

Independent Materials review otherwise found the14earlier draft issues addressed,
but identifies sample closure omission: source+ordinal/PTS clock hashes alone do
not pin receipt/timebase/stream/EOF/profile. Reopen Renderer runtime within its
existing2000cap for a canonical full inspection-closure digest in sample receipt/
binding and comparison to final inspection. Add same-clock/different-timebase or
receipt rejection. Also admit a narrow exported sample-frame planner returning
ordinal/PTS through the existing clock parser and composition.PlanFrames; API
must not duplicate timing. Capture exact frozen prewrite. Tests/evidence separate;
no new pipeline, dependencies or lifecycle. Coordinate with P5b and report before
exceeding cap. Delay new native matrix run until these contract fixes are frozen.

Clarify the existing shared-sample contract before final source freeze: trim-start
PNG look previews must support browser/desktop and explicit manual presentation,
not only AUTO phone colors. Renderer may remove the artificial AUTO-only sample
restriction, validate pinned layout for other supported cases and use nil-safe
resolvedflags within its2000cap. Workspace compiles the corresponding layout and
removes phone-only API rejection within1800cap. Only AUTOphone color acceptance
requires newrevision/sampleclaim; manual/browser preview must not invent source-
derived provenance. Focused tests cover both paths. No additional pipeline.

### P6b workbench and CLI admission

Materials read-only proposal estimates650–900runtime lines for timing/sample/
export UX and CLI. Admit UI/APIhelpers/selectors/en-ar-ja locales within750lines
(expected600) and parent CLI within360lines(expected250), tests/generated separate.
P6a is frozen158; wholeUI/CLI component revised1100→1300 prospectively, aggregate
forecast9300/cap11200 (ceilings10890). The extra CLI allowance includes exact RPC
parity and semantic manifest bindings exposed by current CLI Health, not a new
HTTP client or pipeline. Capture exact prewrites before edits and report caprisk.

The existing Light-table owns the flow: readyrecording selection; frame-clock
paged exacttrim/end and rationalrate; savedrevision samplePNG; explicit AUTOphone
acceptance into newrevision; finalMP4 to thatrevision. Browser/manual lookpreview
uses pinnedsettings and needsno colorprovenance. Editing source/timing/scene or
presentation invalidates sample acceptance; clear only prior source-derivedcolors
when returning toAUTO, preserving explicitmanualchoices. Staleclockpages and late
sample responses cannot attach to a new draft. Original nativeplayer is navigation
only, never timestampauthority. Existing stills/fit/chrome/savedhistory preserved.

Parent CLI reuses cli-core's header/auth/base-resolving HTTP client for rawcontent
streaming; generatedConnect for samplebinding/clockpages/submission. Download into
boundedtemporaryfile, count/hash and validateformat, expose finalpath only after
verification without overwriting existing destination. No fullMP4buffer. Keep
stillPNG defaults. CLI manifest bindings use actual requestfields and structured
JSON-file decoding. Relevant focusedtests+CLIHealth, followed by shared scoped
TestGenie/nativeAPI/UI validation after all source gates. No livelease/lifecycle
admitted by this source boundary.

### P5 public journey harness preparation

Admit Renderer to prepare at most500test-harness lines under evidence/E2V/p5-live
for a parent-prepared paired lease; no live requests or lease/lifecycle commands.
Use explicitloopbackAPI/project/installedCLI arguments and test routingheader,
boundedrequests/polls/output, freshevidencedirectory. Import exact authenticphone
plus retainedsyntheticplate/mask viaCLI; inspectreadyclock488, create typedAUTO
shot, submit sample twice withsamekey, awaitterminal/gettrustedbinding/exportPNG,
reject changedclaim and crossshot claims, saveacceptednewrevision thenresave,
submit final/exportMP4 withCLI andverifypublichash/count/typedmode. Preserve request/
response/failure evidence and resultingjob IDs forparentnative/release inspection.
Successfulharness doesnotclaimrealisticlook,cgroupcancel/restart orfinalrelease.
Use currentgeneratedschema/realpublicAPIs; no inventedtestendpoint, primaryroute,
manualSQLwrite or source mutation. Parent ownsleaseisolation/preflight/cleanup.

### Parent final API/CLI source gates

Parent independently verified all14 P5b frozen source hashes before and after a
nine-package race run including actual main dispatch, rendering/presentations,
shots/projects/renderjobs and their handlers. All passed; evidence
p5-parent-final-source-gate.json and p5-parent-final-race.log. P5b1563/1800;
P5a1875/2000. P6b CLI301/360 passed full race and CLI Health after declaring export
as a local filesystem write backed by ProtoMutation. Manifest bindings, bounded
verified streaming and no-overwrite behavior are included. Exact census/hashes
in p6b-cli/parent-final-source-gate.json. UI remains under final review; no P5/P6
lifecycle or live journey yet. Synthetic native proof does not qualify scene craft.

### Live integration admission after final UI source review

Once P6b UI hash/census and focused checks pass, parent may use supported scenario
restart and install a fresh Test Genie paired SQL/file lease. Before any fixture
write require target eligibility, unchanged primary logical rows and empty native
job queues, verified lease identity/file leg, and tagged-versus-primary read
discrimination. No-lease tagged reads currently return primary; they are not
isolation evidence. Test Genie owns lease heartbeat and cleanup. Keep cleanup
token private, capture only run ID in public evidence. Use retained public harness
and authentic source, then capture desktop/mobile workbench through BAS under the
same routed lease. Run native cancellation only with actual child observation;
confirm all jobs quiescent/released before owner clear then Test Genie cleanup.
Retain failed attempts and compare all primary tables after cleanup. This admits
local validation, not publication, paid calls or final release certification.

### First live P5 failure and bounded abandonment

Supported restart served build16ae44fb…05156e. Fresh TestGenie paired lease
 a46d47ea-816c-4434-9db7-8252f4dc07d5 discriminated seeded project from primary;
installed CLI read proved routing. Harness run1 sent forbidden server-derived
viewport dimensions and was corrected with original bytes preserved. Run2 then
found a real adapter defect: staged inspection-manifest.json collides with the
executor-reserved filename. No native operation was dispatched. A second defect
leaves that failed attempt release-pending because failBeforeNativeStart omits
the existing dispatch-owned not-dispatched proof. Workspace repairs both within
P5b1800cap; exact final-source prewrite and meaningful staging/failure regressions
required. Do not infer a lost native outcome from absence or forge old proof.

Preserve the complete logical fixture database and small pinned manifests/clocks
in p5-live/run-2-preserved. Parent observed exact intended unit not-found and no
matching worker/manifest process (failed-predispatch-host-observation.json).
Permit abandonment of this failed isolated fixture after all browser readers
finish and verifying every actual inspection operation is terminal/released.
Owner clear then TestGenie cleanup is test disposal, NOT a successful release
proof for the stranded attempt. Keep its pending ledger in evidence. Rerun on a
fresh lease after source fix and supported restart; do not retroactively rewrite
its missing proof or claim this failure as E2V acceptance.

### UI runtime and shared lease interruption

Parent UI gate independently passed43tests plus typecheck; exact382netruntime
lines matched all10hashes. Supportedrestart deployed build16ae44fb…05156e.
Desktop BAS observer48a78fe1-5b88-4c0e-8031-84cce4ee452f captured the saved shot and
new frame controls in the approved Light-table shell. Source preview errored
although mediaHTTP200; separate observerb96a69eb-24a3-434d-969c-fe1ead3ef874 reports
BAS Chromium120 lacks H264 canPlayType support (VP9/AV1 available). This strongly
suggests a browser codec limitation, not proof of source transport corruption.
Final browser playback remains unverified.

External TestGenie restart10:49:45Z stopped its seed heartbeat; target lease expired
10:50:46Z. Mobile observer83129e05-518d-4b7a-a928-e01a741c0a56 saw404afterexpiry and
is invalid as UI assessment. Onlyread requests occurred afterward. Ownerclear200
reported1868testrequests/47testfilewrites/5primaryduringtestmode reads; exactprimary
logicaldata unchanged and filetree removed. TGcleanup404sessionnotfound leaves
its isolatedtemporarySQLdatabase as ownercleanupdebt, reported through report-bug
knw-1792493762515810444. No active targetlease, no manualSQLwrite/deletion. Retain
failedattempt releasepending as historicalfailedtest, not success. Nextfreshlease
must be monitored for liveproducercontinuity; neverrevive the expiredroute.

### P5 fixed-source verification — 2026-10-20

Parent confirmed P5b 1,713/1,800 physical runtime lines (the worker estimate 1,699 was inaccurate). Renderjobs, handler and rendering races pass. Scoped Test Genie unit run `20261020-110202-4f5f37b1` remains FAIL on the known `trackingMediaRPCStub`/`SliceRepo` validator false positive; two environment-seam warnings are advisory. No source workaround or green-suite claim. See `evidence/E2V/p5-live/fixed-parent-source-review.json`. Supported restart `startop-1c84d92fe074268832248b9e5e83b38a` is healthy and primary logical data is unchanged. Fresh paired test lease is being admitted before the corrected public journey; no video completion claim yet.

Public run3 launched the real Go video worker and failed its manifest validation: the adapter omitted `CompositorProfile`. FFmpeg was not reached. Native stopped-outcome evidence and successful release are retained in `p5-live/run-3-preserved`; both pending ledgers are empty. Lease `8180ae1a-f73b-4281-bb06-5d8785db7068` was owner-cleared and Test Genie cleanup succeeded; primary data is unchanged. A minimal compositor fix plus shared worker-contract validation before dispatch is admitted within the P5b 1,800-line ceiling. No sample/final completion or browser proof yet.

Public run4 produced a real 1920×1080 sample (488,499 bytes, SHA `8c3d857826f3b506cfd0d165988e16b46e2ecc87032411b451caa0ffc8fbcee6`) and exited zero. Coordinator publication then rejected its native evidence because the reused verifier hardcoded inspection mode. Parent review also caught the next save-path defect: `VerifyVideoSample` looks for the completed video manifest in job input instead of completion outputs. Both repairs are admitted within P5b1800 with actual-shape regressions and an independent remaining-chain review. Native failure release settled; leaseab019214 ownerclear/TGcleanup succeeded and primary remained unchanged. See `p5-live/run-4-diagnosis.json` and preserved native sample/receipt; this is not a successful public render.

### Deterministic queued-failure repair admission

Public run5 now proves native sample publication, accepted/resaved revision3, restart export identity, and rejection of receipt/source/trim/cross-shot tampering without head changes. Final submission exposed omitted saved phone status/home ink in the worker config; the presentation mapping is fixed and focused races pass. The invalid immutable final closure was cancelled without native execution and retained unchanged. A new final submission must use a new idempotency key after deployment.

Prospectively raise P5b runtime ceiling1800→1900 (latest independently counted1733; expected repair55–90 lines). Aggregate cap11200 remains unchanged. Admit a separate transactional pre-dispatch failure transition only for current queued job/current queued attempt with no execution claim, with version/route fences and atomic subject-finalization marker. Renderjobs may use it only for typed deterministic closure/manifest validation failures. Transient I/O/provider/route/persistence failures remain recoverable. Do not broaden Running-only Fail, fabricate native/quiescence/release evidence, add a second pump, or classify errors by text. Meaningful regression coverage must include deterministic rejection and retry, transient queued retention, and concurrent claim CAS. Preserve exact prewrite files and report census before exceeding this allowance.

### Public fixed-video checkpoint — run6

Public API/CLI import, native inspection, AUTO sample publication/export, accepted revision2, unchanged resave3 and final MP4 export passed. Final job80c90785-44ac-4a72-87c3-b044493c6d6f exports230091 bytes, SHA3a3e6b785987465821eadb1b4464f224eee49a3eeaf0880559207e8a0db72aaf. Independent pinned-tool probe/full decode verifies1920×1080 H264/yuv420p BT709,120frames at24fps, every rational output PTS, and exact sample/final pre-encode RGB digest parity. See p5-live/run-6 and run-6-output-verification. Synthetic scene only; no photographic craft acceptance.

Cancellation joba5124625-41d9-4cf3-abfd-b9245ecc0c28 was observed with the exact worker and hash-matched FFmpeg descendant in its owned cgroup, then cancelled through installed CLI. Exact version/input/intent/native invocation quiescence, terminal CANCELLED, durable release and zero pending ledgers passed; see cancel-proof/live-run-1. Parent observer394lines and SHA c3254c14aa9d5c1538423b298d0bb946bac8c2cb547cfc6ecfe5afac5e04879e.

Deterministic manifest repair source accepted at1788/1900 P5b (+55 independently measured), worker full jobs/renderjobs race and parent focused race pass. Saved ink change independently measured +1 net physical line, superseding preliminary +5 report. Supported restart-3 resumes paired leasefd9ac435; final replay verification is active. Broader browser/portrait/rate/camera-entry/photographic obligations remain open.

Run6 restart/replay PASSED: preserved export, idempotent submit and new final job produce identical MP4 bytes. Workspace independently reviewed source clock/output/failure transition and actual receipts without a concrete material finding. This is bounded checkpoint review, not either final-release adversarial pass. Leasefd9ac435 was owner-cleared and Test Genie cleaned up; primary logical census unchanged.64 bounded artifacts plus SQLite retained under run-6-preserved.

### P7 screen-entry boundary admission

Fixed public protocol checkpoint is complete; the original E2V exit gate still includes camera entry. Earlier aggregate estimates omitted its final implementation cost. Prospectively revise forecast9300→10650 and cap11200→12500; preserve prior estimates as history. Existing component ceilings10990 plus separately admitted saved-ink12 and P7 ceilings1330 total12332, below12500. No library/native3D/audio scope is silently accepted by this adjustment.

Target: rendering.md photographic screen-entry contract. Parent reviewed the aspect-safe direct-app handoff design; implementation may proceed only after the small shared API/schema contract below is frozen between owners. Renderer owns composition projection/warp, native manifest/validation/frame loop/verification, expected500/cap750 new runtime lines. Workspace owns protobuf source through its governed generator, shots compiler/recipe and renderjobs mapping, expected250/cap400. Materials UI controls/admission follows the frozen API, expected100/cap180; no UI writes before that interface exists. Tests/generated output/evidence counted separately; preserve exact prewrite files and halt/report before exceeding a boundary. Aggregate caps do not authorize unrelated changes.

Shared contract: ShotSnapshot field5 `camera_motion` of type `PhotographicCameraMotion`; profile ID/version, duration_output_frames and required typed matte RGB. Native optional Camera omitempty uses the same frozen values. Existing fixed profile is unchanged; new screen-entry profile identifies camera jobs. Duration bounded2..120 and <=output frame count; caller-supplied target rectangles/keyframes/argv are excluded. Model/scene/device root motion remains a later native3D route. Direct app endpoint contains intrinsic AppViewport with matte, not arbitrary full-output stretch. Preserve old fixed canonical byte fixtures.

Validate meaningful geometry and timeline regressions first, then scoped current-host public leased journey and pinned-tool full decode with endpoint comparison. Benchmark the extra raster pass before accepting the existing ten-second/300-frame upper bound for this profile; retain a lower explicit camera-specific bound if evidence requires it. No lifecycle changes while an agent writes source. No new dependencies, paid calls, generated-file hand edits or primary test mutations.

P7 interface follow-up: native profile `device-studio-screen-entry-video-v1`; camera subprofile `screen-entry-projective-contain-v1` v1; proto reuses `PresentationRGB` for required matte_rgb (explicit black valid). Renderer may add narrowly scoped `presentations.BuildVideoViewport` within its750 cap, returning only rectangle/error and reusing layout math without returning/persisting placeholder color or hash evidence. Test color-independent geometry and config nonmutation. Camera-only scene plate dimensions are optional/omitted for legacy fixed jobs and must equal decoded plate bounds. The governed SDA local proto refresh updated the existing iframe-bridge optional React peer resolution to already-selected18.3.1; retain owner-produced lock delta and exact invocation/diff evidence, do not hand-edit the lock or attribute unrelated generated changes.

P7 initial performance bound: camera total output <=120frames, separately from fixed300; transition2..120<=clip. Apply the limit in shot eligibility, native preflight and editor guidance before dispatch. This is an initial profile limitation pending actual extra-warp runtime measurement, not acceptance of all duration/output combinations.


### P7 source and first public camera checkpoint

P7 source accepted after parent hash review and independent race tests for composition,
presentations, rendering, shots and renderjobs. Runtime growth: native 557/750,
integration 107/400 and UI 96/180; UI parent 39 tests passed. Exact sources and prewrites
remain under evidence/E2V/p7-camera. Supported restart served worker
cf4eaa75c4e562b6c39040a25b5a089ed00bfe64ed619371e9d4d64cfea7ebdf.

Public routed run1 succeeded through import/inspection, sample, stale-camera sample
rejection, accept/resave and final export. Final job 4866ca5a-c4e4-49d3-839f-7c970791c3f2
exports 582,612 bytes SHA d36a1d8a3141c246c05808eb6babc793cbfaa22b8530b7f20ef76245c74d37dd.
Independent full decode verifies 120 frames at 1080p/24 fps and every output PTS. Sample PNG bytes
and first-frame RGB digest equal the retained fixed run6 sample. Camera manifest and
receipt agree on 48 transition frames and matte RGB (17, 20, 28). Observed submission-to-finish
is 101.823 seconds for the final, 6.077 seconds for the sample; one synthetic scene,
not a throughput promise. Endpoint pixel and live UI checks remain in progress.

Cancellation job 2d2c9146 was observed with exact worker plus hash-verified FFmpeg descendant,
then cancelled once through CLI. The first observer returned exit 2 at its immediate
ledger read after public CANCELLED. A separate read-only settlement followup verified
the same native invocation quiescent, exactly one durable release and no pending
release/finalization. Preserve both the early observation failure and later proof;
no second cancel or product success rewrite. See public/cancel-proof/settlement-followup.
Lease db46617b remains active for browser and restart/replay checks; do not cleanup
while observers are open. This checkpoint does not accept the full E2V output matrix.


P7 sampled-pixel validation passed after correcting the independent verifier's
coordinate mistake: destination FittedApp coordinates cannot crop the original
source frame. The corrected run4 compares the full clock-selected source, using
the pinned input color policy and fitted scaling, to output hold frames72/119.
Mean absolute RGB errors are2.668/2.650 versus16.246/15.945 against frozen source0.
Frame0/sample lossy MAE is2.005, p99=5; matte edge error is at most2 levels. Parent
visually reviewed transition24, endpoint47 and hold119: endpoint excludes simulated
status/home chrome and retains uniform aspect/letterbox with continuing app content.
All prior verifier runs are preserved as superseded; no product change was needed.
This is sampled corroboration, not independent identity of every lossy source frame.

P7 UI desktop1440×1000 and mobile430×932 current/head3 plus historical2 read-only
captures completed and browser readers closed. Parent verified six screenshot
hashes and first-fold layout. BAS original-video playback remains unsupported;
console/network telemetry was not captured, so no clean-console or playback claim.
The lease checkpoint/restart/resume succeeded, and a fresh camera render replay is
running. Do not infer completed replay or cleanup from these preparation receipts.

Next evidence-only preparation is a data-driven extension of the existing public
journey for missing aspect/rate and desktop-browser cases. Workspace proposes the
case contract and bounded source delta before any harness edits; Materials locates
existing portrait fixtures and desktop source facts. Renderer investigates supported
current-host H264-capable browser automation. No product runtime, dependency, lifecycle
or lease mutation is admitted to those workers by this preparation.


P7 restart/replay and cleanup are complete. Preserved export, original idempotency
key and fresh post-restart camera render all retain exact MP4 bytes. Browser readers
were closed;71 bounded native artifacts and SQLite are archived. Owner clear200 and
Test Genie cleanup200 closed lease db46617b; original primary logical data is unchanged.
The bounded P7 camera checkpoint is accepted with the limitations in its README.
Full E2V and release deliverable groups remain open. There is no active test lease.


### P8 public output-matrix evidence admission

P7 is accepted as bounded synthetic protocol proof; the E2V exit matrix remains open.
Admit evidence-only extension of the existing public-video-journey.py to accept one
explicit --case-json per invocation. Preserve its446-line prewrite. Prospectively
raise its previous500-line ceiling to600 (forecast about535) for case validation,
clock-derived trims and phone AUTO versus manual/browser revision behavior. This is
verification-harness growth, not hidden product runtime scope. Workspace owns that
script and case contract. No duplicate journey, renderer, queue or source clock.

Case data pins source/plate/mask paths, hashes and sizes; expected inspected raster
and optional known clock facts; scene surface and plate-raster quad; presentation;
output rate/raster; trim start ordinal and source-frame count; motion and optional
camera transition/matte. Verify every returned clock page and reconstruct its canonical
binary digest against the immutable inspection receipt. Phone known clock facts remain
exact. Desktop facts must come from fresh owned inspection before final qualification;
never copy phone clock assumptions. Derive trim endpoints from the verified clock vector
and exact source EOF. Keep default legacy behavior and all explicit test-route checks.

Run short two-second cases: phone landscape30 fixed/camera, phone portrait24
fixed/camera, and desktop browser landscape30 fixed/camera. Camera entry lasts24 output
frames; whole clips remain within120. Phone AUTO retains sample claim acceptance,
resave and stale/cross-shot rejection without head advancement. Manual/browser cases
preview and final-render the same saved revision without invented derived-color claims.
All exports require public identity/hash/size/raster/type checks and independent full
decode/PTS inspection. Parent reviews the prepared harness/cases before live execution.

Materials may create one deterministic synthetic wide-screen plate/mask pair, plus
case JSONs agreed with Workspace, under evidence/E2V/p8-matrix; at most100 lines of
fixture-generation code, no dependencies or runtime edits. Reuse the existing portrait
2160×3840 fixtures and exact quad for1080×1920 output. A browser calibration surface
is not a laptop model or photographic craft claim. Original research sources stay intact.
Only parent may create/clear a fresh paired lease or perform lifecycle operations.


Audio scope discrepancy for the next contract review: the epoch prose calls explicit
passthrough an implementation obligation, while canonical DS-SCREEN-AUDIO requires
silent default, original-source provenance and declared supported policies; its
verification case explicitly requests an unsupported audio mode and expects an
explanation. The accepted eight-deliverable agreement does not promise an enabled
passthrough codec. Do not silently expand or remove scope based on this discrepancy.
Before an audio implementation boundary, reconcile the canonical requirement and
presentations.md conditional qualified-policy wording, then retain an audible-source
silent-default/rejection test. Current native '-an' and video-only tests alone do not
prove that complete requirement. No audio source edits are admitted here.


P8 verifier reuse admission: Materials may extend the existing P7 public/verify-export.py
(default invocation retained) with explicit run/route/output/case arguments, up to220
physical lines after preserving its prewrite. No parallel verifier implementation.
The manual/browser branch locates the independent sample attempt receipt and compares
first-frame/layout facts without requiring or inventing an AUTO sample binding.
Probe/full decode must compare exact per-case raster/rate/frame count/PTS and native
receipt fields, pin tool/input/manifest/output identities and preserve camera config
only when present. Parent reviews source before live use; fixture-case bytes are
pinned in each run. This is evidence-only code, no product runtime change.


P8 parent review refinement before live execution: both desktop cases now start at
source ordinal30, preserving the60-frame span and same six-case matrix. This exercises
nonzero trim while keeping fixed/camera inputs identical. Exact changed hashes are in
p8-matrix/cases/nonzero-trim-review.json. Phone cases remain zero-start. Harness now
checks exact rational planned output count before CreateShot and pins original case
bytes in each run. Browser caller-derived viewport fields were removed before use;
manual colors were aligned with the dark appearance profile. Those rejected fixture
revisions are retained as review history, not product defects.

Verifier ceiling prospectively220→240 from current210 for concrete parent-found gaps:
retain explicit H264/yuv420p/BT709 limited expectations, compare camera fields across
manifest/final/sample receipts, hash the staged worker, bind run-input case hash, and
cross-check declared output dimensions. Prefer readable assertions to line compression.
This adds no product runtime or new workflow. Parent final source review precedes lease.


P8 narrow runtime exception admitted before live execution: public sample-binding
requests for manual/browser previews currently map domain.ErrOutputUnavailable to
INTERNAL. Renderer may add the canonical sentinel mapping to FAILED_PRECONDITION
(12 changed/new runtime-line ceiling, expected two), with focused sentinel, wrapped
sentinel and unrelated-internal regressions. No domain behavior, lease or lifecycle
changes delegated. Parent deploys before creating the P8 lease. Workspace may extend
the existing harness ceiling600→620 to assert this actual public refusal and stable
shot head, and correct case camera-duration evidence metadata. This remains within
the aggregate epoch runtime allowance.

Parent source review also caught the P8 verifier reading manifest before assignment
in its case-only worker hash assertion. Materials must correct ordering and exercise
the case branch before live use; retained P7 success does not cover that branch.


P8 live started after handler race pass and supported healthy restart. Fresh paired
lease ca0c6643-07b7-445c-b481-ab0f15dffa8c; primary/test/CLI routes discriminate.
Landscape30 fixed and camera public journeys plus independent60-frame decode/PTS
passed. Materials may reuse existing P7 pixel primitives with at most160 lines of
case-specific evidence glue for extracted0/23/24/47/(59) frames and sampled geometry/
source correspondence, using actual case receipt dimensions/colors. No additional
runtime or live API mutation. Parent visually reviews extracted endpoint/hold frames.
TGunit20261020-125441-b73f0101 failed on the same previously reported validator
COMPANION_REIMPLEMENTED finding (knw-1792486584923841356); focused handler race passes.


P8 pixel glue parent review: landscape endpoint23 visually shows app-only centered
containment; frame0/sample and hold comparisons are measured. Generalize that same
144-line evidence script (prewrite retained), ceiling160→200 prospectively, for
case-dir/output flags, both-axis AppViewport-relative crop geometry, per-receipt source
color transformation and trim-start baseline instead of source0. Manual preview receipt
selection and dynamic output rate/frame count are required. No duplicated checker.


P8 desktop fixed run1 exposed a real native-worker defect: public browser sample
succeeded and GetVideoSampleBinding correctly returned FAILED_PRECONDITION, but final
job0d2b4c21/attempt962328dc panicked at readAndVerifySampleReceipt:1304, called by
RunVideoManifest:836 despite legitimate nil Sample for manual browser. Exact journal
and immutable failed attempt remain retained. Hypotheses were final sample requirement
versus layout/manifest mismatch; journal establishes unconditional nil dereference,
not fixture geometry. Admit narrow rendering-owner guard and meaningful failing-first
manual-final regression, max20 runtime changed/new lines, preserving AUTO verification.
Parent owns one checkpoint/restart/resume for this lease after tests; no blind retry.


P8 protocol matrix finished and lease cleared: allsix exports decode/timestamp-check,
allpaired preview bytes identical, but browserendpoint appearance notaccepted. Parent
and independent source review confirm rounded-window outline remains in AppViewport
letterbox because fullbrowser raster is cropped fordirecthandoff. Source cannotcause
those pixels: its FittedApp is strictly interior. Admit correction in presentations
owner plus minimalworkerhandoff,max80 runtimechanged/newlines, preservingprewrite.
Cleanactiveviewport uses shared layout/contain and flatbrowserWindowColor/default or
phoneMatte; no OS wallpaper/chrome. Fullpresentation continuesphotographiccomposition
and exactframe0. Reuseexistingcolorresolution; no crop tohideborder/no newtimeline.
This corrects v1 app-onlycontract, notnewmotionsemantics; workerSHApinscode/caches.
Tests: failingbrowsercornerregression, sourceedgemarksretained, phonecrop parity,
fixed/frame0unchanged. Parent source/race checks and freshleased publicbrowsercamera
verification required. Previousleaseclosed; nextnative validationneedsfreshlease.


Independent browser-endpoint regression measurement: Materials may extend the same
177-line pixel evidence glue to230 lines after preserving prewrite. Measure viewport
letterbox pixels (excluding two output pixels near source/outer boundaries) against
explicit browserWindowColor/default or phoneMatte/default; record mean/max error and
counts above20 without including sourceapp pixels. Optional route-json points at the
retained archive rather than rewriting old routing evidence. Retain pre-fix corner
measurements; compare fresh post-fix output later. No runtime or API mutation.


P9 silent-audio qualification admitted after contract reconciliation in
p8-matrix/audio-contract-review.md. Fixture preparation measured a real 1kHz tone
between0.5 and1.5sec with surrounding silence; native/public proof remains required.
Use receipt-only version2 for new inspection receipts; manifests/protocol/clock stay1.
Accepted v2 requires nested audio facts version1; legacy v1 must omit them and remain
byte-identical. Rejected receipts omit facts. Validate bounded unique original indexes,
nonempty codecs, and no selected video index in audio facts. Preserve untouched source.
Renderer owns inspection native types/parser/verifier only, expected100/cap180 runtime
changed/new lines. Parent owns projects/projection/proto/shots integration expected200/
cap320. Materials owns existing timing-editor audio notice expected35/cap70 after
schema freeze. Proto generated files excluded; proto source included. Tests separate.
Aggregate epoch ceiling12500→13200 prospectively for omitted audio provenance/policy
acceptance, not passthrough implementation. Retain all prewrites and historical bounds.
UNSPECIFIED remains implicit silent without normalizing old recipe bytes; explicit
SILENT supported; PASSTHROUGH saved with typed repair/block before any job/native claim.
Unknown policy rejected. No dependencies/schema migration/second timeline or executor.
Focused compatibility/invalid-facts/policy tests plus one fresh public audible fixture
journey must prove original digest unchanged, default no-audio output and unsupported
request rejection. UI read-only detected/zero/unreported source audio and silent output.
No product completion claim until live proof and parent review.

P9 boundary review found Projects accepted looser codec tokens and missing selected
video identity than the native verifier. Parent aligns validation and regression
fixtures. Valid index pointers then exposed pre-existing projection replay comparing
pointer addresses after JSON reload; replace that equality with deep value equality
and assert legacy/knownzero/populated audio projection replay. This is inside the
admitted320-line integration boundary; media_sqlite.go prewrite captured beforeedit.

P9 publicaudible journey passed oncorrected taggedfixture-v2. Originalfixturev1
was correctly rejected missingmatrix and retained. Live review found generic render
blocker still saysstillrender forvideo; parentcorrects messagewithin320integration
bound. UI may add conditionalUsesilentoutputrepair for savedPASSTHROUGH, not an
enabledpassthroughselector, withlocalized text and regression inside70linebound.
Parentwillverify newmessage/repairafteronecontrolledcheckpoint/restart ifleaseheld.


### P9 bounded acceptance — 2026-10-20

Known-audible tagged fixture-v2 passed the public inspection/sample/final journey.
Original source SHA256656e9884…2c10 and PCM stream remain unchanged; final
SHA256ee93652b…fc63f is silent H.264,1920×1080,60 exact frames at30fps.
Unsupported PASSTHROUGH is saved with a typed blocking issue, SubmitRender returns
failed_precondition, and ledger counts remain4jobs/4attempts. One supported
checkpoint/restart/resume verified the corrected video-neutral error wording and
unchanged export. UI “Use silent output” saved revision2, explicit SILENT, no issues,
renderEligible=true. Parent independently read the API and viewed desktop/mobile.
The repair BAS run failed only its final native-DOM selector assertion after the
save; preserve that failure, later successful read-only mobile proof and API readback.
No additional render was submitted. Focused native/domain race and UI checks passed;
Test Genie unit20261020-133906-431b9dc7 still fails the known companion-stub classifier
(knw-1792486584923841356), so no green-suite claim is made.

Evidence: p8-matrix/audio/{public-proof/live-run-1,ui-live,public/restart}.
Parent archived87 files including SQLite,35,149,556 unique native bytes, all4released
attempts and zero pending release/finalization. OwnerClear200 preceded TestGenie
cleanup200; primary unchanged and test file root removed. P9 lease42402103 is closed.
This accepts the first-profile silent-audio contract only, not passthrough, narration,
photographic1080p,3D or the full release. Original rejected fixture and failures retained.
