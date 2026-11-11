import { clone, create } from "@bufbuild/protobuf";
import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Check, Power, Save, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";
import { ActivityEffectsSchema, HealthPaletteSchema, PolicyConfigSchema, PresetSchema, PresetKind, EvidenceState, type ActivityEffects, type HealthPalette, type PolicyConfig, type Preset } from "@vrooli/proto-types/lighting-control/v1/policy/policy_pb";
import { ColorSchema } from "@vrooli/proto-types/lighting-control/v1/control/control_pb";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { lightingApi } from "../features/lights/api";
import { useLightDesk } from "../features/lights/useLightDesk";

const kinds = [PresetKind.OFF, PresetKind.LOAD, PresetKind.WORK, PresetKind.HEALTH, PresetKind.COMBINED] as const;
const DRAFT_KEY = "lighting-control.preset-drafts.v1";
type Guardrails = Pick<PolicyConfig, "sourceMaxAgeSeconds" | "loadEnterPercent" | "loadExitPercent" | "debounceSeconds" | "staleFallbackColor" | "staleFallbackBrightnessPercent">;
type PresetDraft = { preset: Preset; health: HealthPalette; activity: ActivityEffects; guardrails?: Guardrails; respectReducedMotion?: boolean };

export function PresetsPage() {
  const { t } = useTranslation();
  const copy = strings.lightDesk.presetsUi;
  const phone = useMediaQuery("(max-width: 720px)");
  const { desk, run, busy, refresh } = useLightDesk();
  const config = desk.data?.config;
  const [selected, setSelected] = useState<PresetKind>(PresetKind.COMBINED);
  const [draft, setDraft] = useState<Preset | null>(null);
  const [healthDraft, setHealthDraft] = useState<HealthPalette | null>(null);
  const [activityDraft, setActivityDraft] = useState<ActivityEffects | null>(null);
  const [guardrailDraft, setGuardrailDraft] = useState<Guardrails | null>(null);
  const [respectReducedMotion, setRespectReducedMotion] = useState(true);
  const reducedMotion = useMediaQuery("(prefers-reduced-motion: reduce)");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  useEffect(() => {
    if (!config || draft) return;
    try {
      const drafts = readDrafts();
      const saved = drafts[selected];
      if (saved) {
        setDraft(create(PresetSchema, saved.preset));
        setHealthDraft(create(HealthPaletteSchema, saved.health));
        setActivityDraft(create(ActivityEffectsSchema, saved.activity));
        setGuardrailDraft(saved.guardrails ?? null);
        setRespectReducedMotion(saved.respectReducedMotion ?? true);
      }
    } catch { /* The saved policy remains available when browser storage is blocked. */ }
  }, [config, selected, draft]);
  const selectedPreset = useMemo(() => config?.presets.find((preset) => preset.kind === selected), [config, selected]);
  const current = draft ?? (selectedPreset ? create(PresetSchema, selectedPreset) : null);
  const health = healthDraft ?? config?.healthPalette ?? create(HealthPaletteSchema, {
    healthy: create(ColorSchema, { red: 87, green: 190, blue: 138 }),
    attention: create(ColorSchema, { red: 217, green: 154, blue: 36 }),
    failure: create(ColorSchema, { red: 217, green: 77, blue: 77 }),
  });
  const activity = activityDraft ?? config?.activityEffects ?? create(ActivityEffectsSchema, { activeEffectId: "Breathing", busyEffectId: "Breathing", busyAgentThreshold: 5 });

  const guardrails = guardrailDraft ?? (config ? policyGuardrails(config) : null);
  const editGuardrails = (changes: Partial<Guardrails>) => { if (guardrails) { setGuardrailDraft({ ...guardrails, ...changes }); setNotice(""); } };
  const editActivity = (changes: Partial<ActivityEffects>) => { setActivityDraft(create(ActivityEffectsSchema, { ...activity, ...changes })); setNotice(""); };
  const effectOptions = [activity.activeEffectId, activity.busyEffectId, ...(desk.data?.controllers.flatMap((controller) => controller.zones.flatMap((zone) => zone.supportedEffects.map((effect) => effect.trim()))) ?? [])].filter((effect, index, all) => effect.trim() && all.findIndex((item) => item.trim().toLowerCase() === effect.trim().toLowerCase()) === index);
  const validGuardrails = !guardrails || guardrails.sourceMaxAgeSeconds >= 5;

  const edit = (changes: Partial<Preset>) => {
    if (!selectedPreset) return;
    setDraft(create(PresetSchema, { ...(draft ?? selectedPreset), ...changes }));
    setNotice("");
  };
  const save = async (useNow: boolean) => {
    if (!config || !current) return;
    setError(""); setNotice("");
    const savedDraft: PresetDraft = {
      preset: create(PresetSchema, current),
      health: create(HealthPaletteSchema, health),
      activity: create(ActivityEffectsSchema, activity),
      guardrails: guardrails ?? undefined,
      respectReducedMotion,
    };
    if (!useNow) {
      try {
        const drafts = readDrafts();
        drafts[selected] = savedDraft;
        localStorage.setItem(DRAFT_KEY, JSON.stringify(drafts));
        setDraft(savedDraft.preset);
        setHealthDraft(savedDraft.health);
        setActivityDraft(savedDraft.activity);
        setNotice(t(copy.savedDraft));
      } catch {
        setError(t(copy.draftStorageFailed));
      }
      return;
    }
    const next = clone(PolicyConfigSchema, config);
    if (savedDraft.guardrails) Object.assign(next, savedDraft.guardrails);
    next.presets = next.presets.map((preset) => preset.kind === selected ? savedDraft.preset : preset);
    if (selected === PresetKind.HEALTH || selected === PresetKind.COMBINED) next.healthPalette = savedDraft.health;
    if (selected === PresetKind.WORK || selected === PresetKind.COMBINED) next.activityEffects = savedDraft.activity;
    const progress = { configurationSaved: false };
    try {
      await run(async () => {
        await lightingApi.savePolicy(next);
        progress.configurationSaved = true;
        await lightingApi.choosePreset(selected);
      });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t(progress.configurationSaved ? copy.applyFailed : copy.saveFailed));
      if (progress.configurationSaved) {
        setNotice(t(copy.savedButNotApplied));
        void refresh();
      }
      return;
    }
    try {
      const drafts = readDrafts();
      localStorage.setItem(DRAFT_KEY, JSON.stringify(Object.fromEntries(Object.entries(drafts).filter(([kind]) => kind !== String(selected)))));
    } catch { /* A local cleanup failure does not undo the remote result. */ }
    setDraft(null);
    setHealthDraft(null);
    setActivityDraft(null);
    setGuardrailDraft(null);
    setNotice(t(copy.savedUsed));
  };
  const cancelEdits = () => {
    if (!config || !selectedPreset) return;
    let saved: PresetDraft | undefined;
    try { saved = readDrafts()[selected]; } catch { /* Restore the remote policy if local drafts cannot be read. */ }
    setDraft(create(PresetSchema, saved?.preset ?? selectedPreset));
    setHealthDraft(create(HealthPaletteSchema, saved?.health ?? config.healthPalette ?? health));
    setActivityDraft(create(ActivityEffectsSchema, saved?.activity ?? config.activityEffects ?? activity));
    setGuardrailDraft(saved?.guardrails ?? policyGuardrails(config));
    setRespectReducedMotion(saved?.respectReducedMotion ?? true);
    setError(""); setNotice("");
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

  if (desk.isLoading) return <section className="lightdesk-page"><PageHeader headingId="presets-heading" title={t(strings.pages.presets.title)} description={t(strings.pages.presets.description)}/><p role="status">{t(copy.loading)}</p></section>;
  if (desk.isError || !desk.data || !config) return <section className="lightdesk-page"><PageHeader headingId="presets-heading" title={t(strings.pages.presets.title)} description={t(strings.pages.presets.description)}/><div role="alert" className="lightdesk-alert"><AlertTriangle aria-hidden="true"/><span>{desk.error?.message ?? t(copy.unavailable)}</span></div></section>;

  const manualOff = desk.data.manualOverride?.active && desk.data.manualOverride.allEnrolled;
  const activePreset = manualOff ? PresetKind.OFF : desk.data.effective?.preset;
  const offAction = <button data-testid={selectors.lightDesk.manualOff} className="lightdesk-button lightdesk-button--danger" disabled={busy || !desk.data.controllers.some((controller) => controller.enrolled) || manualOff} onClick={() => void turnOff()}><Power size={16} aria-hidden="true"/>{t(copy.keepOff)}</button>;
  const editorActions = current ? <div data-testid={selectors.lightDesk.presetActions} className="lightdesk-editor-actions"><button className="lightdesk-button" disabled={busy || !current.label.trim() || !validGuardrails} onClick={() => void save(false)}><Save size={16} aria-hidden="true"/>{t(copy.saveDraft)}</button><button className="lightdesk-button lightdesk-button--primary" disabled={busy || !current.label.trim() || !current.enabled || !validGuardrails} onClick={() => void save(true)}>{t(copy.saveUse)}</button></div> : null;
  return <section data-testid={selectors.pages.presets} aria-labelledby="presets-heading" className="lightdesk-page">
    <PageHeader headingId="presets-heading" title={t(strings.pages.presets.title)} description={t(strings.pages.presets.description)} />
    {manualOff ? <div className="lightdesk-alert lightdesk-alert--off"><span>{t(copy.manualOff)}</span><Link className="lightdesk-link" to="/">{t(copy.reviewOverview)}</Link></div> : null}
    {!phone ? <div className="lightdesk-page-actions">{offAction}</div> : null}
    {error ? <p role="alert" className="lightdesk-warning">{error}</p> : null}
    {notice ? <p role="status" className="lightdesk-notice">{notice}</p> : null}
    <div className="lightdesk-preset-layout">
      <ul className="lightdesk-preset-grid" aria-label={t(copy.listLabel)}>
        {kinds.map((kind) => {
          const preset = config.presets.find((item) => item.kind === kind);
          return <li key={kind}><button type="button" className={`lightdesk-preset-card ${selected === kind ? "is-selected" : ""}`} aria-pressed={selected === kind} disabled={busy} onClick={() => {
            if (selected === kind) return;
            setSelected(kind);
            setDraft(null);
            setHealthDraft(null);
            setActivityDraft(null);
            setGuardrailDraft(null);
            setRespectReducedMotion(true);
            setError("");
            setNotice("");
          }}>
            <span className="lightdesk-preset-swatch" style={{ background: kind === PresetKind.OFF ? "#222936" : `rgb(${preset?.color?.red ?? 72} ${preset?.color?.green ?? 128} ${preset?.color?.blue ?? 184})` }} />
            <span className="lightdesk-preset-copy"><strong>{preset?.label || t(presetNameKey(kind))}</strong><small>{t(descriptionKey(kind))}</small></span>
            {activePreset === kind ? <span className="lightdesk-badge"><Check size={14} aria-hidden="true"/>{t(copy.active)}</span> : null}
          </button></li>;
        })}
      </ul>
      {current ? <section className="lightdesk-panel lightdesk-editor" aria-labelledby="customize-heading">
        <div className="lightdesk-panel-heading"><Sparkles aria-hidden="true"/><h2 id="customize-heading">{t(copy.customize, { preset: t(presetNameKey(selected)) })}</h2></div>
        <p className="lightdesk-muted">{t(copy.draftHelp)}</p>
        <label className="lightdesk-field">{t(copy.presetName)}<input value={current.label} maxLength={48} onChange={(event) => edit({ label: event.target.value })}/></label>
        <label className="lightdesk-field">{t(copy.previewColor)}<input type="color" value={selected === PresetKind.OFF ? "#000000" : toHex(current.color)} disabled={selected === PresetKind.OFF} onChange={(event) => edit({ color: create(ColorSchema, fromHex(event.target.value)) })}/></label>
        <p className="lightdesk-meta"><code>{selected === PresetKind.OFF ? "#000000" : toHex(current.color).toUpperCase()}</code></p>
        <label className="lightdesk-field">{t(copy.brightnessCap, { percent: current.brightnessCapPercent })}<input type="range" min="0" max={config.brightnessCapPercent} value={Math.min(current.brightnessCapPercent, config.brightnessCapPercent)} onChange={(event) => edit({ brightnessCapPercent: Number(event.target.value) })}/></label>
        <label className="lightdesk-check"><input type="checkbox" checked={current.enabled} onChange={(event) => edit({ enabled: event.target.checked })}/>{t(copy.enable)}</label>
        {selected === PresetKind.LOAD ? <p className="lightdesk-muted">{t(copy.loadHint, { enter: config.loadEnterPercent, exit: config.loadExitPercent })}</p> : null}
        {selected === PresetKind.WORK || selected === PresetKind.COMBINED ? <p className="lightdesk-muted">{t(copy.activityHint)}</p> : null}
        {selected === PresetKind.WORK || selected === PresetKind.COMBINED ? <label className="lightdesk-field">{t(copy.busyAt, { count: activity.busyAgentThreshold })}<input aria-label={t(copy.activityThreshold)} type="range" min="2" max="5000" value={activity.busyAgentThreshold} onChange={(event) => editActivity({ busyAgentThreshold: Number(event.target.value) })}/></label> : null}
        {selected === PresetKind.HEALTH || selected === PresetKind.COMBINED ? <p className="lightdesk-muted">{t(copy.healthHint)}</p> : null}
        {selected === PresetKind.HEALTH || selected === PresetKind.COMBINED ? <fieldset className="lightdesk-fieldset"><legend>{t(copy.healthColors)}</legend>{(["healthy", "attention", "failure"] as const).map((key) => { const value = health[key]; return <label className="lightdesk-field" key={key}>{t(healthNameKey(key))}<input aria-label={t(copy.colorAria, { name: t(healthNameKey(key)) })} type="color" value={toHex(value)} onChange={(event) => setHealthDraft(create(HealthPaletteSchema, { ...health, [key]: create(ColorSchema, fromHex(event.target.value)) }))}/><code>{toHex(value).toUpperCase()}</code></label>; })}</fieldset> : null}
        {selected === PresetKind.WORK || selected === PresetKind.COMBINED ? <fieldset className="lightdesk-fieldset"><legend>{t(copy.activityEffects)}</legend>
          <p className="lightdesk-muted">{t(copy.noAgentsSteady)}</p>
          {(["activeEffectId", "busyEffectId"] as const).map((key) => <label className="lightdesk-field" key={key}>{t(key === "activeEffectId" ? copy.activeEffect : copy.busyEffect)}<select value={activity[key]} onChange={(event) => editActivity({ [key]: event.target.value })}><option value="">{t(copy.steady)}</option>{effectOptions.map((effect) => <option key={effect} value={effect}>{effect}</option>)}</select></label>)}
          <p className="lightdesk-muted">{t(copy.effectFallback)}</p>
          <label className="lightdesk-check"><input type="checkbox" checked={respectReducedMotion} onChange={(event) => setRespectReducedMotion(event.target.checked)}/>{t(copy.respectPreviewMotion)}</label>
          <p className="lightdesk-muted">{t(copy.physicalMotionHelp)}</p>
        </fieldset> : null}
        {guardrails && selected !== PresetKind.OFF ? <fieldset className="lightdesk-fieldset"><legend>{t(copy.brightnessFallback)}</legend>
          <p className="lightdesk-muted">{t(copy.sharedGuardrails)}</p>
          <label className="lightdesk-field">{t(copy.staleAfter)}<input type="number" min="5" max="86400" value={guardrails.sourceMaxAgeSeconds} aria-invalid={!validGuardrails} onChange={(event) => editGuardrails({ sourceMaxAgeSeconds: clampNumber(event.target.value, 0, 86400) })} onBlur={(event) => editGuardrails({ sourceMaxAgeSeconds: clampNumber(event.target.value, 5, 86400) })}/></label>
          <label className="lightdesk-field">{t(copy.fallbackColor)}<input aria-label={t(copy.fallbackColor)} type="color" value={toHex(guardrails.staleFallbackColor)} onChange={(event) => editGuardrails({ staleFallbackColor: create(ColorSchema, fromHex(event.target.value)) })}/><code>{toHex(guardrails.staleFallbackColor).toUpperCase()}</code></label>
          <label className="lightdesk-field">{t(copy.fallbackBrightness)}<input aria-label={t(copy.fallbackBrightness)} type="range" min="0" max={config.brightnessCapPercent} value={Math.min(guardrails.staleFallbackBrightnessPercent, config.brightnessCapPercent)} onChange={(event) => editGuardrails({ staleFallbackBrightnessPercent: Number(event.target.value) })}/><strong>{Math.min(guardrails.staleFallbackBrightnessPercent, config.brightnessCapPercent)}%</strong></label>
          <p className="lightdesk-muted">{t(copy.fallbackHelp)}</p>
          {selected === PresetKind.LOAD ? <details><summary>{t(copy.loadControls)}</summary>
            <label className="lightdesk-field">{t(copy.loadEnter)}<input type="number" min="0" max="100" value={guardrails.loadEnterPercent} onChange={(event) => { const enter = clampNumber(event.target.value, 0, 100); editGuardrails({ loadEnterPercent: enter, loadExitPercent: Math.min(guardrails.loadExitPercent, enter) }); }}/></label>
            <label className="lightdesk-field">{t(copy.loadExit)}<input type="number" min="0" max={guardrails.loadEnterPercent} value={guardrails.loadExitPercent} onChange={(event) => editGuardrails({ loadExitPercent: clampNumber(event.target.value, 0, guardrails.loadEnterPercent) })}/></label>
            <label className="lightdesk-field">{t(copy.debounceControl)}<input type="number" min="0" max="3600" value={guardrails.debounceSeconds} onChange={(event) => editGuardrails({ debounceSeconds: clampNumber(event.target.value, 0, 3600) })}/></label>
          </details> : null}
        </fieldset> : null}
        <button className="lightdesk-button lightdesk-button--small" disabled={busy} onClick={cancelEdits}>{t(copy.cancelEdits)}</button>
        <div className="lightdesk-editor-preview"><span className={`lightdesk-preview-dot ${([PresetKind.WORK, PresetKind.COMBINED] as PresetKind[]).includes(selected) && activity.activeEffectId && !(respectReducedMotion && reducedMotion) ? "lightdesk-preview-dot--pulse" : ""}`} style={{ background: `rgb(${current.color?.red ?? 0} ${current.color?.green ?? 0} ${current.color?.blue ?? 0})` }}/><span>{t(copy.previewOnly)}</span></div>
        {!phone ? editorActions : null}
        <p className="lightdesk-muted">{t(copy.replacingOff)}</p>
      </section> : <p>{t(copy.missingPreset)}</p>}
    </div>
    <section className="lightdesk-panel lightdesk-evidence" aria-labelledby="preset-source-evidence"><h2 id="preset-source-evidence">{t(strings.lightDesk.sourceEvidence)}</h2>{desk.data.effective?.evidence.length ? desk.data.effective.evidence.map((source) => { const stamp = source.sourceTimestamp ?? source.observedAt; const age = stamp ? Math.max(0, Math.floor(Date.now() / 1000 - Number(stamp.seconds))) : null; return <div className="lightdesk-evidence-row" key={source.source}><strong>{source.source}</strong><span className={source.state === EvidenceState.FRESH && age !== null && age <= config.sourceMaxAgeSeconds ? "is-fresh" : "is-stale"}>{age === null ? t(copy.unknownAge) : t(copy.sourceAge, { age, threshold: config.sourceMaxAgeSeconds })}</span><small>{source.reason}</small></div>; }) : <p className="lightdesk-muted">{t(strings.lightDesk.noEvidence)}</p>}</section>
    <section className="lightdesk-panel lightdesk-policy-settings" aria-labelledby="policy-guardrails"><h2 id="policy-guardrails">{t(copy.guardrails)}</h2><div className="lightdesk-settings-grid">
      <Metric label={t(copy.telemetryStale)} value={t(copy.seconds, { count: config.sourceMaxAgeSeconds })} />
      <Metric label={t(copy.maximumBrightness)} value={`${config.brightnessCapPercent}%`} />
      <Metric label={t(copy.loadThresholds)} value={t(copy.percentagePair, { enter: config.loadEnterPercent, exit: config.loadExitPercent })} />
      <Metric label={t(copy.debounce)} value={t(copy.seconds, { count: config.debounceSeconds })} />
      <Metric label={t(copy.staleBrightness)} value={`${config.staleFallbackBrightnessPercent}%`} />
    </div><Link className="lightdesk-link" to="/settings">{t(copy.editGuardrails)}</Link></section>
    {phone ? <div className="lightdesk-mobile-footer">{editorActions}{offAction}</div> : null}
  </section>;
}

