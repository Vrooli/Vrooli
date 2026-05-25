# Problems — Architecture Cartographer

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

### 2026-05-21 — go-code-graph does not exist yet (initialized 2026-05-23, not implemented)

**Symptom:** Cartographer's Go-side `graph` extraction cannot run end-to-end because the `go-code-graph` scenario it depends on for Go source parsing is initialized (PRD, requirements, docs in place at `scenarios/go-code-graph/` as of 2026-05-23) but no domain implementation has shipped. Cartographer's `graph` and `apply` domains still cannot call real `Extract` / `Rewrite` endpoints for Go.

**Root cause:** Layered scenario architecture (see [`DECISIONS.md`](DECISIONS.md), entry 2026-05-21) requires graph extraction to live in language-specific scenarios. The 2026-05-23 initialization session generated `go-code-graph` from the `react-vite` template, authored a full PRD (10 P0 + 5 P1 + 5 P2), generated and validated requirements (15 modules `healthy`), and filled the docs surface — but did not implement the `graph` or `rewrite` domains.

**Workaround:** Cartographer's `e2e_lang_graph_test.go` remains build-tagged off (`//go:build e2e_lang_graph`). Cartographer's `bas/fixtures/go-cycles/expected-graph.json` is hand-curated and stands in for what `go-code-graph` will eventually return.

**Real fix:** Implement the P0 operational targets in `go-code-graph` per its launch sequencing: deterministic `Extract` against `golang.org/x/tools/go/packages`, two-step `Rewrite`, fixture determinism gate. Once `go-code-graph`'s `Extract` is shipping, unstub `e2e_lang_graph_test.go`. The Go-source fixture for `bas/fixtures/go-cycles/` belongs in `scenarios/go-code-graph/bas/fixtures/go-cycles/` per the fixture-split decision; cartographer keeps `expected-conflicts.json` and references the language-scenario's `expected-graph.json` (or keeps a copy as integration-test input).

**Owner:** Next implementation agent. Plan is to drive each scenario in its own chat per the user's preference.

**Refs:** [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — Intentional Deviations; [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — Scenario Dependencies; PRD.md — Launch sequencing step 1; `scenarios/go-code-graph/` PRD and PROBLEMS.md.

### 2026-05-21 — `vrooli scenario requirements validate` and `lint-prd` are broken by an unrelated test-genie build error

**Symptom:** Running `vrooli scenario requirements validate architecture-cartographer` or `vrooli scenario requirements lint-prd architecture-cartographer --json` returns `go build failed in scenarios/test-genie/cli: exit status 1`.

**Root cause:** Unknown — test-genie CLI build is failing in the user's environment, blocking the wrapping `vrooli scenario` commands that shell out to it.

**Workaround:** Use `prd-control-tower prd validate architecture-cartographer --json` and `prd-control-tower requirements validate architecture-cartographer --json` directly — both ran clean during scenario initialization on 2026-05-21.

**Real fix:** Diagnose the test-genie CLI build failure (likely `go mod tidy` needed, per the error message) outside the architecture-cartographer scope.

**Owner:** Unassigned — global tooling issue, not cartographer-specific.

**Refs:** Initialization session 2026-05-21.

### 2026-05-24 — Cross-scenario code-graph integration wired end-to-end (resolved)

**Symptom (historical):** `architecture-cartographer graph extract <scenario>` returned `unavailable: integration no_adapter_registered`. The TypeScript/Go code-graph adapters never engaged even when the producers were running. The Phase-8 integration shipped correct, unit-tested adapter *code* but was never wired for a live run.

**Resolution (2026-05-24):** Four gaps closed:
1. **Per-call discovery.** Both adapters now take an `api-core/discovery` resolver and resolve the producer URL on every call (`vrooli scenario port <slug>`), so a producer restarting on a new port is handled transparently. `main.go` always registers both; the old `*_CODE_GRAPH_URL` env gating is gone. Tests inject `discovery.NewStaticResolver`.
2. **Name → absolute path.** A new `internal/graph/scenariopath` resolver maps the target scenario *name* → `<repoRoot>/scenarios/<name>` (repo root via `repo-contract-go`) and probes language-appropriate subdirs.
3. **"Which directory" convention.** Path-first, marker-verified: TypeScript probes `ui/` then root for `tsconfig.json`; Go probes `api/`, `cli/`, then root for `go.mod`. Configurable via `CARTOGRAPHER_TS_PROJECT_DIRS` / `CARTOGRAPHER_GO_PROJECT_DIRS`. A scenario with no project of a language contributes an empty graph instead of erroring.
4. **Graceful compose.** `internal/graph/service.go` skips an adapter whose producer is `scenario_unreachable` instead of aborting the whole extract; other errors still propagate.

**Verified:** `architecture-cartographer graph extract architecture-cartographer` → 341 files / 815 packages / 815 symbols / 1005 imports, with both `LANGUAGE_GO` and `LANGUAGE_TYPESCRIPT` present (full chain through the real `tscodegraph` client, not the stub). The TS hop was also confirmed against `web-console` (reached tscg, which correctly reported `workspace_unsupported`).

**Remaining producer-side caveats (not cartographer bugs):**
- **tscg workspaces:** scenarios whose `ui/` contains a `pnpm-workspace.yaml` (e.g. `web-console`) return `unimplemented: workspace_unsupported` — a documented typescript-code-graph v1 limitation (REQ-P2 pnpm-workspace support).
- **go-code-graph packages loader:** `graph extract <scenario>` can return `internal: packages loader` from go-code-graph when the *target* scenario's Go module dependencies aren't present in the local module cache (observed for `chart-generator`). This is a go-code-graph operational/precondition issue, tracked on its side; it does not affect the TS hop (the service still aborts on it because it is `internal`, not `scenario_unreachable`).

**Refs:** `api/main.go` (resolver + project candidates); `internal/graph/scenariopath/`; `internal/graph/{tscodegraph,gocodegraph}/client.go`; `internal/graph/service.go` graceful skip; parent plan §10 #13.

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
