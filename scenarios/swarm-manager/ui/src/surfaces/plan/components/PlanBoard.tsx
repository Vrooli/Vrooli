/**
 * PlanBoard — the Plan lens: a four-column Now/Next/Later/Done board over
 * the server plan projection. Columns are derived (dependency waves +
 * gates); there is deliberately no drag — cards act through explicit
 * levers. Now-column cards ride the proven operations polling path;
 * filters live in the shared drawer and persist in the URL.
 */

import { useMemo, useRef, useState } from "react";
import { AlertTriangle, ChevronDown, Clock, FilePlus2, MessageCircleQuestion, Play } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { OpsBulkActions } from "../../../components/operations/OpsBulkActions";
import type { RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
import { Button } from "../../../components/ui/button";
import { CreateWorkFromPlanDialog } from "../../../components/plan/CreateWorkFromPlanDialog";
import { Popover } from "../../../components/ui/popover";
import { cn } from "../../../lib/utils";
import { graphPath } from "../../../app/routes/route-paths";
import { useAttachToSessionAction } from "../../../components/session/context/useAttachToSessionAction";
import { planDependencyCyclesOption, planEtaOption } from "../../../components/session/context/session-context-refs";
import { useOperationsPolling } from "../../../hooks/useOperationsPolling";
import { useSnoozedKeys } from "../../../stores/snooze-store";
import type { BacklogKind } from "../../../types";
import { usePlanData } from "../hooks/usePlanData";
import { usePlanUrlState } from "../hooks/usePlanUrlState";
import { usePlanDataStore } from "../stores/plan-data-store";
import {
  applySnoozeFilter,
  laterWaveSummary,
  splitBeyondHorizon,
} from "../lib/plan-presentation";
import type { PlanBoardMetaData, PlanCardData } from "../types";
import { GoalPicker } from "../../../components/goals/GoalPicker";
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
function CycleWarning({ cycles }: { cycles: string[] }) {
  const navigate = useNavigate();
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const [open, setOpen] = useState(false);
  const attach = useAttachToSessionAction(cycles.length > 0 ? planDependencyCyclesOption(cycles) : null);
  if (cycles.length === 0) return null;

  const inspectEntity = (entity: string) => {
    const nodeId = cycleEntityToNodeId(entity);
    navigate(graphPath({ lens: "focus", focus: nodeId, select: nodeId }));
    setOpen(false);
  };

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className="inline-flex items-center gap-1 rounded bg-rose-500/15 px-2 py-0.5 text-xs text-rose-300 transition-colors hover:bg-rose-500/25 hover:text-rose-200"
        aria-expanded={open}
        data-testid="plan-cycle-warning"
      >
        <AlertTriangle className="h-3.5 w-3.5" aria-hidden />
        {cycles.length} dependency {cycles.length === 1 ? "cycle" : "cycles"}
      </button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-start"
        className="w-80 p-3 text-xs text-slate-200"
        testId="plan-cycle-popover"
      >
        <div className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold text-rose-200">Dependency cycles</h3>
            <p className="mt-1 text-slate-400">
              These items depend on each other, so the planner cannot produce a clean execution order.
            </p>
          </div>
          <div className="space-y-2">
            {cycles.map((cycle, index) => {
              const entities = cycle.split(/\s*->\s*/).filter(Boolean);
              const first = entities[0] ?? cycle;
              const second = entities[1] ?? first;
              return (
                <div key={`${cycle}-${index}`} className="rounded border border-slate-800 bg-slate-950/70 p-2">
                  <div className="flex flex-wrap items-center gap-1.5">
                    {entities.map((entity, entityIndex) => (
                      <span key={`${entity}-${entityIndex}`} className="inline-flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => inspectEntity(entity)}
                          className="rounded px-1 py-0.5 text-cyan-300 hover:bg-slate-800"
                        >
                          {entity}
                        </button>
                        {entityIndex < entities.length - 1 && <span className="text-slate-600">{"->"}</span>}
                      </span>
                    ))}
                  </div>
                  <button
                    type="button"
                    onClick={() => inspectEntity(first)}
                    className="mt-2 text-left text-rose-200 underline decoration-rose-400/40 underline-offset-2 hover:text-rose-100"
                    data-testid="plan-cycle-resolve"
                  >
                    Inspect dependency edge {first} {"->"} {second}
                  </button>
                </div>
              );
            })}
          </div>
          <div className="border-t border-slate-800 pt-2">
            {attach.button}
          </div>
        </div>
      </Popover>
      {attach.sheet}
    </>
  );
}

