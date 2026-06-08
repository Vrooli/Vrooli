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
wired to a `search.json` writer. Control-token gated (the per-env `*_ENABLED`
flags were removed 2026-06-08 — see the Phase 8 entry). `search.json` now declares
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

### Search Self-Tuning System — Phase 7 (corpus generation + adequacy checks) DONE

The corpus the sweep optimizes against can now be **grown from the index itself**
and **graded** for adequacy — closing the overfit risk a hand-curated dozen-query
corpus carries. Three pieces:

- **`internal/corpusgen/` — transport-free generation core** (seams `Sampler`,
  `Inverter`, `Deduper` — see SEAMS.md; 22 unit tests, `-race` clean). `Generate`
  samples the provider's index, inverts each item to a natural-language query
  (positive case anchored to the item's id), optionally proposes hard negatives
  (`expect_no_strong_hit`), and de-dupes candidates against the existing corpus +
  each other. **Every proposed case is marked `tags:["generated", <stratum>]`** —
  the load-bearing marker the sweep already keys guard #2 on (`generatedCaseIDs`
  → always held out of the tuning fold). Case ids are a stable FNV hash of the
  normalized query, so re-running `generate` is idempotent. Default `Inverter` =
  `OllamaInverter` (local gateway, `qwen3:1.7b`, override `SEARCH_HUB_CORPUSGEN_MODEL`);
  default `Deduper` = `JaccardDeduper` (see the deliberate limitation below).
- **`EvalService.Generate` RPC + `evals generate <suite_id> [--count N] [--negatives] [--apply]`.**
  Preview by default (proposals + resulting-corpus adequacy, no mutation); `--apply`
  appends the proposals and upserts. Production sampler (`handlers/eval/corpusgen.go::corpusSampler`)
  probes the provider's registered search endpoint and collects distinct hits.
- **Warn-level adequacy (`internal/eval/adequacy.go::CheckAdequacy`).** Never fails,
  never gates. Findings: `too_few_cases` (< `MinPositiveCases` = 12), `no_negatives`,
  `thin_difficulty` (one band only), `duplicate_query`, and `coverage_gap` (only when
  a live sample's strata are supplied — corpusgen passes them; `GetSuite`/`RunSuite`
  pass nil and run the structural checks). Surfaced on `evals show`, `evals run`, and
  `evals generate` (`GetSuiteResponse.adequacy`, `RunSuiteResponse.adequacy`,
  `GenerateResponse.adequacy`).

Also: extracted **`internal/ollama`** — the single gateway transport (`Generate` /
`Available` / envelope-unwrap / think-strip) the classifier, reranker, and inverter
now share (was duplicated across the two routing files); utils-unification, behavior
preserved verbatim, routing tests unchanged + green.

**Deliberate limitation (NOT a bug) — two stand-ins gated on follow-ups:**
1. **The sampler enumerates only what its probes surface.** The unified search
   contract has no list-all / dump RPC, so `corpusSampler` discovers items by issuing
   probe queries (the suite's existing case queries + content words from the
   descriptor) and collecting the hits. It therefore cannot see items no probe
   reaches; `Result.Sampled` reports the true count (no silent cap). Phase 8 (live
   validation) widens the probe set against the real index.
2. **De-dup is lexical (token-Jaccard), not embedding-semantic.** search-hub holds
   no embedder of its own (embeddings live provider-side); `JaccardDeduper` is a
   pragmatic stand-in behind the `Deduper` seam. When an embedder reaches search-hub,
   a cosine-over-embeddings `Deduper` drops in with no change to the generator.

