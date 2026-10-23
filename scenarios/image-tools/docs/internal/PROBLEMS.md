# Problems — Image Tools

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-10-21 — Hybrid scene preparation and inference lifetime

Operator direction: support scene-only, photographed-device and geometry-first
routes, proper segmentation/geometry/object reconstruction, governed model
selection, and release unused model resources. Target:
`docs/reference/layered-photography.md#hybrid-device-scenes-target-and-current-limits`.

Work shape: bounded engineering foundation with a durable continuation record;
the complete hybrid pipeline remains an explicitly tracked target. Harness goal:
work until the affected native model execution paths have bounded residency,
cancel-safe worker teardown, correctly prompted SAM masks, and focused passing
regressions. Then report each remaining hybrid stage and its qualification gap.
Boundary: Image Tools owners and Device Studio integration documentation;
control-plane fixes only when a reproduced owner defect requires them. Preserve
the operator-rated Aquila film. Targeted tests and relevant Test Genie phases;
no full certification claim. Blocked means a missing decision, credential or
approval; diagnose ordinary engineering friction. Checkpoint here on interruption.

Full delivery approved 2026-10-22. This supersedes the foundation-only finish
line above. Work until each supported source route produces an editable,
inspected physical scene through native surfaces; interruption recovery and
resource release pass; the accepted Aquila device sequence has a reviewed
replacement. Canonical design: the approved delivery contract in
`docs/reference/layered-photography.md`. Proof is per-route retained inputs,
actual outputs, focused behavior tests and affected Test Genie phases, followed
by review of the encoded film. Preserve unrelated dirty work and accepted assets.

Original delivery sequence (historical checklist; current evidence is summarized below):

1. Shared scene contract, source manifest and known-camera laptop/phone corpus.
2. Geometry-first physical sample through Device Studio's existing renderer.
3. Capacity owner no-reclaim semantics, observable cleanup, bound device and
   measured stage requirements; contention/cancel/crash/model-switch evidence.
4. Full MoGe geometry adapter/archive, postprocess and support-plane fitting.
5. Separate SAM2 adapter, mask refinement/correction and comparison fixtures.
6. Landmark/camera/hinge fitting and corrected photographed-device overlays.
7. Device/shadow/reflection removal, repaired background and overscan inspection.
8. Hybrid physical composition, occlusion/contact/light/color and motion limits.
9. Governed image-to-object prop backend, mesh/material/UV/scale qualification.
10. Checkpointed program, API/CLI/UI progress/preview/correction and retry paths.
11. Relevant UI-unit and dependency-governance repairs; remove superseded paths.
12. Skill integration, short approved sample and revised Aquila encoded film.

Current delivery checkpoint, 2026-10-22 (photo qualification in progress):

- The revised 108s Aquila film is rendered, mastered and technically inspected.
  Local player: `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/film/physical-v1/report.html`.
  Exact MP4 SHA-256 is
  `f1181dfc5a783bf762fda1de1db029115a595ed357229977ebd8980be5a22a7a`.
  The 128-sample physical reveal uses indexed-triangle screen UV geometry,
  a right-side active-content target and arrival hold before its fade. Encoded
  prefix and final-frame reviews cover the actual join. Later layouts, file
  origin story and full native captures remain; the accepted parent is unchanged.
  The parent AAC bytes and packet timing are identical. Fresh sound-on review
  remains open; the local artifact is a review draft, not publication approval.
- MoGe native jobs retain full geometry, estimated camera intrinsics, validity,
  support-plane fit and uncertainty. A genuine empty desk source/geometry passed
  immutable review and typed scene import. The governed `import-scene` stage
  also passed through the current main Device Studio owner, with normal Library
  readback and source-use rationale. A native fixed-source-camera diagnostic
  now preserves the exact photographic plate with a separately rendered known
  device and contact shadow; camera drift is rejected in this explicit mode.
  A native 48-frame 128-sample device-only motion sample passes review, with its
  GPU claim released. Depth-mesh reprojection still
  exposes discontinuity holes and remains experimental; it is not substituted
  into the Aquila film. An older private snapshot used stale candidate
  identity code and is not the authoritative replay proof.
- SAM2 Tiny has native CPU/GPU masks and contention evidence, plus include/exclude
  correction UI. Base+ showed no useful uplift on the tested thin boundaries and
  remains disabled. Segmentation regions are not transparency mattes.
- The governed fixed-1024 LaMa build passed strict weight/export parity, native
  owned-process execution with a 16GiB ceiling, and measured ~12.7GiB peak RSS.
  Local 512/1024 LaMa and tested diffusion repairs were visually rejected on the
  retained laptop/phone plates. The 1024 candidate remains installed but disabled;
  the prior 7GiB build budget remains rejected. An explicit authored ImageGen
  phone-removal edit is retained as a separate clean-plate preparation, with
  full prompt/hash provenance and near-identity registration. Its native PNG
  encoding preserves every decoded tool-output pixel. The preparation passed
  source review and normal Device Studio import using an explicit authored
  provenance schema; it is not a local-model success or outside-mask exact-pixel
  repair. An explicit 16:9 photographic derivative uses identical native crops
  for source/plate/masks, retains the parents/transform, and has fresh native
  geometry. Its manually placed known-phone static passes native render and
  visual inspection; evidence is under
  `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/phone-authored-16x9-v1/`.
  Original-device pose fitting, automatic local repair and camera motion remain
  unqualified. No automatic photographed-device route is claimed ready.
