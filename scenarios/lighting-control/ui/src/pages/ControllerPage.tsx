import { useState } from "react";
import { AlertTriangle, ArrowLeft, CircleHelp, Power, Sun } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";
import { BackendState, OperationState, type ZoneStatus } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PresetKind } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { lightingApi } from "../features/lights/api";
import { hasStableControllerIdentity } from "../features/lights/identity";
import { ManualZoneEditor } from "../features/lights/ManualZoneEditor";
import { manualOffCoversAll, manualOffCoversController } from "../features/lights/manualOff";
import { useManualOffDescription } from "../features/lights/useManualOffDescription";
import { useLightDesk } from "../features/lights/useLightDesk";

export function ControllerPage() {
  const { controllerId = "" } = useParams();
  const phone = useMediaQuery("(max-width: 720px)");
  const { t } = useTranslation();
  const { desk, run, busy, refresh } = useLightDesk();
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const controller = desk.data?.controllers.find((item) => item.controllerId === controllerId);
  const offDescription = useManualOffDescription(desk.data?.manualOverride, desk.data?.controllers ?? []);
  const zoneStatuses = desk.data?.zones ?? [];
  const maxBrightness = desk.data?.config?.brightnessCapPercent;
  const backendReady = !desk.isError && desk.data?.backend?.state === BackendState.READY;

  const apply = async (action: () => Promise<unknown>, success: string) => {
    setError(""); setNotice("");
    try { await run(action); setNotice(success); }
    catch (cause) { setError(cause instanceof Error ? cause.message : t(strings.lightDesk.controllerUi.colorFailed)); }
  };
  const turnOff = async () => {
    setError("");
    setNotice("");
    try {
      await run(() => lightingApi.setOff());
      setNotice(t(strings.lightDesk.controllerUi.offSaved));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.lightDesk.controllerUi.offFailed));
    }
  };
  const enroll = async () => {
    setError("");
    setNotice("");
    try {
      await run(() => lightingApi.enroll(controllerId, true));
      setNotice(t(strings.lightDesk.controllerEnrolled));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(strings.lightDesk.actionFailed));
    }
  };

  if (desk.isLoading) return <section data-testid={selectors.pages.controller} className="lightdesk-page"><p role="status">{t(strings.lightDesk.controllerUi.loading)}</p></section>;
  if (!controller) return <section data-testid={selectors.pages.controller} className="lightdesk-page">
    <PageHeader headingId="controller-heading" title={t(strings.lightDesk.controllerUi.unavailableTitle)} description={t(strings.lightDesk.controllerUi.unavailableDescription)}/>
    <div role="alert" className="lightdesk-alert"><AlertTriangle aria-hidden="true"/>{desk.error?.message || t(strings.lightDesk.controllerUi.unavailableHelp)}</div>
    <button className="lightdesk-button" onClick={() => void refresh()} disabled={desk.isFetching}>{t(strings.lightDesk.retry)}</button>
    <Link className="lightdesk-back-link" to="/">{t(strings.lightDesk.controllerUi.back)}</Link>
  </section>;

  const enrollmentLabel = controller.enrolled ? t(strings.lightDesk.controllerUi.enrolled) : t(strings.lightDesk.controllerUi.notEnrolled);
  const stableIdentity = hasStableControllerIdentity(controller);
  const policyLabel = manualOffCoversController(desk.data?.manualOverride, controllerId) ? t(strings.lightDesk.presets.off) : desk.data?.effective?.preset === undefined ? t(strings.lightDesk.controllerUi.backendUnknown) : t(presetLabelKey(desk.data.effective.preset));
  const observations = zoneStatuses.filter((zone) => zone.desired?.controllerId === controllerId && zone.observation).length;
  const offAction = <button className="lightdesk-button lightdesk-button--danger" disabled={busy || !stableIdentity || !controller.enrolled || manualOffCoversAll(desk.data?.manualOverride)} onClick={() => void turnOff()}><Power size={16} aria-hidden="true"/>{t(strings.lightDesk.controllerUi.keepOff)}</button>;
  return <section data-testid={selectors.pages.controller} aria-labelledby="controller-heading" className="lightdesk-page">
    <Link className="lightdesk-back-link" to="/"><ArrowLeft size={16} aria-hidden="true"/>{t(strings.lightDesk.controllerUi.back)}</Link>
    <PageHeader headingId="controller-heading" title={controller.name || controller.vendor || t(strings.lightDesk.controllerUi.unknownVendor)} description={t(strings.lightDesk.controllerUi.controllerSummary, { vendor: controller.vendor || t(strings.lightDesk.controllerUi.unknownVendor), count: controller.zones.length, status: enrollmentLabel })}/>
    {!backendReady ? <p className="lightdesk-warning">{t(strings.lightDesk.desiredStateRetained)}</p> : null}
    <div className="lightdesk-controller-detail">
      <section className="lightdesk-panel">
        <div className="lightdesk-panel-heading"><Sun aria-hidden="true"/><h2>{t(strings.lightDesk.controllerUi.zoneControls)}</h2></div>
        {controller.zones.map((zone) => <div key={zone.zoneId} role="group" aria-label={zone.name} className="lightdesk-zone-detail">
          <div>
            <strong>{zone.name}</strong>
            <p className="lightdesk-muted">{t(strings.lightDesk.controllerUi.leds, { count: zone.ledCount })} · {zone.supportsStaticColor ? t(strings.lightDesk.controllerUi.staticSupported) : t(strings.lightDesk.controllerUi.staticNotReported)}{zone.supportsBrightness ? ` · ${t(strings.lightDesk.controllerUi.brightnessSupported)}` : ` · ${t(strings.lightDesk.controllerUi.softwareBrightness)}`}</p>
            <p className="lightdesk-muted">{zone.supportedEffects.length ? t(strings.lightDesk.controllerUi.effectsReported, { effects: zone.supportedEffects.join(", ") }) : t(strings.lightDesk.controllerUi.noEffects)}</p>
          </div>
          <ManualZoneEditor controllerId={controllerId} zone={zone} desired={zoneStatuses.find((item) => item.desired?.controllerId === controllerId && item.desired.zoneId === zone.zoneId)?.desired} maxBrightness={maxBrightness} disabled={busy || !stableIdentity || !controller.enrolled || !backendReady} detail act={apply}/>
            <ZoneResult status={zoneStatuses.find((item) => item.desired && item.desired.controllerId === controllerId && item.desired.zoneId === zone.zoneId)} />
        </div>)}
        {!stableIdentity ? <p className="lightdesk-warning">{t(strings.lightDesk.controllerUi.identityUnavailable)}</p> : null}
        {!controller.enrolled ? <button className="lightdesk-button lightdesk-button--primary" disabled={busy || !stableIdentity || !backendReady} onClick={() => void enroll()}>{t(strings.lightDesk.controllerUi.enrollController)}</button> : null}
      </section>
      <div className="lightdesk-panel lightdesk-controller-status">
        <h2>{t(strings.lightDesk.controllerUi.status)}</h2>
        <dl className="lightdesk-inspector-list">
          <div><dt>{t(strings.lightDesk.controllerUi.backend)}</dt><dd>{desk.data?.backend?.state === BackendState.READY ? t(strings.lightDesk.controllerUi.backendReady) : t(strings.lightDesk.controllerUi.backendUnavailable)}</dd></div>
          <div><dt>{t(strings.lightDesk.controllerUi.lastChecked)}</dt><dd>{desk.data?.backend?.checkedAt || t(strings.lightDesk.controllerUi.backendUnknown)}</dd></div>
          <div><dt>{t(strings.lightDesk.controllerUi.sdkProtocol)}</dt><dd>{desk.data?.backend?.sdkProtocolVersion || t(strings.lightDesk.controllerUi.sdkUnknown)}</dd></div>
          <div><dt>{t(strings.lightDesk.manualPriority)}</dt><dd>{offDescription}</dd></div>
          <div><dt>{t(strings.lightDesk.controllerUi.desiredPolicy)}</dt><dd>{policyLabel}</dd></div>
          <div><dt>{t(strings.lightDesk.controllerUi.physicalOutput)}</dt><dd>{observations > 0 ? `${observations}/${controller.zones.length} · ${t(strings.lightDesk.observedOutput)}` : t(strings.lightDesk.controllerUi.physicalUnknown)}</dd></div>
        </dl>
        {desk.error ? <p role="alert" className="lightdesk-warning">{desk.error.message}</p> : null}
        {desk.data?.backend?.detail ? <p className="lightdesk-muted">{desk.data.backend.detail}</p> : null}
        <p className="lightdesk-warning"><CircleHelp aria-hidden="true"/>{t(strings.lightDesk.controllerUi.ackCaveat)}</p>
        {!phone ? offAction : null}
      </div>
    </div>
    {phone ? <div className="lightdesk-mobile-footer">{offAction}</div> : null}
    {error ? <p role="alert" className="lightdesk-alert">{error}</p> : null}
    {notice ? <p role="status" className="lightdesk-notice">{notice}</p> : null}
  </section>;
}

