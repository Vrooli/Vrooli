# Maturation Plan — debt, boundaries, and sequencing

This document is the durable plan for making Image Tools cheaper to extend and
easier to reason about. It is deliberately concrete: every item names the evidence,
the fix, and the gate that closes it.

Companion documents:

- [`CAPABILITY-MANIFEST.md`](CAPABILITY-MANIFEST.md) — the substrate change that removes the per-operation tax
- [`../concepts/CAPABILITY-MODEL.md`](../concepts/CAPABILITY-MODEL.md) — the classification the substrate implements
- [`../concepts/COMPOSITION.md`](../concepts/COMPOSITION.md) — workflows and cross-scenario interop

## Stance

The architecture is fundamentally sound: a single operation vocabulary, a
technique/model/provider spine, explicit seams, proto-owned contracts, and a
headless-completeness tenet that is actually enforced. This is not a rewrite.

The problem is **bookkeeping**, not structure. The system is data-driven at the
vocabulary level and bespoke everywhere below it, so each new capability pays a
seven-surface tax and the payer is usually a human remembering six other files.
The fix is to extend the data-driven property downward, and to delete what has
already drifted.

Two rules govern every item here:

1. **No behavior change in a cleanup phase.** Cleanup and capability work do not
   share a commit.
2. **A guard replaces a habit.** If a rule matters, a test enforces it. If it
   cannot be enforced, it belongs in a doc, not in a checklist.

## Current state (baseline 2026-07-05)

| Signal | Value |
|---|---|
| Source LOC (api + cli + ui/src + proto + python) | 121,787 |
| Tests | API packages green; UI 106 files / 816 tests; CLI green |
| Test LOC | ~46,900 (roughly 1:1 with source) |
| Core package coverage | ai 75%, models 81%, ops 80%, jobs 79%, treatments 94%, storage 72% |
| Thinnest coverage | `handlers/adapters` 4.9%, `handlers/preparation` 0%, `internal/catalogstate` 0%, `main.go` 16% |
| Duplication (`dupl -t 60`, api non-test) | ~762 LOC of 40,780 (1.9%) |
| Operations | 70 total (38 deterministic + 32 vocabulary) |
| Models | 62 seeded, 30 enabled, 18 backends |
| Open debt entries | 17 in `PROBLEMS.md` (5 resolved) |
| Requirement ledger | 14 refs to non-existent packages; 50 validations falsely `failing` |

## Debt register

Ordered by leverage, not severity. Severity is about correctness; leverage is
about how much future work it unblocks.

