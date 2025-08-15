## 🚀 Executive Summary

High-quality code is **readable, small, loosely coupled, low in complexity, and free of surprise side-effects**.
Your job is to enforce those traits everywhere: scan for smells, refactor in safe increments, and leave an updated `AI_CHECK` trail so progress is measurable.

---

## 1 Purpose & Scope

* **Task ID:** `CODE_QUALITY`
* **Goal:** Remove code smells, lower cyclomatic complexity, kill duplication, improve naming, and surface hidden dependency pitfalls.
* **Out-of-scope:** Unit-test improvements (`TEST_QUALITY` / `TEST_COVERAGE`) and type-safety work (`UNSAFE_CASTS`).

---

## 2 Hard Rules (from `CLAUDE.md`)

| Rule                                         | Why                  |
| -------------------------------------------- | -------------------- |
| No Git commands                              | Human review gate.   |
| No new deps without permission               | Avoid stealth drift. |
| Keep `.js` import extensions                 | Node ESM.            |
| All refactors must leave tests & lints green | Safety net.          |

---

## 3 Smell Categories & Fix Strategies

| Category                               | Detection                                                                                                      | Remedy                                     |
| -------------------------------------- | -------------------------------------------------------------------------------------------------------------- | ------------------------------------------ |
| **Long / complex functions**           | `@typescript-eslint/complexity`, `cyclomatic-complexity` CLI ([github.com][1], [medium.com][2])                | Extract functions; early returns.          |
| **High cognitive complexity**          | `eslint-plugin-sonarjs` rule `cognitive-complexity` ([github.com][3])                                          | Split into helpers; reduce nesting.        |
| **Code duplication**                   | `eslint-plugin-sonarjs/no-duplicated-branches` + manual rg search ([github.com][3], [kyiv-tech.medium.com][4]) | DRY refactor; util function.               |
| **Circular dependencies**              | `madge --circular` ([npmjs.com][5], [dev.to][6])                                                               | Restructure modules or use inversion.      |
| **Layer violations / tangled imports** | `depcruise --validate .dependency-cruiser.js` ([github.com][7], [github.com][8])                               | Introduce clear boundaries.                |
| **Poor naming / unclear intent**       | Manual review + Google TS style guide checklist ([google.github.io][9], [google.github.io][10])                | Rename to intention-revealing identifiers. |
| **Missing or inconsistent formatting** | `prettier --check` ([medium.com][11])                                                                          | Run `prettier --write`.                    |
| **Too many `any`s / unsafe casts**     | Hand off to `UNSAFE_CASTS` playbook but flag them here.                                                        |                                            |
| **Dead code**                          | `ts-prune` / `knip` + `rg "TODO(.*delete)"` ([medium.com][12])                                                 | Delete or unexport.                        |
| **Console noise in production**        | `eslint no-console` rule ([github.com][3], [community.sonarsource.com][13])                                    | Replace with logger or remove.             |

---

## 4 Tooling Setup Cheat-Sheet

```bash
# 1. SonarJS rules for smells & complexity
pnpm add -D eslint-plugin-sonarjs
echo '"extends": ["plugin:sonarjs/recommended"]' >> .eslintrc.json

# 2. Complexity audit
pnpm exec cyclomatic-complexity src | sort -n -k2

# 3. Circular dependencies
pnpm exec madge --circular --extensions ts,tsx packages

# 4. Dependency boundaries
pnpm depcruise --validate .dependency-cruiser.js src

# 5. Duplicate code branches (SonarJS)
pnpm eslint --rule 'sonarjs/no-duplicated-branches:error' src
```

---

## 5 Step-by-Step Workflow

1. **Scan** using the commands in §4; dump offenders into a scratch list.
2. **Prioritise**

   * Public API files first.
   * Highest complexity / duplication scores.
   * Files with no `AI_CHECK` or oldest `LAST`.
3. **Refactor**

   * Extract small pure functions.
   * Rename symbols for clarity.
   * Remove dead branches / unused exports.
   * Break cycles by moving helpers or inverting dependencies.
4. **Verify**

   ```bash
   pnpm test && pnpm lint && pnpm tsc --noEmit
   ```
5. **Update the AI\_CHECK** at top of each touched file:

   ```ts
   // AI_CHECK: CODE_QUALITY=<incrementedCount> | LAST: 2025-06-16
   ```
6. **Stdout summary**: `file → smell removed → metric before/after`.

---

## 6 Query Cookbook

