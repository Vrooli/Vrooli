// Ordinary unit coverage owns mocked contracts only. Native/live checks are explicit.
const common = require('./jest.common.config');
module.exports = {
  ...common,
  setupFilesAfterEnv: [...common.setupFilesAfterEnv, '<rootDir>/tests/setup/unit-no-browser.cjs'],
  roots: ['<rootDir>/tests/unit', '<rootDir>/tests/recording'],
  testMatch: ['**/*.test.ts'],
};
