/**
 * GoalDetailsPage — inspect a goal's targets, computed scope, blockers, and ETA.
 */

import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import {
  Activity,
  Archive,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Clock3,
  Edit,
  Files,
  GitPullRequestArrow,
  ListChecks,
  Loader2,
  Network,
  Plus,
  ShieldAlert,
  Target,
  Trash2,
  Workflow,
  X,
} from "lucide-react";
import { BacklogCard } from "../components/backlog/backlog-card";
import { EntityFileWorkspace } from "../components/files/entity-file-workspace";
import type { FileActionType } from "../components/files/entity-file-browser";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { ProposalSessionsPanel } from "../components/session/ProposalSessionsPanel";
import { Button } from "../components/ui/button";
import { CompactTabBar, type CompactTabItem } from "../components/ui/compact-tab-bar";
import { FileServiceProvider } from "../contexts/FileServiceContext";
import { GOAL_LENSES } from "../components/detail/lens-options";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { Drawer } from "../components/ui/drawer";
import { Input } from "../components/ui/input";
import { MarkdownRenderer } from "../components/markdown/MarkdownRenderer";
import type { ActionMenuItem } from "../components/ui/action-menu";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { goalsService } from "../services/goals-service";
import { nextActionService } from "../services/next-action-service";
import { createGoalFileServiceAdapter } from "../services/goals-file-service-adapter";
import { GOALS_QUERY_KEY } from "../surfaces/plan/hooks/useGoals";
import type { BacklogFile } from "../types/backlog";
import type { GoalMilestone, GoalScope, GoalScopeEntities, GoalWithScope } from "../types/goal";
import { ENTITY_TYPE_ICONS, nextActionIcon } from "../types/constants";
import {
  backlogDetailPath,
  graphPath,
	 routeTargetToNodeId,
} from "../app/routes/route-paths";
import { selectors } from "../consts/selectors";
import { useTransientHighlight } from "../hooks/useTransientHighlight";
import { useBacklogStore } from "../stores";

const MAX_PRIORITY = 10;
const MIN_PRIORITY = 0;
const GOAL_TABS = ["overview", "milestones", "decide", "files", "activity", "related"] as const;
type GoalTab = typeof GOAL_TABS[number];

const GOAL_TAB_ITEMS: CompactTabItem<GoalTab>[] = [
  { value: "overview", label: "Overview", icon: ENTITY_TYPE_ICONS.goal },
  { value: "milestones", label: "Milestones", icon: ListChecks },
  { value: "decide", label: "Decide", icon: GitPullRequestArrow },
  { value: "files", label: "Files", icon: Files },
  { value: "activity", label: "Activity", icon: Activity },
  { value: "related", label: "Related", icon: Network },
];

function isGoalTab(value: string | null): value is GoalTab {
  return value !== null && (GOAL_TABS as readonly string[]).includes(value);
}

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

function MilestoneGroups({ milestones, entities, onEdit, onArchive, onManageItems }: { milestones: GoalMilestone[]; entities?: GoalScopeEntities; onEdit: (milestone: GoalMilestone) => void; onArchive: (milestone: GoalMilestone) => void; onManageItems: (milestone: GoalMilestone) => void }) {
  const active = milestones.filter((milestone) => !milestone.archivedAt);
  if (active.length === 0) {
    return <p className="text-sm text-slate-500">This goal does not have milestone groups yet.</p>;
  }

  return (
    <div>
      {active.map((milestone) => (
        <DetailSection
          key={milestone.name}
          title={milestone.title || milestone.name}
          action={<div className="flex gap-1"><button type="button" onClick={() => onManageItems(milestone)} className="rounded p-1 text-slate-400 hover:bg-white/10 hover:text-white" aria-label={`Manage ${milestone.title || milestone.name} items`}><Target className="h-3.5 w-3.5" /></button><button type="button" onClick={() => onEdit(milestone)} className="rounded p-1 text-slate-400 hover:bg-white/10 hover:text-white" aria-label={`Edit ${milestone.title || milestone.name}`}><Edit className="h-3.5 w-3.5" /></button><button type="button" onClick={() => onArchive(milestone)} className="rounded p-1 text-slate-400 hover:bg-red-500/10 hover:text-red-300" aria-label={`Archive ${milestone.title || milestone.name}`}><Archive className="h-3.5 w-3.5" /></button></div>}
          storageKey={`goal.milestone.${milestone.name}`}
          className="first:mt-0"
          data-testid={`goal-milestone-${milestone.name}`}
        >
          <div className="space-y-3">
            {milestone.description && <p className="text-sm text-slate-400">{milestone.description}</p>}
            <p className="text-xs font-medium uppercase tracking-wider text-slate-500">{milestone.items.length} assigned</p>
            <RefList refs={milestone.items} emptyText="No work assigned to this milestone yet." entities={entities} />
          </div>
        </DetailSection>
      ))}
    </div>
  );
}

