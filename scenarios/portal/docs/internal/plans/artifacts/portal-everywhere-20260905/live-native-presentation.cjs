const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');
const { chromium } = createRequire(path.join(process.cwd(), 'scenarios/browser-automation-studio/playwright-driver/package.json'))('rebrowser-playwright');
const artifact = path.join(__dirname, 'native-companion-live.json');
(async () => {
 const evidence = JSON.parse(fs.readFileSync(artifact, 'utf8'));
 const target = evidence.launch.target;
 const browser = await chromium.connectOverCDP(target.cdp_endpoint);
 const checks = [];
 try {
  const pages = browser.contexts().flatMap(c => c.pages()).filter(p => p.url().startsWith('http://127.0.0.1:24965/'));
  if (pages.length !== 1) throw new Error('renderer selection is not unique');
  const page = pages[0]; page.setDefaultTimeout(8000);
  const geometry = async () => {
   const response = await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+evidence.session_id+'/control', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'window_geometry'})});
   const result = await response.json();
   if (!response.ok || result.status !== 'ok') throw new Error('native geometry failed');
   return {bounds:result.data};
  };
  const initial = await geometry();
  const draft = page.getByTestId('chat-composer-input');
  await draft.fill('Native presentation continuity check');
  const draftNode = await draft.elementHandle();
  const surface = page.locator('#surface-selection');
  const desktop = surface.locator('option').filter({ hasText: 'Desktop session 2' });
  if (await desktop.count() === 1) await surface.selectOption(await desktop.getAttribute('value'));
  const selection = await surface.inputValue();
  const surfaceNode = await surface.elementHandle();
  for (const mode of ['palette','pill','expanded']) {
   await page.locator('#companion-presentation').selectOption(mode);
   await page.waitForFunction(mode => document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode') === mode, mode);
   const current = await geometry();
   const sameDraft = await draftNode.evaluate(node => node === document.querySelector('[data-testid="chat-composer-input"]'));
   const sameSurface = await surfaceNode.evaluate(node => node === document.querySelector('#surface-selection'));
   if (!sameDraft || !sameSurface || await draft.inputValue() !== 'Native presentation continuity check' || await surface.inputValue() !== selection) throw new Error('workspace continuity failed');
   const expected = mode === 'pill' ? [320,96] : mode === 'palette' ? [560,480] : [initial.bounds.width,initial.bounds.height];
   if (current.bounds.width !== expected[0] || current.bounds.height !== expected[1]) throw new Error('native bounds mismatch: '+JSON.stringify({mode,current,expected}));
   checks.push({mode,renderer_id:target.renderer_id,bounds:current.bounds,same_draft_node:sameDraft,same_surface_node:sameSurface,draft_preserved:true,target_preserved:true});
  }
  await draft.fill('');
  evidence.presentation = { status:'passed', observed_at:new Date().toISOString(), renderer_id:target.renderer_id, transitions:checks, scope:'Real Electron managed X11 window, same mounted UI and unsent draft/selected target. No active-task, shortcut, attachment, restart or routed data-isolation claim.' };
 } catch (error) { evidence.presentation = {status:'failed',error:error.message,transitions:checks}; throw error; }
 finally { fs.writeFileSync(artifact,JSON.stringify(evidence,null,2)+'\n'); await browser.close(); }
})().catch(error=>{console.error(error.message);process.exitCode=1;});
