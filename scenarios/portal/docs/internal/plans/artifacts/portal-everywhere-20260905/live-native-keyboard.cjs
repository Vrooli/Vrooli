const fs=require('node:fs');const {createRequire}=require('node:module');
const {chromium}=createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const filename='scenarios/portal/docs/internal/plans/artifacts/portal-everywhere-20260905/native-companion-live.json';
(async()=>{const data=JSON.parse(fs.readFileSync(filename));const browser=await chromium.connectOverCDP(data.launch.target.cdp_endpoint);try{
const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));await page.reload();const select=page.locator('#companion-presentation');await select.selectOption('expanded');await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='expanded');await select.focus();
const keyboard=[];for(const [key,mode]of [['ArrowDown','palette'],['ArrowDown','pill'],['Home','expanded']]){await page.keyboard.press(key);await page.keyboard.press('Enter');await page.waitForFunction(m=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')===m,mode);keyboard.push({key,mode,confirmed:true});}
data.keyboard={status:'passed',scope:'Mode selection by keyboard after focusing selector',steps:keyboard};
await select.selectOption('pill');await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='pill');
await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
const response=await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'screenshot'})});const capture=await response.json();if(!response.ok)throw new Error('capture failed');data.pill_capture=capture.data;
await select.selectOption('expanded');await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='expanded');
fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');console.log(JSON.stringify({keyboard:data.keyboard,capture:data.pill_capture}));
}finally{await browser.close();}})().catch(e=>{console.error(e.message);process.exitCode=1;});
