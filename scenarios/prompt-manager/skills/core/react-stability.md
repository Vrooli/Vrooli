## Steer focus: React Stability

Prioritize **hardening React UI components against runtime crashes** across this scenario.

Your goal is to ensure the UI **degrades gracefully** under unexpected data, user actions, and edge cases, rather than crashing with a white screen or cryptic error.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve overall completeness and reliability.

---

### **0. Tooling Prerequisites (Run First)**

Before manual code review, verify that automated tooling is configured to catch React-specific bugs. **Proper TypeScript and ESLint configuration catches 80%+ of runtime crashes automatically.**

#### **Step 1: Verify TypeScript Safety Rules**

Check that `tsconfig.json` (or `tsconfig.node.json` if it extends) has these critical settings with protective comments:

```jsonc
{
  "compilerOptions": {
    // ╔══════════════════════════════════════════════════════════════════════════╗
    // ║  SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN                         ║
    // ║                                                                          ║
    // ║  These rules prevent runtime crashes like:                               ║
    // ║  - "X is not a function"                                                 ║
    // ║  - "Cannot read property Y of undefined"                                 ║
    // ║  - "undefined is not iterable"                                           ║
    // ║                                                                          ║
    // ║  If you encounter type errors from these rules:                          ║
    // ║  ✅ DO: Use optional chaining (?.) or null checks (if (x) { ... })       ║
    // ║  ✅ DO: Use nullish coalescing (??) for defaults                         ║
    // ║  ✅ DO: Add proper type guards before accessing properties               ║
    // ║  ❌ DON'T: Use non-null assertion (!) - it hides bugs, use ?? instead    ║
    // ║  ❌ DON'T: Use type assertions (as X) to silence errors                  ║
    // ║  ❌ DON'T: Add @ts-ignore or @ts-expect-error comments                   ║
    // ║  ❌ DON'T: Remove or weaken these rules                                  ║
    // ║                                                                          ║
    // ║  These rules exist because UI crashes are the #1 production issue.       ║
    // ║  Removing them WILL cause crashes that are much harder to debug than     ║
    // ║  the type errors they produce at compile time.                           ║
    // ╚══════════════════════════════════════════════════════════════════════════╝
    "strict": true,
    "noUncheckedIndexedAccess": true
  }
}
```

**Why `noUncheckedIndexedAccess`?** Without it, `arr[0].trim()` compiles fine but crashes at runtime if the array is empty. With it, TypeScript forces you to handle the undefined case: `arr[0]?.trim()`.

**IMPORTANT:** Always include the protective comment block above when adding these rules. The comments prevent future agents from removing the rules without understanding the consequences.

#### **Step 2: Check if ESLint is configured for React**

```bash
# Look for ESLint config in the UI directory
ls scenarios/{{TARGET}}/ui/eslint.config.* 2>/dev/null || \
ls scenarios/{{TARGET}}/ui/.eslintrc* 2>/dev/null || \
echo "NO ESLINT CONFIG FOUND"
```

#### **Step 3: If no config exists, create one with safety rules**

Create `scenarios/{{TARGET}}/ui/eslint.config.js` with protective comments:

```js
import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parserOptions: {
        project: "./tsconfig.json",  // Enable type-aware linting
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules prevent runtime crashes. If you encounter errors:
      // ✅ DO: Fix the code with optional chaining (?.), null checks, or proper types
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      // Using ! hides bugs that will crash at runtime with "X is not a function"
      // Instead of arr[0]!, use: arr[0] ?? defaultValue or if (arr[0]) { ... }
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // ════════════════════════════════════════════════════════════════════════
      // STANDARD RULES (can be adjusted if needed)
      // ════════════════════════════════════════════════════════════════════════

      // Catches stale closure bugs from missing/incorrect dependencies
      "react-hooks/exhaustive-deps": "warn",

      // Ensures only components are exported for proper HMR
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      // Allow unused vars prefixed with underscore (common pattern for ignored params)
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
    },
  }
);
```

**IMPORTANT:** Always include the protective comment blocks above when creating ESLint configs. The comments explain why each rule exists and prevent future agents from removing them without understanding the consequences.

Ensure required dev dependencies are installed:
```bash
cd scenarios/{{TARGET}}/ui
pnpm add -D eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh
```

#### **Step 4: Run linting and fix errors**

```bash
cd scenarios/{{TARGET}}/ui && pnpm lint
```

**Priority order for fixes:**
1. `rules-of-hooks` errors - **guaranteed crash bugs**, fix ALL before proceeding
2. `no-non-null-assertion` errors - using `!` hides null bugs that crash at runtime
3. `no-unsafe-*` warnings - operations on `any` types that may crash
4. `exhaustive-deps` warnings - potential stale closure bugs

---

### **0.5 Protected Rules - NEVER REMOVE**

The following rules exist because **UI crashes are the #1 production issue**. They catch bugs at compile/lint time that would otherwise crash at runtime.

**⚠️ THESE RULES MUST NEVER BE REMOVED, DISABLED, OR WEAKENED:**

