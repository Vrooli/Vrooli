# Problems — Development Toolchain Validator

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-05-18 — agent-manager integration is stubbed

**Symptom:** `development-toolchain-validator validation start --skill ...`
queues a run that immediately terminates with `verdict=run_failure` and
`error_message="agent-manager unavailable: task/profile wiring not yet
implemented; see PROBLEMS.md"`.

**Root cause:** `agent-manager`'s Connect surface requires a pre-created
`Task` (via `CreateTask`) plus an `AgentProfile.id` before `CreateRun`
can be invoked. DTV does not yet manage those primitives. The
`integrations/agent_manager.Client` adapter is currently a stub that
returns `vrun.ErrDependencyUnavailable` so the worker fails visibly
rather than fabricating success. Tests exercise the worker via
`internal/validation_run/mocks.FakeAgentManager`.

**Workaround:** Use the CLI surface in dry-run mode for development
(create manifests, list goldens, exercise tool-only validation runs).
Skill-tuple runs always fail until the wiring lands.

**Real fix:** Implement the full task+profile provisioning flow in
`integrations/agent_manager/`:
- resolve or create a `Task` keyed on `(skill_id, golden_slug)`
- resolve or create an `AgentProfile` for the run (sandboxed, scoped to
  the golden path)
- call `CreateRun(task_id=..., agent_profile_id=..., run_mode=AUTONOMOUS, ...)`
- poll `GetRun` for terminal status; parse `RunSummary` (diff hash,
  diff path list, token/cost metrics) into `vrun.RunSummary`.

**Owner:** unassigned.

**Refs:**
- `path:scenarios/development-toolchain-validator/api/integrations/agent_manager/client.go`
- `path:packages/proto/schemas/agent-manager/v1/api/service.proto`
- DTV P0 plan §11 risk register (this risk was flagged at plan
  authoring time).

### 2026-05-18 — workspace-sandbox content fetch is stubbed

**Symptom:** Manifest content-rules (`must_contain` / `must_not_contain`)
that depend on file bodies are not enforced; the evaluator only sees
the diff path list.

**Root cause:** `integrations/workspace_sandbox.Client.FetchPathContent`
returns an error pending proper Connect wiring to workspace-sandbox's
content-read RPC. workspace-sandbox is an *optional* dependency in
`.vrooli/service.json`; the worker's evaluator gracefully degrades to
path-only verdicts.

**Workaround:** Manifests authored against the v0 DSL may rely only on
`allowed_paths` / `wildcard_allowed`. Content rules are accepted at
upsert time but skipped at evaluation time until this lands.

**Real fix:** Implement the Connect client against workspace-sandbox's
content-read RPC. Compose into `validation_run.evaluator.Evaluate` so
manifests with content rules trigger a per-path fetch before the
verdict.

**Owner:** unassigned.

**Refs:** `path:scenarios/development-toolchain-validator/api/integrations/workspace_sandbox/client.go`

### 2026-05-18 — prompt-manager skill catalog is REST today

**Symptom:** The `skill_catalog` domain talks to prompt-manager via
REST (`/api/v1/skills/sync`) instead of Connect-RPC like every other
DTV outbound.

**Root cause:** prompt-manager does not yet expose proto schemas. The
adapter lives behind the `SkillCatalogSource` seam declared in
`internal/skill_catalog/`, so the migration to Connect is a local
swap of the adapter without touching the consumer.

**Real fix:** When prompt-manager publishes proto + Connect handlers,
swap `integrations/prompt_manager/skill_catalog_rest.go` for a Connect
client that uses `discovery.ResolveScenarioURLDefault(ctx,
"prompt-manager")`. Remove this entry when complete.

**Owner:** unassigned.

**Refs:** `path:scenarios/development-toolchain-validator/api/integrations/prompt_manager/skill_catalog_rest.go`

### 2026-05-18 — tool expectations parsing is structural-only

**Symptom:** `integrations/dev_tools.Runner.Invoke` shells out to the
named tool binary and treats exit code as the only signal. There is
no per-tool expectation file driving structured pass/fail (e.g.,
`scenario-auditor`'s JSON output is not parsed).

**Root cause:** Per-tool expectation parsing was carved out of P0 to
keep the surface stable. The plan §7 Phase 4 mentions a future
`data/tool-expectations/<tool>.json` parser; nothing reads it today.

**Real fix:** Add `integrations/dev_tools/expectations.go` reading
`data/tool-expectations/*.json` and parsing structured tool output
into a richer `vrun.ToolResult` (e.g., counts, named violations).
Compose into the evaluator alongside the structural pass/fail.

**Owner:** unassigned.

**Refs:** `path:scenarios/development-toolchain-validator/api/integrations/dev_tools/runner.go`

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
