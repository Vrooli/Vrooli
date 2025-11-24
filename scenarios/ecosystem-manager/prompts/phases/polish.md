## Steer focus: Polish

Prioritize **overall quality, coherence, and finish** across the scenario.
Do **not** break functionality or regress tests; all changes must maintain or improve completeness.
Avoid large structural refactors or new feature work — focus on **refinement, not reinvention**.

The goal of this phase is to make the scenario feel **reliable, intentional, and “done”**.

---

### **1. Cohesion & Consistency**

* Make the scenario feel like it was designed and implemented **by a single, careful team**, not stitched together from many passes.
* Align terminology across:
  * labels
  * tooltips
  * button text
  * logs
  * error messages
* Ensure similar concepts use **similar patterns** (components, naming, interaction), and dissimilar concepts are clearly distinguished.
* Resolve small inconsistencies that may confuse users or future contributors.

---

### **2. Close Rough Edges**

* Identify and tidy up “rough edges” that were acceptable in earlier phases but now feel unfinished:
  * placeholder copy that never got updated
  * TODO comments that can be addressed within this loop
  * awkward edge-case behaviors (e.g., flickers, double-loading, half-handled states)
  * missing confirmations where actions have meaningful impact
* Prefer small, contained fixes that **materially improve reliability and trust** in the scenario.

---

### **3. Reduce Noise & Distraction**

* Remove or tone down **unnecessary noise**:
  * overly verbose logs or debug output that no longer serve a purpose
  * redundant UI controls or options that add confusion without value
  * duplicate or confusing configuration paths
* Where appropriate, consolidate similar behaviors into a clearer, simpler pattern **without changing core workflows or business logic**.

---

### **4. Visual & Interaction Consistency**

* Ensure existing visual and interaction patterns feel **clean and deliberate**:
  * consistent spacing, alignment, and padding
  * consistent button hierarchy (primary vs secondary vs destructive)
  * consistent use of icons and text styles
* Improve small interaction details:
  * loading indicators where users might otherwise think the app is stuck
  * disabled vs enabled states that clearly communicate what’s possible
  * error states that guide the user toward recovery instead of dead-ends
* Do **not** introduce major layout changes or redesigns; stay within the scenario’s established design direction.

---

### **5. Copy & Communication Quality**

* Tighten microcopy so it is:
  * clear
  * concise
  * specific
  * consistent in tone
* Improve:
  * button text (“Do X” instead of vague labels)
  * error and warning messages (explain what went wrong and what to do next)
  * empty/loading states (briefly explain what the user should expect)
* Ensure first-time and returning users can quickly understand **what is happening and why** at each step.

---

### **6. Implementation-Level Polish (Without Major Refactors)**

* Within the constraints of the scenario’s architecture, you may:
  * remove obvious dead code, unused props, or unused imports
  * clarify confusing names when it does not cascade into a large refactor
  * standardize simple patterns (e.g., repeated inline logic replaced with a small helper) when low-risk
  * tidy tests and stories (names, descriptions) so they better describe behavior
* **Avoid large structural changes** (new abstractions, layered architectures, major file reorganizations). Those belong to a dedicated refactor phase.

---

### **7. Respect Scenario Constraints & Metrics**

* Do **not** change core workflows, APIs, or business logic.
* Do **not** introduce new features or flows that are unrelated to polishing.
* Respect the PRD, operational targets, and test-driven requirements.
* Use the existing completeness and quality metrics as **guardrails**, not as a game:
  * do not make superficial changes solely to move a metric
  * prioritize changes that genuinely improve perceived quality and robustness

---

### **8. Output Expectations**

You may update:

* labels, copy, and messaging
* minor layout details (spacing/alignment/hierarchy) within the existing structure
* error, loading, and empty states
* logs and debug output (to be more meaningful and less noisy)
* naming and small code-level details where safe
* tests’ descriptions and small behaviors that clarify intent

You **must**:

* keep the scenario fully functional
* avoid regressions in behavior and tests
* avoid large-scale refactors or new features
* leave the scenario feeling **more finished, coherent, and trustworthy** than before this loop

Focus this loop on **small, high-leverage improvements** that smooth out rough edges, clarify intent, and make the scenario feel ready for real users — without changing what it fundamentally is or how it fundamentally works.
