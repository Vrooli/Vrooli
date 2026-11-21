// Normal workbench edits update the actual lens and projection matrix without resource replacement.
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
  await page.waitForTimeout(1800);
  await page.evaluate(() => { window.__lensBefore = window.__zoomRender.getState().camera; });
  for (const [field, value] of [['fov',63], ['near',.23], ['far',1800]]) {
    const input=page.getByLabel(field,{exact:true});
    await input.fill(String(value)); await page.keyboard.press('Tab');
    await page.waitForFunction(({field,value}) => window.__zoomRender.getState().camera[field] === value, {field,value});
    check(field+' updates the existing lens',await page.evaluate(()=>window.__zoomRender.getState().camera===window.__lensBefore));
  }
  check('projection matrix reflects edited lens',await page.evaluate(()=>{
    const c=window.__zoomRender.getState().camera,e=c.projectionMatrix.elements;
    return Math.abs(e[5]-1/Math.tan(63*Math.PI/360))<1e-8 && Math.abs(e[14]+2*1800*.23/(1800-.23))<1e-8;
  }));
  check('lens edits retain controls navigation and generation', await page.evaluate(() => window.__zoomRender.getState().controls === window.__zoomBefore.controls && window.__zoomStore.getState().nav === window.__zoomBefore.nav && window.__worldSim.generation().count === window.__zoomBefore.generation));
  check('no browser errors', errors.length === 0);
} catch (error) { checks.push({ name: 'completed', pass: false, detail: String(error) }); }
finally {
  writeFileSync(new URL('../../evidence/living-world/camera-lens-browser-20260905.json', import.meta.url), JSON.stringify({ checks, errors }, null, 2));
  console.log(checks);
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
