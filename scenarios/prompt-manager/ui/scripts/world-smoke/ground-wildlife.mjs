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
    const { scene, camera, raycaster } = render.getState()
    const mesh = scene.getObjectByName(`ambient-${family}`), intersections = []
    mesh.raycast({}, intersections)
    const canopies = [], bark = [], climbing = [], materials = []
    const matrix = camera.matrix.clone()
    scene.getObjectByName('vegetation').traverse(object => {
      if (object.isInstancedMesh) materials.push({ name: object.material.name, count: object.count })
      if (object.isInstancedMesh && object.material.name === 'woodBark') bark.push(object)
      if (!object.isInstancedMesh || object.material.name !== 'leafsGreen') return
      object.geometry.computeBoundingBox()
      for (let slot = 0; slot < object.count; slot++) {
        object.getMatrixAt(slot, matrix); matrix.premultiply(object.matrixWorld)
        const box = object.geometry.boundingBox.clone().applyMatrix4(matrix)
        canopies.push({ x: (box.min.x + box.max.x) / 2, z: (box.min.z + box.max.z) / 2, bottom: box.min.y })
      }
    })
    if (family === 'squirrels') for (let slot = 0; slot < mesh.count; slot += 19) {
      mesh.getMatrixAt(slot, matrix)
      if (Math.abs(matrix.elements[5]) > .001) continue // Ground poses are independent of the canopy.
      const projected = camera.position.clone().setFromMatrixPosition(matrix).project(camera)
      // Offscreen trees are deliberately absent from the compacted vegetation
      // buffers. Match only visible animals to the rendered geometry oracle.
      if (Math.abs(projected.x) > 1 || Math.abs(projected.y) > 1 || Math.abs(projected.z) > 1) continue
      const x = matrix.elements[12], z = matrix.elements[14]
      const canopy = canopies.reduce((best, c) => !best || Math.hypot(c.x - x, c.z - z) < Math.hypot(best.x - x, best.z - z) ? c : best, null)
      let top = -Infinity
      mesh.geometry.computeBoundingBox()
      for (let head = 2; head <= 8; head++) {
        mesh.getMatrixAt(slot + head, matrix); matrix.premultiply(mesh.matrixWorld)
        top = Math.max(top, mesh.geometry.boundingBox.clone().applyMatrix4(matrix).max.y)
      }
      const savedRay = raycaster.ray.clone(), savedNear = raycaster.near, savedFar = raycaster.far
      let grip = Infinity
      if (canopy) for (const foot of [11, 12]) {
        mesh.getMatrixAt(slot + foot, matrix); matrix.premultiply(mesh.matrixWorld)
        const origin = camera.position.clone().setFromMatrixPosition(matrix)
        const direction = origin.clone().set(canopy.x - origin.x, 0, canopy.z - origin.z).normalize()
        // Probe from outside: a gripping paw centre can sit just inside the bark,
        // where front-face raycasting from the centre would miss the surface.
        origin.addScaledVector(direction, -.5)
        raycaster.set(origin, direction); raycaster.near = 0; raycaster.far = 1
        grip = Math.min(grip, Math.abs((raycaster.intersectObjects(bark, false)[0]?.distance ?? Infinity) - .5))
      }
      raycaster.ray.copy(savedRay); raycaster.near = savedNear; raycaster.far = savedFar
      climbing.push({ top, canopy: canopy?.bottom, clearance: canopy ? canopy.bottom - top : null, grip: Number.isFinite(grip) ? grip : null })
    }
    return { count: mesh.count, capacity: mesh.instanceMatrix.count, positions: [...mesh.instanceMatrix.array.slice(0, mesh.count * 16)],
      climbing, materials,
      hits: intersections.length, time: clock.snapshot(), cost: window.__worldDiagnostics.ambientCosts().families.find(row => row.family === family),
      violations: window.__worldSim.violations(), nav: world.getState().nav.walkable.reduce((sum, value) => sum + value, 0) }
  }, family)
}
async function settings() { await page.getByTitle('World Settings', { exact: true }).click() }
try {
  for (const scene of ['park', 'office'].filter(scene => !process.argv.includes('--scene') || scene === process.argv[process.argv.indexOf('--scene') + 1])) {
    await page.goto(`http://localhost:21235/world?scene=${scene}&actors=25&seed=1&diag=1&period=day&weather=clear&profile=high&workbench=1`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
    for (const [label, family] of [['foraging squirrel', 'squirrels'], ['climbing squirrel', 'squirrels'], ['idle rabbit', 'rabbits'], ['hopping rabbit', 'rabbits']]) {
      if (process.argv.includes('--climb-only') && label !== 'climbing squirrel') continue
      await settings()
      if (!await page.getByRole('button', { name: `Preview ${label}`, exact: true }).isVisible()) await page.getByText('World workbench', { exact: true }).click()
      await page.getByRole('button', { name: `Preview ${label}`, exact: true }).click()
      await page.getByRole('status').filter({ hasText: 'Preview ready:' }).waitFor({ timeout: 30000 })
      await page.waitForTimeout(2200)
      await page.keyboard.press('Escape')
      const before = await inspect(family)
      check(`${scene} ${label} renders within its pool`, before.count > 0 && before.count <= before.capacity && before.cost.active <= before.cost.limit, before.cost)
      check(`${scene} ${label} does not intercept picking`, before.hits === 0)
      if (label === 'climbing squirrel') check(`${scene} climbing heads clear the actual rendered foliage`, before.climbing.length > 0 && before.climbing.every(c => c.clearance > .05), before.climbing)
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
      if (label === 'climbing squirrel') {
        await settings()
        const input = page.getByLabel('UTC instant', { exact: true })
        if (!await input.isVisible()) await page.getByText('Exact date and time', { exact: true }).click()
        await input.fill(new Date(before.time.utcMilliseconds + 3000).toISOString().slice(0, 19))
        await page.getByRole('button', { name: 'Apply UTC instant', exact: true }).click()
        await page.getByRole('radiogroup', { name: 'Time of day', exact: true }).getByRole('radio', { name: 'Day', exact: true }).click()
        await page.keyboard.press('Escape')
        await page.getByRole('button', { name: 'front', exact: true }).click()
        await page.waitForTimeout(500)
        const peak = await inspect(family)
        check(`${scene} highest perch keeps heads below rendered foliage`, peak.climbing.length > 0 && peak.climbing.every(c => c.clearance > .05), peak.climbing)
        check(`${scene} perched paws grip the rendered trunk`, peak.climbing.every(c => c.grip !== null && c.grip < .15), { climbing: peak.climbing, materials: peak.materials })
        await page.screenshot({ path: resolve(root, `${scene}-squirrel-highest-perch.png`) })
      }
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
