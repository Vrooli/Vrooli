---
name: "architecture-lenses"
description: "Ten structural audit lenses for a scenario: boundary, seam, invariant, screaming architecture, domain clarity, decision boundary, change axis, experience architecture, navigation, control surface. Pick the lens the finding belongs to."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","architecture"]
  tags: ["skill", "architecture"]
  icon: "box"
  status: "active"
  targetDimensions: ["structure","cycles","docs"]
  defaultScope: "work-scope"
  revision: 1
  createdAt: "2026-11-20T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Steer focus: Architecture Lenses

**Scope mode: `architecture`.** Hold to that row of `work-scope` for this session.

One skill for the structural audits of `scenarios/{{TARGET}}/`. Each lens is a question asked of the scenario; pick the lens a finding belongs to, hand structural findings to the lens that owns them, and never patch one lens's problem with another's change. Read first when present: `docs/concepts/ARCHITECTURE.md` (surfaces, domain map, zone ownership), `docs/internal/SEAMS.md`, `INVARIANTS.md`, `PROBLEMS.md`. Read and update docs through `knowledge-observatory-tools`. Findings route to those existing docs; no standalone `*_AUDIT.md`. An assessment request permits findings, not code moves; moves need an implementation grant (`docs/agent-system/SCENARIO_DEVELOPMENT.md`). Validation scope follows `docs/TESTING.md`. Why each lens applies, when it backfires and what the qa-contrarian challenges is in `docs/scenario-qa/methods/audit/<lens>.md` where one exists. The react-vite template's `notes` domain (`templates/scenarios/react-vite/api/internal/notes/`) is the worked example and is deleted on generation; `<domain>` below is a placeholder.

For every lens, the maturity ladder is the same shape: 0 implicit, 1 inventoried in the owning doc, 2 declared or mapped, 3 code matches the declaration, 4 tested, 5 drift-gated by a test or validator. Use it to pick the next concrete move, not as a score to inflate; a level whose artifact does not exist is not claimed. Assess each surface (`api/`, `ui/`, `cli/`) separately.

| Finding | Lens |
|---|---|
| Code lives in the wrong folder, package or layer; a layer imports what it should not | Boundary |
| A dependency (time, network, env, logger, repo, integration) cannot be substituted in tests | Seam |
| A rule the code relies on is unstated, unenforced or untested | Invariant |
| Structure is by tech bucket, not capability; product capabilities are hard to find | Screaming architecture |
| Two names for one concept, one name for two, or purpose not visible | Domain clarity |
| An important choice is a buried conditional | Decision boundary |
| One conceptual change forces edits across many places | Change axis |
| Behavior-shaping thresholds, limits, modes or timeouts are hard-coded, or config is a jungle | Control surface |
| Users hunt, remember or drill repeatedly to finish a job | Experience architecture |
| Route, label, destination, reachability or deep-link behavior is implicit | Navigation |
| Lifecycle, retry, cancellation, stale completion | `resilience-lenses` (temporal) |
| Local naming, function shape, comments | `cognitive-load-reduction` |
| Dead code and stale config | `code-cleanup` |

---

### 1. Boundary of responsibility

**Where code lives.** Two agents from the same docs should land code in the same place. Target is the react-vite shape: `api/handlers/<domain>/` translates transport (Connect service, `Module` constructor, `EndpointDescriptor`s); `api/internal/<domain>/` owns transport-free domain logic (types, service, repository interface, schema, workflows, `<domain>_sqlite.go` persistence adapter, `mocks/`); `api/internal/{clock,database,httpc,httpx,middleware,module,server}/` hold business-vocabulary-free substrate; `api/main.go` and `internal/server/` are the composition root; `api/cmd/<tool>/` holds operator binaries.

Zone is decided by what would break if the file disappeared, not by what it imports. Translates a wire request to a typed call: transport edge. Encodes rules that would survive a transport change: domain core. Speaks a driver dialect (SQL, S3, gRPC): a persistence adapter inside the owning domain. No product vocabulary and used by two or more domains: substrate. Constructs concretes and passes them into `Module` or `Deps`: composition root. Most violations are one mis-zoned file; the move is mechanical (extract the symbol, place it, fix imports).

Invariants: `internal/<domain>/` imports no transport package and no sibling domain (coordination happens in `handlers/` or a deliberate orchestration domain); persistence adapters live inside the owning domain, not in a global `internal/persistence/`; `main.go` alone constructs concrete substrate (`clock.System{}`, `&http.Client{}`, `database.Open`), everything else receives it through `server.Deps`; `internal/testutil/` and `<domain>/mocks/` are never imported by production code (`no_prod_import_test.go` pattern).

