// Real native Quit/reload, simulated desktop recovery. Account and
// desktop RPCs are intercepted before any interaction, and interception is proved.
const fs = require('node:fs');
const {createRequire} = require('node:module');
const {chromium} = createRequire(process.cwd()+'/scenarios/browser-automation-studio/playwright-driver/package.json')('rebrowser-playwright');
const unknownOpen=process.argv.includes('--unknown-open');
const filename=__dirname+(unknownOpen?'/native-open-recovery-live.json':'/native-desktop-recovery-live.json');
(async()=>{
 const data=JSON.parse(fs.readFileSync(filename));
 const browser=await chromium.connectOverCDP(data.launch.target.cdp_endpoint);
 try {
  const page=browser.contexts().flatMap(c=>c.pages()).find(p=>p.url().startsWith('http://127.0.0.1:24965/'));
  if(!page)throw Error('Portal renderer missing');
  page.setDefaultTimeout(10000);
  await page.addInitScript(({unknownOpen})=>{
   const session={surface:{ownerScenario:'device-control',surfaceId:'desktop',target:{ownerScenario:'vrooli-bridge',resourceId:'fixture-host',hostNodeId:'fixture-host'}},sessionId:'fixture-desktop-lease',desktopSessionId:'fixture-login'};
   const key='native-desktop-recovery-fixture';
   const state=JSON.parse(sessionStorage.getItem(key)||'{"reads":0,"reconciles":0,"logins":0,"released":false}');
   const persist=()=>sessionStorage.setItem(key,JSON.stringify(state));
   if(!sessionStorage.getItem(key)) {
    if(localStorage.getItem('portal.desktop-recovery.v1'))throw Error('Refusing to overwrite existing desktop recovery');
    localStorage.setItem('portal.desktop-recovery.v1',JSON.stringify({version:2,actor:{id:'fixture-actor',realm:'default'},unknownOpen:false,sessions:unknownOpen?[]:[session],pendingOpen:unknownOpen?{requestId:'50931d3d-4267-4981-a36e-778a8aa9ca9d',surface:session.surface,ttlSeconds:120,control:false}:undefined}));
    persist();
   }
   window.__desktopRecoveryFixture=state;
   const original=window.fetch.bind(window);
   window.fetch=async(input,init)=>{
    const url=typeof input==='string'?input:input instanceof URL?input.href:input.url;
    if(url.endsWith('/desktop-recovery-preflight'))return new Response('intercepted');
    const method=url.split('/').pop(); let body;
    if(url.includes('DesktopSessionService/')) {
     if(method==='ReconcileOpen' && unknownOpen) {
      state.reconciles++;persist();
      if(!state.released)await new Promise(resolve=>{window.__releaseDesktopCleanup=()=>{state.released=true;persist();resolve();};});
      body={requestId:'50931d3d-4267-4981-a36e-778a8aa9ca9d',state:'not_admitted'};
     }else if(method==='ListAdmissions')body={admissions:unknownOpen?[]:[{session,expiresAt:new Date(Date.now()-60000).toISOString(),control:false}]};
     else if(method==='ReadCleanup') {
      state.reads++;persist();
      if(!state.released)await new Promise(resolve=>{window.__releaseDesktopCleanup=()=>{state.released=true;persist();resolve();};});
      body={session,released:true,observedAt:new Date().toISOString()};
     }else throw Error('Unpermitted native fixture desktop RPC '+method);
    }else if(url.includes('OperatorSessionService/')) {
     if(method!=='Login')throw Error('Unpermitted native fixture account RPC '+method);
     state.logins++;persist();
     body={account:{id:'fixture-actor',realm:'default',email:'fixture@example.test'},tokens:{accessToken:'fixture-access',refreshToken:'fixture-refresh',accessTokenExpiresAt:new Date(Date.now()+300000).toISOString()}};
    }else if(/\/vrooli\.portal\.v1\.(chat|message)\./.test(url)) {
     if(method==='ListAgentAdmissions')body={admissions:[]};
     else if(method==='ListChats')body={chats:[],groups:[]};
     else throw Error('Unpermitted native fixture chat RPC '+method);
    }else return original(input,init);
    return new Response(JSON.stringify(body),{headers:{'Content-Type':'application/json'}});
   };
  },{unknownOpen});
  await page.reload();
  await page.getByText('Sign in with the original account to check unresolved desktop sessions.',{exact:true}).waitFor();
  if(await page.evaluate(async()=>await(await fetch('/desktop-recovery-preflight')).text())!=='intercepted')throw Error('Missing interception');
  const control=async()=>{const r=await fetch('http://127.0.0.1:19925/api/v1/livedesktop/sessions/'+data.session_id+'/control',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({action:'key_press',params:{key:'ctrl+q'}})});if(!r.ok)throw Error('Native Quit key failed');};
  const live=()=>{if(!fs.existsSync('/proc/'+data.launch.target.process_id))throw Error('Quit escaped desktop recovery');};
  const cancel=async()=>page.getByRole('dialog').getByRole('button',{name:'Cancel',exact:true}).click();
  const login=async()=>{
   await page.getByLabel('Email',{exact:true}).fill('fixture@example.test');
   await page.getByLabel('Password',{exact:true}).fill('fixture-password');
   await page.getByRole('button',{name:'Sign in',exact:true}).click();
   await page.waitForFunction(()=>typeof window.__releaseDesktopCleanup==='function');
  };
  await control();await page.getByRole('dialog').waitFor();
  await page.getByRole('dialog').getByRole('button',{name:'Stop tasks and quit',exact:true}).click();
  live();await cancel();await login();
  await control();await page.getByRole('dialog').waitFor();
  await page.getByRole('dialog').getByRole('button',{name:'Stop tasks and quit',exact:true}).click();
  await page.reload();await page.getByRole('dialog').waitFor();live();
  await cancel();await login();
  await control();await page.getByRole('dialog').waitFor();
  await page.getByRole('dialog').getByRole('button',{name:'Stop tasks and quit',exact:true}).click();
  const state=await page.evaluate(()=>window.__desktopRecoveryFixture);
  if((unknownOpen?(state.reconciles!==2||state.reads!==0):state.reads!==2)||state.logins!==2)throw Error('Unexpected recovery call count');
  let timer;const closed=new Promise((resolve,reject)=>{browser.once('disconnected',resolve);timer=setTimeout(()=>reject(Error('Terminal owner disposition did not permit Quit')),10000);});
  await page.evaluate(()=>{setTimeout(()=>window.__releaseDesktopCleanup(),0);});await closed;clearTimeout(timer);
  data.desktop_recovery={status:'passed',fixture:unknownOpen?'unknown-open':'known-session',quit_blocked_before_reauthentication:true,pending_quit_survived_reload:true,exact_cleanup_allowed_exit:!unknownOpen,confirmed_cancellation_allowed_exit:unknownOpen,read_calls:state.reads,reconcile_calls:state.reconciles,login_calls:state.logins,open_calls:0,stop_calls:0,observed_at:new Date().toISOString(),scope:'Real native shell/profile/reload with intercepted account and desktop RPCs; no real desktop admission, Stop, or input.'};
  console.log(JSON.stringify(data.desktop_recovery));
 }catch(error){data.desktop_recovery={status:'failed',error:error.message};throw error;}
 finally{fs.writeFileSync(filename,JSON.stringify(data,null,2)+'\n');await browser.close();}
})().catch(error=>{console.error(error.message);process.exitCode=1;});
