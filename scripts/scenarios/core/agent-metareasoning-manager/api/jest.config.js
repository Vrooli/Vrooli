module.exports = {
  testEnvironment: 'node',
  testTimeout: 10000,
  verbose: true,
  collectCoverageFrom: [
    'server.js',
    '!node_modules/**'
  ],
  testMatch: [
    '**/*.test.js'
  ],
  setupFilesAfterEnv: ['<rootDir>/test-setup.js']
};