**Verification (all green):** `search-hub/api` + `cli` build+vet+test (`internal/corpusgen`
`-race`, gofumpt-clean); proto-parity (`TestProtoConnectParity`) + gen-endpoints tests
updated and green; `.vrooli/endpoints.json` regenerated (`evals_generate` + CLI seed);
`buf lint` clean; eval.proto change additive only (new RPC + `AdequacyWarning` /
`GenerateRequest` / `GeneratedCase` / `GenerateResponse`, new `adequacy` fields on
`GetSuiteResponse`/`RunSuiteResponse`) and contained to search-hub (no other Go module
imports `search-hub/v1/eval`). NOTE: `make breaking` reports the same git-workspace
image-count artifact in this environment, not a real break.

**Refs:** `internal/corpusgen/*`; `internal/ollama/*`; `internal/eval/adequacy.go`;
`handlers/eval/{corpusgen,generate,connect_handler,endpoints,module}.go`;
`cli/domains/evals/{handlers,register}.go`; `cli/manifest.json`;
`packages/proto/schemas/search-hub/v1/eval/eval.proto`; plan §7 Phase 7.

### Search Self-Tuning System — Phase 8 (docs + adoption parity) — docs DONE; live + provider-rebuild OPEN

**What landed (docs deliverable, the plan §13 DoD doc item):**
- The control-surface **dashboard** is now documented as the canonical engine
  reference: `packages/aisearch-go/docs/reference/search-json.md` (NEW) carries
  the `.vrooli/search.json` schema + the per-knob factor table (key / tier /
  kind / default / decision rule) + presets + the read/write lifecycle.
  `configuration.md` and `README.md` in `aisearch-go` were **de-staled**: the
  tuning SSOT is `search.json`, the env factor reads are demoted to operator
  overrides, and the tuning loop is `evals sweep`/`generate` write-back — not the
  old env-flag + `vrooli scenario restart` recipe.
- search-hub: `docs/reference/configuration.md` gained a **Search tuning control
  surface** section (the `evals sweep`/`evals generate` operator recipe, the four
  overfit guards, the adequacy codes, the control token, the two documented
  limitations). `ARCHITECTURE.md` gained **The self-tuning authority** zone map
  (sweep / corpusgen / control internal domains + the expanded eval/ollama
  roles). SEAMS.md already registered the new seams (Phases 6–7); no change
  needed.
- Adopting scenarios: cli-health + KO `configuration.md` gained **Search tuning
  (`.vrooli/search.json`)** sections (provider id, the measured tuning + why,
  self-registration, control plane; KO records the recall@5=0.818 guard).

**Provider-side in-process index-time apply — LANDED (2026-06-08).** The Phase-5/6
deferral is closed: cli-health's `WriteConfig` now calls `ApplyTuning`, which
rebuilds the live engine for the new tuning and re-embeds the corpus with the new
recipe **in process, no restart** (engine held behind an RWMutex + swapped
atomically; sync loop driven by `aisearchpkg.NewSyncLoopFunc` so it resolves the
current reconciler each tick). A structural dense↔hybrid change still needs a
manual collection rebuild (the schema guard surfaces it without auto-dropping
data). aisearch-go gained `NewSyncLoopFunc` for the swappable-reconciler case.
See `scenarios/cli-health/docs/internal/PROBLEMS.md`. This unblocks an attended
*index-time* sweep.

