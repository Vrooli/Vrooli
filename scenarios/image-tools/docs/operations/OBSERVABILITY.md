# Observability — Image Tools

This document records logs, metrics, telemetry, health checks, and
business/product signals for the local-first image toolbox.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

The defining signals for image-tools are job-shaped (async queue) and
hardware-aware (fallback tiers, GPU pressure).

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| Job throughput | performance | job queue | Completed jobs per interval | Trends with load; sustained zero under queued jobs = stuck. |
| Per-op latency (p50/p95) | performance | measures domain | Detect slow/regressed ops per model | Per-model budget; p95 spike signals contention or wrong tier. |
| Queue depth / wait time | saturation | heavy-job queue | GPU contention visibility | Rising depth = serialized GPU backlog (expected under load, not under idle). |
| Fallback-tier usage | reliability | provider selector | How often Local-GPU → Local-CPU → BYOK fires | Frequent CPU/BYOK fallback signals missing/over-VRAM models. |
| VRAM headroom | saturation | root `vrooli host inventory` / `internal/hostinventory` probe (via `capabilities` seam) | GPU memory available at run time | Low headroom predicts fallback / OOM. |
| Model install / health | reliability | model manager | Checksum/`/ready` state of opt-in models | Failed download or unready resource gates the op. |
| Error rate | reliability | API | Op/job failure ratio | Near-zero in steady state; spikes warrant log review. |
| `/health`, `/ready` | health | API + resources | Surface and dependency reachability | Healthy locally; `/ready` true before a resource serves. |
| test-genie result | validation | `make test` | Correctness evidence | All required phases pass. |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Includes provider-selection and fallback-tier transition messages. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Per-job logs | job queue / worker | `make logs` (filter by job-id) | Each async job records submission, ETA, tier chosen, progress, and terminal status. |
| Structured job traces | SQLite `job_trace` table | Query by `job_id`, `operation`, or `model_id` | One terminal row per job: operation, model, backend, tier, lane, state, queue wait, run duration, result ref, and error. |
| Resource logs | on-demand model / ComfyUI resources | `make logs` | Surface download progress, checksum, and `/ready` gating for opt-in resources. |

## Metrics

Metrics are emitted through the **measures domain** — `cli/manifest.json`
measure blocks, enforced by the test-genie measures phase (OT-P0-012).

| Metric | Status | Notes |
|---|---|---|
| Per-model op latency (p50/p95) | active | Measure blocks per operation/model. |
| Throughput | active | Completed jobs per interval. |
| Queue wait time | active | Time from submission to start in the serialized GPU queue. |
| Fallback-tier usage | active | Count of Local-GPU / Local-CPU / BYOK selections. |
| Terminal job traces | active | `job_trace` records the exact model/backend/tier/lane and queue/run durations for each finalized job. |
| VRAM headroom | active | Captured from the root `vrooli host inventory` probe (the shared `internal/hostinventory` collector, via the `capabilities` seam) at run time. |
| Requirement coverage | active | Tracked through requirements + test-genie coverage artifacts. |
| Product activation | deferred | Define after real PRD users/workflows exist. |
| BYOK cost per op | deferred | Pre-op estimate exists; aggregated cost dashboards are not yet instrumented. |

## Alerts / Health

The lifecycle runs health checks for the API and UI surfaces. The API
exposes `/health` (reachability) and `/ready` (serve-readiness);
on-demand model/ComfyUI resources gate `/ready` until a download
completes and the backend is loaded, so a not-yet-ready model does not
silently fail a job. Add deployment-specific alerts (queue-depth
threshold, fallback-rate, error-rate, disk-pressure) only when a
deployment target and operator expectations are defined.

## Telemetry Gaps

Telemetry is at the pre-implementation / early-instrumentation stage.
Known gaps:

| Gap | Impact | Revisit Trigger |
|---|---|---|
| BYOK cost dashboards / aggregate cost telemetry | Pre-op estimates exist, but no historical cost view; cannot evaluate hosted/SaaS unit economics. | Before managed/SaaS deployment or monetization review. |
| Cross-vendor GPU metrics (AMD/Intel/Apple-Silicon) | VRAM/headroom metrics are reliable only for NVIDIA today. | When P2 GPU hardening lands in the platform `internal/hostinventory` collector (surfaced via the root CLI host-inventory contract). |
| Product usage telemetry (per-op adoption) | Cannot validate adoption or value delivery. | Before public launch or monetization review. |
| Retention / storage-growth metrics | Blob storage growth is not actively tracked. | Before any hosted tier or when disk pressure recurs. |
| Webhook delivery success/retry metrics | Best-effort callbacks are not yet instrumented. | When webhook callbacks (P1) ship. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
