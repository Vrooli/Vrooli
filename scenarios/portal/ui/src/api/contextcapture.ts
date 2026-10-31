import { create, equals, fromJson } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { ContextCaptureService, ImportRequestSchema, SourceSchema, RegionSchema, StrokeSchema, type ImportRequest, type Document } from "@vrooli/proto-types/portal/v1/contextcapture/contextcapture_pb";
import type { ContextDraft } from "../features/companion/CompanionPresentation";
import { transport } from "./client";
import { operatorHeaders } from "./desktop";

export const contextCaptureClient = createClient(ContextCaptureService, transport);
const uuid = /^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$/;
const validID = (value:string) => uuid.test(value) && value!=="00000000-0000-0000-0000-000000000000";

// The caller saves the request ID before sending. Never persist this request:
// it contains private pixels. A lost reply is resolved with reconcileImport.
export function prepareContextImport(draft:ContextDraft, requestId:string, retentionSeconds:number, now=Date.now()):ImportRequest {
  const {image,region,strokes}=draft, bounds=image.sourceBounds;
  const captured=Date.parse(image.source.capturedAt);
  if(!validID(requestId) || !validID(image.contextId) || image.source.captureId!==image.contextId || image.expiresAt<=now ||
    !Number.isFinite(captured) || captured>now || now-captured>30000 || image.expiresAt>captured+30000 ||
    !Number.isInteger(retentionSeconds) || retentionSeconds<1 || retentionSeconds>86400 ||
    !Number.isInteger(bounds.width) || !Number.isInteger(bounds.height) || bounds.width<1 || bounds.height<1 || bounds.width*bounds.height>16*1024*1024 ||
    (["x","y","width","height"] as const).some(field=>image.source.bounds[field]!==bounds[field]) ||
    !Object.values(region).every(Number.isInteger) || region.x<0 || region.y<0 || region.width<1 || region.height<1 || region.x+region.width>bounds.width || region.y+region.height>bounds.height ||
    strokes.length>32 || strokes.some(points=>points.length<2 || points.length>256 || points.some(p=>!Number.isFinite(p.x)||!Number.isFinite(p.y)||p.x<0||p.y<0||p.x>bounds.width||p.y>bounds.height)))throw new Error("Invalid context draft");
  const prefix="data:image/png;base64,";
  if(!image.dataUrl.startsWith(prefix) || image.dataUrl.length>44739266)throw new Error("Invalid context image");
  const encoded=image.dataUrl.slice(prefix.length);
  if(encoded.length%4!==0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(encoded))throw new Error("Invalid context image");
  const raw=atob(encoded);
  if(!raw.length || raw.length>32*1024*1024)throw new Error("Invalid context image");
  const png=Uint8Array.from(raw,c=>c.charCodeAt(0));
  return create(ImportRequestSchema,{requestId,retentionSeconds,png,source:fromJson(SourceSchema,image.source),region:{...region},strokes:strokes.map(points=>({points:points.map(p=>({...p}))}))});
}

export function validateImportedContext(document:Document, request:ImportRequest, now=Date.now()):Document {
  const created=Number(document.createdAt?.seconds ?? 0n)*1000+(document.createdAt?.nanos ?? 0)/1000000;
  const expires=Number(document.expiresAt?.seconds ?? 0n)*1000+(document.expiresAt?.nanos ?? 0)/1000000;
  if(!validID(document.id) || document.requestId!==request.requestId || !document.source || !request.source || !equals(SourceSchema,document.source,request.source) ||
    !document.region || !request.region || !equals(RegionSchema,document.region,request.region) || document.strokes.length!==request.strokes.length ||
    document.strokes.some((stroke,index)=>{const expected=request.strokes[index];return !expected || !equals(StrokeSchema,stroke,expected);}) || !/^[0-9a-f]{64}$/.test(document.originalSha256) ||
    !document.createdAt || !document.expiresAt || created>now || expires<=now || expires<=created || Math.abs(expires-created-request.retentionSeconds*1000)>1)throw new Error("Invalid context import receipt");
  return document;
}

export async function importReviewedContext(request:ImportRequest, token:string, signal:AbortSignal, client=contextCaptureClient):Promise<Document> {
  if(!token || /\s/.test(token))throw new Error("Account authentication required");
  signal.throwIfAborted();
  const result=await client.import(request,{headers:operatorHeaders(token),signal,timeoutMs:15000});
  signal.throwIfAborted();
  return validateImportedContext(result,request);
}
