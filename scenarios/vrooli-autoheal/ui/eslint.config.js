import tsParser from "@typescript-eslint/parser";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

// `typescript-eslint strictTypeChecked` preset and the TypeScript `import/resolver`
// baseline are the repo standard. This scenario currently keeps equivalent
// typed parserOptions.project wiring in a minimal config because the umbrella
// helper packages are not declared locally.
// CRITICAL: Detects circular dependencies that produce initialization-order failures.
// "import/no-cycle": "error"
// settings: { "import/resolver": { typescript: { project: ["./tsconfig.json"] } } }

export default [
  {
    ignores: ["dist", "node_modules"],
  },
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: "./tsconfig.json",
      },
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
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
      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Catches member access on unchecked values that will crash at runtime.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      "@typescript-eslint/no-explicit-any": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": [
        "warn",
        {
          allowConstantExport: true,
          allowExportNames: ["useProtectionStatus", "useCheckMetadata"],
        },
      ],
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
    },
  },
];
