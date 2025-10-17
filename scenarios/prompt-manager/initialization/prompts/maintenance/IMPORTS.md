## 🚦 Quick Take

The IMPORTS maintenance task covers **three critical areas**:
1. **Missing `.js` extensions** in TypeScript imports (required by ES modules)
2. **Deep imports from monorepo packages** (e.g., `@vrooli/shared/dist/...` instead of `@vrooli/shared`)
3. **Circular dependencies** that break tree-shaking and cause runtime bugs

Circular dependencies slip in slowly, then bite hard: they break tree-shaking, hide runtime bugs, and make mental models brittle. Tools like **Madge** and **dependency-cruiser** spot the cycles instantly, but only **module redesign**—not barrel-file band-aids—truly removes them. This doc shows your overnight agent how to detect and fix all three import issues, update `AI_CHECK`, and leave a clean diff for human review. ([github.com][1], [github.com][2], [stackoverflow.com][3], [stackoverflow.com][4])

---

## 1 Purpose & Scope

| Field         | Value                                                                                                    |
| ------------- | -------------------------------------------------------------------------------------------------------- |
| **Task ID**   | `IMPORTS`                                                                                                |
| **Target**    | 1. Fix missing `.js` extensions<br>2. Fix deep monorepo imports<br>3. Detect & resolve circular imports |
| **Non-goals** | Unused imports, ordering, `import type` upgrades (done by ESLint).                                      |

---

## 2 Why Circular Dependencies Matter

1. **Runtime hazards** – Node/TS hand you *half-initialised* modules, leading to `undefined` errors or subtle state divergence. ([stackoverflow.com][3])
2. **Tree-shaking breakage** – Bundlers skip dead-code elimination when they spot a cycle, inflating bundles. ([github.com][1], [github.com][2])
3. **Type-only illusions** – TypeScript cycles may look harmless (`import type`), but future runtime code can revive them. ([github.com][5])
4. **Cognitive load** – Every new dev now needs a whiteboard to trace “who imports whom”. ([medium.com][6])

---

## 3 Hard Rules (inherits `CLAUDE.md`)

| Rule                                                       | Why                                           |
| ---------------------------------------------------------- | --------------------------------------------- |
| **No Git commands**                                        | Morning review gate.                          |
| **No barrel-file patching** (`index.ts` selective exports) | Masks the cycle; doesn’t remove it. See §5.2. |
| **Refactor over re-export**                                | Real fix = move code / invert deps.           |
| **Update `AI_CHECK`** for `IMPORTS` only.                  |                                               |

---

## 4 Detection Toolkit & Commands

| Tool                           | Install                                     | Core Command                                       | Notes                                                                            |
| ------------------------------ | ------------------------------------------- | -------------------------------------------------- | -------------------------------------------------------------------------------- |
| **Madge**                      | `pnpm add -D madge`                         | `madge --circular src/`                            | Fast graph scan; JSON output option. ([dev.to][7], [npmjs.com][8])               |
| **dependency-cruiser**         | `pnpm add -D dependency-cruiser`            | `depcruise --output-type err --rule no-circular .` | Highly configurable; ignore pure-type cycles. ([github.com][9], [npmjs.com][10]) |
| **VS Code** plug-in            | “Circular Dependency”                       | Live gutter warnings.                              |                                                                                  |
| **Webpack/Rollup** plugins\*\* | Secondary proof; surface production cycles. |                                                    |                                                                                  |

Prioritise **cycles touching `src/` public API** or **>2 modules deep**.

---

## 5 Fix Patterns

### 5.1 Refactor Playbook

| Pattern                  | When                                                                           | How                                                                                      |
| ------------------------ | ------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| **Module extraction**    | Shared helpers cause mutual imports                                            | Move helpers to `shared/` (no inward deps). ([stackoverflow.com][11])                    |
| **Dependency inversion** | A high-level module imports low-level util that also needs the high-level type | Extract an interface; low-level accepts it via param/callback. ([stackoverflow.com][12]) |
| **Feature layering**     | Cross-package imports form a loop                                              | Establish `domain → service → UI` one-way arrows; move violating code. ([medium.com][6]) |

### 5.2 Why **Barrel-Files** Are Not a Fix

> **Selective re-export in `index.ts` only hides the arrows; the underlying modules still import each other.** It postpones the explosion until a new import drags runtime code into the “hidden” cycle. The long-term maintenance cost eclipses the short-term relief. ([reddit.com][13])

---

## 6 Step-by-Step Workflow

### 6.1 Complete IMPORTS Task Workflow

1. **Check for missing `.js` extensions**
   ```bash
   # Find TypeScript imports missing .js extensions
   rg "from ['\"]\.\.?/[^'\"]*[^.js]['\"]" --type ts packages/server/src
   ```
   Fix by adding `.js` to all relative imports.

2. **Check for deep monorepo imports**
   ```bash
   # Find deep imports like @vrooli/shared/dist/...
   rg "from ['\"]@vrooli/[^'\"]+/[^'\"]+['\"]" --type ts packages/server/src
   ```
   Fix by importing from package root only (e.g., `@vrooli/shared`).

