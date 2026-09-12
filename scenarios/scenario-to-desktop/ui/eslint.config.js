import js from "@eslint/js";
import importPlugin from "eslint-plugin-import";
import globals from "globals";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
    files: ["**/*.{ts,tsx}"],
    extends: [
      js.configs.recommended,
      ...tseslint.configs.strictTypeChecked,
    ],
    languageOptions: {
      globals: {
        ...globals.browser,
      },
      parserOptions: {
        project: "./tsconfig.json",
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
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: rules-of-hooks catches React Error #310 (hook count changes between renders).
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: no-non-null-assertion prevents non-null assertions that bypass TypeScript's checks.
      // Using ! hides bugs that will crash at runtime with "X is not a function"
      // Instead of arr[0]!, use: arr[0] ?? defaultValue or if (arr[0]) { ... }
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      // CRITICAL: unsafe member access must remain type-aware.
      "@typescript-eslint/no-unsafe-member-access": "error",
      // CRITICAL: unsafe calls must remain type-aware.
      "@typescript-eslint/no-unsafe-call": "error",
      // CRITICAL: unsafe arguments must remain type-aware.
      "@typescript-eslint/no-unsafe-argument": "error",
      // CRITICAL: unsafe assignments must remain type-aware.
      "@typescript-eslint/no-unsafe-assignment": "error",
      // CRITICAL: unsafe returns must remain type-aware.
      "@typescript-eslint/no-unsafe-return": "error",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: no-cycle detects circular dependencies that cause "Cannot access X before initialization".
      // These runtime errors are extremely hard to debug in production (minified variable names).
      // Requires eslint-plugin-import and eslint-import-resolver-typescript
      "import/no-cycle": "error",

      // ════════════════════════════════════════════════════════════════════════
      // STANDARD RULES (can be adjusted if needed)
      // ════════════════════════════════════════════════════════════════════════

      // Catches stale closure bugs from missing/incorrect dependencies
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
    },
  },
  {
    files: ["src/**/*.{ts,tsx}"],
    ignores: ["src/**/*.test.{ts,tsx}", "src/**/*.spec.{ts,tsx}", "src/test-utils/**", "src/test-setup.ts"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["**/test-utils/**", "**/features/*/mocks/**"],
              message: "Production modules must not import test helpers or feature mocks.",
            },
          ],
        },
      ],
    },
  },
  {
    // Test doubles intentionally expose async-shaped interfaces and callbacks
    // without exercising every runtime branch. Production modules retain the
    // full strict type-aware profile above.
    files: [
      "src/**/*.test.{ts,tsx}",
      "src/**/*.spec.{ts,tsx}",
      "src/test-utils/**/*.{ts,tsx}",
      "src/test-setup.ts",
    ],
    rules: {
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/no-misused-promises": "off",
      "@typescript-eslint/require-await": "off",
      "@typescript-eslint/unbound-method": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-deprecated": "off",
      "@typescript-eslint/no-misused-spread": "off",
      "@typescript-eslint/no-dynamic-delete": "off",
    },
  },
  {
    // Ambient declarations model third-party APIs; their overloads are not
    // executable application code and should not be reshaped by lint rules.
    files: ["src/**/*.d.ts"],
    rules: {
      "@typescript-eslint/unified-signatures": "off",
    },
  },
);
