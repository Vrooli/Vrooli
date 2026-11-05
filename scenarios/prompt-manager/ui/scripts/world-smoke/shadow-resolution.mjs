// Normal quality edits resize actual GPU shadow targets and dispose obsolete targets.
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
  await page.getByRole('button', { name: 'quality', exact: true }).click();
  await page.evaluate(() => { window.__readKeyLight=()=>{let light;window.__zoomRender.getState().scene.traverse(o=>{if(o.isDirectionalLight&&o.castShadow)light=o});return light};window.__initialKeyLight=window.__readKeyLight(); });
  await page.waitForFunction(()=>window.__readKeyLight()?.shadow.map?.width>0);
  for (const resolution of [1024,512]) {
    await page.evaluate(()=>{window.__oldShadowTarget=window.__readKeyLight().shadow.map;window.__shadowDisposals=0;window.__oldShadowTarget.addEventListener('dispose',()=>{window.__shadowDisposals++})});
    await page.getByLabel('profiles.high.shadowMapSize',{exact:true}).fill(String(resolution));
    await page.keyboard.press('Tab');
    await page.waitForFunction(resolution=>{const map=window.__readKeyLight().shadow.map;return map&&map!==window.__oldShadowTarget&&map.width===resolution&&map.height===resolution},resolution);
    check('allocates actual '+resolution+' square shadow target',await page.evaluate(()=>window.__readKeyLight()===window.__initialKeyLight));
    check('disposes previous target exactly once at '+resolution,await page.evaluate(()=>window.__shadowDisposals===1));
  }
  check('shadow resolution retains controls navigation and generation', await page.evaluate(() => window.__zoomRender.getState().controls === window.__zoomBefore.controls && window.__zoomStore.getState().nav === window.__zoomBefore.nav && window.__worldSim.generation().count === window.__zoomBefore.generation));
  check('no browser errors', errors.length === 0);
} catch (error) { checks.push({ name: 'completed', pass: false, detail: String(error) }); }
finally {
  writeFileSync(new URL('../../evidence/living-world/shadow-resolution-browser-20260905.json', import.meta.url), JSON.stringify({ checks, errors }, null, 2));
  console.log(checks);
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
