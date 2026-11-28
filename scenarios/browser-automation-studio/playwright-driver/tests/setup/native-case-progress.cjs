// The reporter identifies suite setup; this hook names each actual running case.
beforeEach(() => {
  process.stderr.write(`[native] case started ${expect.getState().currentTestName}\n`);
});
