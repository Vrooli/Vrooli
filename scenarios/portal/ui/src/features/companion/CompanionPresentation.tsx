/* eslint-disable react-refresh/only-export-components */
import { ContextRetentionActions, ContextRetentionNotice } from "./ContextRetention";
import { createContext, useCallback, useContext, useEffect, useRef, useState, type PointerEvent, type ReactNode } from "react";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { useDialogKeyboard } from "../../hooks/useDialogKeyboard";

export type PresentationMode = "expanded" | "palette" | "pill" | "hidden";
type Shortcut = { accelerator: string; status: "disabled" | "registered" | "unavailable" };
type Activation = { status: "capturing" | "unavailable" } | { status: "ready"; contextId: string; expiresAt: number; hasImage?: boolean };
type Snapshot = { version: number; mode: PresentationMode; revision?: number; canHide?: boolean; shortcut?: Shortcut; activation?: Activation };
type ContextSource = {
  surface:{ownerScenario:string;surfaceId:string;target:{ownerScenario:string;resourceId:string;hostNodeId:string}};
  captureId:string;capturedAt:string;displayId:string;geometryRevision:string;
  bounds:{x:number;y:number;width:number;height:number};
};
type ContextImage = {contextId:string;expiresAt:number;mimeType:"image/png";dataUrl:string;sourceBounds:ContextSource["bounds"];source:ContextSource};
export type ContextDraft = {image:ContextImage;region:ImageRegion;strokes:ImagePoint[][]};

// This projection is provenance for an image, never desktop input authority.
export function contextImageSource(value:unknown, id:string, expiresAt:number, bounds:ContextSource["bounds"], now=Date.now()):ContextSource {
  if(!value || typeof value!=="object")throw new Error("invalid source");
  const source=value as Partial<ContextSource>;
  const surface=source.surface,target=surface?.target;
  const captured=typeof source.capturedAt==="string"?Date.parse(source.capturedAt):NaN;
  const fields=[surface?.ownerScenario,surface?.surfaceId,target?.ownerScenario,target?.resourceId,target?.hostNodeId];
  if(source.captureId!==id || !Number.isFinite(captured) || captured>now || expiresAt<=now || expiresAt>captured+30000 ||
    [source.displayId,source.geometryRevision].some(field=>typeof field!=="string" || !field || field.length>128) ||
    fields.some(field=>typeof field!=="string" || !field || field.length>256) || surface?.ownerScenario!=="device-control" ||
    !source.bounds || (["x","y","width","height"] as const).some(field=>source.bounds?.[field]!==bounds[field]))throw new Error("invalid source");
  if(!target || typeof source.displayId!=="string" || typeof source.geometryRevision!=="string")throw new Error("invalid source");
  return {surface:{ownerScenario:surface.ownerScenario,surfaceId:surface.surfaceId,target:{ownerScenario:target.ownerScenario,resourceId:target.resourceId,hostNodeId:target.hostNodeId}},
    captureId:id,capturedAt:new Date(captured).toISOString(),displayId:source.displayId,geometryRevision:source.geometryRevision,bounds:{...bounds}};
}
type Bridge = { readContextImage?: () => Promise<unknown>; dismissContext?: () => Promise<Snapshot>; setShortcut?: (accelerator: string) => Promise<Snapshot>; onQuit?: (listener: (id: number) => void) => () => void; decideQuit?: (id: number, decision: "quit" | "cancel" | "background") => Promise<unknown>; get: () => Promise<Snapshot>; set: (mode: PresentationMode) => Promise<Snapshot>; subscribe?: (listener: (snapshot: Snapshot) => void) => () => void };
declare global { interface Window { desktopPresentation?: Bridge } }
type Task = { checking?: boolean; kind: "chat" | "desktop"; stop: () => void | Promise<void> };
type State = {
  mode: PresentationMode; available: boolean; pending: boolean; failed: boolean;
  configureShortcut: (accelerator: string) => Promise<void>; canConfigureShortcut: boolean;
  change: (mode: PresentationMode) => Promise<void>;
  activation: Activation | undefined; dismissContext: () => Promise<void>; canDismissContext: boolean;
  shortcut: Shortcut | undefined; canHide: boolean;
  tasks: Map<string, Task>; register: (id: string, task: Task) => () => void;
};
const shortcutPattern = "(CommandOrControl|Control|Alt|Super)(\\+(Shift|Alt))?\\+(Space|[A-Z0-9])";
const Context = createContext<State | null>(null);
const valid = (value: unknown): value is Snapshot => {
  if (typeof value !== "object" || value === null || !("version" in value) || !("mode" in value)) return false;
  if ("revision" in value && (typeof value.revision !== "number" || !Number.isSafeInteger(value.revision) || value.revision < 0)) return false;
  if ("shortcut" in value) {
    const shortcut = value.shortcut;
    if (typeof shortcut !== "object" || shortcut === null || !("accelerator" in shortcut) || !("status" in shortcut) || typeof shortcut.accelerator !== "string" || !["disabled", "registered", "unavailable"].includes(String(shortcut.status))) return false;
  }
  if ("activation" in value) {
    const capture = value.activation;
    if (!capture || typeof capture !== "object" || !("status" in capture) || !["capturing", "unavailable", "ready"].includes(String(capture.status))) return false;
    if ("hasImage" in capture && typeof capture.hasImage !== "boolean") return false;
    if (capture.status === "ready" && (!("contextId" in capture) || typeof capture.contextId !== "string" || !/^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(capture.contextId) || capture.contextId === "00000000-0000-0000-0000-000000000000" || !("expiresAt" in capture) || typeof capture.expiresAt !== "number" || !Number.isSafeInteger(capture.expiresAt) || capture.expiresAt > Date.now()+30000)) return false;
  }
  return value.version === 1 && typeof value.mode === "string" && ["expanded", "palette", "pill", "hidden"].includes(value.mode);
};

