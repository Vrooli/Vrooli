#!/usr/bin/env node
/** Real doorway traversals at off-centre approaches, in both walking cameras. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const option = (key, fallback) => { const i = process.argv.indexOf(key); return i < 0 ? fallback : process.argv[i + 1] }
const root = resolve(option('--evidence-dir', `evidence/doorways-${Date.now()}`))
const sceneName = option('--scene', 'office')
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = []
page.on('pageerror', e => errors.push(e.message))
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`) }
const position = () => page.evaluate(() => window.__worldDiagnostics.cameraNavigation.playerPosition)
const mode = async value => { await page.getByLabel('Camera mode', { exact: true }).selectOption(value); await page.waitForTimeout(250) }
try {
  await page.goto(`http://localhost:21235/world?actors=${option('--actors', '16')}&scene=${sceneName}&intro=0&profile=high&diag=1&period=day&weather=clear`)
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
  const rooms = await page.evaluate(() => {
    const pending = [...window.__cameraFixtureRoots].map(root => root.current); let world, render
    while (pending.length) { const f = pending.pop(); for (const v of [f.memoizedProps?.store, f.memoizedProps?.value]) {
      if (typeof v?.getState !== 'function') continue
      if (v.getState().actors && v.getState().nav) world = v
      if (v.getState().scene?.isScene) render = v
    } if (f.child) pending.push(f.child); if (f.sibling) pending.push(f.sibling) }
    window.__doorways = { world, render }
    return Object.values(world.getState().places).filter(p => p.space).flatMap(p => (p.space.shelters.length ? p.space.shelters : [null]).map(shell => {
      const c = Math.cos(p.rotation), s = Math.sin(p.rotation)
      const [x, z] = shell ? [shell.position[0], shell.position[1] + shell.size[1] / 2] : p.space.entrance
      return { id: shell?.id ?? p.id, variant: p.space.variant, approach: shell ? .7 : 1.15, exitDepth: shell ? .6 : .9,
        door: [p.position[0] + x * c + z * s, p.position[1] - x * s + z * c], outward: [s, c] }
    }))
  })
  const cutaway = await page.evaluate(() => {
    const { scene, camera } = window.__doorways.render.getState(), surfaces = []
    scene.getObjectByName('places').traverse(o => {
      if (!o.isInstancedMesh || o.geometry.userData.cameraObstacle !== 'box') return
      for (let i = 0; i < o.count; i++) {
        const m = camera.matrixWorld.clone().identity(); o.getMatrixAt(i, m)
        const e = m.elements, height = Math.hypot(e[4], e[5], e[6])
        if (height > .5) surfaces.push({ position: [e[12], e[13], e[14]], height })
      }
    }); return surfaces
  })
  if (sceneName === 'office') check('Explore retains full architectural height on far walls', cutaway.some(s => s.height > 1.2 && s.position[1] > 2), cutaway)
  await page.screenshot({ path: resolve(root, `${sceneName}-directional-explore.png`) })
  for (const room of rooms.filter(room => (!option('--room', '') || room.id === option('--room', '')) && (!option('--variant', '') || room.variant === option('--variant', '')))) for (const cameraMode of ['first-person', 'third-person']) for (const offset of option('--offset', '') ? [Number(option('--offset', ''))] : [-.4, 0, .4]) {
    await mode('explore')
    await page.evaluate(({ room, offset }) => {
      const { controls, invalidate } = window.__doorways.render.getState(), [dx, dz] = room.outward
      const x = room.door[0] + dx * room.approach + dz * offset, z = room.door[1] + dz * room.approach - dx * offset
      controls.setLookAt(x + dx * 8, 8, z + dz * 8, x, 0, z, false); controls.update(0); invalidate()
    }, { room, offset })
    await mode(cameraMode)
    const before = await position()
    await page.keyboard.down('w')
    try { await page.waitForFunction(({ door, outward }) => {
      const p = window.__worldDiagnostics.cameraNavigation.playerPosition
      return (p[0] - door[0]) * outward[0] + (p[2] - door[1]) * outward[1] < -1.2
    }, room, { timeout: 3000 }) } catch { /* Retain blocked position and collider evidence below. */ }
    finally { await page.keyboard.up('w') }
    await page.waitForTimeout(150)
    const after = await position()
    const nearby = await page.evaluate(after => {
      const { scene, camera } = window.__doorways.render.getState(); const boxes = []
      scene.updateMatrixWorld(true)
      scene.traverse(o => {
        if (!o.isMesh || o.userData.walkObstacle === false) return
        let eligible = ['box', 'triangles'].includes(o.geometry.userData.cameraObstacle)
        for (let p = o; p; p = p.parent) eligible ||= p.userData.walkObstacle === true
        if (!eligible) return
        o.geometry.computeBoundingBox()
        for (let i = 0; i < (o.isInstancedMesh ? o.count : 1); i++) {
          const m = camera.matrixWorld.clone().identity(); if (o.isInstancedMesh) o.getMatrixAt(i, m); m.premultiply(o.matrixWorld)
          const b = o.geometry.boundingBox.clone().applyMatrix4(m)
          if (b.min.x < after[0] + .4 && b.max.x > after[0] - .4 && b.min.z < after[2] + .6 && b.max.z > after[2] - .6 && b.min.y < after[1] + 1.8 && b.max.y > after[1] + .02)
            boxes.push({ name: o.name, parent: o.parent?.name, min: b.min.toArray(), max: b.max.toArray() })
        }
      }); return { boxes, navigation: window.__worldDiagnostics.cameraNavigation }
    }, after)
    const depth = p => (p[0] - room.door[0]) * room.outward[0] + (p[2] - room.door[1]) * room.outward[1]
    check(`${room.id} ${cameraMode} enters at offset ${offset}`, depth(before) > room.approach - .3 && depth(after) < -1.2, { before, after, room, nearby })
    if (offset === 0) {
      const ceiling = await page.evaluate(() => {
        const { scene, raycaster, camera } = window.__doorways.render.getState()
        const origin = camera.position.clone().set(...window.__worldDiagnostics.cameraNavigation.playerPosition); origin.y += 1.6
        raycaster.set(origin, origin.clone().set(0, 1, 0))
        return raycaster.intersectObject(scene.getObjectByName('places'), true).some(hit => hit.distance > .4 && hit.distance < 1.4)
      })
      if (sceneName === 'office' || room.variant === 'rv') check(`${room.id} ${cameraMode} has a solid overhead ceiling`, ceiling)
      await page.screenshot({ path: resolve(root, `${room.id.replaceAll(':', '-')}-${cameraMode}.png`) })
    }
    await page.keyboard.down('s')
    try { await page.waitForFunction(({ door, outward, exitDepth }) => {
      const p = window.__worldDiagnostics.cameraNavigation.playerPosition
      return (p[0] - door[0]) * outward[0] + (p[2] - door[1]) * outward[1] > exitDepth
    }, room, { timeout: 3000 }) } catch { /* Preserve failure evidence. */ }
    finally { await page.keyboard.up('s') }
    await page.waitForTimeout(150)
    const outside = await position()
    check(`${room.id} ${cameraMode} exits at offset ${offset}`, depth(outside) > room.exitDepth - .1, { after, outside, room })
  }
  check('no browser errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors }, null, 2)); await browser.close() }
if (checks.some(c => !c.pass)) process.exitCode = 1
