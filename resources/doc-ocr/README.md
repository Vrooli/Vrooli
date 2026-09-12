# Doc OCR

`doc-ocr` is the portable tier-2 OCR resource for scanned pages and raster
images. It uses a native Go CLI, a vendorable MIT model artifact, and no
container runtime or first-use network access.

The resource claims conditional desktop support for Linux, macOS, and Windows
on amd64 and arm64. `health` verifies the model checksum and runs a local
recognition probe. `ocr` returns positioned text runs with confidence values;
low-confidence results are visible to callers and can be escalated to the
optional `unstructured-io` handler.

```text
resource-doc-ocr health
resource-doc-ocr status
resource-doc-ocr capabilities
resource-doc-ocr languages
resource-doc-ocr ocr page.png --languages eng
```

The engine comparison, fixture evidence, checksum, and fallback limitations
are recorded in [docs/OPERATIONS.md](docs/OPERATIONS.md).
