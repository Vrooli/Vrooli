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
 *   node scripts/world-smoke/run.mjs --scene park --profile medium --seeds 1,7,99,12345 --sweep 25,100,400,1000
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
const noVsync = flag('--no-vsync')
const onlyScene = opt('--scene', null)
const onlyProfile = opt('--profile', null)
const onlyPeriod = opt('--period', null)
const periods = onlyPeriod ? [onlyPeriod] : (flag('--all-periods') ? ['dawn', 'day', 'dusk', 'night'] : ['day'])
const seed = opt('--seed', '1')
const seedsRaw = opt('--seeds', null)
const seeds = seedsRaw
  ? seedsRaw.split(',').map((value) => value.trim()).filter(Boolean)
  : [seed]
if (seedsRaw && seeds.length === 0) throw new Error('world-smoke: --seeds requires a comma-separated list of seed values')
const actors = opt('--actors', null)
const sweepRaw = opt('--sweep', null)
const sweepActors = sweepRaw
  ? sweepRaw.split(',').map((value) => Number.parseInt(value, 10)).filter((value) => Number.isFinite(value) && value > 0)
  : null
if (sweepRaw && sweepActors.length === 0) throw new Error('world-smoke: --sweep requires a comma-separated list of positive actor counts')
if (!actors && !sweepActors) throw new Error('world-smoke: --actors is required for budget/golden-gated cases')
const actorCounts = sweepActors ?? [Number.parseInt(actors, 10)]
const emptyStage = flag('--empty-stage')
const extraQuery = opt('--query', '')
const requestedWeather = new URLSearchParams(extraQuery).get('weather')
const weatherMatrix = flag('--weather-matrix')
if (seedsRaw && (weatherMatrix || periods.length > 1)) {
  throw new Error('world-smoke: --seeds is an invariant observability matrix and cannot be combined with weather or period matrices')
}
const baseUrl = opt('--base-url', process.env.WORLD_SMOKE_BASE_URL ?? resolveBaseUrl())
const width = 1600
const height = 1000
const readyTimeoutMs = Number(opt('--ready-timeout', '90000'))
const settleFrames = Number(opt('--settle-frames', '30'))

function resolveBaseUrl() {
  try {
    const port = execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()
    if (/^\d+$/.test(port)) return `http://localhost:${port}`
  } catch {
    // fall through
  }
  return 'http://localhost:21235'
}

const scenes = weatherMatrix ? [onlyScene ?? 'park'] : onlyScene ? [onlyScene] : Object.keys(tuning.budgets.scenes)
const profiles = weatherMatrix ? [onlyProfile ?? 'high'] : onlyProfile ? [onlyProfile] : Object.keys(tuning.quality.profiles)

const chromeArgs = useGpu
  ? ['--headless=new', '--ignore-gpu-blocklist', '--enable-gpu-rasterization', '--use-angle=vulkan', '--use-vulkan=native', '--enable-features=Vulkan,VulkanFromANGLE,DefaultANGLEVulkan', '--no-sandbox']
  : ['--use-gl=angle', '--use-angle=swiftshader', '--enable-unsafe-swiftshader', '--ignore-gpu-blocklist', '--no-sandbox']