export function GoalDetailsPage() {
  const { name = "" } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const [confirmArchive, setConfirmArchive] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [activeTab, setActiveTab] = useState<GoalTab>(() => isGoalTab(searchParams.get("tab")) ? searchParams.get("tab") as GoalTab : "overview");
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);
  const [milestoneEditor, setMilestoneEditor] = useState<GoalMilestone | null>(null);
  const [milestoneDraft, setMilestoneDraft] = useState<GoalMilestone | null>(null);
  const [goalEditorOpen, setGoalEditorOpen] = useState(false);
  const [goalDraft, setGoalDraft] = useState({ title: "", description: "", priority: 0 });
  const [highlightTarget, setHighlightTarget] = useState<string | null>(null);
  const [targetEditorOpen, setTargetEditorOpen] = useState(false);
  const [milestoneItemEditor, setMilestoneItemEditor] = useState<GoalMilestone | null>(null);
  const [milestoneItemDraft, setMilestoneItemDraft] = useState<string[]>([]);
  useTransientHighlight({ targetSelector: highlightTarget, highlightClass: "ring-2 ring-cyan-400/60 rounded-lg", scrollIntoView: true });

  const decodedName = useMemo(() => decodeURIComponent(name), [name]);
  const nodeId = routeTargetToNodeId({ entityType: "goal", name: decodedName });

  const query = useQuery({
    queryKey: ["goal", decodedName],
    queryFn: () => goalsService.get(decodedName),
    enabled: decodedName.length > 0,
    ...defaultQueryOptions,
  });
  const filesQuery = useQuery({
    queryKey: ["goal", decodedName, "files"],
    queryFn: () => goalsService.getFiles(decodedName),
    enabled: decodedName.length > 0 && activeTab === "files",
    ...defaultQueryOptions,
  });
  const nextActionsQuery = useQuery({
    queryKey: ["next-actions-feed", "goal", decodedName],
    queryFn: () => nextActionService.getFeed(),
    enabled: decodedName.length > 0,
    staleTime: 15_000,
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
  const planMutation = useMutation({ mutationFn: () => goalsService.startPlan(decodedName), onSuccess: invalidateGoal });
  const closeOutMutation = useMutation({ mutationFn: () => goalsService.closeOut(decodedName), onSuccess: invalidateGoal });
  const discoverMutation = useMutation({ mutationFn: () => goalsService.startDiscover(decodedName), onSuccess: invalidateGoal });
  const saveMilestoneMutation = useMutation({
    mutationFn: (milestone: GoalMilestone) => milestoneEditor ? goalsService.updateMilestone(decodedName, milestone) : goalsService.createMilestone(decodedName, milestone),
    onSuccess: () => { invalidateGoal(); setMilestoneEditor(null); setMilestoneDraft(null); },
  });
  const archiveMilestoneMutation = useMutation({ mutationFn: (milestone: string) => goalsService.archiveMilestone(decodedName, milestone), onSuccess: invalidateGoal });
  const updateMilestoneItemsMutation = useMutation({
    mutationFn: async ({ milestone, items }: { milestone: GoalMilestone; items: string[] }) => {
      const add = items.filter((item) => !milestone.items.includes(item));
      const remove = milestone.items.filter((item) => !items.includes(item));
      if (add.length > 0) await goalsService.assignMilestoneItems(decodedName, milestone.name, add);
      if (remove.length > 0) await goalsService.unassignMilestoneItems(decodedName, milestone.name, remove);
    },
    onSuccess: () => { invalidateGoal(); setMilestoneItemEditor(null); },
  });
  const updateGoalMutation = useMutation({ mutationFn: (input: { title: string; description: string; priority: number }) => goalsService.update(decodedName, input), onSuccess: () => { invalidateGoal(); setGoalEditorOpen(false); } });
  const addGoalTargetMutation = useMutation({ mutationFn: (target: string) => goalsService.addTargets(decodedName, [target]), onSuccess: invalidateGoal });
  const removeGoalTargetMutation = useMutation({ mutationFn: (target: string) => goalsService.removeTargets(decodedName, [target]), onSuccess: invalidateGoal });
  const backlogItems = useBacklogStore((state) => state.items);
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const goalFileService = useMemo(() => createGoalFileServiceAdapter(decodedName), [decodedName]);
  const fileActionMutation = useMutation({
    mutationFn: ({ action, target, destinationPath }: { action: FileActionType; target: BacklogFile; destinationPath?: string }) => {
      switch (action) {
        case "rename": return goalsService.renameFile(decodedName, target.path, destinationPath ?? "");
        case "move": return goalsService.moveFile(decodedName, target.path, destinationPath ?? "");
        case "copy": return goalsService.copyFile(decodedName, target.path, destinationPath ?? "");
        case "delete": return goalsService.deleteFile(decodedName, target.path);
      }
    },
    onSuccess: (result, variables) => {
      setSelectedFile(result.file ?? (variables.action === "delete" ? null : selectedFile));
      void queryClient.invalidateQueries({ queryKey: ["goal", decodedName, "files"] });
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
  const goalAction = nextActionsQuery.data?.entries.find((entry) => entry.entity_kind === "goal" && entry.entity_ref === decodedName);
  const runGoalAction = () => {
    if (!goalAction) return;
    switch (goalAction.action.id) {
      case "close_out": closeOutMutation.mutate(); break;
      case "plan_goal": planMutation.mutate(); break;
      case "decide": setActiveTab("decide"); setSearchParams({ tab: "decide" }, { replace: true }); break;
      default: navigate(`/goals/${encodeURIComponent(decodedName)}?drawer=decisions`);
    }
  };

  const changePriority = (delta: number) => {
    const next = Math.max(MIN_PRIORITY, Math.min(MAX_PRIORITY, goal.priority + delta));
    if (next !== goal.priority) {
      priorityMutation.mutate(next);
    }
  };

  const revealSection = (testId: "goal-scope" | "goal-blocked") => {
    const toggle = document.querySelector<HTMLButtonElement>(`[data-testid="${testId}-toggle"]`);
    if (toggle?.getAttribute("aria-expanded") === "false") toggle.click();
    setHighlightTarget(null);
    requestAnimationFrame(() => setHighlightTarget(`[data-testid="${testId}"]`));
  };

  const menuActions: ActionMenuItem[] = [
    {
      label: "Edit",
      icon: <Edit />,
      disabled: busy,
      onSelect: () => { setGoalDraft({ title: goal.title, description: goal.description ?? "", priority: goal.priority }); setGoalEditorOpen(true); },
      testId: "goal-edit",
    },
    ...(goal.status === "active" ? [{
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
          // No subtitle: the description is rendered in full (and editable) in
          // the Overview tab. A truncated copy here only added header clutter.
          status={goal.status}
          nodeId={nodeId}
          lenses={GOAL_LENSES}
          onDrillToLens={() => navigate(graphPath({ lens: "plan", goal: goal.name }))}
          primaryAction={goalAction ? <GoalPrimaryAction actionId={goalAction.action.id} label={goalAction.action.compact_label} pending={closeOutMutation.isPending || planMutation.isPending} onRun={runGoalAction} /> : undefined}
          menuActions={menuActions}
          showLenses={activeTab === "overview"}
          tabBar={
            <CompactTabBar
              items={GOAL_TAB_ITEMS}
              activeValue={activeTab}
              onValueChange={(value) => {
                setActiveTab(value);
                setSearchParams(value === "overview" ? {} : { tab: value }, { replace: true });
              }}
              aria-label="Goal sections"
              className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3"
              data-testid={selectors.goalDetails.tabRow}
              tabTestIdPrefix="goal-details-tab"
            />
          }
        />
      }
      fullBleed={activeTab === "files"}
      bodyClassName={activeTab === "files" ? undefined : "mx-auto w-full max-w-3xl"}
    >
      <div className={activeTab === "files" ? "flex h-full min-h-0 min-w-0 flex-col" : "min-w-0 space-y-4"} data-testid={selectors.goalDetails.page}>
        {activeTab === "overview" && <DetailSection title="Overview" icon={ENTITY_TYPE_ICONS.goal} hideDivider data-testid="goal-overview">
          <div className="space-y-4">
            <DetailSection title="Description" action={<button type="button" onClick={() => { setGoalDraft({ title: goal.title, description: goal.description ?? "", priority: goal.priority }); setGoalEditorOpen(true); }} className="rounded p-1 text-slate-400 hover:bg-white/10 hover:text-white" aria-label="Edit goal description"><Edit className="h-3.5 w-3.5" /></button>}>
              <MarkdownRenderer content={goal.description || "No description provided."} className="prose-sm-slate text-sm leading-relaxed text-slate-300" />
            </DetailSection>
            <div className="grid grid-cols-2 gap-2">
              <button type="button" onClick={() => revealSection("goal-scope")} className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3 text-left transition-colors hover:border-cyan-500/40">
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
              </button>
              <button type="button" onClick={() => revealSection("goal-scope")} className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3 text-left transition-colors hover:border-cyan-500/40" data-testid="goal-eta-card">
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
              </button>
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
              <button type="button" onClick={() => revealSection("goal-blocked")} className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3 text-left transition-colors hover:border-cyan-500/40">
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
              </button>
            </div>

            <Button variant="outline" size="sm" onClick={() => navigate(graphPath({ lens: "stats", goal: goal.name }))}>Full stats breakdown</Button>

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
        </DetailSection>}

        {activeTab === "overview" && <DetailSection title="Targets" icon={Target} storageKey="goal.targets" data-testid="goal-targets" action={<button type="button" onClick={() => { if (backlogItems.length === 0) void fetchBacklog(); setTargetEditorOpen(true); }} className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-cyan-300 hover:bg-cyan-500/10"><Plus className="h-3.5 w-3.5" />Add</button>}>
          {goal.targets.length === 0 ? <p className="text-sm text-slate-500">This goal does not have explicit targets yet.</p> : <div className="flex flex-wrap gap-2">{goal.targets.map((target) => <span key={target} className="inline-flex items-center gap-1 rounded-full border border-slate-800 bg-slate-900 pr-1"><RefChip refId={target} /><button type="button" onClick={() => removeGoalTargetMutation.mutate(target)} disabled={removeGoalTargetMutation.isPending} className="rounded-full p-1 text-slate-400 hover:bg-red-500/15 hover:text-red-300" aria-label={`Remove ${target} from goal`}><X className="h-3.5 w-3.5" /></button></span>)}</div>}
        </DetailSection>}

        {activeTab === "overview" && <DetailSection title="Scope Progress" icon={Workflow} storageKey="goal.scope-progress" data-testid="goal-scope">
          <ScopeProgress scope={scope} />
        </DetailSection>}

        {activeTab === "milestones" && <DetailSection title="Milestone Groups" icon={ListChecks} storageKey="goal.milestones" data-testid="goal-milestones" action={<button type="button" onClick={() => { setMilestoneEditor(null); setMilestoneDraft({ name: "", title: "", description: "", items: [], acceptanceCriteria: [], dependsOn: [] }); }} className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs text-cyan-300 hover:bg-cyan-500/10"><Plus className="h-3.5 w-3.5" />Add</button>}>
          <MilestoneGroups milestones={goal.milestones} entities={query.data.scopeEntities} onEdit={(milestone) => { setMilestoneEditor(milestone); setMilestoneDraft({ ...milestone }); }} onArchive={(milestone) => archiveMilestoneMutation.mutate(milestone.name)} onManageItems={(milestone) => { setMilestoneItemEditor(milestone); setMilestoneItemDraft(milestone.items); }} />
        </DetailSection>}

        {activeTab === "overview" && <DetailSection title="Ready Work" icon={ListChecks} storageKey="goal.ready-work" data-testid="goal-ready">
          <RefList refs={scope.ready} emptyText="No ready work in this goal right now." entities={query.data.scopeEntities} />
        </DetailSection>}

        {activeTab === "overview" && <DetailSection title="Blocked Work" icon={ListChecks} storageKey="goal.blocked-work" data-testid="goal-blocked">
          <RefList refs={scope.blocked} emptyText="No blocked work in this goal right now." entities={query.data.scopeEntities} />
        </DetailSection>}

        {activeTab === "overview" && <DetailSection title="Scope Creep" icon={Workflow} storageKey="goal.scope-creep" defaultOpen={false} data-testid="goal-history">
          <p className="mb-3 text-sm text-slate-500">Tracks changes in the total dependency-closure size of this goal’s explicit targets over time.</p>
          <ScopeHistory goal={goal} />
        </DetailSection>}
        {activeTab === "decide" && (
          <section className="space-y-4" data-testid="goal-decide">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div><h2 className="text-lg font-semibold text-slate-100">Goal decisions</h2><p className="text-sm text-slate-400">Plan the goal or discover bounded work, then approve typed proposals here.</p></div>
              <div className="flex gap-2"><Button disabled={planMutation.isPending || discoverMutation.isPending} title="Starts a goal-plan workflow from the current immutable snapshot" onClick={() => planMutation.mutate()}>{planMutation.isPending ? "Planning…" : "Plan goal"}</Button><Button variant="outline" disabled={planMutation.isPending || discoverMutation.isPending} title="Starts a goal-discover workflow from the current immutable snapshot" onClick={() => discoverMutation.mutate()}>{discoverMutation.isPending ? "Discovering…" : "Discover"}</Button></div>
            </div>
            {(planMutation.error || discoverMutation.error) && <p className="text-sm text-red-300">{(planMutation.error ?? discoverMutation.error) instanceof Error ? ((planMutation.error ?? discoverMutation.error) as Error).message : "Unable to start goal workflow."}</p>}
            <ProposalSessionsPanel target={{ type: "goal", ref: goal.name, name: goal.title || goal.name }} />
          </section>
        )}
        {activeTab === "files" && (
          <FileServiceProvider value={goalFileService}>
            <EntityFileWorkspace
              files={filesQuery.data}
              isLoadingFiles={filesQuery.isLoading}
              filesError={filesQuery.error instanceof Error ? filesQuery.error : null}
              selectedFile={selectedFile}
              isLocked={goal.status !== "active"}
              onFileSelect={setSelectedFile}
              onRefetchFiles={() => void filesQuery.refetch()}
              onUploadComplete={() => void queryClient.invalidateQueries({ queryKey: ["goal", decodedName, "files"] })}
              fileActionPending={fileActionMutation.isPending}
              onFileAction={(action, target, destinationPath) => fileActionMutation.mutate({ action, target, destinationPath })}
            />
          </FileServiceProvider>
        )}
        {activeTab === "activity" && <p className="py-8 text-sm text-slate-500">Goal activity is recorded with milestone and proposal decisions.</p>}
        {activeTab === "related" && <RefList refs={goal.targets} emptyText="No related targets." entities={query.data.scopeEntities} />}
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
      <Drawer
        isOpen={Boolean(milestoneDraft)}
        onClose={() => { if (!saveMilestoneMutation.isPending) { setMilestoneEditor(null); setMilestoneDraft(null); } }}
        title={milestoneEditor ? "Edit milestone" : "Add milestone"}
        description="Milestones group goal work without changing the goal's target scope."
        footer={<div className="flex justify-end gap-3"><Button variant="outline" onClick={() => { setMilestoneEditor(null); setMilestoneDraft(null); }} disabled={saveMilestoneMutation.isPending}>Cancel</Button><Button onClick={() => milestoneDraft && saveMilestoneMutation.mutate(milestoneDraft)} disabled={saveMilestoneMutation.isPending || !milestoneDraft?.name.trim() || !milestoneDraft?.title.trim()}>{saveMilestoneMutation.isPending ? "Saving..." : "Save milestone"}</Button></div>}
      >
        {milestoneDraft && <div className="space-y-4 p-4"><div><label htmlFor="milestone-name" className="text-sm text-slate-200">Name</label><Input id="milestone-name" value={milestoneDraft.name} disabled={Boolean(milestoneEditor)} onChange={(event) => setMilestoneDraft({ ...milestoneDraft, name: event.target.value })} className="mt-2" /></div><div><label htmlFor="milestone-title" className="text-sm text-slate-200">Title</label><Input id="milestone-title" value={milestoneDraft.title} onChange={(event) => setMilestoneDraft({ ...milestoneDraft, title: event.target.value })} className="mt-2" /></div><div><label htmlFor="milestone-description" className="text-sm text-slate-200">Description</label><textarea id="milestone-description" value={milestoneDraft.description ?? ""} onChange={(event) => setMilestoneDraft({ ...milestoneDraft, description: event.target.value })} className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-sm text-slate-100" rows={5} /></div>{saveMilestoneMutation.error && <p role="alert" className="text-sm text-red-300">{saveMilestoneMutation.error instanceof Error ? saveMilestoneMutation.error.message : "Could not save milestone."}</p>}</div>}
      </Drawer>
      <Drawer isOpen={goalEditorOpen} onClose={() => !updateGoalMutation.isPending && setGoalEditorOpen(false)} title="Edit goal" description="Update the goal's outcome, description, and priority." footer={<div className="flex justify-end gap-3"><Button variant="outline" onClick={() => setGoalEditorOpen(false)} disabled={updateGoalMutation.isPending}>Cancel</Button><Button onClick={() => updateGoalMutation.mutate(goalDraft)} disabled={updateGoalMutation.isPending || !goalDraft.title.trim()}>{updateGoalMutation.isPending ? "Saving..." : "Save goal"}</Button></div>}>
        <div className="space-y-4 p-4"><div><label htmlFor="goal-title" className="text-sm text-slate-200">Title</label><Input id="goal-title" value={goalDraft.title} onChange={(event) => setGoalDraft({ ...goalDraft, title: event.target.value })} className="mt-2" /></div><div><label htmlFor="goal-description" className="text-sm text-slate-200">Description</label><textarea id="goal-description" value={goalDraft.description} onChange={(event) => setGoalDraft({ ...goalDraft, description: event.target.value })} rows={8} className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-3 py-2 text-sm text-slate-100" /></div><div><label htmlFor="goal-priority" className="text-sm text-slate-200">Priority</label><Input id="goal-priority" type="number" min={MIN_PRIORITY} max={MAX_PRIORITY} value={goalDraft.priority} onChange={(event) => setGoalDraft({ ...goalDraft, priority: Number(event.target.value) })} className="mt-2" /></div>{updateGoalMutation.error && <p role="alert" className="text-sm text-red-300">{updateGoalMutation.error instanceof Error ? updateGoalMutation.error.message : "Could not update goal."}</p>}</div>
      </Drawer>
      <Drawer isOpen={targetEditorOpen} onClose={() => setTargetEditorOpen(false)} title="Add target" description="Choose an explicit backlog item for this goal.">
        <div className="space-y-2 p-4">{backlogItems.filter((item) => !item.archivedAt && !goal.targets.includes(`${item.kind}/${item.name}`)).map((item) => <button key={`${item.kind}/${item.name}`} type="button" onClick={() => addGoalTargetMutation.mutate(`${item.kind}/${item.name}`, { onSuccess: () => setTargetEditorOpen(false) })} disabled={addGoalTargetMutation.isPending} className="flex w-full items-center justify-between rounded-lg border border-white/10 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/40 hover:bg-slate-800"><span className="truncate">{item.title || item.name}</span><span className="ml-3 text-xs text-slate-500">{item.kind}/{item.name}</span></button>)}{backlogItems.length === 0 && <p className="text-sm text-slate-500">Loading backlog items…</p>}{addGoalTargetMutation.error && <p role="alert" className="text-sm text-red-300">Unable to add target.</p>}</div>
      </Drawer>
      <Drawer isOpen={Boolean(milestoneItemEditor)} onClose={() => !updateMilestoneItemsMutation.isPending && setMilestoneItemEditor(null)} title="Manage milestone items" description="Assign goal-scope items to this milestone.">
        <div className="space-y-2 p-4">{scope.closure.map((ref) => <label key={ref} className="flex cursor-pointer items-center gap-3 rounded-lg border border-white/10 px-3 py-2 text-sm text-slate-200"><input type="checkbox" checked={milestoneItemDraft.includes(ref)} onChange={() => setMilestoneItemDraft((current) => current.includes(ref) ? current.filter((item) => item !== ref) : [...current, ref])} /><span>{ref}</span></label>)}</div>
        <div className="flex justify-end gap-3 border-t border-white/10 p-4"><Button variant="outline" onClick={() => setMilestoneItemEditor(null)} disabled={updateMilestoneItemsMutation.isPending}>Cancel</Button><Button onClick={() => milestoneItemEditor && updateMilestoneItemsMutation.mutate({ milestone: milestoneItemEditor, items: milestoneItemDraft })} disabled={updateMilestoneItemsMutation.isPending}>{updateMilestoneItemsMutation.isPending ? "Saving..." : "Save items"}</Button></div>
      </Drawer>
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

/**
 * The goal's single header action. Its label arrives from the API per goal, so
 * the icon is resolved from the shared next-action registry rather than being
 * hardcoded — otherwise this button ships as bare text.
 */
function GoalPrimaryAction({ actionId, label, pending, onRun }: { actionId: string; label: string; pending: boolean; onRun: () => void }) {
  const ActionIcon = nextActionIcon(actionId);
  return (
    <Button size="sm" onClick={onRun} disabled={pending}>
      {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <ActionIcon className="h-4 w-4" aria-hidden />}
      {pending ? "Working…" : label}
    </Button>
  );
}
