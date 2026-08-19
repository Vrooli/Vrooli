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
