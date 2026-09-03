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
  revision: 2
  createdAt: "2026-07-27T00:00:00Z"
  updatedAt: "2026-09-02T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "swarm-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager", "swarm-manager goals", "vrooli scenario"]
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
| **W0** | Every P0 operational target agrees with the approved goal, **and** every capability the goal names has a P0 operational target | Read the goal, then read `PRD.md`, then compare in both directions (§3) | `prompt-manager skill read prd-authoring` |
| **W1** | The contract validates and every requirement links to a PRD target | `business-health validate scenario <name>` | `prompt-manager skill read prd-authoring` |
| **W2** | Every status other than `planned` carries a passing validation ref | `vrooli scenario requirements validate <name>` | `prompt-manager skill read requirements-traceability-steer` |
| **W3** | The scenario runs, is safe, and holds its architecture and features | `vrooli scenario test <name>` | `prompt-manager skill read scenario-maturity-ladder` |

**W1 conformance is not W0 truth.** `business-health validate` reports `PASSED` for a contract that is internally consistent and describes the wrong product. Run W0 first, always.

---

### **3. The W0 gate**

W0 is the rung no other skill owns, so its gate is stated here in full.

1. Find every goal that names the scenario. Search names, titles, **and** descriptions — a name match does not end the search, because a second goal can name the scenario only in its description:

   ```
   swarm-manager goals list --json \
     | jq -r --arg s "<name>" '.goals[].goal
         | select((.name + " " + .title + " " + .description) | test($s))
         | .name'
   ```

   Run this even when `swarm-manager goals list` already shows a goal named after the scenario. A plain `grep` over this output is unreliable: the payload is a `{"goals":[{"goal":{…}}]}` envelope whose descriptions sit on single long lines, so a grep that returns nothing means a failed pattern more often than it means a missing goal.

   **This filter is the whole search. Do not widen it.** A goal that constrains the scenario without naming it anywhere is invisible here, and that is accepted: an open-ended semantic sweep across every goal has no stopping rule, so two agents would do different amounts of work and reach different verdicts. The named-mention search is the deterministic floor. When you believe an unnamed goal constrains the scenario, that is a finding for the problems document, not a reason to keep searching.
2. Read each goal the search returned: `swarm-manager goals get --name <goal>`.

   The default output carries the description, the targets, the status, and the scope counts. Read it directly; `--json` adds nothing this gate needs.
3. Read every operational target in `scenarios/<name>/PRD.md`, at all three priorities.
4. Read the reversals in the scenario's decisions document: `docs/internal/SWARM_MANAGER_WORK.md` when it exists, otherwise `docs/SWARM_MANAGER_WORK.md`. Skip this step when neither exists.
5. Compare in both directions. W0 **fails** when any row below is true.

| Condition | Example |
|---|---|
| A P0 target names a capability that a goal directs you to remove | Goal says "drop Huginn entirely"; `OT-P0-002` promises multi-platform scraping |
| A goal directs you to add a capability that no **P0** target names | Goal says "manual paste-first entry"; no operational target covers manual entry |
| A goal directs you to add a capability that the PRD names only at P1 or P2 | Goal makes ideation extraction the v0; the PRD carries it as `OT-P2-00N` |
| A decision record supersedes a P0 target and the target still states the superseded position | `D-00N` retires an approach; the target still promises it |

Row 3 is why step 3 reads all three priorities. A capability the goal makes load-bearing, parked at P2, is a contract that disagrees with the goal about what must ship. Demotion is a contract defect, not a prioritization detail.

6. When W0 fails, stop. Run no gate under W0. Repair the contract through `prd-authoring`.

**W0 evidence is the compared quotations**, not a command's stdout. Record the goal sentence and the operational-target line that contradict each other, in the shape shown in §6. W0 is the one rung whose gate is a comparison rather than a command.

**No goal means W0 is unverifiable.** Record that in the problems document (§6) and file the goal before you continue. A scenario with a contract that nothing can contradict is not a scenario with a true contract.

**A stale goal fails W0 too.** When the goal, not the PRD, is the artifact you suspect, the contradiction still stands and the ladder still stops. Raise the conflict with the operator. Do not pick a side without a decision.

---

### **4. Walking the ladder**

**Phase 1 — Locate.**

**Entry criteria:** A scenario exists and a change is proposed against it.

**Actions:**
0. When an improve skill routed you here with a rung and a sensor reading (`improve-skill-authoring` §5), take that rung as the hypothesis and its reading as the evidence to beat; still run the rung's gate to confirm before repairing, and record the reading in the problems document so the next cycle can compare.
1. Read the scenario's problems document (§6) for the rung a prior session recorded.
2. Run the W0 gate (§3).
3. When W0 passes, run each remaining gate in order and stop at the first failure.

**Exit criteria:** One rung is named, and the evidence that named it is captured.

**Artifacts:** The rung and its gate evidence, written to the problems document (§6).

**Phase 2 — Repair.**

**Entry criteria:** Phase 1 named a rung.

**Actions:**
1. Read the skill that owns the rung. Read no other rung's skill.
2. Repair at that rung only. Work under a broken rung is discarded when the rung above it changes.

