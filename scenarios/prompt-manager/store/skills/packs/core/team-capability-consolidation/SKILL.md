## Practice focus: Team Capability Consolidation

Turn a team's hand-maintained state into a scenario, then re-derive the team's shape from what that scenario now enforces. The second half is the point: a roster that exists to hand state between members can collapse once gates enforce the ordering.

Required reading:
- `path:docs/agent-system/OPERATING_GRAPHS.md` — §"State belongs to scenarios; prose holds judgment" owns the four content classes, the read-time rule, and the `state-in-prose` defect class. This skill extends that rule from documents to rosters; it does not restate it.
- `prompt-manager skill read ecosystem-fit`

Optional reading:
- `prompt-manager skill read agent-system-audit scenario-capability-extraction team-tool-mapping`
- Phase 3 reframe lenses: `prompt-manager skill read screaming-architecture-audit boundary-of-responsibility-enforcement cognitive-load-reduction`

---

### **1. When to Use This Skill**

| Situation | Use this? | Instead use |
|---|---|---|
| A team produces little despite a large roster and canon | Yes | — |
| A team costs more to orient in each cycle, whatever it ships | Yes | — |
| A team hand-maintains files that look like a database | Yes | — |
| A new scenario absorbs work a team currently does by hand | Yes | — |
| "Is the agentic layer meeting our objectives?" | No | `agent-system-audit` |
| One member is vague or blocked | No | `team-member-capability-architecture-audit` |
| A capability already lives in scenarios and needs its own home | No | `scenario-capability-extraction` |
| A team needs to be told a scenario exists | No | `team-tool-mapping` |
| A team's contract fails validation | No | Fix it; that is a `validate` finding |

**In scope:** team output measurement, state-surface classification, capability derivation, scenario boundary selection, roster re-derivation, cutover sequencing.

**Out of scope:** building the scenario (`scenario-generation`), placing it in the ecosystem (`ecosystem-fit`), authoring its tool skill (`team-tool-mapping`), and editing plan-of-record canon, which is operator-curated and moves by decision.

---

### **2. The Loop**

```
 Phase 1          Phase 2           Phase 3         Phase 4        Phase 5         Phase 6
 Measure ──────▶ Locate ─────────▶ Derive ──────▶ Bound ───────▶ Re-derive ────▶ Cut over
 output          state             capability     scenarios      roster          in order
    │               │
    │ proportional  │ fleet-wide pattern
    ▼               ▼
  STOP            STOP — substrate question, not this team's gap
```

Two rejection points, both early and both cheap. Most candidate teams stop at Phase 1 or 2.

---

### **3. Phase 1 — Measure output and orientation cost against doctrine**

**Entry criteria:** a team looks heavy, slow, blocked, or hard to explain.

Two triggers admit a team to Phase 2, and they are independent. Read both before you judge.

**Actions:**
1. Count what the team has actually shipped. Use its terminal artifact, not its activity — published items, filed reports, merged changes.
2. Count the volume it carries: roster size, canon line count, declared topics, declared work types.
3. Read the same volume against the previous audit record and ask whether it rose in a cycle where the team's scenario coverage grew (`path:docs/agent-system/FRAMEWORK_HEALTH.md` §"Team orientation cost").
4. Apply the table.

| Reading | Verdict |
|---|---|
| Output is proportional to volume, and orientation cost is flat or falling | **Stop.** This team is not a consolidation candidate. Route to `agent-system-audit`. |
| Output is near zero while volume is large | Continue to Phase 2. |
| Orientation cost rose while scenario coverage grew | Continue to Phase 2, whatever the output ratio says. |
| No previous audit record names this team | Record this cycle's volume as the baseline. Judge on the output ratio alone. |

**Exit criteria:** you can state one sentence with two numbers in it — the output ratio, or the volume delta and the coverage growth that ran beside it.

**Artifacts:** a dated observation in the team's knowledge topic.

Zero output is a stronger signal than slow output. A team that has never completed its loop end to end has no calibration data, and every downstream loop that depends on that data is unpowered.

The second trigger exists because the first one is blind to a team that ships. Output ratio finds a team that has stalled; orientation cost finds a team that still delivers while costing more to understand every cycle, and the second failure is the one that compounds — every new capability handed to that team arrives as more to read rather than less. `marketing-crew` cleared the first gate only by accident: its output was near zero, but what actually prompted the pass was that its own system had grown past what an operator could hold in mind, and a productive team in the same state would have been waved through.

---

### **4. Phase 2 — Locate hand-maintained state**

**Entry criteria:** Phase 1 continued.

