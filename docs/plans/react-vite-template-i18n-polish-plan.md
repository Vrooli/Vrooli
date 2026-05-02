# React-Vite Template i18n & Test Infrastructure Polish

## 0. Revision History

- **2026-05-02 r3** — Tightened executable details discovered during plan
  review: (A) corrected ESLint anchor for `strings/no-unused-keys` —
  the template only lints `**/*.{ts,tsx}`, so anchoring on `en.json`
  doesn't work; both new rules now anchor on `strings.generated.ts`
  (which is already linted and deterministically one file). (B) Made the
  `_comment` sentinel interaction explicit: `gen-strings.mjs` skips keys
  starting with `_`, and the new ESLint rules + `locales.test.ts` reuse
  the same skip list. (C) Fixed the tsconfig advisory remediation —
  `noWarnOnMultipleProjects` is not a real typescript-eslint option;
  the correct fix is `parserOptions.projectService: true`. (D) Aligned
  the new plural example with the convention already in the template
  (base key + `_one`), rather than introducing a divergent
  `_zero/_one/_other` shape.

- **2026-05-02 r2** — Corrected the CI-enforcement story. The original draft
  claimed `vrooli scenario test` doesn't run lint or type-check; that was wrong.
  test-genie has a dedicated lint phase (`scenarios/test-genie/api/internal/lint/nodejs/linter.go`)
  that invokes `tsc --noEmit` and `eslint .` directly, and the template's
  `.vrooli/testing.json` already sets `node_package: { strict: true }` so
  ESLint warnings fail the phase (verified at
  `scenarios/test-genie/api/internal/lint/runner.go:159`). The narrow gap that
  remains is that the lint phase invokes `tsc` and `eslint` as binaries, not
  through pnpm scripts, so the `pnpm strings:check` chained inside
  `pnpm type-check` never runs in CI — and a future `pnpm strings:audit`
  wouldn't either. Item 8 in §3 has been narrowed accordingly. Phase 4
  (which proposed editing test-genie to invoke pnpm scripts) is removed; the
  codegen-freshness and orphan-key checks are now implemented as custom
  ESLint rules so they ride the existing eslint pass test-genie already runs.

## 1. Purpose