export function CompanionPresentation({ children }: { children: ReactNode }) {
  const { t } = useTranslation();
  const taskRefs = useRef(new Map<string, Task>());
  const [quitRequest, setQuitRequest] = useState<number | null>(null);
  const [stopping, setStopping] = useState(false);
  const deciding = useRef(false);
  const cancelQuitButton = useRef<HTMLButtonElement>(null);
  const quitDialog = useRef<HTMLDivElement>(null);
  const [quitBusy, setQuitBusy] = useState(false);
  const [mode, setMode] = useState<PresentationMode>("expanded");
  const modeRef = useRef<PresentationMode>("expanded");
  const focusBeforePalette = useRef<HTMLElement | null>(null);
  const [available, setAvailable] = useState(false);
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);
  const [tasks, setTasks] = useState(new Map<string, Task>());
  const inFlight = useRef(false);
  const latestRevision = useRef(-1);
  const [canHide, setCanHide] = useState(false);
  const [activation, setActivation] = useState<Activation>();
  useEffect(() => {
    if (activation?.status !== "ready") return;
    const timer = window.setTimeout(() => setActivation(current => current === activation ? { status: "unavailable" } : current), Math.max(0, activation.expiresAt - Date.now()));
    return () => window.clearTimeout(timer);
  }, [activation]);
  const [shortcut, setShortcut] = useState<Shortcut>();
  const accept = useCallback((snapshot: Snapshot) => {
    if (!valid(snapshot)) return false;
    if (snapshot.revision !== undefined) {
      if (snapshot.revision < latestRevision.current) return true;
      latestRevision.current = snapshot.revision;
    }
    const capture = snapshot.activation;
    setActivation(capture?.status === "ready" && capture.expiresAt <= Date.now() ? { status: "unavailable" } : capture);
    const previousMode = modeRef.current;
    if (snapshot.mode === "palette" && previousMode !== "palette" && document.activeElement instanceof HTMLElement) {
      focusBeforePalette.current = document.activeElement;
    }
    modeRef.current = snapshot.mode;
    setMode(snapshot.mode); setShortcut(snapshot.shortcut); setCanHide(snapshot.canHide === true);
    if (snapshot.mode !== "palette" && previousMode === "palette") {
      queueMicrotask(() => {
        const previous = focusBeforePalette.current;
        if (previous?.isConnected) previous.focus();
        focusBeforePalette.current = null;
      });
    }
    return true;
  }, []);
  useEffect(() => {
    const bridge = window.desktopPresentation;
    if (!bridge) return;
    let disposed = false;
    const initialize = async () => {
      try {
        const snapshot = await bridge.get();
        if (!valid(snapshot)) throw new Error("unsupported presentation contract");
        if (!disposed) { accept(snapshot); setAvailable(true); setFailed(false); }
      } catch { if (!disposed) setFailed(true); }
    };
    const unsubscribe = bridge.subscribe?.(snapshot => {
      if (!disposed) { if (accept(snapshot)) setAvailable(true); else setFailed(true); }
    });
    void initialize();
    // The native owner admits calls after the initial document finishes loading.
    const loaded = () => { void initialize(); };
    window.addEventListener("load", loaded);
    return () => { disposed = true; unsubscribe?.(); window.removeEventListener("load", loaded); };
  }, [accept]);
  const change = useCallback(async (requested: PresentationMode) => {
    if (inFlight.current || !window.desktopPresentation) return;
    inFlight.current = true; setPending(true); setFailed(false);
    try {
      const snapshot = await window.desktopPresentation.set(requested);
      if (!valid(snapshot) || snapshot.mode !== requested) throw new Error("unconfirmed presentation");
      accept(snapshot);
    } catch {
      setFailed(true);
      // Read back after a lost reply; never resend a mode-changing operation.
      try {
        const snapshot = await window.desktopPresentation.get();
        if (valid(snapshot)) accept(snapshot);
      } catch { /* Keep the last confirmed mode and the visible error. */ }
    } finally { inFlight.current = false; setPending(false); }
  }, [accept]);
  const configureShortcut = useCallback(async (accelerator: string) => {
    const bridge = window.desktopPresentation;
    if (inFlight.current || !bridge?.setShortcut) return;
    inFlight.current = true; setPending(true); setFailed(false);
    try {
      const snapshot = await bridge.setShortcut(accelerator);
      if (!valid(snapshot) || snapshot.shortcut?.accelerator !== accelerator || snapshot.shortcut.status !== "registered") throw new Error("unconfirmed shortcut");
      accept(snapshot);
    } catch {
      setFailed(true);
      try { const snapshot = await bridge.get(); if (valid(snapshot)) accept(snapshot); } catch { /* Retain confirmed native state. */ }
    } finally { inFlight.current = false; setPending(false); }
  }, [accept]);
  const dismissContext = useCallback(async () => {
    const bridge = window.desktopPresentation;
    if (inFlight.current || !bridge?.dismissContext) return;
    inFlight.current = true; setPending(true); setFailed(false);
    setActivation({status:"unavailable"});
    try {
      const snapshot = await bridge.dismissContext();
      if (!valid(snapshot) || snapshot.activation?.status !== "unavailable") throw new Error("unconfirmed context dismissal");
      accept(snapshot);
    } catch {
      setFailed(true);
      try { const snapshot = await bridge.get(); if (valid(snapshot)) accept(snapshot); } catch { /* Keep the last confirmed reference; never retry dismissal. */ }
    } finally { inFlight.current = false; setPending(false); }
  }, [accept]);
  const register = useCallback((id: string, task: Task) => {
    taskRefs.current.set(id, task); setTasks(new Map(taskRefs.current));
    return () => {
      if (taskRefs.current.get(id) !== task) return;
      taskRefs.current.delete(id); setTasks(new Map(taskRefs.current));
    };
  }, []);
  const decideQuit = useCallback(async (decision: "quit" | "cancel" | "background") => {
    if (quitRequest === null || deciding.current || !window.desktopPresentation?.decideQuit) return;
    if (decision === "quit" && taskRefs.current.size !== 0) return;
    deciding.current = true; setQuitBusy(true);
    try {
      await window.desktopPresentation.decideQuit(quitRequest, decision);
      setQuitRequest(current => current === quitRequest ? null : current); setStopping(false);
    } catch { setFailed(true); setQuitRequest(current => current === quitRequest ? null : current); }
    finally { deciding.current = false; setQuitBusy(false); }
  }, [quitRequest]);
  useEffect(() => window.desktopPresentation?.onQuit?.(id => {
    if (Number.isSafeInteger(id) && id > 0) { setQuitRequest(id); setStopping(false); }
  }), []);
  useEffect(() => {
    if (!quitBusy && quitRequest !== null && tasks.size === 0) void decideQuit("quit");
  }, [quitRequest, tasks, decideQuit, quitBusy]);
  const cancelQuit = useCallback(() => { void decideQuit("cancel"); }, [decideQuit]);
  useDialogKeyboard(quitDialog, cancelQuitButton, quitRequest !== null, cancelQuit);
  const stopTasks = async () => {
    if (stopping) return;
    setStopping(true);
    await Promise.allSettled([...taskRefs.current.values()].map(task => Promise.resolve().then(task.stop)));
    setStopping(false);
    // Only registry removal, not callback completion, permits Quit.
  };
  return <Context.Provider value={{ mode, available, pending, failed, change, activation, dismissContext, canDismissContext: !!window.desktopPresentation?.dismissContext, configureShortcut, canConfigureShortcut: available && shortcut?.status !== "disabled" && !!window.desktopPresentation?.setShortcut, shortcut, canHide, tasks, register }}>
    <div data-companion-mode={available ? mode : "expanded"} className="companion-root">{children}</div>
    {quitRequest !== null && tasks.size > 0 && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div ref={quitDialog} role="dialog" tabIndex={-1} aria-modal="true" aria-labelledby="companion-quit-title" className="max-w-lg rounded border border-app-border bg-app-surface p-4 text-app-foreground">
        <h2 id="companion-quit-title" className="text-lg font-semibold">{t(strings.companion.quitTitle)}</h2>
        <p>{t(strings.companion.quitTasks)}</p>
        <div className="my-3 flex flex-wrap gap-2">{[...tasks].map(([id, task]) => <button key={id} type="button" className="rounded border border-app-border p-2" onClick={() => { void Promise.resolve().then(task.stop).catch(() => setFailed(true)); }}>{t(task.checking ? strings.companion.checkStatus : task.kind === "chat" ? strings.companion.stopChat : strings.companion.stopDesktop)}</button>)}</div>
        <div className="flex flex-wrap gap-2">
          <button type="button" ref={cancelQuitButton} className="rounded border border-app-border p-2" onClick={() => void decideQuit("cancel")}>{t(strings.companion.cancelQuit)}</button>
          <button type="button" disabled={!canHide} className="rounded border border-app-border p-2" onClick={() => void decideQuit("background")}>{t(strings.companion.keepRunning)}</button>
          <button type="button" disabled={stopping} className="rounded border border-app-border p-2" onClick={() => void stopTasks()}>{t(strings.companion.stopAndQuit)}</button>
        </div>
      </div>
    </div>}
  </Context.Provider>;
}

