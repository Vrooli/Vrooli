#!/usr/bin/env node
/**
 * Writes draw-call and triangle budgets into world.tuning.json from the latest
 * smoke evidence with a headroom factor, and p95 budgets when the evidence
 * came from the host GPU. Budgets are levers: this script is how they are
 * earned from measurement rather than guessed.
 *
 *   node scripts/world-smoke/calibrate.mjs [--headroom 1.2]
 */
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const evidenceDir = resolve(uiRoot, 'evidence', 'world-smoke')
const tuningPath = resolve(uiRoot, 'src/world/config/world.tuning.json')
const args = process.argv.slice(2)
const headroom = Number(args.includes('--headroom') ? args[args.indexOf('--headroom') + 1] : '1.2')
const tuning = JSON.parse(readFileSync(tuningPath, 'utf8'))

const records = readdirSync(evidenceDir)
  .filter((f) => f.endsWith('.json') && f !== 'summary.json')
  .map((f) => JSON.parse(readFileSync(resolve(evidenceDir, f), 'utf8')))
  .filter((r) => r.diagnostics && r.checks?.some((c) => c.id === 'ready' && c.pass))

let changed = 0
for (const record of records) {
  const budget = tuning.budgets.scenes[record.scene]?.[record.profile]
  if (!budget) continue
  const drawCalls = Math.ceil(record.diagnostics.drawCalls * headroom)
  const triangles = Math.ceil((record.diagnostics.triangles * headroom) / 1000) * 1000
  if (drawCalls !== budget.drawCalls || triangles !== budget.triangles) changed += 1
  budget.drawCalls = drawCalls
  budget.triangles = triangles
  if (record.gpu && record.timingMethod === 'gpu-timer') budget.p95Ms = Number((record.diagnostics.gpuMsP95 * headroom).toFixed(1))
  if (record.gpu && record.timingMethod === 'vsync-off-fallback') budget.p95Ms = Number((record.diagnostics.frameMsP95 * headroom).toFixed(1))
  budget.provenance = {
    actors: record.actors,
    gpu: record.diagnostics.gpu || 'unreported renderer',
    calibratedAt: new Date(record.capturedAt).toISOString().slice(0, 10),
    method: record.timingMethod,
  }
}
writeFileSync(tuningPath, `${JSON.stringify(tuning, null, 2)}\n`)
console.log(`calibrated ${records.length} evidence record(s), ${changed} budget(s) changed (headroom ${headroom}); run pnpm world:tuning-docs`)
if (!existsSync(resolve(evidenceDir, 'summary.json'))) console.warn('no summary.json: run pnpm world:smoke first')
