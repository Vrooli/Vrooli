/**
 * CapturesTab - Lists disposable capture events while they classify.
 *
 * Classification decisions are reviewed on the proposal rail.
 */

import { memo, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2, Plus, Trash2 } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useCaptureStore } from "../../../../stores";
import { CaptureCard } from "../../../../components/capture/capture-card";
import { captureService } from "../../../../services/capture-service";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";
import { captureDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { Button } from "../../../../components/ui/button";
import { useDeleteConfirm } from "../../../../hooks/useDeleteConfirm";
import { runBulkAction, summarizeBulkOutcomes, type BulkOutcome } from "./bulk-actions";

interface CapturesTabProps {
  searchQuery: string;
  filters: CaptureFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelection?: (id: string) => void;
  onVisibleIdsChange?: (ids: string[]) => void;
  onCreateCapture?: () => void;
}

function applyFilters(items: Capture[], filters: CaptureFilters): Capture[] {
  if (filters.statuses.length === 0) return items;
  return items.filter((c) => filters.statuses.includes(c.status));
}

function applySort(items: Capture[], sort: SortConfig): Capture[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

  sorted.sort((a, b) => {
    switch (sort.field) {
      case "recency":
        return (new Date(b.created).getTime() - new Date(a.created).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return a.text.localeCompare(b.text) * dir;
      default:
        return (new Date(b.created).getTime() - new Date(a.created).getTime()) * dir;
    }
  });

  return sorted;
}

function captureSelectionId(capture: Capture): string {
  return `capture:${capture.id}`;
}

function CapturesTabImpl({
  searchQuery,
  filters,
  sort,
  onItemClick: _onItemClick,
  onClearSearch,
  selectionMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
  onVisibleIdsChange,
  onCreateCapture,
}: CapturesTabProps) {
  const navigate = useNavigate();
  const captures = useCaptureStore((s) => s.captures);

  let filtered = applyFilters(captures, filters);
  if (searchQuery) {
    filtered = filtered.filter((c) => matchesSearch(searchQuery, c.text));
  }
  const sorted = applySort(filtered, sort);
  useEffect(() => {
    onVisibleIdsChange?.(sorted.map(captureSelectionId));
  }, [onVisibleIdsChange, sorted]);
  const selectedCaptures = useMemo(
    () => sorted.filter((capture) => selectedIds.has(captureSelectionId(capture))),
    [selectedIds, sorted],
  );

  if (sorted.length === 0) {
    const filtersActive = filters.statuses.length > 0;
    const title = filtersActive ? "No captures match your filters." : "No captures yet.";
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.captures}
        title={title}
        hint={filtersActive ? undefined : "Quick thoughts and observations land here before classification."}
        query={searchQuery}
        onClearSearch={onClearSearch}
        action={
          !filtersActive && onCreateCapture ? (
            <Button
              type="button"
              size="sm"
              className="mt-1"
              onClick={onCreateCapture}
              data-testid="captures-tab-create-capture"
            >
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Quick capture
            </Button>
          ) : undefined
        }
      />
    );
  }

  return (
    <>
      {selectionMode && <CaptureBulkActions selectedCaptures={selectedCaptures} />}
      <div className="space-y-1.5">
        {sorted.map((capture) => (
          <div
            key={capture.id}
            role="button"
            tabIndex={0}
            onClick={() => navigate(captureDetailPath(capture.id))}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                navigate(captureDetailPath(capture.id));
              }
            }}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-capture-item"
          >
            <div className="flex items-start gap-2">
              {selectionMode && (
                <input
                  type="checkbox"
                  aria-label={`${selectedIds.has(captureSelectionId(capture)) ? "Deselect" : "Select"} ${capture.text.slice(0, 40)}`}
                  checked={selectedIds.has(captureSelectionId(capture))}
                  onClick={(event) => event.stopPropagation()}
                  onChange={(event) => {
                    event.stopPropagation();
                    onToggleSelection?.(captureSelectionId(capture));
                  }}
                  className="mt-1"
                />
              )}
              <CaptureCard
                capture={capture}
                onClick={() => navigate(captureDetailPath(capture.id))}
                className="min-w-0 flex-1 border-0 bg-transparent p-0"
              />
            </div>
          </div>
        ))}
      </div>

    </>
  );
}

export const CapturesTab = memo(CapturesTabImpl);

function CaptureBulkActions({ selectedCaptures }: { selectedCaptures: Capture[] }) {
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const updateCapture = useCaptureStore((s) => s.updateCapture);
  const [action, setAction] = useState<"classify" | "delete">("classify");
  const { requestDelete, dialogProps: deleteDialogProps } = useDeleteConfirm("capture");
  const [running, setRunning] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [outcomes, setOutcomes] = useState<BulkOutcome[]>([]);

  const eligible = action === "classify"
    ? selectedCaptures.filter((capture) => capture.status !== "classifying")
    : selectedCaptures;

  const execute = async () => {
    setRunning(true);
    setSummary(null);
    setOutcomes([]);
    try {
      const next = await runBulkAction(eligible, {
        getId: captureSelectionId,
        getLabel: (capture) => capture.text.slice(0, 48) || capture.id,
        run: async (capture) => {
          if (action === "classify") {
            await captureService.classify(capture.id);
            updateCapture(capture.id, { status: "classifying", classification: null });
          } else {
            await captureService.remove(capture.id);
            removeCapture(capture.id);
          }
        },
      });
      setOutcomes(next);
      setSummary(summarizeBulkOutcomes(next));
      await fetchCaptures({ force: true });
    } finally {
      setRunning(false);
    }
  };

  const failed = outcomes.filter((outcome) => outcome.status === "failed");

  return (
    <div className="mb-2 rounded-lg border border-slate-800 bg-slate-900/70 p-2" data-testid="sidebar-capture-bulk-actions">
      <div className="flex flex-wrap items-center gap-2">
        <select value={action} onChange={(event) => setAction(event.target.value as "classify" | "delete")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Capture bulk action">
          <option value="classify">Retry classification</option>
          <option value="delete">Delete selected</option>
        </select>
        <button
          type="button"
          disabled={selectedCaptures.length === 0 || eligible.length === 0 || running}
          onClick={() => {
            if (action === "delete") {
              requestDelete({
                entityName: eligible.length === 1 ? (eligible[0]?.text.slice(0, 48) || eligible[0]?.id || "capture") : `${eligible.length} captures`,
                count: eligible.length,
                description: `This permanently deletes ${eligible.length} capture${eligible.length === 1 ? "" : "s"}. This action cannot be undone.`,
                confirmLabel: "Delete selected",
                onConfirm: execute,
              });
            } else {
              void execute();
            }
          }}
          className="inline-flex h-8 items-center gap-1.5 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : action === "delete" ? <Trash2 className="h-3.5 w-3.5" /> : null}
          Apply
        </button>
      </div>
      <div className="mt-1.5 text-[11px] text-slate-500">{eligible.length} eligible{summary ? ` - ${summary}` : ""}</div>
      {failed.length > 0 && <div className="mt-1 text-[11px] text-red-300">{failed.map((outcome) => <div key={outcome.id}>{outcome.label}: {outcome.message}</div>)}</div>}
      <ConfirmDialog {...deleteDialogProps} />
    </div>
  );
}
