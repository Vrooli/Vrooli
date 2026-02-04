## Steer focus: Change Axis & Evolution Resilience Audit

Prioritize **making the scenario easier and safer to evolve over time**.

Your goal is to identify the **main ways this scenario is likely to change** (“axes of change”) and adjust the structure so those changes are as **localized, cheap, and low-risk** as possible.

Do **not** break functionality, regress tests, or introduce new product features. All changes must maintain or improve completeness and reliability.

---

### **1. Understand the Scenario's Likely Axes of Change**

* **If `docs/internal/SEAMS.md` exists**, read the Change Axes section first to understand what has already been documented.

* Study the scenario's **PRD, operational targets, and technical requirements** to infer:
  * which parts of the system are most likely to change (e.g. data sources, workflows, policies, UI flows, integrations, pricing, limits)
  * which areas are core and stable vs experimental or evolving.
* From this, identify a small set of **primary change axes**, such as:
  * “new workflow variants or steps”
  * “new integration targets or providers”
  * “new user roles or permissions”
  * “UI changes without engine changes”
  * “policy / rule updates without structural rewrites”

Keep these axes in mind as you evaluate and refactor the code.

---

### **2. Map Where Changes Currently Land (Change Topography)**

* Inspect the current implementation to see **where changes for each axis would need to be made**:
  * which files, modules, or components must be touched
  * whether a single conceptual change requires edits in many places (“shotgun surgery”)
  * whether responsibility for a change axis is scattered or centralized.
* Identify:
  * areas where change is already **nicely localized**
  * areas where change would be **fragile, surprising, or cross-cutting**.

Summarize (for yourself) how “expensive” each axis of change currently is.

---

### **3. Localize Change Along Clear Extension Points**

* Adjust structure so that each primary change axis has **obvious, constrained places** where changes should go.
* Where appropriate, introduce or clarify **extension points**:
  * well-named modules or functions that encapsulate a particular policy, rule, or variation
  * clear configuration or mapping tables that gather related variations together
  * small, focused components or handlers that own a specific kind of change.
* Avoid over-abstraction. Prefer **simple, explicit structures** that make future edits easy and obvious, not generic frameworks that are hard to understand.

The goal is that future changes along a given axis mostly require edits in **a few clear locations**, not scattered adjustments.

---

### **4. Reduce Shotgun Surgery & Cross-Cutting Entanglement**

* Look for change patterns where:
  * a single conceptual change forces updates across many unrelated modules
  * responsibilities for one axis of change are mixed with others.
* Where safe, refactor to:
  * pull axis-specific logic into cohesive units
  * separate concerns so that unrelated axes no longer share the same implementation space.
* Keep refactors **incremental and well-tested**; avoid large, speculative rewrites.

---

### **5. Clarify Stable vs Volatile Areas**

* Distinguish between:
  * **stable cores**: logic that is unlikely to change frequently
  * **volatile edges**: logic expected to evolve (e.g. feature toggles, rules, configuration, adapters, UI variations).
* Make this distinction clearer in code structure and naming:
  * stable cores should feel solid, minimal, and reusable
  * volatile edges should be easy to find and modify without risking the core.
* Where useful, add light comments or documentation that hint at **which modules are intended to change** and which are intended to remain stable.

---

### **6. Strengthen Tests Around Change Axes**

* Add or refine tests that:
  * encode the intended behavior for key change axes
  * ensure that “easy to change” areas still respect invariants and boundaries.
* Prefer tests that describe **behavior under variation** (e.g. multiple rule or config cases) rather than tests tied tightly to existing implementation details.

This helps future changes remain safe even as the implementation evolves.

---

### **7. Documentation**

Update the **Change Axes** section of `docs/internal/SEAMS.md` to record your findings:

* The code is the source of truth. Verify existing claims against actual code before extending.
* Correct any inaccuracies and extend with your new discoveries.
* Create the `docs/internal/` directory if needed.

Include:
* Primary axes of change identified
* Current cost of change along each axis (localized vs. scattered)
* Structural adjustments made to localize changes
* Areas that still need work

---

### **9. Output Expectations**

By the end of this loop, the scenario should:

* have more **obvious, localized places** to change for its primary evolution axes
* require fewer scattered edits for a single conceptual change
* make it clearer which parts of the code are **stable core vs evolving edges**
* retain or improve test coverage around critical change points.

Avoid superficial structural churn (e.g. moving code without clarifying change behavior) that does not meaningfully improve how easily and safely the scenario can evolve over time.

Focus this loop on **practical, targeted structural improvements** that make future changes cheaper, safer, and more predictable along the scenario’s most important axes of evolution.
