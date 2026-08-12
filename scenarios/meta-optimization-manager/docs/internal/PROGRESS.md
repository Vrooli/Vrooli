# Progress — Meta-Optimization Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| 2026-08-12 | codex | done | **Condition axis surfaced end-to-end.** Focus now reads Search Hub's owner-measured condition source derived from live `NOW` Answer cells, ranks condition findings beside coverage gaps, exposes `condition status` / `condition explain-leg`, and the operator console renders condition findings beside the existing focus and gaps views. Focused API, ranking, CLI, UI, strings, and type checks pass; full API validation retains one workspace-artifact trial failure documented in `PROBLEMS.md`/the plan ledger. |

| 2026-08-11 | codex | done | **Trustworthy retrieval and honest readiness plan completed.** Coverage numerator joins now consume typed Search Hub registry, routing reachability, and fresh eval evidence; Answer NOW is emitted only when all three signals hold, with per-cell evidence and method-versioned snapshots. Unavailable owners and unresolved capability gaps remain honest. Focus/list/status agree live; coverage and race tests pass. Comprehensive validation retains only documented pre-existing scenario debt. |

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-24 | claude | done | Documentation-first charter. PRD authored to the canonical template (validates clean, 0 violations) with 4 domains + 8 OTs (P0: readiness/focus/gaps/base-doc; P1: trials/convergence; P2: UI/attested-search). Requirements registry generated (8 target modules, schema + linkage valid). Concept docs written: DOMAINS, ARCHITECTURE, COVERAGE-MODEL (keystone — the 3 space docs reference it), DATA, FLOWS, INTEGRATIONS. DECISIONS + PROBLEMS filled. No domain code yet; first slice = `coverage`. |
| 2026-08-11 | codex | shipped | Added the named `program-runtime` empirical GapSource. Focus reads typed failure-shape, refusal-shape, and unresolved-binding projections through the program-runtime Connect surface with a three-second deadline; recurring evidence becomes ranked empirical gaps, while an unavailable runtime becomes one availability gap and leaves coverage, trials, and agent-manager sources intact. Focus source and independent-degradation tests pass. |
| 2026-06-25 | claude | done | **Trials empirical gate — real implementation** (plan: mom-trials-empirical-gate-real-implementation-validation). Replaced the two load-bearing stubs with working code. **Runner** (`internal/trials/runner.go`) rewritten around agent-manager's REAL primitive — `profile ensure` → `task create` (scope = fixture `target/`) → `run create --run-mode sandboxed` → poll `run get` → `run diff` — through the existing `CommandRunner` seam; parses agent-manager's snake_case API JSON with hand-rolled tolerant structs (verified the run/task/profile handlers use `protoconv` `UseProtoNames=true`; chose this over importing agent-manager's proto types to avoid a cross-scenario dependency + governance ceremony). RunResult now carries EVIDENCE (diff/tokens/duration/changed-files/sandbox-ref), never a verdict. **Evaluator** (NEW `internal/trials/evaluator.go`) — the MoM-owned verdict: deterministic-first (copy fixture `target/` → apply diff via `git apply`/`patch` → run the fixture oracle; exit 0 = pass), negative=correct-abstention, agent-judge fallback (labelled lower-confidence; oracle-less + no judge → honest `VerdictError`). **Fixture substrate** (NEW `trials/fixtures/<family>/` for all 5 families: add-feature/bugfix/comprehend/research/negative) — each a self-contained `target/` + `spec.md` + deterministic `check.sh` (kept OUTSIDE `target/` so the agent can't game it); `FixtureResolver` seam + content-rev hashing for idempotency; authoring contract in `docs/internal/TRIALS-FIXTURES.md`. **Service** wires Fixtures+Evaluator, resolves the fixture per task, reuses a recent identical `(task, model, fixture-rev)` run (idempotency window), evaluates between dispatch and record. **Storage**: added `fixture_rev` column + `LatestRun` (storage-only, no wire change). **CLI**: `trials run` now single-task default; requires `--task`/`--suite`/`--all`. **Docs/deps corrected**: removed `workspace-sandbox` from `service.json` + all trials docs (FLOWS/DECISIONS/DOMAINS/INTEGRATIONS/ARCHITECTURE/DATA/PRD/requirements); regenerated `.vrooli/endpoints.json`; no stale dispatch-verb reference remains. **Tests** (faked-seam, CI never dispatches a live model): Runner sequencing/parse/step-error→VerdictError/poll/abstention/timeout; Evaluator pass/fail/abstention/fallback-selection; Service idempotency (reuse/expiry/error-retry/missing-fixture); fixture corpus + a REAL `git apply`+oracle round-trip proving all 5 committed oracles accept their golden solutions. `go build/vet/test` + `gofumpt` + `golangci-lint` all green (api + cli). **Pending operator steps**: (1) live local-model e2e (`trials run --task <id>` with opencode + a pulled model — host has `llama3.2`/`phi3.5`, not `qwen2.5-coder`; pass `--model` accordingly) capturing a real PASS/FAIL + metrics; (2) regression baseline diff (the pre-change `mom-trials` baseline pin FAILED because test-genie was not running at capture — re-anchor with test-genie up); (3) regenerate `coverage/requirements-sync/latest.json` (generated artifact); (4) requirement statuses left `planned` for the traceability sync to flip. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
