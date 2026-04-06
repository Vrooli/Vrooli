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
import { Target, Archive, List, Network, CircleHelp, Files } from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Button } from "../components/ui/button";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { StatusBadge } from "../components/detail/StatusBadge";
import { INITIATIVE_LENSES } from "../components/detail/lens-options";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { EntityLink } from "../components/ui/entity-link";
import { NoteEditor } from "../components/ui/note-editor";
import { BacklogFileWorkspace } from "../components/backlog/backlog-file-workspace";
import { InitiativeDependencyGraph } from "../components/initiative/InitiativeDependencyGraph";
import { FileServiceProvider } from "../contexts/FileServiceContext";
import { createInitiativeFileServiceAdapter } from "../services/initiative-file-service-adapter";
import { useUrlState } from "../hooks/use-url-state";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { dependencyAwareSort, computeDepthMap } from "../lib/dependency-sort";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
import { findBacklogFileByPath } from "../lib/workshop-files";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { initiativeService } from "../services";
import { selectors } from "../consts/selectors";
import { BACKLOG_STATUS_CHIP_COLORS } from "../types";
import type { BacklogFile, BacklogKind, BacklogStatus } from "../types";
import { useBacklogStore, useDetailSelectionStore } from "../stores";
import type { FileActionType } from "../components/backlog/backlog-file-browser";

type InitiativeTab = "info" | "files";

/** Parse "kind/name" item ref into parts. */
function parseItemRef(ref: string): { kind: string; name: string } | null {
  const slashIdx = ref.indexOf("/");
  if (slashIdx < 1) return null;
  return { kind: ref.slice(0, slashIdx), name: ref.slice(slashIdx + 1) };
}