| # | Item | Severity | Evidence | Fix | Gate |
|---|---|---|---|---|---|
| D1 | Seven hand-synced surfaces per operation | high (velocity) | `CAPABILITY-MANIFEST.md` §problem | manifest + codegen | **done:** manifests cover all 71 ops; the UI catalog, the AI command table, the deterministic command table, the deterministic *parameter forms*, and the wire binding are all generated + drift-gated. The deterministic table and forms are derived from the `OpParams` descriptor, so a parameter is reachable as a flag and as an Inspector control the moment it exists on the wire, and a form field that the wire lacks fails generation |
| D2 | Requirement ledger is false | high (trust) | 9 dead refs; 47 false `failing`; `PROBLEMS.md` 2026-06-18 | delete dead refs, re-derive status from real runs | **done 2026-07-05:** 47 re-derived, 5 remapped (recipes/presets→Looks, byok→gateway), 4 removed; zero refs to missing paths |
| D3 | Typed→untyped param bridge | high (correctness) | `handlers/ai/params.go` `map[string]string`; `handlers/ops/params.go` switch | `Struct` + schema; generated binding | **done:** AI (`op_params` + schema), deterministic (`internal/wireops` generated from `OpParams`), and the typed boundary all bridge by data. `internal/opparams` is generated from each operation's `params_schema` (Go structs + fail-closed decoders) and the UI gets the TypeScript mirror; the technique and builtin argument builders read the typed views instead of parsing strings by hand. The persisted job payload stays a string map on purpose — it is the stable cross-restart contract, so the hand-parsing is what was deleted, not the wire format |
| D4 | Provider quirks selected by string match | medium | `RegisterProviders` name branches | spec fields (`warm_sidecar`, `gpu_probe`, `progress_scan`) | **done 2026-07-05:** no `s.name ==` in registration |
| D5 | `ai/` mixes one execution state machine with two separable concerns | medium (comprehension) | engine + providers + capacity + gateway + warm sidecar | extract by concern; keep the coupled core intact | package sizes + import graph | **done 2026-11-19:** `internal/aigateway` owns the remote AI Gateway provider, its execution-evidence format, and cost parsing; `internal/capacity` owns the GPU/RAM/CPU/disk ledger (and is now imported directly by the CPU scheduler, not through `ai`). The remaining `ai` package is one admission→execution state machine: `execProvider` embeds `backendProcess`, and warm-model residency shares the lease group, so splitting it further would export internals rather than create a boundary. Non-test size 4.8k→4.0k (the earlier 9.5k figure counted tests) |
| D6 | Duplicated perceptual hash | medium | `analysis/duplicate.go` + `diff/metrics.go` | **done** → `internal/percepthash` | single implementation; parity tests |
| D7 | Orphan DB tables | low | live `attachments`, `notes`; zero code refs | drop via versioned migration | **done 2026-07-05:** `migrations/0001_drop_template_tables.sql`, applied at boot |
| D8 | No schema migration story | medium (future) | 7 `schema.sql`, all `IF NOT EXISTS` | versioned migrations | **done 2026-07-05:** `internal/database/migrate.go` (embedded, ordered, transactional, `schema_migrations`); boot runs apply → migrate → verify |
| D9 | Thin adapter/import coverage | medium | `handlers/adapters` 4.9%, `handlers/preparation` 0% | focused tests before feature work there | package floors | **done 2026-11-19:** adapter Connect boundary 4.9%→44.9% (catalog filter, lookup, read-only write refusals, doctor, error mapping) and preparation 0%→71.0% (proto pass-through, error mapping, validation, owned-reference rules). `make coverage-floors` now enforces those plus the model/AI/backend/capacity/gateway floors at a measured baseline |
| D10 | Redirect/template residue | low | `client.test.ts` `/things/...`, stale manifest description | delete template nouns | **done:** no `/things/` or `attachments` fixture remains; the old manifest description is corrected | grep for template nouns |
| D11 | `golangci-lint` binary/Go version skew | low | host binary built with Go 1.25 vs project 1.26 | make the target honest about the environment | **done 2026-11-19:** `lint-go` is a scenario target that runs the full linter when it can analyze the module and otherwise falls back to `go vet` + `gofmt` with a visible notice; CI remains the full-linter authority. The fallback exposed two real vet findings (a self-assignment and a mutex copy in tests) and they are fixed, so `go vet ./...` is clean in both modules. Repo-wide `pnpm lint` also went 55 errors → 0 (generated files are no longer linted; the remaining source errors were fixed, and the pose-control copy is now i18n) |
| D12 | `HostToolBindings` hand-appends | low | llama.cpp/tesseract outside the spec table | **done 2026-07-06:** `hostToolOnlyBindings` is a declared table beside the provider specs, so the projection reads two tables and adding a tool-only binding is a row | bindings derive from declared tables |
| D13 | Two operation registries | medium (coherence) | `internal/ops` (38 deterministic) vs `internal/operations` (32 vocabulary) | converge into one declaration set | **done 2026-07-06:** `internal/operations` deleted. The manifest set declares the engine (`ai` / `analysis` / `compute` / `unwired`) and the application family; `internal/ops` owns execution only; the AI catalog and analysis catalog are projections of the manifest set, proven by an engine-agreement conformance test |

`PROBLEMS.md` remains the register of record for incident-shaped findings; this
table is the *program* view. When an item here closes, note it in both.

## Boundary and package moves

The existing W3 decision — do not fracture `generation`/`enhancement` into
separate packages — is correct and stands. Splitting by *operation category* would
duplicate the engine. Splitting by *concern* is different and warranted.

| Current | Problem | Target |
|---|---|---|
| `api/internal/ai/` | engine, provider exec, capacity admission, warm sidecar, gateway, catalog in one package | **done:** `internal/aigateway` and `internal/capacity` extracted. The rest is one admission→execution state machine and stays together (see D5) |
| `api/handlers/ai/params.go` | typed→untyped bridge + per-op validators | delete; manifest schema + codegen |
| `providerSpecs()` | Go literals + name switches | data file beside `registry.seed.json` |
| `ui/src/features/workspace/{ai,create,analyze,op}Catalog.ts` | four hand-synced presentation maps | one generated catalog keyed by manifest |
| `internal/analysis/duplicate.go` + `internal/diff/metrics.go` | duplicated numeric primitive | **done** → `internal/percepthash` |

