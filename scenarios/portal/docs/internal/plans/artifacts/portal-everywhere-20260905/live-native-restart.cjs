const fs=require('node:fs');const path=require('node:path');const {createRequire}=require('node:module');
const {chromium}=createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const filename=__dirname+'/native-companion-live.json';
(async()=>{const data=JSON.parse(fs.readFileSync(filename));let browser;try{
if(data.close_persistence?.status!=='passed')throw Error('native close evidence required');
const base='http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id;
const response=await fetch(base+'/restart-electron-validation',{method:'POST'});const result=await response.json();if(!response.ok)throw Error(JSON.stringify(result));const target=result.target;
if(target.process_id===data.launch.target.process_id||target.renderer_id===data.launch.target.renderer_id)throw Error('process/renderer did not change');
const cmd=fs.readFileSync('/proc/'+target.process_id+'/cmdline','utf8');const profile=cmd.match(/--user-data-dir=(\/tmp\/vrooli-electron-validation-[a-zA-Z0-9-]+)/)?.[1];if(!profile||path.basename(profile)!==data.close_persistence.profile_basename)throw Error('profile changed');
browser=await chromium.connectOverCDP(target.cdp_endpoint);const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));
await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='expanded');
const r=await fetch(base+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'window_geometry'})});const geometry=(await r.json()).data;const saved=data.close_persistence.saved;if(geometry.width!==saved.width||geometry.height!==saved.height)throw Error('restored bounds mismatch: '+JSON.stringify(geometry));
data.restart={status:'passed',target,same_profile:true,geometry,mode:'expanded',observed_at:new Date().toISOString(),scope:'Real fresh Electron process and renderer restore expanded bounds from the retained owner profile after native quit from pill. No conversation or active-task restart claim.'};console.log(JSON.stringify(data.restart));
}catch(error){data.restart={status:'failed',error:error.message};throw error;}finally{fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');if(browser)await browser.close();}})().catch(e=>{console.error(e.message);process.exitCode=1;});