- Local inference now reserves catalog host RAM before any provider/GPU work,
  renews leases and reaps admitted workers before releasing RAM. Native CPU
  MoGe proofs show exact claim release and no remaining JSONL worker. ONNX
  sessions use bounded threads; the phone geometry proof ran on the adopted
  source. Build cancellation/recovery has focused owner-fake tests; a native
  interrupted-export/restart proof has not been run. Physical render cancellation
  independently proves reap/release and checkpoint survival.
- TripoSR emits real GLBs and has native intake previews. The teapot is
  recognizable but imperfect; this profile has no UVs and remains a candidate.
  Known device meshes stay preferred; generated props are optional and unpromoted.
- PreparationService owns protected immutable bundles and idempotent review
  operation IDs. The bounded stage program uses normal owner APIs rather than a
  new queue. Source/model/mask/preparation assets are protected data; disposable
  outputs and lock-pinned Python runtime use leased cache roots. Native storage
  migration verified 5.3GB before removing the obsolete output tree.
- Image Tools Test Genie unit `20261022-200352-e50ef4e4` and dependency/program
  `20261022-200519-935a11af` phases pass. Two dependency governance warnings remain.
  Device Studio unit `20261022-200421-4e61d91e` passes; its dependency drift was
  corrected and dependency run `20261022-200822-43fff49e` passes. These are scoped
  validations, not full scenario certification or resolution of every advisory.
- Canonical launch-video revision 45 includes source-continuous local joins,
  active-content bounds, settled arrival, actual UV/pixel comparison and material
  review. Its sidecar was generated through the owner; Prompt Manager reads the
  current source. Native base-pack selection is unchanged.

Historical foundation checkpoint, 2026-10-22 (superseded by the evidence above):

- SAM2 Tiny has separate pinned config/weight hashes and fresh managed CPU/GPU
  durable-job proofs. Base+ comparison did not show useful uplift on the laptop
  fixture and remains disabled. Region quality is not universal matte quality.
- MoGe produces a validated float geometry/camera bundle and selected-ROI support
  plane with analytic projection tests plus the actual device-plate output.
  Estimated monocular scale is not measured scene geometry.
- Capacity now honors no-reclaim independently of priority, rejects unsuccessful
  enforced reclaim, binds a freshly selected CUDA UUID per subprocess, and
  exposes release failures. CPU-only Torch runtimes do not advertise CUDA.
- Image operations retain source-byte fingerprints and durable retry identity.
  Corrupt saved operation metadata fails explicitly rather than returning success.
- Select has an include/exclude object-mask correction UI, cancellation and
  download. Its 25 focused UI/API/locale tests pass; visual browser review is due.
  Existing shared Testing Library instances were unified; scoped UI execution
  and policy checks pass. No final scenario certification is claimed.
- SDA resolves the embedded Python compatibility forwarder to the canonical
  input/lock owner, stages both outputs, restores publication errors, and keeps
  distinct PEP 503 package identities. Focused gateway/governance tests pass and
  native adoption succeeded. Interrupted publication still requires verification.
- The isolated Device Studio variant started healthy through native lifecycle.
  A shared dependency symlink correctly caused refusal; there was no missing
  runtime source-root bug, and no speculative lifecycle fallback was retained.
  The physical sample exposed tilted-screen UV/metric defects now under repair.
- TripoSR source acquisition/profile, provider and GLB validation are in progress.
  No native prop reconstruction or completed scene/film is claimed yet.

Evidence: `~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/`
and `evidence/full-delivery/`. These checks do not close the remaining delivery
sequence above; the physical sample, photographic routes, orchestration, skill
adoption and encoded film review remain required.

Orchestration observation: `agent-manager run identity --json` returned
`identity_refresh_unavailable` on 2026-10-22. No CreateRun was attempted and no
native delivery team was enabled; this is not a CreateRun denial. Do not remove
identity headers or substitute principals. Continue authorized direct engineering
with bounded native collaboration under the existing Device Studio direct-session
commission; retain actual child identities without claiming Agent Manager
admission. Restore a verified native handoff only through its supported owner.

Recall: search-hub returned related device-domain and image model work; agent
reranking and source-ledger recall degraded. Code confirms related warm-worker
work, not a previously qualified hybrid reconstruction path.

Hypotheses and experiments:

- The ONNX worker retains CPU sessions without limit or idle exit. Inspect
  `_SESSION_CACHE`, switch model paths and verify bounded retention; run an idle
  timeout regression with a real lightweight worker process.
- GPU memory might remain in that warm worker after claim release. Inspection
  rejects this specific hypothesis: the registered warm backend is CPU-only;
  diffusion backends are one-shot. Verify cancellation reaps children before
  return, and independently audit CPU/GPU selection in one-shot providers.
- Existing SAM inference does not honor object prompts or export preprocessing.
  Exercise a non-square image and off-center positive/negative points/box;
  validate tensor coordinates, padding, mask candidate scores and source sizing.

Foundation outcome, 2026-10-22: repaired the actual installed MobileSAM tensor
contract (HWC embedded preprocessing versus NCHW), prompt coordinates/box labels,
the `has_mask_input` rank collision, and best-score source-resolution logits.
Added typed prompts and `ai segment` through the existing durable model-job
owner. Local model jobs remain serialized even on CPU. The legacy synchronous
smart-selection endpoint remains heuristic; SAM2 is not routed through SAM1.