if (noVsync) chromeArgs.push('--disable-gpu-vsync', '--disable-frame-rate-limit')
chromeArgs.push('--renderer-process-limit=1', '--num-raster-threads=1')

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
        for (const worldSeed of seeds) {
        for (const actorCount of actorCounts) {
        for (const weatherState of weatherMatrix ? ['clear', 'cloudy', 'rain', 'snow'] : [requestedWeather]) {
        const periodName = periods.length > 1 || onlyPeriod ? `${scene}-${profile}-${period}` : `${scene}-${profile}`
        const baseName = weatherState ? `${periodName}-weather-${weatherState}` : periodName
        const seedName = seedsRaw ? `${baseName}-seed${worldSeed}` : baseName
        const name = sweepActors || seedsRaw ? `${seedName}-${actorCount}actors` : seedName
        const context = await browser.newContext({ viewport: { width, height }, deviceScaleFactor: 1, reducedMotion: 'no-preference' })
        const page = await context.newPage()
        const consoleErrors = []
        page.on('pageerror', (err) => consoleErrors.push(String(err)))
        page.on('console', (msg) => { if (msg.type() === 'error') consoleErrors.push(msg.text()) })
        // Golden and budget cases pin weather so live swarm failures cannot drift evidence.
        const query = new URLSearchParams({ scene, profile, period, intro: '0', seed: worldSeed, diag: '1', weather: 'clear', pressure: '0' })
        query.set('actors', String(actorCount))
        for (const [k, v] of new URLSearchParams(extraQuery)) query.set(k, v)
        if (weatherState) query.set('weather', weatherState)
        const url = `${baseUrl}/world?${query.toString()}`
        const started = Date.now()
        await page.goto(url, { waitUntil: 'domcontentloaded' })
        let ready = false
        try {
          await page.waitForFunction(() => {
            const diagnostics = globalThis.__worldDiagnostics
            return diagnostics?.ready === true || (diagnostics?.webgl !== null && diagnostics?.webgl?.ok === false)
          }, null, { timeout: readyTimeoutMs })
          ready = await page.evaluate(() => globalThis.__worldDiagnostics?.ready === true)
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
          await page.evaluate(() => globalThis.__worldDiagnostics?.measure())
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
        const gated = !sweepActors && !seedsRaw
        const checks = []
        const check = (id, pass, detail) => checks.push({ id, pass, detail })
        const webgl = diagnostics?.webgl
        check('webgl', webgl?.ok === true, webgl?.ok ? 'available' : `webgl-unavailable: ${webgl?.reason ?? 'probe did not report'}`)
        check('ready', ready, ready ? `ready after ${Date.now() - started} ms` : webgl && !webgl.ok ? `webgl-unavailable: ${webgl.reason}` : `not ready within ${readyTimeoutMs} ms`)
        check('no-page-errors', consoleErrors.length === 0, consoleErrors.slice(0, 3).join(' | ') || 'none')
        if (diagnostics && budget && gated) {
          check('budget-provenance', budget.provenance.actors === actorCount, `actors ${actorCount} === calibrated ${budget.provenance.actors}`)
          const drawCallBudget = emptyStage ? tuning.budgets.emptyStageDrawCalls : budget.drawCalls
          check('draw-calls', diagnostics.drawCalls <= drawCallBudget, `${diagnostics.drawCalls} <= ${drawCallBudget}${emptyStage ? ' (empty stage)' : ''}`)
          check('triangles', diagnostics.triangles <= budget.triangles, `${diagnostics.triangles} <= ${budget.triangles}`)
          const gpuTimerAvailable = diagnostics.gpuTimerReason === '' && diagnostics.gpuMsP95 > 0
          if (useGpu && gpuTimerAvailable) {
            check('p95-ms', diagnostics.gpuMsP95 <= budget.p95Ms, `${diagnostics.gpuMsP95.toFixed(2)} <= ${budget.p95Ms} (GPU timer)`)
          } else if (useGpu && noVsync) {
            check('p95-ms', diagnostics.frameMsP95 <= budget.p95Ms, `${diagnostics.frameMsP95.toFixed(1)} <= ${budget.p95Ms} (vsync-off fallback: ${diagnostics.gpuTimerReason || 'GPU timer returned no samples'})`)
          } else if (useGpu) {
            check('gpu-timer', false, `${diagnostics.gpuTimerReason || 'GPU timer returned no samples'}; rerun with --no-vsync for labelled fallback`)
          } else {
            check('p95-ms-informational', true, `${diagnostics.frameMsP95.toFixed(1)} ms under SwiftShader (not gated)`)
          }
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
        if (!gated) check('observability-case', true, 'sweep rows record cost without applying golden or scene-budget gates')
        else if (periods.length > 1 && golden.missing) check('period-contrast-case', true, 'no per-period golden; the day/night pixel delta is the visual gate')
        else if (golden.missing) check('golden', updateGoldens, updateGoldens ? 'golden written' : 'golden missing; run pnpm world:goldens')
        else if (golden.sizeMismatch) check('golden', false, 'golden size mismatch')
        else check('golden', golden.ratio <= tuning.budgets.goldenThreshold, `${(golden.ratio * 100).toFixed(2)}% pixels differ (limit ${(tuning.budgets.goldenThreshold * 100).toFixed(2)}%)`)
        const pass = checks.every((c) => c.pass)
        if (!pass) failed = true
        const timingMethod = diagnostics?.gpuTimerReason === '' && diagnostics?.gpuMsP95 > 0 ? 'gpu-timer' : useGpu && noVsync ? 'vsync-off-fallback' : useGpu ? 'unavailable' : 'swiftshader-informational'
        const record = { name, scene, profile, period, weather: weatherState ?? 'clear', seed: worldSeed, actors: actorCount, url, gpu: useGpu, noVsync, timingMethod, budgetProvenance: gated ? budget?.provenance ?? null : null, checks, pass, diagnostics, sim, consoleErrors, capturedAt: new Date().toISOString() }
        writeFileSync(resolve(evidenceDir, `${name}.json`), JSON.stringify(record, null, 2))
        results.push(record)
        log(`${pass ? 'PASS' : 'FAIL'} ${name}  ${checks.map((c) => `${c.pass ? '✓' : '✗'} ${c.id}: ${c.detail}`).join('  ')}`)
        await context.close()
        }
        }
        }
      }
    }
  }
} finally {
  await browser.close()
}

