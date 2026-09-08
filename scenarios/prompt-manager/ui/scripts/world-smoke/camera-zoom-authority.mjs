// Normal settings and workbench controls share the actual camera zoom target.
import { chromium } from 'playwright-core';
import { writeFileSync } from 'node:fs';
import { installFixtureHook } from './camera-fixtures.mjs';
const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox', '--ignore-gpu-blocklist', '--use-gl=angle', '--use-angle=gl-egl'] });
const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
await installFixtureHook(page);
const checks = [], errors = [];
page.on('pageerror', error => errors.push(error.message));
const check = (name, pass) => { checks.push({ name, pass }); if (!pass) throw new Error(name); };
try {
  await page.goto('http://localhost:21235/world?scene=park&actors=25&seed=1&profile=high&diag=1&workbench=1');
  await page.waitForFunction(() => window.__worldDiagnostics?.ready, null, { timeout: 90000 });
  await page.evaluate(() => {
    const pending = [...window.__cameraFixtureRoots].map(root => root.current);
    while (pending.length) {
      const fiber = pending.pop();
      for (const value of [fiber.memoizedProps?.store, fiber.memoizedProps?.value]) {
        if (typeof value?.getState !== 'function') continue;
        if (value.getState().scene?.isScene) window.__zoomRender = value;
        if (value.getState().nav) window.__zoomStore = value;
      }
      if (fiber.child) pending.push(fiber.child);
      if (fiber.sibling) pending.push(fiber.sibling);
    }
    window.__zoomBefore = { nav: window.__zoomStore.getState().nav, controls: window.__zoomRender.getState().controls, generation: window.__worldSim.generation().count };
  });
  await page.getByTitle('World Settings', { exact: true }).click();
  await page.getByText('World workbench', { exact: true }).click();
  await page.getByRole('button', { name: 'camera', exact: true }).click();
  const checkbox = page.getByLabel('dollyToCursor', { exact: true });
  for (const [source, cursor] of [['workbench', false], ['operator', true], ['operator', false], ['workbench', true]]) {
    if (source === 'workbench') await checkbox.setChecked(cursor);
    else await page.getByRole('radio', { name: cursor ? 'Cursor' : 'Center', exact: true }).check();
    await page.waitForFunction(cursor => window.__zoomRender.getState().controls.dollyToCursor === cursor, cursor);
    check(`${source} ${cursor ? 'cursor' : 'center'} agrees across both controls and renderer`,
      await checkbox.isChecked() === cursor && await page.getByRole('radio', { name: cursor ? 'Cursor' : 'Center', exact: true }).isChecked());
  }
  await checkbox.uncheck();
  await page.getByRole('button', { name: 'Reset', exact: true }).click();
  await page.waitForFunction(() => window.__zoomRender.getState().controls.dollyToCursor === true);
  check('reset restores one shared default', await checkbox.isChecked() && await page.getByRole('radio', { name: 'Cursor', exact: true }).isChecked());
  check('zoom edits retain controls navigation and generation', await page.evaluate(() => window.__zoomRender.getState().controls === window.__zoomBefore.controls && window.__zoomStore.getState().nav === window.__zoomBefore.nav && window.__worldSim.generation().count === window.__zoomBefore.generation));
  check('no browser errors', errors.length === 0);
} catch (error) { checks.push({ name: 'completed', pass: false, detail: String(error) }); }
finally {
  writeFileSync(new URL('../../evidence/living-world/camera-zoom-authority-browser-20260905.json', import.meta.url), JSON.stringify({ checks, errors }, null, 2));
  console.log(checks);
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
