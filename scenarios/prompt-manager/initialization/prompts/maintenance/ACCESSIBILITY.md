## 🚀 Quick-Read Summary

Your goal is to align every user-facing surface with WCAG 2.2 AA, relying first on semantic HTML and ARIA best-practice, then on automated checks (eslint-plugin-jsx-a11y, axe-core, Cypress-axe) and, finally, on manual screen-reader and keyboard walkthroughs. You’ll prioritise files never reviewed for `ACCESSIBILITY`, bump the task counter in `AI_CHECK`, and leave the working tree dirty for human review. ([w3.org][1], [developer.mozilla.org][2], [testing-library.com][3], [w3.org][4], [github.com][5], [deque.com][6], [web.dev][7], [webaim.org][8], [deque.com][9], [docs.cypress.io][10])

---

## 1 Purpose & Scope

| Field            | Value                                                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| **Task ID**      | `ACCESSIBILITY`                                                                                                                |
| **Objective**    | Remove barriers for keyboard, screen-reader, low-vision, and cognitive users; meet WCAG 2.2 AA; track progress via `AI_CHECK`. |
| **Out-of-Scope** | Marketing PDFs, external CMS pages (handled separately).                                                                       |

---

## 2 Hard Rules

| Rule                                                                         | Why                      |
| ---------------------------------------------------------------------------- | ------------------------ |
| Don’t run Git commands                                                       | Human review/commit.     |
| Don’t add deps without approval                                              | Prevent bloated bundles. |
| Keep `.js` extensions in TS imports                                          | Node ESM.                |
| Never silence a11y lint errors with `// eslint-disable`; fix the root cause. |                          |

---

## 3 What to Fix

### 3.1 Must-Haves (Fail WCAG 2.2 AA)

| Check                                                        | Quick Test                                                 | WCAG Link |
| ------------------------------------------------------------ | ---------------------------------------------------------- | --------- |
| **Keyboard focus order**                                     | `Tab` cycles visually & logically ([web.dev][7])           | 2.4.3     |
| **Semantic landmarks** (`<header>`, `<nav>`, `<main>`, etc.) | DevTools → Accessibility tree ([developer.mozilla.org][2]) | 1.3.1     |
| **ARIA labels/roles only when needed**                       | Prefer native elements ([developer.mozilla.org][11])       | 4.1.2     |
| **Contrast ≥ 4.5:1**                                         | WebAIM checker ([webaim.org][8])                           | 1.4.3     |
| **Live-region announcements** for async errors/status        | `aria-live="polite"` ([w3.org][4])                         | 4.1.3     |
| **No autoplaying, blinking, or distracting elements**        | eslint rule `no-distracting-elements` ([github.com][12])   | 2.2.2     |

### 3.2 Nice-to-Haves

* Page-level skip links.
* Form field `aria-describedby` for helper/error text.
* Responsive reflow without horizontal scroll.

---

## 4 Detection & Tooling

| Layer                       | Command / Approach                                                                                                                  | Citations |
| --------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | --------- |
| **Static JSX lint**         | `pnpm lint -- --ext .tsx --rule 'jsx-a11y/*:error'` ([github.com][5])                                                               |           |
| **Runtime axe-core**        | `pnpm exec axe ./dist` or `@axe-core/react` in tests ([deque.com][6], [deque.com][9])                                               |           |
| **Cypress-axe** end-to-end  | `cy.injectAxe(); cy.checkA11y();` ([docs.cypress.io][10], [docs.cypress.io][13])                                                    |           |
| **Testing Library queries** | Replace `getByTestId` with `getByRole/LabelText` ([testing-library.com][3])                                                         |           |
| **Manual screen-reader**    | NVDA → `NVDA + n` menu & shortcut quick-ref ([download.nvaccess.org][14], [download.nvaccess.org][15], [download.nvaccess.org][16]) |           |

Prioritise files **without `AI_CHECK`** or with the oldest `LAST` date.

---

## 5 Fix Patterns

### 5.1 Use Semantic HTML First

Replace generic `<div role="button">` with `<button>` to inherit keyboard and ARIA behaviour automatically. ([developer.mozilla.org][2])

### 5.2 Focus Management in React

After opening modals, call `dialogRef.current.focus()` and trap focus inside; return focus to the triggering element on close. ([web.dev][7])

### 5.3 Color Contrast

Adopt design-token pairs that pass the WebAIM contrast checker; document token ratios in the theme system. ([webaim.org][8])

### 5.4 Live Regions for Async Status

Use `<div role="status" aria-live="polite">` to announce loading or save confirmations. ([w3.org][4])

### 5.5 Automated Tests

Add `cy.checkA11y({ includedImpacts: ['critical'] })` after each major route render; break the build on violations. ([docs.cypress.io][17], [docs.cypress.io][18])

---

## 6 Step-by-Step Workflow

1. **Scan for violations**

   ```bash
   pnpm lint            # static JSX a11y
   pnpm test -- --a11y  # axe integration tests
   cy:run --browser chrome --e2e a11y  # Cypress axe
   rg -L "AI_CHECK:.*ACCESSIBILITY" packages -t tsx
   ```
2. **Fix issues** using patterns in §5.
3. **Verify** with `pnpm test && pnpm lint && cy:run`.
4. **Update `AI_CHECK`** at top of each touched file:

   ```ts
   // AI_CHECK: ACCESSIBILITY=<incrementedCount> | LAST: 2025-06-17
   ```
5. **Write a one-line summary to stdout** (`ui/Button.tsx → fixed missing aria-label`).
6. Leave the working directory dirty for human review.

---

