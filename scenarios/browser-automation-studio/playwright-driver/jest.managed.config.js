// Existing managed BAS and routed test storage are operator-owned prerequisites.
const common = require('./jest.common.config');
module.exports = {
  ...common,
  setupFilesAfterEnv: [...common.setupFilesAfterEnv, '<rootDir>/tests/setup/native-case-progress.cjs'],
  roots: ['<rootDir>/tests/integration'],
  testMatch: [
    '<rootDir>/tests/integration/saved-workflow-fresh-context.test.ts',
    '<rootDir>/tests/integration/input-feedback.test.ts',
  ],
  coverageDirectory: 'coverage/managed',
  reporters: ['default', '<rootDir>/tests/setup/native-progress.cjs'],
};
