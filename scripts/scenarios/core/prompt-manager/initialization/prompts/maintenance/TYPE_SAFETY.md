## 🚀 Executive Summary

TypeScript’s power comes from its type system; every bypass (`any`, non-null `!`, lax flags) chips away at that safety. **Your goal is to harden the whole repo**: enable strict compiler options, adopt modern language features, ban unsafe patterns, and track progress with `AI_CHECK` comments. The payoff is fewer runtime bugs and fearless refactors. ([lucid.co][1])

---

## 1 Purpose & Scope

* **Task ID:** `TYPE_SAFETY`
* **Goal:** Improve overall type correctness and expressiveness.
* **Covers:** compiler flags, ESLint rules, unsafe casts, exhaustive checks, generics, runtime validation.
* **Excludes:** API-shape mismatches between docs and code (see `CODE_TO_DOCS` / `DOCS_TO_CODE`).

---

## 2 Hard Rules

| Rule                                                  | Why                                                                               |
| ----------------------------------------------------- | --------------------------------------------------------------------------------- |
| Never run Git commands                                | Human reviews and commits.                                                        |
| Don’t add packages without approval                   | Avoid stealth prod drift.                                                         |
| Keep `.js` extensions in TS imports                   | Required for Node ESM.                                                            |
| Prefer `unknown` over `any`; prove intent with guards | Prevents silent runtime errors. ([stackoverflow.com][2], [dmitripavlutin.com][3]) |

---

## 3 Your Type-Safety Toolbox

### 3.1 Strict Compiler Flags

| Flag                         | What It Catches                  | Docs                      |
| ---------------------------- | -------------------------------- | ------------------------- |
| `strict`                     | Enables all strict rules         | ([dev.to][4])             |
| `strictNullChecks`           | Null / undefined mistakes        | ([dev.to][4])             |
| `noUncheckedIndexedAccess`   | Un-guarded index lookups         | ([typescriptlang.org][5]) |
| `exactOptionalPropertyTypes` | `?` truly means “may be missing” | ([github.com][6])         |

Add or flip these flags, run `pnpm tsc --noEmit`, and fix surfaced errors.

### 3.2 Typed ESLint Rules

Enable `@typescript-eslint/no-unsafe-*`, `consistent-type-imports`, and `no-floating-promises` to catch mis-uses the compiler misses. ([typescript-eslint.io][7])

### 3.3 Modern Language Features

* **`unknown` + type guards**—restricts unsafe operations. ([stackoverflow.com][2], [dmitripavlutin.com][3])
* **`satisfies` operator** for literal objects (TS 4.9). ([typescriptlang.org][8])
* **Exhaustive `switch` with `never`** to future-proof enums/union types. ([stackoverflow.com][9], [meticulous.ai][10])
* **Generics best-practices**—avoid over-wide generics, prefer constraints. ([claritydev.net][11])

---

## 4 Common Risk Patterns & Fixes

| Risky Code                  | Why It’s Bad         | Safer Replacement                   |
| --------------------------- | -------------------- | ----------------------------------- |
| `as any`, double-casts      | Silences checks      | `unknown` + guard                   |
| Non-null `!`                | Masks undefined bugs | Null checks / optional chaining     |
| Untyped JSON / API payloads | Runtime shape drift  | Runtime validators (`zod`, `io-ts`) |
| Implicit `any` in generics  | Loses intent         | Proper generic constraints          |
| Non-exhaustive unions       | Future cases ignored | `assertNever()` pattern             |

---

## 5 Step-by-Step Workflow

1. **Turn on stricter flags** one package at a time.

2. **Run ESLint typed rules**; fix highest-impact warnings first.

3. **Search & destroy obvious escapes**

   ```bash
   rg "as any|as unknown as|!\\." --glob '*.ts*'
   ```

4. **Add type guards / generics / operator fixes** per §3.3.

5. **Verify**: `pnpm tsc --noEmit && pnpm lint && pnpm test`.

6. **Update AI\_CHECK** at top of each edited file:

   ```ts
   // AI_CHECK: TYPE_SAFETY=<incrementedCount> | LAST: 2025-06-16
   ```

7. **Stdout summary**: `file → strictNullChecks fixed, 2 any→unknown`.

---

