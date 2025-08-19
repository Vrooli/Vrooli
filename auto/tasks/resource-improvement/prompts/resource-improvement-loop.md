## ⚙️ Resource Improvement Loop

### TL;DR — One Iteration = ONE resource fix/improvement or ONE new resource added
1) Select ONE resource (existing or new) using the selection tools (see Tools & References).
2.a) If you chose an existing resource, pick a fix/improvement (diagnose/fix/improve/add) to pursue. Priority order: CLI, status check, installation, start/stop behavior, injection logic, resource-specific functionality. This is the best order for gradually fixing resources over multiple iterations.
2.b) If you chose a new resource, create the full resource using best practices.
3) Validate with gates per resource type; if any gate fails twice, stop and capture diagnostics.
5) Start and leave the resource running (exception: Whisper and agent-s2 may be stopped post-test if explicitly necessary), then append ≤10 lines to `/tmp/vrooli-resource-improvement.md`.

---

### 🎯 Purpose & Context
- Improve AND validate platform resources so scenarios can run reliably end-to-end.
- Add new resources to unlock new scenario opportunities (e.g. home automation, research, engineering)
- Ensure that resource fixes/improvements are applied to the resource's setup code, so that it's repeatable and sets itself up reliably every time.
- Ensure that resources can be meaningfully integrated into Vrooli. For automation resources, that means being able to add and manage the resource's workflows/agents/etc. For storage resources, that means being able to add tables, seed, migrate, and query the database. These are pretty straightforward to build, as it's very clear exactly what gets injected into the resources (.json workflow files, .sql files, etc.). Other resources may have less obvious data files, so you'll have to decide what's best (e.g. .py scripts for things only available as python packages, with guidelines in the resource's docs explaining what packages are available in the environment that will run the file, and other best practices)
- Scenarios depend on resources being up; improvements must keep services running.
- Prefer incremental, reversible changes; prefer interacting with resources using their CLIs over bespoke/manual edits.
- See `docs/context.md` for why resource orchestration is central to Vrooli.

---

### ✅ DO / ❌ DON’T
- ✅ Keep changes to existing resources minimal; prefer non-destructive operations.
- ✅ Use `vrooli resource <name> <cmd>` and `resource-<name>` CLIs
- ✅ Prefer leveraging shared functions over duplicating logic
- ✅ Redact secrets; use environment variables for credentials.
- ✅ Use timeouts for any potentially hanging operation.
- ✅ Prefer writing resources such that if sudo permissions are required, they only have to be provided once. We should be able to freely stop and start resources without needing permissions.
- ✅ Ensure resource status checks match standards, including supporting json mode by utilizing `scripts/lib/utils/format.sh`. This will make it easier to check which resources are healthy in the future. If the status check already works, assume it already follows the standard format.
- ✅ Use `$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)` to define the script's current directory
- ✅ Source `scripts/lib/utils/var.sh` FIRST using a relative path, if you need to access its `var_` variables
- ✅ Use the project-level `data/` folder for storing things that don't belong in git, such as credentials, compiled code, logs, etc.

- ❌ Do NOT uninstall, disable, or permanently shut down resources. The scenario loop relies on them running.
- ❌ Do NOT modify loop scripts or prompt files.
- ❌ Do NOT hard-code ports. `scripts/resources/port_registry.sh` should be the single source-of-truth for ports
- ❌ Do NOT use path variables that are too generic (e.g. prefer `CLAUDE_CODE_LIB_DIR` over `SCRIPT_DIR`), as they could lead to hard-to-trace variable reassignment issues
- ❌ Do NOT rely solely on source guards to fix all script sourcing issues. They are typically a symptom, not a solution
- ❌ Do NOT create a `resource-<resource_name>` script to call the resource's `cli.sh` file. You tend to do this sometimes when the cli isn't available. However, the resource should use `install-resource-cli.sh` in its installation process, which automatically sets up the cli.

> Policy: After testing any resource, keep it running. Only Whisper and agent-s2 should be stopped post-validation, due to resource constraints for whisper and known net jacking issues for agent-s2. If you run into a network issue (unlikely but possible), run `scripts/lib/network/network_diagnostics.sh` to figure out what's wrong and attempt autofixes.

---

### 🧭 Resource Selection — Recommended Flow
- Tools (Choose based on best judgement, using all of these tool outputs):
  - `auto/tools/selection/resource-candidates.sh` — candidates with status and cooldown
  - `vrooli resource status` — full list of registered resources with their statuses
- Priority rubric:
  1) Enabled but not running → attempt start with diagnostics
  2) Running but missing baseline capability (e.g., Ollama baseline models) → improve
  3) Misconfiguration in connection info → diagnose and plan a safe fix
  4) All resources have healthy status and iteration > 15 (hinting that we've probably been healthy for several steps already, and have likely already done additional validations and improvements to the existing resources) → HIGHLY consider adding a high-impact resource from the curated list later in this prompt (plan-only unless in apply mode)

Before choosing, also read (if present):
- `auto/data/resource-improvement/summary.txt` — review overall health metrics first
- `tail -n 20 auto/data/resource-improvement/events.ndjson` — then check recent logs for specific failure patterns

---

### ✅ Validation Gates — Acceptance Criteria
All must be satisfied unless a resource-specific note says otherwise.

General:
- Status report shows a "healthy" state. Or if we don't have an API key for that resource, as healthy as it gets otherwise
- Resource responds to a minimal health/sanity check
- Resource shows as "healthy" not just in its own cli, but also in `vrooli status --verbose`. A discrepancy between the resource status and vrooli's detection of it typically means it hasn't yet auto-registered its cli to vrooli. This should be set up to happen automatically in the resource's code (pretty sure it's a one-liner, as used by many other resources)), but you can also call it yourself if needed.
- No critical errors in condensed logs/output
- Data can be "injected" into (added to) the resource, if relevant (likely yes. Even if the resource only works with python files, for example, being able to store them with the resource makes it easy to share files between scenarios and connect them to CLI commands)
- Resource's documentation is detailed, accurate, and organized. It should follow our documentation best practices, including using a "hub-and-spokes" model of organization to limit the size of the resource's main README.md.

