/** Execution collection surface. Domain content remains in ExecutionSummaryCard. */
import { memo, useMemo } from "react";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useExecutionStore } from "../../../../stores";
import { executionService } from "../../../../services";
import { buildExecutionNodeId } from "../../lib/node-id-parser";
import { ExecutionSummaryCard } from "../../../../components/execution/execution-summary-card";
import { CollectionRow } from "../../../../components/ui/collection-row";
import { matchesSearch } from "./useSidebarSearch";
import type { ExecutionRecord } from "../../../../types";
import type { ExecutionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface ExecutionsTabProps {
  searchQuery: string;
  filters: ExecutionFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

const CANCELABLE_STATUSES = new Set(["pending", "running", "in_progress"]);

function compareExecutions(sort: SortConfig) {
  const direction = sort.direction === "asc" ? 1 : -1;
  return (a: ExecutionRecord, b: ExecutionRecord) => {
    if (sort.field === "alphabetical") return a.backlogName.localeCompare(b.backlogName) * direction;
    if (sort.field === "status") return a.status.localeCompare(b.status) * direction;
    return (new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) * direction;
  };
}

function ExecutionsTabImpl({ searchQuery, filters, sort, onItemClick, onClearSearch }: ExecutionsTabProps) {
  const items = useExecutionStore((s) => s.items);
  const sorted = useMemo(
    () =>
      items
        .filter(
          (item) =>
            (filters.statuses.length === 0 || filters.statuses.includes(item.status)) &&
            (filters.modes.length === 0 || filters.modes.includes(item.mode)) &&
            (!searchQuery || matchesSearch(searchQuery, item.backlogName, item.status, item.mode)),
        )
        .sort(compareExecutions(sort)),
    [items, filters.modes, filters.statuses, searchQuery, sort],
  );

  const open = (item: ExecutionRecord) => onItemClick(buildExecutionNodeId(item.executionId));

  if (sorted.length === 0) {
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.executions}
        title={
          searchQuery || filters.statuses.length || filters.modes.length
            ? "No executions match your filters."
            : "No executions yet."
        }
        hint="Runs and reviews appear here as agents start work."
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <CollectionList
      items={sorted}
      getKey={(item) => item.executionId}
      label="Executions"
      virtualize
      onOpen={open}
      selection={{ mode: "none", enterOn: ["shortcut"] }}
      actions={[
        {
          id: "open",
          label: "Open",
          onSelect: ([item]) => {
            if (item) open(item);
          },
        },
        {
          id: "cancel",
          label: "Cancel",
          bulk: true,
          hidden: (item) => !CANCELABLE_STATUSES.has(item.status),
          onSelect: async (rows) => {
            for (const item of rows) await executionService.cancel(item.executionId);
          },
        },
        {
          id: "retry",
          label: "Retry",
          bulk: true,
          onSelect: async (rows) => {
            for (const item of rows) await executionService.retry(item.executionId);
          },
        },
      ]}
      renderItem={(item) => (
        <CollectionRow data-testid="sidebar-execution-item">
          <ExecutionSummaryCard item={item} />
        </CollectionRow>
      )}
    />
  );
}

export const ExecutionsTab = memo(ExecutionsTabImpl);
