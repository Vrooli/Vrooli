import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";

const baseTypeScriptConfig = {
  files: ["**/*.{ts,tsx}"],
  languageOptions: {
    parser: tsParser,
    parserOptions: {
      project: "./tsconfig.json",
      tsconfigRootDir: process.cwd(),
    },
    globals: {
      window: "readonly",
      document: "readonly",
      navigator: "readonly",
      console: "readonly",
      fetch: "readonly",
      localStorage: "readonly",
      sessionStorage: "readonly",
      setTimeout: "readonly",
      clearTimeout: "readonly",
      setInterval: "readonly",
      clearInterval: "readonly",
      requestAnimationFrame: "readonly",
      cancelAnimationFrame: "readonly",
      HTMLElement: "readonly",
      HTMLInputElement: "readonly",
      HTMLTextAreaElement: "readonly",
      HTMLButtonElement: "readonly",
      HTMLDivElement: "readonly",
      HTMLFormElement: "readonly",
      HTMLAnchorElement: "readonly",
      HTMLParagraphElement: "readonly",
      HTMLHeadingElement: "readonly",
      HTMLSpanElement: "readonly",
      HTMLLabelElement: "readonly",
      KeyboardEvent: "readonly",
      MouseEvent: "readonly",
      Event: "readonly",
      CustomEvent: "readonly",
      AbortController: "readonly",
      URL: "readonly",
      URLSearchParams: "readonly",
      FormData: "readonly",
      Blob: "readonly",
      File: "readonly",
      FileReader: "readonly",
      Request: "readonly",
      Response: "readonly",
      Headers: "readonly",
      process: "readonly",
      globalThis: "readonly",
    },
  },
  plugins: {
    "@typescript-eslint": tseslint,
    "react-hooks": reactHooks,
    "react-refresh": reactRefresh,
  },
  rules: {
    ...tseslint.configs.recommended.rules,
    "no-undef": "off",

    "react-hooks/rules-of-hooks": "error",
    "react-hooks/exhaustive-deps": "warn",

    "@typescript-eslint/no-non-null-assertion": "error",
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unused-vars": [
      "error",
      { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
    ],

    "react-refresh/only-export-components": [
      "warn",
      { allowConstantExport: true },
    ],
  },
};

const testOverrides = {
  files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}", "**/test-setup.ts"],
  languageOptions: {
    globals: {
      afterAll: "readonly",
      afterEach: "readonly",
      beforeAll: "readonly",
      beforeEach: "readonly",
      describe: "readonly",
      expect: "readonly",
      it: "readonly",
      test: "readonly",
      vi: "readonly",
    },
  },
  rules: {
    "@typescript-eslint/no-explicit-any": "off",
    "@typescript-eslint/no-unused-vars": "off",
    "@typescript-eslint/no-non-null-assertion": "off",
    "react-refresh/only-export-components": "off",
  },
};

export default [
  {
    ignores: [
      "dist",
      "node_modules",
      "coverage",
      "tailwind.config.ts",
      "vite.config.ts",
      "**/*.backup",
      "**/*.backup2",
    ],
  },
  js.configs.recommended,
  baseTypeScriptConfig,
  testOverrides,
];
