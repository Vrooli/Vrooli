#!/usr/bin/env node
/**
 * Ensure the Chromium build rebrowser-playwright expects is present.
 *
 * The driver verifies browser launch at startup; without this build the
 * /health endpoint reports an error and the API supervisor stops the sidecar,
 * so every workflow fails with "playwright driver is not responding". The
 * browser cache (~/.cache/ms-playwright) is host state that cleanups can and
 * do sweep, so the build step re-checks it instead of assuming an earlier
 * install survived. The installer is idempotent: a present build is a no-op.
 */
import { createRequire } from 'node:module'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { spawnSync } from 'node:child_process'
import { homedir } from 'node:os'

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
const cacheRoot = process.env.PLAYWRIGHT_BROWSERS_PATH || join(homedir(), '.cache', 'ms-playwright')
const expected = join(cacheRoot, `chromium-${chromium.revision}`)
if (existsSync(expected) && process.env.PLAYWRIGHT_ENSURE_BROWSERS !== 'force') {
  console.log(`ensure-browsers: chromium build ${chromium.revision} present at ${expected}`)
  process.exit(0)
}
console.log(`ensure-browsers: chromium build ${chromium.revision} missing at ${expected}; installing`)
const result = spawnSync(process.execPath, [join(coreDir, 'cli.js'), 'install', 'chromium'], { stdio: 'inherit' })
if (result.status !== 0) {
  console.error(`ensure-browsers: install failed with status ${result.status}`)
  process.exit(result.status ?? 1)
}
if (!existsSync(expected)) {
  console.error(`ensure-browsers: install finished but ${expected} is still missing`)
  process.exit(1)
}
console.log(`ensure-browsers: chromium build ${chromium.revision} installed`)
