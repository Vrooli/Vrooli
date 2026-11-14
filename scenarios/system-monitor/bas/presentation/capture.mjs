import fs from 'node:fs/promises';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { responseFor } from './world.mjs';
const [distArg,outArg,puppeteerArg,mode='preview']=process.argv.slice(2);
if(!distArg||!outArg||!puppeteerArg)throw Error('Expected dist, new output directory, Puppeteer module, optional record');
const dist=await fs.realpath(distArg), out=path.resolve(outArg);
await fs.mkdir(out); // Refuse to overwrite evidence.
const {default:puppeteer}=await import(pathToFileURL(path.resolve(puppeteerArg)).href);
const audit=[],errors=[],responses=[];
const fonts=JSON.parse(await fs.readFile(path.join(dist,'capture-font-map.json'),'utf8'));
const browser=await puppeteer.launch({executablePath:'/usr/bin/google-chrome',headless:true,args:['--no-sandbox','--window-size=3200,2087','--disable-gpu','--disable-background-networking','--disable-component-update','--disable-sync','--host-resolver-rules=MAP * ~NOTFOUND']});
const page=await browser.newPage();
await page.setViewport({width:3200,height:2000,deviceScaleFactor:1});
const metricsSession=await page.createCDPSession();
await metricsSession.send('Emulation.setDeviceMetricsOverride',{width:3200,height:2000,deviceScaleFactor:1,mobile:false,screenWidth:3200,screenHeight:2000});
await page.setBypassServiceWorker(true);
await page.evaluateOnNewDocument(()=>{
 document.addEventListener('DOMContentLoaded',()=>{document.documentElement.style.zoom='2';});
 if(navigator.serviceWorker)navigator.serviceWorker.register=async()=>({});
 window.WebSocket=class {constructor(){throw Error('Presentation harness denies WebSocket egress');}};
});
page.on('pageerror',e=>errors.push(String(e)));
page.on('console',m=>{if(m.type()==='error')errors.push(m.text());});
await page.setRequestInterception(true);
const mime={'.html':'text/html','.js':'text/javascript','.css':'text/css','.png':'image/png','.svg':'image/svg+xml','.woff2':'font/woff2','.json':'application/json','.ico':'image/x-icon'};
let step=0;
page.on('request',async req=>{
 const u=new URL(req.url());
 try{
  if(fonts[req.url()]){const f=fonts[req.url()];audit.push({url:req.url(),result:'bundled-font'});await req.respond({status:200,contentType:f.contentType,headers:{'Access-Control-Allow-Origin':'*'},body:await fs.readFile(path.join(dist,f.file))});return;}
  if(u.origin!=='https://vega.presentation.invalid'){audit.push({url:req.url(),result:'denied-origin'});await req.abort();return;}
  let body={};try{body=JSON.parse(req.postData()||'{}');}catch{}
  const isRPC=u.pathname.startsWith('/vrooli.system_monitor.');
  const data=(isRPC?req.method()==='POST':['GET','HEAD'].includes(req.method()))?responseFor(u.pathname,body,Date.now(),step++):undefined;
  if(data!==undefined){audit.push({path:u.pathname,result:'fixture'});responses.push({path:u.pathname,body,response:data});await req.respond({status:200,contentType:'application/json',body:JSON.stringify(data)});return;}
  const rel=decodeURIComponent(u.pathname).replace(/^\/+/,''), p=path.resolve(dist,rel.replace(/^metrics\/(public|assets)\//,'$1/')||'index.html');
  if(!p.startsWith(dist+path.sep)){await req.abort();return;}
  let dataFile;
  if(['GET','HEAD'].includes(req.method()) && ['/', '/metrics/cpu', '/metrics/gpu'].includes(u.pathname)){const html=(await fs.readFile(path.join(dist,'index.html'),'utf8')).replace('<head>','<head><base href="/">');audit.push({path:u.pathname,result:'bundle-route'});await req.respond({status:200,contentType:'text/html',body:html});return;}
  if(['GET','HEAD'].includes(req.method()))try{dataFile=await fs.readFile(p);}catch{}
  if(dataFile){audit.push({path:u.pathname,result:'bundle'});await req.respond({status:200,contentType:req.isNavigationRequest()?'text/html':mime[path.extname(p)]||'application/octet-stream',body:dataFile});return;}
  audit.push({path:u.pathname,method:req.method(),type:req.resourceType(),result:'denied-unfixture'});await req.respond({status:501,contentType:'application/json',body:JSON.stringify({error:'Presentation procedure not authored'})});
 }catch(e){errors.push(String(e));if(!req.isInterceptResolutionHandled())await req.abort();}
});
const wait=ms=>new Promise(r=>setTimeout(r,ms));
const save=async name=>{await page.screenshot({path:path.join(out,name+'.png')});await fs.writeFile(path.join(out,name+'.txt'),await page.evaluate(()=>document.body.innerText));};
try{
 await page.goto('https://vega.presentation.invalid/?theme=dark',{waitUntil:'networkidle0'});await wait(4200);
 await save('dashboard');
 if(mode==='record'){
  // Visible pointer is capture chrome only; product DOM and styles remain intact.
  await page.evaluate(()=>{const c=document.createElement('div');c.id='capture-pointer';c.style.cssText='position:fixed;left:0;top:0;width:18px;height:18px;border:2px solid white;background:#9ec5ff55;border-radius:50%;pointer-events:none;z-index:2147483647;zoom:0.5;transform:translate(-50%,-50%);box-shadow:0 0 0 1px #0008';document.body.append(c);window.addEventListener('mousemove',e=>{c.style.left=e.clientX+'px';c.style.top=e.clientY+'px';});});
  const cdp=await page.createCDPSession();
  const record=async(name,action)=>{
   const dir=path.join(out,name);await fs.mkdir(dir);const frames=[];let pending=Promise.resolve();
   const onFrame=e=>{const n=frames.length;frames.push({name:`${String(n).padStart(6,'0')}.jpg`,time:e.metadata.timestamp});pending=pending.then(()=>fs.writeFile(path.join(dir,frames[n].name),Buffer.from(e.data,'base64')));void cdp.send('Page.screencastFrameAck',{sessionId:e.sessionId});};
   cdp.on('Page.screencastFrame',onFrame);await cdp.send('Page.startScreencast',{format:'jpeg',quality:95,maxWidth:3200,maxHeight:2000,everyNthFrame:1});
   await wait(400);await action();await wait(900);await cdp.send('Page.stopScreencast');cdp.off('Page.screencastFrame',onFrame);await pending;
   const concat=frames.map((f,i)=>`file '${f.name}'\nduration ${i+1<frames.length?Math.max(.01,frames[i+1].time-f.time):.1}`).join('\n')+`\nfile '${frames.at(-1).name}'\n`;
   await fs.writeFile(path.join(dir,'frames.txt'),concat);await fs.writeFile(path.join(dir,'timestamps.json'),JSON.stringify(frames));
   const video=path.join(out,name+'.mp4');const r=spawnSync('ffmpeg',['-v','error','-threads','2','-f','concat','-safe','0','-i',path.join(dir,'frames.txt'),'-an','-r','30','-c:v','libx264','-threads','2','-crf','16','-pix_fmt','yuv420p',video],{encoding:'utf8'});if(r.status!==0)throw Error(r.stderr);
   await save(name+'-end');console.log(name,frames.length,frames.at(-1).time-frames[0].time);
  };
  const move=async(x,y,scale=2)=>{await page.mouse.move(x*scale,y*scale,{steps:20});await wait(350);};
  // Warm lazy detail chunk, then reset via the application's own route.
  await page.evaluate(()=>[...document.querySelectorAll('button')].find(b=>b.textContent.trim()==='Open CPU detail').click());await wait(1600);
  await page.goBack({waitUntil:'networkidle0'});await wait(500);
  await record('01-dashboard-to-cpu',async()=>{
   const box=await page.evaluate(()=>{const e=[...document.querySelectorAll('button')].find(b=>b.textContent.trim()==='Open CPU detail');const b=e.getBoundingClientRect();return{x:b.x+b.width/2,y:b.y+b.height/2};});
   await move(1100,360);await wait(1000);await move(box.x,box.y,1);await page.mouse.click(box.x,box.y);await wait(1600);await move(900,500);await wait(1000);
  });
  await record('02-cpu-chart',async()=>{await move(400,480);await wait(700);await move(800,480);await wait(1200);await move(1180,480);await wait(1000);});
  await record('03-attribution',async()=>{
   await page.evaluate(async()=>{const el=document.querySelector('[data-detail-section="cpu-attribution"]');const target=el.getBoundingClientRect().top+window.scrollY-280;const start=window.scrollY,t0=performance.now();await new Promise(resolve=>{const frame=t=>{const p=Math.min(1,(t-t0)/1700),e=p<.5?2*p*p:1-(-2*p+2)**2/2;window.scrollTo(0,start+(target-start)*e);p<1?requestAnimationFrame(frame):resolve();};requestAnimationFrame(frame);});});await move(950,420);await wait(1600);
  });
  await page.goto('https://vega.presentation.invalid/?theme=dark',{waitUntil:'networkidle0'});await wait(2500);await page.evaluate(()=>[...document.querySelectorAll('button')].find(b=>b.textContent.trim()==='Open GPU detail').click());await wait(1800);
  await page.evaluate(()=>{const c=document.createElement('div');c.style.cssText='position:fixed;width:18px;height:18px;border:2px solid white;background:#9ec5ff55;border-radius:50%;pointer-events:none;z-index:2147483647;zoom:0.5;transform:translate(-50%,-50%)';document.body.append(c);window.addEventListener('mousemove',e=>{c.style.left=e.clientX+'px';c.style.top=e.clientY+'px';});});
  await record('04-gpu',async()=>{await move(820,470);await wait(1600);await page.evaluate(async()=>{const start=window.scrollY,t0=performance.now();await new Promise(resolve=>{const frame=t=>{const p=Math.min(1,(t-t0)/1600);window.scrollTo(0,start+760*(p*p*(3-2*p)));p<1?requestAnimationFrame(frame):resolve();};requestAnimationFrame(frame);});});await wait(1700);});
 }
}finally{
 await fs.writeFile(path.join(out,'network-audit.json'),JSON.stringify(audit,null,2));await fs.writeFile(path.join(out,'responses.json'),JSON.stringify(responses,null,2));await fs.writeFile(path.join(out,'errors.json'),JSON.stringify(errors,null,2));
 const files=await fs.readdir(dist,{recursive:true}),hashes={};for(const f of files){const p=path.join(dist,f);if((await fs.stat(p)).isFile())hashes[f]=createHash('sha256').update(await fs.readFile(p)).digest('hex');}
 await fs.writeFile(path.join(out,'bundle-hashes.json'),JSON.stringify(hashes,null,2));await browser.close();
}
console.log(JSON.stringify({errors,denied:audit.filter(x=>x.result.startsWith('denied')),out},null,2));
