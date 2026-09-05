# Problems — AI Gateway

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

## Work ladder

- Rung: W0
- Evidence: the exact `swarm-manager goals list --json` name/title/description filter returned no goal naming `ai-gateway`; the active user-owned Plan Manager execution `credential-blast-radius-make-minted-secrets-un-losable` explicitly requires migrating its OpenRouter credential read to the shared fail-closed authority seam.
- Blocker: independent goal-to-PRD reconciliation is unavailable through the swarm-manager gate; continue only under the explicit Plan Manager objective and do not claim the scenario contract is independently reconciled.
- Measured: 2026-08-27

### 2026-07-06 — Security Health Hardening Follow-up

**Symptom:** Phase 8 security validation initially reported findings in AI Gateway command execution, path handling, inventory error conversion, scanner file reads, and shared/toolchain surfaces. Focused product security now reports zero findings, while comprehensive Test Genie still includes advisory Go toolchain and pnpm audit observations.

**Root cause:** Product-code command/path findings came from normal scaffold expansion before the scenario's real resource-command boundaries and scanner rules were hardened.

**Workaround:** No AI Gateway product-code workaround is currently required. Treat comprehensive security observations that point at Go standard-library or JS toolchain advisories as dependency/toolchain governance follow-up.

**Real fix:** Keep future provider/resource additions behind the existing command-runner seams, rerun focused security health before marking new slices complete, and address Go/pnpm advisories through normal dependency governance.

**Owner:** ai-gateway maintainers.

**Refs:** `api/internal/providers/runner.go`, `api/internal/conformance/scanner.go`, `api/handlers/inventory/connect_handler.go`, `api/main.go`, `docs/internal/SECURITY.md`.

### 2026-07-06 — Performance Example Flow Produces Workflow Selector Warnings

**Symptom:** `test-genie execute ai-gateway workflow --wait --json` passes but reports advisory `workflow.selector_unregistered` warnings for `bas/flows/perf-example-scroll.json`.

**Root cause:** The performance example intentionally uses literal `[data-testid=...]` selectors because the performance capture path documents literal selector usage. Workflow Health still audits `bas/flows/**` and reports the literal selector as an advisory.

**Workaround:** Do not bind the example flow as functional validation evidence. The real Phase 7 operator workflows under `bas/cases/operator-ui/` pass and use `@selector` references.

**Real fix:** Either teach Workflow Health to treat `metadata.labels.intent = "performance"` flows with the performance-health selector rule, or replace the example with a product-specific performance flow once performance budgets are introduced.

**Owner:** workflow-health/performance-health follow-up.

**Refs:** `bas/flows/perf-example-scroll.json`, `bas/cases/operator-ui/dashboard-readiness.json`, `bas/cases/operator-ui/route-preview-workflow.json`.

### 2026-07-06 — Route Evidence Measures Waiver (RESOLVED)

**Status:** Resolved. The temporary `routing` measures waiver has been removed.

**What shipped:** Route evidence is now a descriptor-backed measures surface. `packages/proto/schemas/ai-gateway/v1/measures/measures.proto` declares a `MeasuresService` with seven read-only, run-eligible route measures (`route_events.total`, `success_rate`, `fallback_rate`, `failure_rate`, `breaker_open`, `capacity_rejections`, `latency_p95`). One shared compute path (`api/handlers/measures`) backs both the typed Connect RPCs and the packages/measures-go serve registry mounted at `/measures`, computed as real SQL aggregates over `route_events`. `cli/manifest.json` declares each measure bound to the `route_events` domain; the `measures.omitted[routing]` waiver is gone and `measures.domains[]` marks `route_events` stateful.

**Evidence:** `measures-health validate scenario ai-gateway --probe` passes (Domain Coverage L3, Declaration Assembly L3, Behavioral Indexing L2, 0 errors). The remaining `measures.architecture-fallback` info is a standing advisory (no `v1/domain/` folder), not introduced by this work.

**Refs:** `cli/manifest.json`, `api/handlers/measures/`, `api/internal/routing/measures_repository.go`, `packages/proto/schemas/ai-gateway/v1/measures/measures.proto`.

