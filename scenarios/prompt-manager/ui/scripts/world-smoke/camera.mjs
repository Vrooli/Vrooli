#!/usr/bin/env node
/** Focus and mouse grammar integration against the built UI; not physical touch verification. */
import { chromium } from 'playwright-core'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'
import { installFixtureHook, insertFixture, removeFixture, overlaps, segmentOverlaps } from './camera-fixtures.mjs'

const args = process.argv.slice(2)
const captureMode = args.includes('--capture')
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
const context = await browser.newContext({ viewport: { width: 1600, height: 1000 }, deviceScaleFactor: 1 })
await context.tracing.start({ screenshots: true, snapshots: true, sources: true })
const page = await context.newPage()
if (args.includes('--obstruction-fixtures')) await installFixtureHook(page)
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
    return { position: d.cameraPosition, target: d.cameraTarget, poseRoute: d.cameraPoseRoute, obstacleGuard: d.cameraObstacleGuard, renderer: d.gpu, ready: d.ready, preparedAssets: d.preparedAssets, captureBridge: Boolean(window.__worldCapture) }
  })
  value.distance = Math.hypot(...value.position.map((v, i) => v - value.target[i]))
  snapshots[name] = value
  await page.screenshot({ path: resolve(root, `${name}.png`) })
  return value
}
const difference = (a, b) => Math.hypot(...a.map((v, i) => v - b[i]))
try {
  if (!args.includes('--performance-only')) {
  const query = new URLSearchParams({ scene: 'park', profile: 'high', period: 'day', actors: '25', seed: '1', diag: '1', focus: 'demo-0-0' })
  if (captureMode) { query.set('intro', '0'); query.set('capture', '1') }
  await page.goto(`${baseUrl}/world?${query}`)
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
  const focus = await snapshot('focused')
  check('requested rendering mode', focus.captureBridge === captureMode, { captureBridge: focus.captureBridge, captureMode })
  check('prepared assets retained', focus.preparedAssets?.references > 0, focus.preparedAssets)
  check('actor-sized focus distance', focus.distance >= 1.5 && focus.distance < 10, focus.distance)
  check('focus finishes at a guarded destination', focus.poseRoute?.status === 'complete' && focus.poseRoute.owner === 'focus', focus.poseRoute)
  await page.getByRole('checkbox', { name: 'Follow', exact: true }).check()
  await settle()
  await page.mouse.move(800, 450)
  await page.mouse.down({ button: 'left' })
  await page.mouse.move(1000, 480, { steps: 20 })
  await page.mouse.up({ button: 'left' })
  const orbit = await snapshot('orbit-follow')
  check('left drag orbits', difference(focus.position, orbit.position) > 0.1, difference(focus.position, orbit.position))
  check('camera motion queries rendered room obstacles', orbit.obstacleGuard?.boxes > 0, orbit.obstacleGuard)
  check('resting follow retains target while orbiting', difference(focus.target, orbit.target) < 0.05, difference(focus.target, orbit.target))
  const rested = await snapshot('orbit-rested')
  check('resting follow does not overwrite orbit', difference(orbit.position, rested.position) < 0.05, difference(orbit.position, rested.position))
  await page.mouse.move(850, 400)
  await page.mouse.wheel(0, -120)
  const zoom = await snapshot('wheel-zoom')
  check('wheel dollies inward', zoom.distance < orbit.distance, { before: orbit.distance, after: zoom.distance })
  await page.mouse.wheel(0, 120)
  const reversed = await snapshot('wheel-reversed')
  check('wheel reversal dollies outward', reversed.distance > zoom.distance, { before: zoom.distance, after: reversed.distance })
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
  if (args.includes('--zoom-preference')) {
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(`${baseUrl}/world?${query}`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
    await settle()
    for (const choice of ['Center', 'Cursor']) {
      // Keep the settings panel open; the off-center probe must hit the canvas.
      const radio = page.getByRole('radio', { name: choice, exact: true })
      if (!await radio.isVisible()) await page.getByTitle('World Settings', { exact: true }).click()
      await radio.check()
      await settle()
      const canvasHit = await page.evaluate(() => document.elementFromPoint(650, 350)?.tagName === 'CANVAS')
      check(`${choice} wheel probe reaches canvas`, canvasHit, { x: 650, y: 350 })
      if (!canvasHit) throw new Error('Zoom preference probe is covered by UI')
      const generation = await page.evaluate(() => window.__worldSim.generation().count)
      const before = await snapshot(`zoom-${choice}-before`)
      await page.mouse.move(650, 350)
      await page.mouse.wheel(0, 120)
      const after = await snapshot(`zoom-${choice}-after`)
      const targetChange = difference(before.target, after.target)
      check(`${choice} zoom changes distance with expected target behavior`,
        after.distance > before.distance && (choice === 'Center' ? targetChange < 0.001 : targetChange > 0.05),
        { before: before.distance, after: after.distance, targetChange })
      check(`${choice} preference avoids world generation`,
        generation === await page.evaluate(() => window.__worldSim.generation().count), generation)
    }
  }
  if (args.includes('--automatic-poses')) {
    await page.setViewportSize({ width: 1400, height: 900 })
    for (const scene of ['park', 'office']) {
      await page.goto(`${baseUrl}/world?scene=${scene}&profile=high&period=day&actors=25&seed=1&diag=1`)
      await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
      const intro = await snapshot(`${scene}-intro`)
      check(`${scene} intro completes through the guarded route`, intro.poseRoute?.owner === 'intro' && intro.poseRoute.status === 'complete', intro.poseRoute)
      const canvasHit = await page.evaluate(() => document.elementFromPoint(800, 700)?.tagName === 'CANVAS')
      if (!canvasHit) throw new Error('Home approach probe is covered by UI')
      await page.mouse.move(800, 700)
      for (let step = 0; step < 3; step++) { await page.mouse.wheel(0, -1200); await settle() }
      const close = await snapshot(`${scene}-before-home`)
      check(`${scene} exploration leaves overview before home`, difference(close.position, intro.position) > 1, { before: intro.position, after: close.position })
      await page.keyboard.press('Escape')
      await page.waitForFunction(() => window.__worldDiagnostics?.cameraPoseRoute?.owner === 'overview' && window.__worldDiagnostics.cameraPoseRoute.status === 'complete', null, { timeout: 15000 })
      const home = await snapshot(`${scene}-home`)
      check(`${scene} home reaches its requested framing`, difference(home.position, intro.position) < 0.05 && difference(home.target, intro.target) < 0.05,
        { route: home.poseRoute, positionDifference: difference(home.position, intro.position), targetDifference: difference(home.target, intro.target) })
    }
  }
  if (args.includes('--obstruction-fixtures')) {
    await page.setViewportSize({ width: 1400, height: 900 })
    for (const scene of ['park', 'office']) {
      await page.goto(`${baseUrl}/world?scene=${scene}&profile=high&period=day&actors=25&seed=1&diag=1`)
      await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
      const overview = await snapshot(`${scene}-fixture-overview`)
      if (!await page.evaluate(() => document.elementFromPoint(800, 700)?.tagName === 'CANVAS')) throw new Error('Fixture approach covered by UI')
      await page.mouse.move(800, 700)
      for (let step = 0; step < 3; step++) { await page.mouse.wheel(0, -1200); await settle() }
      const from = await snapshot(`${scene}-fixture-start`)
      const center = from.position.map((value, axis) => (value + overview.position[axis]) / 2)
      const fixture = await insertFixture(page, center, [4, 20, 4])
      check(`${scene} fixture obstructs direct travel with clear endpoints`, !overlaps(from.position, fixture) && !overlaps(overview.position, fixture) && segmentOverlaps(from.position, overview.position, fixture), fixture)
      await page.keyboard.press('Escape')
      await page.waitForFunction(() => window.__worldDiagnostics?.cameraPoseRoute?.owner === 'overview' && window.__worldDiagnostics.cameraPoseRoute.status === 'complete', null, { timeout: 15000 })
      const home = await snapshot(`${scene}-fixture-home`)
      const frames = await removeFixture(page)
      snapshots[`${scene}-detour-trace`] = { fixture, frames }
      check(`${scene} forced home detours around rendered box`, home.poseRoute.waypoints > 1 && frames.every(frame => !overlaps(frame.position, fixture)), { route: home.poseRoute, frames: frames.length })
      check(`${scene} inter-frame segments retain fixture clearance`, frames.every((frame, index) => !segmentOverlaps(index ? frames[index - 1].position : from.position, frame.position, fixture)))
      check(`${scene} detour reaches requested overview`, difference(home.position, overview.position) < 0.05 && difference(home.target, overview.target) < 0.05)
      const overlap = await insertFixture(page, home.position, [4, 4, 4])
      await page.waitForFunction(y => window.__worldDiagnostics.cameraPosition[1] > y + 2, home.position[1], { timeout: 5000 })
      const recovered = await snapshot(`${scene}-fixture-recovered`)
      const recoveryFrames = await removeFixture(page)
      snapshots[`${scene}-recovery-trace`] = { fixture: overlap, frames: recoveryFrames }
      const eyeDelta = recovered.position.map((v, i) => v - home.position[i])
      const targetDelta = recovered.target.map((v, i) => v - home.target[i])
      check(`${scene} inserted overlap recovers while preserving view direction`, !overlaps(recovered.position, overlap) && difference(eyeDelta, targetDelta) < 0.001 && Math.abs(eyeDelta[0]) < 0.001 && Math.abs(eyeDelta[2]) < 0.001, { eyeDelta, targetDelta })
    }
  }
  if (args.includes('--surface-zoom')) {
    await page.setViewportSize({ width: 1400, height: 900 })
    for (const scene of ['park', 'office']) {
      await page.goto(`${baseUrl}/world?scene=${scene}&profile=high&period=day&actors=25&seed=1&diag=1`)
      await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
      await settle()
      const canvasHit = await page.evaluate(() => document.elementFromPoint(800, 700)?.tagName === 'CANVAS')
      if (!canvasHit) throw new Error('Surface zoom probe is covered by UI')
      const samples = []
      await page.mouse.move(800, 700)
      for (let step = 0; step < 6; step++) {
        await page.mouse.wheel(0, -1200)
        await settle()
        samples.push(await page.evaluate(() => ({
          position: window.__worldDiagnostics.cameraPosition,
          guard: window.__worldDiagnostics.cameraZoomGuard,
        })))
      }
      snapshots[`${scene}-surface-zoom`] = samples
      check(`${scene} surface guard limits inward zoom`, samples.every(s => s.guard?.limited), samples)
      check(`${scene} surface approach converges`, samples[5].guard.travel < samples[0].guard.travel * 0.01, samples.map(s => s.guard.travel))
      await page.mouse.wheel(0, 120)
      const reversed = await snapshot(`${scene}-surface-reversal`)
      check(`${scene} zoom reverses away from surface`, difference(samples[5].position, reversed.position) > 0.05, reversed.position)
    }
  }
  if (args.includes('--cache-switches')) {
    if (!await page.getByRole('radiogroup', { name: 'Scene', exact: true }).isVisible()) {
      await page.getByTitle('World Settings', { exact: true }).click()
    }
    let warm = null
    for (const [index, scene] of ['office', 'park', 'office', 'park'].entries()) {
      await page.getByRole('radiogroup', { name: 'Scene', exact: true }).getByRole('radio', { name: scene === 'park' ? 'Park' : 'Office', exact: true }).click()
      await page.waitForFunction(expected => window.__worldDiagnostics?.scene === expected && window.__worldDiagnostics?.ready, scene, { timeout: 90000 })
      const value = await snapshot(`switch-${index}-${scene}`)
      if (index === 1) warm = value.preparedAssets
      if (index > 1) check(`warm ${scene} avoids preparation`, value.preparedAssets.misses === warm.misses, { before: warm, after: value.preparedAssets })
    }
  }
  if (args.includes('--editor-cancellation')) {
    // A declared seed, viewport and aerial pose make the empty part of Team A's
    // handle reproducible. Reaching editing ownership proves the hit succeeded.
    await page.setViewportSize({ width: 1400, height: 900 })
    await page.goto(`${baseUrl}/world?scene=park&profile=high&period=day&actors=25&seed=1&diag=1`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
    await page.getByRole('button', { name: 'Edit layout', exact: true }).click()
    await settle()
    const readEditor = () => page.evaluate(() => ({
      epoch: window.__worldDiagnostics.presentationEpoch,
      position: window.__worldDiagnostics.cameraPosition,
      target: window.__worldDiagnostics.cameraTarget,
      owner: window.__worldDiagnostics.cameraOwnership,
      generation: window.__worldSim.generation().count,
    }))
    for (const cancellation of ['Escape', 'pointercancel', 'blur', 'lostpointercapture']) {
      const before = await readEditor()
      await page.mouse.move(710, 365)
      await page.mouse.down()
      await page.waitForFunction(() => window.__worldDiagnostics.cameraOwnership?.owner === 'editing', null, { timeout: 5000 })
      await page.mouse.move(770, 405, { steps: 10 })
      const dragging = await readEditor()
      if (cancellation === 'Escape') await page.keyboard.press('Escape')
      else await page.evaluate(name => window.dispatchEvent(new Event(name)), cancellation)
      await page.mouse.up()
      await settle()
      const after = await readEditor()
      snapshots[`editor-${cancellation}`] = { before, dragging, after }
      check(`${cancellation} releases editing without generation or reframing`,
        after.owner.owner === 'explore' && after.epoch === before.epoch && after.generation === before.generation &&
        difference(before.position, after.position) < 0.05 && difference(before.target, after.target) < 0.05,
        { before, after })
    }
    const violations = await page.evaluate(() => window.__worldSim.violations())
    check('editor cancellation leaves valid world', violations.length === 0, violations)
  }
  }
  if (args.includes('--performance') || args.includes('--performance-only')) {
    if (captureMode) throw new Error('Interaction performance requires normal rendering')
    await page.setViewportSize({ width: 1600, height: 1000 })
    for (const scene of ['park', 'office']) {
      await page.goto(`${baseUrl}/world?scene=${scene}&profile=high&period=day&actors=${opt('--actors', '25')}&seed=1&diag=0&intro=0`)
      await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
      await settle()
      const canvas = await page.locator('canvas').boundingBox()
      const x = canvas.x + canvas.width * 0.55
      const y = canvas.y + canvas.height * 0.6
      for (const gesture of ['orbit', 'pan', 'zoom']) {
        await page.mouse.move(x, y)
        await page.evaluate(() => window.__worldDiagnostics.beginInteractionSample())
        if (gesture !== 'zoom') await page.mouse.down({ button: gesture === 'orbit' ? 'left' : 'right' })
        for (let step = 0; step < 120; step++) {
          if (gesture === 'zoom') await page.mouse.wheel(0, step % 40 < 20 ? -30 : 30)
          else await page.mouse.move(x + Math.sin(step / 15) * 100, y + Math.sin(step / 20) * 30)
          await page.evaluate(() => new Promise(resolve => requestAnimationFrame(resolve)))
        }
        if (gesture !== 'zoom') await page.mouse.up({ button: gesture === 'orbit' ? 'left' : 'right' })
        const result = await page.evaluate(() => ({ sample: window.__worldDiagnostics.endInteractionSample(), ao: window.__worldDiagnostics.ao, calls: window.__worldDiagnostics.drawCalls, triangles: window.__worldDiagnostics.triangles, renderer: window.__worldDiagnostics.gpu }))
        snapshots[`${scene}-${gesture}-performance`] = result
        check(`${scene} ${gesture} interval is populated`, result.sample.frames.count >= 60 && result.sample.inputToRenderMs.count > 0 && !result.sample.truncated, result)
        check(`${scene} ${gesture} responsive movement budget`, result.sample.frames.p95 <= 20 && result.sample.inputToRenderMs.p95 <= 50, result.sample)
        check(`${scene} ${gesture} tail-frame budget`, result.sample.frames.p99 <= 34 && result.sample.frames.over50ms / Math.max(1, result.sample.frames.count) <= 0.01, result.sample.frames)
        await settle()
        const aoRestored = await page.evaluate(() => window.__worldDiagnostics.ao)
        check(`${scene} ${gesture} restores AO after settling`, aoRestored, aoRestored)
      }
    }
  }
  check('no browser errors', errors.length === 0, errors)
} catch (error) {
  check('camera integration completed', false, error instanceof Error ? error.message : String(error))
  snapshots.failure = await page.evaluate(() => window.__worldDiagnostics ?? null).catch(() => null)
  await page.screenshot({ path: resolve(root, 'failure.png') }).catch(() => undefined)
} finally {
  await context.tracing.stop({ path: resolve(root, 'trace.zip') })
  writeFileSync(resolve(root, 'result.json'), JSON.stringify({ capturedAt: new Date().toISOString(), mode: captureMode ? 'capture' : 'normal', performanceParameters: { actors: Number(opt('--actors', '25')), profile: 'high', seed: 1, viewport: { width: 1600, height: 1000 }, dpr: 1, diagnosticsOverlay: false }, checks, snapshots, errors, physicalTouchVerified: false }, null, 2))
  await browser.close()
}
for (const entry of checks) console.log(`${entry.pass ? 'PASS' : 'FAIL'} ${entry.name}: ${JSON.stringify(entry.detail)}`)
if (checks.some(entry => !entry.pass)) process.exitCode = 1
