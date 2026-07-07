/**
 * PlanBoard — the Plan lens: a four-column Now/Next/Later/Done board over
 * the server plan projection. Columns are derived (dependency waves +
 * gates); there is deliberately no drag — cards act through explicit
 * levers. Now-column cards ride the proven operations polling path;
 * filters live in the shared drawer and persist in the URL.
 */

import { useMemo, useState } from "react";
import { ChevronDown, Clock, MessageCircleQuestion, Play, RefreshCw, SlidersHorizontal } from "lucide-react";
import { OpsBulkActions } from "../../../components/operations/OpsBulkActions";
import type { RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
import { Button } from "../../../components/ui/button";
import { cn } from "../../../lib/utils";
import { useOperationsPolling } from "../../../hooks/useOperationsPolling";
import { useSnoozedKeys } from "../../../stores/snooze-store";
import type { BacklogKind } from "../../../types";
import { usePlanData } from "../hooks/usePlanData";
import { usePlanUrlState } from "../hooks/usePlanUrlState";
import {
  applySnoozeFilter,
  laterWaveSummary,
  splitBeyondHorizon,
} from "../lib/plan-presentation";
import type { PlanBoardMetaData, PlanCardData } from "../types";
import { GoalPicker } from "./GoalPicker";
import { NowColumn } from "./NowColumn";
import { PlanBoardActions } from "./PlanBoardActions";
import { usePlanCardActions } from "./plan-card-actions-context";
import { PlanCardView } from "./PlanCardView";
import { PlanColumn } from "./PlanColumn";
import { PlanFilterDrawer } from "./PlanFilterDrawer";

const WINDOW_CHOICES = [
  { label: "1h", seconds: 3600 },
  { label: "6h", seconds: 6 * 3600 },
  { label: "24h", seconds: 24 * 3600 },
] as const;

function EmptyHint({ text, testId }: { text: string; testId: string }) {
  return (
    <div
      className="rounded-lg border border-dashed border-slate-800 px-4 py-8 text-center text-sm text-slate-500"
      data-testid={testId}
    >
      {text}
    </div>
  );
}

/**
 * Next-column header bulk actions: "Run all ready" (bulk-threshold
 * confirmed) and "Answer all" (opens the decision drawer). Must render
 * inside PlanBoardActions to reach the context.
 */
function NextHeaderActions({ groups, goal }: { groups: import("../types").PlanCardGroupData[]; goal: string }) {
  const actions = usePlanCardActions();
  if (!actions) return null;

  const readyTargets: RunBacklogTarget[] = (groups.find((g) => g.id === "ready")?.cards ?? [])
    .filter((c) => c.itemKind && c.itemName)
    .map((c) => ({ kind: c.itemKind as BacklogKind, name: c.itemName, title: c.title }));
  const decideCount = (groups.find((g) => g.id === "gates")?.cards ?? [])
    .filter((c) => c.gate?.kind === "decide")
    .reduce((sum, c) => sum + (c.gate?.count ?? 0), 0);
  // When the board is goal-scoped, the ready group already holds only the
  // goal's ready closure, so this bulk run IS "run all ready in goal".
  const runTitle = goal
    ? `Run all ${readyTargets.length} ready items in goal`
    : `Run all ${readyTargets.length} ready items`;

  return (
    <>
      {readyTargets.length > 0 && (
        <button
          type="button"
          onClick={() => actions.runTargets(readyTargets)}
          className="flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-emerald-400 transition-colors hover:bg-slate-800"
          title={runTitle}
          data-testid="plan-next-run-all"
        >
          <Play className="h-3.5 w-3.5" aria-hidden />
          {readyTargets.length}
        </button>
      )}
      {decideCount > 0 && (
        <button
          type="button"
          onClick={() => actions.openDecisions()}
          className="flex items-center gap-1 rounded px-1.5 py-0.5 text-xs text-amber-400 transition-colors hover:bg-slate-800"
          title={`Answer all ${decideCount} pending questions`}
          data-testid="plan-next-answer-all"
        >
          <MessageCircleQuestion className="h-3.5 w-3.5" aria-hidden />
          {decideCount}
        </button>
      )}
    </>
  );
}

const ETA_CONFIDENCE_TONE: Record<string, string> = {
  high: "text-emerald-300",
  medium: "text-amber-300",
  low: "text-slate-400",
};

/**
 * EtaStrip — the board's p50/p80 completion band, divided by execute-lane
 * capacity and honest about its basis (a sample count vs "priors only"). Hidden
 * when there is nothing left to estimate.
 */
function EtaStrip({ eta }: { eta: PlanBoardMetaData["eta"] }) {
  if (!eta || eta.remainingItems === 0) return null;
  const tone = ETA_CONFIDENCE_TONE[eta.confidence] ?? "text-slate-400";
  return (
    <div
      className="flex items-center gap-1.5 rounded bg-slate-800/60 px-2 py-0.5 text-xs"
      title={`${eta.remainingItems} items remaining · ${eta.laneCapacity} execute lanes · ${eta.confidence} confidence`}
      data-testid="plan-eta-strip"
    >
      <Clock className="h-3.5 w-3.5 text-slate-500" aria-hidden />
      <span className="text-slate-400">ETA</span>
      <span className="font-medium text-slate-200" data-testid="plan-eta-p50">
        {eta.p50Label}
      </span>
      <span className="text-slate-600">–</span>
      <span className="font-medium text-slate-200" data-testid="plan-eta-p80">
        {eta.p80Label}
      </span>
      <span
        className={cn("uppercase tracking-wider", tone)}
        data-testid="plan-eta-basis"
      >
        {eta.basisLabel}
      </span>
    </div>
  );
}

function BeyondHorizon({ cards }: { cards: PlanCardData[] }) {
  if (cards.length === 0) return null;
  return (
    <details className="rounded-lg border border-slate-800/60 px-2 py-1" data-testid="plan-beyond-horizon">
      <summary className="cursor-pointer list-none py-1 text-xs font-medium text-slate-500 hover:text-slate-300">
        <ChevronDown className="mr-1 inline h-3.5 w-3.5" aria-hidden />
        beyond horizon ({cards.length})
      </summary>
      <div className="mt-1 space-y-1.5 pb-1">
        {cards.map((card) => (
          <PlanCardView key={card.id} card={card} showWave />
        ))}
      </div>
    </details>
  );
}

export function PlanBoard() {
  const { board, loading, error, windowSeconds, setWindowSeconds, refresh } = usePlanData();
  const urlState = usePlanUrlState();
  const snoozedKeys = useSnoozedKeys();
  const [filterDrawerOpen, setFilterDrawerOpen] = useState(false);

  // Now-column cards ride the operations polling path (D5).
  useOperationsPolling();

  const next = useMemo(
    () => applySnoozeFilter(board?.next.groups ?? [], snoozedKeys, urlState.showSnoozed),
    [board?.next.groups, snoozedKeys, urlState.showSnoozed],
  );
  const later = useMemo(
    () => applySnoozeFilter(board?.later.groups ?? [], snoozedKeys, urlState.showSnoozed),
    [board?.later.groups, snoozedKeys, urlState.showSnoozed],
  );
  const laterSplit = useMemo(() => splitBeyondHorizon(later.groups), [later.groups]);

  if (loading && !board) {
    return (
      <div className="flex h-full items-center justify-center" data-testid="plan-board-loading">
        <p className="animate-pulse text-sm text-slate-500">Loading plan…</p>
      </div>
    );
  }

  if (error && !board) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3" data-testid="plan-board-error">
        <p className="text-sm text-rose-400">{error}</p>
        <Button variant="outline" size="sm" onClick={() => void refresh()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!board) return null;

  const cycles = board.meta.cycles;
  const hiddenSnoozed = next.hiddenCount + later.hiddenCount;

  return (
    <PlanBoardActions onBoardRefresh={() => void refresh()}>
    <div className="flex h-full min-h-0 flex-col" data-testid="plan-board">
      <div className="flex flex-wrap items-center gap-2 px-4 py-2">
        <GoalPicker goal={urlState.goal} onSelect={urlState.setGoal} />
        {cycles.length > 0 && (
          <span
            className="rounded bg-rose-500/15 px-2 py-0.5 text-xs text-rose-300"
            title={cycles.join("\n")}
            data-testid="plan-cycle-warning"
          >
            {cycles.length} dependency {cycles.length === 1 ? "cycle" : "cycles"}
          </span>
        )}
        {hiddenSnoozed > 0 && (
          <span className="text-xs text-slate-600" data-testid="plan-snoozed-hidden-count">
            {hiddenSnoozed} snoozed hidden
          </span>
        )}
        <EtaStrip eta={board.meta.eta} />
        <button
          type="button"
          onClick={() => setFilterDrawerOpen((prev) => !prev)}
          className={cn(
            "ml-auto flex items-center gap-1 rounded px-2 py-1 text-xs transition-colors",
            urlState.hasFilters ? "text-cyan-400" : "text-slate-500 hover:text-slate-300",
          )}
          data-testid="plan-board-filters"
        >
          <SlidersHorizontal className="h-3.5 w-3.5" aria-hidden />
          Filters
        </button>
        <button
          type="button"
          onClick={() => void refresh()}
          className="flex items-center gap-1 rounded px-2 py-1 text-xs text-slate-500 transition-colors hover:text-slate-300"
          data-testid="plan-board-refresh"
        >
          <RefreshCw className="h-3.5 w-3.5" aria-hidden />
          Refresh
        </button>
      </div>
      <div className="flex min-h-0 flex-1 divide-x divide-slate-800 overflow-x-auto border-t border-slate-800">
        <NowColumn />
        <PlanColumn
          title="Next"
          count={next.groups.reduce((sum, g) => sum + g.cards.length, 0)}
          subtitle="actionable now"
          headerAction={<NextHeaderActions groups={next.groups} goal={urlState.goal} />}
          groups={next.groups}
          dimmedIds={next.snoozedIds}
          emptyState={<EmptyHint text="Nothing actionable — everything is running, blocked, or done." testId="plan-next-empty" />}
          testId="plan-column-next"
        />
        <PlanColumn
          title="Later"
          count={later.groups.reduce((sum, g) => sum + g.cards.length, 0)}
          subtitle={laterWaveSummary(later.groups)}
          groups={laterSplit.visible}
          showWaves
          dimmedIds={later.snoozedIds}
          footer={laterSplit.beyond.length > 0 ? <BeyondHorizon cards={laterSplit.beyond} /> : undefined}
          emptyState={<EmptyHint text="No blocked work." testId="plan-later-empty" />}
          testId="plan-column-later"
        />
        <PlanColumn
          title="Done"
          count={board.done.cardCount}
          subtitle={
            <div className="flex items-center gap-1" data-testid="plan-done-window">
              {WINDOW_CHOICES.map((choice) => (
                <button
                  key={choice.seconds}
                  type="button"
                  onClick={() => setWindowSeconds(choice.seconds)}
                  className={
                    choice.seconds === windowSeconds
                      ? "rounded bg-slate-700/80 px-1.5 py-0.5 text-cyan-400"
                      : "rounded px-1.5 py-0.5 text-slate-500 hover:text-slate-300"
                  }
                  data-testid={`plan-done-window-${choice.label}`}
                >
                  {choice.label}
                </button>
              ))}
            </div>
          }
          groups={board.done.groups}
          emptyState={<EmptyHint text="No recent outcomes in this window." testId="plan-done-empty" />}
          testId="plan-column-done"
        />
      </div>
      <OpsBulkActions />
      <PlanFilterDrawer
        isOpen={filterDrawerOpen}
        onClose={() => setFilterDrawerOpen(false)}
        filters={urlState.filters}
        viewMode={urlState.viewMode}
        showSnoozed={urlState.showSnoozed}
        hasActiveFilters={urlState.hasFilters}
        onFiltersChange={urlState.setFilters}
        onViewModeChange={urlState.setViewMode}
        onShowSnoozedChange={urlState.setShowSnoozed}
        onReset={urlState.resetFilters}
      />
    </div>
    </PlanBoardActions>
  );
}
