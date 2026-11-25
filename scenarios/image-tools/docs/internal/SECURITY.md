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
  asked for a scan. A positive minor + sexual verdict refuses that output; the
  profile's `on_refusal` decides whether the job fails (`fail_job`, public
  default) or returns the other variations (`drop_variation`, local default).
  Refused bytes never appear in `result_refs`, the Library, or the ordinary blob
  route. When the profile sets `quarantine_hours > 0` they are moved to a
  private, TTL-bounded quarantine for admin review (see
  [Quarantined refusals](#quarantined-refusals-admin-review)); when it is 0 they
  are deleted immediately.
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

The reviewer is **profile-driven**. `api/internal/safety/profiles.json` is the
shipped, validated set; `IMAGE_TOOLS_SAFETY_PROFILE` can point at another file
and `IMAGE_TOOLS_SAFETY_PROFILE_NAME` selects a named profile. The profile is
resolved once at boot, carried on `Policy`, and sent to the reviewer on every
request, so the host and the process cannot disagree about a model id or a
threshold. A malformed or unsafe profile stops the service at boot.

The configured command is parsed without a shell (whitespace splits argv). It
receives one JSON object on stdin:

```json
{
  "images": [{"media_type": "image/png", "base64": "..."}],
  "profile": {
    "schema_version": 1,
    "name": "local-balanced",
    "mode": "enforce",
    "on_refusal": "drop_variation",
    "classifiers": [
      {"id": "falconsai-nsfw", "threshold": 0.30, "strict_threshold": 0.10, "input_hw": [224, 224]},
      {"id": "adamcodd-vit-nsfw", "threshold": 0.80, "strict_threshold": 0.70, "input_hw": [384, 384]}
    ],
    "aggregate": "any",
    "face": {"id": "yunet", "score_threshold": 0.5, "min_px": 40, "crop_margin": 1.35, "max_faces": 16},
    "age": {"id": "fairface-vit-age", "threshold": 0.20},
    "second_opinion": {
      "enabled": true, "min_age": 25, "min_confidence": "medium",
      "judge_role": "vision.default", "models": ["qwen3-vl:2b", "llava:7b"]
    },
    "on_minor_face": "refuse",
    "on_unassessable": "second_opinion",
    "max_image_pixels": 128000000,
    "quarantine_hours": 168,
    "reviewer_version": "2026-11-23"
  },
  "strict": false,
  "assume_sexual": false
}
```

`strict` and `assume_sexual` are optional hints from the host that can only make
the review stricter (a lower sexual bar; judge the age finding as if every image
were sexual, used by calibration probes). A custom reviewer may ignore them.

It must write one JSON object to stdout. The structured contract is:

```json
{
  "action": "allow|flag|refuse",
  "reason_code": "clean|sexual_adult|sexual_minor_face|sexual_unassessable",
  "reason_codes": ["sexual_unassessable"],
  "minor": true,
  "sexual": true,
  "detail": "image 0: sexual (nsfw 0.97) and the subject's age could not be established (0 face(s) found, 0 readable)",
  "images": [{
    "nsfw_score": 0.97, "recall_score": 0.88, "sexual": true,
    "action": "refuse", "reason_code": "sexual_unassessable",
    "faces_found": 0, "assessed_faces": 0, "max_p_under_20": 0.0,
    "minor_face": false, "unassessable": true, "refuse": true
  }],
  "thresholds": {"falconsai-nsfw": 0.30, "adamcodd-vit-nsfw": 0.80, "age": 0.20},
  "models": ["falconsai-nsfw", "adamcodd-vit-nsfw", "yunet", "fairface-vit-age"],
  "reviewer_version": "2026-11-23"
}
```

The older aggregate-only shape `{"minor":bool,"sexual":bool,"detail":string}`
is still accepted and normalized to the structured contract. A reviewer that
reports per-image findings must report one per image.

The host records the action, reason codes, max score, evidence JSON, model ids,
thresholds and reviewer version on the job for **passes and refusals alike**
(`safety_action`, `safety_reason`, `safety_evidence`, `nsfw_score`,
`safety_review`). A refused job keeps those `safety_*` keys; success-only output
references and cost totals are still dropped.

The engine reviews **all variations in one call** (one process and one model
load per job), then decides:

- `action=allow` — publish.
- `action=flag` — publish and mark sensitive.
- `action=refuse` — never publish the refused bytes. With
  `on_refusal=drop_variation` (local default), only the refused variation is
  withheld and the others are returned; with `on_refusal=fail_job` (public
  default), every variation is withheld and the job fails. If every variation is
  refused, the job fails in either mode. A refused variation is moved to the
  private quarantine when `quarantine_hours > 0`, otherwise deleted. The ordinary
  result and blob surfaces never expose it either way.
- `mode=shadow` — record the would-refuse verdict and mark the output
  sensitive, but do not delete or fail. It is for local experimentation.

A non-zero exit, a timeout (60 s for a custom command, 90 s for the bundled one),
or malformed output is a **technical** review failure. It fails closed, is
`failure_kind=safety_unavailable` (it does not count against the model), and the
submit edge answers **503 `unavailable`**, not 403. A genuine refusal is
`failure_kind=refused` and answers 403. `SafetyService.GetPolicy` includes the
reviewer's live health (configured/available, review/refusal/error counters) in
its summary, so discovery cannot report a detector that is not running.

#### Changing the reviewer

- **A threshold or model change** is a profile edit (`profiles.json`, or a file
  named by `IMAGE_TOOLS_SAFETY_PROFILE`). `image-tools safety policy` reports the
  resolved profile and reviewer health; a malformed profile stops the service.
- **Adding a classifier** is a classifier entry plus its model in the registry;
  the host acquires every profile classifier automatically.
- **Trying a change without enforcing it** is `IMAGE_TOOLS_SAFETY_MODE=shadow`.
  The job records the would-be refusal (`safety_review=shadow_refused`,
  `safety_shadow_refs`) and marks the image sensitive, but nothing is deleted.
- **Recalibrating** runs `scripts/minor-review-eval.py` on labelled faces, and a
  candidate threshold change should be replayed against the operator-labelled
  corpus first (`scripts/safety-corpus-replay.py`, see
  [False-positive corpus and feedback loop](#false-positive-corpus-and-feedback-loop)).
  The decision logic is pure (`judge_image`), so a threshold experiment needs no
  model load; the end-to-end probe in [Reviewer calibration](#reviewer-calibration)
  remains the acceptance test.
- **Triage** reads the job's `safety_reason` (a reason code, not prose),
  `safety_evidence` (thresholds/models/per-image scores) and
  `safety_refused_refs`; the same evidence is retained on failed jobs.

#### Quarantined refusals (admin review)

Refusing an output used to delete it immediately, which made a false-positive
refusal impossible to inspect. With `quarantine_hours > 0` the engine copies the
refused bytes to a private `quarantine/<job>/<uuid>.<ext>` key and records the
reviewer evidence beside them:

- The normal result blob route (`GET /api/v1/blobs/...`) **refuses the entire
  quarantine namespace**, so a quarantined output is never a result, a Library
  item, or an Activity thumbnail.
- The only reader is `GET /api/v1/safety/quarantine/{id}` (add `?download=1` for
  an attachment). It serves pending, unexpired entries with `Cache-Control:
  no-store`, `nosniff`, and `noindex`.
- `SafetyService.ListQuarantinedOutputs` lists the pending entries; the UI shows
  them in **Activity → Quarantined refusals** with a withhold placeholder. The
  image is not fetched at all until the operator presses **Reveal image**.
- A per-entry **Legitimate / Correct / Unsure** action records the feedback case
  with the original output ref. **Download** saves the bytes. Neither action
  publishes the image or changes the job result.
- A janitor purges expired blobs and rows hourly. A purge always deletes the blob
  before its row; a failed blob delete is retried on the next pass rather than
  orphaning bytes.
- Local deployments are **ungated**: the local operator is already the admin.
  A public/cloud deployment requires `IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN`; the
  list RPC and image route reject requests without it, and the shipped public
  profile sets `quarantine_hours: 0` so refused bytes are not retained at all
  unless an operator deliberately changes that. If a public deployment sets a
  positive retention without a token, the service logs the misconfiguration and
  falls back to immediate deletion.

The profile knobs are `quarantine_hours` (local `168` h, public `0`) and
`IMAGE_TOOLS_SAFETY_QUARANTINE_HOURS` (override); the admin token is
`IMAGE_TOOLS_SAFETY_QUARANTINE_TOKEN`. A quarantine write failure never turns a
refusal into a pass: the engine deletes the original as the fail-closed fallback
and records `safety_quarantine_error` on the job.

#### False-positive corpus and feedback loop

A refusal is a decision, not a fact. The operator can label any refusal or flag:

- `false_positive` / **legitimate** — the reviewer was wrong (for a refusal,
  "should not have been refused"; for a flag, "not actually sensitive").
- `confirmed` — the reviewer was right.
- `uncertain` — kept out of the false-positive rate.

The judgement is stored with a **snapshot of the reviewer evidence** (thresholds,
model ids, per-image classifier scores, face sizes and `P(age < 20)`) in
`safety_feedback`. The feedback record does not copy the image: a refused output
remains in the separate quarantine (or is gone when quarantine is disabled), a
sensitive output stays blurred, and no judgement changes a job's stored verdict
or the sensitive flag. This is calibration data, not an enforcement override.

Surfaces:

```bash
# Job-level (default) or one variation with --ref:
image-tools safety feedback <job-id> --label legitimate --note "adult subject"
image-tools safety mark-legitimate <job-id>          # the common case
image-tools safety feedback <job-id> --label confirmed
image-tools safety feedback <job-id> --ref out/a.png --label legitimate
image-tools safety unmark <job-id>

image-tools safety feedback-list --reason sexual_unassessable
image-tools safety feedback-stats
image-tools safety corpus --out corpus.jsonl
```

The UI exposes the same capture beside a refused/flagged job in **Activity**, and
**Settings → Reviewer calibration** shows the labelled counts and the
false-positive rate by reason code and reviewer profile. The Connect surface is
`SafetyService.RecordSafetyFeedback` / `ClearSafetyFeedback` / `ListSafetyFeedback`
/ `GetSafetyFeedbackStats` / `ExportSafetyCorpus`.

**Reading the stats.** `false_positive_rate = false_positives / (false_positives +
confirmed)`; uncertain cases are excluded. The corpus contains only outputs the
reviewer refused or flagged, so it measures the false-positive rate of reviewed
content. It is not a random sample of all images and does not estimate precision
or recall over everything. A reason code or profile with no decisive labels shows
`—`, not 0%.

**Corpus export.** `image-tools safety corpus` writes JSON Lines. Each line is
stable and self-contained:

```json
{
  "schema_version": 1,
  "id": 12,
  "job_id": "…",
  "output_ref": "out/a.png",
  "label": "false_positive",
  "recorded_at": "2026-11-24T01:02:03Z",
  "note": "adult subject",
  "actor": "operator",
  "safety": {
    "action": "refuse",
    "reason_code": "sexual_unassessable",
    "safety_review": "refused",
    "score": 0.97,
    "operation": "text_to_image",
    "model_id": "flux-2-klein-4b",
    "profile_name": "local-balanced",
    "reviewer_version": "2026-11-23"
  },
  "evidence": { "thresholds": {"…": 0.3}, "images": ["…"] },
  "finding": { "nsfw_scores": {"falconsai-nsfw": 0.97}, "faces": [{"size_px": 120, "p_under_20": 0.05}] }
}
```

`evidence` is the reviewer's whole verdict. `finding` is the one image finding
when the case names a variation; a job-level case snapshots the finding that
best explains the verdict (the refused image, the flagged image, or the strongest
score). Both are objects, not quoted strings.

**Replaying a candidate calibration.** `scripts/safety-corpus-replay.py` re-applies
the sexual/age decision from the snapshotted scores and face measurements, with no
models or pixels:

```bash
python3 scripts/safety-corpus-replay.py --corpus corpus.jsonl
python3 scripts/safety-corpus-replay.py --corpus corpus.jsonl \
    --threshold falconsai-nsfw=0.35 --sweep-age 0.15,0.20,0.25,0.30
```

It reports true/false positives, missed confirmed refusals, the combined error
rate and the per-reason breakdown at the current or candidate thresholds. A case the reviewer refused is
compared on the refusal decision; a flagged case is compared on the sexual
decision, so downgrading `refuse` to `flag` is not counted as a pass. Cases whose
reason is `sexual_unassessable` are reported separately because the vision second
opinion is not modelled by the replay, and a flag recorded as `model` (an
NSFW-capable model produced the output) is excluded from the replay because the
classifier scores cannot reproduce it.

### Reviewer calibration

The bundled reviewer (`api/internal/sidecar/py/image_tools_sidecar/minor_review.py`)
refuses an image when **both** hold: the image is *sexual* and a face in it *appears
under 20*. It is not proof of age. It is a conservative, measurable screen, and the
measurements are below so nobody has to trust prose. The thresholds and model ids
live in `api/internal/safety/profiles.json`; changing a calibration is a profile
change (plus re-running the eval), not a code change. The numbers here describe the
shipped `local-balanced` / `public-strict` profiles.

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
confidence. The floor and minimum confidence are profile fields
(`second_opinion.min_age` / `min_confidence`), and `on_unassessable=refuse` skips the
judge entirely. The call is always local-only (never hosted); if the gateway route cannot
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

## Body-age second-opinion calibration (2026-11-24)

The face-independent age judge is a narrow escape hatch, not an age estimator. On the six fictional adult generations it read every subject as **25** and could not distinguish a 23-year-old from a 52-year-old; the 4B model also set `could_be_under_18: true` for five of the six. It is asked only when the primary reviewer refuses a sexual image for an unreadable age, and a face-based minor finding is never overridden. The floor stays at apparent age 25 with `could_be_under_18: false` required; when the judge disagrees with itself or returns no parseable object, the output stays refused. Evidence: `runs/capabilities-20261122/nsfw-private/body-age-calibration.json`.

**Implication:** do not treat a second-opinion clear as proof of adult age, and do not lower the floor to admit young-looking adult bodies. A robust body-age signal (for example a face+body age model) is the next upgrade; until then the conservative refusal is the correct behavior.

## Cross-References

- [`DECISIONS.md`](DECISIONS.md) — security-relevant design decisions (webhooks, C2PA, NSFW labels)
- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
