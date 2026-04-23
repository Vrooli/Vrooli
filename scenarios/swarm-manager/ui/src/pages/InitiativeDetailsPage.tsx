/**
 * Initiative Details Page
 *
 * Displays detailed information about a single initiative including:
 * - Metadata (title, description, status)
 * - Progress rollup (completed/in_progress/failed/pending)
 * - Member backlog items as clickable chips
 * - Created/updated timestamps
 * - Full file workspace (identical to backlog details)
 */

import { useMemo, useState, useRef, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { Target, Archive, ArchiveRestore, List, Network, CircleHelp, Files, Trash2, Link2, ArrowRight, CheckCircle2, Layers3, AlertTriangle, MessageCirclePlus, ClipboardCheck } from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Button } from "../components/ui/button";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { StatusBadge } from "../components/detail/StatusBadge";
import { INITIATIVE_LENSES } from "../components/detail/lens-options";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { NoteEditor } from "../components/ui/note-editor";
import { BacklogFileWorkspace } from "../components/backlog/backlog-file-workspace";
import { InitiativeDependencyGraph } from "../components/initiative/InitiativeDependencyGraph";
import { FeedbackPanel } from "../components/initiative/feedback-panel";
import { FeedbackDialog } from "../components/initiative/feedback-dialog";
import { InitiativeReviewPanel } from "../components/initiative/initiative-review-panel";
import { FileServiceProvider } from "../contexts/FileServiceContext";
import { createInitiativeFileServiceAdapter } from "../services/initiative-file-service-adapter";
import { useUrlState } from "../hooks/use-url-state";
import { useRuntimeConfig } from "../hooks/useRuntimeConfig";
import { useDetailNavigation } from "../hooks/useDetailNavigation";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { dependencyAwareSort, computeDepthMap } from "../lib/dependency-sort";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
import { findBacklogFileByPath } from "../lib/workshop-files";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { initiativeService } from "../services";
import { selectors } from "../consts/selectors";
import { RollupProgressBar, rollupTotal as computeRollupTotal } from "../components/ui/rollup-progress-bar";
import type { BacklogFile, BacklogKind, BacklogStatus, InitiativeStatus, InitiativeWithRollup } from "../types";
import { useBacklogStore, useDetailSelectionStore } from "../stores";
import { useInitiativeStore } from "../stores/initiative-store";
import type { FileActionType } from "../components/backlog/backlog-file-browser";
import { formatDisplayText } from "../lib/format-utils";
import { getStatusColorClasses } from "../surfaces/graph/lib/status-colors";
import { StatusChip } from "../components/ui/status-chip";
import { BACKLOG_STATUS_COLORS } from "../types";

type InitiativeTab = "info" | "feedback" | "review" | "files";
type ItemsView = "list" | "graph";

interface ResolvedInitiativeItem {
  ref: string;
  kind: BacklogKind;
  name: string;
  title: string;
  status: BacklogStatus;
  dependsOn: string[];
  priority: number;
  archivedAt?: string;
  missing: boolean;
}

interface InitiativeDependencyCardData {
  name: string;
  title: string;
  status: string;
  priority: number;
  rollup: InitiativeWithRollup["rollup"];
  archivedAt?: string;
  exists: boolean;
}

/** Parse "kind/name" item ref into parts. */
function parseItemRef(ref: string): { kind: string; name: string } | null {
  const slashIdx = ref.indexOf("/");
  if (slashIdx < 1) return null;
  return { kind: ref.slice(0, slashIdx), name: ref.slice(slashIdx + 1) };
}

function completionPercent(rollup: InitiativeWithRollup["rollup"] | undefined): number {
  if (!rollup) return 0;
  const total = computeRollupTotal(rollup);
  if (total === 0) return 0;
  return Math.round((rollup.completed / total) * 100);
}

