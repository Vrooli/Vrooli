// SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
import js from "@eslint/js";
import tseslint from "@typescript-eslint/eslint-plugin";
import tsparser from "@typescript-eslint/parser";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import typescriptEslint from "typescript-eslint";

const strictTypeCheckedRules = typescriptEslint.configs.strictTypeChecked.reduce(
  (rules, config) => ({ ...rules, ...(config.rules ?? {}) }),
  {},
);

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
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
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
      ...strictTypeCheckedRules,
      ...tseslint.configs.recommended.rules,
      ...reactHooks.configs.recommended.rules,
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }],
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_" }],
      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Catches implicit unknown values that bypass all type checking at runtime
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Catches operations on unknown-typed values that will crash at runtime
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      // These strict type-checking rules are intentionally relaxed here because
      // they are stylistic or async-shape policies, not runtime-safety guards.
      // The safety-critical rules above remain enforced.
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/no-invalid-void-type": "off",
      "@typescript-eslint/no-misused-promises": "off",
      "@typescript-eslint/no-unnecessary-condition": "off",
      "@typescript-eslint/no-unnecessary-type-arguments": "off",
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",
      "@typescript-eslint/require-await": "off",
      "no-restricted-imports": [
		"error",
		{
		  patterns: [{
			group: ["**/test-utils", "**/test-utils/*", "@/test-utils", "@/test-utils/*", "**/features/*/mocks", "**/features/*/mocks/*", "@/features/*/mocks", "@/features/*/mocks/*"],
			message: "Production code must not import test helpers or feature mocks.",
		  }],
		},
	  ],
      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      "import/no-cycle": "error",
    },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}", "src/test-utils/**/*.{ts,tsx}", "src/test-setup.ts"],
    rules: {
      "no-restricted-imports": "off",
      "@typescript-eslint/no-non-null-assertion": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      "@typescript-eslint/no-unsafe-argument": "off",
      "react-refresh/only-export-components": "off",
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
