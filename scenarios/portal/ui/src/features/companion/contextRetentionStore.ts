import { equals } from "@bufbuild/protobuf";
import { DocumentSchema, ImportState, type Document } from "@vrooli/proto-types/portal/v1/contextcapture/contextcapture_pb";
import { contextCaptureClient, importReviewedContext, prepareContextImport } from "../../api/contextcapture";
import { operatorHeaders } from "../../api/desktop";
import type { ContextDraft } from "./CompanionPresentation";

export type ContextAccount={id:string;realm:string;token:string;expires:number};
type Marker={actor:{id:string;realm:string};requestId:string};
export type ContextRetentionSnapshot={preview?:{url:string;sha256:string};marker?:Marker;document?:Document;attachedDocumentId?:string;busy:boolean;failed:boolean};
export const contextRetentionKey="portal.context-retention.v1";
export type ContextRetentionLock=(signal:AbortSignal,run:()=>Promise<void>)=>Promise<void>;
const browserLock:ContextRetentionLock=async(signal,run)=>{
 const locks=(navigator as {locks?:LockManager}).locks;
 if(!locks)throw new Error("Context coordination unavailable");
 await locks.request(contextRetentionKey,{ifAvailable:true},async lock=>{
  signal.throwIfAborted();
  if(!lock)throw new Error("Context operation in another window");
  await run();
 });
};
const same=(a:{id:string;realm:string}|undefined,b:{id:string;realm:string}|undefined)=>Boolean(a&&b&&a.id===b.id&&a.realm===b.realm);
const validID=(id:unknown):id is string=>typeof id==="string"&&/^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(id)&&id!=="00000000-0000-0000-0000-000000000000";

