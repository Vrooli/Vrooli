## Steer focus: Failure Topography & Graceful Degradation

Prioritize **understanding how this scenario fails** and improving how it behaves under stress, error, or partial outages.

Your goals are to:
* map the **failure landscape** (“failure topography”) across key flows
* ensure failures are **safe, observable, and recoverable**
* introduce or refine **graceful degradation** so the system fails in controlled, user-respecting ways

Do **not** break functionality, regress tests, or introduce new features unrelated to failure handling. All changes must maintain or improve overall completeness and reliability.

---

### **1. Map Critical Flows & Dependencies**

* Identify the scenario’s most important flows (e.g. key user journeys, background jobs, external integrations).
* For each, clarify:
  * primary inputs
  * critical internal steps
  * external dependencies (APIs, services, storage, configuration, environment)
  * main outputs or side-effects

Keep this mental map as the canvas on which you’ll sketch the failure topography.

---

### **2. Identify and Classify Failure Modes**

For each critical flow, enumerate **realistic failure modes**, such as:

* external service timeouts or errors
* invalid or missing input
* transient network issues
* inconsistent or missing data
* race conditions or ordering issues
* resource exhaustion or rate limits

Classify them along dimensions like:
* **scope** (local vs systemic)
* **frequency** (common vs rare)
* **impact** (low vs high severity for users and data)

Avoid purely hypothetical, contrived failures; focus on **plausible, impactful scenarios**.

---

### **3. Trace Failure Paths & Outcomes**

* For selected high-impact failure modes, trace how the scenario currently responds:
  * what happens in the UI?
  * what happens in orchestration/logic?
  * what state changes occur (if any)?
  * what gets logged or signaled?
* Identify patterns such as:
  * silent failures or swallowed errors
  * partial updates that leave the system in a confusing state
  * misleading “success” surfaces
  * overly harsh failures where a softer fallback is possible

The aim is to understand how failures propagate today, not to immediately fix everything.

---

### **4. Design and Implement Graceful Degradation**

For the most important failure paths, adjust behavior so the system:

* **fails safe**:
  * avoids corrupting critical data
  * avoids leaving users in irrecoverable or confusing states
* **degrades gracefully**:
  * returns partial but clearly labeled results instead of total failure where appropriate
  * offers read-only or limited modes when full functionality is unavailable
  * defers or retries non-critical work instead of blocking key interactions

Prefer **incremental, localized improvements** that clearly reduce user and system harm over ambitious, risky rewrites.

---

### **5. Improve the User-Facing Failure Experience**

* Ensure that when failures occur, users see:
  * clear, honest messages about what went wrong (at their level of abstraction)
  * simple guidance on what they can do next (retry, refresh, wait, contact support, etc.)
* Avoid:
  * exposing raw technical details, stack traces, or sensitive identifiers
  * ambiguous or generic messages that hide meaningful distinctions (“Something went wrong” for everything)
  * dead-ends where the user is stuck without an obvious next action

Where beneficial, use consistent patterns for error states, fallback views, and recovery actions.

---

### **6. Maintain State Integrity & Recovery Paths**

* Review how state is updated when failures occur mid-flow:
  * avoid partial writes that leave the system in a contradictory state
  * prefer all-or-nothing behavior for critical operations where feasible
* Make it easier to **recover** from failures:
  * ensure important operations are idempotent where appropriate
  * avoid hidden side effects that make retries unsafe
* When you cannot make an operation fully robust in this loop, clearly document its assumptions and risks.

Do not invent complicated recovery mechanisms that require new operational runbooks; prefer simple, self-contained improvements.

---

### **7. Make Failures Observable Without Leaking**

* Ensure key failures are **visible** to operators and future agents through:
  * structured logs
  * status surfaces or flags
  * lightweight metrics or counters where applicable
* Improve existing error logging to:
  * include enough context to understand and reproduce the issue
  * avoid capturing secrets or sensitive data
  * avoid noisy, unstructured spam that obscures real issues

Add or refine tests that assert:
* critical failure paths behave as intended (both for users and internal state)
* important failures are observable in a predictable way.

---

### **8. Maintain Scenario Constraints**

* Do **not** change core product behavior or business rules except where required to fix obviously unsafe or misleading failure behavior.
* Do **not** add new features unrelated to failure handling or graceful degradation.
* Ensure all changes remain consistent with the PRD, operational targets, and test-driven requirements.
* When in doubt, choose **safer, localized adjustments** over broad systemic changes, and surface larger change recommendations in notes or comments.

---

### **9. Output Expectations**

By the end of this loop, the scenario should:

* have a clearer, better-understood **map of how and where it fails**
* respond more **gracefully** to a few of its most important failure modes
* expose failures in ways that are:
  * safe
  * debuggable
  * recoverable
* preserve or improve test coverage around failure paths

Avoid superficial changes (e.g. renaming error messages without improving behavior). Focus this loop on **practical, targeted improvements** to how the system behaves under stress, so that real-world failures are safer, clearer, and easier to work with for both users and future agents.
