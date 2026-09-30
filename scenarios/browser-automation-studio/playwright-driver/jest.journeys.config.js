// Preservation journeys against the goal's shadow BAS and the local fixture site.
// `pnpm test:journeys` prints each [REQ:BAS-RH-Jxx] result; add `--json --outputFile=<path>` for a machine-readable report.
const base = require('./jest.config');

module.exports = {
  ...base,
  roots: ['<rootDir>/tests/journeys'],
  testPathIgnorePatterns: ['/node_modules/'],
  collectCoverage: false,
  coverageThreshold: undefined,
  globalSetup: '<rootDir>/tests/journeys/global-setup.mjs',
  globalTeardown: '<rootDir>/tests/journeys/global-teardown.mjs',
  testTimeout: 60000,
};
