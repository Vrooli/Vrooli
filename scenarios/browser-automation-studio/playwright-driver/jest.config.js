module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: ['**/*.test.ts'],
  collectCoverage: true,
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
      branches: 0,
      functions: 0,
      lines: 0,
      statements: 0,
    },
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
  // Transform ESM packages that Jest can't parse by default
  // @vrooli/proto-types and @bufbuild/protobuf use ESM syntax
  transformIgnorePatterns: [
    'node_modules/(?!(@vrooli/proto-types|@bufbuild)/)',
  ],
  // Configure ts-jest to transform TypeScript files in node_modules
  transform: {
    '^.+\\.tsx?$': [
      'ts-jest',
      {
        // Use isolatedModules for faster transforms
        isolatedModules: true,
        // Allow ts-jest to process node_modules packages
        tsconfig: {
          allowJs: true,
          esModuleInterop: true,
        },
      },
    ],
  },
};
