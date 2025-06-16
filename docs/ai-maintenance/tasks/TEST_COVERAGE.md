## Purpose

Increasing **line, branch, and function** coverage uncovers silent paths and prevents future regressions, but the goal is *robust behaviour assurance*, not an arbitrary 100 %. An 80-90 % target usually balances cost vs. benefit for most codebases ([stouf.medium.com][1]).

---

## Hard Rules (from `CLAUDE.md`)

| Rule                                                 | Why                                            |
| ---------------------------------------------------- | ---------------------------------------------- |
| **No Git usage**                                     | Supervisor schedules jobs; human reviews diffs |
| **Keep `.js` extensions in TS imports**              | Node ESM requirement                           |
| **Never mock Redis/PostgreSQL – use Testcontainers** | Preserve integration fidelity                  |
| **Prefer editing existing files**                    | Control project size & deps                    |
| **Don’t add packages without permission**            | Prevent prod drift                             |

---

## Fast Checklist

1. **Generate baseline**: `pnpm --filter <pkg> run test-coverage` (alias for `vitest run --coverage`, 8 GB heap) ([vitest.dev][2]).
2. **Collect JSON summaries**  
   * High-level: `coverage/coverage-summary.json`  
   * Per-file:   `coverage/coverage-final.json`

     ```bash
     pnpm --filter <pkg> run test-coverage

     # Pretty-print overall numbers
     jq '.total.lines.pct, .total.branches.pct, .total.functions.pct' \
        coverage/coverage-summary.json
     ```

3. **Sort lowest-covered files**  
   ```bash
   jq -r '.[] | select(.lines) | "\(.path):\(.lines.pct)"' \
       coverage/coverage-final.json \
     | sort -t':' -k2,2n | head -20
   ```
4. Target *critical logic* with < 60 % lines *or* 0 % branches.
5. Use **table-driven (`test.each`)** or **property-based (fast-check)** patterns to hit edges quickly ([the-koi.com][4], [npmjs.com][5]).
6. Keep assertions high-level; never add tests that merely touch internals to “paint the wall green” ([dev.to][6]).
7. For rapid loops: `vitest --coverage --changed` – be aware coverage math may under-count untouched files ([github.com][7]).
8. Adjust package-local thresholds **only after** green run; don’t break CI unexpectedly ([vitest.dev][8]).

---

## Prerequisites

* Workspace starts **green** (`pnpm run test`).
* `ripgrep` installed for comment queries.
* Docker running (Testcontainers).
* Each package already exposes:

Vitest v3’s **V8 provider** instruments fast and ships reporters `text, html, clover, json`; c8 remaps TS source-maps automatically ([vitest.dev][2], [github.com][9]).

---

## Step-by-Step Procedure

### 1  Select target files

```bash
# A. Never checked for TEST_COVERAGE. Make sure to check the requested package, or all packages if specific package was specified.
rg -L "AI_CHECK:.*TEST_COVERAGE" packages/shared/src --type ts --type tsx

# B. Poorest performers
jq -r '.[] | "\(.path):\(.lines.pct)"' coverage/coverage-final.json \
  | sort -t':' -k2,2n | head -20
```

### 2  Analyse gaps

| Symptom                             | Remedy / Technique                                                                |
| ----------------------------------- | --------------------------------------------------------------------------------- |
| **0 % branches hit**                | Write parametric tests toggling each conditional path.                            |
| **Early-return functions untested** | Feed edge-case inputs in a `describe.each` table.                                 |
| **Error paths uncovered**           | Simulate invalid params and assert thrown errors.                                 |
| **Over-mocking hides logic**        | Use real modules; stub only I/O or randomness.                                    |
| **Wrong coverage due to maps**      | Ensure `compilerOptions.sourceMap=true`; let c8 remap TS → JS ([github.com][10]). |

### 3  Write / improve tests

* Use `test.each` / `describe.each` for breadth ([the-koi.com][4]).
* Apply `@fast-check/vitest` for property-based fuzz where appropriate ([npmjs.com][5]).
* Keep **Red → Green → Refactor** micro-cycle.

### 4  Run coverage focus loop

