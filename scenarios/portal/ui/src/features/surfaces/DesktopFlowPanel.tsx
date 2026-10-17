import { useEffect, useRef, useState } from "react";
import { create } from "@bufbuild/protobuf";
import type { SessionRef } from "@vrooli/proto-types/common/v1/surface_pb";
// Flow is declared in the shared device-control namespace.  The flows service
// file contains request/response envelopes, but does not export FlowSchema;
// importing it from there leaves an undefined schema at runtime and causes
// draft execution to fail before the request reaches Device Control.
import { FlowSchema } from "@vrooli/proto-types/device-control/v1/shared/flow_pb";
import type { FlowRecord, OwnerRunFlowRequest, OwnerRunSavedFlowRequest, SavedDesktopFlow } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { desktopClient, operatorHeaders } from "../../api/desktop";
import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

type Step = { id: string; kind: "desktop-text" | "desktop-text-assert"; window: string; field: string; text: string; position: number };
type Pending = { kind: "draft"; request: Omit<OwnerRunFlowRequest, "$typeName"> } | { kind: "saved"; request: Omit<OwnerRunSavedFlowRequest, "$typeName"> };
type Props = { token: string; session?: SessionRef; control: boolean; applicationId: string; applicationRevision: string; applicationFresh: boolean; windowName: string; fieldName: string; busy: boolean; blocked: boolean; onUncertain: (uncertain: boolean) => void; onBusy: (busy: boolean) => void };
const inputClass = "block w-full rounded border border-app-border bg-app-background p-2 text-app-foreground";

