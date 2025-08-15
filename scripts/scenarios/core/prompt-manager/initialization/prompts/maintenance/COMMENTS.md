## Summary

This playbook teaches the maintenance agent how to **diagnose, write, lint, and keep up-to-date code comments** so they boost clarity without rotting or duplicating the obvious. It merges industry best-practice (JSDoc / TSDoc, Google style, Clean Code) with repo-specific automation so an overnight run can add or fix comments, push readability metrics up, and leave the tree ready for morning review. ([google.github.io][1], [typescriptlang.org][2], [github.com][3], [tsdoc.org][4], [dzone.com][5], [medium.com][6], [wired.com][7], [wired.com][8])

---

## 1 Purpose & Scope

* **Task ID:** `COMMENTS`
* **Goal:** Elevate maintainability by ensuring every file, type, and public API has concise, up-to-date documentation.
* **Out-of-scope:** Large design docs, README rewrites, architectural ADRs (use `DOCS_TO_CODE` / `CODE_TO_DOCS`).

---

## 2 Hard Rules (inherits from `CLAUDE.md`)

| Rule                                        | Why                                      |
| ------------------------------------------- | ---------------------------------------- |
| No Git commands                             | Human review & commit gate.              |
| No new deps without permission              | Prevents prod drift.                     |
| Keep `.js` extensions in TS imports         | Node ESM.                                |
| Comment **must** match behaviour at PR time | Prevents “comment-rot”. ([dzone.com][5]) |

---

## 3 Comment Classes & Writing Guidelines

### 3.1 File-Header Blocks

* Top of file, after the shebang (if any).
* Explain **what** the module exports and **why it exists**, not how every line works. ([google.github.io][1], [wired.com][7])

### 3.2 Public API Docs (functions, classes, types)

* Use **JSDoc / TSDoc** `/** … */` with `@param`, `@returns`, `@example`, `@throws`, `@deprecated`. ([typescriptlang.org][2], [tsdoc.org][4])
* Describe **side-effects and invariants**; avoid restating the type signature.

### 3.3 Inline Explanatory Comments

* Prefer **why-focused** comments for non-obvious decisions.
* If code can be renamed to remove the need—rename instead. ([medium.com][6], [wired.com][7])

### 3.4 TODO / FIXME

* Format: `// TODO(you@example.com | JIRA-123): actionable text`.
* Add `AI_CHECK: TODO_CLEANUP` after resolving.

### 3.5 Style & Tone

| ✅ Do                                     | ❌ Avoid                        |
| ---------------------------------------- | ------------------------------ |
| Use present tense: “Creates a queue.”    | History lessons / jokes        |
| Wrap at 100 cols                         | ASCII art boxes                |
| Keep punctuation & spelling professional | Abbreviations (`u`, `r`, etc.) |

---

## 4 Tooling & Detection

| Target                            | Command                                              | Note                                    |   |
| --------------------------------- | ---------------------------------------------------- | --------------------------------------- | - |
| Missing JSDoc on exported members | `pnpm eslint --rule 'jsdoc/require-jsdoc:error'`     | `eslint-plugin-jsdoc` ([github.com][3]) |   |
| Tag correctness / param coverage  | `pnpm eslint --rule 'jsdoc/check-param-names:error'` |                                         |   |
| Broken TSDoc blocks               | `pnpm tsdoc --init && pnpm tsdoc --report`           | `tsdoc` CLI ([tsdoc.org][4])            |   |
| Files lacking any block comment   | `rg -L "/\\*" --glob '*.ts*' packages`               |                                         |   |
| Outdated TODOs > 90 days          | \`git grep -n "TODO"                                 | ./scripts/todo-age.ts\`                 |   |

Prioritise:

1. Public files **without any header**.
2. Exports missing JSDoc or failing lint.
3. Comments where `LAST` date > 30 days and code changed since (use `git diff`).

---

## 5 Step-by-Step Workflow

1. **Scan** with queries above.

2. **Fix/Add comments** following §3 guidelines.

3. **Run tests + lint:** `pnpm test && pnpm lint`.

4. **Update AI\_CHECK:**

   ```ts
   // AI_CHECK: COMMENTS=<incrementedCount> | LAST: 2025-06-16
   ```

5. **Stdout summary** (file → reason fixed).

6. Leave working directory dirty.

---

## 6 Fast Checklist

| ✔︎                                                                  | Item |
| ------------------------------------------------------------------- | ---- |
| Header explains **purpose & usage**.                                |      |
| Every exported symbol has accurate JSDoc.                           |      |
| No duplicated information; variable/function names stay expressive. |      |
| Lint & TSDoc pass.                                                  |      |
| `AI_CHECK` updated only for `COMMENTS`.                             |      |

---

## 7 Anti-Patterns & Smells

| Smell                                       | Why Bad                                            |
| ------------------------------------------- | -------------------------------------------------- |
| **Obvious comments** (`i++ // increment i`) | Noise, hides signal. ([wired.com][7])              |
| **Out-of-date behaviour**                   | Mismatched docs mislead clients. ([dzone.com][5])  |
| **Block-comment boxes**                     | Hard to edit, break diffs. ([google.github.io][1]) |
| **Commented-out code**                      | Use Git history; delete. ([helpjuice.com][9])      |

---

## 8 Reference Resources

* Google JavaScript Style Guide – Comments section. ([google.github.io][1])
* TypeScript Handbook – JSDoc supported types. ([typescriptlang.org][2])
* `eslint-plugin-jsdoc` rule set. ([github.com][3])
* TSDoc spec. ([tsdoc.org][4])
* DZone: Importance of comments for maintainable code. ([dzone.com][5])
* Clean Code debate on minimal comments. ([medium.com][6])
* Jeff Atwood on balanced commenting. ([wired.com][7])
* Wired discussion of comment strategies. ([wired.com][8])

---

### Remember

> Comment **why**, not **what**; automate linting; keep docs truthful. Continuous small comment fixes prevent the **big-bang rewrite nobody has time for**.

[1]: https://google.github.io/styleguide/jsguide.html?utm_source=chatgpt.com "Google JavaScript Style Guide"
[2]: https://www.typescriptlang.org/docs/handbook/jsdoc-supported-types.html?utm_source=chatgpt.com "JSDoc Reference - TypeScript: Documentation"
[3]: https://github.com/gajus/eslint-plugin-jsdoc?utm_source=chatgpt.com "gajus/eslint-plugin-jsdoc: JSDoc specific linting rules for ESLint."
[4]: https://tsdoc.org/?utm_source=chatgpt.com "What is TSDoc? | TSDoc"
[5]: https://dzone.com/articles/the-importance-of-comments-for-maintainable-code?utm_source=chatgpt.com "The Importance of Comments for Maintainable Code - DZone"
[6]: https://medium.com/%40bpnorlander/stop-writing-code-comments-28fef5272752?utm_source=chatgpt.com "Stop Writing Code Comments - by Brian Norlander - Medium"
[7]: https://www.wired.com/2008/07/commenting-your-code-what-s-too-much-too-little-?utm_source=chatgpt.com "Commenting Your Code - What's Too Much, Too Little?"
[8]: https://www.wired.com/2012/09/the-best-way-to-comment-your-code?utm_source=chatgpt.com "The Best Way to Comment Your Code"
[9]: https://helpjuice.com/blog/software-documentation?utm_source=chatgpt.com "knowledgSoftware Documentation Best Practices [With Examples]"