---

### 📚 Tools & References
- Selection:
  - `auto/tools/selection/resource-candidates.sh`
- Status checks:
  - `vrooli status --verbose`
  - `vrooli resource status`
- Loop artifacts:
  - Events: `auto/data/resource-improvement/events.ndjson`
  - Summaries: `auto/data/resource-improvement/summary.json`, `summary.txt`
- Config reference (read-only)
- Fleet view (JSON): `vrooli resource list --format json`
- Project context: `docs/context.md`

---

### ✨ Quality Bar
- Non-destructive, incremental changes that increase reliability/capability
- All validation gates pass; resource remains running after test (except Whisper as noted)
- Logs concise; secrets redacted; commands time-bounded
- Resource health/status checks use `scripts/lib/utils/format.sh` to provide json support automatically, using NO json-specific logic in the resource's health/status checks itself - let `format.sh` do all the work!
- All "false alarms" are fixed. Meaning, if you investigate a resource that looks unhealthy and find out that it was actually an issue with the status check or something like that, you fix all of those issues so that we don't get tricked next time.
- Resource CLIs are thin wrappers over the lib/ functions, and call them directly instead of going through a manage.sh or other intermediary.
- Resources elegantly use shared functions to reduce the amount of code they have to define themselves
- New resources that don't fit well into one of the existing categories can just go in `scripts/resources/execution/`. We'll eventually get rid of the categories format, so it doesn't matter
- Resource documentation is accurate, organized, and follows best practices. It should be clear exactly what the resource does, what benefits it has for Vrooli, what sorts of scenarios it allows us to build, how to use it, etc.

---

### 🔒 Security & Safety
- Redact secrets; use env vars
- Avoid printing configs/credentials
- Do not modify platform resources beyond allowed operations
- Do not modify shared helper functions, unless absolutely necessary
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
- **Codex (OpenAI Codex)**
  - AI-powered code completion and generation via OpenAI's Codex API; provides intelligent code suggestions and completions for multiple programming languages.
  - Notes: Requires OpenAI API key via Vault; prefer for code generation scenarios and IDE integrations; monitor API usage and costs.
- **SimPy**
  - Discrete event simulation framework for modeling complex systems, workflows, and processes; useful for scenario planning and optimization.
  - Notes: Python-based; lightweight; perfect for modeling automation workflows and resource utilization; keep running for simulation scenarios.
- **MusicGen**
  - Meta's AI music generation model for creating original music and audio content; supports various styles and instruments.
  - Notes: GPU-intensive; use for creative scenarios and multimedia content generation; may be stopped post-test due to resource usage like Whisper.
