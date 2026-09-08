// Real Electron/native keys with intercepted task RPCs; no real remote task claim.
const fs=require('node:fs');const {createRequire}=require('node:module');
const {chromium}=createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const filename=__dirname+'/native-quit-fetch-live.json';
(async()=>{const data=JSON.parse(fs.readFileSync(filename));const browser=await chromium.connectOverCDP(data.launch.target.cdp_endpoint);const context=browser.contexts()[0];const page=context.pages().find(p=>p.url().startsWith('http://127.0.0.1:24965/'));if(!page)throw Error('portal renderer not found');const cdp=await context.newCDPSession(page);let stopped=false,stopCalls=0,reads=0;const requests=[];try{
const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));page.setDefaultTimeout(10000);
const control=async(key)=>{const r=await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'key_press',params:{key}})});if(!r.ok)throw Error('native key failed');};
// Prove the document fixture is active before any submission.
page.on('console',message=>{if(message.text().startsWith('QUIT_FIXTURE_STATE ')){const state=JSON.parse(message.text().slice(19));stopCalls=state.stopCalls;reads=state.reads;requests.splice(0,requests.length,...state.requests);}});
await page.addInitScript(()=>{
 const original=window.fetch.bind(window);let stopped=false;const state={stopCalls:0,reads:0,requests:[]};window.__quitFixture=state;
 window.fetch=async(input,init)=>{
  const url=typeof input==='string'?input:input instanceof URL?input.href:input.url;
  if(url.endsWith('/quit-fixture-preflight'))return new Response('intercepted');
  if(!/\/vrooli\.portal\.v1\.(chat|message)\./.test(url))return original(input,init);
  const method=url.split('/').pop();state.requests.push(method);let body={},status=200;
  switch(method){
   case 'ListAgentAdmissions':body={admissions:[]};break;
   case 'ListChats':body={chats:[{id:'quit-fixture-chat',title:'Quit fixture',mode:'CHAT_MODE_AGENT',model:'fixture'}],groups:[]};break;
   case 'GetTree':body={messages:[]};break;
   case 'SendMessage':body={userMessage:{id:'quit-fixture-message',chatId:'quit-fixture-chat',role:'MESSAGE_ROLE_USER',content:'Controlled quit fixture'}};break;
   case 'StreamCompletion':status=503;body={code:'unavailable',message:'Controlled disconnected stream'};break;
   case 'GetAgentRun':state.reads++;body={runId:'fixture-run',status:stopped?'cancelled':'running',terminal:stopped};break;
   case 'StopAgentRun':state.stopCalls++;stopped=true;status=503;body={code:'unavailable',message:'Controlled lost Stop reply'};break;
   default:throw Error('Unexpected fixture task RPC '+method);
  }
  if(method==='GetAgentRun'&&stopped)await new Promise(resolve=>{window.__releaseQuitRead=resolve;});
  return new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
 };
});
await page.reload();await page.waitForFunction(()=>window.__quitFixture?.requests.includes('ListChats'));if(await page.evaluate(async()=>await(await fetch('/quit-fixture-preflight')).text())!=='intercepted')throw Error('Fixture interception missing');await page.getByRole('button',{name:'Quit fixture',exact:true}).waitFor();await page.getByTestId('chat-mode-select').selectOption('2');await page.getByTestId('chat-composer-input').fill('Controlled quit fixture');await page.getByTestId('chat-send-button').click();
await page.waitForFunction(()=>document.querySelector('[data-testid="chat-stop-button"]')&&!document.querySelector('[data-testid="chat-stop-button"]').disabled);
// Wait for the fixture's initial authoritative read, without triggering another task.
await page.waitForFunction(()=>window.__quitFixture?.reads>=1);
await page.waitForFunction(()=>document.querySelector('[data-testid="chat-stop-button"]')&&!document.querySelector('[data-testid="chat-stop-button"]').disabled);
await control('ctrl+q');await page.getByRole('dialog').waitFor();await page.getByRole('button',{name:'Cancel',exact:true}).click();await page.getByRole('dialog').waitFor({state:'hidden'});
await control('ctrl+q');await page.getByRole('button',{name:'Keep running in background',exact:true}).click();await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='hidden');
await control('ctrl+shift+space');await page.waitForFunction(()=>document.querySelector('[data-companion-mode]')?.getAttribute('data-companion-mode')==='palette');
await control('ctrl+q');await page.getByRole('button',{name:'Stop tasks and quit',exact:true}).click();await page.getByRole('dialog').getByRole('button',{name:'Check status',exact:true}).waitFor();
if(!fs.existsSync('/proc/'+data.launch.target.process_id))throw Error('quit before task resolution');
let timer;const disconnected=new Promise((resolve,reject)=>{browser.once('disconnected',resolve);timer=setTimeout(()=>reject(Error('quit not completed')),10000);});
await page.getByRole('dialog').getByRole('button',{name:'Check status',exact:true}).click();await page.waitForFunction(()=>typeof window.__releaseQuitRead==='function');const counts=await page.evaluate(()=>window.__quitFixture);stopCalls=counts.stopCalls;reads=counts.reads;requests.push(...counts.requests);await page.evaluate(()=>{setTimeout(()=>window.__releaseQuitRead(),0);});await disconnected;clearTimeout(timer);
if(stopCalls!==1||reads<2)throw Error('wrong task operation counts');
data.quit_fixture={status:'passed',cancel_preserved_process:true,background_recovered:true,uncertain_stop_retained_dialog:true,terminal_read_allowed_native_exit:true,stop_calls:stopCalls,read_calls:reads,observed_at:new Date().toISOString(),scope:'Real Electron native quit/shortcut with simulated task RPC responses; no real agent stopped.'};console.log(JSON.stringify(data.quit_fixture));
}catch(error){data.quit_fixture={status:'failed',error:error.message,stop_calls:stopCalls,read_calls:reads,requests};throw error;}finally{fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');await browser.close();}})().catch(e=>{console.error(e.message);process.exitCode=1;});