**Actions:**
1. List the team's shared files and its plan-of-record data tables.
2. Classify each surface with the table below.
3. Survey the same surface across every team in the store.
   - If most teams carry the pattern: **stop.** This is a substrate question about the team system, not a missing capability for this team. Raise it as a `capability-work`.
   - If one or two teams carry it: continue. Accretion in a minority of teams marks the teams whose work product has no scenario.

| Surface holds | Classification | Destination |
|---|---|---|
| Evidence, observations, scans | Judgment input | Stays a knowledge topic |
| Charters, philosophies, ranking criteria | Judgment frame | Stays prose |
| Records with a status and a lifecycle | Operational state | Moves to a scenario |
| Counters, registers, coverage, history | Operational state | Moves to a scenario |
| A rule written as prose that nothing can check | Capability requirement in disguise | Becomes a gate |

**Exit criteria:** every surface is classified, and the operational rows are listed separately.

**Artifacts:** the classification list, recorded with the Phase 1 observation.

The last row is the one agents miss. A rule that says a thing "must" happen, with no code that can refuse when it does not, is a requirement waiting for a home.

---

### **5. Phase 3 — Derive the capability**

**Entry criteria:** Phase 2 produced at least one operational row or one unenforceable rule.

**Actions:**
1. Read each operational surface. Write what it was trying to be.
2. Read each unenforceable rule. Write what it would refuse if it could.
3. State the capability as the union of those two lists.
4. Restate the capability the way a member would recognize it, not the way the current files are arranged. Run whichever lens fits — all three are written for code and apply to a roster unchanged.
5. Check the capability against the team's stated mission. Drop anything the mission does not ask for.

| Lens | Ask of the team |
|---|---|
| `screaming-architecture-audit` | Does the team's shape announce what it is for, or only how it was assembled? |
| `boundary-of-responsibility-enforcement` | Which member owns each decision, and where do two members share one? |
| `cognitive-load-reduction` | What must a member read before it can act at all? |

**Exit criteria:** one paragraph naming the capability, and a list of the gates it enforces.

**Artifacts:** the capability statement, which becomes the new scenario's PRD purpose line.

Derive the capability from the evidence rather than from a blank page. The files and the unenforceable rules already describe the product; they describe it badly, which is why they are files.

Step 4 decides whether the rest of the pass is worth running. A capability named after the existing file layout reproduces that layout in a scenario and buys nothing; the roster collapses in Phase 5 only when the new name makes the old separations look arbitrary. This is a screaming-architecture exercise performed on a team instead of a package tree, which is why the lens list is borrowed rather than restated.

---

### **6. Phase 4 — Bound the scenarios**

**Entry criteria:** a capability statement exists.

**Actions:**
1. Split the capability by the table below. Two capabilities that differ in both failure mode and clock belong in different scenarios.
2. Name each seam as a single question, not a shared table.
3. Route placement to `ecosystem-fit`.

| Split by | Ask | Same answer ⇒ same scenario |
|---|---|---|
| Failure mode | What goes wrong, and who notices? | Yes |
| Clock | How often does state change, and for how long does one unit of work live? | Yes |
| State shape | Documents, long-running schedules, or rendered assets? | Yes |

**Exit criteria:** one or more scenario boundaries, each with its seams stated as questions.

**Artifacts:** boundary rationale, which becomes the scenario's `ARCHITECTURE.md` boundary section.

Adjacency is not a boundary. Capabilities that appear in the same workflow often fail differently and tick differently, and merging them produces one scenario that serves neither.

---

### **7. Phase 5 — Re-derive the roster**

**Entry criteria:** the scenario's domain map and gates are documented. The scenario does not need to be built.

This is the phase that distinguishes this skill. Do not update the roster to match the scenario. Re-derive it.

**Actions:**
1. List every separation between current members.
2. Classify each with the table below.
3. Collapse the collapsible ones. Keep the controls.
4. Reassign every work type, topic, and state surface owned by a removed member.

| Separation exists because | Classification | Action |
|---|---|---|
| State had to hand off between stages | Pipeline-stage | **Collapse.** The gates now enforce the ordering. |
| Same mechanics, different field value | Lane | **Collapse.** Lane is a parameter on the record. |
| One member checks another's work | Adversarial | **Keep.** This is a control, and the gate does not replace it. |
| Different cadence and different failure mode | Clock | **Move** to the team that owns that clock. |
| Canon judgment versus production | Judgment | **Keep.** The member that wants approval must not own the canon. |

**Exit criteria:** a new roster where every remaining separation is a control, a clock, or a judgment boundary.

