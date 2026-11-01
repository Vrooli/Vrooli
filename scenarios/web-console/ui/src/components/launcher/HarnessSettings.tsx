/**
 * The settings panel for the selected harness. Rows appear only when that
 * harness supports them, and every value states what will really happen: an
 * unknown window is stated, a limit that cannot reach the command says so, and a
 * usage mode's lowering shows in the model row.
 */
import type { ReactNode } from "react";
import { Card } from "@vrooli/react-component-library/Card";
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { Switch } from "@vrooli/react-component-library/Switch";
import { Select } from "@vrooli/react-component-library/Select";
import type { RoleModelChoice } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";
import { TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";

import type { UsageModeAgent, UsageModeView } from "../../api/usageMode";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { formatClockTime, formatLongDuration, formatTokens } from "../../lib/localeFormat";
import { slugify } from "../../lib/slugify";
import type { AgentCard } from "./agentGrid";
import { FOCUS_RING, SELECT_TEXT } from "./controls";
import type { AgentActionRun } from "./useAgentUpdates";
import type { BillingAccountController } from "./useBillingAccount";
import type { Harness } from "./useHarnesses";
import { choiceBlocked, limitOptions, modelLabel, roleChoiceKey, type ModelPolicyController } from "./useModelPolicy";

const LOWERED_VALUE = "__launcher_lowered__";
const HINT = "text-xs leading-[18px] text-wc-text-muted";
const FIELD_GRID = "grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2";

function Row({ children, testId }: { children: ReactNode; testId?: string }) {
  return <div data-testid={testId} className="grid gap-2 border-b border-wc-default px-3 py-2.5 last:border-b-0">{children}</div>;
}

/** Label and control on one line. The control fills the rest, so its list never covers the label. */
function Field({ label, labelId, children }: { label: string; labelId?: string; children: ReactNode }) {
  return (
    <div className={FIELD_GRID}>
      <span id={labelId} className="text-sm font-medium text-wc-text-primary">{label}</span>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

const Value = ({ children, testId }: { children: ReactNode; testId?: string }) => (
  <span data-testid={testId} className="block min-h-11 truncate text-end text-[13px] leading-[44px] text-wc-text-muted">{children}</span>
);

function ModelRows({ card, model, canChoose, canChooseProvider, saveAgent, fullPower, onFullPower, onOpenDetail, refresh }: {
  refresh: CatalogRefresh;
  card: AgentCard;
  model: ModelPolicyController;
  canChoose: boolean;
  canChooseProvider: boolean;
  saveAgent?: UsageModeAgent;
  fullPower: boolean;
  onFullPower: (on: boolean) => void;
  onOpenDetail: () => void;
}) {
  const { t, i18n } = useTranslation();
  const { policy, choices, sources, sourceChoices, explicitChoice, defaultChoice, stale } = model;
  const selectable = model.selectable && canChoose && !model.literal;
  const lowered = saveAgent?.model && !explicitChoice && !fullPower ? saveAgent.model : null;
  const usualName = modelLabel(model, saveAgent?.usualModel);
  const loweredName = modelLabel(model, lowered ?? undefined);
  const defaultName = loweredName ?? (fullPower ? usualName : undefined) ?? defaultChoice?.displayName;

  // Each model once. The default is marked, never listed a second time beside itself.
  const choiceLabel = (choice: RoleModelChoice) => choice.isDefault ? t(strings.terminalLauncher.modelIsDefault, { model: choice.displayName }) : choice.displayName;
  const loweredChoice = lowered ? choices.find((choice) => choice.model === lowered) : undefined;
  let currentValue = explicitChoice ? roleChoiceKey(explicitChoice) : defaultChoice ? roleChoiceKey(defaultChoice) : "";
  if (lowered) currentValue = loweredChoice ? roleChoiceKey(loweredChoice) : LOWERED_VALUE;
  const modelOptions = [
    ...(lowered && !loweredChoice ? [{ value: LOWERED_VALUE, label: lowered }] : []),
    ...sourceChoices.map((choice) => ({ value: roleChoiceKey(choice), label: choiceLabel(choice), disabled: choiceBlocked(choice, stale) })),
  ];

  let control: ReactNode;
  let hint: string | undefined;
  if (model.loading) control = <Value testId="launcher-model-loading">{t(strings.terminalLauncher.modelChoicesLoading)}</Value>;
  else if (selectable && !fullPower) {
    control = (
      <Select className={SELECT_TEXT} aria-label={t(strings.terminalLauncher.modelChoice)} data-testid="launcher-model-choice" value={currentValue} options={modelOptions}
        onValueChange={(value) => {
          if (value === LOWERED_VALUE || value === currentValue) return;
          // Choosing the default with no mode lowering it needs no override. Under
          // a mode it does: an explicit model is what beats the mode.
          model.choose(defaultChoice && value === roleChoiceKey(defaultChoice) && !saveAgent?.model ? null : value);
        }} />
    );
  } else {
    control = <Value testId="launcher-model-value">{model.literal ? t(strings.terminalLauncher.modelSetByCommand) : defaultName ?? t(strings.terminalLauncher.modelDefault)}</Value>;
    if (model.literal) hint = t(strings.launcher.settingsDontApply, { name: card.label });
    else if (model.failed || !policy || policy.policyStatus === "unknown") hint = t(strings.terminalLauncher.modelChoicesUnknown);
    else if (policy.policyStatus === "unsupported" || choices.length === 0) hint = policy.detail || undefined;
  }

  const age = policy?.observedAt ? Date.now() - Date.parse(policy.observedAt) : NaN;
  const hasMetered = choices.some((choice) => choice.metered);
  const offerFullPower = !explicitChoice && !model.literal && saveAgent?.model && saveAgent.fullPowerArgs;
  const refreshRun = refresh.run;
  const refreshing = refreshRun?.phase === "running";
  const providerSource = explicitChoice?.source ?? defaultChoice?.source ?? "";

  return (
    <>
      {stale && !model.loading && (
        <div data-testid="launcher-catalog-stale" className="flex items-center gap-2 border-b border-wc-default bg-wc-warning/10 px-3 py-1 text-xs leading-[18px] text-wc-warning">
          <TriangleAlert className="h-3.5 w-3.5 flex-none" aria-hidden />
          <span className="min-w-0 flex-1 py-1.5">
            {Number.isFinite(age) ? t(strings.terminalLauncher.modelListAge, { age: formatLongDuration(age, i18n.resolvedLanguage) }) : t(strings.terminalLauncher.modelListOld)}
            {hasMetered && ` ${t(strings.terminalLauncher.modelListMeteredOff)}`}
          </span>
          {policy?.operatorCommand
            ? <button type="button" data-testid="launcher-catalog-fix" onClick={onOpenDetail} className={cn("min-h-11 flex-none rounded-[var(--radius-control)] px-1 font-semibold underline underline-offset-2", FOCUS_RING)}>{t(strings.terminalLauncher.modelListDetails)}</button>
            : <button type="button" data-testid="launcher-catalog-refresh" disabled={refreshing} onClick={refresh.start} className={cn("min-h-11 flex-none rounded-[var(--radius-control)] px-1 font-semibold underline underline-offset-2 disabled:no-underline disabled:opacity-70", FOCUS_RING)}>{refreshing ? t(strings.terminalLauncher.modelListRefreshing) : t(strings.terminalLauncher.modelListRefresh)}</button>}
        </div>
      )}
      {refreshRun?.phase === "failed" && <p role="alert" data-testid="launcher-catalog-refresh-error" className="break-words border-b border-wc-default px-3 py-2 text-xs leading-[18px] text-wc-error-text">{refreshRun.message || t(strings.launcher.actionFailed)}</p>}
      {selectable && canChooseProvider && !fullPower && sources.length > 1 && (
        <Row>
          <Field label={t(strings.terminalLauncher.modelSource)}>
            <Select className={SELECT_TEXT} aria-label={t(strings.terminalLauncher.modelSource)} data-testid="launcher-model-source" value={providerSource}
              options={sources.map((source) => {
                const inSource = choices.filter((choice) => choice.source === source);
                return { value: source, label: inSource[0]?.sourceLabel || source, disabled: inSource.every((choice) => choiceBlocked(choice, stale)) };
              })}
              onValueChange={(value) => {
                if (value === defaultChoice?.source) { model.choose(null); return; }
                const next = choices.find((choice) => choice.source === value && !choiceBlocked(choice, stale));
                if (next) model.choose(roleChoiceKey(next));
              }} />
          </Field>
        </Row>
      )}
      <Row testId="launcher-model-selection">
        <Field label={t(strings.terminalLauncher.modelChoice)}>{control}</Field>
        {hint && <span className={HINT} role="status">{hint}</span>}
        {saveAgent?.learnModeId && !saveAgent.model && <span className={HINT} data-testid={`launcher-mode-note-${slugify(card.key)}`}>{t(strings.launcher.usageModeLearnArm)}</span>}
        {lowered && saveAgent && (
          <span className={HINT} data-testid={`launcher-mode-note-${slugify(card.key)}`}>{t(strings.launcher.usageModeLowers, { usual: usualName ?? saveAgent.usualModel, lower: loweredName ?? lowered })}</span>
        )}
        {fullPower && saveAgent && <span className={HINT}>{t(strings.launcher.usageModeFullPowerHint, { model: usualName ?? saveAgent.usualModel })}</span>}
        {offerFullPower && (
          <button type="button" data-testid={`launcher-full-power-${slugify(card.key)}`} aria-pressed={fullPower} onClick={() => { onFullPower(!fullPower); }}
            className={cn("min-h-11 justify-self-start rounded text-[13px] font-semibold text-wc-accent", FOCUS_RING)}>
            {fullPower ? t(strings.launcher.usageModeFullPowerOff) : t(strings.launcher.usageModeFullPower, { model: usualName ?? saveAgent.usualModel })}
          </button>
        )}
      </Row>
    </>
  );
}

function CompactRow({ card, choice, modelName, limit, onLimit, saveAgent }: {
  card: AgentCard;
  choice: RoleModelChoice | undefined;
  modelName: string | undefined;
  limit: number | null;
  onLimit: (tokens: number | null) => void;
  saveAgent?: UsageModeAgent;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage;
  const window = choice ? Number(choice.contextWindowTokens) : 0;
  const options = limitOptions(choice);
  const fmt = (tokens: number) => formatTokens(tokens, locale);
  const modeLimit = saveAgent?.model ? saveAgent.contextLimitTokens : undefined;
  const windowText = window > 0 ? t(strings.launcher.contextWindow, { window: fmt(window) }) : t(strings.launcher.contextWindowUnknown);
  let hint: string;
  if (limit !== null) {
    hint = window > 0 && limit >= window
      ? t(strings.launcher.contextLimitFull, { window: fmt(window) })
      : t(strings.launcher.contextLimitNear, { limit: fmt(limit) });
    if (saveAgent?.model) hint = `${hint} ${t(strings.launcher.contextLimitBeatsMode)}`;
  } else if (modeLimit) hint = t(strings.launcher.contextLimitFromMode, { limit: fmt(modeLimit) });
  else if (saveAgent?.model) hint = t(strings.launcher.contextLimitMaybeMode);
  else if (modelName) hint = t(strings.launcher.contextLimitDefault, { name: card.label });
  else hint = t(strings.launcher.contextLimitDefaultNoModel, { name: card.label });

  return (
    <Row testId="launcher-context-limit">
      <Field label={t(strings.launcher.contextLimitLabel)}>
        {options.length > 0 ? (
          <Select className={SELECT_TEXT} aria-label={t(strings.launcher.contextLimitLabel)} data-testid="launcher-context-limit-select" value={limit === null ? "default" : String(limit)}
            onValueChange={(value) => { onLimit(value === "default" ? null : Number(value)); }}
            options={[{ value: "default", label: t(strings.launcher.contextLimitDefaultOption) }, ...options.map((tokens) => ({ value: String(tokens), label: fmt(tokens) }))]} />
        ) : <Value>{t(strings.launcher.contextLimitDefaultOption)}</Value>}
      </Field>
      {/* Only a chosen limit has something to show. At Default a full bar says nothing. */}
      {limit !== null && window > 0 && (
        <span data-wc-rcl-theme data-testid="launcher-context-gauge" data-gauge-percent={Math.round(Math.min(100, (limit / window) * 100))} className="flex">
          <Progress value={Math.min(limit, window)} max={window} size="sm" showValue={false} label={windowText} style={{ minInlineSize: 0, width: "100%" }} />
        </span>
      )}
      <span className={HINT}><span data-testid="launcher-context-hint">{hint}</span> <span data-testid="launcher-context-window" className="text-wc-text-faint">{windowText}</span></span>
    </Row>
  );
}

function SaveUsageRow({ card, agent, model, mode, canControl, busy, error, onStart, onEnd }: {
  card: AgentCard;
  agent: UsageModeAgent;
  model: ModelPolicyController;
  mode?: UsageModeView;
  canControl: boolean;
  busy: boolean;
  error: boolean;
  onStart: () => void;
  onEnd: (modeId: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const labelId = `launcher-save-usage-${slugify(card.key)}`;
  const on = Boolean(mode);
  const lower = modelLabel(model, agent.lowerModel) ?? agent.lowerModel;
  const usual = modelLabel(model, agent.usualModel) ?? agent.usualModel;
  // When the default already is the lower model, this launch runs the same either
  // way, so the row says what the mode does change rather than implying a saving.
  const defaultStays = model.defaultChoice?.model === agent.lowerModel;
  const until = !mode || mode.untilKind === "off" || !mode.expiresAt
    ? t(strings.launcher.usageModeUntilOffShort)
    : t(strings.launcher.usageModeUntil, { time: formatClockTime(mode.expiresAt, i18n.resolvedLanguage) });
  return (
    <Row testId="launcher-save-usage">
      <div className="flex min-h-11 items-center justify-between gap-3">
        <span id={labelId} className="text-sm font-medium text-wc-text-primary">{t(strings.launcher.usageModeStartButton)}</span>
        <Switch aria-labelledby={labelId} data-testid="launcher-save-usage-switch" checked={on} disabled={busy || !canControl}
          onCheckedChange={(next) => { if (next) onStart(); else if (mode) onEnd(mode.id); }} />
      </div>
      <span className={HINT}>
        {on
          ? t(defaultStays ? strings.launcher.usageModeOnHintDefaultStays : strings.launcher.usageModeOnHint, { until, name: card.label, lower, usual })
          : t(defaultStays ? strings.launcher.usageModeOffHintDefaultStays : strings.launcher.usageModeOffHint, { lower, usual })}
      </span>
      {!canControl && <span className={HINT}>{t(strings.launcher.usageModeOperatorOnly)}</span>}
      {error && <span role="alert" className="text-xs text-wc-error-text">{t(strings.launcher.usageModeError)}</span>}
    </Row>
  );
}

function BillingRow({ billing }: { billing: BillingAccountController }) {
  const { t, i18n } = useTranslation();
  const label = t(strings.terminalLauncher.billingAccount);
  const credits = (value: number | undefined) => {
    if (value === undefined) return t(strings.terminalLauncher.creditsUnknown);
    if (value < 0) return t(strings.terminalLauncher.creditsUnlimited);
    return new Intl.NumberFormat(i18n.resolvedLanguage).format(value);
  };
  const usable = !billing.accountsLoading && !billing.accountsFailed && billing.accounts.length > 0;
  return (
    <Row>
      <Field label={label}>
        {billing.accountsLoading ? <Value testId="launcher-business-account-loading">{t(strings.terminalLauncher.billingLoading)}</Value>
          : usable ? (
            <Select className={SELECT_TEXT} aria-label={label} data-testid="launcher-business-account" value={billing.accountId}
              options={billing.accounts.map((account) => ({ value: account.id, label: account.display_name }))} onValueChange={billing.setAccountId} />
          ) : null}
      </Field>
      {billing.accountsFailed && <span role="alert" data-testid="launcher-business-account-error" className="text-xs text-wc-error-text">{t(strings.terminalLauncher.billingUnavailable)}</span>}
      {!billing.accountsLoading && !billing.accountsFailed && billing.accounts.length === 0 && <span role="status" data-testid="launcher-business-account-empty" className={HINT}>{t(strings.terminalLauncher.billingEmpty)}</span>}
      {usable && (billing.usageLoading
        ? <span role="status" data-testid="launcher-business-account-usage-loading" className={HINT}>{t(strings.terminalLauncher.billingUsageLoading)}</span>
        : billing.usageFailed || !billing.usage
          ? <span role="alert" data-testid="launcher-business-account-usage-error" className="text-xs text-wc-error-text">{t(strings.terminalLauncher.billingUsageUnavailable)}</span>
          : <span data-testid="launcher-business-account-usage" className={HINT}>{t(strings.terminalLauncher.billingCredits, { remaining: credits(billing.usage.remaining.ai_credits), reserved: credits(billing.usage.reserved.ai_credits) })}</span>)}
    </Row>
  );
}

/** The model list's refresh, and how the last one went. */
export interface CatalogRefresh {
  run?: AgentActionRun;
  start: () => void;
}

export interface HarnessSettingsProps {
  refresh: CatalogRefresh;
  card: AgentCard;
  harness?: Harness;
  /** Whether descriptors answered. Unknown features stay honest instead of guessed. */
  harnessKnown: boolean;
  model: ModelPolicyController;
  /** The choice the launch will really use, for the window and options. */
  effectiveChoice: RoleModelChoice | undefined;
  effectiveModelName: string | undefined;
  limit: number | null;
  onLimit: (tokens: number | null) => void;
  saveAgent?: UsageModeAgent;
  saveMode?: UsageModeView;
  canControlUsage: boolean;
  usageBusy: boolean;
  usageError: boolean;
  onStartSave: () => void;
  onEndSave: (modeId: string) => void;
  fullPower: boolean;
  onFullPower: (on: boolean) => void;
  billing: BillingAccountController;
  onOpenDetail: () => void;
}

export function HarnessSettings({ card, harness, harnessKnown, model, effectiveChoice, effectiveModelName, limit, onLimit, saveAgent, saveMode, canControlUsage, usageBusy, usageError, onStartSave, onEndSave, fullPower, onFullPower, billing, onOpenDetail, refresh }: HarnessSettingsProps) {
  // With no descriptor to ask, the model's own offered limits are the evidence
  // that the harness takes one. They are only offered for harnesses that do.
  const showLimit = !model.literal && (harnessKnown ? Boolean(harness?.supportsContextLimit) : limitOptions(effectiveChoice).length > 0);
  const showSave = Boolean(saveAgent) && (!harnessKnown || Boolean(harness?.supportsUsageMode));
  return (
    <Card aria-label={card.label} role="group" data-testid="launcher-settings" className="overflow-hidden">
      <ModelRows card={card} model={model} canChoose={!harnessKnown || Boolean(harness?.supportsModelSelection)} canChooseProvider={!harnessKnown || Boolean(harness?.supportsProviderSelection)} saveAgent={saveAgent} fullPower={fullPower} onFullPower={onFullPower} onOpenDetail={onOpenDetail} refresh={refresh} />
      {model.usesVrooliCredits && <BillingRow billing={billing} />}
      {showLimit && <CompactRow card={card} choice={effectiveChoice} modelName={effectiveModelName} limit={limit} onLimit={onLimit} saveAgent={saveAgent} />}
      {showSave && saveAgent && <SaveUsageRow card={card} agent={saveAgent} model={model} mode={saveMode} canControl={canControlUsage} busy={usageBusy} error={usageError} onStart={onStartSave} onEnd={onEndSave} />}
    </Card>
  );
}
