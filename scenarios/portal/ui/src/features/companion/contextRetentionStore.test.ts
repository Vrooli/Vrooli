import { webcrypto } from "node:crypto";
import { afterEach, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { DocumentSchema, RenderResponseSchema, ReconcileImportResponseSchema, DeleteResponseSchema, ImportState } from "@vrooli/proto-types/portal/v1/contextcapture/contextcapture_pb";
import { contextCaptureClient } from "../../api/contextcapture";
import { ContextRetention, contextRetentionKey, type ContextRetentionLock } from "./contextRetentionStore";
import type { ContextDraft } from "./CompanionPresentation";
afterEach(()=>{vi.restoreAllMocks();vi.unstubAllGlobals();localStorage.clear();});
const directLock:ContextRetentionLock=(_signal,run)=>run();
const makeStore=(storage:Pick<Storage,"getItem"|"setItem"|"removeItem">=localStorage)=>new ContextRetention(storage,contextCaptureClient,directLock);
function fixture(){
 const account={id:"alice",realm:"realm",token:"secret",expires:Date.now()+60000};
 const id="59a6140b-a9cd-43d5-992b-2506b0177424",bounds={x:0,y:0,width:1,height:1};
 const draft:ContextDraft={image:{contextId:id,expiresAt:Date.now()+20000,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",sourceBounds:bounds,source:{surface:{ownerScenario:"device-control",surfaceId:"desktop",target:{ownerScenario:"vrooli-bridge",resourceId:"host",hostNodeId:"host"}},captureId:id,capturedAt:new Date().toISOString(),displayId:"d",geometryRevision:"g",bounds}},region:{x:0,y:0,width:1,height:1},strokes:[]};
 const store=makeStore();store.setAccount(account);
 return {store,account,draft};
}
it("saves only recovery identity before upload and resolves a lost reply without replay",async()=>{
 const {store,account,draft}=fixture();
 const send=vi.spyOn(contextCaptureClient,"import").mockImplementation((request)=>{
  const raw=localStorage.getItem(contextRetentionKey)??"";
  expect(JSON.parse(raw)).toEqual({version:1,actor:{id:"alice",realm:"realm"},requestId:request.requestId});
  expect(raw).not.toContain("secret");expect(raw).not.toContain("png");
  return Promise.reject(new Error("lost reply"));
 });
 await store.retain(draft);expect(store.getSnapshot().failed).toBe(true);
 const marker=store.getSnapshot().marker;expect(marker).toBeDefined();
 const restored=makeStore();restored.setAccount(account);
 const doc=create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId:marker?.requestId,expiresAt:{seconds:BigInt(Math.floor(Date.now()/1000)+3600)}});
 const recover=vi.spyOn(contextCaptureClient,"reconcileImport").mockResolvedValue(create(ReconcileImportResponseSchema,{requestId:marker?.requestId,state:ImportState.READY,document:doc}));
 await restored.recover();expect(restored.getSnapshot().document).toEqual(doc);expect(send).toHaveBeenCalledTimes(1);
 expect(recover.mock.calls[0]?.[1]?.headers).toEqual({Authorization:"Bearer secret"});
 vi.spyOn(contextCaptureClient,"delete").mockResolvedValue(create(DeleteResponseSchema));
 await restored.remove();expect(restored.getSnapshot().marker).toBeUndefined();expect(localStorage.getItem(contextRetentionKey)).toBeNull();
});
it("refuses upload when durable recovery cannot be saved",async()=>{
 const {account,draft}=fixture();
 const store=makeStore({getItem:()=>null,setItem:()=>{throw new Error("quota");},removeItem:()=>undefined});store.setAccount(account);
 const send=vi.spyOn(contextCaptureClient,"import");await store.retain(draft);
 expect(send).not.toHaveBeenCalled();expect(store.getSnapshot().failed).toBe(true);
});
it("fences late import success after logout even if the same account returns",async()=>{
 const {store,account,draft}=fixture();
 let finish:()=>void=()=>{throw new Error("not started");};
 vi.spyOn(contextCaptureClient,"import").mockImplementation(request=>new Promise(resolve=>{finish=()=>resolve(create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId:request.requestId,source:request.source,region:request.region,strokes:request.strokes,originalSha256:"a".repeat(64),createdAt:{seconds:BigInt(Math.floor(Date.now()/1000))},expiresAt:{seconds:BigInt(Math.floor(Date.now()/1000)+3600)}}));}));
 const pending=store.retain(draft);store.setAccount(undefined);store.setAccount(account);finish();await pending;
 expect(store.getSnapshot().document).toBeUndefined();expect(store.getSnapshot().marker).toBeDefined();expect(store.getSnapshot().failed).toBe(true);
 const recover=vi.spyOn(contextCaptureClient,"reconcileImport");store.setAccount({...account,id:"bob"});await store.recover();expect(recover).not.toHaveBeenCalled();
});
it("keeps staging unresolved and clears an authoritative unavailable receipt",async()=>{
 const {store,draft}=fixture();vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const requestId=store.getSnapshot().marker?.requestId;
 const recover=vi.spyOn(contextCaptureClient,"reconcileImport").mockResolvedValue(create(ReconcileImportResponseSchema,{requestId,state:ImportState.STAGING}));
 await store.recover();expect(store.getSnapshot().marker).toBeDefined();expect(store.getSnapshot().document).toBeUndefined();
 recover.mockResolvedValue(create(ReconcileImportResponseSchema,{requestId,state:ImportState.UNAVAILABLE}));await store.recover();expect(store.getSnapshot().marker).toBeUndefined();
});

