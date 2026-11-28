const exactVersion = /^\d+\.\d+\.\d+$/

/** Validate the exact Playwright wrapper/core/browser inventory used by BAS. */
export function inspectBrowserPair({ driver, wrapper, core, browsers }) {
  const problems = []
  const declared = driver?.dependencies?.['rebrowser-playwright']
  if (!exactVersion.test(declared ?? '')) {
    problems.push(`driver must pin rebrowser-playwright exactly; got ${declared ?? '(missing)'}`)
  }
  if (wrapper?.name !== 'rebrowser-playwright') {
    problems.push(`wrapper package name mismatch: ${wrapper?.name ?? '(missing)'}`)
  }
  if (core?.name !== 'rebrowser-playwright-core') {
    problems.push(`core package name mismatch: ${core?.name ?? '(missing)'}`)
  }
  if (declared !== wrapper?.version) {
    problems.push(`driver pin ${declared ?? '(missing)'} does not match wrapper ${wrapper?.version ?? '(missing)'}`)
  }
  if (wrapper?.version !== core?.version) {
    problems.push(`wrapper ${wrapper?.version ?? '(missing)'} does not match core ${core?.version ?? '(missing)'}`)
  }

  const chromium = browsers?.find((browser) => browser.name === 'chromium')
  const headlessShell = browsers?.find((browser) => browser.name === 'chromium-headless-shell')
  if (!chromium || !headlessShell) {
    problems.push('browser inventory must include chromium and chromium-headless-shell')
  } else {
    if (String(chromium.revision) !== String(headlessShell.revision)) {
      problems.push(`Chromium revision ${chromium.revision} does not match headless-shell revision ${headlessShell.revision}`)
    }
    if (chromium.browserVersion && headlessShell.browserVersion && chromium.browserVersion !== headlessShell.browserVersion) {
      problems.push(`Chromium version ${chromium.browserVersion} does not match headless-shell version ${headlessShell.browserVersion}`)
    }
  }

  if (problems.length) throw new Error(problems.join('; '))
  return {
    driverPackage: driver.name,
    wrapperVersion: wrapper.version,
    coreVersion: core.version,
    chromiumRevision: String(chromium.revision),
    chromiumVersion: chromium.browserVersion ?? null,
    headlessShellRevision: String(headlessShell.revision),
    headlessShellVersion: headlessShell.browserVersion ?? null,
  }
}