function policyGuardrails(config: PolicyConfig): Guardrails { return { sourceMaxAgeSeconds: config.sourceMaxAgeSeconds, loadEnterPercent: config.loadEnterPercent, loadExitPercent: config.loadExitPercent, debounceSeconds: config.debounceSeconds, staleFallbackColor: config.staleFallbackColor ? create(ColorSchema, config.staleFallbackColor) : undefined, staleFallbackBrightnessPercent: config.staleFallbackBrightnessPercent }; }
function clampNumber(value: string, min: number, max: number) { return Math.min(max, Math.max(min, Math.round(Number(value) || 0))); }

function readDrafts(): Partial<Record<number, PresetDraft>> {
  const raw = localStorage.getItem(DRAFT_KEY);
  try {
    const value: unknown = JSON.parse(raw ?? "{}");
    return value && typeof value === "object" && !Array.isArray(value) ? value : {};
  } catch {
    return {};
  }
}

function Metric({ label, value }: { label: string; value: string }) { return <div className="lightdesk-metric"><span>{label}</span><strong>{value}</strong></div>; }
function presetNameKey(value: PresetKind) { return value === PresetKind.OFF ? strings.lightDesk.presets.off : value === PresetKind.LOAD ? strings.lightDesk.presets.load : value === PresetKind.WORK ? strings.lightDesk.presets.work : value === PresetKind.HEALTH ? strings.lightDesk.presets.health : strings.lightDesk.presets.combined; }
function descriptionKey(value: (typeof kinds)[number]) { return value === PresetKind.OFF ? strings.lightDesk.presetDescription.off : value === PresetKind.LOAD ? strings.lightDesk.presetDescription.load : value === PresetKind.WORK ? strings.lightDesk.presetDescription.work : value === PresetKind.HEALTH ? strings.lightDesk.presetDescription.health : strings.lightDesk.presetDescription.combined; }
function healthNameKey(value: "healthy" | "attention" | "failure") { return value === "healthy" ? strings.lightDesk.presetsUi.healthy : value === "attention" ? strings.lightDesk.presetsUi.attention : strings.lightDesk.presetsUi.failure; }
function toHex(color?: { red: number; green: number; blue: number }) { return color ? `#${[color.red, color.green, color.blue].map((v) => Math.max(0, Math.min(255, v)).toString(16).padStart(2, "0")).join("")}` : "#57be8a"; }
function fromHex(value: string) { return { red: parseInt(value.slice(1, 3), 16), green: parseInt(value.slice(3, 5), 16), blue: parseInt(value.slice(5, 7), 16) }; }
