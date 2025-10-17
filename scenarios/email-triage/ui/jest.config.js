module.exports = {
    testEnvironment: 'node',
    testMatch: ['**/__tests__/**/*.test.js'],
    collectCoverage: false,
    verbose: true,
    testTimeout: 10000,
    maxWorkers: 1, // Run tests serially to avoid port conflicts
};
