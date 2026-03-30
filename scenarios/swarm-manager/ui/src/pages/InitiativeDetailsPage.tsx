/**
 * Initiative Details Page
 *
 * Displays detailed information about a single initiative including:
 * - Metadata (title, description, status)
 * - Progress rollup (completed/in_progress/failed/pending)
 * - Member backlog items as clickable chips
 * - Created/updated timestamps
 */

import { useMemo, useState, useRef, useEffect, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { Target, FolderOpen } from "lucide-react";
import { Card } from "../components/ui/card";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailActionButtons } from "../components/detail/DetailActionButtons";
import { StatusBadge } from "../components/detail/StatusBadge";
import { useDrillToLens } from "../hooks/useDrillToLens";
import { LensBar, INITIATIVE_LENSES } from "../components/detail/LensBar";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { FileTree, type TreeFile } from "../components/ui/file-tree";
import { defaultQueryOptions, formatRelativeTime, getFileExtension } from "../lib";
import { renderMarkdown } from "../lib/render-markdown";
import { initiativeService } from "../services";
import { selectors } from "../consts/selectors";
import { BACKLOG_STATUS_CHIP_COLORS } from "../types";
import type { BacklogStatus } from "../types";
import { useBacklogStore, useDetailSelectionStore } from "../stores";

/** Parse "kind/name" item ref into parts. */
function parseItemRef(ref: string): { kind: string; name: string } | null {
  const slashIdx = ref.indexOf("/");
  if (slashIdx < 1) return null;
  return { kind: ref.slice(0, slashIdx), name: ref.slice(slashIdx + 1) };
}

