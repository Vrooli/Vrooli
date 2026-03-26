import js from "@eslint/js";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

export default [
  { ignores: ["dist", "node_modules"] },
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: "./tsconfig.json",
      },
      globals: {
        // Browser globals
        window: "readonly",
        document: "readonly",
        fetch: "readonly",
        confirm: "readonly",
        HTMLElement: "readonly",
        HTMLDivElement: "readonly",
        HTMLInputElement: "readonly",
        HTMLTextAreaElement: "readonly",
        HTMLButtonElement: "readonly",
        URL: "readonly",
        URLSearchParams: "readonly",
        Response: "readonly",
        React: "readonly",
        console: "readonly",
      },
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      "import": importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    settings: {
      "import/resolver": {
        typescript: {
          project: "./tsconfig.json",
        },
      },
    },
    rules: {
      ...js.configs.recommended.rules,
      ...tsPlugin.configs.recommended.rules,

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

      // CRITICAL: Prevents non-null assertion (!) which bypasses null checks
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches member access on 'any' typed values that crash at runtime (e.g. "v.trim is not a function")
      "@typescript-eslint/no-unsafe-member-access": "warn",
      // CRITICAL: Catches function calls on 'any' typed values that crash at runtime
      "@typescript-eslint/no-unsafe-call": "warn",
      // CRITICAL: Catches passing 'any' typed values as arguments, bypassing type checking
      "@typescript-eslint/no-unsafe-argument": "warn",
      // CRITICAL: Catches assigning 'any' typed values, which propagates unchecked types
      "@typescript-eslint/no-unsafe-assignment": "warn",
      // CRITICAL: Catches returning 'any' typed values, which spreads unchecked types to callers
      "@typescript-eslint/no-unsafe-return": "warn",

      // CRITICAL: Prevents explicit 'any' which disables all type checking and allows unsafe operations
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization" — extremely hard to debug in production (minified variable names)
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

      // Allow unused vars prefixed with underscore
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
    },
  },
];
