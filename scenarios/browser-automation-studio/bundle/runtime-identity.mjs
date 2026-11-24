import { createHash } from 'node:crypto'
import { lstatSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

const safeToken = /^[A-Za-z0-9][A-Za-z0-9.+_-]{0,63}$/

export function validateRuntimeIdentity(identity) {
  if (!identity || identity.schema_version !== 'v1' || !['chromium', 'chrome', 'camoufox'].includes(identity.engine)) {
    throw new Error('browser runtime identity schema or engine is unsupported')
  }
  for (const [label, value] of [['browser version', identity.browser_version], ['driver', identity.driver], ['driver version', identity.driver_version]]) {
    if (typeof value !== 'string' || !safeToken.test(value)) throw new Error(`browser runtime ${label} must be a pinned path-safe token`)
  }
  for (const value of [identity.browser_version, identity.driver_version]) {
    if (['latest', 'stable', 'current', 'tip', 'nightly'].includes(value.toLowerCase())) throw new Error(`browser runtime version is floating: ${value}`)
  }
  const expectedVersion = `br1-${identity.engine}-${identity.browser_version}-${identity.driver}-${identity.driver_version}`
  if (identity.bundle_version !== expectedVersion || expectedVersion.length > 255 || !/^[0-9a-f]{64}$/.test(identity.sha256 ?? '')) {
    throw new Error('browser runtime bundle version or digest does not match its pinned pair')
  }
  return identity
}

// Hash sorted regular payload entries by relative path, mode, size and file
// content digest. The manifest and offline receipt are excluded to avoid a
// recursive digest. Keep framing in sync with packages/browser-closure.
export function runtimeDigest(root) {
  const entries = []
  const visit = (directory, relativeDirectory = '') => {
    for (const name of readdirSync(directory)) {
      const absolute = join(directory, name)
      const rel = relativeDirectory ? `${relativeDirectory}/${name}` : name
      const stat = lstatSync(absolute)
      if (stat.isSymbolicLink()) throw new Error(`symlink in browser runtime tree: ${rel}`)
      if (stat.isDirectory()) visit(absolute, rel)
      else if (stat.isFile() && rel !== 'browser-closure.json' && rel !== '.offline-install-receipt.json') {
        entries.push({ path: rel, mode: stat.mode & 0o777, size: stat.size, sha: createHash('sha256').update(readFileSync(absolute)).digest('hex') })
      } else if (!stat.isFile()) throw new Error(`unsupported browser runtime entry: ${rel}`)
    }
  }
  visit(root)
  entries.sort((a, b) => Buffer.compare(Buffer.from(a.path), Buffer.from(b.path)))
  if (entries.length === 0) throw new Error('browser runtime payload is empty')
  const hash = createHash('sha256')
  for (const entry of entries) {
    hash.update(`${Buffer.byteLength(entry.path)}:${entry.path}:${entry.mode.toString(8).padStart(4, '0')}:${entry.size}:${entry.sha}\n`)
  }
  return hash.digest('hex')
}

export function newRuntimeIdentity({ directory, engine, browserVersion, driver, driverVersion }) {
  return validateRuntimeIdentity({
    schema_version: 'v1',
    engine,
    browser_version: String(browserVersion),
    driver,
    driver_version: String(driverVersion),
    bundle_version: `br1-${engine}-${browserVersion}-${driver}-${driverVersion}`,
    sha256: runtimeDigest(directory),
  })
}
