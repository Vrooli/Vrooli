module.exports = {
    extends: ['../../.eslintrc'],
    parserOptions: {
        ecmaVersion: 2018,
        sourceType: 'module',
        project: ['./tsconfig.test.json']
    },
    rules: {
        // Test-specific rule overrides
        '@typescript-eslint/no-explicit-any': 'off',
        '@typescript-eslint/no-non-null-assertion': 'off',
        '@typescript-eslint/ban-ts-comment': 'off',
        // Allow console.log in tests
        'no-console': 'off',
        // Allow unused variables in test files (e.g., for mocking)
        '@typescript-eslint/no-unused-vars': 'off',
        // Allow require() in test files for dynamic imports
        '@typescript-eslint/no-var-requires': 'off',
        // Allow magic numbers in tests
        'no-magic-numbers': 'off',
        // Allow function expressions in tests
        'func-style': 'off'
    },
    env: {
        node: true,
        es6: true,
        jest: true
    }
};