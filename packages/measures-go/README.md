# `measures-go` — the Measures contract library

`measures-go` is the SSOT for what a **measure** is and the helpers a scenario
imports to declare, resolve, and serve them. A *measure* is a **named, typed,
parameterized analytical query** — "how many backlog items closed this week",
"what's the median execution cycle time this month" — declared once per
scenario so that `search-hub` can semantically match a natural-language
question to it, fill its parameters deterministically where possible, and (for
safe read-only measures at high confidence) execute and return the answer.

This package is the foundation of the Measures plan. It has no scenario
dependencies; everything here is transport-agnostic and unit-tested in
isolation. Wiring into scenarios happens in later phases (cli-health discovery,
the search-hub provider, the `measures-health` enforcer, the swarm-manager
dogfood).

---

## The model

A `MeasureDeclaration` is **assembled at runtime from three sources joined on
the CLI manifest `binding`**:

```
MeasureDeclaration  =  { manifest `measure` block }   ← curated prose
                     ⊕ { manifest `governance` block } ← effect / run_eligible
                     ⊕ { proto-derived param schema }  ← types/enums/bounds (Phase 0 reader)
```

The split is deliberate and enforced:

- **Curated prose lives in the manifest** — `intent`, `questions[]`, the
  `result` presentation, a param `default`, a dynamic-enum `values_source`.
- **Param validation comes from proto** — a param's `type`, enum membership,
  numeric bounds, `uuid` format, and required-ness are read from the proto
  descriptor (`SchemaReader`, `paramschema.go`) and are **never duplicated** in
  the manifest. A manifest param naming a field that does not exist on the
  request message is drift and fails assembly.

```go
protoParams, _ := reader.RequestParams(binding.Service, binding.Method) // Phase 0 reader
decl, err := measures.Assemble(
    "backlog.completed", "backlog", binding,
    manifestMeasure,   // intent, questions, result, param defaults/values_source
    governance,        // effect, run_eligible
    protoParams,        // proto-derived param schema
)
```

`decl.Validate()` is called for you by `Assemble`; it is also what
`cli-health`/`measures-health` enforce statically.

---

## Canonical param conventions

Extraction of natural-language parameters degrades through **three tiers**
(`ResolveParams`):

| Tier | When | How |
|---|---|---|
| 1 — canonical-typed | `time_window` | **deterministic**, no LLM (`MatchTimeWindowToken` → `ResolveToken`) |
| 2 — constrained | static/dynamic `enum`, numeric bounds | LLM extraction **constrained** to the proto-derived value space |
| 3 — best-effort | a bare field | LLM extraction with proto comment/format as grounding |

```go
res, _ := measures.ResolveParams(ctx, question, decl, measures.ResolveOptions{
    Now:       now,            // explicit — never an ambient wall-clock read
    Loc:       loc,            // explicit timezone
    Extractor: extractor,      // Phase 3 wires the constrained-LLM impl; nil = abstain
    Values:    valuesProvider, // resolves dynamic-enum value_source → live values
})
// res.Params  — resolved values
// res.Needs   — REQUIRED params that could not be resolved (abstain, never guess)
// res.Confidence — min confidence across resolved params
```

The hard rule: **a required param that cannot be resolved goes to `Needs`, never
a guess.** Wrong numbers are worse than no number. The auto-execution gate
(Phase 3) only fires when `len(Needs)==0 && Confidence ≥ θ && effect=="read" &&
run_eligible`.

### `time_window` — the deterministic canonical type

The shared proto type `vrooli.measures.v1.TimeWindow` carries either a relative
`token` or an absolute `custom{from,to}` range. `ResolveToken(token, now, loc)`
maps the six canonical tokens to a concrete `[from, to)` range (From inclusive,
To exclusive), timezone-explicit and reproducible:

| token | range |
|---|---|
| `this_week` | Monday 00:00 (in `loc`) of the current week … now |
| `last_7d` | rolling now-7d … now |
| `last_30d` | rolling now-30d … now |
| `this_month` | 1st 00:00 of the current month … now |
| `last_month` | previous month 1st 00:00 … this month 1st 00:00 |
| `this_quarter` | first day 00:00 of the current quarter … now |

---

## Serving measures

A scenario builds one `Registry`, registers each declaration with a compute
func, and either mounts `Handler()` (framework-agnostic JSON) or adapts
`Execute` onto its Connect service. Every result carries **mandatory
provenance** (`executed_query` + `computed_at`) — auto-answers must be
auditable, so the helper stamps `computed_at` and validates params (required
presence, enum/time-window membership) before the compute func runs.

```go
reg := measures.NewRegistry()
_ = reg.Register(decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
    rng, _ := measures.ResolveToken(measures.TimeWindowToken(req.Params["window"]), now, loc)
    n := countCompleted(ctx, rng)
    return measures.MeasureResult{
        Value:      strconv.Itoa(n),
        Provenance: measures.Provenance{ExecutedQuery: fmt.Sprintf("count completed in [%s,%s)", rng.From, rng.To)},
    }, nil
})
http.Handle("/measures/", http.StripPrefix("/measures", reg.Handler()))
```

---

## The provider engine — match → resolve → gate → execute

The reusable *brain* a measures provider runs to answer a question. It is shared
by the search-hub reference provider (Phase 3) and the `measures-health` central
provider (Phase 4); only the seams differ. The provider stays the place all
measure logic lives — the search-hub router carries the result thinly.

