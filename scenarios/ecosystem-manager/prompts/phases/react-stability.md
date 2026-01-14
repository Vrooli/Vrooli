## Steer focus: React Stability

Prioritize **hardening React UI components against runtime crashes** across this scenario.

Your goal is to ensure the UI **degrades gracefully** under unexpected data, user actions, and edge cases, rather than crashing with a white screen or cryptic error.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve overall completeness and reliability.

---

### **1. Error Boundaries: Strategic Placement**

* Verify that **Error Boundaries** wrap major UI sections so that failures are isolated rather than cascading:
  * route-level views or pages
  * complex feature panels (modals, sidebars, dashboards)
  * components that render dynamic or external data
  * areas with heavy computation or transformation logic
* Ensure fallback UIs are **meaningful and actionable**:
  * clearly describe what failed without exposing stack traces to end users
  * offer retry, refresh, or navigation options where appropriate
  * avoid dead-ends where the user cannot recover
* Log errors to the console (or a structured logger if available) with enough context for debugging, but **never expose raw stack traces in production UI**.

**Anti-patterns to fix:**
* Single top-level boundary that catches everything (failures nuke the whole app)
* Missing boundaries around data-driven components
* Fallback UIs that just say "Error" with no recovery path

---

### **2. Defensive Data Access**

Audit components for **unsafe assumptions about data shape**:

* Use **optional chaining** (`?.`) for all nested property access where values may be null/undefined
* Use **nullish coalescing** (`??`) to provide sensible defaults
* Never assume API responses have complete shapes; treat all external data as potentially incomplete
* For array iteration, prefer the pattern:
  ```tsx
  {(data?.items ?? []).map((item) => (
    // render item
  ))}
  ```
* Guard against common crash sources:
  * `.length` on potentially undefined arrays
  * `.map()`, `.filter()`, `.find()` on non-arrays
  * accessing properties on objects that may be null

**Priority areas:**
* Components that receive data from hooks (useQuery, useContext)
* Components that transform or filter data before rendering
* Props passed through multiple component layers

---

### **3. Hook Discipline (Critical)**

React hooks have strict rules that, when violated, cause runtime crashes or subtle bugs. Audit and fix:

**State updates during render (causes React Error #310):**
* Never call `setState` directly in the render body
* Conditionally calling hooks based on render logic is forbidden
* Side effects belong in `useEffect`, not in render

**useMemo / useCallback hygiene:**
* `useMemo` must be pure; no side effects, no state updates inside
* Return stable references; avoid returning new objects/arrays if inputs haven't changed
* Avoid complex logic that could throw; wrap in try/catch if necessary

**useEffect correctness:**
* Always provide proper cleanup functions for subscriptions, timers, event listeners
* Avoid stale closure bugs by ensuring dependency arrays are complete
* Never omit dependencies to "make it work"; fix the underlying issue instead

**Dependency stability:**
* Avoid creating objects, arrays, or functions inline in render if they're used as hook dependencies
* Extract stable references using `useMemo`, `useCallback`, or move definitions outside the component
* Ensure `eslint-plugin-react-hooks` exhaustive-deps rule is enabled and respected

**Common violations to hunt:**
* `if (condition) { useState(...) }` - conditional hook calls
* `useEffect(() => { setState(compute()) }, [])` - missing dependencies
* `useMemo(() => { setSomething(); return value; }, [deps])` - side effects in memo

---

### **4. TypeScript Strictness**

Verify TypeScript configuration provides adequate safety:

* Confirm `strict: true` is enabled in tsconfig
* Consider enabling `noUncheckedIndexedAccess` for array access safety
* Ensure `useState` and `useReducer` have explicit type parameters where inference is ambiguous
* Audit for `as any` casts that bypass type safety in data handling code

Do not force `noUncheckedIndexedAccess` if it causes excessive churn; document it as a future recommendation instead.

---

### **5. Component State Management**

Ensure components explicitly handle all states of async data:

* Every data-fetching component should handle:
  * **Loading**: skeleton, spinner, or placeholder UI
  * **Error**: meaningful error message with recovery options
  * **Empty**: clear empty state (not just blank space)
  * **Success**: the actual data rendering

* Use discriminated unions for complex state machines
* Never render UI assuming data exists
* When using React Query or similar, handle `isLoading`, `isError`, and `data` states explicitly

---

### **6. Optional: Zod Validation at High-Risk Boundaries**

Where runtime crashes are frequent due to unexpected data shapes, consider adding **Zod schemas** at critical boundaries:

* **When to use Zod:**
  * API responses that have caused crashes in the past
  * Data from untrusted sources (user input, external integrations)
  * Complex transformation pipelines where shape assumptions are fragile

* **When NOT to use:**
  * Every API call (overhead without benefit)
  * Internal data flows with strong TypeScript coverage
  * Simple, well-typed data structures

This is **optional and targeted**; do not introduce Zod everywhere.

---

### **7. Memory Management with Visited Tracker**

To ensure **systematic coverage without repetition**, use `visited-tracker`:

**At the start of each iteration:**
```bash
visited-tracker least-visited \
  --location scenarios/{{TARGET}}/ui \
  --pattern "**/*.{ts,tsx}" \
  --tag react-stability \
  --name "{{TARGET}} - React Stability" \
  --limit 5
```

**After analyzing each file:**
```bash
visited-tracker visit <file-path> \
  --location scenarios/{{TARGET}}/ui \
  --tag react-stability \
  --note "<summary of stability improvements made and what remains>"
```

**When a file is irrelevant (config, build scripts, etc.):**
```bash
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}}/ui \
  --tag react-stability \
  --reason "Not a React component - build config/server/tooling/types/etc."
```

**When a file is fully hardened:**
```bash
visited-tracker exclude <file-path> \
  --location scenarios/{{TARGET}}/ui \
  --tag react-stability \
  --reason "All stability improvements complete"
```

**Before ending your session:**
```bash
visited-tracker campaigns note \
  --location scenarios/{{TARGET}}/ui \
  --tag react-stability \
  --name "{{TARGET}} - React Stability" \
  --note "<overall progress summary, crash-prone areas identified>"
```

---

### **8. Maintain Scenario Constraints**

* Do **not** change the scenario's core workflows, APIs, or business logic
* Do **not** introduce new features unrelated to React stability
* Do **not** replace the existing Protobuf/API setup; work within it
* Prefer **incremental, localized improvements** over ambitious rewrites

---

### **9. Output Expectations**

You may update:
* Error boundary placement and fallback UIs
* Optional chaining and nullish coalescing for data access
* Hook implementations (useEffect cleanup, dependency arrays, stable references)
* Loading, error, and empty state handling in components
* TypeScript types where they improve safety
* Targeted Zod validation at high-risk boundaries (sparingly)

You **must**:
* Keep the scenario fully functional and non-regressed
* Reduce or eliminate React runtime crashes
* Improve the resilience of components to unexpected data
* Avoid breaking existing functionality in pursuit of "safety"

Focus this loop on **practical, targeted stability improvements** that prevent white-screen crashes and make the UI resilient to real-world data variability.

**Avoid superficial changes that rename variables or restructure code without materially improving crash resistance.**
