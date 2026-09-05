# AI Search Routing — The Search-Context Map

**Status:** Transitional reference (active). The `search-hub` federated router has **shipped** (see `docs/plans/unified-search-hub-plan.md`): all currently-live providers below are now *also* reachable via one command — `search-hub query "<anything>" [--type a,b | --all]` — which classifies, fans out, and cross-encoder reranks into one list. Each scenario keeps its own search (non-destructive). Rows still marked ❌ gap have no semantic search anywhere yet; they are tracked as `capability_gap` stubs in the hub registry (`search-hub providers list --state capability_gap`).

> **For agents:** Before hand-rolling a grep or guessing, find the right search context below and use it. Every row that says **✅ live** is a real, working semantic search you can call right now. Rows marked **❌ gap** have no semantic search yet — fall back to filesystem grep / `ast-grep` and consider filing the gap.

---

## 1. The mental model: a search is `(corpus) × (intent)`

There is no single "search the project." There are *many* corpora, each with a different shape (a CLI command embeds differently than a doc paragraph than a Go function), and each answers a different *question*. The questions cluster into four buckets that map onto Vrooli's build loop. When you go to do work, you are always asking one of these:

| Bucket | The question | Nature |
|---|---|---|
| **DO** | "What can I *do* right now that I can invoke?" | imperative — invoke and something happens |
| **REUSE** | "What already *exists* that I can build on?" | compositional — things you assemble |
| **KNOW** | "What is *known* / what's been learned or explained?" | informational — things you read |
| **STATE** | "What is the system *supposed* to do / what *is* its state?" | declarative — descriptions of intent & state |

Picking the bucket narrows you to a handful of corpora. Picking the corpus tells you which tool to call.

---

## 2. The routing table

Legend: **✅ live** = working semantic/AI search · **⚠️ partial** = exists but limited/non-semantic/being-fixed · **❌ gap** = no semantic search yet.

### DO — imperative surface

| Corpus | Question it answers | Provider | Status | How to search today |
|---|---|---|---|---|
| **Actions** (parametrized single commands) | "Is there a ready command that does X?" | prompt-manager | ✅ live | `prompt-manager discover "<intent>" --type action` · also via `search-hub query --type action` |
| **Skills** (how-to judgment / guidelines) | "Is there guidance on how to do X?" | prompt-manager | ✅ live | `prompt-manager discover "<intent>" --type skill` (or `--type all`) · also via `search-hub query --type skill` |
| **CLI commands** (capabilities across all scenarios) | "What capability can I invoke via a CLI?" | cli-health | ✅ live | `cli-health search "<intent>"` · also via `search-hub query --type command` |
| **Workflow flows** (scenario-owned BAS journeys) | "Is there a safe workflow I can run or inspect?" | workflow-health | ✅ live | `workflow-health workflows search "<intent>" --scenario <id> --type workflow.flow` · workflow-health's own catalog also registers with Search Hub as `search-hub query --type workflow.flow` |

For implementation-plan authoring, keep one distinction: `search-hub query
--type skill` is useful broad/federated skill lookup, while `prompt-manager
discover --type skill --complexity <tier>` is the authoritative curated
skill-bundle surface. Prompt Manager discovery returns topic-pack inclusion,
budget status, and the recommended `prompt-manager skill read ...` command that
Plan Manager should store as setup context.

### REUSE — compositional surface

