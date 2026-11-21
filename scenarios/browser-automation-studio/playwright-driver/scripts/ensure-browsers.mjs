#!/usr/bin/env node
/**
 * Verify the Chromium build rebrowser-playwright expects is present.
 *
 * The driver verifies browser launch at startup; without this build the
 * /health endpoint reports an error and the API supervisor stops the sidecar,
 * so every workflow fails with "playwright driver is not responding". The
 * Browser acquisition belongs to the governed dependency/artifact owner. This
 * script is verification-only so a driver build cannot silently download a
 * browser into the build host's user cache.
 */
import { createRequire } from 'node:module'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'

// rebrowser-playwright depends on its core under the alias `playwright-core`
// (pnpm keeps the alias next to the wrapper), and the driver does not depend
// on the core directly, so resolve the alias from the wrapper's directory and
// check the package name rather than trusting the alias.
const driverRequire = createRequire(import.meta.url)
const wrapperDir = dirname(driverRequire.resolve('rebrowser-playwright/package.json'))
const corePackage = createRequire(join(wrapperDir, 'package.json')).resolve('playwright-core/package.json')
const coreName = JSON.parse(readFileSync(corePackage, 'utf8')).name
if (coreName !== 'rebrowser-playwright-core') {
  console.error(`ensure-browsers: expected rebrowser-playwright-core behind the playwright-core alias, found ${coreName}`)
  process.exit(1)
}
const coreDir = dirname(corePackage)
const browsers = JSON.parse(readFileSync(join(coreDir, 'browsers.json'), 'utf8')).browsers
const chromium = browsers.find((b) => b.name === 'chromium')
if (!chromium) {
  console.error('ensure-browsers: rebrowser-playwright-core/browsers.json lists no chromium build')
  process.exit(1)
}
const expected = driverRequire('rebrowser-playwright').chromium.executablePath()
if (existsSync(expected) && process.env.PLAYWRIGHT_ENSURE_BROWSERS !== 'force') {
  console.log(`ensure-browsers: chromium build ${chromium.revision} executable present at ${expected}`)
  process.exit(0)
}
console.error(`ensure-browsers: chromium build ${chromium.revision} missing at ${expected}`)
console.error('ensure-browsers: browser acquisition is not performed here; provision the versioned closure through the approved dependency/artifact owner')
process.exit(1)
