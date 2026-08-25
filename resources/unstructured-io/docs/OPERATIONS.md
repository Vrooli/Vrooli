# Operations

`unstructured-io` is a Linux amd64 `managed-service` fallback handler for
broad document formats, typed elements, and OCR. It is optional for the
document-manager free path. Its pinned OCI filesystem is extracted into the
Vrooli artifact store and launched directly with the bundled Python runtime;
Docker is not part of resource lifecycle.

## Image and platform evidence

The pinned image is:

`downloads.unstructured.io/unstructured-io/unstructured-api@sha256:a43ab55898599157fb0e0e097dabb8ecdd1d8e3df1ae5b67c6e15a136b171a6c`

`docker manifest inspect` on 2026-08-17 returned exactly one platform:
`linux/amd64` (`sha256:9f1c0c46f2f721303c2f003c82fa0ca1724294ffb28e2e7ac2accef230d79239`).
The pulled image occupied 9,811,325,528 bytes (~9.81 GB). The resource
therefore claims no arm64, macOS, or Windows deployment architecture. The
image digest and the extracted artifact tree digest are both pinned in
`resource.json`.

## Measured runtime

On this host, a real `hi_res` parse of
`resources/doc-parse/testdata/corpus/scan.pdf` returned two positioned
elements in 6.12 seconds. `docker stats` observed 1.373 GiB resident memory
during the request; the manifest rounds the requirement to 1,536 MiB and two
CPU cores. These are warm-container measurements and do not include the
9.81 GB first pull.

The same synthetic low-resolution OCR probe previously took 9.143 seconds and
misread `SCANNED` as `SCANMED` and `scanned` as `scaimed`. This is why OCR
accuracy remains a documented limitation rather than an implied guarantee.

## Readiness

The shared control plane invokes `resource-unstructured-io health` as a
readiness check. The command calls `/healthcheck` and then submits a small
text document through the primary partition endpoint. A live service that
cannot complete that parse is unhealthy. `health`, `formats`, and `process`
are the supported resource CLI commands; lifecycle remains owned by
`vrooli resource ...`.

The frozen runtime does not contain `detectron2`; `hi_res` is therefore a
declared partial capability. Operators must treat that limitation as evidence,
not silently substitute a different processing strategy.

## Architecture boundary

- `resource.json` owns the pinned image, extracted artifact, platform claims,
  ports, exports, storage, and readiness semantics.
- `cli/` owns the thin command entrypoint.
- `cli/internal/unstructured` owns the typed HTTP client and probe behavior.
- No normal path invokes the retired shell files or a floating image tag.
