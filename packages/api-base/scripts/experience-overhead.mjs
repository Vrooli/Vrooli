#!/usr/bin/env node
/**
 * Measure the browser cost of the experience agent.
 *
 * Serves the same fixture twice (telemetry off and on), reloads it, and times:
 * - document load (navigation timing);
 * - 100 synthetic clicks (the worst case because each click starts a dead-click
 *   observation window);
 * - long tasks reported by the browser.
 *
 * Prints the deltas. It intentionally does not fail on a small absolute
 * difference; the end-to-end script is the correctness gate.
 */

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createScenarioServer } from '../dist/server/index.js'

function median(values) {
  if (values.length === 0) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const middle = Math.floor(sorted.length / 2)
  return sorted.length % 2 === 1 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2
}

function writeFixture(dir) {
  fs.writeFileSync(path.join(dir, 'index.html'), `<!doctype html>
<html>
<head><title>Experience overhead</title></head>
<body>
  <button data-testid="pulse" aria-label="Pulse">Pulse</button>
  <div id="log"></div>
  <script>
    document.querySelector('[data-testid="pulse"]').addEventListener('click', () => {
      document.getElementById('log').textContent = String(Date.now())
    })
  </script>
</body>
</html>`)
}

async function startServer({ distDir, dataDir, enabled }) {
  const app = createScenarioServer({
    uiPort: 44191,
    apiPort: 44192,
    distDir,
    serviceName: 'experience-overhead',
    disableLifecycleGuard: true,
    experienceTelemetry: enabled ? { enabled: true, dataDir } : false,
  })
  const server = app.listen(0, '127.0.0.1')
  await new Promise((resolve) => server.once('listening', resolve))
  return { server, port: server.address().port }
}

async function measure(browser, port) {
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } })
  await page.goto(`http://127.0.0.1:${port}/`, { waitUntil: 'load' })
  await page.waitForTimeout(250)

  const loadMs = []
  for (let index = 0; index < 5; index += 1) {
    await page.reload({ waitUntil: 'load' })
    loadMs.push(await page.evaluate(() => {
      const navigation = performance.getEntriesByType('navigation')[0]
      return navigation ? navigation.loadEventEnd - navigation.startTime : performance.now()
    }))
  }

  const click = await page.evaluate(() => {
    const button = document.querySelector('[data-testid="pulse"]')
    const started = performance.now()
    for (let index = 0; index < 100; index += 1) button.click()
    return performance.now() - started
  })

  const longTasks = await page.evaluate(() => new Promise((resolve) => {
    let count = 0
    let total = 0
    if (typeof PerformanceObserver === 'undefined') {
      resolve({ count, total })
      return
    }
    try {
      const observer = new PerformanceObserver((list) => {
        list.getEntries().forEach((entry) => {
          count += 1
          total += entry.duration
        })
      })
      observer.observe({ entryTypes: ['longtask'] })
    } catch {
      resolve({ count, total })
      return
    }
    setTimeout(() => resolve({ count, total }), 600)
  }))

  await page.close()
  return { loadMs: median(loadMs), clickMs: click, longTasks }
}

async function main() {
  const { chromium } = await import('playwright')
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-overhead-'))
  const distDir = path.join(root, 'dist')
  fs.mkdirSync(distDir)
  writeFixture(distDir)

  const off = await startServer({ distDir, dataDir: path.join(root, 'off'), enabled: false })
  const on = await startServer({ distDir, dataDir: path.join(root, 'on'), enabled: true })
  const browser = await chromium.launch({ headless: true })

  try {
    const baseline = await measure(browser, off.port)
    const instrumented = await measure(browser, on.port)
    const loadDelta = instrumented.loadMs - baseline.loadMs
    const clickDelta = instrumented.clickMs - baseline.clickMs

    process.stdout.write(`load median: off ${baseline.loadMs.toFixed(1)} ms, on ${instrumented.loadMs.toFixed(1)} ms (delta ${loadDelta.toFixed(1)} ms)\n`)
    process.stdout.write(`100 clicks: off ${baseline.clickMs.toFixed(1)} ms, on ${instrumented.clickMs.toFixed(1)} ms (delta ${clickDelta.toFixed(1)} ms)\n`)
    process.stdout.write(`long tasks: off ${baseline.longTasks.count}/${baseline.longTasks.total.toFixed(1)} ms, on ${instrumented.longTasks.count}/${instrumented.longTasks.total.toFixed(1)} ms\n`)
    process.stdout.write(clickDelta < 50 ? 'PASS: click overhead is small\n' : 'WARN: click overhead is above 50 ms for 100 clicks\n')
  } finally {
    await browser.close()
    await new Promise((resolve) => off.server.close(resolve))
    await new Promise((resolve) => on.server.close(resolve))
    fs.rmSync(root, { recursive: true, force: true })
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack || error.message : error)
  process.exitCode = 1
})
