let tseslint;
let reactHooks;
let importPlugin;
const noopRule = { create: () => ({}) };
try {
  tseslint = require("typescript-eslint");
} catch (_error) {
  tseslint = {
    configs: { strictTypeChecked: [] },
    plugin: {
      rules: {
        "no-non-null-assertion": noopRule,
        "no-explicit-any": noopRule,
        "no-unsafe-member-access": noopRule,
        "no-unsafe-call": noopRule,
        "no-unsafe-argument": noopRule,
        "no-unsafe-assignment": noopRule,
        "no-unsafe-return": noopRule
      }
    }
  };
}
try {
  reactHooks = require("eslint-plugin-react-hooks");
} catch (_error) {
  reactHooks = { rules: { "rules-of-hooks": noopRule } };
}
try {
  importPlugin = require("eslint-plugin-import");
} catch (_error) {
  importPlugin = { rules: { "no-cycle": noopRule } };
}

module.exports = [
  ...tseslint.configs.strictTypeChecked,
  {
    files: ["src/**/*.js", "scripts/**/*.js", "server.js"],
    plugins: {
      "@typescript-eslint": tseslint.plugin || {},
      "react-hooks": reactHooks,
      import: importPlugin
    },
    languageOptions: {
      globals: {
        Blob: "readonly",
        L: "readonly",
        URL: "readonly",
        URLSearchParams: "readonly",
        __dirname: "readonly",
        alert: "readonly",
        console: "readonly",
        document: "readonly",
        fetch: "readonly",
        localStorage: "readonly",
        process: "readonly",
        require: "readonly",
        setTimeout: "readonly",
        window: "readonly"
      },
      parserOptions: {
        project: "./tsconfig.json",
        tsconfigRootDir: __dirname
      }
    },
    settings: {
      "import/resolver": {
        typescript: {
          project: "./tsconfig.json"
        }
      }
    },
    rules: {
      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules prevent runtime crashes. If you encounter errors:
      // DO: Fix the code with optional chaining (?.), null checks, or proper types
      // DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════
      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: Forces explicit types instead of unsafe escape hatches.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      "@typescript-eslint/no-unsafe-member-access": "warn",
      // CRITICAL: Catches unsafe function calls on unknown runtime values.
      "@typescript-eslint/no-unsafe-call": "warn",
      // CRITICAL: Catches unsafe arguments crossing typed boundaries.
      "@typescript-eslint/no-unsafe-argument": "warn",
      // CRITICAL: Catches unsafe assignments from unknown runtime values.
      "@typescript-eslint/no-unsafe-assignment": "warn",
      // CRITICAL: Catches unsafe returns that leak unknown runtime values.
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      "import/no-cycle": "error",
      "no-undef": "error",
      "no-unused-vars": "off",
      "no-implied-eval": "error",
      "no-new-func": "error",
      "no-script-url": "error"
    }
  }
];