export function DesktopFlowPanel(props: Props) {
 const { t } = useTranslation();
 const [steps, setSteps] = useState<Step[]>([]);
 const [contextKey, setContextKey] = useState("");
 const [savedId, setSavedId] = useState("");
 const [version, setVersion] = useState(1);
 const [saved, setSaved] = useState<SavedDesktopFlow>();
 const [pending, setPending] = useState<Pending>();
 const [result, setResult] = useState<FlowRecord>();
 const [verified, setVerified] = useState<{ session: SessionRef; runId: string }>();
 const [failed, setFailed] = useState(false);
 const operation = useRef<AbortController>();
 const currentSession = useRef(props.session?.sessionId);
 currentSession.current = props.session?.sessionId;
 useEffect(() => () => operation.current?.abort(), []);
 useEffect(() => { operation.current?.abort(); setPending(undefined); setResult(undefined); }, [props.session?.sessionId]);
 const admitted = Boolean(props.session && props.control);
 const runnable = admitted && !props.blocked && props.applicationFresh && Boolean(props.applicationId && props.applicationRevision);
 const uncertain = Boolean(pending && result?.disposition !== "passed");
 function change(next: Step[]) { setSteps(next); setVerified(undefined); setResult(undefined); setPending(undefined); }
 function add(kind: Step["kind"]) { change([...steps, { id: crypto.randomUUID(), kind, window: props.windowName, field: props.fieldName, text: "", position: 0 }]); }
 function edit(index: number, patch: Partial<Step>) { change(steps.map((step, i) => i === index ? { ...step, ...patch } : step)); }
 async function execute(existing?: Pending) {
  if (!props.session || !admitted || props.busy || operation.current || (!(existing && existing === pending) && !runnable)) return;
  const job = existing ?? { kind: "draft" as const, request: { session: props.session, runId: crypto.randomUUID(), applicationId: props.applicationId, applicationRevision: props.applicationRevision, flow: create(FlowSchema, { transport: "desktop", requireUnlocked: true, steps: steps.map(step => ({ id: step.id, kind: step.kind, target: step.field, arguments: { window: step.window, text: step.text, ...(step.kind === "desktop-text" ? { position: step.position } : {}) } })) }) } };
  const controller = new AbortController(); operation.current = controller;
  props.onBusy(true); props.onUncertain(true); setFailed(false); setPending(job); setResult(undefined); setVerified(undefined);
  try {
   const options = { headers: operatorHeaders(props.token), signal: controller.signal, timeoutMs: 35000 };
   const record = job.kind === "draft" ? await desktopClient.runFlow(job.request, options) : await desktopClient.runSavedFlow(job.request, options);
   if (controller.signal.aborted || currentSession.current !== job.request.session?.sessionId) return;
   if (record.runId !== job.request.runId) throw new Error("run identity changed");
   setResult(record); props.onUncertain(record.disposition !== "passed");
   if (record.disposition === "passed" && job.kind === "draft" && record.confirmed === job.request.flow?.steps.length && job.request.flow.steps.at(-1)?.kind === "desktop-text-assert") setVerified({ session: props.session, runId: record.runId });
  } catch { if (!controller.signal.aborted) setFailed(true); }
  finally { if (operation.current === controller) { operation.current = undefined; props.onBusy(false); } }
 }
 async function library(promote: boolean) {
  if (!props.session || props.busy || operation.current || !contextKey.trim() || (promote && (!verified || !admitted))) return;
  const controller = new AbortController(); operation.current = controller;
  const session = props.session; props.onBusy(true); setFailed(false);
  try {
   const options = { headers: operatorHeaders(props.token), signal: controller.signal };
   const value = promote && verified ? await desktopClient.promoteFlow({ session, sourceSession: verified.session, sourceRunId: verified.runId, contextKey: contextKey.trim() }, options) : await desktopClient.getSavedFlow({ session, id: savedId, version, contextKey: contextKey.trim() }, options);
   if (controller.signal.aborted || currentSession.current !== session.sessionId) return;
   setSaved(value); setSavedId(value.id); setVersion(value.version);
  } catch { if (!controller.signal.aborted) setFailed(true); }
  finally { if (operation.current === controller) { operation.current = undefined; props.onBusy(false); } }
 }
 const valid = steps.length > 0 && steps.every(step => step.window.trim() && step.field.trim() && (step.kind === "desktop-text-assert" || step.text) && Number.isInteger(step.position) && step.position >= 0 && step.position <= 2147483647);
 return <details className="space-y-3 border-t border-app-border pt-3">
  <summary>{t(strings.desktopFlows.title)}</summary>
  <p>{t(strings.desktopFlows.help)}</p>
  {failed && <p role="alert">{t(strings.desktopFlows.failed)}</p>}
  {steps.map((step, index) => <fieldset key={step.id} className="space-y-2 rounded border border-app-border p-2" disabled={props.busy || uncertain}>
   <legend>{index + 1}. {t(step.kind === "desktop-text" ? strings.desktopFlows.insert : strings.desktopFlows.assert)}</legend>
   <label>{t(strings.desktopFlows.window)}<input className={inputClass} value={step.window} maxLength={4096} onChange={event => edit(index, { window: event.target.value })} /></label>
   <label>{t(strings.desktopFlows.field)}<input className={inputClass} value={step.field} maxLength={4096} onChange={event => edit(index, { field: event.target.value })} /></label>
   <label>{t(step.kind === "desktop-text" ? strings.desktopFlows.text : strings.desktopFlows.expected)}<textarea className={inputClass} value={step.text} maxLength={4096} onChange={event => edit(index, { text: event.target.value })} /></label>
   {step.kind === "desktop-text" && <label>{t(strings.desktopFlows.position)}<input className={inputClass} type="number" min={0} max={2147483647} value={step.position} onChange={event => edit(index, { position: Number(event.target.value) })} /></label>}
   <Button type="button" variant="outline" onClick={() => change(steps.filter((_, i) => i !== index))}>{t(strings.desktopFlows.remove)}</Button>
  </fieldset>)}
  <div className="flex flex-wrap gap-2">
   <Button type="button" disabled={props.busy || uncertain || steps.length >= 32} onClick={() => add("desktop-text")}>{t(strings.desktopFlows.addInsert)}</Button>
   <Button type="button" disabled={props.busy || uncertain || steps.length >= 32} onClick={() => add("desktop-text-assert")}>{t(strings.desktopFlows.addAssert)}</Button>
   <Button type="button" disabled={props.busy || uncertain || !runnable || !valid} onClick={() => void execute()}>{t(strings.desktopFlows.run)}</Button>
   {uncertain && <Button type="button" disabled={props.busy || !admitted} onClick={() => void execute(pending)}>{t(strings.desktopFlows.check)}</Button>}
  </div>
  {pending && <p role="status">{t(result?.disposition === "passed" ? strings.desktopFlows.passed : strings.desktopFlows.uncertain, { count: result?.confirmed ?? 0 })}</p>}
  <label>{t(strings.desktopFlows.context)}<input className={inputClass} value={contextKey} maxLength={256} disabled={props.busy} onChange={event => { setContextKey(event.target.value); setSaved(undefined); }} /></label>
  <Button type="button" disabled={props.busy || !admitted || !verified || !contextKey.trim()} onClick={() => void library(true)}>{t(strings.desktopFlows.save)}</Button>
  <label>{t(strings.desktopFlows.savedId)}<input className={inputClass} value={savedId} disabled={props.busy} onChange={event => { setSavedId(event.target.value); setSaved(undefined); }} /></label>
  <label>{t(strings.desktopFlows.version)}<input className={inputClass} type="number" min={1} value={version} disabled={props.busy} onChange={event => { setVersion(Number(event.target.value)); setSaved(undefined); }} /></label>
  <Button type="button" disabled={props.busy || !props.session || !savedId || !Number.isInteger(version) || version < 1 || !contextKey.trim()} onClick={() => void library(false)}>{t(strings.desktopFlows.load)}</Button>
  {saved && <>
   <p>{t(strings.desktopFlows.loaded, { id: saved.id, version: saved.version })}</p>
   <ol className="list-decimal pl-5">{saved.flow?.steps.map(step => <li key={step.id}>{step.kind === "desktop-text-assert" ? t(strings.desktopFlows.assert) : t(strings.desktopFlows.insert)}: {step.target}<p className="whitespace-pre-wrap">{t(strings.desktopFlows.window)}: {(typeof step.arguments?.window === "string" ? step.arguments.window : "")}<br />{t(step.kind === "desktop-text-assert" ? strings.desktopFlows.expected : strings.desktopFlows.text)}: {(typeof step.arguments?.text === "string" ? step.arguments.text : "")}</p></li>)}</ol>
   <Button type="button" disabled={props.busy || uncertain || !runnable} onClick={() => props.session && void execute({ kind: "saved", request: { session: props.session, id: saved.id, version: saved.version, contextKey: saved.contextKey, runId: crypto.randomUUID(), applicationId: props.applicationId, applicationRevision: props.applicationRevision } })}>{t(strings.desktopFlows.replay)}</Button>
  </>}
 </details>;
}
