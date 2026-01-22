import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import globals from "globals";

/**
 * ESLint flat config for prompt-manager UI
 *
 * STABILITY CRITICAL RULES (DO NOT REMOVE):
 * - react-hooks/rules-of-hooks: Prevents hook ordering crashes
 * - react-hooks/exhaustive-deps: Prevents stale closure bugs
 * - @typescript-eslint/no-non-null-assertion: Prevents hidden null bugs
 * - @typescript-eslint/no-unsafe-* rules: Prevents operations on 'any' types
 *
 * See react-stability skill Section 0 for rationale.
 */
export default tseslint.config(
  { ignores: ["dist", "node_modules", "*.config.js"] },
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
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      // STABILITY CRITICAL - DO NOT REMOVE
      // These rules prevent guaranteed crash bugs and hidden null issues
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",

      // TypeScript safety rules
      "@typescript-eslint/no-non-null-assertion": "error",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // React refresh for HMR
      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],

      // Allow unused variables prefixed with underscore
      "@typescript-eslint/no-unused-vars": [
        "warn",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }
      ],

      // Relax some strict rules that are too noisy
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",
    },
  }
);
