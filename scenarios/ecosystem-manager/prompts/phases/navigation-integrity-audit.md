## 🧭 **Phase: Navigation Integrity Audit**

Prioritize **navigation integrity**: every control that changes location, view, or mode should take the user where they reasonably expect to go, with honest labels and clear feedback.

Do **not** break functionality, regress tests, or alter core workflows. All changes must maintain or improve completeness and reliability.

Focus on **correctness, coherence, and honesty** in navigation and interaction contracts.

---

### 1. Establish the Navigation Mental Model

* Use the scenario’s **PRD, operational targets, and key user journeys** to infer:
  * the main “places” or modes in the scenario (pages, panels, major views)
  * how users are expected to move between them
  * what “back”, “home”, and “next” are supposed to mean in this context
* Treat this as the **intended navigation graph**: the conceptual map you are aligning the implementation to.

Keep this mental model in mind when evaluating every navigation control and shortcut.

---

### 2. Verify Label → Destination Truthfulness

* Identify all navigation affordances:
  * links, buttons, menus, list items
  * tabs, breadcrumbs, “view more” areas
  * calls-to-action that open specific states (e.g. “Open with template”, “Start from example”)
* For each:
  * Verify that the **actual destination and state** matches the label, icon, and context.
  * If something claims to open examples/templates/demo data, ensure it does so (and not a blank/default state).
  * If an action suggests a mode change (edit vs view vs create), ensure the resulting state is consistent with that promise.
* When mismatches exist, adjust:
  * the **behavior** to match the label, or
  * the **label** to match the behavior, choosing whichever best aligns with the PRD and user expectations.

Do not introduce misleading or vague labels to avoid fixing behavior; prioritize honest, predictable navigation.

---

### 3. Back, Forward & Return Path Coherence

* Examine how **back/close/cancel** controls behave across the scenario:
  * in-page back buttons
  * “X” close icons on dialogs/panels
  * “return” links or breadcrumbs
* Ensure that:
  * “Back” means “**return to where I just was**”, not “go to some fixed canonical page” the user may never have seen in this session.
  * Multi-entry pages (reachable from multiple places) either:
    * send the user back to their actual prior context, or
    * clearly communicate where they will go and why (e.g. breadcrumb to a canonical parent).
* Avoid surprising jumps that:
  * take the user to a page they never visited
  * lose important context or in-progress work without warning.

When multiple return paths are possible, prefer behavior that **preserves the user’s sense of continuity**.

---

### 4. Interaction Feedback for Navigation Actions

* Identify navigation actions that:
  * do nothing
  * sometimes fail silently
  * appear to work but leave the user in the same confusing state.
* For each, ensure:
  * the action **always does something observable** (route change, visible state change, or a clear message).
  * failures produce **timely, visible feedback** (e.g. non-intrusive error or toast) instead of silent no-ops.
  * long-running navigations (e.g. loading a heavy view) provide **loading indicators** or progress cues.
* Remove or fix “dead” controls that cannot work in the current context, or make them visibly disabled/inactive with clear affordances.

Avoid adding noisy alerts; feedback should be **proportionate and informative**, not disruptive.

---

### 5. Shortcut & Accelerator Consistency

* Identify all keyboard shortcuts and accelerators related to navigation:
  * global shortcuts (e.g. go to dashboard, open search, open command palette)
  * local shortcuts (e.g. next/previous, open details, switch panel)
* Ensure that:
  * shortcuts **work reliably** wherever they’re advertised.
  * they do not unexpectedly override essential browser shortcuts or conflict with each other in common contexts.
  * the same shortcut has the **same meaning** across similar views.
* If a shortcut cannot be supported in certain views, make sure:
  * it fails gracefully, or
  * it is scoped appropriately so it is not advertised or bound in those contexts.

Prefer small, targeted fixes over introducing a complex shortcut system in this phase.

---

### 6. Edge Cases: Deep Links, Refresh & Multi-Path Navigation

* Consider how navigation behaves in edge cases:
  * direct links into deeper views
  * browser refresh on non-root pages
  * using back/forward via browser controls
* Where possible, ensure:
  * the scenario can **recover a sensible state** when revisited or refreshed.
  * critical navigation state is not lost in trivial, avoidable ways.
* For multi-path flows (e.g. reaching the same view from several funnels), verify that:
  * follow-up navigation (back/next/close) still respects **the user’s actual path**, not just an assumed canonical funnel.

Avoid major routing overhauls in this phase; focus on making current flows **predictable and non-deceptive**.

---

### 7. Maintain Scenario Constraints

* Do **not** introduce new features unrelated to navigation integrity.
* Do **not** re-architect the entire routing system in this phase.
* Keep all changes aligned with the scenario’s:
  * PRD
  * operational targets
  * test-driven requirements
* When faced with a choice between a risky global change and a smaller, safer improvement, choose the **safer incremental fix** and surface larger redesign ideas as recommendations.

---

### 8. Output Expectations

You may update:

* navigation handlers and routing targets
* button/link/menu behavior and labels
* back/close/cancel and breadcrumb logic
* shortcut bindings and their scopes
* microcopy related to navigation and destinations
* tests that exercise navigation flows and edge cases

You **must**:

* keep the scenario fully functional and non-regressed
* avoid introducing misleading or ambiguous navigation behaviors
* ensure navigation controls do what they promise, consistently
* improve the user’s ability to **predict where actions will take them**

Focus this loop on **practical, targeted navigation fixes** that make movement through the scenario honest, stable, and easy to understand.

Avoid superficial changes (e.g. renaming buttons without fixing their behavior, or shuffling links without clarifying flows) that do not materially improve navigation integrity.
