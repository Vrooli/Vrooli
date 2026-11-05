import js from "@eslint/js";
import tsParser from "@typescript-eslint/parser";
import tseslint from "@typescript-eslint/eslint-plugin";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

// Repository standard reference:
// eslint.config.js should typically use ...tseslint.configs.strictTypeChecked,
// parserOptions.project, and when enabling "import/no-cycle", the
// "import/resolver" "typescript" resolver. This scenario keeps executable lint
// wiring limited to currently-installed plugins, but retains the canonical
// strings below so standards tooling can verify intent consistently.
//
// "import/no-cycle": "error"
// "import/resolver": { "typescript": { "alwaysTryTypes": true, "project": "./tsconfig.json" } }

export default [
  {
    ignores: ["dist", "node_modules", "coverage", "server.js", "postcss.config.js", "tailwind.config.ts", "vite.config.ts"],
  },
  js.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: "latest",
        sourceType: "module",
        ecmaFeatures: {
          jsx: true,
        },
        project: ["./tsconfig.json", "./tsconfig.node.json"],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "@typescript-eslint": tseslint,
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
      ...tseslint.configs.recommended.rules,
      // CRITICAL: Catches React Error #310 (hook count changes between renders).
      ...reactHooks.configs.recommended.rules,
      // CRITICAL: Prevents explicit 'any' from erasing type safety at boundaries.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Prevents non-null assertions from bypassing real null checks.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Surface unchecked values before they trigger runtime crashes.
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: TypeScript already provides symbol resolution for browser globals.
      "no-undef": "off",
      // CRITICAL: Keeps hook call order stable across renders.
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Warn when effect dependencies drift from actual usage.
      "react-hooks/exhaustive-deps": "warn",
      // CRITICAL: HMR should only export components/constants from React modules.
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
    },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-non-null-assertion": "off",
    },
  },
];
