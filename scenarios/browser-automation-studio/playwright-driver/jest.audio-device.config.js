// Explicit operator-admitted host audio topology qualification, never ordinary unit.
const common = require('./jest.common.config');
module.exports = {
  ...common,
  setupFilesAfterEnv: [...common.setupFilesAfterEnv, '<rootDir>/tests/setup/native-case-progress.cjs'],
  roots: ['<rootDir>/tests/integration'],
  testMatch: ['<rootDir>/tests/integration/audio-device-evidence.test.ts'],
  coverageDirectory: 'coverage/audio-device',
  reporters: ['default', '<rootDir>/tests/setup/native-progress.cjs'],
};
