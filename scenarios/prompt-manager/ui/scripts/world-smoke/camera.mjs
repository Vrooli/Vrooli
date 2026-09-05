#!/usr/bin/env node
/** Focus and mouse grammar integration against the built UI; not physical touch verification. */
import { chromium } from 'playwright-core'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'

const args = process.argv.slice(2)
const opt = (name, fallback) => {
  const index = args.indexOf(name)
  return index >= 0 && args[index + 1] ? args[index + 1] : fallback
}
const configuredBase = opt('--base-url', null)
const baseUrl = configuredBase ?? `http://localhost:${execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()}`
const root = resolve(opt('--evidence-dir', resolve(import.meta.dirname, '../../evidence/world-camera')))
if (existsSync(resolve(root, 'result.json'))) throw new Error(`camera evidence already exists at ${root}; choose a fresh --evidence-dir`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({
  executablePath: process.env.WORLD_SMOKE_CHROME ?? '/usr/bin/google-chrome',
  headless: true,
  args: ['--headless=new', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl', '--no-sandbox'],
})
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 }, deviceScaleFactor: 1 })
const errors = []
page.on('pageerror', error => errors.push(error.message))
const snapshots = {}
const checks = []
const check = (name, pass, detail) => checks.push({ name, pass, detail })
async function settle() {
  await page.evaluate(() => new Promise(resolve => {
    let frames = 0
    const tick = () => { if (++frames >= 120) resolve(); else requestAnimationFrame(tick) }
    requestAnimationFrame(tick)
  }))
}
async function snapshot(name) {
  await settle()
  const value = await page.evaluate(() => {
    const d = window.__worldDiagnostics
    return { position: d.cameraPosition, target: d.cameraTarget, renderer: d.gpu, ready: d.ready }
  })
  value.distance = Math.hypot(...value.position.map((v, i) => v - value.target[i]))
  snapshots[name] = value
  await page.screenshot({ path: resolve(root, `${name}.png`) })
  return value
}
const difference = (a, b) => Math.hypot(...a.map((v, i) => v - b[i]))
try {
  await page.goto(`${baseUrl}/world?scene=park&profile=high&period=day&actors=25&seed=1&intro=0&capture=1&diag=1&focus=demo-0-0`)
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
  const focus = await snapshot('focused')
  check('actor-sized focus distance', focus.distance >= 1.5 && focus.distance < 10, focus.distance)
  await page.getByRole('checkbox', { name: 'Follow', exact: true }).check()
  await settle()
  await page.mouse.move(800, 450)
  await page.mouse.down({ button: 'left' })
  await page.mouse.move(1000, 480, { steps: 20 })
  await page.mouse.up({ button: 'left' })
  const orbit = await snapshot('orbit-follow')
  check('left drag orbits', difference(focus.position, orbit.position) > 0.1, difference(focus.position, orbit.position))
  check('resting follow retains target while orbiting', difference(focus.target, orbit.target) < 0.05, difference(focus.target, orbit.target))
  const rested = await snapshot('orbit-rested')
  check('resting follow does not overwrite orbit', difference(orbit.position, rested.position) < 0.05, difference(orbit.position, rested.position))
  await page.mouse.move(850, 400)
  await page.mouse.wheel(0, -120)
  const zoom = await snapshot('wheel-zoom')
  check('wheel dollies inward', zoom.distance < orbit.distance, { before: orbit.distance, after: zoom.distance })
  await page.mouse.move(800, 450)
  await page.mouse.down({ button: 'right' })
  await page.mouse.move(880, 450, { steps: 20 })
  await page.mouse.up({ button: 'right' })
  const pan = await snapshot('right-pan')
  check('right drag pans', difference(zoom.target, pan.target) > 0.05, difference(zoom.target, pan.target))
  await page.mouse.move(800, 450)
  await page.mouse.down({ button: 'middle' })
  await page.mouse.move(880, 450, { steps: 20 })
  await page.mouse.up({ button: 'middle' })
  const middlePan = await snapshot('middle-pan')
  check('middle drag pans', difference(pan.target, middlePan.target) > 0.05, difference(pan.target, middlePan.target))
  check('no browser errors', errors.length === 0, errors)
} catch (error) {
  check('camera integration completed', false, error instanceof Error ? error.message : String(error))
} finally {
  writeFileSync(resolve(root, 'result.json'), JSON.stringify({ capturedAt: new Date().toISOString(), checks, snapshots, errors, physicalTouchVerified: false }, null, 2))
  await browser.close()
}
for (const entry of checks) console.log(`${entry.pass ? 'PASS' : 'FAIL'} ${entry.name}: ${JSON.stringify(entry.detail)}`)
if (checks.some(entry => !entry.pass)) process.exitCode = 1
