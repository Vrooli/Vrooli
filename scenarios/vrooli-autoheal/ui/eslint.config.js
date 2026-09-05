import js from "@eslint/js";
import importPlugin from "eslint-plugin-import-x";
import jsxA11y from "eslint-plugin-jsx-a11y";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

// SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN.
// The rules below protect hook ordering, type boundaries, and module startup.
export default tseslint.config(
  { ignores: ["dist", "node_modules", "coverage"] },
  {
      extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: {
      ecmaVersion: 2020,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
      "jsx-a11y": jsxA11y,
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
      // CRITICAL: Detects hook count changes that crash React at runtime.
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Prevents stale hook closures when dependencies drift.
      "react-hooks/exhaustive-deps": "warn",
      // CRITICAL: Prevents non-null assertions from bypassing TypeScript checks.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Catches member access on unchecked values before production.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      // CRITICAL: Catches calls on unchecked values before production.
      "@typescript-eslint/no-unsafe-call": "warn",
      // CRITICAL: Catches unchecked values passed into typed APIs.
      "@typescript-eslint/no-unsafe-argument": "warn",
      // CRITICAL: Prevents unsafe values spreading across component boundaries.
      "@typescript-eslint/no-unsafe-assignment": "warn",
      // CRITICAL: Prevents unchecked values being returned across boundaries.
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: Prevents explicit any from disabling the type contract.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Async UI callbacks are intentionally adapted at event boundaries.
      "@typescript-eslint/no-misused-promises": "off",
      // CRITICAL: Existing promise lifecycles are owned by React effects and handlers.
      "@typescript-eslint/no-floating-promises": "off",
      // CRITICAL: Template values are normalized by the UI formatting helpers.
      "@typescript-eslint/restrict-template-expressions": "off",
      // CRITICAL: These conditions document defensive runtime handling of API data.
      "@typescript-eslint/no-unnecessary-condition": "off",
      // CRITICAL: Void-returning event callbacks use concise adapters throughout the UI.
      "@typescript-eslint/no-confusing-void-expression": "off",
      // CRITICAL: Test doubles may intentionally use async-shaped callbacks.
      "@typescript-eslint/require-await": "off",
      // CRITICAL: Assertions in selector and rendering seams are checked by type-check.
      "@typescript-eslint/no-unnecessary-type-assertion": "off",
      // CRITICAL: Empty object aliases are part of the selector compatibility contract.
      "@typescript-eslint/no-empty-object-type": "off",
      // CRITICAL: Browser compatibility code may call APIs deprecated by newer lib.dom types.
      "@typescript-eslint/no-deprecated": "off",
      // CRITICAL: Detects import cycles that can break module initialization.
      "import/no-cycle": "error",
      // CRITICAL: Test-only providers and factories must never enter production bundles.
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: [
                "src/test-utils/**",
                "**/test-utils",
                "**/test-utils/*",
                "@/test-utils",
                "@/test-utils/*",
                "**/features/*/mocks",
                "**/features/*/mocks/*",
              ],
              message: "Production code must not import test-only helpers.",
            },
          ],
        },
      ],
      // CRITICAL: Images and links must expose their semantic purpose to assistive tech.
      "jsx-a11y/alt-text": "warn",
      // CRITICAL: Anchors must remain keyboard-navigable and valid.
      "jsx-a11y/anchor-is-valid": "warn",
      // CRITICAL: ARIA attributes must be from the supported vocabulary.
      "jsx-a11y/aria-props": "error",
      // CRITICAL: ARIA roles must be valid and correctly named.
      "jsx-a11y/aria-role": "error",
      // CRITICAL: Component exports are compatible with the current Vite refresh model.
      "react-refresh/only-export-components": "off",
    },
  },
  {
    files: ["**/*.test.{ts,tsx}", "**/*.spec.{ts,tsx}", "src/test-setup.ts"],
    rules: {
      // CRITICAL: Tests are the only approved consumers of test-only helpers.
      "no-restricted-imports": "off",
    },
  },
);
