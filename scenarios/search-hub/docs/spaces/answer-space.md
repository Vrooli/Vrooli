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
| 23 | Repo contract · Conformance — "Does the project conform to its repo contract; what's the layout?" | `contract-registry.contracts` | IN-REACH (gap stub) | DERIVED / VALIDATED | The registered contract leaf carries this project-wide question. |
| 24 | ⭐ Control-plane CLI · Anatomy — "How is the top-level `vrooli` CLI structured/composed?" | _(none)_ | MISSING | DERIVED | Not "find a command" (cli-health does that) — the *structure* of `cmd/vrooli`. Needs a control-plane registry. |
| 25 | Shared packages · Inventory + Connection — "What `packages/*` exist and who consumes each?" | `code-reference.code` | IN-REACH | DERIVED | The code-reference provider is the declared substrate for package inventory and usage edges; it is currently a capability gap. |
| 26 | Resource fleet · Inventory + Anatomy — "What resources exist (ports/health/driver) and how do they fit together?" | _(none)_ | MISSING | DECLARED_UNVERIFIED | Per-`resource.json` exists; no fleet aggregator/provider. |
| 27 | Contracts/schemas · Inventory — "What contracts/schemas govern the project and what do they enforce?" | `contract-registry.contracts` | IN-REACH (gap stub) | DERIVED | The registered contract leaf owns the project-wide contract inventory. |

### G1 — Scenario (whole)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 1 | Surfaces · Inventory — "What surfaces (API/CLI/UI) does X expose?" | `ui-health.surfaces` + `cli-health.commands` | NOW (UI, CLI) / IN-REACH (API) | DERIVED | UI + CLI live; API surface evidence remains a documented limitation, not an owner token. |
| 2 | Domains · Inventory + Anatomy — "What domains does X have and what does each own?" | `architecture-cartographer.domain-map` | NOW | DERIVED / VALIDATED | **First-slice provider, live.** Semantic search over the derived domain map (`ExtractDomains` responsibility/purpose/glossary) — term-agnostic: "how does authoring work in plan-manager" → `plan-manager/authoring`. |
| 3 | Doc-set · Conformance — "Do X's docs match its code? Where's the drift?" | `architecture-cartographer.domain-map` | IN-REACH | VALIDATED / CONTRADICTED | Cartographer's registered domain-map leaf is the current architectural substrate; drift detectors remain a documented limitation. |
| 4 | Requirements/maturity · State — "What's X's maturity rung + requirements status?" | _(none)_ | MISSING | DERIVED | The completeness scorer is not a registered search leaf; this row remains an explicit capability gap. |
| 5 | Dependencies · Connection — "What resources + scenarios does X depend on?" | `scenario-dependency-analyzer.scenarios` + `scenario-dependency-analyzer.resources` | NOW (active) | DERIVED | Both registered leaves are live and federated. |
| 6 | Test suite · Verification / State — "How is X tested; which phases are red and why?" | `workflow-health.tests` | IN-REACH | DERIVED | The registered workflow-test corpus is the current searchable test substrate; fleet health remains an external signal. |

### G2 — Within-scenario

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 7 | Zones · Anatomy — "What are X's zones (handlers / domain / substrate / composition root)?" | `architecture-cartographer.domain-map` | NOW | DERIVED / VALIDATED | The registered domain-map leaf carries the current architectural map; a dedicated zones leaf remains a follow-on. |
| 8 | Shared substrate · Inventory — "What non-domain/shared packages exist (server, module, clock, httputil…)?" | `code-reference.code` | IN-REACH | DERIVED | The registered code-reference leaf is the current code-graph substrate. |
| 9 | ⭐ Shared substrate · Connection — "How do non-domain packages connect to domains? Who wires them (composition root / `Deps`)?" | _(none)_ | MISSING | DERIVED | Needs a provider; approach: `Deps`-struct + import-edge analysis. |
| 10 | Seams · Inventory + Connection — "Where are the ambient seams (clock/http/env/logger); who wires prod vs test?" | `code-reference.code` | IN-REACH | DECLARED_UNVERIFIED → DERIVED | Static seam evidence remains the current approach. |
| 11 | Proto contracts · Inventory + Conformance — "What proto services/messages exist; do they validate + are they adopted?" | `code-reference.code` | IN-REACH | DERIVED | The registered code-reference leaf carries the current contract inventory. |
| 12 | CLI substrate · Anatomy — "How is X's CLI composed (cli-core `ScenarioApp`, command groups)?" | `cli-health.commands` | IN-REACH | DERIVED | The live CLI command provider is the closest serving substrate; composition anatomy remains partial. |
| 13 | UI substrate · Inventory + Anatomy — "What shared UI substrate exists (components/hooks/contexts/client layer)?" | `ui-health.surfaces` | IN-REACH (surfaces/widgets NOW) | DERIVED | The surfaces leaf is the serving provider; shared substrate anatomy needs extension. |
| 14 | ⭐ Testing entities · Anatomy — "How is X's testing set up (phases, test data, mocks/fakes, fixtures)?" | `workflow-health.tests` | IN-REACH | DERIVED | The registered workflow-test corpus is the current searchable test substrate. |
| 15 | Coupling graph · Connection — "What's X's internal import/coupling graph; any cycles or layering violations?" | `architecture-cartographer.domain-map` | NOW | DERIVED | The registered domain-map leaf is the current architectural substrate; a dedicated coupling leaf is a follow-on. |