function ZoneResult({ status }: { status?: ZoneStatus }) {
  const { t } = useTranslation();
  const result = status?.lastApply;
  const label = result?.operation === OperationState.BACKEND_ACKNOWLEDGED ? t(strings.lightDesk.controllerUi.backendAccepted) : result?.operation === OperationState.COMMAND_SENT ? t(strings.lightDesk.controllerUi.commandSent) : result?.operation === OperationState.FAILED ? t(strings.lightDesk.controllerUi.failed) : result?.operation === OperationState.UNSUPPORTED ? t(strings.lightDesk.controllerUi.unsupported) : result?.operation === OperationState.DEFERRED ? t(strings.lightDesk.controllerUi.deferred) : t(strings.lightDesk.controllerUi.noCommand);
  return <p className="lightdesk-operation">{label}{result?.reason ? ` · ${result.reason}` : ""}{status?.observation ? ` · ${t(strings.lightDesk.controllerUi.observed, { source: status.observation.source })}` : ` · ${t(strings.lightDesk.controllerUi.notVerified)}`}</p>;
}

function presetLabelKey(value: PresetKind) {
  switch (value) {
    case PresetKind.OFF: return strings.lightDesk.presets.off;
    case PresetKind.LOAD: return strings.lightDesk.presets.load;
    case PresetKind.WORK: return strings.lightDesk.presets.work;
    case PresetKind.HEALTH: return strings.lightDesk.presets.health;
    case PresetKind.COMBINED: return strings.lightDesk.presets.combined;
    default: return strings.lightDesk.presets.static;
  }
}
