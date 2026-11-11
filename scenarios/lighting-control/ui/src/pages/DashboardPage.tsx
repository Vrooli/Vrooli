import { useState } from "react";
import { Activity, AlertTriangle, Check, CircleHelp, LoaderCircle, Power, Sparkles, Sun } from "lucide-react";

import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";
import { BackendState, OperationState, type Controller, type ZoneStatus } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PresetKind, type EffectivePolicy, type EvidenceSource } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { EvidenceState } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { useTranslation } from "../i18n";
import { HealthCard } from "../features/health/HealthCard";
import { lightingApi } from "../features/lights/api";
import { hasStableControllerIdentity } from "../features/lights/identity";
import { useLightDesk } from "../features/lights/useLightDesk";
import { ManualZoneEditor } from "../features/lights/ManualZoneEditor";
import { Link } from "react-router-dom";

const presets = [PresetKind.OFF, PresetKind.LOAD, PresetKind.WORK, PresetKind.HEALTH, PresetKind.COMBINED, PresetKind.STATIC] as const;
const backendStateNames: Partial<Record<number, string>> = BackendState;

export function DashboardPage() {
  const { t } = useTranslation();
  const { desk, run, busy, refresh } = useLightDesk();
  const phone = useMediaQuery("(max-width: 720px)");
  const [notice, setNotice] = useState("");
  const [actionError, setActionError] = useState("");
  const act = async (action: () => Promise<unknown>, success: string) => {
    setNotice("");
    setActionError("");
    try { await run(action); setNotice(success); }
    catch (error) { setActionError(error instanceof Error ? error.message : t(strings.lightDesk.actionFailed)); }
  };
  const data = desk.data;
  const controllers = data?.controllers ?? [];
  const enrolled = controllers.filter((c) => c.enrolled);
  const statuses = new Map((data?.zones ?? []).map((zone) => [`${zone.desired?.controllerId}/${zone.desired?.zoneId}`, zone]));
  const backendReady = !desk.isError && data?.backend?.state === BackendState.READY;
  const controllerPanel = <ControllerPanel controllers={controllers} statuses={statuses} busy={busy} backendReady={backendReady} maxBrightness={data?.config?.brightnessCapPercent} act={act}/>;

  return <section data-testid={selectors.pages.dashboard} aria-labelledby="dashboard-heading" className="flex flex-col gap-space-md pb-20 md:pb-0">
    <PageHeader headingId="dashboard-heading" title={t(strings.pages.dashboard.title)} description={t(strings.pages.dashboard.description)} testId={selectors.pages.dashboardHeader} />
    {desk.isLoading ? <div className="lightdesk-loading" role="status"><LoaderCircle className="animate-spin" aria-hidden="true" />{t(strings.lightDesk.loading)}</div> : null}
    {desk.isError ? <div className="lightdesk-alert" role="alert"><AlertTriangle aria-hidden="true"/><span>{t(strings.lightDesk.loadFailed)}: {desk.error.message}</span><button className="lightdesk-button" onClick={() => void refresh()}>{t(strings.lightDesk.retry)}</button></div> : null}
    {data ? <>
      {data.effective?.evidence.some((source) => { const stamp = source.sourceTimestamp ?? source.observedAt; return source.state !== EvidenceState.FRESH || !stamp || (data.config && Date.now() - Number(stamp.seconds) * 1000 > data.config.sourceMaxAgeSeconds * 1000); }) ? <div role="status" className="lightdesk-alert lightdesk-alert--stale"><AlertTriangle aria-hidden="true"/><span>{t(strings.lightDesk.staleWarning)}</span></div> : null}
      <div className={phone ? "lightdesk-mobile" : "lightdesk-desktop"}>
        {phone ? <>
          <BackendPanel state={data.backend?.state} detail={data.backend?.detail ?? ""} protocol={data.backend?.sdkProtocolVersion ?? 0} count={enrolled.length} onCheck={() => void refresh()}/>
          <PolicyPanel effective={data.effective} evidence={data.effective?.evidence ?? []} maxAgeSeconds={data.config?.sourceMaxAgeSeconds ?? 60} busy={busy} onPreset={(kind) => void act(() => lightingApi.choosePreset(kind), t(strings.lightDesk.presetRequested))} onResume={() => void act(() => lightingApi.clearPreset(), t(strings.lightDesk.autoResumed))}/>
          <HealthCard/>
          {controllerPanel}
          <Inspector controllers={controllers} statuses={statuses} offActive={Boolean(data.manualOverride?.active)}/>
        </> : <>
          <div className="lightdesk-column"><BackendPanel state={data.backend?.state} detail={data.backend?.detail ?? ""} protocol={data.backend?.sdkProtocolVersion ?? 0} count={enrolled.length} onCheck={() => void refresh()}/>{controllerPanel}</div>
          <div className="lightdesk-column"><PolicyPanel effective={data.effective} evidence={data.effective?.evidence ?? []} maxAgeSeconds={data.config?.sourceMaxAgeSeconds ?? 60} busy={busy} onPreset={(kind) => void act(() => lightingApi.choosePreset(kind), t(strings.lightDesk.presetRequested))} onResume={() => void act(() => lightingApi.clearPreset(), t(strings.lightDesk.autoResumed))}/><HealthCard/></div>
          <div className="lightdesk-column"><Inspector controllers={controllers} statuses={statuses} offActive={Boolean(data.manualOverride?.active)}/></div>
        </>}
      </div>
      {notice ? <p role="status" className="lightdesk-notice">{notice}</p> : null}
      {actionError ? <p role="alert" className="lightdesk-alert">{actionError}</p> : null}
      {data.manualOverride?.active ? <div className="lightdesk-alert lightdesk-alert--off" role="status"><Power aria-hidden="true"/><span>{t(strings.lightDesk.manualOffActive)}</span><button className="lightdesk-button" disabled={busy} onClick={() => void act(() => lightingApi.clearOff(), t(strings.lightDesk.offCleared))}>{t(strings.lightDesk.resumeLighting)}</button></div> : null}
      <div className="lightdesk-mobile-action"><button data-testid={selectors.lightDesk.manualOff} className="lightdesk-button lightdesk-button--danger" disabled={busy || enrolled.length === 0 || data.manualOverride?.active} onClick={() => void act(() => lightingApi.setOff(), t(strings.lightDesk.offRequested))}><Power aria-hidden="true"/>{t(strings.lightDesk.turnOff)}</button></div>
    </> : null}
  </section>;
}

