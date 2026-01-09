## Steer focus: Assumption Mapping & Hardening

Prioritize **discovering, validating, and safely handling assumptions** in this scenario’s code, tests, configuration, and interactions.

Your goal is to make important assumptions **explicit, verified, and robust**, so the scenario is harder to break as it evolves.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve completeness and reliability.

---

### **1. Identify Hidden Assumptions**

Systematically scan the scenario (code, tests, configuration, comments) to uncover **implicit assumptions**, such as:

* **Data shape & nullability**
  * assumed properties always present on objects
  * lists assumed non-empty
  * values assumed in a certain format, type, or range

* **User behavior**
  * assumed user flows (what they do first, what they never do)
  * assumed input quality (e.g. “they won’t paste huge blobs”)
  * assumptions about permissions or roles

* **External systems & APIs**
  * responses always conforming to a certain schema
  * certain fields always returned or non-null
  * assumed latency, availability, or error behaviors
  * webhooks or events arriving in a specific order

* **Timing & environment**
  * functions assuming a particular call order
  * components assuming they mount only after some state is ready
  * assumptions about environment variables, config files, or runtime mode

* **Tests and fixtures**
  * tests relying on narrow “happy path” fixtures that hide real-world variability
  * test setups that encode assumptions not enforced in production logic

Capture (for yourself) a mental list of the most important and risky assumptions you find.

---

### **2. Evaluate Risk & Validity**

For each important assumption, consider:

* **Is this actually guaranteed?**  
  * by the domain, by upstream systems, or by explicit validation?
* **If this assumption fails, what happens?**
  * user-facing error, silent data corruption, crash, security issue?
* **How likely is failure in realistic usage?**
  * common edge case vs. rare corner vs. impossible in this domain?

Prioritize assumptions that are:

* high-impact if violated  
* plausible to fail in real usage  
* currently unchecked or only weakly implied

---

### **3. Codify Assumptions as Checks & Tests**

Turn fragile, implicit assumptions into **explicit, verifiable behavior**:

* Add or strengthen **input validation and guards**:
  * check for null/undefined/missing fields where they matter
  * validate formats, ranges, and types before use
  * provide clear, user-friendly errors when assumptions are broken

* Extend or refine **tests**:
  * add tests that confirm critical assumptions hold under normal conditions
  * add tests that show how the system behaves when assumptions are violated
  * avoid brittle tests that merely encode the assumption without exercising behavior

* Add **targeted comments or docstrings** only where necessary:
  * clarify assumptions that cannot be easily enforced in code
  * describe why certain shortcuts are safe in this scenario’s context

Prefer enforced assumptions (validation + tests) over comments alone.

---

### **4. Soften or Remove Fragile Assumptions**

Where assumptions are risky or unrealistic, make the code more forgiving and explicit:

* Replace “it must be this way” assumptions with:
  * guarded branches
  * safe defaults
  * explicit fallback behavior

* Handle edge cases gracefully:
  * empty lists
  * missing optional fields
  * slow or failing external services
  * partial or degraded states

* Avoid silently proceeding when a critical assumption is violated; favor:
  * clear error paths
  * safe early returns
  * controlled degradation of functionality

Do not introduce confusing complexity; aim for **simple, explicit, defensive** behavior.

---

### **5. Align Assumptions with Domain & PRD**

* Cross-check assumptions against:
  * the scenario’s PRD and operational targets
  * technical requirements and tests
* Remove assumptions that conflict with the domain’s intent.
* Where the domain truly guarantees an assumption (e.g. “this field always exists”), ensure this is:
  * documented in the appropriate place (e.g. type definitions, central schema, or comments)
  * validated at the boundary where data enters the system

This keeps assumptions **coherent with the scenario’s actual purpose**.

---

### **6. Maintain Scenario Constraints**

* Do **not** add new features in this phase.
* Do **not** change business rules or workflows unless they are clearly incorrect relative to the PRD and fixing them is required to handle assumptions properly.
* Avoid large-scale rewrites; prefer **localized improvements** that reduce risk and clarify behavior.
* Keep public interfaces and external contracts stable where possible; if you must change them, update all call sites consistently and retain behavior expectations.

When you discover a necessary but **high-risk or broad change**, document it clearly rather than partially implementing it.

---

### **7. Output Expectations**

You may update:

* validation and guard logic
* error handling and fallback behavior
* tests that express and enforce assumptions
* comments/docs explaining critical domain assumptions
* minor refactors that clarify and enforce assumptions

You **must**:

* preserve or improve existing behavior and completeness
* make important assumptions more **explicit, enforced, and testable**
* reduce the likelihood that future changes silently break hidden assumptions
* avoid superficial changes that do not materially improve robustness

Focus this loop on **practical, targeted assumption hardening** that makes the scenario more resilient to real-world variation and evolution, without overcomplicating the design or drifting from the PRD.

Avoid “security theater”–style changes; only introduce checks and adjustments that genuinely reduce risk or clarify behavior.
