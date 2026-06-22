## Steer focus: Performance & Responsiveness

> **Ladder position:** R3 (features hardened — performance/responsiveness). See `prompt-manager skill read scenario-maturity-ladder` for rung context and `prompt-manager skill read improvement-do-and-dont` for what counts as a real improvement.

Prioritize **runtime performance and perceived responsiveness** across this entire scenario.
Do **not** break functionality or regress tests; all changes must maintain or improve completeness.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

### Measurement counterpart: the `performance-health` scenario

This skill says *what* to optimize and what counts as a real improvement. **`performance-health` is the *how* — the engine that measures whether a change actually produced one.** What used to be the hand-rolled `scenario-performance-audit` CDP/Playwright methodology is now a real, productized capability; drive it instead of hand-rolling a capture script:

- `performance-health audit {{TARGET}}` — capture → analyse → per-component commit table (count / avg / max) with `file:line`-anchored hotspots.
- `performance-health analysis ...` / `performance-health benchmark {{TARGET}}` — re-analyse a captured trace; time the Go + UI build.
- `performance-health budget ...` — declare/check a tighten-only budget (build time, bundle size, LCP, startup, component-commit avg/max) so a regression FAILS `git-control-tower baseline diff` like any other health regression.
- `vrooli scenario test {{TARGET}}` **Performance phase** — the same engine, run in execution mode: it benchmarks the build, runs Lighthouse-if-UI, persists a sample, and gates on budgets + native build-time thresholds. This is the regression gate your changes must keep green.

> This skill's detection has **graduated into a programmatic engine** (`programmaticHome: performance-health:performance`). Treat performance-health as the source of truth for the numbers; this steer is the judgment layer over them.

**Measurement discipline (don't ship on vibes):**
- Capture a baseline → make the change → re-measure → compare. Beyond a trivial fix — anything where you'd otherwise be guessing whether it helped — get before/after numbers from `performance-health`.
- **Watch avg-per-commit, not just total.** When a fix changes commit cardinality (e.g. virtualization mounts many cheap rows instead of one giant render), the *total* can rise while each commit got far cheaper. The per-commit average is the metric that tracks "did rendering this actually get cheaper"; confirm it drops. The long-task delta corroborates from a different angle and is the cleanest correlate of *felt* performance.
- **Measure deployed behaviour, not a dev build** (dev mode disables batching, double-renders under StrictMode, and skips tree-shaking — the numbers mislead). performance-health's benchmark/audit run the production build for you.
- Captures vary 5–15% run-to-run from background noise; if before/after are within that band, the change isn't distinguishable — say so rather than claiming a win.
- Findings are `file:line`-anchored and quantified. Persist a durable record under `docs/perf/<date>-<slug>.md` when an audit produces something worth keeping.

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

### **6. Safe Optimizations & Non-Gaming**

* Do **not** “optimize” by:
  * hiding slow operations without actually improving them
  * weakening error handling or observability
  * reducing data quality, accuracy, or reliability
* Prefer improvements that would be **meaningfully noticeable to end users**:
  * faster initial load
  * snappier navigation
  * smoother typing and interaction
  * less jank during complex operations

### **7. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `performance`.

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

**Avoid superficial "optimizations" that add complexity or risk without clearly improving real-world performance. Only make changes that genuinely reduce latency, unnecessary work, or interaction jank.**
