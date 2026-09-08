#!/usr/bin/env node
/** Actual sky/material/light inspection, with normal time controls and walking
 * look controls. Synthetic agents avoid any live management mutations. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const root = resolve(process.argv[2] ?? `evidence/moon-${Date.now()}`)
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 }, timezoneId: 'America/New_York', deviceScaleFactor: 2 })
await installFixtureHook(page)
const checks = [], samples = [], errors = []
page.on('pageerror', error => errors.push(error.message))
page.on('console', message => { if (message.type() === 'error' && /shader|WebGL|THREE/.test(message.text())) errors.push(message.text()) })
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
async function read() {
  return page.evaluate(() => {
    const { render, clock } = window.__moonFixture, { scene, camera, gl } = render.getState()
    const moon = scene.getObjectByName('celestial-moon'), sun = scene.getObjectByName('celestial-sun'), key = scene.getObjectByName('celestial-key')
    const moonDirection = moon.position.clone().sub(camera.position).normalize()
    const lightDirection = key.position.clone().sub(key.target.position).normalize()
    return { time: clock.snapshot(), moonVisible: moon.visible, sunVisible: sun.visible, moonDirection: moonDirection.toArray(), lightDirection: lightDirection.toArray(), alignment: moonDirection.dot(lightDirection),
      illumination: (1 + moon.material.uniforms.lightDirection.value.z) / 2, lightVector: moon.material.uniforms.lightDirection.value.toArray(), moonOpacity: moon.material.uniforms.opacity.value,
      keyIntensity: key.intensity, keyColor: key.color.getHexString(), stars: scene.getObjectByName('celestial-stars').material.uniforms.visibility.value, exposure: gl.toneMappingExposure }
  })
}
async function instant(value) {
  await page.getByTitle('World Settings', { exact: true }).click()
  const field = page.getByLabel('UTC instant', { exact: true })
  if (!(await field.isVisible())) await page.getByText('Exact date and time', { exact: true }).click()
  await field.fill(value)
  await page.getByRole('button', { name: 'Apply UTC instant', exact: true }).click()
  await page.keyboard.press('Escape'); await page.waitForTimeout(350)
}
async function lookAtMoon(mode = 'first-person') {
  await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
  await page.getByRole('button', { name: 'Capture mouse to look', exact: true }).click()
  await page.waitForFunction(() => Boolean(document.pointerLockElement))
  await page.evaluate(() => {
    const { camera, scene, gl } = window.__moonFixture.render.getState()
    const moon = scene.getObjectByName('celestial-moon').position.clone().sub(camera.position).normalize()
    const current = camera.getWorldDirection(camera.position.clone())
    const yaw = Math.atan2(moon.x, -moon.z) - Math.atan2(current.x, -current.z)
    const pitch = Math.asin(moon.y) - Math.asin(current.y)
    gl.domElement.dispatchEvent(new PointerEvent('pointermove', { bubbles: true, movementX: Math.round(Math.atan2(Math.sin(yaw), Math.cos(yaw)) / .0025), movementY: Math.round(-pitch / .0025) }))
  })
  await page.waitForTimeout(350)
  await page.evaluate(() => document.exitPointerLock())
}
try {
  await page.goto('http://localhost:21235/world?actors=16&scene=park&seed=7&intro=0&profile=high&diag=1&weather=clear')
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 })
  await page.evaluate(() => {
    const pending = [...window.__cameraFixtureRoots].map(root => root.current); let render, clock
    while (pending.length) { const fiber = pending.pop();
      for (const value of [fiber.memoizedProps?.store, fiber.memoizedProps?.value]) if (typeof value?.getState === 'function' && value.getState().scene?.isScene) render = value
      if (fiber.memoizedProps?.clock?.snapshot) clock = fiber.memoizedProps.clock
      if (fiber.child) pending.push(fiber.child); if (fiber.sibling) pending.push(fiber.sibling)
    }
    if (!render || !clock) throw Error('Missing sky inspection handles')
    window.__moonFixture = { render, clock }
  })
  for (const [name, utc, min, max] of [
    ['full', '2026-01-03T05:00', .98, 1], ['new', '2026-01-18T05:00', 0, .02],
    ['waxing-quarter', '2026-01-26T02:00', .45, .55], ['waning-quarter', '2026-01-10T08:00', .45, .58],
    ['waxing-crescent', '2026-01-22T00:00', .03, .2], ['waning-crescent', '2026-01-15T10:00', .03, .2],
  ]) {
    await instant(utc)
    await page.getByRole('button', { name: 'Home view', exact: true }).click()
    await page.waitForFunction(() => window.__worldDiagnostics.cameraPoseRoute?.status !== 'moving', null, { timeout: 30000 })
    await page.waitForTimeout(350)
    const sample = { name, ...await read() }; samples.push(sample)
    check(name + ' rendered phase matches the expected phase', sample.illumination >= min && sample.illumination <= max, sample)
    await page.screenshot({ path: resolve(root, `${name}-ground.png`) })
    if (name === 'new') { check('new moon below the night horizon casts no direct moonlight', !sample.moonVisible && sample.keyIntensity === 0, sample); continue }
    check(name + ' actual key light follows the visible moon', sample.moonVisible && !sample.sunVisible && sample.alignment > .9999 && sample.keyIntensity > 0, sample)
    await lookAtMoon()
    await page.screenshot({ path: resolve(root, `${name}-sky.png`) })
    const crop = await page.evaluate(() => {
      const { camera, scene, gl } = window.__moonFixture.render.getState(), rect = gl.domElement.getBoundingClientRect()
      const point = scene.getObjectByName('celestial-moon').position.clone().project(camera)
      return { x: Math.round(rect.left + (point.x + 1) * rect.width / 2 - 55), y: Math.round(rect.top + (1 - point.y) * rect.height / 2 - 55), width: 110, height: 110 }
    })
    check(name + ' moon is visible through normal walking look controls', crop.x >= 0 && crop.y >= 0 && crop.x + 110 <= 1600 && crop.y + 110 <= 1000, crop)
    await page.screenshot({ path: resolve(root, `${name}-moon-detail.png`), clip: crop })
    if (name === 'full') { await lookAtMoon('third-person'); check('third person retains the same moon and light direction', (await read()).alignment > .9999); await page.screenshot({ path: resolve(root, 'full-third-person-sky.png') }) }
  }
  const full = samples.find(s => s.name === 'full'), empty = samples.find(s => s.name === 'new')
  check('full moon illuminates the world more than quarters and crescents', samples.filter(s => !['full', 'new'].includes(s.name)).every(s => s.keyIntensity < full.keyIntensity / 2), samples.map(s => ({ name: s.name, intensity: s.keyIntensity })))
  check('moonless nights retain navigation exposure and reveal more stars', empty.exposure >= .7 && empty.stars > full.stars, { full, empty })
  await instant('2026-01-03T05:00')
  await page.getByTitle('World Settings', { exact: true }).click()
  await page.getByRole('radio', { name: 'Rain', exact: true }).click(); await page.keyboard.press('Escape'); await page.waitForTimeout(400)
  const wet = await read(); check('cloud cover dims both moon disc and moonlight', wet.moonOpacity < full.moonOpacity && wet.keyIntensity < full.keyIntensity / 2, wet)
  check('no browser or shader errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, samples, errors }, null, 2)); await browser.close() }
if (checks.some(check => !check.pass)) process.exitCode = 1