Scoped work-ladder disposition: W3, established operator expectations for
correct model execution and idle release. Reproducing tests and owner checks are
the gate; this record does not claim a full W0–W2 or scenario maturity review.
Final review also reproduced a GPU-only catalog model selected through a
CPU-only backend (`model-backend-fit-before.log`). The resolver now skips that
candidate (or rejects an explicit override), reports actual backend GPU viability,
and the runner rechecks CPU compatibility if the backend changes after admission.
`model-backend-fit-final.log` passes the resolver, engine and handler packages.

Residency repair: two ONNX sessions maximum, clear on model change, reap after
30 seconds idle, join child/protocol teardown, and close providers at shutdown.
Failed inference runs once. Diffusers receives the admitted CPU/CUDA decision;
sd.cpp CPU requests pin both compute and parameters to CPU. Capacity grants use
the host owner, renew with generation guards, cancel on renewal loss and release
under a bounded context independent of canceled jobs. No resource model server,
private GPU scheduler, heavyweight dependency or new weight download was added.

Evidence root:
`~/.vrooli/plan-artifacts/efforts/hybrid-device-scenes-2026-10-21/evidence/`.
`cache-before.log` reproduced ten retained sessions; `owner-tests-final.log`,
`race-tests-final.log`, `cli-tests-final.log` and `sam-admission-tests-final.log`
pass. `sam-native-job.log` records a real 1672×941 laptop mask from installed
weights through HTTP and a temporary durable queue after client disconnect.
`capacity-native-{claim,heartbeat,release}.json` prove a bounded 1MiB host-owner
reservation cycle returned to released. `unit-health-api-final.json` passes the
full API workspace; protobuf verification and canonical skill validation/sync
pass. Launch Video revision 43 records source-dependent route selection and
preserves the accepted film.

Broader Test Genie run `20261022-035644-88fe586b` failed the unit phase. Its initial
API/CLI expectation failures were repaired and checked above. Existing UI
execution/policy debt remains; Scenario QA report `knw-1792642077947355550`
records it. Dependencies completed with preexisting approval-range and graph
warnings. This is not full scenario certification.

Remaining delivery: full MoGe geometry contract/post-processing and a known-camera
geometry-first physical sample; landmark/hinge pose fit; repaired photographic
background with shadow/reflection removal; physical device/shadow/occlusion/screen
composition; then SAM2 quality escalation and optional prop reconstruction with
separate governed installs and qualification. Prepared-layer parallax and one
qualified mask do not close those gaps. No new film was rendered in this repair.

### 2026-06-16 — Documentation foundation complete; implementation not started — RESOLVED 2026-06-17

**Symptom (original):** Orientation reported 5/8 gates: `scaffold-health`,
`dependency-decisions`, and `example-domain-removed` were red because only the
planning foundation existed.

**Resolution (2026-06-17):** Phase 0 + Phase 1 landed. The `notes` worked
example is **removed** end-to-end (Gate 7); the `models`/`jobs` spine domains
replace it across api/cli/proto/ui; `.vrooli/service.json` now declares the
optional ComfyUI resource + optional ffmpeg hostTool (`dependency-decisions`).
Host hardware facts come from the root `vrooli host inventory --json` CLI behind
the `capabilities` seam (not system-monitor). API/CLI/proto green + e2e boot
test green; UI tests green (162/162). See [`PROGRESS.md`](PROGRESS.md).

**Remaining for `scaffold-health` / full `make test`:** run the complete
`vrooli scenario test image-tools` suite (standards/unit/coverage/measures/
smoke) — measures phase has nothing to grade until ops declare measures
(Phase 2+).

**Refs:** `PRD.md`, `requirements/`, [`DECISIONS.md`](DECISIONS.md), [`PROGRESS.md`](PROGRESS.md).

### 2026-06-17 — react-vite template lint + standards debt (the lone `vrooli scenario test` failure)

**Symptom:** Two surfaces, both confined to **pristine template files** (verified
present at HEAD, unchanged by feature work):
1. `phase-standards` fails — OWASP "Security Headers" HIGH ×4 on
   `api/internal/httpx/errors.go` (template error writer) + three template
   test files (`internal/middleware/logging_test.go`,
   `internal/module/module_test.go`, `internal/server/server_test.go`). As of
   Phase 4 the suite is 15/18 green; the other two reds are tracked separately
   (playbooks `@scenario/self` infra regression; proto commit-gated) and are also
   not feature-introduced.
2. `pnpm lint` (`eslint .`) reports a handful of errors (plus react-refresh
   warnings): `theme.choice.{light,dark,system}` "no callsite" (dynamic-access
   false positive — they ARE used via `strings.theme.choice[c]`),
   `AppShell.tsx` aria-label string-literal, and `ThemeProvider.tsx`
   "unnecessary conditional (always falsy)" SSR guards.

**Root cause:** All are present on the **pristine react-vite scaffold at HEAD** —
not introduced by the jobs/models migration. (Standards) the OWASP
"Security Headers" rule flags any HTTP-response write lacking security headers,
including template error-rendering (`httpx/errors.go`) and HTTP-constructing
test files — a broad heuristic that over-fires on tests; this is the documented
fleet-wide standards campaign. (Lint-1) `theme.choice.*` ARE used, via dynamic
`t(strings.theme.choice[c])` in `TopBar`/`SettingsPage`; the
`strings/no-unused-keys` rule only follows static member access, so dynamic keys
read as unused (false positive). (Lint-2) The `AppShell` `aria-label="Main
content"` literal and the `ThemeProvider` `typeof window === "undefined"` SSR
guards live in files unchanged by this work.

