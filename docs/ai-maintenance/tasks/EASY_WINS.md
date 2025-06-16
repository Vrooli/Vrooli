## 🚀 What Counts as an *Easy Win*?

*Easy wins* are **tiny, mechanical fixes that:**

* are *automatically detectable* (linters, formatters, simple static analysis)
* produce an **immediate quality or DX boost** (readability, smaller bundle, fewer warnings)
* **never change runtime behaviour** in a non-obvious way
* can be finished, lint-clean and test-clean inside a single conversation

Think of them as the “🍌 bananas at eye level” of the repo. ESLint autofixes, Prettier formatting, unused-code removal, one-line React perf tweaks, patch-level dependency bumps—all perfect examples.

---

## 📏 Hard Rules (inherit from `CLAUDE.md`)

| Rule                                      | Why                         |
| ----------------------------------------- | --------------------------- |
| **No Git commands**                       | Human must review & commit. |
| **No new deps** unless explicitly allowed | Prevents prod drift.        |
| **Keep `.js` extensions in TS imports**   | Node ESM requirement.       |
| **Never mock infra** (Redis / Postgres)   | Fidelity in tests.          |

---

## 📝 Step-by-Step Procedure

### 1 Discover the Lowest-Hanging Fruit

| Target                             | Command                                                   | Notes                                                                                                      |                                                                               |
| ---------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| **Autofixable ESLint issues**      | `pnpm lint -- --fix`                                      | ESLint can repair many problems automatically ([eslint.org][1]).                                           |                                                                               |
| **Unformatted files**              | `pnpm prettier -c "**/*.{ts,tsx,js,jsx}"`                 | Prettier enforces consistent style in seconds ([prettier.io][2]).                                          |                                                                               |
| **Unused imports / locals**        | `pnpm tsc --noEmit --noUnusedLocals --noUnusedParameters` | Surfaced by the TS compiler ([effectivetypescript.com][3]).                                                |                                                                               |
| **Dead code**                      | `pnpm ts-prune --ignore "**/*.test.*"` or `pnpm knip`     | `ts-prune` / `knip` find unreachable exports ([effectivetypescript.com][3], [effectivetypescript.com][4]). |                                                                               |
| **Stray `console.*`**              | `rg "console\\." --glob '!**/*.test.*'`                   | `no-console` keeps logs clean ([eslint.org][5]).                                                           |                                                                               |
| **Inline funcs in React render**   | \`rg "=>.\*(" packages/ui/src                             | rg -B1 -A1 "(<.*on\[A-Z].*{)"\`                                                                            | Inline callbacks can trigger extra re-renders ([ishwar-rimal.medium.com][6]). |
| **Missing ARIA roles/props**       | `pnpm lint -- --rule 'jsx-a11y/*:error'`                  | `eslint-plugin-jsx-a11y` flags easy a11y wins ([github.com][7]).                                           |                                                                               |
| **Patch/minor dependency bumps**   | Dependabot or `pnpm up --latest`                          | Automated PRs keep you secure & green ([docs.github.com][8], [colbyfayock.com][9]).                        |                                                                               |
| **Incremental `strictNullChecks`** | Add to one package’s tsconfig & fix surfaced null bugs    | Small steps = safer migration ([kevinwil.de][10], [dev.to][11]).                                           |                                                                               |

Prioritise **files with no `AI_CHECK`**, then those whose `EASY_WINS` count is lowest or `LAST` date oldest.

---

### 2 Fix, Format, Verify

1. **Run the single autofix / change.**
2. **`pnpm run test && pnpm lint`** must stay green.
3. Optionally run `pnpm exec size-limit` (or equivalent) to confirm bundle hasn’t grown; tree-shaking plus dead-code removal often *shrinks* it ([dev.to][12]).

---

### 3 Update the `AI_CHECK` Comment

Place immediately after imports (top of file):

```ts
// AI_CHECK: EASY_WINS=<incrementedCount> | LAST: 2025-06-16
```

* Increment **only** `EASY_WINS`.
* Do **not** touch other task counters.
* Leave working tree dirty; supervisor handles next job.

---

## ✅ Fast Checklist Before Moving On

| ✔︎                                                        | Item |
| --------------------------------------------------------- | ---- |
| Code passes **tests & lints** (including `--fix` run).    |      |
| No new packages added.                                    |      |
| Change is mechanical / behaviour-neutral.                 |      |
| `AI_CHECK` updated correctly.                             |      |
| You wrote a one-line summary to stdout (file path + fix). |      |

---

## 🔍 Helpful Query Cookbook

```bash
# Files never touched for EASY_WINS
rg -L "AI_CHECK:.*EASY_WINS" packages -t ts -t tsx