export function InitiativeDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);

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

  const queryClient = useQueryClient();
  const archiveMutation = useMutation({
    mutationFn: async () => {
      if (!name) throw new Error("Initiative name is required");
      await defaultApiClient.put(API_ENDPOINTS.initiativeByName(name), { status: "archived" });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["initiative", name] });
    },
  });

  const isArchived = initiative?.status === "archived";

  // --- Tab state ---
  const [activeTab, setActiveTab] = useUrlState<InitiativeTab>("tab", "info", {
    validate: (v): v is InitiativeTab => ["info", "files"].includes(v),
  });

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
        onClick={() => archiveMutation.mutate()}
        disabled={isArchived || archiveMutation.isPending}
      >
        <Archive className="mr-1.5 h-4 w-4" />
        {archiveMutation.isPending ? "Archiving..." : "Archive"}
      </Button>
    </div>
  ) : undefined;

  // Resolve member items against the backlog store
  const resolvedItems = useMemo(() => {
    if (!initiative?.items) return [];
    return initiative.items.map((ref) => {
      const parsed = parseItemRef(ref);
      if (!parsed) return { ref, kind: "" as BacklogKind, name: ref, title: ref, status: "backlog" as BacklogStatus, dependsOn: [] as string[] };
      const found = backlogItems.find((bi) => bi.kind === parsed.kind && bi.name === parsed.name);
      return {
        ref,
        kind: parsed.kind as BacklogKind,
        name: parsed.name,
        title: found?.title ?? `${parsed.kind}/${parsed.name}`,
        status: (found?.status ?? "archived") as BacklogStatus,
        dependsOn: found?.dependsOn ?? [],
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
        { kind: item.kind as BacklogKind, name: item.name, dependsOn: item.dependsOn },
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
  const [itemsView, setItemsView] = useState<"list" | "graph">("list");

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
  const rollupTotal = rollup ? rollup.completed + rollup.inProgress + rollup.failed + rollup.pending : 0;

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
        />
      }
      mobileActions={mobileActions}
      mobileActionsTitle="Initiative Actions"
    >
      <div className="space-y-0 md:mx-auto md:max-w-3xl" data-testid={selectors.initiativeDetails.page}>
        {activeTab === "info" && (
          <>
            {/* Overview section */}
            <DetailSection
              title="Overview"
              icon={Target}
              hideDivider
              action={
                <StatusBadge
                  status={initiative.status}
                  data-testid={selectors.initiativeDetails.status}
                />
              }
            >
              <div className="space-y-3">
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
            {rollup && rollupTotal > 0 && (
              <DetailSection title="Progress" data-testid={selectors.initiativeDetails.rollup}>
                <div className="space-y-3">
                  {/* Segmented progress bar */}
                  <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-slate-800">
                    {rollup.completed > 0 && (
                      <div
                        className="bg-emerald-500 transition-all"
                        style={{ width: `${(rollup.completed / rollupTotal) * 100}%` }}
                        title={`${rollup.completed} completed`}
                      />
                    )}
                    {rollup.inProgress > 0 && (
                      <div
                        className="bg-purple-500 transition-all"
                        style={{ width: `${(rollup.inProgress / rollupTotal) * 100}%` }}
                        title={`${rollup.inProgress} in progress`}
                      />
                    )}
                    {rollup.failed > 0 && (
                      <div
                        className="bg-red-500 transition-all"
                        style={{ width: `${(rollup.failed / rollupTotal) * 100}%` }}
                        title={`${rollup.failed} failed`}
                      />
                    )}
                    {rollup.pending > 0 && (
                      <div
                        className="bg-slate-600 transition-all"
                        style={{ width: `${(rollup.pending / rollupTotal) * 100}%` }}
                        title={`${rollup.pending} pending`}
                      />
                    )}
                  </div>

                  {/* Numeric breakdown */}
                  <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs">
                    <span className="text-emerald-400">{rollup.completed} completed</span>
                    <span className="text-purple-400">{rollup.inProgress} in progress</span>
                    {rollup.failed > 0 && <span className="text-red-400">{rollup.failed} failed</span>}
                    <span className="text-slate-400">{rollup.pending} pending</span>
                    <span className="text-slate-500">{rollupTotal} total</span>
                  </div>
                </div>
              </DetailSection>
            )}

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
                    >
                      <List className="h-4 w-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => setItemsView("graph")}
                      className={`rounded p-1 transition-colors ${itemsView === "graph" ? "text-slate-200 bg-slate-700/50" : "text-slate-500 hover:text-slate-300"}`}
                      title="Dependency graph"
                    >
                      <Network className="h-4 w-4" />
                    </button>
                  </div>
                }
              >
                {itemsView === "list" ? (
                  <div className="flex flex-col gap-1.5" data-testid={selectors.initiativeDetails.itemsListView}>
                    {sortedItems.map((item) => {
                      const key = `${item.kind}/${item.name}`;
                      const chipColors = BACKLOG_STATUS_CHIP_COLORS[item.status] ?? "bg-slate-600/20 text-slate-300";
                      const depth = depthMap.get(key) ?? 0;
                      const relations = depRelationsMap.get(key);
                      return (
                        <div key={item.ref} className="flex items-center gap-2">
                          {depth > 0 && (
                            <span className="shrink-0 text-[10px] font-mono text-slate-500 w-4 text-right">L{depth}</span>
                          )}
                          <EntityLink
                            entityType="backlog"
                            kind={item.kind}
                            name={item.name}
                            label={item.title}
                            className={`hover:brightness-125 ${chipColors}`}
                          />
                          {relations && relations.parentCount > 0 && (
                            <span className="shrink-0 text-[10px] text-amber-500/70">
                              &larr; blocked by {relations.parentCount}
                            </span>
                          )}
                          {relations && relations.childCount > 0 && (
                            <span className="shrink-0 text-[10px] text-slate-500">
                              &rarr; unblocks {relations.childCount}
                            </span>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div data-testid={selectors.initiativeDetails.itemsGraphView}>
                    <InitiativeDependencyGraph items={resolvedItems} />
                  </div>
                )}
              </DetailSection>
            )}
          </>
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
    </DetailPageLayout>
  );
}
