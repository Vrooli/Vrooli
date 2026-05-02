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
| **1 — Registry + lint rule** | `templates/scenarios/react-vite/` (template default) | Already paid |
| **2 — i18next + first non-English locale** | This skill | A day or two per scenario |
| **3 — Shared `@vrooli/i18n` package** | Future, after 3+ scenarios converge | Out of scope |

If a scenario hasn't done Phase 1, fix that first — see `templates/scenarios/react-vite/ui/src/consts/strings.ts` and the `no-restricted-syntax` rule in `eslint.config.js` for the canonical shape.

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
2. Confirm `eslint.config.js` has the `no-restricted-syntax` rule rejecting JSX literals.
3. Grep for directional Tailwind utilities and verify they've been migrated to logical equivalents:

```bash
rg -n '\b(ml-|mr-|pl-|pr-|left-[0-9]|right-[0-9]|text-left|text-right|rounded-l|rounded-r)' scenarios/{{TARGET}}/ui/src
```

**Exit:** All three pass. If any fail, halt and run the Phase 1 prep first — bolting i18next onto a codebase still inlining strings creates a partial registry and loses the type-safety wins.

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

#### **Phase H — Tests**

**Actions:**
1. Ensure `scenarios/{{TARGET}}/ui/src/test-setup.ts` imports `@testing-library/jest-dom/vitest`.
2. Reference it from `vite.config.ts` via `test: { setupFiles: ['./src/test-setup.ts'] }`.
3. Add a vitest suite that:
   - Renders the App in each locale, asserts locale-specific text appears.
   - Toggles via the switcher, asserts the DOM updates and `document.documentElement.lang` changes.
   - Confirms localStorage persists the chosen locale.
   - Imports the locale JSON (`import en from "./i18n/locales/en.json"`) and references its keys — never hard-code translated strings in test assertions.
4. Run `corepack pnpm test`.

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

| Question | Add to `LOCALE_CONFIG` | Inline in component | Add to JSON |
|----------|------------------------|---------------------|-------------|
| Locale-dependent (text direction)? | YES | NO | NO |
| Native language label for switcher? | YES | NO | NO |
| One-off translated string? | NO | NO | YES |
| Locale-aware date formatting? | NO — use `Intl.DateTimeFormat(currentLocale)` | NO | NO |

---

### **5. Common Pitfalls**

- **Bare registry refs in JSX.** `{strings.app.title}` renders the *key path* (`"app.title"`), not the translation. The lint rule won't catch it — the registry reference is "valid." Code-review for `t()` wrappers.
- **Server-side English leaking through.** API responses with user-facing text need to ship error codes the client maps to localized strings. Flag this during adoption even though it's out of scope.
- **Hand-rolled date formatting.** Use `Intl.DateTimeFormat(currentLocale, …)` and `Intl.NumberFormat(currentLocale, …)`. Don't write per-locale formatters.
- **CJK whitespace assumptions.** Japanese has no inter-word spaces; Chinese punctuation has different widths. Be skeptical of layout assertions that depend on text width.
- **shadcn primitives with directional classes.** Some primitives use `ml-`/`pl-` directly. Audit on first RTL locale; the Phase 1 logical-utility convention only covered code under direct ownership.
- **i18next init race.** If `<App>` renders before init resolves, you'll see a flash of the key path. With no backend, init is synchronous — but if you add a backend later, gate render on the init promise.

---

### **6. Output Expectations**

When this skill completes:

- `corepack pnpm install` succeeds.
- `corepack pnpm lint` shows no new errors in any file modified during adoption.
- `corepack pnpm test` passes the locale-switching suite.
- `corepack pnpm build` succeeds.
- `<html lang>` and `<html dir>` reflect the active locale at runtime.
- Switching languages via the switcher persists across reloads.
- The `no-restricted-syntax` ESLint rule still passes (no inline JSX literals introduced).

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

---

### **8. Anti-Patterns**

- **Inlining `t("app.title")` instead of `t(strings.app.title)`.** The `strings` registry is the type-safety mechanism — bypassing it forfeits compile-time key checking.
- **Per-component locale state.** i18next is a singleton; one source of truth.
- **Translating without a `LOCALE_CONFIG` entry.** Every supported locale needs a config entry — otherwise `<html dir>` and the switcher's native label can't resolve.
- **Adopting i18n preemptively without concrete demand.** Cost is real; benefit is conditional. See Section 2.
- **Splicing values with template literals into a registry string.** Use i18next's `{{var}}` JSON placeholders + `t(key, params)`. One way to do interpolation.