Ladder: 0 SQL in handlers or domain importing transport (`rg "database/sql|sqlx|gorm" handlers/`, `rg '"connectrpc.com/connect"' internal/<domain>/`); 1 `ARCHITECTURE.md` Zone Map names every directory under `api/` one to one; 2 folder shape matches the template and handlers hold no SQL; 3 domain layer is transport-free, encoded as an import-graph Go test; 4 no `time.Now()`, `os.Getenv()`, `http.DefaultClient` or `log.Default()` inside domain packages; 5 a reconciliation test fails on a new top-level package without a Zone Map entry, a domain importing another domain, or handlers importing persistence drivers.

Red flags: `db.Query` or `sql.Open` in a handler; a domain file importing `connect`, `net/http` (outside `httpc`) or `gorilla/mux`; ambient time, env or default client in a domain; `internal/utils`, `common` or `helpers` buckets; adapters under `internal/persistence/<domain>/`; production importing testutil or mocks; undeclared new top-level packages.

### 2. Seam discovery

A seam is a consumer-owned interface or injected function with explicit production wiring and a controllable test substitution. The interface is declared in the package that consumes it. Need a seam when a call crosses a process boundary (network, disk, syscall), reads a global (env, default client, default logger, wall clock), or varies per environment or tenant; none for a pure stdlib transform; extend an existing seam with a method rather than adding a parallel one.

Archetypes: ambient substrate (single-method interface in `internal/<substrate>/`, struct in the same package, fake in `internal/testutil/mocks/`); outbound integration (narrow interface in the domain over an `httpc.Doer` adapter, fake in `<domain>/mocks/`); persistence (`Repository` in `internal/<domain>/repository.go`, `_sqlite.go` or `_postgres.go`, mock beside it); process-out side effect (`EventPublisher`, `BlobStore`); policy or decision (`Authorizer`, `RateLimiter`, programmable fake); clock-shaped derivatives (extend `clock.Clock` with tickers, sleeps, jitter). Candidate ambient seams are wall clock (`clock.Clock`), outbound HTTP (`httpc.Doer`), env reader and logger; add one only when substitution or configuration needs it, and prefer an existing narrow configuration or logger boundary to inventing `envx` or `logx` packages. Shared fakes carry a `var _ <Seam> = ...` assertion. `SEAMS.md` registers declared interfaces (a registry test can reconcile them; verify it exists before claiming so).

Red flags: tests that `time.Sleep` for a clock branch or set env vars to drive behavior; a `*http.Client` built inside a domain function; an interface mirroring `*sql.DB` verbatim; overlapping seams (`Clock` and `TimeProvider`); a fake without an assertion; `SEAMS.md` rows pointing at deleted files or `// seam:` interfaces missing from it; a "test mode" boolean in production code.

### 3. Invariants

An invariant is a condition that must always hold; bugs are violations. Discovery is archaeology: only rules the code currently relies on count, and true rules are stabilized, never accidental behavior or workarounds. Local invariants (one function, file or module) are declared and enforced in code with a `// INVARIANT: <camelCaseName>` tag at the enforcement site (the line you would bypass to violate it) and need no registry entry. Registry-worthy ones (multiple domains depend on it, layers must agree, system-wide impact such as security, billing or corruption, or the enforcement is itself cross-cutting) also get a row in `docs/internal/INVARIANTS.md` (name, statement, mechanism, `path:file:line` anchor, enforcing test). Each tag has a test whose name or comment references it; where the build is the enforcement (codegen validator, type encoding) the build is the evidence.

Triage every discovered rule into exactly one resolution: **Enforce** (tag, test, and registry row if registry-worthy), **Soften** (change the code so it stops depending on the rule; record under "Softened-out rules" in `PROBLEMS.md`, not in the registry), or **Accept** (enforcement costs more than impact times likelihood; Gaps section with rationale and revisit trigger).

