## Steer focus: Invariant Discovery & Enforcement

Prioritize **discovering, clarifying, and safely enforcing invariants** in this scenario.

An invariant is a condition that **must always be true** for the system to behave correctly (for example: “this list is always sortable by date”, “this route requires an authenticated user”, “this state always has an ID”).

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve overall completeness and reliability.

---

### **1. Understand Intended Behavior & Domain Rules**

* Use the **PRD, operational targets, technical requirements, and existing tests** to infer the scenario’s intended behavior.
* Identify:
  * core domain concepts
  * valid / invalid states
  * constraints on inputs and outputs
  * expectations around identity, ownership, and lifecycle
* Distinguish between:
  * **intended invariants** (what should always be true)
  * **incidental quirks** or bugs (what happens to be true today, but shouldn’t be)

Only promote conditions to invariants if they are consistent with the PRD, usage patterns, and user expectations.

---

### **2. Discover Existing Implicit Invariants**

* Inspect code paths, components, handlers, and data flows to find **assumptions** such as:
  * “this value is never null here”
  * “this ID always corresponds to an existing record”
  * “this function is only called for authenticated users”
  * “this state is always in one of these finite cases”
  * “this collection never contains duplicates”
* Look for:
  * repeated checks or guards that hint at important conditions
  * branches that assume certain shapes of data or state
  * error handling paths that reveal what “must not happen”
* Identify **critical invariants** whose violation would cause:
  * data corruption
  * security issues
  * crashes
  * severe UX failure

Treat these as high-priority candidates for clarification and enforcement.

---

### **3. Clarify and Name Invariants**

* Give important invariants **explicit, descriptive names** (in comments, types, helper functions, or tests), such as:
  * `invariant: userMustOwnResource`
  * `invariant: activeScenarioHasValidConfig`
  * `invariant: jobQueueItemsHaveUniqueIds`
* Capture the intent in **short, clear language**:
  * what must be true
  * for which domain objects or flows
  * under which conditions (if not global)
* Prefer concise, high-signal descriptions over verbose commentary.

This step is about making the system’s **rules visible and understandable** to future agents and humans.

---

### **4. Encode Invariants Safely**

Where appropriate and safe, encode invariants using mechanisms already natural to the scenario, such as:

* **Types / schemas** (e.g. stricter shapes, non-nullable fields where truly required)
* **Runtime checks / assertions** (guarding critical entrypoints or transitions)
* **Validation functions** with clear, reusable semantics
* **Tests** that:
  * assert invariants hold in normal flows
  * verify invariant violations fail in controlled ways
* **State modeling** that makes illegal states unrepresentable where feasible

When enforcing invariants, prefer **incremental, low-risk improvements** over sweeping changes. Ensure tests reflect the intended rules without becoming brittle.

---

### **5. Handle Violations Gracefully**

* Decide what should happen when an invariant is violated:
  * fail fast in trusted, internal flows (e.g. development, test)
  * fail gracefully for user-facing flows (clear error states, no data corruption)
* Improve error handling paths so they:
  * avoid leaking internal details
  * provide enough context for debugging
  * maintain system integrity
* Where an invariant cannot yet be safely enforced, **document the risk** and, if helpful, add tests or comments that highlight the gap.

Avoid introducing noisy or user-hostile failures; focus on **protecting correctness** while preserving a reasonable experience.

---

### **6. Avoid Over-Constraining the System**

* Do **not** turn current bugs, temporary workarounds, or incomplete behavior into “invariants.”
* Ensure new invariants:
  * align with real usage and PRD intent
  * do not block legitimate future extensions
  * do not break realistic edge cases that should be supported
* When in doubt, prefer:
  * clearly documenting a candidate invariant
  * adding targeted tests
  * leaving room for future refinement

The goal is to **stabilize true rules**, not freeze accidental behavior.

---

### **7. Maintain Scenario Constraints**

* Do **not** add unrelated features in this phase.
* Do **not** change core product behavior unless:
  * it clearly contradicts the PRD or operational targets, and
  * the change is necessary to correct an invariant violation.
* Ensure all changes:
  * keep the scenario fully functional
  * respect existing tests and completeness metrics
  * stay aligned with the scenario’s architecture and goals

If you identify a necessary but **large, risky invariants overhaul**, describe it in comments or notes instead of partially implementing it.

---

### **8. Output Expectations**

You may:

* add or refine validation logic and guardrails
* strengthen or clarify type-level guarantees
* add or improve tests around critical invariants
* introduce well-named helpers or modules that enforce important rules
* add concise documentation or comments describing key invariants

You **must**:

* preserve and, where possible, improve behavior correctness
* avoid regressions and unnecessary user-visible disruptions
* ensure important invariants are more **visible, enforced, and testable** after this loop
* avoid superficial edits that do not strengthen the system’s real guarantees

Focus this loop on **practical, targeted invariant discovery and enforcement** that makes the scenario more robust, predictable, and safe to evolve over time.