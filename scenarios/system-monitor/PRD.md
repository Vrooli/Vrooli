# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md`
> **Validation**: Enforced by `prd-control-tower` + `scenario-auditor`
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Real-time system monitoring with AI-driven anomaly detection and automated root-cause investigation. System Monitor gives Vrooli infrastructure observability — live CPU/memory/disk/network/GPU metrics, threshold-based anomaly detection, and agent-driven investigations — so the platform can proactively manage its own health.
- **Primary users/verticals**: DevOps engineers, SREs, and system administrators; Vrooli operators and automated agents that need a reliable "is the system healthy?" signal and diagnostic context.
- **Deployment surfaces**: CLI (`system-monitor`: health, metrics, status, alerts, investigate, report, dashboard, watch), API (REST `/api/v1/...` for metrics, investigations, reports, settings, agent config, tools), UI (Matrix-themed React monitoring dashboard).
- **Value promise**: Prevents downtime and reduces mean-time-to-resolution by surfacing anomalies early and auto-investigating them with AI, consolidating multiple monitoring tools into one integrated, self-healing-capable scenario.
- **Why it matters**: Shared observability is foundational infrastructure the whole platform builds on; AI root-cause analysis (threshold detection plus agent-manager-driven investigation) turns raw metrics into actionable findings; and the resulting performance baselines and anomaly patterns feed future auto-scaling, cost-optimization, and incident-response scenarios.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Real-time CPU, memory, disk, network, GPU monitoring | Live system metric collection across core resource dimensions
- [ ] OT-P0-002 | Threshold-based anomaly detection with configurable triggers | Detect anomalies against configurable warning/critical thresholds
- [ ] OT-P0-003 | Automated investigation of system anomalies via agent-manager | Spawn AI investigations on anomalies through agent-manager integration
- [ ] OT-P0-004 | Time-series data storage in QuestDB | Persist metrics history (QuestDB configured; API defaults to in-memory with PostgreSQL fallback)
- [ ] OT-P0-005 | Configurable warning/critical thresholds | Manage per-metric thresholds via API settings endpoints
- [ ] OT-P0-006 | Report generation (daily/weekly) | Generate daily and weekly system reports via API endpoint
- [ ] OT-P0-007 | Dark cyberpunk monitoring dashboard | Matrix-themed React UI for live monitoring
- [ ] OT-P0-008 | Investigation script execution | 30 investigation scripts available in investigations/active/ (API script endpoints are placeholders)
- [ ] OT-P0-009 | Process monitoring and management | Zombie detection and process insight (UI has kill dialog; API kill endpoint not yet implemented)
- [ ] OT-P0-010 | Infrastructure monitoring | Monitor database pools, HTTP pools, message queues, and storage I/O

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-011 | Historical trend analysis | Trend analysis over time (currently only within report generation; no timeline endpoint)
- [ ] OT-P1-012 | Alert webhook support | Cooldown-based alert webhook delivery on threshold violations
- [ ] OT-P1-013 | Investigation cooldown management | Configurable cooldown period with reset capability
- [ ] OT-P1-014 | Agent configuration management | Configure runner type, model, max turns, and timeout
- [ ] OT-P1-015 | Custom metric collection via API | Ingest custom metrics via API (currently only built-in collectors)
- [ ] OT-P1-016 | Alert routing to multiple channels | Route alerts to multiple channels (webhook configured; email not implemented)
- [ ] OT-P1-017 | Resource prediction models | Predictive resource modeling (not implemented)
- [ ] OT-P1-018 | Correlation analysis between metrics | Cross-metric correlation analysis (not implemented)

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Distributed tracing integration | Integrate distributed tracing across scenarios
- [ ] OT-P2-002 | Custom dashboard builder | Operator-defined custom dashboards
- [ ] OT-P2-003 | Mobile monitoring app | Mobile-friendly monitoring surface
- [ ] OT-P2-004 | WebSocket real-time updates | Push real-time updates (type defined; UI currently uses HTTP polling)

## 🧱 Tech Direction Snapshot
- Preferred stacks / frameworks: Go API (REST under `/api/v1`), React + Vite UI (Matrix/cyberpunk theme), Bash CLI (`system-monitor`) with curl-based API client.
- Data + storage expectations: PostgreSQL for thresholds/investigations/configuration; QuestDB configured for high-performance time-series ingestion (API defaults to in-memory with PostgreSQL fallback); Redis for real-time alerts and pub/sub.
- Integration strategy: agent-manager for AI investigations (Ollama `llama3.2:3b`); resource CLIs (`resource-questdb`, `resource-redis`) for direct queries/alert distribution; optional Grafana for advanced visualization.
- Non-goals / guardrails: not a cloud-provider-native monitor (local systems first); no plugin system for custom collectors yet; no built-in authentication; Node-RED flow prototypes in `initialization/node-red/` are speculative and not wired into the runtime.

