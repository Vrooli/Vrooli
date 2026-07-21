/**
 * PlanBoard — the Plan lens: a four-column Now/Next/Later/Done board over
 * the server plan projection. Columns are derived (dependency waves +
 * gates); there is deliberately no drag — cards act through explicit
 * levers. Now-column cards ride the proven operations polling path;
 * filters live in the shared drawer and persist in the URL.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { AlertTriangle, ArrowRight, ChevronDown, Clock, CornerDownLeft, MessageCircleQuestion, Play, X } from "lucide-react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { OpsBulkActions } from "../../../components/operations/OpsBulkActions";
import type { RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
import { Button } from "../../../components/ui/button";
import { Popover } from "../../../components/ui/popover";
import { cn } from "../../../lib/utils";
import { backlogDetailPath, detailPathFromNodeId, graphPath } from "../../../app/routes/route-paths";
import { StatusChip } from "../../../components/ui/status-chip";
import { getStatusColorClasses } from "../../graph/lib/status-colors";
import { useAttachToSessionAction } from "../../../components/session/context/useAttachToSessionAction";
import { planDependencyCyclesOption, planEtaOption } from "../../../components/session/context/session-context-refs";
import { useOperationsPolling } from "../../../hooks/useOperationsPolling";
import { useOperationsStore } from "../../../stores/operations-store";
import { useSnoozedKeys } from "../../../stores/snooze-store";
import type { ActivityRow } from "../../../types/operations";
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
import { EtaExplainerContent } from "../../../components/stats/EtaExplainer";
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
    .filter((c) => c.gate?.kind === "decide" || c.gate?.kind === "proposal")
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
          title={`Review all ${decideCount} pending decisions and proposals`}
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

function CycleWarning({ cycles, cardsByEntity }: { cycles: string[]; cardsByEntity: Map<string, PlanCardData> }) {
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
  const openItem = (entity: string) => {
    const card = cardsByEntity.get(entity);
    if (card?.itemKind && card.itemName) navigate(backlogDetailPath(card.itemKind as BacklogKind, card.itemName));
    else inspectEntity(entity);
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
        aria-label={`${cycles.length} dependency ${cycles.length === 1 ? "cycle" : "cycles"}`}
        title={`${cycles.length} dependency ${cycles.length === 1 ? "cycle" : "cycles"}`}
        data-testid="plan-cycle-warning"
      >
        <AlertTriangle className="h-3.5 w-3.5" aria-hidden />
        {/* Terse on purpose — "dependency" is implied and the popover carries
            the full explanation; the row must fit one line on mobile. */}
        {cycles.length} {cycles.length === 1 ? "cycle" : "cycles"}
      </button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-start"
        mobileSheet
        mobileTitle="Dependency cycles"
        className="w-80 p-3 text-xs text-slate-200"
        testId="plan-cycle-popover"
      >
        <div className="space-y-3 text-xs text-slate-200">
          <div>
            {/* The sheet header already carries the title on mobile. */}
            <h3 className="hidden text-sm font-semibold text-rose-200 md:block">Dependency cycles</h3>
            <p className="mt-1 text-slate-400">
              These items depend on each other, so the planner cannot produce a clean execution order.
            </p>
          </div>
          <div className="space-y-2">
            {cycles.map((cycle, index) => {
              const entities = cycle.split(/\s*->\s*/).filter(Boolean);
              const chain = entities.length > 1 && entities[0] === entities[entities.length - 1] ? entities.slice(0, -1) : entities;
              const first = entities[0] ?? cycle;
              return (
                <div key={`${cycle}-${index}`} className="rounded border border-slate-800 bg-slate-950/70 p-2">
                  <div className="flex flex-wrap items-center gap-1.5">
                    {chain.map((entity, entityIndex) => {
                      const card = cardsByEntity.get(entity);
                      const colors = card ? getStatusColorClasses(card.status) : { background: "bg-slate-700/50", border: "border-slate-600", text: "text-slate-300" };
                      return (
                      <span key={`${entity}-${entityIndex}`} className="inline-flex items-center gap-1">
                        <StatusChip label={card?.title ?? entity} colors={colors} onClick={() => openItem(entity)} />
                        {entityIndex < chain.length - 1 && <ArrowRight className="h-3 w-3 text-slate-500" aria-hidden />}
                      </span>
                      );
                    })}
                  </div>
                  <p className="mt-2 flex items-center gap-1 text-slate-400"><CornerDownLeft className="h-3 w-3" /> Loops back to {cardsByEntity.get(first)?.title ?? first}.</p>
                  <p className="mt-1 text-slate-400">Remove one of these dependencies to break the loop.</p>
                  <div className="mt-2 flex gap-2">
                    <button type="button" onClick={() => openItem(first)} className="rounded bg-cyan-500/20 px-2 py-1 text-cyan-200 hover:bg-cyan-500/30" data-testid="plan-cycle-resolve">Open backlog item</button>
                    <button type="button" onClick={() => inspectEntity(first)} className="rounded px-2 py-1 text-slate-300 hover:bg-slate-800">View in graph</button>
                  </div>
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

/** Node id for a Now-column activity row, so deep links to running work don't misreport a miss. */
function activityNodeId(row: ActivityRow): string | null {
  switch (row.ownerType) {
    case "backlog":
      return row.ownerKind ? `backlog-item/${row.ownerKind}/${row.ownerName}` : null;
    case "initiative":
      return `initiative/${row.ownerName}`;
    case "scenario":
      return `scenario/${row.ownerName}`;
    case "capture":
      return `capture/${row.ownerName}`;
    default:
      return null;
  }
}

function cycleEntityToNodeId(entity: string): string {
  const trimmed = entity.trim();
  if (trimmed.startsWith("backlog-item/") || trimmed.startsWith("scenario/") || trimmed.startsWith("initiative/") || trimmed.startsWith("goal/")) {
    return trimmed;
  }
  const parts = trimmed.split("/");
  return parts.length === 2 ? `backlog-item/${trimmed}` : trimmed;
}

/**
 * Merge the p50/p80 labels into one compact range ("~5 days" + "~10 days" →
 * "~5–10 days"). Falls back to the verbose pair when the units differ.
 */
function compactEtaLabel(p50Label: string, p80Label: string): string {
  const parse = (label: string) => /^(~?)([\d.]+)\s+(.+)$/.exec(label.trim());
  const p50 = parse(p50Label);
  const p80 = parse(p80Label);
  if (p50 && p80 && p50[3] === p80[3]) {
    return p50[2] === p80[2]
      ? `${p50[1]}${p50[2]} ${p50[3]}`
      : `${p50[1]}${p50[2]}–${p80[2]} ${p50[3]}`;
  }
  return `${p50Label} – ${p80Label}`;
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
      {/* Closed state stays deliberately terse — clock icon + merged range.
          Basis ("27 samples" / "priors only") and confidence live in the
          popover, where the full estimator context is legible. */}
      <button
        ref={triggerRef}
        type="button"
        className="flex items-center gap-1.5 rounded bg-slate-800/60 px-2 py-0.5 text-xs transition-colors hover:bg-slate-800"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        aria-label={`ETA ${eta.p50Label} to ${eta.p80Label} (${eta.basisLabel})`}
        title={`ETA ${eta.p50Label} – ${eta.p80Label} · ${eta.basisLabel}`}
        data-testid="plan-eta-strip"
      >
        <Clock className={cn("h-3.5 w-3.5", tone)} aria-hidden />
        <span className="font-medium text-slate-200" data-testid="plan-eta-label">
          {compactEtaLabel(eta.p50Label, eta.p80Label)}
        </span>
      </button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-start"
        mobileSheet
        mobileTitle="How the ETA is computed"
        className="w-72 p-3 text-xs text-slate-200"
        testId="plan-eta-popover"
      >
        <div className="space-y-3 text-xs text-slate-200">
          {/* The sheet header already carries the title on mobile. */}
          <h3 className="hidden text-sm font-semibold text-slate-100 md:block">How the ETA is computed</h3>
          <EtaExplainerContent
            band={{
              p50Label: eta.p50Label,
              p80Label: eta.p80Label,
              remainingItems: eta.remainingItems,
              laneCapacity: eta.laneCapacity,
              basis: eta.basis,
              basisLabel: eta.basisLabel,
              confidence: eta.confidence,
            }}
          />
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

function BeyondHorizon({ cards, highlightedId }: { cards: PlanCardData[]; highlightedId?: string | null }) {
  if (cards.length === 0) return null;
  return (
    <details
      className="rounded-lg border border-slate-800/60 px-2 py-1"
      data-testid="plan-beyond-horizon"
      open={highlightedId ? cards.some((card) => card.id === highlightedId) || undefined : undefined}
    >
      <summary className="cursor-pointer list-none py-1 text-xs font-medium text-slate-500 hover:text-slate-300">
        <ChevronDown className="mr-1 inline h-3.5 w-3.5" aria-hidden />
        beyond horizon ({cards.length})
      </summary>
      <div className="mt-1 space-y-1.5 pb-1">
        {cards.map((card) => (
          <PlanCardView key={card.id} card={card} showWave highlighted={card.id === highlightedId} />
        ))}
      </div>
    </details>
  );
}

/**
 * Shown when a detail page deep-linked to a card (?select=) that isn't on
 * the current board — filtered out, goal-scoped away, or completed outside
 * the Done window. An honest notice with escape hatches instead of silently
 * rewriting the user's filters.
 */
function SelectMissNotice({
  nodeId,
  hasFilters,
  onClearFilters,
  onDismiss,
}: {
  nodeId: string;
  hasFilters: boolean;
  onClearFilters: () => void;
  onDismiss: () => void;
}) {
  const navigate = useNavigate();
  const label = nodeId.split("/").pop() || nodeId;
  const detailHref = detailPathFromNodeId(nodeId);
  const linkClass = "underline decoration-amber-400/40 underline-offset-2 hover:text-amber-100";
  return (
    <span
      className="inline-flex max-w-full flex-wrap items-center gap-x-2 gap-y-1 rounded bg-amber-500/10 px-2 py-0.5 text-xs text-amber-200"
      data-testid="plan-select-miss"
    >
      <AlertTriangle className="h-3.5 w-3.5 shrink-0" aria-hidden />
      <span className="max-w-[14rem] truncate font-medium" title={nodeId}>{label}</span>
      <span className="text-amber-200/70">isn&apos;t on the current board</span>
      {detailHref && (
        <Link to={detailHref} className={linkClass} data-testid="plan-select-miss-details">
          details
        </Link>
      )}
      <button
        type="button"
        onClick={() => navigate(graphPath({ lens: "focus", focus: nodeId, select: nodeId }))}
        className={linkClass}
        data-testid="plan-select-miss-graph"
      >
        graph
      </button>
      {hasFilters && (
        <button type="button" onClick={onClearFilters} className={linkClass} data-testid="plan-select-miss-clear">
          clear filters
        </button>
      )}
      <button
        type="button"
        onClick={onDismiss}
        aria-label="Dismiss"
        className="rounded p-0.5 hover:bg-amber-500/20"
        data-testid="plan-select-miss-dismiss"
      >
        <X className="h-3 w-3" aria-hidden />
      </button>
    </span>
  );
}

export function PlanBoard() {
  const { board, loading, error, windowSeconds, setWindowSeconds, refresh } = usePlanData();
  const urlState = usePlanUrlState();
  const snoozedKeys = useSnoozedKeys();
  const filterDrawerOpen = usePlanDataStore((s) => s.filterDrawerOpen);
  const setFilterDrawerOpen = usePlanDataStore((s) => s.setFilterDrawerOpen);

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

  // ----- ?select= deep link (detail-page "Plan" pill) -----------------------
  const [searchParams, setSearchParams] = useSearchParams();
  const urlSelect = searchParams.get("select");
  const [highlightId, setHighlightId] = useState<string | null>(null);
  const [missedSelect, setMissedSelect] = useState<string | null>(null);
  const operationsView = useOperationsStore((s) => s.view);

  // Every node id currently represented on the board: column cards plus
  // Now-column activity rows (running work never appears as a plan card).
  const presentCardIds = useMemo(() => {
    const ids = new Set<string>();
    const groups = [...next.groups, ...laterSplit.visible, ...(board?.done.groups ?? [])];
    for (const group of groups) {
      for (const card of group.cards) ids.add(card.id);
    }
    for (const card of laterSplit.beyond) ids.add(card.id);
    for (const row of operationsView?.activities ?? []) {
      const nodeId = activityNodeId(row);
      if (nodeId) ids.add(nodeId);
    }
    return ids;
  }, [board?.done.groups, laterSplit, next.groups, operationsView?.activities]);

  // Consume the param once the board is loaded: highlight a present card,
  // otherwise surface the miss notice. The param is one-shot state, so it is
  // stripped from the URL either way.
  useEffect(() => {
    if (!urlSelect || !board) return;
    setSearchParams((prev) => {
      const nextParams = new URLSearchParams(prev);
      nextParams.delete("select");
      nextParams.delete("focus");
      return nextParams;
    }, { replace: true });
    if (presentCardIds.has(urlSelect)) {
      setHighlightId(urlSelect);
      setMissedSelect(null);
    } else {
      setMissedSelect(urlSelect);
    }
  }, [board, presentCardIds, setSearchParams, urlSelect]);

  // After a miss, "clear filters" (or any board change) may reveal the card —
  // promote the miss to a highlight as soon as it shows up.
  useEffect(() => {
    if (missedSelect && presentCardIds.has(missedSelect)) {
      setHighlightId(missedSelect);
      setMissedSelect(null);
    }
  }, [missedSelect, presentCardIds]);

  // Scroll the highlighted card into view, then let the ring fade.
  useEffect(() => {
    if (!highlightId) return;
    const raf = window.requestAnimationFrame(() => {
      document
        .querySelector(`[data-testid="plan-card-${CSS.escape(highlightId)}"]`)
        ?.scrollIntoView({ behavior: "smooth", block: "center", inline: "center" });
    });
    const timer = window.setTimeout(() => setHighlightId(null), 4000);
    return () => {
      window.cancelAnimationFrame(raf);
      window.clearTimeout(timer);
    };
  }, [highlightId]);

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
  const cycleCardsByEntity = new Map<string, PlanCardData>();
  for (const group of [...next.groups, ...later.groups, ...board.done.groups]) {
    for (const card of group.cards) {
      if (card.itemKind && card.itemName) cycleCardsByEntity.set(`${card.itemKind}/${card.itemName}`, card);
    }
  }
  const hiddenSnoozed = next.hiddenCount + later.hiddenCount;

  return (
    <PlanBoardActions onBoardRefresh={() => void refresh()}>
    <div className="flex h-full min-h-0 flex-col" data-testid="plan-board">
      <div className="flex flex-wrap items-center gap-2 px-4 py-2">
        <GoalPicker goal={urlState.goal} onSelect={urlState.setGoal} />
        <CycleWarning cycles={cycles} cardsByEntity={cycleCardsByEntity} />
        {hiddenSnoozed > 0 && (
          <span
            className="text-xs text-slate-600"
            title={`${hiddenSnoozed} snoozed ${hiddenSnoozed === 1 ? "item is" : "items are"} hidden`}
            data-testid="plan-snoozed-hidden-count"
          >
            {hiddenSnoozed} snoozed
          </span>
        )}
        <EtaStrip eta={board.meta.eta} goal={urlState.goal} />
        {missedSelect && (
          <SelectMissNotice
            nodeId={missedSelect}
            hasFilters={urlState.hasFilters || Boolean(urlState.goal)}
            onClearFilters={() => {
              urlState.resetFilters();
              urlState.setGoal("");
            }}
            onDismiss={() => setMissedSelect(null)}
          />
        )}
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
          highlightedId={highlightId}
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
          highlightedId={highlightId}
          footer={laterSplit.beyond.length > 0 ? <BeyondHorizon cards={laterSplit.beyond} highlightedId={highlightId} /> : undefined}
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
          highlightedId={highlightId}
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
