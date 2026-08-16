# Decisions — Prose Studio

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-08-16 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-08-16 | **D-001 — Ship two independent diversity mechanisms, not one.** Specified diversity (bind an axis space, assign each of k slots a distinct cell) alongside emergent diversity (verbalized distribution under a probability threshold). | The source research reports large diversity gains, but they are *emergent with model capability*, and we could not know in advance whether they reproduce on the models ai-gateway routes to. Specified diversity works on a weak local model because the variation comes from the plan, not the model. | Two sampler paths to build and measure. Set diversity can be *guaranteed* (axis) as well as *hoped for* (verbalized). Hedges the scenario's central risk instead of betting on it. | The acceptance measurement shows the verbalized strategy reliably beats direct prompting on every routed model — then specified diversity becomes an optimisation rather than a hedge. |
| 2026-08-16 | **D-002 — Quality is a floor; choice is by rarity or coverage. No judge-based ranker ships at any priority.** An inferential judge may gate pass/fail; it may never rank. | The research's root-cause finding is that mode collapse comes from *typicality bias in preference data*. A judge asked "which is best" reintroduces exactly that bias at selection time, undoing what the sampler bought. `judge.default` also routes to a small local model — the least capable possible arbiter for a task whose gains scale with capability. | Selection logic is deterministic and explainable. Agent mode defaults to rarest-above-threshold; human mode to coverage. A static check enforces that no policy reads the model's own ordering signal. | Never for the ranking case. A learned scorer may only ever be used as a *floor*, with rarity selection above it. |
| 2026-08-16 | **D-003 — Store the verbalized ordering signal as an ordinal marked `calibrated: false`, scoped to its own round, and bar it from selection by static check.** | 2026 work documents verbal-confidence saturation and extreme value sparsity; the numbers models report are not calibrated probabilities. They are still useful as a within-round ordering hint. | The field exists for analysis and is structurally unable to influence output. Aggregating, averaging, or thresholding it anywhere fails the build. | A provider ships genuinely calibrated per-candidate probabilities, verified independently rather than claimed. |
| 2026-08-16 | **D-004 — Candidate sets are taken at outline and section level, never whole-document.** | k=5 on a 1,200-word post costs roughly 7,800 output tokens and discards ~6,240 of them in one irreversible act, and a set of five complete posts is not reviewable by a human in any practical sense. | Single-passage remains the primitive; long-form composes above it. Sections are independently re-rollable. Review burden stays bounded as documents grow. | Real usage shows operators routinely accept whole sections unchanged, making document-level variation cheap enough to matter. |
| 2026-08-16 | **D-005 — Section context is feed-forward but bounded by a declared context policy.** Each section carries the outline, the committed text of prior sections, the declared intents of following sections, and its resolved profile. | Coherence is the entire point of long-form; fixed-input generation produces sections that do not know about each other. Bounding accumulation is what keeps feasibility decidable. | Context exhaustion within a document is *progressive* — a profile can pass at section 1 and fail at section 8 — so the static feasibility check must target the **worst-case** section, not the first. Rerolling a committed section invalidates downstream context snapshots, which the UI must surface before the reroll. | Coherence measurements show prior-section text contributes little, in which case a bounded summary would be cheaper. |
| 2026-08-16 | **D-006 — For a consumer-declared record the file is authority and the database row is a projection.** Two provenance classes with disjoint namespaces; `local/` reserved for operator-authored records. | A consuming team must own its own voice by editing files in its own repository, with zero integration code here. Two writable sources for one record is the classic drift bug. | `authority: file` is checked on every write path; API writes to a declared record are refused naming the file; version is the file's content hash; deletion marks `unregistered` rather than deleting so historical provenance still resolves; promotion is one-way and there is deliberately **no round-trip sync**. | A consumer needs to mutate a declared record at runtime — which would be a signal the record is really operator-authored. |
| 2026-08-16 | **D-007 — Record selections from day one; learn nothing from them in v1.** | Selection data is cheap to collect now and impossible to collect retroactively. But a preference model trained on which candidate the operator picked is, definitionally, preference data — from a dataset of one annotator, which is strictly worse than the crowd-annotated data the source research indicts. | `selection_event` carries the chosen candidate, the considered candidates, and measurements snapshotted at choice time, plus a reserved `outcome_ref`. Nothing reads it. Candidate-level ranking is additionally *ill-posed*: Bradley-Terry and Elo need a stable item pool and candidates are ephemeral, so `elo-swipe` is the wrong engine for this object. | Legitimate learning exists at other levels: style learning from operator *edits* (P1), configuration bandits over durable entities (P2), and outcome scoring from measured external events (P2). Never per-candidate preference. |
| 2026-08-16 | **D-008 — All inference routes through ai-gateway. This scenario names no model, holds no credential, and speaks no vendor protocol.** | The gateway owns provider selection, locality, capacity, and metering. A direct provider call here would bypass the boundary ai-gateway's own conformance phase exists to enforce — and it makes metering Class A for free. | Two prose roles (`write.default`, quality lane, local-first; `write.diverse`, candidate-set lane, hosted-first) plus request-level sampling control are **hard prerequisites**; this scenario cannot ship its generation slice before they exist. Max output tokens are derived per request from profile data rather than relying on a role default. | Never. Holding a provider credential here would be an architectural error. |
| 2026-08-16 | **D-009 — Metrics live here behind a shared package; take no dependency on `llm-evaluator`.** | `llm-evaluator`'s PRD claims this charter, but it is an abandoned scaffold: 705 lines of non-test Go across 8 files, one internal subdirectory, zero implementations of any named metric, zero references anywhere else in the repo, and `service.json` carrying `name: null`. | `packages/textmetrics` is created from the first commit and is consumable with no runtime dependency on this scenario's API, so `llm-evaluator` can adopt it later rather than being blocked on it. | `llm-evaluator` is genuinely built and adopted, at which point the shared package is the migration path rather than an obstacle. |
| 2026-08-16 | **D-010 — "Human-sounding" means prose quality with machine-generation disclosure preserved, explicitly not detector evasion.** | Detector evasion is a permanently moving target, conflicts with disclosure obligations, and would change what the scenario optimises. | Disclosure is carried on every candidate at birth as a constant. No target, verb, or interface copy is framed in terms of a detector score. Detector *methodology* is still mined for measurable properties of machine text, because those are legitimate quality signals. | Never. |
| 2026-08-16 | **D-011 — The declaration validator ships at P0; the `prose-conformance` test-genie phase wrapper ships at P1.** Operator decision. | The validation logic and the phase packaging are separable. The phase contract requires a North Star, a gated L0–L4 ladder with per-rung capability summaries and next-unlock statements, a runtime-computed assessment, and structured remediation docs — days of work whose value depends on the validator being stable first. | A consumer can check its declarations at P0 over RPC and CLI. What waits is only *automatic* invocation during that consumer's own suite. Known trap for the P1 work: maturity ladders start at max, so a phase emitting nothing reports "complete" unless its rungs declare a clean requirement. | The validator is stable and a second consumer has declarations worth gating a build on. |

## Acceptance Evidence

On 2026-08-16, the live routed-model signal ran through ai-gateway with the same
query and k=3 for both lanes. `direct` routed to `gemma4:12b` through Ollama
and measured diversity `0.5116337190702418`; `vs_standard` routed to
`anthropic/claude-fable-5` through OpenRouter and measured diversity
`0.7406874592131234`. Both used the same lexical token 1–3 gram
cosine/Jaccard basis and profile output cap of 1024 tokens. Because the roles
route to different models, this is evidence that the signal is operational,
not a causal model-controlled comparison; future acceptance runs must report
within-model comparisons before claiming a strategy effect.

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
