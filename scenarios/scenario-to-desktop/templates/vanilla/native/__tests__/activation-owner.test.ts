import { afterEach, expect, it } from 'vitest';
import { createServer, type Server } from 'node:http';
import { mkdtemp, rm, writeFile, chmod, symlink } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { admittedLocalActivationCapture, configuredLocalActivationCapture, localActivationCapture, nativeWindowIdentity, type LocalActivationBinding } from '../activation-owner';
const handle = () => { const b=Buffer.alloc(4); b.writeUInt32LE(42); return b; };
let server: Server | undefined;
let directory: string | undefined;
afterEach(async () => { if (server) { server.closeAllConnections(); await new Promise<void>(done => server!.close(() => done())); server = undefined; } if (directory) await rm(directory, { recursive: true, force: true }); });
it('accepts only valid native X11 window handles', () => {
 expect(nativeWindowIdentity(handle())).toBe('42');
 expect(() => nativeWindowIdentity(Buffer.alloc(4))).toThrow();
 expect(() => nativeWindowIdentity(Buffer.alloc(3))).toThrow();
 const tooLarge=Buffer.alloc(8);tooLarge.writeBigUInt64LE(0x100000000n);expect(() => nativeWindowIdentity(tooLarge)).toThrow();
});
it('uses only the configured Unix owner and cloned exact session with the main-process window identity', async () => {
 directory = await mkdtemp(join(tmpdir(), 'activation-'));
 const socket = join(directory, 'owner.sock'); let calls = 0;
 server = createServer((req, res) => {
  calls++; expect(req.headers.authorization).toBeUndefined();
  if (req.url?.endsWith('/DeleteActivation')) {
   let deletion = ''; req.on('data', chunk => { deletion += String(chunk); });
   req.on('end', () => { expect(JSON.parse(deletion)).toEqual({ session: {sessionId:'original',desktopSessionId:'session2',surface:{ownerScenario:'device-control',surfaceId:'desktop',target:{ownerScenario:'vrooli-bridge',resourceId:'host',hostNodeId:'host'}}},contextId:'59a6140b-a9cd-43d5-992b-2506b0177424' }); res.end('{}'); });
   return;
  }
  expect(req.url).toBe('/vrooli.device_control.v1.desktop.DesktopOwnerService/CaptureCompanionActivation');
  let body = ''; req.on('data', chunk => { body += String(chunk); });
  req.on('end', () => { expect(JSON.parse(body).session.sessionId).toBe('original');expect(JSON.parse(body).companionWindow).toBe('42');expect(JSON.parse(body).companionPid).toBeUndefined(); res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify({ contextId: '59a6140b-a9cd-43d5-992b-2506b0177424', capturedAt: new Date().toISOString(), expiresAt: new Date(Date.now()+20000).toISOString(), nativeWindow: 'must-not-escape' })); });
 });
 await new Promise<void>(done => server!.listen(socket, done));
 const binding: LocalActivationBinding = { socket, session: { sessionId: 'original', desktopSessionId: 'session2', surface: { ownerScenario: 'device-control', surfaceId: 'desktop', target: { ownerScenario: 'vrooli-bridge', resourceId: 'host', hostNodeId: 'host' } } } };
 const adapter = localActivationCapture(binding, '42'); binding.session.sessionId = 'mutated';
 const result = await adapter.capture(new AbortController().signal);
 expect(Object.keys(result).sort()).toEqual(['contextId', 'discard', 'expiresAt']); expect(calls).toBe(1);
 const config = join(directory!, 'binding.json'); binding.session.sessionId = 'original';
 await writeFile(config, JSON.stringify(binding), { mode: 0o600 });
 const configured = configuredLocalActivationCapture(config, handle);
 await expect(configured.capture(new AbortController().signal)).resolves.toMatchObject({ contextId: result.contextId }); expect(calls).toBe(2);
 await chmod(config, 0o644);
 await expect(configured.capture(new AbortController().signal)).rejects.toThrow(); expect(calls).toBe(2);
 await chmod(config, 0o600);
 const linked = join(directory!, 'linked'); await symlink(config, linked);
 await expect(configuredLocalActivationCapture(linked, handle).capture(new AbortController().signal)).rejects.toThrow(); expect(calls).toBe(2);
 await expect(configuredLocalActivationCapture(undefined, handle).capture(new AbortController().signal)).rejects.toThrow();
 binding.session.sessionId = 'renewed';
 await result.discard!(); expect(calls).toBe(3);


});