### 2026-07-06 — Tidiness Still Reports Info-level Hardening Debt

**Symptom:** The Test Genie `tidiness` phase passes, but still reports info-level duplication and complexity findings across scaffold tests, generated-style CLI domain handlers, and shared test utilities.

**Root cause:** The hard Phase 8 blocker was fixed by shrinking duplicated manifest/no-production-import helpers. The remaining findings are lower-severity scaffold and test-support hardening debt.

**Workaround:** Treat these as non-blocking while `test-genie execute ai-gateway tidiness --wait --json` passes and product validation remains clean.

**Real fix:** Run a dedicated tidiness cleanup campaign after the scenario API/CLI surfaces stabilize; avoid mixing those refactors with feature phases.

**Owner:** ai-gateway/platform tidiness follow-up.

**Refs:** `api/internal/testutil/no_prod_import_test.go`, `coverage/logs/*/tidiness.log`.

### 2026-08-16 — `routing execute` has no `--temperature` flag

**Symptom:** `RoutingService.ExecuteRoute` honours `request.sampling.temperature`
and persists it to route evidence, but the CLI exposes no flag for it. Only API
and program-runtime callers can set sampling on the routing path.

**Root cause:** The manifest's argument resolver
(`cliapp.ResolveArgField`) matches a flag name against the request message and
then auto-descends exactly **one** envelope level. `ExecuteRouteRequest` wraps
`GatewayRequest`, so `request.sampling.temperature` sits two levels down and no
flag name reaches it. `bind.field` cannot help: its schema pattern
(`^[a-z][a-z0-9_]*$`) forbids dotted paths, and the bind lookup runs against the
top-level request message only. Declaring the flag anyway produces a hard
`binding.arg_unmapped` error in the `contracts` phase. `inference run` is
unaffected because `RunRequest` carries `sampling` directly, one level down.

**Workaround:** Use `ai-gateway inference run --temperature` for typed inference,
or call `ExecuteRoute` through the API for the routing path.

**Real fix:** Either teach the resolver to descend a declared dotted path, or —
if a caller actually needs the flag — add a JSON-valued `--sampling` flag whose
name resolves to `request.sampling` under a stated `bind_waiver`. Neither is
worth doing before a caller asks.

**Owner:** unassigned.

**Refs:** `packages/cli-core/cliapp/protobindings.go:157` (`ResolveArgField`),
`scenarios/ai-gateway/cli/domains/internal/gatewayreq/request.go`,
`.vrooli/schemas/cli-manifest.schema.json` (`$defs.Flag.bind.field`).

### 2026-08-16 — Sampling declarations need a resource-CLI rebuild to take effect

**Symptom:** After `sampling_support` and the ollama role `max_tokens` were added
to resource policy, `resource-ollama policy resolve --role write.default --json`
still omits both fields, so the gateway resolves every ollama role as
`unknown` support and an uncapped role.

**Root cause:** The installed `~/.vrooli/bin/resource-ollama` predates the struct
change and drops fields it does not know. It cannot currently be rebuilt:
`go build ./...` in `resources/ollama/cli` fails with `updates to go.mod needed`
(`golang.org/x/sys` v0.44.0 → v0.47.0), and `vrooli resource install ollama`
fails on the same build. The governed remedy does not reach here either —
`scenario-dependency-analyzer deps install` resolves surfaces under
`scenarios/` only and rejects a `resources/` path.

**Workaround:** None needed for correctness. The role-declared sampling path
deliberately sends its value on anything other than `rejected`, so an undeclared
role still receives the role's temperature and deterministic roles stay
deterministic. Only the reported `temperature_support` is less precise than it
should be, and the ollama role cap is not yet applied.

**Real fix:** Extend the dependency-analyzer surface vocabulary to cover
`resources/<name>/cli`, then bump the drifted module and reinstall.

**Owner:** unassigned (dependency governance).

**Refs:** `resources/ollama/cli/go.mod`, `resources/openrouter/cli/go.mod`,
`resources/ollama/cli/internal/policy/policy.go`.

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