```bash
# Files never reviewed for CODE_QUALITY
rg -L "AI_CHECK:.*CODE_QUALITY" packages -t ts -t tsx

# Find highest cyclomatic complexity (>15)
pnpm exec cyclomatic-complexity src | awk '$2>15' | sort -nr -k2

# Oldest CODE_QUALITY reviews
rg "AI_CHECK:.*CODE_QUALITY=.*\\| LAST:" -n | sort -t':' -k3,3
```

---

## 7 Fast Checklist

| ✔︎                                              | Item |
| ----------------------------------------------- | ---- |
| Complexity <= 10 per function (or justified).   |      |
| No duplicate branches or dead code left behind. |      |
| No circular dependencies in `madge --circular`. |      |
| Prettier & ESLint pass without `--fix`.         |      |
| Tests and type-check green.                     |      |
| `AI_CHECK` updated only for `CODE_QUALITY`.     |      |

---

## 8 Reference Reading

* **Clean-Code-TypeScript** examples ([github.com][14])
* **SonarJS** rule set ([github.com][3])
* Cyclomatic complexity intro ([github.com][1])
* **Madge** docs ([npmjs.com][5])
* **Dependency-Cruiser** docs ([github.com][7])
* Google TypeScript Style Guide ([google.github.io][9])
* Prettier consistency article ([medium.com][11])
* DRY vs WET nuance ([kyiv-tech.medium.com][4])
* Clean Code principles for TS ([medium.com][12])
* SonarJS vs SonarQube thread ([community.sonarsource.com][13])
* Cyclomatic complexity thresholds guide ([medium.com][2])
* Madge circular-dep blog post ([dev.to][6])
* Dependency-Cruiser FAQ ([github.com][8])
* Google general style guides ([google.github.io][10])
* Reddit formatter debate ([reddit.com][15])

---

### Remember

> Clarity first, cleverness last.
> A smaller, flatter, well-named function beats a large, tangled clever one every time.

[1]: https://github.com/pilotpirxie/cyclomatic-complexity?utm_source=chatgpt.com "Detect cyclomatic complexity of your JavaScript and TypeScript code"
[2]: https://medium.com/beyond-the-code-by-typo/understanding-cyclomatic-complexity-a-developers-comprehensive-guide-820772732514?utm_source=chatgpt.com "Understanding Cyclomatic Complexity: A Developer's ... - Medium"
[3]: https://github.com/SonarSource/eslint-plugin-sonarjs?utm_source=chatgpt.com "SonarSource/eslint-plugin-sonarjs - GitHub"
[4]: https://kyiv-tech.medium.com/advanced-guide-on-dry-and-wet-principles-in-typescript-a9cc82eb3d64?utm_source=chatgpt.com "Advanced Guide on DRY and WET Principles in TypeScript"
[5]: https://www.npmjs.com/package/madge?utm_source=chatgpt.com "madge - NPM"
[6]: https://dev.to/greenroach/detecting-circular-dependencies-in-a-reacttypescript-app-using-madge-229?utm_source=chatgpt.com "Detecting Circular Dependencies in a React/TypeScript App using ..."
[7]: https://github.com/sverweij/dependency-cruiser?utm_source=chatgpt.com "sverweij/dependency-cruiser: Validate and visualize ... - GitHub"
[8]: https://github.com/sverweij/dependency-cruiser/blob/main/doc/faq.md?utm_source=chatgpt.com "dependency-cruiser/doc/faq.md at main - GitHub"
[9]: https://google.github.io/styleguide/tsguide.html?utm_source=chatgpt.com "Google TypeScript Style Guide"
[10]: https://google.github.io/styleguide/?utm_source=chatgpt.com "Google Style Guides | styleguide"
[11]: https://medium.com/geekculture/3-reasons-to-use-prettier-for-code-formatting-91d04c064538?utm_source=chatgpt.com "3 reasons to use Prettier for code formatting | by Nic Chong - Medium"
[12]: https://medium.com/%40infistem/clean-code-in-typescript-an-introduction-66b1cbefbba7?utm_source=chatgpt.com "Clean Code in TypeScript — An Introduction | by 'Goke Akintoye"
[13]: https://community.sonarsource.com/t/do-i-need-sonarqube-if-i-am-using-eslint-plugin-sonarjs-and-i-dont-need-just-a-reporting-dashboard/43766?utm_source=chatgpt.com "Do I need Sonarqube if I am using eslint-plugin-sonarjs and I don't ..."
[14]: https://github.com/labs42io/clean-code-typescript?utm_source=chatgpt.com "labs42io/clean-code-typescript - GitHub"
[15]: https://www.reddit.com/r/node/comments/w95n80/why_use_prettier_if_eslint_can_format/?utm_source=chatgpt.com "Why use prettier if ESLint can format? : r/node - Reddit"
