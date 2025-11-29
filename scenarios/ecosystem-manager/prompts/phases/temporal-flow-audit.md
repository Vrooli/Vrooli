## Steer focus: Temporal Flow Audit

Prioritize **stabilizing and improving time-related behavior** in this scenario.

Your goal is to understand how the system behaves **over time**—across async operations, background work, loading flows, and lifecycle events—and make that behavior more predictable, robust, and user-friendly.

Do **not** break functionality, regress tests, or change core product behavior. All changes must maintain or improve completeness and reliability.

---

### **1. Map the Temporal Flows**

* Identify the key **time-based flows** in this scenario, including:
  * async operations (requests, jobs, tasks, background work)
  * loading sequences and initialization steps
  * periodic jobs, polling, or scheduled work
  * long-running operations or streaming flows
* Understand, for each flow:
  * what triggers it
  * what it depends on
  * what it updates
  * when and how it completes or fails

Keep a clear mental model of how time, state, and events interact in this scenario.

---

### **2. Stabilize Asynchronous Behavior & Ordering**

* Ensure asynchronous steps run in a **well-defined, intentional order**:
  * avoid relying on implicit ordering or timing coincidences
  * remove hidden assumptions like “this usually finishes first”
* Make it clear when operations:
  * may run in parallel
  * must run in sequence
  * can be safely retried
* Replace fragile timing hacks (e.g., arbitrary delays or sleeps) with **event- or state-based coordination** where possible.

Avoid introducing new timing-based workarounds; prefer explicit coordination and clear contracts.

---

### **3. Concurrency, Races & Shared State**

* Look for potential **race conditions** or conflicts, such as:
  * multiple operations updating the same state or resource simultaneously
  * overlapping requests or jobs that assume exclusive control
  * rapid repeated user actions that trigger conflicting flows
* Where appropriate:
  * add guards or checks to avoid conflicting updates
  * enforce idempotency for repeated actions
  * ensure shared state is updated in a consistent, predictable way

Use tests to prove that critical flows behave correctly when invoked quickly, repeatedly, or in parallel.

---

### **4. Initialization, Loading & Teardown**

* Review initialization and loading behavior:
  * ensure initial state is well-defined and not dependent on “lucky” timing
  * avoid flickering or rapidly changing UI states during load
  * make sure users see clear, stable feedback while data is being fetched or processed
* Review teardown or cleanup behavior:
  * ensure timers, listeners, subscriptions, or background work are correctly stopped when no longer needed
  * prevent work from continuing after the relevant context has ended

Aim for **smooth, predictable transitions** between loading, ready, and teardown states.

---

### **5. Scheduling, Polling & Retries**

* Identify any **polling loops, scheduled tasks, or recurring jobs**.
* Ensure they:
  * use sensible intervals and limits
  * handle failures gracefully
  * avoid overwhelming external services, logs, or resources
* Review retry behavior:
  * make retries intentional, bounded, and observable
  * avoid infinite or excessive retries that can cause hidden load or noisy behavior

Prefer **calm, controlled scheduling** over aggressive or ad-hoc repetition.

---

### **6. Time-Sensitive Error Handling & Recovery**

* Improve handling of time-related failures:
  * distinguish between transient errors and persistent failures
  * where appropriate, recover automatically in a controlled manner
  * provide clear user feedback when an operation is taking longer than usual
* Ensure error paths do not leave the system in a **half-updated, confusing state**:
  * either complete work correctly or fail in a way that is visible and recoverable
  * avoid silently ignoring important failures in time-based flows

Favor **fail-fast + clear recovery paths** over silent, lingering failure modes.

---

### **7. Safe Refactoring Guidelines**

You may:

* restructure async flows to make sequencing and concurrency explicit
* extract helper functions for common temporal patterns (e.g., polling, debouncing, batching)
* adjust state transitions to better reflect real-world timing and lifecycle
* clarify or simplify logic that depends on timing quirks
* add or refine tests for time-sensitive behavior using appropriate test utilities

You must:

* preserve existing user-visible behavior unless it is clearly buggy or unstable
* keep public contracts and external interfaces consistent where feasible
* avoid large, risky rewrites of core flows in this single loop
* avoid weakening or removing tests that guard important timing behavior

If you identify a major timing or concurrency redesign that is too large for this loop, document it clearly instead of attempting a partial, risky change.

---

### **8. Maintain Scenario Constraints**

* Do **not** introduce new product features in this phase.
* Do **not** change business rules or core workflows except to fix clear timing-related bugs or inconsistencies.
* Ensure all changes remain consistent with the PRD, operational targets, and test-driven requirements.
* Prefer **predictability, clarity, and robustness** over clever timing tricks.

---

### **9. Output Expectations**

By the end of this loop, the scenario should:

* handle time-based and async behavior **more predictably**
* have fewer hidden assumptions about ordering or speed
* behave more gracefully under rapid, repeated, or delayed interactions
* be easier for future agents (and humans) to reason about when debugging time-related issues

Avoid superficial changes that do not materially improve temporal stability.

Focus this loop on **practical, targeted improvements to time, sequencing, and async behavior** that make the scenario more robust, understandable, and reliable over time.