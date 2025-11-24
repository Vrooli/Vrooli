## ⚡ **Phase: Performance & Responsiveness**

During this loop, prioritize **runtime performance and perceived responsiveness** across this entire scenario.
Do **not** break functionality or regress tests; all changes must maintain or improve completeness.

Focus on delivering a **faster, smoother experience**, guided by the following principles:

### **1. Perceived Responsiveness**

* Optimize for how fast the interface **feels**, not just raw benchmarks.
* Ensure primary user actions receive **immediate visual feedback**:
  * loading indicators
  * disabled states while processing
  * optimistic or progressive updates where safe
* Avoid blocking the UI with long-running work wherever possible.

### **2. Reduce Unnecessary Work**

* Identify and remove **redundant computations**, repeated formatting, or unnecessary state transformations.
* Avoid doing the same work in multiple places if it can be **shared, cached, or memoized** in a safe, framework-agnostic way.
* Simplify data flows to avoid unnecessary reprocessing or duplication.
* Only recompute or refresh what actually needs to change.

### **3. Trim Expensive Interactions**

* Focus first on **common user journeys** and interaction hotspots:
  * initial load and first-use flows
  * main dashboard or primary views
  * frequently used forms or editors
* Reduce:
  * excessive re-rendering or DOM updates
  * heavy computations triggered on every keystroke, scroll, or mouse move
  * unnecessary re-fetching of the same data

Where appropriate, introduce **debouncing**, **throttling**, or **batching** of operations to keep interactions smooth.

### **4. Data Loading & Network Efficiency**

* Avoid fetching more data than the user reasonably needs at once.
* Prefer:
  * incremental or paginated loading for large lists
  * reusing existing data in memory when it remains valid
* Eliminate obviously duplicated calls, or consolidate them into a single, well-structured request where safe.
* Do not introduce caching that risks serving stale or incorrect data without clear invalidation rules.

### **5. Rendering & Structure Efficiency**

* Simplify complex UI structures that do not contribute to clarity or functionality.
* Avoid deeply nested or overly complex hierarchies when a simpler structure offers the same behavior.
* For large collections or heavy views, consider patterns that **render less at once** (conditional rendering, chunked views, virtualization) when they fit naturally into the existing design.
* Only introduce performance-oriented abstractions if they **reduce** complexity or clearly improve maintainability.

### **6. Maintain Scenario Constraints**

* Do **not** change the scenario’s core workflows, APIs, or business logic.
* Do **not** remove safeguards, validation, or tests purely to make things “faster.”
* Ensure all performance improvements remain consistent with:
  * the PRD and operational targets
  * existing tests and requirements
  * the scenario’s architectural direction

### **7. Safe Optimizations & Non-Gaming**

* Do **not** “optimize” by:
  * hiding slow operations without actually improving them
  * weakening error handling or observability
  * reducing data quality, accuracy, or reliability
* Prefer improvements that would be **meaningfully noticeable to end users**:
  * faster initial load
  * snappier navigation
  * smoother typing and interaction
  * less jank during complex operations

### **8. Output Expectations**

You may update:

* data loading strategies
* state and data flow organization
* component decomposition and reuse
* background/foreground work separation
* interaction patterns for frequent flows
* performance-related configuration and guards

You **must**:

* keep the scenario fully functional
* avoid regressions and preserve correctness
* maintain or improve the completeness score
* make the experience **measurably or obviously faster and smoother** for real users

Focus this loop on **practical, targeted performance improvements** that make the scenario feel lighter, more responsive, and more efficient to use.

**Avoid superficial “optimizations” that add complexity or risk without clearly improving real-world performance. Only make changes that genuinely reduce latency, unnecessary work, or interaction jank.**
