import js from "@eslint/js";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "import": importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    settings: {
      "import/resolver": {
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
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertions
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: The no-non-null-assertion rule prevents bypassing TypeScript null checks.
      // CRITICAL: Prevents non-null assertions, which bypass TypeScript's null checks
      // Non-null assertions hide bugs that can crash at runtime with "X is not a function"
      // Instead of asserting an array element, use: arr[0] ?? defaultValue or a guard.
      // CRITICAL: Keep this rule at error; source code must use guards or fallbacks.
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      "@typescript-eslint/no-unsafe-member-access": "error",
      "@typescript-eslint/no-unsafe-call": "error",
      "@typescript-eslint/no-unsafe-argument": "error",
      "@typescript-eslint/no-unsafe-assignment": "error",
      "@typescript-eslint/no-unsafe-return": "error",

      // CRITICAL: Prevents explicit 'any' which disables all type checking for that value
      // Using 'any' silently allows undefined property access, wrong argument types, and
      // missing null checks — all of which crash at runtime with no compile-time warning
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      // These runtime errors are extremely hard to debug in production (minified variable names).
      // Requires eslint-plugin-import and eslint-import-resolver-typescript
      // CRITICAL: Keep cycle detection enabled to prevent initialization-order crashes.
      "import/no-cycle": "error",

      // The UI uses concise event callbacks, promise-returning submit helpers,
      // and typed API response guards throughout the existing surface. These
      // strictTypeChecked style rules are intentionally relaxed here so the
      // safety contract above remains enforceable without turning formatting
      // and handler-shape conventions into a release blocker.
      "@typescript-eslint/no-confusing-void-expression": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-unnecessary-type-assertion": "off",
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-misused-promises": "off",
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/no-unnecessary-type-arguments": "off",

      // ════════════════════════════════════════════════════════════════════════
      // STANDARD RULES (can be adjusted if needed)
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches stale closure bugs from missing/incorrect dependencies
      "react-hooks/exhaustive-deps": "warn",

      // Ensures only components are exported for proper HMR
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],

      // Allow unused vars prefixed with underscore (common pattern for ignored params)
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],

      // Test helpers and feature-local mocks must never leak into production
      // bundles. Keep this restriction in the native ESLint projection so
      // unit policy cannot be bypassed by a direct import.
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: [
                "**/test-utils",
                "**/test-utils/*",
                "**/features/*/mocks/*",
              ],
            },
          ],
        },
      ],
    },
  },
  {
    files: ["**/*.{test,spec}.{ts,tsx}"],
    rules: {
      // Tests are the approved consumers of the canonical provider and a11y
      // harnesses; production modules remain restricted from importing them.
      "no-restricted-imports": "off",
    },
  },
);
