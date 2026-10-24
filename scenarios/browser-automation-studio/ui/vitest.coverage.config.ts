import baseConfig from './vite.config';

// One designated project instruments the complete existing source denominator;
// the remaining projects add executed coverage without repeating that transform.
// Thresholds are applied once to the complete, validated run by the owner merger.
export default async (env: { mode: string; command: string }) => {
  const config = await baseConfig(env);
  return {
    ...config,
    test: {
      ...config.test,
      coverage: {
        ...config.test?.coverage,
        enabled: true,
        reporter: ['json'],
        all: process.env.BAS_COVERAGE_SOURCE_CLOSURE === '1',
        reportsDirectory: process.env.BAS_COVERAGE_REPORTS_DIRECTORY ?? './coverage',
        thresholds: { lines: 0, functions: 0, branches: 0, statements: 0 },
      },
    },
  };
};
