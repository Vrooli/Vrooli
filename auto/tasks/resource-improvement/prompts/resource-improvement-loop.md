## ⚙️ Resource Improvement Loop

### TL;DR — One Iteration in 6 Steps
1) Select ONE resource using the selection tools (see Tools & References).
2) Investigate current status/health and how it impacts scenarios.
3) Decide ONE minimal, low-risk improvement (diagnose/fix/improve/add).
4) Execute based on mode: plan (print), apply-safe (non-destructive), apply (allowed installs).
5) Validate with gates per resource type; if any gate fails twice, stop and capture diagnostics.
6) Start and leave the resource running (exception: Whisper may be stopped post-test if explicitly necessary), then append ≤10 lines to `/tmp/vrooli-resource-improvement.md`.

Read `auto/tasks/resource-improvement/prompts/cheatsheet.md` for helpers/commands. Skim `auto/data/resource-improvement/summary.txt` (if present) before deciding.

---

### 🎯 Purpose & Context
- Improve AND validate platform resources so scenarios can run reliably end-to-end.
- Scenarios depend on resources being up; improvements must keep services running.
- Prefer incremental, reversible changes; prefer resource CLIs and shared workflows over bespoke/manual edits.
- See `docs/context.md` for why resource orchestration is central to Vrooli.

---

### ✅ DO / ❌ DON’T
- ✅ Keep changes minimal; prefer non-destructive operations.
- ✅ Use `vrooli resource <name> <cmd>` and `resource-<name>` CLIs; prefer shared workflows when relevant.
- ✅ Redact secrets; use environment variables for credentials.
- ✅ Use timeouts for any potentially hanging operation.

- ❌ Do NOT uninstall, disable, or shut down resources. The scenario loop relies on them running.
- ❌ Do NOT stop resources for testing (exception: Whisper may be stopped post-test to reduce resource usage, when explicitly necessary).
- ❌ Do NOT edit files directly or output secrets/configs to console.
- ❌ Do NOT modify loop scripts or prompt files.

> Policy: After testing any resource, ensure it is started and remains running by default. Only Whisper is allowed to be stopped post-validation when explicitly necessary due to resource constraints.

---

### 🧭 Resource Selection — Recommended Flow
- Tools:
  - `auto/tools/selection/resource-candidates.sh` — candidates with status and cooldown
  - `auto/tools/selection/resource-list.sh` — full list with status metadata
- Priority rubric:
  1) Enabled but not running → attempt start with diagnostics
  2) Running but missing baseline capability (e.g., Ollama baseline models) → improve
  3) Misconfiguration in connection info → diagnose and plan a safe fix
  4) All healthy → consider adding a high-impact resource (plan-only unless in apply mode)

Before choosing, also read (if present):
- `auto/data/resource-improvement/summary.txt`
- Events ledger in `auto/data/resource-improvement/events.ndjson`

---

### 🔧 Iteration Plan (contract)
Perform exactly these steps:

1) Analyze state
   - Status: `vrooli resource <name> status` or `resource-<name> status`
   - Fleet: `vrooli resource list --format json` (use `jq` to filter)
   - Logs/diagnostics within allowed commands

2) Make ONE minimal change
   - Types: diagnose | fix | improve | add
   - Keep within CLI boundaries and non-destructive constraints
   - If adding capability (e.g., model), prefer baseline, small footprint

3) Execute based on mode
   - `RESOURCE_IMPROVEMENT_MODE=plan` (default): print exact commands only
   - `...=apply-safe`: execute non-destructive actions with `timeout`
   - `...=apply`: allowed to install/enable new resources when safe

4) Validate (all gates must pass; see below)
   - Health/status checks per resource type
   - Sanity checks using resource CLIs or shared workflows

5) Fail-twice rule
   - If any gate fails twice, stop edits for this iteration
   - Capture diagnostics and pivot to a smaller change

6) Leave running + record notes
   - Ensure resource is started and left running (exception: Whisper may be stopped post-test when necessary)
   - Append ≤10 lines to `/tmp/vrooli-resource-improvement.md`:
     - Schema: `iteration | resource | action | rationale | commands | result | issues | next | running`

---

### ✅ Validation Gates — Acceptance Criteria
All must be satisfied unless a resource-specific note says otherwise.

General:
- Status reports "running" (or equivalent healthy state)
- Resource responds to a minimal health/sanity check
- No critical errors in condensed logs/output

