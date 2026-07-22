/**
 * GoalDetailsPage — inspect a goal's targets, computed scope, blockers, and ETA.
 */

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  Archive,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Clock3,
  ListChecks,
  ShieldAlert,
  Target,
  Trash2,
  Workflow,
} from "lucide-react";
import { BacklogCard } from "../components/backlog/backlog-card";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { GOAL_LENSES } from "../components/detail/lens-options";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import type { ActionMenuItem } from "../components/ui/action-menu";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { goalsService } from "../services/goals-service";
import { GOALS_QUERY_KEY } from "../surfaces/plan/hooks/useGoals";
import type { GoalScope, GoalScopeEntities, GoalWithScope } from "../types/goal";
import { ENTITY_TYPE_ICONS } from "../types/constants";
import {
  backlogDetailPath,
  graphPath,
	 routeTargetToNodeId,
} from "../app/routes/route-paths";

const MAX_PRIORITY = 10;
const MIN_PRIORITY = 0;

interface ParsedGoalRef {
  ref: string;
  label: string;
  href: string | null;
  kindLabel: string;
}

function parseGoalRef(ref: string): ParsedGoalRef {
  const slashIdx = ref.indexOf("/");
  if (slashIdx > 0) {
    const kind = ref.slice(0, slashIdx);
    const name = ref.slice(slashIdx + 1);
    return {
      ref,
      label: name,
      href: name ? backlogDetailPath(kind, name) : null,
      kindLabel: kind,
    };
  }

  return {
    ref,
    label: ref,
    href: null,
    kindLabel: "ref",
  };
}

function RefChip({ refId }: { refId: string }) {
  const parsed = parseGoalRef(refId);
  const content = (
    <>
      <span className="rounded bg-slate-950 px-1.5 py-0.5 text-[10px] uppercase tracking-wider text-slate-500">
        {parsed.kindLabel}
      </span>
      <span className="truncate">{parsed.label}</span>
    </>
  );

  const className = "inline-flex min-w-0 max-w-full items-center gap-2 rounded-full border border-slate-800 bg-slate-900 px-3 py-1.5 text-sm text-slate-200 transition-colors hover:border-slate-700";

  if (!parsed.href) {
    return <span className={className}>{content}</span>;
  }

  return (
    <Link to={parsed.href} className={className} data-testid={`goal-ref-${parsed.ref}`}>
      {content}
    </Link>
  );
}

/**
 * One scope ref rendered as the standard BacklogCard fed by the server's scope
 * hydration. Falls back to the lightweight RefChip when a ref didn't
 * resolve (e.g. a target typo or a deleted item).
 */
function ScopeEntityCard({ refId, entities }: { refId: string; entities?: GoalScopeEntities }) {
  const navigate = useNavigate();

  const item = entities?.items[refId];
  if (item) {
    return (
      <button
        type="button"
        onClick={() => navigate(backlogDetailPath(item.kind, item.name))}
        className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
        data-testid={`goal-ref-card-${refId}`}
      >
        <BacklogCard item={item} />
      </button>
    );
  }

  return <RefChip refId={refId} />;
}

function RefList({ refs, emptyText, entities }: { refs: string[]; emptyText: string; entities?: GoalScopeEntities }) {
  if (refs.length === 0) {
    return <p className="text-sm text-slate-500">{emptyText}</p>;
  }
  if (!entities) {
    return (
      <div className="flex flex-wrap gap-2">
        {refs.map((ref) => <RefChip key={ref} refId={ref} />)}
      </div>
    );
  }
  return (
    <div className="grid gap-2 lg:grid-cols-2">
      {refs.map((ref) => <ScopeEntityCard key={ref} refId={ref} entities={entities} />)}
    </div>
  );
}

