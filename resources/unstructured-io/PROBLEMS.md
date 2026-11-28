# Unstructured.io Resource — Known Constraints

## Current constraints

### Docker and platform coverage

The resource requires Docker or Docker Desktop. Its pinned image is
approximately 9.81 GB and publishes only `linux/amd64`; arm64 is not a
declared route. macOS amd64 and Windows amd64 remain conditional Docker
Desktop routes and have not been smoke-tested on this host.

### Startup and resource cost

The image takes substantial disk space on first install. A warm `hi_res` scan
parse measured 6.12 seconds and 1.373 GiB resident memory on 2026-08-17. The
manifest rounds the runtime requirement to 1,536 MiB RAM and two CPU cores.
Startup and first-pull time are separate from the warm parse measurement.

### OCR accuracy

Tesseract and the layout pipeline are useful fallbacks, but low-resolution
synthetic text is not uniformly accurate. The recorded 900x240 probe took
9.143 seconds and returned `SCANMED` for `SCANNED` and `scaimed` for
`scanned`. Callers must preserve confidence and position metadata and may
escalate when accuracy is insufficient.

## Resolved stale documentation debt

The resource no longer has a required normal path through the retired
shell-era files. That shell-owned lifecycle contract is not part of the
current resource. The current operator surface is the manifest, the Go CLI,
and the shared resource control plane.

## Resolved: mutable Matplotlib cache changed the staged artifact

On 2026-11-02, the first OCR request created
`home/notebook-user/.cache/matplotlib/fontlist-v390.json` under the immutable
staged artifact. Owner status then reported `needs_reacquire` and an artifact
hash mismatch. The manifest now points `XDG_CACHE_HOME` and `MPLCONFIGDIR` at
its declared relocatable `RESOURCE_CACHE_DIR`. A post-fix owner reacquisition
and single synthetic OCR request confirmed that `fontlist-v390.json` was
created under `/home/matthalloran8/.cache/vrooli/resources/unstructured-io/`
and the artifact hash remained pinned at
`374c913bafc0a33fb051711205fb515b31fe0d61856a95f0dca63fc3eb515714` before,
during and after the request. The resource was stopped through its owner after
the check. This resolves artifact-cache mutation only; the same synthetic
image-only OCR request still returns `Unable to get page count` and remains an
unresolved capability limitation.


## 2026-11-02 — invoice PDF fast-path failure and RAM admission repair

A valid one-page synthetic text PDF was detected as `text/plain` by the pinned Unstructured image, then failed with HTTP 500 while decoding the PDF bytes as UTF-8. A corrected image-only PDF was detected as `application/pdf`, but `fast` returned an empty array with HTTP 200 and no review signal. Neither result supports invoice readiness. The current resource exposes only raw partition elements, so callers must not infer acceptance from an empty success.

The first bounded trial also exposed an owner contract gap: RAM/CPU requirements existed without a capacity claim, and the shared capacity companion could only select VRAM. The manifest now carries a measured 1.5 GiB RAM claim and required capacity-sync companion. The shared companion accepts resource kind and defaults to VRAM for existing callers; PII Detector explicitly selects RAM. Owner validation and focused resource/shared-package tests passed. A no-reclaim start was admitted, the required companion stayed alive, and owner stop released the claim. Detailed frozen evidence is in the local research run's `invoice-layout-discriminator/RESULTS.md`.


## Resolved caller-side PDF MIME and empty-result ambiguity — 2026-11-02

The resource CLI now sends a known `.pdf` input as `application/pdf` instead of multipart's default `application/octet-stream`. On a valid ASCII-heavy synthetic PDF, Unstructured otherwise guessed `text/plain` and tried to decode PDF bytes as UTF-8. With explicit media type, the same frozen PDF returns all fields in the one known invoice fixture. The CLI now also rejects a successful empty element array. The image-only `fast` request returned an empty array (no OCR text), which the CLI surfaces as an error. These changes repair caller MIME and empty-result signaling only; they do not add OCR, structured invoice fields, a human-review workflow, or general layout support.
