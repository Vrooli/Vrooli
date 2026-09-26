# Signal Ladder

This document is the canonical reference for the cartographer's
pluggable scoring signals: what they measure, what their default
weights are, how they combine, what their known failure modes are,
and what the rules for adding or reprioritizing them are.

The signal ladder is load-bearing for everything downstream:
auto-placement of chunks during graph-vs-derived-domain-map comparison,
suggested-fix ranking on conflicts, and confidence reporting in the
CLI workbench. The explainability of every cartographer decision
flows from this contract.

## Purpose Of This Document

Use this document to answer:

- Which signals exist by default?
- What does each signal actually measure?
- What is each signal's default weight, and why?
- When does each signal fail or produce a misleading score?
- How are signals combined into a final verdict?
- How do I add a new signal?
- How do I reprioritize the existing set?

The interface seams live in [`DOMAINS.md`](DOMAINS.md) under the
`signals` domain. The flow that consumes signals lives in
[`FLOWS.md`](FLOWS.md) under "Auto-placement verdict."

## The Signal Contract

Every signal implements the same interface:

```go
type Signal interface {
    Name() string
    Score(chunk Chunk, domain Domain, ctx GraphContext) Score
}

type Score struct {
    Value    float64  // 0.0–1.0 confidence this chunk belongs to this domain
    Reason   string   // one-line human-readable explanation
    Evidence []string // concrete data points the agent can verify
}
```

Five invariants are non-negotiable:

1. **Pure function** — given the same `(chunk, domain, ctx)`, a
   signal must produce the same `Score`. No clocks, no random
   numbers, no network IO during scoring.
2. **No graph mutation** — signals receive an immutable graph
   snapshot and cannot modify it. This makes scoring parallelizable
   and reproducible.
3. **Self-explaining** — nothing is returned without explanation.
   Every `Score` carries a non-empty `Reason` and ≥1 `Evidence`.
   A signal that has no data for a chunk emits an explicit
   `Abstention` (also with `Reason` + ≥1 `Evidence`) — never an
   empty `ScoreResult`. The aggregator counts abstaining signals
   toward participation confidence, but not toward the direction
   denominator; a "silent skip" would amount to letting one signal
   disappear from the quorum calculation.
4. **Bounded** — `Score.Value` is always in `[0.0, 1.0]`. No
   negative scores; no scores above 1.0. Aggregation logic depends
   on this.
5. **Cheap** — signals should complete in milliseconds for a single
   `(chunk, domain)` pair. They run inside hot loops during large
   graph comparison; expensive scoring blocks the whole verdict.

## The Aggregator

The Aggregator combines all registered signals into a verdict per
`(chunk, candidate_domain)` pair. It keeps direction and confidence
separate:

```
direction_value = Σ(signal_weight × signal_score) / Σ(scoring_signal_weights)
confidence      = Σ(scoring_signal_weights) / Σ(available_signal_weights)
```