| Corpus | Question it answers | Provider | Status | How to search today |
|---|---|---|---|---|
| **UI components** | "What component exists I can reference/reuse?" | ui-health | ✅ live | `ui-health search "<intent>"` · also via `search-hub query --type component` |
| **Widgets** (chat-embeddable surfaces) | "What can I embed into an interface?" | ui-health | ✅ live | `ui-health search "<intent>"` (widget contract owned by ui-health) · also via `search-hub query --type widget` |
| **Scenarios** (whole capabilities/products) | "Does a scenario already deliver this capability?" | scenario-dependency-analyzer *(planned)* | ❌ gap | `vrooli scenario list` + read PRDs; no semantic search yet |
| **Resources** (local substrate: ollama, qdrant, redis…) | "What resource exists that offers X?" | scenario-dependency-analyzer *(planned)* | ❌ gap | browse `resources/`; no semantic search yet |
| **Code / reference implementations** | "How have we solved X before in code?" | *(new — code-reference, planned)* | ❌ gap | `ast-grep --lang <l> --pattern '…'`, grep; structural-only via go/ts-code-graph |
| **API / proto contracts** (RPCs, message types) | "What endpoints/messages exist for domain X?" | *(new — contract registry, planned)* | ❌ gap | grep `packages/proto/schemas/`; no semantic search yet |
| **Design kits / tokens** | "What design pattern/kit exists?" | ui-health (partial) / branding *(planned)* | ⚠️ partial | `templates/design/`, scenario `DESIGN.md` |
| **Workflow tests/fragments** (BAS validation cases and reusable actions) | "What workflow proves this, or what action fragment composes it?" | workflow-health | ✅ live | `workflow-health workflows search "<intent>" --scenario <id> --type workflow.test` or `--type workflow.fragment` · workflow-health's own catalog also registers with Search Hub as `search-hub query --type workflow.test` / `workflow.fragment` |

### KNOW — informational surface

| Corpus | Question it answers | Provider | Status | How to search today |
|---|---|---|---|---|
| **Documentation** (project + scenario prose) | "What's been written/explained about X?" | knowledge-observatory | ✅ live | `knowledge-observatory docs search-text "<q>"`; **semantic search cutover landed** (6098 docs, bge-reranker-v2-m3) · also via `search-hub query --type doc` |
| **Records** (past work: how we solved X) | "How did a prior agent solve this?" | swarm-manager | ✅ live | `swarm-manager search-ai "<q>"` (records indexed) · also via `search-hub query --type record` |
| **Captures** (raw friction/intent) | "Has this friction been logged before?" | swarm-manager | ⚠️ partial | `swarm-manager captures …`; semantic coverage via aisearch |
| **Bugs / known issues** | "Is this a known problem?" | scenario-qa / bug-inbox | ⚠️ partial | `report-bug` write-side; search-side limited |
| **Git history / change provenance** | "When/why did this change; what runs/prompts produced it?" | git-control-tower *(planned)* | ❌ gap | `git log`; rich commit↔run↔prompt↔initiative graph is being built |

### STATE — declarative surface

| Corpus | Question it answers | Provider | Status | How to search today |
|---|---|---|---|---|
| **Backlog / initiatives** (planned work) | "Is there a planned item for X?" | swarm-manager | ✅ live | `swarm-manager search-ai "<q>" --entity backlog\|initiative\|both` · also via `search-hub query --type backlog,initiative` |
| **Requirements / operational targets** | "What is scenario X supposed to do?" | product-manager-agent / PRD *(planned)* | ❌ gap | read `requirements/index.json`, `PRD.md`; test-genie owns targets |
| **Configuration / operator state** | "What config/integration options exist?" | vrooli-onboarding *(planned)* | ❌ gap | `docs/configuration/` (canonical, prose); no semantic search |
| **Agent runs** (execution history) | "What run worked on X; what prompt/outcome?" | agent-manager *(planned)* | ❌ gap | agent-manager run listing; no semantic search |
| **Domain map / architecture** | "Who owns domain X; what's the boundary?" | architecture-cartographer *(planned)* | ❌ gap | `DOMAINS.md`, cartographer evidence-based convergence |
| **Runtime data / metrics** | "What is the system doing right now?" | command-center *(planned)* | ❌ gap | command-center aggregator; no semantic search |
| **Measures** (analytical questions → computed answers) | "How many / what rate / what's next?" — a *computed value* about a scenario's state | measures-health (central index) | ✅ live | `search-hub query "how many backlog items closed this week"` — matches a declared measure and deterministically resolves canonical parameters inside the interactive budget. The owning scenario's direct measures surface executes the selected read-only measure with provenance; write/destructive measures are never auto-run. · also `measures-health search query "<q>"` |

