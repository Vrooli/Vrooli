/**
 * One harness in Manage: available actions first, then usage with absolute reset
 * times, version and the update checks. It has one state and at most one primary
 * action. Web Console never elevates: a state only the operator can fix shows
 * the exact command to run.
 */
import { Fragment, useEffect, useState } from "react";
import { Check, CircleAlert, Loader2, TriangleAlert } from "lucide-react";
import { Alert } from "@vrooli/react-component-library/Alert";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge";
import type { GetRoleModelChoicesResponse } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";
import { useTranslation } from "react-i18next";

import { modelSelectionClient } from "../../api/modelSelection";
import { strings } from "../../consts/strings";
import { writeText } from "../../lib/clipboard";
import { cn } from "../../lib/classnames";
import { formatDate, formatDateTime } from "../../lib/localeFormat";
import { slugify } from "../../lib/slugify";
import AgentMark from "./AgentMark";
import { FOCUS_RING } from "./controls";
import { agentAppearance } from "./agentAppearance";
import type { AgentCard } from "./agentGrid";
import { harnessAction, harnessState, PILL_TONE, type HarnessAction, type HarnessState, type PillTone } from "./agentStatus";
import { UsageDetail } from "./AgentUsage";
import type { AgentInstallsController } from "./useAgentInstalls";
import type { AgentActionRun, AgentUpdate, AgentUpdateAction, AgentUpdatesController } from "./useAgentUpdates";
import type { AgentUsage } from "./useAgentUsage";

const SECTION_HEAD = "px-4 pb-2 pt-4 text-[11px] font-semibold uppercase tracking-[0.08em] text-wc-text-muted";
const CARD = "mx-4 grid gap-2 p-3";
const BODY = "text-[13px] leading-5 text-wc-text-muted";

const PILL_TONE_NAME: Record<PillTone, "success" | "info" | "warning" | "neutral"> = { ok: "success", up: "info", fix: "warning", muted: "neutral" };

function useStateLabel() {
  const { t } = useTranslation();
  return (state: HarnessState): string => t({
    missing: strings.launcher.statusMissing,
    shadowed: strings.launcher.statusAttention,
    unlinked: strings.launcher.statusAttention,
    needs_setup: strings.launcher.statusAttention,
    update_ready: strings.launcher.statusUpdateReady,
    up_to_date: strings.launcher.statusUpToDate,
    check_failed: strings.launcher.statusCheckFailed,
    not_applicable: strings.launcher.statusNotApplicable,
    checking: strings.launcher.updateChecking,
  }[state]);
}

export function StatusPill({ state, testId }: { state: HarnessState; testId?: string }) {
  const label = useStateLabel();
  return <StatusBadge data-testid={testId} data-harness-state={state} tone={PILL_TONE_NAME[PILL_TONE[state]]}>{label(state)}</StatusBadge>;
}

