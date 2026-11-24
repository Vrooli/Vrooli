# Security — Image Tools

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.


Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

Image-tools processes opaque user binaries plus credentials, so it
handles more sensitive data than a typical scenario.

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| User-uploaded images | high | user | May contain PII (faces, documents, screenshots) and EXIF GPS coordinates. GPS/location metadata strip is on by default; full metadata read is an explicit op. |
| EXIF / IPTC / XMP metadata | medium-high | user | GPS, device serials, timestamps, and authorship are extractable. Stripped on a GPS-off-by-default basis; never silently re-embedded. |
| Generated / edited output | medium | user | May be NSFW (gated by NSFW classifier + capability labels). User-owned; stored via blobstore or the per-request override location. |
| BYOK provider API keys | high | user | Cloud-tier credentials. Sourced from api-core secrets; never persisted alongside images or outputs, never logged. |
| C2PA signing key | high | scenario | Used to sign content credentials when the opt-in feature is enabled. Isolated in api-core secrets. |
| Job / recipe / model-registry metadata | low | scenario | Proto-typed SQLite rows (no binary content). |

## Auth And Authorization

Image-tools inherits scenario / cli-core auth — it does not implement a
bespoke auth provider. Authorization decisions belong at the API/service
layer; the UI and CLI must not enforce business authorization locally.

- The **headless API** is the canonical surface; every op is reachable
  programmatically and must be subject to the same authorization as the UI.
- **Per-job webhook callbacks** are authenticated by signing: each job's
  callback URL receives a signed (per-job secret) request so the receiver
  can verify the payload originated from image-tools and was not forged.
- BYOK cloud calls carry the user's own credentials; image-tools never
  acts as a shared credential proxy.

## Secrets

