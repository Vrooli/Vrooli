# Decisions — Search Hub

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-03 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-11 | Record the ten durable retrieval and readiness decisions as one cross-domain contract. | The trustworthy-retrieval plan spans registration, routing, evals, scheduling, governance, and coverage. | Future changes must preserve: (D1) thin descriptor-driven routing; (D2) bounded scoped widening; (D3) provider-owned index age; (D4) exact leaf matching; (D5) direct and federated eval tiers over one corpus; (D6) no corpus content in Search Hub; (D7) generic breakers and demotion; (D8) registry-driven scheduling; (D9) Answer NOW requires active, reachable, fresh-eval signals; (D10) explicit-confirmation governance with provider files as SSOT. | Revisit only when a replacement contract is documented with migration and validation evidence. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

## Honest-signals decisions

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-12 | Keep transport failure separate from graded emptiness. | A failed provider call must not become evidence that a corpus is empty. | Circuits expose transport outages; zero-yield demotion only consumes successful graded-empty responses. | Revisit reporting only if operators cannot explain either state. |
| 2026-08-12 | Treat fresh junk-leaking eval evidence as an automatic-routing hold. | Aggregate relevance alone can hide a provider that promotes gibberish. | The run id and gate reason are visible; explicit selection remains available for repair. | Revisit when the eval model adds a stronger calibrated junk discriminator. |
| 2026-08-12 | Use a short address cache with failure invalidation and preserve partial fan-out results. | Process-spawn cost and deadline cancellation made large federations look empty. | Re-ported endpoints converge within the TTL or first failure; completed provider groups are not discarded. | Revisit after measured fleet latency and active-provider count materially change. |
| 2026-08-12 | Run demoted-provider recovery as an unattended, provider-scoped probe. | Waiting for the next user query made recovery depend on traffic, while treating the probe as explicit traffic would erase honest zero-yield evidence. | A one-minute default loop checks expired demotions; graded hits restore automatic eligibility, and empty/unavailable probes restart decay without pretending transport failure is corpus evidence. | Revisit when provider-owned validation supplies a cheaper or stronger recovery signal. |

## End-to-end retrieval trust plan decisions

| Date | Decision | Tradeoff | Revisit Trigger |
|---|---|---|---|
| 2026-08-13 | Keep one federated eval tier and split its verdict into routing precision and retrieval recall. | Existing pass-rate history remains comparable, while attribution adds the ownership signal needed to diagnose misses. | Add another tier only if the two rates cannot distinguish shortlist loss from provider retrieval loss. |
| 2026-08-13 | Compose the router suite from registered provider suites. | The suite is self-maintaining and agnostic, but inherits weak or stale provider labels. | Add authored router cases only if composed evidence misses a demonstrated routing failure. |
| 2026-08-13 | Recovery fails open; adoption fails closed. | Degraded recovery probes release a stuck slot, while providers without evidence remain explicit-only. | Revisit when provider-owned health evidence can replace either state machine. |
| 2026-08-13 | Never remove an unproven provider; expose it as experimental/incubating. | The registry remains a durable adoption ledger and parked providers become visible operational work. | Revisit only with an explicit registry-retirement policy and migration. |
| 2026-08-13 | Validate corpora with descriptor-shape predicates, never provider identity. | Generic rules catch self-echoing and collapsed families without scenario-specific branches. | Add an opt-out only when a provider documents why identifier-shaped queries are its real contract. |
| 2026-08-13 | Bound reranking from the measured serving leg and prefer fast grouped degradation to an unbounded wait. | Slow fallback answers are truncated more often, but query latency and failure attribution remain honest. | Revisit when the active reranker leg or its measured tail changes materially. |
| 2026-08-13 | Sequence restoration, attribution, corpus/evidence governance, then scale and board projections. | Later metrics consume evidence established by earlier phases, so intermediate results remain interpretable. | Reorder only if a new dependency makes a phase validation impossible. |
| 2026-08-14 | Treat a generated empty BAS registry as non-mutating for file-isolation analysis. | Search Hub has no workflow that can write application files; demanding unused file seams would block validation without proving an additional safety property. Missing or malformed registries still fail closed. | Revisit when Search Hub adds a mutating file-backed workflow. |
| 2026-08-14 | Keep scenario-owned UI provider composition in the scenario test adapter. | `api-base` supplies generic Query/i18n plumbing, while Search Hub must provide its initialized catalog and ThemeProvider so component tests match production. | Revisit if api-base gains an explicit scenario-provider registration contract used by all UI scenarios. |

