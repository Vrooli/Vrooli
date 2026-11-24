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
import { createHash } from 'node:crypto'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join, relative, resolve, sep } from 'node:path'
import { inspectBrowserPair } from '../../bundle/browser-pair-contract.mjs'
import { runtimeDigest, validateRuntimeIdentity } from '../../bundle/runtime-identity.mjs'

// rebrowser-playwright depends on its core under the alias `playwright-core`
// (pnpm keeps the alias next to the wrapper), and the driver does not depend
// on the core directly, so resolve the alias from the wrapper's directory and
// check the package name rather than trusting the alias.
const driverRequire = createRequire(import.meta.url)
const driverPackage = JSON.parse(readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
const wrapperDir = dirname(driverRequire.resolve('rebrowser-playwright/package.json'))
const wrapperPackagePath = driverRequire.resolve('rebrowser-playwright/package.json')
const wrapperPackage = JSON.parse(readFileSync(wrapperPackagePath, 'utf8'))
const corePackagePath = createRequire(join(wrapperDir, 'package.json')).resolve('playwright-core/package.json')
const corePackage = JSON.parse(readFileSync(corePackagePath, 'utf8'))
const coreDir = dirname(corePackagePath)
const browserInventory = JSON.parse(readFileSync(join(coreDir, 'browsers.json'), 'utf8'))

function runtimePlatform() {
  const platform = process.platform === 'win32' ? 'windows' : process.platform
  const arch = process.arch === 'amd64' ? 'x64' : process.arch
  return `${platform}-${arch}`
}

function validTargetLifecycle(platform, lifecycle) {
  const expected = platform === 'linux-x64' ? 'unqualified-no-receipt'
    : ['linux-arm64', 'darwin-x64', 'darwin-arm64', 'windows-x64'].includes(platform) ? 'unqualified-no-node' : ''
  const required = ['runtime_launch', 'browser_task', 'web_console_pane', 'update_rollback', 'uninstall']
  if (!expected || !lifecycle || typeof lifecycle !== 'object' || Array.isArray(lifecycle)) return false
  const keys = Object.keys(lifecycle).sort()
  if (JSON.stringify(keys) !== JSON.stringify([...required].sort())) return false
  return required.every((capability) => lifecycle[capability] === expected)
}

let pair
try {
  pair = inspectBrowserPair({ driver: driverPackage, wrapper: wrapperPackage, core: corePackage, browsers: browserInventory.browsers })
} catch (error) {
  console.error(`ensure-browsers: unsupported driver/browser package pair: ${error instanceof Error ? error.message : String(error)}`)
  process.exit(1)
}
if (!process.env.PLAYWRIGHT_BROWSERS_PATH) {
  console.error('ensure-browsers: PLAYWRIGHT_BROWSERS_PATH must name an explicit provisioned closure; implicit user-cache lookup is disabled')
  process.exit(1)
}
const closureRoot = resolve(process.env.PLAYWRIGHT_BROWSERS_PATH)
const closureManifestPath = join(closureRoot, 'browser-closure.json')
let closure
try {
  closure = JSON.parse(readFileSync(closureManifestPath, 'utf8'))
  validateRuntimeIdentity(closure.runtime_identity)
} catch (error) {
  console.error(`ensure-browsers: invalid versioned browser closure: ${error instanceof Error ? error.message : String(error)}`)
  process.exit(1)
}
const identity = closure.runtime_identity
if (closure.schema_version !== 'v3' || closure.support_matrix_schema !== 'v3' || closure.platform !== runtimePlatform() || !validTargetLifecycle(closure.platform, closure.target_lifecycle_status) || identity.engine !== 'chromium' || identity.browser_version !== pair.chromiumRevision || identity.driver !== 'rebrowser-playwright-core' || identity.driver_version !== pair.coreVersion || closure.headless_shell_revision !== pair.headlessShellRevision) {
  console.error('ensure-browsers: staged browser runtime identity does not match the installed driver and browser inventory')
  process.exit(1)
}
let payloadDigest
try { payloadDigest = runtimeDigest(closureRoot) } catch (error) {
  console.error(`ensure-browsers: cannot verify browser runtime payload: ${error instanceof Error ? error.message : String(error)}`)
  process.exit(1)
}
if (payloadDigest !== identity.sha256) {
  console.error('ensure-browsers: staged browser runtime digest mismatch')
  process.exit(1)
}
const expected = driverRequire('rebrowser-playwright').chromium.executablePath()
const declaredExecutable = process.env.BROWSER_EXECUTABLE_PATH
const closureExecutable = resolve(closureRoot, closure.executable_path || '')
const executableRelative = relative(closureRoot, closureExecutable)
if (!declaredExecutable || resolve(declaredExecutable) !== resolve(expected) || resolve(expected) !== closureExecutable || executableRelative === '..' || executableRelative.startsWith(`..${sep}`)) {
  console.error('ensure-browsers: BROWSER_EXECUTABLE_PATH must match the exact executable selected by the staged closure and installed driver')
  process.exit(1)
}
if (!existsSync(expected)) {
  console.error(`ensure-browsers: Chromium revision ${pair.chromiumRevision} is missing at ${expected}`)
  console.error('ensure-browsers: browser acquisition is not performed here; provision the versioned closure through the approved dependency/artifact owner')
  process.exit(1)
}
const executableSha = createHash('sha256').update(readFileSync(expected)).digest('hex')
if (executableSha !== closure.executable_sha256) {
  console.error('ensure-browsers: staged browser executable checksum mismatch')
  process.exit(1)
}
const receiptPath = join(closureRoot, '.offline-install-receipt.json')
if (existsSync(receiptPath)) {
  try {
    const receipt = JSON.parse(readFileSync(receiptPath, 'utf8'))
    validateRuntimeIdentity(receipt.runtime_identity)
    if (receipt.schema_version !== 'v3' || !/^[0-9a-f]{64}$/.test(receipt.archive_sha256 ?? '') || receipt.runtime_identity.bundle_version !== identity.bundle_version || receipt.runtime_identity.sha256 !== identity.sha256) throw new Error('receipt does not match staged identity')
  } catch (error) {
    console.error(`ensure-browsers: offline install receipt mismatch: ${error instanceof Error ? error.message : String(error)}`)
    process.exit(1)
  }
}
console.log(`ensure-browsers: selected ${identity.bundle_version}; executable ${expected}`)
if (existsSync(expected) && process.env.PLAYWRIGHT_ENSURE_BROWSERS !== 'force') {
  console.log(`ensure-browsers: Chromium executable present at ${expected}`)
  process.exit(0)
}
console.error(`ensure-browsers: Chromium ${pair.chromiumVersion ?? 'unknown'} revision ${pair.chromiumRevision} missing at ${expected}`)
console.error('ensure-browsers: browser acquisition is not performed here; provision the versioned closure through the approved dependency/artifact owner')
process.exit(1)
