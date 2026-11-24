---
name: "ui-i18n-adoption"
description: "Adopt multi-language support in a scenario UI: wire i18next through the typed strings.ts/selectors.ts registries, default tests to cimode, layer accessibility and locale-parity contracts on top, so adding new locales is a config change and tests don't drift when copy moves"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","react","ux"]
  tags: ["skill"]
  icon: "languages"
  status: "active"
  defaultScope: "work-scope"
  revision: 2
  createdAt: "2026-05-02T03:51:50Z"
  updatedAt: "2026-05-02T15:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "vrooli"]
  origin:
    kind: "authored"
---
## Steer focus: UI Internationalization Adoption

**Scope mode: `architecture`.** Hold to that row of `work-scope` for this session.

Adopt **multi-language support** for `scenarios/{{TARGET}}/ui/` by wiring `i18next` through the existing `strings.ts` registry, with RTL/locale-formatting hooks pre-wired so adding the next locale is a config change rather than a code rewrite.

Your goal is to ship a second (or Nth) locale without breaking tests, without scattering inline strings, and without committing to translation infrastructure beyond what the demand justifies.

Required reading:
- `prompt-manager skill read ui-health`
- `prompt-manager skill read visited-tracker-tools`

---

### **0. Why This Skill Exists**

UI scenarios drift toward inline JSX strings. When the first localization request arrives — Japanese for an enterprise customer, Spanish for a market expansion — the team either ships English-only or burns weeks migrating component-by-component.

This skill keeps both outcomes off the table by treating i18n as a **layered adoption**:

| Phase | Where it lives | Cost |
|-------|----------------|------|
| **1 — Full template substrate** | `path:templates/scenarios/react-vite/` | Already paid (every new scenario inherits) |
| **2 — Per-scenario adaptation + extra locales** | This skill | A day or two per scenario |
| **3 — Shared `@vrooli/i18n` package** | Future, after 3+ scenarios converge | Out of scope |

**What Phase 1 already gives every new scenario:**
- Typed `strings.ts` registry (key-path tree derived from `en.json`).
- Typed `selectors.ts` registry (`createSelectorRegistry` with literal + dynamic trees).
- i18next + react-i18next wired up with English + Japanese locale catalogs.
- `<html lang>`/`<html dir>` mirroring on `languageChanged`, localStorage persistence, browser-language auto-detect.
- Locale-aware `formatDate`/`formatNumber`/`formatCurrency`/`formatRelativeTime`/`formatList` wrappers (`i18n/format.ts`).
- Two `no-restricted-syntax` ESLint rules: ban JSX literals in production, ban string/template literals as `*ByText` query args in tests.
- Test setup defaulting i18next to **cimode** so `t("app.title")` returns `"app.title"` — tests assert via `strings.*` and survive copy changes.
- Three-layer test pattern modeled in `App.test.tsx` (selectors + strings + smoke).
- Locale parity contract test (`locales.test.ts`).
- axe-core accessibility test (`App.a11y.test.tsx`).

If a scenario predates this template, fix that first — Phase 2 only makes sense on top of the full substrate. Reference: `path:templates/scenarios/react-vite/ui/`.

---

### **1. Scope Boundaries**

**In scope:**
- Adding `i18next` + `react-i18next` to a scenario's UI
- Migrating callsites from `{strings.x.y}` to `{t(strings.x.y)}`
- Locale JSON authoring (en + Nth)
- HTML `lang`/`dir` mirroring on language change
- localStorage persistence + browser-language auto-detection on first visit
- Locale-aware date/number formatting via native `Intl.*` APIs
- Language switcher component
- Vitest coverage of locale switching

**Out of scope:**
- Server-side i18n — API responses with user-facing text should ship error *codes* the client maps to localized strings; raw English in API responses is a separate refactor
- Translation Memory or TMS integrations
- ICU plurals / gender / context-sensitive forms — only if the target locale actually needs them; document the deferral
- Full RTL CSS audit — separate effort if/when the first RTL locale lands
- Performing professional translations yourself — for shipped scenarios, get a native review

---

### **2. Adoption Decision Tree**

```
                  Does the scenario have a concrete locale demand?
                                    │
                  ┌─────────────────┴─────────────────┐
                YES                                   NO
                  │                                   │
                  ▼                                   ▼
   Run this skill (Phase 2 adoption)       Phase 1 only — registry +
                                           lint rule are enough.
                                           Don't preemptively adopt.
```

**Concrete demand:**
- Customer or market explicitly requested a specific language
- Compliance or contractual requirement (EU, JP enterprise, gov, etc.)
- Brand strategy treats language as a differentiator

**NOT concrete demand:**
- "It's good practice"
- "Maybe someday"
- "Big companies do it"