## 6 Query Cookbook

```bash
# Files never reviewed for TYPE_SAFETY
rg -L "AI_CHECK:.*TYPE_SAFETY" packages -t ts -t tsx

# Oldest reviews first
rg "AI_CHECK:.*TYPE_SAFETY=.*\\| LAST:" -n | sort -t':' -k3,3
```

---

## 7 Fast Checklist

| ✔︎                                        | Item |
| ----------------------------------------- | ---- |
| Strict flags enabled (or noted why not).  |      |
| ESLint `no-unsafe-*` and friends pass.    |      |
| Zero `any` unless explicitly justified.   |      |
| Exhaustive union checks in place.         |      |
| Guards or validators for external data.   |      |
| Tests & type-check green.                 |      |
| `AI_CHECK` updated only for TYPE\_SAFETY. |      |

---

## 8 Reference Resources

* Dev.to – Five essential strict flags. ([dev.to][4])
* StackOverflow – `unknown` vs `any`. ([stackoverflow.com][2])
* TypeScript 4.9 release notes (`satisfies`). ([typescriptlang.org][8])
* Exhaustive switch pattern (SO). ([stackoverflow.com][9])
* typescript-eslint typed linting docs. ([typescript-eslint.io][7])
* `noUncheckedIndexedAccess` docs. ([typescriptlang.org][5])
* `exactOptionalPropertyTypes` issue discussion. ([github.com][6])
* Generics best practices guide. ([claritydev.net][11])
* Lucid blog – Why strict flags matter. ([lucid.co][1])
* Dmitri Pavlutin – deep dive on `unknown`. ([dmitripavlutin.com][3])
* Reddit – `satisfies` operator insights. ([reddit.com][12])
* Meticulous – safer exhaustive switches. ([meticulous.ai][10])

---

### Remember

Tight types today = fearless refactors tomorrow. Turn the compiler into your strongest test suite—remove escape hatches, lean on modern TS features, and prove every assumption.

[1]: https://lucid.co/techblog/2018/06/20/how-to-actually-improve-type-safety-with-the-typescript-strict-flags?utm_source=chatgpt.com "How to Actually Improve Type Safety with the TypeScript Strict Flags"
[2]: https://stackoverflow.com/questions/51439843/unknown-vs-any?utm_source=chatgpt.com "'unknown' vs. 'any' - typescript - Stack Overflow"
[3]: https://dmitripavlutin.com/typescript-unknown-vs-any/?utm_source=chatgpt.com "unknown vs any in TypeScript - Dmitri Pavlutin"
[4]: https://dev.to/geraldhamiltonwicks/5-essential-flags-to-enable-on-your-typescript-code-271k?utm_source=chatgpt.com "5 Essential Flags to Enable on Your TypeScript Code"
[5]: https://www.typescriptlang.org/tsconfig/noUncheckedIndexedAccess.html?utm_source=chatgpt.com "TSConfig Option: noUncheckedIndexedAccess - TypeScript"
[6]: https://github.com/radix-ui/primitives/issues/3535?utm_source=chatgpt.com "Support `exactOptionalPropertyTypes` compiler flag in TypeScript"
[7]: https://typescript-eslint.io/getting-started/typed-linting?utm_source=chatgpt.com "Linting with Type Information | typescript-eslint"
[8]: https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-9.html?utm_source=chatgpt.com "Documentation - TypeScript 4.9"
[9]: https://stackoverflow.com/questions/39419170/how-do-i-check-that-a-switch-block-is-exhaustive-in-typescript?utm_source=chatgpt.com "How do I check that a switch block is exhaustive in TypeScript?"
[10]: https://www.meticulous.ai/blog/safer-exhaustive-switch-statements-in-typescript?utm_source=chatgpt.com "Safer Exhaustive Switch Statements in TypeScript - Meticulous"
[11]: https://claritydev.net/blog/typescript-generics-guide?utm_source=chatgpt.com "Using Generics In TypeScript: A Practical Guide | ClarityDev blog"
[12]: https://www.reddit.com/r/webdev/comments/zrt1rb/the_satisfies_operator_in_typescript_49_is_a_game/?utm_source=chatgpt.com "The `satisfies` operator in TypeScript 4.9 is a game changer - Reddit"