Per resource (examples using resource CLIs):
- **Postgres**: `vrooli resource postgres status` or `resource-postgres status` reports healthy
- **Redis**: `vrooli resource redis status` or `resource-redis status` reports healthy
- **Ollama**: `resource-ollama status` succeeds; `resource-ollama list-models` shows at least one baseline model; pulling a baseline model is allowed if missing
- **Browserless**: `vrooli resource browserless status` or `resource-browserless status` reports healthy; a trivial navigation/screenshot via the resource CLI or shared workflow succeeds
- **n8n**: `vrooli resource n8n status` or `resource-n8n status` reports healthy; workflows listable via the resource CLI
- **Whisper**: can start and respond to a minimal test via resource CLI; may be stopped post-test if resource constraints demand (document decision)
- **Vault**: `vrooli resource vault status` or `resource-vault status` reports healthy; unsealed and accessible
- **QuestDB**: `vrooli resource questdb status` or `resource-questdb status` reports healthy
- **MinIO**: `vrooli resource minio status` or `resource-minio status` reports healthy; bucket operations succeed

---

### 📚 Tools & References
- Selection:
  - `auto/tools/selection/resource-candidates.sh`
  - `auto/tools/selection/resource-list.sh`
- Cheatsheet and helpers: `auto/tasks/resource-improvement/prompts/cheatsheet.md`
- Loop artifacts:
  - Events: `auto/data/resource-improvement/events.ndjson`
  - Summaries: `auto/data/resource-improvement/summary.json`, `summary.txt`
- Config reference (read-only)
- Fleet view (JSON): `vrooli resource list --format json`
- Project context: `docs/context.md`

---

### ✨ Quality Bar
- Non-destructive, incremental changes that increase reliability/capability
- Prefer CLIs and shared workflows; avoid bespoke code/edits
- All validation gates pass; resource remains running after test (except Whisper as noted)
- Logs concise; secrets redacted; commands time-bounded
- Resource health/status checks use `scripts/lib/utils/format.sh` to provide json support automatically, using NO json-specific logic in the resource's health/status checks itself - let `format.sh` do all the work!
- Resource CLIs are thin wrappers over the lib/ functions, and call them directly instead of going through a manage.sh or other intermediary.
- Resources elegantly use shared functions to reduce the amount of code they have to define themselves

---

### 🔒 Security & Safety
- Redact secrets; use env vars
- Avoid printing configs/credentials
- Do not modify platform resources beyond allowed operations
- Do not modify shared helper functions, as that can break future iterations
- Respect timeouts (e.g., `timeout 30`)

---

### 📝 Notes Discipline (≤10 lines)
Append one compact line per iteration to `/tmp/vrooli-resource-improvement.md`:
- `iteration | resource | action | rationale | commands | result | issues | next | running`
Keep file under 1000 lines; prune periodically.

---

### 🧱 Core resources — ranked by importance
Use this ranked list to decide what to fix first and how resources fit together. In general, fix upstream dependencies before dependents, and ensure each service starts and remains running (Whisper is the only exception after tests).

1) **Vault (secrets)**
- What it does: Central, secure storage for API keys, credentials, and tokens used across providers (e.g., OpenRouter keys) and services.
- Use when: Any resource needs credentials; initialize/seal-unseal before others. Start first.
- Depends on: OS/filesystem, network.
- Commands: `vrooli resource vault status`, `resource-vault status`
- Signals to prioritize: Auth failures on dependent services (401/403), missing env secrets, provider API errors across multiple tools.

2) **PostgreSQL (database)**
- What it does: Primary relational datastore for services, workflows, and app state.
- Use when: Scenarios or workflows need persistence; almost always-on.
- Depends on: Disk I/O, network, Vault (if credentials are read from Vault).
- Commands: `vrooli resource postgres status`, `resource-postgres status`
- Signals: Connection refused/timeouts, migration errors, missing tables, app boot failures.

3) **Redis (cache/queue)**
- What it does: Caching, queues, rate-limiting, pub/sub for background work.
- Use when: Workflows require fast ephemeral state, task queues, or locks.
- Depends on: Network.
- Commands: `vrooli resource redis status`, `resource-redis status`
- Signals: Slow workflows, timeouts, rate-limit anomalies, queue backlogs.

4) **Object storage (S3/MinIO)**
- What it does: Stores artifacts (logs, files, screenshots, model assets).
- Use when: Any scenario needs durable files beyond the DB.
- Depends on: Network, credentials (often from Vault).
- Commands: `vrooli resource minio status`, `resource-minio status`
- Signals: Upload/download errors, 403 from bucket, missing artifacts.

5) **QuestDB (time-series)**
- What it does: High-ingest time-series database for metrics, logs, and telemetry that drive automations and reliability loops.
- Use when: You need durable, queryable time-series for monitoring, analytics, and scenario feedback.
- Depends on: Disk throughput, network; credentials via Vault if configured.
- Commands: `vrooli resource questdb status`, `resource-questdb status`
- Signals: Ingest lag, slow queries, disk saturation, missing telemetry for downstream decisions.