The `templates/scenarios/react-vite` template was recently rebuilt around a typed
string registry (codegen'd from `en.json`), `cimode` test default, lint-enforced
JSX/test boundaries, locale parity tests, axe-core a11y regression, and locale-
aware `Intl` formatters. End-to-end validation (scaffold → install → strings:check
→ type-check → lint → test → build) passes cleanly on a freshly generated
scenario.

Before broad adoption across hundreds of future scenarios, close a small set of
correctness, coverage, and ergonomic gaps so the template ships at the standard
that scale demands. This plan enumerates the gaps with concrete file references
and lands them as a single greenfield change to the template — no migration
shims, no deprecation paths.

## 2. Required Reading

Future agents executing this plan must run:

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Plus, for context on the test/runner side:

```bash
prompt-manager skill read test
```

In-repo references:

- Source template: `templates/scenarios/react-vite/`
- Generator entry point: `vrooli scenario generate react-vite ...` (see
  `vrooli scenario template show react-vite` for required vars)
- Scenario CI surface (read-only context — this plan does NOT edit test-genie):
  - Unit phase runner: `scenarios/test-genie/api/internal/unit/nodejs/runner.go`
    (invokes `pnpm test`)
  - Lint phase runner: `scenarios/test-genie/api/internal/lint/nodejs/linter.go`
    (invokes `tsc --noEmit` and `eslint .` directly, not via pnpm)
  - Strict-mode wiring: `scenarios/test-genie/api/internal/lint/runner.go:159`
    (when `testing.json` declares `strict: true` for a handler, lint warnings
    fail the phase). The template's `.vrooli/testing.json` already sets this
    for `node_package`.

## 3. Problem Statement

The template's i18n + test architecture is correct but has nine concrete leaks
that compound across hundreds of scaffolded scenarios:

1. **`as Locale` cast in `ui/src/App.tsx:13`** — `i18n.language as Locale` is a
   runtime lie (the value can be `cimode`, `en-US`, or a fallback). Casts in
   templates teach the wrong pattern at scale.

2. **`format.test.ts:32-37` is a false-positive test** — asserts
   `formatNumber(1234.5) === "1,234.5"` for both `en` and `ja`, but those
   produce identical output. The test passes but doesn't validate that
   locale-switching takes effect.

3. **`SUPPORTED_LOCALES` and `LOCALE_CONFIG` are duplicated** in
   `ui/src/i18n/index.ts:29` and `:39`. Adding a locale requires editing both;
   the order they fall out of sync is one rebase away.

4. **No type augmentation for `t()`** — the typed `strings` registry stops
   bad paths *into* `t()`, but a hand-written `t("not.a.key")` still
   type-checks. With `returnNull: false`, missing keys silently render the
   key string at runtime.

5. **Test-stability lint rule covers only `*ByText`** —
   `eslint.config.js:187` bans string literals to `getByText`/`findByText`
   etc., but not `getByLabelText`, `getByPlaceholderText`, `getByTitle`,
   `getByAltText`, `getByDisplayValue`. Same brittleness, no rule.

6. **No demonstration of non-English plural variants** — only `en._one` is
   shown. Scenarios reaching for Russian, Arabic, Polish, or any locale with
   richer CLDR plural categories will guess at the JSON shape.

7. **`{{SCENARIO_DISPLAY_NAME}}` placeholder in `ui/src/i18n/locales/*.json`
   is a generator-time placeholder, not an i18next interpolation** — verified
   via `vrooli scenario generate`: the value is replaced before runtime.
   Authors editing en.json after generation will assume it's a runtime
   `{{var}}` and try `t(strings.app.title, { SCENARIO_DISPLAY_NAME: ... })`.
   No comment in the locale files distinguishes the two interpolation
   surfaces.

8. **Codegen-freshness (`strings.generated.ts` ↔ `en.json`) isn't part of
   the lint surface.** Lint and type-check *are* enforced by test-genie
   (`scenarios/test-genie/api/internal/lint/nodejs/linter.go`), and ESLint
   warnings already fail the phase under
   `node_package: { strict: true }` in `.vrooli/testing.json`. But the
   lint phase invokes `tsc --noEmit` and `eslint .` *as binaries*, not
   through pnpm scripts. The template's `pnpm type-check` script is
   `pnpm strings:check && tsc --noEmit`, so the `strings:check` half —
   which validates that the generated file matches the catalog — is
   never exercised in CI. Realistic failure mode: someone hand-edits
   `strings.generated.ts` (they shouldn't — it's generated) and CI
   misses the divergence because the Vite plugin only auto-regenerates
   when a build/test path is taken locally.

9. **No unused-key detection.** Drift goes both ways. Today the parity
   test catches keys missing from non-English locales; nothing catches
   orphan keys in `en.json` that no callsite uses. The main Vrooli app
   already has tooling for this (`9d67d8fdee i18n-unused`); the template
   does not. Like (8), this needs to ride a surface CI already runs —
   ESLint — rather than a new pnpm script test-genie wouldn't invoke.

Two quality items below the line:

10. **Decorative icon (`ArrowRight` in `App.tsx:112`) lacks `aria-hidden`**
    — axe doesn't flag it today because the surrounding `<Button>` has an
    accessible name, but icons next to text labels should be hidden from AT.
    Templates set the example for hundreds of downstream UIs.

11. **`tsconfig` produces "Multiple projects found" warning on lint** —
    cosmetic but visible to every scenario author the first time they
    run `pnpm lint`. Resolved by switching `parserOptions` to the
    modern `projectService: true` API in `eslint.config.js`, with a
    `tsconfig.eslint.json` + `references` fallback if `projectService`
    interacts badly with the `import/resolver` settings.

## 4. Scope

### In scope

- All eleven items above, applied directly to
  `templates/scenarios/react-vite/`
- Two custom ESLint rules (in `templates/scenarios/react-vite/ui/eslint-rules/`,
  registered from `eslint.config.js`):
  1. **`strings/codegen-fresh`** — fails when
     `src/consts/strings.generated.ts` diverges from
     `src/i18n/locales/en.json`. Replaces the need to invoke
     `pnpm strings:check` from CI; the rule rides the existing eslint
     pass that test-genie already runs (and is already strict-mode
     gated for `node_package`).
  2. **`strings/no-unused-keys`** — walks `en.json` and grep-scans
     `src/**/*.{ts,tsx}` for usage of each leaf path; reports orphan
     keys as eslint errors. Same rationale: rides the existing pass.
- Validation by re-running `vrooli scenario generate react-vite ...` into a
  scratch path and exercising the full pipeline

### Out of scope

- Migrating *existing* scenarios to the new template. That's a separate
  rollout described in §10 but not executed by this plan.
- Adding new locales beyond `en` + `ja`. The template stays bilingual; the
  goal is to make adding a locale a 3-line change, not to ship more.
- Changes to the codegen output format
  (`ui/src/consts/strings.generated.ts`) beyond what new keys add.
- Changes to `selectors.ts` registry shape — that file is already mature and
  not the subject of this plan.
- **Any changes to `scenarios/test-genie/`.** The earlier draft proposed
  editing the Node runner; that's unnecessary because lint and type-check
  are already enforced. Codegen-freshness and orphan-key detection ride
  the existing eslint pass via custom rules instead.
- Adding `i18next-parser` (or any new pnpm script) for unused-key
  detection — superseded by the custom ESLint rule. Avoids growing a
  new tool surface that test-genie wouldn't invoke anyway.
- Anything outside `templates/scenarios/react-vite/`.

## 5. Current Technical Context

Files this plan touches:

| File | Purpose |
|---|---|
| `templates/scenarios/react-vite/ui/src/App.tsx` | Demo component; locale cast (item 1), missing `aria-hidden` (item 10) |
| `templates/scenarios/react-vite/ui/src/i18n/index.ts` | i18n init, locale registry; duplication (item 3), needs `getCurrentLocale()` (item 1) |
| `templates/scenarios/react-vite/ui/src/i18n/format.test.ts` | False-positive locale test (item 2) |
| `templates/scenarios/react-vite/ui/src/i18n/locales/en.json` | Add `_comment` sentinel for the interpolation-surfaces explanation (item 7); add the plural example matching the existing `refreshCount` shape: base + `_zero` + `_one` (item 6) |
| `templates/scenarios/react-vite/ui/src/i18n/locales/ja.json` | Add `_comment` sentinel; mirror the plural example with the single base form (item 6) |
| `templates/scenarios/react-vite/ui/src/i18n/locales/locales.test.ts` | Strip underscore-prefixed keys before parity comparison (alongside the existing plural-suffix stripper) per item 5's sentinel convention |
| `templates/scenarios/react-vite/ui/scripts/gen-strings.mjs` | In `buildKeys`, skip entries whose key starts with `_` (item 5) so `_comment` and any future sentinel doesn't leak into `strings.generated.ts` |
| `templates/scenarios/react-vite/ui/scripts/gen-strings.test.mjs` | NEW — unit test asserting `_comment` and other underscore-prefixed keys are excluded from the generated output (item 5) |
| `templates/scenarios/react-vite/ui/src/react-i18next.d.ts` | NEW — module augmentation for typed `t()` (item 4) |
| `templates/scenarios/react-vite/ui/eslint.config.js` | Extend `*ByText` regex to all `*By*` literal-string queries (item 5); register the two new custom rules (items 8 + 9); switch `parserOptions` to `projectService: true` (item 11) |
| `templates/scenarios/react-vite/ui/eslint-rules/codegen-fresh.js` | NEW — custom ESLint rule that fails when `strings.generated.ts` diverges from `en.json`; reuses `generateContents()` from `gen-strings.mjs` (item 8) |
| `templates/scenarios/react-vite/ui/eslint-rules/no-unused-keys.js` | NEW — custom ESLint rule that fails on orphan keys in `en.json`; anchored on `strings.generated.ts`; skips underscore-prefixed sentinels (item 9) |
| `templates/scenarios/react-vite/ui/eslint-rules/index.js` | NEW — flat-config plugin export wrapping both rules so `eslint.config.js` can register them as a plugin |
| `templates/scenarios/react-vite/ui/tsconfig.json` / `tsconfig.node.json` | No edits unless `projectService: true` requires the fallback `tsconfig.eslint.json` (item 11) |

Files this plan does NOT touch (deliberately):

- `ui/src/consts/strings.ts` (entry shim, fine as-is)
- `ui/src/consts/strings.generated.ts` (regenerated automatically)
- `ui/src/consts/selectors.ts` and `selectors.test.ts`
- `ui/scripts/gen-strings.mjs` and `vite-plugin-strings-codegen.mjs`
- `ui/src/test-setup.ts` — pattern is correct
- `ui/src/App.a11y.test.tsx` — pattern is correct
- `ui/src/App.test.tsx` (only test additions, not refactors)

## 6. Target End State

After this plan executes, all of the following hold for a freshly scaffolded
scenario:

- `App.tsx` contains zero `as` casts. The current locale flows through a
  named helper exported from `i18n/index.ts`.
- `format.test.ts` "switches output when i18n.language changes" actually
  asserts divergent output between two locales (e.g., decimal style or
  currency formatting), or the test is removed and replaced with one that
  does.
- `LOCALE_CONFIG` is the single source of truth; `SUPPORTED_LOCALES` is
  derived from `Object.keys(LOCALE_CONFIG)`.
- `t("does.not.exist")` is a TypeScript error, not a silent runtime fallback.
- `getByLabelText("…")`, `getByPlaceholderText("…")`, `getByTitle("…")`,
  `getByAltText("…")`, `getByDisplayValue("…")` with string-literal first
  arguments all fail lint with the same message family as `getByText`.
- `en.json` and `ja.json` each contain at least one realistic plural example
  beyond `_one`, with a 1-line comment in the JSON pointing readers to
  i18next CLDR plural docs.
- `en.json` has a leading `_comment` field (or sibling sentinel) explaining
  that `{{NAME}}` placeholders inside locale files are *generator-time* if
  uppercase + matched by a generator var, and *runtime* otherwise.
- The custom ESLint rule `strings/no-unused-keys` reports zero violations
  on a freshly generated scenario, and reports a violation when a key in
  `en.json` is not referenced anywhere under `src/`.
- The custom ESLint rule `strings/codegen-fresh` reports zero violations
  on a freshly generated scenario, and reports a violation when
  `strings.generated.ts` diverges from `en.json`.
- Decorative icons used purely as glyph-affordances next to text labels
  carry `aria-hidden="true"`. Lint or jsx-a11y already covers labelless
  icon-only buttons; this is the complementary case.
- `pnpm lint` runs without the "Multiple projects found" advisory.
- `vrooli scenario test <name>` enforces every rule above. The lint phase
  (`scenarios/test-genie/api/internal/lint/nodejs/linter.go`) already
  invokes `eslint .` and `tsc --noEmit`, and the template's
  `.vrooli/testing.json` already gates `node_package` with `strict: true`,
  so any new rule failure fails the lint phase. No test-genie change
  required.

## 7. Implementation Strategy

Phased so each phase is independently verifiable. All phases are
template-only — no test-genie changes (see §0 for why).

### Phase 1 — Template ergonomics & correctness (touches only `templates/scenarios/react-vite/ui/`)

1. **`getCurrentLocale()` helper.** In
   `ui/src/i18n/index.ts`, add and export
   `getCurrentLocale(): Locale` that calls `isSupported(i18n.language)` and
   falls back to `"en"`. Replace `App.tsx:13` `i18n.language as Locale`
   with `const currentLocale = getCurrentLocale();`. No more casts in the
   demo file.
2. **Derive `SUPPORTED_LOCALES`.** Reorder `i18n/index.ts` so
   `LOCALE_CONFIG` is declared first; export
   `SUPPORTED_LOCALES = Object.keys(LOCALE_CONFIG) as Locale[]` and
   re-derive `Locale = keyof typeof LOCALE_CONFIG`. Update `Locale` JSDoc.
3. **Fix `format.test.ts:32-37`.** Replace the false-positive test with one
   that asserts `formatNumber(1234.5)` returns `"1,234.5"` for `en-US` and
   `"1.234,5"` for `de-DE` *via the language-switch path* (not via
   `localeOverride`). Use a locale where output diverges; explain in a
   comment why English vs Japanese isn't a useful comparison for thousand
   separators.
4. **Plural example.** Add a `notifications.summary` (or similar) key to
   `en.json` matching the convention already used by `refreshCount`:
   the bare key acts as `_other` (CLDR fallback), with explicit
   `_zero` and `_one` siblings. So the en entries are
   `notifications.summary` (other), `notifications.summary_zero`, and
   `notifications.summary_one`. The `ja` entry is the single base
   form, since Japanese has no plural distinction. Do **not**
   introduce a `_other` sibling — that would teach a different
   convention than the existing `refreshCount` shape. Render the new
   key conditionally in `App.tsx` near the refresh count, with a test
   asserting the right form selects at `count = 0, 1, 5`. The plural-
   suffix stripper in `locales.test.ts` already handles this; verify
   with a quick test rather than re-implementing.
5. **Locale-file comments + underscore-prefix sentinel convention.**
   Add a leading `"_comment"` field in both `en.json` and `ja.json`
   explaining the two interpolation surfaces: uppercase-snake-case
   `{{NAME}}` is generator-time; lowercase `{{count}}` is
   i18next-runtime.

   The `_comment` field is a sentinel — and adding it surfaces a
   three-way interaction that must be resolved consistently across
   the codegen, the parity test, and the new ESLint rules. Pick one
   convention (**any key whose final segment starts with `_` is a
   sentinel and is ignored by every consumer of the catalog**) and
   apply it in four places:

   1. **`scripts/gen-strings.mjs`** — in `buildKeys`, skip entries
      whose key starts with `_`. The generated `strings.generated.ts`
      must not contain `_comment` (or any future underscore-prefixed
      key). This is the *only* file where the convention matters for
      typing — every downstream consumer reads `strings.generated.ts`
      or the catalog directly.
   2. **`src/i18n/locales/locales.test.ts`** — strip underscore-
      prefixed keys before parity comparison, alongside the existing
      plural-suffix stripper.
   3. **`src/eslint-rules/no-unused-keys.js`** (new, Phase 3) — when
      enumerating catalog keys to check for orphans, skip
      underscore-prefixed segments. Default ignore list is
      `["_*"]`, configurable via rule options.
   4. **`src/eslint-rules/codegen-fresh.js`** (new, Phase 3) — since
      this rule re-invokes `generateContents()` from the same
      `gen-strings.mjs`, it inherits the skip behavior automatically.
      No extra work, but document the dependency so future authors
      don't break parity by forking the codegen logic.

   Verify the convention with a unit test in `gen-strings.test.mjs`
   (new, alongside `gen-strings.mjs`) asserting that an `en.json`
   containing `_comment` produces a `strings.generated.ts` without a
   `_comment` leaf.
6. **Decorative icon.** Add `aria-hidden="true"` to the `<ArrowRight>` in
   `App.tsx:112`.
7. **tsconfig hygiene.** Silence the typescript-eslint "Multiple
   projects found, consider using a single `tsconfig` with
   `references` to speed up" advisory by switching
   `eslint.config.js`'s `parserOptions` from the explicit
   `project: ["./tsconfig.json", "./tsconfig.node.json"]` array to
   the modern projectService API: `parserOptions.projectService: true`
   (drop the explicit `project` field; keep `tsconfigRootDir`). This
   is the canonical typescript-eslint v8+ remediation. The earlier
   draft of this plan named `noWarnOnMultipleProjects` — that is
   not a real typescript-eslint option; do not introduce it. If
   `projectService: true` causes any unexpected behavior with the
   `import/resolver` settings block, the fallback is to add a
   `tsconfig.eslint.json` that uses `references` to compose the two
   existing tsconfigs and point `parserOptions.project` at it.

### Phase 2 — Typed `t()` (NEW file)

8. **Module augmentation.** Add
   `ui/src/react-i18next.d.ts` declaring
   ```ts
   import "react-i18next";
   import en from "./i18n/locales/en.json";
   declare module "react-i18next" {
     interface CustomTypeOptions {
       defaultNS: "translation";
       resources: { translation: typeof en };
     }
   }
   ```
   Verify `t("not.a.key")` becomes a TypeScript error. Confirm the
   existing `t(strings.app.title)` callsites remain valid (string-typed
   key paths from the registry must continue to satisfy the augmented
   `t()` signature). If they don't, the registry `as const` types narrow
   to literal union — that's the desired behavior; document any
   conflicts and resolve by typing registry leaves as `keyof typeof en`
   chains.

### Phase 3 — Lint coverage & registry-drift detection

Why this phase is template-only: test-genie's lint phase already invokes
`eslint .` with `node_package: { strict: true }`, so any rule we register
in `eslint.config.js` is automatically a CI gate. That's a much smaller
change than wiring new pnpm scripts into the runner — and avoids the
risk of breaking other scenarios that share the runner.

9. **Extend `*By*` literal ban.** Update
   `ui/eslint.config.js:187,193` regex to cover the full Testing Library
   surface: `getByText|findByText|queryByText|...|getByLabelText|
   findByLabelText|...|getByPlaceholderText|...|getByTitle|...|
   getByAltText|...|getByDisplayValue|...`. Update the message text to
   mention the broader rule, point at `selectors.x.y` and `strings.x.y`
   as the two correct paths, and call out that `getByRole(role, { name: t(strings.x.y) })`
   is the i18n-correct way to query by name. Verify with probe files
   (deliberate violations) that the rule fires on every covered query.

10. **Custom rule `strings/codegen-fresh`.** Add
    `ui/eslint-rules/codegen-fresh.js` — a flat-config-compatible custom
    rule that:
    - Anchors on `src/consts/strings.generated.ts` — the rule emits
      *only* when `context.filename` ends in `strings.generated.ts`.
      That file is linted by the template's existing `files:
      ["**/*.{ts,tsx}"]` glob and exists exactly once, so the gate
      runs exactly once per lint pass.
    - Reads `src/i18n/locales/en.json` and the current contents of
      `src/consts/strings.generated.ts` from disk.
    - Recomputes the expected generated content by reusing
      `scripts/gen-strings.mjs`'s `generateContents()` export. This
      automatically inherits the underscore-prefix skip convention
      from Phase 1 item 5 — the rule must NOT re-implement key
      walking, since drift between the rule's logic and
      `gen-strings.mjs`'s logic would defeat the entire freshness
      check.
    - Reports a diagnostic on `strings.generated.ts` at line 1,
      column 1 when the contents diverge. Message points the author
      at `pnpm strings:gen` and includes a short diff hint
      (e.g., first divergent line) so the author can see what's out
      of sync.

11. **Custom rule `strings/no-unused-keys`.** Add
    `ui/eslint-rules/no-unused-keys.js` — a flat-config-compatible custom
    rule that:
    - **Anchors on `src/consts/strings.generated.ts`**, NOT `en.json`.
      The template's `eslint.config.js` has
      `files: ["**/*.{ts,tsx}"]`, so ESLint never lints `en.json`.
      Adding a JSON-files block just to give the rule an anchor would
      pull `en.json` through the TypeScript parser, which is wrong.
      `strings.generated.ts` IS already linted, exists exactly once
      per scenario, and has the same logical content as the catalog —
      it's the correct anchor.
    - On the first lint visit to `strings.generated.ts` in a given
      run (track via a module-level `Map` keyed on `context.cwd`),
      reads `src/i18n/locales/en.json`, flattens to dotted paths, and
      reads every `*.{ts,tsx}` under `src/` (excluding
      `strings.generated.ts` itself, which trivially "uses" every
      key). For each leaf key, checks for a usage signal: literal
      `t("foo.bar")`, `t(strings.foo.bar)` member access, or string
      occurrence of the dotted key in source.
    - Reports each orphan key as a separate diagnostic on
      `strings.generated.ts` at line 1, column 1, with the message
      naming the orphan key and pointing the author at the catalog
      file. (en.json is referenced in the message text — not as the
      diagnostic location — so editors can still link to it.)
    - **Skips underscore-prefixed catalog keys** as part of the
      sentinel convention defined in Phase 1 item 5 — including but
      not limited to `_comment`. Configurable via rule options
      (`ignorePrefix: "_"` default).
    - Side note: detecting *missing* keys at type-check time is
      already covered by Phase 2's module augmentation (`t("nope")` is
      a TS error), so this rule only handles the orphan direction.

12. **Plugin wiring.** Add `ui/eslint-rules/index.js` exporting
    ```js
    export default {
      rules: {
        "codegen-fresh": codegenFresh,
        "no-unused-keys": noUnusedKeys,
      },
    };
    ```
    Register in `eslint.config.js` under `plugins: { strings: ... }` and
    enable both rules at error severity. Verify (via probe edits) both
    rules fire and produce useful messages.

### Phase 4 — Validation

13. Re-run the same end-to-end check used during the assessment: scaffold
    a scenario via `vrooli scenario generate react-vite ...`, run
    `corepack pnpm install --ignore-workspace`, then `pnpm strings:check`,
    `pnpm type-check`, `pnpm lint`, `pnpm test`, `pnpm build`. All green.
14. Probe-file pass: introduce deliberate violations and confirm each
    fails with the documented message:
    - JSX text/expression/string-attribute literals (existing rules)
    - String-literal first-arg to each newly-banned `*By*` query
      (`getByLabelText`, `getByPlaceholderText`, `getByTitle`,
      `getByAltText`, `getByDisplayValue`)
    - A line `t("definitely.not.a.key")` (TS error from Phase 2
      augmentation)
    - Hand-edit `strings.generated.ts` to break sync (codegen-fresh
      rule)
    - Add an unused key (`extras.dead`) to `en.json`
      (no-unused-keys rule)
    - Sentinel-skip probe: add a key `_lint_probe` to `en.json`,
      regenerate via `pnpm strings:gen`, confirm
      `strings.generated.ts` does NOT contain `_lint_probe`,
      confirm `locales.test.ts` passes despite ja.json not having
      it, and confirm `strings/no-unused-keys` does NOT flag it.
      All four are tied to the underscore-prefix sentinel
      convention from Phase 1 item 5.
    - tsconfig advisory probe: confirm `pnpm lint` runs without the
      "Multiple projects found" message after applying item 11.
15. Run `vrooli scenario test <validation-scenario-name>` and confirm the
    lint phase reports each violation (verifies the rules ride the
    existing test-genie eslint pass under `strict: true`).
16. Revert all probes; final clean run of `vrooli scenario test` passes.
17. Delete the validation scenario; `git status` confirms a clean tree.

## 8. Contract Decisions

- **Locale identity is `keyof typeof LOCALE_CONFIG`.** No string-typed
  locale parameters anywhere in the template's source tree. Any future
  helper that takes a locale takes `Locale`, not `string`.
- **`t()` is typed against `en.json`.** That makes English the contract
  surface for keys (parity tests already enforce shape across other
  locales). Authors who delete a key from `en.json` get type errors at
  every callsite — desired.
- **Underscore-prefixed catalog keys are sentinels** and are
  excluded from every consumer of `en.json` by convention:
  `gen-strings.mjs` skips them in `buildKeys`, `locales.test.ts`
  strips them before parity comparison, `strings/no-unused-keys`
  skips them by default (`ignorePrefix: "_"`), and
  `strings/codegen-fresh` inherits the skip via `generateContents()`
  reuse. `_comment` is the canonical first user of this convention.
  Authors may add other sentinels (e.g., `_meta`, `_unused`) without
  changing rule code.
- **Orphan keys are a hard gate.** The `strings/no-unused-keys` ESLint
  rule is `error` severity, and `node_package: { strict: true }` in
  `.vrooli/testing.json` already converts ESLint warnings into phase
  failures — so any orphan key blocks CI. Fix: delete the key or
  reference it.
- **Codegen drift is a hard gate.** Same wiring: the
  `strings/codegen-fresh` rule is `error` severity and rides the same
  strict-mode pass.
- **CI enforcement requires no test-genie change.** All new gates live
  in ESLint rules registered from the template's `eslint.config.js`.
  test-genie's existing lint phase
  (`scenarios/test-genie/api/internal/lint/nodejs/linter.go` →
  `eslint . --format json`) picks them up automatically.

## 9. Testing Plan

Automated, no manual checklists:

| Layer | Test | Asserts |
|---|---|---|
| Codegen | existing `strings:check` | `strings.generated.ts` matches `en.json` |
| Locale parity | `locales.test.ts` | every locale shares the same logical keys; `_comment` is ignored |
| Plural rules | NEW test in `App.test.tsx` | the new `notifications.summary` key resolves to the right plural form for `count = 0, 1, 5` in real English |
| Format helpers | `format.test.ts` updated | locale-divergent output proves `i18n.language` actually drives `Intl` |
| Typed `t()` | `tsc --noEmit` (covered by `type-check`) | a probe line `t("nope.nope")` causes a build failure when temporarily added |
| Lint coverage | `eslint .` against committed sources passes; probe files with each kind of literal-string Testing Library query fail with the new message family | confirmed during validation |
| Unused-key rule | `strings/no-unused-keys` reports zero on the template; with a deliberately-orphaned key, reports an error on `en.json` | confirmed during validation |
| Codegen-fresh rule | `strings/codegen-fresh` reports zero on the template; hand-editing `strings.generated.ts` produces an error pointing at `pnpm strings:gen` | confirmed during validation |
| CI enforcement | `vrooli scenario test <validation-scenario>` runs the lint phase (`tsc --noEmit` + `eslint .`); under `node_package: { strict: true }` any new rule violation fails the phase | confirmed end-to-end during validation |

No manual test checklists. Where a check requires a probe, the probe is
ephemeral (rm'd at end of validation), not committed.

## 10. Rollout / Validation Checklist

This plan ships the template improvements only. Rollout to existing
scenarios is a separate effort, but the plan must leave the door open for
it cleanly:

- `vrooli scenario generate react-vite ...` produces a scenario where
  `pnpm install && pnpm strings:check && pnpm type-check && pnpm lint &&
  pnpm test && pnpm build` all pass on a fresh clone.
- `vrooli scenario test <new-scenario>` lint phase exercises every new
  rule (codegen-fresh, no-unused-keys, the broadened *By* rule). Verified
  by introducing each kind of violation in the validation scenario and
  observing the lint phase fail under `strict: true`.
- No changes to `scenarios/test-genie/` are made by this plan, so no
  cross-scenario regression risk.
- After merge, a follow-up issue is filed listing all existing scenarios
  with a `package.json` and a TODO to retrofit them to the new template
  conventions. (Not executed by this plan; just enumerate.)

## 11. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Typed `t()` augmentation conflicts with existing `t(strings.x.y)` callsites because `strings.x.y` is a `string` literal type and the augmented `t()` expects a key of `typeof en` | Validate during Phase 2; the registry's `as const` already narrows leaves to literal types. If TS still widens, type registry leaves as `keyof typeof en`-derived chains. |
| `strings/codegen-fresh` rule double-fires (one diagnostic per file lint visits) | Anchor the rule on `context.filename` — only emit when the file being linted is `src/consts/strings.generated.ts`. ESLint already lints that file, so the gate runs exactly once per pass. |
| `strings/no-unused-keys` rule has high I/O if it re-reads en.json + greps src/ for every linted file | Memoize the catalog read and the source-file scan in module-level state keyed by `context.cwd`. First lint visit to `strings.generated.ts` pays the cost; the rule no-ops on every other file. |
| Custom ESLint rules don't pick up file-system changes during a single lint run (e.g., en.json edited mid-run) | Acceptable: a lint run is a snapshot. The pre-commit/CI flow re-runs lint, which sees the new state. |
| Re-running `vrooli scenario generate` for validation pollutes `scenarios/`. Failed cleanup leaves a stray test scenario | Validation scenario name prefixed `_` or `_validation-` to make it obvious; the validation phase ends with explicit `rm -rf` and `git status` confirms it's gone. |
| `parserOptions.projectService: true` (item 11) is incompatible with the `import/resolver` typescript settings | Validation step will catch this. Documented fallback: introduce a `tsconfig.eslint.json` with `references` to compose the two existing tsconfigs, then point `parserOptions.project` at it. |
| Underscore-prefix sentinel skip is implemented inconsistently across the four touchpoints (gen-strings.mjs, locales.test.ts, the two ESLint rules) | Phase 1 item 5 makes the convention explicit and the unit test in `gen-strings.test.mjs` plus the parity test plus the `_comment` validation probe in Phase 4 each cover one of the touchpoints; codegen-fresh inherits via `generateContents()` reuse. Drift would have to break a committed test. |
| Custom rules require `eslint-rules/` files to be loaded by `eslint.config.js`, but flat-config plugin loading varies between eslint versions | Pin to flat-config syntax (already in use); `index.js` exports the standard `{ rules }` shape. Confirm by running `pnpm lint -- --print-config src/App.tsx` during validation. |

## 12. Non-Goals / Prohibited Patterns

- **No backwards-compatibility shim** for scenarios that haven't adopted the
  new lint rules. The Node runner feature-detects scripts; that's the only
  compatibility surface, and it's intentional rather than a polyfill.
- **No silent fixups in CI.** The codegen plugin already auto-heals
  `strings.generated.ts` during local dev/test, but
  `pnpm strings:check` (in `type-check`) catches drift in CI. Don't add a
  step that auto-commits regenerated files.
- **No mock data in tests** that reaches into i18next internals. Tests
  switch locale via the public `setLocale()` (or set `cimode` via the
  setup file) — never via `i18n.changeLanguage()` directly outside
  `setLocale`'s body.
- **No string literals in JSX or `*By*` queries** (covered by the
  expanded lint rule).
- **No new `as` casts** in template source (this is the whole point of
  Phase 1 item 1).
- **No README-only fixes** for design issues. Where the docs say "do X,"
  lint or types must enforce X.

## 13. Definition of Done

The plan is complete when *all* of the following are true on `master`:

1. All 11 items in §3 are addressed; each one has a concrete file diff or
   test in the merged change.
2. A freshly scaffolded scenario passes `pnpm install && pnpm
   strings:check && pnpm type-check && pnpm lint && pnpm test && pnpm
   build` from a clean state with zero warnings, zero advisories, and
   zero errors.
3. `vrooli scenario test <freshly-generated-scenario>` passes its lint
   phase (which invokes `tsc --noEmit` and `eslint .`) and its unit
   phase (which invokes `pnpm test`). No new test-genie scripts are
   needed.
4. Probe edits demonstrating every newly-covered lint rule were tested
   ephemerally during validation and each rule fired with the documented
   message. Probes are not committed.
5. `t("definitely.not.a.key")` is a build error, verified by temporarily
   adding the line, observing the error, removing it.
6. Orphan-key detection (item 9) is enforced via the
   `strings/no-unused-keys` ESLint rule — adding an unused key to
   `en.json` produces a lint error in CI under
   `node_package: { strict: true }`, verified by ephemeral probe.
7. Codegen-freshness (item 8) is enforced via the
   `strings/codegen-fresh` ESLint rule — hand-editing
   `strings.generated.ts` to break sync produces a lint error in CI,
   verified by ephemeral probe.
8. The underscore-prefix sentinel convention (Phase 1 item 5) is
   enforced consistently across all four touchpoints — verified by
   the sentinel-skip probe described in Phase 4 step 14: a
   `_lint_probe` key added to `en.json` is excluded from
   `strings.generated.ts`, doesn't trip parity, doesn't trip
   `no-unused-keys`, and doesn't trip `codegen-fresh`. The unit test
   in `gen-strings.test.mjs` is committed and passes.
9. `pnpm lint` runs without the typescript-eslint "Multiple
   projects found" advisory (item 11), via
   `parserOptions.projectService: true`.
10. The plural example (item 6) follows the existing `refreshCount`
    convention — `notifications.summary` is the base form,
    `notifications.summary_zero` and `notifications.summary_one` are
    explicit siblings; ja.json has only the base form. The
    plural-rendering test passes for `count = 0, 1, 5`.
11. `App.tsx` contains no `as` casts and no inline string literals
    in JSX text or banned attributes.
12. Validation scenario was deleted; `git status` is clean.
13. No file outside `templates/scenarios/react-vite/` is modified.
14. The plan does **not** address rollout to existing scenarios;
    that's a follow-up.
