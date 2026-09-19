---
name: "web-search-improve"
description: "Regulate web-search against its setpoint: findings surfaced and used rates, never-surfaced decay, live-versus-local routing ratio, cache and governor telemetry, provider lifecycle, and external friction. Routes each out-of-band row to a ledger curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["web-search", "improve", "self-improvement", "control-loop", "setpoint", "findings", "curation", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 4
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["web-search", "search-hub", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["web-search findings effectiveness", "web-search findings count", "web-search findings used-rate", "web-search findings never-surfaced", "web-search findings gc", "web-search findings flag", "web-search findings supersede", "web-search findings prune", "web-search disputes list", "web-search disputes resolve", "search-hub insights insights", "search-hub providers list", "search-hub evals runs", "search-hub evals run", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---

## Practice focus: Web Search Improve

Improve research quality and task completion before optimizing effort. Skills carry judgment, governed programs carry stable sequences, and web-search owns evidence rules and findings. Repeated friction should move to the owning operation and remove work from the layers above.

Read `web-search`, `improvement-do-and-dont`, and `scenario-work-ladder` before a cycle. Read `measures-adoption` when adding an owner measure. Execute repairs within the session's authorized scope; report defects outside it through `report-bug`.

### Observe

Run `web-search.learning-read` with fixed from/to, operation, and context_key selectors. Keep current-fact requests, stable-fact reuse, comparisons, and L3 investigations in separate cohorts. Context includes method revision, source policy, source population, and freshness requirements. Memory measures attempts, completed and unresolved tasks, recurring failures, advice outcomes, first-action latency, tool round trips, and workflow reuse. Missing values, empty windows, truncated cohorts, and test attempts cannot establish an operator baseline.

Run `web-search.setpoint-read` for supporting curation and routing signals. Its legacy usage counters are historical diagnostics; the attempt ledger owns observed task outcomes. Cache and governor aggregate telemetry remain explicitly unavailable until the owner provides windowed measures. All-time routing ratios and capture counts have no quality band. More captures, less live traffic, or more programs do not establish improvement.

Before efficiency comparisons, inspect claim-to-citation support, retrieval dates, question coverage, unresolved contradictions, appropriate abstention, and correction evidence. `web-search.record-attempt` stores execution facts and appends a typed linked Memory observation for the assessment and contributing findings. Quality booleans require referenced observations; they are caller declarations, not independent proof of correctness. Retrieved findings are not automatically used findings.

### Compare methods

Use the same question corpus, freshness policy, source population, and result assertions for the existing and candidate method. Retain failed, unavailable, and unknown attempts in the denominator. Record source snapshots or references and dates, so a changed source is distinguishable from a changed method. Test fixtures exercise changed facts, wrong citations, stale/disputed findings, insufficient sources, partial outages, conflicts, and wait timeouts.

`web-search.compare-sources` collects bounded evidence for two to four explicit questions as optional subject, dimension, and temporal cells. `web-search.change-investigation` compares a prior cell set with a current one and keeps changed, unchanged, unverified, and incomparable results distinct. The skill still judges cross-cell contradictions and coverage. `web-search.research-l3` owns one governed start/wait handoff and preserves identity on timeout. Neither a successful RPC nor an agent exit proves the answer is correct.

Derive quality and effort targets from two comparable operator windows. Preserve previous method revisions and failing evidence through Program Runtime's revision history. Require quality floors before promoting lower latency or fewer calls. Empty baselines remain pending; never manufacture observations to close the loop.

### Repair

| Observed friction | Owning repair | Evidence required |
|---|---|---|
| Repeated operation or source rediscovery | Usage predicate or contextual advice | Better first-action latency with equal answer quality |
| Repeated successful bounded sequence | Versioned governed program | Same corpus and source-policy assertions |
| Stale or misattributed answer | Research evidence-policy operation | Wrong-source, stale, disputed, and outage regressions |
| Agent completion without usable evidence | Declared result contract and owner validation | Stable execution identity; invalid success rejected |
| Method fails after source changes | Candidate revision | Recorded old/new sources and bounded live smoke |
| Missing or unreliable learning | Capture or measurement owner | Fixed cohorts and unchanged denominators |
| Repeated external friction | Owning operation or skill | Episode evidence and regression |

Use `web-search.findings-curate` for evidence-backed proposals. Read proposed IDs before applying `gc` or `prune`; do not collect garbage to improve a ratio. Supersede a stale claim only with a supported replacement. Flag a wrong claim with evidence. Resolve disputes only after reviewing a fresh investigation. Keep disputed rows in diagnostic denominators. Never increase confidence or widen freshness solely to get more reuse.

Provider promotion requires two comparable passing Search Hub evals while the provider is reachable and enough real routed samples. Preserve the corpus assertions; a replacement finding must satisfy the original expectation. The live smoke corpus proves reachability, not recall or correctness. Do not lower a quality floor or promote an unreachable provider.

### Finish

Write one `vrooli-memory journal note --kind work-record` with trigger, approach, before/after evidence, affected finding or execution IDs, and outcome. Remove redundant prose/program workarounds after an owner repair. Close a cycle only when its selected quality and effort targets are supported by comparable observations. Keep unavailable sensors and pending baselines visible.

Stop on a required missing grant, exhausted session budget, or a failed quality floor. Scenario lifecycle work may proceed when the operator has already authorized it. Do not restart sessions to evade budgets, claim improvements from test fixtures, mark retrieved findings used, or repeatedly file the same authorized repair instead of fixing it.
