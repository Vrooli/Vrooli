# Answer Space — Architectural Question Coverage

> **Model & terminology** — the attestation contract (basis × sufficiency), the
> entity × archetype × aspect model, the denominator/numerator split, and the Status/Basis
> legend are defined once in the canonical model doc:
> `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md`. This document does not re-explain them.

> **Condition** — every cell that resolves `NOW` here puts its **provider** into the Condition
> population, and `search-hub` owes that provider's serving, freshness, and exercise signals as
> declared Measures. Condition is not a denominator and is not authored in this file; the model,
> the required signals, and this owner's current instrumentation are in
> `meta-optimization-manager/docs/concepts/CONDITION-MODEL.md`.

## Purpose

The **denominator** for the *Answer* projection: the bounded set of architectural questions,
each mapped to its owning provider, status, and best attainable basis. Coverage (the numerator)
is computed against the live `search-hub` provider registry — not stored here. The **Notes**
column doubles as the gap registry.

## This Space

| | |
|---|---|
| Projection | Answer |
| Owner | `search-hub` (holds the `capability_gap` provider registry this extends) |
| Tiers | G0 Project · G1 Scenario · G2 Within-scenario · G3 Domain · G4 Symbol · G5 Ecosystem |
| Denominator confidence | `PARTIAL` — cells enumerated from the model + provider map, not swept exhaustively or validated against real question logs; G0/G5 are sketchier than G1–G3. |
| Sibling spaces | `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md` |
| Legend | `NOW` live provider · `IN-REACH` substrate exists, build/attest the provider · `MISSING` no provider · ⭐ requester example. |

## Coverage Grid

### G0 — Project (whole repo)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 23 | Repo contract · Conformance — "Does the project conform to its repo contract; what's the layout?" | `contract-registry.contracts` (reads `repo-contract-go`) | IN-REACH (gap stub) | DERIVED / VALIDATED | contract-registry is the home; `repo-contract-go` is a package, can't be a provider. |
| 24 | ⭐ Control-plane CLI · Anatomy — "How is the top-level `vrooli` CLI structured/composed?" | _(none)_ | MISSING | DERIVED | Not "find a command" (cli-health does that) — the *structure* of `cmd/vrooli`. Needs a control-plane registry. |
| 25 | Shared packages · Inventory + Connection — "What `packages/*` exist and who consumes each?" | `code-facts` (`DescribeFleetImports`) / `code-reference.code` | IN-REACH | DERIVED | Fleet imports + code-reference. |
| 26 | Resource fleet · Inventory + Anatomy — "What resources exist (ports/health/driver) and how do they fit together?" | _(none)_ | MISSING | DECLARED_UNVERIFIED | Per-`resource.json` exists; no fleet aggregator/provider. |
| 27 | Contracts/schemas · Inventory — "What contracts/schemas govern the project and what do they enforce?" | `contract-registry.contracts` | IN-REACH (gap stub) | DERIVED | |

### G1 — Scenario (whole)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 1 | Surfaces · Inventory — "What surfaces (API/CLI/UI) does X expose?" | `ui-health.surfaces` + `cli-health.commands` + `code-facts` (API) | NOW (UI, CLI) / IN-REACH (API) | DERIVED | UI + CLI live; API surfaces via code-facts `SURFACES` need a search surface + attestation. |
| 2 | Domains · Inventory + Anatomy — "What domains does X have and what does each own?" | `architecture-cartographer.domain-map` | NOW | DERIVED / VALIDATED | **First-slice provider, live.** Semantic search over the derived domain map (`ExtractDomains` responsibility/purpose/glossary) — term-agnostic: "how does authoring work in plan-manager" → `plan-manager/authoring`. |
| 3 | Doc-set · Conformance — "Do X's docs match its code? Where's the drift?" | `architecture-cartographer` (drift detectors) | IN-REACH | VALIDATED / CONTRADICTED | Cartographer's core job; expose drift findings as attested answers. |
| 4 | Requirements/maturity · State — "What's X's maturity rung + requirements status?" | `completeness-scoring` (`GetScore`) | IN-REACH | DERIVED | Read `GetScore`, wrap in attestation. |
| 5 | Dependencies · Connection — "What resources + scenarios does X depend on?" | `scenario-dependency-analyzer` (`.scenarios` / `.resources`) | NOW (active) | DERIVED | Both leaves live + federated. `.scenarios` from the interface graph (depends-on/used-by); `.resources` from a fleet `service.json` scan. |
| 6 | Test suite · Verification / State — "How is X tested; which phases are red and why?" | `test-genie` (`health` / `fleet status`) | IN-REACH | DERIVED | Read existing RPCs, attest. |

