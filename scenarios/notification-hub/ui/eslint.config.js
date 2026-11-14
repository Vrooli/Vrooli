import js from "@eslint/js";
import globals from "globals";
import importPlugin from "eslint-plugin-import";
import jsxA11y from "eslint-plugin-jsx-a11y";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";
import stringsPlugin from "./eslint-rules/index.js";

/** Copy that reaches the DOM only through JSX text, expression strings or string attributes. */
const JSX_COPY_SELECTORS = [
  {
    selector: "JSXText[value=/[a-zA-Z]/]",
    message:
      "User-facing strings must go through the i18n registry. Reference them as `{t(strings.feature.key)}` instead of inlining JSX text.",
  },
  {
    selector: "JSXExpressionContainer > Literal[value=/[a-zA-Z]/]",
    message:
      "User-facing strings must go through the i18n registry. Reference them as `{t(strings.feature.key)}` instead of `{\"...\"}`.",
  },
  {
    selector:
      "JSXAttribute[name.name=/^(aria-label|aria-description|aria-placeholder|aria-roledescription|aria-valuetext|placeholder|title|alt|label)$/] > Literal[value=/[a-zA-Z]/]",
    message:
      "User-facing string attributes must go through the i18n registry. Use `aria-label={t(strings.feature.key)}` (etc.) instead of a string literal.",
  },
];

/**
 * English prose written straight into code: a string with at least one common
 * English function word. Class names, test ids, keys and selectors never carry
 * one, so this catches copy that should be in the registry (`const COPY = {
 * en: "Save changes" }`, `t(key, { defaultValue: "…" })`) without a word list
 * of the product's own vocabulary. Messages of `new Error(...)` are developer
 * text and are exempt. A single word, such as "Retry", is not detectable here
 * and still relies on review.
 */
