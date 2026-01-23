## Steer focus: Progress Continuity & Interruption Resilience

Prioritize **making work safely stoppable and resumable** for both users and agents.

Your goal is to ensure that when someone (or something) is in the middle of using this scenario, they can be **interrupted and later resume without losing progress, context, or intent**.

Do **not** break functionality, regress tests, or introduce unrelated new features. All changes must maintain or improve completeness and reliability.

---

### **1. Identify Core Progress Flows & Checkpoints**

* Identify the scenario’s **main progress-based journeys**, such as:
  * creating, editing, or configuring something over multiple steps
  * long-running operations or jobs
  * agent-driven workflows that span multiple actions or iterations
* For each, clarify:
  * what “progress” actually means (what has been achieved so far)
  * what the **natural checkpoints** are (safe points to pause/resume)
  * what “completion” looks like

Keep this mental model in mind as the basis for continuity and resumption.

---

### **2. Make Progress State Explicit**

* Distinguish between:
  * **ephemeral UI state** (e.g. focus, scroll, open panel)
  * **meaningful progress state** (e.g. steps completed, decisions made, data entered)
* Ensure important progress is **represented explicitly**:
  * clear flags, statuses, or phases
  * stored state that can be inspected and reloaded
  * identifiable “current position” in a flow
* Avoid relying on fragile, implicit signals (e.g. “if this field is non-empty, we must be on step 3”).

The system should be able to explain “where we are” in any important flow.

---

### **3. Persist Progress at Safe Boundaries**

* For important flows, ensure progress is **safely persisted** at sensible checkpoints, such as:
  * after completing a meaningful step
  * after confirming a choice
  * after a long-running operation finishes or fails
* Prefer checkpoints that:
  * are **idempotent** (re-applying them doesn’t break things)
  * capture enough context to resume without guessing
  * avoid partially-applied side effects that are hard to reason about

If full automatic persistence is too risky in this loop, at least **make the checkpoint concept clear** and prepare the structure for it.

---

### **4. Design Clear Resume Entrypoints**

* Ensure that when a user or agent returns to a partially completed flow, the scenario can:
  * locate their **current progress state**
  * present a **clear next step** (or explicit choice of next steps)
  * recap enough context so they remember what they were doing
* Avoid dropping users or agents into ambiguous states where it’s unclear whether:
  * something is already done
  * something needs to be redone
  * something is safe to retry

Make resumption **unsurprising and confident**: “You were here, doing this. Here’s what’s next.”

---

### **5. Handle Interruptions & Restarts Gracefully**

* Consider interruptions such as:
  * app/page reloads or navigation away
  * network glitches or backend restarts
  * agent loops being paused, retried, or rescheduled
* Ensure that after an interruption:
  * important progress is not lost
  * duplicate actions do not create inconsistent or confusing results
  * users and agents are not forced to start from scratch unless truly necessary
* Prefer **fail-safe** behavior where, in ambiguous cases, the system guides the user/agent through confirming or reconciling state instead of silently guessing.

---

### **6. Clarify Behavior of Repeats, Retries & Partial Runs**

* Where relevant, define how the scenario behaves when:
  * an action is retried
  * a partially completed flow is re-entered
  * a step is repeated (intentionally or accidentally)
* Ensure that:
  * repeated attempts are either **idempotent** or clearly communicated
  * users/agents don’t accidentally double-apply actions or create confusing duplicates
  * the system has a consistent stance on “redo vs continue”

Document or encode these behaviors so future changes don’t break continuity assumptions.

---

### **7. Maintain Scenario Constraints**

* Do **not** add entirely new product flows or major features in this phase.
* Do **not** change business rules or semantics of completion unless necessary to fix obvious contradictions revealed by continuity concerns.
* Ensure all changes stay aligned with the PRD, operational targets, and test-driven requirements.
* Prefer **incremental improvements** to continuity and resumption over large, risky rewrites.

If you discover the need for a broader redesign, describe it clearly rather than partially implementing it.

---

### **8. Output Expectations**

You may update:

* how progress and status are represented and stored
* how multi-step or long-running flows track their current position
* how users/agents are reintroduced into in-progress flows
* error and status messages related to partial completion or retries
* tests that cover interruption, resumption, and retry behavior

You **must**:

* keep the scenario fully functional and non-regressed
* improve the ability to pause and resume meaningful work
* reduce the likelihood of lost progress or duplicated actions
* make progress and continuation points clearer to both users and agents

Focus this loop on **practical, targeted improvements** that make it safe and natural to stop and resume work without losing the plot.

Avoid superficial changes that do not materially improve continuity, resumability, or confidence when returning to in-progress work.
