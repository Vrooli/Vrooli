#!/usr/bin/env node
/**
 * Contact sheet: every frame the last smoke run captured, tiled on one page
 * with its counters printed under it, rendered to a single PNG. One image to
 * look at per batch of work instead of one per scene × profile × period.
 *
 *   pnpm world:sheet                 evidence/world-smoke/contact-sheet.png
 *   node scripts/world-smoke/sheet.mjs --tile 480 --out /tmp/sheet.png
 *
 * Reads evidence/world-smoke/summary.json and the PNGs beside it; needs no
 * running scenario, only the Chrome the smoke tool already uses.
 */
import { chromium } from 'playwright-core'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const evidenceDir = resolve(uiRoot, 'evidence', 'world-smoke')
const args = process.argv.slice(2)
const opt = (name, fallback) => {
  const i = args.indexOf(name)
  return i >= 0 && args[i + 1] ? args[i + 1] : fallback
}
const tileWidth = Number(opt('--tile', '480'))
const columns = Number(opt('--columns', '4'))
const out = resolve(opt('--out', resolve(evidenceDir, 'contact-sheet.png')))

const summaryPath = resolve(evidenceDir, 'summary.json')
if (!existsSync(summaryPath)) {
  process.stderr.write(`no smoke summary at ${summaryPath}; run pnpm world:smoke first\n`)
  process.exit(2)
}
const summary = JSON.parse(readFileSync(summaryPath, 'utf8'))
const frames = summary.results.filter((r) => r.drawCalls !== null)
if (frames.length === 0) {
  process.stderr.write('summary has no captured frames\n')
  process.exit(2)
}

const fmt = (n, digits = 0) => (n === null || n === undefined ? '—' : Number(n).toFixed(digits))
const tiles = frames.map((r) => {
  const png = resolve(evidenceDir, `${r.name}.png`)
  const src = existsSync(png) ? `data:image/png;base64,${readFileSync(png).toString('base64')}` : ''
  const failing = r.checks.filter((c) => !c.pass).map((c) => c.id)
  return `
    <figure class="${r.pass ? 'pass' : 'fail'}">
      <img src="${src}" alt="${r.name}">
      <figcaption>
        <strong>${r.name}</strong><span class="verdict">${r.pass ? 'PASS' : `FAIL ${failing.join(', ')}`}</span>
        <span>${fmt(r.drawCalls)} draws · ${fmt(r.triangles / 1000, 0)}k tris · p50 ${fmt(r.p50Ms, 1)} ms · p95 ${fmt(r.p95Ms, 1)} ms · fill ${fmt(r.fill * 100)}% · ${fmt(r.violations)} violations</span>
      </figcaption>
    </figure>`
})

const html = `<!doctype html><meta charset="utf-8"><title>world contact sheet</title>
<style>
  body { margin: 0; background: #14161c; color: #e7e9ee; font: 12px/1.35 ui-monospace, SFMono-Regular, Menlo, monospace; }
  header { padding: 10px 14px; border-bottom: 1px solid #2b2f3a; display: flex; gap: 18px; }
  main { display: grid; grid-template-columns: repeat(${columns}, ${tileWidth}px); gap: 10px; padding: 10px; }
  figure { margin: 0; border: 2px solid #2b2f3a; border-radius: 4px; overflow: hidden; background: #0d0f14; }
  figure.fail { border-color: #d9484f; }
  figure.pass { border-color: #3d8f5a; }
  img { display: block; width: ${tileWidth}px; height: auto; }
  figcaption { padding: 6px 8px; display: grid; gap: 2px; }
  figcaption strong { font-weight: 600; }
  .verdict { color: #9aa3b2; }
  figure.fail .verdict { color: #ff8a8f; }
</style>
<header>
  <span>${summary.gpu ? 'host GPU' : 'SwiftShader (frame times informational)'}</span>
  <span>${frames.length} frames</span>
  <span>${summary.capturedAt ?? ''}</span>
</header>
<main>${tiles.join('')}</main>`

const browser = await chromium.launch({ executablePath: process.env.WORLD_SMOKE_CHROME ?? '/usr/bin/google-chrome', headless: true, args: ['--no-sandbox'] })
try {
  const page = await browser.newPage({ viewport: { width: columns * (tileWidth + 10) + 10, height: 800 }, deviceScaleFactor: 1 })
  await page.setContent(html, { waitUntil: 'load' })
  await page.screenshot({ path: out, fullPage: true, type: 'png' })
  process.stdout.write(`${out}\n`)
} finally {
  await browser.close()
}