```go
eng := measures.NewEngine(matcher,                       // semantic match seam (Phase 4 = aisearch-go index)
    measures.WithExtractor(measures.NewLLMExtractor(completer)), // constrained extraction (tier 2/3)
    measures.WithValues(valuesProvider),                 // dynamic-enum values
    measures.WithExecutor(measures.NewHTTPExecutor(resolver)),   // execution-proxy to the owning scenario
    measures.WithThreshold(measures.DefaultConfidenceThreshold), // θ (default 0.8, conservative)
    measures.WithEngineClock(now), measures.WithLocation(loc),
)
hit, err := eng.Answer(ctx, "how many backlog items did we complete this week")
// hit == nil           → no measure matched (honest empty result)
// hit.Answer != ""      → read-only measure auto-executed (hit.ExecutedQuery set)
// hit.Needs != nil      → a required param was unresolved — ask, never guess
// hit.Answer=="" & no Needs & effect write/destructive → resolved, withheld for confirmation
```

`Engine.Answer` returns a `MeasureHit` — the Go mirror of `routing.proto`'s
`MeasureHit`, which a provider serializes (snake_case keys) into its search-hub
response; the search-hub adapter decodes it into `SearchHit.measure` via the
descriptor's `measure_field` (zero provider-specific router code).

**The auto-execution gate** (`Gate`, pure) is the single enforcement point of
the read-vs-confirm contract, ordered safety-first:

1. missing required param → `GateNeedsParams` (ask, don't guess);
2. `write`/`destructive` or not `run_eligible` → `GateConfirm` (**never** auto-run,
   even at full confidence);
3. confidence `< θ` → `GateConfirm` (resolved but not sure enough to run unattended);
4. otherwise → `GateExecute`.

**`LLMExtractor`** is the constrained-extraction impl behind the `ParamExtractor`
seam. It depends only on a `Completer` (single-shot text completion) — measures-go
stays free of any model/transport, so the owning scenario supplies a
`resource-ollama` shell-out exactly like search-hub's classifier. It abstains
(found=false) on any ambiguity, and `ResolveParams` is the single point that
rejects an out-of-`allowed` answer.

**`MeasureComposer`** is the `aisearch-go` `EmbeddingTextComposer` for the central
index: it embeds the joined `questions[]` (+ `intent` grounding) — *how a user
phrases the question* — while the retrievable payload stays the measure
declaration. (Phase 4 builds the populated index over this composer.)

---

## The CQRS substrate (optional)

A measure may be backed by a plain SQL aggregate, a live external call, or the
**event-sourced read-model** substrate generalized here from swarm-manager's
stats engine. The substrate is *offered, not mandated*.

- `EventLog` — an append-only store (`Append/Since/All/MaxID`) with a
  monotonically increasing ID. Implementations: `MemoryEventLog` (reference /
  test) and `SQLEventLog` (driver-agnostic `*sql.DB`, configurable table — adopt
  an existing `events` table in place).
- `Projection` — the adopter's fold: `Apply(Event)` updates aggregate state,
  `Reset()` clears it for a rebuild. The projection owns its own query surface.
- `ReadModel` — drives the **watermark** pattern: `Rebuild` replays the whole
  log once at startup; `Refresh` folds only events appended since the last
  watermark. Call `Refresh` before each read so the aggregate is current.

```go
log := measures.NewSQLEventLog(db, "events")
_ = log.InitSchema(ctx)
rm := measures.NewReadModel(log, myProjection)
_ = rm.Rebuild(ctx)
// per read:
_ = rm.Refresh(ctx)
rm.Read(func() { answer = myProjection.Count("backlog.created") })
```

---

## Seams (testability)

| Seam | Interface | Production | Test double |
|---|---|---|---|
| Param extraction | `ParamExtractor` | `LLMExtractor` (constrained, over a `Completer`) | `NoopExtractor` (abstains) / a programmable fake |
| Single-shot completion | `Completer` | scenario `resource-ollama` shell-out | `CompleterFunc` returning canned JSON |
| Semantic match | `Matcher` | Phase 4 aisearch-go central index | a fixed-declaration fake |
| Execution proxy | `Executor` | `HTTPExecutor` (POST to the serve `Handler`) | a recording fake |
| Measures base URL | `BaseURLResolver` | search-hub cross-scenario URL resolver | a static func |
| Dynamic-enum values | `ValuesProvider` | scenario callback (e.g. live initiative names) | a static map |
| Event store | `EventLog` | `SQLEventLog` | `MemoryEventLog` |
| Read-model fold | `Projection` | scenario aggregate | a counting fake |
| Clock (provenance) | `Registry` `WithClock` | `time.Now` | a fixed clock |
| Time/timezone (resolution) | `ResolveOptions.Now`/`Loc` | request-time values | a fixed anchor |

Time is **always an explicit input**, never an ambient read, so resolution is
deterministic and tests are reproducible.

---

## Contract invariants (do not break in later phases)

- Param types/enums/bounds are derived from proto, never duplicated in the
  manifest.
- A required, unresolved param **abstains** (`Needs`) — it is never guessed.
- `write`/`destructive` measures are **never** auto-executed, at any confidence.
- Provenance is **mandatory** on every `MeasureResult`.
- Time-window resolution is deterministic and timezone-explicit (no LLM, no
  ambient clock).