If demand is hypothetical, stop. The Phase 1 registry is sufficient and the migration path stays cheap.

---

### **3. Adoption Process**

Phases run in order; the reference implementation for each file is under `templates/scenarios/react-vite/ui/src/` (`i18n/index.ts`, `i18n/format.ts`, `i18n/locales/locales.test.ts`, `test-setup.ts`, `App.test.tsx`, `App.a11y.test.tsx`). Copy them rather than re-deriving.

**A. Verify the Phase 1 substrate.** `ui/src/consts/strings.ts` (typed key tree) and `selectors.ts` (`createSelectorRegistry`, literal and dynamic trees with `satisfies`) exist; `eslint.config.js` has both `no-restricted-syntax` rules (no JSX literals in production code, no string or template literal as the first argument to `*ByText` queries in tests); directional Tailwind utilities (`ml- mr- pl- pr- left-N right-N text-left text-right rounded-l rounded-r`) are migrated to logical ones (`rg` over `ui/src`). If any fails, halt and finish Phase 1 first: i18next on inlined strings yields a partial registry.

**B. Dependencies.** Add `i18next` and `react-i18next` (latest stable), plus `jsdom` as a dev dependency if `vite.config.ts` uses the jsdom environment; install with `cd scenarios/{{TARGET}}/ui && corepack pnpm install --ignore-workspace`. Do not add `i18next-browser-languagedetector` (five lines of localStorage), `i18next-http-backend` (bundle the JSON) or `i18next-icu` (only if pluralization demands it).

**C. Locale files.** `ui/src/i18n/locales/en.json` takes the canonical copy from `strings.ts`; add `<code>.json` per target locale with identical nested key shapes. Machine translation is acceptable for internal or non-shipped work (state the source in the PR); shipped locales need native review with the reviewer named; leave brand strings untranslated unless the brand owner approves.

**D. Init module** `ui/src/i18n/index.ts`: `SUPPORTED_LOCALES` const tuple; `LOCALE_CONFIG` with `nativeLabel` and `dir`; `detectInitialLocale()` (localStorage `vrooli.locale`, then `navigator.language` primary subtag if supported, then `en`); `applyDocumentLocale(lng)` mirroring `<html lang>` and `<html dir>`; i18next init with bundled resources, `interpolation: { escapeValue: false }`, `returnNull: false`; a `languageChanged` listener that persists and re-applies; re-export `useTranslation`. Import it for side effects in `main.tsx` before the first render. Do not wrap `<App>` in a custom provider or initialize inside a component.

**E-F. Registry and callsites.** Derive `strings.ts` from `en.json` (each leaf is its dotted path, so key changes fail compilation) and drop the old `format()` helper. Wrap every `{strings.x.y}` as `{t(strings.x.y)}`; interpolation uses `t(strings.x.y, { count })` with `{{count}}` placeholders; non-component code uses `i18n.t(...)` from the singleton.

**G. Switcher.** List `SUPPORTED_LOCALES` by `nativeLabel`, mark the active one `aria-pressed="true"`, call the module's `setLocale(lng)`, and place it where the IA expects (not behind three taps). Ship it even if the default locale matches the browser: locale wiring is hard to test without it, and returning users on another machine are stuck on auto-detect.

**H. Test setup** (`test-setup.ts`, referenced from `vite.config.ts` `test.setupFiles`): import `@testing-library/jest-dom/vitest`; `beforeEach` sets `i18n.changeLanguage("cimode")` (returns the key, so tests survive any wording change) and clears localStorage; stub `HTMLCanvasElement.prototype.getContext` with `vi.spyOn(...).mockReturnValue(null)` so axe-core is quiet. Setup `beforeEach` runs first, so per-file `await setLocale("en")` wins.

**I. Three-layer tests.** Find elements with `selectors.x.y` test ids (`data-testid`), assert which key renders with `strings.x.y` under cimode, and validate the real pipeline with `en.x.y` after `setLocale("en")` in an explicit real-locale describe block (catalogs reach the DOM, the switcher works, `<html lang>` updates, storage persists). Regex matchers stay allowed; string literals to `*ByText` do not.

**J. Parity contract test** (`locales.test.ts`): flatten every catalog to dotted keys, strip CLDR plural suffixes (`_zero|one|two|few|many|other`) before comparing, assert identical logical keys across locales, no empty or whitespace values, and string-only values.