The target `ls api/internal/` should read as the domain model itself:
`operations · capability · technique · models · resolver · infer · compute ·
compose · looks · jobs · storage · safety · backends`. Capability words visible at
the top level, infrastructure kept behind them.

## Delete list

Delete in this order; each is independently safe.

1. ~~`AIParams` per-op field groups, `paramsMap`, `validate*`~~ — **done**; and the
   deterministic equivalent, `handlers/ops/params.go`, is generated in
   `internal/wireops`.
2. ~~Hand-written per-op CLI command registrations~~ — **done:** the deterministic
   table is generated from `OpParams`; the AI table from the manifests.
3. ~~Hand-written UI presentation maps~~ — **done:** one generated catalog.
4. ~~`HostToolBindings` hand-appended rows~~ — **done:** declared as data.
5. Duplicate providerSpec entries that share a name (`python-sidecar` appears twice).
6. Orphan `attachments` / `notes` tables.
7. Template-vocabulary test fixtures (`/things/...`).

Do **not** delete: the operation vocabulary, the technique spine, the model seed,
the conformance tests, or the seam registry — those are load-bearing.

## Quality gates

Existing, keep:

- `go build ./...`, `go vet ./...`
- full `go test ./...` (all packages green)
- UI `pnpm type-check`, `pnpm test`, `strings:check`, `selector:check`
- `make endpoints`, `make backends-doc`
- `packages/proto && make verify-committed-gen`

Add as the plan lands:

| Gate | Catches | Status |
|---|---|---|
| `TestManifestsAreTheOperationSet` | a vocabulary op with no manifest | **landed** |
| `TestManifestBackendsExistInModelSeed` | a typo'd or retired backend | **landed** |
| `TestManifestConsentWeightIsAuthoritative` | a second consent table drifting from the manifest | **landed** |
| `TestGeneratedUICatalogIsUpToDate` | the silent-drift hole (op renders as a raw name) | **landed** |
| `TestDeterministicWireCoverage` | a deterministic op with no typed params, or a generated/exception mismatch | **landed** |
| `TestGeneratedOpsCLICommandsUpToDate` | a param added to `ops.proto` that never reached the CLI | **landed** |
| `TestOpParamsValidateAndThread` | schema params lost or mistyped on the wire | **landed** |
| `dupl -t 60` budget in CI | reintroduced duplication | proposed |
| `make coverage-floors` | thin packages, including adapters/preparation | **enforced** |

A compile/type-check failure naming a generated artifact is a freshness failure
first: run the owning generator before diagnosing the consumer.

## Phased plan

### Phase 0 — stop the bleeding (1–2 sessions)

D2, D6, D7, D10, D12, and the `ai` package split (D5). No behavior change.

**Exit:** `make check` green; ledger has no dead refs; duplication budget down;
orphan tables documented for removal.

### Phase 1 — manifest for existing ops (the pivot)

D1. **Landed:** `internal/capability` with the manifest type, loader, seed, and
five conformance tests; manifests for all 32 vocabulary operations; consent
weights now read from the manifest set. **Done:** CLI + UI catalogs are
generated (`make capability-gen`) and drift-gated; the REST-edge parameter
translation is generated from `OpParams` (`internal/wireops`); and the vocabulary
is one declaration set (D13).

**Exit:** deleting a manifest row fails the build; adding a model of an existing
family needs no Go change; a single declaration set covers both the vocabulary and
the deterministic registry.

### Phase 2 — params contract

D3. `Struct params` + schema codegen; delete the bridge. Prove with one new op
that touches zero `.proto` files.

**Exit:** the proof op runs from CLI, API, and UI; `paramsMap` is gone.

### Phase 3 — composition

See [`COMPOSITION.md`](../concepts/COMPOSITION.md). **Landed:** typed ports +
`scenario` on `LookStep`, lane derivation, chain validation (unknown op,
non-executable op, lane mismatch, unknown/mismatched port, image-after-text), and
the seeded `screenshot-cleanup` Look that runs the chain headlessly.

