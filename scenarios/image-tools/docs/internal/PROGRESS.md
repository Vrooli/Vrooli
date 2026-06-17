# Progress — Image Tools

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-16 | Generator Agent | done | Scaffolded from `react-vite` template (design kit `vrooli-default`). Authored full `PRD.md` via prd-control-tower (13 P0 + 10 P1 + 5 P2 operational targets), validates healthy. Generated requirements registry: 28 modules (one per OT), all 28 targets linked, validates healthy. Fully authored docs foundation: concepts (DOMAINS/ARCHITECTURE/DATA/FLOWS/INTEGRATIONS), internal (DECISIONS/SECURITY/PERFORMANCE), operations (DEPLOYMENT/RUNBOOK/OBSERVABILITY), business (MONETIZATION/GO-TO-MARKET); updated manifest maturities. Orientation 5/8 (all doc gates green; remaining 3 are implementation-phase). |
| 2026-06-17 | Impl Agent | done | **Phase 0 (hardware probe).** Host facts now read through the root `vrooli host inventory --json` CLI (NOT system-monitor) via the typed client `packages/vrooli-cli-go`. Extended the consumer-subset proto `packages/proto/schemas/cli/v1/host_inventory.proto` (`HostInventoryResponse`) with `os`/`arch`/`cpu.cores`/`gpus[]` (the producer already emitted them via `CliHostSnapshot`); regenerated Go/TS/Python. Built `api/internal/capabilities` `Probe` seam (`CLIProbe` real + `FakeProbe`), live-verified decoding os/arch/cores/mem/GPU+VRAM. Swept ~10 spec docs to drop the stale "system-monitor dependency". |
| 2026-06-17 | Impl Agent | done | **Phase 1 integration (proto contracts + handlers + wiring + notes removal + service.json).** Authored proto `jobs` (JobsService: GetJob/WaitJob/ListJobs/CancelJob/WatchJob-stream) + `models` (ModelsService: ListModels/GetModel/ListOperations/SelectModel/SetModelEnabled/ListBlocklist) under `packages/proto/schemas/image-tools/v1/`; regenerated Go/TS/Python. Added `internal/models` SQLite enabled-state overlay (store + schema; seed stays read-only) and `internal/jobrunner` dispatcher seam (per-op handlers register here; empty in Phase 1 so a submit for an unknown op fails clean). Built Connect handlers `handlers/jobs` + `handlers/models` (jobs WatchJob server-streams the Manager's Subscribe; models SelectModel previews hardware-fit selection via the capabilities probe). Wired `main.go` (load+validate registry at boot, CLIProbe, durable Manager under a server-lifetime ctx + Close on shutdown) and the modules registry (jobs+models endpoints/schemas/proto-files). **Deleted the `notes` reference domain end-to-end** (api/cli/proto + gen). CLI: new `jobs`/`models` manifest groups + domain packages (`jobs watch` appended manually as the server-stream exception, documented in manifest `omitted`); cli_commands_seed + gen-endpoints test updated. `.vrooli/service.json`: optional ComfyUI resource + optional ffmpeg hostTool (both never-required per headless tenet). **DEVIATIONS:** `common.proto` deferred to Phase 2 (its op-envelope/image-ref messages have no consumer yet — would be dead code); `/measures` substrate unmounted until the first op declares a measure. API + CLI green, race+lint clean, e2e boot test green; UI rework (notes→jobs/models) in progress. |
| 2026-06-17 | Impl Agent | done | **Phase 1 spine internals (all green, race+lint clean).** `internal/models` registry loader+validator+hardware-fit selector over `registry.seed.json` (rejects malformed; seed-integrity gate: commercial-clean [`no` rejected, `conditional` must be disabled+annotated], ComfyUI-optional, every op has a default). `internal/storage` api-core blobstore seam (outside-repo, per-request `OutputTarget` override) + decompression-bomb ingest guard. `internal/backends` Provider seam + boot invariant (≥1 standalone non-ComfyUI provider per op) + Local→ComfyUI→BYOK selection ladder. `internal/jobs` server-owned durable job system (SQLite-persisted, GPU-serialized + CPU-concurrent lanes, block-once Wait, Cancel, SSE progress Subscribe, restart recovery, disconnect-survival). **Seed fix:** enabled `birefnet-general` (MIT, CPU-capable) — it was the lone op-default seeded `enabled:false`, leaving `background_replace` with no enabled CPU default (violates headless tenet). **Remaining Phase 1:** proto contracts (jobs/common/models) + Connect handlers + service.json hostTools/optional resources + delete `notes` reference domain + server/modules wiring. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
