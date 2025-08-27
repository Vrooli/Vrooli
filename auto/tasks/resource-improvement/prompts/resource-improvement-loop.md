## ⚙️ Resource Improvement Loop

### TL;DR — One Iteration = ONE resource fix/improvement or ONE new resource added
1) Select ONE resource (existing or new) using the selection tools (see Tools & References).
2.a) If you chose an existing resource, pick a fix/improvement (diagnose/fix/improve/add) to pursue. Priority order: PRD.md compliance, CLI, status check, installation, start/stop behavior, content management (replacing inject), resource-specific functionality. This is the best order for gradually fixing resources over multiple iterations.
2.b) If you chose a new resource, create the full resource with PRD.md and implementation using best practices.
3) Validate with gates per resource type and PRD requirements; if any gate fails twice, stop and capture diagnostics.
5) Start and leave the resource running (exception: Whisper and agent-s2 may be stopped post-test if explicitly necessary), then append ≤10 lines to `/tmp/vrooli-resource-improvement.md`.

---

### 🎯 Purpose & Context
- Improve AND validate platform resources so scenarios can run reliably end-to-end.
- Add new resources to unlock new scenario opportunities (e.g. home automation, research, engineering)
- Ensure that resource fixes/improvements are applied to the resource's setup code, so that it's repeatable and sets itself up reliably every time.
- Ensure that resources can be meaningfully integrated into Vrooli. For automation resources, that means being able to add and manage the resource's workflows/agents/etc. For storage resources, that means being able to add tables, seed, migrate, and query the database. These are pretty straightforward to build, as it's very clear exactly what gets stored/managed in the resources (.json workflow files, .sql files, etc.). Other resources may have less obvious data files, so you'll have to decide what's best (e.g. .py scripts for things only available as python packages, with guidelines in the resource's docs explaining what packages are available in the environment that will run the file, and other best practices)
- Scenarios depend on resources being up; improvements must keep services running.
- Prefer incremental, reversible changes; prefer interacting with resources using their CLIs over bespoke/manual edits.
- See `docs/context.md` for why resource orchestration is central to Vrooli.

### 📄 PRD Compliance & Progress Tracking
- **Every resource MUST have a PRD.md** based on `scripts/resources/templates/PRD.md`
- Use the PRD as the single source of truth for resource requirements and progress
- Track implementation status directly in the PRD (check off completed items)
- PRD sections that must be validated:
  - Standard interfaces (CLI commands, content management)
  - Port registry integration (no hardcoded ports)
  - Docker image versioning (no 'latest' tags)
  - Test specifications (BATS co-location, shared fixtures)
  - Content management implementation
- When improving a resource, first check its PRD.md for gaps

### 🔄 Content Management Migration (inject → content)
- **IMPORTANT**: The `inject` command is being phased out in favor of `content` management
- `inject` was confusing - it implied command execution but actually stored data
- New `content` command has clear subcommands: add, list, get, remove, execute
- Migration approach:
  1. Keep `inject` working temporarily for backwards compatibility
  2. Implement `content` management alongside
  3. Update documentation to use `content` examples
  4. Eventually deprecate `inject` with clear migration path
- Content management enables sharing data between scenarios (workflows, SQL, scripts, etc.)

---

### ✅ DO / ❌ DON'T
- ✅ Keep changes to existing resources minimal; prefer non-destructive operations.
- ✅ Use `vrooli resource <name> <cmd>` and `resource-<name>` CLIs
- ✅ Redact secrets; use environment variables for credentials.
- ✅ Use timeouts for any potentially hanging operation.
- ✅ Prefer writing resources such that if sudo permissions are required, they only have to be provided once. We should be able to freely stop and start resources without needing permissions.
- ✅ Use the project-level `data/` folder for storing things that don't belong in git, such as credentials, compiled code, logs, etc.
- ✅ Refer to the `PRD.md` file for more specific implementation requirements

- ❌ Do NOT uninstall, disable, or permanently shut down resources. The scenario loop relies on them running.
- ❌ Do NOT modify loop scripts or prompt files.
- ❌ Do NOT create a `resource-<resource_name>` script to call the resource's `cli.sh` file. You tend to do this sometimes when the cli isn't available. However, the resource should use `install-resource-cli.sh` in its installation process, which automatically sets up the cli.

> Policy: After testing any resource, keep it running. Only Whisper and agent-s2 should be stopped post-validation, due to resource constraints for whisper and known net jacking issues for agent-s2. If you run into a network issue (unlikely but possible), run `scripts/lib/network/diagnostics/network_diagnostics.sh` to figure out what's wrong and attempt autofixes.

---

### 🧭 Resource Selection — Recommended Flow
- Tools (Choose based on best judgement, using all of these tool outputs):
  - `auto/tools/selection/resource-candidates.sh` — candidates with status and cooldown
  - `vrooli resource status` — full list of registered resources with their statuses
- Priority rubric:
  1) Enabled but not running → attempt start with diagnostics
  2) Running but missing baseline capability (e.g., Ollama baseline models) → improve
  3) Misconfiguration in connection info → diagnose and plan a safe fix
  4) All resources have healthy status and iteration > 15 (hinting that we've probably been healthy for several steps already, and have likely already done additional validations and improvements to the existing resources) → HIGHLY consider adding a high-impact resource from the curated list later in this prompt
  5) If all resources are healthy and every new resource in the list was added too, then work on improving/cleaning up existing resources