An unavailable signal is skipped entirely. An abstaining signal is
available-but-non-scoring: it does not dilute `direction_value`, but it
does lower `confidence`. This preserves the original goal ("one
surviving signal cannot pretend to be certainty") through an explicit
quorum gate instead of by suppressing every directional score.

Weights use the day-one defaults below. Tier and quorum thresholds are
cartographer-**global** tunable levers (`CARTOGRAPHER_AUTO_PLACE_MIN`,
`CARTOGRAPHER_SUGGEST_MIN`, `CARTOGRAPHER_TIE_DELTA`,
`CARTOGRAPHER_QUORUM_HIGH`, `CARTOGRAPHER_QUORUM_LOW` — see
[`../reference/configuration.md`](../reference/configuration.md)), NOT
per-scenario declarations. The candidate domain with the highest
`direction_value` is selected; its confidence tier is determined by
the direction thresholds plus quorum:

| Tier | Default Threshold | Action |
|---|---|---|
| `auto_place` | `direction_value ≥ 0.85` AND `confidence ≥ 0.45` | Auto-assign the chunk to this domain. |
| `suggest` | `direction_value ≥ 0.55` AND `confidence ≥ 0.30` | Surface as a ranked suggestion; agent must confirm. |
| `conflict` | Below `suggest`, below low quorum, or top-two domains within 0.10 | Emit a `mislocated_file` (or similar) conflict for agent decision. |

Aggregation produces explainability output that combines every
signal's `Reason` and `Evidence` into a single verdict explanation.
Sample output:

```
Verdict: auto-place session.go in `auth` (direction_value=0.91, confidence=0.83)
  Evidence:
    [path-token, w=1.5] file path contains "auth" segment
    [import-cluster, w=1.0] file is in import-graph community {auth} (purity=0.92)
    [symbol-glossary, w=0.9] defines symbols matching auth glossary: Token, Session
    [importer-voting, w=0.8] 6 of 7 importers are in `auth` domain
```

## Finding Classes

Signals feed both deterministic checks and advisory checks. The output
contract keeps those separate:

- **Deterministic** findings are graph or declaration facts (for example
  import cycles, wrong-direction layering imports, missing declared
  surfaces). Deterministic `error`/`blocker` findings can gate audit
  outcomes.
- **Heuristic** findings are taste, placement, vocabulary, or coupling
  signals (for example `mislocated_file`, `naming`, `glossary_drift`, and
  `coupling_smell`). They remain visible and actionable, but are capped at
  `warn` and never hard-fail CI.

`finding_class` is excluded from stable identity for both native
cartographer conflicts (`csid:`) and shared architecture findings (`afid:`).
Reclassifying a finding is therefore a policy correction, not a new
regression.

## Day-One Signals

The ladder ships with six deterministic signals. Each has a default
weight chosen to reflect its independence and reliability.

### 1. `path-token`

- **Measures**: whether the chunk's file path contains a token that
  case-insensitively matches a declared domain name (or alias).
- **Default weight**: `1.5` (highest — paths are the most direct
  authorial signal of intent).
- **Score**: `1.0` if exact token match; `0.7` if a glossary alias
  matches; `0.0` otherwise.
- **Reason example**: `"file path contains 'auth' segment"`.
- **Failure modes**: scenarios with intentionally generic paths
  (`src/lib/`, `internal/`) yield `0.0` for everything; very deep
  paths with multiple domain tokens can match multiple domains
  (broken tie handled by aggregator).
- **When to disable**: scenarios with an unusual path convention
  that's incompatible with token matching.

### 2. `import-cluster`

- **Measures**: graph community membership via Louvain modularity
  (or label propagation) on the file-import graph. A chunk is
  scored by the domain purity of its community.
- **Default weight**: `1.0` (strong but indirect — clusters can be
  large and spurious in pathological scenarios).
- **Score**: cluster purity for the candidate domain, weighted by
  cluster cohesion. A file in a 90% `auth` cluster scores `0.9` for
  `auth`.
- **Reason example**: `"file is in import-graph community {auth} (purity=0.92)"`.
- **Failure modes**: very small clusters (3 files or fewer) produce
  noisy purity scores; cycles inside a cluster can blur boundaries.
- **When to disable**: extremely small scenarios where community
  detection has insufficient signal (≤20 files).
- **Implementation note**: deterministic Louvain community detection
  runs once per graph snapshot; results are cached on the signal graph
  context. Stable node ordering and tie-breaks make repeated runs over
  the same graph return the same communities.

### 3. `importer-voting`

- **Measures**: majority domain among files that import the candidate
  chunk's file.
- **Default weight**: `0.8`.
- **Score**: fraction of importers in the candidate domain, weighted
  toward higher counts. `(6 of 7 importers in auth)` scores
  `(6/7) × 0.95` = `0.81`.
- **Reason example**: `"6 of 7 importers are in `auth` domain"`.
- **Failure modes**: files with very few importers (`≤2`) produce
  unstable scores; widely-imported utility files (≥20 importers
  spread across domains) yield low scores for every domain.
- **When to disable**: scenarios where most files are widely shared
  (e.g., utility-heavy codebases).

### 4. `test-coupling`

- **Measures**: which domain's test files reference symbols defined
  in the candidate file.
- **Default weight**: `0.7`.
- **Score**: fraction of test references coming from the candidate
  domain. A file whose only test references come from
  `features/auth/__tests__/` scores high for `auth`.
- **Reason example**: `"tested exclusively by auth-domain tests"`.
- **Failure modes**: untested files yield `0.0` everywhere (no
  signal, not negative signal); cross-cutting test helpers ignored.
- **When to disable**: rarely useful to disable; this is a clean
  signal when tests exist.

### 5. `symbol-glossary`

- **Measures**: whether the chunk's exported symbol names match the
  candidate domain's declared glossary (case-insensitive, token-based).
  The glossary is the optional `Glossary` column of the derived domain
  map's source (`docs/concepts/DOMAINS.md`), built via
  `domains.BuildGlossary` — there is no per-scenario manifest.
- **Default weight**: `0.9` (high — the glossary is explicitly declared
  in DOMAINS.md, so matches are intentional).
- **Score**: fraction of exported symbols matching the glossary. A
  file exporting `Token`, `Session`, `LoginRequest` matches `auth`'s
  glossary `{Token, Session, Login*, Logout*, Credential}` at
  `3/3 = 1.0`.
- **Reason example**: `"defines symbols matching auth glossary:
  Token, Session"`.
- **Failure modes**: scenarios with sparse or generic glossaries
  yield uniformly low scores; overlap between domain glossaries (a
  `Notification` type owned by both `notifications` and `audit`
  domains) causes ambiguity.
- **When to disable**: DOMAINS.md does not declare glossaries.

### 6. `git-co-edit`

- **Measures**: file co-edit frequency in git history with files
  already placed in the candidate domain.
- **Default weight**: `0.6` (weakest of the day-one set — historical
  signal, can be noisy for new files).
- **Score**: normalized co-edit count with same-domain files over a
  rolling window (default 90 days).
- **Reason example**: `"co-edited with 4 auth-domain files in last
  90 commits"`.
- **Failure modes**: newly added files have no history (score
  `0.0`); reorganization commits create spurious co-edit links.
- **When to disable**: shallow git history (e.g., shallow clones in
  CI); brand-new scenarios.
- **Implementation note**: shells out to `git log --name-only`; if
  `git` is unavailable, signal disables itself rather than failing
  the verdict.

## Default Weights Summary

| Signal | Default Weight | Why |
|---|---|---|
| `path-token` | 1.5 | Most direct authorial signal; lowest false-positive rate. |
| `import-cluster` | 1.0 | Strong structural signal but can be coarse. |
| `symbol-glossary` | 0.9 | DOMAINS.md-declared intent; depends on glossary quality. |
| `importer-voting` | 0.8 | Direct usage signal; can be diluted by utility files. |
| `test-coupling` | 0.7 | Very clean when present, often absent. |
| `git-co-edit` | 0.6 | Historical, can be noisy; weakest standalone. |

Weight values are starting points. The `arch-cart calibrate` command
(P2, OT-P2-005) proposes adjustments based on observed override
history. Weight changes always require explicit human acceptance —
never auto-applied. See [`../internal/DECISIONS.md`](../internal/DECISIONS.md)
when weights are durably changed.

## Adding A New Signal

1. Implement the `Signal` interface in a new file under
   `api/internal/signals/<name>/`.
2. Add unit tests under the same directory covering: pure-function
   reproducibility, evidence non-emptiness, bounded `[0,1]` output,
   default-weight rationale.
3. Register the signal in `api/internal/signals/registry.go`.
4. Add an entry to this document with: what it measures, default
   weight, failure modes, when to disable.
5. Add the signal's default weight to the aggregator defaults in
   `api/internal/signals` (weights live in code; per-instance tier
   thresholds are global config, not per-scenario).
6. If the signal depends on an external tool (like `git-co-edit`
   does on `git`), document the dependency in
   [`INTEGRATIONS.md`](INTEGRATIONS.md).
7. Record the decision and the rationale in
   [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

A signal that does not document a failure mode is incomplete. Every
signal must declare at least one situation where it is unreliable.
Honesty about failure modes is what makes the explainability
contract work.

## Reprioritizing The Existing Set

Default weights live in code (the aggregator); there is no per-scenario
weight overlay. Tier thresholds are cartographer-global levers. To
reprioritize:

- **Thresholds for an instance**: set `CARTOGRAPHER_AUTO_PLACE_MIN` /
  `CARTOGRAPHER_SUGGEST_MIN` / `CARTOGRAPHER_TIE_DELTA` (see
  [`../reference/configuration.md`](../reference/configuration.md)).
- **For the default ladder**: update the aggregator defaults and ship a
  new cartographer release. Update this document with the new
  numbers and the rationale.

Reprioritization should be driven by override data from analytics.
The `arch-cart calibrate` command surfaces candidate adjustments:
"signal X scored chunk Y in domain A, but agent moved it to domain B
in N cases — consider lowering signal X's weight or upweighting
signal Z which would have predicted B."

## What Is NOT A Signal

These would seem like signals but are intentionally excluded:

- **Embedding cosine similarity** — deferred to P2 (OT-P2-001), and
  even then only as a suggestion ranker, never auto-place. Embeddings
  produce silently-wrong placements; the failure mode is
  incompatible with the auto-apply tier.
- **Code style / formatting** — irrelevant to domain ownership.
- **Author name (git blame)** — privacy-sensitive and noisy.
- **Last-modified date** — strongly correlates with recency, not
  ownership.
- **Manual annotations in comments** — signals never read free-form
  intent comments; domain intent comes only from the derived domain map
  (DOMAINS.md → folders → CLI groups). The one sanctioned comment channel
  is the `// arch:allow` suppression marker, which excuses findings rather
  than declaring placement.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — the `signals` domain owns the registry
  and aggregator
- [`FLOWS.md`](FLOWS.md) — the auto-placement verdict flow
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — durable seams and intentional
  deviations
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external tools that signals
  depend on (e.g., `git`)
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable
  weight or threshold changes
- [`../internal/TESTING.md`](../internal/TESTING.md) — signal-testing
  patterns
