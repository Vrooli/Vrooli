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
- Status reports “running” (or equivalent healthy state)
- Resource responds to a minimal health/sanity check
- No critical errors in condensed logs/output

Per resource (examples using resource CLIs):
- Postgres: `vrooli resource postgres status` (or `resource-postgres status`) reports healthy
- Redis: `vrooli resource redis status` (or `resource-redis status`) reports healthy
- Ollama: `resource-ollama status` succeeds; `resource-ollama list-models` shows at least one baseline model; pulling a baseline model is allowed if missing
- Browserless: `vrooli resource browserless status` reports healthy; a trivial navigation/screenshot via the resource CLI or shared workflow succeeds
- n8n: `vrooli resource n8n status` reports healthy; workflows listable via the resource CLI
- Whisper: can start and respond to a minimal test via resource CLI; may be stopped post-test if resource constraints demand (document decision)

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

---

### 🔒 Security & Safety
- Redact secrets; use env vars
- Avoid printing configs/credentials
- Do not modify platform resources beyond allowed operations
- Respect timeouts (e.g., `timeout 30`)

---

### 📝 Notes Discipline (≤10 lines)
Append one compact line per iteration to `/tmp/vrooli-resource-improvement.md`:
- `iteration | resource | action | rationale | commands | result | issues | next | running`
Keep file under 1000 lines; prune periodically.

---

### 📎 Appendix — Command Patterns
- Status/health:
  - `vrooli resource status`
  - `vrooli resource <name> status`
  - `resource-<name> status`
- Start/Stop:
  - `vrooli resource start <name>`
  - `vrooli resource stop <name>` (avoid stopping; Whisper post-test only, if needed)
- Install/Improve:
  - `vrooli resource install <name>`
  - `resource-ollama list-models`
  - `resource-ollama pull-model llama3.2:3b`
- Sanity checks:
  - Prefer resource CLIs or shared workflows (avoid direct client binaries unless explicitly allowed) 