export function useCompanionStop(id: string, kind: Task["kind"], active: boolean, stop: Task["stop"], checking = false) {
  const register = useContext(Context)?.register;
  const latest = useRef(stop); latest.current = stop;
  useEffect(() => {
    if (!active || !register) return;
    return register(id, { kind, checking, stop: () => latest.current() });
  }, [active, id, kind, register, checking]);
}

type ImagePoint = {x:number;y:number};
type ImageRegion = {x:number;y:number;width:number;height:number};
// DOM coordinates are CSS pixels; the frozen image has its own pixel axes.
// Desktop origins and subsequent native window movement do not enter this map.
export function contextImagePoint(x:number,y:number,rect:{left:number;top:number;width:number;height:number},source:{width:number;height:number}): ImagePoint | undefined {
  if(![x,y,rect.left,rect.top,rect.width,rect.height,source.width,source.height].every(Number.isFinite) || rect.width<=0 || rect.height<=0 || source.width<=0 || source.height<=0)return;
  return {x:Math.max(0,Math.min(source.width,(x-rect.left)*source.width/rect.width)),y:Math.max(0,Math.min(source.height,(y-rect.top)*source.height/rect.height))};
}

function ContextImageEditor({value,onChange,onError}: {value:ContextDraft;onChange:(value:ContextDraft)=>void;onError:()=>void}) {
  const {image:preview,region,strokes}=value;
  const setRegion=(region:ImageRegion)=>onChange({...value,region});
  const setStrokes=(strokes:ImagePoint[][])=>onChange({...value,strokes});
  const {t}=useTranslation();
  const bounds=preview.sourceBounds;
  const [tool,setTool]=useState<"region"|"draw">("region");
  const [draft,setDraft]=useState<ImagePoint[]>([]);
  const gesture=useRef<{id:number;points:ImagePoint[]}>();
  const cancel=()=>{gesture.current=undefined;setDraft([]);};
  const finish=()=>{
    const points=gesture.current?.points;cancel();
    if(!points || points.length<2)return;
    const first=points[0],last=points[points.length-1];
    if(!first || !last)return;
    if(tool==="draw"){setStrokes(strokes.length<32?[...strokes,points]:strokes);return;}
    const x=Math.floor(Math.min(first.x,last.x)),y=Math.floor(Math.min(first.y,last.y));
    const width=Math.ceil(Math.max(first.x,last.x))-x,height=Math.ceil(Math.max(first.y,last.y))-y;
    if(width>0 && height>0)setRegion({x,y,width,height});
  };
  const point=(event:PointerEvent<SVGSVGElement>)=>contextImagePoint(event.clientX,event.clientY,event.currentTarget.getBoundingClientRect(),bounds);
  return <>
    <div className="flex gap-2">
      <button type="button" aria-pressed={tool==="region"} onClick={()=>{cancel();setTool("region");}}>{t(strings.companion.selectRegion)}</button>
      <button type="button" aria-pressed={tool==="draw"} disabled={strokes.length>=32} onClick={()=>{cancel();setTool("draw");}}>{t(strings.companion.drawAnnotation)}</button>
      <button type="button" onClick={()=>{cancel();onChange({...value,strokes:[],region:{x:0,y:0,width:bounds.width,height:bounds.height}});}}>{t(strings.companion.resetAnnotations)}</button>
    </div>
    <div className="relative inline-block max-w-full align-top leading-none">
      <img src={preview.dataUrl} alt={t(strings.companion.previewAlt)} draggable={false} className="max-h-64 max-w-full" onError={onError}/>
      <svg data-testid="context-image-overlay" role="img" aria-label={t(strings.companion.annotationOverlay)} className="absolute inset-0 h-full w-full touch-none" viewBox={`0 0 ${bounds.width} ${bounds.height}`} preserveAspectRatio="none"
        onPointerDown={event=>{if(event.button!==0 || gesture.current)return;const p=point(event);if(!p)return;event.currentTarget.setPointerCapture(event.pointerId);gesture.current={id:event.pointerId,points:[p]};setDraft([p]);}}
        onPointerMove={event=>{const active=gesture.current;if(!active || active.id!==event.pointerId || active.points.length>=256)return;const p=point(event);if(!p)return;active.points=tool==="region"?[active.points[0] ?? p,p]:[...active.points,p];setDraft(active.points);}}
        onPointerUp={event=>{if(gesture.current?.id!==event.pointerId)return;const p=point(event);if(p && gesture.current.points.length<256)gesture.current.points.push(p);finish();}}
        onPointerCancel={cancel} onLostPointerCapture={cancel}>
        <rect data-testid="context-image-region" {...region} fill="rgba(59,130,246,0.08)" stroke="#2563eb" strokeWidth="2" vectorEffect="non-scaling-stroke"/>
        {[...strokes,...(draft.length?[draft]:[])].map((points,index)=><polyline key={index} data-testid="context-image-mark" points={points.map(p=>`${p.x},${p.y}`).join(" ")} fill="none" stroke="#dc2626" strokeWidth="2"/>)}
      </svg>
    </div>
    <fieldset className="flex flex-wrap gap-2 text-xs"><legend>{t(strings.companion.regionCoordinates)}</legend>
      {(["x","y","width","height"] as const).map(field=>{
        const fieldLabel = field === "x" ? strings.companion.x : field === "y" ? strings.companion.y : field === "width" ? strings.companion.width : strings.companion.height;
        return <label key={field}>{t(fieldLabel)}<input className="ml-1 w-20 rounded border border-app-border bg-app-background p-1" type="number" step="1" min={field==="x"||field==="y"?0:1} max={field==="x"?bounds.width-1:field==="y"?bounds.height-1:field==="width"?bounds.width-region.x:bounds.height-region.y} value={region[field]} onChange={event=>{
        const value=event.currentTarget.valueAsNumber;if(!Number.isInteger(value))return;
        const next={...region,[field]:value};if(next.x<0||next.y<0||next.width<1||next.height<1||next.x+next.width>bounds.width||next.y+next.height>bounds.height)return;
        cancel();setRegion(next);
        }}/></label>;
      })}
    </fieldset>
  </>;
}

