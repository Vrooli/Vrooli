import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { createRequire } from "node:module";
import tsParser from "@typescript-eslint/parser";

const __dirname = dirname(fileURLToPath(import.meta.url));
const require = createRequire(import.meta.url);
const pluginRoot = dirname(require.resolve("@typescript-eslint/eslint-plugin/package.json"));
const { plugin: tsPlugin } = require(join(pluginRoot, "dist/raw-plugin.js"));

const browserGlobals = {
  window: "readonly",
  document: "readonly",
  navigator: "readonly",
  fetch: "readonly",
  Response: "readonly",
  Request: "readonly",
  URLSearchParams: "readonly",
  setTimeout: "readonly",
  FormData: "readonly",
  Headers: "readonly",
  HTMLFormElement: "readonly",
  HTMLButtonElement: "readonly"
};

/** @type {import("eslint").Linter.FlatConfig[]} */
export default [
  {
    ignores: ["dist", "coverage", "node_modules"]
  },
  js.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        ecmaVersion: 2021,
        sourceType: "module",
        tsconfigRootDir: __dirname
      },
      globals: browserGlobals
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh
    },
    rules: {
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": ["warn", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      "@typescript-eslint/consistent-type-imports": "warn",
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-refresh/only-export-components": ["warn", { allowConstantExport: true }]
    }
  }
];
