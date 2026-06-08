# Problems — Search Hub

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

### 2026-06-08 — Provider descriptors retired from the binary; providers self-register

**Symptom:** (by design, not a bug) search-hub no longer ships any provider's
`ProviderDescriptor`. Until now, every live/gap provider descriptor was a
`//go:embed`'d seed in `api/internal/providers/seeds/*.json`, surfaced via
`providers.Seeds()`/`SeedIDs()`/`CLIHealthCommands()`.

**Root cause / change:** Search Self-Tuning System plan Phase 2 makes provider
registration **scenario-owned**. A scenario reads its `.vrooli/search.json` SSOT
and calls `RegistryService.RegisterProvider` at boot via the shared
`packages/searchregister-go` helper (descriptor mapping + discovery + bounded
retry + graceful degrade). The embedded-seed path (`providers/seeds.go`, the
`seeds/` dir, `seeds_test.go`) is **deleted** — greenfield, no fallback.

**What moved, not deleted:** the 19 descriptor fixtures were relocated to
`api/internal/routing/testdata/provider_corpus/` because the Ollama-gated
classifier **routing-recall gate** (`routing/classifier_recall_test.go`) still
needs a representative multi-provider landscape to route across. They are now
**test data only** (loaded via `os.ReadFile`, never embedded/registered);
`routing/provider_corpus_test.go` (always-on) guards their validity and the
live-vs-gap split — inheriting the assertions from the deleted `seeds_test.go`.
See `provider_corpus/README.md`.

**Side effect:** `eval/seeds_test.go` previously cross-checked each eval suite's
`provider_id` against the shipped provider catalog. With a dynamic, self-registered
provider set there is no static catalog to check against, so that assertion was
dropped (the suite's provider is resolved at **run** time; an unknown id fails the
run loudly). The suite-validity assertions remain.

**Note on the §10 grep:** the plan's `rg "go:embed seeds" scenarios/search-hub`
check still matches `eval/seeds.go` — that is the **eval-suite** embed (search-hub's
own golden suites, registered at boot via `eval.RegisterSeeds`), a separate
in-design mechanism the plan keeps. Only the **provider** seed embed was retired.

**Deferred to Phase 3+ (proto):** the registration payload currently carries only
the descriptor. Persisting the richer `tuning` + `tests` blocks and minting/returning
the per-provider **control token** require the `registry.proto` deltas scheduled for
Phase 3; until then `RegisterProvider` behaves exactly as before for the descriptor.

**Refs:** `packages/searchregister-go/`, `api/internal/routing/testdata/provider_corpus/`,
`scenarios/{cli-health,knowledge-observatory}/api` boot paths; Search Self-Tuning
System plan §7 Phase 2.

### Search Self-Tuning System — Phase 3 (proto deltas) DONE

The contract surface for the secured override/reindex/config-write loop is now in
the protos (additive-only; `buf breaking` exit 0 vs a pre-change baseline image):

- **`registry.proto`** — new `Tuning` + `FloorConfig` messages (wire carrier for the
  `search.json` `tuning` block; the *authoritative* factor taxonomy stays in
  `packages/aisearch-go`, this only transports values, each factor tagged
  INDEX-TIME vs QUERY-TIME). `ProviderDescriptor` gains `reindex_endpoint (13)`,
  `config_endpoint (14)`, `tuning (15)`. `RegisterProvider` request/response gain
  `control_token` (minted on first registration, echoed thereafter, presented as
  ownership proof on re-registration).
- **`routing.proto`** — `QueryRequest` gains `overrides (7)` + `control_token (8)`;
  new `SearchOverrides` message (query-time factors only, all `optional` for proto3
  presence). Index-time factors are excluded by construction (they need a reindex).
- **NEW `control.proto`** (`vrooli.search_hub.v1.control.SearchControlService`) — the
  ONE shared, token-gated control plane: `Reindex`/`ReindexStatus`/`ReindexCancel`
  (generalized from cli-health's private `cli-health/v1/reindex`) + `WriteConfig`
  (sweep write-back; reindexes automatically when an index-time factor changed).
  Every RPC carries `control_token`. Imports `registry` for `Tuning`.

**Deliberate scoping decisions (carried into Phase 5 / final audit):**
1. **Tests carrier lives in the eval domain, not registry.** `RegisterProvider`
   carries descriptor + tuning + control_token; the corpus (`tests`) self-registers
   via the existing `EvalService.RegisterSuite` — honoring eval.proto's stated
   "sibling, not fork" boundary and avoiding a duplicate `EvalCase` schema. The
   scenario's `search.json` is still the single SSOT; self-registration fans out to
   two RPCs. This refines §6.2's "one call" wording; reversible (a `tests` field can
   be added additively later).
