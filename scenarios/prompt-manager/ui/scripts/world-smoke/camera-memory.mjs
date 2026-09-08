#!/usr/bin/env node
/** Read-only live roster; only browser-local camera state is changed. */
import { chromium } from 'playwright-core'
import { mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
const option = (key, fallback) => { const i = process.argv.indexOf(key); return i < 0 ? fallback : process.argv[i + 1] }
const root = resolve(option('--evidence-dir', `evidence/camera-memory-${Date.now()}`)); mkdirSync(root, { recursive: true })
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
const checks = [], errors = [], samples = []
page.on('pageerror', e => errors.push(e.message))
const check = (name, pass, detail) => { checks.push({ name, pass, detail }); console.log(`${pass ? 'PASS' : 'FAIL'} ${name}`); if (!pass) throw Error(name) }
const dist = (a, b) => Math.hypot(...a.map((v, i) => v - b[i]))
const read = () => page.evaluate(() => ({ ...window.__worldDiagnostics.cameraNavigation }))
async function ready() { await page.waitForFunction(() => window.__worldDiagnostics?.ready && window.__worldDiagnostics.cameraNavigation?.position, null, { timeout: 120000 }); await page.waitForTimeout(1000) }
async function key(value, ms = 220) { await page.locator('canvas').focus(); await page.keyboard.down(value); await page.waitForTimeout(ms); await page.keyboard.up(value); await page.waitForTimeout(200) }
try {
  for (const scene of ['park', 'office']) {
    const url = `http://localhost:21235/world?scene=${scene}&seed=7&profile=medium&intro=0&diag=1&period=day&weather=clear`
    await page.goto(url); await ready()
    await key('d'); await key('ArrowRight')
    await page.getByRole('button', { name: 'Zoom in', exact: true }).click(); await page.waitForTimeout(500)
    const before = await read(); await page.goto(url.replace('intro=0', 'intro=1')); await ready(); const after = await read()
    check(`${scene} reload restores Explore eye and target`, after.mode === 'explore' && dist(before.position, after.position) < .03 && dist(before.target, after.target) < .03, { before, after })
    for (const mode of ['first-person', 'third-person']) {
      await page.getByLabel('Camera mode', { exact: true }).selectOption(mode); await page.waitForTimeout(700)
      await key('ArrowRight'); await key('ArrowUp', 100); await key('w', 150)
      if (mode === 'third-person') { await page.getByRole('button', { name: 'Zoom out', exact: true }).click(); await page.getByRole('button', { name: 'Zoom out', exact: true }).click(); await page.waitForTimeout(300) }
      const before = await read(); await page.reload(); await ready(); const after = await read(); samples.push({ scene, mode, before, after })
      check(`${scene} reload restores ${mode} body position`, after.mode === mode && dist(before.playerPosition, after.playerPosition) < .03, { before, after })
      check(`${scene} reload restores ${mode} look and camera distance`, dist(before.position, after.position) < .05 && dist(before.target, after.target) < .05, { before, after })
      check(`${scene} restored walking mode owns keyboard focus`, await page.locator('canvas').evaluate(c => c === document.activeElement))
      await page.screenshot({ path: resolve(root, `${scene}-${mode}-restored.png`) })
    }
    await page.getByLabel('Camera mode', { exact: true }).selectOption('explore'); await page.waitForTimeout(300)
    const returned = await read()
    check(`${scene} restored walking retains the saved Explore view`, dist(returned.position, after.position) < .03 && dist(returned.target, after.target) < .03, { expected: after, returned })
    // A second scene starts with its own view; returning recovers this scene.
  }
  check('no browser errors', errors.length === 0, errors)
} catch (error) { checks.push({ name: 'journey completed', pass: false, detail: String(error) }); console.error(error) }
finally { writeFileSync(resolve(root, 'result.json'), JSON.stringify({ checks, errors, samples }, null, 2)); await browser.close() }
if (checks.some(c => !c.pass)) process.exitCode = 1
