# Architecture Validation: the four responsibilities and the two-layer model

This doctrine pins how Vrooli validates a scenario's architecture and how it
*drives* a scenario toward well-architected over time. It is the mental model
behind the `architecture-audit` test-genie preset and the
`architecture-cartographer campaign` tracker. Read it before reasoning about
where an architecture check belongs or why the campaign tracker exists.

## Four responsibilities, two axes

Architecture validation is not one thing. Four distinct responsibilities, owned
by four different surfaces:

| # | Responsibility | Question | Owner | Nature |
|---|---|---|---|---|
| A | Per-surface conformance | "Is each surface built right?" (CLI/UI/docs manifests, proto bindings) | cli-health, ui-health, knowledge-observatory, scenario-auditor | point-in-time, **gating** |
| B | Code quality | "Is this code clean?" (file/function length, complexity, duplication) | tidiness-manager | point-in-time, metric |
| C | Structural architecture | "Does the structure cohere & scream its purpose?" (cycles, coupling, convergence, mislocation) | architecture-cartographer *(detection)* | point-in-time, confidence-gated |
| D | Architecture finding lifecycle | "Which findings persist, regress, or are verified resolved?" | architecture-cartographer *(campaign)* | **longitudinal, stateful** |

A, B, and C answer *whether* something is right — they are **validation**
(point-in-time). D retains finding-level continuity across those observations.
It supports development but does not own the work's authorization, product
targets, or final acceptance. The
[scenario development method](../agent-system/SCENARIO_DEVELOPMENT.md) owns that
separation; an architecture campaign is not a second project control plane.

Code quality (B, tidiness-manager) is a **separate axis** — file/function
metrics, not structural cohesion. It is not folded into the architecture audit;
it may become an ingest source for the tracker later, but it is not wired today.

## The two-layer model and the seam

```
  test-genie  ──ArchitectureFinding──▶  architecture-cartographer
  (run and evidence owner)              (finding lifecycle owner)
  audit AGGREGATOR                      campaign TRACKER
```

- **test-genie is the camera.** It runs the per-surface validators it already
  orchestrates (A) plus a structural `architecture` phase that delegates to
  cartographer's read-only audit (C), and emits **one normalized findings
  report**. It retains run history and evidence under the
  [testing contract](../TESTING.md); one run is an observation, not the work plan.
- **architecture-cartographer's campaign domain tracks findings.** It
  ingests that report, tracks every finding through a lifecycle, hands the agent
  a profile-ranked worklist, and on each re-audit reconciles by stable ID. It
  owns finding lifecycle and regression reconciliation, not all project memory.

The normalized findings report is the **seam** — the shared
`ArchitectureFinding` contract (`packages/proto/schemas/architecture/v1/`).
Run evidence and finding lifecycle have distinct owners.

**Cartographer never calls test-genie or the health CLIs.** Findings arrive only
by ingest (push). There is no cycle: the camera produces, the tracker consumes.

### Stable identity (afid)

Reconciliation matches purely on a content-hash stable ID:

```
afid:<8 hex> = sha256(scenario ∥ source ∥ code ∥ sorted(locations))
```

Severity, message, and domains are **excluded** so cosmetic changes never
manufacture a false regression. The same defect collapses to one ID across runs
and across the test-genie→cartographer boundary, because both sides compute the
afid from the same shared helper (`packages/proto/architecture/findingid`).

## The validation → campaign loop

An audit can recommend tracking when findings outgrow a local repair. That
recommendation does not authorize implementation or require a new Swarm item.
For an authorized multi-cycle engagement, `scenario-improvement-campaign` owns
the driving loop and selects the tracker when finding continuity is needed.
An assessment-only caller reports findings without creating or mutating a campaign.

Use [docs/TESTING.md](../TESTING.md) for audit scope, durable evidence, and waiting.
Use the current cartographer campaign contract for ingest, ranked worklists,
resolution notes, and re-audit reconciliation. Do not treat a launch response as
an ingestible completed audit. A resolution note is a claim; re-audit supplies
verification. Stable IDs preserve persistent and returning findings.

The ranking profile orders candidate repairs. It cannot change acceptance floors
or expand the work's scope. Closing a finding tracker does not establish that the
scenario's other required outcomes pass. Stop and acceptance decisions remain
with the authorized development grant and its finish line, not an unconditional "until clean" loop.

The campaign nudge remains the primary steering mechanism. The `architecture`
phase preserves cartographer's graded semantics for warnings, errors, and
low-confidence authority, but only `finding_class=deterministic` findings at
error/blocker severity can hard-fail when
`TEST_GENIE_ARCHITECTURE_GATE=high-confidence` (default) and cartographer
reports high authority. Operators can use `TEST_GENIE_ARCHITECTURE_GATE=off`
for an advisory rollout or `all` for strict deterministic gating across
low-authority targets. Heuristic findings still surface prominently and drive
the nudge, but they do not fail CI.

## Cross-references

- [`intent-alignment`](intent-alignment.md) — the vertical axis (PRD ↔
  requirements ↔ domains ↔ code) that complements this structural, horizontal
  architecture validation model.
- `docs/scenario-qa/methods/audit/screaming-architecture-audit.md` — the audit
  *lens* (when structural-cohesion auditing applies, when it backfires). This
  doctrine is the *why* behind its L5 "programmatic drift checks" maturity rung.
- `scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md`
  — the executable procedure an agent follows.
- `scenarios/prompt-manager/store/skills/packs/core/scenario-improvement-campaign/SKILL.md`
  — the authorized development loop that can use D; profile ranks candidate
  repairs while the approved target determines acceptance.
- `scenarios/test-genie/docs/phases/architecture/README.md` — the architecture
  phase that produces the C findings.