| Rule | Location | What it prevents |
|------|----------|------------------|
| `strict: true` | tsconfig.json | Entire class of null/undefined bugs |
| `noUncheckedIndexedAccess: true` | tsconfig.json | `arr[0].method()` crashes when array is empty |
| `react-hooks/rules-of-hooks` | eslint.config.js | React Error #310 (early returns before hooks) |
| `@typescript-eslint/no-non-null-assertion` | eslint.config.js | `!` operator that hides null bugs |
| `@typescript-eslint/no-explicit-any` | eslint.config.js | `any` type that disables all checking |

**When you encounter errors from these rules:**

✅ **DO:**
- Use optional chaining: `obj?.prop?.method()`
- Use nullish coalescing: `value ?? defaultValue`
- Add proper null checks: `if (arr.length > 0) { arr[0].method() }`
- Use type guards: `if (typeof x === 'string') { x.trim() }`

❌ **DON'T:**
- Use non-null assertion: `arr[0]!.method()` - this HIDES the bug
- Use type assertion: `(value as string).trim()` - this LIES to TypeScript
- Add `@ts-ignore` or `@ts-expect-error` - this SILENCES the warning
- Disable ESLint rules inline or globally - this REMOVES the protection
- Remove or weaken the rules in config - this CAUSES PRODUCTION CRASHES

**If fixing errors causes significant churn:** Fix them incrementally. A scenario with these rules and some unfixed errors is safer than one without the rules at all - the errors serve as documentation of known risks

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

React hooks have strict rules that, when violated, cause runtime crashes. **This section covers bugs that ESLint catches automatically (Section 0) plus manual detection for edge cases.**

#### **3.1 Rules of Hooks Violations (React Error #310)**

React Error #310 ("Rendered fewer hooks than expected") occurs when the **number of hooks changes between renders**. This is a **guaranteed crash** - no error boundary can catch it.

**Root cause:** React tracks hooks by call order. If render N calls 8 hooks but render N+1 calls 9 hooks, React crashes.

**The #1 cause: Early returns placed BEFORE hooks**

This pattern looks innocent but is a crash waiting to happen:
```tsx
// ❌ WRONG - hooks after early return won't always execute
function MessageList({ messages }) {
  const [filter, setFilter] = useState("");           // Hook 1
  const theme = useContext(ThemeContext);             // Hook 2

  // Early return BEFORE all hooks are called!
  if (messages.length === 0) {
    return <EmptyState />;  // Returns here when empty
  }

  // Hook 3 - ONLY executes when messages.length > 0
  const filtered = useMemo(() =>
    messages.filter(m => m.includes(filter)),
    [messages, filter]
  );

  return <List items={filtered} />;
}
// When messages changes from [] to [msg], hook count changes 2→3 = CRASH
```

```tsx
// ✅ CORRECT - all hooks execute every render, early return AFTER
function MessageList({ messages }) {
  const [filter, setFilter] = useState("");           // Hook 1
  const theme = useContext(ThemeContext);             // Hook 2

  // Hook 3 - ALWAYS executes, handles empty case internally
  const filtered = useMemo(() => {
    if (messages.length === 0) return [];
    return messages.filter(m => m.includes(filter));
  }, [messages, filter]);

  // Early return AFTER all hooks
  if (messages.length === 0) {
    return <EmptyState />;
  }

  return <List items={filtered} />;
}
```

**Other causes:**
- `if (condition) { useState(...) }` - conditional hook calls
- Hooks inside loops with varying iteration counts
- Hooks inside try/catch where catch returns early

**Manual detection methodology (when ESLint isn't catching it):**
1. **Count all hooks** in each component (useState, useEffect, useMemo, useCallback, useContext, useRef, useReducer, custom `use*` hooks)
2. **Identify ALL early returns** - any `return` statement before the component's final return
3. **Verify NO hooks appear after ANY early return**
4. If hooks appear after early returns → MOVE the early return after all hooks

#### **3.2 useMemo / useCallback Hygiene**

* `useMemo` must be pure; no side effects, no state updates inside
* Return stable references; avoid returning new objects/arrays if inputs haven't changed
* For empty arrays/objects, use module-level constants to prevent new references:
  ```tsx
  // ❌ WRONG - new array on every render when items is empty
  const filtered = useMemo(() => items?.filter(x => x.active) ?? [], [items]);

  // ✅ CORRECT - stable reference for empty case
  const EMPTY_ITEMS: Item[] = [];  // Outside component
  const filtered = useMemo(() => {
    if (!items || items.length === 0) return EMPTY_ITEMS;
    return items.filter(x => x.active);
  }, [items]);
  ```
* Avoid complex logic that could throw; wrap in try/catch if necessary

#### **3.3 useEffect Correctness**

* Always provide proper cleanup functions for subscriptions, timers, event listeners
* Avoid stale closure bugs by ensuring dependency arrays are complete
* Never omit dependencies to "make it work"; fix the underlying issue instead

#### **3.4 Dependency Stability**

* Avoid creating objects, arrays, or functions inline in render if they're used as hook dependencies
* Extract stable references using `useMemo`, `useCallback`, or move definitions outside the component
* The `exhaustive-deps` ESLint rule catches most of these issues automatically

#### **3.5 State Updates During Render**

* Never call `setState` directly in the render body (causes infinite loops)
* Side effects belong in `useEffect`, not in render
* This is a different error from #310 - causes "Too many re-renders"

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
