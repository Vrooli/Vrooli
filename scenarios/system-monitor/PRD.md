# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview
- **Purpose**: Real-time system monitoring with AI-driven anomaly detection and automated root-cause investigation. System Monitor gives Vrooli infrastructure observability — live CPU/memory/disk/network/GPU metrics, threshold-based anomaly detection, and agent-driven investigations — so the platform can proactively manage its own health.
- **Primary users/verticals**: DevOps engineers, SREs, and system administrators; Vrooli operators and automated agents that need a reliable "is the system healthy?" signal and diagnostic context.
- **Deployment surfaces**: CLI (`system-monitor`: health, metrics, status, metrics/investigations/reports/settings/maintenance/capacity domains), API (proto-first Connect-RPC contract plus health/log/forensics/tool REST exceptions), UI (governed responsive Vega dashboard using the selected Signal Workbench / Optical Observatory A target).
- **Value promise**: Prevents downtime and reduces mean-time-to-resolution by surfacing anomalies early and auto-investigating them with AI, consolidating multiple monitoring tools into one integrated, self-healing-capable scenario.
- **Why it matters**: Shared observability is foundational infrastructure the whole platform builds on; AI root-cause analysis (threshold detection plus agent-manager-driven investigation) turns raw metrics into actionable findings; and the resulting performance baselines and anomaly patterns feed future auto-scaling, cost-optimization, and incident-response scenarios.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Real-time CPU, memory, disk, network, GPU monitoring | Live system metric collection across core resource dimensions
- [ ] OT-P0-002 | Threshold-based anomaly detection with configurable triggers | Detect anomalies against configurable warning/critical thresholds
- [ ] OT-P0-003 | Automated investigation of system anomalies via agent-manager | Spawn AI investigations on anomalies through agent-manager integration
- [ ] OT-P0-004 | Persistent time-series storage | The scenario shall persist metric history through its scenario-owned durable backend.
- [ ] OT-P0-005 | Configurable warning/critical thresholds | Manage per-metric thresholds via API settings endpoints
- [ ] OT-P0-006 | Report generation (daily/weekly) | Generate daily and weekly system reports via API endpoint
- [ ] OT-P0-007 | Governed responsive monitoring dashboard | The Vega UI shall deliver the selected Signal Workbench / Optical Observatory A experience across desktop and phone, preserve existing workflows and truthful metric states, and expose live states to assistive technology; the effective target is docs/internal/DESIGN-ADOPTION.md
- [ ] OT-P0-008 | Investigation script execution | The API shall expose the investigation catalog with typed native execution where collected facts are sufficient and explicitly declared shell-gated execution otherwise
- [ ] OT-P0-009 | Process monitoring and management | Zombie detection and process insight (UI has kill dialog; API kill endpoint not yet implemented)
- [ ] OT-P0-010 | Infrastructure monitoring | Monitor database pools, HTTP pools, message queues, and storage I/O

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-011 | Historical trend analysis | The scenario should expose metric history on a shared time axis so dashboard views can correlate trends with logs, forensic events, and investigations
- [ ] OT-P1-012 | Alert webhook support | Cooldown-based alert webhook delivery on threshold violations
- [ ] OT-P1-013 | Investigation cooldown management | Configurable cooldown period with reset capability
- [ ] OT-P1-014 | Portable agent profile adoption | Reconcile a scenario-owned role-only profile; Agent Manager owns runner and model resolution
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
- Preferred stacks / frameworks: Go API with proto-first Connect-RPC as the target contract, React + Vite UI using the selected Vega design target in docs/internal/DESIGN-ADOPTION.md with suitable governed shared components, Go CLI (`system-monitor`) generated from `cli/manifest.json` and migrating to generated Connect clients.
- Data + storage expectations: Scenario-owned persistence for metric history and investigation/configuration state; SQLite is the runtime repository default, with explicit non-production memory mode. Redis supports alert distribution.
- Integration strategy: agent-manager for AI investigations (Ollama `llama3.2:3b`); Redis for alert distribution; optional Grafana for advanced visualization; stable programmatic reuse through generated proto/Connect clients.
- Non-goals / guardrails: not a cloud-provider-native monitor (local systems first); no plugin system for custom collectors yet; no built-in authentication.

## 🤝 Dependencies & Launch Plan
- Required resources: `postgres` (thresholds/investigations/config), `redis` (alerts/pub-sub), `ollama` (AI anomaly analysis via agent-manager).
- Optional resources: `grafana` (advanced dashboards; disabled by default via `GRAFANA_URL`).
- Scenario dependencies: `agent-manager` (orchestrates AI-driven investigations via `api/internal/agentmanager/`).
- Operational risks: explicit development memory mode loses history on restart; false-positive alerts (mitigated by configurable thresholds + cooldown); dashboard performance under many streams; AI investigation timeouts (async processing + timeout limits).
- Launch sequencing: (1) core metric collection + thresholds; (2) anomaly detection + agent-manager investigations; (3) dashboard + alerting; (4) reports; (5) extend trend analysis, custom metrics, and multi-channel routing post-launch.