3. **Detect circular dependencies**
   ```bash
   # Install madge if not available
   pnpm add -D madge
   # Run circular dependency detection
   madge --circular packages/server/src/
   ```

4. **Fix circular dependencies** (see patterns in §5.1)

5. **Verify all fixes**
   ```bash
   pnpm lint && pnpm test
   ```

6. **Update AI_CHECK comment**
   ```ts
   // AI_CHECK: IMPORTS=<incrementedCount> | LAST: 2025-06-16
   ```

### 6.2 Circular Dependency Resolution Steps

1. **Detect cycles**

   ```bash
   madge --json --circular src/ > cycles.json
   ```
2. **Pick a cycle** (fewest visits in `AI_CHECK`, deepest path length).
3. **Choose a pattern** (§5.1) and refactor.
4. **Verify**

   ```bash
   pnpm lint && pnpm test && madge --circular src/ # should report 0
   ```
5. **Update comment**

   ```ts
   // AI_CHECK: IMPORTS=<incrementedCount> | LAST: 2025-06-16
   ```
6. **Print summary** to stdout; leave tree dirty.

---

## 7 Checklist Before Leaving File

* [ ] All relative imports have `.js` extensions
* [ ] No deep imports from monorepo packages (only root imports)
* [ ] Cycle removed (`madge --circular` returns 0)
* [ ] No new barrel files as patch
* [ ] Lint & tests green
* [ ] `AI_CHECK` updated

---

## 8 Anti-Patterns & Smells

| Smell                                             | Why Bad                                                                   |
| ------------------------------------------------- | ------------------------------------------------------------------------- |
| **`export * from "./module"` quick-fix**          | Hides but doesn’t break cycle. ([reddit.com][13])                         |
| **Lazy `require()` inside function**              | Runtime order bugs remain, tests may still pass. ([stackoverflow.com][3]) |
| **Marking import as `type` only** to silence tool | Breaks when later code needs value import. ([github.com][5])              |

---

## 9 Resources

1. Madge docs & usage examples. ([dev.to][7], [npmjs.com][8])
2. dependency-cruiser rules reference. ([github.com][9])
3. Rollup issue: tree shaking fails with cycles. ([github.com][1])
4. UI-Router case: bundle bloat via cycles. ([github.com][2])
5. Node.js circular require pitfalls. ([stackoverflow.com][3])
6. Type-only cycle nuance. ([github.com][5])
7. Reddit /r/typescript barrel-file warning. ([reddit.com][13])
8. Medium “fix nasty cycles” guide. ([medium.com][6])
9. StackOverflow best-practice answers. ([stackoverflow.com][11], [stackoverflow.com][12])
10. When to worry about cycles (dynamic import limits). ([stackoverflow.com][4])

---

### Remember

> **Eliminate, don’t obfuscate, the import arrows.** Future maintainers (and bundlers) will thank you.

[1]: https://github.com/rollup/rollup/issues/1554?utm_source=chatgpt.com "[bug] tree shaking should not happen with circular dependencies."
[2]: https://github.com/ui-router/core/issues/309?utm_source=chatgpt.com "Circular dependency cause tree-shaking not working. #309 - GitHub"
[3]: https://stackoverflow.com/questions/10869276/how-to-deal-with-cyclic-dependencies-in-node-js?utm_source=chatgpt.com "How to deal with cyclic dependencies in Node.js - Stack Overflow"
[4]: https://stackoverflow.com/questions/64677971/when-does-circular-dependency-become-a-problem?utm_source=chatgpt.com "When does circular dependency become a problem? - Stack Overflow"
[5]: https://github.com/sverweij/dependency-cruiser/issues/695?utm_source=chatgpt.com "Issue: no-circular does not follow dependencyTypesNot: ['type-only ..."
[6]: https://medium.com/visual-development/how-to-fix-nasty-circular-dependency-issues-once-and-for-all-in-javascript-typescript-a04c987cf0de?utm_source=chatgpt.com "How to fix nasty circular dependency issues once and for all in ..."
[7]: https://dev.to/greenroach/detecting-circular-dependencies-in-a-reacttypescript-app-using-madge-229?utm_source=chatgpt.com "Detecting Circular Dependencies in a React/TypeScript App using ..."
[8]: https://www.npmjs.com/package/madge?utm_source=chatgpt.com "madge - NPM"
[9]: https://github.com/sverweij/dependency-cruiser/blob/main/doc/rules-reference.md?utm_source=chatgpt.com "dependency-cruiser/doc/rules-reference.md at main - GitHub"
[10]: https://www.npmjs.com/package/dependency-cruiser?utm_source=chatgpt.com "dependency-cruiser - NPM"
[11]: https://stackoverflow.com/questions/73668598/best-way-to-resolve-circular-dependencies-in-typescript?utm_source=chatgpt.com "Best way to resolve circular dependencies in TypeScript"
[12]: https://stackoverflow.com/questions/71413477/how-to-resolve-circular-dependencies-in-typescript?utm_source=chatgpt.com "How to resolve circular dependencies in typescript - Stack Overflow"
[13]: https://www.reddit.com/r/typescript/comments/15fr1rr/can_i_use_barrel_export_to_handle_circular/?utm_source=chatgpt.com "Can I use barrel export to handle circular dependency? - Reddit"