**Workaround:** None needed for correctness — the app and API behave correctly,
and every other feature-introduced finding the suite raised (security G109, docs
manifest, go-mod-tidy, pnpm release-age, structure/proto `proto_payloads`) was
fixed. This matches the codebase's existing treatment of react-vite
template/scaffolding standards+lint debt as a tracked **upstream-template**
campaign (cf. the swarm-manager UI and the fleet standards campaign), not a
per-scenario fix — fixing it per scenario diverges from the template and breaks
on the next template sync.

**Real fix:** Upstream in `templates/scenarios/react-vite` + api-core: add a
security-headers middleware to the api-core server chain (so `httpx/errors.go`
responses inherit them) and exempt `*_test.go` from the OWASP Security-Headers
rule; teach `strings/no-unused-keys` to recognize dynamic `strings.x.y[expr]`
namespace access; route the `AppShell` main-content label through i18n; and drop
the dead SSR guards in `ThemeProvider` (the SPA is client-only). Then every
generated scenario inherits the fix.

**Owner:** unassigned (react-vite template + api-core maintainers).

**Refs:** `api/internal/httpx/errors.go`, `api/internal/{middleware,module,server}/*_test.go`,
`ui/src/consts/strings.generated.ts`, `ui/src/layout/AppShell.tsx`,
`ui/src/theme/ThemeProvider.tsx`, `ui/src/layout/TopBar.tsx`,
`ui/src/pages/SettingsPage.tsx`.

### 2026-06-16 — prd-control-tower generation gotchas (tooling, not scenario)

**Symptom:** `prd-control-tower prd generate --publish` failed with
`ORPHANED_CRITICAL_TARGETS`; `prd-control-tower requirements generate`
exceeded the client HTTP timeout for a large (28-target) registry.

**Root cause:** `--publish` enforces that every P0 target already has a
linked requirement (stricter than the START-HERE flow implies), and the
requirements generator builds all modules in one synchronous LLM call
that can outrun the CLI's HTTP client timeout.

**Workaround:** Generate the PRD draft without `--publish`, read the
markdown from `.generation.generated_text` in the `--json` output, and
write `PRD.md` directly; author the requirements modules directly from the
PRD's OT ids (`prd_ref: "OT-Pn-NNN"`). Then validate with
`prd-control-tower prd validate` and `... requirements validate` (both
report healthy).

**Real fix:** N/A for this scenario — upstream prd-control-tower
ergonomics. Captured so the next scenario author doesn't re-discover it.

**Owner:** unassigned.

**Refs:** prd-control-tower CLI; `requirements/index.json`.

### 2026-06-17 — metadata `strip-gps` removes ALL metadata in v1 (privacy-safe superset)

**Symptom:** `image-tools ops metadata --strip-gps` and `--strip-all` behave
identically: both produce an output with no EXIF/IPTC/XMP at all, not just GPS
removed.

**Root cause:** Selective GPS-IFD removal that preserves other EXIF requires
rebuilding the EXIF block (dsoprea/go-exif IFD builder). v1 implements strip by
re-encoding through Go's encoders, which write no metadata — so GPS is
definitely gone, but so is everything else.

**Workaround:** None needed for the privacy goal (location IS removed). Callers
who must preserve non-GPS tags should not strip in v1.

**Real fix:** Surgically rebuild EXIF minus the GPS IFD via the go-exif builder
and re-embed it; flip this entry to resolved. Tracked for a later enhancement.

**Owner:** unassigned.

**Refs:** `api/internal/ops/metadata.go` (`stripMetadata`); DECISIONS.md.

### 2026-06-17 — Runtime measures: recorder built; per-model enrichment + query surface deferred — PARTIALLY RESOLVED

**Resolution (Phase 4):** `internal/measures` now records op latency (p50/p95),
queue-wait, throughput, and terminal-state mix into SQLite `op_measure`, fed by a
new generic `jobs.Manager.OnComplete` hook — so EVERY finalized job (deterministic
+ AI) accrues op-level metrics. `Recorder.Stats` computes the aggregates on read
(`measures_test.go` covers it). The schema + `Sample` API carry `model_id`/`tier`/
`fallback_used` columns.

**Still deferred:** (1) per-model latency + fallback-tier ENRICHMENT — the AI
runner knows the resolved model + provider tier but does not yet call
`Recorder.Record` with a model-backed sample (kept the AI engine decoupled from
measures); op-level facts are captured, model attribution is not. (2) No
read/query SURFACE (proto/handler/CLI) exposes the aggregates yet — they are
recorded but not served. The test-genie `measures` phase is inference-based
(stateful-domain coverage) and is GREEN without a manifest measures block.

**Real fix:** Have the AI runner record a model-backed `Sample` on completion
(model id + tier + fallback flag), and add a `MeasuresService.Stats` RPC + CLI
`measures` verb when an operator/consumer needs the aggregates.

**Owner:** unassigned.

**Refs:** `api/internal/measures/`, `api/internal/jobs/manager.go` (`OnComplete`),
`api/main.go` (recorder wiring), `api/internal/ai/engine.go` (enrichment seam).

### 2026-06-17 — proto red: `shared.ErrorEnvelope` REST-error payload is commit-gated