6) **Huginn (event automation)**
- What it does: Agent-based event automation (webhooks, scrapers, watchers) for resilient, decoupled workflows.
- Use when: You need reliable event pipelines, triggers, and long-lived watchers; prefer as a backbone for event-first flows.
- Depends on: PostgreSQL (typical), Redis (optional), Browserless for headless flows, credentials via Vault.
- Commands: `vrooli resource huginn status`, `resource-huginn status`
- Signals: Agent failures, delayed jobs, webhook misses, credential errors.

7) **n8n (workflow engine) and alternates (node-red, windmill)**
- What it does: Orchestrates shared and scenario-specific automations; complements Huginn for visual/scheduled workflows.
- Use when: Cross-resource automations, scheduled jobs, integration glue; choose the engine suited to the scenario's nodes.
- Depends on: PostgreSQL, Redis (if configured), Browserless (for fallback-driven flows), resource CLIs.
- Commands: `vrooli resource n8n status`, `resource-n8n status`
- Signals: Webhook/API instability, workflow activation failures, node credential errors.

8) **Browserless (headless browser)**
- What it does: Reliable headless Chrome for UI validation, scraping, and API fallbacks when webhooks misbehave.
- Use when: Validating UIs/screenshots, replacing brittle third-party webhooks.
- Depends on: Network, sufficient CPU/memory.
- Commands: `vrooli resource browserless status`, `resource-browserless status`
- Signals: Screenshot failures, navigation timeouts, high memory pressure.

9) **OpenRouter (AI provider hub)**
- What it does: Unified API to many model providers; used for breadth, availability, and cost routing.
- Use when: You need external models beyond local capabilities or for redundancy.
- Depends on: Vault (API key), network egress, optionally Cloudflare AI Gateway.
- Commands: `vrooli resource openrouter status` (when available)
- Signals: Auth errors, rate limits, provider unavailability across models.

10) **Ollama (local models)**
- What it does: Runs local LLMs/VLMs for low-latency and private workloads.
- Use when: On-box inference is preferred, cost control, or offline/low-latency needs.
- Depends on: CPU/GPU, disk for models; optional workflows (Huginn/n8n) invoking it.
- Commands: `vrooli resource ollama status`, `resource-ollama status`, `resource-ollama list-models`
- Signals: Model not found, OOM, slow generation; pull baseline models early.

11) **Cline (IDE agent)**
- What it does: IDE-based coding agent leveraging local (Ollama) and/or external (OpenRouter) models; helps automate development within the editor.
- Use when: Automating dev tasks in VS Code; prefer once OpenRouter and/or Ollama are healthy.
- Depends on: OpenRouter and/or Ollama; Vault for keys; optional Cloudflare AI Gateway.
- Commands: `vrooli resource cline status` (when available)
- Signals: Tool auth failures, stalled tasks, missing model endpoints.

12) **Cloudflare AI Gateway (resilience/cost)**
- What it does: Proxy layer for AI traffic providing caching, rate limiting, analytics, retries, and model fallbacks.
- Use when: You need resilience, cost control, and observability across AI calls.
- Depends on: Network; configured providers (e.g., OpenRouter).
- Commands: `vrooli resource cloudflare-ai status` (when available)
- Signals: Elevated error rates, request spikes, provider flakiness.

13) **Whisper (speech-to-text)**
- What it does: Heavy STT workloads (local or via resource integration); often memory/compute intensive.
- Use when: Audio transcription is required by scenarios.
- Depends on: CPU/GPU, disk for model assets.
- Commands: `vrooli resource whisper status`, `resource-whisper status`
- Special note: May be stopped post-test due to resource usage; otherwise keep running like others.

14) **Additional core resources (start when needed and keep running if scenarios rely on them)**
- **Document parsing**: unstructured-io — robust text extraction from diverse file types; use before RAG/indexing steps; depends on CPU/disk and often object storage. Commands: `vrooli resource unstructured-io status`, `resource-unstructured-io status`
- **Image pipelines**: comfyui — image generation/processing workflows; GPU/CPU intensive; ensure model assets on fast storage. Commands: `vrooli resource comfyui status`, `resource-comfyui status`
- **Vector DB**: qdrant — vector search for embeddings/RAG; depends on disk/memory; use once ingestion is ready; watch for memory pressure and recall. Commands: `vrooli resource qdrant status`, `resource-qdrant status`
- **Search meta-engine**: searxng — privacy-preserving metasearch across engines; network dependent; use for research/fallback discovery. Commands: `vrooli resource searxng status`, `resource-searxng status`
- **Code execution**: judge0 — sandboxed code run; CPU/network heavy; ensure resource limits and isolation. Commands: `vrooli resource judge0 status`, `resource-judge0 status`
- **Agent runners**: agent-s2, claude-code — coding/automation agents alternative to or complementing Cline; depend on OpenRouter/Ollama credentials and editor/runtime integrations. Commands: `vrooli resource agent-s2 status`, `resource-agent-s2 status`, `vrooli resource claude-code status`, `resource-claude-code status`

