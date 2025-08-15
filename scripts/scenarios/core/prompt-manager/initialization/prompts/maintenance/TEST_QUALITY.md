## Purpose

Behaviour-driven tests written with **Vitest** act as executable specifications: they break when user-visible behaviour changes and pass when it is correct. This playbook shows the maintenance agent how to *find*, *evaluate* and *upgrade* such tests while leaving the working tree dirty for morning review.

> **‼️ Never run any Git commands (commit, stash, push, rebase, etc.).**
> Overnight maintenance must finish with uncommitted changes so a human can review and commit.

---

## Hard Rules (from `CLAUDE.md`)

| Rule                                                    | Why                                             |
| ------------------------------------------------------- | ----------------------------------------------- |
| **No Git usage**                                        | Supervisor schedules jobs; human reviews diffs. |
| **Keep `.js` extensions in TS imports**                 | Required for Node ESM.                          |
| **Never mock Redis or PostgreSQL – use Testcontainers** | Preserves integration fidelity.                 |
| **Prefer editing existing files**                       | Controls project size & deps.                   |
| **Don’t add packages** without permission               | Prevents unexpected prod drift.                 |

---

## Fast Checklist

1. One behaviour per test; follow **Arrange → Act → Assert** pattern. ([vitest.dev][1])
2. Query the UI via **@testing-library** accessible selectors (`getByRole`, etc.). ([vitest.dev][2])
3. **Minimise mocks** – stub only network, timers or randomness; keep real collaborators. ([vitest.dev][3])
4. Replace **snapshot abuse** with targeted assertions. ([github.com][4])
5. Honour the **Test Pyramid** (unit ≫ integration ≫ e2e). ([github.com][5])
6. Ensure determinism: seed RNG, use Vitest fake timers. ([vitest.dev][6])
7. Use Vitest’s **`--changed`** flag to loop over only-modified suites. ([vitest.dev][7])
8. Apply **Red → Green → Refactor** for every fix. ([v0.vitest.dev][8])

---

## Prerequisites

* Workspace must start **green**: `pnpm run test` passes.
* `ripgrep (rg)` installed for comment queries.
* Docker running (Testcontainers).
* The project’s root `pnpm run test` script invokes Vitest; all CLI examples below extend that.

### ⏱️ Timeout Reminder for Long-Running Commands

Many test and build commands take longer than the default 2-minute timeout. Always remember to set appropriate extended timeouts when running:
- Single test files: Usually need 3+ minutes
- Package test suites: Usually need 5+ minutes  
- Full test suite: Can take up to 10 minutes
- Type checking: Usually needs 3-4 minutes

---

## UI Component Testing Strategy

When testing UI components, focus on **behavior over visuals**:

### What to Test in UI Components
- **User interactions**: Click handlers, form submissions, keyboard navigation
- **Accessibility**: ARIA attributes, roles, labels, keyboard support
- **Component state**: Conditional rendering, prop changes, state transitions
- **Data flow**: Props validation, event emission, callback invocation
- **Error handling**: Loading states, error boundaries, validation messages

### What NOT to Test in UI Components
- ❌ **CSS classes or styles**: These are implementation details
- ❌ **Visual appearance**: Layout, colors, spacing (use Storybook instead)
- ❌ **Computed styles**: getComputedStyle() results
- ❌ **Exact HTML structure**: Unless semantically important

### Example: Good vs Bad UI Tests
```typescript
// ❌ BAD: Testing implementation details
expect(button.className).toContain("tw-bg-blue-500");
expect(wrapper.style.marginTop).toBe("16px");

// ✅ GOOD: Testing behavior
expect(screen.getByRole("button", { name: "Submit" })).toBeDefined();
expect(onSubmit).toHaveBeenCalledTimes(1);
expect(input.getAttribute("aria-invalid")).toBe("true");
```

### Visual Testing with Storybook
Reserve visual regression testing for Storybook:
- Component appearance across themes
- Responsive design breakpoints
- Animation and transition behavior
- Visual consistency across browsers

---

## Step-by-Step Procedure

### 1 Select target files

```bash
# A. Tests in shared package never checked for TEST_QUALITY. Make sure to check the requested package, or all packages if specific package was specified.
find packages/shared -name "*.test.ts*" \
  | xargs rg -L "AI_CHECK:.*TEST_QUALITY"

# B. Re-review oldest tests
rg "AI_CHECK:.*TEST_QUALITY=.*\\| LAST:" -n \
  | sort -t':' -k3,3
```

Work through list A first, then oldest in B.

---

### 2 Verify test is actually running before making changes

**CRITICAL**: Before modifying any test, verify it passes in its current state:

```bash
# Run the specific test file first
# ⚠️ Use extended timeout for test runs
cd packages/[package] && pnpm test path/to/specific.test.ts  # needs 3+ min timeout

# If that passes, make your changes
```

⚠️ **If the test doesn't pass initially, DO NOT PROCEED** - investigate why it's failing first.

