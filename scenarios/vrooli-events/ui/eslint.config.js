// SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
import js from "@eslint/js";
import tseslint from "@typescript-eslint/eslint-plugin";
import tsparser from "@typescript-eslint/parser";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

export default [
  js.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 2020,
        sourceType: "module",
        ecmaFeatures: { jsx: true },
        project: "./tsconfig.json",
      },
      globals: {
        fetch: "readonly",
        URLSearchParams: "readonly",
        EventSource: "readonly",
        Event: "readonly",
        HTMLButtonElement: "readonly",
        React: "readonly",
        console: "readonly",
        window: "readonly",
        document: "readonly",
      },
    },
    plugins: {
      "@typescript-eslint": tseslint,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
    },
    rules: {
      ...tseslint.configs.recommended.rules,
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],
      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Catches implicit 'any' values that bypass all type checking at runtime
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      // "import/no-cycle": "error", // TODO: install eslint-plugin-import to enable
    },
  },
  {
    files: ["server.js"],
    languageOptions: {
      globals: { process: "readonly" },
    },
  },
  { ignores: ["dist/", "node_modules/"] },
];
