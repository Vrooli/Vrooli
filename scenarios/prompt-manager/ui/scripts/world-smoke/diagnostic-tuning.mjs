// Normal workbench edits retain GPU collector ownership while configuring its limits.
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
  await page.evaluate(()=>{
    window.__readGpuTimer=()=>{
      const pending=[...window.__cameraFixtureRoots].map(r=>r.current);
      while(pending.length){const f=pending.pop();let hook=f.memoizedState;while(hook&&typeof hook==='object'){
        const value=hook.memoizedState?.current;
        if(value&&typeof value.stats==='function'&&typeof value.drain==='function'&&typeof value.begin==='function'&&typeof value.beginFrame!=='function')return value;
        hook=hook.next;
      }if(f.child)pending.push(f.child);if(f.sibling)pending.push(f.sibling)}return null;
    };window.__gpuBefore=window.__readGpuTimer();
  });
  check('mounted GPU collector found',await page.evaluate(()=>Boolean(window.__gpuBefore)));
  for (const [field,value] of [['diagnostics.publishEveryFrames',2],['diagnostics.overlayRefreshMs',700],['diagnostics.gpuSampleWindow',2],['diagnostics.gpuMaxInFlight',1]]) {
    await page.getByLabel(field,{exact:true}).fill(String(value));await page.keyboard.press('Tab');await page.waitForTimeout(200);
    check(field+' retains the collector',await page.evaluate(()=>window.__readGpuTimer()===window.__gpuBefore));
  }
  check('new sample window is respected',await page.evaluate(()=>window.__readGpuTimer().stats().samples<=2));
  check('diagnostic edits retain controls navigation and generation',await page.evaluate(()=>window.__zoomRender.getState().controls===window.__zoomBefore.controls&&window.__zoomStore.getState().nav===window.__zoomBefore.nav&&window.__worldSim.generation().count===window.__zoomBefore.generation));
  check('no browser errors', errors.length === 0);
} catch (error) { checks.push({ name: 'completed', pass: false, detail: String(error) }); }
finally {
  writeFileSync(new URL('../../evidence/living-world/diagnostic-tuning-browser-20260905.json', import.meta.url), JSON.stringify({ checks, errors }, null, 2));
  console.log(checks);
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
