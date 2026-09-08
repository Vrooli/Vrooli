// Real browser UI -> Portal -> signed native helper -> disposable physical GTK.
// Retains metadata only; credentials, desktop pixels, and fixture text are temporary.
const fs = require('node:fs');
const path = require('node:path');
const {createRequire} = require('node:module');
const {spawn} = require('node:child_process');
const readline = require('node:readline');
const root = path.resolve(__dirname, '../../../../../../..');
const {chromium} = createRequire(path.join(root,'scenarios/browser-automation-studio/playwright-driver/package.json'))('rebrowser-playwright');
(async()=>{
 const base='/run/user/1000/vrooli-desktop-owner';
 const identity=JSON.parse(fs.readFileSync(path.join(base,'portal-test-account.json'),'utf8'));
 const config=JSON.parse(fs.readFileSync(path.join(base,'owner.json'),'utf8'));
 const work=fs.mkdtempSync(path.join(base,'portal-unicode-'));
 const resultPath=path.join(work,'result');
 const child=spawn('/usr/bin/python3',[path.join(root,'scenarios/device-control/api/internal/native/atspi/testdata/gtk_fixture.py'),resultPath],{env:{...process.env,DISPLAY:':'+config.helper.display,XAUTHORITY:config.helper.xauthority_file,DBUS_SESSION_BUS_ADDRESS:'unix:path=/run/user/1000/bus',XDG_RUNTIME_DIR:'/run/user/1000',GSETTINGS_BACKEND:'memory',GIO_USE_VFS:'local'},stdio:['ignore','pipe','ignore']});
 let browser,page,opened,stage='fixture'; const observations=[];
 try {
  const ready=await new Promise((resolve,reject)=>{
   const lines=readline.createInterface({input:child.stdout});
   const timer=setTimeout(()=>{lines.close();reject(new Error('fixture timeout'));},5000);
   lines.once('line',line=>{clearTimeout(timer);lines.close();try{resolve(JSON.parse(line));}catch{reject(new Error('fixture metadata'));}});
   child.once('error',()=>{clearTimeout(timer);reject(new Error('fixture spawn'));});
  });
  if(ready.pid!==child.pid)throw new Error('fixture PID mismatch');
  browser=await chromium.launch({headless:true});page=await browser.newPage({viewport:{width:1440,height:1000}});page.setDefaultTimeout(10000);
  page.on('response',async response=>{if(response.url().endsWith('DesktopSessionService/Open')&&response.ok()){try{opened=(await response.json()).session;}catch{}}});
  page.on('response',async response=>{if(response.url().endsWith('DesktopSessionService/Observe')){try{const data=await response.json();observations.push({status:response.status(),semantic_process:data.semantic?.processId,expires:data.semantic?.expiresAt,elements:data.semantic?.elements?.map(e=>({name:e.name,editable:e.editable}))});}catch{observations.push({status:response.status(),json:false});}}});
  stage='desktop selection';await page.goto('http://127.0.0.1:24965');
  const option=page.locator('#surface-selection option').filter({hasText:'Desktop session 2'});await option.waitFor({state:'attached'});await page.selectOption('#surface-selection',await option.getAttribute('value'));
  stage='login';await page.getByLabel('Email',{exact:true}).fill(identity.email);await page.getByLabel('Password',{exact:true}).fill(identity.password);await page.getByRole('button',{name:'Sign in',exact:true}).click();
  stage='control';await page.getByRole('button',{name:'Request control',exact:true}).click();await page.locator('section img[src^="blob:"]').waitFor();
  stage='prepare';await page.getByText('Application fields (advanced)',{exact:true}).click();
  const text='日本語 العربية café e\u0301 🧪';
  await page.getByLabel('Application process ID',{exact:true}).fill(String(child.pid));
  await page.getByLabel('Text to insert — prepare before inspecting',{exact:true}).fill(text);
  await page.getByRole('button',{name:'Inspect application',exact:true}).click();
  stage='field selection';await page.getByLabel(/^Editable field/).selectOption({label:'go-unicode-entry · 1'});
  await page.getByLabel('Character position (0 is the start)',{exact:true}).fill('2');
  stage='insert';await page.getByRole('button',{name:'Insert text',exact:true}).click();
  await page.waitForFunction(()=>document.querySelector('textarea')?.value==='');
  if(fs.readFileSync(resultPath,'utf8')!=='é|'+text+'tail')throw new Error('persisted text mismatch');
  const storage=await page.evaluate(()=>[...Object.values(localStorage),...Object.values(sessionStorage)]);
  if(storage.some(v=>v.includes(identity.password)||v.includes(text)||v.includes('accessToken')||v.includes('eyJ')))throw new Error('sensitive persistence');
  stage='stop';await page.getByRole('button',{name:'Stop session',exact:true}).click();await page.locator('section img[src^="blob:"]').waitFor({state:'detached'});
  stage='logout';await page.getByRole('button',{name:'Sign out',exact:true}).click();await page.getByRole('button',{name:'Sign in',exact:true}).waitFor();
  const evidence={timestamp:new Date().toISOString(),browser:'installed Chromium',physical_desktop_session:config.helper.session_id,fixture_pid:child.pid,login:true,control_open:true,semantic_inspection:true,exact_opaque_field_selection:true,unicode_persisted_exactly:true,no_sensitive_browser_storage:true,stop:true,logout:true};
  fs.writeFileSync(path.join(__dirname,'live-portal-unicode.json'),JSON.stringify(evidence,null,2)+'\n');console.log(JSON.stringify(evidence));
 }catch(error){console.error(JSON.stringify({stage,observations,selector:page?await page.getByLabel(/^Editable field/).evaluate(e=>({disabled:e.disabled,options:Array.from(e.options).map(o=>o.textContent),now:new Date().toISOString()})).catch(()=>null):null}));throw new Error(stage+': '+error.name);}
 finally {
  if(page){try{const stop=page.getByRole('button',{name:'Stop session',exact:true});if(await stop.count())await stop.click({timeout:3000});}catch{}}
  if(opened){try{await fetch('http://127.0.0.1:17476/vrooli.portal.v1.surfaces.DesktopSessionService/Stop',{method:'POST',headers:{'Content-Type':'application/json','Connect-Protocol-Version':'1',Authorization:'Bearer '+identity.account.tokens.accessToken},body:JSON.stringify({session:opened}),signal:AbortSignal.timeout(5000)});}catch{}}
  if(browser)await browser.close();
  child.kill('SIGTERM');await new Promise(resolve=>{if(child.exitCode!==null)return resolve();child.once('exit',resolve);setTimeout(()=>{child.kill('SIGKILL');resolve();},2000).unref();});
  fs.rmSync(work,{recursive:true,force:true});
 }
})().catch(error=>{console.error(error.message);process.exitCode=1;});
