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

### 2026-05-18 — agent-manager reconcile failure does not block boot

**Symptom:** If agent-manager is unreachable at DTV startup, the API
boots normally and only fails at the first skill-validation spawn
(`verdict=run_failure`, error names the missing profile).

**Root cause:** `main.go` logs the reconcile error and continues so DTV
can still serve manifest/golden/report/staleness/tool-validation
surfaces — none of which depend on agent-manager. A hard fail would
take down 80% of the API for a degraded skill-validation path.

**Workaround:** Operator must check the startup log line
`agent-manager profile reconciliation failed: …`. If present, restart
DTV after agent-manager is healthy, or accept that skill-tuple runs
will fail until the next API restart picks up the reconciled profile.

**Real fix:** Add a background reconcile loop (e.g., retry every 60s
until success) so the operator does not need to restart DTV. Track the
reconcile state via `/health` so a degraded-but-up state is visible to
upstream monitors.

**Owner:** unassigned.

**Refs:** `path:scenarios/development-toolchain-validator/api/main.go` (Initialize block)

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

### 2026-05-18 — tool expectations parsing is structural-only — RESOLVED 2026-05-29

**Symptom (was):** `integrations/dev_tools.Runner.Invoke` shelled out to the
named tool binary and treated exit code as the only signal. There was
no per-tool expectation file driving structured pass/fail.

**Resolution:** The runner was rewritten into a generic, typed tool
registry (`integrations/dev_tools/registry.go`) seeded with `test-genie`
(expectation: every catalog phase passes, via `execute --preset
comprehensive --json`) and `scenario-completeness-scoring` (expectation:
`score >= floor`, default 96). Per-tool expectations load from
`data/tool-expectations/<tool>.json` (`integrations/dev_tools/expectations.go`).
The runner captures stdout/stderr separately and produces a richer
`vrun.ToolResult{Ran, ExpectationMet, Detail, RawOutput}`; the evaluator
maps the two-layer signal to verdicts (couldn't-run → `run_failure`,
ran-but-expectation-missed → `tool_failure`, ran+met → `pass`). Raw
output + detail are persisted on the validation record and surfaced via
the report read path. `scenario-auditor` was removed from the registry
(it is covered inside test-genie's standards/architecture phases).

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