function ScopeProgress({ scope }: { scope: GoalScope }) {
  const pct = Math.max(0, Math.min(100, Math.round(scope.progressPct)));
  const remaining = Math.max(0, scope.total - scope.completedCount - scope.blockedCount);
  return (
    <div className="space-y-3">
      <div className="flex items-end justify-between gap-4">
        <div>
          <p className="text-3xl font-semibold text-slate-100">{pct}%</p>
          <p className="text-sm text-slate-500">{scope.completedCount} of {scope.total} complete</p>
        </div>
        {scope.blockedCount > 0 && (
          <span className="rounded-full border border-red-500/30 bg-red-500/10 px-3 py-1 text-sm text-red-300">
            {scope.blockedCount} blocked
          </span>
        )}
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-slate-800" aria-label={`Goal progress ${pct}%`}>
        <div className="h-full bg-emerald-500" style={{ width: `${pct}%` }} />
      </div>
      <div className="grid gap-2 text-sm sm:grid-cols-4">
        <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
          <p className="text-xs text-slate-500">Closure</p>
          <p className="text-lg font-semibold text-slate-100">{scope.closure.length}</p>
        </div>
        <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
          <p className="text-xs text-slate-500">Ready</p>
          <p className="text-lg font-semibold text-cyan-300">{scope.ready.length}</p>
        </div>
        <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
          <p className="text-xs text-slate-500">Blocked</p>
          <p className="text-lg font-semibold text-red-300">{scope.blockedCount}</p>
        </div>
        <div className="rounded-lg border border-slate-800 bg-slate-900/60 p-3">
          <p className="text-xs text-slate-500">Remaining</p>
          <p className="text-lg font-semibold text-slate-100">{remaining}</p>
        </div>
      </div>
    </div>
  );
}

function ScopeHistory({ goal }: { goal: GoalWithScope["goal"] }) {
  if (goal.scopeHistory.length === 0) {
    return <p className="text-sm text-slate-500">No scope snapshots have been recorded yet.</p>;
  }

  return (
    <div className="overflow-hidden rounded-lg border border-slate-800">
      <div className="grid grid-cols-[1.4fr_0.8fr_0.8fr_0.8fr] border-b border-slate-800 bg-slate-900/80 px-3 py-2 text-xs font-medium uppercase tracking-wider text-slate-500">
        <span>Snapshot</span>
        <span>Targets</span>
        <span>Closure</span>
        <span>Done</span>
      </div>
      {goal.scopeHistory.map((snapshot) => (
        <div
          key={`${snapshot.at}-${snapshot.closureSize}-${snapshot.completed}`}
          className="grid grid-cols-[1.4fr_0.8fr_0.8fr_0.8fr] border-b border-slate-900 px-3 py-2 text-sm last:border-b-0"
        >
          <span className="text-slate-300">{snapshot.at ? formatRelativeTime(snapshot.at) : "Unknown"}</span>
          <span className="text-slate-400">{snapshot.targetCount}</span>
          <span className="text-slate-400">{snapshot.closureSize}</span>
          <span className="text-slate-400">{snapshot.completed}</span>
        </div>
      ))}
    </div>
  );
}