function BackendPanel({ state, detail, protocol, count, onCheck }: { state?: BackendState; detail: string; protocol: number; count: number; onCheck: () => void }) {
  const { t } = useTranslation();
  const ready = state === BackendState.READY;
  const stateLabel = state === undefined ? undefined : backendStateNames[state]?.replace("BACKEND_STATE_", "");
  return <section className="lightdesk-panel" aria-labelledby="backend-title"><div className="lightdesk-panel-heading"><Activity aria-hidden="true"/><h2 id="backend-title">{t(strings.lightDesk.backend)}</h2></div>
    <p className={`lightdesk-status ${ready ? "is-ready" : "is-unavailable"}`}><i/> {ready ? t(strings.lightDesk.backendReady) : t(strings.lightDesk.backendUnavailable)} · {stateLabel ?? t(strings.lightDesk.controllerUi.backendUnknown)}</p>
    <p className="lightdesk-muted">{detail || t(strings.lightDesk.backendUnknown)}</p><p className="lightdesk-meta">{ready ? `${t(strings.lightDesk.protocolVersion)} ${protocol} · ${count} ${t(strings.lightDesk.enrolledCount)}` : t(strings.lightDesk.desiredStateRetained)}</p><button className="lightdesk-button lightdesk-button--small" onClick={onCheck}>{t(strings.lightDesk.checkBackend)}</button>
  </section>;
}

