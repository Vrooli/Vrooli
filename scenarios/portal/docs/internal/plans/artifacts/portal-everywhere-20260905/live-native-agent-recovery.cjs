// Native shell reload/Quit proof. All task RPCs are replaced before interaction;
// fixture identities must be observed before any Quit or Stop is requested.
const fs = require('node:fs');
const {createRequire} = require('node:module');
const {chromium} = createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const filename=__dirname+'/native-agent-recovery-live.json';
(async()=>{
 const data=JSON.parse(fs.readFileSync(filename));
 const browser=await chromium.connectOverCDP(data.launch.target.cdp_endpoint);
 try {
  const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));
  page.setDefaultTimeout(10000);
  await page.addInitScript(()=>{
   const state=JSON.parse(sessionStorage.getItem('native-agent-recovery-fixture')||'{"stopped":false,"stops":0,"reads":0}');
   window.__recoveryFixture=state;
   const persist=()=>sessionStorage.setItem('native-agent-recovery-fixture',JSON.stringify(state));
   const original=window.fetch.bind(window);
   window.fetch=async(input,init)=>{
    const url=typeof input==='string'?input:input instanceof URL?input.href:input.url;
    if(url.endsWith('/recovery-fixture-preflight'))return new Response('intercepted');
    if(!/\/vrooli\.portal\.v1\.(chat|message)\./.test(url))return original(input,init);
    const method=url.split('/').pop();let body={},status=200;
    switch(method){
     case 'ListChats':body={chats:[{id:'recovery-fixture-chat',title:'Recovery fixture',mode:'CHAT_MODE_AGENT'}],groups:[]};break;
     case 'GetTree':body={messages:[]};break;
     case 'ListAgentAdmissions':
      await new Promise(resolve=>{window.__releaseAdmission=resolve;});
      body={admissions:[{chatId:'recovery-fixture-chat',messageId:'recovery-fixture-message'}]};break;
     case 'GetAgentRun':
      state.reads++;persist();
      if(state.stopped)await new Promise(resolve=>{window.__releaseTerminal=resolve;});
      body={runId:'fixture-run',status:state.stopped?'cancelled':'running',terminal:state.stopped};break;
     case 'StopAgentRun':state.stops++;state.stopped=true;persist();status=503;body={code:'unavailable',message:'Fixture lost Stop reply'};break;
     default:throw Error('Unpermitted fixture RPC '+method);
    }
    return new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
   };
  });
  await page.reload();
  await page.waitForFunction(()=>typeof window.__releaseAdmission==='function');
  if(await page.evaluate(async()=>await(await fetch('/recovery-fixture-preflight')).text())!=='intercepted')throw Error('Missing fixture interception');
  const control=async key=>{const r=await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'key_press',params:{key}})});if(!r.ok)throw Error('Native key failed');};
  await control('ctrl+q');await page.getByRole('dialog').waitFor();
  if(!fs.existsSync('/proc/'+data.launch.target.process_id))throw Error('Quit bypassed recovery');
  await page.evaluate(()=>window.__releaseAdmission());
  await page.getByRole('dialog').getByRole('button',{name:'Stop chat',exact:true}).waitFor();
  await page.getByRole('dialog').getByRole('button',{name:'Stop tasks and quit',exact:true}).click();
  await page.getByRole('dialog').getByRole('button',{name:'Check status',exact:true}).waitFor();
  // Reload after a lost Stop reply, while native still owns the same Quit token.
  await page.reload();await page.waitForFunction(()=>typeof window.__releaseAdmission==='function');
  await page.getByRole('dialog').waitFor();
  if(!fs.existsSync('/proc/'+data.launch.target.process_id))throw Error('Reload lost quit guard');
  await page.evaluate(()=>window.__releaseAdmission());
  await page.waitForFunction(()=>typeof window.__releaseTerminal==='function');
  const state=await page.evaluate(()=>window.__recoveryFixture);
  if(state.stops!==1||state.reads!==2)throw Error('Recovery repeated Stop or missed readback');
  let timer;const closed=new Promise((resolve,reject)=>{browser.once('disconnected',resolve);timer=setTimeout(()=>reject(Error('Terminal read did not permit exit')),10000);});
  await page.evaluate(()=>{setTimeout(()=>window.__releaseTerminal(),0);});await closed;clearTimeout(timer);
  data.agent_recovery={status:'passed',quit_waited_for_enumeration:true,pending_quit_survived_reload:true,lost_stop_not_repeated:true,terminal_read_allowed_exit:true,stop_calls:state.stops,read_calls:state.reads,observed_at:new Date().toISOString(),scope:'Real native shell and renderer reload with simulated task RPCs; no real task mutations.'};
  console.log(JSON.stringify(data.agent_recovery));
 }catch(error){data.agent_recovery={status:'failed',error:error.message};throw error;}
 finally{fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');await browser.close();}
})().catch(error=>{console.error(error.message);process.exitCode=1;});