2. **cli-health's private `cli-health/v1/reindex` proto + handler are NOT deleted
   in Phase 3.** Deleting now would red cli-health's build (its handler/CLI still
   import the generated `reindex_v1`). The shared contract now *exists*; **Phase 5**
   implements it in cli-health (the "repointing") and deletes the private one in the
   same change. Transient coexistence across the phase boundary — not a shipped
   "v2 alongside v1"; greenfield end-state still reaches deletion by plan completion.
3. **`eval.proto` `ConfigSnapshot` left untouched** — its completion
   (`embed_task_prefix`, `rerank_blend`, `engine`, `floor_regime`) is **Phase 6**.

**Verification:** `buf lint` clean; `buf generate` regenerated go/python/ts for the
three search-hub domains (gen footprint scoped to `search-hub/v1/{registry,routing,
control}`); generated Go compiles + vets; search-hub/api, searchregister-go,
cli-health/api, knowledge-observatory/api all `go build ./...` green; search-hub
`internal/{registry,eval}` + `handlers/*` tests pass.

**Note for Phase 4/5 agent:** `make breaking` as written (`--against master`) cannot
run here — master predates the entire search-hub proto surface AND the vendored buf
module structure differs, so buf reports "3 images vs 1". Verify additive-ness with
image-vs-image instead: build a baseline from HEAD (or a `/tmp` copy reverting your
edits), `buf build -o /tmp/before.binpb`, then `buf build -o /tmp/after.binpb` and
`buf breaking /tmp/after.binpb --against /tmp/before.binpb`.

**Refs:** `packages/proto/schemas/search-hub/v1/{registry,routing,control}/*.proto`;
Search Self-Tuning System plan §7 Phase 3 + §8 Contract Decisions.

### Search Self-Tuning System — Phase 4 (query-time override channel, token-gated) DONE

The secured, query-time override channel + its control-token foundation, end to end
and green. Four parts:

1. **aisearch-go override channel (the mechanism).** New `SearchOverrides` (pointer
   fields mirroring proto `routing.SearchOverrides`; query-time factors only — no
   engine/embed, those are index-time), an `OverridePolicy` seam (`DenyOverrides`
   default / `AllowOverrides` / custom subset), and `Service.Search(ctx, q, ...SearchOption)`
   with `WithOverrides`. The Service resolves a per-call `effectiveParams` (defaults
   overlaid by policy-permitted, **always-clamped** overrides; re-enforces
   `rerank_blend⇒rerank_enabled`) and never mutates the shared Service. The shared
   header transport (`override_transport.go`: `X-Search-Control-Token`,
   `X-Search-Overrides` + JSON marshal/parse) lives here so sender (search-hub) and
   receiver (providers) cannot drift.

2. **Control-token mint/persist (the Phase 2 tail Phase 3 unblocked).** Registry
   `providers.control_token` column; `Store.Upsert(ctx, d, presentedToken)` mints on
   first insert, echoes thereafter, and rejects a **non-empty mismatched** token on
   re-register (`ErrTokenMismatch`→`PermissionDenied`); empty presented is allowed
   (the in-memory-only restart case). New `Store.Token(id)` lookup for the eval/sweep
   client. `RegisterProvider` returns the token; `searchregister.Register` now pushes
   `tuning` on the descriptor **and** captures the returned token via an
   `OnControlToken` callback.

