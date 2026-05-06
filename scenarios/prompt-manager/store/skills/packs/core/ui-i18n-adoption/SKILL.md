## Steer focus: UI Internationalization Adoption

Adopt **multi-language support** for `scenarios/{{TARGET}}/ui/` by wiring `i18next` through the existing `strings.ts` registry, with RTL/locale-formatting hooks pre-wired so adding the next locale is a config change rather than a code rewrite.

Your goal is to ship a second (or Nth) locale without breaking tests, without scattering inline strings, and without committing to translation infrastructure beyond what the demand justifies.

Required reading:
- `prompt-manager skill read react-coherence react-stability vrooli-ui-interop`
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

Each phase has explicit entry/exit criteria. Do not skip phases.

#### **Phase A — Verify Phase 1 Substrate**

**Entry:** Adoption decision is YES.

**Actions:**
1. Confirm `scenarios/{{TARGET}}/ui/src/consts/strings.ts` exists and follows the typed-key-tree shape.
2. Confirm `scenarios/{{TARGET}}/ui/src/consts/selectors.ts` exists with the `createSelectorRegistry` shape (literal + dynamic trees, `satisfies` annotations).
3. Confirm `eslint.config.js` has both `no-restricted-syntax` rules: one rejecting JSX literals (production code), one rejecting string/template literals as the first arg to `*ByText` queries (test code).
4. Grep for directional Tailwind utilities and verify they've been migrated to logical equivalents:

```bash
rg -n '\b(ml-|mr-|pl-|pr-|left-[0-9]|right-[0-9]|text-left|text-right|rounded-l|rounded-r)' scenarios/{{TARGET}}/ui/src
```

**Exit:** All four pass. If any fail, halt and run the Phase 1 prep first — bolting i18next onto a codebase still inlining strings creates a partial registry and loses the type-safety wins.

#### **Phase B — Install Dependencies**

**Actions:**
1. Add `i18next` and `react-i18next` (both latest stable) to `dependencies` in `scenarios/{{TARGET}}/ui/package.json`.
2. If `vite.config.ts` declares `test: { environment: 'jsdom' }` but `jsdom` isn't installed, add `jsdom` to `devDependencies`.
3. Run `cd scenarios/{{TARGET}}/ui && corepack pnpm install --ignore-workspace`.

**Do NOT add:**

| Package | Why not |
|---------|---------|
| `i18next-browser-languagedetector` | 5 lines of localStorage handles it; the package is overkill |
| `i18next-http-backend` | Bundle the JSON. Lazy-loading locales matters at TMS scale, not at scenario scale |
| `i18next-icu` | Only if pluralization is actually needed by the target locale |

#### **Phase C — Locale Files**

**Actions:**
1. Create `scenarios/{{TARGET}}/ui/src/i18n/locales/en.json` with the canonical English copy moved from `strings.ts`.
2. Create `scenarios/{{TARGET}}/ui/src/i18n/locales/<code>.json` per target locale.
3. Use **identical nested key shapes** across locales — diverging shapes break the type-safe key tree.

**Translation strategy:**

| Audience | Bar |
|----------|-----|
| Internal demo / non-shipped | Machine translation acceptable as a starting point. State the source in the PR description. |
| External / shipped to customers | Native review mandatory before release. Document the reviewer in the PR description. |
| Brand strings (product name, tagline) | Leave untranslated unless brand team approves a translated form. |

#### **Phase D — i18n Init Module**

**Actions:**
1. Create `scenarios/{{TARGET}}/ui/src/i18n/index.ts` that:
   - Exports `SUPPORTED_LOCALES` as a `const` tuple.
   - Defines a `LOCALE_CONFIG` map keyed by locale code with at minimum `nativeLabel` (string) and `dir` (`"ltr" | "rtl"`).
   - Implements `detectInitialLocale()` resolving in this order: localStorage → `navigator.language` primary subtag → `"en"`.
   - Implements `applyDocumentLocale(lng)` that mirrors the active locale to `<html lang>` and `<html dir>`.
   - Initializes i18next with bundled resources, `interpolation: { escapeValue: false }`, and `returnNull: false`.
   - Subscribes to `i18n.on("languageChanged", …)` to persist + re-apply DOM attrs.
   - Re-exports `useTranslation` so components import locale APIs from one place.
