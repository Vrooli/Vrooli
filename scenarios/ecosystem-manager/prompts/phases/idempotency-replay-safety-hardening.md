## Steer focus: Idempotency & Replay Safety Hardening

Prioritize making the scenario’s behaviors **idempotent, replay-safe, and predictable** under retries, partial failures, or repeated execution of the same action.

Do **not** introduce new features, modify core workflows, or reduce existing safety guarantees.  
All changes must maintain or improve completeness and reliability.

Your goal is to ensure that **“running something twice is no worse than running it once,”** and that replays—intentional or accidental—produce stable, correct, and unsurprising results.

---

### **1. Identify Side Effects & Points of Irreversibility**

Locate code paths where actions:

* write or mutate state
* issue external calls
* enqueue work
* trigger downstream side effects
* rely on timestamps, randomness, or implicit global state

For each such path, check whether:

* repeating the action produces duplicate, inconsistent, or corrupted state
* there is a natural **idempotency key**, stable identifier, or replay guard
* there is a clear distinction between “already done” vs “not yet done”

Do not remove legitimate side effects; instead, make them **deterministic under repetition**.

---

### **2. Establish Clear Commit Boundaries**

Clarify what counts as a **successful completion** versus a **partial attempt**.

* Ensure that partially completed flows do **not** commit incomplete state.
* Make it possible to detect whether an action was:
  * **never applied**
  * **applied once**
  * **applied already and should be a no-op**
* Prefer atomic or transaction-like behavior where appropriate, even if conceptually simulated.

Communicate commit boundaries clearly in code organization and tests.

---

### **3. Stabilize Inputs, Outputs & Execution Semantics**

Ensure the scenario behaves predictably even when:

* the same request/input arrives multiple times  
* a step is retried internally  
* a workflow resumes after interruption  
* a caller replays a command due to network or agent uncertainty

Produce **stable, reproducible results** for identical inputs, unless variation is intentional and documented.

Avoid relying on:

* current time (unless wrapped in safe semantics)
* random values without seeding or scoping
* global counters that don’t survive retries

Prefer computations that remain correct under re-evaluation.

---

### **4. Prevent Duplicate Work & Double Application**

Design safe guards so repeated action attempts:

* do not re-run expensive or destructive operations
* do not create duplicate records, events, calls, or UI effects
* do not cause cascading side effects downstream

Where appropriate, introduce:

* idempotency keys
* stable request identifiers
* memoization for expensive deterministic actions
* “already completed” short-circuit paths

Avoid adding unnecessary flags or state unless they improve correctness or clarity.

---

### **5. Improve Recovery from Partial or Interrupted Execution**

When a multi-step process can be interrupted, ensure that resuming:

* picks up exactly where it should  
* does not skip required steps  
* does not repeat irreversible mutations  
* clarifies the current state instead of guessing

This may include:

* safe checkpoints
* explicit intermediate states
* validation of what has or hasn’t been done yet

Favor small, explicit states instead of ambiguous “in progress” markers.

---

### **6. Strengthen Tests for Replay & Retry Scenarios**

Add or refine tests that:

* call the same function or flow **multiple times** with identical inputs  
* simulate partial failure followed by retry  
* assert that no duplicate records, side effects, or regressions occur  
* ensure deterministic output where appropriate

Tests should reveal:
* gaps in idempotency  
* divergent states across runs  
* unintended accumulation of effects

Do not weaken coverage to accommodate poor idempotency; improve the implementation instead.

---

### **7. Maintain Scenario Constraints**

* Do **not** change product behavior unless it produces incorrect or unsafe replay semantics.
* Do **not** introduce new user-facing features.
* Keep the scenario fully functional and regression-free.
* Ensure all changes remain aligned with PRD goals, architectural boundaries, and test-driven requirements.

When encountering a high-risk idempotency flaw that requires a substantial architectural change, document the needed improvement rather than partially implementing a risky rewrite.

---

### **8. Output Expectations**

You may update:

* state-management logic
* flow control & branching
* error-handling & retry guards
* data writes & side-effect boundaries
* deduplication or replay-detection mechanisms
* tests that validate deterministic or idempotent behavior

You **must**:

* keep observable behavior intact for single execution  
* ensure repeated or resumed executions behave predictably  
* eliminate sources of double application or inconsistent state  
* improve the scenario’s resilience to retries, replays, and interruptions  

Focus on **practical, targeted hardening** that makes the scenario robust under repetition, and ensures that both users and agents can retry actions safely and confidently.

Avoid superficial changes; prioritize modifications that materially improve **correctness under repeated execution**.
