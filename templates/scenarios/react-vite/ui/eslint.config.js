import js from "@eslint/js";
import globals from "globals";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: ["./tsconfig.json", "./tsconfig.node.json"],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: ["./tsconfig.json", "./tsconfig.node.json"],
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
      // User-facing strings must live in src/consts/strings.ts so that:
      //   - tests can assert against `strings.x.y` instead of brittle literals,
      //   - copy edits are one-line changes, not haystack greps,
      //   - the registry is ready to swap for a `t()` accessor if/when a
      //     scenario adopts react-i18next for multi-language support.
      //
      // If you hit this rule:
      //   ✅ DO: add the string to src/consts/strings.ts and reference it as
      //         `{strings.feature.key}` (use `format()` for interpolation).
      //   ❌ DON'T: disable the rule, or work around it with a template literal
      //         (`{`Hello ${name}`}`) — that hides the string from the registry.
      //
      // The rule deliberately only catches JSX text and JSXExpressionContainer
      // literals — string attributes (placeholder, aria-label, title, …) are
      // out of scope for Phase 1. Migrate them when a scenario adopts a real
      // i18n library.
      // ════════════════════════════════════════════════════════════════════════
      "no-restricted-syntax": [
        "error",
        {
          selector: "JSXText[value=/[a-zA-Z]/]",
          message:
            "User-facing strings must live in src/consts/strings.ts. Reference them as `{strings.feature.key}` instead of inlining JSX text.",
        },
        {
          selector: "JSXExpressionContainer > Literal[value=/[a-zA-Z]/]",
          message:
            "User-facing strings must live in src/consts/strings.ts. Reference them as `{strings.feature.key}` instead of `{\"...\"}`.",
        },
      ],
    },
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

      // ════════════════════════════════════════════════════════════════════════
      // TEST STABILITY ENFORCEMENT
      //
      // Banning string literals as the first argument to text-based queries
      // (`getByText("Reload")`) forces tests through one of three load-bearing
      // patterns:
      //
      //   1. `getByTestId(selectors.x.y)`   — copy-independent, structure-
      //      independent. The default for "find this element."
      //   2. `getByText(strings.x.y)`       — combined with cimode (default
      //      via test-setup.ts), `t()` returns the key, so this is a typed,
      //      copy-independent assertion that the right *key* renders.
      //   3. `getByText(en.x.y)`            — explicit real-locale tests
      //      (validating the i18n pipeline end-to-end). MemberExpressions
      //      pass this rule; only string and template literals are banned.
      //
      // Regex matchers (`getByText(/loading/i)`) remain allowed — they're
      // explicit pattern matchers, not exact-string assertions.
      //
      // If you hit this rule:
      //   ✅ DO: replace with `selectors.x.y` (test ID) or `strings.x.y` (key).
      //   ❌ DON'T: disable the rule. The whole point is that tests don't
      //         silently break when copy changes — that's a test failure
      //         every time copy moves, not a real regression.
      // ════════════════════════════════════════════════════════════════════════
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "CallExpression[callee.property.name=/^(getByText|findByText|queryByText|getAllByText|findAllByText|queryAllByText)$/] > Literal[value=/[a-zA-Z]/]:first-child",
          message:
            "Don't pass string literals to *ByText queries. Use getByTestId(selectors.x.y) or getByText(strings.x.y) (with the cimode default) so tests survive copy changes.",
        },
        {
          selector:
            "CallExpression[callee.property.name=/^(getByText|findByText|queryByText|getAllByText|findAllByText|queryAllByText)$/] > TemplateLiteral:first-child",
          message:
            "Don't pass template literals to *ByText queries. Use getByTestId(selectors.x.y) or getByText(strings.x.y) (with the cimode default) so tests survive copy changes.",
        },
      ],
    },
  }
);
