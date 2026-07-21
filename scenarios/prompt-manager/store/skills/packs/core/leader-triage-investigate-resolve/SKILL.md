## Practice focus: Leader Triage-Investigate-Resolve Pipeline

Orchestrate the full lifecycle of delegated problem resolution: triage severity and blast radius, drive hypothesis-based investigation through team members, then coordinate the fix with regression testing and verification. This is a three-phase gated pipeline over two leaf skills, with an investigation cycle inside Phase 2.

Required reading:
- `prompt-manager skill read team-coordination-leader-led` — the shared gated-pipeline contract (phase shape, gate/rework semantics, delegation template, convergence checklists, shared anti-patterns). This skill adds only what is specific to triage-investigate-resolve.
- `prompt-manager skill read triage-methodology` — the Triage-phase leaf (severity matrix, blast radius).
- `prompt-manager skill read scientific-debugging` — the Investigate/Resolve leaf (hypothesis testing, fix + regression test).

---

### 1. Phase-to-leaf mapping

| Phase | Leaf skill / activity | Artifact |
|---|---|---|
| Triage | `triage-methodology` | Triage report (severity, blast radius, response decision) |
| Investigate | `scientific-debugging` (cycles until a hypothesis is confirmed) | Hypothesis log + confirmed root cause with evidence |
| Resolve | `scientific-debugging` Phase 5 (fix + verify) | Regression test, fix, root-cause documentation |

The leader typically performs Triage personally — severity and blast-radius judgment should not be delegated — but may delegate symptom gathering. Investigate and Resolve are delegated per the shared delegation template.

### 2. Pipeline entry decision table

| You have... | Severity known? | Root cause known? | Entry point |
|---|---|---|---|
| Bug report, severity unknown | No | No | Phase 1: Triage |
| Bug report, severity assessed | Yes | No | Phase 2: Investigate |
| Root cause confirmed, fix needed | Yes | Yes | Phase 3: Resolve |
| Not a bug / known fix | N/A | N/A | Apply the known fix directly |
| Performance issue | N/A | N/A | Not this pipeline — needs profiling methodology |
| Security incident | N/A | N/A | Not this pipeline — containment-first response |
| Single-agent debugging | N/A | N/A | Not this pipeline — use `scientific-debugging` directly |

### 3. Gate criteria

**Gate 1 (Triage → Investigate):** severity assigned (P0-P3) with reasoning; blast radius documented (isolated / cross-scenario / expanding); response urgency set (immediate / same-day / next-sprint / backlog); problem reproducible (or intermittent conditions documented). If it cannot be reproduced, assign monitoring and re-triage on new evidence; if severity is unclear, escalate for a second opinion.

**Gate 2 (Investigate → Resolve):** root cause identified and confirmed by experiment; evidence supports it (not just correlation); the root cause explains all observed symptoms; the hypothesis log is documented. If it explains only some symptoms, there may be multiple bugs — triage the unexplained symptoms separately.

### 4. Investigation cycle

Phase 2 cycles until a hypothesis is confirmed. Generate 2-3 falsifiable hypotheses, test the most likely with a minimal experiment, and act on the result:

| Test result | Status | Next action |
|---|---|---|
| Evidence supports hypothesis | Confirmed | Proceed to Gate 2 |
| Evidence contradicts hypothesis | Rejected | Delegate a new hypothesis with the updated evidence |
| Evidence inconclusive | Needs refinement | Design a more targeted test; re-run |
| Unexpected evidence found | New information | Fold into the next hypothesis cycle |
| 3+ hypotheses rejected | Stalled | Re-examine symptoms; consider re-triaging as a different problem |

### 5. Rework triggers

| Signal | During phase | Action |
|---|---|---|
| Cannot reproduce the problem | Triage | Assign monitoring; re-triage on new evidence |
| 3+ hypotheses rejected | Investigate | Re-examine symptoms; consider re-triaging |
| Root cause explains only some symptoms | Investigate | Likely multiple bugs; triage the rest separately |
| Fix does not resolve the bug | Resolve | Return to Investigate — root cause was wrong/incomplete |
| Fix introduces new regressions | Resolve | Return to Triage for the new issue |
| Investigation reveals an architectural problem | Investigate | Escalate; may need `leader-explore-plan-implement` instead |

### 6. Resolve completion and output

Resolve completes when: a failing regression test was written before the fix and passes after; the fix addresses the root cause, not symptoms; the full test suite passes; the original bug no longer reproduces; the root cause is documented (bug → root cause → fix → prevention → related areas).

Covers leader-coordinated problem resolution through triage, investigation, and fix. Does not cover single-agent debugging, performance profiling, security incidents, feature work (`leader-explore-plan-implement`), or upstream prioritization. Produce the three phase artifacts (or verify a pre-existing one at partial entry); record how many hypothesis cycles ran and check for the same pattern elsewhere in the codebase.
