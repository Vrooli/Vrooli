# `models` — declarative model registry (OT-P0-006)

This package owns the **model registry** and the **hardware-fit selector**.
The data lives in [`registry.seed.json`](registry.seed.json); the Go code that
loads, validates, and selects from it is built against the tests referenced by
[`requirements/06-model-registry`](../../../requirements/06-model-registry/module.json)
(`registry_test.go`, `selector_test.go`).

## What `registry.seed.json` is

The **v1 seed** for the registry: one entry per model/library backing each
image operation, with a CPU-capable commercial-clean **default** plus one
higher-quality tier per op. It was authored in the 2026-06-16 design workshop
from license-verified web research; the policy choices behind it are recorded in
[`docs/internal/DECISIONS.md`](../../../docs/internal/DECISIONS.md) and the
human-readable catalog (with rationale and the do-not-ship blocklist) is
[`docs/reference/model-registry.md`](../../../docs/reference/model-registry.md).

Coverage today: **49 entries** across **26 operations** (every op has a default),
plus a **23-entry blocklist** of popular-but-license-encumbered models.

## How it is expected to load

Embed the seed and merge user/custom entries on top:

```go
//go:embed registry.seed.json
var seedBytes []byte
```

1. Parse `seedBytes` into the registry on boot.
2. Validate every entry against the schema (malformed → reject, per `registry_test.go`).
3. Overlay user-managed entries (enable/disable, custom/fine-tuned local models)
   from persistent storage so the seed stays the read-only baseline.
4. The selector picks the best-fit **enabled** model for the probed host
   (via the `internal/capabilities` `CapabilityProbe` seam, which reads
   `vrooli host inventory --json` through `packages/vrooli-cli-go` — **not**
   system-monitor), honoring per-op `default_for` and any user override
   (per `selector_test.go`).

## Field schema

See `field_reference` and `operations_vocabulary` inside the JSON. Every entry
carries: `id`, `name`, `operations`, `default_for`, `tier`, `backend` +
`alt_backends`, `requires_comfyui` (always `false` — ComfyUI is an optional
plug-in only), `size_mb_approx` + `quant_variants`, `hardware`
(`cpu_capable`/`gpu_required`/`min_vram_gb`/`min_ram_gb`/`os_arch`/`speed_note`),
`io`, `capability_labels` (`nsfw_capable`/`license`/`commercial_use`/
`commercial_use_notes`/`base_model_lineage`/`known_risks`), `source`
(`download_url`/`source_repo`/`docs_url`/`update_source`/`assets[]`/`checksum`),
and `enabled`.

## ⚠️ Checksum policy — do NOT fabricate hashes

Every `source.checksum.value` is intentionally **empty** with
`status: "unverified-capture-on-download"` (or `"n/a-library"` /
`"n/a-computed"` / `"n/a-builtin"` where there is no downloadable model weight).
Hashes are **captured and pinned on first real download**, then `status` flips
to `"pinned"`; later downloads must match or fail. Hand-written hashes would be
false verification — never invent them.

Enabled weight-backed seed models must use `assets[]` with direct artifact URLs,
artifact `kind`, and positive `min_bytes`. A source repository, model card, or
release page is documentation, not an install source, and `models doctor` reports
it until the entry is migrated or disabled.

## ⚠️ License discipline (commercial product)

`commercial_use` is `yes` for every seeded model and the verified traps live in
`blocklist` (CodeFormer, GFPGAN, bria-RMBG, Ultralytics-YOLO, FastSAM,
InsightFace, SD3.5/Turbo, community-NC ESRGANs, …). Two rules from the research:

- **Check the *weights* license separately from the *code* license** — the worst
  traps pair a permissive code license with restricted weights (Surya,
  InsightFace, YOLO-NAS, Qwen-3B, LLaVA).
- **Exporting to ONNX/GGUF does not strip AGPL or a non-commercial weight license.**

`sizes`/`min_vram_gb` are approximate — verify at ingest. When adding entries,
keep the seed read-only and route user changes through the management layer.