### G3 — Domain (single slice)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 16 | ⭐ Slice · Anatomy — "How is domain X implemented end-to-end (proto→handler→internal→cli→ui)?" | `architecture-cartographer.domain-map` | NOW (PARTIAL) | DERIVED / PARTIAL | The registered domain-map leaf carries partial architecture evidence; UI and cross-rung edges remain incomplete. |
| 17 | ⭐ Feature · Flow — "Visualize how feature X works at runtime (control/data flow)." | `code-reference.code` | MISSING | PARTIAL (reconstructed) | The registered code-reference leaf lacks a complete call graph; flow remains reconstructed and lower-basis. |
| 18 | Proto · Conformance — "Does domain X's code match its proto contract?" | `code-reference.code` | IN-REACH | DERIVED | The registered code-reference leaf carries the current contract substrate. |
| 19 | Invariants · Verification — "What invariants does X enforce, and how?" | `code-reference.code` | IN-REACH | DECLARED_UNVERIFIED → DERIVED | Basis depends on the enforcement mechanism (type/db/test/runtime). |
| 20 | Archetype · Inventory / Anatomy — "What's domain X's archetype (one of the canonical fleet vocabulary: reporting / service / mutation / classification / orchestration / scoring / query)?" | `architecture-cartographer.domain-map` | NOW | HEURISTIC → DECLARED | The registered domain-map leaf carries the inferred/declared archetype evidence. |
| 21 | Persistence · Anatomy — "What's X's storage/persistence pattern (schema, migrations, seams)?" | `code-reference.code` | IN-REACH | DERIVED | The registered code-reference leaf carries the current persistence substrate. |
| 22 | Intent · Provenance — "Why does domain X exist / when should it change?" | `business-health.intent` (PRD purpose/OTs/requirements corpus) | IN-REACH | DERIVED | Pointer-only by contract: hits are anchors into `PRD.md`/`requirements/`, never synthesized rationale (business-health.intent, 2026-07-02). The provider is registered, but this pointer-only contract is not yet a fully attested NOW capability. |

### G4 — Symbol / file

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 28 | Symbol · Pointer + Connection — "Where is this function/type defined and where is it used?" | `code-reference.code` | IN-REACH | DERIVED | The code-reference leaf is the registered code-graph substrate. |
| 29 | Call graph · Connection — "What calls/references symbol X (the graph around it)?" | `code-reference.code` | IN-REACH | DERIVED | The code-reference leaf is the declared call/reference graph substrate and remains a capability gap. |
| 30 | File · Inventory — "What files/declarations are in package Y (and of what kind)?" | `code-reference.code` | IN-REACH | DERIVED | The registered code-reference leaf is the current code-graph substrate. |

### G5 — Ecosystem (inter-scenario)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 31 | Scenario graph · Connection — "What does X depend on; where is X used (reverse)?" | `scenario-dependency-analyzer.scenarios` | NOW (active) | DERIVED | Live federated leaf over the interface graph: per scenario, depends-on (forward edges) + used-by (reverse edges). `SearchInterfaceGraph`. |
| 32 | ⭐ Dependency rationale · Provenance — "Why does this cross-scenario dependency exist?" | _(none)_ | MISSING (query) | DECLARED_UNVERIFIED | Rationale stored in governance records, not queryable. |
| 33 | Resource usage · Connection — "Which scenarios use resource Y?" | `scenario-dependency-analyzer.resources` | NOW (active) | DERIVED | Live federated leaf: a fleet `service.json` scan inverted to resource → consuming-scenarios. `SearchResourceUsage`. |
| 34 | Capability map · Inventory — "Which scenario provides capability X / where should I build Y?" | `business-health.intent` + `prompt-manager.skill` | NOW | DERIVED | Semantic capability lookup over intent plus the registered skills corpus. |
| 35 | Package deps · Connection — "What approved packages does the fleet use; security gaps?" | `scenario-dependency-analyzer.dependencies` | NOW (active) | DERIVED | Already live. |
| 36 | Federation · Inventory / State — "Which providers are registered; what's federation health?" | `measures-health.measures` | NOW | DERIVED | The registered measures leaf carries fleet telemetry; Search Hub's registry/status remains the control-plane source. |

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
