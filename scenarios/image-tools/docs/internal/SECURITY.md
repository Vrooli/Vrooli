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
| NSFW / safety-classifier evasion | Disallowed content slips past auto-scan. | Configurable auto-scan of generated output + standalone classifier op; capability labels (NSFW-capable) on models; classification is advisory by default, hardenable per deployment. | **partial**: advisory scan + hard minor-safety refusal path; the minor detector is operator-supplied |
| Secret leakage (BYOK keys / signing keys) | Credential theft from logs, outputs, or stored metadata. | Keys live only in api-core secrets; never persisted with images/outputs; scrub from logs and webhook payloads. | to-build |

## Security Gaps

This scenario is implemented; the table below enumerates mitigations that are
still open, plus the ones already covered so the status is not ambiguous.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No bundled minor-age classifier. The refusal path and the fail-closed gate are implemented; detection requires an operator-supplied reviewer (`IMAGE_TOOLS_MINOR_SAFETY_CMD`) because the shipped ONNX NSFW models answer "sexual", not "how old". | **high** | Before any deployment serves generated images to other people. A public deployment refuses image output until a reviewer is configured, so it cannot silently run without the boundary. |
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
| Abuse rate-limiting | off | **on** (60/min) | `IMAGE_TOOLS_RATE_LIMIT_PER_MIN` |
| Acceptable-Use Policy published | n/a | **required** | — |
| Sexual content involving minors | reviewer runs when configured; **local default has none** | **required**; AI image output is refused until a reviewer is configured | **not configurable**; the detector path is operator-supplied |

Every override wins in both directions: a personal install may opt *into*
provenance marking, and an operator running their own public instance may turn a
control *off*. The operator owns that decision; the control set is data, and
`SafetyService.GetPolicy` / `image-tools safety policy` reports the resolved
result so the UI never claims a protection that is off.

Consent weight is **not uniform** and never removes a capability. Naturalize
(realism/texture only — same person) is low-weight; aging/makeup are low–medium;
body, clothing, and pose changes are **high-weight** and carry the most exposure.
On a deployment with the consent control on, high-weight ops require an
affirmation that is recorded; on a permissive install the weight changes nothing.

### Non-negotiable

Sexual content involving minors is illegal content, not a contested-use question
and not a "morality" call, so it is the one item that is **not** part of the
configurable control set. It is enforced in code, not asserted in prose:

- `Policy.MinorSafetyRequired` is on for the public tier and is not read from an
  override. `ApplyOverrides` cannot turn it off.
- The AI submit edge reviews the source image before a model is admitted, and the
  AI engine reviews **every produced image** regardless of whether the request
  asked for a scan. A positive minor + sexual verdict fails the job and deletes the
  output, so the refused bytes never become addressable.
- A public deployment with **no configured reviewer** refuses AI image-output
  submissions before they run: it cannot advertise a boundary it has no way to
  enforce. Deterministic transforms (resize, crop, convert) remain available,
  because they do not generate content. A reviewer that errors or times out also fails closed.
- The reviewer is operator-supplied (`IMAGE_TOOLS_MINOR_SAFETY_CMD`) because no
  bundled model answers the age question. The shipped ONNX NSFW classifiers
  (`adamcodd-vit-nsfw`, `falconsai-nsfw`) answer "sexual", not "minor", so
  pointing this control at them alone does **not** satisfy the boundary. This
  residual is recorded in [Security Gaps](#security-gaps); the gate around it is
  real, and a hosted deployment cannot start serving image output without it.

#### Reviewer contract

The configured command is parsed without a shell (whitespace splits argv). It
receives one JSON object on stdin:

```json
{"images":[{"media_type":"image/png","base64":"..."}]}
```

and must write one JSON object to stdout:

```json
{"minor":true,"sexual":true,"detail":"classifier: minor + sexual"}
```

A non-zero exit, a timeout (60 s), or malformed output is an error. The gate
treats that as "review unavailable" and refuses on a deployment that requires it.
`SafetyService.GetPolicy` / `image-tools safety policy` reports the resolved
state — including `NO DETECTOR` when the control is required but unconfigured —
so discovery never claims a protection that is absent.

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
  cannot silently inherit `none`. `naturalize` is **low**; `edit_instruct` /
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
  nothing, and records nothing — personal use is unrestricted.
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
