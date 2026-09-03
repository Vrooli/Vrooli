#!/usr/bin/env node
/**
 * World smoke tool.
 *
 * Loads /world for every scene and quality profile in headless Chrome
 * (SwiftShader by default, --gpu for the host GPU), waits for
 * window.__worldDiagnostics.ready, reads the renderer counters and the sim
 * invariants, screenshots the frame, asserts the budgets declared in
 * world.tuning.json and diffs the frame against the checked-in golden.
 *
 * Numbers first: every verdict is a counter or an invariant. The PNG diff is
 * a regression tripwire, never a quality judgement; look at pictures through
 * `pnpm world:sheet`, one contact sheet per batch.
 *
 *   pnpm world:smoke                 run every scene × profile, fail on budget/golden
 *   pnpm world:goldens               rewrite the goldens from this run
 *   node scripts/world-smoke/run.mjs --gpu --scene park --profile high --period night
 *
 * Evidence: evidence/world-smoke/<scene>-<profile>[-<period>].{json,png,diff.png}
 */
import { chromium } from 'playwright-core'
import { mkdirSync, readFileSync, writeFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'
import { PNG } from 'pngjs'
import pixelmatch from 'pixelmatch'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const tuning = JSON.parse(readFileSync(resolve(uiRoot, 'src/world/config/world.tuning.json'), 'utf8'))
const evidenceDir = resolve(uiRoot, 'evidence', 'world-smoke')
const goldenDir = resolve(uiRoot, 'src', 'world', '__goldens__')
mkdirSync(evidenceDir, { recursive: true })
mkdirSync(goldenDir, { recursive: true })

const args = process.argv.slice(2)
const flag = (name) => args.includes(name)
const opt = (name, fallback) => {
  const i = args.indexOf(name)
  return i >= 0 && args[i + 1] ? args[i + 1] : fallback
}
const updateGoldens = flag('--update-goldens')
const useGpu = flag('--gpu')
const onlyScene = opt('--scene', null)
const onlyProfile = opt('--profile', null)
const onlyPeriod = opt('--period', null)
const periods = onlyPeriod ? [onlyPeriod] : (flag('--all-periods') ? ['dawn', 'day', 'dusk', 'night'] : ['day'])
const seed = opt('--seed', '1')
const actors = opt('--actors', null)
const emptyStage = flag('--empty-stage')
const extraQuery = opt('--query', '')
const baseUrl = opt('--base-url', process.env.WORLD_SMOKE_BASE_URL ?? resolveBaseUrl())
const width = 1600
const height = 1000
const readyTimeoutMs = Number(opt('--ready-timeout', '90000'))
const settleFrames = 30

function resolveBaseUrl() {
  try {
    const port = execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()
    if (/^\d+$/.test(port)) return `http://localhost:${port}`
  } catch {
    // fall through
  }
  return 'http://localhost:21235'
}

const scenes = onlyScene ? [onlyScene] : Object.keys(tuning.budgets.scenes)
const profiles = onlyProfile ? [onlyProfile] : Object.keys(tuning.quality.profiles)

const chromeArgs = useGpu
  ? ['--headless=new', '--ignore-gpu-blocklist', '--enable-gpu-rasterization', '--use-angle=vulkan', '--use-vulkan=native', '--enable-features=Vulkan,VulkanFromANGLE,DefaultANGLEVulkan', '--no-sandbox']
  : ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox']

const browser = await chromium.launch({
  executablePath: process.env.WORLD_SMOKE_CHROME ?? '/usr/bin/google-chrome',
  headless: true,
  args: chromeArgs,
})

const results = []
let failed = false

function log(line) {
  process.stdout.write(`${line}\n`)
}

function comparePng(actualBuffer, goldenPath, diffPath) {
  if (!existsSync(goldenPath)) return { pixels: -1, ratio: -1, missing: true }
  const actual = PNG.sync.read(actualBuffer)
  const golden = PNG.sync.read(readFileSync(goldenPath))
  if (actual.width !== golden.width || actual.height !== golden.height) {
    return { pixels: -1, ratio: 1, missing: false, sizeMismatch: true }
  }
  const diff = new PNG({ width: actual.width, height: actual.height })
  const pixels = pixelmatch(actual.data, golden.data, diff.data, actual.width, actual.height, { threshold: 0.12 })
  writeFileSync(diffPath, PNG.sync.write(diff))
  return { pixels, ratio: pixels / (actual.width * actual.height), missing: false }
}

try {
  for (const scene of scenes) {
    for (const profile of profiles) {
      for (const period of periods) {
        const name = periods.length > 1 || onlyPeriod ? `${scene}-${profile}-${period}` : `${scene}-${profile}`
        const context = await browser.newContext({ viewport: { width, height }, deviceScaleFactor: 1, reducedMotion: 'no-preference' })
        const page = await context.newPage()
        const consoleErrors = []
        page.on('pageerror', (err) => consoleErrors.push(String(err)))
        page.on('console', (msg) => { if (msg.type() === 'error') consoleErrors.push(msg.text()) })
        const query = new URLSearchParams({ scene, profile, period, intro: '0', seed, diag: '1' })
        if (actors) query.set('actors', actors)
        for (const [k, v] of new URLSearchParams(extraQuery)) query.set(k, v)
        const url = `${baseUrl}/world?${query.toString()}`
        const started = Date.now()
        await page.goto(url, { waitUntil: 'domcontentloaded' })
        let ready = false
        try {
          await page.waitForFunction(() => globalThis.__worldDiagnostics?.ready === true, null, { timeout: readyTimeoutMs })
          ready = true
        } catch {
          ready = false
        }
        // Let the frame stats settle after ready.
        if (ready) {
          const target = (await page.evaluate(() => globalThis.__worldDiagnostics.framesRendered)) + settleFrames
          try {
            await page.waitForFunction((t) => globalThis.__worldDiagnostics.framesRendered >= t, target, { timeout: 30000 })
          } catch {
            // keep whatever we have
          }
        }
        const diagnostics = await page.evaluate(() => globalThis.__worldDiagnostics ?? null)
        const sim = await page.evaluate(() => {
          const probe = globalThis.__worldSim
          if (!probe) return null
          return { violations: probe.violations(), actors: probe.actorCount(), revision: probe.revision() }
        })
        const png = await page.screenshot({ type: 'png', fullPage: false })
        const pngPath = resolve(evidenceDir, `${name}.png`)
        writeFileSync(pngPath, png)
        const goldenPath = resolve(goldenDir, `${name}.png`)
        if (updateGoldens) writeFileSync(goldenPath, png)
        const golden = comparePng(png, goldenPath, resolve(evidenceDir, `${name}.diff.png`))
        const budget = tuning.budgets.scenes[scene]?.[profile]
        const checks = []
        const check = (id, pass, detail) => checks.push({ id, pass, detail })
        check('ready', ready, ready ? `ready after ${Date.now() - started} ms` : `not ready within ${readyTimeoutMs} ms`)
        check('no-page-errors', consoleErrors.length === 0, consoleErrors.slice(0, 3).join(' | ') || 'none')
        if (diagnostics && budget) {
          const drawCallBudget = emptyStage ? tuning.budgets.emptyStageDrawCalls : budget.drawCalls
          check('draw-calls', diagnostics.drawCalls <= drawCallBudget, `${diagnostics.drawCalls} <= ${drawCallBudget}${emptyStage ? ' (empty stage)' : ''}`)
          check('triangles', diagnostics.triangles <= budget.triangles, `${diagnostics.triangles} <= ${budget.triangles}`)
          const p95 = diagnostics.frameMsP95
          if (useGpu) check('p95-ms', p95 <= budget.p95Ms, `${p95.toFixed(1)} <= ${budget.p95Ms}`)
          else check('p95-ms-informational', true, `${p95.toFixed(1)} ms under SwiftShader (not gated)`)
          check('tone-mapping', diagnostics.toneMapping === 'agx', diagnostics.toneMapping)
          check('min-clearance', diagnostics.nearestHit < 0 || diagnostics.nearestHit >= tuning.camera.minClearance, `nearest hit ${diagnostics.nearestHit.toFixed(2)} m >= ${tuning.camera.minClearance} m`)
          const { minFill, maxFill } = tuning.budgets.framing
          const fill = diagnostics.footprintFill
          check('framing', fill >= minFill && fill <= maxFill, `footprint fills ${(fill * 100).toFixed(0)}% of the viewport (want ${minFill * 100}-${maxFill * 100}%)`)
        }
        if (sim) {
          const worst = sim.violations.slice(0, 3).map((v) => `${v.rule}: ${v.detail}`).join(' | ')
          check('sim-invariants', sim.violations.length === 0, sim.violations.length === 0 ? `0 violations across ${sim.actors} actors` : `${sim.violations.length} violations; ${worst}`)
        } else {
          check('sim-invariants', false, 'window.__worldSim missing')
        }
        if (golden.missing) check('golden', updateGoldens, updateGoldens ? 'golden written' : 'golden missing; run pnpm world:goldens')
        else if (golden.sizeMismatch) check('golden', false, 'golden size mismatch')
        else check('golden', golden.ratio <= tuning.budgets.goldenThreshold, `${(golden.ratio * 100).toFixed(2)}% pixels differ (limit ${(tuning.budgets.goldenThreshold * 100).toFixed(2)}%)`)
        const pass = checks.every((c) => c.pass)
        if (!pass) failed = true
        const record = { name, scene, profile, period, url, gpu: useGpu, checks, pass, diagnostics, sim, consoleErrors, capturedAt: new Date().toISOString() }
        writeFileSync(resolve(evidenceDir, `${name}.json`), JSON.stringify(record, null, 2))
        results.push(record)
        log(`${pass ? 'PASS' : 'FAIL'} ${name}  ${checks.map((c) => `${c.pass ? '✓' : '✗'} ${c.id}: ${c.detail}`).join('  ')}`)
        await context.close()
      }
    }
  }
} finally {
  await browser.close()
}

// Period contrast: when several periods were captured, day and night must differ by
// more than budgets.periodPixelDelta of pixels, proving the period rig changes the world.
if (periods.includes('day') && periods.includes('night')) {
  for (const scene of scenes) {
    for (const profile of profiles) {
      const day = resolve(evidenceDir, `${scene}-${profile}-day.png`)
      const night = resolve(evidenceDir, `${scene}-${profile}-night.png`)
      if (!existsSync(day) || !existsSync(night)) continue
      const a = PNG.sync.read(readFileSync(day))
      const b = PNG.sync.read(readFileSync(night))
      const diff = new PNG({ width: a.width, height: a.height })
      const pixels = pixelmatch(a.data, b.data, diff.data, a.width, a.height, { threshold: 0.1 })
      const ratio = pixels / (a.width * a.height)
      const pass = ratio >= tuning.budgets.periodPixelDelta
      if (!pass) failed = true
      const record = { name: `${scene}-${profile}-period-delta`, pass, ratio, minimum: tuning.budgets.periodPixelDelta }
      writeFileSync(resolve(evidenceDir, `${record.name}.json`), JSON.stringify(record, null, 2))
      results.push({ name: record.name, pass, checks: [{ id: 'period-delta', pass, detail: `${(ratio * 100).toFixed(1)}% pixels differ day vs night (min ${(tuning.budgets.periodPixelDelta * 100).toFixed(0)}%)` }], diagnostics: null })
      log(`${pass ? 'PASS' : 'FAIL'} ${record.name}  ${pass ? '✓' : '✗'} period-delta: ${(ratio * 100).toFixed(1)}%`)
    }
  }
}

const rows = results.map(({ name, pass, checks, diagnostics, sim }) => ({
  name,
  pass,
  checks,
  drawCalls: diagnostics?.drawCalls ?? null,
  triangles: diagnostics?.triangles ?? null,
  p50Ms: diagnostics?.frameMsP50 ?? null,
  p95Ms: diagnostics?.frameMsP95 ?? null,
  fill: diagnostics?.footprintFill ?? null,
  violations: sim?.violations.length ?? null,
  gpuName: diagnostics?.gpu ?? null,
}))
writeFileSync(resolve(evidenceDir, 'summary.json'), JSON.stringify({ gpu: useGpu, baseUrl, capturedAt: new Date().toISOString(), results: rows }, null, 2))

// One table a reader can scan without opening a single PNG.
const cell = (value, width) => String(value ?? '—').padStart(width)
log('')
log(`${'case'.padEnd(24)}${cell('draws', 7)}${cell('tris', 9)}${cell('p50', 7)}${cell('p95', 7)}${cell('fill', 6)}${cell('viol', 6)}  verdict`)
for (const row of rows) {
  if (row.drawCalls === null) continue
  const failing = row.checks.filter((c) => !c.pass).map((c) => c.id).join(',')
  log(`${row.name.padEnd(24)}${cell(row.drawCalls, 7)}${cell(row.triangles, 9)}${cell(row.p50Ms?.toFixed(1), 7)}${cell(row.p95Ms?.toFixed(1), 7)}${cell(row.fill === null ? null : `${Math.round(row.fill * 100)}%`, 6)}${cell(row.violations, 6)}  ${row.pass ? 'PASS' : `FAIL ${failing}`}`)
}
log(`${results.filter((r) => r.pass).length}/${results.length} passed; evidence in ${evidenceDir}${useGpu ? '' : ' (SwiftShader: frame times informational)'}`)
process.exit(failed ? 1 : 0)
