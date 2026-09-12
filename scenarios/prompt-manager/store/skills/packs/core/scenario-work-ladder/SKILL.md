---
name: "scenario-work-ladder"
description: "Locate the highest broken layer of an existing scenario - contract (W0), obligations (W1), evidence (W2), implementation (W3) - using a runnable gate per rung, then route the repair to the skill that owns that rung. W2 is the L0-L4 traceability ladder and W3 is the R0-R4 maturity ladder; W0 and W1 are the contract rungs neither one covers."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","scenario","ladder","contract","prd","requirements","routing","methodology"]
  icon: "layers"
  status: "active"
  revision: 5
  createdAt: "2026-07-27T00:00:00Z"
  updatedAt: "2026-09-08T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill read", "vrooli scenario"]
  origin:
    kind: "authored"
---
## Practice focus: Scenario Work Ladder

Locate the highest broken layer in an existing scenario's artifact stack — contract, obligations, evidence, implementation — then repair that layer before any layer under it. Your goal is to work on the layer that is wrong, rather than the layer that is easiest to measure.

Required reading: none. Each rung names the one skill that owns it; read only the rung you land on.

Optional reading:
- `prompt-manager skill read ecosystem-fit`

---

### **1. The stack**

A scenario is four layers. Each layer is a claim about the layer under it. A false claim at any layer makes every layer under it unverifiable, because the layers under it measure conformance to the false claim.

| Rung | Layer | The claim it makes |
|---|---|---|
| **W0** | Contract — `PRD.md` operational targets | "This is what the scenario must be" |
| **W1** | Obligations — `requirements/` registry | "These promises satisfy the contract" |
| **W2** | Evidence — validation refs and tests | "These promises are kept" |
| **W3** | Implementation — code, docs, UI | "The behavior exists and holds" |

**W2 is the L0–L4 ladder. W3 is the R0–R4 ladder.** This ladder does not replace either one. It states which of them applies, and it adds the two rungs above them that neither one covers.

---

### **2. The rungs and their gates**

Every rung has a gate. W1, W2, and W3 are gated by a command. W0 is gated by a comparison between two named artifacts (§3). Two agents that run the same gate against the same scenario land on the same rung.

| Rung | Satisfied when | Gate | Owning skill |
|---|---|---|---|
| **W0** | Operational targets agree with applicable governing outcomes and their explicit priority | Compare the approved contract, operator instruction, and incorporated constraints with `PRD.md` (§3) | `prompt-manager skill read prd-authoring` |
| **W1** | The contract validates and every requirement links to a PRD target | `business-health validate scenario <name>` | `prompt-manager skill read prd-authoring` |
| **W2** | Every status other than `planned` carries a passing validation ref | `vrooli scenario requirements validate <name>` | `prompt-manager skill read requirements-traceability-steer` |
| **W3** | The affected behavior holds under the declared validation scope | Scoped Test Genie phases under `path:docs/TESTING.md` §"Ordinary iteration versus certification" | Local defects: scientific-debugging when the cause is unknown; full maturity reviews: scenario-maturity-ladder |

**W1 conformance is not W0 truth.** `business-health validate` reports `PASSED` for a contract that is internally consistent and describes the wrong product. Start at W0 for a full readiness review or a disputed contract. For a localized defect with established expected behavior, use the scoped route below.

---

### **3. The W0 gate**

Read `path:docs/agent-system/SCENARIO_DEVELOPMENT.md`
§"Authorization and change classification" for governing-target applicability.
A separate Swarm goal is not required. A name match discovers context, not authority.

1. Read the supplied approved contract or operator instruction and its incorporated artifacts.
2. Read referenced governing goals and accepted decisions when the work context requires them.
3. For a full readiness review, inspect additional goals naming the scenario with `swarm-manager goals list`.
   Read candidate details with `swarm-manager goals get --name <goal>`.
   Use JSON when the human view omits names, descriptions, or status needed for this comparison.
4. Establish each candidate's applicability and explicit supersession before treating it as a constraint.
5. Read the scenario's `PRD.md` targets at every priority and its maintained decisions document.
   Prefer `docs/internal/DECISIONS.md`; follow the scenario's documentation map for other locations.
6. Compare the applicable governing target and PRD in both directions.