**Live engine validated through the refactor (2026-06-08, resources up).** Ran
cli-health's live recall gate (`CLI_HEALTH_AISEARCH_LIVE=1 go test -run
TestCommandRecall`, ~145 s): the full engine exercised the refactored
`aisearch.Service` end-to-end on real ollama + qdrant — embed (nomic task prefix)
→ qdrant retrieve → cross-encoder rerank+blend → score — into a scratch collection
(dropped on defer, live index untouched). **recall@5 = 0.714 (15/21)**, at/above
the documented ~0.70 post-`embed_task_prefix`+`rerank_blend` baseline, so the
engine-swap refactor **did not regress retrieval**. The 0.714 < 0.80 gate gap is
the pre-existing, documented label/manifest/vocab gap — every MISS is one of:
`git-control-tower baseline snapshot` (the GCT baseline-manifest omission, bug
`knw-1780810146590717122`, out of scope per plan §4), the `vrooli help`
non-command label, and ~4 vocab-gap cases (bug `knw-1780702659191582434`, stays
`in_progress`). This validates the read + reindex path ApplyTuning reuses; the
swap + re-embed mechanics are covered by `apply_tuning_test.go` (fakes).

**Env-flag gating REMOVED → token-only (2026-06-08, greenfield).** The Phase-4/5
`CLI_HEALTH_SEARCH_OVERRIDES_ENABLED` / `CLI_HEALTH_SEARCH_CONTROL_ENABLED`
default-OFF feature flags were deleted. They were redundant with (a) the
registration-minted control token (search-hub is its only holder) and (b) the
declarative opt-out (a provider omits its control endpoints in `search.json`), and
a default-OFF flag contradicted the DoD's "auto-tuning for free." The override gate
+ control gate are now token-only, so a plain restart is sweep-ready with no env
setup. (`handlers/search`/`searchcontrol` gate structs, `main.go`, tests, SEAMS.md
all repointed; dead `boolEnv` deleted.)

**Live sweep + generate VALIDATED end-to-end (2026-06-08, resources up).** After
restarting cli-health (fresh binary, token-only gating):
- `evals sweep cli-health.commands.primary --query-time-only` ran the suite across
  arms **via the token-gated override channel** — arms produced *different* recall
  (incumbent 0.286 vs query-time 1.000), proving cli-health actually applied the
  per-request overrides (token honored). **All four overfit guards fired:** the
  best arm (margin +0.500) was **not promoted** because its 95% CI [0,+1.0]
  overlaps 0 (significance); held-out fold winner=incumbent=0.000 (held-out); one
  arm flagged **INFEASIBLE "gibberish leakage above ceiling"** (constraint) →
  "no significant improvement — incumbent retained." The sweep correctly **refused
  a within-noise win** (the plan's contract). `search.json` left **unchanged**
  (sha-verified; no `--apply` needed since nothing was promoted).
- `evals show … --json` surfaced the adequacy warning `too_few_cases` ("only 7
  positive case(s) — below the floor of 12").
- `evals generate cli-health.commands.primary --count 6 --negatives` (preview):
  sampled 6 → LLM-inverted via ollama → 6 positives + 2 hard negatives, each
  marked `{generated,…}` (the sweep's held-out marker), preview-only.

**Defect found + filed during validation — `knw-1780935292042898963` (major).**
The sweep first failed `no such column: control_token`: the registry SQLite DB
predated the Phase-4 column and there is **no working additive-column migration**
(`ADD COLUMN IF NOT EXISTS` is invalid SQLite; `EnsureSchemas` execs the file
atomically), so **all** provider registration had been silently failing on any
non-fresh DB. Surgically unblocked (non-destructive `ALTER TABLE providers ADD
COLUMN control_token …` + cli-health re-register → real token minted). The
misleading "ADD COLUMN IF NOT EXISTS" guidance in `registry/schema.go` +
`eval/schema.go` was corrected; the real fix is the deferred brownfield
additive-migration mechanism (escalate per storage-steer §6) — see the bug.

**Still OPEN:**
1. **Index-time sweep** (`embed_task_prefix` arm, now unblocked by the in-process
   apply) + a `--apply` write-back demo (snapshot `search.json` to /tmp + restore;
   only meaningful once a corpus large enough to clear the guards exists — grow it
   first with `evals generate --apply`). Reconfirm **KO recall@5 = 0.818**
   (`KO_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestAccuracyCorpus`).
2. On plan completion, write the `kind: execute` record
   (`swarm-manager records create`) per plan §13.

**Refs:** `packages/aisearch-go/docs/reference/{search-json,configuration}.md`,
`packages/aisearch-go/README.md`; `scenarios/{search-hub,cli-health,
knowledge-observatory}/docs/`; bug `knw-1780935292042898963`; plan §7 Phase 8 +
§13 DoD.

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
