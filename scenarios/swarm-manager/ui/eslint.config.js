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
      "import": importPlugin,
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

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks.
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
      "@typescript-eslint/no-misused-promises": [
        "error",
        {
          checksVoidReturn: {
            attributes: false,
          },
        },
      ],
      "@typescript-eslint/no-deprecated": "off",
      "@typescript-eslint/no-dynamic-delete": "off",
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/no-invalid-void-type": "off",
      "@typescript-eslint/no-meaningless-void-operator": "off",
      "@typescript-eslint/no-redundant-type-constituents": "off",
      "@typescript-eslint/no-unnecessary-type-arguments": "off",
      "@typescript-eslint/no-unnecessary-type-parameters": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",
      "@typescript-eslint/unbound-method": "off",
      "@typescript-eslint/use-unknown-in-catch-callback-variable": "off",
    },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
    rules: {
      "@typescript-eslint/await-thenable": "off",
      "@typescript-eslint/no-base-to-string": "off",
      "@typescript-eslint/no-non-null-assertion": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/unbound-method": "off",
      "react-refresh/only-export-components": "off",
    },
  }
);