export function InitiativeDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);
  const clearSelection = useDetailSelectionStore((s) => s.clearSelection);
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const name = selection?.name;
  const nodeId = selectionToNodeId(selection);
  const { drillToLens } = useDrillToLens();

  const backlogItems = useBacklogStore((s) => s.items);

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

  // Resolve member items against the backlog store
  const resolvedItems = useMemo(() => {
    if (!initiative?.items) return [];
    return initiative.items.map((ref) => {
      const parsed = parseItemRef(ref);
      if (!parsed) return { ref, kind: "", name: ref, title: ref, status: "backlog" as BacklogStatus };
      const found = backlogItems.find((bi) => bi.kind === parsed.kind && bi.name === parsed.name);
      return {
        ref,
        kind: parsed.kind,
        name: parsed.name,
        title: found?.title ?? `${parsed.kind}/${parsed.name}`,
        status: (found?.status ?? "archived") as BacklogStatus,
      };
    });
  }, [initiative?.items, backlogItems]);

  // Collapsible description
  const [descExpanded, setDescExpanded] = useState(false);
  const descRef = useRef<HTMLParagraphElement>(null);
  const [descOverflows, setDescOverflows] = useState(false);

  useEffect(() => {
    if (descRef.current) {
      setDescOverflows(descRef.current.scrollHeight > descRef.current.clientHeight);
    }
  }, [initiative?.description]);

  // Files query
  const {
    data: files,
    isLoading: isLoadingFiles,
  } = useQuery({
    queryKey: ["initiative", name, "files"],
    queryFn: () => {
      if (!name) throw new Error("Initiative name is required");
      return initiativeService.listFiles(name);
    },
    enabled: !!name,
    ...defaultQueryOptions,
  });

  // File selection state
  const [selectedFile, setSelectedFile] = useState<TreeFile | null>(null);
  const [fileContent, setFileContent] = useState<string | null>(null);
  const [isLoadingContent, setIsLoadingContent] = useState(false);

  const handleFileSelect = useCallback(
    async (file: TreeFile) => {
      if (!name || file.type === "directory") return;
      setSelectedFile(file);
      setIsLoadingContent(true);
      try {
        const content = await initiativeService.getFileContent(name, file.path);
        setFileContent(content);
      } catch {
        setFileContent(null);
      } finally {
        setIsLoadingContent(false);
      }
    },
    [name],
  );

  // Filter out initiative.json from the file tree for display
  const displayFiles = useMemo(() => {
    if (!files) return [];
    return files.filter((f) => f.name !== "initiative.json");
  }, [files]);

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
            title={name ?? "Unknown"}
            onClose={clearSelection}
          />
        }
      >
        <div className="mx-auto max-w-3xl">
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

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="initiative"
          title={initiative.title || initiative.name}
          status={initiative.status}
          onClose={clearSelection}
          actions={<DetailActionButtons entityType="initiative" />}
        />
      }
    >
      <div className="mx-auto max-w-3xl space-y-6" data-testid={selectors.initiativeDetails.page}>
      {nodeId && <LensBar nodeId={nodeId} lenses={INITIATIVE_LENSES} onDrillToLens={drillToLens} />}
      {/* Metadata Card */}
      <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5">
        <div className="space-y-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-2.5">
              <Target className="h-5 w-5 text-sky-400 shrink-0" />
              <h1
                className="text-xl font-semibold text-slate-100"
                data-testid={selectors.initiativeDetails.title}
              >
                {initiative.title || initiative.name}
              </h1>
            </div>
            <StatusBadge
              status={initiative.status}
              data-testid={selectors.initiativeDetails.status}
            />
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
      </Card>

      {/* Progress Rollup Card */}
      {rollup && rollupTotal > 0 && (
        <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5" data-testid={selectors.initiativeDetails.rollup}>
          <div className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
              Progress
            </h2>

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
        </Card>
      )}

      {/* Member Items Card */}
      {resolvedItems.length > 0 && (
        <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5" data-testid={selectors.initiativeDetails.itemsList}>
          <div className="space-y-3">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
              Items ({resolvedItems.length})
            </h2>
            <div className="flex flex-wrap gap-1.5">
              {resolvedItems.map((item) => {
                const chipColors = BACKLOG_STATUS_CHIP_COLORS[item.status] ?? "bg-slate-600/20 text-slate-300";
                return (
                  <button
                    key={item.ref}
                    type="button"
                    onClick={() => selectBacklog(item.kind, item.name)}
                    className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium transition-colors hover:brightness-125 ${chipColors}`}
                  >
                    {item.title}
                  </button>
                );
              })}
            </div>
          </div>
        </Card>
      )}

      {/* Files Card */}
      {!isLoadingFiles && displayFiles.length > 0 && (
        <Card className="rounded-lg border-slate-700/60 bg-slate-900/45 p-5" data-testid="initiative-files">
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <FolderOpen className="h-4 w-4 text-slate-400" />
              <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
                Files ({displayFiles.length})
              </h2>
            </div>

            <div className={selectedFile ? "grid grid-cols-1 gap-4 lg:grid-cols-2" : ""}>
              <FileTree
                files={displayFiles}
                onFileSelect={handleFileSelect}
                selectedPath={selectedFile?.path}
                className="rounded-lg border border-slate-700/40 bg-slate-800/30 p-2"
              />

              {selectedFile && (
                <div className="rounded-lg border border-slate-700/40 bg-slate-800/30 p-3">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-medium text-slate-400 truncate">{selectedFile.path}</span>
                    <button
                      type="button"
                      onClick={() => { setSelectedFile(null); setFileContent(null); }}
                      className="text-xs text-slate-500 hover:text-slate-300"
                    >
                      Close
                    </button>
                  </div>
                  {isLoadingContent ? (
                    <div className="py-8 text-center text-sm text-slate-500">Loading...</div>
                  ) : fileContent !== null ? (
                    (() => {
                      const ext = getFileExtension(selectedFile.path);
                      const isMarkdown = ext === "md" || ext === "markdown";
                      if (isMarkdown) {
                        return (
                          <div
                            className="prose prose-sm prose-invert max-w-none overflow-auto text-sm"
                            dangerouslySetInnerHTML={{ __html: renderMarkdown(fileContent) }}
                          />
                        );
                      }
                      return (
                        <pre className="max-h-96 overflow-auto rounded bg-slate-900/80 p-3 text-xs text-slate-300">
                          <code>{fileContent}</code>
                        </pre>
                      );
                    })()
                  ) : (
                    <div className="py-8 text-center text-sm text-slate-500">Unable to load file</div>
                  )}
                </div>
              )}
            </div>
          </div>
        </Card>
      )}
      </div>
    </DetailPageLayout>
  );
}
