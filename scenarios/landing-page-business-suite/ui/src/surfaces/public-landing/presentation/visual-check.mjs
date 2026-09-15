
/* global process, URL, document, innerWidth, window, console, getComputedStyle */
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
const videoOnly = process.env.PRESENTATION_VIDEO_ONLY === '1';
const editorOnly = process.env.PRESENTATION_EDITOR_ONLY === '1';
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
  const providerRequests = [];
  const blockedAdminFonts = [];
  await page.route('**/*', async route => {
    const url = new URL(route.request().url());
    if (url.origin !== 'https://presentation.test') {
      // Existing admin stylesheet's remote font is not video traffic. Stub and report it;
      // this harness does not qualify the private editor's font delivery.
      if (editorOnly && url.origin === 'https://fonts.googleapis.com') {
        blockedAdminFonts.push(url.href);
        return route.fulfill({ status: 200, contentType: 'text/css', body: '/* Offline admin review: no remote font delivery claim. */' });
      }
      providerRequests.push(url.href);
      return route.abort();
    }
    const directory = url.pathname.startsWith('/presentation/') ? join(ui, 'public') : built;
    const path = resolve(directory, '.' + url.pathname);
    if (!path.startsWith(directory + sep)) return route.abort();
    try {
      const body = await readFile(path);
      const contentType = { '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.png': 'image/png', '.ttf': 'font/ttf' }[extname(path)] || 'application/octet-stream';
      await route.fulfill({ status: 200, contentType, body });
    } catch { await route.fulfill({ status: 404, body: 'Missing fixture artifact' }); }
  });
  if (videoOnly) {
    for (const provider of ['youtube', 'vimeo']) for (const layout of ['stacked', 'split']) for (const width of [320, 390, 768, 1440]) {
      providerRequests.length = 0;
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(`https://presentation.test/preview.html?video=1&provider=${provider}&layout=${layout}`);
      const button = page.getByRole('button', { name: 'Load configured player' });
      await button.waitFor();
      await page.evaluate(async () => { await document.fonts.ready; await Promise.all([...document.images].map(img => img.decode())); });
      assert.equal(await page.locator('iframe,video').count(), 0);
      assert.deepEqual(providerRequests, []);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth), width);
      assert.equal(await page.locator('.presentation-page').getAttribute('data-presentation-preview'), 'true');
      const label = `video-${provider}-${layout}-${width}`;
      await audit(label);
      const screenshot = join(out, label + '.png');
      await page.screenshot({ path: screenshot, fullPage: true });
      report.captures.push({ design: label, width, screenshot });
      await button.focus(); await page.keyboard.press('Enter');
      const frame = page.locator('iframe'); await frame.waitFor();
      assert.equal(await frame.getAttribute('referrerpolicy'), 'strict-origin');
      assert.equal(await page.evaluate(() => document.activeElement?.tagName), 'IFRAME');
      assert.match(await frame.getAttribute('src'), provider === 'youtube' ? /^https:\/\/www.youtube-nocookie.com\/embed\// : /^https:\/\/player.vimeo.com\/video\//);
      assert.equal((await frame.getAttribute('allow')).includes('autoplay'), false);
      assert.equal(await page.getByText('Configured caption remains visible.').isVisible(), true);
    }
    providerRequests.length = 0;
    await page.goto('https://presentation.test/preview.html?video=1&poster=missing');
    await page.getByRole('alert').waitFor();
    assert.equal(await page.locator('iframe').count(), 0); assert.deepEqual(providerRequests, []);
    report.checks.push('External video: 16 provider/layout/viewport cases, no pre-click provider requests, keyboard activation/focus, caption, poster failure, axe', 'Provider requests blocked by test harness after click: no actual playback/subtitle claim');
  }
  if (editorOnly) {
    for (const width of [320, 390, 768, 1440]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto('https://presentation.test/preview.html?editor=1');
      await page.getByRole('region', { name: 'Unsaved document preview' }).waitFor();
      await page.evaluate(async () => {
        for (const img of document.images) img.loading = 'eager';
        await document.fonts.ready; await Promise.all([...document.images].map(img => img.decode()));
      });
      const layout = await page.evaluate(() => {
        const grid = document.querySelector('.presentation-editing-grid');
        const left = grid.children[0].getBoundingClientRect(); const right = grid.children[1].getBoundingClientRect();
        return { scroll: document.documentElement.scrollWidth, sideBySide: right.left >= left.right, stacked: right.top >= left.bottom };
      });
      assert.equal(layout.scroll, width, JSON.stringify(layout));
      assert.equal(width >= 1200 ? layout.sideBySide : layout.stacked, true, JSON.stringify(layout));
      await audit('editor-native-preview-' + width);
      const screenshot = join(out, 'editor-' + width + '.png');
      await page.screenshot({ path: screenshot, fullPage: true });
      report.captures.push({ design: 'private-editor', width, screenshot, layout });
      const source = page.getByRole('textbox', { name: 'Complete document JSON' });
      const current = JSON.parse(await source.inputValue()); current.pages[0].title = 'Unsaved browser-check title';
      await source.fill(JSON.stringify(current));
      await page.getByRole('region', { name: 'Unsaved document preview' }).waitFor();
      assert.equal((await source.inputValue()).includes('Unsaved browser-check title'), true);
      await source.fill('{invalid');
      await page.getByText('Preview hidden until the document syntax is valid.').waitFor();
      assert.equal(await page.getByRole('region', { name: 'Unsaved document preview' }).count(), 0);
    }
    assert.deepEqual(providerRequests, []);
    report.checks.push('Mounted editor: native preview, desktop columns/mobile stacking, 300ms local read path and invalid-edit hiding, 4 viewports, no network/API/backend or write operations');
    if (blockedAdminFonts.length) report.checks.push(`Existing admin font stylesheet requests stubbed offline (${blockedAdminFonts.length}); private editor font delivery not qualified`);
  }
  for (const design of videoOnly || editorOnly ? [] : ['signal', 'studio']) {
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
        placementMismatch: [...document.images].some(image => {
          const computed = getComputedStyle(image);
          return !image.style.objectFit || !image.style.objectPosition || computed.objectFit !== image.style.objectFit || computed.objectPosition !== image.style.objectPosition;
        }),
      }));
      assert.equal(state.scroll, width, JSON.stringify({ design, ...state }));
      assert.equal(state.missing, 0);
      assert.equal(state.placementMismatch, false, 'Released image placement was overridden');
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
  if (!videoOnly && !editorOnly) {
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
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 1050 });
    await page.goto('https://presentation.test/preview.html?commerce=1');
    await page.getByRole('combobox').selectOption('0');
    await page.evaluate(async () => { await document.fonts.ready; });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth), width);
    const screenshot = join(out, 'commerce-' + width + '.png');
    await page.screenshot({ path: screenshot, fullPage: true });
    report.captures.push({ design: 'commerce', width, screenshot });
    await audit('commerce-' + width);
  }
  report.checks.push('Both designs: 320/390/768/1440, no overflow or missing images', 'Mobile menu/Escape, workspace conversation, artifact End/Home', 'axe browser audit: no violations');
  }
  assert.equal(report.accessibility.every(result => result.violations.length === 0), true, JSON.stringify(report.accessibility));
  assert.deepEqual(report.errors, []);
} finally {
  await browser.close();
  await writeFile(join(out, 'report.json'), JSON.stringify(report, null, 2));
  console.log('PRESENTATION_EVIDENCE=' + out);
  console.log(JSON.stringify(report, null, 2));
}
