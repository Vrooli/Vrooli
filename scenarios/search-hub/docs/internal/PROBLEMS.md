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
