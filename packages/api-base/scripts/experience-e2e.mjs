#!/usr/bin/env node
/**
 * End-to-end check for local experience telemetry.
 *
 * Serves a throwaway UI through the real shared server, drives it with
 * Chromium, then analyzes the recorded session. Exits non-zero on any broken
 * link in the chain: injection, capture, ingest, storage, or report.
 *
 * Run: node scripts/experience-e2e.mjs
 */

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createScenarioServer } from '../dist/server/index.js'
import { readSessions, sessionStatus } from '../dist/experience/store.js'
import { analyzeExperience, buildSessions, renderFrictionMarkdown } from '../dist/experience/report.js'
import {
  createObservedEvidence,
  createPacketObservedBlock,
  deriveAdvisoryFindings,
  deriveCorrectionCandidates,
  validateReport,
} from '../dist/experience/contracts.js'

function assert(condition, message) {
  if (!condition) throw new Error(`FAIL: ${message}`)
}

function writeFixture(dir) {
  fs.writeFileSync(path.join(dir, 'index.html'), `<!doctype html>
<html>
<head><title>Experience E2E</title></head>
<body>
  <main data-experience-region="workspace">
    <button data-testid="save" aria-label="Save changes">Save</button>
    <button data-testid="inert" aria-label="Inert">Inert</button>
    <input data-testid="name" aria-label="Name" />
    <button data-testid="next" aria-label="Next">Next</button>
    <div id="log"></div>
    <div id="page">first</div>
  </main>
  <script>
    document.querySelector('[data-testid="save"]').addEventListener('click', () => {
      document.getElementById('log').textContent += '+'
    })
    document.querySelector('[data-testid="next"]').addEventListener('click', () => {
      document.getElementById('page').textContent = 'second'
      history.pushState({}, '', '/next')
    })
  </script>
</body>
</html>`)
}

async function main() {
  const { chromium } = await import('playwright')
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-e2e-'))
  const distDir = path.join(root, 'dist')
  const dataDir = path.join(root, 'sessions')
  const reportDir = path.join(root, 'report')
  fs.mkdirSync(distDir, { recursive: true })
  writeFixture(distDir)

  const app = createScenarioServer({
    uiPort: 43991,
    apiPort: 43992,
    distDir,
    serviceName: 'experience-e2e',
    disableLifecycleGuard: true,
    experienceTelemetry: { enabled: true, dataDir, captureLabels: true },
  })
  const server = app.listen(0, '127.0.0.1')
  await new Promise((resolve) => server.once('listening', resolve))
  const port = server.address().port
  const base = `http://127.0.0.1:${port}`

  const browser = await chromium.launch({ headless: true })
  try {
    const page = await browser.newPage({ viewport: { width: 1280, height: 800 } })
    await page.goto(`${base}/`, { waitUntil: 'load' })
    await page.waitForFunction(() => Boolean(window.__vrooliExperience), null, { timeout: 5000 })

    await page.click('[data-testid="save"]')
    for (let index = 0; index < 4; index += 1) await page.click('[data-testid="inert"]')
    // Let the dead-click window close before anything else changes the DOM.
    await page.waitForTimeout(1000)
    await page.fill('[data-testid="name"]', 'matthew')
    await page.click('[data-testid="next"]')
    // The agent flushes every 2.5s. Wait past one flush so the session exists
    // even if the final beacon is raced by shutdown.
    await page.waitForTimeout(3000)

    await page.close()
  } finally {
    await browser.close()
    await new Promise((resolve) => setTimeout(resolve, 300))
    await new Promise((resolve) => server.close(resolve))
  }

  // pagehide uses sendBeacon; give the server a moment to append the last batch.
  let read = readSessions(dataDir)
  for (let attempt = 0; attempt < 20 && read.batches.length === 0; attempt += 1) {
    await new Promise((resolve) => setTimeout(resolve, 100))
    read = readSessions(dataDir)
  }
  assert(sessionStatus(dataDir).sessions === 1, 'expected one session file')
  assert(read.batches.length > 0, 'no batches were ingested')

  const events = read.batches.flatMap((batch) => batch.events)
  const actions = events.filter((event) => event.type === 'action')
  const clicks = actions.filter((event) => event.action === 'click')
  const testIds = clicks.map((event) => event.element && event.element.testId)
  assert(testIds.includes('save'), 'save click was not captured')
  assert(clicks.some((event) => event.element && event.element.region === 'workspace'), 'region was not resolved')
  assert(events.some((event) => event.type === 'route' && event.route === '/next'), 'pushState route was not captured')

  const frictionKinds = events.filter((event) => event.type === 'friction').map((event) => event.kind)
  assert(frictionKinds.includes('dead_click'), 'dead click was not detected')
  assert(frictionKinds.includes('rage_click'), 'rage click was not detected')
  assert(frictionKinds.includes('repeated_action'), 'repeated action was not detected')

  const field = events.find((event) => event.type === 'field' && event.phase === 'blur')
  assert(field && field.field.durationMs >= 0, 'field focus duration was not captured')
  assert(JSON.stringify(events).includes('matthew') === false, 'field value leaked into telemetry')

  const { report } = analyzeExperience(read.batches, { scenario: 'experience-e2e', minFlowCount: 2 })
  const uniqueActions = buildSessions(read.batches)
    .flatMap((session) => session.events)
    .filter((event) => event.type === 'action')
  assert(report.totals.sessions === 1, 'report did not see the session')
  assert(report.totals.actions === uniqueActions.length, 'report action count disagrees with de-duplicated capture')
  assert(report.totals.actions > 0, 'report saw no actions')
  assert(report.friction.some((item) => item.kind === 'dead_click'), 'report dropped friction')
  assert(report.recommendations.length > 0, 'report produced no recommendations')
  assert(/^sha256:[a-f0-9]{64}$/.test(report.reportDigest || ''), 'report digest is missing')
  assert(validateReport(report).ok, 'report failed its own validation')

  const evidence = createObservedEvidence(report)
  assert(evidence.state === 'fresh', 'observed evidence should be fresh')
  assert(evidence.reportDigest === report.reportDigest, 'evidence digest does not match the report')

  const advisory = deriveAdvisoryFindings(report)
  assert(advisory.length > 0, 'no advisory findings were derived')
  assert(advisory.every((finding) => finding.advisory === true), 'a finding was not marked advisory')

  const candidates = deriveCorrectionCandidates(report)
  assert(candidates.some((candidate) => candidate.kind === 'dead_click'), 'dead-click candidate was not derived')
  assert(candidates.every((candidate) => candidate.status === 'candidate'), 'a candidate was not review-only')

  const packet = createPacketObservedBlock(report)
  assert(packet.reportDigest === report.reportDigest, 'packet digest does not match the report')
  assert(packet.caveat.includes('Advisory only'), 'packet block lost its caveat')

  fs.mkdirSync(reportDir, { recursive: true })
  fs.writeFileSync(path.join(reportDir, 'friction-report.md'), renderFrictionMarkdown(report))
  fs.writeFileSync(path.join(reportDir, 'friction-report.json'), JSON.stringify(report, null, 2))

  process.stdout.write(`PASS: ${events.length} events, ${actions.length} actions, ${report.recommendations.length} recommendations\n`)
  process.stdout.write(`Report: ${path.join(reportDir, 'friction-report.md')}\n`)
  process.stdout.write(`${renderFrictionMarkdown(report).split('\n').slice(0, 12).join('\n')}\n`)
}

main().catch((error) => {
  console.error(error instanceof Error ? error.stack || error.message : error)
  process.exitCode = 1
})
