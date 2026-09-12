import baseConfig from './vite.config';

// The normal configuration declares the scenario-wide coverage floor. Focused
// projects emit raw V8 maps here; the runner enforces that floor after merging
// the complete selected suite.
export default async (env: { mode: string; command: string }) => {
  const config = await baseConfig(env);
  const thresholds = { lines: 0, functions: 0, branches: 0, statements: 0 };
  const reportsDirectory = process.env.BAS_COVERAGE_REPORTS_DIRECTORY ?? './coverage';
  const test = config.test as { coverage?: object; projects?: Array<{ test?: { coverage?: object } }> } | undefined;
  const { include: _rootInclude, ...rootCoverage } = (test?.coverage ?? {}) as { include?: unknown };

  return {
    ...config,
    test: {
      ...test,
      coverage: { ...rootCoverage, all: false, reportsDirectory, thresholds },
      projects: test?.projects?.map((project) => ({
        ...project,
        ...(() => {
          const { include: _projectInclude, ...projectCoverage } = (project.test?.coverage ?? {}) as { include?: unknown };
          return {
            test: {
              ...project.test,
              coverage: { ...projectCoverage, all: false, enabled: true, reportsDirectory, thresholds },
            },
          };
        })(),
      })),
    },
  };
};