// One unresolved import at a time. Only identity hints survive reload; neither
// pixels, source metadata nor credentials enter browser storage.
export class ContextRetention {
 private state:ContextRetentionSnapshot={busy:false,failed:false};
 private listeners=new Set<()=>void>();
 private account:ContextAccount|undefined;
 private controller:AbortController|undefined;
 private storageFailed=false;
 constructor(private storage:Pick<Storage,"getItem"|"setItem"|"removeItem">=localStorage,private client=contextCaptureClient,private exclusive:ContextRetentionLock=browserLock) {
  try{this.state.marker=this.readMarker();}catch{this.storageFailed=true;this.state.failed=true;}
 }
 private readMarker():Marker|undefined {
  try {
   const raw=this.storage.getItem(contextRetentionKey);
   if(raw!==null){
    if(raw.length>2048)throw new Error("Invalid recovery marker");
    const value:unknown=JSON.parse(raw);
    if(!value||typeof value!=="object"||!("version"in value)||value.version!==1||!("requestId"in value)||!validID(value.requestId)||!("actor"in value)||!value.actor||typeof value.actor!=="object"||!("id"in value.actor)||!("realm"in value.actor)||typeof value.actor.id!=="string"||typeof value.actor.realm!=="string"||!value.actor.id||!value.actor.realm||value.actor.id.length>256||value.actor.realm.length>256)throw new Error("Invalid recovery marker");
    return {actor:{id:value.actor.id,realm:value.actor.realm},requestId:value.requestId};
   }
  }catch{throw new Error("Invalid recovery marker");}
 }
 getSnapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private publish(patch:Partial<ContextRetentionSnapshot>){
  if("document" in patch)patch={...patch,preview:undefined,...(!patch.document?{attachedDocumentId:undefined}:{})};
  if("preview" in patch&&this.state.preview?.url!==patch.preview?.url&&this.state.preview)URL.revokeObjectURL(this.state.preview.url);
  this.state={...this.state,...patch};this.listeners.forEach(listener=>listener());
 }
 setAccount(account:ContextAccount|undefined){
  if(!same(account,this.account)){this.controller?.abort();this.publish({document:undefined});}
 this.account=account;
 }
 attach=()=>{
  try{
   const account=this.authorized(),doc=this.state.document;
   if(!doc||!this.current(account))throw new Error("Context unavailable");
   this.publish({attachedDocumentId:doc.id,failed:false});
  }catch{this.publish({failed:true});}
 };
 detach=()=>this.publish({attachedDocumentId:undefined});
 private authorized(){
  const account=this.account;
  if(!account||!account.id||!account.realm||!account.token||account.expires<=Date.now()||this.storageFailed||(this.state.marker&&!same(account,this.state.marker.actor)))throw new Error("Original account required");
  return account;
 }
 private current(account:ContextAccount){return same(account,this.account)&&Boolean(this.account&&this.account.expires>Date.now());}
 private async operation(run:(account:ContextAccount,signal:AbortSignal)=>Promise<void>){
  if(this.state.busy)return;
  this.publish({busy:true,failed:false});
  this.controller=new AbortController();
  const signal=this.controller.signal;
  try{await this.exclusive(signal,async()=>{
   signal.throwIfAborted();this.synchronize();
   await run(this.authorized(),signal);
  });}catch{this.publish({failed:true});}
  finally{this.controller=undefined;this.publish({busy:false});}
 }
 synchronize=()=>{
  try{
   const marker=this.readMarker(),previous=this.state.marker;
   if(marker?.requestId!==previous?.requestId||!same(marker?.actor,previous?.actor))this.publish({marker,document:undefined});
  }catch{this.storageFailed=true;this.publish({failed:true,document:undefined});}
 };
 private clear(){
  const stored=this.readMarker();
  if(stored?.requestId!==this.state.marker?.requestId||!same(stored?.actor,this.state.marker?.actor))throw new Error("Recovery marker changed");
  this.storage.removeItem(contextRetentionKey);this.publish({marker:undefined,document:undefined});
 }
 retain=(draft:ContextDraft)=>this.operation(async(account,signal)=>{
  if(this.state.marker)throw new Error("Context remains unresolved");
  const request=prepareContextImport(draft,crypto.randomUUID(),3600);
  const marker:Marker={actor:{id:account.id,realm:account.realm},requestId:request.requestId};
  this.storage.setItem(contextRetentionKey,JSON.stringify({version:1,...marker}));
  this.publish({marker});
  const document=await importReviewedContext(request,account.token,signal,this.client);
  if(!this.current(account))throw new Error("Account changed");
  this.publish({document});
 });
 recover=()=>this.operation(async(account,signal)=>{
  const marker=this.state.marker;if(!marker)return;
  const result=await this.client.reconcileImport({requestId:marker.requestId},{headers:operatorHeaders(account.token),signal,timeoutMs:15000});
  signal.throwIfAborted();
  if(!this.current(account)||result.requestId!==marker.requestId)throw new Error("Invalid recovery reply");
  if(result.state===ImportState.UNAVAILABLE){this.clear();return;}
  const doc=result.document;
  if(result.state!==ImportState.READY||!doc||doc.requestId!==marker.requestId||!validID(doc.id)||Number(doc.expiresAt?.seconds??0n)*1000<=Date.now())throw new Error("Context unavailable");
  this.publish({document:doc});
 });
 hidePreview=()=>this.publish({preview:undefined});
 previewFailed=()=>this.publish({preview:undefined,failed:true});
 render=()=>this.operation(async(account,signal)=>{
  this.hidePreview();
  const doc=this.state.document;if(!doc)throw new Error("Recover context first");
  const response=await this.client.render({id:doc.id},{headers:operatorHeaders(account.token),signal,timeoutMs:15000});
  const pixels=new Uint8Array(response.png);
  if(!doc.region||!response.document||!equals(DocumentSchema,response.document,doc)||pixels.length<33||pixels.length>32*1024*1024||!/^[0-9a-f]{64}$/.test(response.renderedSha256))throw new Error("Invalid rendered context");
  const header=new DataView(pixels.buffer);
  if(header.getUint32(0)!==0x89504e47||header.getUint32(4)!==0x0d0a1a0a||header.getUint32(8)!==13||header.getUint32(12)!==0x49484452||header.getUint32(16)!==doc.region.width||header.getUint32(20)!==doc.region.height)throw new Error("Invalid rendered dimensions");
  const digest=Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256",pixels)),byte=>byte.toString(16).padStart(2,"0")).join("");
  signal.throwIfAborted();
  if(digest!==response.renderedSha256||!this.current(account)||this.state.document!==doc||Number(doc.expiresAt?.seconds??0n)*1000<=Date.now())throw new Error("Rendered context unavailable");
  this.publish({preview:{url:URL.createObjectURL(new Blob([pixels],{type:"image/png"})),sha256:digest}});
 });
 cancel=()=>this.operation(async(account,signal)=>{
  this.hidePreview();
  const marker=this.state.marker;if(!marker)return;
  await this.client.cancelImport({requestId:marker.requestId},{headers:operatorHeaders(account.token),signal,timeoutMs:15000});
  signal.throwIfAborted();if(!this.current(account))throw new Error("Account changed");
  this.clear();
 });
 remove=()=>this.operation(async(account,signal)=>{
  this.hidePreview();
  const doc=this.state.document;if(!doc)throw new Error("Recover context first");
  await this.client.delete({id:doc.id},{headers:operatorHeaders(account.token),signal,timeoutMs:15000});
  signal.throwIfAborted();
  if(!this.current(account))throw new Error("Account changed");
  this.clear();
 });
 expire=()=>{const doc=this.state.document;if(doc&&(!this.account||this.account.expires<=Date.now()||Number(doc.expiresAt?.seconds??0n)*1000<=Date.now()))this.publish({document:undefined});};
 close=()=>{this.controller?.abort();this.hidePreview();};
}
