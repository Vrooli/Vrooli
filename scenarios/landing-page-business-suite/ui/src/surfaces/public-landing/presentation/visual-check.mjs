
/* global process, URL, document, innerWidth, window, console */
// Standalone browser proof; no scenario start, production mount, or API access.
// Run from ui/ with PRESENTATION_PLAYWRIGHT_MODULE pointing at installed Playwright.
import { mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, extname, join, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import { build } from 'vite';
import react from '@vitejs/plugin-react';
import assert from 'node:assert/strict';
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PRESENTATION_PLAYWRIGHT_MODULE || 'playwright');
const root = dirname(fileURLToPath(import.meta.url));
const ui = resolve(root, '../../../..');
const out = await mkdtemp(join(tmpdir(), 'lpbs-presentation-review-'));
const built = join(out, 'build');
await build({ configFile: false, root, base: '/', publicDir: false, plugins: [react()],
  build: { outDir: built, emptyOutDir: true, rollupOptions: { input: join(root, 'preview.html') } } });
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true, args: ['--no-sandbox', '--disable-dev-shm-usage'] });
const report = { scope: 'isolated native React fixtures; no public mounting', checks: [], captures: [], errors: [], accessibility: [] };
try {
  const page = await browser.newPage({ reducedMotion: 'reduce', deviceScaleFactor: 1 });
  const axeSource = await readFile(require.resolve('axe-core/axe.min.js'), 'utf8');
  async function audit(label) {
    await page.addScriptTag({ content: axeSource });
    const violations = await page.evaluate(async () => {
      const result = await window.axe.run(document.querySelector('.presentation-page'));
      return result.violations.map(item => ({ id: item.id, impact: item.impact, nodes: item.nodes.map(node => ({ target: node.target, summary: node.failureSummary })) }));
    });
    report.accessibility.push({ label, violations });
  }
  page.on('pageerror', error => report.errors.push(error.message));
  await page.route('**/*', async route => {
    const url = new URL(route.request().url());
    if (url.origin !== 'https://presentation.test') return route.abort();
    const directory = url.pathname.startsWith('/presentation/') ? join(ui, 'public') : built;
    const path = resolve(directory, '.' + url.pathname);
    if (!path.startsWith(directory + sep)) return route.abort();
    try {
      const body = await readFile(path);
      const contentType = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.png': 'image/png', '.ttf': 'font/ttf' }[extname(path)] || 'application/octet-stream';
      await route.fulfill({ status: 200, contentType, body });
    } catch { await route.fulfill({ status: 404, body: 'Missing fixture artifact' }); }
  });
  for (const design of ['signal', 'studio']) {
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: width > 600 ? 1050 : 844 });
      await page.goto('https://presentation.test/preview.html?design=' + design);
      await page.locator('h1').waitFor();
      await page.evaluate(async () => {
        for (const image of document.images) image.loading = 'eager';
        await document.fonts.ready;
        await Promise.all([...document.images].map(image => image.decode()));
      });
      const state = await page.evaluate(() => ({
        width: innerWidth, scroll: document.documentElement.scrollWidth,
        images: [...document.images].length, heading: document.querySelector('h1')?.innerText,
        missing: [...document.images].filter(image => !image.naturalWidth).length,
      }));
      assert.equal(state.scroll, width, JSON.stringify({ design, ...state }));
      assert.equal(state.missing, 0);
      const screenshot = join(out, design + '-' + width + '.png');
      await page.screenshot({ path: screenshot });
      report.captures.push({ design, width, screenshot, state });
      if (width === 390 || width === 1440) {
        await page.screenshot({ path: join(out, design + '-' + width + '-full.png'), fullPage: true });
        const regions = design === 'signal' ? ['artifacts', 'voice', 'mobile', 'roadmap', 'get-started'] : ['apps', 'get-started'];
        for (const region of regions) await page.locator('#' + region).screenshot({ path: join(out, design + '-' + width + '-' + region + '.png') });
        await audit(design + '-' + width);
      }
    }
  }
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('https://presentation.test/preview.html?design=signal');
  await page.getByRole('button', { name: 'Toggle navigation' }).click();
  assert.equal(await page.getByRole('navigation', { name: 'Main navigation' }).isVisible(), true);
  await page.keyboard.press('Escape');
  assert.equal(await page.getByRole('navigation', { name: 'Main navigation' }).isVisible(), false);
  await page.getByRole('button', { name: 'Conversation', exact: true }).click();
  assert.equal(await page.locator('.messages-view').isVisible(), true);
  const tabs = page.getByRole('tab');
  await tabs.first().focus();
  await page.keyboard.press('End');
  assert.equal(await tabs.last().getAttribute('aria-selected'), 'true');
  await page.keyboard.press('Home');
  assert.equal(await tabs.first().getAttribute('aria-selected'), 'true');
  for (let index = 0; index < await tabs.count(); index++) {
    await tabs.nth(index).click();
    await audit('signal-390-conversation-artifact-' + index);
    await page.locator('#artifacts').screenshot({ path: join(out, 'signal-390-artifact-' + index + '.png') });
  }
  assert.equal(report.accessibility.every(result => result.violations.length === 0), true, JSON.stringify(report.accessibility));
  assert.deepEqual(report.errors, []);
  report.checks.push('Both designs: 320/390/768/1440, no overflow or missing images', 'Mobile menu/Escape, workspace conversation, artifact End/Home', 'axe browser audit: no violations');
} finally {
  await browser.close();
  await writeFile(join(out, 'report.json'), JSON.stringify(report, null, 2));
  console.log('PRESENTATION_EVIDENCE=' + out);
  console.log(JSON.stringify(report, null, 2));
}