// Mount per exact reference. Pixels stay in transient component state and never
// enter draft text, local storage, or a request without an attachment operation.
function ContextImagePreview({activation,hidden}: {hidden:boolean;activation: Extract<Activation,{status:"ready"}>}) {
  const {t}=useTranslation();
  const [draft,setDraft]=useState<ContextDraft>();
  const [busy,setBusy]=useState(false);
  const [failed,setFailed]=useState(false);
  const alive=useRef(true), reading=useRef(false);
  useEffect(()=>{alive.current=true;return()=>{alive.current=false;};},[]);
  const read=async()=>{
    if(reading.current || !window.desktopPresentation?.readContextImage || activation.expiresAt<=Date.now())return;
    reading.current=true;setBusy(true);setFailed(false);
    try {
      const value=await window.desktopPresentation.readContextImage();
      if(!alive.current)return;
      if(!value || typeof value!=="object")throw new Error("invalid preview");
      const image=value as Partial<Omit<ContextImage,"mimeType">> & {mimeType?:unknown}, bounds=image.sourceBounds;
      if(image.contextId!==activation.contextId || image.expiresAt!==activation.expiresAt || image.expiresAt<=Date.now() || image.mimeType!=="image/png" || typeof image.dataUrl!=="string" || image.dataUrl.length>44739266 || !image.dataUrl.startsWith("data:image/png;base64,") ||
        !bounds || !Number.isInteger(bounds.width) || !Number.isInteger(bounds.height) || bounds.width<=0 || bounds.height<=0 || bounds.width>65535 || bounds.height>65535 || bounds.width*bounds.height>16*1024*1024 || !Number.isInteger(bounds.x) || !Number.isInteger(bounds.y) || bounds.x < -2147483648 || bounds.y < -2147483648 || bounds.x+bounds.width>2147483647 || bounds.y+bounds.height>2147483647)throw new Error("invalid preview");
      const source=contextImageSource(image.source,image.contextId,image.expiresAt,bounds);
      setDraft({image:{contextId:image.contextId,expiresAt:image.expiresAt,mimeType:"image/png",dataUrl:image.dataUrl,sourceBounds:{...bounds},source},
        region:{x:0,y:0,width:bounds.width,height:bounds.height},strokes:[]});
    } catch {if(alive.current){setDraft(undefined);setFailed(true);}}
    finally {reading.current=false;if(alive.current)setBusy(false);}
  };
  return <div className="w-full" hidden={hidden} aria-busy={busy}>
    {!draft && <button type="button" disabled={busy} className="rounded border border-app-border px-2 py-1 text-sm" onClick={()=>void read()}>{t(busy?strings.companion.previewLoading:strings.companion.previewContext)}</button>}
    {draft && <figure className="max-w-xl space-y-2">
      <ContextImageEditor value={draft} onChange={setDraft} onError={()=>{setDraft(undefined);setFailed(true);}} />
      <ContextRetentionActions draft={draft}/>
      <ContextRetentionNotice/>
      <button type="button" className="rounded border border-app-border px-2 py-1 text-sm" onClick={()=>setDraft(undefined)}>{t(strings.companion.closePreview)}</button>
    </figure>}
    {failed && <p role="alert" className="text-xs">{t(strings.companion.previewFailed)}</p>}
  </div>;
}

