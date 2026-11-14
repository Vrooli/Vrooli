import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, readFileSync, writeFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { calibrateEvidence, proposeCalibration } from './calibrate.mjs'

const now = Date.parse('2026-09-04T23:00:00Z')
const tuning = {
  quality: { profiles: { low: { dpr: 1 }, high: { dpr: 1.5 } } },
  budgets: { scenes: Object.fromEntries(['park', 'office'].map((scene) => [scene, Object.fromEntries(['low', 'high'].map((profile) => [profile, { drawCalls: 100, triangles: 100000, p95Ms: 10 }]))])) },
}
const checks = ['ready', 'settled-frames', 'webgl', 'hardware-renderer', 'no-page-errors', 'no-request-errors', 'app-loaded', 'governor', 'sim-invariants', 'vegetation-dry']
function records() {
  return Object.keys(tuning.budgets.scenes).flatMap((scene) => Object.keys(tuning.quality.profiles).map((profile) => ({
    name: scene + '-' + profile, scene, profile, actors: 25, gpu: true, gpuTier: 'igpu', renderer: 'ANGLE AMD hardware',
    deviceScaleFactor: tuning.quality.profiles[profile].dpr, timingMethod: 'gpu-timer', capturedAt: new Date(now - 1000).toISOString(),
    checks: checks.map((id) => ({ id, pass: true })), consoleErrors: [], requestErrors: [],
    diagnostics: { gpu: 'ANGLE AMD hardware', profile, auto: false, drawCalls: 50, triangles: 10000, gpuMsP95: 5 },
  })))
}
test('aggregates worst measurements independently of record order and preserves input', () => {
  const input = records()
  input.push({ ...structuredClone(input[0]), name: 'park-low-night', diagnostics: { ...input[0].diagnostics, drawCalls: 80, gpuMsP95: 8 } })
  const before = structuredClone(tuning)
  const a = proposeCalibration(tuning, input, { now }).tuning
  const b = proposeCalibration(tuning, input.toReversed(), { now }).tuning
  assert.deepEqual(a, b)
  assert.equal(a.budgets.scenes.park.low.drawCalls, 92)
  assert.equal(a.budgets.scenes.park.low.p95Ms, 9.2)
  assert.deepEqual(tuning, before)
})
test('rejects incomplete scene/profile coverage', () => {
  assert.throws(() => proposeCalibration(tuning, records().slice(1), { now }), /Missing scene\/profile/)
})
test('explicit scene scope preserves other budgets and still requires every selected profile', () => {
  const input = records().filter((r) => r.scene === 'park')
  const result = proposeCalibration(tuning, input, { now, scene: 'park' })
  assert.equal(result.changes.length, 2)
  assert.deepEqual(result.tuning.budgets.scenes.office, tuning.budgets.scenes.office)
  assert.throws(() => proposeCalibration(tuning, input.slice(1), { now, scene: 'park' }), /Missing scene\/profile/)
  assert.throws(() => proposeCalibration(tuning, records(), { now, scene: 'park' }), /outside requested scene/)
  assert.throws(() => proposeCalibration(tuning, input, { now, scene: 'unknown' }), /Unknown requested scene/)
})
test('publication eligibility does not hide the underlying failed runtime checks', () => {
  const input = records()
  input[0].checks.push({ id: 'p95-ms', pass: false }, { id: 'golden-update-eligible', pass: false })
  assert.equal(proposeCalibration(tuning, input, { now }).changes.length, 4)
  input[0].checks.find((c) => c.id === 'ready').pass = false
  assert.throws(() => proposeCalibration(tuning, input, { now }), /ready/)
})
for (const [label, mutate, pattern] of [
  ['software renderer', (r) => { r.renderer = 'SwiftShader' }, /Hardware/],
  ['fallback timing', (r) => { r.timingMethod = 'vsync-off-fallback' }, /Hardware/],
  ['stale evidence', (r) => { r.capturedAt = '2026-09-01T00:00:00Z' }, /Stale/],
  ['wrong DPR', (r) => { r.deviceScaleFactor = 3 }, /provenance/],
  ['wrong actors', (r) => { r.actors = 400 }, /provenance/],
  ['mixed renderer', (r) => { r.renderer = r.diagnostics.gpu = 'ANGLE NVIDIA hardware' }, /Mixed/],
  ['HTTP errors', (r) => { r.requestErrors.push({ status: 500 }) }, /Capture errors/],
  ['failed readiness', (r) => { r.checks[0].pass = false }, /ready/],
  ['missing overlay check', (r) => { r.checks = r.checks.filter((c) => c.id !== 'app-loaded') }, /app-loaded/],
  ['invalid timing', (r) => { r.diagnostics.gpuMsP95 = NaN }, /measurements/],
]) test('rejects ' + label, () => {
  const input = records()
  mutate(input[1])
  assert.throws(() => proposeCalibration(tuning, input, { now }), pattern)
})
test('accepts old budget and golden failures, but rejects unrelated failures', () => {
  const input = records()
  input[0].checks.push({ id: 'golden', pass: false }, { id: 'draw-calls', pass: false })
  assert.equal(proposeCalibration(tuning, input, { now }).changes.length, 4)
  input[0].checks.push({ id: 'lamp-pool', pass: false })
  assert.throws(() => proposeCalibration(tuning, input, { now }), /non-budget/)
})
test('rejects invalid headroom', () => {
  for (const headroom of [NaN, 0, 0.9, 3]) assert.throws(() => proposeCalibration(tuning, records(), { now, headroom }), /headroom/)
})

test('file workflow defaults to proposal, requires explained increases, and applies complete provenance', (t) => {
  const dir = mkdtempSync(join(tmpdir(), 'world-calibration-test-'))
  t.after(() => rmSync(dir, { recursive: true, force: true }))
  const tuningPath = join(dir, 'tuning.json')
  const original = JSON.stringify(tuning)
  writeFileSync(tuningPath, original)
  const input = records()
  input[0].diagnostics.gpuMsP95 = 20
  for (const r of input) writeFileSync(join(dir, r.name + '.json'), JSON.stringify(r))
  writeFileSync(join(dir, 'summary.json'), JSON.stringify({ results: input.map((r) => ({ name: r.name })) }))
  // Unlisted stale files are not part of this batch.
  writeFileSync(join(dir, 'old.json'), 'invalid JSON')
  const report = calibrateEvidence(tuningPath, dir, { now })
  assert.equal(readFileSync(tuningPath, 'utf8'), original)
  assert.equal(report.changes.length, 4)
  assert.throws(() => calibrateEvidence(tuningPath, dir, { now, apply: true, reason: ' ' }), /increases require/)
  assert.equal(readFileSync(tuningPath, 'utf8'), original)
  calibrateEvidence(tuningPath, dir, { now, apply: true, reason: 'Test fixture deliberately increases the measured GPU cost.' })
  const updated = JSON.parse(readFileSync(tuningPath, 'utf8'))
  assert.equal(updated.budgets.scenes.park.low.p95Ms, 23)
  assert.equal(updated.budgets.scenes.park.low.provenance.measuredP95Ms, 20)
  assert.equal(updated.budgets.scenes.park.low.provenance.calibratedAt, '2026-09-04')
  assert.match(JSON.parse(readFileSync(join(dir, 'calibration-proposal.json'), 'utf8')).reason, /deliberately/)
  input[0].name = 'wrong-name'
  writeFileSync(join(dir, 'park-low.json'), JSON.stringify(input[0]))
  assert.throws(() => calibrateEvidence(tuningPath, dir, { now }), /identity differs/)
})
