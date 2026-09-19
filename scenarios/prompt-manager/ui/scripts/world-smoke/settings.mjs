#!/usr/bin/env node
/** Normal settings → preparation → URL reload and weather reproducibility. */
import { chromium } from 'playwright-core'
import { existsSync, mkdirSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { execFileSync } from 'node:child_process'

const root = resolve(process.argv[2] ?? 'evidence/world-seed')
if (existsSync(resolve(root, 'result.json'))) throw new Error('Choose a fresh evidence directory')
mkdirSync(root, { recursive: true })
const port = execFileSync('vrooli', ['scenario', 'port', 'prompt-manager', 'ui'], { encoding: 'utf8' }).trim()
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome',
  args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] })
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } })
const checks = []
const errors = []
page.on('pageerror', error => errors.push(error.message))
const check = (name, pass, detail) => {
  checks.push({ name, pass, detail })
  if (!pass) throw new Error(name)
}
const read = () => page.evaluate(() => ({
  digest: document.querySelector('[data-seed-digest]')?.getAttribute('data-seed-digest'),
  epoch: window.__worldDiagnostics.presentationEpoch,
  count: window.__worldSim.generation().count,
}))
const ready = () => page.waitForFunction(() => window.__worldDiagnostics?.ready &&
  document.querySelector('[data-seed-digest]')?.getAttribute('data-seed-digest'), null, { timeout: 90000 })
try {
  await page.goto(`http://localhost:${port}/world?scene=park&actors=25&seed=1&diag=1&period=day&profile=high`)
  await ready()
  const before = await read()
  await page.getByTitle('World Settings', { exact: true }).click()
  const field = page.getByRole('spinbutton', { name: 'World seed', exact: true })
  await field.fill('42')
  await field.press('Enter')
  check('seed input retains focus on apply', await field.evaluate(element => element === document.activeElement))
  await page.waitForFunction(epoch => window.__worldDiagnostics.ready && window.__worldDiagnostics.presentationEpoch > epoch,
    before.epoch, { timeout: 90000 })
  const after = await read()
  check('seed changes terrain', after.digest !== before.digest, { before, after })
  check('one generation requested', after.count === before.count + 1, { before: before.count, after: after.count })
  check('focus survives preparation', await field.evaluate(element => element === document.activeElement))
  check('seed URL retains other settings', new URL(page.url()).searchParams.get('seed') === '42' && new URL(page.url()).searchParams.get('actors') === '25')
  await field.press('Enter')
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))))
  check('same seed does not regenerate', (await read()).count === after.count)
  await page.reload()
  await ready()
  const reloaded = await read()
  check('reload reproduces terrain', reloaded.digest === after.digest, { after, reloaded })
  await page.getByTitle('World Settings', { exact: true }).click()
  check('reload restores selected seed', await field.inputValue() === '42')
  const weather = page.getByRole('radiogroup', { name: 'Weather', exact: true })
  const weatherBefore = await read()
  for (const [label, id] of [['Rain', 'rain'], ['Snow', 'snow'], ['Cloudy', 'cloudy'], ['Clear', 'clear'], ['Automatic', 'clear']]) {
    const choice = weather.getByRole('radio', { name: label, exact: true })
    await choice.click()
    await page.waitForFunction(id => window.__worldDiagnostics.weather === id, id)
    await page.evaluate(() => new Promise(resolve => {
      let frames = 0
      const tick = () => { if (++frames === 30) resolve(); else requestAnimationFrame(tick) }
      requestAnimationFrame(tick)
    }))
    const current = await read()
    check(`${label} weather preserves generated world`, current.count === weatherBefore.count && current.epoch === weatherBefore.epoch && current.digest === weatherBefore.digest, current)
    check(`${label} weather selection remains focused`, await choice.evaluate(element => element === document.activeElement))
    check(`${label} weather updates URL`, new URL(page.url()).searchParams.get('weather') === (label === 'Automatic' ? null : id))
    if (label === 'Snow') await page.screenshot({ path: resolve(root, 'snow.png') })
  }
  const pressureUrl = new URL(page.url())
  pressureUrl.searchParams.set('pressure', '1')
  await page.goto(pressureUrl.toString())
  await ready()
  await page.getByTitle('World Settings', { exact: true }).click()
  await weather.getByRole('radio', { name: 'Automatic', exact: true }).click()
  await page.waitForFunction(() => window.__worldDiagnostics.weather === 'clear')
  check('Automatic clears pressure override', !new URL(page.url()).searchParams.has('pressure'))
  check('no browser errors', errors.length === 0, errors)
} catch (error) {
  checks.push({ name: 'seed browser replay completed', pass: false, detail: String(error) })
  await page.screenshot({ path: resolve(root, 'failure.png') }).catch(() => {})
} finally {
  writeFileSync(resolve(root, 'result.json'), JSON.stringify({ capturedAt: new Date().toISOString(), checks, errors }, null, 2))
  await browser.close()
}
console.log(JSON.stringify(checks, null, 2))
if (checks.some(check => !check.pass)) process.exitCode = 1