function buildDependencyCardData(
  names: string[],
  allInitiatives: InitiativeWithRollup[],
): InitiativeDependencyCardData[] {
  return names.map((initiativeName) => {
    const match = allInitiatives.find((item) => item.initiative.name === initiativeName);
    if (match) {
      return {
        name: match.initiative.name,
        title: match.initiative.title || match.initiative.name,
        status: match.initiative.status,
        priority: (match.initiative as { priority?: number }).priority ?? 0,
        rollup: match.rollup,
        archivedAt: match.initiative.archivedAt,
        exists: true,
      };
    }
    return {
      name: initiativeName,
      title: initiativeName,
      status: "unknown",
      priority: 0,
      rollup: {
        total: 0,
        completed: 0,
        inProgress: 0,
        failed: 0,
        pending: 0,
        archived: 0,
      },
      exists: false,
    };
  });
}

function DependencyGroup({
  title,
  caption,
  items,
  onOpen,
}: {
  title: string;
  caption: string;
  items: InitiativeDependencyCardData[];
  onOpen: (name: string) => void;
}) {
  if (items.length === 0) return null;

  return (
    <div className="min-w-0 space-y-3">
      <div className="flex items-baseline justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-100">{title}</h3>
          <p className="text-xs text-slate-500">{caption}</p>
        </div>
        <span className="rounded-full border border-slate-700/80 bg-slate-900/70 px-2 py-0.5 text-[11px] text-slate-400">
          {items.length}
        </span>
      </div>
      <div className="grid min-w-0 gap-2">
        {items.map((item) => {
          const statusColors = getStatusColorClasses(item.status);
          const total = computeRollupTotal(item.rollup);
          const complete = completionPercent(item.rollup);
          return (
            <button
              key={item.name}
              type="button"
              onClick={() => onOpen(item.name)}
              className="min-w-0 overflow-hidden rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3 text-left transition-colors hover:border-slate-700 hover:bg-slate-800/60"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <p className="break-words text-sm font-semibold leading-snug text-slate-100">{item.title}</p>
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    <StatusChip
                      label={formatDisplayText(item.status)}
                      colors={{
                        background: statusColors.background,
                        border: statusColors.border,
                        text: statusColors.text,
                        dot: BACKLOG_STATUS_COLORS[item.status as keyof typeof BACKLOG_STATUS_COLORS] ?? "bg-slate-500",
                      }}
                      leadingDot
                      pulse={item.status === "in_review"}
                    />
                    {item.priority > 0 && (
                      <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300">
                        P{item.priority}
                      </span>
                    )}
                    {item.archivedAt && (
                      <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-300">
                        Archived
                      </span>
                    )}
                    {!item.exists && (
                      <span className="rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-[10px] font-medium text-red-300">
                        Missing
                      </span>
                    )}
                  </div>
                  <p className="mt-1 truncate text-[11px] text-slate-500">{item.name}</p>
                </div>
                <span className="shrink-0 text-[11px] text-slate-500">{complete}%</span>
              </div>
              {total > 0 ? (
                <>
                  <RollupProgressBar rollup={item.rollup} barHeight="h-1.5" className="mt-3" />
                  <div className="mt-2 flex flex-wrap gap-3 text-[11px]">
                    <span className="text-emerald-400">{item.rollup.completed} done</span>
                    <span className="text-purple-400">{item.rollup.inProgress} active</span>
                    {item.rollup.failed > 0 && <span className="text-red-400">{item.rollup.failed} failed</span>}
                    <span className="text-slate-500">{item.rollup.pending} pending</span>
                  </div>
                </>
              ) : (
                <p className="mt-3 text-[11px] text-slate-500">No item rollup available yet.</p>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}

export function InitiativeDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);

  const name = selection?.name;
  const nodeId = selectionToNodeId(selection);

  const backlogItems = useBacklogStore((s) => s.items);
  const [searchParams, setSearchParams] = useSearchParams();

  const {
    data,
    error,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: ["initiative", name],
    queryFn: async () => {
      if (!name) {
        throw new Error("Initiative name is required");
      }
      return initiativeService.get(name);
    },
    enabled: !!name,
    ...defaultQueryOptions,
  });

  const initiative = data?.initiative;
  const rollup = data?.rollup;

  // Pull all initiatives for upstream/downstream chip rendering.
  const allInitiatives = useInitiativeStore((s) => s.items);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  useEffect(() => {
    if (allInitiatives.length === 0) {
      void fetchInitiatives();
    }
  }, [allInitiatives.length, fetchInitiatives]);

  const downstreamNames = useMemo(() => {
    if (!initiative) return [];
    return allInitiatives
      .filter((other) => {
        const deps = (other.initiative as { dependsOn?: string[] }).dependsOn ?? [];
        return deps.includes(initiative.name);
      })
      .map((other) => other.initiative.name);
  }, [allInitiatives, initiative]);

  const upstreamDependencyCards = useMemo(() => {
    const upstream = (initiative as { dependsOn?: string[] } | undefined)?.dependsOn ?? [];
    return buildDependencyCardData(upstream, allInitiatives);
  }, [allInitiatives, initiative]);

  const downstreamDependencyCards = useMemo(
    () => buildDependencyCardData(downstreamNames, allInitiatives),
    [allInitiatives, downstreamNames],
  );

  const queryClient = useQueryClient();
  const archiveMutation = useMutation({
    mutationFn: async () => {
      if (!name) throw new Error("Initiative name is required");
      return defaultApiClient.patch<unknown>(API_ENDPOINTS.initiativeArchiveItem(name), {});
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["initiative", name] });
      void queryClient.invalidateQueries({ queryKey: ["initiatives"] });
    },
  });

  const unarchiveMutation = useMutation({
    mutationFn: async () => {
      if (!name) throw new Error("Initiative name is required");
      return defaultApiClient.delete<unknown>(API_ENDPOINTS.initiativeArchiveItem(name));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["initiative", name] });
      void queryClient.invalidateQueries({ queryKey: ["initiatives"] });
    },
  });

  const detailNav = useDetailNavigation();
  const { closeDetail } = detailNav;
  const { getDeleteConfirmLevel } = useRuntimeConfig();
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!name) throw new Error("Initiative name is required");
      return defaultApiClient.delete<unknown>(API_ENDPOINTS.initiativeByName(name));
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["initiatives"] });
      closeDetail();
    },
  });

  const handleDeleteClick = useCallback(() => {
    if (getDeleteConfirmLevel("initiative") === "none") {
      deleteMutation.mutate();
    } else {
      setShowDeleteDialog(true);
    }
  }, [getDeleteConfirmLevel, deleteMutation]);

  const isArchived = initiative?.archivedAt != null;
  const isArchiveActionPending = archiveMutation.isPending || unarchiveMutation.isPending;
  const archiveActionError = archiveMutation.isError
    ? archiveMutation.error instanceof Error ? archiveMutation.error.message : "Failed to archive initiative."
    : unarchiveMutation.isError
      ? unarchiveMutation.error instanceof Error ? unarchiveMutation.error.message : "Failed to unarchive initiative."
      : null;

  // --- Tab state ---
  const [activeTab, setActiveTab] = useUrlState<InitiativeTab>("tab", "info", {
    validate: (v): v is InitiativeTab => ["info", "feedback", "review", "files"].includes(v),
  });

  // --- Feedback dialog (header button entry point) ---
  const [feedbackDialogOpen, setFeedbackDialogOpen] = useState(false);

  // --- File service ---
  const fileService = useMemo(
    () => name ? createInitiativeFileServiceAdapter(name) : null,
    [name],
  );

  // --- Files query ---
  const {
    data: files,
    isLoading: isLoadingFiles,
    error: filesError,
    refetch: refetchFiles,
  } = useQuery({
    queryKey: ["initiative", name, "files"],
    queryFn: () => {
      if (!name) throw new Error("Initiative name is required");
      if (!fileService) throw new Error("File service is not available");
      return fileService.getFiles();
    },
    enabled: !!name && !!fileService,
    ...defaultQueryOptions,
  });

  // --- File selection state ---
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);

  // Sync selected file from URL param
  const selectedFileParam = searchParams.get("file");
  useEffect(() => {
    if (!files || files.length === 0) return;
    const requestedPath = selectedFileParam;
    if (requestedPath) {
      const resolvedFile = findBacklogFileByPath(files, requestedPath);
      if (resolvedFile) {
        setSelectedFile((prev) => (prev?.path === resolvedFile.path ? prev : resolvedFile));
        return;
      }
    }
    // No file param or file not found — clear selection
    if (selectedFileParam) {
      setSelectedFile(null);
    }
  }, [files, selectedFileParam]);

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "directory") return;
    setSelectedFile(file);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("file", file.path);
      return next;
    }, { replace: true });
  }, [setSearchParams]);

  const handleUploadComplete = useCallback(() => {
    void refetchFiles();
  }, [refetchFiles]);

  const [isFileActionPending, setIsFileActionPending] = useState(false);
  const handleFileAction = useCallback(
    async (action: FileActionType, target: BacklogFile, destinationPath?: string) => {
      if (!fileService) return;
      setIsFileActionPending(true);
      try {
        switch (action) {
          case "rename":
            if (destinationPath) await fileService.renameFile(target.path, destinationPath);
            break;
          case "move":
            if (destinationPath) await fileService.moveFile(target.path, destinationPath);
            break;
          case "copy":
            if (destinationPath) await fileService.copyFile(target.path, destinationPath);
            break;
          case "delete":
            await fileService.deleteFile(target.path);
            if (selectedFile?.path === target.path) {
              setSelectedFile(null);
            }
            break;
        }
        void refetchFiles();
      } finally {
        setIsFileActionPending(false);
      }
    },
    [fileService, refetchFiles, selectedFile?.path],
  );

  const mobileActions = initiative ? (
    <div className="flex flex-col gap-2 p-4">
      <Button
        variant="outline"
        size="sm"
        onClick={() => setFeedbackDialogOpen(true)}
        data-testid={selectors.initiativeDetails.addFeedbackButton}
      >
        <MessageCirclePlus className="mr-1.5 h-4 w-4" />
        Add Feedback
      </Button>
      {isArchived ? (
        <Button
          variant="outline"
          size="sm"
          onClick={() => unarchiveMutation.mutate()}
          disabled={isArchiveActionPending}
        >
          <ArchiveRestore className="mr-1.5 h-4 w-4" />
          {unarchiveMutation.isPending ? "Restoring..." : "Unarchive"}
        </Button>
      ) : (
        <Button
          variant="outline"
          size="sm"
          onClick={() => archiveMutation.mutate()}
          disabled={isArchiveActionPending}
        >
          <Archive className="mr-1.5 h-4 w-4" />
          {archiveMutation.isPending ? "Archiving..." : "Archive"}
        </Button>
      )}
      <Button
        variant="destructive"
        size="sm"
        onClick={handleDeleteClick}
        disabled={deleteMutation.isPending}
      >
        <Trash2 className="mr-1.5 h-4 w-4" />
        {deleteMutation.isPending ? "Deleting..." : "Delete"}
      </Button>
    </div>
  ) : undefined;

  // Resolve member items against the backlog store
  const resolvedItems = useMemo<ResolvedInitiativeItem[]>(() => {
    if (!initiative?.items) return [];
    return initiative.items.map((ref) => {
      const parsed = parseItemRef(ref);
      if (!parsed) return {
        ref,
        kind: "" as BacklogKind,
        name: ref,
        title: ref,
        status: "backlog" as BacklogStatus,
        dependsOn: [],
        priority: 0,
        missing: true,
      };
      const found = backlogItems.find((bi) => bi.kind === parsed.kind && bi.name === parsed.name);
      return {
        ref,
        kind: parsed.kind as BacklogKind,
        name: parsed.name,
        title: found?.title ?? `${parsed.kind}/${parsed.name}`,
        status: (found?.status ?? "backlog"),
        dependsOn: found?.dependsOn ?? [],
        priority: found?.priority ?? 0,
        archivedAt: found?.archivedAt,
        missing: found == null,
      };
    });
  }, [initiative?.items, backlogItems]);

  // Topologically sorted items + dependency metadata
  const sortedItems = useMemo(
    () => dependencyAwareSort(resolvedItems, (a, b) => a.title.localeCompare(b.title), backlogItems),
    [resolvedItems, backlogItems],
  );

  const depthMap = useMemo(() => computeDepthMap(backlogItems), [backlogItems]);

  const depRelationsMap = useMemo(() => {
    const initiativeKeys = new Set(resolvedItems.map((i) => `${i.kind}/${i.name}`));
    const map = new Map<string, { parentCount: number; childCount: number }>();
    for (const item of resolvedItems) {
      const relations = computeDependencyRelations(
        { kind: item.kind, name: item.name, dependsOn: item.dependsOn },
        backlogItems,
      );
      map.set(`${item.kind}/${item.name}`, {
        parentCount: relations.parents.filter((p) => initiativeKeys.has(`${p.kind}/${p.name}`)).length,
        childCount: relations.children.filter((c) => initiativeKeys.has(`${c.kind}/${c.name}`)).length,
      });
    }
    return map;
  }, [resolvedItems, backlogItems]);

  // Items view mode toggle
  const [itemsView, setItemsView] = useUrlState<ItemsView>("items", "graph", {
    validate: (v): v is ItemsView => ["list", "graph"].includes(v),
  });

  // Collapsible description
  const [descExpanded, setDescExpanded] = useState(false);
  const descRef = useRef<HTMLParagraphElement>(null);
  const [descOverflows, setDescOverflows] = useState(false);

  useEffect(() => {
    if (descRef.current) {
      setDescOverflows(descRef.current.scrollHeight > descRef.current.clientHeight);
    }
  }, [initiative?.description]);

  // Rollup total for progress bar
  const rollupTotalCount = rollup ? computeRollupTotal(rollup) : 0;
  const completion = rollup ? completionPercent(rollup) : 0;
  const priority = (initiative as { priority?: number } | undefined)?.priority ?? 0;
  const dependencyCount = upstreamDependencyCards.length + downstreamDependencyCards.length;
  const missingItemCount = resolvedItems.filter((item) => item.missing).length;
  const archivedItemCount = resolvedItems.filter((item) => item.archivedAt != null).length;

  if (isLoading) {
    return <PageLoadingState label="Loading initiative..." />;
  }

  if (error || !initiative) {
    return (
      <DetailPageLayout
        header={
          <DetailPageHeader
            entityType="initiative"
            entityIcon={Target}
            title={name ?? "Unknown"}
            nodeId={null}
            lenses={[]}
          />
        }
      >
        <div className="md:mx-auto md:max-w-3xl">
          <ErrorState
            error={error as Error | undefined}
            title="Failed to load initiative"
            message={`Could not load initiative "${name}".`}
            onRetry={() => refetch()}
          />
        </div>
      </DetailPageLayout>
    );
  }

  const tabBar = (
    <div className="border-t border-slate-800/50" data-testid={selectors.initiativeDetails.tabRow}>
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as InitiativeTab)}>
        <TabsList className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3">
          <TabsTrigger value="info" className="gap-2" data-testid={selectors.initiativeDetails.tabInfo}>
            <CircleHelp className="h-4 w-4" />
            Info
          </TabsTrigger>
          <TabsTrigger value="feedback" className="gap-2" data-testid={selectors.initiativeDetails.tabFeedback}>
            <MessageCirclePlus className="h-4 w-4" />
            Feedback
          </TabsTrigger>
          <TabsTrigger value="review" className="gap-2" data-testid={selectors.initiativeDetails.tabReview}>
            <ClipboardCheck className="h-4 w-4" />
            Review
          </TabsTrigger>
          <TabsTrigger value="files" className="gap-2" data-testid={selectors.initiativeDetails.tabFiles}>
            <Files className="h-4 w-4" />
            Files
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="initiative"
          entityIcon={Target}
          title={initiative.title || initiative.name}
          status={initiative.status}
          nodeId={nodeId}
          lenses={INITIATIVE_LENSES}
          tabBar={tabBar}
          actions={
            <Button
              variant="outline"
              size="sm"
              onClick={() => setFeedbackDialogOpen(true)}
              data-testid={selectors.initiativeDetails.addFeedbackButton}
            >
              <MessageCirclePlus className="mr-1.5 h-4 w-4" />
              Add Feedback
            </Button>
          }
        />
      }
      mobileActions={mobileActions}
      mobileActionsTitle="Initiative Actions"
    >
      <div className="space-y-0 md:mx-auto md:max-w-3xl" data-testid={selectors.initiativeDetails.page}>
        {isArchived && (
          <div className="mb-3 flex items-center justify-between rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
            <div className="flex items-center gap-2 text-sm text-amber-300">
              <Archive className="h-4 w-4" />
              <span>Archived {formatRelativeTime(initiative.archivedAt ?? "")}</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              className="border-amber-500/30 text-amber-300 hover:bg-amber-500/10"
              onClick={() => unarchiveMutation.mutate()}
              disabled={isArchiveActionPending}
            >
              <ArchiveRestore className="mr-2 h-4 w-4" />
              {unarchiveMutation.isPending ? "Restoring..." : "Unarchive"}
            </Button>
          </div>
        )}
        {archiveActionError && (
          <div className="mb-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {archiveActionError}
          </div>
        )}
        {activeTab === "info" && (
          <>
            {/* Overview section */}
            <DetailSection
              title="Overview"
              icon={Target}
              hideDivider
            >
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-2">
                  <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                    <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      Progress
                    </div>
                    <div className="mt-2 flex items-end gap-2">
                      <span className="text-xl font-semibold text-slate-100 sm:text-2xl">{completion}%</span>
                      <StatusBadge
                        status={initiative.status}
                        className="translate-y-[-1px]"
                        data-testid={selectors.initiativeDetails.status}
                      />
                    </div>
                    <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                      {rollupTotalCount > 0 ? `${rollup?.completed ?? 0} of ${rollupTotalCount} items complete` : "No tracked items yet"}
                    </p>
                  </div>
                  <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                    <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                      <Layers3 className="h-3.5 w-3.5" />
                      Scope
                    </div>
                    <div className="mt-2 text-xl font-semibold text-slate-100 sm:text-2xl">{resolvedItems.length}</div>
                    <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                      backlog {resolvedItems.length === 1 ? "item" : "items"}
                      {archivedItemCount > 0 ? ` • ${archivedItemCount} archived` : ""}
                    </p>
                  </div>
                  <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                    <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                      <Link2 className="h-3.5 w-3.5" />
                      Dependencies
                    </div>
                    <div className="mt-2 text-xl font-semibold text-slate-100 sm:text-2xl">{dependencyCount}</div>
                    <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                      {upstreamDependencyCards.length} upstream • {downstreamDependencyCards.length} downstream
                    </p>
                  </div>
                  <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                    <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
                      <AlertTriangle className="h-3.5 w-3.5" />
                      Priority
                    </div>
                    <div className="mt-2 text-xl font-semibold text-slate-100 sm:text-2xl">{priority > 0 ? `P${priority}` : "Unset"}</div>
                    <p className="mt-1 text-[11px] text-slate-500 sm:text-xs">
                      {missingItemCount > 0 ? `${missingItemCount} unresolved item ${missingItemCount === 1 ? "ref" : "refs"}` : "All item refs resolved"}
                    </p>
                  </div>
                </div>
                {initiative.description && (
                  <div data-testid={selectors.initiativeDetails.description}>
                    <p
                      ref={descRef}
                      className={`text-sm text-slate-300 leading-relaxed ${descExpanded ? "" : "line-clamp-4"}`}
                    >
                      {initiative.description}
                    </p>
                    {(descOverflows || descExpanded) && (
                      <button
                        type="button"
                        onClick={() => setDescExpanded(!descExpanded)}
                        className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                      >
                        {descExpanded ? "Show less" : "Show more\u2026"}
                      </button>
                    )}
                  </div>
                )}

                <NoteEditor
                  note={initiative.note ?? ""}
                  onSave={async (note) => {
                    await initiativeService.updateNote(initiative.name, note);
                    void refetch();
                  }}
                />

                <div className="flex gap-6 text-xs text-slate-500">
                  <div>
                    <span className="uppercase tracking-wider">Created</span>{" "}
                    <span className="text-slate-400">{formatRelativeTime(initiative.created)}</span>
                  </div>
                  <div>
                    <span className="uppercase tracking-wider">Updated</span>{" "}
                    <span className="text-slate-400">{formatRelativeTime(initiative.updated)}</span>
                  </div>
                </div>
              </div>
            </DetailSection>

            {/* Progress Rollup */}
            {rollup && rollupTotalCount > 0 && (
              <DetailSection title="Progress" data-testid={selectors.initiativeDetails.rollup}>
                <RollupProgressBar rollup={rollup} showLabels />
              </DetailSection>
            )}

            {/* Initiative-level dependency ordering */}
            {(() => {
              if (priority === 0 && upstreamDependencyCards.length === 0 && downstreamDependencyCards.length === 0) return null;
              return (
                <DetailSection title="Dependencies" icon={Link2}>
                  <div className="space-y-5">
                    <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
                      {priority > 0 && (
                        <span className="rounded-full border border-slate-700/80 bg-slate-900/80 px-2.5 py-1 text-slate-300">
                          Priority P{priority}
                        </span>
                      )}
                      <span className="rounded-full border border-slate-700/80 bg-slate-900/80 px-2.5 py-1">
                        {upstreamDependencyCards.length} blocked-by
                      </span>
                      <span className="rounded-full border border-slate-700/80 bg-slate-900/80 px-2.5 py-1">
                        {downstreamDependencyCards.length} unlocked
                      </span>
                    </div>
                    <div className="grid min-w-0 gap-5 xl:grid-cols-2">
                      <DependencyGroup
                        title="Blocked By"
                        caption="Upstream initiatives that should land first."
                        items={upstreamDependencyCards}
                        onOpen={(dep) => detailNav.openDetail({ entityType: "initiative", name: dep })}
                      />
                      <DependencyGroup
                        title="Unblocks"
                        caption="Downstream initiatives waiting on this one."
                        items={downstreamDependencyCards}
                        onOpen={(dep) => detailNav.openDetail({ entityType: "initiative", name: dep })}
                      />
                    </div>
                  </div>
                </DetailSection>
              );
            })()}

            {/* Member Items */}
            {resolvedItems.length > 0 && (
              <DetailSection
                title={`Items (${resolvedItems.length})`}
                data-testid={selectors.initiativeDetails.itemsList}
                action={
                  <div className="flex gap-0.5" data-testid={selectors.initiativeDetails.itemsViewToggle}>
                    <button
                      type="button"
                      onClick={() => setItemsView("list")}
                      className={`rounded p-1 transition-colors ${itemsView === "list" ? "text-slate-200 bg-slate-700/50" : "text-slate-500 hover:text-slate-300"}`}
                      title="List view"
                      aria-pressed={itemsView === "list"}
                    >
                      <List className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => setItemsView("graph")}
                      className={`rounded p-1 transition-colors ${itemsView === "graph" ? "text-slate-200 bg-slate-700/50" : "text-slate-500 hover:text-slate-300"}`}
                      title="Dependency graph"
                      aria-pressed={itemsView === "graph"}
                    >
                      <Network className="h-4 w-4" />
                    </button>
                  </div>
                }
              >
                {itemsView === "list" ? (
                  <div className="flex flex-col gap-2" data-testid={selectors.initiativeDetails.itemsListView}>
                    {sortedItems.map((item) => {
                      const key = `${item.kind}/${item.name}`;
                      const depth = depthMap.get(key) ?? 0;
                      const relations = depRelationsMap.get(key);
                      const stateLabel = formatDisplayText(item.status);
                      const statusColors = getStatusColorClasses(item.status);
                      return (
                        <div
                          key={item.ref}
                          className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3"
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0 space-y-2">
                              <div className="flex items-start justify-between gap-3">
                                <button
                                  type="button"
                                  onClick={() => selectBacklog(item.kind, item.name)}
                                  className="min-w-0 text-left text-base font-semibold leading-snug text-slate-100 transition-colors hover:text-cyan-300"
                                >
                                  {item.title}
                                </button>
                                <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-slate-600" />
                              </div>
                              <div className="flex flex-wrap items-center gap-2">
                                <StatusChip
                                  label={stateLabel}
                                  colors={{
                                    background: statusColors.background,
                                    border: statusColors.border,
                                    text: statusColors.text,
                                    dot: BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500",
                                  }}
                                  leadingDot
                                  pulse={item.status === "in_review"}
                                />
                                {item.priority > 0 && (
                                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-300">
                                    P{item.priority}
                                  </span>
                                )}
                                {depth > 0 && (
                                  <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[10px] font-medium text-slate-400">
                                    Layer {depth}
                                  </span>
                                )}
                                {item.archivedAt && (
                                  <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-300">
                                    Archived
                                  </span>
                                )}
                                {item.missing && (
                                  <span className="rounded-full border border-red-500/30 bg-red-500/10 px-2 py-0.5 text-[10px] font-medium text-red-300">
                                    Missing from backlog
                                  </span>
                                )}
                              </div>
                              <div className="flex flex-wrap gap-3 text-[11px] text-slate-500">
                                <span>{item.kind}/{item.name}</span>
                                {relations && relations.parentCount > 0 && (
                                  <span className="text-amber-400/80">blocked by {relations.parentCount}</span>
                                )}
                                {relations && relations.childCount > 0 && (
                                  <span>unblocks {relations.childCount}</span>
                                )}
                                {relations && relations.parentCount === 0 && relations.childCount === 0 && (
                                  <span>no in-initiative dependencies</span>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="space-y-3" data-testid={selectors.initiativeDetails.itemsGraphView}>
                    <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
                      <div className="flex flex-wrap items-center gap-3 text-[11px]">
                        <span className="text-slate-400">Default view: dependency graph for faster sequencing and blocker scans.</span>
                        <span className="text-emerald-400">Done</span>
                        <span className="text-purple-400">Active</span>
                        <span className="text-red-400">Failed</span>
                        <span className="text-slate-500">Pending</span>
                      </div>
                    </div>
                    <InitiativeDependencyGraph items={resolvedItems} />
                  </div>
                )}
              </DetailSection>
            )}
          </>
        )}

        {activeTab === "feedback" && (
          <FeedbackPanel
            initiativeName={initiative.name}
            previewItems={resolvedItems.map((item) => ({
              kind: item.kind,
              name: item.name,
              title: item.title,
              status: item.status,
              dependsOn: item.dependsOn,
              priority: item.priority,
              archivedAt: item.archivedAt,
              missing: item.missing,
            }))}
          />
        )}

        {activeTab === "review" && (
          <InitiativeReviewPanel
            initiativeName={initiative.name}
            initiativeStatus={initiative.status as InitiativeStatus}
            onDecided={() => void refetch()}
          />
        )}

        {activeTab === "files" && fileService && (
          <FileServiceProvider value={fileService}>
            <BacklogFileWorkspace
              files={files}
              isLoadingFiles={isLoadingFiles}
              filesError={filesError ?? null}
              selectedFile={selectedFile}
              isLocked={false}
              onFileSelect={handleFileSelect}
              onRefetchFiles={() => void refetchFiles()}
              onUploadComplete={handleUploadComplete}
              fileActionPending={isFileActionPending}
              onFileAction={handleFileAction}
            />
          </FileServiceProvider>
        )}
      </div>

      <FeedbackDialog
        initiativeName={initiative.name}
        isOpen={feedbackDialogOpen}
        onClose={() => setFeedbackDialogOpen(false)}
        onSubmitted={() => {
          setActiveTab("feedback");
          void queryClient.invalidateQueries({ queryKey: ["initiative-feedback", initiative.name] });
        }}
      />

      {/* Delete confirmation dialog */}
      {(() => {
        const deleteLevel = getDeleteConfirmLevel("initiative");
        return deleteLevel !== "none" ? (
          <ConfirmDialog
            isOpen={showDeleteDialog}
            onClose={() => setShowDeleteDialog(false)}
            onConfirm={() => { setShowDeleteDialog(false); deleteMutation.mutate(); }}
            title="Delete Initiative"
            description={`Are you sure you want to delete "${initiative.title || initiative.name}"? This will remove the initiative and its metadata permanently.`}
            confirmationText={deleteLevel === "strong" ? initiative.name : undefined}
            confirmLabel="Delete Initiative"
            isLoading={deleteMutation.isPending}
          />
        ) : null;
      })()}
    </DetailPageLayout>
  );
}
