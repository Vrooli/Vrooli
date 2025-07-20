module.exports = {
    extends: ['./.eslintrc'],
    plugins: ['./src/__test/eslint-rules'],
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
        'func-style': 'off',
        // Test cleanup pattern enforcement
        './src/__test/eslint-rules/test-cleanup-patterns': [
            'warn', // Start with warnings during transition
            {
                enforceCleanupGroups: true,
                enforceValidation: false, // Don't require validation yet during migration
                requireImports: true,
                allowedTables: ['user', 'email', 'session'], // Allow some manual cleanup during migration
            },
        ],
        './src/__test/eslint-rules/factory-usage-patterns': [
            'warn',
            {
                enforceFactoryRegistry: false,
                enforceTrackingSession: false, // Don't enforce during migration
                maxFactoryUsage: 15, // Higher threshold during migration
                requireComposition: false,
            },
        ],
    },
    env: {
        node: true,
        es6: true,
        jest: true
    }
};