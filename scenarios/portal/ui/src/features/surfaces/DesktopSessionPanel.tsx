import { useDesktopSession, type DesktopAccount as Account, type DesktopSession as Active } from "./useDesktopSession";
import { useEffect, useRef, useState } from "react";
import type { SurfaceRef } from "@vrooli/proto-types/common/v1/surface_pb";
import type { ObserveResponse, ApplicationsResponse } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { PointerAction_Kind, PointerAction_Button } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { desktopClient, operatorClient, operatorHeaders } from "../../api/desktop";
import { Button } from "../../components/ui/button";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

import { DesktopFlowPanel } from "./DesktopFlowPanel";

const fieldClass = "block w-full rounded border border-app-border bg-app-background p-2 text-app-foreground";


export function DesktopSessionPanel({ surface, onActive }: { surface?: SurfaceRef; onActive: (active: boolean) => void }) {
  const { t } = useTranslation();
  const { account, active, uncertain, stopping, stopRequested, opening, recoveryRequired, recovering, needsAuthentication, store } = useDesktopSession();
  const setAccount = store.setAccount;
  const setUncertain = (value: boolean) => store.setUncertain(value, active?.ref);
  const [frame, setFrame] = useState<ObserveResponse>();
  const [imageURL, setImageURL] = useState<string>();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [register, setRegister] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  const [applications, setApplications] = useState<ApplicationsResponse>();
  const [applicationId, setApplicationId] = useState("");
  const [windowId, setWindowId] = useState("");
  const [elementId, setElementId] = useState("");
  const [textDraft, setTextDraft] = useState("");
  const [position, setPosition] = useState(0);
  const [now, setNow] = useState(Date.now);
  const windows = frame?.semantic?.elements.filter(item => item.windowId === item.elementId && item.elementId !== "") ?? [];
  const selectedWindow = windows.find(item => item.elementId === windowId);
  const fields = selectedWindow ? frame?.semantic?.elements.filter(item => item.editable && item.windowId === selectedWindow.elementId) ?? [] : [];
  const semanticFresh = Boolean(frame?.semantic?.expiresAt && Number(frame.semantic.expiresAt.seconds) * 1000 > now);
  const applicationsFresh = Boolean(applications?.expiresAt && Number(applications.expiresAt.seconds) * 1000 > now);
  const selectedApplication = applications?.applications.find(item => item.applicationId === applicationId);
  const mounted = useRef(true);
  const inputPending = useRef(false);

  useEffect(() => { onActive(Boolean(active) || busy || opening || recoveryRequired); return () => onActive(false); }, [active, busy, opening, recoveryRequired, onActive]);
  useEffect(() => { mounted.current = true; return () => {
    mounted.current = false;
  }; }, []);
  useEffect(() => {
    if (!frame) { setImageURL(undefined); return; }
    const url = URL.createObjectURL(new Blob([new Uint8Array(frame.png)], { type: "image/png" }));
    setImageURL(url);
    return () => URL.revokeObjectURL(url);
  }, [frame]);
  useEffect(() => {
    const timer = setInterval(() => {
      setNow(Date.now());
    }, 500);
    return () => clearInterval(timer);
  }, []);

  async function signIn(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setError(false);
    try {
      const result = register ? await operatorClient.register({ email: email.trim(), password }) : await operatorClient.login({ email: email.trim(), password });
      const tokens = result.tokens;
      if (!tokens?.accessToken || !tokens.accessTokenExpiresAt) throw new Error("missing account session");
      setAccount({ id: result.account?.id ?? "", realm: result.account?.realm ?? "", token: tokens.accessToken, refresh: tokens.refreshToken, expires: Number(tokens.accessTokenExpiresAt.seconds) * 1000, email: result.account?.email ?? email.trim() });
    } catch { setError(true); }
    finally { setPassword(""); setBusy(false); }
  }
  useEffect(() => { setElementId(""); setWindowId(""); }, [frame?.semantic?.revision]);
  useEffect(() => { if (!active) { setTextDraft(""); setElementId(""); setApplicationId(""); setApplications(undefined); } }, [active]);

  async function observe(session: Active, identity: Account, application?: { applicationId: string; applicationRevision: string }) {
    const stillActive = () => mounted.current && store.canObserve(session.ref);
    if (!stillActive()) return;
    const snapshot = await desktopClient.observe({ session: session.ref, ...application }, { headers: operatorHeaders(identity.token) });
    if (!stillActive()) return;
    setFrame(snapshot); setUncertain(false);
  }
  async function open(control: boolean) {
    if (!surface || !account || active) return;
    setBusy(true); setError(false);
    try {
      const session = await store.open(surface, control);
      await observe(session, account);
    } catch { setError(true); }
    finally { setBusy(false); }
  }
  async function refreshFrame() {
    if (!account || !active) return;
    setBusy(true); setError(false);
    try { await observe(active, account); } catch { setError(true); }
    finally { setBusy(false); }
  }
  async function stop() {
    setError(false);
    if (!await store.stop()) setError(true);
  }
  useEffect(() => {
    if (!active || stopRequested || needsAuthentication) { setFrame(undefined); setApplications(undefined); }
  }, [active, stopRequested, needsAuthentication]);

  async function click(event: React.MouseEvent<HTMLButtonElement>) {
    if (!account || !active?.control || !store.canObserve(active.ref) || !frame || busy || inputPending.current || uncertain || active.expires <= Date.now()) return;
    const box = event.currentTarget.getBoundingClientRect();
    if (!box.width || !box.height) return;
    const x = event.detail === 0 ? frame.width / 2 : (event.clientX - box.left) * frame.width / box.width;
    const y = event.detail === 0 ? frame.height / 2 : (event.clientY - box.top) * frame.height / box.height;
    if (x < 0 || y < 0 || x >= frame.width || y >= frame.height) return;
    inputPending.current = true;
    setBusy(true); setError(false);
    try {
      const result = await desktopClient.act({ session: active.ref, commandId: crypto.randomUUID(), geometryRevision: frame.geometryRevision, action: { action: { case: "pointer", value: { kind: PointerAction_Kind.CLICK, button: PointerAction_Button.PRIMARY, displayId: frame.displayId, x, y } } } }, { headers: operatorHeaders(account.token) });
      if (result.receipt?.outcome !== "applied") { setUncertain(true); return; }
      await observe(active, account);
    } catch { setUncertain(true); setError(true); }
    finally { inputPending.current = false; setBusy(false); }
  }
  async function scroll(horizontalTicks: number, verticalTicks: number) {
    if (!account || !active?.control || !store.canObserve(active.ref) || !frame || busy || inputPending.current || uncertain || active.expires <= Date.now()) return;
    inputPending.current = true;
    setBusy(true); setError(false);
    try {
      const result = await desktopClient.act({ session: active.ref, commandId: crypto.randomUUID(), geometryRevision: frame.geometryRevision, action: { action: { case: "wheel", value: { displayId: frame.displayId, x: frame.width / 2, y: frame.height / 2, horizontalTicks, verticalTicks } } } }, { headers: operatorHeaders(account.token) });
      if (result.receipt?.outcome !== "applied") { setUncertain(true); return; }
      await observe(active, account);
    } catch { setUncertain(true); setError(true); }
    finally { inputPending.current = false; setBusy(false); }
  }
  async function discoverApplications() {
    if (!active || !account || !store.canObserve(active.ref) || busy || inputPending.current) return;
    inputPending.current = true; setBusy(true); setError(false);
    setApplications(undefined); setApplicationId(""); setElementId("");
    setFrame(previous => previous ? { ...previous, semantic: undefined } : previous);
    try {
      const result = await desktopClient.applications({ session: active.ref }, { headers: operatorHeaders(account.token) });
      if (mounted.current && store.canObserve(active.ref)) setApplications(result);
    } catch { setError(true); }
    finally { inputPending.current = false; setBusy(false); }
  }
  async function inspectApplication() {
    if (!active || !account || !selectedApplication || !applications?.expiresAt || Number(applications.expiresAt.seconds) * 1000 <= Date.now() || busy || inputPending.current) return;
    setBusy(true); setError(false);
    try { await observe(active, account, { applicationId, applicationRevision: applications.revision }); } catch { setError(true); }
    finally { setBusy(false); }
  }
  async function insertText() {
    const semantic = frame?.semantic;
    const element = fields.find(item => item.elementId === elementId);
    if (!active?.control || !store.canObserve(active.ref) || !account || !frame || !semantic?.expiresAt || !element || !textDraft || busy || inputPending.current || uncertain || !Number.isInteger(position) || position < 0 || position > 2147483647 || Number(semantic.expiresAt.seconds) * 1000 <= Date.now() || active.expires <= Date.now()) return;
    inputPending.current = true; setBusy(true); setError(false);
    try {
      const result = await desktopClient.act({ session: active.ref, commandId: crypto.randomUUID(), geometryRevision: frame.geometryRevision, action: { action: { case: "text", value: { text: textDraft, elementId, observationRevision: semantic.revision, position } } } }, { headers: operatorHeaders(account.token) });
      if (result.receipt?.outcome !== "applied") { setUncertain(true); return; }
      setTextDraft(""); setElementId("");
      await observe(active, account);
    } catch { setUncertain(true); setError(true); }
    finally { inputPending.current = false; setBusy(false); }
  }
  async function signOut() {
    if (active || !account) return;
    setBusy(true);
    try { await operatorClient.logout({ accessToken: account.token }); }
    catch { /* Local logout still clears in-memory credentials. */ }
    finally { setAccount(undefined); setFrame(undefined); setBusy(false); }
  }

  return <section className="mt-4 space-y-3 border-t border-app-border pt-3" aria-label={t(strings.desktop.title)}>
    <h4 className="font-semibold">{t(strings.desktop.title)}</h4>
    {needsAuthentication && <p role="alert">{t(account ? strings.desktop.reauthenticate : strings.desktop.recoverSignIn, { email: account?.email ?? "" })}</p>}
    {error && <p role="alert">{t(strings.desktop.failed)}</p>}
    {(!account || needsAuthentication) ? <form onSubmit={event => void signIn(event)} className="space-y-2">
      <label className="block">{t(strings.desktop.email)}<input className="block w-full rounded border border-app-border bg-app-background p-2 text-app-foreground" type="email" autoComplete="username" required value={email} onChange={event => setEmail(event.target.value)} disabled={busy} /></label>
      <label className="block">{t(strings.desktop.password)}<input className="block w-full rounded border border-app-border bg-app-background p-2 text-app-foreground" type="password" autoComplete={register ? "new-password" : "current-password"} required value={password} onChange={event => setPassword(event.target.value)} disabled={busy} /></label>
      <Button type="submit" disabled={busy}>{t(register ? strings.desktop.register : strings.desktop.signIn)}</Button>
      <Button type="button" variant="outline" disabled={busy} onClick={() => setRegister(!register)}>{t(register ? strings.desktop.signIn : strings.desktop.register)}</Button>
    </form> : <>
      <p>{t(strings.desktop.signedIn, { email: account.email })}</p>
      {recoveryRequired && !active && <Button type="button" disabled={recovering || opening} onClick={() => void store.recover()}>{t(strings.companion.checkStatus)}</Button>}
      {!active ? <div className="flex flex-wrap gap-2">
        <Button type="button" disabled={!surface || busy || opening || recoveryRequired} onClick={() => void open(false)}>{t(strings.desktop.observe)}</Button>
        <Button type="button" variant="outline" disabled={!surface || busy || opening || recoveryRequired} onClick={() => void open(true)}>{t(strings.desktop.control)}</Button>
        <Button type="button" variant="outline" disabled={busy || opening || recoveryRequired} onClick={() => void signOut()}>{t(strings.desktop.signOut)}</Button>
      </div> : <div className="flex flex-wrap gap-2">
        <p>{t(active.control ? strings.desktop.controlActive : strings.desktop.observeActive)}</p>
        <Button type="button" variant="outline" disabled={busy} onClick={() => void refreshFrame()}>{t(strings.desktop.refresh)}</Button>
        <Button type="button" disabled={stopping} onClick={() => void stop()}>{t(stopRequested ? strings.companion.checkStatus : strings.desktop.stop)}</Button>
      </div>}
    </>}
    {active && !needsAuthentication && <details className="space-y-2">
      <summary>{t(strings.desktop.applicationFields)}</summary>
      <Button type="button" disabled={busy} onClick={() => void discoverApplications()}>{t(strings.desktop.discoverApplications)}</Button>
      {applications && <>
        <label className="block">{t(strings.desktop.application)}<select aria-label={t(strings.desktop.application)} className={fieldClass} value={applicationId} disabled={busy || !applicationsFresh} onChange={event => { setApplicationId(event.target.value); setElementId(""); setFrame(previous => previous ? { ...previous, semantic: undefined } : previous); }}>
          <option value="">{t(strings.desktop.chooseApplication)}</option>
          {applications.applications.map((item, index) => <option key={item.applicationId} value={item.applicationId}>{item.name || t(strings.desktop.unnamedApplication)} · {index + 1}</option>)}
        </select></label>
        {applications.applications.length === 0 && <p role="status">{t(strings.desktop.noApplications)}</p>}
        {!applicationsFresh && <p role="status">{t(strings.desktop.applicationsExpired)}</p>}
      </>}
      <label className="block">{t(strings.desktop.insertDraft)}<textarea className={fieldClass} value={textDraft} maxLength={4096} disabled={!active.control || busy} onChange={event => setTextDraft(event.target.value)} /></label>
      <Button type="button" disabled={busy || !selectedApplication || !applicationsFresh} onClick={() => void inspectApplication()}>{t(strings.desktop.inspectApplication)}</Button>
      {frame?.semantic && <>
        <label className="block">{t(strings.desktop.window)}<select aria-label={t(strings.desktop.window)} className={fieldClass} value={windowId} disabled={busy || !semanticFresh} onChange={event => { setWindowId(event.target.value); setElementId(""); }}>
          <option value="">{t(strings.desktop.chooseWindow)}</option>
          {windows.map((item, index) => <option key={item.elementId} value={item.elementId}>{item.name || t(strings.desktop.unnamedWindow)} · {index + 1}</option>)}
        </select></label>
        {windows.length === 0 && <p role="status">{t(strings.desktop.noWindows)}</p>}
        {selectedWindow && fields.length === 0 && <p role="status">{t(strings.desktop.noFields)}</p>}
        <label className="block">{t(strings.desktop.editableField)}<select className={fieldClass} aria-label={t(strings.desktop.editableField)} value={elementId} disabled={busy || !semanticFresh || !selectedWindow} onChange={event => setElementId(event.target.value)}>
          <option value="">{t(strings.desktop.chooseField)}</option>
          {fields.map((item, index) => <option key={item.elementId} value={item.elementId}>{item.name || t(strings.desktop.unnamedField)} · {index + 1}</option>)}
        </select></label>
        <label className="block">{t(strings.desktop.characterPosition)}<input className={fieldClass} type="number" min="0" max="2147483647" value={position} disabled={busy} onChange={event => setPosition(Number(event.target.value))} /></label>
        {!semanticFresh && <p role="status">{t(strings.desktop.fieldsExpired)}</p>}
        <Button type="button" disabled={!active.control || busy || uncertain || !semanticFresh || !fields.some(item => item.elementId === elementId) || !textDraft || !Number.isInteger(position) || position < 0 || position > 2147483647} onClick={() => void insertText()}>{t(strings.desktop.insertText)}</Button>
      </>}
    </details>}
    {account && !needsAuthentication && <DesktopFlowPanel key={account.email} token={account.token} session={active?.ref} control={Boolean(active?.control)} applicationId={applicationId} applicationRevision={applications?.revision ?? ""} applicationFresh={applicationsFresh} windowName={selectedWindow?.name ?? ""} fieldName={fields.find(item => item.elementId === elementId)?.name ?? ""} busy={busy} blocked={uncertain} onUncertain={setUncertain} onBusy={setBusy} />}
    {uncertain && <p role="alert">{t(strings.desktop.uncertain)}</p>}
    {imageURL && active?.control && <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t(strings.desktop.scrollCenter)}>
      <span>{t(strings.desktop.scrollCenter)}</span>
      <Button type="button" variant="outline" disabled={busy || uncertain} onClick={() => void scroll(0, -3)}>{t(strings.desktop.scrollUp)}</Button>
      <Button type="button" variant="outline" disabled={busy || uncertain} onClick={() => void scroll(0, 3)}>{t(strings.desktop.scrollDown)}</Button>
      <Button type="button" variant="outline" disabled={busy || uncertain} onClick={() => void scroll(-3, 0)}>{t(strings.desktop.scrollLeft)}</Button>
      <Button type="button" variant="outline" disabled={busy || uncertain} onClick={() => void scroll(3, 0)}>{t(strings.desktop.scrollRight)}</Button>
    </div>}
    {imageURL && <button type="button" className="block max-w-full border-0 p-0" aria-label={t(strings.desktop.clickImage)} disabled={!active?.control || busy || uncertain} onClick={event => void click(event)}><img src={imageURL} alt={t(strings.desktop.image)} className="block h-auto max-w-full" draggable={false} /></button>}
  </section>;
}


const ignoreActivity = () => undefined;
export function DesktopRecoveryPanel() {
  const state = useDesktopSession();
  return state.restored && state.recoveryRequired ? <DesktopSessionPanel onActive={ignoreActivity} /> : null;
}