## 🎨 UX & Branding
- Look & feel: Vega — Signal Workbench composition with Optical Observatory A styling, five persistent history charts, the verified Vega logo, complete compact phone tasks, genuine code/JSON workspaces and calibrated dark/light appearances. The selected originals plus applicable owner amendments, retained capabilities and verified refinements form the effective target in docs/internal/DESIGN-ADOPTION.md.
- Accessibility: high-contrast mode, screen-reader support for alerts, keyboard navigation; CLI output human-readable by default with `--json` for machine consumption.
- Voice & messaging: technical, authoritative, slightly ominous — conveys constant vigilance and god-mode visibility without sacrificing clarity.
- Branding hooks: the selected Optical semantic palette with truthful severity coding, readable dense telemetry, the unchanged official Vega mark and purposeful reduced-motion-aware transitions; improvements over imperfect mockups require preserved behavior and evidence.

## 📎 Appendix

**Capability definition** — Core capability: real-time observability with AI-driven anomaly detection and automated root-cause analysis across all Vrooli resources and scenarios. Intelligence amplification: performance baselines guide agent execution; anomaly patterns catch cascading issues early; resource-usage profiles inform scheduling; diagnostic workflows feed self-healing. Recursive value: enables Auto-Scaling Orchestrator, Cost Optimization Advisor, Security Threat Detector, Performance Tuner, and Incident Response Manager scenarios.

**Performance criteria** — Metric collection < 100ms latency (scrape duration); anomaly detection < 30s from occurrence (alert timestamp comparison); dashboard refresh 5s current+detailed metrics / 60s process/infra/investigations / 4s agent status (WebSocket not implemented, UI polling interval); query performance < 500ms for 24h data once a persistent backend is implemented; AI investigation < 2min per anomaly (workflow execution time). Collectors should be efficient by construction: steady-state host collection should prefer native `/proc`/syscall reads, avoid shell pipelines, share process walks across metrics and attribution, and surface self-metrics so monitor overhead remains measurable.

**API surface (summary)** — Target contract: generated Connect services under `vrooli.system_monitor.v1.{health,metrics,investigations,reports,settings,maintenance,capacity,scripts}`. Proto-owned manual REST routes have been removed from the runtime router. REST exceptions are limited to Health `GET /health`, `GET /api/v1/health`; development pprof; raw logs/forensics; and tool discovery/execution protocols. Long-term REST exceptions are limited to ops probes, raw logs/forensics, third-party/webhook shapes, protocol-specific tool surfaces, and multipart/blob cases.

**CLI surface (summary)** — `system-monitor` (Go, manifest-backed): built-in lifecycle/status commands plus domain groups for `metrics`, `investigations`, `reports`, `settings`, `maintenance`, `capacity`, and overview/watch flows. `metrics processes --json` is the dogfood command for process health; the richer owner-attributed timeline remains an API/data surface. Scenario-domain commands use generated Connect clients.

**Known limitations** — WebSocket: UI types defined but not implemented; dashboard uses HTTP polling. UI proto-owned calls dispatch to Connect JSON procedure paths without adding a Connect-Web runtime dependency. Missing API endpoint referenced by UI: `POST /processes/{pid}/kill` (UI kill dialog silently fails — endpoint not in the API router). Custom metrics: limited to built-in collectors (CPU, Memory, Network, Disk, Process, GPU); no ingestion endpoint. Disk detail is observational and points remediation to storage-manager policy/audit instead of mutating host state. Storage: SQLite is the runtime default; explicit development memory mode is non-durable and refused in production. Authentication: none on API endpoints.

**References** — README.md (quick start); [Prometheus Metric Types](https://prometheus.io/docs/concepts/metric_types/).

---

**Last Updated**: 2026-10-25
**Status**: Full Vega redesign target defined; implementation acceptance pending. Operational-target completion requires current evidence.
**Owner**: AI Agent - Infrastructure Intelligence Module

**Redesign scope** — The full existing-product redesign covers all ten existing routes and shared workspaces under `docs/internal/DESIGN-ADOPTION.md`, including necessary enabling repairs. Existing operational targets retain their IDs, priorities and obligations. Deferred capabilities are outside this delivery scope; redesign acceptance requires evidence for the affected journeys and does not establish full product certification.
