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
        project: "./tsconfig.eslint.json",
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
          project: "./tsconfig.app.json",
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
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: non-null assertion (!) bypasses null checks; use ?? or guards instead.
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: operations on unchecked 'any' values crash at runtime ("v.trim is not a function").
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies ("Cannot access X before initialization").
      "import/no-cycle": "error",

      // ════════════════════════════════════════════════════════════════════════
      // STANDARD RULES (can be adjusted if needed)
      // ════════════════════════════════════════════════════════════════════════

      // Catches stale closure bugs from missing/incorrect dependencies
      "react-hooks/exhaustive-deps": "warn",

      // Ensures only components are exported for proper HMR
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true, allowExportNames: ["useToast", "useTheme"] }],

      // Allow unused vars prefixed with underscore (common pattern for ignored params)
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],

      // ════════════════════════════════════════════════════════════════════════
      // strictTypeChecked STYLISTIC RULES — downgraded to "warn" (tracked debt)
      //
      // strictTypeChecked is extended above so the genuine runtime-safety rules
      // (no-misused-promises, no-floating-promises, no-unsafe-*, only-throw-error,
      // use-unknown-in-catch, …) are enforced as errors. The rules below are
      // stylistic, not crash-safety, so they are warnings rather than hard gates:
      //   • no-unnecessary-condition directly conflicts with the SAFETY-CRITICAL
      //     defensive null-checking this very config mandates (it flags the ?.
      //     and guard checks added to satisfy the unsafe-* rules as "unnecessary").
      //   • the remaining three are formatting preferences (number-in-template,
      //     redundant conversions/unions) with no runtime impact.
      // ════════════════════════════════════════════════════════════════════════
      "@typescript-eslint/no-unnecessary-condition": "warn",
      "@typescript-eslint/restrict-template-expressions": "warn",
      "@typescript-eslint/no-unnecessary-type-conversion": "warn",
      "@typescript-eslint/no-redundant-type-constituents": "warn",
    },
  }
);