function ControllerPanel({ controllers, statuses, busy, backendReady, maxBrightness, act }: { controllers: Controller[]; statuses: Map<string, ZoneStatus>; busy: boolean; backendReady: boolean; maxBrightness?: number; act: (fn: () => Promise<unknown>, success: string) => Promise<void> }) {
  const { t } = useTranslation();
  return <section className="lightdesk-panel" aria-labelledby="controllers-title"><div className="lightdesk-panel-heading"><Sun aria-hidden="true"/><h2 id="controllers-title">{t(strings.lightDesk.controllers)}</h2></div>
    {controllers.length === 0 ? <div className="lightdesk-empty"><Sun aria-hidden="true"/><h3>{t(strings.lightDesk.noControllers)}</h3><p>{t(strings.lightDesk.noControllersHelp)}</p></div> : null}
    <ul className="lightdesk-controller-list">{controllers.map((controller) => <li key={controller.controllerId} className="lightdesk-controller">
      <div className="lightdesk-controller-head"><div><h3><Link className="lightdesk-link" to={`/controllers/${encodeURIComponent(controller.controllerId)}`}>{controller.name || controller.vendor || controller.controllerId}</Link></h3><p className="lightdesk-muted">{controller.vendor} · {controller.zones.length} {t(strings.lightDesk.zones)}</p></div><button className="lightdesk-button lightdesk-button--small" disabled={busy || (!controller.enrolled && (!backendReady || !hasStableControllerIdentity(controller)))} onClick={() => void act(() => lightingApi.enroll(controller.controllerId, !controller.enrolled), controller.enrolled ? t(strings.lightDesk.controllerUnenrolled) : t(strings.lightDesk.controllerEnrolled))}>{controller.enrolled ? t(strings.lightDesk.unenroll) : t(strings.lightDesk.enroll)}</button></div>
      {!hasStableControllerIdentity(controller) ? <p className="lightdesk-warning">{t(strings.lightDesk.controllerUi.identityUnavailable)}</p> : null}
      {controller.zones.map((zone) => {
        const status = statuses.get(`${controller.controllerId}/${zone.zoneId}`);
        const state = status?.desired?.desiredMode;
        const swatch = state?.case === "staticColor" ? `rgb(${state.value.color?.red ?? 0} ${state.value.color?.green ?? 0} ${state.value.color?.blue ?? 0})` : undefined;
        return <div className="lightdesk-zone" key={zone.zoneId}><div className="lightdesk-zone-main"><span className="lightdesk-color-dot" style={{ background: swatch }}/><div><strong>{zone.name}</strong><p className="lightdesk-muted">{zone.ledCount} {t(strings.lightDesk.leds)} · {zone.supportsStaticColor ? t(strings.lightDesk.staticSupported) : t(strings.lightDesk.staticUnsupported)}</p></div></div>
          {controller.enrolled ? <ManualZoneEditor controllerId={controller.controllerId} zone={zone} desired={status?.desired} maxBrightness={maxBrightness} disabled={busy || !backendReady || !hasStableControllerIdentity(controller)} act={act}/> : null}
          <p className="lightdesk-operation">{t(operationKey(status?.lastApply?.operation))}{status?.lastApply?.backendAcknowledged ? ` · ${t(strings.lightDesk.acknowledgedNotObserved)}` : ""}{status?.lastApply?.reason ? ` · ${status.lastApply.reason}` : ""}</p>
          {zone.supportedEffects.length ? <p className="lightdesk-muted">{t(strings.lightDesk.effectsAvailable)}: {zone.supportedEffects.join(", ")}</p> : null}
        </div>;
      })}
    </li>)}</ul>
  </section>;
}