2. Import the module for side-effects in `main.tsx` before the first render: `import "./i18n";`.

The reference implementation is `templates/scenarios/react-vite/ui/src/i18n/index.ts`.

**Do NOT:**
- Wrap `<App>` in a custom provider — `react-i18next` works through the singleton + hook.
- Initialize i18next inside a component (re-initializes on remount).

#### **Phase E — Refactor `strings.ts`**

**Actions:**
1. Replace literal string values with a key-path tree derived from `en.json`. Each leaf becomes the dotted path string (`strings.app.title === "app.title"`).
2. Type-derive from `import en from "../i18n/locales/en.json"` so adding/removing keys triggers compile errors at every callsite when the registry regenerates.
3. Drop any `format()` helper from the Phase 1 shape — i18next's `t(key, params)` with `{{var}}` JSON placeholders replaces it.

#### **Phase F — Migrate Callsites**

**Actions:**
1. Find every `{strings.x.y}` reference and wrap in `t()`: `{t(strings.x.y)}`.
2. For interpolation, change `format(strings.x.y, { count })` to `t(strings.x.y, { count })` and update the JSON to use `{{count}}` placeholders.
3. For non-component code (utilities, store actions), use the singleton: `import { i18n } from "../i18n"; i18n.t(strings.x.y);`.

The `no-restricted-syntax` ESLint rule still catches inline literals; its enforcement target is unchanged.

#### **Phase G — Language Switcher**

**Actions:**
1. Add a UI element listing `SUPPORTED_LOCALES` with their `nativeLabel`s.
2. Mark the active one with `aria-pressed="true"`.
3. Click handler calls `setLocale(lng)` exported from the i18n module — the `languageChanged` listener handles persistence + DOM updates.
4. Place the switcher where the scenario's IA expects it (header, footer, settings). Don't bury it behind 3+ taps.

Even if the target locale matches the user's browser default, ship a visible switcher anyway:
- Tests of locale wiring become awkward without it.
- Returning users on a different machine are stuck on auto-detect.
- The team forgets the wiring exists, and inline strings creep back in.

#### **Phase H — Test Setup (cimode default + canvas stub)**

**Actions:**
1. `scenarios/{{TARGET}}/ui/src/test-setup.ts` should:
   - Import `@testing-library/jest-dom/vitest` (matchers).
   - Default `i18n.changeLanguage("cimode")` in a `beforeEach`. cimode is i18next's "return the key" pseudo-locale — `t("app.title")` returns `"app.title"`. Tests then assert via `strings.*` paths and survive any wording change in any locale.
   - Stub `HTMLCanvasElement.prototype.getContext` via `vi.spyOn(...).mockReturnValue(null)` so axe-core (Phase K) doesn't spam stderr in jsdom.
   - Clear localStorage in the same `beforeEach` so locale-persistence tests start from a clean slate.
2. Reference it from `vite.config.ts` via `test: { setupFiles: ['./src/test-setup.ts'] }`.

The reference implementation is `templates/scenarios/react-vite/ui/src/test-setup.ts`. Order matters: setup-file `beforeEach` runs before per-file `beforeEach`, so per-file overrides (e.g., `await setLocale("en")`) win when a test needs a real locale.

#### **Phase I — Three-Layer Test Pattern**

The whole point of the registry is that tests stop breaking when copy changes. Apply this layered pattern to every component test:

| Layer | Use for | Example |
|-------|---------|---------|
| **`selectors.x.y`** (test ID) | Finding elements | `screen.getByTestId(selectors.health.refreshButton)` |
| **`strings.x.y`** + cimode | Asserting *which key* renders | `screen.getByText(strings.app.title)` (returns `"app.title"` in cimode) |
| **`en.x.y`** + `setLocale("en")` | Validating the i18n pipeline end-to-end | `screen.getByText(en.app.title)` — only inside an explicit real-locale describe block |

