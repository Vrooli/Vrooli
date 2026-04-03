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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Target, FolderOpen, Archive } from "lucide-react";
import { Button } from "../components/ui/button";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { StatusBadge } from "../components/detail/StatusBadge";
import { INITIATIVE_LENSES } from "../components/detail/lens-options";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { FileTree, type TreeFile } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { EntityLink } from "../components/ui/entity-link";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { initiativeService } from "../services";
import { selectors } from "../consts/selectors";
import { BACKLOG_STATUS_CHIP_COLORS } from "../types";
import type { BacklogKind, BacklogStatus } from "../types";
import { useBacklogStore, useDetailSelectionStore } from "../stores";

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

  const handleFileSelect = useCallback(
    (file: TreeFile) => {
      if (file.type === "directory") return;
      setSelectedFile(file);
    },
    [],
  );

  const { data: fileContent } = useQuery({
    queryKey: ["initiative", name, "files", selectedFile?.path, "content"],
    queryFn: () => initiativeService.getFileContent(name ?? "", selectedFile?.path ?? ""),
    enabled: !!name && !!selectedFile,
    ...defaultQueryOptions,
  });

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
        />
      }
      mobileActions={mobileActions}
      mobileActionsTitle="Initiative Actions"
    >
      <div className="space-y-0 md:mx-auto md:max-w-3xl" data-testid={selectors.initiativeDetails.page}>
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
        <DetailSection title={`Items (${resolvedItems.length})`} data-testid={selectors.initiativeDetails.itemsList}>
          <div className="flex flex-wrap gap-1.5">
            {resolvedItems.map((item) => {
              const chipColors = BACKLOG_STATUS_CHIP_COLORS[item.status] ?? "bg-slate-600/20 text-slate-300";
              return (
                <EntityLink
                  key={item.ref}
                  entityType="backlog"
                  kind={item.kind}
                  name={item.name}
                  label={item.title}
                  className={`hover:brightness-125 ${chipColors}`}
                />
              );
            })}
          </div>
        </DetailSection>
      )}

      {/* Files */}
      {!isLoadingFiles && displayFiles.length > 0 && (
        <DetailSection title={`Files (${displayFiles.length})`} icon={FolderOpen} data-testid="initiative-files">
          <div className={selectedFile ? "grid grid-cols-1 gap-4 lg:grid-cols-2" : ""}>
            <FileTree
              files={displayFiles}
              onFileSelect={handleFileSelect}
              selectedPath={selectedFile?.path}
              className="rounded-lg border border-slate-700/40 bg-slate-800/30 p-2"
            />

            {selectedFile && (
              <FilePreview
                backlogKind={"initiative" as BacklogKind}
                backlogName={name ?? ""}
                filePath={selectedFile.path}
                fileName={selectedFile.name}
                content={fileContent ?? undefined}
                readOnly
                compactHeader
                className="rounded-lg border border-slate-700/40"
                headerActions={
                  <button
                    type="button"
                    onClick={() => setSelectedFile(null)}
                    className="text-xs text-slate-500 hover:text-slate-300"
                  >
                    Close
                  </button>
                }
              />
            )}
          </div>
        </DetailSection>
      )}
      </div>
    </DetailPageLayout>
  );
}
