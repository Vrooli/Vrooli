---
name: "scientific-debugging"
description: "Hypothesis-driven debugging methodology: generate falsifiable hypotheses, design experiments to validate them, and systematically narrow to root cause. Produces regression tests and documented findings."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","debugging","testing","methodology","investigation-technique"]
  icon: "bug"
  status: "active"
  revision: 29
  createdAt: "2026-02-03T02:40:00Z"
  updatedAt: "2026-09-06T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager ai-search", "swarm-manager backlog", "swarm-manager record", "swarm-manager records", "swarm-manager scenarios"]
  origin:
    kind: "authored"
---
## Practice focus: Scientific Debugging

Find and fix the cause of an uncertain defect using evidence that can distinguish plausible explanations. Scale the method to the risk and uncertainty; do not turn diagnosis into a mandatory paperwork exercise.

### Scope

Use when the cause is unclear or prior fixes failed. For an obvious cause, fix and verify directly. Performance, security, corruption and races may need specialized measurement or containment alongside this method. Follow the task's authority and the owner of the affected behavior.

### Process

1. **Orient.** Define expected and observed behavior and reuse relevant prior work already found. Reproduce when practical; an intermittent or destructive failure may require logs, a safe proxy or a narrower observation instead.
2. **Discriminate.** Form the most plausible explanation and identify what observation would contradict it. Consider alternatives when the evidence does not distinguish them; do not invent a fixed number. Run the cheapest informative check, then revise the explanation if evidence disagrees.
3. **Repair.** Fix the understood cause at its owner, including a larger cross-component change when needed and authorized. Investigate or repair infrastructure blocking the work; do not stop at a report when implementation is in scope. Preserve real safety and authority boundaries.
4. **Verify and retain.** Exercise the affected behavior and relevant counterexamples with validation proportional to the change. Add a regression when it protects meaningful behavior; a failing test before the fix is useful when practical, not a prerequisite for every repair. Report the cause, change, evidence and remaining uncertainty in the existing work artifact; follow AGENTS.md for non-trivial work records. Check related patterns when evidence or impact warrants it.

If the result contradicts the hypothesis, return to diagnosis. If the cause remains uncertain, report what was observed and the next discriminating check; continue independent authorized work.

### Common failure modes

| Failure | Response |
|---|---|
| Repeated speculative changes | Choose a check that distinguishes causes. |
| A plausible explanation is treated as proof | Test the mechanism or state the uncertainty. |
| A narrow patch leaves the actual failure in place | Change the owning contract or implementation when evidence supports it. |
| Diagnostic ritual displaces the fix | Reuse evidence and choose only artifacts and checks that protect the outcome. |
