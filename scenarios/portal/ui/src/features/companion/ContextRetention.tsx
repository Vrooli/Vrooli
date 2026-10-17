/* eslint-disable react-refresh/only-export-components */
import { createContext, useContext, useEffect, useState, useSyncExternalStore, type ReactNode } from "react";
import { ContextRetention, contextRetentionKey, type ContextAccount, type ContextRetentionSnapshot } from "./contextRetentionStore";
import type { ContextDraft } from "./CompanionPresentation";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
const Context=createContext<ContextRetention|null>(null);
export function ContextRetentionProvider({account,children}:{account:ContextAccount|undefined;children:ReactNode}){
 const [store]=useState(()=>new ContextRetention());
 useEffect(()=>{store.setAccount(account);},[store,account]);
 useEffect(()=>{const changed=(event:StorageEvent)=>{if(event.key===contextRetentionKey&&!store.getSnapshot().busy)store.synchronize();};window.addEventListener("storage",changed);const timer=setInterval(store.expire,500);return()=>{window.removeEventListener("storage",changed);clearInterval(timer);store.close();};},[store]);
 return <Context.Provider value={store}>{children}</Context.Provider>;
}
function Actions({store,draft}:{store:ContextRetention;draft?:ContextDraft}){
 const state=useSyncExternalStore(store.subscribe,store.getSnapshot,store.getSnapshot);
 const {t}=useTranslation();
 return <div className="space-y-1 text-sm" aria-busy={state.busy}>
  {draft&&!state.marker&&<button type="button" disabled={state.busy} onClick={()=>void store.retain(draft)}>{t(strings.companion.retainContext)}</button>}
  {!draft&&state.marker&&<>
   <p>{t(state.document?strings.companion.contextRetained:strings.companion.contextPending)}</p>
   <button type="button" disabled={state.busy} onClick={()=>void store.recover()}>{t(strings.companion.recoverContext)}</button>
   {!state.document&&<button type="button" disabled={state.busy} onClick={()=>void store.cancel()}>{t(strings.companion.cancelContextImport)}</button>}
   {state.document&&<button type="button" disabled={state.busy} onClick={()=>void store.render()}>{t(strings.companion.previewRenderedContext)}</button>}
   {state.document&&!state.attachedDocumentId&&<button type="button" disabled={state.busy} onClick={store.attach}>{t(strings.companion.attachContext)}</button>}
   {state.document&&state.attachedDocumentId&&<><p>{t(strings.companion.contextAttached)}</p><button type="button" disabled={state.busy} onClick={store.detach}>{t(strings.companion.detachContext)}</button></>}
   {state.preview&&<figure><img src={state.preview.url} alt={t(strings.companion.renderedContextAlt)} className="max-h-64 max-w-full" onError={store.previewFailed}/><button type="button" onClick={store.hidePreview}>{t(strings.companion.closePreview)}</button></figure>}
   {state.document&&<button type="button" disabled={state.busy} onClick={()=>void store.remove()}>{t(strings.companion.deleteRetainedContext)}</button>}
  </>}
  {state.failed&&<p role="alert">{t(strings.companion.retainContextFailed)}</p>}
 </div>;
}
export function ContextRetentionActions({draft}:{draft?:ContextDraft}){const store=useContext(Context);return store?<Actions store={store} draft={draft}/>:null;}

export function useContextRetention(){
 const store=useContext(Context);
 const subscribe=store?.subscribe??(()=>()=>{});
 const getSnapshot=store?.getSnapshot??(()=>undefined as ContextRetentionSnapshot|undefined);
 const state=useSyncExternalStore(subscribe,getSnapshot,getSnapshot);
 return store?{state,attach:store.attach,detach:store.detach}:undefined;
}

function Notice({store}:{store:ContextRetention}){
 const state=useSyncExternalStore(store.subscribe,store.getSnapshot,store.getSnapshot),{t}=useTranslation();
 return <figcaption className="text-xs">{t(state.document?strings.companion.contextRetained:state.marker?strings.companion.contextPending:strings.companion.previewLocal)}</figcaption>;
}
export function ContextRetentionNotice(){const store=useContext(Context),{t}=useTranslation();return store?<Notice store={store}/>:<figcaption className="text-xs">{t(strings.companion.previewLocal)}</figcaption>;}
