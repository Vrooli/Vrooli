import { AccountSession } from "../components/AccountSession";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useEffect, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { Select } from "@vrooli/react-component-library/Select/1";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { AppNotificationSettings } from "@vrooli/react-component-library/AppNotificationSettings/1";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { SUPPORTED_LOCALES, getCurrentLocale, getLocaleConfig, setLocale, useTranslation, type Locale } from "../i18n";
import { useTheme, type ThemeChoice } from "../theme/ThemeProvider";
import { useLightDesk } from "../features/lights/useLightDesk";
import { lightingSetupApi } from "../features/lights/setup";
import { InstallationSource, LightingSelectionSchema, LightingDeviceAccessResultSchema, type LightingRuntimeStatus, type LightingSelection, type LightingDeviceDiscovery, type LightingDevicePlan, type LightingDeviceAccessResult } from "@vrooli/proto-types/lighting-control/v1/setup/setup_pb";
import { lightingApi } from "../features/lights/api";
import { BackendState, ColorSchema, type BackendStatus } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { PolicyConfigSchema, type PolicyConfig } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";

const THEME_CHOICES: readonly ThemeChoice[] = ["light", "dark", "system"];
// Literal references so the strings lint can see every catalog key in use.
const THEME_LABEL_KEY: Record<ThemeChoice, (typeof strings.theme.choice)[ThemeChoice]> = {
  light: strings.theme.choice.light,
  dark: strings.theme.choice.dark,
  system: strings.theme.choice.system,
};

/**
 * Settings owns every preference; nothing else in the shell duplicates one.
 * Appearance and language are the two every scenario has. Add the rest of
 * yours as rows in the same list, grouped by what they change.
 */
