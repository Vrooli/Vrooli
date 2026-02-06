## Steer focus: React Stability

Prioritize **hardening React UI components against runtime crashes** across this scenario.

Your goal is to ensure the UI **degrades gracefully** under unexpected data, user actions, and edge cases, rather than crashing with a white screen or cryptic error.

Do **not** break functionality, regress tests, or introduce new features. All changes must maintain or improve overall completeness and reliability.

Required reading:
- `prompt-manager skills read visited-tracker-tools knowledge-observatory-tools`

---

### **0. Tooling Prerequisites (Run First)**

* Read the `problems` doc for `{{TARGET}}` using `knowledge-observatory-tools` to understand existing stability issues.

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
import importPlugin from "eslint-plugin-import";
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
      "import": importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: "./tsconfig.json",
        },
      },
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

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      // These runtime errors are extremely hard to debug in production (minified variable names).
      // Requires eslint-plugin-import and eslint-import-resolver-typescript
      "import/no-cycle": "error",

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
pnpm add -D eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh eslint-plugin-import eslint-import-resolver-typescript
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
| `import/no-cycle` | eslint.config.js | Circular dependencies causing "Cannot access X before initialization" |

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

### **1. Layered Architecture (Strongly Recommended)**

Structure React applications with clear layers to create **natural testing seams** and ensure **validation happens at the right boundary**:

```
Components (Pure rendering - receives validated data, no logic)
    ↓
Hooks (React state management - calls controllers, handles UI state)
    ↓
Controllers (Page-specific orchestration - coordinates services)
    ↓
Services (Pure functions & API calls - handles validation, returns typed data)
```

**Why this pattern:**
- **Testing seams**: Each layer can be tested in isolation. Components test rendering, hooks test state transitions, controllers test orchestration, services test data transformation.
- **Validation boundary**: Runtime validation (Zod) happens in the Service layer at the API boundary - once validated, data flows up as trusted types.
- **Separation of concerns**: UI rendering is decoupled from data fetching and business logic.
- **Customization without degradation**: You can swap out or customize any layer without breaking others.

**Layer responsibilities:**

| Layer | Responsibility | Tests |
|-------|---------------|-------|
| **Components** | Pure rendering, props → JSX | Snapshot/visual tests |
| **Hooks** | UI state, context, call controllers | Hook unit tests |
| **Controllers** | Orchestrate services, page-level logic | Integration tests |
| **Services** | API calls, data transform, **runtime validation** | Unit tests, contract tests |

**Key principle:** Components should never see invalid data. Validation failures are caught in the Service layer and surfaced as typed error states that the UI can handle gracefully.

This pattern is **strongly recommended** for all React scenarios in Vrooli. The exact directory structure may vary (e.g., `src/services/`, `src/hooks/`, etc.), but the layer separation should be maintained.

---

### **2. Error Boundaries: Strategic Placement**

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

### **3. Defensive Data Access**

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

### **4. Hook Discipline (Critical)**

React hooks have strict rules that, when violated, cause runtime crashes. **This section covers bugs that ESLint catches automatically (Section 0) plus manual detection for edge cases.**

#### **4.1 Rules of Hooks Violations (React Error #310)**

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

#### **4.2 useMemo / useCallback Hygiene**

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

#### **4.3 useEffect Correctness**

* Always provide proper cleanup functions for subscriptions, timers, event listeners
* Avoid stale closure bugs by ensuring dependency arrays are complete
* Never omit dependencies to "make it work"; fix the underlying issue instead

#### **4.4 Dependency Stability**

* Avoid creating objects, arrays, or functions inline in render if they're used as hook dependencies
* Extract stable references using `useMemo`, `useCallback`, or move definitions outside the component
* The `exhaustive-deps` ESLint rule catches most of these issues automatically

#### **4.5 State Updates During Render**

* Never call `setState` directly in the render body (causes infinite loops)
* Side effects belong in `useEffect`, not in render
* This is a different error from #310 - causes "Too many re-renders"

---

### **5. TypeScript Strictness**

Verify TypeScript configuration provides adequate safety:

* Confirm `strict: true` is enabled in tsconfig
* Consider enabling `noUncheckedIndexedAccess` for array access safety
* Ensure `useState` and `useReducer` have explicit type parameters where inference is ambiguous
* Audit for `as any` casts that bypass type safety in data handling code

Do not force `noUncheckedIndexedAccess` if it causes excessive churn; document it as a future recommendation instead.

---

### **6. Component State Management**

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

### **7. Runtime Validation with Protobuf + Zod (Recommended)**

TypeScript types are erased at runtime - they cannot catch API shape mismatches, null values in unexpected places, or data corruption. For production reliability, use **runtime validation at system boundaries**.

#### **7.1 The Type Safety Chain**

```
Proto Definitions (packages/proto/schemas/)
    ↓ code generation
Generated TS Types (@vrooli/proto-types)
    ↓ derive validation schemas
Zod Schemas (ui/src/shared/api/schemas/)
    ↓ runtime validation in services
Validated Data → Components
```

**Why this chain:**
- **Proto definitions** are the source of truth for API contracts (see `packages/proto/README.md`)
- **Generated types** provide compile-time safety
- **Zod schemas** provide runtime validation at the service layer
- Once data passes Zod validation, it flows up as trusted typed data

#### **7.2 When to Validate**

| Boundary | Validate? | Why |
|----------|-----------|-----|
| API responses | **YES** | External data, contract may drift |
| WebSocket messages | **YES** | External data, parsing may fail |
| User input | **YES** | Never trust user input |
| Internal service calls | No | Already validated at entry |
| Component props | No | Data validated before reaching components |

**Key principle:** Validate once at the system boundary (in Services), then trust the data as it flows through the application.

#### **7.3 Validation Pattern**

Services should return a discriminated union that makes validation failures explicit:

```typescript
// In your service layer
import { z } from 'zod';

const PlanSchema = z.object({
  plan_name: z.string(),
  plan_tier: z.string(),
  billing_interval: z.enum(['month', 'year', 'one_time']),
  // ... derived from proto types
});

type ParseResult<T> =
  | { success: true; data: T }
  | { success: false; error: string };

export function fetchPlan(id: string): Promise<ParseResult<Plan>> {
  // Fetch, validate, return typed result or error
}
```

This pattern:
- Makes validation failures **explicit** rather than crashing
- Lets components handle errors gracefully with error UI
- Keeps validation logic **centralized** in services

#### **7.4 Proto Integration**

When creating or modifying API contracts:
1. Define the schema in `packages/proto/schemas/` (see `packages/proto/README.md` for guidance)
2. Run `cd packages/proto && make generate` to regenerate types
3. Create corresponding Zod schemas that mirror the proto structure
4. Use the generated TS types for compile-time safety, Zod for runtime validation

**Note:** Zod schemas should mirror proto types closely. If the proto has `optional string field`, the Zod schema should use `z.string().optional()`. This keeps the validation contract aligned with the API contract.

---

### **8. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}/ui` and TAG set to `react-stability`.

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

---

### **11. Documentation**

Use `knowledge-observatory-tools` to read the current `problems` doc for `{{TARGET}}`, then update the **Stability Issues** section with your findings (crash-prone patterns, error boundary gaps, TypeScript/ESLint improvements, remaining stability risks).
