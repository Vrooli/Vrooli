import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

const browserGlobals = {
  window: "readonly",
  document: "readonly",
  navigator: "readonly",
  console: "readonly",
  fetch: "readonly",
  URL: "readonly",
  URLSearchParams: "readonly",
  setTimeout: "readonly",
  clearTimeout: "readonly",
  setInterval: "readonly",
  clearInterval: "readonly",
  requestAnimationFrame: "readonly",
  cancelAnimationFrame: "readonly",
  localStorage: "readonly",
  sessionStorage: "readonly",
  HTMLElement: "readonly",
  HTMLInputElement: "readonly",
  HTMLDivElement: "readonly",
  HTMLButtonElement: "readonly",
  Event: "readonly",
  EventTarget: "readonly",
  CustomEvent: "readonly",
  MouseEvent: "readonly",
  KeyboardEvent: "readonly",
  FormData: "readonly",
  Blob: "readonly",
  File: "readonly",
  FileReader: "readonly",
  AbortController: "readonly",
  AbortSignal: "readonly",
  Headers: "readonly",
  Request: "readonly",
  Response: "readonly",
  location: "readonly",
  history: "readonly",
  alert: "readonly",
  confirm: "readonly",
  prompt: "readonly",
  globalThis: "readonly",
  process: "readonly",
  Buffer: "readonly",
  __dirname: "readonly",
  __filename: "readonly",
  module: "readonly",
  require: "readonly",
  queueMicrotask: "readonly",
  structuredClone: "readonly",
  performance: "readonly",
};

export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
    files: ["src/**/*.{ts,tsx}"],
    extends: [tseslint.configs.strictTypeChecked],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: "module",
      globals: browserGlobals,
      parserOptions: {
        ecmaFeatures: { jsx: true },
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
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
      // CRITICAL: Catches React Error #310 (hook count changes between renders).
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Catches stale-closure bugs when dependencies drift.
      "react-hooks/exhaustive-deps": "warn",
      // CRITICAL: Prevents explicit any from disabling type safety.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Prevents non-null assertions from bypassing null checks.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Catches unsafe arguments flowing into typed APIs.
      "@typescript-eslint/no-unsafe-argument": "warn",
      // CRITICAL: Catches unchecked values spreading any through the UI.
      "@typescript-eslint/no-unsafe-assignment": "warn",
      // CRITICAL: Catches calls on unchecked values that can crash at runtime.
      "@typescript-eslint/no-unsafe-call": "warn",
      // CRITICAL: Catches member access on unchecked values that can crash.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      // CRITICAL: Catches unsafe values returned across component boundaries.
      "@typescript-eslint/no-unsafe-return": "warn",
      "@typescript-eslint/no-deprecated": "off",
      "@typescript-eslint/no-dynamic-delete": "off",
      "@typescript-eslint/prefer-reduce-type-parameter": "off",
      "@typescript-eslint/require-await": "off",
      // CRITICAL: Detects circular dependencies causing initialization failures.
      "import/no-cycle": "error",
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_", varsIgnorePattern: "^_", caughtErrorsIgnorePattern: "^_|^err$" }],
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
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            { group: ["**/test-utils/**"], message: "Production code must not import test utilities." },
            { group: ["**/features/*/mocks/**"], message: "Production code must not import feature mocks." },
          ],
        },
      ],
      "@typescript-eslint/no-empty-object-type": "off",
      "no-undef": "off",
      "no-empty-pattern": "off",
      "no-empty": ["error", { allowEmptyCatch: true }],
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: "./tsconfig.json",
        },
      },
    },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}", "**/__tests__/**/*.{ts,tsx}"],
    rules: {
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-unused-vars": "off",
      "no-restricted-imports": "off",
      "react-refresh/only-export-components": "off",
    },
  },
);
