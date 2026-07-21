## Steer focus: Utils Unification

Prioritize **extracting, standardizing, and consolidating utilities** so shared logic is consistent, discoverable, and testable. The goal is to prevent duplication and drift while keeping utilities sharply scoped and aligned to screaming architecture.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

---

### **0. Why This Skill Exists**

Utility code is a silent entropy engine:
- The same logic is reimplemented with tiny differences across files
- "Helper" functions become dumping grounds with unclear ownership
- App behavior diverges because each copy evolves independently
- Tests become brittle because utilities are not structured for reuse

This skill provides a shared mental model for **when to extract utilities, where to place them, how to name them, and how to test them** so the codebase converges instead of drifting.

---

### **1. Extraction Decision Tree**

Use this flow before writing any new helper:

```
                 Need a helper function?
                          |
                          v
       +---- Search: Does similar logic exist? ----+
       |           (rg / ast-grep)                 |
       |                                           |
     FOUND                                       NOT FOUND
       |                                           |
       v                                           v
Reuse/extend existing                    Is this logic used
utility (do not clone)                   in 2+ places or
       |                                 likely to be reused?
       v                                           |
Is the existing API adequate?                      |
       |                                           |
  +----+----+                                      |
YES        NO                                      |
  |          |                                     |
  v          v                                     v
Use it   Improve it (backward-compatible)   Extract into shared
                                                   utility
```

**Rule:** If similar logic exists, **extend it**. Do not create a parallel variant.

---

### **2. Utility Classification (What Kind Is This?)**

Utilities are not all the same. Classify before placing.

```
Call Sites
   |
   v
Feature-specific helpers -> Domain utilities -> Core utilities
   (local scope)             (business logic)     (pure, generic)
```

#### **Decision Table: Utility Type**

| Question | If YES | If NO |
|----------|--------|-------|
| Is it used only within one feature/surface? | Keep local to feature | Continue... |
| Is it tied to business concepts or domain language? | Domain utility | Continue... |
| Is it framework-specific (React, Vue, Node)? | Framework utility | Continue... |
| Is it pure, general-purpose logic? | Core utility | Keep local |

**Guiding principle:** Put code where its **meaning** is clearest, not where it is most convenient.

---

### **3. Screaming Architecture for Utilities**

Avoid a single catch-all `utils/` folder. Instead, make utility structure **express intent**.

```
src/
+-- shared/
|   +-- core/            # Pure, general utilities (string, array, math)
|   +-- domain/          # Business-specific utilities (billing, auth, search)
|   +-- framework/       # React/Vue/Node-specific utilities
|   +-- validation/      # Shared validators and schemas
|   +-- formatting/      # Shared formatters (date, currency, names)
|   +-- testing/         # Test-only helpers (factories, fixtures)
+-- features/
    +-- ...
```

Adapt naming to the existing codebase, but **preserve the intent boundaries**:

