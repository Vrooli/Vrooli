// DOC: docs/reference/configuration.md#launcher-shortcuts
/**
 * Manage agents: the list (order, updates, status, visibility) and each
 * harness's detail page. Order and visibility persist to the active profile, and
 * every status and action comes from what the control plane observed.
 */
import { useCallback, useRef } from "react";
import { ChevronLeft, ChevronRight, RefreshCw } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { Card } from "@vrooli/react-component-library/Card";
import { Switch } from "@vrooli/react-component-library/Switch";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { useTranslation } from "react-i18next";

import type { TerminalTarget } from "../../api/targets";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { formatAge } from "../../lib/localeFormat";
import { slugify } from "../../lib/slugify";
import AgentMark, { ReorderGrip } from "./AgentMark";
import { MISSING_APPEARANCE, agentAppearance } from "./agentAppearance";
import type { AgentCard } from "./agentGrid";
import { harnessAction, harnessState } from "./agentStatus";
import { ManageAgentDetail, StatusPill } from "./ManageAgentDetail";
import { FOCUS_RING } from "./controls";
import type { AgentInstallsController } from "./useAgentInstalls";
import type { AgentOrderMode, AgentPreferencesController } from "./useAgentPreferences";
import type { AgentUpdatesController } from "./useAgentUpdates";
import type { AgentUsage } from "./useAgentUsage";
import { useListReorder } from "./useListReorder";

/** "list", an agent's id, or null when closed. */
export type ManagePage = "list" | { agentId: string } | null;

export interface ManageAgentsProps {
  page: ManagePage;
  onPage: (page: ManagePage) => void;
  returnFocusRef: React.RefObject<HTMLButtonElement | null>;
  target: TerminalTarget;
  /** Every catalogued agent on this machine, in the operator's order. */
  cards: readonly AgentCard[];
  preferences: AgentPreferencesController | null;
  preferencesFailed: boolean;
  onRetryPreferences?: () => void;
  onOrder: (agentIds: string[]) => void;
  onVisibility: (agentId: string, hidden: boolean) => void;
  updates: AgentUpdatesController;
  usages: Readonly<Record<string, AgentUsage | null>>;
  usageLoading: boolean;
  declaredWindows: (agentId: string) => number;
  installs: AgentInstallsController;
  canInstall: boolean;
  onEditShortcuts?: () => void;
}