export function SettingsPage() {
  const { t } = useTranslation();
  const copy = strings.lightDesk.settingsUi;
  const currentLocale = getCurrentLocale();
  const { choice, setTheme, storageAvailable } = useTheme();
  const { desk, run, busy, refresh } = useLightDesk();
  const [saved, setSaved] = useState("");
  const [saveError, setSaveError] = useState("");
  const config = desk.data?.config;
  const [draftConfig, setDraftConfig] = useState<PolicyConfig | null>(null);
  const [dirtyConfig, setDirtyConfig] = useState(false);
  useEffect(() => { if (config && !dirtyConfig) setDraftConfig(create(PolicyConfigSchema, config)); }, [config, dirtyConfig]);
  const updateConfig = (changes: Partial<PolicyConfig>) => {
    if (!draftConfig) return;
    setDraftConfig(create(PolicyConfigSchema, { ...draftConfig, ...changes }));
    setDirtyConfig(true);
    setSaved("");
  };
  const saveConfig = async () => {
    if (!draftConfig) return;
    const next = create(PolicyConfigSchema, draftConfig);
    setSaved(""); setSaveError("");
    try {
      await run(async () => { await lightingApi.savePolicy(next); return lightingApi.evaluatePolicy(); });
      setDirtyConfig(false); setSaved(t(copy.saved));
    }
    catch (error) { setSaveError(error instanceof Error ? error.message : t(copy.saveFailed)); }
  };

  return (
    <section data-testid={selectors.pages.settings} aria-labelledby="settings-heading" className="flex flex-col gap-space-md">
      <PageHeader headingId="settings-heading" title={t(strings.pages.settings.title)} description={t(strings.pages.settings.description)} />
      <AccountSession />
      <NativeLightingSetup />
      <section className="lightdesk-panel" aria-labelledby="backend-observation-heading">
        <div className="lightdesk-panel-heading"><h2 id="backend-observation-heading">{t(copy.backendObservation)}</h2><button className="lightdesk-button lightdesk-button--small" onClick={() => void refresh()}><RefreshCw size={15} aria-hidden="true"/>{t(copy.refresh)}</button></div>
        {desk.isLoading ? <p role="status">{t(copy.checking)}</p> : desk.isError ? <div role="alert" className="lightdesk-alert"><AlertTriangle aria-hidden="true"/>{t(copy.statusUnavailable, { message: desk.error.message })}<button className="lightdesk-button" onClick={() => void refresh()}>{t(copy.retry)}</button></div> : <BackendSetup backend={desk.data?.backend}/>}
        <button className="lightdesk-button" disabled={busy || desk.data?.backend?.state !== BackendState.READY} onClick={() => void run(() => lightingApi.reconcile())}>{t(copy.reconcile)}</button>
      </section>
      <SettingsList variant="auto">
        <SettingsList.Group label={t(strings.pages.settings.preferences)}>
          <SettingsList.Row label={t(strings.pages.settings.themeHeading)} hint={t(strings.pages.settings.themeHint)}>
            <Select
              aria-label={t(strings.theme.switcherLabel)}
              data-testid={selectors.settingsPage.themeSelect}
              value={choice}
              onChange={(event) => setTheme(event.target.value as ThemeChoice)}
              options={THEME_CHOICES.map((c) => ({ value: c, label: t(THEME_LABEL_KEY[c]) }))}
            />
            {!storageAvailable && <p role="status" className="lightdesk-muted">{t(copy.themeSessionOnly)}</p>}
          </SettingsList.Row>
          <SettingsList.Row label={t(strings.pages.settings.localeHeading)} hint={t(strings.pages.settings.localeHint)}>
            <Select
              aria-label={t(strings.locale.switcherLabel)}
              data-testid={selectors.settingsPage.localeSelect}
              value={currentLocale}
              onChange={(event) => void setLocale(event.target.value as Locale)}
              options={SUPPORTED_LOCALES.map((lng) => ({ value: lng, label: getLocaleConfig(lng).nativeLabel }))}
            />
          </SettingsList.Row>
        </SettingsList.Group>
      </SettingsList>
      <section className="lightdesk-panel" aria-labelledby="policy-settings-heading">
        <div className="lightdesk-panel-heading"><h2 id="policy-settings-heading">{t(copy.policyTitle)}</h2></div>
        {!draftConfig ? <p className="lightdesk-muted">{t(copy.notAvailable)}</p> : <div className="lightdesk-settings-form">
          <label className="lightdesk-field">{t(copy.staleAfter)} <strong>{t(copy.secondsValue, { count: draftConfig.sourceMaxAgeSeconds })}</strong><input type="range" min="10" max="600" step="10" value={draftConfig.sourceMaxAgeSeconds} onChange={(event) => updateConfig({ sourceMaxAgeSeconds: Number(event.target.value) })}/></label>
          <label className="lightdesk-field">{t(copy.maximumBrightness)} <strong>{draftConfig.brightnessCapPercent}%</strong><input type="range" min="0" max="100" value={draftConfig.brightnessCapPercent} onChange={(event) => updateConfig({ brightnessCapPercent: Number(event.target.value), staleFallbackBrightnessPercent: Math.min(draftConfig.staleFallbackBrightnessPercent, Number(event.target.value)) })}/></label>
          <label className="lightdesk-field">{t(copy.loadEnters)} <strong>{draftConfig.loadEnterPercent}%</strong><input type="range" min="10" max="100" value={draftConfig.loadEnterPercent} onChange={(event) => updateConfig({ loadEnterPercent: Number(event.target.value), loadExitPercent: Math.min(draftConfig.loadExitPercent, Number(event.target.value) - 1) })}/></label>
          <label className="lightdesk-field">{t(copy.loadExits)} <strong>{draftConfig.loadExitPercent}%</strong><input type="range" min="0" max={Math.max(0, draftConfig.loadEnterPercent - 1)} value={Math.min(draftConfig.loadExitPercent, draftConfig.loadEnterPercent - 1)} onChange={(event) => updateConfig({ loadExitPercent: Number(event.target.value) })}/></label>
          <label className="lightdesk-field">{t(copy.debounce)} <strong>{t(copy.secondsValue, { count: draftConfig.debounceSeconds })}</strong><input type="range" min="0" max="60" value={draftConfig.debounceSeconds} onChange={(event) => updateConfig({ debounceSeconds: Number(event.target.value) })}/></label>
          <label className="lightdesk-field">{t(copy.fallbackColor)}<input type="color" value={colorHex(draftConfig.staleFallbackColor)} onChange={(event) => updateConfig({ staleFallbackColor: create(ColorSchema, fromHex(event.target.value)) })}/></label>
          <label className="lightdesk-field">{t(copy.fallbackBrightness)} <strong>{draftConfig.staleFallbackBrightnessPercent}%</strong><input type="range" min="0" max={draftConfig.brightnessCapPercent} value={Math.min(draftConfig.staleFallbackBrightnessPercent, draftConfig.brightnessCapPercent)} onChange={(event) => updateConfig({ staleFallbackBrightnessPercent: Number(event.target.value) })}/></label>
          <button className="lightdesk-button lightdesk-button--primary" disabled={busy || !dirtyConfig} onClick={() => void saveConfig()}>{t(copy.savePolicy)}</button>
          {saved ? <p role="status" className="lightdesk-notice">{saved}</p> : null}{saveError ? <p role="alert" className="lightdesk-alert">{saveError}</p> : null}
        </div>}
        <p className="lightdesk-muted">{t(copy.recoveryNote)}</p>
      </section>
      {/* This scenario's notification settings. Renders nothing until
          .vrooli/notifications.json sets "appDelivery": true. */}
      <AppNotificationSettings />
    </section>
  );
}