it.each(['success','lost-open','capture-failed','pending-cleanup'])('bounds on-demand admission and reconciles without replay: %s', async scenario => {
 directory=await mkdtemp(join(tmpdir(),'activation-admit-'));
 const socket=join(directory,'owner.sock');
 const surface={ownerScenario:'device-control',surfaceId:'desktop',target:{ownerScenario:'vrooli-bridge',resourceId:'host',hostNodeId:'host'}};
 const session={surface,sessionId:'original',desktopSessionId:'session2'};
 const methods:string[]=[];
 let intent:unknown;
 server=createServer((req,res)=>{
  const method=req.url!.split('/').pop()!;methods.push(method);
  let body='';req.on('data',chunk=>{body+=String(chunk);});
  req.on('end',()=>{
   const payload=JSON.parse(body);
   res.setHeader('Content-Type','application/json');
   if(method==='Open') {
    expect(payload).toMatchObject({surface,control:false,ttlSeconds:30});
    expect(payload.requestId).toMatch(/^[0-9a-f-]{36}$/);intent=payload;
    if(scenario==='lost-open'){res.statusCode=503;res.end('{}');return;}
    res.end(JSON.stringify({session,control:false,expiresAt:new Date(Date.now()+29000).toISOString()}));return;
   }
   if(method==='ReconcileOpen'){expect(payload).toEqual(intent);res.end(JSON.stringify({state:'forwarding',session}));return;}
   expect(payload.session).toEqual(session);
   if(method==='CaptureCompanionActivation') {
    expect(payload.companionWindow).toBe('42');
    if(scenario==='capture-failed'){res.statusCode=503;res.end('{}');return;}
    res.end(JSON.stringify({contextId:'59a6140b-a9cd-43d5-992b-2506b0177424',capturedAt:new Date().toISOString(),expiresAt:new Date(Date.now()+20000).toISOString()}));return;
   }
   if(method==='DeleteActivation'){expect(payload.contextId).toBe('59a6140b-a9cd-43d5-992b-2506b0177424');res.end('{}');return;}
   if(method==='Stop'){res.end('{}');return;}
   expect(method).toBe('ReadCleanup');res.end(JSON.stringify({session,released:scenario!=='pending-cleanup'}));
  });
 });
 await new Promise<void>(done=>server!.listen(socket,done));
 const source={version:2 as const,socket,surface};
 const config=join(directory,'source.json');await writeFile(config,JSON.stringify(source),{mode:0o600});
 const adapter=scenario==='success'?configuredLocalActivationCapture(config,handle):admittedLocalActivationCapture(source,'42');
 if(scenario==='lost-open'||scenario==='capture-failed') {
  await expect(adapter.capture(new AbortController().signal)).rejects.toThrow();
  expect(methods).toEqual(scenario==='lost-open'?['Open','ReconcileOpen','Stop','ReadCleanup']:['Open','CaptureCompanionActivation','Stop','ReadCleanup']);
 } else {
  const captured=await adapter.capture(new AbortController().signal);
  expect(methods).toEqual(['Open','CaptureCompanionActivation']);
  if(scenario==='pending-cleanup') await expect(captured.discard!()).rejects.toThrow(); else await captured.discard!();
  expect(methods).toEqual(['Open','CaptureCompanionActivation','DeleteActivation','Stop','ReadCleanup']);
 }
 expect(methods.filter(method=>method==='Open')).toHaveLength(1);
});

it('reads only the exact opted-in frozen image and rejects substituted evidence', async () => {
 directory=await mkdtemp(join(tmpdir(),'activation-image-'));
 const socket=join(directory,'owner.sock');
 const session={sessionId:'original',desktopSessionId:'session2',surface:{ownerScenario:'device-control',surfaceId:'desktop',target:{ownerScenario:'vrooli-bridge',resourceId:'host',hostNodeId:'host'}}};
 const ref={contextId:'59a6140b-a9cd-43d5-992b-2506b0177424',capturedAt:new Date().toISOString(),expiresAt:new Date(Date.now()+20000).toISOString(),hasImage:true,sourceBounds:{x:-1,y:2,width:1,height:1},displayId:'d',geometryRevision:'g'};
 const png='iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=';
 let fault='valid';let captures=0;
 server=createServer((req,res)=>{let body='';req.on('data',chunk=>{body+=String(chunk);});req.on('end',()=>{
  const payload=JSON.parse(body);expect(payload.session).toEqual(session);
  if(req.url?.endsWith('/CaptureCompanionActivation')){captures++;expect(payload.includeImage).toBe(true);res.end(JSON.stringify(ref));return;}
  expect(req.url?.endsWith('/ReadActivationImage')).toBe(true);expect(payload.contextId).toBe(ref.contextId);
  const reference=structuredClone(ref);
  if(fault==='id')reference.contextId='aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa';
  if(fault==='bounds')reference.sourceBounds.width=2;
  if(fault==='display')reference.displayId='other';
  if(fault==='geometry')reference.geometryRevision='other';
  res.end(JSON.stringify({reference,png:fault==='png'?'aW52YWxpZA==':png,ignored:'x'.repeat(20000)}));
 });});
 await new Promise<void>(done=>server!.listen(socket,done));
 const config=join(directory,'source.json');await writeFile(config,JSON.stringify({version:2,socket,surface:session.surface,includeImage:'yes'}),{mode:0o600});
 await expect(configuredLocalActivationCapture(config,handle).capture(new AbortController().signal)).rejects.toThrow();expect(captures).toBe(0);
 const result=await localActivationCapture({socket,session,includeImage:true},'42').capture(new AbortController().signal);
 const preview=await result.readImage!(new AbortController().signal);
 expect(preview).toEqual({contextId:ref.contextId,expiresAt:Date.parse(ref.expiresAt),mimeType:'image/png',dataUrl:`data:image/png;base64,${png}`,sourceBounds:ref.sourceBounds,source:{surface:session.surface,captureId:ref.contextId,displayId:ref.displayId,geometryRevision:ref.geometryRevision,capturedAt:ref.capturedAt,bounds:ref.sourceBounds}});
 preview.source.surface.target.resourceId="mutated";
 preview.source.bounds.x=42;
 const again=await result.readImage!(new AbortController().signal);
 expect(again.source.surface).toEqual(session.surface);
 expect(again.source.bounds).toEqual(ref.sourceBounds);
 for(fault of ['id','bounds','png','display','geometry'])await expect(result.readImage!(new AbortController().signal)).rejects.toThrow();
 expect(captures).toBe(1);
 fault='valid';
 for(const invalid of ['', 'x'.repeat(129)]) {
  ref.displayId=invalid;
  await expect(localActivationCapture({socket,session,includeImage:true},'42').capture(new AbortController().signal)).rejects.toThrow();
 }
});
