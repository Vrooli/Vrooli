## Steer focus: Boundary-of-Responsibility Enforcement

Prioritize **ensuring that each part of the scenario is only doing the work it is responsible for**.

Your goal is to keep behavior the same while tightening **who owns what**: presentation, coordination, domain rules, integrations, and cross-cutting concerns.

Do **not** break functionality, regress tests, or introduce new product features. All changes must maintain or improve completeness and reliability.

---

### **1. Identify the Major Responsibility Areas**

Scan the scenario and identify the main responsibility “zones” that already exist, such as:

* **Entry/presentation**: UI components, routes, handlers, commands, CLI or job entrypoints.
* **Coordination/orchestration**: glue code that wires steps together, sequences operations, and manages flows.
* **Domain rules**: core logic that defines how the scenario behaves conceptually (business rules, transformations, decisions).
* **Integrations/infrastructure**: code that talks to external services, storage, file systems, networks, or frameworks.
* **Cross-cutting concerns**: logging, error handling, metrics, tracing, configuration access.

You don’t need to invent a new architecture; work with the structure that already exists and make responsibilities clearer and more consistent within it.

---

### **2. Detect Responsibility Leaks**

Look for places where responsibilities have **bled across boundaries**, for example:

* Presentation/entry code performing **domain decisions**, complex validation, or data reshaping that belongs deeper in the system.
* Domain logic depending directly on **transport or UI details** (request/response objects, component state, styling concerns).
* Integration concerns (storage, APIs, low-level IO) scattered into many modules instead of flowing through clear boundary points.
* Cross-cutting concerns (logging, metrics, feature flags) intertwined with core logic instead of being applied at seams.
* “Helper” utilities that mix multiple unrelated responsibilities in one place.

Make a mental map of the worst leaks and prioritize those that increase complexity, coupling, or risk of bugs.

---

### **3. Move Logic to Its Proper Home**

Refactor code so that each responsibility lives in the most appropriate place:

* Move domain rules and decisions **out of** UI/presentation-only code into reusable domain or service modules.
* Move integration details (e.g. specific API calls, database calls, file access) into dedicated modules or adapters instead of leaving them in high-level flow or domain code.
* Extract cross-cutting concerns (logging, metrics, configuration lookups) into helpers or boundary points where they don’t obscure core logic.
* Keep coordination/orchestration focused on **ordering and wiring**, not on implementing raw rules or external protocols.

Prefer **small, incremental moves** over large rewrites:
* extract a function
* move it to a better file
* call it from the original location

Behavior should remain the same; only the **location and clarity of responsibilities** should change.

---

### **4. Clarify Interfaces Between Responsibilities**

Make boundaries explicit and easier to reason about:

* Ensure data passed between layers is **simple and intentional** (clear parameters, well-named structures), not entire contexts or unfiltered blobs.
* Avoid leaking low-level concerns upward (e.g. raw response objects, transport-specific errors, internal IDs that shouldn’t leave a layer).
* Reduce reliance on global mutable state or shared singletons that blur who owns what.
* Where appropriate, introduce **thin adapters** that translate between domain concepts and integration or presentation details.

Keep interfaces as small and clear as possible while still practical.

---

### **5. Consolidate & Simplify Responsibility Patterns**

Where you see multiple competing patterns for similar responsibilities:

* Choose the **clearest existing pattern** and align other call sites to it.
* Merge near-duplicate functions or modules that serve the same responsibility into a single, well-named implementation.
* Avoid introducing new abstraction layers unless they clearly reduce responsibility leakage and complexity.
* Prefer explicit, straightforward code over clever indirection that makes ownership ambiguous.

The goal is for future agents (and humans) to quickly see **where to put new logic** and **where to look when changing behavior**.

---

### **6. Tests & Safety Nets**

You may:

* Add or adjust tests so that:
  * domain rules are testable without going through UI or integration layers
  * critical flows remain covered after responsibilities are moved
* Improve test naming and grouping so it’s obvious which responsibility a test is validating.

You must:

* Preserve existing behavior and public contracts wherever feasible; if you must adjust a contract, update all call sites consistently.
* Avoid weakening or deleting meaningful tests just to make refactors pass.
* Keep the scenario in a fully working, non-regressed state with all operational targets and requirements still satisfied.

If you discover a responsibility tangle that would require a broad, risky change, document the recommended restructuring in comments or notes instead of partially applying it.

---

### **7. Maintain Scenario Constraints**

* Do **not** add new end-user features in this phase.
* Do **not** change core workflows or business rules except where necessary to fix obvious inconsistencies caused by responsibility confusion.
* Ensure all changes respect the scenario’s PRD, operational targets, and existing test-driven requirements.
* Favor **steady, incremental tightening** of boundaries over disruptive reorganizations.

---

### **8. Output Expectations**

By the end of this loop, the scenario should:

* Have **clearer ownership** of responsibilities across modules and layers.
* Make it obvious where to add or modify behavior for presentation, coordination, domain rules, integrations, and cross-cutting concerns.
* Show reduced responsibility leakage and less mixing of unrelated concerns in the same place.
* Retain or improve test coverage and stability.

Avoid superficial changes (e.g. pure renames or file shuffling) that do not materially improve **who is responsible for what**.

Focus this loop on **practical, targeted boundary clarifications** that make the scenario easier to evolve safely and correctly over many future agent loops.