function BackendSetup({ backend }: { backend?: BackendStatus }) {
  const { t } = useTranslation();
  const copy = strings.lightDesk.settingsUi;
  const state = backend?.state;
  const ready = state === BackendState.READY;
  const stateName = state === undefined ? "Unknown" : BackendState[state].replace("BACKEND_STATE_", "").toLowerCase().replace(/_/g, " ");
  const reason = backend?.reason ? backend.reason.toString().replace("BACKEND_REASON_", "").toLowerCase().replace(/_/g, " ") : "";
  return <div className="lightdesk-backend-setup"><p className={`lightdesk-status ${ready ? "is-ready" : "is-unavailable"}`}><i/>{ready ? t(copy.backendReady) : t(copy.backendNamed, { state: stateName })} · {reason || backend?.detail || t(copy.noDetail)}</p><dl className="lightdesk-inspector-list"><div><dt>{t(copy.version)}</dt><dd>{backend?.version || t(copy.notReported)}</dd></div><div><dt>{t(copy.sdkProtocol)}</dt><dd>{backend?.sdkProtocolVersion || t(copy.notReported)}</dd></div><div><dt>{t(strings.lightDesk.controllerUi.lastChecked)}</dt><dd>{backend?.checkedAt ? new Date(backend.checkedAt).toLocaleString() : t(copy.notChecked)}</dd></div></dl></div>;
}

function colorHex(color?: { red: number; green: number; blue: number }) { return color ? `#${[color.red, color.green, color.blue].map((value) => Math.max(0, Math.min(255, value)).toString(16).padStart(2, "0")).join("")}` : "#203040"; }
function fromHex(value: string) { return { red: parseInt(value.slice(1, 3), 16), green: parseInt(value.slice(3, 5), 16), blue: parseInt(value.slice(5, 7), 16) }; }


const setupStatusKey = ["lighting-control", "native-setup"] as const;

