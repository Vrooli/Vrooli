# Problems — TypeScript Code Graph

Persistent register of known issues, tech debt, and deferred work specific to **this** scenario. Future agents read this file to avoid re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from the code

## What does NOT belong here

- **Generic template issues** — those go in [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

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

### 2026-05-23 — Deferred: persistent Operation Log (REQ-P1-002)

**Symptom:** `RewritePlan` results live in an in-memory `MemoryPlanStore`; restarting the API forgets every plan that hasn't been applied. Callers that planned just before a restart see `FailedPrecondition` on `RewriteApply`.

**Root cause:** P1 scope only — SQLite-backed persistence is intentionally not implemented for v1.

**Workaround:** Consumers re-plan immediately before applying. Plan derivation is deterministic (PlanID is content-hashed), so re-planning yields the same `plan_id` as long as the operation list and scenario state are unchanged.

**Real fix:** Add a `SQLitePlanStore` implementation behind the existing `rewrite.PlanStore` seam; wire it in `main.go` instead of `NewMemoryPlanStore()`. The seam is already in place; this is purely additive.

**Owner:** Unassigned (P1 backlog).

**Refs:** REQ-P1-002. Seam: `docs/internal/SEAMS.md` "PlanStore".

### 2026-05-23 — Deferred: Fixture Validator UI (REQ-P1-001)

**Symptom:** No UI surface for browsing or validating extraction fixtures (`bas/fixtures/ts-junk-drawer`, `ts-jsdoc-tags`, `ts-usage-facts`). Fixture drift is caught only by `internal/graph/integration_test.go` byte-for-byte comparison.

**Root cause:** Out of scope for this plan. The debug graph explorer covers per-scenario extraction; a dedicated fixture-validator surface is P1.

**Workaround:** Re-run `go test ./internal/graph/... -run TestIntegration -race` and read the diff on failure.

**Real fix:** Add a UI route under `ui/src/features/fixtures/` consuming `Extract` + a fixture-diff helper. Out of scope here.

**Owner:** Unassigned (P1 backlog).

**Refs:** REQ-P1-001.

### 2026-05-23 — Deferred: react-component-library migration cutover (REQ-P1-006)

**Symptom:** `react-component-library` still uses regex JSDoc-tag scraping in production. It has not yet adopted typescript-code-graph's `leading_comments[]` field on declaration nodes.

**Root cause:** Migration is coordinated separately from this plan. The cutover blocks on `leading_comments` being stable across releases — which it now is.

**Workaround:** rcl continues to ship its regex parser. typescript-code-graph's `ts-jsdoc-tags` fixture (`bas/fixtures/ts-jsdoc-tags/expected-graph.json`) is the **contract anchor**: any change to leading-comment fidelity must update that fixture and is reviewed as a breaking change.

**Real fix:** rcl's own migration plan, tracked in its scenario. Verified prerequisite from this side: `leading_comments` round-trips verbatim (whitespace + trailing newlines) — covered by `internal/graph/leading_comments_test.go` and pinned by the fixture.

**Owner:** react-component-library maintainer.

**Refs:** REQ-P1-006. Contract anchor: `bas/fixtures/ts-jsdoc-tags/expected-graph.json`.

### 2026-05-23 — Deferred: in-process Go-native TS parser (REQ-P2-006)

**Symptom:** The Node sidecar is the only TypeScript parser implementation. Spawning a Node child adds startup cost (~hundreds of ms cold), occupies a process slot, and introduces a non-Go runtime in the deployment surface.

**Root cause:** No Go-native TypeScript parser with `ts-morph` parity currently exists. Building one is a P2 research item, not a P0 deliverable.

**Workaround:** None. The sidecar is the production path.

**Real fix:** When (and only when) a Go-native parser reaches `ts-morph` parity on the fixture suite, add a second `SidecarClient` implementation. **Important:** the `SidecarClient` seam exists for tests, NOT as a strategy/plugin abstraction over hypothetical alternative parsers. Adding a second production implementation requires re-justifying the seam shape (it currently leaks IPC-flavoured concerns like `Heartbeat`).

**Owner:** Unassigned (P2 research).

**Refs:** REQ-P2-006. Architecture deviation: `docs/concepts/ARCHITECTURE.md` "Intentional Deviations" 2026-05-23 Node sidecar entry.

### 2026-05-24 — Deferred: cross-scenario CI smoke test (E1)

**Symptom:** No automated coverage of the live cross-scenario Connect hop (cartographer's `tscodegraph.Client` → typescript-code-graph API → Node sidecar). The in-process integration tests in `internal/graph/integration_test.go` prove the IPC chain and the Connect wire path within this module, but nothing exercises cartographer's real client wrapper end-to-end, so a regression that stubbed `architecture-cartographer/.../tscodegraph/client.go` would not be caught by CI.

**Root cause:** The two scenarios are separate Go modules that intentionally talk only over Connect/HTTP (no cross-module import), and CI runs targeted Go tests rather than booting scenarios. A faithful test therefore needs either a `tscg`-binary exec harness (cartographer test execs the SQLite-backed tscg API + node sidecar on a random port and points its real client at it) or a full scripted scenario boot — both are new CI machinery.

**Workaround:** Manual operator validation (parent plan §10 #10–14): restart both scenarios and run `architecture-cartographer graph extract <ts scenario>` to confirm a non-empty real graph.

**Real fix:** Add the binary-exec smoke test (preferred: lives in cartographer's module, uses the real client, no Postgres needed) plus a CI step that pre-builds the tscg API binary + sidecar dist. Must be proven to fail when the client is reverted to a stub.

**Owner:** Unassigned. Deferred 2026-05-24 by maintainer decision (correctness fixes D1–D5 landed; E1 follows separately).

**Refs:** `~/.vrooli/plans/typescript-code-graph-hardening-validation-gates-loose-ends-and-enhancements.md` Phase E1; DoD #7.

### 2026-05-24 — Deferred: sidecar hardening enhancements E2–E6

**Symptom:** Six operability enhancements from the hardening plan are scoped but not yet implemented:
- **E2 — ts-morph upgrade workflow:** no `make update-ts-morph` target to bump the pin, rebuild `dist/`, regenerate fixtures, and print the diff for review. The next ts-morph bump risks silent fixture drift.
- **E3 — performance benchmark gate (MOD-P0-012 / OT-P0-012):** no `bench`-tagged Go benchmarks asserting the synthetic 200-file (<5s) / 2000-file (<30s) budgets.
- **E4 — structured sidecar logging:** the supervisor's stderr pump (`supervisor.go`, the `io.Copy(s.cfg.StderrSink, stderr)` goroutine) routes raw bytes to a sink instead of the API's structured logger with `request_id` correlation. (Invariant to preserve: the sidecar must never write non-framed data to stdout.)
- **E5 — cancel acknowledgment IPC:** cancel is fire-and-forget; the sidecar does not emit a `{type:"cancel_ack", request_id}` reply, so the supervisor cannot distinguish "work discarded" from "result arrived just before cancel". Additive — bump `ProtocolVersion` only if the handshake contract changes.
- **E6 — PlanStore TTL eviction:** `rewrite.MemoryPlanStore` never evicts; a long-running API grows unbounded. A ~1h TTL is an independent stopgap, superseded by the future SQLite store (REQ-P1-002).

**Root cause:** Scoped as post-merge-optional in the hardening plan; only E1 was gating, and E1 was itself deferred.

**Workaround:** None needed — these are operability/regression-safety improvements, not correctness bugs.

**Real fix:** Implement per the plan's Phase E (priority order E2→E6).

**Owner:** Unassigned. Deferred 2026-05-24.

**Refs:** `~/.vrooli/plans/typescript-code-graph-hardening-validation-gates-loose-ends-and-enhancements.md` Phase E.

### 2026-05-23 — Implementation has not started (historical — superseded)

**Symptom (historical):** No `graph`, `rewrite`, or `sidecar` domain code exists.

**Resolution:** Resolved by Phases 1-8 of `typescript-code-graph-proto-api-cli-implementation-with-node-sidecar.md`. All three domains shipped; `notes` removed; cartographer's `tscodegraph/client.go` upgraded from stub to real Connect client. Phase 9 (docs + lifecycle) closes the plan. Kept as a historical marker; remove on next janitorial pass.

### 2026-05-23 — Template `notes` test fails linter on generation

**Symptom:** During `vrooli scenario generate react-vite --run-hooks`, the `golangci-lint --fix` post-hook reports `handlers/notes/module_test.go: cannot use d (variable of type *sql.DB) as *RoutedDB` errors.

**Root cause:** Template-level type mismatch in the `notes` example domain. This is a template defect, not a scenario-level issue.

**Workaround:** The lint failure is non-fatal to generation. The scenario itself is correctly created. The `notes` example will be removed in Gate 7.

**Real fix:** File a template-level fix against the `react-vite` template. Not in scope for typescript-code-graph implementation.

**Owner:** react-vite template maintainer.

**Refs:** `templates/scenarios/react-vite/`. Same issue surfaces in `go-code-graph` (sibling scenario generated minutes earlier).

### 2026-05-23 — Node sidecar bootstrap (historical — resolved)

**Symptom (historical):** No `sidecar/` directory at the scenario root.

**Resolution:** Resolved by Phase 2 of the implementation plan. `sidecar/{package.json, tsconfig.json, src/, tests/, scripts/build.mjs}` ship complete; `dist/index.js` is gated by `.vrooli/service.json` lifecycle setup. Kept as a historical marker; remove on next janitorial pass.

## Architecture Drift

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _(none currently tracked)_ | — | — | — |

Previous entries (`notes` example domain present, `sidecar/` absent) were resolved by Phases 1 and 2 of the implementation plan; see historical entries above.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
