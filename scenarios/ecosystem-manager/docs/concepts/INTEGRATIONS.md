# Integrations — Ecosystem Manager

This document is the canonical dependency contract for resources, other
scenarios, and third-party services used by ecosystem-manager.

## Purpose Of This Document

Use this document to answer:

- What does ecosystem-manager depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

Ecosystem-manager is an *orchestrator*: it does not run agents itself. It
queues work, hands execution to agent-manager, watches metrics, and decides
what to steer next. That makes its dependency contract central to its
behavior — the controller is only as available as its inputs.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| PostgreSQL | resource | yes | analytics, auto-steer, steering, insights | DB `vrooli_ecosystem_manager`; `schema.sql` bootstrap | `/health` unhealthy; execution state cannot persist. |
| agent-manager | scenario | yes | tasks, auto-steer | Start/stop/stream every agent run | Tasks cannot execute; queue stalls. |
| prompt-manager | resource/scenario | yes (for steering) | auto-steer (skills) | `GET /api/skills`, `POST /api/skills/sync` | Steering prompts/skills unavailable; graceful degradation. |
| claude-code | resource | yes | execution | AI model backing agent runs | Generation/improvement runs cannot proceed. |
| qdrant | resource | yes | discovery | Semantic search / similarity matching | Similarity-based discovery degraded. |
| ollama | resource | yes | analysis | Local models (`llama3.1:8b`, `llama3.2:3b`) | Local-model analysis unavailable. |
| minio | resource | yes | storage | Object storage | Object-backed features degraded. |
| openrouter | resource | no | recycler / model selection | `OPENROUTER_API_KEY` | Cloud-model recycler runs + model discovery unavailable. |
| redis | resource | no | caching | Optional cache layer | Falls back to uncached paths. |
| visited-tracker | scenario | no | campaign mgmt | Proxied via API | Visited-tracking unavailable; non-fatal. |

Declared in [CODE: `.vrooli/service.json`] under `dependencies.resources`
and `dependencies.scenarios`.

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| PostgreSQL | required | Source of truth for run history, metrics, and live execution state. | n/a |
| prompt-manager | required for steering | Skills catalog sync (`GET /api/skills`, `POST /api/skills/sync`) feeds Auto Steer prompts. | If steering moves off the skills catalog. |
| claude-code | required | Backs agent generation/improvement runs. | n/a |
| qdrant | required | Semantic similarity for discovery. | n/a |
| ollama | required | Local model inference for analysis heuristics. | n/a |
| minio | required | Object storage. | n/a |
| openrouter | optional | Cloud-model recycler + model discovery; `try_start`. | When `OPENROUTER_API_KEY` is provisioned. |
| redis | optional (disabled) | Caching layer; off by default. | When a hot path needs caching. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| agent-manager | **required** | The execution boundary — every agent run (start/stop/stream) goes through agent-manager. Ecosystem-manager never spawns agents directly. | Run lifecycle API |
| scenario-completeness-scoring | metric source (see note) | PRD completion percentage is used as a run metric (`prd_completion_before`/`prd_completion_after`). | Completion-percentage read |
| prompt-manager | required for steering | Steering prompts/skills for Auto Steer. | Skills sync API |
| visited-tracker | optional | Tracks visited resources/scenarios for campaign management; proxied via API. | Visited-tracking API |

> **Note on completeness scoring.** PRD completion is consumed today by
> parsing each scenario's `PRD.md` locally (`GetScenarioPRDStatus` in
> `api/pkg/discovery/scenarios.go`). The scenario-completeness-scoring
> capability is the intended authoritative source for that percentage, but
> it is **not currently declared in `.vrooli/service.json`**. Treat it as
> the metric's logical owner; wire it explicitly before relying on it as a
> live dependency.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| OpenRouter | optional | Cloud model provider for recycler workflows and model selection. | `OPENROUTER_API_KEY`; degrades gracefully when absent. |
| Others | not-applicable | No other external API, webhook, auth, or payment dependency. | Add when a requirement appears. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| agent-manager down | Run start/stream errors | Tasks cannot execute; queue **stalls** (entries stay in `queue/pending`/`in-progress`). | task-runner tests |
| scenario-completeness-scoring / PRD unavailable | PRD percentage cannot be read | PRD metric unavailable; a stop condition referencing it raises `MetricUnavailableError`. | stop-condition tests |
| PostgreSQL down | `PingContext` error | `/health` reports unhealthy dependency; execution state cannot persist. | health handler tests |
| prompt-manager down | Skills sync error | Steering prompts unavailable; Auto Steer degrades gracefully. | steering tests |
| openrouter missing key | No `OPENROUTER_API_KEY` | OpenRouter recycler runs + model discovery disabled; core flow unaffected. | n/a |

### Controller inputs — wired

These are the inputs to the **closed-loop controller** described in
[`CONTROL-MODEL.md`](CONTROL-MODEL.md). Both are integrated.

| Input | Role in the controller | Status |
|---|---|---|
| test-genie | Findings-based state — what's broken/risky feeds the controller's next move. | Wired — `findings.TestGenieRunner` is the DIAGNOSE/MEASURE audit runner. |
| development-toolchain-validator (DTV) | Skill trust + cost priors, an eligibility gate, and a Layer-1 thrashing-prevention layer over steering choices. | Wired — `DTVEligibilityFilter` + `DTVPriorProvider` over the `dtv.Client` read seam (fail-open when DTV is unreachable; proceed-cap-flag when the gate degrades). |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`CONTROL-MODEL.md`](CONTROL-MODEL.md) — the closed-loop controller and its planned inputs
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