Before choosing, also read (if present):
- `auto/data/resource-improvement/summary.txt` — review overall health metrics first
- `tail -n 20 auto/data/resource-improvement/events.ndjson` — then check recent logs for specific failure patterns

---

### ✅ Validation Gates — Acceptance Criteria
All must be satisfied unless a resource-specific note says otherwise.

General:
- **PRD.md exists and follows template** from `scripts/resources/templates/PRD.md`
- Status report shows a "healthy" state. Or if we don't have an API key for that resource, as healthy as it gets otherwise
- Resource responds to a minimal health/sanity check
- Resource shows as "healthy" not just in its own cli, but also in `vrooli status --verbose`. A discrepancy between the resource status and vrooli's detection of it typically means it hasn't yet auto-registered its cli to vrooli. This should be set up to happen automatically in the resource's code (pretty sure it's a one-liner, as used by many other resources)), but you can also call it yourself if needed.
- No critical errors in condensed logs/output
- **Content management implemented** (`resource-[name] content add/list/get/remove/execute`) for storing and managing resource data (workflows, SQL, scripts, etc.). This replaces the old "inject" pattern.
- Resource's documentation is detailed, accurate, and organized. It should follow our documentation best practices, including using a "hub-and-spokes" model of organization to limit the size of the resource's main README.md.
- Resource does not use a manage.sh script. Instead, the resource is managed through its `cli.sh` script, which acts as an ultra thin wrapper around the library functions.
- The resource has integration tests, which are put in a `test/` folder. These use files from `__test/fixtures/` instead of putting the tests directly in the resources's folder, so that we can reuse them across other resources. 
- Test results are included in the resource's `status` result, with a timestamp for the last time they were run.
- The resource status correctly supports text and json output
- The resource includes at least one example in an `examples/` folder.
- All `.bats` files (if present) are co-located with the file they test, using the same name (e.g. `docker.sh` and `docker.bats`)
- Any code that's labelled as being "legacy", "backwards compatability", or "deprecated" is removed (except temporary inject→content migration).

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
- New resources that don't fit well into one of the existing categories can just go in `resources/`. We'll eventually get rid of the categories format, so it doesn't matter
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

### 🧱 Resource Hierarchy & Dependencies

**Command Pattern (applies to ALL properly installed resources):**
- Status: `resource-{name} status` or `vrooli resource {name} status`
- Lifecycle: `resource-{name} manage {install|start|stop}`
- Testing: `resource-{name} test {smoke|integration|all}`
- Content: `resource-{name} content {add|list|get|remove|execute}`

**Dependency Chain (fix in order):**
```
Vault → Ollama → Qdrant → Browserless → PostgreSQL/Redis/MinIO → QuestDB → Huginn/n8n → AI Routers (OpenRouter/Cloudflare AI Gateway) → the rest
```

### 📦 EXISTING Resources


#### Infrastructure (Always Running)
| Resource | Purpose | Key Dependencies | Critical Signals |
|----------|---------|------------------|------------------|
| **vault** | Secrets management | OS/filesystem | Auth failures (401/403) |
| **postgres** | Primary database | Disk I/O, Vault | Connection refused, missing tables |
| **redis** | Cache/queue | Network | Slow workflows, queue backlogs |
| **minio** | Object storage | Network, Vault | Upload errors, missing artifacts |
| **questdb** | Time-series data | Disk throughput | Ingest lag, missing telemetry |

#### Automation (Core Capability)
| Resource | Purpose | Key Dependencies | Critical Signals |
|----------|---------|------------------|------------------|
| **huginn** | Event automation | PostgreSQL, Redis | Agent failures, webhook misses |
| **n8n** | Visual workflows | PostgreSQL, Redis | Workflow activation failures |
| **node-red** | Flow automation | PostgreSQL | Node errors, flow failures |
| **windmill** | Script workflows | PostgreSQL | Execution failures |
| **browserless** | Headless browser | CPU/memory | Screenshot failures, timeouts |