function PolicyPanel({ effective, evidence, maxAgeSeconds, busy, onPreset, onResume }: { effective?: EffectivePolicy; evidence: EvidenceSource[]; maxAgeSeconds: number; busy: boolean; onPreset: (kind: PresetKind) => void; onResume: () => void }) {
  const { t } = useTranslation();
  return <section className="lightdesk-panel" aria-labelledby="policy-title"><div className="lightdesk-panel-heading"><Sparkles aria-hidden="true"/><h2 id="policy-title">{t(strings.lightDesk.policy)}</h2></div>
    {effective ? <><p className="lightdesk-policy-name">{t(presetKey(effective.preset))}</p><p className="lightdesk-muted">{effective.reason || t(strings.lightDesk.noPolicyReason)}</p><div className="lightdesk-policy-details"><span>{t(strings.lightDesk.brightness)} <strong>{effective.brightnessPercent}%</strong></span><span>{effective.effectId ? `${t(strings.lightDesk.effect)}: ${effective.effectId}` : t(strings.lightDesk.staticColor)}</span></div>
      {effective.staticFallback ? <p className="lightdesk-warning"><AlertTriangle aria-hidden="true"/>{t(strings.lightDesk.staticFallback)}</p> : null}
      <label className="lightdesk-select-label">{t(strings.lightDesk.choosePreset)}<select value={effective.preset} disabled={busy} onChange={(event) => onPreset(Number(event.target.value))}>{presets.map((kind) => <option value={kind} key={kind}>{t(presetKey(kind))}</option>)}</select></label>
      {effective.manualPriority ? <button className="lightdesk-button" disabled={busy} onClick={onResume}>{t(strings.lightDesk.resumeAutomatic)}</button> : null}
    </> : <p className="lightdesk-muted">{t(strings.lightDesk.policyUnavailable)}</p>}
    <div className="lightdesk-evidence"><h3>{t(strings.lightDesk.sourceEvidence)} <CircleHelp aria-label={t(strings.lightDesk.sourceEvidenceHelp)}/></h3>{evidence.length ? evidence.map((source) => { const observed = source.sourceTimestamp ?? source.observedAt; const ageSeconds = observed ? Math.max(0, Math.floor(Date.now() / 1000 - Number(observed.seconds))) : null; const stale = ageSeconds === null || ageSeconds > maxAgeSeconds || source.state !== EvidenceState.FRESH; return <div className="lightdesk-evidence-row" key={source.source}><strong>{source.source}</strong><span className={stale ? "is-stale" : "is-fresh"}>{source.state === EvidenceState.FRESH ? "FRESH" : source.state === EvidenceState.STALE ? "STALE" : source.state === EvidenceState.UNAVAILABLE ? "UNAVAILABLE" : source.state === EvidenceState.DEGRADED ? "DEGRADED" : "UNKNOWN"}</span><small>{ageSeconds === null ? "Source age unknown" : `${ageSeconds}s old · stale after ${maxAgeSeconds}s`}{source.reason ? ` · ${source.reason}` : ""}</small></div>; }) : <p className="lightdesk-muted">{t(strings.lightDesk.noEvidence)}</p>}</div>
  </section>;
}

function Inspector({ controllers, statuses, offActive }: { controllers: Controller[]; statuses: Map<string, ZoneStatus>; offActive: boolean }) {
  const { t } = useTranslation();
  const zones = [...statuses.values()];
  const total = controllers.filter((controller) => controller.enrolled).reduce((n, controller) => n + controller.zones.length, 0);
  const acknowledgements = zones.filter((zone) => zone.lastApply?.backendAcknowledged).length;
  const observations = zones.filter((zone) => zone.observation).length;
  return <section className="lightdesk-panel" aria-labelledby="evidence-title"><div className="lightdesk-panel-heading"><Check aria-hidden="true"/><h2 id="evidence-title">{t(strings.lightDesk.commandEvidence)}</h2></div><dl className="lightdesk-inspector-list"><div><dt>{t(strings.lightDesk.enrolledZones)}</dt><dd>{total}</dd></div><div><dt>{t(strings.lightDesk.backendAccepted)}</dt><dd>{acknowledgements}</dd></div><div><dt>{t(strings.lightDesk.observedOutput)}</dt><dd>{observations}</dd></div><div><dt>{t(strings.lightDesk.manualPriority)}</dt><dd>{offActive ? t(strings.lightDesk.active) : t(strings.lightDesk.inactive)}</dd></div></dl><p className="lightdesk-warning"><CircleHelp aria-hidden="true"/>{t(strings.lightDesk.acknowledgedNotObserved)}</p></section>;
}

function operationKey(value: OperationState | undefined) {
  const key = value === OperationState.BACKEND_ACKNOWLEDGED ? strings.lightDesk.backendAccepted : value === OperationState.COMMAND_SENT ? strings.lightDesk.commandSent : value === OperationState.DEFERRED ? strings.lightDesk.deferred : value === OperationState.UNSUPPORTED ? strings.lightDesk.unsupported : value === OperationState.FAILED ? strings.lightDesk.failed : strings.lightDesk.noCommand;
  return key;
}

function presetKey(value: PresetKind) {
  switch (value) {
    case PresetKind.OFF: return strings.lightDesk.presets.off;
    case PresetKind.LOAD: return strings.lightDesk.presets.load;
    case PresetKind.WORK: return strings.lightDesk.presets.work;
    case PresetKind.HEALTH: return strings.lightDesk.presets.health;
    case PresetKind.COMBINED: return strings.lightDesk.presets.combined;
    case PresetKind.STATIC: return strings.lightDesk.presets.static;
    default: return strings.lightDesk.presets.static;
  }
}
