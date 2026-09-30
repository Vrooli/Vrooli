module.exports = {
  root: true,
  env: {
    browser: true,
    es2022: true,
    node: true,
  },
  ignorePatterns: ["dist", "node_modules", "*.config.js", "*.config.cjs", "server.js"],
  overrides: [
    {
      files: ["**/*.{ts,tsx}"],
      excludedFiles: ["vite.config.ts", "*.config.ts"],
      parser: "@typescript-eslint/parser",
      parserOptions: {
        ecmaVersion: "latest",
        sourceType: "module",
        ecmaFeatures: {
          jsx: true,
        },
        project: ["./tsconfig.json"],
      },
      plugins: ["@typescript-eslint", "react-hooks", "react-refresh"],
      extends: ["eslint:recommended", "plugin:@typescript-eslint/recommended"],
      rules: {
        // SAFETY-CRITICAL: preserve hook ordering guarantees.
        "react-hooks/rules-of-hooks": "error",
        "@typescript-eslint/no-explicit-any": "off",
        "@typescript-eslint/no-unused-vars": ["error", { "argsIgnorePattern": "^_", "varsIgnorePattern": "^_" }],
      },
    },
  ],
}
