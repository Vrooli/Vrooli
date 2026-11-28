// Unit contracts may override the configured provider with explicit fixtures.
// A permanent refusal is the default underneath fixture spies, so mockRestore()
// cannot accidentally restore a real native launch between unit tests.
function guardNativeBrowser() {
  const playwright = jest.requireActual('rebrowser-playwright');
  for (const name of ['chromium', 'firefox', 'webkit']) {
    const browser = playwright[name];
    if (!browser) continue;
    for (const operation of ['launch', 'launchPersistentContext', 'connect', 'connectOverCDP']) {
      const current = browser[operation];
      if (typeof current === 'function' && !jest.isMockFunction(current) && !current.unitBrowserBoundary) {
        browser[operation] = Object.assign(async function refuseNativeBrowser() {
          throw new Error(`Native browser ${name}.${operation} is forbidden in unit tests; use the configured provider fixture or admitted test:integration`);
        }, { unitBrowserBoundary: true });
      }
    }
  }
}
guardNativeBrowser();
beforeEach(guardNativeBrowser);
