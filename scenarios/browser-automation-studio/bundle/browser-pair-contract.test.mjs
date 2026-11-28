import assert from 'node:assert/strict'
import test from 'node:test'
import { inspectBrowserPair } from './browser-pair-contract.mjs'

const pair = () => ({
  driver: { name: '@vrooli/playwright-driver', dependencies: { 'rebrowser-playwright': '1.52.0' } },
  wrapper: { name: 'rebrowser-playwright', version: '1.52.0' },
  core: { name: 'rebrowser-playwright-core', version: '1.52.0' },
  browsers: [
    { name: 'chromium', revision: '1169', browserVersion: '136.0.7103.25' },
    { name: 'chromium-headless-shell', revision: '1169', browserVersion: '136.0.7103.25' },
  ],
})

test('reports a coherent exact driver, wrapper, core and Chromium pair', () => {
  assert.deepEqual(inspectBrowserPair(pair()), {
    driverPackage: '@vrooli/playwright-driver', wrapperVersion: '1.52.0', coreVersion: '1.52.0',
    chromiumRevision: '1169', chromiumVersion: '136.0.7103.25',
    headlessShellRevision: '1169', headlessShellVersion: '136.0.7103.25',
  })
})

test('rejects wrapper/core skew, driver-pin skew, browser-revision skew and floating pins', () => {
  const coreSkew = pair(); coreSkew.core.version = '1.64.102'
  assert.throws(() => inspectBrowserPair(coreSkew), /does not match core/)
  const wrapperSkew = pair(); wrapperSkew.wrapper.version = '1.64.102'
  assert.throws(() => inspectBrowserPair(wrapperSkew), /does not match wrapper/)
  const browserSkew = pair(); browserSkew.browsers[1].revision = '1170'
  assert.throws(() => inspectBrowserPair(browserSkew), /revision 1169 does not match/)
  const floating = pair(); floating.driver.dependencies['rebrowser-playwright'] = '^1.52.0'
  assert.throws(() => inspectBrowserPair(floating), /must pin rebrowser-playwright exactly/)
})