**Actions:**
1. Add `data-testid={selectors.x.y}` attributes to renderable elements you want tests to find (titles, buttons, form fields, status surfaces).
2. Write component tests in two describe blocks:
   - **`describe("X rendering (cimode — copy-independent)")`** — runs in cimode (default). Asserts via `selectors.*` and `strings.*`. These tests don't break when copy changes.
   - **`describe("X locale switching (real locales — end-to-end)")`** — overrides to `setLocale("en")` in `beforeEach`. Validates that catalogs flow through to the DOM, that the switcher works, that `document.documentElement.lang` updates, that localStorage persists.
3. The new `no-restricted-syntax` ESLint rule (Phase A item 3) bans string and template literals as the first argument to `getByText` / `findByText` / `queryByText` (and their plural variants). It does NOT ban regex matchers (`/loading/i`) — explicit pattern matchers are still allowed.
4. Run `corepack pnpm test`.

The reference implementation is `templates/scenarios/react-vite/ui/src/App.test.tsx`.

#### **Phase J — Locale Parity Contract Test**

**Actions:**
1. Add `scenarios/{{TARGET}}/ui/src/i18n/locales/locales.test.ts` that:
   - Flattens every locale catalog to dotted keys.
   - Strips CLDR plural suffixes (`_zero|one|two|few|many|other`) before comparing — different locales need different plural variants, but each *logical* key must exist in every locale.
   - Asserts identical logical key shapes across all locales.
   - Asserts no value is empty/whitespace.
   - Asserts every value is a string (not array/object/null).
2. The reference implementation is `templates/scenarios/react-vite/ui/src/i18n/locales/locales.test.ts`.

This catches "added a key to en, forgot every other locale" drift before it ships, *and* "the translator left this entry blank."

---

#### **Phase K — Locale-Aware Formatters**

**Actions:**
1. Create `scenarios/{{TARGET}}/ui/src/i18n/format.ts` exporting:
   - `formatDate(value, options?, localeOverride?)` → `Intl.DateTimeFormat`
   - `formatNumber(value, options?, localeOverride?)` → `Intl.NumberFormat`
   - `formatCurrency(value, currency, options?, localeOverride?)` → `Intl.NumberFormat` with `style: "currency"`
   - `formatRelativeTime(value, unit, options?, localeOverride?)` → `Intl.RelativeTimeFormat`
   - `formatList(items, options?, localeOverride?)` → `Intl.ListFormat`
2. Each helper reads `i18n.language` at call time so output reflects the latest `setLocale(...)` choice.
3. Strip pseudo-locales (`cimode`, `dev`) before passing to `Intl.*` constructors so tests don't trip `RangeError`.
4. Verify `tsconfig.json` has `"ES2021.Intl"` in `lib` (required for `Intl.ListFormat`). If missing, add it next to `ES2020`.
5. Cover the helpers with unit tests asserting:
   - `formatNumber(12345.6, undefined, "de-DE")` → `"12.345,6"` (German thousands/decimal separators)
   - `formatDate(date, ..., "en-US")` and `formatDate(date, ..., "de-DE")` produce *different* shapes
   - `formatRelativeTime(-3, "day", undefined, "en-US")` → `"3 days ago"`
   - `formatList(["a", "b", "c"], undefined, "en-US")` → `"a, b, and c"`

The reference implementation is `templates/scenarios/react-vite/ui/src/i18n/format.{ts,test.ts}`.

**Why a wrapper instead of inline `new Intl.NumberFormat(i18n.language, …)`:**
- Centralizes pseudo-locale handling.
- Lets non-component code format without grabbing `useTranslation()`.
- Makes "locale-aware" a one-line code-review check (`format*(...)`) instead of a regex hunt for `Intl.`.

#### **Phase L — Pluralization (only if needed)**

**Actions:** Only add this if the target locale or English copy actually needs distinct singular/plural forms. Don't add it speculatively.

1. In `en.json`, add the base form under the bare key, and the singular under `<key>_one`:

```json
{
  "items": {
    "count": "{{count}} items",
    "count_one": "1 item"
  }
}
```

