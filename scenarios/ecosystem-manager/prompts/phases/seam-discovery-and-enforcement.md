## Steer focus: Seam Discovery & Enforcement

Prioritize **discovering, strengthening, and enforcing “seams”** in this scenario.

A **seam** is a deliberate boundary where behavior can vary or be substituted without invasive changes (e.g. for testing, swapping implementations, isolating integrations, or controlling side effects).

Your goal is to make the scenario **easier to test, safer to change, and more modular** by improving where and how seams exist.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve completeness and reliability.

---

### **1. Understand Core Flows & Variation Points**

* Review the scenario’s **main workflows and responsibilities** (from PRD, operational targets, requirements, and tests).
* Identify where behavior naturally **varies or could need to vary**:
  * external services or resources
  * I/O operations (network, storage, filesystem, browser automation, etc.)
  * environment-specific behavior
  * feature flags or conditional branches
  * expensive operations or long-running tasks
* Note where these concerns are currently **entangled with core logic** instead of being isolated behind clear boundaries.

Keep these natural variation points in mind as prime candidates for stronger seams.

---

### **2. Discover Existing Seams**

* Scan the code for **existing boundaries** where behavior is already somewhat isolated, such as:
  * wrapper functions or adapters around integrations
  * helper utilities that encapsulate side effects
  * modules that centralize certain decisions or operations
* Evaluate whether these seams are:
  * well-named and clearly documented, or confusing
  * actually respected throughout the codebase, or bypassed
  * easy to use in tests, or awkward and brittle

Where seams exist but are weak or frequently violated, plan to **clarify and reinforce** them.

---

### **3. Identify Missing or Eroded Seams**

Look for areas where code is hard to test or change because seams are missing or have eroded:

* core logic directly performing side effects
* repeated low-level calls scattered across many places
* components or handlers reaching deeply into multiple layers
* duplicated integration code instead of going through a single boundary
* tests that require excessive setup because there is no clean place to substitute behavior

These are good candidates for **introducing or restoring seams** in a focused, incremental way.

---

### **4. Strengthen & Enforce Seams**

Where appropriate, improve seams so they are **clear, robust, and easy to use**:

* Group related side effects or integration calls behind **well-named, focused modules or functions**.
* Ensure callers depend on **stable, intention-revealing boundaries**, not low-level details.
* Avoid leaking external or infrastructure-specific details into core logic and UI layers.
* Make seams practical for testing:
  * callers should have a straightforward way to control or substitute behavior in tests
  * tests should be able to exercise core logic without triggering unnecessary external work

Favor **small, incremental improvements** over sweeping refactors. Improve the most impactful seams first.

---

### **5. Improve Testability & Variation at Seams**

* When you strengthen or create a seam, ensure tests take advantage of it:
  * add or refine tests that exercise core logic through the seam
  * adjust existing tests to rely on the clearer boundary instead of fragile setups
* Keep tests **expressive and intent-focused**:
  * avoid over-mocking internal details
  * prefer tests that emphasize behavior at the seam (inputs/outputs) rather than internal wiring

The goal is to make it **easy for future loops** to test new behavior by using the seams you’ve clarified.

---

### **6. Safe Refactoring Guidelines**

You may:

* rename or relocate modules to better reflect their boundary role
* extract integration or side-effectful code into more focused units
* consolidate duplicated integration logic behind a seam
* adjust call sites to use existing or newly improved seams
* update tests to reflect the new, clearer boundaries

You must:

* preserve observable behavior and user-facing workflows
* avoid changes that require broad configuration or deployment shifts in this loop
* not weaken test coverage or delete meaningful tests to “force” refactors through
* introduce seams incrementally, avoiding large, risky rewrites

If you detect a seam that **should** exist but requires a major redesign, document it clearly (in comments or notes) rather than applying a risky partial change.

---

### **7. Maintain Scenario Constraints**

* Do **not** add new product features in this phase.
* Do **not** change business rules or UX flows except where strictly required to fix issues caused by unclear boundaries.
* Ensure all changes remain consistent with the scenario’s PRD, operational targets, and test-driven requirements.
* Prefer **clarity and explicit boundaries** over clever abstractions that are hard to understand.

---

### **8. Output Expectations**

By the end of this loop, the scenario should:

* have **clearer, more intentional seams** at key variation points
* be easier to test without heavy, brittle setup
* isolate integrations and side effects behind obvious boundaries
* reduce the risk that future changes accidentally bypass or break important boundaries

Avoid superficial changes (e.g. renaming things without clarifying boundaries, or shuffling code without improving testability).

Focus this loop on **practical, targeted seam improvements** that make the scenario easier to evolve safely, especially around external dependencies, side effects, and behavior that may need to vary over time.
