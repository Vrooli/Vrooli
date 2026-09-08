#!/usr/bin/env node
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const i = process.argv.indexOf('--evidence-dir'), root = resolve(i < 0 ? `evidence/ground-wildlife-${Date.now()}` : process.argv[i + 1])
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = []
page.on('pageerror', e => errors.push(e.message))
page.on('console', m => { if (m.type() === 'error' && /shader|WebGL|THREE/.test(m.text())) errors.push(m.text()) })
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
async function inspect(family) {
  return page.evaluate(family => {
    const pending = [...window.__cameraFixtureRoots].map(r => r.current); let render, world, clock
    while (pending.length) { const f = pending.pop(); for (const value of [f.memoizedProps?.store, f.memoizedProps?.value]) {
      if (typeof value?.getState !== 'function') continue
      if (value.getState().scene?.isScene) render = value
      if (value.getState().actors && value.getState().nav) world = value
    }
    if (f.memoizedProps?.clock?.snapshot) clock = f.memoizedProps.clock
    if (f.child) pending.push(f.child); if (f.sibling) pending.push(f.sibling) }
    const mesh = render.getState().scene.getObjectByName(`ambient-${family}`), intersections = []
    mesh.raycast({}, intersections)
    return { count: mesh.count, capacity: mesh.instanceMatrix.count, positions: [...mesh.instanceMatrix.array.slice(0, mesh.count * 16)],
      hits: intersections.length, time: clock.snapshot(), cost: window.__worldDiagnostics.ambientCosts().families.find(row => row.family === family),
      violations: window.__worldSim.violations(), nav: world.getState().nav.walkable.reduce((sum, value) => sum + value, 0) }
  }, family)
}
async function settings() { await page.getByTitle('World Settings', { exact: true }).click() }
try {
  for (const scene of ['park', 'office']) {
    await page.goto(`http://localhost:21235/world?scene=${scene}&actors=25&seed=1&diag=1&period=day&weather=clear&profile=high&workbench=1`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
    for (const [label, family] of [['foraging squirrel', 'squirrels'], ['climbing squirrel', 'squirrels'], ['idle rabbit', 'rabbits'], ['hopping rabbit', 'rabbits']]) {
      await settings()
      if (!await page.getByRole('button', { name: `Preview ${label}`, exact: true }).isVisible()) await page.getByText('World workbench', { exact: true }).click()
      await page.getByRole('button', { name: `Preview ${label}`, exact: true }).click()
      await page.getByRole('status').filter({ hasText: 'Preview ready:' }).waitFor({ timeout: 30000 })
      await page.waitForTimeout(2200)
      await page.keyboard.press('Escape')
      const before = await inspect(family)
      check(`${scene} ${label} renders within its pool`, before.count > 0 && before.count <= before.capacity && before.cost.active <= before.cost.limit, before.cost)
      check(`${scene} ${label} does not intercept picking`, before.hits === 0)
      await page.screenshot({ path: resolve(root, `${scene}-${label.replaceAll(' ', '-')}.png`) })
      await page.waitForTimeout(250)
      const frozen = await inspect(family)
      check(`${scene} ${label} frozen pose is stable`, JSON.stringify(before.positions) === JSON.stringify(frozen.positions) && before.time.utcMilliseconds === frozen.time.utcMilliseconds, { before: { count: before.count, time: before.time }, after: { count: frozen.count, time: frozen.time }, delta: Math.max(...before.positions.map((v, i) => Math.abs(v - (frozen.positions[i] ?? v)))) })
      await settings()
      await page.getByRole('button', { name: 'Play from here', exact: true }).click()
      await page.keyboard.press('Escape')
      await page.waitForTimeout(900)
      const moving = await inspect(family)
      check(`${scene} ${label} animates from the same scheduled pose`, moving.time.utcMilliseconds > before.time.utcMilliseconds && JSON.stringify(moving.positions) !== JSON.stringify(before.positions), { before: before.time, after: moving.time })
      check(`${scene} ${label} preserves navigation and simulation`, moving.nav === before.nav && moving.violations.length === 0, moving.violations)
    }
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.waitForTimeout(400)
    check(scene + ' reduced motion suppresses ground animal motion', (await inspect('squirrels')).count === 0 && (await inspect('rabbits')).count === 0)
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await settings()
    await page.getByRole('checkbox', { name: 'Ambient life', exact: true }).uncheck()
    await page.waitForTimeout(400)
    check(scene + ' ambient off clears both ground animal pools', (await inspect('squirrels')).count === 0 && (await inspect('rabbits')).count === 0)
    await page.keyboard.press('Escape')
  }
  check('no page or GPU errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors }, null, 2)); await browser.close() }
if (checks.some(check => !check.pass)) process.exitCode = 1
