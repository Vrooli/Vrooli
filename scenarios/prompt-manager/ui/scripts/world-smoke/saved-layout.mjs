#!/usr/bin/env node
/** Real editor and reload/growth journey. World/roster wire fixtures contain all writes. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const root = resolve(process.argv[2] ?? `evidence/saved-layout-${Date.now()}`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = [], writes = [], layouts = { park: { scene: 'park', overrides: [] }, office: { scene: 'office', overrides: [] } }
let members = 4
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
const agents = () => ['a', 'b'].flatMap(team => Array.from({ length: team === 'a' ? members : 4 }, (_, i) => ({ id: `layout-${team}-${i}`, displayName: `Camper ${team.toUpperCase()}${i + 1}`, status: 'active', createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' })))
const teams = () => ['a', 'b'].map(id => ({ id: `layout-team-${id}`, displayName: id === 'a' ? 'Juniper' : 'Pinecone', enabled: true, memberCount: id === 'a' ? members : 4,
  runtime: { mode: 'multi-process' }, coordination: { pattern: 'independent', reportingMode: 'none', messagingMode: 'disabled', capabilities: { showOrgContext: false, injectInbox: false, allowPeerTriggers: false, showTaskBoardGuidance: false, showKnowledgeLogGuidance: false, requireHandoff: false } },
  execution: { queuePolicy: 'serialized', maxConcurrentRuns: 1 }, operatingContract: { schemaVersion: 1, documents: { planOfRecord: [], sharedState: [] }, knowledgeTopics: {}, members: {} }, createdAt: '2026-01-01T00:00:00Z', updatedAt: '2026-01-01T00:00:00Z' }))
await page.route('**/ListAgents', route => route.fulfill({ json: { agents: agents() } }))
await page.route('**/ListTeams', route => route.fulfill({ json: { teams: teams() } }))
await page.route('**/GetTeam', route => {
  const id = route.request().postDataJSON().id, team = teams().find(t => t.id === id)
  return route.fulfill({ json: { ...team, members: agents().filter(a => a.id.startsWith(`layout-${id.at(-1)}-`)).map(a => ({ agentId: a.id, displayName: a.displayName, status: 'active', roles: [] })) } })
})
await page.route('**/GetLayout', route => route.fulfill({ json: layouts[route.request().postDataJSON().scene] }))
await page.route('**/SetLayout', route => { const { layout } = route.request().postDataJSON(); writes.push(layout); layouts[layout.scene] = structuredClone(layout); return route.fulfill({ json: layout }) })
await page.route('**/SetWorldConfig', route => route.fulfill({ json: route.request().postDataJSON().config }))
await page.route('**/CreateRun', route => { errors.push('Unexpected agent run request'); return route.abort() })
page.on('pageerror', error => errors.push(error.message))
async function inspect() {
  return page.evaluate(() => {
    const pending = [...window.__cameraFixtureRoots].map(r => r.current); let world, render
    while (pending.length) {
      const f = pending.pop()
      for (const value of [f.memoizedProps?.store, f.memoizedProps?.value]) if (typeof value?.getState === 'function') {
        const s = value.getState(); if (s.actors && s.nav) world = value; if (s.scene?.isScene) render = value
      }
      if (f.child) pending.push(f.child); if (f.sibling) pending.push(f.sibling)
    }
    if (!world || !render) throw Error('World is not rendered')
    window.__savedLayoutFixture = { world, render }
    const state = world.getState(), { camera, gl } = render.getState(), rect = gl.domElement.getBoundingClientRect()
    const places = state.placeOrder.map(id => state.places[id])
    return { rooms: places.filter(p => p.kind === 'room'), places, agents: state.actorOrder.map(id => ({ id, desk: state.actors[id].deskSeatId })), violations: window.__worldSim.violations(),
      projections: places.filter(p => p.kind === 'room').map(room => { const point = camera.position.clone().set(room.position[0], 0, room.position[1]).project(camera); return { id: room.id, x: rect.left + (point.x + 1) * rect.width / 2, y: rect.top + (1 - point.y) * rect.height / 2, depth: point.z } }) }
  })
}
async function ready(scene, reload = false) {
  if (reload) await page.reload()
  else await page.goto(`http://localhost:21235/world?scene=${scene}&seed=7&profile=medium&intro=0&diag=1&period=day&weather=clear`)
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
  await page.waitForTimeout(400)
  return inspect()
}
function overlaps(a, b) {
  const axes = [a.rotation, b.rotation].flatMap(yaw => [[Math.cos(yaw), -Math.sin(yaw)], [Math.sin(yaw), Math.cos(yaw)]])
  const corners = room => [-1, 1].flatMap(x => [-1, 1].map(z => [room.position[0] + x * room.size[0] / 2 * Math.cos(room.rotation) + z * room.size[1] / 2 * Math.sin(room.rotation), room.position[1] - x * room.size[0] / 2 * Math.sin(room.rotation) + z * room.size[1] / 2 * Math.cos(room.rotation)]))
  return axes.every(axis => { const left = corners(a).map(p => p[0] * axis[0] + p[1] * axis[1]), right = corners(b).map(p => p[0] * axis[0] + p[1] * axis[1]); return Math.min(...left) < Math.max(...right) - .01 && Math.min(...right) < Math.max(...left) - .01 })
}
async function walkSavedDoor(room) {
  const direction = [Math.sin(room.rotation), Math.cos(room.rotation)]
  const door = [room.position[0] + direction[0] * room.size[1] / 2, room.position[1] + direction[1] * room.size[1] / 2]
  const position = () => page.evaluate(() => window.__worldDiagnostics.cameraNavigation.playerPosition)
  const depth = p => (p[0] - door[0]) * direction[0] + (p[2] - door[1]) * direction[1]
  for (const mode of ['first-person', 'third-person']) for (const offset of [-.4, 0, .4]) {
    await page.getByLabel('Camera mode', { exact: true }).selectOption('explore')
    await page.waitForTimeout(250)
    await page.evaluate(({ door, direction: [dx, dz], offset }) => {
      const { controls, invalidate } = window.__savedLayoutFixture.render.getState()
      const x = door[0] + dx * 1.15 + dz * offset, z = door[1] + dz * 1.15 - dx * offset
      controls.setLookAt(x + dx * 8, 8, z + dz * 8, x, 0, z, false); controls.update(0); invalidate()
    }, { door, direction, offset })
    await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
    await page.waitForTimeout(250)
    const before = await position()
    await page.keyboard.down('w')
    try { await page.waitForFunction(({ door, direction }) => {
      const p = window.__worldDiagnostics.cameraNavigation.playerPosition
      return (p[0] - door[0]) * direction[0] + (p[2] - door[1]) * direction[1] < -1.2
    }, { door, direction }, { timeout: 3000 }) } catch { /* Report the actual blocked position. */ }
    finally { await page.keyboard.up('w') }
    const after = await position()
    check(`saved grown office ${mode} enters at offset ${offset}`, depth(before) > .8 && depth(after) < -1.2, { before, after, door })
    if (offset === 0) {
      const ceiling = await page.evaluate(() => {
        const { scene, raycaster, camera } = window.__savedLayoutFixture.render.getState()
        const origin = camera.position.clone().set(...window.__worldDiagnostics.cameraNavigation.playerPosition); origin.y += 1.6
        raycaster.set(origin, origin.clone().set(0, 1, 0))
        return raycaster.intersectObject(scene.getObjectByName('places'), true).some(hit => hit.distance > .4 && hit.distance < 1.4)
      })
      check(`saved grown office ${mode} has an overhead ceiling`, ceiling)
      await page.screenshot({ path: resolve(root, `office-grown-${mode}.png`) })
    }
    await page.keyboard.down('s')
    try { await page.waitForFunction(({ door, direction }) => {
      const p = window.__worldDiagnostics.cameraNavigation.playerPosition
      return (p[0] - door[0]) * direction[0] + (p[2] - door[1]) * direction[1] > .9
    }, { door, direction }, { timeout: 3000 }) } catch { /* Report the actual blocked position. */ }
    finally { await page.keyboard.up('s') }
    const outside = await position()
    check(`saved grown office ${mode} exits at offset ${offset}`, depth(outside) > .7, { after, outside, door })
  }
}
try {
  for (const scene of ['park', 'office']) {
    members = 4
    await ready(scene)
    await page.getByLabel('Camera mode', { exact: true }).selectOption('explore')
    await page.getByRole('button', { name: 'top', exact: true }).click(); await page.waitForTimeout(900)
    const before = await inspect(), roomId = 'room:layout-team-a', room = before.rooms.find(r => r.id === roomId)
    check(scene + ' fixture loads both named teams', before.rooms.filter(r => r.teamId).length === 2 && room?.space?.occupantIds.length === 4)
    await page.getByRole('button', { name: 'Edit layout', exact: true }).click()
    await page.waitForTimeout(900)
    const point = (await inspect()).projections.find(p => p.id === roomId)
    check(scene + ' room handle is in the viewport', point && point.x > 550 && point.x < 1450 && point.y > 130 && point.y < 850, point)
    const count = writes.length
    await page.mouse.move(point.x, point.y); await page.mouse.down(); await page.mouse.move(point.x + 45, point.y + 30, { steps: 12 }); await page.mouse.up()
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
    await page.waitForTimeout(1800)
    const moved = await inspect(), edited = moved.rooms.find(r => r.id === roomId)
    check(scene + ' dragging the room persists its transform', writes.length > count && JSON.stringify(edited.position) !== JSON.stringify(room.position), { writes: writes.slice(count), before: room.position, after: edited.position })
    const dx = edited.position[0] - room.position[0], dz = edited.position[1] - room.position[1]
    const children = before.places.filter(p => p.parentId === roomId)
    check(scene + ' furniture and entrance move with the space', children.every(child => { const next = moved.places.find(p => p.id === child.id); return next && Math.abs(next.position[0] - child.position[0] - dx) < .001 && Math.abs(next.position[1] - child.position[1] - dz) < .001 }), children.map(p => p.id))
    await page.getByRole('button', { name: 'Done', exact: true }).click()
    await page.screenshot({ path: resolve(root, `${scene}-edited.png`) })
    const restored = await ready(scene, true)
    check(scene + ' reload restores the complete saved layout', JSON.stringify(restored.places) === JSON.stringify(moved.places))
    members = 12
    const grown = await ready(scene, true), grownRoom = grown.rooms.find(r => r.id === roomId)
    const adjustment = Math.hypot(grownRoom.position[0] - edited.position[0], grownRoom.position[1] - edited.position[1])
    check(scene + ' growth retains the team and stations with a bounded site adjustment', adjustment < 10 && grownRoom.rotation === edited.rotation && grownRoom.space.occupantIds.length === 12 && before.agents.every(a => grown.agents.find(b => b.id === a.id)?.desk === a.desk), { adjustment, previous: edited.position, current: grownRoom.position })
    const savedPosition = layouts[scene].overrides.find(o => o.placeId === roomId)?.position
    if (savedPosition && Math.hypot(savedPosition.x - grownRoom.position[0], savedPosition.z - grownRoom.position[1]) > .01) check(scene + ' explains adjusted saved spaces', await page.getByText('Spaces adjusted to keep the layout clear', { exact: true }).isVisible())
    const roomChildren = grown.places.filter(p => p.parentId === roomId), seatPositions = roomChildren.flatMap(p => p.seats.map(s => s.position.join(',')))
    check(scene + ' growth has distinct seats and nonoverlapping spaces', new Set(seatPositions).size === seatPositions.length && grown.rooms.every((a, i) => grown.rooms.slice(i + 1).every(b => !overlaps(a, b))), grown.rooms.map(r => ({ id: r.id, position: r.position, size: r.size, rotation: r.rotation })))
    check(scene + ' regenerated simulation has no invariant violations', grown.violations.length === 0, grown.violations)
    await page.screenshot({ path: resolve(root, `${scene}-grown.png`) })
    if (scene === 'office') await walkSavedDoor(grownRoom)
  }
  check('no browser errors or agent launches', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error); await page.screenshot({ path: resolve(root, 'failure.png') }).catch(() => {}) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors, writes }, null, 2)); await browser.close() }
if (checks.some(c => !c.pass)) process.exitCode = 1