const PROSE = String.raw`/(^|\s)(the|and|in|of|for|to|is|are|was|your|you|this|that|with|from|not|could|can|now|on|has|have|will|be)(\s|[.,;:!?]|$)/`;
const PROSE_SELECTORS = [
  {
    selector: `Literal[value=${PROSE}]:not(NewExpression > Literal)`,
    message: "English prose in code must live in src/i18n/locales/<locale>/*.json and be read through `t(strings.feature.key)`.",
  },
  {
    selector: `TemplateElement[value.cooked=${PROSE}]:not(NewExpression TemplateElement)`,
    message: "English prose in code must live in src/i18n/locales/<locale>/*.json and be read through `t(strings.feature.key)`.",
  },
];

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage", "src/**/flow/generated/**"] },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.strictTypeChecked,
      // jsx-a11y catches lint-time accessibility issues that complement the
      // axe-core runtime checks in *.a11y.test.tsx — missing alt text, buttons
      // without accessible names, invalid ARIA props, etc. Strict (not just
      // recommended) because every scenario UI is a product surface that must
      // pass WCAG; weakening this for individual rules is preferable to
      // dropping the whole ruleset.
      jsxA11y.flatConfigs.strict,
    ],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        // `projectService: true` is the typescript-eslint v8+ canonical
        // way to pick up TS project info without listing each tsconfig
        // explicitly. Avoids the "Multiple projects found, consider using
        // a single tsconfig with references" advisory that the older
        // explicit `project: [...]` array shape produces.
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
      strings: stringsPlugin,
    },
    settings: {
      "import/resolver": {
        // `eslint-import-resolver-typescript` follows TS project references
        // automatically when given a single root tsconfig (tsconfig.json
        // already references tsconfig.node.json), so we point at the root
        // and let the resolver walk references. Listing both explicitly
        // produces the same "Multiple projects found" advisory we silence
        // on the parser side via `projectService: true`.
        typescript: {
          alwaysTryTypes: true,
          project: "./tsconfig.json",
        },
      },
    },
    rules: {
      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules prevent runtime crashes. If you encounter errors:
      // ✅ DO: Fix the code with optional chaining (?.), null checks, or proper types
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Catches stale-closure bugs when dependencies drift from actual usage.
      "react-hooks/exhaustive-deps": "warn",

      // CRITICAL: Prevents explicit 'any' from disabling type safety at UI boundaries.
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Prevents non-null assertion (!) from bypassing TypeScript null checks.
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches unsafe arguments flowing from unchecked values into typed APIs.
      "@typescript-eslint/no-unsafe-argument": "warn",

      // CRITICAL: Catches assigning unchecked values that spread `any` through the codebase.
      "@typescript-eslint/no-unsafe-assignment": "warn",

      // CRITICAL: Catches invoking unchecked values that will crash at runtime.
      "@typescript-eslint/no-unsafe-call": "warn",

      // CRITICAL: Catches member access on unchecked values that will crash at runtime.
      "@typescript-eslint/no-unsafe-member-access": "warn",

      // CRITICAL: Catches returning unchecked values that leak unsafe types to callers.
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Detects circular dependencies that produce initialization-order failures.
      "import/no-cycle": "error",

      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",

      // ════════════════════════════════════════════════════════════════════════
      // STRING REGISTRY ENFORCEMENT
      //
      // User-facing strings live in src/i18n/locales/<locale>/<namespace>.json and are
      // referenced through the typed `strings.*` registry in src/consts/strings.ts.
      // The registry derives dotted key paths from the English catalog at build/load time,
      // so `strings.app.titel` is a TypeScript error and renames in the catalog
      // produce errors at every callsite.
      //
      // The rules below ensure the registry stays the *only* path that copy
      // takes into the DOM. They cover three classes of leak:
      //
      //   1. JSX text             — `<h1>Hello</h1>`
      //   2. JSX expression body  — `<h1>{"Hello"}</h1>`
      //   3. JSX string attribute — `aria-label="Language"`, `placeholder="…"`,
      //                              `title="…"`, `alt="…"`
      //
      // If you hit one of these rules:
      //   ✅ DO: add the string to its namespace file under
      //         src/i18n/locales/en/ (and every other locale), then reference it as `{t(strings.feature.key)}`.
      //         For interpolation, use i18next's `{{var}}` placeholders in
      //         the JSON and call `t(strings.feature.key, { var: value })`.
      //   ❌ DON'T: disable the rule, or work around it with a template literal
      //         (`{`Hello ${name}`}`) — that hides the string from translators.
      //
      // Note: regex matchers like `placeholder:text-white/40` are Tailwind
      // class names (string content of `className`), not the HTML `placeholder`
      // attribute, so they are not affected by rule #3.
      // ════════════════════════════════════════════════════════════════════════
      // String registry custom rules — see eslint-rules/index.js for the
      // contract. Both rules anchor on `src/consts/strings.generated.ts` and
      // emit exactly once per lint pass.
      "strings/codegen-fresh": "error",
      "strings/no-unused-keys": "error",

      "no-restricted-syntax": ["error", ...JSX_COPY_SELECTORS, ...PROSE_SELECTORS],

      // ════════════════════════════════════════════════════════════════════════
      // TEST-UTILS QUARANTINE (mirrors api/internal/testutil/no_prod_import_test.go)
      //
      // src/test-utils/ AND every per-feature `mocks/` sub-folder hold
      // test-only scaffolding — fakes that mutate process-wide state,
      // factories with arbitrary defaults, render helpers that wire mock
      // providers. None of it should ever ship in a production bundle.
      //
      // Without this guardrail, a single accidental `import { makeHealthResponse }
      // from "@/test-utils"` in production code drags the whole tree (including
      // any future test-only deps) into the build. The Go side enforces this at
      // the AST level via no_prod_import_test.go; this is its TS counterpart.
      //
      // Co-located mocks live under `features/<dom>/mocks/` (Pass-3 pattern:
      // domain-specific test doubles ship with the feature so deletion is
      // folder-scoped). Production code under `features/<dom>/` never imports
      // from its own `mocks/` sub-folder; only the matching test files do.
      //
      // The patterns cover both relative paths and the `@/` alias. The override
      // block below for *.test.{ts,tsx} / *.spec.{ts,tsx} turns the rule off so
      // tests can — and should — import freely from these locations.
      //
      // If you hit this rule:
      //   ✅ DO: confirm the import is in a test file (`*.test.{ts,tsx}` /
      //         `*.spec.{ts,tsx}`). If yes, it's already exempt — your filename
      //         may not match the override pattern.
      //   ✅ DO: if you genuinely need a helper in production code, move it out
      //         of test-utils (or features/<dom>/mocks/) into an appropriate
      //         non-test location.
      //   ❌ DON'T: disable the rule for a "one-off" production import. There
      //         is no path back from a test-utils leak — every future build
      //         carries it.
      // ════════════════════════════════════════════════════════════════════════
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: [
                "**/test-utils",
                "**/test-utils/*",
                "@/test-utils",
                "@/test-utils/*",
                "**/features/*/mocks",
                "**/features/*/mocks/*",
                "@/features/*/mocks",
                "@/features/*/mocks/*",
              ],
              message:
                "Production code must not import from src/test-utils/ or any features/<dom>/mocks/ — these helpers are test-only. Move shared helpers out of those locations, or move the importing file to *.test.{ts,tsx}.",
            },
          ],
        },
      ],
    },
  },
  {
    // Developer-facing descriptions, test fixtures and the sealed explainer
    // kit are not operator copy.
    files: ["src/consts/selectors*.ts", "src/**/mocks/**", "src/features/packet/explainer/kit/**"],
    rules: { "no-restricted-syntax": ["error", ...JSX_COPY_SELECTORS] },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-non-null-assertion": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      "react-refresh/only-export-components": "off",
      // Tests are the consumers of src/test-utils/ — the production-side ban
      // (in the main config block) is precisely what makes this directory
      // safe to put fakes, factories, and provider plumbing in.
      "no-restricted-imports": "off",

      // ════════════════════════════════════════════════════════════════════════
      // TEST STABILITY ENFORCEMENT
      //
      // Banning string and template literals as the first argument to every
      // copy- or label-driven Testing Library query forces tests through one
      // of three load-bearing patterns:
      //
      //   1. `getByTestId(selectors.x.y)`   — copy-independent, structure-
      //      independent. The default for "find this element."
      //   2. `getByText(strings.x.y)`       — combined with cimode (default
      //      via test-setup.ts), `t()` returns the key, so this is a typed,
      //      copy-independent assertion that the right *key* renders.
      //      The same pattern works with the role-by-name shape:
      //      `getByRole(role, { name: t(strings.x.y) })`.
      //   3. `getByText(en.x.y)`            — explicit real-locale tests
      //      (validating the i18n pipeline end-to-end). MemberExpressions
      //      pass this rule; only string and template literals are banned.
      //
      // Regex matchers (`getByText(/loading/i)`) remain allowed — they're
      // explicit pattern matchers, not exact-string assertions.
      //
      // Coverage: every copy-driven *By* family from Testing Library —
      // ByText, ByLabelText, ByPlaceholderText, ByTitle, ByAltText,
      // ByDisplayValue. ByRole and ByTestId are NOT covered: ByRole takes
      // a role name (not copy), and ByTestId is the canonical escape hatch
      // we want authors to use.
      //
      // If you hit this rule:
      //   ✅ DO: replace with `selectors.x.y` (test ID), `strings.x.y` (key),
      //         or `getByRole(role, { name: t(strings.x.y) })` for accessible
      //         names.
      //   ❌ DON'T: disable the rule. The whole point is that tests don't
      //         silently break when copy changes — that's a test failure
      //         every time copy moves, not a real regression.
      // ════════════════════════════════════════════════════════════════════════
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "CallExpression[callee.property.name=/^(get|find|query|getAll|findAll|queryAll)By(Text|LabelText|PlaceholderText|Title|AltText|DisplayValue)$/] > Literal[value=/[a-zA-Z]/]:first-child",
          message:
            "Don't pass string literals to copy-driven Testing Library queries. Use getByTestId(selectors.x.y), getByText(strings.x.y) (with the cimode default), or getByRole(role, { name: t(strings.x.y) }) so tests survive copy changes.",
        },
        {
          selector:
            "CallExpression[callee.property.name=/^(get|find|query|getAll|findAll|queryAll)By(Text|LabelText|PlaceholderText|Title|AltText|DisplayValue)$/] > TemplateLiteral:first-child",
          message:
            "Don't pass template literals to copy-driven Testing Library queries. Use getByTestId(selectors.x.y), getByText(strings.x.y) (with the cimode default), or getByRole(role, { name: t(strings.x.y) }) so tests survive copy changes.",
        },
      ],
    },
  }
);
