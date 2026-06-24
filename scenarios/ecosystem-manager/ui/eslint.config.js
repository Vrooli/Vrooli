import js from '@eslint/js';
import globals from 'globals';
import importPlugin from 'eslint-plugin-import';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  {
    ignores: ['dist', 'coverage', 'node_modules', 'tailwind.config.ts', 'vite.config.ts', 'server.js'],
  },
  {
    extends: [js.configs.recommended, ...tseslint.configs.strictTypeChecked],
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: {
        ...globals.browser,
        ...globals.es2021,
        ...globals.node,
      },
      parserOptions: {
        // `projectService: true` is the typescript-eslint v8+ canonical way to
        // pick up TS project info without listing each tsconfig explicitly.
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      import: importPlugin,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    settings: {
      'import/resolver': {
        // eslint-import-resolver-typescript follows TS project references when
        // given the root tsconfig; required for the circular-import rule to resolve.
        typescript: {
          alwaysTryTypes: true,
          project: './tsconfig.json',
        },
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
      'react-hooks/rules-of-hooks': 'error',

      // CRITICAL: Catches stale-closure bugs when dependencies drift from actual usage.
      'react-hooks/exhaustive-deps': 'warn',

      // CRITICAL: Prevents explicit 'any' from disabling type safety at UI boundaries.
      '@typescript-eslint/no-explicit-any': 'error',

      // CRITICAL: Prevents non-null assertion (!) from bypassing TypeScript null checks.
      '@typescript-eslint/no-non-null-assertion': 'error',

      // CRITICAL: Catches unsafe arguments flowing from unchecked values into typed APIs.
      '@typescript-eslint/no-unsafe-argument': 'warn',

      // CRITICAL: Catches assigning unchecked values that spread `any` through the codebase.
      '@typescript-eslint/no-unsafe-assignment': 'warn',

      // CRITICAL: Catches invoking unchecked values that will crash at runtime.
      '@typescript-eslint/no-unsafe-call': 'warn',

      // CRITICAL: Catches member access on unchecked values that will crash at runtime.
      '@typescript-eslint/no-unsafe-member-access': 'warn',

      // CRITICAL: Catches returning unchecked values that leak unsafe types to callers.
      '@typescript-eslint/no-unsafe-return': 'warn',

      // CRITICAL: Detects circular dependencies that produce initialization-order failures.
      'import/no-cycle': 'error',

      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      '@typescript-eslint/restrict-template-expressions': 'off',
      '@typescript-eslint/no-confusing-void-expression': 'off',
      'no-undef': 'off',
    },
  },
  {
    files: ['src/**/*.test.{ts,tsx}', 'src/**/*.spec.{ts,tsx}'],
    rules: {
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-unsafe-call': 'off',
      '@typescript-eslint/no-unsafe-member-access': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-unsafe-return': 'off',
      'react-refresh/only-export-components': 'off',
    },
  }
);