Mechanism by archetype, leftmost that can carry the rule: shape or nullability to the type system then `protovalidate`; range or value to `protovalidate` then a boundary guard; ordering to a state machine in the owning domain (temporal lens); referential integrity to a DB foreign key; identity to opaque ID type plus uniqueness constraint; cross-aggregate sums to a DB or transactional check; cross-surface parity to a codegen-time validator (`RESTReason` with `validateTransport` is the worked example); authorization to an interceptor. Prefer types over declarative annotations over runtime guards over tests. Ladder: 0 implicit, 1 tagged, 2 tag on the real guard, 3 violation test, 4 mechanism named and real, 5 drift-gated (tag and registry parity checked by a test). Discovery signals: repeated guards, branches that panic on other shapes, "must, always, never, assume" comments, error types naming a violation, existing constraints and `protovalidate` annotations, PRD "must, never, exactly, at most". Density sanity: CRUD 2 to 4 critical, orchestration 6 to 12, security or billing 10+; zero in a high-risk domain is a gap, thirty in CRUD is conflation. A leftover `ASSUMPTIONS.md` is triaged the same way and deleted when empty.

### 4. Screaming architecture

Make structure express the product's capabilities. Separate surfaces (API: logic, state, integrations; UI: feature-shaped presentation; CLI: thin wrapper over the API; contracts: canonical wire shape), domains (product vocabulary: if deleting the capability should delete the code, it belongs there; a helper for one domain's tests stays local) and substrate (generic, no product vocabulary, used by unrelated domains). In Go, `internal/<domain>/` is domain and `internal/<substrate>/` is infrastructure; "internal" does not mean infrastructure.

Choose shape by behavior: CRUD or entity (repository, schema, service, handler, list UI, CLI list/create/get), temporal workflow (matrix and trace tests, invariants), binary or blob (blob store seam, multipart edge), integration or client (client seam, retry, webhook, contract tests), orchestration (planner, policy seams, checkpoints), reporting or query, policy or rules (pure rule functions, table tests), configuration or settings. Pick a primary archetype, then secondary traits. Ladder: 0 opaque structure; 1 `ARCHITECTURE.md` names surfaces, capabilities, flows; 2 each capability has an owner domain and source paths; 3 domain-owned implementation (logic, schema, handlers, UI feature, CLI, tests and fakes with the capability); 4 substrate is vocabulary-free and seams are registered; 5 validation catches stale manifests, misplaced docs, forbidden generic buckets, production imports of test utilities. Structural detection is delegated to `architecture-cartographer` (finding identity in `docs/reference/architecture-validation-responsibilities.md`); a large refactor can be preserved across re-audits by `scenario-improvement-campaign` when the caller authorizes that engagement.

### 5. Domain clarity

The conceptual model: the smallest set of concepts, each named once, with visible purpose, answerable from the code and `docs/concepts/` alone. Three lenses. **Concept compression**: collapse near-duplicate types and parallel flows for one journey to one canonical concept, recording the merge in `ARCHITECTURE.md`, without merging genuinely distinct concepts the PRD keeps apart. **Vocabulary unification**: one primary term per concept across code, tests and UI; split an overloaded name; align with the PRD's language; record canonical terms and old-to-new mappings in `GLOSSARY.md`. **Intent**: names answer what it is for and when it changes; add a why-comment only for a non-obvious decision or invariant. Ladder: 0 untracked; 1 documented and verified against code; 2 compressed (no parallel near-duplicates); 3 unified vocabulary (a grep for each retired synonym returns only historical or adapter mentions); 4 intent explicit. Prefer localized safe renames to repo-wide ones, and remove code only when provably unused or superseded. Reconciling `job`, `run` and `task` is this lens; renaming a loop variable is `cognitive-load-reduction`.

### 6. Decision boundary

A decision is any place the system chooses between alternatives (branches, thresholds, modes, strategies, fallbacks, routing). Read `SEAMS.md`, scan for decision points, and for each know what is decided, why, from which inputs, and what it controls, focusing on those deeply nested, scattered or duplicated with variations. Extract important decisions into named, intention-revealing helpers (`selectEligibleNotes`, not `check`) so a reader sees "this is where we decide X versus Y". Make inputs explicit (no hidden global state), remove redundant checks and merge branches that do the same thing, document or encode the criteria and each branch's intended behavior, and group related decisions by domain. Prefer a slightly longer obvious decision to a compact opaque one. Do not change business rules except to fix clearly incorrect behavior, and do not extract a helper whose indirection cost exceeds its clarity.

### 7. Change axis

Make the scenario cheap and safe to evolve. Infer the few primary axes of change from the PRD and targets (new workflow variants, new integration providers, new roles, UI changes without engine changes, policy updates without structural rewrites), map where each change would land today (files touched, shotgun surgery, scattered versus centralized responsibility), and give each axis obvious constrained extension points: a well-named module owning a policy, a mapping table gathering variations, a small handler owning one kind of change. Separate stable cores from volatile edges (rules, toggles, configuration, adapters, UI variants), pull axis-specific logic into cohesive units, and keep refactors incremental and tested. Prefer simple explicit structure to generic frameworks, and do not build for an axis that has no evidence of changing.