export function ManageAgents(props: ManageAgentsProps) {
  const { page, onPage, returnFocusRef, target, cards, preferences, preferencesFailed, onRetryPreferences, onOrder, onVisibility, updates, usages, usageLoading, declaredWindows, installs, canInstall, onEditShortcuts } = props;
  const { t, i18n } = useTranslation();
  const prefs = preferences?.preferences ?? null;
  const orderMode: AgentOrderMode = prefs?.orderMode ?? "custom";
  const busy = preferences?.busy ?? false;
  const reordering = page === "list" && prefs !== null && orderMode === "custom";
  const commitOrder = useCallback((next: AgentCard[]) => { onOrder(next.map((card) => card.agentID)); }, [onOrder]);
  const reorder = useListReorder<AgentCard>({ source: cards, onCommit: commitOrder, enabled: reordering && !busy });
  const updating = useRef(false);

  const hidden = prefs?.hiddenAgentIds ?? [];
  const detail = page !== null && page !== "list" ? cards.find((card) => card.agentID === page.agentId) ?? null : null;
  const close = () => { onPage(null); reorder.reset(); };
  const back = () => { if (detail) onPage("list"); else close(); };

  const stateOf = (card: AgentCard) => harnessState(card, updates.updates[card.agentID], updates.loading);
  const ready = cards.filter((card) => stateOf(card) === "update_ready" && harnessAction("update_ready", updates.updates[card.agentID]) === "update");
  const anyRunning = cards.some((card) => updates.runs[card.agentID]?.phase === "running");
  const updateAll = async () => {
    // One at a time: installs share a package manager and its lock.
    if (updating.current) return;
    updating.current = true;
    for (const card of ready) await updates.run(card.agentID, "update");
    updating.current = false;
  };
  const checkedAt = Object.values(updates.updates).map((update) => update?.checkedAt ?? "").filter(Boolean).sort().at(-1);
  const checkedAge = checkedAt ? formatAge(checkedAt, i18n.resolvedLanguage) : "";

  const items = reordering ? reorder.items : cards;
  return (
    <FullPageDrawer avoidKeyboard open={page !== null} onClose={close} title={detail ? detail.label : t(strings.launcher.manageAgents)}
      closeLabel={t(strings.launcher.backToLauncher)} centerTitle returnFocusRef={returnFocusRef} testId="launcher-agent-management" panelClassName="wc-rcl-theme wc-launcher-panel md:mx-auto md:max-w-3xl"
      headerLeading={<IconButton data-testid="launcher-management-back" aria-label={detail ? t(strings.launcher.backToManage) : t(strings.launcher.backToLauncher)} onClick={back}><ChevronLeft aria-hidden className="rtl:rotate-180" /></IconButton>}>
      {detail ? (
        <ManageAgentDetail card={detail} targetId={target.id} targetKind={target.kind} updates={updates} usage={usages[detail.agentID]} usageLoading={usageLoading}
          declared={declaredWindows(detail.agentID)} installs={installs} canInstall={canInstall} />
      ) : (
        <div className="grid gap-3 p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            {prefs && (
              <div className="w-[9.5rem]">
                <Tabs mode="controlled" variant="segmented" density="compact" ariaLabel={t(strings.launcher.cardOrder)} testId="launcher-order-mode" active={orderMode}
                  onChange={(value) => { void preferences?.update({ orderMode: value as AgentOrderMode }); }}
                  items={[{ id: "custom", label: t(strings.launcher.orderCustom) }, { id: "recent", label: t(strings.launcher.orderRecent) }]}
                  itemTestId={(id) => `launcher-order-${id}`} />
              </div>
            )}
            {ready.length > 0 && (
              <Button type="button" variant="primary" size="md" shape="square" data-testid="launcher-update-ready" disabled={anyRunning} onClick={() => { void updateAll(); }} className="ms-auto font-semibold">
                {anyRunning ? t(strings.launcher.updateAllRunning) : t(strings.launcher.updateReady, { count: ready.length })}
              </Button>
            )}
          </div>
          {busy && <p role="status" className="sr-only">{t(strings.launcher.preferencesLoading)}</p>}
          {preferencesFailed && (
            <div role="alert" className="rounded-lg border border-wc-error-text/30 p-3 text-sm text-wc-error-text">
              {t(strings.launcher.preferencesError)}
              {onRetryPreferences && <button type="button" className={cn("ms-2 min-h-11 font-semibold underline", FOCUS_RING)} onClick={onRetryPreferences}>{t(strings.launcher.installRetry)}</button>}
            </div>
          )}
          <Card className="overflow-hidden">
            {items.map((card, index) => {
              const lifted = reordering && reorder.draggingIndex === index && reorder.active;
              const isTarget = reordering && reorder.active && reorder.targetIndex === index && reorder.draggingIndex !== index;
              const isHidden = hidden.includes(card.agentID);
              const testKey = slugify(card.key);
              const missing = card.state === "missing" || card.state === "not-applicable";
              return (
                <div key={card.key} ref={(node) => { reorder.registerItem(index, node); }} data-testid={`launcher-agent-card-${testKey}`} data-agent-state={card.state}
                  className={cn("flex min-h-[60px] items-center gap-1.5 border-b border-wc-default px-2 last:border-b-0", isTarget && "border-dashed border-wc-accent", lifted && "bg-wc-surface-input", isHidden && "[&>[data-row-body]]:opacity-55")}>
                  {reordering && (
                    <button type="button" aria-label={t(strings.launcher.reorderGrip, { name: card.label })} data-testid={`launcher-agent-grip-${testKey}`} disabled={busy}
                      className={cn("flex h-11 w-9 flex-none cursor-grab touch-none items-center justify-center rounded", FOCUS_RING)}
                      onPointerDown={(event) => { reorder.onGripPointerDown(index, event); }}
                      onKeyDown={(event) => {
                        if (!event.altKey) return;
                        const delta = event.key === "ArrowUp" || event.key === "ArrowLeft" ? -1 : event.key === "ArrowDown" || event.key === "ArrowRight" ? 1 : 0;
                        if (delta === 0) return;
                        event.preventDefault();
                        reorder.moveFocused(index, delta);
                      }}>
                      <ReorderGrip active={lifted} />
                    </button>
                  )}
                  <button type="button" data-row-body data-testid={`launcher-manage-open-${testKey}`} data-capability-id={card.agentID} data-capability-state={card.state} data-capability-version={card.version}
                    onClick={() => { onPage({ agentId: card.agentID }); }}
                    className={cn("flex min-h-11 min-w-0 flex-1 items-center gap-2.5 rounded text-start", !reordering && "ps-1", FOCUS_RING)}>
                    <AgentMark mark={card.agentID} muted={card.state === "not-applicable"} appearance={missing ? MISSING_APPEARANCE : agentAppearance(card.agentID)} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-semibold text-wc-text-primary">{card.label}</span>
                      <span className="mt-0.5 block"><StatusPill state={stateOf(card)} testId={`launcher-agent-status-${testKey}`} /></span>
                    </span>
                    <ChevronRight className="h-4 w-4 flex-none text-wc-text-faint rtl:rotate-180" aria-hidden />
                  </button>
                  {prefs && (
                    <Switch data-testid={`launcher-agent-visibility-${testKey}`} checked={!isHidden} disabled={busy}
                      aria-label={t(strings.launcher.showAgentAria, { name: card.label })}
                      onCheckedChange={(shown) => { onVisibility(card.agentID, !shown); }} />
                  )}
                </div>
              );
            })}
          </Card>
          <p data-testid="launcher-reorder-hint" className="text-xs leading-[18px] text-wc-text-muted">
            {reordering && t(strings.launcher.reorderHint)}
            {reordering && " "}
            {prefs && t(strings.launcher.manageHiddenHint)}
            {checkedAge && ` ${t(strings.launcher.manageCheckedAge, { age: checkedAge })}`}
          </p>
          <div className="flex flex-wrap items-center justify-between gap-2">
            {onEditShortcuts && (
              <button type="button" data-testid="launcher-edit-shortcuts" onClick={onEditShortcuts} className={cn("flex min-h-11 items-center gap-1 rounded px-1 text-[13px] font-semibold text-wc-accent", FOCUS_RING)}>
                {t(strings.launcher.editShortcuts)} <ChevronRight className="h-4 w-4 rtl:rotate-180" aria-hidden />
              </button>
            )}
            <span className="flex-1" />
            <span className="text-xs text-wc-text-faint">{target.label}</span>
            <RefreshRow loading={updates.loading} cards={cards} onRetry={updates.retryCheck} />
          </div>
        </div>
      )}
    </FullPageDrawer>
  );
}

/** Re-reads every check. Reachable beside the time it would change. */
function RefreshRow({ loading, cards, onRetry }: { loading: boolean; cards: readonly AgentCard[]; onRetry: (agentId: string) => void }) {
  const { t } = useTranslation();
  return (
    <IconButton data-testid="launcher-update-refresh" aria-label={t(strings.launcher.refreshChecks)} disabled={loading} onClick={() => { for (const card of cards) onRetry(card.agentID); }}>
      <RefreshCw aria-hidden className={cn(loading && "animate-spin")} />
    </IconButton>
  );
}
