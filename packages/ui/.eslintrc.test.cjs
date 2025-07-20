module.exports = {
    extends: ['./.eslintrc'],
    parserOptions: {
        ecmaVersion: 2018,
        sourceType: 'module',
        project: ['./tsconfig.test-all.json']
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
        // React-specific relaxations for tests
        'react/display-name': 'off',
        'react-perf/jsx-no-new-object-as-prop': 'off',
        'react-perf/jsx-no-new-array-as-prop': 'off',
        'react-perf/jsx-no-new-function-as-prop': 'off',
        'react-perf/jsx-no-jsx-as-prop': 'off',
        // A11y relaxations for tests/stories
        'jsx-a11y/no-static-element-interactions': 'off',
        'jsx-a11y/click-events-have-key-events': 'off',
        'jsx-a11y/no-noninteractive-element-interactions': 'off'
    },
    env: {
        node: true,
        es6: true,
        jest: true,
        browser: true
    }
};