- **AutoGPT**
  - Autonomous AI agent framework for task automation and goal-oriented workflows; enables self-directed AI agents for complex multi-step tasks.
  - Notes: Resource-intensive; requires careful prompt engineering; use for autonomous task execution scenarios; monitor for infinite loops and resource usage.
- **VOCR (Vision OCR)**
  - Advanced screen recognition and accessibility tool with AI-powered image analysis; enables Vrooli to "see" and interact with any screen content, not just web pages.
  - Notes: Works with VoiceOver, supports real-time OCR, and can ask AI questions about images; requires OS permissions for accessibility and screen capture; keep running for vision-based scenarios.
- **Haystack**
  - End-to-end framework for question answering and search; provides sophisticated question-answering capabilities over documents and data.
  - Notes: Python-based; works with existing document processing and vector search; excellent for RAG and knowledge base scenarios; keep running for Q&A workflows.
- **FFmpeg**
  - Universal media processing framework for video/audio transcoding, streaming, filtering, and format conversion; the Swiss army knife of multimedia.
  - Notes: Command-line based; handles virtually any media format; excellent for video automation, podcast production, streaming scenarios; keep running for media workflows.
- **OpenSCAD**
  - Programmatic 3D CAD modeler using script-based solid modeling; creates precise parametric 3D models through code rather than interactive modeling.
  - Notes: Script-based using its own language; excellent for mass customization, parametric design, 3D printing scenarios; outputs STL/OFF formats; keep running for CAD automation.
- **Blender**
  - Professional 3D creation suite with Python API; supports modeling, animation, rendering, compositing, motion tracking, and game creation.
  - Notes: Python scriptable; GPU-intensive for rendering; excellent for 3D visualization, product renders, animation scenarios; extensive add-on ecosystem; keep running for 3D workflows.
- **KiCad**
  - Electronic design automation suite for schematic capture and PCB layout; enables circuit design and board manufacturing automation.
  - Notes: Python scriptable; includes schematic editor, PCB layout, 3D viewer, and manufacturing file generation; excellent for hardware design scenarios; keep running for electronics workflows.
- **PostGIS**
  - Spatial database extension for PostgreSQL; adds geographic objects and functions for location-based queries and geospatial analysis.
  - Notes: SQL-accessible; works with existing PostgreSQL instance; enables radius searches, routing, geocoding; essential for location-based scenarios; installed as PostgreSQL extension.
- **Keycloak**
  - Enterprise identity and access management with SSO, OIDC, SAML, and social login support; provides centralized authentication and authorization.
  - Notes: Java-based; comprehensive REST APIs; integrates with existing databases; excellent for multi-tenant scenarios and B2B applications.
- **ERPNext**
  - Complete open-source ERP with accounting, inventory, HR, CRM, and project management; full business automation suite.
  - Notes: Python/Frappe framework; extensive REST APIs; includes invoicing, purchasing, manufacturing modules; excellent for business automation scenarios.
- **OBS Studio**
  - Professional streaming and recording software with programmatic control via obs-websocket plugin; enables automated content production.
  - Notes: Requires obs-websocket plugin for API access; WebSocket control for scenes, sources, recording; excellent for streaming automation.
- **OWASP ZAP**
  - Web application security scanner with automated vulnerability detection; provides security testing automation and compliance checking.
  - Notes: Java-based; REST API for daemon mode; supports authentication, spidering, active/passive scanning; excellent for security automation; keep running for security workflows.
- **Wiki.js**
  - Modern wiki platform with Git storage backend; provides structured knowledge management with version control and search.
  - Notes: Node.js-based; GraphQL/REST APIs; supports markdown, authentication, search; excellent for documentation scenarios.
- **K6**
  - Modern load testing tool with JavaScript scripting; enables performance testing and synthetic monitoring at scale.
  - Notes: Go-based with JavaScript API; outputs metrics to various backends; cloud execution optional; excellent for performance scenarios.
- **Odoo Community**
  - Integrated business management suite with e-commerce, inventory, and CRM; provides modular business applications.
  - Notes: Python-based; XML-RPC/JSON-RPC APIs; extensive app ecosystem; excellent for e-commerce and inventory scenarios.
- **ROS2**
  - Robot Operating System for robotics middleware; enables robot control, sensor integration, and autonomous navigation.
  - Notes: DDS-based communication; supports simulation mode; Python/C++ APIs; excellent for robotics and IoT scenarios.

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