## Steer focus: Cognitive Load Reduction**

Prioritize **making the code easier to understand, reason about, and safely modify**.

Your goal is to reduce **developer cognitive load**: how much context, jumping around, and mental juggling is required to understand and work with this scenario.

Do **not** break functionality, regress tests, or change user-facing behavior. All changes must maintain or improve completeness and reliability.

---

### **1. Optimize for Local Readability**

* Make it possible to understand what a function/module does by reading **as little code as possible in one place**.
* Prefer straightforward, explicit code over clever or “compressed” code.
* Inline or simplify logic where indirection adds confusion without meaningful reuse.
* Ensure the “happy path” is easy to follow and not buried in branching, flags, or edge-case handling.

Ask:  
> “If a new contributor opened this file, could they understand what’s going on within a minute?”

---

### **2. Simplify Control Flow**

* Untangle complex conditional logic where possible:
  * reduce deeply nested conditionals
  * prefer early returns or guard clauses over pyramid-shaped logic
  * split large, multi-purpose functions into smaller, cohesive helpers
* Make branching decisions explicit and clearly named.
* Avoid unnecessary flags or mode switches that are hard to track mentally.

The goal is for the flow of execution to feel **obvious and predictable**, not like a puzzle.

---

### **3. Prune Over-Abstraction & Excess Indirection**

* Identify abstractions that:
  * exist “just in case”
  * wrap other calls without adding real clarity
  * require multiple jumps to see what actually happens
* Where safe, **collapse over-generalized abstractions** back into simpler, concrete code.
* Avoid adding new layers of abstraction unless they clearly reduce duplication *and* increase clarity.

Favor **“simple and clear”** over **“generic and clever.”**

---

### **4. Improve Naming & Intent Signaling**

* Rename functions, variables, modules, and types so they:
  * reflect what they actually do, not how they’re implemented
  * distinguish clearly between similar concepts
  * avoid abbreviations or ambiguous terms
* Ensure names communicate **intent**, not just mechanics:
  * e.g. `selectEligibleItems` is better than `filterList`
* Align names with the scenario’s PRD, domain language, and operational targets.

The name alone should give a strong hint of why something exists.

---

### **5. Reduce Scattering of Related Logic**

* Identify features or flows whose logic is **spread across many distant locations**.
* Where safe and appropriate:
  * co-locate related pieces
  * group helpers that logically belong together
  * reduce the number of files a developer must open to understand a single behavior
* Avoid spreading “small bits” of a feature across many cross-cutting utilities unless it truly improves reuse and clarity.

The goal is to lower the number of mental “tabs” required to track a change.

---

### **6. Clarify State & Data Flow**

* Make it clear:
  * where important state is defined
  * who owns it
  * who is allowed to change it
* Avoid surprising side effects and hidden mutations where possible.
* Prefer explicit input/output relationships:
  * functions that clearly take inputs and return outputs
  * fewer hidden dependencies on ambient/global state
* Where state transitions are important, make them easy to trace and, when useful, add tests or comments that capture the expected lifecycle.

This helps both humans and agents reason safely about changes.

---

### **7. Safe Refactoring Guidelines**

You may:

* split large functions/modules into smaller, well-named units
* collapse unnecessary wrappers and indirections
* reorder code to bring related pieces closer together
* improve naming, comments, and lightweight documentation
* adjust tests to better match clarified behavior (without weakening coverage)

You must:

* preserve observable behavior and user-facing workflows
* keep public interfaces stable where feasible; if you change them, update all call sites consistently
* avoid deleting meaningful tests or reducing their strictness to “make changes pass”
* avoid large, high-risk rewrites in a single loop; prefer incremental steps that clearly improve clarity

If you discover areas that need substantial redesign, describe them in comments or notes rather than partially rewriting them here.

---

### **8. Maintain Scenario Constraints**

* Do **not** add new product features in this phase.
* Do **not** change business rules or domain semantics, except to fix clearly incorrect or contradictory behavior.
* Ensure all changes remain aligned with the scenario’s PRD, operational targets, and test-driven requirements.
* Prefer changes that will make future loops (and future agents) more effective and less error-prone.

---

### **9. Output Expectations**

When you’re finished, the scenario should:

* be **easier to read and navigate**
* require **fewer concepts and files** in mind to understand a given behavior
* have **clearer names**, **simpler control flow**, and **more obvious data/state paths**
* remain fully functional with tests passing and completeness preserved or improved

Avoid superficial edits (e.g. reformatting without purpose, renaming for style only) that don’t meaningfully reduce cognitive load.

Focus on **practical, targeted simplifications** that make the codebase more approachable, safer to modify, and more agent-friendly over time.