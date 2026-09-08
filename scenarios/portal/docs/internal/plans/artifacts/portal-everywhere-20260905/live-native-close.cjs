const fs=require('node:fs');const path=require('node:path');const {createRequire}=require('node:module');
const {chromium}=createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const filename=__dirname+'/native-companion-live.json';
(async()=>{const data=JSON.parse(fs.readFileSync(filename));const browser=await chromium.connectOverCDP(data.launch.target.cdp_endpoint);try{
const args=fs.readFileSync('/proc/'+data.launch.target.process_id+'/cmdline','utf8').split('\0');const profile=args.join(' ').match(/--user-data-dir=(\/tmp\/vrooli-electron-validation-[a-zA-Z0-9-]+)/)?.[1];if(!profile?.startsWith('/tmp/'))throw Error('unexpected validation profile');
const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));const control=async(action,params)=>{const r=await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action,params})});if(!r.ok)throw Error('native control failed');return r.json();};
const before=(await control('window_geometry')).data;
await page.locator('#companion-presentation').selectOption('pill');await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='pill');
await control('window_activate');await control('key_press',{key:'ctrl+q'});
const stateFile=path.join(profile,'window-state.json');const deadline=Date.now()+5000;let saved;while(Date.now()<deadline){try{saved=JSON.parse(fs.readFileSync(stateFile,'utf8'));break;}catch{await new Promise(resolve=>setTimeout(resolve,50));}}
if(!saved||saved.width!==before.width||saved.height!==before.height)throw Error('close did not persist expanded bounds: '+JSON.stringify(saved));
data.close_persistence={status:'passed',profile_basename:path.basename(profile),before,saved,observed_at:new Date().toISOString(),scope:'Real native quit from pill writes expanded geometry; fresh-process restoration remains separate evidence.'};console.log(JSON.stringify(data.close_persistence));
}catch(error){data.close_persistence={status:'failed',error:error.message};throw error;}finally{fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');await browser.close();}})().catch(e=>{console.error(e.message);process.exitCode=1;});