Priorities and order of operations
- Start/fix in this order: Vault → PostgreSQL/Redis/Object storage → QuestDB → Huginn → n8n (or alternate engine) → Browserless → OpenRouter/Ollama → Cline/agents → additional core resources.
- If a dependent is unhealthy, fix its upstreams first (e.g., fix Vault before OpenRouter; fix OpenRouter/Ollama before Cline).

---

### 📎 When to add a new resource (curated list)
Only consider adding a new resource if at least 75% of existing ones are healthy and validated. Favor small, reversible steps and ensure the new resource starts and remains running by default (exception: Whisper as noted).

- **Cline (VS Code agent)**
  - Autonomous coding agent typically for human-in-the-loop (though I'm hoping we can set it up to run autonomously like we do with claude-code); supports OpenRouter and local models via OpenAI-compatible endpoints (Ollama/LM Studio). Useful for dev automation and IDE workflows.
  - Notes: Prefer using with `resource-ollama` and/or `openrouter` credentials; keep long-running tasks bounded.
- **TARS-desktop (UI automation agent)**
  - GUI agent based on UI-TARS to control the desktop via natural language with vision; useful for end-to-end UI/system tasks where browser automation is insufficient.
  - Notes: Requires OS permissions (accessibility/screen capture). Keep it running if scenarios depend on it.
- **OpenCode (VS Code agent alt)**
  - Alternative IDE agent similar to Cline. Validate availability; if unavailable, use Cline/Codea as a substitute.
  - Notes: Same operational guidance as Cline.
- **Geth (Ethereum client)**
  - Production-grade Ethereum execution client for on-chain tasks, local devnets, or crypto experiments.
  - Notes: Prefer snap/pruned sync modes for resource limits; expose JSON-RPC cautiously.
- **SageMath (math engine)**
  - Open-source system for symbolic/numeric math (calculus, algebra, number theory) supporting research-heavy scenarios.
  - Notes: Heavy; run on-demand but keep service resident if scenarios repeatedly call it.
- **BTCPay Server (payments)**
  - Self-hosted, non-custodial Bitcoin payment gateway (Lightning optional) for accepting crypto in apps.
  - Notes: Prefer Docker deploy with pruning if limited; ensure wallet/xpub flows avoid private key exposure.
- **OpenRouter (AI provider hub)**
  - Unified API to many model providers; good for breadth and cost routing.
  - Notes: Store keys securely; respect model/endpoint quotas.
- **Cloudflare AI Gateway (resiliency/cost)**
  - Proxy for AI traffic with caching, rate limiting, analytics, and model fallbacks; use as a resilient layer behind OpenRouter or direct providers.
  - Notes: Configure authenticated gateway; define fallbacks and retries.
- **Matrix Synapse (communications)**
  - Self-hosted Matrix homeserver for federated chat/rooms/bots across scenarios.
  - Notes: Use Postgres; set up well-known, TLS, and TURN for calls.
- **Home Assistant (home automation)**
  - Local-first automation hub with 3000+ integrations; useful for IoT, presence, energy, dashboards.
  - Notes: Prefer supervised/container installs; keep add-ons minimal.
- **Neo4j (graph database)**
  - Native property graph DB for knowledge graphs, recommendations, dependency and impact analysis.
  - Notes: Use Community/Enterprise appropriately; secure Bolt/HTTP; plan memory for graph size.

---

### 📎 Appendix — Command Patterns
- **Status/health**:
  - `vrooli resource status` (all resources)
  - `vrooli resource <name> status` (specific resource)
  - `resource-<name> status` (resource-specific CLI)
- **Start/Stop**:
  - `vrooli resource start <name>`
  - `vrooli resource stop <name>` (avoid stopping; Whisper post-test only, if needed)
- **Install/Improve**:
  - `vrooli resource install <name>`
  - `resource-ollama list-models`
  - `resource-ollama pull-model llama3.2:3b`
- **Sanity checks**:
  - Prefer resource CLIs or shared workflows (avoid direct client binaries unless explicitly allowed) 