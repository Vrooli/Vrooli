#!/usr/bin/env node
/**
 * Builds evidence/world-smoke/compare.html: the review's "before" panels next
 * to the current smoke captures, embedded as data URIs so the page is
 * self-contained.
 *
 *   node scripts/world-smoke/compare.mjs --before <dir-with-jpg-or-png>
 */
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { extname, resolve } from 'node:path'

const uiRoot = resolve(import.meta.dirname, '..', '..')
const evidenceDir = resolve(uiRoot, 'evidence', 'world-smoke')
const args = process.argv.slice(2)
const beforeDir = args.includes('--before') ? args[args.indexOf('--before') + 1] : null

function dataUri(path) {
  const ext = extname(path).slice(1).toLowerCase()
  const mime = ext === 'jpg' || ext === 'jpeg' ? 'image/jpeg' : 'image/png'
  return `data:${mime};base64,${readFileSync(path).toString('base64')}`
}

const before = beforeDir && existsSync(beforeDir)
  ? readdirSync(beforeDir).filter((f) => /\.(jpe?g|png)$/i.test(f)).sort().map((f) => ({ name: f, src: dataUri(resolve(beforeDir, f)) }))
  : []
const after = readdirSync(evidenceDir)
  .filter((f) => f.endsWith('.png') && !f.endsWith('.diff.png'))
  .sort()
  .map((f) => {
    const json = resolve(evidenceDir, f.replace(/\.png$/, '.json'))
    const record = existsSync(json) ? JSON.parse(readFileSync(json, 'utf8')) : null
    return { name: f, src: dataUri(resolve(evidenceDir, f)), record }
  })

const card = (title, src, meta) => `<figure><img src="${src}" alt="${title}"><figcaption><strong>${title}</strong>${meta ? `<br>${meta}` : ''}</figcaption></figure>`
const html = `<!doctype html><meta charset="utf-8"><title>World before and after</title>
<style>body{font:14px system-ui;margin:24px;background:#f6f6f4;color:#222}h1,h2{font-weight:600}.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(420px,1fr));gap:16px}figure{margin:0;background:#fff;border:1px solid #ddd;border-radius:8px;padding:8px}img{width:100%;height:auto;border-radius:4px}figcaption{margin-top:6px;font-size:12px;color:#444}</style>
<h1>prompt-manager world: before and after</h1>
<p>Generated ${new Date().toISOString()}. Left: the review's roast panels (legacy world). Right: current smoke captures with their measured counters.</p>
<h2>Before (${before.length} panels)</h2><div class="grid">${before.map((b) => card(b.name, b.src, '')).join('')}</div>
<h2>After (${after.length} captures)</h2><div class="grid">${after
  .map((a) => card(a.name, a.src, a.record?.diagnostics ? `draw calls ${a.record.diagnostics.drawCalls} · triangles ${a.record.diagnostics.triangles.toLocaleString()} · p95 ${a.record.diagnostics.frameMsP95.toFixed(1)} ms · ${a.record.gpu ? 'GPU' : 'SwiftShader'} · ${a.record.pass ? 'PASS' : 'FAIL'}` : ''))
  .join('')}</div>`
writeFileSync(resolve(evidenceDir, 'compare.html'), html)
console.log(`wrote ${resolve(evidenceDir, 'compare.html')} (${before.length} before, ${after.length} after)`)
