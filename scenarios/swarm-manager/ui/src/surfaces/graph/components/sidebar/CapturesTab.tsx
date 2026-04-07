/**
 * CapturesTab - Lists captures with inline triage via CaptureCard.
 *
 * Users can accept, edit, dismiss, and retry classification directly
 * from the sidebar without leaving the graph view.
 */

import { useState } from "react";
import { MessageSquare } from "lucide-react";
import { useCaptureStore } from "../../../../stores";
import { CaptureCard } from "../../../../components/capture/capture-card";
import { BacklogFormDialog } from "../../../../components/backlog/backlog-form-dialog";
import { backlogService } from "../../../../services/backlog-service";
import { useBacklogStore } from "../../../../stores";
import { useDetailSelectionStore } from "../../../../stores/detail-selection-store";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture, BacklogFormValues } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";

interface CapturesTabProps {
  searchQuery: string;
  filters: CaptureFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
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

export function CapturesTab({ searchQuery, filters, sort, onItemClick: _onItemClick }: CapturesTabProps) {
  const captures = useCaptureStore((s) => s.captures);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);
  const selectCapture = useDetailSelectionStore((s) => s.selectCapture);

  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editPrefill, setEditPrefill] = useState<BacklogFormValues | undefined>();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  let filtered = applyFilters(captures, filters);
  if (searchQuery) {
    filtered = filtered.filter((c) => matchesSearch(searchQuery, c.text));
  }
  const sorted = applySort(filtered, sort);

  const handleEditItem = (prefill: BacklogFormValues) => {
    setEditPrefill(prefill);
    setSubmitError(null);
    setShowEditDialog(true);
  };

  const handleEditSubmit = async (values: BacklogFormValues) => {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const created = await backlogService.create({ ...values, suggestedSkills: [] });
      upsertBacklogItem(created);
      setShowEditDialog(false);
      setEditPrefill(undefined);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to create backlog item");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <MessageSquare className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery || filters.statuses.length > 0 ? "No captures match your filters." : "No captures yet."}</p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-1.5">
        {sorted.map((capture) => (
          <CaptureCard
            key={capture.id}
            capture={capture}
            onEditItem={handleEditItem}
            onClick={() => selectCapture(capture.id)}
            className="rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5"
          />
        ))}
      </div>

      <BacklogFormDialog
        isOpen={showEditDialog}
        mode="create"
        initialValues={editPrefill}
        isSubmitting={isSubmitting}
        submitError={submitError}
        onClose={() => {
          setShowEditDialog(false);
          setEditPrefill(undefined);
        }}
        onSubmit={handleEditSubmit}
      />
    </>
  );
}