## Discoverability decisions (2026-11-09)

| Date | Decision | Alternatives rejected | Switch cost | Revisit Trigger |
|---|---|---|---|---|
| 2026-11-09 | Measure discoverability with a client-side `intents` command over the public Query path and authored corpora, not as another eval suite. | Provider suites cannot express "owner scenario, any leaf type" and federated suites auto-route without the `--type` set agents are told to use; adding an RPC and proto for a read-only measure was costlier than a local command. | Move to an RPC + persisted runs when run history or UI display is needed (add an `IntentService`, keep the corpus format). | When intent runs need stored history or a board projection. |
| 2026-11-09 | Intent corpora are data with strict loading; scenarios may own `.vrooli/intents.json`. | Hardcoded cases in Go tests (invisible to owners); one central file (collisions between owners). | Cheap: the loader already globs both locations. | If owners ignore their own files, require one via `maturity scan`. |
| 2026-11-09 | `--group <name>` that is not a provider group is read as scenario scope (DO and REUSE leaves, filtered by ownership) and always explains itself. | Fail with an error only (loses the agent's evident intent); map scenarios to providers in the registry (providers hold many scenarios' leaves, so the map is wrong by construction). | Ownership is inferred from id/title/path shapes; if providers gain an explicit `owner_scenario` result field, replace `hitOwnedByScenario` with it. | A provider exposing owner scenario in its result mapping. |
| 2026-11-09 | Federation status stops presenting the retired LLM classifier as a fault; the wire field stays. | Remove `classifier_available` (breaks older clients); keep printing "unavailable" (reads as an outage on every status). | Remove the field in the next proto-breaking window. | Next routing proto revision. |
| 2026-11-09 | Test files no longer participate in any CLI freshness fingerprint. | Per-CLI `SkipSuffixes` (each of ~130 CLIs, easy to drift); narrowing the shared cli-core input to the import closure (larger, riskier change to a deliberate representation). | The first call of each CLI after this lands rebuilds once (fingerprint basis changed). | If a CLI embeds a `_test.go` file (none does). |
| 2026-11-09 | The standing guard for "a scenario's skill does not recall its own intents" lives in the existing Test Genie `skill-set` phase (prompt-manager's validator), as advisory warnings `skill-set.intent_missing` and `skill-set.intent_recall`. It queries prompt-manager's own skill search for the scenario's declared intents. | (a) A new Test Genie phase or a new `search-hub` finding: the search phase applies only to scenarios that own a search descriptor, which is exactly the set that is not failing; a new phase would duplicate skill-set's applicability. (b) Make recall a failing finding: it would turn red every scenario that has no corpus yet and conflate a wording defect with a broken declaration. (c) Call Search Hub from prompt-manager: adds a cross-scenario dependency; the skill leaf can be measured with the provider prompt-manager already owns. | Promote a warning to an error per code by changing `severity_default` in `scenarios/prompt-manager/.vrooli/test-genie.json` and the severity in `appendIntentWarnings`. Cross-type recall (what an agent sees over library, skill and command) stays a Search Hub measure (`search-hub intents run --assert`) because it needs the router. | When most scenarios declare intents, make `intent_recall` blocking; or when skill-set gains a router client. |
| 2026-11-09 | A case that names exact `expect_ids` is satisfied by one of them whatever the leaf is called. | Requiring scenario-name ownership too: breaks for skills whose id does not carry their scenario's name (`workspace-sandbox` ships `sandbox`). | None; ownership by name remains the default when no ids are given. | If skill ids gain an explicit owner field. |