it("serializes two windows and prevents stale stores from replacing a pending import",async()=>{
 const {account,draft}=fixture();let held=false;
 const lock:ContextRetentionLock=async(signal,run)=>{signal.throwIfAborted();if(held)throw new Error("busy");held=true;try{await run();}finally{held=false;}};
 const first=new ContextRetention(localStorage,contextCaptureClient,lock),second=new ContextRetention(localStorage,contextCaptureClient,lock);
 first.setAccount(account);second.setAccount(account);
 let finish:()=>void=()=>{throw new Error("not started");};
 const send=vi.spyOn(contextCaptureClient,"import").mockImplementation(()=>new Promise((_resolve,reject)=>{finish=()=>reject(new Error("lost"));}));
 const pending=first.retain(draft);const saved=localStorage.getItem(contextRetentionKey);
 await second.retain(draft);expect(send).toHaveBeenCalledTimes(1);expect(localStorage.getItem(contextRetentionKey)).toBe(saved);
 finish();await pending;
 await second.retain(draft);expect(send).toHaveBeenCalledTimes(1);expect(second.getSnapshot().marker).toEqual(first.getSnapshot().marker);
});
it("does not treat an absent receipt as proof a delayed import cannot arrive",async()=>{
 const {store,draft}=fixture();vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const marker=store.getSnapshot().marker;
 vi.spyOn(contextCaptureClient,"reconcileImport").mockResolvedValue(create(ReconcileImportResponseSchema,{requestId:marker?.requestId,state:ImportState.ABSENT}));
 await store.recover();expect(store.getSnapshot().marker).toEqual(marker);expect(store.getSnapshot().document).toBeUndefined();
});
it("refuses to clear a successor marker changed during an old recovery",async()=>{
 const {store,draft}=fixture();vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const old=store.getSnapshot().marker;const successor={version:1,actor:old?.actor,requestId:"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"};
 vi.spyOn(contextCaptureClient,"reconcileImport").mockImplementation(()=>{
  localStorage.setItem(contextRetentionKey,JSON.stringify(successor));
  return Promise.resolve(create(ReconcileImportResponseSchema,{requestId:old?.requestId,state:ImportState.UNAVAILABLE}));
 });
 await store.recover();expect(JSON.parse(localStorage.getItem(contextRetentionKey)??"{}")).toEqual(successor);expect(store.getSnapshot().failed).toBe(true);
});