export function CompanionToolbar() {
  const state = useContext(Context);
  const { t } = useTranslation();
  const [shortcutDraft, setShortcutDraft] = useState("CommandOrControl+Alt+Space");
  if (!state || (!state.available && !state.failed)) return null;
  return <div className="companion-toolbar flex flex-wrap items-center gap-2 border-b border-app-border bg-app-surface p-2">
    <label className="sr-only" htmlFor="companion-presentation">{t(strings.companion.presentation)}</label>
    <select id="companion-presentation" title={state.shortcut?.accelerator} className="rounded border border-app-border bg-app-background p-1 text-sm" value={state.mode} disabled={!state.available} aria-disabled={!state.available || state.pending} aria-busy={state.pending} onKeyDown={event => { if (state.pending) event.preventDefault(); }} onChange={event => void state.change(event.target.value as PresentationMode)}>
      <option value="expanded">{t(strings.companion.expanded)}</option>
      <option value="palette">{t(strings.companion.palette)}</option>
      <option value="pill">{t(strings.companion.pill)}</option>
      {state.canHide && <option value="hidden">{t(strings.companion.hidden)}</option>}
    </select>
    {[...state.tasks].map(([id, task]) => <button key={id} type="button" className="rounded border border-app-border px-2 py-1 text-sm" onClick={() => { void task.stop(); }}>{t(task.checking ? strings.companion.checkStatus : task.kind === "chat" ? strings.companion.stopChat : strings.companion.stopDesktop)}</button>)}
    {state.shortcut?.status === "registered" && <span className={state.mode === "pill" ? "sr-only" : "text-xs"}>{t(strings.companion.shortcut, { shortcut: state.shortcut.accelerator })}</span>}
    {state.shortcut?.status === "unavailable" && <p role="status" className="text-xs">{t(strings.companion.shortcutUnavailable, { shortcut: state.shortcut.accelerator })}</p>}
    {state.canConfigureShortcut && state.mode !== "pill" && <form className="flex items-center gap-2" onSubmit={event => { event.preventDefault(); void state.configureShortcut(shortcutDraft); }}>
      <label htmlFor="companion-shortcut" className="text-xs">{t(strings.companion.shortcutSetting)}</label>
      <input id="companion-shortcut" className="rounded border border-app-border bg-app-background p-1 text-sm" value={shortcutDraft} maxLength={64} pattern={shortcutPattern} onChange={event => setShortcutDraft(event.target.value)} required />
      <button type="submit" disabled={state.pending} className="rounded border border-app-border p-1 text-sm">{t(strings.companion.applyShortcut)}</button>
    </form>}
    {state.activation && <p role="status" className="text-xs" data-testid="companion-context-status">{t(state.activation.status === "ready" ? strings.companion.contextReady : state.activation.status === "capturing" ? strings.companion.contextCapturing : strings.companion.contextUnavailable)}</p>}
    {state.canDismissContext && state.activation && state.activation.status !== "unavailable" && <button type="button" disabled={state.pending} className="rounded border border-app-border px-2 py-1 text-sm" onClick={() => void state.dismissContext()}>{t(strings.companion.dismissContext)}</button>}
    {state.activation?.status === "ready" && state.activation.hasImage && window.desktopPresentation?.readContextImage && <ContextImagePreview hidden={state.mode === "pill"} key={`${state.activation.contextId}:${state.activation.expiresAt}`} activation={state.activation} />}
    {state.failed && <p role="alert" className="text-xs">{t(strings.companion.failed)}</p>}
  </div>;
}
