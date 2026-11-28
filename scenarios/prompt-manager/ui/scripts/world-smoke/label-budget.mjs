// Normal workbench edits enforce zero and positive label pool budgets.
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
  await page.getByRole('button', { name: 'labels', exact: true }).click();
  const count = () => page.evaluate(() => window.__zoomRender.getState().scene.getObjectByName('labels').children.length);
  check('default label pool exists', await count() > 0);
  const edit = async (field,value) => { await page.getByLabel(field,{exact:true}).fill(String(value)); await page.keyboard.press('Tab'); };
  await edit('budget',0);
  await page.waitForFunction(()=>window.__zoomRender.getState().scene.getObjectByName('labels').children.length===0);
  check('zero label budget removes all text meshes', await count()===0);
  await edit('budget',3);
  await page.waitForFunction(()=>window.__zoomRender.getState().scene.getObjectByName('labels').children.length===3);
  check('positive label budget restores bounded text pool',await count()===3);
  await page.getByRole('button',{name:'quality',exact:true}).click();
  await edit('profiles.high.labelBudget',0);
  await page.waitForFunction(()=>window.__zoomRender.getState().scene.getObjectByName('labels').children.length===0);
  check('zero profile budget also removes all text meshes',await count()===0);
  await edit('profiles.high.labelBudget',2);
  await page.waitForFunction(()=>window.__zoomRender.getState().scene.getObjectByName('labels').children.length===2);
  check('pool respects smaller profile budget',await count()===2);
  check('budget edits retain navigation and generation', await page.evaluate(() => window.__zoomStore.getState().nav === window.__zoomBefore.nav && window.__worldSim.generation().count === window.__zoomBefore.generation));
  check('no browser errors', errors.length === 0);
} catch (error) { checks.push({ name: 'completed', pass: false, detail: String(error) }); }
finally {
  writeFileSync(new URL('../../evidence/living-world/label-budget-browser-20260905.json', import.meta.url), JSON.stringify({ checks, errors }, null, 2));
  console.log(checks);
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
