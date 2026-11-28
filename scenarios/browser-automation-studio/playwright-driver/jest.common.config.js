// Shared transforms, source coverage denominator and unchanged numeric floors.
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  maxWorkers: 1,
  testPathIgnorePatterns: ['/node_modules/'],
  setupFilesAfterEnv: ['<rootDir>/tests/setup/silent-logger.ts'],
  collectCoverage: true,
  // Browser-bound evaluate callbacks must remain free of Node-only counters.
  coverageProvider: 'v8',
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'json-summary', 'lcov'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/types/**',
    '!src/server.ts',
  ],
  coverageThreshold: {
    global: {
      branches: 10,
      functions: 15,
      lines: 30,
      statements: 30,
    },
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
    // Jest resolves test imports through CommonJS conditions, while the
    // generated proto-types package intentionally exports only ESM/type
    // targets. Resolve its source tree directly for this test process.
    '^@vrooli/proto-types/(.*)$': '<rootDir>/../../../packages/proto/gen/typescript/$1',
    '^@bufbuild/protobuf$': '<rootDir>/node_modules/@bufbuild/protobuf/dist/cjs/index.js',
    '^@bufbuild/protobuf/(.*)$': '<rootDir>/node_modules/@bufbuild/protobuf/dist/cjs/$1/index.js',
  },
  // Transform ESM packages that Jest can't parse by default
  // @vrooli/proto-types and @bufbuild/protobuf use ESM syntax
  transformIgnorePatterns: [
    'node_modules/(?!((?:\\.pnpm/@vrooli\\+proto-types@[^/]+/node_modules/)?@vrooli/proto-types|@bufbuild)/)',
  ],
  // Configure ts-jest to transform TypeScript files in node_modules
  transform: {
    '^.+\\.tsx?$': [
      'ts-jest',
      {
        // Allow ts-jest to process node_modules packages
        tsconfig: {
          allowJs: true,
          esModuleInterop: true,
        },
      },
    ],
  },
};