**Symptom:** `phase-proto` fails with 4 ERROR `proto.rest_payload_unknown_message`
findings — endpoints `ai_submit`/`analysis_run`/`ops_run`/`ops_blob_get` declare
their REST-exception error body as `vrooli.image_tools.v1.shared.ErrorEnvelope`,
which proto-health reports as an unknown message.

**Root cause:** Commit-gated, not a working-tree defect. The working tree is
internally consistent — `packages/proto/schemas/image-tools/v1/shared/errors.proto`
declares `package vrooli.image_tools.v1.shared` with `ErrorEnvelope`, the
generated Go/TS/Python carry `shared`, and `.vrooli/endpoints.json` references
`shared.ErrorEnvelope`. proto-health validates against the COMMITTED descriptor
(HEAD), where the `errors`→`shared` move is still uncommitted (HEAD has it in
package `errors`), so the working-tree endpoints don't match the committed
descriptor. Same commit-gated pattern as Phases 2-3.

**Workaround:** None — this branch commits via an external `image-tools pN`
process; the working tree is correct and verified.

**Real fix:** The external committer commits the working tree (including the
regenerated descriptor with `shared`); proto goes green immediately. No code
change required.

**Owner:** the branch's phase-committer.

**Refs:** `packages/proto/schemas/image-tools/v1/shared/errors.proto`,
`scenarios/image-tools/.vrooli/endpoints.json`, proto-health
`rest_payload_unknown_message`.

### 2026-06-17 — AI backend live-execution unverified on CI hosts (attended acceptance gate) — PARTIALLY RESOLVED 2026-06-18

**Update (2026-06-18, plan Phase 1):** `background_removal` now runs end-to-end
on CPU for real (onnxruntime sidecar + validated `u2netp.onnx`; see the
2026-06-18 install-stub entry, RESOLVED). The headless-completeness acceptance is
met for that op. The remaining AI ops (`generate`/`img2img`/`inpaint`/
`object_removal`/`upscale`/`denoise`-onnx, `analyze ocr|nsfw`) still need their
backend programs/weights provisioned and an attended run; they are built as later
phases of the advanced-editing plan.

**Symptom:** `image-tools ai generate|img2img|inpaint|object-removal|upscale|bg-removal|denoise`
and `analyze ocr|nsfw` return HTTP 409/503 with an actionable install hint on a
host without the backend binaries/models. Only `analyze probe` runs end-to-end
out of the box.

**Root cause:** The standalone backends (`sd` / stable-diffusion.cpp, `iopaint`,
`realesrgan-ncnn-vulkan`, `rembg`, the onnxruntime + diffusers python sidecars,
`tesseract`) and the CPU-default model weights are not installed in CI or on this
dev host, and model download-on-first-use is owned by the Phase-4 model
management layer (IMG-P0-007). The selection → plan → refuse path is correct and
live-proven (HTTP 409 `run image-tools models install sd-1.5`).

**Workaround:** The full vertical (select → execute → persist → auto-scan) is
covered by unit/integration tests with fake providers; backend arg-builders are
unit-tested for assembly. `probe` is the live headless proof.

**Real fix:** The IMG-P0-002/003/004 **headless-completeness acceptance** is an
**attended** run on a host where the CPU default models are installed (Phase 4
`models install`). Capture it as a checklist item; flip this entry to resolved
once an attended run exercises each AI op from the CLI with no GPU/ComfyUI.

**Update (2026-06-18):** The attended gate was performed. It revealed a second,
more fundamental blocker beyond absent backends — the `models install` path is a
stub that fetches landing pages, not weights (see the 2026-06-18 entry below).
Deterministic ops + `analyze probe` run end-to-end for real; no AI op can.

**Update (2026-06-24, diffusers-edit substrate):** `edit_instruct` now runs
end-to-end on a GPU host. Qwen-Image-Edit-2509 was installed via the new
revision-pinned repo-snapshot path (`Source.Repo` + `HFSnapshotFetcher`, 57.7 GB,
rev `d3968ef9`), loaded through the generic `_diffusers` runner (family
`qwen-image-edit-plus`, class auto-resolved + asserted), and run with adaptive
sequential CPU-offload on a contended 16 GB RTX 4070 Ti SUPER — producing a
correct instruction edit (red→green circle + yellow border; identity preserved;
8 steps, 161 s). torch/torchvision/transformers/accelerate/huggingface_hub were
provisioned via SDA-governed records. `qwen-image-edit` is now `enabled`. This is
the first heavy diffusers AI op proven live end-to-end (background_removal proved
the onnxruntime path 2026-06-18). InstructPix2Pix (CPU edit default) and the
inpaint/onnx ops still await their own attended runs.

**Owner:** unassigned (Phase 4 wires model download; then run the attended gate).

**Refs:** `api/internal/ai/`, `api/internal/analysis/`, `docs/internal/TESTING.md`
(headless-completeness acceptance), `api/internal/models/registry.seed.json`.

### 2026-06-17 — govulncheck advisories in golang.org/x/image — RESOLVED for image-tools

**Resolution (2026-06-17):** Bumped `golang.org/x/image` v0.25.0 → **v0.42.0**
(api/go.mod; pulled x/text → v0.38.0) with user approval; `phase-security` now
passes (0 ERROR findings) and `phase-dependencies` stays green. Governance note
recorded in `.vrooli/dependencies/approved-dependencies.json` (security floor
>=0.42.0). Codec golden/round-trip tests pass on v0.42.0 (API-compatible).
**Fleet caveat:** `chart-generator` still pins x/image v0.18.0 (indirect) and is
also affected — its bump is tracked in the governance `security_notes` but not
done here (out of Phase-3 scope). Kept `version_range: "*"` to avoid red-flagging
chart-generator before its own migration.

