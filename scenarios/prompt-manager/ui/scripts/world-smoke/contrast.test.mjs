import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { PNG } from 'pngjs'
import { captureContrasts, weatherContrast } from './contrast.mjs'

const { budgets } = JSON.parse(readFileSync(new URL('../../src/world/config/world.tuning.json', import.meta.url)))
function solid(rgb) {
  const png = new PNG({ width: 8, height: 8 })
  for (let i = 0; i < png.data.length; i += 4) png.data.set([...rgb, 255], i)
  return png
}
test('recognizes a pale green landscape becoming snow blue without changing the required area', () => {
  const result = weatherContrast(solid([194, 224, 189]), solid([194, 209, 215]), budgets)
  assert.equal(result.pass, true)
  assert.equal(result.ratio, 1)
  assert.equal(budgets.periodPixelDelta, 0.25)
})
test('rejects identical frames and single-channel quantization noise', () => {
  const a = solid([194, 224, 189])
  assert.equal(weatherContrast(a, a, budgets).pass, false)
  assert.equal(weatherContrast(a, solid([195, 224, 189]), budgets).pass, false)
})
test('rejects isolated changed pixels that do not cover the required area', () => {
  const a = solid([194, 224, 189])
  const b = solid([194, 224, 189])
  b.data.set([0, 0, 0, 255], 0)
  assert.equal(weatherContrast(a, b, budgets).pass, false)
})

const capture = (period, weather) => ({ name: `park-high-${period}-weather-${weather}`, scene: 'park', profile: 'high', period, weather, seed: 1, actors: 25, diagnostics: {}, checks: [{ id: 'snapshot', pass: true }] })
test('reports failed and missing capture pairs without throwing or comparing error pages', () => {
  const records = [capture('day', 'clear'), { ...capture('night', 'clear'), diagnostics: null }, capture('night', 'snow')]
  const rows = captureContrasts(records, { weatherStates: ['clear', 'snow'], periods: ['day', 'night'], budgets, readImage: () => { throw new Error('must not read an invalid pair') } })
  assert.equal(rows.length, 4)
  assert.ok(rows.every(row => !row.pass && row.ratio === null))
  assert.ok(rows.some(row => row.checks[0].detail.includes('incomplete')))
  assert.ok(rows.some(row => row.checks[0].detail.includes('no valid world snapshot')))
})
test('continues reporting subsequent valid pairs after an image error', () => {
  const records = ['clear', 'cloudy', 'snow'].map(weather => capture('day', weather))
  const rows = captureContrasts(records, { weatherStates: ['clear', 'cloudy', 'snow'], budgets, readImage: name => {
    if (name.endsWith('clear')) throw new Error('PNG missing')
    return solid(name.endsWith('cloudy') ? [194, 224, 189] : [194, 209, 215])
  } })
  assert.deepEqual(rows.map(row => row.pass), [false, false, true])
  assert.equal(rows[2].ratio, 1)
})
test('treats mismatched image sizes as failed evidence', () => {
  const rows = captureContrasts([capture('day', 'clear'), capture('night', 'clear')], { periods: ['day', 'night'], budgets, readImage: name => name.includes('-day-') ? solid([0, 0, 0]) : new PNG({ width: 1, height: 1 }) })
  assert.equal(rows.length, 1)
  assert.equal(rows[0].pass, false)
  assert.match(rows[0].checks[0].detail, /different dimensions/)
})
