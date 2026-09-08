#!/usr/bin/env node
/** Normal time/weather UI against a synthetic world; renderer reads establish what is actually drawn. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { installFixtureHook } from './camera-fixtures.mjs'
const option = (name, fallback) => { const i = process.argv.indexOf(name); return i < 0 ? fallback : process.argv[i + 1] }
const root = resolve(option('--evidence-dir', `evidence/atmosphere-${Date.now()}`))
mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
await installFixtureHook(page)
const checks = [], errors = [], samples = []
page.on('pageerror', e => errors.push(e.message))
page.on('console', m => { if (m.type() === 'error' && /shader|WebGL|THREE/.test(m.text())) errors.push(m.text()) })
function check(name, pass, detail) { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
async function read() {
  return page.evaluate(() => {
    const { render, clock } = window.__atmosphere
    const { scene, gl } = render.getState()
    const galaxy = scene.getObjectByName('celestial-milky-way'), stars = scene.getObjectByName('celestial-stars')
    const fire = scene.getObjectByName('campfire-effects')
    let resolved
    const pending = [...window.__cameraFixtureRoots].map(r => r.current)
    while (pending.length) { const f = pending.pop(); if (f.memoizedProps?.quiet !== undefined) resolved = { quiet: f.memoizedProps.quiet, lamp: f.memoizedProps.period?.lampEmissive }; if (f.child) pending.push(f.child); if (f.sibling) pending.push(f.sibling) }
    let lights = 0, intensity = 0
    scene.traverse(o => { if (o.isPointLight && o.visible && o.intensity > 0) { lights++; intensity += o.intensity } })
    return { resolved, fire: fire ? { visible: fire.visible, flame: fire.material.uniforms.flame.value, embers: fire.material.uniforms.embers.value, time: fire.material.uniforms.time.value, instances: fire.count, capacity: fire.instanceMatrix.count } : null, time: clock.snapshot(), galaxy: { visible: galaxy.visible, opacity: galaxy.material.uniforms.visibility.value }, stars: stars.material.uniforms.visibility.value, starCount: stars.geometry.drawRange.count,
      lights, intensity, exposure: gl.toneMappingExposure, background: scene.background?.getHexString?.(), camera: window.__worldDiagnostics.cameraNavigation?.mode }
  })
}
async function settings() { await page.getByTitle('World Settings', { exact: true }).click() }
try {
  for (const scene of ['park', 'office']) {
    await page.goto(`http://localhost:21235/world?actors=16&scene=${scene}&intro=0&profile=high&diag=1&period=night&weather=clear`)
    await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 60000 })
    await page.evaluate(() => {
      const pending = [...window.__cameraFixtureRoots].map(r => r.current); let render, clock, world
      while (pending.length) { const f = pending.pop();
        for (const value of [f.memoizedProps?.store, f.memoizedProps?.value]) if (typeof value?.getState === 'function' && value.getState().scene?.isScene) render = value
        for (const value of [f.memoizedProps?.store, f.memoizedProps?.value]) if (typeof value?.getState === 'function' && value.getState().actors && value.getState().nav) world = value
        if (f.memoizedProps?.clock?.snapshot) clock = f.memoizedProps.clock
        if (f.child) pending.push(f.child); if (f.sibling) pending.push(f.sibling)
      }
      if (!render || !clock) throw Error('Missing world inspection handles')
      window.__atmosphere = { render, clock, world }
    })
    await page.waitForTimeout(800)
    const night = await read(); samples.push({ scene, stage: 'night', ...night })
    check(scene + ' ordinary night keeps local illumination', night.lights > 0 && !night.galaxy.visible, night)
    if (scene === 'park') {
      check('park ordinary night draws flames and embers within capacity', night.fire?.flame > 0 && night.fire?.embers > 0 && night.fire?.instances <= night.fire?.capacity, night.fire)
      await page.evaluate(() => { const { render, world } = window.__atmosphere; const hearth = Object.values(world.getState().places).find(p => p.id === 'hearth'); const s = render.getState(); s.controls.setLookAt(hearth.position[0] + 5, 3, hearth.position[1] + 6, hearth.position[0], .5, hearth.position[1], false); s.controls.update(0); s.invalidate() })
      await page.waitForTimeout(400)
      await page.screenshot({ path: resolve(root, 'park-burning-fire.png') })
      await settings()
      await page.getByRole('radio', { name: 'Rain', exact: true }).click()
      await page.waitForFunction(() => !window.__atmosphere.render.getState().scene.getObjectByName('campfire-effects').visible)
      check('rain extinguishes exposed flames and embers', !(await read()).fire.visible)
      await page.getByRole('radio', { name: 'Clear', exact: true }).click()
      await page.getByRole('button', { name: 'Freeze time', exact: true }).click()
      await page.waitForTimeout(400)
      const frozenFire = await read()
      await page.waitForTimeout(300)
      check('frozen time retains a stable burning fire', frozenFire.fire.flame > 0 && frozenFire.fire.time === (await read()).fire.time)
      await page.emulateMedia({ reducedMotion: 'reduce' })
      await page.waitForTimeout(400)
      check('reduced motion retains static fire appearance', (await read()).fire.flame > 0 && (await read()).fire.time === 0)
      await page.emulateMedia({ reducedMotion: 'no-preference' })
      await page.getByText('Exact date and time', { exact: true }).click()
      // Seek a quiet-hours time through the exact-time UI; no simulation mutation.
      const emberUTC = await page.evaluate(() => { const s = window.__atmosphere.clock.snapshot(); return new Date(s.utcMilliseconds + (35 - s.localMinutes) * 60000).toISOString().slice(0, 19) })
      await page.getByLabel('UTC instant', { exact: true }).fill(emberUTC.slice(0, 16))
      await page.getByRole('button', { name: 'Apply UTC instant', exact: true }).click()
      await page.waitForFunction(() => { const f = window.__atmosphere.render.getState().scene.getObjectByName('campfire-effects'); return f.material.uniforms.flame.value === 0 && f.material.uniforms.embers.value > 0 })
      const ember = await read(); check('park quiet hours retain embers after flames stop', ember.fire?.flame === 0 && ember.fire?.embers > 0 && ember.lights > 0, ember)
      await page.keyboard.press('Escape')
      await page.screenshot({ path: resolve(root, 'park-ember-fire.png') })
    }
    await settings()
    check(scene + ' deep-night control is available without workbench', await page.getByRole('button', { name: 'Deep night', exact: true }).isVisible())
    await page.getByRole('button', { name: 'Deep night', exact: true }).click()
    await page.waitForTimeout(400)
    const deep = await read(); samples.push({ scene, stage: 'deep-night', ...deep })
    check(scene + ' preset controls actual civil time and freezes it', deep.time.localMinutes === 150 && deep.time.timeScale === 0, deep)
    check(scene + ' deep night extinguishes lamps and fire lights', deep.lights === 0 && (!deep.fire || !deep.fire.visible), deep)
    check(scene + ' clear deep night reveals Milky Way and stars', deep.galaxy.visible && deep.galaxy.opacity > .1 && deep.stars > .1, deep)
    check(scene + ' deep night reveals additional stars', deep.starCount > night.starCount, { night: night.starCount, deep: deep.starCount })
    check(scene + ' navigation exposure remains usable', deep.exposure >= .7, deep)
    await page.keyboard.press('Escape')
    for (const mode of ['explore', 'first-person', 'third-person']) {
      await page.getByLabel('Camera mode', { exact: true }).selectOption(mode)
      await page.waitForTimeout(400)
      await page.screenshot({ path: resolve(root, `${scene}-${mode}-deep-night.png`) })
      const current = await read()
      check(scene + ' ' + mode + ' retains quiet-night lighting', current.lights === 0 && current.galaxy.visible && current.time.localMinutes === 150, current)
    }
    await page.getByLabel('Camera mode', { exact: true }).selectOption('explore')
    // Use existing Explore controls to inspect the sky above the central landscape.
    await page.evaluate(() => { const s = window.__atmosphere.render.getState(); s.controls.setLookAt(0, 3, 16, 50, 68, -44, false); s.controls.update(0); s.invalidate() })
    await page.waitForTimeout(400)
    await page.screenshot({ path: resolve(root, `${scene}-milky-way.png`) })
    await settings()
    await page.getByRole('radio', { name: 'Rain', exact: true }).click()
    await page.waitForTimeout(400)
    const rain = await read()
    check(scene + ' weather suppresses galactic visibility', rain.galaxy.opacity < deep.galaxy.opacity * .2, rain)
    await page.getByRole('radio', { name: 'Clear', exact: true }).click()
    await page.getByRole('button', { name: 'Midday', exact: true }).click()
    await page.waitForTimeout(400)
    const day = await read(); samples.push({ scene, stage: 'midday', ...day })
    check(scene + ' midday restores daylight and hides night sky', day.time.localMinutes === 720 && !day.galaxy.visible && day.stars === 0 && day.background !== deep.background, day)
    await page.keyboard.press('Escape')
    await page.screenshot({ path: resolve(root, `${scene}-day-sky.png`) })
    await settings()
    await page.getByRole('button', { name: 'Resume live time', exact: true }).click()
    const live = await read()
    check(scene + ' resume restores the live clock', live.time.mode === 'clock' && Math.abs(live.time.utcMilliseconds - Date.now()) < 2000, live)
    await page.keyboard.press('Escape')
  }
  check('no page or GPU errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors, samples }, null, 2)); await browser.close() }
if (checks.some(c => !c.pass)) process.exitCode = 1
