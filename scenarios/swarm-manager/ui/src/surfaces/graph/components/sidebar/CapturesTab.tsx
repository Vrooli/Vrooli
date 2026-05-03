/**
 * CapturesTab - Lists captures with inline triage via CaptureCard.
 *
 * Users can accept, edit, dismiss, and retry classification directly
 * from the sidebar without leaving the graph view.
 */

import { memo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { MessageSquare } from "lucide-react";
import { useCaptureStore } from "../../../../stores";
import { CaptureCard } from "../../../../components/capture/capture-card";
import { BacklogFormDialog } from "../../../../components/backlog/backlog-form-dialog";
import { backlogService } from "../../../../services/backlog-service";
import { useBacklogStore } from "../../../../stores";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture, BacklogFormValues } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";
import { captureDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface CapturesTabProps {
  searchQuery: string;
  filters: CaptureFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
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

function CapturesTabImpl({ searchQuery, filters, sort, onItemClick: _onItemClick, onClearSearch }: CapturesTabProps) {
  const navigate = useNavigate();
  const captures = useCaptureStore((s) => s.captures);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

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
    const filtersActive = filters.statuses.length > 0;
    const title = filtersActive ? "No captures match your filters." : "No captures yet.";
    return (
      <SidebarEmptyState
        icon={MessageSquare}
        title={title}
        hint={filtersActive ? undefined : "Quick thoughts and observations land here before classification."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
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
            onClick={() => navigate(captureDetailPath(capture.id))}
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

export const CapturesTab = memo(CapturesTabImpl);
