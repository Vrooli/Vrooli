const fs=require('node:fs');
const {execFileSync,spawn}=require('node:child_process');
const {createRequire}=require('node:module');
const {chromium}=createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const run=(...args)=>execFileSync('xdotool',args,{encoding:'utf8',timeout:5000}).trim();
(async()=>{
 const [endpoint,title,pid]=process.argv.slice(2);
 const browser=await chromium.connectOverCDP(endpoint);
 const imageMode=process.env.VROOLI_PORTAL_CONTEXT_IMAGE==='1';let backing;

 const receipt={observed_at:new Date().toISOString(),scope:'Selected desktop, managed Electron target/profile and owned fixture window; no screenshot, text selection or chat submission',validation_only_unsandboxed:process.platform==='linux'};
 if(imageMode)receipt.scope='Selected desktop, managed Electron and owned xmessage image preview; no chat submission or saved image bytes';
 try {
  const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));
  if(!page)throw Error('Portal renderer missing');page.setDefaultTimeout(10000);
  const cdp=await page.context().newCDPSession(page);
  const snapshot=async()=>{const r=await cdp.send('Runtime.evaluate',{expression:'window.desktopPresentation.get()',awaitPromise:true,returnByValue:true});if(r.exceptionDetails)throw Error('native snapshot failed');return r.result.value;};
  const select=page.locator('#companion-presentation');await select.selectOption('expanded');
  await page.locator('#companion-shortcut').fill('Control+Alt+P');await page.getByRole('button',{name:'Apply shortcut',exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('#companion-presentation')?.querySelector('option[value=hidden]'));
  const configured=await snapshot();if(configured.shortcut?.status!=='registered'||configured.shortcut.accelerator!=='Control+Alt+P')throw Error('test shortcut was not registered');
  const draft=page.getByTestId('chat-composer-input');await draft.fill('Context activation continuity');const node=await draft.elementHandle();
  const windows=run('search','--sync','--name','^'+title+'$').split('\n').filter(id=>execFileSync('xprop',['-id',id,'WM_STATE'],{encoding:'utf8',timeout:5000}).includes('window state:'));if(windows.length!==1)throw Error('ambiguous fixture window');
  const fixture=windows[0];execFileSync('xprop',['-id',fixture,'-f','_NET_WM_PID','32c','-set','_NET_WM_PID',pid],{timeout:5000});
  if(imageMode){
   backing=spawn('python3',[__dirname+'/live-native-session.py','--fixture-backing',fixture],{stdio:['pipe','pipe','pipe']});
   await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('fixture backing timeout')),5000);backing.once('error',reject);backing.once('exit',()=>{clearTimeout(timer);reject(Error('fixture backing exited'));});backing.stdout.once('data',chunk=>{clearTimeout(timer);if(chunk.toString().trim()!=='ready')reject(Error('fixture backing not ready'));else resolve();});});
   receipt.fixture_backing_redirected=true;
  }
  await select.selectOption('hidden');run('windowactivate','--sync',fixture);
  if(run('getactivewindow')!==fixture)throw Error('fixture did not own focus before activation');
  receipt.context_lock=await page.evaluate(async()=>{
   if(!navigator.locks)throw Error('context coordination unavailable');
   return navigator.locks.request('portal.context-retention.v1',{ifAvailable:true},async lock=>{
    if(!lock)throw Error('context coordination already busy');
    const refused=await navigator.locks.request('portal.context-retention.v1',{ifAvailable:true},other=>other===null);
    if(!refused)throw Error('context coordination admitted overlapping owner');
    return {available:true,overlap_refused:true};
   });
  });
  receipt.prior_app_focused=true;receipt.before=await snapshot();run('key','--clearmodifiers','ctrl+alt+p');
  await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='palette');
  receipt.after=await snapshot();if(receipt.after.activation?.status!=='ready')throw Error('context was not ready: '+JSON.stringify(receipt.after.activation));
  if(imageMode){
   if(receipt.after.activation.hasImage!==true)throw Error('captured image missing');
   await page.getByRole('button',{name:'Preview captured window',exact:true}).click();
   const img=page.getByRole('img',{name:'Captured source window',exact:true});await img.waitFor();
   await page.waitForFunction(()=>{const image=document.querySelector('img[alt="Captured source window"]');return image?.complete&&image.naturalWidth>0;});
   receipt.preview=await img.evaluate(image=>({width:image.naturalWidth,height:image.naturalHeight}));
   const geometry=Object.fromEntries(run('getwindowgeometry','--shell',fixture).split('\n').map(line=>line.split('=')));
   if(receipt.preview.width!==Number(geometry.WIDTH)||receipt.preview.height!==Number(geometry.HEIGHT))throw Error('preview dimensions differ from source client');
   receipt.provenance=await page.evaluate(async()=>{
    const image=await window.desktopPresentation.readContextImage();
    const source=image.source;
    if(!source || source.captureId!==image.contextId || JSON.stringify(source.bounds)!==JSON.stringify(image.sourceBounds) ||
       !source.displayId || !source.geometryRevision || !Number.isFinite(Date.parse(source.capturedAt)) ||
       source.surface?.ownerScenario!=='device-control' || !source.surface.target?.resourceId)throw Error('frozen source provenance missing');
    if(Object.keys(source).sort().join(',')!=='bounds,captureId,capturedAt,displayId,geometryRevision,surface')throw Error('unexpected source fields');
    return {capture_matches:true,bounds_match:true,surface_present:true,display_geometry_present:true,captured_time_present:true};
   });
   if(!await page.getByText('Local preview only. This image has not been attached or sent.',{exact:true}).isVisible())throw Error('local preview notice missing');
   await select.selectOption('expanded');
   await page.getByLabel('Width',{exact:true}).fill(String(Math.floor(receipt.preview.width/2)));
   await page.getByLabel('Height',{exact:true}).fill(String(Math.floor(receipt.preview.height/2)));
   await page.getByLabel('X',{exact:true}).fill('5');await page.getByLabel('Y',{exact:true}).fill('5');
   await page.getByRole('button',{name:'Draw annotation',exact:true}).click();
   const overlay=page.getByTestId('context-image-overlay');await overlay.scrollIntoViewIfNeeded();
   const box=await overlay.boundingBox();if(!box)throw Error('annotation overlay not rendered');
   await page.mouse.move(box.x+box.width*0.25,box.y+box.height*0.25);await page.mouse.down();
   await page.mouse.move(box.x+box.width*0.5,box.y+box.height*0.5,{steps:4});await page.mouse.up();
   const mark=page.getByTestId('context-image-mark');await mark.waitFor();
   const points=await mark.getAttribute('points');if(!points||points.split(' ').length<2)throw Error('annotation missing source points');
   await select.selectOption('pill');await select.selectOption('expanded');
   if(await mark.getAttribute('points')!==points)throw Error('compact transition changed source annotation');
   const region=page.getByTestId('context-image-region');
   if(await region.getAttribute('x')!=='5'||await region.getAttribute('width')!==String(Math.floor(receipt.preview.width/2)))throw Error('compact transition lost source region');
   receipt.annotation={source_points_preserved:true,region_preserved:true,stroke_count:await mark.count()};

  }
  await page.getByRole('button',{name:'Dismiss local app reference',exact:true}).click();
  await page.getByRole('button',{name:'Dismiss local app reference',exact:true}).waitFor({state:'hidden'});
  await page.waitForFunction(()=>document.querySelector('#companion-presentation')?.getAttribute('aria-busy')==='false');
  if(await page.locator('.companion-toolbar [role=alert]').count())throw Error('native cleanup not confirmed');
  if(imageMode){if(await page.getByRole('img',{name:'Captured source window',exact:true}).count())throw Error('dismissed pixels remain');receipt.preview_removed=true;}
  receipt.cleanup_confirmed_by_ui=true;receipt.dismissed=await snapshot();if(receipt.dismissed.activation?.status!=='unavailable')throw Error('dismissal unconfirmed');
  if(!await node.evaluate(n=>n===document.querySelector('[data-testid="chat-composer-input"]'))||await draft.inputValue()!=='Context activation continuity')throw Error('draft continuity lost');
  receipt.status='passed';receipt.same_draft_node=true;await draft.fill('');
 } catch(error){receipt.status='failed';receipt.error=error.message;throw error;}
 finally {if(backing){backing.stdin.end();await new Promise(resolve=>{const timer=setTimeout(()=>{backing.kill();resolve();},2000);backing.once('exit',()=>{clearTimeout(timer);resolve();});});}fs.writeFileSync(__dirname+(imageMode?'/native-context-image-selected-desktop-live.json':'/native-context-selected-desktop-live.json'),JSON.stringify(receipt,null,2)+'\n');await browser.close();console.log(JSON.stringify(receipt));}
})().catch(error=>{console.error(error.message);process.exitCode=1;});
