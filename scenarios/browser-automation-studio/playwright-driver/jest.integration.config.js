// Self-contained local native contracts. Requires explicit browser/media admission.
const common = require('./jest.common.config');
module.exports = {
  ...common,
  setupFilesAfterEnv: [...common.setupFilesAfterEnv, '<rootDir>/tests/setup/native-case-progress.cjs'],
  roots: ['<rootDir>/tests/integration'],
  testMatch: ['**/*.test.ts'],
  testPathIgnorePatterns: [
    ...common.testPathIgnorePatterns,
    '<rootDir>/tests/integration/motion-qualification.test.ts',
    '<rootDir>/tests/integration/saved-workflow-fresh-context.test.ts',
    '<rootDir>/tests/integration/input-feedback.test.ts',
    '<rootDir>/tests/integration/audio-device-evidence.test.ts',
  ],
  coverageDirectory: 'coverage/native',
  reporters: ['default', '<rootDir>/tests/setup/native-progress.cjs'],
};