**K. Formatters** (`i18n/format.ts`): `formatDate`, `formatNumber`, `formatCurrency`, `formatRelativeTime`, `formatList`, each reading `i18n.language` at call time and stripping pseudo-locales (`cimode`, `dev`) before `Intl.*` so tests avoid `RangeError`; components never call `Intl.*` directly. `tsconfig.json` `lib` needs `ES2021.Intl` for `Intl.ListFormat`. Unit-test that de-DE and en-US differ (`formatNumber(12345.6, undefined, "de-DE")` is `"12.345,6"`).

**L. Plurals (only when the copy needs them).** Base key is the CLDR `other` form and `<key>_one` the singular; locales without plural forms (ja, zh, ko) need only the base key; callers always pass the base key with `{ count }`. cimode bypasses plural logic and returns the base key, so plural tests must `await setLocale("en")`.

**M. Accessibility (recommended).** Use `axe-core` directly (`await axe.run(container)`, assert `results.violations` equals `[]`; `jest-axe`'s matcher does not fit vitest) in `<Component>.a11y.test.tsx`, rendered in real English, one run per high-traffic surface.

---

### **4. Convergence Patterns**

`en.json` (canonical, drives types) feeds `strings.ts` (key-path tree), components call `t(strings.feature.key)`, and i18next resolves `resources[currentLocale]`. Locale resolution on first visit: stored `vrooli.locale` if supported, else the browser's primary subtag if supported, else `en`. Placement: text direction and switcher label belong in `LOCALE_CONFIG`, translated strings in JSON, date, number and currency formatting in `format.ts`. The lint rule on `*ByText` literals is what keeps the three test layers from collapsing into brittle literal matches.

---

### **5. Common Pitfalls**

- A bare `{strings.app.title}` in JSX renders the key path and the lint rule will not catch it; review for `t()` wrappers. Inlining `t("app.title")` instead of `t(strings.app.title)` forfeits compile-time key checking.
- Server-side English leaks through: API responses with user-facing text should ship error codes the client maps to localized strings (flag it; out of scope).
- Hand-rolled date or number formatting: use `i18n/format.ts`.
- CJK text has no inter-word spaces and can be ~30% longer in pixels: distrust width-dependent layout assertions and use `min-width: max-content` on labels.
- shadcn primitives that use `ml-` or `pl-` directly: audit on the first RTL locale.
- i18next init race: with no backend init is synchronous; if you add one, gate render on the init promise.
- `selectors.x is possibly undefined`: annotate literal trees with `satisfies LiteralSelectorTree`, never `: LiteralSelectorTree`, which widens to the index signature under `noUncheckedIndexedAccess`.
- cimode returns keys: asserting translated copy (`getByText("Refresh")`) fails there, and plural-form assertions need `setLocale("en")`. Do not disable the test-matcher lint rule for one test; move the assertion to a test id or a real-locale block.
- Per-component locale state, translating without a `LOCALE_CONFIG` entry, splicing values into registry strings with template literals (use `{{var}}` placeholders) and adopting i18n without concrete demand (Section 2) are all anti-patterns.

---

### **6. Output Expectations**

Done means `corepack pnpm install`, `type-check`, `lint` (zero errors and warnings), `test` and `build` pass; the test suite includes the parity contract, cimode rendering tests, real-locale switching tests, formatter unit tests and a11y tests; `<html lang>` and `<html dir>` follow the active locale and the switcher choice survives reload; both `no-restricted-syntax` rules still pass, proven by a negative check (add `screen.getByText("hardcoded")` to a test file and confirm the rule's error). Otherwise adoption is incomplete.

---

### **7. Troubleshooting & Edge Cases**

| Symptom | Fix |
|---|---|
| `Cannot find package 'jsdom'` | `corepack pnpm add -D jsdom`. |
| Renders show the key path | i18next initialized after render or the resource is unregistered: confirm `import "./i18n"` in `main.tsx` and that the test imports it, or `await setLocale("en")` in `beforeEach`. |
| Switcher does nothing | `changeLanguage(lng)` is a no-op when `lng` equals the current language (no `languageChanged` event); compare `i18n.language` first. |
| `Cannot find module './locales/en.json'` | `tsconfig.json` needs `resolveJsonModule: true`. |
| `t()` return type errors (`string | $T`) | Set `returnNull: false` in init (react-i18next v15+). |
| `<html dir>` not updating | Guard `applyDocumentLocale` with `typeof document === "undefined"` for SSR and tests. |
| `Property 'ListFormat' does not exist on type 'typeof Intl'` | Add `"ES2021.Intl"` to `tsconfig.json` `lib`. |
| `Not implemented: HTMLCanvasElement.prototype.getContext` | axe probes canvas; stub it in `test-setup.ts` (Phase H). Noise only, but it hides real failures. |
| `getByText(strings.x.y)` finds nothing in cimode | The component renders `{strings.x.y}` instead of `{t(strings.x.y)}`. |
