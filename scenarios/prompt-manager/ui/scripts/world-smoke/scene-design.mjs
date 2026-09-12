#!/usr/bin/env node
/** Scene acceptance journey. Uses rendered objects and normal UI; synthetic roster avoids management writes. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'
import { installFixtureHook } from './camera-fixtures.mjs'
import { PNG } from 'pngjs'

const option = (name, fallback) => { const i = process.argv.indexOf(name); return i < 0 ? fallback : process.argv[i + 1] }
const period = option('--period', 'day')
const root = resolve(option('--evidence-dir', `evidence/scene-design-${Date.now()}`))
const base = option('--base-url', `http://localhost:${execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()}`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: process.env.WORLD_SMOKE_CHROME ?? '/usr/bin/google-chrome', headless: true,
  args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = [], spaces = []
page.on('pageerror', error => errors.push(error.message))
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`) }
async function captureReference(name) {
  await page.evaluate(() => window.__sceneDesign.render.getState().invalidate())
  await page.waitForTimeout(100)
  const png = PNG.sync.read(await page.screenshot({ path: resolve(root, name) }))
  let visible = 0, total = 0
  for (let y = 150; y < 900; y += 8) for (let x = 750; x < 1550; x += 8) {
    const p = (y * png.width + x) * 4; total++
    if (png.data[p] + png.data[p + 1] + png.data[p + 2] > 6) visible++
  }
  check(`${name} contains a rendered scene`, visible / total > .5, { visible, total })
}

async function inspect() {
  return page.evaluate(() => {
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
    if (!world || !render) throw new Error('Cannot inspect rendered world')
    window.__sceneDesign = { world, render }
    const state = world.getState()
    return { spaces: Object.values(state.places).filter(p => p.space), violations: window.__worldSim.violations(), draws: window.__worldDiagnostics.drawCalls }
  })
}
async function captureReferences(scene, state) {
  const references = scene === 'park'
    ? ['tent', 'cabin', 'rv'].map(variant => state.spaces.find(room => room.space.variant === variant))
    : [state.spaces.find(room => room.teamId)]
  check(`${scene} reference variants are present`, references.every(Boolean))
  for (const room of references.filter(room => room && (!option('--variant', '') || room.space.variant === option('--variant', '')))) {
    const name = scene === 'park' ? room.space.variant : 'studio'
    await page.getByLabel('Camera mode', { exact: true }).selectOption('explore')
    await page.waitForTimeout(250)
    await page.evaluate(room => {
      const { controls, invalidate } = window.__sceneDesign.render.getState()
      const s = Math.sin(room.rotation), c = Math.cos(room.rotation), distance = Math.max(...room.size) * 1.25
      controls.setLookAt(room.position[0] + s * distance + c * distance * .35, distance,
        room.position[1] + c * distance - s * distance * .35, room.position[0], 0, room.position[1], false)
      controls.update(0); invalidate()
    }, room)
    await page.waitForTimeout(700)
    await captureReference(`${scene}-${name}-explore-${period}.png`)
    const entrance = await page.evaluate(room => {
      const { controls, invalidate } = window.__sceneDesign.render.getState()
      const s = Math.sin(room.rotation), c = Math.cos(room.rotation), local = room.space.entrance
      const door = [room.position[0] + local[0] * c + local[1] * s, room.position[1] - local[0] * s + local[1] * c]
      const approach = room.space.kind === 'campsite' ? 3.5 : 1.15
      const side = room.space.kind === 'campsite' ? -1.4 : 0
      const x = door[0] + s * approach + c * side, z = door[1] + c * approach - s * side
      controls.setLookAt(x + s * 8, 8, z + c * 8, x, 0, z, false); controls.update(0); invalidate()
      return { door, outward: [s, c] }
    }, room)
    for (const mode of ['first-person', 'third-person']) {
      await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
      await page.waitForTimeout(350)
      if (scene === 'office' && mode === 'first-person') {
        await page.keyboard.down('w')
        try { await page.waitForFunction(({ door, outward }) => {
          const p = window.__worldDiagnostics.cameraNavigation.playerPosition
          return (p[0] - door[0]) * outward[0] + (p[2] - door[1]) * outward[1] < -1.2
        }, entrance, { timeout: 5000 }) } finally { await page.keyboard.up('w') }
      }
      await page.waitForTimeout(500)
      check(`${scene} ${name} ${mode} reference is ready`, await page.evaluate(mode => window.__worldDiagnostics.ready && window.__worldDiagnostics.cameraNavigation.mode === mode, mode))
      await captureReference(`${scene}-${name}-${mode}-${period}.png`)
    }
  }
}
try {
  for (const scene of ['park', 'office'].filter(scene => !option('--scene', '') || scene === option('--scene', ''))) {
    await page.goto(`${base}/world?actors=16&profile=high&scene=${scene}&intro=0&diag=1&period=${period}&weather=clear`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
    await page.waitForTimeout(1000)
    const state = await inspect(); spaces.push({ scene, ...state })
    check(`${scene} generated spaces have named entrances and occupants`, state.spaces.length > 0 && state.spaces.every(p => (p.teamId ? p.space.occupantIds.length > 0 : p.space.occupantIds.length === 0) && p.space.entrance.length === 2), state)
    check(`${scene} simulation invariants`, state.violations.length === 0, state.violations)
    if (scene === 'office') {
      const furnishingBounds = await page.evaluate(() => {
        const { world, render } = window.__sceneDesign, state = world.getState()
        const { scene, camera } = render.getState(); scene.updateMatrixWorld(true)
        const byProp = new Map()
        for (const spot of state.decor.filter(spot => spot.roomId)) {
          const list = byProp.get(spot.propId) ?? []; list.push(spot); byProp.set(spot.propId, list)
        }
        return [...byProp].flatMap(([id, spots]) => spots.map((spot, index) => {
          let bounds
          scene.getObjectByName(`furnishing:${id}`)?.traverse(mesh => {
            if (!mesh.isInstancedMesh || mesh.count <= index) return
            mesh.geometry.computeBoundingBox()
            const matrix = camera.matrixWorld.clone().identity(); mesh.getMatrixAt(index, matrix); matrix.premultiply(mesh.matrixWorld)
            const box = mesh.geometry.boundingBox.clone().applyMatrix4(matrix)
            if (bounds) bounds.union(box); else bounds = box
          })
          const room = scene.getObjectByName(`space:${spot.roomId}`)
          const ground = room?.getWorldPosition(camera.position.clone()).y
          const center = bounds?.getCenter(camera.position.clone())
          return { id: spot.id, prop: id, bottom: bounds?.min.y, ground, center: center?.toArray(), target: spot.position,
            pass: !!bounds && Math.abs(bounds.min.y - ground - .06) < .01 && Math.hypot(center.x - spot.position[0], center.z - spot.position[1]) < .01 }
        }))
      })
      check('office authored furnishings are centred on their sockets above the finished floor', furnishingBounds.length > 0 && furnishingBounds.every(item => item.pass), furnishingBounds)
    }
    if (process.argv.includes('--references')) { await captureReferences(scene, state); continue }
    await page.screenshot({ path: resolve(root, `${scene}-explore-${period}.png`) })
    // Project physical enclosure surfaces into the viewport, then use a real click.
    const candidates = await page.evaluate(() => {
      const { world, render } = window.__sceneDesign
      const { camera, gl, scene } = render.getState(), rect = gl.domElement.getBoundingClientRect()
      return Object.values(world.getState().places).filter(p => p.space).map(room => {
        const group = scene.getObjectByName(`space:${room.id}`)
        const point = camera.position.clone().set(room.space.entrance[0] - (room.space.kind === 'campsite' ? 1.9 : 2), 1.65, room.space.entrance[1] + .2)
        group.localToWorld(point); point.project(camera)
        return { id: room.id, label: room.label, x: rect.left + (point.x + 1) * rect.width / 2, y: rect.top + (1 - point.y) * rect.height / 2, depth: point.z }
      }).filter(p => p.x > 550 && p.x < 1450 && p.y > 120 && p.y < 850 && p.depth < 1)
    })
    let selected
    for (const candidate of candidates) {
      await page.mouse.click(candidate.x, candidate.y)
      if (await page.getByRole('dialog', { name: `${candidate.label} space` }).isVisible()) { selected = candidate; break }
    }
    check(`${scene} physical entrance sign opens its space menu`, !!selected, candidates)
    if (selected) {
      check(`${scene} menu exposes occupants and management`, await page.getByRole('button', { name: 'Manage space', exact: true }).isVisible() && await page.getByRole('list', { name: 'Space occupants' }).isVisible())
      check(`${scene} menu owns keyboard focus`, await page.getByRole('button', { name: 'Close space menu' }).evaluate(button => button === document.activeElement))
      await page.screenshot({ path: resolve(root, `${scene}-space-menu.png`) })
      await page.keyboard.press('Escape')
      check(`${scene} Escape dismisses the menu and restores world focus`, await page.getByRole('dialog', { name: `${selected.label} space` }).count() === 0 && await page.locator('canvas').evaluate(canvas => canvas === document.activeElement))
    }
    // Establish a reproducible approach using the existing Explore controls.
    // The actual traversal below uses keyboard input and the production body.
    const entrance = await page.evaluate(() => {
      const { world, render } = window.__sceneDesign
      const room = Object.values(world.getState().places).find(p => p.space && p.teamId)
      const shell = room.space.shelters[0]
      const local = shell ? [shell.position[0], shell.position[1] + shell.size[1] / 2] : room.space.entrance
      const c = Math.cos(room.rotation), s = Math.sin(room.rotation)
      const door = [room.position[0] + local[0] * c + local[1] * s, room.position[1] - local[0] * s + local[1] * c]
      const outward = [s, c], target = [door[0] + outward[0] * 1.1, door[1] + outward[1] * 1.1]
      const { controls } = render.getState()
      controls.setLookAt(target[0] + outward[0] * 8, 8, target[1] + outward[1] * 8, target[0], 0, target[1], false)
      controls.update(0)
      render.getState().invalidate()
      return { roomId: room.id, door, outward }
    })
    for (const mode of ['first-person', 'third-person']) {
      await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
      await page.waitForFunction(mode => window.__worldDiagnostics.cameraNavigation?.mode === mode, mode)
      await page.waitForTimeout(400)
      check(`${scene} ${mode} owns keyboard focus`, await page.locator('canvas').evaluate(canvas => canvas === document.activeElement))
      const capacity = await page.evaluate(() => {
        const meshes = []
        window.__sceneDesign.render.getState().scene.getObjectByName('places').traverse(object => {
          if (object.isInstancedMesh) meshes.push({ name: object.name, count: object.count, capacity: object.instanceMatrix.count })
        })
        return meshes
      })
      check(`${scene} ${mode} architecture fits allocated GPU buffers`, capacity.every(mesh => mesh.count <= mesh.capacity), capacity)
      if (scene === 'park' && period === 'night') {
        const fireLights = await page.evaluate(() => {
          const lights = []
          window.__sceneDesign.render.getState().scene.getObjectByName('lamp-lights')?.traverse(object => {
            if (object.isPointLight && object.color.getHexString() === 'ffab52' && object.intensity > 0) lights.push({ position: object.position.toArray(), intensity: object.intensity })
          })
          return lights
        })
        check(`${scene} ${mode} campfires illuminate their surroundings`, fireLights.length > 0, fireLights)
      }
      if (mode === 'first-person') {
        const before = await page.evaluate(() => window.__worldDiagnostics.cameraNavigation.playerPosition)
        await page.keyboard.down('KeyW')
        await page.waitForTimeout(950)
        await page.keyboard.up('KeyW')
        await page.waitForTimeout(200)
        const after = await page.evaluate(() => window.__worldDiagnostics.cameraNavigation.playerPosition)
        const nearby = await page.evaluate(after => {
          const { scene, camera } = window.__sceneDesign.render.getState()
          const result = []
          scene.updateMatrixWorld(true)
          scene.traverse(object => {
            if (!object.isMesh) return
            let eligible = ['box', 'triangles'].includes(object.geometry.userData.cameraObstacle)
            for (let p = object; p; p = p.parent) eligible ||= p.userData.walkObstacle === true
            if (!eligible) return
            object.geometry.computeBoundingBox()
            for (let i = 0; i < (object.isInstancedMesh ? object.count : 1); i++) {
              const matrix = camera.matrixWorld.clone().identity()
              if (object.isInstancedMesh) object.getMatrixAt(i, matrix)
              matrix.premultiply(object.matrixWorld)
              const box = object.geometry.boundingBox.clone().applyMatrix4(matrix)
              if (box.min.x < after[0] + .4 && box.max.x > after[0] - .4 && box.min.z < after[2] + .4 && box.max.z > after[2] - .4 && box.max.y > after[1] + .001 && box.min.y < after[1] + 1.75) result.push({ mesh: object.name, parent: object.parent?.name, min: box.min.toArray(), max: box.max.toArray() })
            }
          })
          return result
        }, after)
        const inside = (after[0] - entrance.door[0]) * entrance.outward[0] + (after[2] - entrance.door[1]) * entrance.outward[1]
        const beforeDepth = (before[0] - entrance.door[0]) * entrance.outward[0] + (before[2] - entrance.door[1]) * entrance.outward[1]
        const lateral = Math.abs((after[0] - entrance.door[0]) * entrance.outward[1] - (after[2] - entrance.door[1]) * entrance.outward[0])
        check(`${scene} visitor walks through the actual entrance`, beforeDepth > .3 && inside < -.3 && lateral < .6, { before, after, entrance, inside, lateral, beforeDepth, nearby })
      }
      await page.screenshot({ path: resolve(root, `${scene}-${mode}-${period}.png`) })
    }
    // Return outside by walking, then invite different enclosed occupants in
    // both modes. Selection alone is insufficient: observe motion and arrival.
    await page.locator('canvas').focus()
    await page.keyboard.down('KeyS')
    try {
      await page.waitForFunction(({ door, outward }) => {
        const p = window.__worldDiagnostics.cameraNavigation.playerPosition
        return (p[0] - door[0]) * outward[0] + (p[2] - door[1]) * outward[1] > 1.8
      }, entrance, { timeout: 5000 })
    } finally { await page.keyboard.up('KeyS') }
    for (const mode of ['first-person', 'third-person']) {
      await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
      await page.waitForTimeout(400)
      // From the tent threshold, look up slightly to select its gable.
      if (scene === 'park' && mode === 'first-person') {
        await page.mouse.move(1000, 500); await page.mouse.down()
        await page.mouse.move(1000, 380, { steps: 4 }); await page.mouse.up()
        await page.waitForTimeout(200)
      }
      const candidates = await page.evaluate(({ roomId }) => {
        const { world, render } = window.__sceneDesign
        const room = world.getState().places[roomId], shell = room.space.shelters[0]
        const { scene, camera, gl } = render.getState(), group = scene.getObjectByName(`space:${room.id}`), rect = gl.domElement.getBoundingClientRect()
        const x = shell?.position[0] ?? 0, z = shell ? shell.position[1] + shell.size[1] / 2 + .3 : room.space.entrance[1]
        return [[x, 2.55, z], [x + 1.05, 1.4, z], [x - 1.05, 1.4, z]].map(local => {
          const point = camera.position.clone().set(...local); group.localToWorld(point); point.project(camera)
          return { x: rect.left + (point.x + 1) * rect.width / 2, y: rect.top + (1 - point.y) * rect.height / 2, depth: point.z }
        }).filter(p => p.depth > -1 && p.depth < 1 && p.x > 550 && p.x < 1450 && p.y > 100 && p.y < 800)
      }, entrance)
      const room = state.spaces.find(room => room.id === entrance.roomId)
      const dialog = page.getByRole('dialog', { name: `${room.label} space` })
      for (const point of candidates) { await page.mouse.click(point.x, point.y); if (await dialog.isVisible()) break }
      const opened = await dialog.isVisible()
      check(`${scene} ${mode} enclosure opens its occupant menu`, opened, candidates)
      if (!opened) continue
      const occupant = await page.evaluate(({ roomId, door, outward }) => {
        const state = window.__sceneDesign.world.getState()
        return state.places[roomId].space.occupantIds.map(id => state.actors[id]).find(actor => actor && (actor.position[0] - door[0]) * outward[0] + (actor.position[1] - door[1]) * outward[1] < -.5)
      }, entrance)
      check(`${scene} ${mode} has an enclosed occupant to invite`, !!occupant)
      if (!occupant) { await page.keyboard.press('Escape'); continue }
      const before = await page.evaluate(() => window.__worldDiagnostics.cameraNavigation.playerPosition)
      await dialog.getByRole('listitem').filter({ hasText: occupant.name }).getByRole('button', { name: 'Come here' }).click()
      await page.waitForFunction(id => {
        const state = window.__sceneDesign.world.getState(), visitor = state.visitorConversation
        return visitor?.agentId === id && visitor.goal && visitor.path?.length === 0
      }, occupant.id, { timeout: 45000 })
      const arrived = await page.evaluate(({ id, initial }) => {
        const state = window.__sceneDesign.world.getState(), actor = state.actors[id], visitor = state.visitorConversation
        const dx = visitor.position[0] - actor.position[0], dz = visitor.position[1] - actor.position[1]
        return { distance: Math.hypot(dx, dz), movement: Math.hypot(actor.position[0] - initial[0], actor.position[1] - initial[1]), facing: Math.atan2(Math.sin(actor.facing - Math.atan2(dx, dz)), Math.cos(actor.facing - Math.atan2(dx, dz))), player: window.__worldDiagnostics.cameraNavigation.playerPosition }
      }, { id: occupant.id, initial: occupant.position })
      check(`${scene} ${mode} occupant actually approaches and faces the visitor`, arrived.movement > .5 && arrived.distance < 3 && Math.abs(arrived.facing) < .2, arrived)
      check(`${scene} ${mode} invitation preserves player position`, Math.hypot(...arrived.player.map((v, i) => v - before[i])) < .01)
      await page.screenshot({ path: resolve(root, `${scene}-${mode}-conversation-${period}.png`) })
      await page.getByRole('button', { name: 'End conversation', exact: true }).click()
    }
  }
  check('no browser runtime errors', errors.length === 0, errors)
} catch (error) {
  errors.push(error.stack ?? error.message)
  check('scene journey completed', false, error.message)
} finally {
  writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors, spaces }, null, 2))
  await browser.close()
}
process.exitCode = checks.every(c => c.pass) && errors.length === 0 ? 0 : 1