---

### 3 Evaluate each test

| Symptom                                                                   | Remedy                                                                          |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **Implementation assertions** (private state, enzyme‐style introspection) | Switch to public API / DOM behaviour assertions. ([vitest.dev][2])              |
| **Multiple behaviors in one test**                                       | Split into focused test cases with clear names.                                 |
| **Excessive mocks**                                                       | Keep real modules; mock only side-effects (fetch, Date, RNG). ([vitest.dev][3]) |
| **Giant snapshots**                                                       | Replace with explicit expects or slim snapshots. ([github.com][4])              |
| **Brittle CSS selectors**                                                 | Prefer role / label queries. ([vitest.dev][2])                                  |
| **Visual/style assertions** (CSS classes, computed styles)                | Test behavior only; use Storybook for visual testing                           |
| **Flaky randomness / timers**                                             | Use `vi.useFakeTimers()` / seeded RNG. ([vitest.dev][6])                        |

---

### 4 Critical checks before refactoring

**Before changing ANY test code, verify these common breakage patterns:**

| Check | Why It Matters | How to Verify |
|-------|----------------|---------------|
| **Import extensions** | All TS imports MUST use `.js` extension | Ensure all relative imports end with `.js` |
| **Database setup** | Tests need proper DB init/shutdown | Look for `beforeAll` with `DbProvider.init()` |
| **Logger mocking** | Prevents noisy test output | Check for logger spy implementations |
| **Mock restoration** | Prevents test pollution | Verify `vi.restoreAllMocks()` in `afterAll` |
| **Transaction wrapper** | Ensures DB isolation | Use `withDbTransaction()` for DB tests |
| **Async handling** | Prevents flaky tests | All async operations properly awaited |
| **Test data cleanup** | Prevents constraint violations | Track and clean up created entities |

**Example of proper test structure:**
```typescript
import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { DbProvider } from "../../db/provider.js"; // Note .js extension!
import { withDbTransaction } from "../../__test/helpers/transactionTest.js";
import { logger } from "../../utils/logger.js";

describe("MyFeature", () => {
    beforeAll(async () => {
        await DbProvider.init();
        vi.spyOn(logger, "error").mockImplementation(() => logger);
    });
    
    afterAll(async () => {
        vi.restoreAllMocks();
    });
    
    it("should behave correctly", async () => {
        await withDbTransaction(async (sessionContext) => {
            // Test implementation
        });
    });
});
```

---

### 5 Refactor tests (and source when necessary)

1. **Red** – confirm a failing state (or write a failing test).
2. **Green** – fix code or improve test until it passes.
3. **Refactor** – clean up while tests stay green. ([v0.vitest.dev][8])

⚠️ **After EVERY change**, run the test again to ensure it still passes!

---

### 6 Run focused test loop

```bash
# Run the specific test file after EACH change
# ⚠️ IMPORTANT: Tests can take 3-5+ minutes. Set extended timeout
cd packages/[package] && pnpm test path/to/specific.test.ts  # needs 3-5 min timeout

# Only after the individual test passes, run full package tests
cd packages/[package] && pnpm run test  # needs 5+ min timeout
```

### 7 Run full suite

```bash
# Run package-specific tests first
# ⚠️ IMPORTANT: Full test suites require extended timeouts
cd packages/[package] && pnpm run test  # needs 5+ min timeout

# Only if that passes, run full suite
pnpm run test               # root - needs 10+ min timeout
pnpm --filter shared test   # package-scoped - needs 3+ min timeout
```

All tests must be green before updating comments.

---

### 6 Update the AI\_CHECK comment

Insert or update **after imports** in each changed test file:

```ts
// AI_CHECK: TEST_QUALITY=<incrementedCount> | LAST: 2025-06-16
```

Increment only `TEST_QUALITY`; leave other task counts untouched.
⚠️ **Do not run Git commands** – leave the working tree dirty for manual commit.

---

## Common Test Breakage Scenarios

These are the most frequent ways tests break during maintenance:

### 1. **Import Extension Forgotten**
```typescript
// ❌ WRONG - Will break in Node ESM
import { helper } from "./testHelper";

// ✅ CORRECT
import { helper } from "./testHelper.js";
```

### 2. **Database Tests Without Transaction Wrapper**
```typescript
// ❌ WRONG - No isolation, may affect other tests
it("should create user", async () => {
    const user = await DbProvider.get().user.create({...});
    expect(user).toBeDefined();
});

// ✅ CORRECT - Wrapped in transaction that auto-rollbacks
it("should create user", withDbTransaction(async () => {
    const user = await DbProvider.get().user.create({...});
    expect(user).toBeDefined();
}));
```

### 3. **Circular Dependencies from Deep Imports**
```typescript
// ❌ WRONG - May cause circular dependency
import { snowflake } from "@vrooli/shared/id/snowflake.js";

// ✅ CORRECT - Import from package root
import { snowflake } from "@vrooli/shared";
```

