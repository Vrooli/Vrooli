## Steer focus: Architecture Alignment & Refactoring (“Screaming Architecture Audit”)

Prioritize **improving the internal architecture and structure** of this scenario.

Your goal is to make the codebase’s structure and naming **clearly express the scenario’s purpose and mental model**, while keeping behavior stable.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve completeness and reliability.

---

### **1. Understand the Domain & Mental Model**

* Study the scenario’s **PRD, operational targets, and technical requirements** to understand:
  * what this scenario exists to do
  * who/what its primary “actors” are
  * the main workflows and concepts it operates on
* From this, derive a **clear, simple mental model**:
  * key domain concepts (entities, resources, operations)
  * main flows (inputs → processing → outputs)
  * natural boundaries (e.g. UI vs orchestration vs integration vs persistence)

Keep this mental model in mind as the “true architecture” you want the code to scream.

---

### **2. Map the Current Logical Architecture**

* Identify the current **logical layers and responsibilities**:
  * entrypoints (UI screens, routes, handlers, jobs, commands)
  * orchestration / coordination logic
  * domain or business rules
  * integration boundaries (APIs, services, storage, external tools)
* Note where responsibilities are:
  * clearly separated and cohesive
  * mixed together or blurred
  * duplicated or inconsistently modeled

Summarize (for yourself) how the system is *actually* structured today.

---

### **3. Inspect the Physical Structure**

* Examine the **file, folder, and module structure**:
  * how components, handlers, utilities, and domain logic are grouped
  * how names reflect (or fail to reflect) the domain
  * where cross-cutting concerns live
* Look for:
  * “god modules” or files doing too many things
  * feature logic scattered across unrelated locations
  * circular or overly tangled dependencies
  * low-level details exposed at high-level entrypoints

Your aim is to see where the **physical** structure diverges from the **mental** model.

---

### **4. Align Architecture with the Domain (“Make It Scream Its Purpose”)**

* Adjust naming, grouping, and boundaries so that:
  * the **top-level structure** makes it obvious what the scenario does
  * domain concepts are **first-class** and easy to locate
  * each module or component has a **clear, focused responsibility**
* Prefer organizing by **domain/feature** over arbitrary technical categories where it improves clarity.
* Make boundaries more explicit:
  * separate orchestration from domain rules
  * separate domain rules from infrastructure details
  * isolate integration-specific concerns behind clear interfaces or modules

Do not perform a “big-bang rewrite.” Favor **incremental, local improvements** that clearly move the architecture toward the mental model.

---

### **5. Simplify Dependencies & Reduce Coupling**

* Reduce unnecessary coupling between modules, components, or layers.
* Remove dead code, unused helpers, or obsolete paths where it is safe to do so.
* Prefer **clear, explicit dependencies** over hidden or implicit ones.
* Avoid creating deep, rigid dependency chains; keep key flows easy to trace.

When adjusting dependencies, ensure tests are updated to reflect the new structure without weakening coverage.

---

### **6. Safe Refactoring Guidelines**

You may:

* rename modules, functions, components, and types to better match the domain
* move code into more appropriate files or folders
* split overly large modules into smaller, cohesive units
* extract well-named helpers for repeated patterns
* consolidate duplicate logic into a single, clear implementation
* add or adjust tests to capture the intended behavior of newly clarified boundaries

You must:

* preserve observable behavior and user-facing workflows
* keep public contracts stable where feasible; if you change them, update all call sites consistently
* avoid refactors that require large configuration or deployment changes in this loop
* avoid weakening tests or deleting meaningful coverage to “make changes pass”

If you identify a necessary but **high-risk, broad architectural change**, describe it clearly in comments or notes rather than partially implementing it.

---

### **7. Maintain Scenario Constraints**

* Do **not** add new product features in this phase.
* Do **not** change business rules or workflows except where needed to fix obvious inconsistencies or bugs revealed by architectural clarification.
* Ensure all changes stay consistent with the PRD, operational targets, and test-driven requirements.
* Prefer **clarity, cohesion, and explicitness** over cleverness or abstraction for its own sake.

---

### **8. Output Expectations**

By the end of this loop, the scenario should:

* have a **clearer, more expressive structure**
* make it easier for future agents (and humans) to find where to change what
* show improved alignment between:
  * the domain mental model
  * the logical architecture
  * the physical file/module layout
* retain or improve test coverage and stability

Avoid superficial changes (e.g. meaningless renames or file shuffling) that do not materially improve how well the architecture communicates the scenario’s purpose and boundaries.

Focus this loop on **practical, targeted architectural improvements** that make the codebase easier to reason about, safer to extend, and more obviously aligned with what the scenario is supposed to do.
