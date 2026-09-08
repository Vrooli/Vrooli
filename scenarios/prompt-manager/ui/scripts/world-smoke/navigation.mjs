#!/usr/bin/env node
/** Real browser navigation journeys. Synthetic rosters avoid management writes. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'
import { installFixtureHook, insertFixture, insertLogFixture, removeFixture } from './camera-fixtures.mjs'

const option = (name, fallback) => { const index = process.argv.indexOf(name); return index < 0 ? fallback : process.argv[index + 1] }
const root = resolve(option('--evidence-dir', `evidence/navigation-${Date.now()}`))
const base = option('--base-url', `http://localhost:${execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()}`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: process.env.WORLD_SMOKE_CHROME ?? '/usr/bin/google-chrome', headless: true,
  args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } })
const page = await context.newPage()
await installFixtureHook(page)
const checks = [], errors = [], samples = []
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`) }
page.on('pageerror', error => errors.push(error.message))
const distance = (a, b) => Math.hypot(...a.map((v, i) => v - b[i]))
// Pose and input receipt must come from one sample. The general render probe
// intentionally publishes its camera fields less often than navigation does.
const snapshot = () => page.evaluate(() => ({ position: window.__worldDiagnostics.cameraNavigation.position, target: window.__worldDiagnostics.cameraNavigation.target,
  navigation: window.__worldDiagnostics.cameraNavigation, frameP95: window.__worldDiagnostics.frameMsP95 }))
async function ready(scene = 'park') {
  await page.goto(`${base}/world?actors=25&profile=high&scene=${scene}&intro=0&diag=1&period=day`)
  await page.waitForFunction(() => window.__worldDiagnostics?.ready && window.__worldDiagnostics?.cameraNavigation !== null && window.__worldDiagnostics?.cameraPosition?.[1] > 0, null, { timeout: 60000 })
  await page.waitForTimeout(250)
}
async function home() {
  await page.getByRole('button', { name: 'Home view', exact: true }).click()
  await page.waitForFunction(() => window.__worldDiagnostics.cameraPoseRoute?.status === 'complete', null, { timeout: 10000 })
  await page.waitForTimeout(200)
}
async function mode(value) {
  await page.getByLabel('Camera mode', { exact: true }).selectOption(value)
  await page.waitForFunction(mode => window.__worldDiagnostics.cameraNavigation?.mode === mode, value, { timeout: 10000 })
  await page.waitForTimeout(200)
}
async function key(key, milliseconds = 350) {
  await page.locator('canvas').focus()
  await page.keyboard.down(key)
  await page.waitForTimeout(milliseconds)
  await page.keyboard.up(key)
  await page.waitForTimeout(150)
}
try {
  await ready()
  const initial = await snapshot()
  check('camera toolbar available', await page.getByRole('region', { name: 'Camera navigation' }).isVisible())
  await page.mouse.move(1000, 400)
  await page.mouse.wheel(0, -120)
  await page.waitForFunction(sequence => window.__worldDiagnostics.cameraNavigation.inputSequence > sequence, initial.navigation.inputSequence)
  const wheel = await snapshot()
  check('wheel moves inward with a fixed lens', distance(wheel.position, wheel.target) < distance(initial.position, initial.target) && wheel.navigation.lensZoom === 1)
  await page.waitForTimeout(600)
  check('wheel has no delayed motion after release', distance(wheel.position, (await snapshot()).position) < .01)
  await home()
  await page.getByText('Input settings', { exact: true }).click()
  await page.getByLabel('Navigation device').selectOption('trackpad')
  await page.getByLabel('Navigation zoom target').selectOption('center')
  await page.getByText('Input settings', { exact: true }).click()
  const centered = await snapshot()
  await page.mouse.move(1000, 400)
  await page.mouse.wheel(40, 30)
  await page.waitForFunction(sequence => window.__worldDiagnostics.cameraNavigation.inputSequence > sequence, centered.navigation.inputSequence)
  const pan = await snapshot()
  check('trackpad scroll pans without changing orbit distance', distance(centered.target, pan.target) > .1 && Math.abs(distance(centered.position, centered.target) - distance(pan.position, pan.target)) < .01)
  await home()
  const pinchStart = await snapshot()
  await page.keyboard.down('Control')
  await page.mouse.move(1000, 400)
  await page.mouse.wheel(0, -120)
  await page.waitForFunction(sequence => window.__worldDiagnostics.cameraNavigation.inputSequence > sequence, pinchStart.navigation.inputSequence)
  await page.keyboard.up('Control')
  const pinch = await snapshot()
  check('browser Ctrl-wheel pinch dollies rather than changing magnification', distance(pinch.position, pinch.target) < distance(pinchStart.position, pinchStart.target) && pinch.navigation.lensZoom === 1, { before: pinchStart, after: pinch })
  await home()
  const beforePan = await snapshot()
  await key('d')
  const afterRight = await snapshot()
  const facing = beforePan.target.map((v, i) => v - beforePan.position[i])
  const right = [-facing[2], 0, facing[0]]
  const projectRight = target => target.reduce((sum, value, i) => sum + (value - beforePan.target[i]) * right[i], 0)
  check('D pans right after focusing the world', projectRight(afterRight.target) > .1)
  await key('w')
  check('W pans up on screen', (await snapshot()).target[1] > afterRight.target[1])
  await home()
  const panLeftStart = await snapshot()
  await page.getByRole('button', { name: 'Pan left', exact: true }).click()
  await page.waitForTimeout(150)
  check('Pan left button moves the camera left', (await snapshot()).target.reduce((sum, value, i) => sum + (value - panLeftStart.target[i]) * right[i], 0) < -.1)
  await home()
  const explore = await snapshot()
  for (const view of ['top', 'front', 'isometric']) {
    await page.getByRole('button', { name: view, exact: true }).click()
    await page.waitForFunction(() => window.__worldDiagnostics.cameraPoseRoute?.status === 'complete', null, { timeout: 10000 })
    check(`${view} preset reaches a finite guarded pose`, (await snapshot()).position.every(Number.isFinite))
  }
  await page.getByRole('button', { name: 'top', exact: true }).click()
  await page.waitForFunction(() => window.__worldDiagnostics.cameraPoseRoute?.status === 'complete')
  await page.waitForTimeout(200)
  const top = await snapshot()
  await mode('first-person')
  await mode('explore')
  check('walking round trip preserves a top-down inspection pose', distance(top.position, (await snapshot()).position) < .01)
  await home()
  const saved = await snapshot()
  await mode('first-person')
  check('entering first person focuses the world', await page.locator('canvas').evaluate(canvas => document.activeElement === canvas))
  await page.keyboard.press('ArrowUp')
  check('movement keys after mode selection do not change modes', (await snapshot()).navigation.mode === 'first-person')
  const first = await snapshot()
  check('first person enters at eye height', Math.abs(first.position[1] - first.navigation.playerPosition[1] - 1.6) < .02)
  await page.locator('canvas').focus()
  await page.keyboard.down('Space')
  await page.waitForFunction(y => window.__worldDiagnostics.cameraNavigation.playerPosition[1] > y + .25, first.navigation.playerPosition[1])
  check('first person Space lifts the visitor and eye', (await snapshot()).position[1] > first.position[1] + .25)
  await page.waitForTimeout(1200)
  check('holding Space lands without automatic repeated jumps', Math.abs((await snapshot()).navigation.playerPosition[1] - first.navigation.playerPosition[1]) < .02)
  await page.keyboard.up('Space')
  let moved = false
  for (const direction of ['w', 'd', 's', 'a']) {
    const before = await snapshot()
    await key(direction)
    if (distance(before.navigation.playerPosition, (await snapshot()).navigation.playerPosition) > .2) { moved = true; break }
  }
  check('operator can walk independently', moved)
  const stopped = await snapshot()
  await page.waitForTimeout(700)
  check('walking stops on key release', distance(stopped.navigation.playerPosition, (await snapshot()).navigation.playerPosition) < .001)
  await page.locator('canvas').focus()
  await page.keyboard.down('w')
  await page.waitForTimeout(100)
  await page.evaluate(() => window.dispatchEvent(new Event('blur')))
  await page.waitForTimeout(150)
  const blurred = await snapshot()
  await page.waitForTimeout(400)
  check('focus loss clears held movement', distance(blurred.navigation.playerPosition, (await snapshot()).navigation.playerPosition) < .001)
  await page.keyboard.up('w')
  await page.getByRole('button', { name: 'Capture mouse to look' }).click()
  await page.waitForTimeout(150)
  const captured = await page.evaluate(() => Boolean(document.pointerLockElement))
  check('explicit mouse capture succeeds', captured)
  if (captured) {
    await page.mouse.move(1100, 420)
    await page.keyboard.press('Escape')
    await page.waitForTimeout(150)
    check('releasing pointer lock preserves walking mode', (await snapshot()).navigation.mode === 'first-person')
  }
  await mode('third-person')
  check('entering third person focuses the world', await page.locator('canvas').evaluate(canvas => document.activeElement === canvas))
  const third = await snapshot()
  check('third person preserves the same player body', distance(third.navigation.playerPosition, blurred.navigation.playerPosition) < .001)
  await page.getByRole('button', { name: 'Jump · Space', exact: true }).click()
  await page.waitForFunction(y => window.__worldDiagnostics.cameraNavigation.playerPosition[1] > y + .25, third.navigation.playerPosition[1])
  check('third person HUD jump lifts the visitor', (await snapshot()).navigation.playerPosition[1] > third.navigation.playerPosition[1] + .25)
  await page.waitForTimeout(1200)
  check('third person lands at the original floor', Math.abs((await snapshot()).navigation.playerPosition[1] - third.navigation.playerPosition[1]) < .02)
  check('third-person camera follows behind the player', distance(third.position, third.navigation.playerPosition) > 1.7)
  await page.screenshot({ path: resolve(root, 'third-person.png') })
  await page.evaluate(() => window.__worldDiagnostics.beginInteractionSample())
  await key('a', 1000)
  const performance = await page.evaluate(() => window.__worldDiagnostics.endInteractionSample())
  samples.push(performance)
  check('walking performance has measured frames', performance?.frames.count > 10, performance)
  check('walking frame p95 stays below 50 ms', performance?.frames.p95 < 50, performance?.frames)
  await mode('explore')
  const restored = await snapshot()
  check('return to Explore restores position, target and lens', distance(saved.position, restored.position) < .01 && distance(saved.target, restored.target) < .01 && restored.navigation.lensZoom === 1)
  await page.screenshot({ path: resolve(root, 'explore.png') })
  // Locate a rendered actor through the fixture-only React inspection hook.
  for (const cameraMode of ['first-person', 'third-person']) {
    await ready()
    await mode(cameraMode)
    const pick = await page.evaluate(async () => {
      const pending = [...window.__cameraFixtureRoots].map(root => root.current)
      let world, render
      while (pending.length) {
        const fiber = pending.pop()
        for (const value of [fiber.memoizedProps?.store, fiber.memoizedProps?.value]) {
          if (typeof value?.getState !== 'function') continue
          const state = value.getState()
          if (state.actors && state.nav) world = value
          if (state.scene?.isScene) render = value
        }
        if (fiber.child) pending.push(fiber.child)
        if (fiber.sibling) pending.push(fiber.sibling)
      }
      if (!world || !render) throw new Error('World fixture stores unavailable')
      window.__visitorWorld = world
      window.__visitorRender = render
      const { scene, camera, size, gl } = render.getState()
      // Camera/picking fixture: place one synthetic actor in open terrain.
      // Enclosure-to-visitor arrivals are exercised by scene-design.mjs using
      // real space menus; a roof-hidden projection is not a visible target.
      const fixtureState = world.getState(), visitor = window.__worldDiagnostics.cameraNavigation.playerPosition
      const forward = camera.getWorldDirection(camera.position.clone()); forward.y = 0; forward.normalize()
      let point
      for (const distance of [3, 4, 5]) {
        for (const lateral of [0, -1, 1, -2, 2]) {
          const x = visitor[0] + forward.x * distance - forward.z * lateral
          const z = visitor[2] + forward.z * distance + forward.x * lateral
          const nav = fixtureState.nav, col = Math.floor((x - nav.originX) / nav.cellSize), row = Math.floor((z - nav.originZ) / nav.cellSize)
          const inside = Object.values(fixtureState.places).some(place => {
            if (!place.space) return false
            const dx = x - place.position[0], dz = z - place.position[1], c = Math.cos(place.rotation), s = Math.sin(place.rotation)
            return Math.abs(dx * c - dz * s) < place.size[0] / 2 + .5 && Math.abs(dx * s + dz * c) < place.size[1] / 2 + .5
          })
          if (!inside && col >= 0 && row >= 0 && col < nav.cols && row < nav.rows && nav.walkable[row * nav.cols + col] === 1) { point = [x, z]; break }
        }
        if (point) break
      }
      if (!point) throw new Error('No open ground for visible-agent camera fixture')
      const actor = fixtureState.actors[fixtureState.actorOrder[0]]
      actor.position = point; actor.path = []; actor.destination = undefined; actor.seatId = undefined; actor.state = 'idle'; actor.anim.seated = false
      actor.idle = { activity: 'rest', until: fixtureState.time + 3600 }
      fixtureState.occupancy = Object.fromEntries(Object.entries(fixtureState.occupancy).filter(([, id]) => id !== actor.id))
      world.advance(world.tuning().sim.tickSeconds); render.getState().invalidate()
      await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
      const rect = gl.domElement.getBoundingClientRect()
      let slime
      scene.traverse(object => { if (object.isInstancedMesh && object.geometry.getAttribute('aSquash')) slime = object })
      if (!slime) throw new Error('Rendered slimes unavailable')
      const state = world.getState(), candidates = []
      for (let i = 0; i < state.actorOrder.length; i++) {
        const matrix = slime.matrixWorld.clone()
        slime.getMatrixAt(i, matrix)
        const point = camera.position.clone().setFromMatrixPosition(matrix).applyMatrix4(slime.matrixWorld)
        const distance = point.distanceTo(camera.position)
        point.project(camera)
        const x = (point.x + 1) * size.width / 2, y = (1 - point.y) * size.height / 2
        if (point.z > -1 && point.z < 1 && x > 300 && x < size.width - 100 && y > 100 && y < size.height - 100) candidates.push({ id: state.actorOrder[i], x: x + rect.left, y: y + rect.top, distance })
      }
      return candidates.sort((a, b) => a.distance - b.distance)[0]
    })
    check(cameraMode + ' has a visible agent to select', Boolean(pick))
    if (pick) {
      const before = await snapshot()
      await page.mouse.click(pick.x, pick.y)
      await page.waitForTimeout(300)
      check(cameraMode + ' click invites the rendered agent', await page.evaluate(id => window.__visitorWorld.getState().visitorConversation?.agentId === id, pick.id))
      const approachStart = await page.evaluate(id => window.__visitorWorld.getState().actors[id].position, pick.id)
      await page.waitForTimeout(2000)
      const approachEnd = await page.evaluate(id => {
        const s = window.__visitorWorld.getState()
        return { position: s.actors[id].position, invitation: s.visitorConversation, tick: s.tick }
      }, pick.id)
      check(cameraMode + ' selected agent actually starts approaching', distance(approachStart, approachEnd.position) > .1, approachEnd)
      await page.waitForFunction(() => {
        const visitor = window.__visitorWorld.getState().visitorConversation
        return visitor?.goal && visitor.path?.length === 0
      }, null, { timeout: 45000 })
      await page.waitForTimeout(500)
      const arrived = await page.evaluate(id => {
        const state = window.__visitorWorld.getState(), actor = state.actors[id], visitor = state.visitorConversation
        const dx = visitor.position[0] - actor.position[0], dz = visitor.position[1] - actor.position[1]
        const error = Math.atan2(Math.sin(actor.facing - Math.atan2(dx, dz)), Math.cos(actor.facing - Math.atan2(dx, dz)))
        return { distance: Math.hypot(dx, dz), facingError: error }
      }, pick.id)
      check(cameraMode + ' invited agent arrives and faces visitor', arrived.distance < 3 && Math.abs(arrived.facingError) < .2, arrived)
      check(cameraMode + ' invitation preserves visitor camera', distance(before.position, (await snapshot()).position) < .01)
      await page.getByRole('button', { name: 'End conversation', exact: true }).click()
      check(cameraMode + ' dismisses conversation', await page.evaluate(() => !window.__visitorWorld.getState().visitorConversation))
      await page.getByRole('button', { name: 'Capture mouse to look', exact: true }).click()
      await page.waitForFunction(() => Boolean(document.pointerLockElement))
      await page.evaluate(id => {
        const { scene, camera, gl } = window.__visitorRender.getState()
        let slime
        scene.traverse(object => { if (object.isInstancedMesh && object.geometry.getAttribute('aSquash')) slime = object })
        const matrix = slime.matrixWorld.clone()
        slime.getMatrixAt(window.__visitorWorld.getState().actorOrder.indexOf(id), matrix)
        const head = camera.position.clone().fromArray(window.__worldDiagnostics.cameraNavigation.playerPosition)
        head.y += 1.6
        const direction = camera.position.clone().setFromMatrixPosition(matrix).applyMatrix4(slime.matrixWorld).sub(head)
        const current = camera.getWorldDirection(camera.position.clone())
        const delta = Math.atan2(direction.x, -direction.z) - Math.atan2(current.x, -current.z)
        const yaw = Math.atan2(Math.sin(delta), Math.cos(delta))
        const pitch = Math.atan2(direction.y, Math.hypot(direction.x, direction.z)) - Math.atan2(current.y, Math.hypot(current.x, current.z))
        gl.domElement.dispatchEvent(new PointerEvent('pointermove', { movementX: Math.round(yaw / .0025), movementY: Math.round(-pitch / .0025), bubbles: true }))
      }, pick.id)
      await page.waitForTimeout(150)
      await page.mouse.down()
      await page.mouse.up()
      await page.waitForTimeout(300)
      check(cameraMode + ' captured click selects the agent at the centre aim', await page.evaluate(id => window.__visitorWorld.getState().visitorConversation?.agentId === id, pick.id))
      await page.evaluate(() => document.exitPointerLock())
      await page.waitForFunction(() => !document.pointerLockElement)
      await page.getByRole('button', { name: 'End conversation', exact: true }).click()
    }
  }
  await ready()
  await mode('first-person')
  const logStart = await snapshot()
  const logDirection = logStart.target.map((v, i) => v - logStart.position[i])
  const logFixture = await insertLogFixture(page, logStart.navigation.playerPosition.map((v, i) => v + (i === 1 ? 0 : logDirection[i] * 1)), Math.atan2(logDirection[0], logDirection[2]) + Math.PI / 2)
  await key('w', 850)
  const logEnd = await snapshot()
  const logFrames = await removeFixture(page)
  check('visitor steps onto actual park log geometry', logFrames.some(frame => frame.position[1] > logStart.position[1] + .3), logFixture)
  check('visitor clears actual park log without jumping', distance(logStart.navigation.playerPosition, logEnd.navigation.playerPosition) > 1.8)
  await ready('office')
  await mode('first-person')
  const office = await snapshot()
  check('office supports grounded walking', Math.abs(office.position[1] - office.navigation.playerPosition[1] - 1.6) < .02)
  // Surround the visitor with a box just ahead of its actual look direction.
  const direction = office.target.map((v, i) => v - office.position[i])
  const center = office.navigation.playerPosition.map((v, i) => v + (i === 1 ? 1 : direction[i] * .8))
  await insertFixture(page, center, [.3, 2, .3])
  await key('w', 700)
  const hit = await snapshot()
  check('browser body sweep stops at a rendered obstruction', distance(hit.navigation.playerPosition, center.map((v, i) => i === 1 ? hit.navigation.playerPosition[1] : v)) > .3, hit.navigation)
  await removeFixture(page)
  const stepStart = await snapshot()
  await insertFixture(page, stepStart.navigation.playerPosition.map((v, i) => v + (i === 1 ? .2 : direction[i] * .8)), [.6, .4, .6])
  await key('w', 700)
  const stepEnd = await snapshot()
  const stepFrames = await removeFixture(page)
  check('visitor steps onto a rendered low obstacle', stepFrames.some(frame => frame.position[1] > stepStart.position[1] + .35), { start: stepStart.navigation, end: stepEnd.navigation, maxEyeHeight: Math.max(...stepFrames.map(frame => frame.position[1])) })
  check('visitor clears a low obstacle without jumping', distance(stepStart.navigation.playerPosition, stepEnd.navigation.playerPosition) > 1)
  await page.locator('canvas').focus()
  await page.keyboard.press('Escape')
  await page.waitForTimeout(200)
  check('Escape exits uncaptured walking', (await snapshot()).navigation.mode === 'explore')
  await page.reload()
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
  await page.getByText('Input settings', { exact: true }).click()
  check('device preference survives reload', await page.getByLabel('Navigation device').inputValue() === 'trackpad')
  check('no browser runtime errors', errors.length === 0, errors)
  check('home is deterministic across exploration', distance(explore.position, saved.position) < .01)
} catch (error) {
  check('navigation journey completes', false, String(error))
  await page.screenshot({ path: resolve(root, 'failure.png') }).catch(() => {})
} finally {
  writeFileSync(resolve(root, 'result.json'), JSON.stringify({ at: new Date().toISOString(), base, checks, errors, samples }, null, 2))
  await browser.close()
}
console.log(`Evidence: ${root}`)
process.exitCode = checks.some(check => !check.pass) ? 1 : 0
