import assert from 'node:assert/strict'
import { chromium } from 'playwright-core'

const baseURL = process.env.PROMPT_MANAGER_URL ?? 'http://127.0.0.1:21235'
const browser = await chromium.launch({
  executablePath: process.env.CHROME_BIN ?? '/usr/bin/google-chrome',
  headless: true,
  args: ['--no-sandbox'],
})

try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 800 } })
  await page.goto(`${baseURL}/world`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(800)

  const viewport = await page.evaluate(() => ({
    viewportHeight: window.innerHeight,
    documentHeight: document.documentElement.scrollHeight,
  }))
  assert.equal(
    viewport.documentHeight,
    viewport.viewportHeight,
    `app shell overflows viewport: ${JSON.stringify(viewport)}`,
  )

  await page.getByTestId('skill-sidebar-tab-agents').click()
  const agentRow = page.getByTestId('agent-row').first()
  await agentRow.waitFor()
  await agentRow.click()
  assert.match(page.url(), /\/agents\/[^/]+$/)

  await page.goto(`${baseURL}/world`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(800)
  await page.getByRole('tab', { name: 'Teams' }).click()
  const teamRow = page.getByTestId('team-row').first()
  await teamRow.waitFor()
  await teamRow.click()
  assert.match(page.url(), /\/teams\/[^/]+$/)

  await page.goto(`${baseURL}/world`, { waitUntil: 'domcontentloaded' })
  await page.waitForTimeout(800)
  await page.getByRole('tab', { name: 'Runs' }).click()
  await page.getByTestId('run-list').waitFor()
  await page.waitForTimeout(500)
  assert.equal(await page.getByText('Failed to load runs').count(), 0)
} finally {
  await browser.close()
}
