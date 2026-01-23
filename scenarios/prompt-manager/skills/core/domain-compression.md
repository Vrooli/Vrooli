## Steer focus: Domain Compression

Prioritize **simplifying and clarifying the domain model** of this scenario.

Your goal is to discover the **smallest, clearest set of domain concepts** that accurately describe what this scenario does, and align the code, naming, and flows around that core.

Do **not** break functionality, regress tests, or introduce new product features. All changes must maintain or improve completeness and reliability.

---

### **1. Reconstruct the Domain’s Core**

* Study the scenario’s **PRD, operational targets, and technical requirements** to understand:
  * what problem this scenario solves  
  * who/what the primary “actors” are  
  * the main flows and outcomes it cares about
* From this, infer a **minimal domain vocabulary**:
  * a small set of core entities/concepts  
  * the primary actions on those entities  
  * key states and transitions that matter

Keep this minimal vocabulary as your reference point for the rest of the loop.

---

### **2. Identify Duplicate or Overlapping Concepts**

* Scan the codebase for **conceptual duplicates**, such as:
  * multiple names for the same idea  
  * slightly different variants of the same structure or operation  
  * parallel flows that represent the same user journey in different ways
* Look for:
  * type/shape duplication (similar objects with minor differences)  
  * function or component duplication that encodes the same domain rule  
  * “legacy” concepts that no longer match the current PRD

Your aim is to see where the domain has **fanned out unnecessarily**.

---

### **3. Collapse to a Smaller, Clearer Domain**

* Where multiple names or structures represent the same concept, **choose one clear, expressive concept** and align code toward it.
* Prefer:
  * one canonical data shape per core concept  
  * one primary name per domain idea  
  * one primary flow per common user journey
* Merge or remove secondary concepts only where it is **clearly safe** and consistent with the PRD and tests.

Avoid speculative unification; compress the domain where the intent is obvious.

---

### **4. Unify Terminology & Data Shapes**

* Align naming across:
  * types, models, interfaces  
  * functions and methods  
  * components, routes, and handlers  
  * tests and fixtures
* Ensure that when the same concept appears in multiple layers, it:
  * uses **consistent terminology**  
  * exposes **compatible shapes** or a clearly intentional mapping
* Reduce ad hoc “one-off” shapes where a shared, canonical representation would be clearer and simpler.

When adjusting shapes, update all impacted call sites and tests without weakening coverage.

---

### **5. Simplify Flows & Remove Redundant Paths**

* Identify flows that are:
  * conceptually equivalent but implemented separately  
  * minor variations that could be expressed as parameters or simple options  
  * leftover from earlier iterations and no longer aligned with the PRD
* Where safe, **consolidate these into fewer, clearer flows**:
  * one main path per core use case  
  * optional behavior expressed via explicit, well-named options
* Remove or deprecate dead or obsolete flows that are no longer reachable or meaningful.

Only remove code when you are confident it is truly unused or superseded, and tests continue to pass meaningfully.

---

### **6. Guard Against Over-Compression**

* Do **not** aggressively merge genuinely distinct concepts just to reduce file count or API surface.
* Preserve important distinctions where:
  * behavior, constraints, or meaning are materially different  
  * the PRD distinguishes them as separate targets or workflows
* Prefer **clarity over cleverness**:
  * compressed domain ≠ cryptic domain  
  * the result should be *easier* for future agents (and humans) to understand, not harder

If a more radical merging would be beneficial but risky, describe it in comments or notes rather than partially implementing it.

---

### **7. Maintain Scenario Constraints**

* Do **not** introduce new product capabilities or flows in this phase.
* Do **not** change business rules except where needed to resolve obvious inconsistencies exposed by domain cleanup.
* Ensure all changes remain aligned with the scenario’s PRD, operational targets, and test-driven requirements.
* Avoid any refactor that requires large configuration or deployment changes within this loop.

---

### **8. Output Expectations**

You may update:

* naming of domain types, entities, and operations  
* data shapes and models (where safe)  
* flow structure for core journeys  
* tests and fixtures to reflect the compressed domain  
* comments or light documentation that clarify the new, simpler domain

You **must**:

* preserve observable behavior and user-facing workflows  
* keep or improve test coverage and correctness  
* reduce conceptual duplication and unnecessary variation  
* make the core domain concepts **smaller, clearer, and easier to reason about**

Focus this loop on **practical, targeted domain simplifications** that remove redundant concepts, unify terminology, and make the scenario’s true domain feel compact, well-defined, and easy for future loops to extend correctly.