| Condition | W0 disposition |
| --- | --- |
| An applicable directive requires a capability the PRD omits or contradicts. | Report the missing or conflicting outcome. |
| The PRD priority contradicts the directive's explicit release obligation. | Report the priority conflict. A request alone does not imply P0. |
| An accepted decision supersedes a target that still states the old position. | Report the stale target with the supersession evidence. |
| A matching goal is historical, proposed, or unrelated to this engagement. | Retain it as context; check incorporated standing constraints before treating it as governing. |
| Two applicable sources conflict without a recorded resolution. | Request a decision; do not choose by recency or convenience. |
| No governing target is supplied or discoverable. | Report W0 as unverifiable; do not create a goal to make it pass. |
| A required owner read is unavailable. | Preserve the unverified constraint; do not claim the full gate passed. |

When W0 fails, stop this dependency chain. Route authorized contract repair through
`prd-authoring`; a target amendment still requires approval. Independent authorized
work can continue. A scoped repair does not require unrelated goal discovery.

**W0 evidence is the compared quotations and their applicability**, not a successful
list command. Record the directive, corresponding target or omission, and any
supersession reference under §6. A stale applicable goal remains a conflict until
resolved; archived status alone neither grants work nor erases a standing constraint.

---

### **4. Walking the ladder**

**Phase 1 — Locate.**

**Entry criteria:** A scenario review or repair needs layer selection.

**Actions:**
1. Read existing problem evidence once; reuse relevant evidence from this session.
2. Choose the route:

| Evidence and purpose | Next step |
|---|---|
| Localized implementation defect with established expected behavior | Reproduce the affected behavior; use scientific-debugging if its cause is unknown, otherwise apply the known fix. Validate under `path:docs/TESTING.md`. Do not claim unmeasured W0–W2 gates passed. |
| A sensor identifies a specific broken layer | Confirm that layer with its relevant gate. |
| Expected behavior, obligations, or ownership are disputed | Inspect the highest implicated layer; start at W0 for a contract dispute. |
| Full scenario readiness or maturity review | Start at W0 and descend in order, stopping at the first failure. |

3. Escalate when evidence contradicts an upstream contract or obligation.
   Missing unrelated readiness evidence does not block a scoped implementation fix.

**Exit criteria:** One rung is named, and the evidence that named it is captured.

**Artifacts:** The rung and its gate evidence, retained under §6.

**Phase 2 — Repair.**

**Entry criteria:** Phase 1 named a rung.

**Actions:**
1. Use the repair method selected by §4's route; load its skill only when needed. A localized W3 defect does not require the maturity-review skill.
2. Repair at that rung only. Work under a broken rung is discarded when the rung above it changes.

**Exit criteria:** The rung's gate passes.

**Artifacts:** Whatever the owning skill produces.

**Phase 3 — Re-measure.**

**Entry criteria:** Phase 2 closed a rung.

**Actions:**
1. Re-measure affected gates. For an ordinary implementation fix, reuse unchanged contract and obligation evidence; run the relevant W3 regressions and phases under `path:docs/TESTING.md`. Re-run from W0 when the repair changes the contract or obligations. A complete scenario maturity review retains its full declared gates.
2. Expect a repair to re-open a rung under it. A W0 overhaul invalidates the W1 registry that linked to the retired targets, and invalidates the W2 evidence attached to those requirements.

**Exit criteria:** The declared affected gates pass with limitations recorded, or new evidence names another implicated rung.

**Artifacts:** The updated rung record in the problems document (§6).

The full review descends W0 → W1 → W2 → W3. A scoped repair re-measures
affected gates. Contract changes reopen dependent obligations and evidence.

---

### **5. Boundaries**

**In scope:** locating the broken layer of an existing scenario, and routing the repair to the skill that owns that layer.

**Out of scope:**
- Scenarios that do not exist yet. Use `ecosystem-fit` for placement, then `scenario-generation` to build. This ladder starts at the first change made against a scenario that already exists.
- Severity assessment of an incoming problem. That is `triage-methodology`. It ranks problems against each other; this ladder locates one problem in one scenario's stack.
- The repair work itself. Each rung's owning skill holds it.
- Resources. `docs/resources/maturity-migration.md` owns the M0–M5 resource ladder.

---

### **6. Memory loop**

For read-only reviews, return the rung and evidence in the report. Do not mutate
the problems document, work ledger, or runtime. The writes below apply only to
authorized repair/documentation work; reuse the active workflow's record when present.

**The problems document is one file per scenario.** Use `scenarios/<name>/docs/internal/PROBLEMS.md` when it exists. Otherwise use `scenarios/<name>/docs/PROBLEMS.md`. Read that file at session start and write the rung record to the same file. Never create a second problems document — a forked problem log hides the rung from the next session, which is the failure this loop exists to prevent.