export function GoalDetailsPage() {
  const { name = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [confirmArchive, setConfirmArchive] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const decodedName = useMemo(() => decodeURIComponent(name), [name]);
  const nodeId = routeTargetToNodeId({ entityType: "goal", name: decodedName });

  const query = useQuery({
    queryKey: ["goal", decodedName],
    queryFn: () => goalsService.get(decodedName),
    enabled: decodedName.length > 0,
    ...defaultQueryOptions,
  });

  const invalidateGoal = () => {
    void queryClient.invalidateQueries({ queryKey: ["goal", decodedName] });
    void queryClient.invalidateQueries({ queryKey: GOALS_QUERY_KEY });
  };

  const priorityMutation = useMutation({
    mutationFn: (priority: number) => goalsService.setPriority(decodedName, priority),
    onSuccess: invalidateGoal,
  });

  const archiveMutation = useMutation({
    mutationFn: () => goalsService.archive(decodedName),
    onSuccess: () => {
      invalidateGoal();
      setConfirmArchive(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => goalsService.remove(decodedName),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: GOALS_QUERY_KEY });
      navigate("/plan", { replace: true });
    },
  });

  if (query.isLoading) {
    return (
      <DetailPageLayout
        header={<DetailPageHeader entityType="Goal" title="Loading goal" nodeId={null} lenses={[]} />}
      >
        <PageLoadingState label="Loading goal..." variant="detail" />
      </DetailPageLayout>
    );
  }

  if (query.error || !query.data) {
    return (
      <DetailPageLayout
        header={<DetailPageHeader entityType="Goal" title="Goal unavailable" nodeId={null} lenses={[]} />}
      >
        <ErrorState
          error={query.error instanceof Error ? query.error : undefined}
          title="Unable to load goal"
          onRetry={() => void query.refetch()}
        />
      </DetailPageLayout>
    );
  }

  const goal = query.data.goal;
  const scope = query.data.scope;
  const eta = query.data.eta;
  const busy = priorityMutation.isPending || archiveMutation.isPending || deleteMutation.isPending;

  const changePriority = (delta: number) => {
    const next = Math.max(MIN_PRIORITY, Math.min(MAX_PRIORITY, goal.priority + delta));
    if (next !== goal.priority) {
      priorityMutation.mutate(next);
    }
  };

  const menuActions: ActionMenuItem[] = [
    ...(goal.status !== "archived" ? [{
      label: "Archive",
      icon: <Archive />,
      disabled: busy,
      onSelect: () => setConfirmArchive(true),
      testId: "goal-archive",
    }] : []),
    {
      label: "Delete",
      icon: <Trash2 />,
      disabled: busy,
      destructive: true,
      onSelect: () => setConfirmDelete(true),
      testId: "goal-delete",
    },
  ];

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Goal"
          title={goal.title || goal.name}
          subtitle={goal.description || goal.name}
          status={goal.status}
          nodeId={nodeId}
          lenses={GOAL_LENSES}
          onDrillToLens={() => navigate(graphPath({ lens: "plan", goal: goal.name }))}
          menuActions={menuActions}
        />
      }
      bodyClassName="mx-auto w-full max-w-3xl"
    >
      <div className="min-w-0 space-y-4">
        <DetailSection title="Overview" icon={ENTITY_TYPE_ICONS.goal} hideDivider data-testid="goal-overview">
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-2">
              <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  Progress
                </div>
                <div className="mt-2 text-xl font-semibold text-slate-100 sm:text-2xl">
                  {Math.max(0, Math.min(100, Math.round(scope.progressPct)))}%
                </div>
                <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                  {scope.completedCount} of {scope.total} complete
                </p>
              </div>
              <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3" data-testid="goal-eta-card">
                <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                  <Clock3 className="h-3.5 w-3.5" />
                  ETA Band
                </div>
                <div className="mt-2 text-xl font-semibold text-slate-100 sm:text-2xl">
                  {eta ? `${eta.p50Label}-${eta.p80Label}` : "—"}
                </div>
                <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                  {eta
                    ? `${eta.basisLabel || eta.basis} · ${eta.confidence || "unknown"} confidence · lane capacity ${eta.laneCapacity}`
                    : "No ETA is available for this goal yet."}
                </p>
              </div>
              <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                  <ChevronUp className="h-3.5 w-3.5" />
                  Priority
                </div>
                <div className="mt-2 flex items-center gap-2">
                  <span className="text-xl font-semibold text-slate-100 sm:text-2xl">P{goal.priority}</span>
                  <span className="flex flex-col">
                    <button
                      type="button"
                      onClick={() => changePriority(1)}
                      disabled={busy || goal.priority >= MAX_PRIORITY}
                      className="rounded p-0.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 disabled:opacity-40"
                      aria-label="Raise goal priority"
                      data-testid="goal-priority-up"
                    >
                      <ChevronUp className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => changePriority(-1)}
                      disabled={busy || goal.priority <= MIN_PRIORITY}
                      className="rounded p-0.5 text-slate-400 hover:bg-slate-800 hover:text-slate-200 disabled:opacity-40"
                      aria-label="Lower goal priority"
                      data-testid="goal-priority-down"
                    >
                      <ChevronDown className="h-3.5 w-3.5" />
                    </button>
                  </span>
                </div>
                <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">Drives goal-directed drain order</p>
              </div>
              <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                  <ShieldAlert className="h-3.5 w-3.5" />
                  Blocked
                </div>
                <div className={`mt-2 text-xl font-semibold sm:text-2xl ${scope.blockedCount > 0 ? "text-red-300" : "text-slate-100"}`}>
                  {scope.blockedCount}
                </div>
                <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                  {scope.blockedCount > 0 ? "items blocked right now" : "nothing blocked right now"}
                </p>
              </div>
            </div>

            <div className="flex flex-wrap gap-6 text-xs text-slate-500">
              <div>
                <span className="uppercase tracking-wider">Name</span>{" "}
                <span className="text-slate-400">{goal.name}</span>
              </div>
              <div>
                <span className="uppercase tracking-wider">Created</span>{" "}
                <span className="text-slate-400">{goal.created ? formatRelativeTime(goal.created) : "unknown"}</span>
              </div>
              <div>
                <span className="uppercase tracking-wider">Updated</span>{" "}
                <span className="text-slate-400">{goal.updated ? formatRelativeTime(goal.updated) : "never"}</span>
              </div>
            </div>
          </div>
        </DetailSection>

        <DetailSection title="Targets" icon={Target} storageKey="goal.targets" data-testid="goal-targets">
          <RefList refs={goal.targets} emptyText="This goal does not have explicit targets yet." entities={query.data.scopeEntities} />
        </DetailSection>

        <DetailSection title="Scope Progress" icon={Workflow} storageKey="goal.scope-progress" data-testid="goal-scope">
          <ScopeProgress scope={scope} />
        </DetailSection>

        <DetailSection title="Ready Work" icon={ListChecks} storageKey="goal.ready-work" data-testid="goal-ready">
          <RefList refs={scope.ready} emptyText="No ready work in this goal right now." entities={query.data.scopeEntities} />
        </DetailSection>

        <DetailSection title="Blocked Work" icon={ListChecks} storageKey="goal.blocked-work" data-testid="goal-blocked">
          <RefList refs={scope.blocked} emptyText="No blocked work in this goal right now." entities={query.data.scopeEntities} />
        </DetailSection>

        <DetailSection title="Scope Creep" icon={Workflow} storageKey="goal.scope-creep" defaultOpen={false} data-testid="goal-history">
          <ScopeHistory goal={goal} />
        </DetailSection>
      </div>

      <ConfirmDialog
        isOpen={confirmArchive}
        onClose={() => setConfirmArchive(false)}
        onConfirm={() => archiveMutation.mutate()}
        title="Archive goal"
        description={`Archive "${goal.title || goal.name}"? It will stop appearing in active goal lists.`}
        confirmLabel="Archive"
        isLoading={archiveMutation.isPending}
        testIds={{ dialog: "goal-archive-confirm", confirmButton: "goal-archive-confirm-submit" }}
      />
      <ConfirmDialog
        isOpen={confirmDelete}
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => deleteMutation.mutate()}
        title="Delete goal"
        description={`Permanently delete "${goal.title || goal.name}"? This does not delete the target work items.`}
        confirmationText={goal.name}
        confirmLabel="Delete"
        isLoading={deleteMutation.isPending}
        testIds={{ dialog: "goal-delete-confirm", confirmButton: "goal-delete-confirm-submit" }}
      />
    </DetailPageLayout>
  );
}
