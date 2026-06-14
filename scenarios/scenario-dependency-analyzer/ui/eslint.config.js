import js from "@eslint/js";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname
      }
    },
    plugins: {
      import: importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh
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

      // CRITICAL: Catches React Error #310 (hook count changes between renders).
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Catches stale closures when hook dependencies drift.
      "react-hooks/exhaustive-deps": "warn",

      // CRITICAL: Prevents non-null assertions that bypass TypeScript null checks.
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches member access on unchecked values that will crash.
      "@typescript-eslint/no-unsafe-member-access": "warn",

      // CRITICAL: Catches calls on unchecked values that will crash.
      "@typescript-eslint/no-unsafe-call": "warn",

      // CRITICAL: Catches unchecked values passed into typed APIs.
      "@typescript-eslint/no-unsafe-argument": "warn",

      // CRITICAL: Catches assignments that spread unsafe types through the UI.
      "@typescript-eslint/no-unsafe-assignment": "warn",

      // CRITICAL: Catches unsafe values returned across component boundaries.
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Prevents explicit any from disabling safety checks.
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects cycles that can cause initialization-order failures.
      "import/no-cycle": "error",
      "@typescript-eslint/no-base-to-string": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/no-invalid-void-type": "off",
      "@typescript-eslint/no-meaningless-void-operator": "off",
      "@typescript-eslint/no-misused-promises": "off",
      "@typescript-eslint/no-redundant-type-constituents": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-unnecessary-type-assertion": "off",
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/use-unknown-in-catch-callback-variable": "off",
      "react-refresh/only-export-components": "off"
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: "./tsconfig.json"
        }
      }
    }
  }
);