Reuse an active workflow's durable rung and evidence record when it already
covers this task; cite its location in the handoff instead of duplicating it.
Otherwise record the located rung in the existing problems document, in this shape:

```markdown
## Work ladder

- Rung: W0
- Evidence: goal `<goal-name>` directs "drop Huginn entirely"; `OT-P0-002` promises X/Reddit/TikTok scraping
- Blocker: contract overhaul not started
- Measured: 2026-07-27
```

A rung record without its evidence is not a rung record. The next session re-runs the gate and needs the prior evidence to detect a change.

**Leave the document's existing entries alone.** A problems document written before this ladder existed holds findings at several rungs and marks none of them. Add the `## Work ladder` section and change nothing else. Those findings become actionable again as the ladder descends, so deleting or rewriting them destroys work.

---

### **7. Anti-patterns**

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| Ladder-on-a-lie — climbing R0–R4 against a PRD that contradicts the approved goal | The rungs measure conformance to a false target, so closing them builds the capability the operator directed you to remove | Resolve the contract contradiction at W0 before relying on downstream conformance |
| Conformance as truth — treating `business-health validate` `PASSED` as W0 evidence | That command checks structure and linkage, not agreement with governing intent | Compare applicable directives and targets through W0 |
| Contract editing to match code — rewriting a P0 target so drifted code validates | The contract stops describing the product and starts describing the implementation, so nothing can contradict the code again | Decide which side is wrong. `requirements-traceability-steer` owns drift, and its anti-gaming bans apply |
| Rung inversion — writing tests to close W2 while W1 is open | The evidence attaches to obligations that do not link to the contract, so the evidence is discarded when W1 is repaired | Repair W1 first, then re-measure |
| Gate skipping by reading — declaring a rung satisfied from a document that describes it | Documents state intent. Gates measure state | Run the rung's gate and record its evidence: command stdout for W1–W3, the compared quotations for W0 |
| Gate widening — running the gates under a failed rung to build a fuller picture | The outputs describe a stack that the repair above them will change, so the picture is stale before it is read | Stop at the first failing gate. Re-measure after the repair (Phase 3) |

---

### **8. Output expectations**

During layer selection, retain the rung and evidence in the report for read-only
work, or the existing authorized record under §6. The subsequent repair may change affected
implementation, tests, and documentation. A review request alone does not
authorize that repair.

You must:
- Select the route in §4; reserve a full W0-first walk for its stated triggers.
- Retain the rung and evidence before the session ends, reusing existing workflow evidence under §6.
- Stop at the first failing gate and run no gate under it.
- Repair only the rung the gate named.
- Record the evidence that named the rung.
- Re-measure affected gates after every repair; restart at W0 when the contract or obligations change (§Phase 3).

You must NOT:
- Run a gate under a failed rung.
- Repair more than one rung in a pass.
- Change a P0 operational target under any rung other than W0.
- Declare a rung satisfied without its evidence.
- Create a second problems document for a scenario that already has one.

---

### **9. Troubleshooting & Edge Cases**

| Situation | Cause | Response |
|---|---|---|
| `goals get` prints a title and empty `Results:` | The human view may omit goal content | Read the goal with `--json`; restart through the lifecycle only when authorized and evidence identifies stale runtime behavior. |
| A goal listing omits descriptions or status | The human view is insufficient for the full-review search | Inspect the JSON response; distinguish an empty result from a failed read. |
| A goal's description defers to another document ("see `orchestration-summary.md`") | The goal is a pointer, not the whole directive | Read the referenced document and compare against it too. The goal text plus what it incorporates is the contract side of the comparison |
| The scenario has no goal | No separate goal was found | Compare the supplied approved item contract or operator instruction; without any governing target, report W0 as unverifiable. |
| The scenario has no `PRD.md` | Contract never authored | Report the missing contract. Route authorized authoring to `prd-authoring`; a review stops with the finding. |
| `business-health validate` passes on a starter-template PRD | Template text is conformant and says nothing real | W1 passes, W0 fails. §3 catches it; the P0 targets name no capability the goal names |
| Several goals touch one scenario | Overlapping initiatives | Establish applicability under §3; unresolved conflicts between governing sources fail W0. |
| A gate command does not exist for the scenario's shape | Scenario predates the contract tooling | Record the gap in the problems document (§6) and treat the rung as failing, not passing |

**Compression target.** The W0 gate is prose because its rules are still settling. `business-health phase` is specified and unbuilt, and it is the destination that owns this computation. Promote per `path:docs/agent-system/PROMOTION_LADDER.md` when the W0 rules stop changing.