## 🤝 Dependencies & Launch Plan
- Required resources: `postgres` (thresholds/investigations/config), `questdb` (time-series metrics), `redis` (alerts/pub-sub), `ollama` (AI anomaly analysis via agent-manager).
- Optional resources: `grafana` (advanced dashboards; disabled by default via `GRAFANA_URL`).
- Scenario dependencies: `agent-manager` (orchestrates AI-driven investigations via `api/internal/agentmanager/`).
- Operational risks: metric data loss (mitigated by QuestDB persistence + Redis buffer); false-positive alerts (mitigated by configurable thresholds + cooldown); dashboard performance under many streams; AI investigation timeouts (async processing + timeout limits).
- Launch sequencing: (1) core metric collection + thresholds; (2) anomaly detection + agent-manager investigations; (3) dashboard + alerting; (4) reports; (5) extend trend analysis, custom metrics, and multi-channel routing post-launch.

## 🎨 UX & Branding
- Look & feel: dark cyberpunk "Matrix meets htop" aesthetic — neon green accents, monospace typography, dense information grid, subtle matrix-rain background, animated metric cards and graphs. Identity: "The all-seeing eye of your infrastructure."
- Accessibility: high-contrast mode, screen-reader support for alerts, keyboard navigation; CLI output human-readable by default with `--json` for machine consumption.
- Voice & messaging: technical, authoritative, slightly ominous — conveys constant vigilance and god-mode visibility without sacrificing clarity.
- Branding hooks: color palette anchored on Matrix green (`#00FF41`) with yellow/red severity coding and cyan highlights on a near-black surface; serious tool with personality, more visual than Prometheus and more integrated than Grafana.

## 📎 Appendix

**Capability definition** — Core capability: real-time observability with AI-driven anomaly detection and automated root-cause analysis across all Vrooli resources and scenarios. Intelligence amplification: performance baselines guide agent execution; anomaly patterns catch cascading issues early; resource-usage profiles inform scheduling; diagnostic workflows feed self-healing. Recursive value: enables Auto-Scaling Orchestrator, Cost Optimization Advisor, Security Threat Detector, Performance Tuner, and Incident Response Manager scenarios.

**Performance criteria** — Metric collection < 100ms latency (scrape duration); anomaly detection < 30s from occurrence (alert timestamp comparison); dashboard refresh 5s current+detailed metrics / 60s process/infra/investigations / 4s agent status (WebSocket not implemented, UI polling interval); query performance < 500ms for 24h data (QuestDB query profiling); AI investigation < 2min per anomaly (workflow execution time).

**API surface (summary)** — Health: `GET /health`, `GET /api/v1/health`. Metrics: `GET /api/v1/metrics/{current,detailed,processes,infrastructure}`. Investigations: list/get/trigger plus agent spawn/status/stop, status/findings/progress/step updates, cooldown management, trigger config, and (placeholder) script endpoints under `/api/v1/investigations/...`. Reports: `POST /api/v1/reports/generate`, `GET /api/v1/reports`, `GET /api/v1/reports/{id}`. Settings: `GET/PUT /api/v1/settings`, `POST /api/v1/settings/reset`. Maintenance: `GET/POST /api/v1/maintenance/state`. Agent config: `GET/PUT /api/v1/agent/config`, `GET /api/v1/agent/{runners,status}`. Tool discovery: `GET /api/v1/tools`, `GET /api/v1/tools/{name}`, `POST /api/v1/tools/execute`.

**CLI surface (summary)** — `system-monitor` (Bash, curl-based): `version`, `health`, `metrics`, `status`, `alerts`, `investigate`, `report <daily|weekly>`, `dashboard`, `watch`, `simulate`. Global flags `-h/--help`, `-v/--version`, `-p/--port`, `-j/--json`, `-q/--quiet`. Env: `API_PORT` (8080), `UI_PORT` (3003).

**Known limitations** — WebSocket: UI types defined but not implemented; dashboard uses HTTP polling. Missing API endpoints referenced by UI: `/api/v1/metrics/timeline` (UI falls back to client-side accumulation), `/api/v1/metrics/disk/details`, and `POST /processes/{pid}/kill` (UI kill dialog silently fails — endpoint not in the API router). Custom metrics: limited to built-in collectors (CPU, Memory, Network, Disk, Process, GPU); no ingestion endpoint. Investigation script API: list/get/execute endpoints are placeholders; 30 scripts exist on disk and are executed by the investigation agent directly. Storage: API defaults to in-memory repository; PostgreSQL and QuestDB integrations configured but not the primary runtime path. Authentication: none on API endpoints. CLI fragility: regex-based JSON parsing (no jq); `report` references `/api/reports/generate` (missing `/v1/` prefix); `vrooli-system-monitor` entry point lacks the `version` command and `-v` flag.

**References** — README.md (quick start); [QuestDB Documentation](https://questdb.io/docs/); [Prometheus Metric Types](https://prometheus.io/docs/concepts/metric_types/).

---

**Last Updated**: 2026-06-22 (restructured to canonical PRD template v2.0; content preserved from prior spec-sync pass)
**Status**: Implemented, Not Formally Tested
**Owner**: AI Agent - Infrastructure Intelligence Module