function NativeLightingSetup() {
  const { t } = useTranslation();
  const copy = strings.lightDesk.settingsUi;
  const queryClient = useQueryClient();
  const status = useQuery({ queryKey: setupStatusKey, queryFn: () => lightingSetupApi.status(), retry: false });
  const [selection, setSelection] = useState<LightingSelection | null>(null);
  const [edited, setEdited] = useState(false);
  const [pending, setPending] = useState(false);
  const [devicePending, setDevicePending] = useState(false);
  const [deviceRecovery, setDeviceRecovery] = useState(false);
  const [actionError, setActionError] = useState("");
  const [uncertain, setUncertain] = useState(false);
  const observed = status.data;
  useEffect(() => {
    const selected = observed?.selection ?? observed?.pinnedSelection;
    if (selected && !edited) setSelection(create(LightingSelectionSchema, { ...selected, devicePolicyId: observed?.pinnedSelection?.devicePolicyId ?? selected.devicePolicyId }));
  }, [observed, edited]);
  const refreshStatus = async () => {
    const result = await status.refetch();
    if (!result.isError) {
      setUncertain(false);
      setDeviceRecovery(Boolean(result.data?.deviceAccessPending));
      const latest = result.data?.selection ?? result.data?.pinnedSelection;
      const policyId = result.data?.pinnedSelection?.devicePolicyId;
      if (latest && policyId !== undefined) {
        setSelection((current) => create(LightingSelectionSchema, { ...(current ?? latest), devicePolicyId: policyId }));
        setEdited((wasEdited) => wasEdited || !result.data?.setupVerified);
      }
    }
  };
  const changeSelection = (changes: Partial<LightingSelection>) => {
    if (!selection) return;
    setSelection(create(LightingSelectionSchema, { ...selection, ...changes }));
    setEdited(true);
    setActionError("");
  };
  const chooseSource = (source: InstallationSource) => {
    const pinned = observed?.pinnedSelection;
    setSelection(create(LightingSelectionSchema, {
      source, version: pinned?.version ?? "", sha256: source === InstallationSource.MANAGED ? pinned?.sha256 ?? "" : "",
      devicePolicyId: pinned?.devicePolicyId ?? selection?.devicePolicyId ?? "",
    }));
    setEdited(true);
    setActionError("");
  };
  const execute = async (action: () => Promise<LightingRuntimeStatus>) => {
    setPending(true);
    setActionError("");
    try {
      const result = await action();
      queryClient.setQueryData(setupStatusKey, result);
      setEdited(false);
      setUncertain(false);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : t(copy.ownerActionFailed));
      setUncertain(true);
    } finally { setPending(false); }
  };
  const validSelection = selection !== null && selection.version.trim() !== "" &&
    (selection.source === InstallationSource.MANAGED ? selection.version.trim() !== "" && /^[a-fA-F0-9]{64}$/.test(selection.sha256) : selection.source === InstallationSource.EXISTING && (selection.sha256 === "" || /^[a-fA-F0-9]{64}$/.test(selection.sha256)));
  const disabled = observed?.phase === "disabled";
  const actionsBlocked = pending || devicePending || status.isFetching || status.isError || uncertain;
  const setupBlocked = actionsBlocked || deviceRecovery || Boolean(observed?.deviceAccessPending);
  return <section className="lightdesk-panel" aria-labelledby="setup-heading" aria-busy={pending || status.isFetching}>
    <div className="lightdesk-panel-heading"><h2 id="setup-heading">{t(copy.setupTitle)}</h2><button className="lightdesk-button lightdesk-button--small" disabled={pending || status.isFetching} onClick={() => void refreshStatus()}><RefreshCw size={15} aria-hidden="true"/>{t(copy.ownerRefresh)}</button></div>
    <p className="lightdesk-muted">{t(copy.ownerIntro)}</p>
    {status.isLoading ? <p role="status">{t(copy.ownerLoading)}</p> : status.isError ? <div role="alert" className="lightdesk-alert">{t(copy.ownerStatusFailed, { message: status.error.message })}<button className="lightdesk-button" onClick={() => void refreshStatus()}>{t(copy.ownerRetry)}</button></div> : observed && <>
      <dl className="lightdesk-inspector-list">
        <div><dt>{t(copy.intentPhase)}</dt><dd>{observed.phase || t(copy.notReported)}</dd></div>
        <div><dt>{t(copy.ownerState)}</dt><dd>{observed.state || t(copy.notReported)}</dd></div>
        <div><dt>{t(copy.setupVerified)}</dt><dd>{t(observed.setupVerified ? copy.yes : copy.no)}</dd></div>
        <div><dt>{t(copy.installed)}</dt><dd>{observed.installed ? t(copy.yes) : !observed.resourceObserved ? t(copy.unobserved) : t(copy.no)}</dd></div>
        <div><dt>{t(copy.processRunning)}</dt><dd>{!observed.resourceObserved ? t(copy.unobserved) : t(observed.running ? copy.yes : copy.no)}</dd></div>
        <div><dt>{t(copy.sdkReady)}</dt><dd>{!observed.resourceObserved ? t(copy.unobserved) : t(observed.ready ? copy.yes : copy.no)}</dd></div>
      </dl>
      {observed.message && <p role="status">{observed.message}</p>}
      {observed.launchBlocker && <p className="lightdesk-alert" role="status">{t(copy.launchBlocked, { message: observed.launchBlocker })}</p>}
      {observed.ready && <p className="lightdesk-muted">{t(copy.physicalUnverified)}</p>}
      {selection && <fieldset className="lightdesk-settings-form" disabled={setupBlocked || !disabled}>
        <legend>{t(copy.selectionTitle)}</legend>
        <label className="lightdesk-select-label">{t(copy.installationSource)}<select value={selection.source} onChange={(event) => chooseSource(event.target.value === String(InstallationSource.MANAGED) ? InstallationSource.MANAGED : InstallationSource.EXISTING)}>
          <option value={InstallationSource.MANAGED}>{t(copy.managedSource)}</option><option value={InstallationSource.EXISTING}>{t(copy.existingSource)}</option>
        </select></label>
        <label className="lightdesk-field">{t(copy.selectedVersion)}<input maxLength={64} readOnly={selection.source === InstallationSource.MANAGED} value={selection.version} onChange={(event) => changeSelection({ version: event.target.value })}/></label>
        <label className="lightdesk-field">{t(copy.selectedHash)}<input maxLength={64} readOnly={selection.source === InstallationSource.MANAGED} spellCheck={false} autoComplete="off" value={selection.sha256} onChange={(event) => changeSelection({ sha256: event.target.value })}/></label>
        <div><span>{t(copy.devicePolicy)}</span><p className="lightdesk-muted">{selection.devicePolicyId || t(copy.policyNotEnrolled)}</p></div>
        <p className="lightdesk-muted">{t(selection.source === InstallationSource.EXISTING ? copy.existingHint : copy.managedHint)}</p>
      </fieldset>}
      <LightingDeviceAccess
        disabled={pending || status.isFetching || uncertain || !disabled}
        currentPlanId={observed.deviceAccessPlanId}
        currentPlanPending={observed.deviceAccessPending}
        onBusyChange={setDevicePending}
        onRecoveryChange={setDeviceRecovery}
        onPolicyChange={(policyId) => { changeSelection({ devicePolicyId: policyId }); void refreshStatus(); }}
      />
      <div className="lightdesk-editor-actions">
        <button className="lightdesk-button" disabled={setupBlocked || !disabled || !validSelection} onClick={() => selection && void execute(() => lightingSetupApi.setup(selection))}>{t(copy.prepareSetup)}</button>
        <button className="lightdesk-button lightdesk-button--primary" disabled={setupBlocked || !disabled || !observed.setupVerified || !validSelection || !selection.devicePolicyId.trim() || edited || !observed.selection} onClick={() => { const selected = observed.selection; if (selected) void execute(() => lightingSetupApi.enable(selected)); }}>{t(copy.enableLighting)}</button>
        <button className="lightdesk-button" disabled={actionsBlocked || !(observed.phase === "enabled" || observed.phase === "disabling")} onClick={() => void execute(() => lightingSetupApi.disable())}>{t(copy.disableLighting)}</button>
      </div>
      <p className="lightdesk-muted">{t(copy.setupDoesNotEnable)}</p>
    </>}
    {pending && <p role="status">{t(copy.ownerWorking)}</p>}
    {actionError && <div role="alert" className="lightdesk-alert">{actionError}<p>{t(copy.refreshAfterError)}</p><button className="lightdesk-button" disabled={pending || status.isFetching} onClick={() => void refreshStatus()}>{t(copy.ownerRetry)}</button></div>}
  </section>;
}


