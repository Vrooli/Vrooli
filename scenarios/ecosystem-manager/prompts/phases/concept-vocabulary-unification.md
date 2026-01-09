## Steer focus: Concept Vocabulary Unification

Prioritize **unifying the language of the scenario** so that the same concept is referred to consistently across code, UI, tests, and documentation.

Your goal is to reduce **naming drift and conceptual ambiguity**, making the scenario easier for both humans and agents to understand and evolve.

Do **not** change core behavior, workflows, or business rules. All changes must maintain or improve completeness, pass tests, and preserve the scenario’s intent.

---

### **1. Discover the Core Domain Vocabulary**

* Start from the **PRD, operational targets, and technical requirements** to identify:
  * the primary domain concepts (nouns)
  * key actions and workflows (verbs)
  * important roles, entities, and resources
* Treat this as the **authoritative vocabulary** for the scenario’s domain, unless strong evidence suggests otherwise.

Keep a short internal list (for yourself) of the most important domain terms and what they mean.

---

### **2. Map Existing Names to Concepts**

* Scan the scenario’s:
  * core modules and components
  * API interfaces and handlers
  * types, models, and DTOs
  * tests and fixtures
  * user-facing labels and messages
* Identify where:
  * the **same concept** is represented by multiple names (e.g. “job”, “run”, “task”)
  * a **single name** is used for **different concepts** (overloaded or ambiguous)
  * names diverge from the domain language in the PRD without good reason.

Your objective is to understand the current **concept → name** and **name → concept** mappings.

---

### **3. Choose Clear, Domain-Consistent Terms**

* For each important concept, choose **one primary term** that:
  * matches the PRD / domain language where possible
  * is concise, descriptive, and unambiguous
  * works well across code, tests, and UI
* Avoid vague or generic names (e.g. “data”, “item”, “info”) where a domain-specific term is available.
* Where different concepts share a name, **split the vocabulary** so each concept gets its own distinct term.

Do not invent elaborate terminology; prefer **simple words** that match how a practitioner in this domain would talk.

---

### **4. Align Code, Tests, and UI with the Chosen Vocabulary**

* Incrementally update names to reflect the chosen vocabulary:
  * types, interfaces, and models
  * functions, props, and parameters
  * internal constants and enums
  * test names and descriptions
  * user-facing labels and messages
* Keep changes **cohesive**: when you rename a concept, update its uses in that part of the codebase so the language is consistent.
* Ensure tests still pass and remain meaningful; adjust them to reflect the unified vocabulary, not just mechanically rename.

Avoid broad, high-risk renames that span the entire repository in one step. Prefer **localized, safe improvements** that meaningfully reduce confusion.

---

### **5. Resolve Ambiguity at Boundaries**

* Pay special attention to **integration and boundary layers**:
  * public APIs and handlers
  * key components reused across screens
  * events, messages, or commands
* Make sure the terms exposed at these boundaries:
  * align with the domain mental model
  * clearly signal what each thing represents
  * avoid leaking low-level implementation jargon into higher layers.

Where external systems impose naming you cannot change, adapt internally with **clear mapping points** (e.g. adapters, translators) rather than spreading mixed vocabulary everywhere.

---

### **6. Preserve Behavior and Intent**

You may:

* rename variables, functions, types, components, and files to better match the chosen vocabulary
* adjust UI labels and messages for clarity and consistency
* update tests to use unified terminology
* add brief comments where a name choice might not be obvious from context

You must:

* preserve existing behavior and user workflows
* keep public contracts stable where they are consumed externally; if you must change them, update all call sites in this scenario consistently
* avoid weakening tests or removing coverage just to make renames easier
* ensure the scenario’s completeness and test health are maintained or improved

If you identify naming or conceptual issues that require a **large, cross-cutting change**, prefer to document them clearly (e.g., in notes or comments) rather than partially applying a risky rename.

---

### **7. Maintain Scenario Constraints**

* Do **not** add new product features during this phase.
* Do **not** change business rules, permissions, or domain logic except where a minor adjustment is needed to resolve an obvious inconsistency uncovered by vocabulary alignment.
* Keep all changes aligned with the PRD, operational targets, and test-driven requirements.

Favor **clarity and conceptual honesty** over cleverness or brevity.

---

### **8. Output Expectations**

By the end of this loop, the scenario should:

* use a **coherent, unified vocabulary** for its core concepts
* have reduced ambiguity between code, tests, and UI language
* make it easier for future agents (and humans) to:
  * understand what each part does
  * find where to change behavior
  * reason about the domain model

Avoid superficial or noisy changes (e.g. renaming purely stylistic symbols or local variables with no conceptual impact). Focus on **meaningful vocabulary unification** that reduces confusion and strengthens the shared mental model of the scenario.