### G2 — Within-scenario

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 7 | Zones · Anatomy — "What are X's zones (handlers / domain / substrate / composition root)?" | `architecture-cartographer` (`GetZoneMap`) | NOW | DERIVED / VALIDATED | Zone Map (`ARCHITECTURE.md`) cross-checked against the import graph — live via `GetZoneMap`. A dedicated `architecture-cartographer.zones` search leaf is a follow-on sibling (same engine/recipe as domain-map). |
| 8 | Shared substrate · Inventory — "What non-domain/shared packages exist (server, module, clock, httputil…)?" | `code-facts` / code-graph | IN-REACH | DERIVED | No provider yet; derive from code-graph + the domain-vs-substrate zone rule. |
| 9 | ⭐ Shared substrate · Connection — "How do non-domain packages connect to domains? Who wires them (composition root / `Deps`)?" | _(none)_ → cartographer / code-graph | MISSING | DERIVED | Needs a provider; approach: `Deps`-struct + import-edge analysis. |
| 10 | Seams · Inventory + Connection — "Where are the ambient seams (clock/http/env/logger); who wires prod vs test?" | `SEAMS.md` + static grep | IN-REACH | DECLARED_UNVERIFIED → DERIVED | `SEAMS.md` is declared; cross-check vs grep for `time.Now`/`os.Getenv`/`http.DefaultClient`. |
| 11 | Proto contracts · Inventory + Conformance — "What proto services/messages exist; do they validate + are they adopted?" | `proto-health` / `code-facts` (`PROTO_ADOPTION`) | IN-REACH | DERIVED | Read proto-health, attest. |
| 12 | CLI substrate · Anatomy — "How is X's CLI composed (cli-core `ScenarioApp`, command groups)?" | `cli-health` / `code-facts` | IN-REACH | DERIVED | Partial today; extend. |
| 13 | UI substrate · Inventory + Anatomy — "What shared UI substrate exists (components/hooks/contexts/client layer)?" | `ui-health` | IN-REACH (surfaces/widgets NOW) | DERIVED | `surfaces` + `widgets` live; substrate anatomy needs extension. |
| 14 | ⭐ Testing entities · Anatomy — "How is X's testing set up (phases, test data, mocks/fakes, fixtures)?" | `test-genie` (`health`) + bas `registry.json` + `SEAMS.md` | IN-REACH | DERIVED | Compose test-genie health + bas registry + seams. |
| 15 | Coupling graph · Connection — "What's X's internal import/coupling graph; any cycles or layering violations?" | `architecture-cartographer` (`BoundaryHealth` + cycle/layering detectors) | NOW | DERIVED | `BoundaryHealth` + the cycle/layering detectors are live. A dedicated `architecture-cartographer.coupling` search leaf is a follow-on sibling. |