**Exit criteria:** The rung's gate passes.

**Artifacts:** Whatever the owning skill produces.

**Phase 3 — Re-measure.**

**Entry criteria:** Phase 2 closed a rung.

**Actions:**
1. Re-run every gate from W0 down.
2. Expect a repair to re-open a rung under it. A W0 overhaul invalidates the W1 registry that linked to the retired targets, and invalidates the W2 evidence attached to those requirements.

**Exit criteria:** Every gate passes, or a new rung is named and Phase 2 repeats.

**Artifacts:** The updated rung record in the problems document (§6).

```
   W0 contract true? ──NO──▶ prd-authoring (contract overhaul) ──┐
         │ YES                                                    │
         ▼                                                        │
   W1 obligations conformant? ──NO──▶ prd-authoring (registry) ───┤
         │ YES                                                    │
         ▼                                                        │
   W2 evidence real? ──NO──▶ requirements-traceability-steer ─────┤
         │ YES                                                    │
         ▼                                                        │
   W3 implementation ──▶ scenario-maturity-ladder (R0–R4) ────────┤
         │                                                        │
         ▼                                                        │
   done ◀── every gate passes ◀── re-measure from W0 ◀────────────┘
```

The loop back to W0 is the rejection path. A repair is a hypothesis that the layer above it was true. Re-measuring is how that hypothesis fails.

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

**The problems document is one file per scenario.** Use `scenarios/<name>/docs/internal/PROBLEMS.md` when it exists. Otherwise use `scenarios/<name>/docs/PROBLEMS.md`. Read that file at session start and write the rung record to the same file. Never create a second problems document — a forked problem log hides the rung from the next session, which is the failure this loop exists to prevent.

Write the located rung at session end, in this shape:

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
| Ladder-on-a-lie — climbing R0–R4 against a PRD that contradicts the approved goal | The rungs measure conformance to a false target, so closing them builds the capability the operator directed you to remove | Run the W0 gate before you read any test result |
| Conformance as truth — treating `business-health validate` `PASSED` as W0 evidence | That command checks structure and linkage. It never reads the goal | Run the W0 gate. It is the only gate that reads the goal |
| Contract editing to match code — rewriting a P0 target so drifted code validates | The contract stops describing the product and starts describing the implementation, so nothing can contradict the code again | Decide which side is wrong. `requirements-traceability-steer` owns drift, and its anti-gaming bans apply |
| Rung inversion — writing tests to close W2 while W1 is open | The evidence attaches to obligations that do not link to the contract, so the evidence is discarded when W1 is repaired | Repair W1 first, then re-measure |
| Gate skipping by reading — declaring a rung satisfied from a document that describes it | Documents state intent. Gates measure state | Run the rung's gate and record its evidence: command stdout for W1–W3, the compared quotations for W0 |
| Gate widening — running the gates under a failed rung to build a fuller picture | The outputs describe a stack that the repair above them will change, so the picture is stale before it is read | Stop at the first failing gate. Re-measure after the repair (Phase 3) |

---

### **8. Output expectations**

The only file you may change is the scenario's problems document (§6), and the only change you may make to it is adding or updating its `## Work ladder` section. Writing that section is required, not optional — see the first line of the must-list.

You must:
- Run the W0 gate before any other gate.
- Write the rung record before the session ends. A located rung that is not written is lost, and the next session repeats the gate from nothing.
- Stop at the first failing gate and run no gate under it.
- Repair only the rung the gate named.
- Record the evidence that named the rung.
- Re-measure from W0 after every repair.

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
| `goals get` prints a title and an empty `Results:` block | A `swarm-manager` build from before 2026-07-27, when `goals get` rendered the title only | Restart the scenario: `vrooli scenario restart swarm-manager`. Read the goal with `--json` until it is back |
| The §3 step 1 search returns nothing | A failed pattern, or no goal exists | Re-run the `jq` filter exactly as written before you conclude the scenario has no goal. Empty output from a hand-rolled `grep` over the JSON envelope is the common cause |
| A goal's description defers to another document ("see `orchestration-summary.md`") | The goal is a pointer, not the whole directive | Read the referenced document and compare against it too. The goal text plus what it incorporates is the contract side of the comparison |
| The scenario has no goal | Work reached the tree without passing through swarm-manager | W0 is unverifiable. Record it in the problems document (§6) and file the goal |
| The scenario has no `PRD.md` | Contract never authored | W0 and W1 both fail. Drive `business-health wizard` per `prd-authoring` |
| `business-health validate` passes on a starter-template PRD | Template text is conformant and says nothing real | W1 passes, W0 fails. §3 catches it; the P0 targets name no capability the goal names |
| Several goals touch one scenario | Overlapping initiatives | Compare against every one. A contradiction in any single goal fails W0 |
| A gate command does not exist for the scenario's shape | Scenario predates the contract tooling | Record the gap in the problems document (§6) and treat the rung as failing, not passing |

**Compression target.** The W0 gate is prose because its rules are still settling. `business-health phase` is specified and unbuilt, and it is the destination that owns this computation. Promote per `path:docs/agent-system/PROMOTION_LADDER.md` when the W0 rules stop changing.
