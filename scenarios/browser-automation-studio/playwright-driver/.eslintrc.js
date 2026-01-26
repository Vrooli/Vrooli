module.exports = {
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
    project: './tsconfig.eslint.json',
  },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
  ],
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

    // CRITICAL: Prevents non-null assertion (!) which bypasses null checks
    '@typescript-eslint/no-non-null-assertion': 'error',

    // CRITICAL: Catches operations on 'any' typed values that may crash at runtime
    '@typescript-eslint/no-unsafe-member-access': 'warn',
    '@typescript-eslint/no-unsafe-call': 'warn',
    '@typescript-eslint/no-unsafe-argument': 'warn',
    '@typescript-eslint/no-unsafe-assignment': 'warn',
    '@typescript-eslint/no-unsafe-return': 'warn',

    // Prevents explicit 'any' which disables all type checking for that value
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/explicit-function-return-type': 'warn',
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/no-floating-promises': 'error',
    '@typescript-eslint/no-misused-promises': 'error',
    'no-console': ['warn', { allow: ['warn', 'error'] }],
  },
  ignorePatterns: ['dist', 'node_modules', '*.js'],
};