3. **search-hub eval `ProviderClient.Search` forwards overrides + token.** New
   `internaleval.SearchCallOptions{Overrides, ControlToken}` (zero value = baseline,
   the runner's path today); the HTTP client sets the override/token headers via the
   shared aisearch contract when non-zero. search-hub now depends on `aisearch-go`
   (lightweight, no transitive vrooli deps) — aligned with §8 "search-hub consumes
   the factor taxonomy from aisearch-go".

**Deliberate scoping (for the final audit):**
- The override channel is **dark by default**: the provider's experiment flag
  (`<PREFIX>_SEARCH_OVERRIDES_ENABLED`) defaults off, and an unregistered provider has
  no token, so a mismatch/disabled gate degrades to ordinary public search — never an
  error. This matches the plan's "public search path untouched".
- The **router** path (client→hub `routing.QueryRequest.overrides`/`control_token`)
  is **not** wired to the provider call in Phase 4 — only the **eval/sweep** client is
  (the plan's Phase 4 item). The sweep (Phase 6) fills `SearchCallOptions` per arm
  (overrides from the factor enumeration; token from `Store.Token`).
- KO is **not** yet adopted for overrides (its `Service` still gets the default deny
  policy) — Phase 8 adoption-parity item.

**Pre-existing blocker hit while verifying (NOT this plan's work):** `packages/proto/
schemas/measures/v1/measures.proto` exists but its Go gen (`packages/proto/gen/go/
measures/v1/`) was never generated in this working tree, so `cli-health/api` (which
imports `measures-go` transitively via `internal/aisearch/discovery.go →
internal/measurescan`) could not build at all. Generated the measures gen with
`buf generate` to unblock verification; reverted the unrelated tree-wide gen churn
(audio-tools/BAS/data-backup-manager/DTV/ecosystem-manager/flow-verifier — 55 files)
back to HEAD so the Phase 4 footprint stays scoped. **The measures gen is left
untracked for the measures-plan owner to commit; filed as a bug.**

**Ripple fixed:** making `aisearch-go` `Service.Search` variadic (`...SearchOption`)
changed its method signature, so KO's narrow `docSearchEngine` seam +
`fakeDocSearch` (both spelled the non-variadic signature) stopped matching. Updated
both to the variadic signature (KO passes no options — overrides are a cli-health
adopter feature; KO keeps its measured default). A non-variadic interface does NOT
accept a variadic method, so any future narrow seam over `Service.Search` must spell
the `...SearchOption` tail.

**Verification (all green):** `aisearch-go`, `searchregister-go` build+vet+test +
gofumpt-clean; `search-hub/api` full `go test ./...` green; `cli-health/api`
build+vet + `handlers/search` (incl. `-race`) + `internal/aisearch` tests green;
`knowledge-observatory/api` build+vet+`go test ./...` green.

**Refs:** `packages/aisearch-go/{override,override_transport}.go`;
`scenarios/cli-health/api/handlers/search/`; plan §7 Phase 4 + §8 Contract Decisions.

### Search Self-Tuning System — Phase 5 (reindex/config-write contract end-to-end) DONE

**What:** the shared, token-gated `SearchControlService` (Phase-3 proto) is now
implemented end-to-end. **Provider side (cli-health):** the cli-health-private
`cli-health/v1/reindex` proto + handler are **deleted**; cli-health implements
`SearchControlService` (`handlers/searchcontrol`) — `Reindex`/`ReindexStatus`/
`ReindexCancel` wired to aisearch-go's reconcile job control, and `WriteConfig`
wired to a `search.json` writer. Token + per-env-flag gated
(`CLI_HEALTH_SEARCH_CONTROL_ENABLED`, default OFF). `search.json` now declares
`reindex_endpoint` + `config_endpoint`; the search.schema.json + aisearch-go
`ProviderConfig` carry them; searchregister-go maps them onto the descriptor.

**Registry-side client (this scenario):** `internal/control/client.go::Client` is
the counterpart to the eval/read client — it calls a provider's `SearchControlService`
over the generated Connect client, resolving the live base URL via discovery from
the descriptor's `reindex_endpoint`/`config_endpoint`, presenting the control token
(from `registry.Store.Token`), with bounded retry on transient transport failures
only. **It is not yet mounted/consumed** — the Phase-6 sweep is its caller.

**aisearch-go write primitive:** `WriteProviderTuning(path, id, tuning, dryRun)`
loads + validates `search.json`, atomically rewrites one provider's `tuning` block
(whole-file reserialize; descriptor sub-objects round-trip as `json.RawMessage`,
tests corpus through its typed shape), and reports `indexTimeChanged` so the caller
reindexes. `TuningConfig.IndexTimeChanged` compares engine/embed_model/embed_task_prefix.

**Deliberately deferred (NOT bugs):**
- The control client exists but is unwired — Phase 6 (sweep) drives it.
- `WriteConfig`'s in-process reindex reconciles with the boot-time embedding recipe;
  an index-time recipe change fully applies only after the provider restarts (boot
  rebuilds the engine + recipe-aware drift hash re-embeds). Live engine rebuild so
  an index-time change applies without restart is the Phase-6 sweep's job. See
  cli-health PROBLEMS.md for the provider-side detail.
- KO is not yet a control-plane adopter (Phase 8).

**Manifest-validation ripple:** cli-health's manifest validator learned the shared
control plane — `internal/services/manifestvalidation` loads `search-hub/v1/control`
alongside each scenario's own protos and treats its services as **Shared** (bindable
but not coverage-checked), so a provider's CLI may bind `SearchControlService.*` and
leave `WriteConfig` unbound without `binding.unknown_service`/orphan findings.

**Verification (all green):** `aisearch-go`, `searchregister-go` build+vet+test +
gofumpt-clean; `search-hub/api` + `cli-health/api` (incl. manifestvalidation +
searchcontrol, `-race` where relevant) + `cli-health/cli` + `knowledge-observatory/api`
build+vet+test green; `buf lint` + `buf build` clean on the affected schemas. NOTE:
`make breaking` will flag the intentional `cli-health/v1/reindex` deletion (greenfield
§0) — expected, not a regression.

**Refs:** `internal/control/`; `scenarios/cli-health/api/handlers/searchcontrol/`;
`packages/aisearch-go/searchjson_write.go`; `packages/searchregister-go/descriptor.go`;
`packages/proto/schemas/search-hub/v1/control/`; plan §7 Phase 5.

### Search Self-Tuning System — Phase 6 (sweep orchestrator + overfit guards + write-back) DONE

The closed optimization loop is built. `ConfigSnapshot` is **completed** (added
`embed_task_prefix (6)`, `rerank_blend (7)`, `engine (8)`, `floor_regime (9)`) so every
swept arm is fully self-describing; a new `EvalService.Sweep` RPC +
`SweepRequest/SweepResult/SweepArm/SweepStats` carries the two-tier sweep. The
transport-free core lives in `internal/sweep/` (seams: `SuiteReader`,
`ProviderReader`, `ArmRunner`, `ConfigController`, clock, rand — see SEAMS.md):

- **Query-time tier** — full-factorial over the query-time factors (`taxonomy.go`)
  via per-request overrides (the Phase-4 channel); one stored, tagged run per arm.
- **Index-time tier** — coordinate-ascent (one factor moved per arm, to bound reindex
  cost); each arm = config-push → reindex → poll-to-terminal → run; un-explored
  interactions are **reported** (`SweepStats.dropped_index_interactions`), never
  silently capped. The incumbent config is restored at the end of the tier.
- **Four overfit guards (each independently test-backed, `guards.go`):** (1) paired
  bootstrap CI of the per-case recall margin — promote only when the CI excludes 0;
  (2) held-out validation split (auto-`generated` cases always held out) — the winner
  must not regress on the held-out fold; (3) multi-objective constraints (gibberish
  ceiling + latency budget, anchored to the incumbent); (4) complexity / incumbent
  tie-break (dense < hybrid, rerank-off < on, …; switch only past the noise band).
- **Write-back** — `--apply` persists a cleared winner via the control client's
  `WriteConfig`; without it the sweep previews the ranked table + recommendation.

CLI: `search-hub evals sweep <suite_id> [--query-time-only] [--apply] [--limit N]`.

**Deliberately deferred (NOT a bug) — index-time live-apply is a provider-side prerequisite for live validation.**
The index-time tier drives the contract correctly (write config → poll reindex → run),
but it is only *meaningful live* once the provider applies an index-time recipe change
**in-process** — i.e. rebuilds its engine + re-embeds without a restart (Phase-5
boundary #1, above; `WriteConfig`'s reconcile still uses the boot-time recipe). That
in-process rebuild is a **cli-health (provider) change**, not a search-hub one; until
it lands, run the sweep with `--query-time-only` for trustworthy results (the cheap,
full-factorial path needs no reindex). The index-time orchestration is fully unit-tested
against the `ConfigController`/reindex-poll seams; its live correctness is gated on the
provider-side rebuild + the Phase-8 live validation. **This is the one provider-side
item the Phase-7/8 agent must land before an attended index-time sweep.**

**Verification (all green):** `search-hub/api` + `cli` build+vet+test (`internal/sweep`
`-race`, gofumpt-clean); `.vrooli/endpoints.json` regenerated (`evals_sweep` + CLI seed);
`packages/aisearch-go`, `searchregister-go`, `cli-health/api`, `knowledge-observatory/api`
build+test green (additive eval.proto change is contained to search-hub — no other Go
module imports `search-hub/v1/eval`); `buf lint` clean. NOTE: `make breaking` reports a
git-workspace image-count artifact in this environment, not a real break — the eval.proto
change is additive only (new field numbers 6–9, a new RPC, new messages).

**Refs:** `internal/sweep/{sweep,taxonomy,score,guards}.go` (+ tests);
`handlers/eval/{module,connect_handler,endpoints}.go`; `cli/domains/evals/`;
`packages/proto/schemas/search-hub/v1/eval/eval.proto`; plan §7 Phase 6.

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
