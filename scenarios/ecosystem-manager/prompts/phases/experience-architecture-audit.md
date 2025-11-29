## Steer focus: Experience Architecture Audit

Prioritize understanding and improving the **experience architecture** of this scenario:
how different types of users move through the product to achieve their goals.

Your goal is to:
1. Derive a clear mental model of **who** is using this scenario and **what they’re trying to do**.
2. Map **current flows vs. ideal flows** for key jobs.
3. Identify where users encounter **unnecessary friction**.
4. **Implement safe, targeted UX flow changes** that better align the experience with user intent.

Do **not** break functionality, regress tests, or introduce new product directions. All changes must maintain or improve completeness and reliability.

---

### **1. Clarify Scenario Purpose & User Intent**

* Review the scenario’s **PRD, operational targets, and UI** to answer:
  * What problem does this scenario solve?
  * What are the primary “jobs to be done” for users of this scenario?
* Derive a concise **purpose statement** in your own words, such as:
  * “This scenario helps users ___ so that they can ___.”

Keep this purpose statement as your north star for the rest of the audit.

---

### **2. Identify Core Personas & Key Jobs**

* Infer a small set of **core user personas** from the PRD and current UX (e.g. “first-time user”, “returning builder”, “monitoring/ops user”, “viewer-only stakeholder”).
* For each persona, identify **1–3 primary jobs** they come to this scenario to accomplish.
  * Example pattern (do not hard-code this; infer from the scenario):  
    * First-time user → “understand what this does”, “create first X”
    * Returning user → “continue where I left off”
    * Ops user → “check health/state”, “inspect failures”
* Keep the number of personas and jobs small and focused. This is about clarity, not exhaustiveness.

---

### **3. Map Current Flows for Key Jobs**

For each persona + job:

* Trace the **current flow** through the UI as it exists today:
  * Entry point(s)
  * Screens/pages/views visited
  * Major actions (clicks, navigations, form submissions, mode switches)
* Note:
  * Navigation depth (how many steps/clicks)
  * Places where the user must **remember** something (which item, which project, which filter, etc.)
  * Places where the user must **hunt** for the next action or is forced to “drill down” repeatedly

Capture these current flows in a succinct, structured way (e.g. “Current: Dashboard → … → …”) in your internal reasoning and summary note.

---

### **4. Sketch Ideal Flows from User Intent**

For the same persona + job pairs:

* Propose an **ideal flow** that:
  * Minimizes unnecessary navigation and mode switches
  * Surfaces the most relevant action as directly as possible
  * Respects the scenario’s existing constraints and architecture
* Think in terms of:
  * “From the user’s point of view, what is the **shortest, clearest path** from intent → success?”
  * “What would they expect to see or do first if the product screamed its purpose?”

Do **not** design a completely different product. Focus on reshaping how users reach existing value.

---

### **5. Identify Friction & Misalignments**

For each persona + job, compare **Current vs. Ideal**:

* Highlight where friction accumulates:
  * Excessive steps or deep hierarchies
  * Hidden or non-obvious entry points
  * Forced detours through organizational layers that don’t match user mental models
  * Missing “resume” or “shortcut” affordances
* Categorize friction where useful:
  * **Mechanical** (too many clicks/scrolls/inputs)
  * **Cognitive** (user must remember too much or guess)
  * **Discoverability** (important capabilities are buried or invisible)

Focus on **concrete, observable gaps** rather than vague “this feels clunky” opinions.

---

### **6. Prioritize & Implement Experience Improvements**

* Aggregate your findings across personas and flows, then select a **small set of high-impact, low-risk improvements** to implement in this loop.
* Prefer changes that:
  * Significantly reduce friction in **common** jobs
  * Surface important existing capabilities more directly
  * Require **localized** code changes (not a full redesign)
* Think in terms of:
  * “What 1–3 changes can I safely ship in this loop that make the biggest difference to real user flows?”

You should **both**:
1. Implement these improvements in code, and  
2. Document additional, larger opportunities that are too big or risky for this loop.

---

### **7. Safe Experience-Level Implementation Guidelines**

Where it is clearly safe and consistent with tests and PRD, you may:

* Add or adjust **entry points** to common jobs, such as:
  * “Continue where you left off”
  * “Recent items” or “Recent activity”
  * Quick access to frequently-used objects or actions
* Improve **navigation affordances**:
  * More direct links or buttons to common destinations
  * Clearer call-to-action for the most important next step
* Surface existing capabilities in more intuitive locations:
  * Expose important functionality that is currently buried, without changing what it does.
* Improve contextual guidance:
  * Labels, hint text, short explanatory copy that helps users understand what to do next.

You must:

* Avoid large structural rewrites in this phase.
* Avoid introducing entirely new product features; focus on **how** users reach existing value.
* Keep all existing flows working; do not remove an entry path without replacing it with something strictly better.
* Maintain or improve test coverage and completeness metrics; do not weaken tests to fit changes.

If an improvement would require a broad redesign or major backend changes, **do not partially implement it**. Instead, clearly describe it in your summary note for future loops.

---

### **8. Maintain Scenario Constraints**

* Do **not** change core business rules, pricing models, or system-of-record logic.
* Do **not** repurpose the scenario into a different product.
* Ensure all changes remain aligned with the PRD, operational targets, and test-driven requirements.
* Respect any existing design system or component library; work with it rather than fighting it.

---

### **9. Output Expectations**

By the end of this loop, you should:

* Have **implemented** a focused set of UX flow improvements that:
  * reduce friction for one or more common persona jobs
  * make key entry points more direct and discoverable
  * preserve or improve tests and completeness scores
* Produce a structured note that includes:
  * The scenario’s purpose (in your words)
  * The core personas and their primary jobs
  * A few key **Current vs. Ideal** flow comparisons
  * A concise list of:
    * changes you actually implemented in this loop
    * additional higher-effort improvements recommended for future loops

Focus this loop on **experience-level clarity**: making it easier for real users to accomplish their goals by aligning navigation and entry points with intent.

Avoid superficial UI tweaks or purely aesthetic changes. Prioritize improvements that make the scenario’s experience **scream its purpose** as clearly as its internal architecture does.
