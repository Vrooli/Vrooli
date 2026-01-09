## Steer focus: Decision Boundary Extraction

Prioritize **making important decisions in the scenario explicit, intentional, and easy to locate**.

A “decision” is any place where the system **chooses between alternatives** (branches, thresholds, rules, modes, strategies, fallbacks, routing, etc.).

Do **not** break functionality, regress tests, or change business rules except to fix clearly incorrect behavior. All changes must maintain or improve completeness and reliability.

---

### **1. Identify Decisions and Their Context**

* Scan the scenario for **decision points**, such as:
  * conditionals (`if/else`, switches, guards)
  * strategy or mode selection
  * routing between different flows or implementations
  * thresholds, scoring, or ranking logic
  * feature toggles or behavior flags
* For each decision, understand:
  * **what** is being decided
  * **why** that decision exists (intent)
  * **what inputs** it depends on
  * **what outputs or behaviors** it controls

Focus especially on decisions that are:
* deeply nested  
* scattered across multiple modules  
* duplicated with slight variations  

---

### **2. Make Decisions Explicit and Named**

Where safe and appropriate:

* Extract complex or important decisions into **named helpers, functions, or modules** whose names clearly express:
  * what is being decided
  * under what conditions
* Replace inlined “mystery conditionals” with calls to clearly named decision helpers.
* Avoid overly generic names; prefer **domain-relevant, intention-revealing names**.

The goal is for future agents (and humans) to quickly see **“this is where we decide X vs Y”**.

---

### **3. Clarify Criteria, Inputs, and Outcomes**

For each important decision:

* Make dependencies **explicit**:
  * avoid hidden reliance on global state or unrelated fields
  * pass required inputs directly where possible
* Simplify conditions where possible:
  * remove redundant checks
  * merge overlapping branches that do the same thing
  * avoid deeply chained logic that is hard to reason about
* Document or encode:
  * the **criteria** used (e.g., thresholds, flags, statuses)
  * the **intended behavior** of each branch

Prefer clarity over cleverness; a slightly longer but obvious decision is better than a compact, opaque one.

---

### **4. Group Related Decisions by Domain**

Where helpful and safe:

* Collect related decisions into **cohesive modules or areas**, such as:
  * decisions about user roles / permissions
  * decisions about scenario modes / strategies
  * decisions about error handling or fallbacks
  * decisions about how data is presented or transformed
* Avoid scattering closely related decisions across many unrelated files.
* Do not over-abstract; only group decisions when it materially improves understanding and cohesion.

The aim is that each important domain concern has a **clear “home”** where its decisions live.

---

### **5. Improve Testability and Observability of Decisions**

* Add or refine tests that:
  * exercise important branches of decision logic
  * verify behavior at boundaries and edge cases
  * ensure decisions behave consistently under different inputs and states
* Where appropriate and safe, improve logging or metrics around critical decisions:
  * log enough context to understand what was decided and why
  * avoid logging sensitive data or noisy, high-volume details

Do not weaken existing tests to make refactors pass; instead, align tests with clarified behavior.

---

### **6. Maintain Scenario Constraints**

* Do **not** introduce new product features in this phase.
* Do **not** change business rules unless:
  * the current behavior clearly contradicts the PRD or operational targets, or
  * the behavior is obviously incorrect or inconsistent across the scenario.
* Keep all changes aligned with:
  * the PRD and operational targets
  * existing test-driven requirements
  * the scenario’s architectural goals

When you discover decisions that seem wrong but are risky to change, prefer to **clarify and document them**, and surface a recommendation rather than partially rewriting them.

---

### **7. Output Expectations**

You may update:

* conditionals, branching logic, and decision helpers
* function and module naming to better reflect decisions
* where decisions are physically located (moving them into clearer homes)
* tests that validate decision behavior and boundaries
* comments or lightweight documentation describing decision intent

You **must**:

* preserve observable behavior, except when fixing clearly incorrect decisions
* keep the scenario fully functional and non-regressed
* make key decisions **easier to find, understand, and test**
* avoid superficial refactors that do not materially improve decision clarity

Focus this loop on **practical, targeted improvements** that make the scenario’s decision-making structure explicit, intentional, and resilient—so future loops can reason about behavior without guessing where or why choices are made.
