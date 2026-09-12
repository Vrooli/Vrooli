import tsPlugin from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import importPlugin from "eslint-plugin-import";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";

const strictTypeChecked = tsPlugin.configs["flat/strict-type-checked"];
const hooksRecommended = reactHooks.configs["recommended-latest"];
const refreshRecommended = reactRefresh.configs.vite;

export default [
  {
    files: ["**/*.{ts,tsx}", "*.{ts,tsx}"],
    ignores: ["dist"],
    languageOptions: {
      parser: strictTypeChecked[0]?.languageOptions?.parser ?? tsParser,
      parserOptions: {
        projectService: true
      },
      ecmaVersion: 2020,
      sourceType: "module",
      globals: {
        window: "readonly",
        document: "readonly",
        navigator: "readonly",
        console: "readonly"
      }
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      import: importPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh
    },
    settings: {
      "import/resolver": {
        typescript: true
      }
    },
    rules: {
      ...(hooksRecommended?.rules ?? {}),
      ...(refreshRecommended?.rules ?? {}),
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      // This block is managed by Quality Health as a minimum static-quality baseline.
      // CRITICAL: hooks must be enforced to prevent invalid React execution.
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: non-null assertions bypass runtime safety.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: explicit any hides type contract breaks.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: unsafe member/call/argument/assignment/return checks catch runtime crashes.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: cycles destabilize module initialization.
      "import/no-cycle": "error",
      // Test helpers must never leak into the production bundle.
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            "**/test-utils/**",
            "**/features/*/mocks/**"
          ]
        }
      ]
    }
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}"],
    rules: {
      "no-restricted-imports": "off"
    }
  }
];
