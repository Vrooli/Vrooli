import eslint from "@eslint/js";
import tseslint from "typescript-eslint";
import globals from "globals";

export default tseslint.config(
  {
    ignores: [
      "node_modules/**",
      "dist/**",
      "dist-dev/**",
      "scripts/**",
      "examples/**",
      "**/__tests__/**",    // Tests excluded (would need Jest types)
      "*.js",               // Exclude JS files (like this config)
      "main.ts",            // Contains template placeholders
      "preload.ts",         // Contains template placeholders
      "splash-preload.ts",  // Contains template placeholders
    ],
  },
  eslint.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  {
    files: ["**/*.ts"],
    languageOptions: {
      ecmaVersion: 2020,
      globals: { ...globals.node, Electron: "readonly" },
      parserOptions: {
        project: "./tsconfig.dev.json",
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      // ════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT DISABLE
      // See: prompt-manager skill read react-stability
      // ════════════════════════════════════════════════════════════════
      "@typescript-eslint/no-non-null-assertion": "error",
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-misused-promises": "error",

      // ════════════════════════════════════════════════════════════════
      // STANDARD RULES
      // ════════════════════════════════════════════════════════════════
      "@typescript-eslint/no-unused-vars": ["error", {
        argsIgnorePattern: "^_",
        varsIgnorePattern: "^_"
      }],
      "@typescript-eslint/no-require-imports": "off",
      "no-void": ["error", { allowAsStatement: true }],
      // Allow numbers in template literals (perfectly valid)
      "@typescript-eslint/restrict-template-expressions": ["error", {
        allowNumber: true,
        allowBoolean: true,
      }],
      // Allow void expressions in arrow shorthand (common pattern)
      "@typescript-eslint/no-confusing-void-expression": ["error", {
        ignoreArrowShorthand: true,
      }],
      // Type assertions are sometimes useful; disable unnecessary check
      "@typescript-eslint/no-unnecessary-type-assertion": "warn",
    },
  },
  // Test file overrides - relaxed rules for mocks
  {
    files: ["**/__tests__/**/*.ts", "**/*.test.ts"],
    rules: {
      "@typescript-eslint/no-explicit-any": "warn",
      "@typescript-eslint/no-unsafe-assignment": "off",
      "@typescript-eslint/no-unsafe-member-access": "off",
      "@typescript-eslint/no-unsafe-call": "off",
      "@typescript-eslint/no-unsafe-return": "off",
      "@typescript-eslint/no-unsafe-argument": "off",
      "@typescript-eslint/no-floating-promises": "off",
      "@typescript-eslint/no-non-null-assertion": "warn",
    },
  }
);