## 7 Query Cookbook

```bash
# Components missing any a11y check
rg -L "AI_CHECK:.*ACCESSIBILITY" packages/ui -t tsx

# Oldest a11y reviews
rg "AI_CHECK:.*ACCESSIBILITY=.*\\| LAST:" -n | sort -t':' -k3,3
```

---

## 8 Fast Checklist

| ✔︎                                                                              | Item |
| ------------------------------------------------------------------------------- | ---- |
| Keyboard nav works (Tab, Shift+Tab, Enter, Space, Esc).                         |      |
| Screen-reader reads page title, landmarks, and control labels in logical order. |      |
| Contrast ≥ 4.5:1 (normal text) or 3:1 (UI graphics).                            |      |
| No eslint a11y errors; axe critical violations = 0.                             |      |
| `AI_CHECK` comment updated only for `ACCESSIBILITY`.                            |      |

---

## 9 Reference Links

* WCAG 2.2 spec ([w3.org][1])
* WCAG “Understanding” docs ([w3.org][19])
* MDN Semantic HTML guide ([developer.mozilla.org][2])
* WAI ARIA practices ([developer.mozilla.org][11])
* ESLint jsx-a11y plugin docs ([github.com][5])
* Axe-core library ([deque.com][9])
* Cypress a11y testing intro ([docs.cypress.io][10])
* Testing-Library ByRole docs ([testing-library.com][3])
* WebAIM contrast checker ([webaim.org][8])
* NVDA user guide (keyboard shortcuts) ([download.nvaccess.org][14])
* NVDA commands quick ref ([download.nvaccess.org][15])
* ARIA live region technique ARIA19 ([w3.org][4])
* Cypress axe violation details ([docs.cypress.io][20])
* Wired contrast-ratio tool article ([wired.com][21])
* eslint rule `no-distracting-elements` ([github.com][12])

---

### Remember

You create inclusive experiences by default: **use the platform (HTML) first, sprinkle ARIA only when necessary, automate checks, then verify with real assistive tech**.

[1]: https://www.w3.org/TR/WCAG22/?utm_source=chatgpt.com "Web Content Accessibility Guidelines (WCAG) 2.2 - W3C"
[2]: https://developer.mozilla.org/en-US/docs/Learn_web_development/Core/Accessibility/HTML?utm_source=chatgpt.com "HTML: A good basis for accessibility - Learn web development | MDN"
[3]: https://testing-library.com/docs/queries/byrole/?utm_source=chatgpt.com "ByRole | Testing Library"
[4]: https://www.w3.org/WAI/WCAG22/Techniques/aria/ARIA19?utm_source=chatgpt.com "ARIA19: Using ARIA role=alert or Live Regions to Identify Errors - W3C"
[5]: https://github.com/jsx-eslint/eslint-plugin-jsx-a11y?utm_source=chatgpt.com "jsx-eslint/eslint-plugin-jsx-a11y: Static AST checker for a11y ... - GitHub"
[6]: https://www.deque.com/axe/devtools/mobile-accessibility/react-native/?utm_source=chatgpt.com "React Native App Accessibility Testing & axe DevTools Mobile | Deque"
[7]: https://web.dev/learn/accessibility/focus?utm_source=chatgpt.com "Keyboard focus | web.dev"
[8]: https://webaim.org/resources/contrastchecker/?utm_source=chatgpt.com "Contrast Checker - WebAIM"
[9]: https://www.deque.com/axe/?utm_source=chatgpt.com "Accessibility Testing Tools & Software: Axe - Deque Systems"
[10]: https://docs.cypress.io/app/guides/accessibility-testing?utm_source=chatgpt.com "Accessibility Testing - Cypress Documentation"
[11]: https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA?utm_source=chatgpt.com "ARIA - Accessibility - MDN Web Docs - Mozilla"
[12]: https://github.com/jsx-eslint/eslint-plugin-jsx-a11y/blob/main/docs/rules/no-distracting-elements.md?utm_source=chatgpt.com "eslint-plugin-jsx-a11y/docs/rules/no-distracting-elements.md at main"
[13]: https://docs.cypress.io/accessibility/get-started/introduction?utm_source=chatgpt.com "Accessibility Testing | Cypress Documentation"
[14]: https://download.nvaccess.org/releases/2024.4.2/documentation/userGuide.html?utm_source=chatgpt.com "NVDA 2024.4.2 User Guide"
[15]: https://download.nvaccess.org/releases/2021.2/documentation/keyCommands.html?utm_source=chatgpt.com "NVDA 2021.2 Commands Quick Reference"
[16]: https://download.nvaccess.org/releases/2022.3/documentation/is/userGuide.html?utm_source=chatgpt.com "NVDA 2022.3 User Guide - NV Access"
[17]: https://docs.cypress.io/accessibility/core-concepts/how-it-works?utm_source=chatgpt.com "Cypress Accessibility | How it works"
[18]: https://docs.cypress.io/accessibility/guides/accessibility-automation?utm_source=chatgpt.com "Accessibility automation principles - Cypress Documentation"
[19]: https://www.w3.org/WAI/WCAG22/Understanding/?utm_source=chatgpt.com "Understanding WCAG 2.2 | WAI - W3C"
[20]: https://docs.cypress.io/accessibility/core-concepts/inspecting-violation-details?utm_source=chatgpt.com "Inspecting accessibility violation details | Cypress Documentation"
[21]: https://www.wired.com/2012/10/create-more-accessible-color-schemes-with-contrast-ratio?utm_source=chatgpt.com "Create More Accessible Color Schemes With 'Contrast Ratio'"