2. In locales without `_one` (Japanese, Chinese, Korean, …), the base key alone is enough — CLDR rules pick it for every count. The locale parity test (Phase J) strips suffixes and won't flag the asymmetry.

3. Call `t(strings.items.count, { count: N })` — i18next picks `count_one` when `N === 1`, `count` otherwise.

4. **Important:** cimode bypasses CLDR plural logic entirely. `t("items.count", { count: 1 })` returns `"items.count"` (the base key), not `"items.count_one"`. Tests that need to verify plural-form selection must opt back into a real locale via `await setLocale("en")` in `beforeEach`.

#### **Phase M — Accessibility Tests (recommended)**

**Actions:**
1. Add `axe-core` to `devDependencies` (no need for `jest-axe` — its matcher signature doesn't line up with vitest's `RawMatcherFn`, which forces casts).
2. Add `scenarios/{{TARGET}}/ui/src/<Component>.a11y.test.tsx` per high-traffic surface:

```tsx
import axe from "axe-core";
// render the component in REAL English (axe scans actual UI copy)
const { container } = renderApp();
await waitFor(() => { /* wait for async data to settle */ });
const results = await axe.run(container);
expect(results.violations).toEqual([]);
```

3. The canvas stub in `test-setup.ts` (Phase H) keeps axe's color-contrast probe quiet. Without it, every a11y test logs a `Not implemented: HTMLCanvasElement.prototype.getContext` warning — the test still passes but the noise hides real failures.

4. One axe run per high-traffic surface is enough; full-page scans across every route are expensive and produce duplicate findings.

The reference implementation is `templates/scenarios/react-vite/ui/src/App.a11y.test.tsx`.

---

### **4. Convergence Patterns**

#### Registry shape

```
i18n/locales/en.json  (canonical, drives types)
        │
        ▼
   strings.ts  (key-path tree, types derived from en.json)
        │
        ▼
t(strings.feature.key)   ←  components consume
        │
        ▼
i18next  →  resources[currentLocale][feature.key]  →  rendered string
```

#### Locale resolution on first visit

| State | Result |
|-------|--------|
| `localStorage["vrooli.locale"]` set & in `SUPPORTED_LOCALES` | Use stored value |
| localStorage empty, `navigator.language` primary subtag matches | Use browser language |
| Neither matches | Fall back to `"en"` |

#### Where does this belong?

| Question | Add to `LOCALE_CONFIG` | Inline in component | Add to JSON | Add to `format.ts` |
|----------|------------------------|---------------------|-------------|----|
| Locale-dependent (text direction)? | YES | NO | NO | NO |
| Native language label for switcher? | YES | NO | NO | NO |
| One-off translated string? | NO | NO | YES | NO |
| Date / number / currency formatting? | NO | NO | NO | YES |
| Hand-rolling `new Intl.DateTimeFormat(...)` in a component? | NO | **NO — call `formatDate(...)` from `format.ts`** | NO | YES |

#### Three-Layer Test Pattern (memorize this)

```
SETUP                    test-setup.ts → cimode default + canvas stub
                                  │
                                  ▼
LAYER 1 (find)           getByTestId(selectors.x.y)        ← copy- and structure-independent
LAYER 2 (assert key)     getByText(strings.x.y)            ← cimode echoes the key path
LAYER 3 (smoke)          setLocale("en") → getByText(en.x.y)  ← real-locale, opt-in describe block
```

The ESLint rule banning string literals to `*ByText` queries is what keeps Layer 2 and Layer 3 from collapsing back into brittle literal matches.

#### Plural keys shape

```
en.json:    "count": "{{count}} items"      ← CLDR "other" form (base key)
            "count_one": "1 item"            ← CLDR "one" form

ja.json:    "count": "{{count}} アイテム"   ← Japanese has only one plural form

caller:     t(strings.items.count, { count: N })   ← always the base key
```

Add `_one` (or `_zero`, `_two`, `_few`, `_many`) only when the locale actually distinguishes them. The locale parity test strips suffixes before comparing.

---

### **5. Common Pitfalls**