### 8. Experience architecture

How user types move through the product to reach their goals. Derive a purpose statement ("this scenario helps users ___ so they can ___"), a small set of personas (first-time, returning, ops, viewer) with one to three jobs each, trace each persona and job's current flow (entry, screens, actions, navigation depth, where the user must remember or hunt), compare with the ideal flow, and make safe targeted flow changes that cut friction. Check expected affordances: tables without sort, filter or search, long lists without search, forms without validation states, destructive actions without confirmation and feedback. Do not introduce new product directions. If the IA itself is wrong, record the candidate and hand a broad redesign to a plan. Record findings in `ARCHITECTURE.md` or `PROBLEMS.md` (read a legacy `docs/internal/EXPERIENCE-AUDIT.md` only as input).

### 9. Navigation integrity

Navigation is a verifiable contract: every URL, container, affordance, return path, shortcut, and reachability and deep-link policy is declared once in `ui/flow/navigation.json` and checked by `flow-verifier`, so label-to-destination drift, hidden auth or role gaps and viewport-keyed reachability surprises fail statically. `ui/src/routes.generated.ts` is generated from the spec and is the only place route paths are declared. Structural navigation problems (wrong places or groupings) belong to experience architecture; gamepad and spatial-nav runtime mechanics belong to `ui-health` §4.4.

Ladder per `flowId`: 0 routes as string literals in `App.tsx`; 1 inventory in the canonical architecture or problem doc; 2 a schema-valid `navigation.json` (`flow-verifier flows validate --kind navigation`); 3 `flows reconcile --flow <id>` returns zero discrepancies (every `<Route path>`, `<Link to>` and `useNavigate` resolves to a registered route id, no orphans either way); 4 reachability and deep-link policy pass on the static graph, Vitest with RTL asserts each affordance's declared label and destination, and at least one BAS flow walks the spec; 5 CI runs validate, reconcile and `verify run` and rejects raw path strings.

Every gated route has a positive invariant (`must_reach` within `max_clicks` per viewport given a context) and a negative one (`must_not_reach` when the predicate fails), and every route whose `requires` names a gated context has a `deep_link_policy` with its redirect target. Coverage percentages do not prove navigation completeness: a suite can render every component and never assert that an admin link is hidden. Commands (all take `--root <scenario>`): `flow-verifier flows list|validate|reconcile|studio` with `--kind navigation` or `--flow <id>`, `flows codegen --flow <id> --write`, `flows new <ui-dir> --flow-id <id> --kind navigation`, and `verify run --flow <id>`. If a recurring manual check has no command, add it to flow-verifier rather than working around it here. Record accepted exceptions in `PROBLEMS.md`, and findings in the Navigation section of the canonical architecture or problem doc.

### 10. Control surface

The small set of meaningful, safe levers (config, options, flags, parameters) that let humans and agents steer the scenario without touching implementation. Read `docs/reference/configuration.md` first. Find hard-coded decisions that matter (thresholds, limits, timeouts, batch sizes, modes, toggles, retry counts, backoff, cache windows, debounce and pagination) and promote only high-value, low-regret ones, judged by impact, audience (end user, operator, developer, agent), misconfiguration risk and adjustment frequency. Few well-chosen knobs beat a configuration jungle; keep internals internal. Group levers by how operators think (performance, safety limits, UX, integration), name them for the tradeoff not the mechanism, give each one responsibility and a stated effect when raised, lowered or toggled, and prefer booleans, enums and ranges to entangled multi-parameter controls. Make them safe: sane defaults that keep current tests green, validated ranges with documented types and units, extreme but valid values degrade safely, monotonic where possible, and no dangerous lever without guardrails. Extract constants into the existing config mechanism, preserve behavior under current defaults, update call sites and tests, and describe (do not half-do) a lever needing a large migration. Document the surface in `docs/reference/configuration.md`, verified against code.

---

### Safe change and output

You may move code to its zone, add seams and fakes, tag invariants with tests, rename for one canonical term, extract named decision helpers, add extension points, and add drift tests. You must preserve observable behavior and wire compatibility, keep tests at least as strict, change incrementally, never reverse another lens's outcome on one lens's grounds alone, record substantial redesign in `PROBLEMS.md` rather than half-doing it, and finish with at least one lens advanced a level and findings filed in existing docs. Avoid superficial renames or reshuffles that improve no lens.