it("keeps uncertain cancellation tracked until its server acknowledgement",async()=>{
 const {store,draft}=fixture();vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const marker=store.getSnapshot().marker;
 const cancel=vi.spyOn(contextCaptureClient,"cancelImport").mockRejectedValueOnce(new Error("lost cancellation reply"));
 await store.cancel();expect(store.getSnapshot().marker).toEqual(marker);expect(store.getSnapshot().failed).toBe(true);
 cancel.mockResolvedValue(create(DeleteResponseSchema));await store.cancel();
 expect(cancel.mock.calls[0]?.[0]).toEqual({requestId:marker?.requestId});
 expect(cancel.mock.calls[0]?.[1]?.headers).toEqual({Authorization:"Bearer secret"});
 expect(store.getSnapshot().marker).toBeUndefined();expect(localStorage.getItem(contextRetentionKey)).toBeNull();
});

it("verifies rendered bytes and revokes preview URLs when the account changes",async()=>{
 vi.stubGlobal("crypto",webcrypto);
 const makeURL=vi.fn(()=>"blob:private-preview"),revoke=vi.fn();
 vi.stubGlobal("URL",class extends URL{static createObjectURL=makeURL;static revokeObjectURL=revoke;});
 const {store,account,draft}=fixture();vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const requestId=store.getSnapshot().marker?.requestId;
 const doc=create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId,region:{width:1,height:1},expiresAt:{seconds:BigInt(Math.floor(Date.now()/1000)+3600)}});
 vi.spyOn(contextCaptureClient,"reconcileImport").mockResolvedValue(create(ReconcileImportResponseSchema,{requestId,state:ImportState.READY,document:doc}));await store.recover();
 const pixels=new Uint8Array(Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=","base64"));
 const hash=Buffer.from(await webcrypto.subtle.digest("SHA-256",pixels)).toString("hex");
 const render=vi.spyOn(contextCaptureClient,"render").mockResolvedValue(create(RenderResponseSchema,{document:doc,png:pixels,renderedSha256:"0".repeat(64)}));
 await store.render();expect(store.getSnapshot().preview).toBeUndefined();expect(makeURL).not.toHaveBeenCalled();
 render.mockResolvedValue(create(RenderResponseSchema,{document:doc,png:pixels,renderedSha256:hash}));await store.render();
 expect(store.getSnapshot().preview).toEqual({url:"blob:private-preview",sha256:hash});expect(makeURL).toHaveBeenCalledTimes(1);
 expect(localStorage.getItem(contextRetentionKey)).not.toContain(hash);
 store.setAccount({...account,id:"bob"});expect(store.getSnapshot().preview).toBeUndefined();expect(revoke).toHaveBeenCalledWith("blob:private-preview");
});

it("requires the retained document to be explicitly attached and clears that choice on account change",async()=>{
 const {store,account,draft}=fixture();
 vi.spyOn(contextCaptureClient,"import").mockRejectedValue(new Error("lost"));await store.retain(draft);
 const requestId=store.getSnapshot().marker?.requestId;
 const doc=create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId,expiresAt:{seconds:BigInt(Math.floor(Date.now()/1000)+3600)}});
 vi.spyOn(contextCaptureClient,"reconcileImport").mockResolvedValue(create(ReconcileImportResponseSchema,{requestId,state:ImportState.READY,document:doc}));await store.recover();
 expect(store.getSnapshot().attachedDocumentId).toBeUndefined();store.attach();
 expect(store.getSnapshot().attachedDocumentId).toBe(doc.id);store.setAccount({...account,id:"bob"});
 expect(store.getSnapshot().attachedDocumentId).toBeUndefined();
});
