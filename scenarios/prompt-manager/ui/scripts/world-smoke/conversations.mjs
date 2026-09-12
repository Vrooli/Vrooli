#!/usr/bin/env node
/** Live roster is read-only; all conversation run traffic is intercepted with
 * wire fixtures. No agent is launched or messaged by this browser journey. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const root = resolve(process.argv[2] ?? `evidence/conversations-${Date.now()}`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = [], requests = []
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
page.on('pageerror', error => errors.push(error.message))
let reply = 0
const run = () => ({ id: 'world-conversation-wire-fixture', task_id: 'task-fixture', status: reply ? 'RUN_STATUS_COMPLETE' : 'RUN_STATUS_RUNNING', actions: { can_continue: reply > 0 } })
await page.route('**/CreateRun', async route => {
  const body = route.request().postDataJSON()
  requests.push(body)
  if (!body.body?.conversation) throw Error('Unexpected non-conversation run creation')
  await route.fulfill({ json: { data: { run: run() } } })
})
await page.route('**/GetRun', async route => {
  if (route.request().postDataJSON().runId !== run().id) return route.continue()
  await route.fulfill({ json: { data: { run: run() } } })
})
await page.route('**/GetRunEvents', async route => {
  const body = route.request().postDataJSON()
  if (body.runId !== run().id) return route.continue()
  const after = Number(body.query?.after_sequence ?? 0)
  await route.fulfill({ json: { data: { events: reply > after ? [{ id: `reply-${reply}`, run_id: run().id, sequence: String(reply), timestamp: '2026-09-08T08:00:00Z', event_type: 'RUN_EVENT_TYPE_MESSAGE', message: { role: 'assistant', content: `Fixture reply ${reply}: the garden is doing well.` } }] : [] } } })
})
await page.route('**/ContinueRun', async route => { requests.push(route.request().postDataJSON()); await route.fulfill({ json: { data: { run: run() } } }) })
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
    if (!world || !render) throw Error('Missing rendered world')
    window.__conversationFixture = { world, render }
    const { scene, camera, gl } = render.getState(), rect = gl.domElement.getBoundingClientRect(), state = world.getState()
    let slime
    scene.traverse(object => { if (object.isInstancedMesh && object.geometry.getAttribute('aSquash')) slime = object })
    return state.actorOrder.map((id, i) => {
      const matrix = slime.matrixWorld.clone(); slime.getMatrixAt(i, matrix)
      const point = camera.position.clone().setFromMatrixPosition(matrix).applyMatrix4(slime.matrixWorld).project(camera)
      return { id, name: state.actors[id].name, teamId: state.actors[id].teamId, x: (point.x + 1) * rect.width / 2 + rect.left, y: (1 - point.y) * rect.height / 2 + rect.top, depth: point.z }
    }).filter(p => p.x > 600 && p.x < 1400 && p.y > 180 && p.y < 850 && p.depth > -1 && p.depth < 1)
  })
}
try {
  await page.goto('http://localhost:21235/world?scene=park&seed=7&profile=medium&intro=0&diag=1&period=day&weather=clear')
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
  await page.getByLabel('Camera mode', { exact: true }).selectOption('explore')
  await page.waitForTimeout(1000)
  const candidates = await inspect()
  let selected
  for (const candidate of candidates) {
    await page.mouse.click(candidate.x, candidate.y); await page.waitForTimeout(200)
    if (await page.getByTestId('world-conversation').isVisible()) { selected = candidate; break }
    const close = page.getByRole('button', { name: 'Close space menu' }); if (await close.isVisible()) await close.click()
  }
  check('Explore agent click opens greeting without a model invocation', Boolean(selected) && requests.length === 0, selected)
  const panel = page.getByTestId('world-conversation'), input = panel.getByRole('textbox')
  await input.fill('How is the garden?'); await input.press('Enter')
  await page.waitForTimeout(300)
  check('first send uses selected member identity', requests[0]?.body?.conversation?.agent_id === selected.id && requests[0]?.body?.conversation?.team_id === selected.teamId, requests)
  await page.waitForFunction(() => window.__worldDiagnostics?.cameraPoseRoute?.status !== 'moving', null, { timeout: 30000 })
  await page.waitForTimeout(500)
  await page.screenshot({ path: resolve(root, 'explore-conversation.png') })
  await panel.getByRole('button', { name: 'Close conversation' }).click()
  await page.getByRole('button', { name: 'Home view', exact: true }).click()
  await page.waitForFunction(() => window.__worldDiagnostics?.cameraPoseRoute?.status !== 'moving', null, { timeout: 30000 })
  reply = 1
  await page.waitForFunction(() => {
    let found = false
    window.__conversationFixture.render.getState().scene.traverse(object => { if (object.visible && typeof object.text === 'string' && object.text.includes('New message')) found = true })
    return found
  }, null, { timeout: 15000 })
  await page.waitForTimeout(1000)
  const markers = await page.evaluate(() => { const { scene, camera } = window.__conversationFixture.render.getState(); const markers = []; scene.traverse(o => { if (o.visible && typeof o.text === 'string' && o.text.includes('New message')) markers.push({ text: o.text, renderedText: o.textRenderInfo?.parameters?.text, position: o.position.toArray(), projected: o.position.clone().project(camera).toArray() }) }); return markers })
  check('closed bubble receives reply and displays an overhead unread marker', markers.length > 0 && markers.every(marker => marker.text === marker.renderedText), markers)
  await page.screenshot({ path: resolve(root, 'unread-marker.png') })
  // A normal focus deep link reopens this member; browser-local receipt and run
  // identity survive a full reload, while the transcript is fetched again.
  await page.goto(`http://localhost:21235/world?scene=park&seed=7&profile=medium&intro=0&diag=1&period=day&weather=clear&focus=${encodeURIComponent(selected.id)}`)
  await panel.getByText('Fixture reply 1: the garden is doing well.').waitFor({ timeout: 90000 })
  check('reload restores the conversation and real event transcript', requests.length === 1)
  await input.fill('Tell me more'); await input.press('Enter')
  await page.waitForTimeout(300)
  check('follow-up continues the existing run', requests[1]?.runId === run().id && requests[1]?.body?.message === 'Tell me more', requests[1])
  await page.waitForFunction(() => window.__worldDiagnostics?.cameraPoseRoute?.status !== 'moving', null, { timeout: 30000 })
  await page.waitForTimeout(500)
  await input.focus(); const before = await page.evaluate(() => window.__worldDiagnostics.cameraNavigation.position)
  await input.press('w'); await input.press('ArrowRight'); await page.waitForTimeout(300)
  const after = await page.evaluate(() => window.__worldDiagnostics.cameraNavigation.position)
  check('composer keyboard input does not move the camera', Math.hypot(...before.map((v, i) => v - after[i])) < .01, { before, after, route: await page.evaluate(() => window.__worldDiagnostics.cameraPoseRoute) })
  await page.screenshot({ path: resolve(root, 'restored-conversation.png') })
  check('no browser errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors, requests }, null, 2)); await browser.close() }
if (checks.some(check => !check.pass)) process.exitCode = 1
