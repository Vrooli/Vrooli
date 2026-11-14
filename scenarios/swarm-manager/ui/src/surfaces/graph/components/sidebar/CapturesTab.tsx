/** Capture collection surface. Domain content remains in CaptureCard. */
import { memo, useMemo } from "react";
import { Plus } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useCaptureStore } from "../../../../stores";
import { CaptureCard } from "../../../../components/capture/capture-card";
import { CollectionRow } from "../../../../components/ui/collection-row";
import { captureService } from "../../../../services/capture-service";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";
import { captureDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { Button } from "../../../../components/ui/button";

interface CapturesTabProps {
  searchQuery: string;
  filters: CaptureFilters;
  sort: SortConfig;
  onClearSearch?: () => void;
  onCreateCapture?: () => void;
}

function compareCaptures(sort: SortConfig) {
  const direction = sort.direction === "asc" ? 1 : -1;
  return (a: Capture, b: Capture) => {
    if (sort.field === "alphabetical") return a.text.localeCompare(b.text) * direction;
    if (sort.field === "status") return a.status.localeCompare(b.status) * direction;
    return (new Date(b.created).getTime() - new Date(a.created).getTime()) * direction;
  };
}

function CapturesTabImpl({ searchQuery, filters, sort, onClearSearch, onCreateCapture }: CapturesTabProps) {
  const navigate = useNavigate();
  const captures = useCaptureStore((s) => s.captures);
  const sorted = useMemo(
    () =>
      captures
        .filter(
          (capture) =>
            (filters.statuses.length === 0 || filters.statuses.includes(capture.status)) &&
            (!searchQuery || matchesSearch(searchQuery, capture.text)),
        )
        .sort(compareCaptures(sort)),
    [captures, filters.statuses, searchQuery, sort],
  );

  const open = (capture: Capture) => navigate(captureDetailPath(capture.id));

  if (sorted.length === 0) {
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.captures}
        title={searchQuery || filters.statuses.length ? "No captures match your filters." : "No captures yet."}
        hint="Quick thoughts and observations land here before classification."
        query={searchQuery}
        onClearSearch={onClearSearch}
        action={
          onCreateCapture ? (
            <Button type="button" size="sm" data-testid="captures-tab-create-capture" onClick={onCreateCapture}>
              <Plus className="mr-1.5 h-3.5 w-3.5" />
              Quick capture
            </Button>
          ) : undefined
        }
      />
    );
  }

  return (
    <CollectionList
      items={sorted}
      getKey={(capture) => capture.id}
      label="Captures"
      virtualize
      onOpen={open}
      selection={{ mode: "none", enterOn: ["shortcut"] }}
      actions={[
        {
          id: "open",
          label: "Open",
          onSelect: ([capture]) => {
            if (capture) open(capture);
          },
        },
        {
          id: "classify",
          label: "Classify",
          bulk: true,
          onSelect: async (rows) => {
            for (const capture of rows) await captureService.classify(capture.id);
          },
        },
        {
          id: "delete",
          label: "Delete",
          tone: "destructive",
          bulk: true,
          onSelect: async (rows) => {
            for (const capture of rows) await captureService.remove(capture.id);
          },
        },
      ]}
      renderItem={(capture) => (
        <CollectionRow>
          <CaptureCard capture={capture} />
        </CollectionRow>
      )}
    />
  );
}

export const CapturesTab = memo(CapturesTabImpl);
