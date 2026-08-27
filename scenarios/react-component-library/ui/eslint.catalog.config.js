import js from "@eslint/js";
import globals from "globals";
import importPlugin from "eslint-plugin-import";
import jsxA11y from "eslint-plugin-jsx-a11y";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";
import { designSystem } from "./eslint-rules/index.js";

const catalogTSConfig = process.env.RCL_CATALOG_TSCONFIG;

if (!catalogTSConfig) {
  throw new Error(
    "RCL_CATALOG_TSCONFIG must name the generated catalog TypeScript project; " +
      "run catalog lint through scripts/catalog-conformance.mjs",
  );
}

export default tseslint.config(
  {
    ignores: [
      "dist",
      "node_modules",
      "coverage",
      ".catalog-conformance-*",
    ],
  },
  {
    extends: [
      js.configs.recommended,
      ...tseslint.configs.strictTypeChecked,
      jsxA11y.flatConfigs.strict,
    ],
    files: [
      "../library/{foundations,hooks,services,primitives,components}/**/*.{ts,tsx}",
      "library/{foundations,hooks,services,primitives,components}/**/*.{ts,tsx}",
      "**/.catalog-conformance-*/library/{foundations,hooks,services,primitives,components}/**/*.{ts,tsx}",
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: [catalogTSConfig],
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
      "react-hooks": reactHooks,
      "design-system": designSystem,
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
          project: catalogTSConfig,
        },
      },
    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-non-null-assertion": "error",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/restrict-template-expressions": "off",
      "@typescript-eslint/no-confusing-void-expression": "off",
      "import/no-cycle": "error",

      // The library corpus is already clean of raw dimensions — the Go
      // `tokens` catalog gate has been enforcing this over library/** all
      // along, and reports 0 findings across 331 active sources. The rule is
      // registered here at "error" to keep it that way at authoring time
      // rather than only at gate time, and because a library asset is copied
      // verbatim into every adopting scenario: a raw dimension here is a raw
      // dimension in each of them.
      "design-system/no-raw-dimensions": "error",
    },
  },
);