interface LightingDeviceAccessProps {
  disabled: boolean;
  currentPlanId: string;
  currentPlanPending: boolean;
  onBusyChange: (busy: boolean) => void;
  onRecoveryChange: (required: boolean) => void;
  onPolicyChange: (policyId: string) => void;
}

function LightingDeviceAccess({ disabled, currentPlanId, currentPlanPending, onBusyChange, onRecoveryChange, onPolicyChange }: LightingDeviceAccessProps) {
  const { t } = useTranslation();
  const copy = strings.lightDesk.settingsUi;
  const [discovery, setDiscovery] = useState<LightingDeviceDiscovery | null>(null);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [plan, setPlan] = useState<LightingDevicePlan | null>(null);
  const [result, setResult] = useState<LightingDeviceAccessResult | null>(null);
  const [appliedPlanId, setAppliedPlanId] = useState(currentPlanPending ? "" : currentPlanId);
  const [recoveryPlanId, setRecoveryPlanId] = useState(currentPlanPending ? currentPlanId : "");
  useEffect(() => {
    setAppliedPlanId(currentPlanPending ? "" : currentPlanId);
    setRecoveryPlanId(currentPlanPending ? currentPlanId : "");
  }, [currentPlanId, currentPlanPending]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [uncertain, setUncertain] = useState(false);
  const perform = async (action: () => Promise<void>, mutationPlanId?: string) => {
    setBusy(true);
    onBusyChange(true);
    setError("");
    try { await action(); }
    catch (failure) {
      setError(failure instanceof Error ? failure.message : t(copy.ownerActionFailed));
      setUncertain(true);
      if (mutationPlanId) {
        onRecoveryChange(true);
        const detail = ConnectError.from(failure).findDetails(LightingDeviceAccessResultSchema).find((receipt) => receipt.planId === mutationPlanId);
        if (detail) {
          setResult(detail);
          if (detail.state === "pending" || detail.state === "applied") setRecoveryPlanId(detail.planId);
        }
      }
    }
    finally {
      setBusy(false);
      onBusyChange(false);
    }
  };
  const discover = () => perform(async () => {
    const next = await lightingSetupApi.devices();
    setDiscovery(next);
    const retained = plan && (uncertain || recoveryPlanId === plan.id);
    setSelectedIds(retained ? plan.devices.map((device) => device.id) : next.candidates.filter((candidate) => candidate.selected).map((candidate) => candidate.id));
    if (!retained) setPlan(null);
    setUncertain(false);
  });
  const selectDevice = (id: string, selected: boolean) => {
    setSelectedIds((current) => selected ? [...current, id] : current.filter((value) => value !== id));
    setPlan(null);
    setResult(null);
  };
  const planDevices = () => perform(async () => {
    const next = await lightingSetupApi.planDevices(selectedIds);
    if (!next.id) throw new Error(t(copy.devicePlanMissing));
    setPlan(next);
    setResult(null);
  });
  const apply = () => perform(async () => {
    if (!plan) return;
    const next = await lightingSetupApi.applyDevices(plan.id);
    if (next.planId !== plan.id) throw new Error(t(copy.deviceReceiptMismatch));
    setResult(next);
    if (next.state === "applied" && !next.needsSudo && next.devicePolicyId) {
      setAppliedPlanId(plan.id);
      setRecoveryPlanId("");
      onRecoveryChange(false);
      onPolicyChange(next.devicePolicyId);
    }
  }, plan?.id);
  const rollback = () => {
    const target = recoveryPlanId || appliedPlanId;
    return perform(async () => {
      const next = await lightingSetupApi.rollbackDevices(target);
      if (next.planId !== target) throw new Error(t(copy.deviceReceiptMismatch));
      setResult(next);
      if ((next.state === "rolled_back" || next.state === "unchanged") && !next.needsSudo) {
        setAppliedPlanId("");
        setRecoveryPlanId("");
        setPlan(null);
        onRecoveryChange(false);
        onPolicyChange(next.devicePolicyId);
      }
    }, target);
  };
  const blocked = disabled || busy;
  return <section aria-labelledby="lighting-device-access-heading" aria-busy={busy}>
    <h3 id="lighting-device-access-heading">{t(copy.deviceAccessTitle)}</h3>
    <p className="lightdesk-muted">{t(copy.deviceDiscoveryHint)}</p>
    <button className="lightdesk-button" disabled={blocked} onClick={() => void discover()}>{t(copy.discoverDevices)}</button>
    {discovery && <>
      {discovery.message && <p role="status">{discovery.message}</p>}
      {discovery.candidates.length === 0 ? <p>{t(copy.deviceListEmpty)}</p> : <fieldset disabled={blocked || uncertain || recoveryPlanId !== ""}>
        <legend>{t(copy.deviceSelectionTitle)}</legend>
        {discovery.candidates.map((candidate) => <label key={candidate.id} className="lightdesk-check">
          <input type="checkbox" checked={selectedIds.includes(candidate.id)} onChange={(event) => selectDevice(candidate.id, event.target.checked)}/>
          <span>{candidate.label}<small className="lightdesk-muted"> {candidate.kind} {candidate.detail}</small></span>
        </label>)}
      </fieldset>}
      <button className="lightdesk-button" disabled={blocked || uncertain || recoveryPlanId !== "" || selectedIds.length === 0 || selectedIds.length > 32} onClick={() => void planDevices()}>{t(copy.reviewDevicePlan)}</button>
    </>}
    {plan && <div className="lightdesk-setup-guide">
      <h4>{t(copy.devicePlanTitle)}</h4>
      <p>{plan.message}</p>
      <ul>{plan.devices.map((device) => <li key={device.id}>{device.label}</li>)}</ul>
      <ul>{plan.changes.map((change, index) => <li key={index}>{change}</li>)}</ul>
      {plan.warnings.length > 0 && <div className="lightdesk-alert"><ul>{plan.warnings.map((warning, index) => <li key={index}>{warning}</li>)}</ul></div>}
      {plan.requiresPrivilege && <p>{t(copy.devicePrivilegeRequired)}</p>}
      <button className="lightdesk-button" disabled={blocked || uncertain || appliedPlanId === plan.id} onClick={() => void apply()}>{t(copy.applyDevicePlan)}</button>
    </div>}
    {result && <div role="status" className={result.needsSudo ? "lightdesk-alert" : "lightdesk-operation"}>
      <p>{t(copy.deviceOutcome, { state: result.state })}</p>
      <p>{result.message}</p>
      {result.needsSudo && <p>{t(copy.deviceNeedsSudo)}</p>}
    </div>}
    {(recoveryPlanId || appliedPlanId) && <button className="lightdesk-button" disabled={blocked || uncertain} onClick={() => void rollback()}>{t(copy.rollbackDevicePlan)}</button>}
    {busy && <p role="status">{t(copy.deviceWorking)}</p>}
    {error && <div role="alert" className="lightdesk-alert">{error}<p>{t(copy.deviceRefreshAfterError)}</p><button className="lightdesk-button" disabled={blocked} onClick={() => void discover()}>{t(copy.retryDevices)}</button></div>}
  </section>;
}