if (weatherMatrix) {
  const weatherStates = ['clear', 'cloudy', 'rain', 'snow']
  for (let left = 0; left < weatherStates.length; left += 1) for (let right = left + 1; right < weatherStates.length; right += 1) {
    const aName = weatherStates[left]
    const bName = weatherStates[right]
    const prefix = `${scenes[0]}-${profiles[0]}${onlyPeriod ? `-${onlyPeriod}` : ''}-weather-`
    const a = PNG.sync.read(readFileSync(resolve(evidenceDir, `${prefix}${aName}.png`)))
    const b = PNG.sync.read(readFileSync(resolve(evidenceDir, `${prefix}${bName}.png`)))
    const pixels = pixelmatch(a.data, b.data, null, a.width, a.height, { threshold: 0.12 })
    const ratio = pixels / (a.width * a.height)
    const pass = ratio >= tuning.budgets.periodPixelDelta
    if (!pass) failed = true
    const name = `${prefix}${aName}-vs-${bName}`
    const checks = [{ id: 'weather-delta', pass, detail: `${(ratio * 100).toFixed(2)}% pixels differ (min ${(tuning.budgets.periodPixelDelta * 100).toFixed(0)}%)` }]
    results.push({ name, pass, checks, diagnostics: null, sim: null })
    writeFileSync(resolve(evidenceDir, `${name}.json`), JSON.stringify({ name, pass, ratio, minimum: tuning.budgets.periodPixelDelta }, null, 2))
    log(`${pass ? 'PASS' : 'FAIL'} ${name}  ${checks[0].detail}`)
  }
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
  gpuMsP50: diagnostics?.gpuMsP50 ?? null,
  gpuMsP95: diagnostics?.gpuMsP95 ?? null,
  gpuSamples: diagnostics?.gpuSamples ?? null,
  gpuTimerReason: diagnostics?.gpuTimerReason ?? null,
  fill: diagnostics?.footprintFill ?? null,
  violations: sim?.violations.length ?? null,
  gpuName: diagnostics?.gpu ?? null,
}))
writeFileSync(resolve(evidenceDir, 'summary.json'), JSON.stringify({ gpu: useGpu, baseUrl, capturedAt: new Date().toISOString(), results: rows }, null, 2))

// One table a reader can scan without opening a single PNG.
const cell = (value, width) => String(value ?? '—').padStart(width)
log('')
log(`${'case'.padEnd(32)}${cell('draws', 7)}${cell('tris', 9)}${cell('CPU95', 8)}${cell('GPU95', 8)}${cell('GPU n', 7)}${cell('fill', 6)}${cell('viol', 6)}  verdict`)
for (const row of rows) {
  if (row.drawCalls === null) continue
  const failing = row.checks.filter((c) => !c.pass).map((c) => c.id).join(',')
  log(`${row.name.padEnd(32)}${cell(row.drawCalls, 7)}${cell(row.triangles, 9)}${cell(row.p95Ms?.toFixed(1), 8)}${cell(row.gpuMsP95?.toFixed(2), 8)}${cell(row.gpuSamples, 7)}${cell(row.fill === null ? null : `${Math.round(row.fill * 100)}%`, 6)}${cell(row.violations, 6)}  ${row.pass ? 'PASS' : `FAIL ${failing}`}`)
}
if (sweepActors) {
  log('')
  log('Scaling sweep (GPU p95 per actor uses the timer when available; otherwise CPU p95 is labelled by the record method)')
  log(`${'case'.padEnd(32)}${cell('actors', 8)}${cell('draws', 8)}${cell('tris', 10)}${cell('GPU95', 9)}${cell('per actor', 12)}`)
  for (const row of rows.filter((entry) => entry.drawCalls !== null)) {
    const actorCount = Number(row.name.match(/-(\d+)actors$/)?.[1] ?? 0)
    const cost = row.gpuMsP95 > 0 ? row.gpuMsP95 : row.p95Ms
    log(`${row.name.padEnd(32)}${cell(actorCount, 8)}${cell(row.drawCalls, 8)}${cell(row.triangles, 10)}${cell(row.gpuMsP95?.toFixed(2), 9)}${cell(actorCount > 0 ? (cost / actorCount).toFixed(4) : '—', 12)}`)
  }
}
log(`${results.filter((r) => r.pass).length}/${results.length} passed; evidence in ${evidenceDir}${useGpu ? '' : ' (SwiftShader: frame times informational)'}`)
process.exit(failed ? 1 : 0)