function cycleEntityToNodeId(entity: string): string {
  const trimmed = entity.trim();
  if (trimmed.startsWith("backlog-item/") || trimmed.startsWith("scenario/") || trimmed.startsWith("initiative/") || trimmed.startsWith("goal/")) {
    return trimmed;
  }
  const parts = trimmed.split("/");
  return parts.length === 2 ? `backlog-item/${trimmed}` : trimmed;
}

function EtaStrip({ eta, goal }: { eta: PlanBoardMetaData["eta"]; goal: string }) {
  const navigate = useNavigate();
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const [open, setOpen] = useState(false);
  const attach = useAttachToSessionAction(eta ? planEtaOption(eta) : null);
  if (!eta || eta.remainingItems === 0) return null;
  const tone = ETA_CONFIDENCE_TONE[eta.confidence] ?? "text-slate-400";
  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        className="flex items-center gap-1.5 rounded bg-slate-800/60 px-2 py-0.5 text-xs transition-colors hover:bg-slate-800"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        data-testid="plan-eta-strip"
      >
        <Clock className="h-3.5 w-3.5 text-slate-500" aria-hidden />
        <span className="text-slate-400">ETA</span>
        <span className="font-medium text-slate-200" data-testid="plan-eta-p50">
          {eta.p50Label}
        </span>
        <span className="text-slate-600">-</span>
        <span className="font-medium text-slate-200" data-testid="plan-eta-p80">
          {eta.p80Label}
        </span>
        <span
          className={cn("uppercase tracking-wider", tone)}
          data-testid="plan-eta-basis"
        >
          {eta.basisLabel}
        </span>
      </button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-start"
        className="w-72 p-3 text-xs text-slate-200"
        testId="plan-eta-popover"
      >
        <div className="space-y-3">
          <div>
            <h3 className="text-sm font-semibold text-slate-100">ETA basis</h3>
            <p className="mt-1 text-slate-400">
              Completion band from current remaining work and execute-lane capacity.
            </p>
          </div>
          <dl className="grid grid-cols-2 gap-2">
            <div className="rounded bg-slate-950/70 p-2">
              <dt className="text-slate-500">p50</dt>
              <dd className="font-medium text-slate-100">{eta.p50Label}</dd>
            </div>
            <div className="rounded bg-slate-950/70 p-2">
              <dt className="text-slate-500">p80</dt>
              <dd className="font-medium text-slate-100">{eta.p80Label}</dd>
            </div>
            <div className="rounded bg-slate-950/70 p-2">
              <dt className="text-slate-500">Remaining</dt>
              <dd className="font-medium text-slate-100">{eta.remainingItems} items</dd>
            </div>
            <div className="rounded bg-slate-950/70 p-2">
              <dt className="text-slate-500">Execute lanes</dt>
              <dd className="font-medium text-slate-100">{eta.laneCapacity}</dd>
            </div>
          </dl>
          <p className={cn("text-xs", tone)}>
            {eta.confidence} confidence · {eta.basisLabel}
          </p>
          <button
            type="button"
            onClick={() => {
              navigate(graphPath({ lens: "stats", goal }));
              setOpen(false);
            }}
            className="text-cyan-300 underline decoration-cyan-400/40 underline-offset-2 hover:text-cyan-200"
            data-testid="plan-eta-stats-link"
          >
            Open Stats throughput and timing
          </button>
          <div className="border-t border-slate-800 pt-2">
            {attach.button}
          </div>
        </div>
      </Popover>
      {attach.sheet}
    </>
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
  const filterDrawerOpen = usePlanDataStore((s) => s.filterDrawerOpen);
  const setFilterDrawerOpen = usePlanDataStore((s) => s.setFilterDrawerOpen);
  const [createFromPlanOpen, setCreateFromPlanOpen] = useState(false);

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
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-8 gap-1.5 px-3"
          onClick={() => setCreateFromPlanOpen(true)}
          data-testid="plan-board-create-from-plan"
        >
          <FilePlus2 className="h-4 w-4" aria-hidden />
          Create from plan
        </Button>
        <CycleWarning cycles={cycles} />
        {hiddenSnoozed > 0 && (
          <span className="text-xs text-slate-600" data-testid="plan-snoozed-hidden-count">
            {hiddenSnoozed} snoozed hidden
          </span>
        )}
        <EtaStrip eta={board.meta.eta} goal={urlState.goal} />
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
      <CreateWorkFromPlanDialog
        isOpen={createFromPlanOpen}
        onClose={() => setCreateFromPlanOpen(false)}
        onImported={() => void refresh()}
      />
    </div>
    </PlanBoardActions>
  );
}