- **Bare registry refs in JSX.** `{strings.app.title}` renders the *key path* (`"app.title"`), not the translation. The lint rule won't catch it — the registry reference is "valid." Code-review for `t()` wrappers.
- **Server-side English leaking through.** API responses with user-facing text need to ship error codes the client maps to localized strings. Flag this during adoption even though it's out of scope.
- **Hand-rolled date formatting.** Use `formatDate` / `formatNumber` / `formatCurrency` from `i18n/format.ts`. Don't write per-locale formatters and don't call `Intl.*` constructors directly from components.
- **CJK whitespace assumptions.** Japanese has no inter-word spaces; Chinese punctuation has different widths. Be skeptical of layout assertions that depend on text width.
- **shadcn primitives with directional classes.** Some primitives use `ml-`/`pl-` directly. Audit on first RTL locale; the Phase 1 logical-utility convention only covered code under direct ownership.
- **i18next init race.** If `<App>` renders before init resolves, you'll see a flash of the key path. With no backend, init is synchronous — but if you add a backend later, gate render on the init promise.
- **`selectors.x is possibly undefined`.** The literal-tree variable was annotated `: LiteralSelectorTree`, which widens it to the index signature and makes every branch `T | undefined` under `noUncheckedIndexedAccess`. Use `satisfies LiteralSelectorTree` instead — same validation, narrower inferred type.
- **cimode + plurals.** cimode bypasses CLDR plural logic and returns the bare base key. Don't try to assert `health.refreshCount_one` in cimode — those tests must use `setLocale("en")`.
- **Asserting on translated copy in cimode tests.** If a test does `expect(getByText("Refresh"))` while cimode is active, the assertion fails because the rendered text is `"health.refresh"`. The ESLint rule for test text matchers prevents this — don't disable it.
- **`jest-axe` matcher type mismatch.** vitest's `expect.extend` expects `RawMatcherFn<MatcherState>`; jest-axe's `toHaveNoViolations` doesn't structurally match. Use `axe-core` directly via `await axe.run(container)` and assert `results.violations` is empty.

---

### **6. Output Expectations**

When this skill completes:

- `corepack pnpm install` succeeds.
- `corepack pnpm type-check` shows zero errors.
- `corepack pnpm lint` shows zero errors and zero warnings.
- `corepack pnpm test` passes:
  - Locale parity contract (`locales.test.ts`).
  - cimode-default rendering tests (`<Component>.test.tsx` "rendering" describe).
  - Real-locale switching tests (`<Component>.test.tsx` "locale switching" describe).
  - Formatter unit tests (`format.test.ts`).
  - Accessibility tests (`<Component>.a11y.test.tsx`).
- `corepack pnpm build` succeeds.
- `<html lang>` and `<html dir>` reflect the active locale at runtime.
- Switching languages via the switcher persists across reloads.
- Both `no-restricted-syntax` ESLint rules still pass:
  - No inline JSX literals introduced (production code).
  - No string/template literals as the first arg to `*ByText` queries (test code).
- A negative ESLint check (introduce `screen.getByText("hardcoded")` in a test file) reproduces the new rule's error message — proves the rule is wired to the right files.

If any of these fail, the adoption is incomplete. Do not declare done.

---

### **7. Troubleshooting & Edge Cases**

#### `Cannot find package 'jsdom'` when running tests
`vite.config.ts` declares `environment: 'jsdom'` but the package isn't installed. Add it: `corepack pnpm add -D jsdom`.

#### Test renders show the key path (`app.title`) instead of the translation
i18next initialized after the render, or the resource isn't registered. Verify `import "./i18n"` is present in `main.tsx` *and* that the test imports it transitively. For deterministic tests, call `await setLocale("en")` in `beforeEach`.

#### Switcher buttons appear to do nothing
`i18n.changeLanguage(lng)` is a no-op when `lng` matches the current language — the `languageChanged` event doesn't fire. Check `i18n.language` against the requested code before assuming the listener is broken.

#### `Cannot find module './locales/en.json'`
`tsconfig.json` needs `resolveJsonModule: true`. The react-vite template has it set; scenarios that diverged may not.

