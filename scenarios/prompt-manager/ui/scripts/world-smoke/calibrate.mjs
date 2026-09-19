#!/usr/bin/env node
/** Fresh, complete hardware evidence -> reviewable budget proposal.
 * --evidence-dir is required. --apply writes tuning; increases need --reason.
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const requiredChecks = ['ready', 'settled-frames', 'webgl', 'hardware-renderer', 'no-page-errors', 'no-request-errors', 'app-loaded', 'governor', 'sim-invariants', 'vegetation-dry']
const allowedFailures = new Set(['draw-calls', 'triangles', 'p95-ms', 'budget-provenance', 'golden', 'golden-update-eligible'])

export function proposeCalibration(tuning, records, { headroom = 1.15, now = Date.now(), scene: onlyScene } = {}) {
  if (onlyScene && !tuning.budgets.scenes[onlyScene]) throw new Error('Unknown requested scene: ' + onlyScene)
  if (!Number.isFinite(headroom) || headroom < 1 || headroom > 2) throw new Error('headroom must be between 1 and 2')
  if (!records.length) throw new Error('No capture records')
  const next = structuredClone(tuning)
  const groups = new Map()
  let hardware
  for (const r of records) {
    if (onlyScene && r.scene !== onlyScene) throw new Error('Capture outside requested scene: ' + r.scene)
    const key = r.scene + '/' + r.profile
    if (!tuning.budgets.scenes[r.scene]?.[r.profile]) throw new Error('Unknown capture pair: ' + key)
    const age = now - Date.parse(r.capturedAt)
    if (!Number.isFinite(age) || age < 0 || age > 86400000) throw new Error('Stale or invalid capture date: ' + key)
    if (!r.gpu || r.timingMethod !== 'gpu-timer' || !r.renderer || /swiftshader|software|llvmpipe|unreported/i.test(r.renderer)) throw new Error('Hardware GPU timer required: ' + key)
    if (!['igpu', 'dgpu'].includes(r.gpuTier) || r.deviceScaleFactor !== tuning.quality.profiles[r.profile].dpr || r.actors !== 25) throw new Error('Capture provenance mismatch: ' + key)
    const identity = JSON.stringify([r.renderer, r.gpuTier])
    if (hardware && hardware !== identity) throw new Error('Mixed hardware provenance')
    hardware = identity
    for (const id of requiredChecks) {
      if (!r.checks?.some((c) => c.id === id && c.pass === true)) throw new Error('Missing or failed ' + id + ': ' + key)
    }
    if (r.checks.some((c) => !c.pass && !allowedFailures.has(c.id))) throw new Error('Invalid non-budget evidence: ' + key)
    if (!Array.isArray(r.consoleErrors) || r.consoleErrors.length || !Array.isArray(r.requestErrors) || r.requestErrors.length) throw new Error('Capture errors: ' + key)
    const d = r.diagnostics
    if (!d || d.gpu !== r.renderer || d.profile !== r.profile || d.auto !== false || ![d.drawCalls, d.triangles, d.gpuMsP95].every((n) => Number.isFinite(n) && n > 0)) throw new Error('Invalid measurements: ' + key)
    const group = groups.get(key) ?? []
    group.push(r)
    groups.set(key, group)
  }
  const changes = []
  for (const [scene, profiles] of Object.entries(tuning.budgets.scenes)) {
    if (onlyScene && scene !== onlyScene) continue
    for (const profile of Object.keys(tuning.quality.profiles)) {
      const key = scene + '/' + profile
      const group = groups.get(key)
      if (!group?.length || !profiles[profile]) throw new Error('Missing scene/profile: ' + key)
      const slowest = group.reduce((a, b) => a.diagnostics.gpuMsP95 >= b.diagnostics.gpuMsP95 ? a : b)
      const values = {
        drawCalls: Math.ceil(Math.max(...group.map((r) => r.diagnostics.drawCalls)) * headroom),
        triangles: Math.ceil(Math.max(...group.map((r) => r.diagnostics.triangles)) * headroom / 1000) * 1000,
        p95Ms: Number((slowest.diagnostics.gpuMsP95 * headroom).toFixed(1)),
      }
      const increases = Object.keys(values).filter((field) => values[field] > profiles[profile][field])
      changes.push({ scene, profile, before: profiles[profile], proposed: values, increases, sources: group.map((r) => r.name), timingSource: slowest.name })
      Object.assign(next.budgets.scenes[scene][profile], values, { provenance: {
        actors: slowest.actors, gpu: slowest.renderer, renderer: slowest.renderer,
        gpuTier: slowest.gpuTier, deviceScaleFactor: slowest.deviceScaleFactor,
        measuredP95Ms: slowest.diagnostics.gpuMsP95, target: false,
        calibratedAt: new Date(slowest.capturedAt).toISOString().slice(0, 10),
        method: 'gpu-timer; ' + slowest.gpuTier + '; dsf ' + slowest.deviceScaleFactor + '; worst of ' + group.length + ' captures; observed-plus-' + Math.round((headroom - 1) * 100) + 'pct-headroom',
      } })
    }
  }
  return { tuning: next, changes }
}

export function calibrateEvidence(tuningPath, evidenceDir, { apply = false, reason, headroom = 1.15, now = Date.now(), scene } = {}) {
  const tuning = JSON.parse(readFileSync(tuningPath, 'utf8'))
  const summary = JSON.parse(readFileSync(resolve(evidenceDir, 'summary.json'), 'utf8'))
  // Read only this completed run's named captures, never a directory glob.
  const records = summary.results.map((row) => {
    if (!/^[a-zA-Z0-9_-]+$/.test(row.name)) throw new Error('Invalid evidence name')
    const record = JSON.parse(readFileSync(resolve(evidenceDir, row.name + '.json'), 'utf8'))
    if (record.name !== row.name) throw new Error('Evidence identity differs from summary: ' + row.name)
    return record
  }).filter((r) => r.scene)
  const proposal = proposeCalibration(tuning, records, { headroom, now, scene })
  const report = { capturedAt: new Date(now).toISOString(), evidenceDir, scene: scene ?? null, reason: reason?.trim() || null, changes: proposal.changes }
  writeFileSync(resolve(evidenceDir, 'calibration-proposal.json'), JSON.stringify(report, null, 2) + '\n')
  if (apply) {
    if (proposal.changes.some((c) => c.increases.length) && !report.reason) throw new Error('Budget increases require --reason with measured-cost attribution; proposal saved, tuning unchanged')
    writeFileSync(tuningPath, JSON.stringify(proposal.tuning, null, 2) + '\n')
  }
  return report
}

function main() {
  const args = process.argv.slice(2)
  const opt = (name) => { const i = args.indexOf(name); return i < 0 ? undefined : args[i + 1] }
  if (!opt('--evidence-dir')) throw new Error('--evidence-dir is required; mixed legacy evidence is not accepted')
  const evidenceDir = resolve(opt('--evidence-dir'))
  const tuningPath = resolve(import.meta.dirname, '../../src/world/config/world.tuning.json')
  const apply = args.includes('--apply')
  const report = calibrateEvidence(tuningPath, evidenceDir, { apply, reason: opt('--reason'), scene: opt('--scene'), headroom: Number(opt('--headroom') ?? 1.15) })
  console.log((apply ? 'Applied' : 'Proposed') + ' ' + report.changes.length + ' budgets; see ' + evidenceDir + '/calibration-proposal.json')
  if (apply) console.log('Run pnpm world:tuning-docs to refresh generated documentation.')
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) main()
