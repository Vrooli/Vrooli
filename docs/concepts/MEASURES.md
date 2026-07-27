# Measures — a federated metrics layer

> **What this is.** The mental model for **Measures**: a standardized way for every
> scenario to declare named, typed, parameterized analytical queries so that one
> natural-language question — "how many backlog items closed this week" — can be
> matched to a measure, have its parameters filled deterministically, and (for
> safe read-only measures) be answered with a computed number and provenance,
> sometimes without routing to a coding/reasoning agent at all.
>
> **Siblings.** The *contract* (the API you import) is `packages/measures-go`
> (`path:packages/measures-go/README.md`). The *enforcer* is the `measures-health`
> scenario. The *adoption how-to* is `prompt-manager skill read measures-adoption`.
> Measures ride the federation described in
> [`../reference/ai-search-routing.md`](../reference/ai-search-routing.md).

## Why this exists

`search-hub` federates **retrieval** — "find me things." It had no answer to
**analytical questions** — "how many / what's the rate / what's next" — whose
answer is a *computed value* parameterized by a time window, a filter, and a
grouping. Before measures:

- Some scenarios exposed rich stats, most exposed none, and there was **no way to
  check** which did or to drive adoption.
- Some machine-facing analytics endpoints were **monolithic `/stats` blobs**
  with hardcoded 7d/30d windows and an implicit event-type switch — not
  individually addressable, declared, or parameterized. A rich Stats product is
  not itself a defect; Measures provide a complementary programmatic contract.
- An agent answering "how many backlog items closed this week" had to discover
  tools, learn their parameters, and assemble a query by hand. There was no
  pre-declared "this question maps to this typed query" layer.

Measures close that gap: a **measure** is a *named, typed, parameterized query*,
declared once per scenario, that `search-hub` can semantically match, fill
deterministically where possible, and safely auto-answer when it is a read.

## The model: a measure = a typed, parameterized query

A measure has a stable id `<domain>.<name>` (e.g. `backlog.completed`) and is
**assembled at runtime from three sources joined on the CLI manifest `binding`**:

```
MeasureDeclaration  =  { manifest `measure` block }    ← curated prose
                     ⊕ { manifest `governance` block }  ← effect / run_eligible
                     ⊕ { proto-derived param schema }   ← types / enums / bounds
```

The split is deliberate and **enforced**:

- **Curated prose lives in the manifest** — the `intent`, the example
  `questions[]`, the `result` presentation, a param `default`, a dynamic-enum
  `values_source`.
- **Param validation comes from proto** — a param's `type`, enum membership,
  numeric bounds, `uuid` format, and required-ness are read from the proto
  descriptor of the bound request message and are **never duplicated** in the
  manifest. A manifest param naming a field that does not exist on the request
  message is drift and fails assembly.

A measure also carries a **result** shape (`scalar` / `table` / `series`, a
`value_field`, a `unit`, a `summary_template`) and a **governance effect**
(`read` / `write` / `destructive` + `run_eligible`) that decides whether it may
auto-execute. Every computed result carries **mandatory provenance**
(`executed_query` + `computed_at`) — an auto-answer must be auditable.

> Measures are **not** prompt-manager *actions* (agent-discovered intents over a
> single command), and **not** operational/Prometheus telemetry. They are a
> distinct analytical-query class with their own provider, gate, and proto-derived
> schemas. See `measures-go` README "Contract invariants".

## Domains, statefulness, and waivers

What a scenario is *expected* to cover is derived the same way cli-health and
security-health derive expectation — from its **stateful domains**, minus waivers.
The derivation is **mode-aware**, because `v1/domain/` (screaming architecture) is
the authoritative SSOT for stateful domains and the manifest `measures.domains[]`
is only a transitional crutch:

- **Conformant mode** — `packages/proto/schemas/<scenario>/v1/domain/` exists.
  **Expected = the stateful domains derived from that folder** (each domain proto
  is a persisted entity type; pure config/structural domains like `settings` or a
  positional `graph` are **substrate-filtered out** → *not expected*). Adding a
  *new* stateful domain via `measures.domains[]` is an **ERROR**
  (`measures.illegal-domain-declaration`) — add a `v1/domain/<d>.proto` instead.
  Down-grading a folder domain to `stateful:false`, and `measures.omitted[]`
  waivers, stay legal.
- **Fallback mode** — no `v1/domain/` folder (a flat `v1/<entity>/`-layout or
  template scenario). **Expected = the `measures.domains[]` entries marked
  `stateful:true`**, and the scenario carries a standing
  `measures.architecture-fallback` **INFO** advisory nudging it to adopt screaming
  architecture (after which the crutch disappears). The fallback path is a
  first-class transitional mode, not a back-compat bridge.
- **Substrate cross-check (layout-agnostic teeth).** Independently of the
  proto/manifest signal, measures-health detects persisted countable entities from
  evidence — a SQL `CREATE TABLE` with a `created_at` column (the conservative
  "rows accumulate over time" signal; singleton state rows and infrastructure
  tables like a CQRS `events` log are excluded). A detected entity that is neither
  covered nor waived raises `measures.undeclared-substrate` (**WARNING**), so a
  fallback scenario can't dodge the gate by simply omitting a real entity from
  `measures.domains[]`.
