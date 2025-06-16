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
* The project’s root `pnpm test` script invokes Vitest; all CLI examples below extend that.

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

### 2 Evaluate each test

| Symptom                                                                   | Remedy                                                                          |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| **Implementation assertions** (private state, enzyme‐style introspection) | Switch to public API / DOM behaviour assertions. ([vitest.dev][2])              |
| **Multiple behaviours in one test**                                       | Split into focused test cases with clear names.                                 |
| **Excessive mocks**                                                       | Keep real modules; mock only side-effects (fetch, Date, RNG). ([vitest.dev][3]) |
| **Giant snapshots**                                                       | Replace with explicit expects or slim snapshots. ([github.com][4])              |
| **Brittle CSS selectors**                                                 | Prefer role / label queries. ([vitest.dev][2])                                  |
| **Flaky randomness / timers**                                             | Use `vi.useFakeTimers()` / seeded RNG. ([vitest.dev][6])                        |

---

### 3 Refactor tests (and source when necessary)

1. **Red** – confirm a failing state (or write a failing test).
2. **Green** – fix code or improve test until it passes.
3. **Refactor** – clean up while tests stay green. ([v0.vitest.dev][8])

---

### 4 Run focused test loop

```bash
# Fast iteration – only uncommitted changes
pnpm run test -- --changed

# To compare with main branch
pnpm run test -- --changed origin/main
```

`--changed` tells Vitest to execute only suites related to changed files. ([vitest.dev][7])

### 5 Run full suite

```bash
pnpm run test               # root
pnpm --filter shared test   # package-scoped if preferred
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
