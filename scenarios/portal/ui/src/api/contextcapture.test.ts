import { create, clone } from "@bufbuild/protobuf";
import { DocumentSchema } from "@vrooli/proto-types/portal/v1/contextcapture/contextcapture_pb";
import { expect, it, vi } from "vitest";
import type { ContextDraft } from "../features/companion/CompanionPresentation";
import { contextCaptureClient, importReviewedContext, prepareContextImport, validateImportedContext } from "./contextcapture";

function fixture() {
 const now=Date.now(), requestId="59a6140b-a9cd-43d5-992b-2506b0177424";
 const bounds={x:-100,y:-50,width:200,height:100};
 const draft:ContextDraft={image:{contextId:requestId,expiresAt:now+20000,mimeType:"image/png",dataUrl:"data:image/png;base64,aW1hZ2U=",sourceBounds:bounds,source:{surface:{ownerScenario:"device-control",surfaceId:"desktop",target:{ownerScenario:"vrooli-bridge",resourceId:"host",hostNodeId:"host"}},captureId:requestId,capturedAt:new Date(now).toISOString(),displayId:"display",geometryRevision:"geometry",bounds}},region:{x:10,y:5,width:50,height:40},strokes:[[{x:10,y:5},{x:15,y:10}]]};
 const request=prepareContextImport(draft,requestId,60,now);
 const timestamp=(ms:number)=>({seconds:BigInt(Math.floor(ms/1000)),nanos:ms%1000*1000000});
 const document=create(DocumentSchema,{id:"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",requestId,source:request.source,region:request.region,strokes:request.strokes,originalSha256:"a".repeat(64),createdAt:timestamp(now),expiresAt:timestamp(now+60000)});
 return {draft,request,document,now};
}
it("freezes reviewed source, local crop, strokes and original bytes into the import",()=>{
 const {draft,request}=fixture();
 expect(request.source?.bounds?.x).toBe(-100);
 expect(request.region?.x).toBe(10);
 expect([...request.png]).toEqual([...new TextEncoder().encode("image")]);
 draft.region.x=20;draft.strokes=[[{x:30,y:5},{x:15,y:10}]];draft.image.source.surface.target.resourceId="changed";
 expect(request.region?.x).toBe(10);expect(request.strokes[0]?.points[0]?.x).toBe(10);expect(request.source?.surface?.target?.resourceId).toBe("host");
});
it("refuses expired capture, invalid crop, nonfinite marks and malformed image before upload",()=>{
 const {draft,request,now}=fixture();
 for(const mutate of [(d:ContextDraft)=>{d.image.expiresAt=now;},(d:ContextDraft)=>{d.region.width=1000;},(d:ContextDraft)=>{d.strokes=[[{x:NaN,y:5},{x:15,y:10}]];},(d:ContextDraft)=>{d.image.source.bounds={...d.image.source.bounds,x:0};},(d:ContextDraft)=>{d.image.dataUrl="data:image/png;base64,?";}]) {
  const bad=structuredClone(draft);mutate(bad);expect(()=>prepareContextImport(bad,request.requestId,60,now)).toThrow();
 }
});
it("requires an exact live publication receipt for the reviewed request",()=>{
 const {request,document,now}=fixture();
 expect(validateImportedContext(document,request,now)).toBe(document);
 for(const mutate of [(d:typeof document)=>{d.requestId="other";},(d:typeof document)=>{if(d.region)d.region.x++;},(d:typeof document)=>{if(d.source)d.source.geometryRevision="other";},(d:typeof document)=>{d.strokes=[];},(d:typeof document)=>{if(d.expiresAt)d.expiresAt.seconds+=1n;}]) {
  const bad=clone(DocumentSchema,document);mutate(bad);expect(()=>validateImportedContext(bad,request,now)).toThrow();
 }
 expect(()=>validateImportedContext(document,request,now+60000)).toThrow();
});
it("uses explicit account authority, bounds the call, and never replays lost imports",async()=>{
 const {request,document}=fixture();
 const call=vi.spyOn(contextCaptureClient,"import").mockResolvedValue(document);
 try {
  const controller=new AbortController();
  await expect(importReviewedContext(request,"account-token",controller.signal)).resolves.toEqual(document);
  expect(call).toHaveBeenCalledWith(request,{headers:{Authorization:"Bearer account-token"},signal:controller.signal,timeoutMs:15000});
  call.mockRejectedValueOnce(new Error("lost reply"));
  await expect(importReviewedContext(request,"account-token",controller.signal)).rejects.toThrow("lost reply");
  expect(call).toHaveBeenCalledTimes(2);
  await expect(importReviewedContext(request,"",controller.signal)).rejects.toThrow();
  controller.abort();await expect(importReviewedContext(request,"account-token",controller.signal)).rejects.toThrow();
  expect(call).toHaveBeenCalledTimes(2);
 } finally {call.mockRestore();}
});
