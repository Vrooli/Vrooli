## 🧹 **Phase: Refactor & Structural Improvement**

During this loop, prioritize **code quality, structure, and maintainability** across this scenario’s implementation.
Do **not** change the intended behavior, regress tests, or weaken safety guarantees; all changes must maintain or improve completeness.

Focus on producing a **cleaner, clearer, easier-to-extend codebase**, guided by the following principles:

### **1. Preserve Behavior & Intent**

* Treat existing passing tests, the PRD, and operational targets as the **source of truth** for behavior.
* Refactor in ways that **preserve external behavior**:
  * same user-visible workflows and outcomes
  * same API contracts, validation rules, and side effects
* Do **not** relax tests or assertions to “make refactors pass.” If you must adjust tests, ensure the new expectations are *more accurate* to the PRD, not looser.

### **2. Improve Code Structure & Readability**

* Shorten or simplify **overly long functions, components, or modules** by extracting well-named helpers.
* Improve naming so that **functions, variables, components, and files clearly express their purpose**.
* Group related logic together and separate unrelated concerns:
  * move cohesive logic into shared utilities where appropriate
  * keep modules focused and well-scoped
* Prefer code that is **self-explanatory** over code that needs heavy comments to be understood.

### **3. Reduce Duplication & Complexity**

* Identify duplicated logic and consolidate it into **shared, reusable abstractions** where it genuinely reduces complexity.
* Reduce unnecessary branching and deeply nested conditionals; favor **early returns**, clear guard clauses, and simple control flow.
* Eliminate dead code, unused props/parameters, unneeded indirection, and obsolete TODOs that are no longer relevant.
* Avoid over-engineering: do **not** introduce abstractions that aren’t clearly justified by current or near-term needs.

### **4. Align with Existing Patterns & Architecture**

* Follow the **established patterns, conventions, and architectural decisions already present** in this scenario.
* Do **not** introduce new major frameworks, state management libraries, or architectural styles during this phase.
* Do **not** migrate technologies (e.g., switching UI frameworks, build tools, or data layers).
* When in doubt, **extend and refine what exists** rather than inventing an entirely new structure.

### **5. Strengthen Tests & Safety (Without Gaming Metrics)**

* Where refactors touch important behavior, **add or tighten tests** to lock in the improved structure and prevent regressions.
* Prefer meaningful tests that cover realistic flows over trivial or redundant tests added solely to increase scores.
* Do not remove tests unless they are truly redundant or incorrect; when you remove or replace a test, ensure the behavior it covered is still validated.

### **6. Scope & Change Size**

* Prefer **small, coherent refactors** that can be understood and reviewed as a single, logical improvement.
* Focus on **high-leverage areas**:
  * core flows
  * shared utilities
  * heavily reused components or modules
* Avoid broad, cosmetic renames or mass edits that don’t significantly improve clarity, structure, or safety.

### **7. Maintain Scenario Constraints**

* Do **not** introduce new product features in this phase.
* Do **not** change user-facing copy, UX flows, or visual design except where necessary to support structural improvements.
* Ensure all changes remain aligned with the scenario’s PRD, operational targets, and test-driven requirements.

### **8. Output Expectations**

You may update:

* module boundaries and file organization
* function/component structure and naming
* shared utilities and helpers
* error handling and edge-case handling
* test organization and coverage (when supporting refactors)

You **must**:

* keep the scenario fully functional and aligned with its PRD
* avoid regressions and weakened safety
* leave the code **simpler, clearer, and easier to change**
* avoid gaming metrics or making superficial changes that don’t materially improve structure

Focus this loop on delivering **practical, targeted refactors** that reduce complexity, remove duplication, and improve clarity, so future loops (and agents) can build on a stronger foundation.