#### TypeScript errors about `t()` return type
react-i18next v15+ uses TypeScript template literal types for return inference. If you see `string | $T` errors, set `returnNull: false` in the i18n init config (the template default does).

#### `<html dir>` doesn't update on language change
The `applyDocumentLocale` helper requires `document.documentElement` to exist. In SSR/test contexts, guard with `if (typeof document === "undefined") return;`. The template implementation already does this.

#### CJK text overflowing fixed-width buttons
Japanese strings can be ~30% longer in pixels even though shorter in characters. Use `min-width: max-content` on labels and avoid fixed-width buttons that fit English snugly.

#### `Property 'ListFormat' does not exist on type 'typeof Intl'`
The TypeScript lib doesn't include ES2021 Intl types. Add `"ES2021.Intl"` to the `lib` array in `tsconfig.json` (next to `ES2020`). Runtime support has been universal in Node and browsers since 2020 — this is purely a type-resolution issue.

#### Tests show `Not implemented: HTMLCanvasElement.prototype.getContext`
axe-core probes canvas during color-contrast / icon-ligature checks. jsdom doesn't implement canvas. Stub it in `test-setup.ts`:

```ts
vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
```

The test still passes without the stub — it's pure stderr noise — but the noise hides real failures.

#### `selectors.app is possibly 'undefined'`
The literal-tree const was annotated `: LiteralSelectorTree`, which widens it to `Record<string, ...>`. Under `noUncheckedIndexedAccess`, every member access becomes `T | undefined`. Use `satisfies` instead:

```ts
const literalSelectors = {
  app: { title: "app-title" },
} satisfies LiteralSelectorTree;
```

Same validation, narrow inferred type, ergonomic `selectors.app.title` access.

#### Test asserts the wrong thing in cimode
If `getByText(strings.x.y)` returns nothing in cimode, the component probably isn't passing the registry path through `t()`. Check that the JSX is `{t(strings.x.y)}` not `{strings.x.y}` — the bare reference renders the key path itself, but `t()` is what i18next intercepts and (in cimode) echoes back as the key.

#### Plural test fails: expected `_one`, got base key
You're running in cimode. cimode bypasses CLDR plural logic. Either:
- Move the plural test into the real-locale describe block (`await setLocale("en")`).
- Or change the assertion to expect the base key (cimode behavior).

---

### **8. Anti-Patterns**

- **Inlining `t("app.title")` instead of `t(strings.app.title)`.** The `strings` registry is the type-safety mechanism — bypassing it forfeits compile-time key checking.
- **Per-component locale state.** i18next is a singleton; one source of truth.
- **Translating without a `LOCALE_CONFIG` entry.** Every supported locale needs a config entry — otherwise `<html dir>` and the switcher's native label can't resolve.
- **Adopting i18n preemptively without concrete demand.** Cost is real; benefit is conditional. See Section 2.
- **Splicing values with template literals into a registry string.** Use i18next's `{{var}}` JSON placeholders + `t(key, params)`. One way to do interpolation.
- **`screen.getByText("Refresh")` in a component test.** Hardcoded copy in tests is the failure mode this whole skill exists to eliminate. Use `getByTestId(selectors.x.y)` or `getByText(strings.x.y)` (cimode default) instead. The ESLint rule should catch it before review.
- **Disabling the test-text-matcher ESLint rule "just for this test."** That's the sound of test stability rotting in slow motion. If the rule fires, the test is asserting on copy that will move. Move the assertion to a test ID or to a real-locale describe block.
- **Calling `new Intl.NumberFormat(...)` directly from a component.** Use `formatNumber(...)` from `i18n/format.ts`. The wrapper handles cimode/dev pseudo-locales and centralizes the locale-resolution policy.
- **Using `jest-axe` with vitest.** The matcher signature doesn't match vitest's `RawMatcherFn`, forcing casts. Use `axe-core` directly — `await axe.run(container)` then assert `results.violations.toEqual([])`.
- **Annotating selector trees with `: LiteralSelectorTree`.** Widens the type and breaks ergonomic access under `noUncheckedIndexedAccess`. Use `satisfies` instead.