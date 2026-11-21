const reporters = ["json", "./events-reporter.mjs"];
if (process.env.QUALITY_REQUIREMENT_REPORTER) {
  const { default: RequirementReporter } = await import(process.env.QUALITY_REQUIREMENT_REPORTER);
  reporters.push(new RequirementReporter({ outputFile: 'requirements.json', verbose: false, emitStdout: false, autoClear: false }));
}
export default {
  test: {
    environment: "node",
    expect: { requireAssertions: true },
    minWorkers: 1,
    maxWorkers: 1,
    reporters,
  },
};
