import assert from 'node:assert/strict'
import test from 'node:test'
import path from 'node:path'
import { resolveUIDistDir } from '../../server-dist-path.js'

test('resolves the selected instance asset directory inside the UI package', () => {
  assert.equal(
    resolveUIDistDir('dist/shadow', '/scenario/ui'),
    path.join('/scenario/ui', 'dist/shadow'),
  )
})

test('preserves the existing default asset directory', () => {
  assert.equal(resolveUIDistDir('', '/scenario/ui'), path.join('/scenario/ui', 'dist'))
})

test('rejects an asset directory outside the UI package', () => {
  assert.throws(
    () => resolveUIDistDir('../../shared/dist', '/scenario/ui'),
    /must stay within the BAS UI package/,
  )
})
