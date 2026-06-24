# Answer Space — Architectural Question Coverage

> **Model & terminology** — the attestation contract (basis × sufficiency), the
> entity × archetype × aspect model, the denominator/numerator split, and the Status/Basis
> legend are defined once in the canonical model doc:
> `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` _(planned — create before this
> space ships)_. This document does not re-explain them.

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
| Sibling spaces | `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md` _(planned)_ |
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
| 2 | Domains · Inventory + Anatomy — "What domains does X have and what does each own?" | `architecture-cartographer.domain-map` | IN-REACH (gap stub) | DERIVED / VALIDATED | **First-slice provider.** Reads `file_domain` + `ExtractDomains` + `DOMAINS.md`. |
| 3 | Doc-set · Conformance — "Do X's docs match its code? Where's the drift?" | `architecture-cartographer` (drift detectors) | IN-REACH | VALIDATED / CONTRADICTED | Cartographer's core job; expose drift findings as attested answers. |
| 4 | Requirements/maturity · State — "What's X's maturity rung + requirements status?" | `completeness-scoring` (`GetScore`) | IN-REACH | DERIVED | Read `GetScore`, wrap in attestation. |
| 5 | Dependencies · Connection — "What resources + scenarios does X depend on?" | `scenario-dependency-analyzer` (`.scenarios` / `.resources`) | IN-REACH (gap stubs) | DERIVED | Interface graph + `service.json`. |
| 6 | Test suite · Verification / State — "How is X tested; which phases are red and why?" | `test-genie` (`health` / `fleet status`) | IN-REACH | DERIVED | Read existing RPCs, attest. |

### G2 — Within-scenario

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 7 | Zones · Anatomy — "What are X's zones (handlers / domain / substrate / composition root)?" | `architecture-cartographer` | IN-REACH | DERIVED / VALIDATED | Zone Map (`ARCHITECTURE.md`) cross-checked against the import graph. |
| 8 | Shared substrate · Inventory — "What non-domain/shared packages exist (server, module, clock, httputil…)?" | `code-facts` / code-graph | IN-REACH | DERIVED | No provider yet; derive from code-graph + the domain-vs-substrate zone rule. |
| 9 | ⭐ Shared substrate · Connection — "How do non-domain packages connect to domains? Who wires them (composition root / `Deps`)?" | _(none)_ → cartographer / code-graph | MISSING | DERIVED | Needs a provider; approach: `Deps`-struct + import-edge analysis. |
| 10 | Seams · Inventory + Connection — "Where are the ambient seams (clock/http/env/logger); who wires prod vs test?" | `SEAMS.md` + static grep | IN-REACH | DECLARED_UNVERIFIED → DERIVED | `SEAMS.md` is declared; cross-check vs grep for `time.Now`/`os.Getenv`/`http.DefaultClient`. |
| 11 | Proto contracts · Inventory + Conformance — "What proto services/messages exist; do they validate + are they adopted?" | `proto-health` / `code-facts` (`PROTO_ADOPTION`) | IN-REACH | DERIVED | Read proto-health, attest. |
| 12 | CLI substrate · Anatomy — "How is X's CLI composed (cli-core `ScenarioApp`, command groups)?" | `cli-health` / `code-facts` | IN-REACH | DERIVED | Partial today; extend. |
| 13 | UI substrate · Inventory + Anatomy — "What shared UI substrate exists (components/hooks/contexts/client layer)?" | `ui-health` | IN-REACH (surfaces/widgets NOW) | DERIVED | `surfaces` + `widgets` live; substrate anatomy needs extension. |
| 14 | ⭐ Testing entities · Anatomy — "How is X's testing set up (phases, test data, mocks/fakes, fixtures)?" | `test-genie` (`health`) + bas `registry.json` + `SEAMS.md` | IN-REACH | DERIVED | Compose test-genie health + bas registry + seams. |
| 15 | Coupling graph · Connection — "What's X's internal import/coupling graph; any cycles or layering violations?" | `architecture-cartographer` (`BoundaryHealth` + cycle/layering detectors) | IN-REACH | DERIVED | Expose `BoundaryHealth` as an attested answer. |