- **core/** is framework-agnostic and pure
- **domain/** speaks the business language
- **framework/** wraps or adapts framework APIs
- **testing/** is for tests only (never imported into prod)

#### **Dependency Direction**

```
features -> domain -> core
features -> framework -> core
domain  X-> framework
core    X-> app code
```

**Rule:** Core utilities must not import app code, React, or environment-specific APIs.

#### **React-Specific Utility Placement**

| Utility type | Preferred location | Notes |
|--------------|--------------------|------|
| Hooks (`useX`) | `path:shared/hooks/` or `literal:framework/react/` | Hooks are utilities; keep them discoverable and stable |
| JSX helpers | `path:shared/components/` | If it returns JSX, treat it as a component |
| Classname helpers | `path:shared/core/` | Keep pure string logic out of React code |
| Data adapters | `path:shared/domain/` | Transform API data to UI-ready shapes |

Connect transport factories are framework utilities and should live in
`path:shared/framework/` or the app's API boundary substrate. Thin RPC client
wrappers that speak domain language may live in `path:shared/domain/`, but avoid
wrapping generated clients unless the wrapper removes real duplication.

---

### **4. Utility API Design Standards**

Utilities should be boring and predictable.

**Do:**
- Use explicit, typed inputs and outputs
- Prefer pure functions (no hidden globals)
- Keep return shapes stable
- Use names that describe capability, not implementation

**Avoid:**
- `helpers.ts` or `misc.ts`
- Functions that do unrelated work
- Hidden side effects (mutating input, global state)
- Optional behavior via opaque boolean flags

#### **Decision Table: API Shape**

| Situation | Preferred API | Avoid |
|-----------|----------------|-------|
| One optional behavior | Options object with defaults | Boolean flags |
| Many variations | Multiple focused functions | One mega-function |
| Needs extensibility | Named config with defaults | Position-dependent args |

Example:
```ts
// GOOD
formatCurrency(amount, { locale, currency, showSymbol })

// BAD
formatCurrency(amount, true, "USD", "en-US")
```

---

### **5. Extraction Workflow (Opinionated)**

1. **Locate duplicates** (search first)
2. **Pick a canonical implementation**
3. **Generalize only what is necessary**
4. **Create a shared utility in the right tier**
5. **Update all call sites**
6. **Delete duplicates**
7. **Add or update tests**

**Do not** extract prematurely if the code is clearly one-off.

---

### **6. Testing Seams for Utilities**

Utilities should be testable without mocking the world. Use seams.

#### **Common Seams**

| Concern | Seam Pattern | Example |
|---------|-------------|---------|
| Time | Inject clock | `now = () => new Date()` |
| Randomness | Inject RNG | `random = Math.random` |
| Environment | Pass explicit config | `env = getEnv()` |
| IO | Pass adapters | `readFile`, `fetch` |

**Rule:** Prefer dependency injection over implicit globals.

---

### **7. Test Utilities (Factories and Fixtures)**

Test helpers should be **stable, reusable, and isolated**:

```
tests/
+-- utils/          # test-only helpers
|   +-- factories/  # build data objects
|   +-- fixtures/   # static test data
|   +-- matchers/   # custom assertions
```

**Guidance:**
- Test utilities must not be imported by production code
- Factories should accept overrides (`buildUser({ name: "..." })`)
- Fixtures should be minimal and focused

---

### **8. Duplication Audit**

Run this audit before major utility refactors or extraction work.

#### **Step 1: Discover Repeated Patterns**

```bash
rg "function (format|parse|normalize|serialize|deserialize|validate)" --type ts
rg "const (format|parse|normalize|serialize|deserialize|validate)" --type ts
rg "helpers|utils|misc" --type ts -l
```

#### **Step 2: Identify Domain Drift**

```bash
rg "date|time|currency|price|amount" --type ts -g "**/*.{ts,tsx}"
rg "error|exception|fail|invalid" --type ts -g "**/*.{ts,tsx}"
```

#### **Red Flags**
- [ ] Same function name implemented in multiple files
- [ ] Slightly different formatting logic per feature
- [ ] "utils" or "helpers" files with mixed responsibilities
- [ ] Utility functions that import React or app-level modules

#### **Document Findings**

Update or create `docs/internal/UTILS_UNIFICATION_NOTES.md`:

```markdown
# Utils Unification Notes

## Last Updated
YYYY-MM-DD

## Summary
[Short state of utilities and duplication]

## Duplications
- [Function or pattern] -> [files]

## Consolidation Candidates
1. [Top target]
2. [Second target]

## Notes
- [Boundary risks, dependency issues, testing gaps]
```

---

### **9. Memory Management with Visited Tracker**

Use the `visited-tracker-tools` skill for tracking visited files, with LOCATION set to `scenarios/{{TARGET}}` and TAG set to `utils-unification`.

---

### **10. Scenario Constraints**

* Do **not** change scenario workflows, APIs, or business logic
* Do **not** introduce new features unrelated to utility consolidation
* Prefer **incremental extractions** over large rewrites
* Keep behavior identical unless a bug is explicitly targeted

---

### **11. Output Expectations**

You may update:
* Utility placement to follow architecture tiers
* Duplicate helpers consolidated into shared utilities
* Utility APIs to be consistent and explicit
* Tests to validate shared utilities

You **must**:
* Reduce duplication and prevent drift
* Keep utilities discoverable and well-named
* Preserve behavior (unless fixing known bugs)
* Ensure new utilities are testable and have seams

**Avoid superficial changes** that only rename files or move code without real consolidation.

Last updated: 2026-05-04 (Connect-RPC adoption)