/** Copy on the row it belongs to, so the operator can paste it into a terminal. */
function OperatorCommand({ command }: { command: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState<"idle" | "copied" | "failed">("idle");
  return (
    <>
      <code data-testid="launcher-operator-command" className="block overflow-x-auto whitespace-nowrap rounded-[var(--radius-control)] border border-wc-default bg-wc-surface-base px-3 py-2.5 font-mono text-xs text-wc-text-primary">{command}</code>
      <Button type="button" variant="outline" size="lg" shape="square" data-testid="launcher-operator-command-copy" className="w-full border border-wc-default"
        onClick={() => { void writeText(command).then((result) => { setCopied(result.ok ? "copied" : "failed"); }); }}>
        {copied === "copied" ? t(strings.launcher.commandCopied) : copied === "failed" ? t(strings.launcher.commandCopyFailed) : t(strings.launcher.commandCopy)}
      </Button>
    </>
  );
}

/** The primary action's button in its idle state. */
const ACTION_LABELS: Record<HarnessAction, keyof typeof strings.launcher> = {
  install: "installAgent",
  use_managed_copy: "actionUseManagedCopy",
  link_managed_copy: "actionLinkManagedCopy",
  update: "actionUpdate",
  retry: "installRetry",
};
const RUNNING_LABELS: Record<AgentUpdateAction, keyof typeof strings.launcher> = {
  retry: "updateChecking",
  install: "agentInstalling",
  use_managed_copy: "actionUseManagedCopyRunning",
  link_managed_copy: "actionLinkManagedCopyRunning",
  update: "actionUpdateRunning",
  rollback: "actionRollbackRunning",
  refresh_catalog: "actionRefreshRunning",
};

function RunNote({ run }: { run: AgentActionRun }) {
  const { t } = useTranslation();
  if (run.phase === "done") {
    const moved = run.action === "update" && run.previous && run.current && run.previous !== run.current;
    return (
      <div data-testid="launcher-action-done" role="status" className="mx-4">
        <Alert tone="success" title={t(strings.launcher.actionDone)} description={<>
          {run.message && <span className="block">{run.message}</span>}
          {moved && <span className="block">{t(strings.launcher.actionSessionsKeep, { previous: run.previous, current: run.current })}</span>}
        </>} />
      </div>
    );
  }
  return null;
}

function ActionCard({ card, update, state, run, updates, installs, canInstall }: {
  card: AgentCard;
  update: AgentUpdate | null | undefined;
  state: HarnessState;
  run?: AgentActionRun;
  updates: AgentUpdatesController;
  installs: AgentInstallsController;
  canInstall: boolean;
}) {
  const { t } = useTranslation();
  const action = harnessAction(state, update);
  const command = update?.operatorCommand ?? "";
  const testKey = slugify(card.key);
  const running = run?.phase === "running";
  const outcome = installs.outcomes[card.agentID];
  const installing = installs.installing.includes(card.agentID);
  // A missing harness uses the control-plane action when the server offers one,
  // and the governed install this machine already has when it does not.
  const legacyInstall = state === "missing" && !update?.primaryAction && canInstall;

  const title: string | null = {
    missing: t(strings.launcher.stateMissingTitle),
    shadowed: t(strings.launcher.stateShadowedTitle),
    unlinked: t(strings.launcher.stateUnlinkedTitle),
    needs_setup: t(strings.launcher.stateNeedsSetupTitle),
    update_ready: t(strings.launcher.stateUpdateTitle, { version: update?.latest ?? "" }),
    check_failed: t(strings.launcher.stateCheckFailedTitle),
    not_applicable: t(strings.launcher.stateNotApplicableTitle),
    up_to_date: null,
    checking: null,
  }[state];
  if (!title) return null;
  const body = update?.reason || card.detail || {
    missing: t(strings.launcher.stateMissingBody, { name: card.label }),
    shadowed: t(strings.launcher.stateShadowedBody),
    unlinked: t(strings.launcher.stateUnlinkedBody),
    needs_setup: t(strings.launcher.stateNeedsSetupBody),
    update_ready: t(strings.launcher.stateUpdateBody),
    check_failed: t(strings.launcher.stateCheckFailedBody),
    not_applicable: t(strings.launcher.agentUnsupported),
    up_to_date: "",
    checking: "",
  }[state];

  const press = () => {
    if (legacyInstall) void installs.startInstall(card.agentID);
    else if (action) void updates.run(card.agentID, action);
  };
  // A state the server gave no action for, such as a system package's shadow, has only its reason.
  const showButton = legacyInstall || (action !== null && !(state === "missing" && !update?.primaryAction && !canInstall));
  const unsettledInstall = outcome?.status === "failed" || outcome?.status === "unconfirmed" ? outcome.status : null;
  const busy = run?.phase === "running" || installing;
  const buttonTestId = unsettledInstall ? `launcher-agent-install-${unsettledInstall}-${testKey}`
    : legacyInstall || action === "install" ? `launcher-agent-install-${testKey}` : `launcher-action-primary-${testKey}`;
  let buttonLabel: string;
  if (run?.phase === "running") buttonLabel = t(strings.launcher[RUNNING_LABELS[run.action]]);
  else if (installing) buttonLabel = t(strings.launcher.agentInstalling);
  else if (run?.phase === "failed" || unsettledInstall) buttonLabel = t(strings.launcher.installRetry);
  else buttonLabel = t(strings.launcher[ACTION_LABELS[legacyInstall ? "install" : action ?? "retry"]]);

  return (
    <Card data-testid={`launcher-action-${testKey}`} data-harness-state={state} className={CARD}>
      <div className="flex items-center gap-3">
        <div className="min-w-0 flex-1">
          <h4 className="text-sm font-semibold text-wc-text-primary">{title}</h4>
          <p className={cn(BODY, "break-words")}>{body}</p>
        </div>
        {showButton && outcome?.status !== "installed" && outcome?.status !== "not_applicable" && (
          <Button type="button" variant="primary" size="md" shape="square" data-testid={buttonTestId} disabled={busy} aria-busy={busy} onClick={press} className="flex-none font-semibold">
            {busy && <Loader2 className="me-1.5 inline h-4 w-4 animate-spin" aria-hidden />}{buttonLabel}
          </Button>
        )}
      </div>
      {command && <OperatorCommand command={command} />}

      {run?.phase === "failed" && (
        <p role="alert" data-testid="launcher-action-error" className="flex gap-1.5 break-words text-[13px] text-wc-error-text">
          <CircleAlert className="mt-0.5 h-3.5 w-3.5 flex-none" aria-hidden />
          <span>{run.unsupported ? t(strings.launcher.actionUnsupported) : run.outcomeUnknown ? t(strings.launcher.actionOutcomeUnknown) : run.message || t(strings.launcher.actionFailed)}</span>
        </p>
      )}
      {running && <p role="status" className="text-xs text-wc-text-muted">{t(strings.launcher.actionLongRunning)}</p>}

      {installing && <span role="progressbar" aria-label={t(strings.launcher.agentInstalling)} data-testid={`launcher-agent-progress-${testKey}`} className="block h-[3px] w-full overflow-hidden rounded bg-wc-surface-base"><span className="block h-full w-2/5 animate-pulse rounded-e-full bg-wc-accent" /></span>}
      {outcome?.status === "installed" && <p data-testid={`launcher-agent-installed-${testKey}`} className="flex items-center gap-1 text-[13px] font-semibold text-emerald-300"><Check className="h-3.5 w-3.5" aria-hidden />{t(strings.launcher.agentInstalled)}</p>}
      {outcome?.status === "not_applicable" && <p data-testid={`launcher-agent-install-unsupported-${testKey}`} className={BODY}>{outcome.message || t(strings.launcher.agentUnsupported)}</p>}
      {(outcome?.status === "failed" || outcome?.status === "unconfirmed") && (
        <p role="alert" data-testid={`launcher-agent-install-reason-${testKey}`} className="break-words text-[13px] text-wc-error-text">
          {outcome.status === "failed" ? t(strings.launcher.installFailed) : t(strings.launcher.installUnconfirmed)}{outcome.message ? `: ${outcome.message}` : ""}
        </p>
      )}

    </Card>
  );
}

/** Reads the model list's provenance, so a stale list's notice has somewhere to lead. */
function ModelListSection({ agentId, targetId, updates }: { agentId: string; targetId: string; updates: AgentUpdatesController }) {
  const { t, i18n } = useTranslation();
  const [policy, setPolicy] = useState<GetRoleModelChoicesResponse | null>(null);
  const run = updates.runs[agentId];
  const refreshing = run?.action === "refresh_catalog" && run.phase === "running";
  const refreshed = run?.action === "refresh_catalog" && run.phase !== "running" ? run.phase : "";
  useEffect(() => {
    let cancelled = false;
    setPolicy(null);
    modelSelectionClient.getRoleModelChoices({ agentId, targetId, launchCommand: "" }).then((value) => { if (!cancelled) setPolicy(value); }).catch(() => undefined);
    return () => { cancelled = true; };
  }, [agentId, refreshed, targetId]);
  const stale = policy?.catalogStale || policy?.policyStatus === "stale";
  // A current list has nothing to say here. Its source only matters when it is in question.
  if (!policy || !stale) return null;
  const observed = policy.observedAt ? formatDate(policy.observedAt, i18n.resolvedLanguage) : "";
  return (
    <>
      <h4 className={SECTION_HEAD}>{t(strings.launcher.detailModelList)}</h4>
      <Card className={CARD} data-testid="launcher-model-list-detail">
        <p className="flex gap-1.5 text-[13px] leading-5 text-wc-warning"><TriangleAlert className="mt-0.5 h-3.5 w-3.5 flex-none" aria-hidden /><span>{policy.detail || t(strings.terminalLauncher.modelListOld)}</span></p>
        {policy.provenance && <p className={cn(BODY, "break-words")}>{t(strings.terminalLauncher.modelPolicyProvenance, { date: observed || t(strings.launcher.versionUnknown), source: policy.provenance })}</p>}
        {run?.action === "refresh_catalog" && run.phase === "failed" && <p role="alert" data-testid="launcher-catalog-refresh-error" className="break-words text-[13px] text-wc-error-text">{run.message || t(strings.launcher.actionFailed)}</p>}
        {policy.operatorCommand
          ? <OperatorCommand command={policy.operatorCommand} />
          : <Button type="button" variant="primary" size="md" shape="square" data-testid="launcher-catalog-refresh" disabled={refreshing} onClick={() => { void updates.run(agentId, "refresh_catalog"); }} className="justify-self-start font-semibold">{refreshing ? t(strings.terminalLauncher.modelListRefreshing) : t(strings.terminalLauncher.modelListRefresh)}</Button>}
      </Card>
    </>
  );
}

export interface ManageAgentDetailProps {
  card: AgentCard;
  targetId: string;
  targetKind: string;
  updates: AgentUpdatesController;
  usage?: AgentUsage | null;
  usageLoading: boolean;
  declared: number;
  installs: AgentInstallsController;
  canInstall: boolean;
}

export function ManageAgentDetail({ card, targetId, updates, usage, usageLoading, declared, installs, canInstall }: ManageAgentDetailProps) {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage;
  const update = updates.updates[card.agentID];
  const state = harnessState(card, update, updates.loading);
  const run = updates.runs[card.agentID];
  const testKey = slugify(card.key);
  const running = run?.phase === "running";
  const unknown = t(strings.launcher.versionUnknown);
  const installKind = update?.installKind ? t({
    managed: strings.launcher.installKindManaged,
    upstream_installer: strings.launcher.installKindUpstream,
    standalone: strings.launcher.installKindStandalone,
    root_owned: strings.launcher.installKindRootOwned,
  }[update.installKind] ?? strings.launcher.versionUnknown) : "";
  const checked = update?.checkedAt ? formatDateTime(update.checkedAt, locale) : "";
  const canRollback = Boolean(update?.rollbackVersion);
  const usageNotReported = declared === 0 && !usageLoading && (!usage || usage.match === "unavailable") && !usage?.reason;
  return (
    <div className="pb-5 pt-3" data-testid={`launcher-agent-detail-${testKey}`}>
      <div className="flex items-center gap-2.5 px-4 pb-3">
        <AgentMark mark={card.agentID} appearance={agentAppearance(card.agentID)} />
        <StatusPill state={state} testId={`launcher-agent-status-${testKey}`} />
      </div>
      <div className="grid gap-2.5">
        {run && <RunNote run={run} />}
        <ActionCard card={card} update={update} state={state} run={run} updates={updates} installs={installs} canInstall={canInstall} />
      </div>

      {(card.state === "ready" || card.state === "unknown") && !usageNotReported && (
        <>
          <h4 className={SECTION_HEAD}>{t(strings.launcher.detailUsage)}</h4>
          <Card className={CARD}><UsageDetail usage={usage} loading={usageLoading} declared={declared} agentLabel={card.label} /></Card>
        </>
      )}

      <h4 className={SECTION_HEAD}>{t(strings.launcher.detailVersion)}</h4>
      <Card className={CARD} data-testid="launcher-version">
        <dl className="m-0 grid grid-cols-[auto_minmax(0,1fr)] gap-x-3.5 gap-y-1.5 text-[13px] leading-5">
          {[
            { term: t(strings.launcher.versionRunning), description: update?.installed || card.version || unknown },
            { term: t(strings.launcher.versionLatest), description: update?.latest || unknown },
            { term: t(strings.launcher.versionReviewedFor), description: update?.pinned || unknown },
            ...(installKind ? [{ term: t(strings.launcher.versionInstalledBy), description: installKind }] : []),
            ...(checked ? [{ term: t(strings.launcher.versionChecked), description: checked }] : []),
            ...(usageNotReported ? [{ term: t(strings.launcher.detailUsage), description: t(strings.launcher.usageNotReported) }] : []),
          ].map(({ term, description }) => (
            <Fragment key={term}>
              <dt className="text-wc-text-muted">{term}</dt>
              <dd className="m-0 break-words text-end tabular-nums text-wc-text-primary">{description}</dd>
            </Fragment>
          ))}
        </dl>
        {update?.newerThanReviewed && <p data-testid="launcher-newer-than-reviewed" className="flex gap-1.5 text-[13px] text-amber-200"><TriangleAlert className="mt-0.5 h-3.5 w-3.5 flex-none" aria-hidden /><span>{t(strings.launcher.versionNewerThanReviewed, { installed: update.installed || unknown, reviewed: update.pinned || unknown })}</span></p>}
        {update?.note && <p className={cn(BODY, "break-words")}>{update.note}</p>}
        {canRollback && update && (
          <button type="button" data-testid={`launcher-rollback-${testKey}`} disabled={running} onClick={() => { void updates.run(card.agentID, "rollback"); }}
            className={cn("min-h-11 justify-self-start rounded-[var(--radius-control)] text-xs font-medium text-wc-text-muted underline underline-offset-2 hover:text-wc-text-secondary disabled:opacity-60", FOCUS_RING)}>
            {running && run.action === "rollback" ? t(strings.launcher.actionRollbackRunning) : t(strings.launcher.actionRollback, { version: update.rollbackVersion })}
          </button>
        )}
      </Card>

      {update && update.checks.length > 0 && (
        <>
          <h4 className={SECTION_HEAD}>{t(strings.launcher.detailChecks)}</h4>
          <Card className={CARD}>
            <ul data-testid="launcher-update-checks" className="m-0 grid list-none gap-[5px] p-0 text-[13px]">
              {update.checks.map((check) => (
                <li key={check.id} className="grid grid-cols-[18px_minmax(0,1fr)] gap-1 text-wc-text-secondary">
                  {check.passed
                    ? <Check className="mt-0.5 h-3.5 w-3.5 text-emerald-300" role="img" aria-label={t(strings.launcher.checkPass)} />
                    : <TriangleAlert className="mt-0.5 h-3.5 w-3.5 text-amber-300" role="img" aria-label={t(strings.launcher.checkProblem)} />}
                  <span className="break-words">{check.message}</span>
                </li>
              ))}
            </ul>
          </Card>
        </>
      )}
      <ModelListSection agentId={card.agentID} targetId={targetId} updates={updates} />
    </div>
  );
}
