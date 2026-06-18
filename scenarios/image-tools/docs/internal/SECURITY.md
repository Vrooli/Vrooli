# Security — Image Tools

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

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
| Decompression bomb / pixel-flood image | Memory exhaustion, OOM, DoS at the ingestion boundary. | Enforce max decoded dimensions and pixel-count caps + max upload size before full decode; reject oversized inputs at the multipart/REST edge. | to-build |
| Malicious image file (crafted EXIF, embedded payloads, SVG with scripts) | Code execution / parser exploitation; SVG is the highest-risk surface. | Treat SVG as raster import only (rasterize in a constrained renderer, never execute embedded scripts); validate/normalize metadata; use hardened decoders. | to-build |
| Path traversal on overridable save location | Write outside intended storage, overwrite arbitrary files. | Canonicalize and confine the per-request save path to allowed roots; reject `..`/absolute escapes; default to blobstore. | to-build |
| SSRF via input-URL fetch and webhook callback URL | Internal-network probing / metadata-endpoint theft. | Allowlist/denylist outbound fetch + callback targets; block private/link-local ranges and cloud metadata IPs; cap redirects. | to-build |
| Resource exhaustion via unbounded jobs (GPU/disk) | GPU/VRAM contention, disk fill from model weights + outputs. | GPU-serializing job queue (one heavy job at a time); per-user/job concurrency limits; disk-space awareness before model installs and output writes. | to-build |
| Model supply-chain (tampered/poisoned weights) | Malicious or corrupted model code/weights executed locally. | Checksum-verified, opt-in downloads from declared sources in the model registry; reject on checksum mismatch; pin known-good sources. | to-build |
| Prompt injection in text-guided ops (inpaint/text-to-image prompts) | Coerced model behavior, safety-bypass attempts. | Treat prompts as untrusted; apply NSFW/safety auto-scan to output; capability labels surface model risk; do not interpolate prompts into shell/file paths. | to-build |
| NSFW / safety-classifier evasion | Disallowed content slips past auto-scan. | Configurable auto-scan of generated output + standalone classifier op; capability labels (NSFW-capable) on models; classification is advisory by default, hardenable per deployment. | to-build |
| Secret leakage (BYOK keys / signing keys) | Credential theft from logs, outputs, or stored metadata. | Keys live only in api-core secrets; never persisted with images/outputs; scrub from logs and webhook payloads. | to-build |

## Security Gaps

This scenario is **pre-implementation**; the table below enumerates the
mitigations that must be built, not deficiencies in shipped code.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Upload validation (dimension/pixel/size caps, decoder hardening) not yet implemented | high | Before the deterministic-op ingestion edge accepts external uploads. |
| SSRF allowlist for input-URL fetch and webhook targets not yet built | high | Before input-URL fetch or webhook callbacks ship. |
| Path confinement for overridable save location not yet enforced | high | Before per-request save-location override is exposed. |
| Signed-callback verification contract not yet implemented | medium | Before webhook callbacks ship (P1). |
| Model-download checksum verification not yet enforced | high | Before any opt-in model install path lands. |
| Secret isolation (no key persistence with images/logs) not yet audited | high | Before BYOK cloud tier or C2PA signing ships. |
| NSFW auto-scan defaults and hardening policy undefined | medium | Before AI generation output is exposed to end users. |

## Responsible Use & Deployment Gate

image-tools can edit images of real, identifiable people (inpaint,
instruction-edit, naturalize, smart-select). The **capability** is unrestricted
for local / personal use and during development. **Public or monetized
deployment** is gated behind a Responsible-Use checklist (OT-P1-015) — the
capability is built freely; only public *exposure* is gated:

| Control | Local / personal | Public / monetized deploy |
|---|---|---|
| NSFW/CSAM safety auto-scan | optional | **on by default** (CSAM refusal is non-negotiable) |
| C2PA content-credential provenance | optional | **on by default** (≥ real-person edits) |
| Acceptable-Use Policy published | n/a | **required** |
| Consent affirmation + logging on identity-altering ops | n/a | **required** |
| Abuse rate-limiting / monitoring | n/a | **required** |
| Hard non-goals (no recognition / face-swap / deepfake) | enforced | enforced |

Consent weight is **not uniform**: Naturalize (realism/texture only — same
person, same everything) is low-weight; aging/makeup are low–medium; body,
clothing (esp. removal), and pose changes are **high-weight** and carry the most
exposure. Most of this gate is policy + a few default flips, not new
architecture, and it must not block personal use or capability development.

## Cross-References

- [`DECISIONS.md`](DECISIONS.md) — security-relevant design decisions (webhooks, C2PA, NSFW labels)
- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