### G3 — Domain (single slice)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 16 | ⭐ Slice · Anatomy — "How is domain X implemented end-to-end (proto→handler→internal→cli→ui)?" | `architecture-cartographer` (`GetSlice`) | NOW (PARTIAL) | DERIVED / PARTIAL | `GetSlice` is live but self-attests `DERIVED/PARTIAL` (UI rung + cross-rung wired edges incomplete); hardening is a separate plan. |
| 17 | ⭐ Feature · Flow — "Visualize how feature X works at runtime (control/data flow)." | code-graph (call/ref graph) | MISSING | PARTIAL (reconstructed) | **Call-graph-blocked**: needs a call/ref graph that exists on neither architecture-cartographer nor code-graph (net-new substrate, separate plan) — not merely provider-missing. Flow is reconstructed → lower basis; render as sequence/mermaid. |
| 18 | Proto · Conformance — "Does domain X's code match its proto contract?" | `proto-health` / cartographer | IN-REACH | DERIVED | Proto adoption + endpoint proofs. |
| 19 | Invariants · Verification — "What invariants does X enforce, and how?" | `INVARIANTS.md` + `// INVARIANT` tags + proto validation | IN-REACH | DECLARED_UNVERIFIED → DERIVED | Basis depends on the enforcement mechanism (type/db/test/runtime). |
| 20 | Archetype · Inventory / Anatomy — "What's domain X's archetype (one of the canonical fleet vocabulary: reporting / service / mutation / classification / orchestration / scoring / query)?" | `architecture-cartographer` (`InferArchetype`) | NOW | HEURISTIC → DECLARED | Live via `InferArchetype` (also folded into the domain-map provider's corpus as a metadata filter). Inferred from code shape and converged with the declared DOMAINS.md value; confidence from signal specificity; declared-vs-inferred disagreement reported as drift, never silently overridden. |
| 21 | Persistence · Anatomy — "What's X's storage/persistence pattern (schema, migrations, seams)?" | `storage-manager` + code-graph | IN-REACH | DERIVED | Read storage-manager, attest. |
| 22 | Intent · Provenance — "Why does domain X exist / when should it change?" | `business-health.intent` (PRD purpose/OTs/requirements corpus) | ACTIVE (pointer-only) | DERIVED | Pointer-only by contract: hits are anchors into `PRD.md`/`requirements/`, never synthesized rationale (business-health.intent, 2026-07-02). |

### G4 — Symbol / file

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 28 | Symbol · Pointer + Connection — "Where is this function/type defined and where is it used?" | `code-reference.code` (gap stub) | IN-REACH | DERIVED | code-graph over Go/TS. |
| 29 | Call graph · Connection — "What calls/references symbol X (the graph around it)?" | `code-reference.code` / code-graph | IN-REACH | DERIVED | |
| 30 | File · Inventory — "What files/declarations are in package Y (and of what kind)?" | `code-facts` (`SYMBOLS`) / code-graph | IN-REACH | DERIVED | |

### G5 — Ecosystem (inter-scenario)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 31 | Scenario graph · Connection — "What does X depend on; where is X used (reverse)?" | `scenario-dependency-analyzer.scenarios` | NOW (active) | DERIVED | Live federated leaf over the interface graph: per scenario, depends-on (forward edges) + used-by (reverse edges). `SearchInterfaceGraph`. |
| 32 | ⭐ Dependency rationale · Provenance — "Why does this cross-scenario dependency exist?" | `scenario-dependency-analyzer` (governance rationale) | MISSING (query) | DECLARED_UNVERIFIED | Rationale stored in governance records, not queryable. |
| 33 | Resource usage · Connection — "Which scenarios use resource Y?" | `scenario-dependency-analyzer.resources` | NOW (active) | DERIVED | Live federated leaf: a fleet `service.json` scan inverted to resource → consuming-scenarios. `SearchResourceUsage`. |
| 34 | Capability map · Inventory — "Which scenario provides capability X / where should I build Y?" | `business-health.intent` (fleet PRD/requirements corpus) + `prompt-manager` (skills/actions) | ACTIVE | DERIVED | Semantic capability lookup over every scenario's stated intent (business-health.intent, 2026-07-02); prompt-manager still owns the skills/actions angle. |
| 35 | Package deps · Connection — "What approved packages does the fleet use; security gaps?" | `scenario-dependency-analyzer.dependencies` | NOW (active) | DERIVED | Already live. |
| 36 | Federation · Inventory / State — "Which providers are registered; what's federation health?" | `search-hub` (`providers list` / `Status`) | NOW | DERIVED | Already live; could attest. |

## Known Gaps & Approaches

The `MISSING` cells, grouped by tier:

- **Scenario-internal** — #9 non-domain↔domain wiring (composition-root + import-edge provider); #16/#17 domain slice anatomy + feature flow (slice-walker; Flow is inherently `PARTIAL`); #22 intent/"why" (now `DERIVED` via `business-health.intent` — still pointer-only by contract: anchors into `PRD.md`/comments, never synthesized).
- **Project (G0)** — #24 control-plane CLI *structure* (a `cmd/vrooli` structure registry — distinct from cli-health's command search); #26 resource fleet aggregation (no fleet provider over per-`resource.json`).
- **Ecosystem (G5)** — #32 dependency-rationale query (governance stores rationale; no query-by-rationale surface).

Cross-cutting/global gaps that belong to no single cell are tracked in the `meta-optimization-manager`
`PROBLEMS.md`, not here.

## Sources Of Truth

- `search-hub providers list --state capability_gap` — the seed denominator this grid extends.
- `packages/proto/schemas/common/v1/code_graph.proto` — node/edge vocabulary behind `DERIVED` cells.
- `code-facts` `FACT_FAMILY_*` (surfaces, file_domain, proto adoption, endpoint/CLI/widget proofs).
- `architecture-cartographer` detectors + `BoundaryHealth` + `ExtractDomains`.
- `packages/proto/schemas/search-hub/v1/routing/routing.proto` — `MeasureHit`, the precedent carrier for an attested answer on `SearchHit`.

## Cross-References

- `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — canonical model, contract, and legend.
- `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md` — sibling projection denominators.
- `../concepts/ARCHITECTURE.md`, `../concepts/DOMAINS.md` — search-hub's own architecture (a live example target for these questions).
