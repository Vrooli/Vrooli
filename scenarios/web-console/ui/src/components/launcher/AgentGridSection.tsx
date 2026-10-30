// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * Compose the launcher grid and agent management over one target's readiness.
 * Install outcomes live with this target and never imply success until confirmed.
 * The profile owner persists ordering and visibility independently of commands.
 *
 * [REQ:P0-006a] Terminal Launch Flow UI
 * [REQ:P0-014a] Launcher Destination And Appearance Disclosure
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ArrowUpCircle, Check, ChevronLeft, ChevronRight, CircleAlert, HelpCircle, Settings2 } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { useTranslation } from "react-i18next";

import { strings } from "../../consts/strings";
import type { ShortcutEntry } from "../../consts/shortcuts";
import type { InstallOutcome } from "../../api/capabilities";
import type { TerminalTarget } from "../../api/targets";
import { slugify } from "../../lib/slugify";
import { cn } from "../../lib/classnames";
import AgentMark, { ReorderGrip } from "./AgentMark";
import { MISSING_APPEARANCE, agentAppearance } from "./agentAppearance";
import {
  buildAgentGrid,
  cardInstalls,
  cardLaunches,
  type AgentCard,
} from "./agentGrid";
import { useListReorder } from "./useListReorder";
import { useAgentUpdates } from "./useAgentUpdates";
import { useAgentUsage } from "./useAgentUsage";
import { AgentUsage } from "./AgentUsage";

/** Shared card shell. Overflow hidden so the rail clips to the rounded corner. */
const CARD_SHELL =
  "relative flex min-w-0 flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-input transition";

/** The pressable body of a card. */
const CARD_BODY =
  "flex min-h-[54px] w-full flex-col gap-1.5 px-2.5 pb-2 pt-2.5 text-start disabled:cursor-not-allowed";

/** How long the transient "Installed" confirmation stays on the card. */
const CONFIRMATION_MS = 6000;
const EMPTY_IDS: readonly string[] = [];

/**
 * The one rail a card can carry. "none" is a real member rather than an
 * absent value so every branch has to be answered.
 */
type RailKind = "none" | "install" | "progress" | "installed" | "unconfirmed" | "failed" | "unsupported";

/** Drops keys from a record without mutating it or deleting computed keys. */
function without(source: Record<string, InstallOutcome>, keys: string[]): Record<string, InstallOutcome> {
  const drop = new Set(keys);
  return Object.fromEntries(Object.entries(source).filter(([key]) => !drop.has(key)));
}

/** Shared rail geometry, so switching between rails never resizes the card. */
// Border colour is deliberately NOT set here: each rail supplies its own, and
// two border-colour utilities on one element resolve by stylesheet order
// rather than class order, so the shared one would win unpredictably.
const RAIL_BASE = "w-full border-t px-2 py-1.5 text-center text-[11.5px] font-semibold transition";

export interface AgentGridSectionProps {
  shortcuts: readonly ShortcutEntry[];
  target: TerminalTarget;
  /** True when nothing in the grid may be pressed (creating, offline, …). */
  disabled: boolean;
  onLaunch: (command?: string) => void;
  /**
   * Runs the governed installer and resolves with the machine's own verdict.
   * The card never infers success from the call resolving: "the installer ran"
   * and "the agent is there" are different claims, and only the second one is
   * an install the operator can act on.
   */
  onInstall?: (agentID: string) => Promise<InstallOutcome>;
  onEditShortcuts?: () => void;
  /** Persists a new order. Absent means the grid offers no reorder mode. */
  onReorder?: (next: string[]) => void | Promise<void>;
  orderedAgentIds?: readonly string[];
  hiddenAgentIds?: readonly string[];
  onVisibilityChange?: (agentID: string, hidden: boolean) => void;
  updatesEnabled?: boolean;
  preferencesBusy?: boolean;
  preferencesError?: string;
  onRetryPreferences?: () => void;
}