> **Measures vs the rest of this table.** Every other row answers *retrieval* —
> "find me the thing." **Measures** answer *analysis* — they return a *computed
> number* (a count, a rate, a median) for an analytical question, parameterized by
> a time window/filter/grouping. Scenarios declare measures via a manifest
> `measure` block on `packages/measures-go`; `measures-health` harvests them into
> one central index and owns the single registered search-hub "measure" provider.
> See [`../concepts/MEASURES.md`](../concepts/MEASURES.md).

---

## 3. The gaps, ordered by why they matter

The compositional and historical gaps hurt the recursive-improvement loop most — they are exactly the "before I build, has this been done?" questions:

1. **Scenarios** — "does a scenario already deliver this capability?" (prevents rebuilding capabilities)
2. **Records** is *closed* (swarm-manager) — the historical-learning read-side works; keep using it.
3. **Code / reference implementations** — semantic "how have we done X" (structural exists via code-graph; semantic does not)
4. **Resources** — "what substrate can I compose?"
5. **API/proto contracts** — likely justifies a new contract-registry scenario (managing the contract surface is a job, not just search)
6. **Agent runs**, **git provenance**, **requirements**, **config**, **domain map**, **runtime metrics** — STATE-surface gaps; each has a natural home scenario (see table).

**Placement rule (why some gaps become features, not scenarios):** *If a corpus already has a natural home scenario, search is a **feature** that scenario registers. Only create a new scenario when the corpus has no home **and** managing it is a job in its own right.* By this rule, Scenarios/Resources search are features of scenario-dependency-analyzer; Code-reference and Contracts plausibly justify new scenarios.

---

## 4. Where this is going (the one-command future)

This table is a stopgap. The target is a **federated search router** scenario (`search-hub`) where:

- Each corpus owner **self-registers a search provider** (mirroring how scenarios self-register an agent profile to agent-manager).
- A single entry point — `search-hub query "<anything>" [--type a,b,c | --all]` — routes via a local Ollama classifier (or explicit `--type`), fans out to providers, and **cross-encoder reranks** heterogeneous results into one ranked list.
- The router owns only the **registry + classifier + reranker + metrics**. It never holds corpus data; providers stay authoritative and keep their own indexes fresh.
- Shared retrieval *logic* (embedding, chunking, hybrid BM25+dense, drift reconcile) lives in the `packages/ai-go/search` library — not in the router — so providers dedup implementation without the router becoming a monolith.

When that ships, most rows above collapse to: **just call `search-hub`.** The buckets/`--type` values survive as routing facets. Until Search Hub exposes provider-specific plan-authoring artifacts, specialized corpus contracts such as Prompt Manager's curated skill-bundle output remain authoritative. See `docs/plans/unified-search-hub-plan.md`.

**Search quality is now baselined in `search-hub`.** Beyond *routing*, the
`search-hub` **eval domain** lets each provider register a golden suite of
queries with expected results + score bands, run it as an immutable **tagged**
run, and compare runs over time (CLI `search-hub evals …`, plus an "Evals" UI
tab). This is how a corpus owner proves a retrieval change (e.g. enabling the
cross-encoder reranker) actually helped — soft labels and stored history, not a
pass/fail gate. See `scenarios/search-hub/README.md` (eval domain) and
`docs/plans/search-quality-baseline-harness-and-rerank-enablement-plan.md`.

The eval domain is the **single source of truth for A/B and tuning**. A scenario
may keep a thin recall@k per-build gate (e.g. KO's `TestAccuracyCorpus`) as a
smoke check that its corpus still resolves its own goldens, but that gate must
not duplicate the eval domain — no second run store, no second comparison
surface. Tune a knob in search-hub (register → tagged A/B → `evals compare`),
then set the per-scenario flag (`<PREFIX>_RERANK_ENABLED`) from the result.
For maturity work, the Test Genie `search` phase and `search-hub maturity scan`
read that same eval store and probe live corpus labels to report missing, stale,
failed, or unavailable evidence explicitly. `search-hub maturity scan --fast`
is only a quick inventory mode and intentionally skips live retrieval proof.

---

*Maintainer note: keep this table in sync as providers come online. A row flips from ❌/⚠️ to ✅ the moment its semantic search is callable; once `search-hub` federates a provider, note it as reachable via the router too.*