- **Covered** = the domain has ≥1 manifest `measure` block bound to it.
- **Waived** = the domain is listed in the manifest `measures.omitted[]` with a
  reason (it is stateful but genuinely has no historical/countable value — e.g.
  an ephemeral pre-triage inbox, a live derived feed).
- A stateful domain **neither covered nor waived → ERROR** (holds the maturity
  rung). A waiver pointing at a **non-stateful/nonexistent** domain → **WARNING**
  (stale). These are exactly cli-health's expected/covered/waived semantics
  (orphan → error, stale → warning).

```
swarm-manager — measures coverage
  backlog       ✓ covered   2 measures   tier: full
  execution     ✓ covered   1 measure    tier: full
  initiative    ✓ covered   1 measure    tier: full
  agent_session ✓ covered   1 measure    tier: full
  capture       ⊘ waived    "ephemeral pre-triage inbox; no historical value"
  settings      – not expected (stateless)
  verdict: PASS   maturity: clears the soft measures rung
```

**Anti-gaming.** Adding a waiver to hide a domain that *should* be covered is
suppression-shaped; the EM `gameguard` classifies it and zeroes the credit.
Declaring a measure whose endpoint 404s or returns the wrong shape is caught by
the behavioral probe as a `measures.hollow-declaration` ERROR. You cannot pass by
declaring without implementing, or by waiving a regression away.

**Producer probe-if-reachable.** The test-genie `measures` phase (the producer
that feeds EM's soft R4 `measures` rung) shells `measures-health validate
scenario <s> --probe` **when the target scenario is reachable**, and the static
path otherwise (an unreachable target is a skip, never a failure — test-genie
sweeps non-running scenarios). So the rung reflects "the measure *actually
answers*", not merely "a measure is declared".

## The answer-vs-resolve gate

Parameter extraction degrades through **three tiers**, safest first:

| Tier | When | How |
|---|---|---|
| 1 — canonical-typed | a `time_window` param | **deterministic**, no LLM — token → `[from, to)` range, timezone-explicit |
| 2 — constrained | a static/dynamic `enum` or numeric bound | LLM extraction **constrained** to the proto-derived value space |
| 3 — best-effort | a bare field | LLM extraction with the proto comment/format as grounding |

The hard rule: **a required param that cannot be resolved abstains** (goes to
`needs[]`) — it is *never* guessed. Wrong numbers are worse than no number.

Whether a matched, resolved measure is **executed** or merely **returned for
confirmation** is decided by a single pure gate, ordered safety-first:

1. a required param is missing → **ask** (return `needs[]`, no answer);
2. the measure is `write`/`destructive`, or not `run_eligible` → **confirm**
   (never auto-run, *even at full confidence*);
3. confidence `< θ` → **confirm** (resolved but not sure enough to run unattended);
4. otherwise → **execute** (call the owning scenario, fill `answer` +
   `executed_query`).

So `search-hub query "how many backlog items closed this week"` matches the
measure, deterministically resolves `window = this_week`, and — because the
measure is `read` + `run_eligible` at confidence ≥ θ — **executes and returns the
number with provenance**. A `write`/`destructive` measure returns resolved-but-
unexecuted with a confirmation prompt. θ is a conservative, tunable config lever.

## Where measures live in the system

- **`packages/measures-go`** — the contract SSOT: the `MeasureDeclaration` type,
  the deterministic `time_window` resolver, the serve `Registry`, the auto-exec
  `Gate`, the constrained `LLMExtractor`, the `MeasureComposer` (embeds
  `questions[]`), and the optional CQRS event-sourced read-model substrate.
- **The CLI manifest** — each adopter authors a `measure` block on a command (the
  curated prose) bound to a Connect-RPC method (the param schema). cli-health
  discovers and statically validates these blocks.
- **`measures-health`** — harvests every scenario's `measure` blocks into the
  **central index**, owns the *single* registered `search-hub` "measure" provider
  (match → extract → gate → execute-proxy), grades coverage + behavior into a
  **tier**, and feeds a **soft** EM ladder dimension (a scenario stays runnable
  and safe without measures, but cannot reach top maturity without them).
- **`search-hub`** — the thin router carries a `SearchHit.measure` sub-message
  (`measure_id, scenario, params, answer?, needs[], effect, executed_query?,
  confidence`); all measure logic stays in the provider, never in the router.
- **`templates/scenarios/react-vite`** — ships a reference measure at full tier so
  every new scenario is measure-ready by default.

## Adopting measures

Read `prompt-manager skill read measures-adoption` for the step-by-step (identify
stateful domains → choose a substrate → classify any existing analytics surface
as a transport contract or an analytics product → author the binding + manifest
block → serve via the Registry → prove it with `measures-health --probe`). A
transport-only aggregate contract can retire after semantic parity and consumer
migration. A Stats dashboard, operational CLI, or curated analytics workflow is
a separate consumer surface and remains in place over shared analytical
semantics. The package README is the API reference the skill cites.