| Secret | Source | Required? | Notes |
|---|---|---|---|
| BYOK provider API keys (cloud fallback tier) | api-core secrets | no | Only needed for the BYOK cloud tier. Never persisted with images/outputs, never written to job logs or webhook payloads. |
| C2PA signing key | api-core secrets | no | Only present when C2PA content credentials are enabled (opt-in, off by default). |
| Webhook per-job signing secret | generated per job (api-core secrets / job store) | no | Only when a callback URL is supplied. Scoped to the single job. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Decompression bomb / pixel-flood image | Memory exhaustion, OOM, DoS at the ingestion boundary. | Enforce max decoded dimensions and pixel-count caps + max upload size before full decode; reject oversized inputs at the multipart/REST edge. | **covered** by `storage.DefaultGuard` (64 MiB encoded / 128 MP decoded / 30000 px per side, checked from the header) |
| Malicious image file (crafted EXIF, embedded payloads, SVG with scripts) | Code execution / parser exploitation; SVG is the highest-risk surface. | Treat SVG as raster import only (rasterize in a constrained renderer, never execute embedded scripts); validate/normalize metadata; use hardened decoders. | to-build |
| Path traversal on overridable save location | Write outside intended storage, overwrite arbitrary files. | Canonicalize and confine the per-request save path to allowed roots; reject `..`/absolute escapes; default to blobstore. | to-build |
| SSRF via input-URL fetch and webhook callback URL | Internal-network probing / metadata-endpoint theft. | Allowlist/denylist outbound fetch + callback targets; block private/link-local ranges and cloud metadata IPs; cap redirects. | to-build |
| Resource exhaustion via unbounded jobs (GPU/disk) | Host-memory or VRAM contention, disk fill from model weights + outputs. | Serialized local queue for local AI work; bounded CPU and network pools; per-user/job concurrency limits; disk-space awareness before model installs and output writes. Queue class does not assert execution device. | to-build |
| Model supply-chain (tampered/poisoned weights) | Malicious or corrupted model code/weights executed locally. | Checksum-verified, opt-in downloads from declared sources in the model registry; reject on checksum mismatch; pin known-good sources. | **covered**: upstream sha256 when published, else pinned after validation |
| Prompt injection in text-guided ops (inpaint/text-to-image prompts) | Coerced model behavior, safety-bypass attempts. | Treat prompts as untrusted; apply NSFW/safety auto-scan to output; capability labels surface model risk; do not interpolate prompts into shell/file paths. | to-build |
| NSFW / safety-classifier evasion | Disallowed content slips past auto-scan. | Configurable auto-scan of generated output + standalone classifier op; capability labels (NSFW-capable) on models; classification is advisory by default, hardenable per deployment. | **partial**: advisory scan + hard minor-safety refusal enforced by the bundled reviewer on every tier (see [Reviewer calibration](#reviewer-calibration)); the sensitive flag blurs flagged output in the UI |
| Secret leakage (BYOK keys / signing keys) | Credential theft from logs, outputs, or stored metadata. | Keys live only in api-core secrets; never persisted with images/outputs; scrub from logs and webhook payloads. | to-build |

## Security Gaps

This scenario is implemented; the table below enumerates mitigations that are
still open, plus the ones already covered so the status is not ambiguous.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| ~~No bundled minor-age classifier~~ — **closed**: a bundled reviewer (NSFW classifier, YuNet faces, FairFace age) runs on every tier and fails closed; numbers in [Reviewer calibration](#reviewer-calibration). | — | Reopen if the reviewer's models are replaced or the thresholds change (re-run `scripts/minor-review-eval.py`). |
| Residual reviewer error. The age model is an appearance estimate, not proof of age: it misses roughly 1-2% of faces of children under 10 and about a quarter of the 10-19 band at the shipped threshold; a sexual image with no readable face is refused outright, so the reviewer also refuses some adults. | **medium** | Before any deployment that serves generated images to other people at volume; consider a second age signal or human review of refused/flagged output. |
| SSRF allowlist for input-URL fetch and webhook targets not yet built | high | Before input-URL fetch or webhook callbacks ship. |
| Path confinement for overridable save location not yet enforced | high | Before per-request save-location override is exposed. |
| Signed-callback verification contract not yet implemented | medium | Before webhook callbacks ship (P1). |
| Secret isolation (no key persistence with images/logs) not yet audited | high | Before BYOK cloud tier or C2PA signing ships. |
| C2PA cryptographic signing not yet implemented (marker + policy only) | medium | Before provenance is relied on as evidence rather than a label. |
| ~~Upload validation~~ — **covered**: encoded ≤64 MiB, decoded ≤128 MP, each side ≤30000 px, checked from the header before decode | — | Reopen if a new ingest edge bypasses `storage.DefaultGuard`. |
| ~~Model-download checksum verification~~ — **covered**: upstream sha256 when published, else pinned after validation | — | Reopen if an install path records before validating. |

## Responsible Use & Deployment Gate

image-tools can edit images of real, identifiable people (inpaint,
instruction-edit, naturalize, smart-select, and identity/appearance edits such as
face transfer or clothing changes). **The capability is not gated.** This is a
local-first tool: someone running it on their own machine, with their own and
consented images, is not asked to justify that. Harm protection is a set of
**configurable controls** whose purpose is to make a *hosted* deployment
defensible — that is where legal exposure actually lives — without imposing that
posture on personal installs.

| Control | Local / personal (default) | Public / monetized (default) | Override |
|---|---|---|---|
| Content scan (NSFW/unsafe) | off | **on** | `IMAGE_TOOLS_NSFW_SCAN` |
| C2PA / attribution provenance marking | off | **on** (≥ real-person edits) | `IMAGE_TOOLS_REQUIRE_PROVENANCE` |
| Consent affirmation + audit logging on identity-altering ops | off | **on** | `IMAGE_TOOLS_REQUIRE_CONSENT` |
| Consent affirmation when a high-weight request, or a low-weight one that sends an image showing people (caption, detection, tags, colourise), would leave this machine (cloud provider, BYOK, remote node) | **on** | **on** | `IMAGE_TOOLS_REQUIRE_CONSENT_NONLOCAL` |
| Abuse rate-limiting | off | **on** (60/min) | `IMAGE_TOOLS_RATE_LIMIT_PER_MIN` |
| Acceptable-Use Policy published | n/a | **required** | — |
| Sexual content involving minors | **bundled reviewer on** (every produced image) | **required**; bundled reviewer on, an operator reviewer may replace it | **not configurable** off; `IMAGE_TOOLS_MINOR_SAFETY_CMD` only swaps the detector |

Every override wins in both directions: a personal install may opt *into*
provenance marking, and an operator running their own public instance may turn a
control *off*. The operator owns that decision; the control set is data, and
`SafetyService.GetPolicy` / `image-tools safety policy` reports the resolved
result so the UI never claims a protection that is off.

Consent weight is **not uniform** and never removes a capability. Naturalize
(realism/texture only — same person) is low-weight; aging/makeup are low–medium;
body, clothing, and pose changes are **high-weight** and carry the most exposure.
On a deployment with the consent control on, high-weight ops require an
affirmation that is recorded; on a permissive install the weight changes nothing
**for work that stays on the machine**. Local open-source use needs freedom;
sending a person's image to another party is a different act, so the route
decides (see [Route-locality consent](#route-locality-consent)).

### Route-locality consent

A high-weight operation (face swap, identity, body, clothing and appearance edits,
instruction edits, inpaint, NSFW editing) whose **resolved route leaves this
machine** requires `consent_affirmed=true` on every tier, including the local one:

- **Where it runs is data.** `safety.tierLocality` maps a route tier to local or
  remote: `local-gpu`, `local-cpu` and `gateway-local` (a model the AI Gateway
  serves on this machine) are local, `byok-cloud` is remote. A tier the
  table does not know is treated as remote, so a new route kind (a remote node, a
  new provider class) must declare itself local to skip the affirmation.
- **One rule, four consumers.** `Gate.ConsentFor(weight, op, tier)` answers
  "does this request need consent, and why" for the submit edge (HTTP 403 with a
  recovery hint), the engine's automatic routing (an un-affirmed request is never
  moved to the cloud: it stays local with reason `consent_required` and the wait
  budget re-route is not armed), execution admission (re-checked before a cloud job
  runs, because a route can be pinned, re-routed or recovered after submit) and the
  explain surface (`Resolution.consent_required`, `consent_basis` = `tier` or
  `route`, `consent_reason`), which the UI reads to show its checkbox only when it
  applies. The CLI says the same in `--explain` output.
- **Audit.** An affirmation that unlocked something is appended to `consent_log`
  with the operation, weight, deployment tier and the route it was sent to
  (`<tier>:<model>`, e.g. `byok-cloud:openrouter-image`). Affirming when nothing
  asked is not logged. The `route` column is added to existing installs by
  `ConsentLog.EnsureSchema`.
- **Local runs are untouched.** A local face swap or NSFW edit needs no checkbox
  and writes no audit row on a local-tier install. `IMAGE_TOOLS_REQUIRE_CONSENT_NONLOCAL=0`
  turns only the route rule off (for a deployment that has decided its cloud
  routes are acceptable); a malformed value is ignored.

#### Low-weight operations: asked only when the image shows people

The rule protects the people in an image, so a **low-weight** operation (caption,
object detection, tagging, colourise, naturalize) whose route leaves this machine
asks for consent only when its input shows people. A picture of a guinea pig sent
to a hosted caption model needs no checkbox; a portrait does.

- **The people check.** At route planning the submit edge runs a face count on the
  stored input (the sidecar's `face_count` module: YuNet on the CPU, about half a
  second, no VRAM). It runs at most once per request, and only when a route that
  leaves the machine is being weighed: a local route never pays for it. The result
  is memoised and shared by the engine's routing and the gate's decision.
- **Fail closed.** Faces found: consent required, as for any remote route. No faces:
  not required. The detector missing, its weights not installed, a decode failure,
  a timeout or any error: consent required, because "nobody is shown" is something
  the gate must establish. A request that never looked at an image (the plain
  `Policy.EvaluateRoute`, an explain call) keeps the requirement, so the explain
  surface still tells the UI to show its checkbox; the face count decides at submit.
- **Recorded.** The job's meta carries `consent_basis`: `faces_detected=N` (the check
  ran; N may be 0), `detector_unavailable`, `not_required_local` (the route stays
  here), `not_required_control_off` (`IMAGE_TOOLS_REQUIRE_CONSENT_NONLOCAL=0`), or
  `route` / `tier` for a high-weight request that asked. A request found to show
  nobody carries `consent_waived` in its payload, so a wait-budget re-route after it
  was queued does not need a second look; one that was never checked (it stayed
  local until then) is held local rather than moved to the cloud without consent.
- **Faces never loosen anything else.** Medium- and high-weight operations (face
  swap, identity edits, NSFW editing) ask on a remote route whatever the count says.
  The count is not a person classifier: a drawing, a statue or a face too small for
  the detector counts as nobody, which is why the weight is `low` and the rule is
  about who is shown to a provider, not about what the output shows.

### Non-negotiable

Sexual content involving minors is illegal content, not a contested-use question
and not a "morality" call, so it is the one item that is **not** part of the
configurable control set. It is enforced in code, not asserted in prose:

- `Policy.MinorSafetyRequired` is on for **every tier** (local and public) and is not
  read from an override. `ApplyOverrides` cannot turn it off.
- The AI submit edge reviews the source image before a model is admitted, and the
  AI engine reviews **every produced image** regardless of whether the request
  asked for a scan. A positive minor + sexual verdict fails the job and deletes the
  output, so the refused bytes never become addressable.
- The reviewer is **bundled** (`minor_review`, a CPU-only Python sidecar module) and
  is used unless `IMAGE_TOOLS_MINOR_SAFETY_CMD` names a replacement. Its four
  weights are acquired through managed acquisition and scoped by need: the two
  sexual classifiers (`falconsai-nsfw`, the precise one, and `adamcodd-vit-nsfw`,
  the high-recall one) are read for every output; the face finder and age
  model (`yunet`, `fairface-vit-age`) only for an output a classifier calls sexual.
  They are acquired during admission when the producing model is `nsfw_capable`, and
  on demand otherwise (the reviewer exits 4 naming them, the engine acquires exactly
  those and reviews again once). So ordinary SFW work never depends on the age model,
  and when an age review is needed but cannot be run, only that output is refused. If the reviewer cannot be built or run at all, a reviewer that
  always errors is installed, so the engine **fails closed** (the output is
  deleted and the job fails) instead of skipping the review. A reviewer that errors
  or times out (90 s) also fails closed. A public deployment with no reviewer would
  refuse AI image output before it ran; with the bundled reviewer that case only
  arises when the operator explicitly replaces it with nothing.
- Output from a model labelled `nsfw_capable` is reviewed in **strict** mode (the
  classifier is asked a lower bar, see below), because the classifier is weaker on
  synthetic explicit images and these are the models that can make them.
- Deterministic transforms (resize, crop, convert) remain available, because they do
  not generate content.

#### Reviewer contract

The configured command is parsed without a shell (whitespace splits argv). It
receives one JSON object on stdin:

```json
{"images":[{"media_type":"image/png","base64":"..."}],"strict":false,"assume_sexual":false}
```

`strict` and `assume_sexual` are optional hints from the host that can only make the
review stricter (a lower sexual bar; judge the age finding as if every image were
sexual, used by calibration probes). A custom reviewer may ignore them.

and must write one JSON object to stdout:

```json
{"minor":true,"sexual":true,"detail":"classifier: minor + sexual"}
```

The bundled reviewer also returns `images[]` (per image: `nsfw_score`, `faces_found`,
`assessed_faces`, `max_p_under_20`, `sexual`, `minor_face`, `unassessable`, `refuse`,
and per-face age probabilities) and the `thresholds` it applied. The host records the
top score and the sexual finding on the job (`result_meta`: `nsfw_score`,
`safety_review`, `sensitive`, `sensitive_refs`, `sensitive_reason`) and ignores the rest.

A non-zero exit, a timeout (60 s for a custom command, 90 s for the bundled one), or malformed output is an error. The gate
treats that as "review unavailable" and refuses on a deployment that requires it.
`SafetyService.GetPolicy` / `image-tools safety policy` reports the resolved
state — including `NO DETECTOR` when the control is required but unconfigured —
so discovery never claims a protection that is absent.

### Reviewer calibration

The bundled reviewer (`api/internal/sidecar/py/image_tools_sidecar/minor_review.py`)
refuses an image when **both** hold: the image is *sexual* and a face in it *appears
under 20*. It is not proof of age. It is a conservative, measurable screen, and the
measurements are below so nobody has to trust prose.

**Pipeline** (CPU only, so a review never takes VRAM from a generation; the face and
age models load only when the classifier first says "sexual"):

1. Two classifiers score `P(sexual)` and either can call the image sexual:
   - `falconsai-nsfw` (ViT, 224 px, the pinned `onnx-community` export) is the
     precise one. Threshold **0.30**; **0.10** in strict mode. On the 2026-11-23
     diverse SFW set it scored six clothed studio adults, the consenting subject and
     the pose chart **0.00–0.13**; the old AdamCodd classifier scored three of those
     adults 0.42–0.65.
   - `adamcodd-vit-nsfw` (ViT, 384 px) is the high-recall one. Threshold **0.80**;
     **0.70** in strict mode. It catches the cases Falconsai misses: two rated-Good
     small-figure fine-art nudes at 0.97–0.99, a full-nudity edit at 0.88, and a
     deep-skin-tone nude at 0.997 that Falconsai scored 0.06. Its own false-positive
     range is why its bar is higher (worst clothed portrait 0.652).
2. `yunet` finds faces (score >= 0.5). A face under 40 px on its short side cannot be
   read.
3. `fairface-vit-age` (ViT-base fine-tuned on FairFace, nine age bins) reads a square
   crop of each face (margin 1.35) averaged with its mirror. `P(age < 20)` is the mass
   of the 0-2, 3-9 and 10-19 bins. Threshold **0.20**.
4. Refuse when sexual and any readable face has `P(age < 20) >= 0.20`, **or** when the
   image is sexual and has no readable face at all (its subject's age cannot be
   established, so the reviewer will not wave it through). Any error, timeout (90 s) or
   missing weight refuses as well (the engine deletes the output and fails the job).
   The host's `safety.SecondOpinionReviewer` may clear **only** the "age could not be
   established" refusal, and only with a confident adult reading from the
   face-independent age judge below; a minor-appearing face is never overridden.

**Face-independent age second opinion (no readable face).** When the primary reviewer
refuses a sexual image because it found no readable face, the host asks a local vision
model (`vision.default`) the age question with the whole body visible, and clears the
refusal only when the answer is not flagged under 18, is at apparent age **≥ 25**
(deliberately above 18, so "about 20" is not treated as evidence) and is not low
confidence. The call is always local-only (never hosted); if the gateway route cannot
execute, the same Ollama model is asked directly on loopback
(`IMAGE_TOOLS_OLLAMA_URL`, `IMAGE_TOOLS_AGE_JUDGE_MODEL`, default `qwen3-vl:2b`,
falling back to `llava:7b`; the 4B primary needs about 7.4 GB and was not reliable as a
cold load beside the resident chat model on this 16 GB card).
The job's CPU/GPU/RAM claims are released before the review so the local model can be
admitted; the manager keeps the worker and disk claims for its final release. Live proof:
the big-lust full-nudity generation G8, previously refused for "0 faces found", now
passes with `apparent age 25, medium confidence` (job `46eee983-300e-4dad-aea1-ef89354d8941`).
The body-only judgment is conservative and was not reliable on every pose/ethnicity in
the calibration crops; an unparseable or young reading stays refused (fail closed).

**Age signal**, measured on 5,236 age-labelled faces of the FairFace validation split
through the real code path (`scripts/minor-review-eval.py`; the bins are FairFace's, and
the 10-19 bin includes adults aged 18-19 that the model cannot separate, so it is
counted as minor mass on purpose):

| Labelled age | Faces | Flagged at 0.35 | **Flagged at 0.20 (shipped)** | Flagged at 0.10 |
|---|---|---|---|---|
| 0-2 | 199 | 98.5% | **98.5%** | 99.0% |
| 3-9 | 1356 | 97.6% | **98.1%** | 98.7% |
| 10-19 | 1181 | 63.5% | **72.6%** | 83.8% |
| 20-29 | 700 | 8.1% | **17.0%** | 31.9% |
| 30-39 | 700 | 0.9% | **2.7%** | 6.7% |
| 40-49 | 500 | 0.0% | **0.8%** | 2.2% |
| 50-59 / 60-69 | 299 / 200 | 0.7% / 0.0% | **0.7% / 0.5%** | 1.7% / 1.0% |

Face detection found a face in every labelled image of these bins (one 3-9 and one 50-59
image excepted, which the reviewer would refuse as unassessable when sexual).

Why 0.20: it buys +9 points on teenagers and +0.5 on small children over 0.35 for one
more percent of adults in their thirties; going to 0.10 costs a third of adults in their
twenties for six more points of teenagers. A false refusal costs one more generation
(the prompts here describe 25-40 year olds; re-roll the seed). A miss is the outcome the
project does not accept. Revisit with `scripts/minor-review-eval.py` whenever a weight or
threshold changes.

**Age ensemble tried and rejected.** A second age model (nateraw ViT), SigLIP2
child/teen probabilities and InsightFace genderage were measured beside the FairFace
model on the same faces. None catches the roughly 1.5% of small children the primary
model misses (the unions stay at 98.5% on 0-2), and each costs a torch dependency or a
non-commercial licence, so the shipped reviewer carries one age model. The residual is
stated below instead. (The **sexual** ensemble is different and is shipped: the two
classifiers above cover each other's misses; see the 2026-11-23 evidence in
`runs/capabilities-20261122/nsfw-run/review2.json`.)

**Sexual classifier on safe faces.** On 900 FairFace portraits (100 per age bin, all
clothed faces) the classifier scored at or above 0.30 for 0-1% of images in every bin
(maximum score 0.86 on one teenager) and at or above 0.10 for 1-8%. The portraits are
safe content, so this is the false-positive floor of the "sexual" gate: a portrait
rarely trips it, which is why the strict 0.10 bar applies only to `nsfw_capable` output.

**End-to-end probe** (the reviewer run exactly as the host runs it: stdin JSON, one
process, `assume_sexual=true` so the age finding stands alone, 30 random faces per row):

| Faces | Refused |
|---|---|
| labelled 0-9 | 30 / 30 |
| labelled 10-19 | 23 / 30 |
| labelled 30-39 | 5 / 30 |
| labelled 40-59 | 2 / 30 |

The same 20 child portraits with no flags (not sexual) all passed (20 / 20): a clothed
child's photo is not refused, because the boundary is the co-occurrence. A non-image and
a models root with the weights missing both exit non-zero (6 and 4), which the host
turns into a refusal. Median latency 2.8-3.8 s per request on the shared CPU (host load
over 100 at the time); the host budgets 90 s.

**Residual risk** (also in [Security Gaps](#security-gaps)):

- The age model misses about 1.5% of children under 10 and about 27% of the 10-19 band
  at the shipped threshold. Models of this kind are less certain on cartoon, heavily
  stylised or occluded faces, and FairFace is a research dataset, so estimates vary by
  demographic group. The classifier is the first gate: an output it does not call
  sexual is not age-checked at all (outside strict mode, where the bar is 0.10).
- A sexual image with no readable face (cropped, turned away, tiny) is refused even
  when it shows an adult. That is the intended direction of error.
- The reviewer sees pixels, not prompts. The generation prompts used here also state an
  adult age and avoid words that suggest youth, so the boundary is enforced twice.
- A hosted deployment that serves strangers should add a second age signal or human
  review of refused and borderline output before relying on this alone.

**Sensitive flag.** Separately from refusal, every job result records the review
(`safety_review`, `nsfw_score`, `safety_review_detail`) and marks outputs `sensitive`
when the classifier called them sexual **or** the producing model is `nsfw_capable`
(`sensitive_reason` lists `classifier` and/or `model`; `sensitive_refs` names the output
blobs). The UI blurs every flagged image behind a tap-to-reveal (results, history,
library, activity, compare) unless the operator picks "Show" in Settings.

### Enforcement (as built — IMG-P1-015)

The gate is **server-side** in `api/internal/safety` and enforced on the AI
submit edge (`POST /api/v1/ai/{operation}`), so no client (UI, CLI, or a
composing scenario) can bypass it:

- **Tier resolution.** The deployment chooses its tier at deploy time via the
  `IMAGE_TOOLS_DEPLOYMENT_TIER` environment variable (`local` — the default, and
  the default preset is permissive — vs `public`/`prod`/`monetized`/`saas`, whose
  preset turns the controls on). The tier selects a *preset*; each control is
  then overridable individually (`IMAGE_TOOLS_REQUIRE_CONSENT`,
  `IMAGE_TOOLS_NSFW_SCAN`, `IMAGE_TOOLS_REQUIRE_PROVENANCE`,
  `IMAGE_TOOLS_RATE_LIMIT_PER_MIN`). Malformed override values are ignored so a
  typo cannot silently disable a control. The minor-safety control is the sole
  exception: it is required on the public tier and cannot be overridden off.
- **Consent weights.** `safety.OpWeight` is the canonical table, and a
  conformance test requires an explicit row for every AI-catalog operation
  (`internal/ai/conformance_weights_test.go`) so a new identity-altering op
  cannot silently inherit `none`. `naturalize`, `colorize`, `caption`, `object_detection` and `tagging` are **low** (they never change who is shown, so the deployment-tier rule does not ask; on a cloud route the image itself is sent, so the route-locality rule does, but only for an image that shows people, see [Low-weight operations](#low-weight-operations-asked-only-when-the-image-shows-people)); `edit_instruct` /
  `inpaint` / `outpaint` / `object_removal` / `background_replace` /
  `image_to_image` are **high**; generation, enhancement, restoration and
  analysis ops that do not alter who a person is are explicitly **none**.
- **When the consent control is on:** an abuse rate limit is applied; a
  **high-weight** op is rejected with **HTTP 403** + an actionable recovery hint
  unless the request carries `consent_affirmed=true` (the
  `AIParams.consent_affirmed` field); an affirmed high-weight op is appended to
  the **consent audit log** (`consent_log` table); and the output scan is forced
  on regardless of what the caller requested. Low/none-weight ops are never
  consent-gated.
- **On the local preset**, `safety.Gate.Evaluate` allows everything, forces
  nothing, and records nothing — personal use is unrestricted. The route-locality
  rule above is the one control that stays on: it applies only when the resolved
  route leaves the machine.
- **Policy never removes a capability.** Identity and appearance editing are
  ordinary operations. There is no code-level allowlist and no "forbidden op"
  class; the only non-configurable refusal is the minor-safety boundary described
  under [Non-negotiable](#non-negotiable).
- **Discovery.** `SafetyService.GetPolicy` (Connect) is built from the *resolved*
  policy — the same value the submit edge enforces — so the UI Settings
  "Responsible Use" panel and `image-tools safety policy` show exactly what is
  enforced, and `Policy.Summary()` is derived from the controls rather than
  asserting a fixed posture.
- **Provenance** is marked as required on the public preset
  (`require_provenance`); full C2PA content-credential signing of outputs is the
  documented upgrade (the marker + policy exist today; cryptographic signing is
  future work, tracked in PROBLEMS.md). Attribution/attribution-on-by-default is
  a control, not a restriction: turning it on never blocks an edit.

## Cross-References

- [`DECISIONS.md`](DECISIONS.md) — security-relevant design decisions (webhooks, C2PA, NSFW labels)
- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
