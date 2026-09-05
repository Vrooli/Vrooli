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
| D | Improvement campaign | "Drive this scenario TO a goal, over time, handholding the agent" | architecture-cartographer *(campaign)* | **longitudinal, stateful** |

A, B, and C answer *whether* something is right — they are **validation**
(point-in-time). D *drives* a scenario toward right — it is **process**
(longitudinal). Making cartographer "just a test phase" would discard D, which is
its entire reason to exist: large screaming-architecture refactors fail when an
agent cannot track the surface area (this is what happened to the swarm-manager
refactor). The tracker is the substrate that handholds the agent through it.

Code quality (B, tidiness-manager) is a **separate axis** — file/function
metrics, not structural cohesion. It is not folded into the architecture audit;
it may become an ingest source for the tracker later, but it is not wired today.

## The two-layer model and the seam

```
  test-genie  ──ArchitectureFinding──▶  architecture-cartographer
  (the camera)                          (the project plan)
  stateless audit AGGREGATOR            stateful campaign TRACKER
```

- **test-genie is the camera.** It runs the per-surface validators it already
  orchestrates (A) plus a structural `architecture` phase that delegates to
  cartographer's read-only audit (C), and emits **one normalized findings
  report**. It has no memory: each run is a fresh photograph.
- **architecture-cartographer's campaign domain is the project plan.** It
  ingests that report, tracks every finding through a lifecycle, hands the agent
  a profile-ranked worklist, and on each re-audit reconciles by stable ID. It
  holds all the memory: history, lifecycle, regressions.

The normalized findings report is the **seam** — the shared
`ArchitectureFinding` contract (`packages/proto/schemas/architecture/v1/`).
Detection has no memory; tracking does all of it.

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

When a single audit surfaces more findings than one pass can responsibly fix,
test-genie's output **nudges** the agent to open a tracked campaign rather than
fixing ad-hoc. The loop (driven by the `scenario-improvement-campaign` skill):

```bash
# 1. AUDIT — take the first photograph
test-genie execute <scenario> --preset architecture-audit --json > audit-1.json
#    Above the single-pass threshold, the output appends:
#      ⚠️  Improvement campaign recommended …
#         architecture-cartographer campaign create <scenario> --from-audit audit-1.json

# 2. CREATE — ingest the photograph
architecture-cartographer campaign create <scenario> --from-audit audit-1.json

# 3. NEXT — the handholding. --profile picks the ordering:
#    fast (cheapest path to green) | balanced (default) | long-term (root-cause-first)
architecture-cartographer campaign next <campaign-id> --profile balanced

# 4. FIX + MARK OFF
architecture-cartographer campaign resolve <campaign-id> --finding <afid> --note "…"

# 5. RE-AUDIT — new photograph, reconcile by stable ID
test-genie execute <scenario> --preset architecture-audit --json > audit-2.json
architecture-cartographer campaign reaudit <campaign-id> --from-audit audit-2.json
#    → gone → validated · persists → open · (re)appeared → flagged regression

# 6. REPEAT → CLOSE
architecture-cartographer campaign status <campaign-id>
architecture-cartographer campaign close <campaign-id>
```

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
  — the driving loop for responsibility D: profile (ordering) vs target (stop
  condition), and the `campaign next --profile fast|balanced|long-term` knob.
- `scenarios/test-genie/docs/phases/architecture/README.md` — the architecture
  phase that produces the C findings.
