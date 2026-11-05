// Explicit managed qualification: required runtime inputs fail fast in the test.
// This five-minute workload is not an implicit ordinary unit coverage dependency.
const base = require('./jest.common.config');
module.exports = {
  ...base,
  roots: ['<rootDir>/tests/integration'],
  testMatch: ['<rootDir>/tests/integration/motion-qualification.test.ts'],
  testPathIgnorePatterns: ['/node_modules/'],
  collectCoverage: false,
  coverageThreshold: undefined,
};