The chain is intentionally linear: each step consumes the previous output or the
workflow source. A DAG (fan-out/fan-in) is a different execution contract with
partial-failure semantics, not a cleanup of this one, so it is named in
COMPOSITION.md as a limitation rather than counted as debt.

**Exit:** a mistyped chain fails before queueing, and the cleanup workflow runs
headless — both met.

### Phase 4 — fleet interop

**Publish done (2026-07-06):** `common.v1.MediaOperation` +
`CatalogService/GetOperationCatalog` serve the manifest projection over a versioned
contract, so a workflow builder can read Image Tools' operations without importing
the scenario. The consumer side (a peer step such as caption → audio) belongs to
the workflow surface of the scenario that owns the peer operation; Image Tools
must not grow a dependency on audio-tools to prove it. Peer adoption is therefore
a consumer decision, not an open Image Tools task.

**Exit:** an audio or video step consumes an image output without either scenario
importing the other.

## Risks and non-goals

- **Risk: manifest sprawl.** 69 hand-written manifests could become a new
  seven-surface problem if the schema is allowed to grow per-op escape hatches.
  Mitigation: schema versioning and a rule that a new field must be justified by
  two operations, not one.
- **Risk: generated-code opacity.** Generated CLI/UI must remain debuggable.
  Mitigation: generate readable output, keep the generator in-repo, and document
  the regeneration command beside it.
- **Risk: phase drift.** Cleanup and capability work interleaving reintroduces the
  debt. Mitigation: the two rules under "Stance".
- **Non-goal: a plugin sandbox.** Tiers 3–4 stay reviewed code.
- **Non-goal: unifying media scenarios into one repo/module.** Interop is a
  contract, not a merge.
- **Non-goal: rewriting the engine.** The spine works; extend it downward.

## Measuring progress

Track these in `PROGRESS.md` at each phase boundary:

| Metric | Baseline | Now |
|---|---|---|
| Surfaces per new model-backed op | 7 | **1 manifest row** (plus a provider/technique row and a model seed row for a new *backend*) |
| Surfaces per new deterministic op | 3 code + manifest | **proto message + compute function + manifest row**; CLI, UI catalog, parameter form, and wire binding are generated |
| Declaration sets per operation | 2 (vocabulary + deterministic) + wire | **1** (manifest), with runners projecting from it |
| Files touched per new op | ~20 | 1–3 |
| Duplicated API LOC | 1.9% | not re-measured since the CLI/binding deletions |
| False ledger entries | 14 refs / 50 flags | 0 |
| Unguarded surfaces | 2 (CLI, UI) | **0** — CLI, UI catalog, parameter forms, and the wire binding are drift-gated |
| Thinnest package coverage | 0–4.9% | **27% (capacity), 44.9% (adapters), 71% (preparation)**; enforced by `make coverage-floors` |

D5 (package split), D9 (coverage), D10, and D11 are closed. The minor-safety
boundary is now enforced by a real refusal path rather than a documented
aspiration: the policy requires a reviewer for image output on a public
deployment, refuses when none is configured, fails closed when a configured
reviewer errors, and the engine deletes outputs that the reviewer flags. The
reviewer is now bundled and on by default (2026-11-22) — see SECURITY.md
"Reviewer calibration". D13 (one operation declaration set) is closed: the
manifest set is the declaration, `internal/ops` owns execution only, and both the
AI and analysis catalogs are projections of it.

**No open debt items remain.** D3 closed with generated typed parameter views:
`api/internal/opparams/params.generated.go` is rendered from each operation's
`params_schema`, and the technique/builtin argument builders read those types
instead of parsing the persisted string map by hand. The string map stays as the
stable cross-restart job contract; every reader now goes through a generated
decoder that fails closed on a value the schema no longer explains, so a schema
edit and its readers cannot drift silently.

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — incident-shaped findings
- [`PROGRESS.md`](PROGRESS.md) — lifecycle log and phase checkpoints
- [`CAPABILITY-MANIFEST.md`](CAPABILITY-MANIFEST.md) — Phase 1–2 detail
- [`../concepts/COMPOSITION.md`](../concepts/COMPOSITION.md) — Phase 3–4 detail
- [`TESTING.md`](TESTING.md) — validation scope