export default function AgentGridSection({
  shortcuts,
  target,
  disabled,
  onLaunch,
  onInstall,
  onEditShortcuts,
  onReorder,
  orderedAgentIds = EMPTY_IDS,
  hiddenAgentIds = EMPTY_IDS,
  onVisibilityChange,
  updatesEnabled = false,
  preferencesBusy = false,
  preferencesError,
  onRetryPreferences,
}: AgentGridSectionProps) {
  const { t } = useTranslation();
  const [manageOpen, setManageOpen] = useState(false);
  const manageButton = useRef<HTMLButtonElement>(null);
  const reordering = manageOpen && Boolean(onReorder);
  const [installing, setInstalling] = useState<string[]>([]);
  const [outcomes, setOutcomes] = useState<Record<string, InstallOutcome>>({});
  const confirmationTimers = useRef<number[]>([]);

  useEffect(() => () => {
    for (const timer of confirmationTimers.current) window.clearTimeout(timer);
  }, []);

  const { agents, commands } = useMemo(() => {
    const grid = buildAgentGrid({ readiness: target.readiness, shortcuts, installing });
    const positions = new Map(orderedAgentIds.map((id, index) => [id, index]));
    return { ...grid, agents: grid.agents.map((card, index) => ({ card, index })).sort((a, b) =>
      (positions.get(a.card.agentID) ?? orderedAgentIds.length + a.index) -
      (positions.get(b.card.agentID) ?? orderedAgentIds.length + b.index)).map(({ card }) => card) };
  }, [installing, orderedAgentIds, shortcuts, target.readiness]);

  const updateState = useAgentUpdates(updatesEnabled && target.available, target.id, agents.filter((card) => card.state === "ready").map((card) => card.agentID).join(","));
  const usageState = useAgentUsage(target.available, target.id, agents.filter((card) => card.state === "ready" || card.state === "unknown").map((card) => card.agentID).join(","));

  const commitOrder = useCallback((next: AgentCard[]) => {
    if (!onReorder) return;
    void onReorder(next.map((card) => card.agentID));
  }, [onReorder]);

  const reorder = useListReorder<AgentCard>({ source: agents, onCommit: commitOrder, enabled: reordering && !preferencesBusy });

  // The rail has room for "Install failed" and nothing else. The reason lived
  // only in a title tooltip, which a phone never shows, so it is also listed
  // under the grid in the machine's own words.
  const installNotes = agents.filter((card) => {
    const outcome = outcomes[card.agentID];
    return Boolean(outcome?.message) && (outcome?.status === "failed" || outcome?.status === "unconfirmed");
  });

  const startInstall = useCallback(async (agentID: string) => {
    if (!onInstall) return;
    // Clear any previous verdict first: a retry that still showed the last
    // failure underneath its own progress bar would be reporting two answers.
    setOutcomes((current) => (agentID in current ? without(current, [agentID]) : current));
    setInstalling((current) => (current.includes(agentID) ? current : [...current, agentID]));
    let outcome: InstallOutcome;
    try {
      outcome = await onInstall(agentID);
    } catch (error) {
      // A transport failure is a failed install with no message from the
      // machine, which is still more than the operator had before.
      console.error("capability install failed", agentID, error);
      outcome = { status: "failed" };
    }
    setInstalling((current) => current.filter((id) => id !== agentID));
    setOutcomes((current) => ({ ...current, [agentID]: outcome }));
    // Only the success is transient. It answers "did that work?" and then gets
    // out of the way of the card's real state. A failure or an unconfirmed
    // install carries the only explanation the operator will get, so it stays
    // until they retry it or the machine reports the agent.
    if (outcome.status === "installed") {
      confirmationTimers.current.push(window.setTimeout(() => {
        setOutcomes((current) => (current[agentID]?.status === "installed" ? without(current, [agentID]) : current));
      }, CONFIRMATION_MS));
    }
  }, [onInstall]);

  // An outcome describes a machine state at a moment. Once the machine reports
  // the agent as present, the stored verdict is stale by definition — an
  // "unconfirmed" rail left over a working agent is its own kind of lie.
  const readyAgentKey = agents.filter((card) => card.state === "ready").map((card) => card.agentID).join(",");
  useEffect(() => {
    const ready = new Set(readyAgentKey ? readyAgentKey.split(",") : []);
    setOutcomes((current) => {
      const stale = Object.keys(current).filter((id) => ready.has(id) && current[id]?.status !== "installed");
      return stale.length === 0 ? current : without(current, stale);
    });
  }, [readyAgentKey]);

  const statusLine = (card: AgentCard): string => {
    switch (card.state) {
      case "ready":
        return card.version || t(strings.launcher.agentReady);
      case "missing":
        return t(strings.launcher.agentMissing);
      case "installing":
        return t(strings.launcher.agentInstalling);
      case "not-applicable":
        return card.detail || t(strings.launcher.agentUnsupported);
      default:
        return card.detail || card.description || card.command;
    }
  };

  /**
   * The single rail a card may show, chosen once.
   *
   * The order is the whole point: a decided outcome outranks the install
   * offer, so "Install" and "Installed" cannot appear together no matter what
   * the catalog says. That pairing is what an operator saw when a relayed
   * install completed against a machine that never reported the agent.
   */
  const railFor = (card: AgentCard, installs: boolean): RailKind => {
    if (card.state === "installing") return "progress";
    const outcome = outcomes[card.agentID];
    // A machine can refuse an install the catalog still thinks is offerable —
    // no published build for its platform. Falling through to "none" would
    // remove the button and say nothing, so the refusal gets its own rail.
    if (outcome) return outcome.status === "not_applicable" ? "unsupported" : outcome.status;
    return installs ? "install" : "none";
  };

  const renderRail = (card: AgentCard, rail: RailKind) => {
    const testID = slugify(card.key);
    switch (rail) {
      case "install":
        return (
          <button
            type="button"
            data-testid={`launcher-agent-install-${testID}`}
            disabled={disabled}
            onClick={() => { void startInstall(card.agentID); }}
            className={cn(RAIL_BASE, "border-wc-default bg-wc-warning/10 text-wc-warning hover:bg-wc-warning/20 disabled:cursor-not-allowed disabled:opacity-50")}
          >
            {t(strings.launcher.installAgent)}
          </button>
        );
      case "progress":
        return (
          <span
            role="progressbar"
            aria-label={t(strings.launcher.agentInstalling)}
            data-testid={`launcher-agent-progress-${testID}`}
            className="block h-[3px] w-full overflow-hidden bg-wc-surface-base"
          >
            <span className="block h-full w-2/5 animate-pulse rounded-e-full bg-wc-accent" />
          </span>
        );
      case "installed":
        return (
          <span
            data-testid={`launcher-agent-installed-${testID}`}
            className={cn(RAIL_BASE, "flex items-center justify-center gap-1 border-wc-success/30 bg-wc-success/10 text-wc-success")}
          >
            <Check className="h-3 w-3" aria-hidden />
            {t(strings.launcher.agentInstalled)}
          </span>
        );
      // Unconfirmed and failed are both retryable and both carry the machine's
      // own words in the tooltip, which is the only place the reason exists.
      case "unconfirmed":
      case "failed": {
        const failed = rail === "failed";
        return (
          <button
            type="button"
            data-testid={`launcher-agent-${failed ? "install-failed" : "install-unconfirmed"}-${testID}`}
            title={outcomes[card.agentID]?.message}
            disabled={disabled}
            onClick={() => { void startInstall(card.agentID); }}
            className={cn(
              RAIL_BASE,
              "flex items-center justify-center gap-1 disabled:cursor-not-allowed disabled:opacity-50",
              failed
                ? "border-wc-error-text/30 bg-wc-error-text/10 text-wc-error-text hover:bg-wc-error-text/20"
                : "border-wc-warning/30 bg-wc-warning/10 text-wc-warning hover:bg-wc-warning/20",
            )}
          >
            {failed ? <CircleAlert className="h-3 w-3 shrink-0" aria-hidden /> : <HelpCircle className="h-3 w-3 shrink-0" aria-hidden />}
            <span className="truncate">{failed ? t(strings.launcher.installFailed) : t(strings.launcher.installUnconfirmed)}</span>
            <span className="shrink-0 opacity-70">· {t(strings.launcher.installRetry)}</span>
          </button>
        );
      }
      case "unsupported":
        return (
          <span
            data-testid={`launcher-agent-install-unsupported-${testID}`}
            title={outcomes[card.agentID]?.message}
            className={cn(RAIL_BASE, "border-wc-default bg-wc-surface-base/60 text-wc-text-faint")}
          >
            {t(strings.launcher.agentUnsupported)}
          </span>
        );
      default:
        return null;
    }
  };

  const renderAgentCard = (card: AgentCard, index: number) => {
    const launches = cardLaunches(card);
    const installs = cardInstalls(card) && Boolean(onInstall);
    const inert = card.state === "not-applicable" || (card.state === "missing" && !onInstall);
    const lifted = reordering && reorder.draggingIndex === index && reorder.active;
    const isTarget = reordering && reorder.active && reorder.targetIndex === index && reorder.draggingIndex !== index;
    const rail = railFor(card, installs);
    const update = updateState.updates[card.agentID];
    const status = update?.probeVerified ? update.status : "unknown";
    const statusText = status === "behind" ? t(strings.launcher.updateAvailable) : status === "up-to-date" ? t(strings.launcher.updateCurrent) : status === "ahead" ? t(strings.launcher.updateAhead) : t(strings.launcher.updateUnknown);
    const hidden = hiddenAgentIds.includes(card.agentID);
    return (
      <div key={card.key} ref={(node) => { reorder.registerItem(index, node); }}
        data-testid={`launcher-agent-card-${slugify(card.key)}`} data-agent-state={card.state}
        className={cn("min-w-0 border-b border-wc-default py-2", isTarget && "border-wc-accent border-dashed", lifted && "bg-wc-surface-input") }>
        <div className="flex min-w-0 items-start gap-2">
          {reordering && <span role="button" tabIndex={preferencesBusy ? -1 : 0}
            aria-label={t(strings.launcher.reorderGrip, { name: card.label })}
            data-testid={`launcher-agent-grip-${slugify(card.key)}`}
            className="flex min-h-11 w-11 shrink-0 cursor-grab touch-none items-center justify-center"
            onPointerDown={(event) => { reorder.onGripPointerDown(index, event); }}
            onKeyDown={(event) => {
              if (!event.altKey) return;
              const delta = event.key === "ArrowUp" || event.key === "ArrowLeft" ? -1 : event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : 0;
              if (delta === 0) return;
              event.preventDefault(); reorder.moveFocused(index, delta);
            }}><ReorderGrip active={lifted} /></span>}
          <span className="pt-2.5"><AgentMark mark={card.agentID || "command"} muted={inert} appearance={card.state === "missing" && !inert ? MISSING_APPEARANCE : agentAppearance(card.agentID)} /></span>
          <div className="min-w-0 flex-1 py-2">
            <button type="button" data-testid={`launcher-agent-${slugify(card.label)}`}
              data-capability-id={card.agentID || undefined} data-capability-state={card.state} data-capability-version={card.version}
              disabled={disabled || reordering || !launches} onClick={() => { if (launches) onLaunch(card.command); }}
              className="block max-w-full truncate text-start text-sm font-semibold text-wc-text-primary disabled:cursor-default">{card.label}</button>
            <p className="mt-0.5 break-words text-xs text-wc-text-muted">{statusLine(card)}</p>
            {(card.state === "ready" || card.state === "unknown") && <div className="mt-2"><AgentUsage agent={card.label} usage={usageState.usages[card.agentID]} loading={usageState.loading} detailed /></div>}
            {card.state === "ready" && <div data-testid={`launcher-agent-update-${slugify(card.key)}`} className="mt-1 text-xs text-wc-text-muted">
              <span className={status === "behind" ? "font-semibold text-wc-text-primary" : "font-medium"}>{updateState.loading && !(card.agentID in updateState.updates) ? t(strings.launcher.updateChecking) : statusText}</span>
              {update && <details className="group mt-1"><summary className="flex min-h-11 cursor-pointer items-center gap-1 text-wc-accent"><ChevronRight className="h-3.5 w-3.5 transition-transform group-open:rotate-90" aria-hidden />{t(strings.launcher.versionDetails)}</summary>
                <p className="break-words">{t(strings.launcher.updateDetails, { installed: update.installed || t(strings.launcher.versionUnknown), latest: update.latest || t(strings.launcher.versionUnknown), pinned: update.pinned || t(strings.launcher.versionUnknown) })}</p>
                {update.checkedAt && Number.isFinite(Date.parse(update.checkedAt)) && <p className="mt-1">{t(strings.launcher.updateChecked, { time: new Date(update.checkedAt).toLocaleString() })}</p>}
                {(update.sourceKind || update.sourceId) && <p className="mt-1 break-words">{t(strings.launcher.updateSource, { source: [update.sourceKind, update.sourceId].filter(Boolean).join(" · ") })}</p>}
                {update.note && <p className="mt-1 break-words">{update.note}</p>}
              </details>}
            </div>}
          </div>
          {onVisibilityChange && card.agentID && <button type="button" data-testid={`launcher-agent-visibility-${slugify(card.key)}`}
            aria-label={t(hidden ? strings.launcher.showAgentAria : strings.launcher.hideAgentAria, { name: card.label })} aria-pressed={!hidden}
            disabled={preferencesBusy} onClick={() => { onVisibilityChange(card.agentID, !hidden); }}
            className="min-h-11 shrink-0 rounded-lg px-2 text-xs font-semibold text-wc-accent hover:bg-wc-surface-input disabled:opacity-50">
            {hidden ? t(strings.launcher.hiddenState) : t(strings.launcher.shownState)}</button>}
        </div>
        {renderRail(card, rail)}
      </div>
    );
  };

  const visibleAgents = agents.filter((card) => !hiddenAgentIds.includes(card.agentID) && (card.state === "ready" || card.state === "unknown"));
  return (
    <section aria-labelledby="launcher-actions-heading" className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h3 id="launcher-actions-heading" className="text-sm font-semibold text-wc-text-primary">{t(strings.launcher.agents)}</h3>
        <button ref={manageButton} type="button" data-testid="launcher-manage-agents"
          onClick={() => { setManageOpen(true); }}
          className="flex min-h-11 items-center gap-2 rounded-lg px-2 text-xs font-semibold text-wc-accent hover:bg-wc-surface-input focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent">
          {t(strings.launcher.manageAgents)} <Settings2 className="h-4 w-4" aria-hidden />
        </button>
      </div>
      <div data-testid="launcher-agent-grid" className="grid grid-cols-2 items-stretch gap-2">
        {visibleAgents.map((card) => {
          const canLaunch = card.state === "ready" || (card.state === "unknown" && target.kind === "local");
          return (
            <div key={card.key} className={cn(CARD_SHELL, "hover:border-wc-accent")}>
              <button type="button" data-testid={`launcher-agent-${slugify(card.label)}`}
                data-capability-id={card.agentID} data-capability-state={card.state} data-capability-version={card.version}
                disabled={disabled || !canLaunch} onClick={() => { onLaunch(card.command); }} title={card.command}
                className={cn(CARD_BODY, "focus-visible:outline focus-visible:outline-2 focus-visible:outline-wc-accent disabled:opacity-60")}>
                <span className="flex min-w-0 items-center gap-2"><AgentMark mark={card.agentID} />
                  <span className="min-w-0 flex-1 truncate text-[13.5px] font-semibold text-wc-text-primary">{card.label}</span>
                  {updateState.updates[card.agentID]?.probeVerified && updateState.updates[card.agentID]?.status === "behind" &&
                    <ArrowUpCircle className="h-4 w-4 shrink-0 text-wc-warning" role="img" aria-label={t(strings.launcher.updateAvailable)} />}</span>
              </button>
              <div className="min-w-0 px-2.5 pb-2"><AgentUsage agent={card.label} usage={usageState.usages[card.agentID]} loading={usageState.loading} /></div>
            </div>
          );
        })}
        <button type="button" data-testid="launcher-empty-shell" onClick={() => { onLaunch(); }} disabled={disabled}
          className={cn(CARD_SHELL, CARD_BODY, "hover:border-wc-accent disabled:opacity-50")}>
          <span className="flex min-w-0 items-center gap-2"><AgentMark mark="shell" />
            <span className="min-w-0 truncate text-[13.5px] font-semibold text-wc-text-primary">{t(strings.terminalLauncher.emptyShell)}</span></span>
          <span className="text-[11px] text-wc-text-muted">{t(strings.launcher.emptyShellStatus)}</span>
        </button>
      </div>
      {commands.length > 0 && <div className="space-y-2">
        <div className="px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">{t(strings.terminalLauncher.shortcuts)}</div>
        {commands.map((card) => <button key={card.key} type="button" data-testid={`launcher-shortcut-${slugify(card.label)}`}
          onClick={() => { onLaunch(card.command); }} disabled={disabled} title={card.command}
          className="flex min-h-11 w-full items-center gap-2.5 rounded-xl border border-wc-default bg-wc-surface-input px-3 py-2 text-start hover:border-wc-accent disabled:opacity-50">
          <AgentMark mark="command" /><span className="min-w-0 flex-1"><span className="block truncate text-[13.5px] font-medium text-wc-text-primary">{card.label}</span>
          <span className="block truncate font-mono text-[10.5px] text-wc-text-faint">{card.description || card.command}</span></span></button>)}
      </div>}
      <FullPageDrawer avoidKeyboard open={manageOpen} onClose={() => { setManageOpen(false); reorder.reset(); }}
        title={t(strings.launcher.manageAgents)} closeLabel={t(strings.launcher.backToLauncher)} centerTitle
        returnFocusRef={manageButton} testId="launcher-agent-management" panelClassName="wc-rcl-theme md:mx-auto md:max-w-3xl"
        headerLeading={<IconButton data-testid="launcher-management-back" aria-label={t(strings.launcher.backToLauncher)} onClick={() => { setManageOpen(false); reorder.reset(); }}><ChevronLeft aria-hidden /></IconButton>}>
        <div className="space-y-4 p-5">
          <p className="rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-sm text-wc-text-muted">{t(strings.launcher.manageContext, { target: target.label })}</p>
          <p className="text-xs text-wc-text-muted">{t(strings.launcher.usageUnavailableReason)}</p>
          <button type="button" onClick={usageState.refresh} disabled={usageState.loading || !target.available} className="min-h-11 rounded-lg px-2 text-xs font-semibold text-wc-accent hover:bg-wc-surface-input disabled:opacity-50">{t(strings.launcher.usageRefresh)}</button>
          {preferencesBusy && <p role="status" className="text-xs text-wc-text-muted">{t(strings.launcher.preferencesLoading)}</p>}
          {preferencesError && <div role="alert" className="rounded-lg border border-wc-error-text/30 p-3 text-sm text-wc-error-text">{preferencesError}
            {onRetryPreferences && <button type="button" className="ms-2 min-h-11 font-semibold underline" onClick={onRetryPreferences}>{t(strings.launcher.installRetry)}</button>}</div>}
          <div className="flex items-center justify-between gap-2"><h3 className="text-sm font-semibold text-wc-text-primary">{t(strings.launcher.agents)}</h3>
            {onEditShortcuts && <button type="button" data-testid="launcher-edit-shortcuts" onClick={onEditShortcuts} className="min-h-11 text-xs font-semibold text-wc-accent">{t(strings.launcher.editShortcuts)}</button>}</div>
          <div className="grid grid-cols-1 gap-2">{(reordering ? reorder.items : agents).map(renderAgentCard)}</div>
          {installNotes.length > 0 && <ul data-testid="launcher-install-reasons" className="space-y-1">{installNotes.map((card) =>
            <li key={card.key} role="alert" data-testid={`launcher-agent-install-reason-${slugify(card.key)}`} className="break-words rounded-lg border border-wc-error-text/30 bg-wc-error-text/10 px-3 py-2 text-xs text-wc-error-text">{t(strings.launcher.installFailedReason, { name: card.label, reason: outcomes[card.agentID]?.message })}</li>)}</ul>}
          {reordering && <p data-testid="launcher-reorder-hint" className="text-xs text-wc-text-muted">{t(strings.launcher.reorderHint)}</p>}
        </div>
      </FullPageDrawer>
      <span aria-live="polite" className="sr-only">{installing.length > 0 ? t(strings.launcher.agentInstalling) : ""}</span>
    </section>
  );
}