**Artifacts:** the updated team contract, member files, and roles.

The discriminator is: gates replace pipeline-stage separation and never replace adversarial separation. A producer that declares its own claims needs a different agent to find the claims it did not declare.

---

### **8. Phase 6 — Cut over in order**

**Entry criteria:** the new roster is decided and the team is paused.

**Actions, in this order:**
1. Settle the scenario's CLI surface.
2. Author its tool skill against that surface. Route to `team-tool-mapping`.
3. Run the state import. Verify counts per source file.
4. Edit the team contract. Remove the absorbed declarations.
5. Collapse the roster. Delete the removed members' files.
6. Propose the canon edits by decision.
7. Run the team's contract validators.
8. Resume the team.

**Exit criteria:** validators report findings only against documents that are queued for a canon decision.

**Artifacts:** the migrated contract, and a `PROBLEMS.md` entry in the scenario for anything left open.

Two ordering rules carry the risk:

- **Never remove a state declaration before the importer has consumed it.** The declared files are the importer's source. Remove them first and the import has nothing to read.
- **Change structure now, change numbers later.** Carry decision caps, pending ceilings, budgets, and cadence across unchanged. They were tuned for the old shape, and the new shape has produced no data yet.

---

### **9. Knowledge Capture**

| Artifact | Lands in |
|---|---|
| Output ratio and state classification | The team's knowledge topic, dated |
| Capability statement | The new scenario's `PRD.md` |
| Boundary rationale and rejected splits | The scenario's `docs/concepts/ARCHITECTURE.md` |
| Roster decisions and what each separation was | The scenario's `docs/internal/SWARM_MANAGER_WORK.md` |
| Anything left open at cutover | The scenario's `docs/internal/PROBLEMS.md` |
| A capability nobody will build yet | A `capability-work` decision |

Do not write a standalone audit report. Findings that live outside durable docs freeze one session's view and rot.

---

### **10. Anti-Patterns**

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| Update the roster to match the new scenario | Keeps separations that existed only to hand off state | Re-derive: classify every separation, collapse the pipeline ones |
| Merge the checker into the producer | Removes the compensating control for self-declared work | Collapse pipeline stages; keep adversarial separation |
| Delete the team's state files with the contract edit | The importer's source disappears | Import, verify counts per file, then delete |
| Rewrite the operating model before the scenario is real | Encodes a command surface you are guessing at | Structural edits first; the document follows the built contract |
| Recalibrate caps and budgets during the merge | Old numbers fit the old shape; the new shape has no data | Carry numbers unchanged; recalibrate from observed output |
| Treat accreted files as a naming or convention problem | Fixes the symptom and leaves the cause | Ask what capability the files substitute for |
| Silence the contract validator mid-migration | Hides real drift alongside expected noise | Expect findings against the not-yet-edited document; fix the document |
| Claim the team now improves automatically | Overstates it — data compounds, procedure does not | Separate what arrives as data from what still needs authored work |

---

### **11. Output Expectations**

You may:
- Edit the team contract, member files, and roles.
- Create and remove members, and reassign their work types, topics, and state surfaces.
- Repair references to deleted scenarios and skills.

You must:
- Preserve every state surface an importer has not yet consumed.
- Keep adversarial separation.
- Reassign every ownership reference held by a removed member, and confirm no reference points at a member that no longer exists.
- Route plan-of-record canon edits through the owning work type.
- Report the validator state after the pass, including findings you left open and why.

You must not:
- Delete a state surface before its import is verified.
- Author the scenario's tool skill before its command surface is settled.
- Change numeric calibration in the same pass as the structural change.

---

### **12. Troubleshooting & Edge Cases**

**The validator's error count rises after the roster collapse.** This is expected. Removing declarations from the contract removes the backing for edges the operating-model document still declares. The count falls to zero when the document is rewritten. Confirm the findings name one file; findings spread across member files mean the reassignment in Phase 5 is incomplete.

**Members that survive still reference removed topics.** Their `topics.json`, `RESPONSIBILITIES.md`, and `HEARTBEAT.md` are edited separately from the team contract. Search every surviving member for the removed topic and decision names before you run the validator.

**The pass surfaces drift that predates it.** Members commonly read topics the contract never declared. Reconcile it in the same pass; you are already editing both surfaces, and carrying it forward into a smaller roster hides it.

**A member's whole topic surface moves to the scenario.** Its declared reads and outputs become empty. That is the correct end state when the member's records now live in the scenario. Keep the member if its separation is a control.

**The team is not paused.** Pause it before Phase 6. A running team writes to surfaces that the import is reading.