#### AI/ML (Intelligence Layer)
| Resource | Purpose | Special Notes |
|----------|---------|---------------|
| **ollama** | Local LLMs | Pull baseline models early |
| **openrouter** | AI provider hub | Monitor rate limits |
| **litellm** | LLM proxy | Fallback configuration |
| **cloudflare-ai-gateway** | AI resilience | Caching, retries |
| **whisper** | Speech-to-text | May stop post-test (resource heavy) |
| **comfyui** | Image generation | Model assets on fast storage |
| **musicgen** | Music generation | May stop post-test |

#### Agents & Development
| Resource | Purpose | Special Notes |
|----------|---------|---------------|
| **cline** | VS Code agent | IDE integration |
| **claude-code** | Claude coding | Alternative to Cline |
| **agent-s2** | Browser agent | May cause network issues |
| **opencode** | VS Code alt | Check availability |
| **autogpt** | Autonomous agent | Monitor for loops |
| **codex** | Code generation | Monitor API costs |

#### Data & Search
**Available:** qdrant, neo4j, searxng, unstructured-io, haystack, pandas-ai

#### Business & Productivity  
**Available:** erpnext, odoo, keycloak, wikijs, btcpay

#### Development Tools
**Available:** judge0, k6, geth, simpy, sagemath

#### Media & Creative
**Available:** ffmpeg, obs-studio, blender, openscad, kicad, vocr

#### Infrastructure Extensions
**Available:** postgis, home-assistant, twilio, pushover, mail-in-a-box

---

### 📎 NEW Resources to Add
*Only add new resources when 75%+ of existing ones are healthy and validated*

#### **Matrix Synapse** (Communications)
- **What it does:** Self-hosted Matrix homeserver for federated chat/rooms/bots across scenarios. Enables secure, decentralized communication with E2E encryption, bridging to other platforms (Slack, Discord, IRC), and bot integration for automation.
- **Use when:** Scenarios need real-time communication, team collaboration, notification channels, or chatbot interfaces. Essential for multi-user scenarios and customer support applications.
- **Technical requirements:** 
  - Use PostgreSQL backend (not SQLite) for production
  - Configure well-known delegation for federation
  - Set up TURN server for voice/video calls
  - TLS certificates required for federation
  - Consider Dendrite as lighter alternative for resource-constrained environments
- **Integration points:** Can trigger n8n/Huginn workflows via webhooks, integrate with authentication systems via SSO, bridge to external chat platforms
- **Resource usage:** Medium-heavy; scales with user count and federation traffic

#### **OWASP ZAP** (Security)
- **What it does:** Web application security scanner with automated vulnerability detection, providing security testing automation and compliance checking. Includes active/passive scanning, authentication handling, and API testing capabilities.
- **Use when:** Need automated security testing for generated apps, compliance validation, or continuous security monitoring. Critical for scenarios handling sensitive data or requiring security certifications.
- **Technical requirements:**
  - Java-based; runs in daemon mode for API access
  - REST API for integration with CI/CD pipelines
  - Supports authentication (forms, OAuth, SAML)
  - Can spider and scan single-page applications
  - Configurable scan policies and alert thresholds
- **Integration points:** Output to various formats (JSON, XML, HTML), webhook notifications for findings, integration with bug trackers
- **Best practices:** Start with passive scanning in production, use active scanning in test environments, maintain baseline scans for comparison
- **Resource usage:** CPU-intensive during scans; memory scales with target complexity

#### **ROS2** (Robotics)
- **What it does:** Robot Operating System 2 - middleware for robotics applications enabling robot control, sensor integration, navigation, and distributed computing across robot components.
- **Use when:** Building robotics scenarios, IoT device orchestration, sensor data processing pipelines, or simulation of physical systems. Enables hardware abstraction and modular robot software.
- **Technical requirements:**
  - DDS-based communication (Fast-DDS or CycloneDX)
  - Supports simulation mode with Gazebo
  - Python and C++ client libraries
  - Real-time capable with proper kernel configuration
  - ROS2 domain isolation for multi-robot setups
- **Key packages to include:** 
  - Navigation2 for autonomous navigation
  - MoveIt2 for manipulation planning
  - ros2_control for hardware interfaces
  - Lifecycle nodes for deterministic startup/shutdown
- **Integration points:** Can publish/subscribe to MQTT, integrate with computer vision (OpenCV), machine learning inference, and cloud services
- **Resource usage:** Varies greatly; simulation is CPU/GPU intensive, real robot control can be lightweight

### 💡 Selection Priority
1. Fix non-running enabled resources
2. Add baseline content (Ollama models, n8n workflows)
3. Fix misconfigured connections
4. Add new resources only when 75%+ healthy

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
- **Content Management** (replaces inject):
  - `resource-n8n content add --file workflow.json`
  - `resource-postgres content add --file schema.sql`
  - `resource-<name> content list`
  - `resource-<name> content execute --name my-workflow`
- **Sanity checks**:
  - Prefer resource CLIs or shared workflows (avoid direct client binaries unless explicitly allowed) 