### 4. **Async Operations Not Awaited**
```typescript
// ❌ WRONG - Test may pass but operation incomplete
it("should update user", () => {
    updateUser(userId, data); // Missing await!
    expect(user.name).toBe("New Name");
});

// ✅ CORRECT
it("should update user", async () => {
    await updateUser(userId, data);
    expect(user.name).toBe("New Name");
});
```

### 5. **Mocks Not Restored**
```typescript
// ❌ WRONG - Pollutes other tests
beforeAll(() => {
    vi.spyOn(logger, "error").mockImplementation(() => {});
});
// No cleanup!

// ✅ CORRECT
afterAll(() => {
    vi.restoreAllMocks();
});
```

---

## Proper Test Structure Template

```typescript
import { describe, it, expect, beforeAll, afterAll, vi } from "vitest";
import { withDbTransaction } from "../../__test/helpers/transactionTest.js";
import { logger } from "../../utils/logger.js";

describe("MyFeature", () => {
    beforeAll(async () => {
        // Mock logger to reduce noise
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
    });
    
    afterAll(async () => {
        vi.restoreAllMocks();
    });
    
    // For database tests, always use withDbTransaction
    it("should work with database", withDbTransaction(async () => {
        // All DB operations here are automatically rolled back
        const result = await DbProvider.get().user.create({...});
        expect(result).toBeDefined();
    }));
    
    // For non-database tests, no wrapper needed
    it("should work without database", async () => {
        const result = someNonDbFunction();
        expect(result).toBe(expected);
    });
});
```

---

## Pre-Flight Checklist

**Before submitting ANY test changes, verify:**

- [ ] Test passed BEFORE you made changes
- [ ] All imports use `.js` extensions
- [ ] Database tests wrapped with `withDbTransaction()`
- [ ] Logger is mocked to reduce noise
- [ ] All async operations are properly awaited
- [ ] Mocks are restored in `afterAll`
- [ ] No deep imports from `@vrooli/*` packages
- [ ] Test still passes after EACH change
- [ ] Package-specific tests pass (`cd packages/[pkg] && pnpm run test`)

---

## After the Run

* Output a plaintext summary of touched files & reasons (stdout).
* Exit with status 0 if **all** tests pass; non-zero otherwise.
* Let the night-time supervisor move on to the next job.

---

## Reference Anti-Patterns

| Anti-Pattern               | Why It Hurts                                      | Source |
| -------------------------- | ------------------------------------------------- | ------ |
| **Assertion-less tests**   | Provide zero protection. ([stackoverflow.com][9]) |        |
| **Everything mocked**      | Misses integration bugs. ([github.com][10])       |        |
| **Blind snapshot updates** | Masks UI regressions. ([github.com][4])           |        |
| **Random flakiness**       | Breaks CI trust. ([vitest.dev][3])                |        |

---

### Remember

* No Git operations.
* No new dependencies.
* Keep `.js` import extensions.
* Use Testcontainers for infra.
* Vitest only – no Jest/Mocha/Chai/Sinon.

Following these steps ensures tests remain behaviour-focused and dependable while fitting seamlessly into the overnight maintenance pipeline.

## Related Documentation

* **[Fixtures and Data Flow](../../testing/fixtures-and-data-flow.md)**: Comprehensive guide to fixture management and round-trip testing across UI, API, and database layers. Essential for implementing robust test data strategies that ensure data integrity throughout your application stack.

[1]: https://vitest.dev/guide/cli.html?utm_source=chatgpt.com "Command Line Interface | Guide | Vitest"
[2]: https://vitest.dev/guide/filtering?utm_source=chatgpt.com "Test Filtering | Guide - Vitest"
[3]: https://vitest.dev/guide/features?utm_source=chatgpt.com "Features | Guide - Vitest"
[4]: https://github.com/vitest-dev/vitest/issues/3962?utm_source=chatgpt.com "command line flags don't override config file · Issue #3962 - GitHub"
[5]: https://github.com/vitest-dev/vitest/issues/1113?utm_source=chatgpt.com "--changed option only takes into account changed source files and ..."
[6]: https://vitest.dev/config/?utm_source=chatgpt.com "Configuring Vitest | Vitest"
[7]: https://vitest.dev/guide/cli?utm_source=chatgpt.com "Command Line Interface | Guide - Vitest"
[8]: https://v0.vitest.dev/guide/cli?utm_source=chatgpt.com "Command Line Interface | Guide | Vitest v0.34"
[9]: https://stackoverflow.com/questions/78197248/vitest-running-but-not-not-updating-results-of-tests-after-modification-of-file?utm_source=chatgpt.com "Vitest running, but not not updating results of tests after ..."
[10]: https://github.com/vitest-dev/vitest/discussions/4057?utm_source=chatgpt.com "Coverage only modified files · vitest-dev vitest · Discussion #4057"