# Oldest EASY_WINS checks
rg "AI_CHECK:.*EASY_WINS=.*\\| LAST:" -n \
  | sort -t':' -k3,3

# Count total easy wins done so far
rg "AI_CHECK:.*EASY_WINS=(\\d+)" -o -r '$1' -t ts -t tsx \
  | awk '{sum+=$1} END {print "Total EASY_WINS:", sum}'
```

---

## 🤔 Why These Particular Wins Matter

* **ESLint `--fix`** eliminates whole classes of bugs with zero thinking ([github.com][13]).
* **Prettier** enforces repo-wide formatting consistency, cutting PR noise ([medium.com][14]).
* **Dead-code pruning** reduces bundle size & mental overhead ([effectivetypescript.com][3], [effectivetypescript.com][4]).
* **No `console.*` in production** keeps logs actionable & hides sensitive data ([eslint.org][5]).
* **Micro React perf tweaks** (memoised callbacks) prevent unnecessary re-renders ([ishwar-rimal.medium.com][6], [dev.to][12]).
* **A11y lint rules** catch invalid ARIA usage early, boosting inclusivity ([github.com][7], [github.com][15]).
* **Automated dependency bumps** patch CVEs swiftly with minimal risk ([docs.github.com][8], [colbyfayock.com][9]).
* **Gradual `strictNullChecks`** adoption surfaces real null bugs while staying tractable ([kevinwil.de][10], [dev.to][11]).

---

### Remember

* **Keep it mechanical.** If you find yourself debating architecture, it’s not an easy win.
* **Ship the low-hanging fruit, then move on.** Continuous tiny gains add up faster than rare big refactors.

Happy hunting! 🛠️

[1]: https://eslint.org/?utm_source=chatgpt.com "Find and fix problems in your JavaScript code - ESLint - Pluggable ..."
[2]: https://prettier.io/docs/why-prettier?utm_source=chatgpt.com "Why Prettier?"
[3]: https://effectivetypescript.com/2020/10/20/tsprune/?utm_source=chatgpt.com "Finding dead code (and dead types) in TypeScript"
[4]: https://effectivetypescript.com/2023/07/29/knip/?utm_source=chatgpt.com "️ Use knip to detect dead code and types - Effective TypeScript"
[5]: https://eslint.org/docs/latest/rules/no-console?utm_source=chatgpt.com "no-console - ESLint - Pluggable JavaScript Linter"
[6]: https://ishwar-rimal.medium.com/stop-passing-inline-functions-as-a-prop-in-react-cf39efc60b82?utm_source=chatgpt.com "Avoid passing inline functions as a prop in React. | by Ishwar Rimal"
[7]: https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/blob/main/docs/rules/aria-role.md?utm_source=chatgpt.com "eslint-plugin-jsx-a11y/docs/rules/aria-role.md at main - GitHub"
[8]: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference?utm_source=chatgpt.com "Dependabot options reference - GitHub Docs"
[9]: https://colbyfayock.com/posts/automatically-update-npm-package-dependencies-with-dependabot?utm_source=chatgpt.com "Automatically Update npm Package Dependencies with Dependabot"
[10]: https://kevinwil.de/incremental-migration/?utm_source=chatgpt.com "Incremental Migration - Kevin Wilde"
[11]: https://dev.to/viridia/typescript-strictnullchecks-a-migration-guide-3glo?utm_source=chatgpt.com "TypeScript \"strictNullChecks\" - a migration guide - DEV Community"
[12]: https://dev.to/wallacefreitas/optimizing-re-rendering-in-react-best-practices-5683?utm_source=chatgpt.com "Optimizing Re-Rendering in React: Best Practices - DEV Community"
[13]: https://github.com/eslint/eslint/issues/5329?utm_source=chatgpt.com "Fixing autofix · Issue #5329 · eslint/eslint - GitHub"
[14]: https://medium.com/geekculture/3-reasons-to-use-prettier-for-code-formatting-91d04c064538?utm_source=chatgpt.com "3 reasons to use Prettier for code formatting | by Nic Chong - Medium"
[15]: https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/blob/main/docs/rules/aria-props.md?utm_source=chatgpt.com "eslint-plugin-jsx-a11y/docs/rules/aria-props.md at main - GitHub"