### G3 — Domain (single slice)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 16 | ⭐ Slice · Anatomy — "How is domain X implemented end-to-end (proto→handler→internal→cli→ui)?" | `architecture-cartographer` / code-graph | MISSING | DERIVED / PARTIAL | Needs a slice-walker provider over the vertical slice. |
| 17 | ⭐ Feature · Flow — "Visualize how feature X works at runtime (control/data flow)." | code-graph (call/ref graph) | MISSING | PARTIAL (reconstructed) | Flow is reconstructed → lower basis; render as sequence/mermaid. |
| 18 | Proto · Conformance — "Does domain X's code match its proto contract?" | `proto-health` / cartographer | IN-REACH | DERIVED | Proto adoption + endpoint proofs. |
| 19 | Invariants · Verification — "What invariants does X enforce, and how?" | `INVARIANTS.md` + `// INVARIANT` tags + proto validation | IN-REACH | DECLARED_UNVERIFIED → DERIVED | Basis depends on the enforcement mechanism (type/db/test/runtime). |
| 20 | Archetype · Inventory / Anatomy — "What's domain X's archetype (CRUD / temporal / integration / orchestration)?" | `architecture-cartographer` | IN-REACH | HEURISTIC → DECLARED | Inferred from code shape; mark confidence honestly. |
| 21 | Persistence · Anatomy — "What's X's storage/persistence pattern (schema, migrations, seams)?" | `storage-health` + code-graph | IN-REACH | DERIVED | Read storage-health, attest. |
| 22 | Intent · Provenance — "Why does domain X exist / when should it change?" | `PRD.md` + code comments | MISSING (pointer-only) | ABSENT | Judgment; contract forces pointer-only to PRD/comments. |

### G4 — Symbol / file

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 28 | Symbol · Pointer + Connection — "Where is this function/type defined and where is it used?" | `code-reference.code` (gap stub) | IN-REACH | DERIVED | code-graph over Go/TS. |
| 29 | Call graph · Connection — "What calls/references symbol X (the graph around it)?" | `code-reference.code` / code-graph | IN-REACH | DERIVED | |
| 30 | File · Inventory — "What files/declarations are in package Y (and of what kind)?" | `code-facts` (`SYMBOLS`) / code-graph | IN-REACH | DERIVED | |

### G5 — Ecosystem (inter-scenario)

| # | Question (Entity · Archetype) | Owner (provider) | Status | Basis | Notes / approach |
|---|---|---|---|---|---|
| 31 | Scenario graph · Connection — "What does X depend on; where is X used (reverse)?" | `scenario-dependency-analyzer.scenarios` (gap stub) | IN-REACH | DERIVED | Interface graph; reverse/ownership query thin. |
| 32 | ⭐ Dependency rationale · Provenance — "Why does this cross-scenario dependency exist?" | `scenario-dependency-analyzer` (governance rationale) | MISSING (query) | DECLARED_UNVERIFIED | Rationale stored in governance records, not queryable. |
| 33 | Resource usage · Connection — "Which scenarios use resource Y?" | `scenario-dependency-analyzer.resources` (gap stub) | IN-REACH | DERIVED | |
| 34 | Capability map · Inventory — "Which scenario provides capability X / where should I build Y?" | `prompt-manager` (skills/actions) + ecosystem-fit | IN-REACH (partial) | DECLARED_UNVERIFIED | Capability map is fuzzy; partly judgment. |
| 35 | Package deps · Connection — "What approved packages does the fleet use; security gaps?" | `scenario-dependency-analyzer.dependencies` | NOW (active) | DERIVED | Already live. |
| 36 | Federation · Inventory / State — "Which providers are registered; what's federation health?" | `search-hub` (`providers list` / `Status`) | NOW | DERIVED | Already live; could attest. |

## Known Gaps & Approaches

The `MISSING` cells, grouped by tier:

- **Scenario-internal** — #9 non-domain↔domain wiring (composition-root + import-edge provider); #16/#17 domain slice anatomy + feature flow (slice-walker; Flow is inherently `PARTIAL`); #22 intent/"why" (by design `ABSENT` → pointer to `PRD.md`/comments, never synthesized).
- **Project (G0)** — #24 control-plane CLI *structure* (a `cmd/vrooli` structure registry — distinct from cli-health's command search); #26 resource fleet aggregation (no fleet provider over per-`resource.json`).
- **Ecosystem (G5)** — #32 dependency-rationale query (governance stores rationale; no query-by-rationale surface).

Cross-cutting/global gaps that belong to no single cell are tracked in the `meta-optimization-manager`
`PROBLEMS.md` _(planned)_, not here.

## Sources Of Truth

- `search-hub providers list --state capability_gap` — the seed denominator this grid extends.
- `packages/proto/schemas/common/v1/code_graph.proto` — node/edge vocabulary behind `DERIVED` cells.
- `code-facts` `FACT_FAMILY_*` (surfaces, file_domain, proto adoption, endpoint/CLI/widget proofs).
- `architecture-cartographer` detectors + `BoundaryHealth` + `ExtractDomains`.
- `packages/proto/schemas/search-hub/v1/routing/routing.proto` — `MeasureHit`, the precedent carrier for an attested answer on `SearchHit`.

## Cross-References

- `meta-optimization-manager/docs/concepts/COVERAGE-MODEL.md` — canonical model, contract, and legend _(planned)_.
- `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md` — sibling projection denominators _(planned)_.
- `../concepts/ARCHITECTURE.md`, `../concepts/DOMAINS.md` — search-hub's own architecture (a live example target for these questions).
