import js from "@eslint/js";
import tseslint from "typescript-eslint";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import importPlugin from "eslint-plugin-import";

export default tseslint.config(
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      "react-hooks": reactHooks,
      "react-refresh": reactRefresh,
      "import": importPlugin,
    },
    settings: {
      "import/resolver": {
        typescript: true,
      },
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

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks — use ?. or ?? instead
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization" at runtime
      "import/no-cycle": "error",

      "react-refresh/only-export-components": [
        "warn",
        { allowConstantExport: true },
      ],
    },
  },
  {
    ignores: ["dist/**", "eslint.config.js", "postcss.config.js", "server.js", "tailwind.config.ts"],
  }
);