```bash
vitest run --coverage --changed   # quick iteration
pnpm --filter shared run test-coverage  # full package sweep. Pick the correct package, or all packages if specific package was specified.

# Print delta since last run (requires jq & previous json saved as baseline)
jq -r '.total.lines.pct' coverage/coverage-summary.json > .coverage.latest
diff -u .coverage.baseline .coverage.latest || true

# (Optional) store .coverage.latest for the next iteration.
```

### 5  Enforce / tune thresholds

```ts
// vitest.config.ts in package root
export default defineConfig({
  test: {
    coverage: {
      lines: 75,
      branches: 70,
      functions: 80,
      statements: 75,
      thresholdAutoUpdate: false, // manual bumps only
    },
  },
})
```

Use numbers from `coverage-summary.json` to decide when to raise thresholds:

```bash
newLines=$(jq '.total.lines.pct' coverage/coverage-summary.json)
[[ $newLines -gt 75 ]] && echo "Consider bumping lines threshold to $newLines"
```

Thresholds apply per-package; keep them realistic to avoid “coverage hell”.  

**Never** attempt to launch a browser (`open`, `xdg-open`) inside the agent.  
Headless environments often lack display servers and the call would hang.

### 6  Update the `AI_CHECK` comment

Insert **after imports**:

```ts
// AI_CHECK: TEST_COVERAGE=<incrementedCount> | LAST: 2025-06-16
```

Increment only **TEST\_COVERAGE**; leave other counts intact. **Do not run Git**.

---

## After the Run

* Print a plaintext summary:

  * Files touched
  * Old → new coverage (%)
  * Remaining notable gaps
* Exit **0** if thresholds met; non-zero otherwise.

---

## Reference Anti-Patterns

| Anti-Pattern                     | Why It Hurts                                               | Source |
| -------------------------------- | ---------------------------------------------------------- | ------ |
| **Shallow assertion stubs**      | Inflates numbers without guarding behaviour ([dev.to][6])  |        |
| **Chasing 100 %**                | Diminishing returns, wasted effort ([stouf.medium.com][1]) |        |
| **Instrumenting debug sessions** | Sourcemap noise, skewed metrics ([stackoverflow.com][12])  |        |

---

### Remember

* No Git operations.
* No new dependencies unless approved.
* Keep `.js` import extensions.
* Use Testcontainers for infra.

Following this guide lifts *meaningful* coverage across the monorepo while fitting neatly into the night-shift maintenance pipeline.

[1]: https://stouf.medium.com/test-coverage-n-does-not-really-matter-d069bf9ccd57?utm_source=chatgpt.com "The (in)famous 80% of code coverage | by stouf - Medium"
[2]: https://vitest.dev/guide/coverage?utm_source=chatgpt.com "Coverage | Guide - Vitest"
[3]: https://vitest.dev/guide/reporters?utm_source=chatgpt.com "Reporters | Guide - Vitest"
[4]: https://www.the-koi.com/projects/parameterized-data-driven-tests-in-vitest-example/?utm_source=chatgpt.com "Parameterized (data-driven) Tests in Vitest + example - The Koi"
[5]: https://www.npmjs.com/package/%40fast-check/vitest?utm_source=chatgpt.com "@fast-check/vitest - npm"
[6]: https://dev.to/d_ir/do-you-aim-for-80-code-coverage-let-me-guess-which-80-it-is-1fj9?utm_source=chatgpt.com "Do you aim for 80% code coverage? Let me guess which 80% you ..."
[7]: https://github.com/vitest-dev/vitest/issues/5237?utm_source=chatgpt.com "Coverage with Changed · Issue #5237 · vitest-dev/vitest - GitHub"
[8]: https://vitest.dev/config/?utm_source=chatgpt.com "Configuring Vitest"
[9]: https://github.com/bcoe/c8?utm_source=chatgpt.com "bcoe/c8: output coverage reports using Node.js' built in ... - GitHub"
[10]: https://github.com/vitest-dev/vitest/discussions/3813?utm_source=chatgpt.com "Coverage sourcemap remapping · vitest-dev vitest · Discussion #3813"
[11]: https://github.com/vitest-dev/vitest/discussions/6143?utm_source=chatgpt.com "Configuring Vitest coverage thresholds decimal place significance"
[12]: https://stackoverflow.com/questions/79020692/vitest-coverage-enabled-but-missing-html-reporter?utm_source=chatgpt.com "Vitest \"coverage enabled but missing html reporter\" - Stack Overflow"
