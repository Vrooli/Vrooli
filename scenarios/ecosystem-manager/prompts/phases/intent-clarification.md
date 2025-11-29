## Steer focus: Intent Clarification

Prioritize **making the purpose and intent of the scenario’s code and behavior explicit**.

Your goal is to ensure future agents (and humans) can quickly answer:

> “Why does this exist, what is it for, and when should it change?”

Do **not** change core behavior, regress tests, or add new features. All changes must maintain or improve completeness, reliability, and clarity.

---

### **1. Reconstruct the Intended Purpose**

* Use the **PRD, operational targets, technical requirements, and tests** to infer the intended behavior of:
  * key components and modules
  * routes, handlers, commands, and jobs
  * domain rules and workflows
* For important pieces of logic, identify:
  * what problem it solves
  * what inputs and outputs are expected
  * when it is supposed to be used
  * which invariants or constraints it relies on

Keep this as your reference mental model for “what this code is trying to do.”

---

### **2. Make Intent Visible Through Naming**

* Adjust names so they reflect **purpose over implementation details**:
  * prefer domain- or behavior-oriented names over generic ones
  * avoid vague labels like `doStuff`, `helper`, `data`, `manager`, `utils`
* Name things to answer questions like:
  * “What does this represent?”
  * “When would I use this?”
  * “What decision is made here?”
* Keep public contracts stable where practical. If renaming public APIs or types:
  * ensure all call sites are updated consistently
  * preserve behavior and expectations
  * avoid breaking external integrations

Do not rename purely for style or personal taste; rename only where intent becomes materially clearer.

---

### **3. Document Non-Obvious Intent (Explain the “Why”)**

* Add **concise comments or docblocks** where intent is not obvious from the code alone, especially when:
  * there is a non-trivial tradeoff or design decision
  * behavior depends on subtle invariants or constraints
  * edge cases or historical context matter
* Focus comments on **why** something is done, not line-by-line descriptions of **what** the code does.
* Keep comments:
  * accurate
  * brief
  * updated to match the implementation
* Remove or update misleading, outdated, or redundant comments that obscure true intent.

Avoid over-commenting or narrating straightforward code; prioritize places where lack of intent is likely to cause mistakes.

---

### **4. Clarify Flows and Responsibilities**

* Make the **main flows and responsibilities** easier to understand:
  * group related steps into clearly named functions or modules
  * separate “what we’re doing” from “how we’re doing it” where it clarifies intent
  * ensure the primary entrypoints read like a high-level narrative of the behavior
* Where control flow is convoluted or surprising, simplify or restructure it so the intent of each branch is clear.
* Ensure each component, function, or module has a **single, understandable responsibility** that aligns with its name.

Do not introduce new patterns or architectures; clarify and refine what’s already there.

---

### **5. Identify and Handle Unclear or Obsolete Code**

* Look for code whose purpose is:
  * unclear
  * outdated
  * unused
  * inconsistent with the PRD or tests
* Where it is clearly **dead or redundant**, and safe to do so:
  * remove it and adjust tests if needed
* Where the purpose is uncertain or tightly coupled:
  * add a short note or TODO describing the ambiguity
  * avoid speculative deletions that could break behavior

When in doubt, prefer **explicitly marking uncertainty** over aggressive removal.

---

### **6. Align Intent with Behavior and Tests**

* Ensure that names, comments, and tests **accurately reflect what the code actually does**.
* If you find discrepancies where:
  * the name/description says one thing, but
  * the tests and implementation clearly do another,
  
  then:
  * clarify which behavior is correct according to the PRD and operational targets
  * adjust names, comments, or tests so they all agree on the true intent
* Where behavior is clearly wrong relative to the PRD, you may:
  * fix it if the change is local and safe, updating tests accordingly, or
  * document the mismatch and recommend a focused follow-up change

Avoid masking inconsistencies with wording; resolve or surface them clearly.

---

### **7. Maintain Scenario Constraints**

* Do **not** introduce new product features or major behavior changes in this phase.
* Do **not** relax validation, security, or robustness in the name of simplicity.
* Ensure all changes remain consistent with the PRD, operational targets, architectural goals, and test-driven requirements.
* Prefer changes that **reduce future confusion and surprise** over clever abstractions.

---

### **8. Output Expectations**

You may update:

* names of functions, components, modules, and types
* comments and docblocks, especially around non-obvious logic
* grouping and extraction of functions to make flows easier to follow
* tests where they misrepresent intent or lack clarity about behavior
* small structural adjustments that make responsibilities more obvious

You **must**:

* preserve observable behavior and keep the scenario functional
* maintain or improve tests and coverage related to clarified intent
* reduce ambiguity and surprise for future changes
* make it easier to see where to modify the system for a given requirement

Avoid superficial edits (e.g. renaming things without improved clarity, reformatting code, or shuffling files) that do not materially improve understanding.

Focus this loop on **practical, targeted intent clarification** so that the codebase tells a truthful, easy-to-understand story about what it is doing and why.
