## TL;DR

* Dead code is still discovered with `ts-prune`, `knip`, `depcheck`, etc.
* **Instead of deleting**, you should **wrap or annotate** the code with special comments and the JSDoc `@deprecated` tag.
* These flags keep the repo compiling/tests green, while letting a human grep for `DEAD_CODE_MARK` to finish the job manually.
* The `AI_CHECK` counter is still incremented, signalling a quarantine pass.

---

## 1  Purpose & Scope

`DEAD_CODE` tasks now **identify *and quarantine*** unused exports, files, or dependencies without removing them. This balances technical-debt reduction with maximal safety in an automated run. Deleting code too aggressively can break hidden runtime references or docs; postponing the final cut to human review is a proven best-practice ([reddit.com][1]).

---

## 2  Hard Rules (inherits `CLAUDE.md`)

| Rule                                                 | Why                       |
| ---------------------------------------------------- | ------------------------- |
| No Git commands                                      | Human review gate.        |
| **Never delete code** – only quarantine with markers | Ensures reversibility.    |
| Keep `.js` extensions in TS imports                  | Node ESM.                 |
| Tests, lints, and compile must stay green            | Guarantees safe flagging. |

---

## 3  Quarantine Format

### 3.1  Symbol-Level Marker

```ts
/**
 * @deprecated UNUSED – flagged by ts-prune on 2025-06-16
 * @tags DEAD_CODE_MARK
 */
export function unusedFoo() {
  // …original body…
}
```

* Uses JSDoc `@deprecated` so IDEs strike-out the symbol ([stackoverflow.com][2], [jsdoc.app][3]).
* `@tags DEAD_CODE_MARK` (or any unique string you like) makes grepping trivial.
* Leave the original identifier/export in place—imports won’t fail.

### 3.2  Block / File Marker

```ts
// ☠ DEAD_CODE_MARK BEGIN – unused file detected by knip (2025-06-16)
// Original filename: src/utils/legacyThing.ts
/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/ban-ts-comment */

…entire file content…

// ☠ DEAD_CODE_MARK END
```

* ESLint suppressions quiet “unused” warnings ([stackoverflow.com][4]).
* Keep TS compile happy until final deletion.

---

## 4  Detection Toolkit (unchanged)

| Tool                          | Finds                     | Command                               |
| ----------------------------- | ------------------------- | ------------------------------------- |
| TypeScript `--noUnusedLocals` | unused locals             | `pnpm tsc …` ([stackoverflow.com][5]) |
| **ts-prune**                  | unused exports            | `pnpm ts-prune` ([github.com][6])     |
| **knip**                      | unused files/exports/deps | `pnpm knip` ([stackoverflow.com][2])  |
| **depcheck**                  | unused npm deps           | `pnpm depcheck` ([reddit.com][7])     |

---

## 5  Step-by-Step Workflow

1. **Baseline green check** – `pnpm test && pnpm lint`.
2. **Run detectors**; pick highest-ROI items (no previous `DEAD_CODE_MARK`).
3. **Quarantine**

   * Add JSDoc+marker above symbol **OR** wrap file in block markers (see §3).
   * Add lint suppressions only inside the quarantined block.
4. **Re-run tests & lints** – must stay green.
5. **Update AI\_CHECK** in *every quarantined file*:

   ```ts
   // AI_CHECK: DEAD_CODE=<incrementedCount> | LAST: 2025-06-16
   ```
6. **Stdout summary** (file → reason flagged).

---

## 6  Grep & Review Commands

```bash
# List all quarantined symbols/files
rg "DEAD_CODE_MARK"

# Count quarantines per package
rg "DEAD_CODE_MARK" packages/* | cut -d/ -f1 | sort | uniq -c

# Show oldest quarantines
git log --reverse --pretty=format:"%ad %h %f" -S"DEAD_CODE_MARK"
```

---

## 7  Fast Checklist