**Symptom (original):** `phase-security` reported 4 ERROR-severity govulncheck
findings — GO-2026-4815 (tiff IFD-offset OOM), GO-2026-4962 (sfnt excessive
allocation), GO-2026-5031 (bmp out-of-bounds palette panic), GO-2026-5032 (tiff
PackBits resource consumption) — all in `golang.org/x/image`.

**Root cause:** PRE-EXISTING. `golang.org/x/image v0.25.0` is the version at HEAD
(added by Phase 2's codec layer), and these decoders are reachable from
`internal/ops/codec.go::Decode` (committed in Phase 2). The advisories are
2026-dated and were published into the vuln DB after Phase 2's green security
run, so they surface now regardless of Phase 3. Phase 3's `analysis.Probe` reuses
the same `Decode`, but does not introduce the reachability.

**Workaround:** None at the code level — the ingest guard
(`internal/storage/guard.go`) already bounds decode dimensions/bytes, mitigating
the OOM/resource-consumption class at the boundary.

**Real fix:** Bump `golang.org/x/image` to a patched release (cached versions up
to v0.42.0 are available) across `scenarios/image-tools/api` (+ `cli` if it
imports it) and update `.vrooli/dependencies/approved-dependencies.json`. This is
a dependency change (Critical Rule #5 — requires explicit permission) and is
fleet-wide (any scenario decoding images via x/image is affected), so it should
land as a coordinated bump, not a silent per-scenario edit.

**Owner:** unassigned (needs maintainer approval for the dep bump).

**Refs:** `scenarios/image-tools/api/go.mod` (x/image v0.25.0),
`internal/ops/codec.go`, `internal/storage/guard.go`.

### 2026-06-17 — playbooks red: `@scenario/self` port resolution regression (fleet infra)

**Symptom:** `phase-playbooks` fails before any step runs (0 steps, 0 asserts):
`failed to resolve port for scenario @scenario/self: ... instance key: missing
scenario name in "@scenario/self"`.

**Root cause:** PRE-EXISTING / fleet-wide. The BAS case files use the canonical
self-reference `"scenario": "@scenario/self"` (so does every other scenario —
code-facts, search-hub, device-sync-hub, measures-health, …). The root CLI's
instance-key parser now rejects it: `vrooli scenario port image-tools API_PORT`
returns the port, but `vrooli scenario port "@scenario/self" API_PORT` errors
`missing scenario name in "@scenario/self"`. A concurrent change to the
`vrooli scenario port` / api-core instance-key parsing broke the `@scenario/self`
sentinel after Phase 2's green playbooks run. Not introduced by Phase 3 (the
editor page + ops discovery are never reached).

**Workaround:** None at the scenario level — the sentinel is resolved by the BAS
workflow runner, not editable per-scenario without diverging the case template.

**Real fix:** Restore `@scenario/self` resolution in the root `vrooli scenario
port` instance-key parser (treat `@scenario/self` as "the current scenario"
rather than parsing it as `name@variant`). Filed to scenario-qa.

**Owner:** unassigned (root vrooli CLI / api-core discovery maintainers).

**Refs:** `bas/cases/deterministic-ops/ui/editor-lists-operations.json`,
`bas/cases/routed-database/proves-test-pool-routing.json`, root `vrooli scenario
port`.

### 2026-06-18 — `models install` fetches landing pages, not weights — install path is a stub for all 49 models — RESOLVED (substrate fix) 2026-06-18

**Resolution (2026-06-18, advanced-editing plan Phase 1):** The install-stub bug
is fixed at the root and the first AI op now runs end-to-end on CPU:

- **Artifact validation** (`api/internal/models/artifact.go`): every downloaded
  asset is validated before an install is recorded — HTML pages are rejected via
  `http.DetectContentType` (the exact stub symptom), size floors catch truncation,
  and per-`Kind` magic (ONNX ir_version tag, GGUF, safetensors) rejects wrong
  formats. A page URL can no longer be recorded as a model. Regression test:
  `TestInstall_RejectsHTMLPageDownload`.
- **Resolvable assets** (`Source.Assets`): seed models now declare direct,
  validated weight URLs. The two background-removal CPU defaults (`u2netp`,
  `isnet-general-use`) carry real rembg-release `.onnx` asset refs; the page URL
  moved to `docs_url`. Un-migrated entries fail LOUD (page → HTML → rejected)
  instead of false success.
- **Provisionable CPU backend**: a real in-repo Python sidecar
  (`api/internal/sidecar/py/image_tools_sidecar`, embedded + materialized on
  PYTHONPATH at boot) runs the ONNX background-removal via onnxruntime —
  CPU-only, no GPU, no `rembg` program needed. The two bg models now use
  `backend: onnxruntime` (`rembg` is an alt).
- **Honest tier**: the onnxruntime sidecar declares `GPUCapable()=false`, so the
  selector labels its runs `local-cpu` even on a GPU host.

**Live proof (2026-06-18):** `models install u2netp` downloaded the real 4.5 MB
`u2netp.onnx` (validated), then `ai bg-removal … --model u2netp --wait` produced
a correct RGBA matte (subject opaque, background transparent) — reported
`succeeded on u2netp/local-cpu`. The host already had onnxruntime/Pillow/numpy
(the documented provisioning step).

**Remaining (separate, tracked):** the other ~46 seed models still need real
asset refs (migrate as each op's vertical is built); the non-ONNX backends
(`sd`, `realesrgan`, `iopaint`, `tesseract`) remain absent/unmanaged for their
ops; BYOK cloud is designed (ladder present) but its real key-gated provider is
not yet implemented. See the advanced-editing plan Phases 1→4.

---

**Symptom (original):** `image-tools models install <id> --wait` reports ✅ success in ~1s
for any model, pins a real sha256, and marks it `installed`. The downloaded
"weights" are not a model. Reproduced live: `models install u2netp` produced a
**441 KB GitHub HTML page** (`<!DOCTYPE html>…`) saved as
`~/.vrooli/data/vrooli/image-tools/models/u2netp/rembg`. The selection gate then
treats the model as installed, yet no op can run.

**Root cause:** The installer downloads whatever `source.download_url` returns
and pins *its* checksum, with **no validation that the bytes are a real model
artifact** (no content-type / magic-number / expected-extension / size-band
check, no HF-resolve or release-asset resolution). The seed's `download_url`s
are HuggingFace/GitHub **landing & release pages**, not direct weight assets —
**0 of 49 seeded models** point to a `.onnx/.safetensors/.pth/.gguf/.zip` file.
So "checksum pinned on first download" gives genuine integrity of the **wrong
bytes** — false verification. This is independent of, and compounds, the absent
backend programs (2026-06-17 entry above): an AI op is blocked by *both*.

**Workaround:** None — AI ops cannot run end-to-end. Deterministic editing +
`analyze probe` are the only genuinely-working paths today (both verified live
2026-06-18).

**Real fix:** (1) `models install` must resolve and fetch the actual weight
asset(s) per model + backend file layout (HF resolve URLs / release assets),
**validate** the downloaded artifact (expected type/magic/size, not just
self-checksum), and pin an upstream-verified checksum. (2) Add a backend-program
provisioning story — install/detect `rembg`/`sd`/`realesrgan`/`iopaint`/
`tesseract`, or pivot CPU defaults toward provisionable pure-Go/ONNX-runtime
paths — backends are currently unmanaged host prerequisites with no install
path. (3) Make the registry/selector honest about *runnable* vs *declared*. This
is the foundational phase of the image-tools advanced-editing plan.

**Owner:** unassigned (image-tools model-management).

**Refs:** `api/internal/models/` (installer / `Downloader`),
`api/internal/models/registry.seed.json` (`source.download_url` ×49), the
2026-06-17 attended-acceptance entry above.

### 2026-06-18 — requirements registry falsely marks unbuilt P1 modules "complete" (traceability drift)

**Symptom:** 23 of 28 requirement modules report `status: complete`, but modules
**14 (gen-breadth), 15 (enh-breadth), 16 (analysis-breadth), 17 (recipes),
18 (batch/watch), 19 (webhooks), 20 (safety/provenance), 22 (image-diff),
23 (presets)** are NOT implemented — their ops are absent from the AI/analysis
catalogs (`ai list` = 7 ops, `analyze list` = 3) and their packages don't exist.
Verified phantom validation refs: `api/internal/ai/sam_test.go`,
`api/internal/ai/restoration_test.go`, `api/internal/provenance/`,
`api/internal/byok/` — all MISSING, yet cited with `status: implemented`.
(Module 21/editor-layer IS genuinely built via the Lume UX; 24–28 are honestly
`planned`.)

**Root cause:** `requirements validate` checks schema + PRD linkage, not whether
the cited validation-ref files exist or the ops are wired — so optimistic
"complete" statuses and phantom refs pass validation. The modules were marked
complete at generation / early-phase time and never corrected as scope reality
diverged.

**Workaround:** Trust PROGRESS.md + the live op catalogs over the requirement
`status` fields until corrected.

**Real fix:** A focused, per-module pass (NOT a mass-update script — edit each
file individually per the repo rule): flip the 9 modules' requirement status to
`planned`, set their validation entries to `planned`, and repoint or remove the
phantom validation refs. Scheduled as the first task of the advanced-editing
plan's Phase 1 prep. Consider teaching `requirements validate` to verify ref
existence for `implemented` entries.

**Update (2026-06-18, plan Phase 3.3):** The drift cuts BOTH ways. A
prd-control-tower auto-sync (it syncs requirement statuses from the latest
`test-genie` run) flipped the *genuinely shipped* modules 31 (looks), 32
(smart-select), 33 (instruction-edit) to `status: in_progress` and their
test-ref statuses to `failing` — because the run it synced from
(`20260618-230047`) had a **flaky** unit-phase failure under concurrent load
(that same unit phase passes deterministically in isolation: api/cli
multi-package `-covermode=atomic` runs + a unit-only `test-genie execute …
unit` → PASS, and the new code is `-race`-clean). Corrected all three back to
`complete`/`implemented`; re-validated; the correction **held** (auto-sync is a
one-time-on-run sync, not a continuous re-writer). Takeaway: the auto-sync
trusts a single run's verdict, so a flaky red silently rewrites honest module
statuses to failing — confirm against an isolated re-run before believing it.

**Owner:** unassigned (image-tools requirements).

**Refs:** `requirements/{14..20,22,23}/module.json`, `requirements/index.json`,
`api/internal/ai/catalog.go`, `docs/internal/PROGRESS.md`.

### 2026-06-18 — Phase 4 suite residuals: proto commit-gating (now incl. diff/safety), unit flake, DCT-hash duplication

**Symptom:** Full `vrooli scenario test image-tools` after Phase 4 = 13 pass / 6 red.
Three things to know for the next agent (none a Phase-4 code defect — confirmed
by `test-genie runs compare` vs `20260618-230047` + isolated re-runs):

1. **`proto` regression is commit-gated** (extends the 2026-06-17 ErrorEnvelope
   entry). The NEW `diff` + `safety` proto domains + the `analysis` oneof
   (`DuplicateResult`/`QualityResult`) + `AIParams.consent_affirmed` are correct
   in the working tree — `packages/proto/gen/descriptor/image.binpb` carries them
   (verified) — but proto-health validates **committed HEAD**, so
   `diff_compare → DiffResult unknown` / `handler domain "diff"/"safety" has no
   matching proto domain` fire until the branch's committer commits the proto +
   regenerated code. Agents never commit; goes green on commit.
2. **`unit` red is a concurrent-load flake** (the recurring pattern): under suite
   parallelism `pnpm test:coverage` (ui) intermittently fails; **in isolation**
   go `-covermode=atomic ./...` PASSES and ui `pnpm test:coverage` = 740 @98.16%.
   Confirm unit reds with an isolated re-run before trusting them; the
   prd-control-tower auto-sync will flip requirement modules to `failing` off such
   a run (corrected modules 14/22/34 back to `implemented` 2026-06-18).
3. **Minor self-introduced debt (deliberate, scoped):** the DCT perceptual-hash +
   2D-DCT + median helpers are duplicated between `internal/diff/metrics.go` and
   `internal/analysis/duplicate.go` (~70 lines). They can't share directly (the
   sibling-domain layering rule), so the clean fix is to extract a leaf
   `internal/imagehash` package both import. Not done now: the `tidiness`
   duplication phase is a fleet-wide campaign red already (plan §4/§12 out-of-scope),
   so deduping would not flip it; tracked here as the right follow-up. Owner:
   unassigned.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues


### 2026-10-22 — Storage placement and cache migration

The native storage phase exposed undeclared derived outputs and cache declarations
that did not match the data-root writer. The writer now routes `out/` to the
request-selected cache class; uploads, masks, comparison assets and retained
preparation bundles/previews remain protected data. Both classes honor leased
test roots. Startup fails explicitly on an unmigrated legacy output tree.

On a stopped native owner, Storage Manager placement copied and verified
5,322,714,869 bytes of outputs before removing the old tree. It also migrated
the materialized sidecar and requirements lock with verified receipts. The old
virtualenv was atomically retained as cache `legacy-pyenv`, never executed after
relocation; native lock-pinned provisioning rebuilt cache `pyenv` and reported
ready. Original result refs and protected preparation readback pass. The old
runtime remains temporary rollback retention until replacement qualification,
then can be explicitly retired. Retention enforcement receipts are declared in
their actual state class. Focused routing, leased-root isolation, legacy-tree
refusal, preparation persistence and AI lifetime regressions pass. Scoped storage
revalidation remains in progress; filesystem audit advisories are not full
maturity certification.

The warning claiming raw SQL in `handlers/models/module.go:159` is a validator
false positive on the endpoint example text "Select for upscale"; there is no
SQL call there. Filed through report-bug as `knw-1792684261250157000` on
`bug-inbox/code-defect/storage-sql-analyzer-treats-model-help-text-as`.


### 2026-10-22 — Preparation service and native backend environment checkpoint

Normal PreparationService create/get/list/review calls, manifest CLI bindings,
protected content-addressed ZIP/preview retention, and the Library comparison
and immutable review controls are implemented. Focused preparation persistence,
source substitution, review immutability, API-domain parity, CLI and UI tests
pass. Native UI/complete photographic conversion qualification remains pending.
Preparation metadata and preview/bundle bytes are declared as protected data;
they survive disposal of source job outputs. The governed stage program now
includes assembly/inspection/review boundaries; its new fixtures still need the
program owner run after temporary CLI-build pressure clears.

Image Tools now adopts the same typed GPU capacity client as Device Studio.
The shared owner validates fresh UUID/index identity before and after admission,
retains the exact generation, joins renewal before release, and distinguishes
valid capacity denial from transport/protocol/cleanup failures. The private
scenario CLI JSON implementation is removed. Focused shared-client and engine
checks pass; earlier native GPU/CPU contention evidence remains retained.

The resource CPU downgrade exposed a control-plane defect: native managed
services ignored the selected backend's declared environment. The launch owner
now applies that environment after generic configuration. Ollama's CPU profile
explicitly disables CUDA/ROCm/Vulkan visibility using upstream documented
settings. Native PID 722047 was serving on CPU with those exact process values
and no Ollama GPU ledger claim; evidence is in full-delivery/
`ollama-cpu-process-environment.json`. Focused regressions pass. Broader resource
package tests also exposed independent blueprint fixture, PII-detector fleet
contract and Kokoro legacy footprint failures; those are not claimed passing.

SDXL native GPU jobs mechanically passed but failed visual review: original
conditioning regenerated the laptop; explicit neutralized conditioning removed
it but invented an oval object and left seams. Both remain rejected and the
runtime candidate was disabled. Preserve these outputs when qualifying a more
appropriate photographed-device fixture; do not treat a pinned checkpoint or
successful inference as photographic readiness.
