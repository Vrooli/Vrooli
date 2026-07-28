import js from "@eslint/js";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage", "vite.config.ts", "vitest.coverage.config.ts", "vite.config.ts.timestamp-*"] },
  // Main source files - with type-aware linting
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.{ts,tsx}"],
    ignores: ["**/*.test.{ts,tsx}", "**/__tests__/**", "src/test-setup.ts", "src/test-utils/**"],
    languageOptions: {
      parserOptions: {
        project: "./tsconfig.json",  // Enable type-aware linting
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
          project: "./tsconfig.json",
        },
      },
    },
    rules: {
	  // Test helpers and feature mocks are test-only infrastructure. Production
	  // modules must depend on product interfaces, never test substitutions.
      "no-restricted-imports": ["error", {
        patterns: [{
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
        }],
      }],
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
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      // Using ! hides bugs that will crash at runtime with "X is not a function"
      // Instead of arr[0]!, use: arr[0] ?? defaultValue or if (arr[0]) { ... }
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      // CRITICAL: unsafe member access can invoke methods that are absent at runtime.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      // CRITICAL: unsafe calls can execute non-callable values from untrusted data.
      "@typescript-eslint/no-unsafe-call": "warn",
      // CRITICAL: unsafe arguments bypass the receiving API's runtime contract.
      "@typescript-eslint/no-unsafe-argument": "warn",
      // CRITICAL: unsafe assignments hide unvalidated values at the boundary.
      "@typescript-eslint/no-unsafe-assignment": "warn",
      // CRITICAL: unsafe returns leak unvalidated values to downstream callers.
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Detects circular dependencies that produce initialization-order failures.
      "import/no-cycle": "error",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // ════════════════════════════════════════════════════════════════════════
      // STANDARD RULES (can be adjusted if needed)
      // ════════════════════════════════════════════════════════════════════════

      // Catches stale closure bugs from missing/incorrect dependencies
      "react-hooks/exhaustive-deps": "warn",

      // Ensures only components are exported for proper HMR
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],

      // Allow unused vars prefixed with underscore (common pattern for ignored params)
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" }],
    },
  },
  // Test files - without type-aware linting (excluded from tsconfig.json)
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ["**/*.test.{ts,tsx}", "**/__tests__/**/*.{ts,tsx}", "src/test-utils/**/*.{ts,tsx}"],
    plugins: {
      "react-hooks": reactHooks,
    },
    rules: {
      // Test files can be more lenient with type safety
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/no-non-null-assertion": "warn",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_" }],
      "react-hooks/rules-of-hooks": "error",
    },
  }
);