| ✔︎                                    | Item |
| ------------------------------------- | ---- |
| Marker added & `@deprecated` present. |      |
| ESLint/TS compile pass.               |      |
| No runtime behaviour changed.         |      |
| AI\_CHECK updated for `DEAD_CODE`.    |      |
| Stdout summary printed.               |      |

---

## 8  Why Not Just Delete?

* **Hidden reachability:** runtime `require()` / reflection can mask usage ([github.com][6]).
* **Docs and blog snippets:** external consumers may still rely on APIs; deprecation period recommended ([css-tricks.com][8]).
* **CI safety:** quarantine keeps builds green for incremental clean-up, aligned with gradual strategies advocated by teams on Reddit & HN ([reddit.com][7], [news.ycombinator.com][9]).
* **Tool false positives:** knip/ts-prune allow ignores but a visible marker is self-documenting ([github.com][10]).

---

## 9  Reference Reading

* *Pros vs. cons of deleting unused code* – Stack Overflow ([stackoverflow.com][5])
* *ts-prune* repo ([github.com][6])
* *Knip* docs + ignore rules ([stackoverflow.com][2], [github.com][10])
* *depcheck* package ([reddit.com][7])
* *JSDoc @deprecated* tag ([jsdoc.app][3])
* ESLint suppression patterns ([stackoverflow.com][4])
* CSS-Tricks on deprecating JavaScript ([css-tricks.com][8])
* Azul blog on dead-code inventory ([azul.com][11])
* DEV.to guide to better comments ([dev.to][12])
* Reddit & HN discussions on dead-code strategies ([reddit.com][1], [news.ycombinator.com][9])

---

### Remember

> **Automated pruning should be *reversible*.** Flag first, let the human review and delete later.

[1]: https://www.reddit.com/r/AskProgramming/comments/1dfiqmg/mostly_dead_code_what_do/?utm_source=chatgpt.com "(Mostly) dead code -- what do? : r/AskProgramming - Reddit"
[2]: https://stackoverflow.com/questions/53177858/how-to-mark-a-class-as-deprecated?utm_source=chatgpt.com "How to mark a class as deprecated? - Stack Overflow"
[3]: https://jsdoc.app/tags-deprecated?utm_source=chatgpt.com "deprecated tag - Use JSDoc"
[4]: https://stackoverflow.com/questions/46284405/how-can-i-use-eslint-no-unused-vars-for-a-block-of-code?utm_source=chatgpt.com "How can I use ESLint no-unused-vars for a block of code?"
[5]: https://stackoverflow.com/questions/15699995/could-someone-explain-the-pros-of-deleting-or-keeping-unused-code?utm_source=chatgpt.com "Could someone explain the pros of deleting (or keeping) unused ..."
[6]: https://github.com/nadeesha/ts-prune?utm_source=chatgpt.com "nadeesha/ts-prune: Find unused exports in a typescript project."
[7]: https://www.reddit.com/r/webdev/comments/qmn4rq/what_do_you_do_about_todo_comments_in_your/?utm_source=chatgpt.com "What do you do about TODO comments in your codebase? - Reddit"
[8]: https://css-tricks.com/approaches-to-deprecating-code-in-javascript/?utm_source=chatgpt.com "Approaches to Deprecating Code in JavaScript | CSS-Tricks"
[9]: https://news.ycombinator.com/item?id=41554014&utm_source=chatgpt.com "Remove unused code from your TypeScript project - Hacker News"
[10]: https://github.com/webpro-nl/knip/issues/837?utm_source=chatgpt.com "Flag unused ignore/ignoreDependencies rules · Issue #837 - GitHub"
[11]: https://www.azul.com/blog/code-inventory-remove-dead-code-for-easier-maintenance/?utm_source=chatgpt.com "Code Inventory®: Remove Unused and Dead Code for Easier ..."
[12]: https://dev.to/adammc331/todo-write-a-better-comment-4c8c?utm_source=chatgpt.com "//TODO: Write a better comment - DEV Community"
