/**
 * OpsFilterBar — filter controls for the Operations Center.
 *
 * Five-input filter row: free-text search, status select, lane select,
 * owner-type select, time-window select. State lives in the
 * operations-store and is mirrored onto the URL by the page-level
 * controller so filter values survive reload and deep links.
 *
 * Inputs are deliberately uncontrolled in the React sense — they read
 * the current store value and dispatch a single field update on change.
 * This keeps render cycles short and avoids the typing-lag pattern that
 * controlled inputs sometimes introduce in Zustand pages.
 */

import { Search, X } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { Select } from "../ui/select";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";
import { OPERATIONS_LANES } from "../../types/operations";
import type { OperationsFilters } from "../../types/operations";
import { laneLabel } from "./utils";

const STATUS_OPTIONS: ReadonlyArray<{ value: string; label: string }> = [
  { value: "", label: "All statuses" },
  { value: "running", label: "Running" },
  { value: "starting", label: "Starting" },
  { value: "pending", label: "Pending" },
  { value: "needs_review", label: "Needs review" },
  { value: "complete", label: "Complete" },
  { value: "failed", label: "Failed" },
  { value: "cancelled", label: "Cancelled" },
];

const OWNER_TYPE_OPTIONS: ReadonlyArray<{ value: string; label: string }> = [
  { value: "", label: "All owners" },
  { value: "initiative", label: "Initiative" },
  { value: "backlog", label: "Backlog item" },
  { value: "scenario", label: "Scenario" },
  { value: "capture", label: "Capture" },
  { value: "session", label: "Session" },
];

const WINDOW_OPTIONS: ReadonlyArray<{ value: number; label: string }> = [
  { value: 60 * 60, label: "1 hour" },
  { value: 3 * 60 * 60, label: "3 hours" },
  { value: 6 * 60 * 60, label: "6 hours" },
  { value: 12 * 60 * 60, label: "12 hours" },
  { value: 24 * 60 * 60, label: "24 hours" },
];

export interface OpsFilterBarProps {
  filters: OperationsFilters;
  onFiltersChange(next: Partial<OperationsFilters>): void;
  onReset(): void;
  className?: string;
}

export function OpsFilterBar({
  filters,
  onFiltersChange,
  onReset,
  className,
}: OpsFilterBarProps) {
  const status = filters.statuses?.[0] ?? "";
  const lane = filters.lanes?.[0] ?? "";
  const ownerType = filters.ownerTypes?.[0] ?? "";
  const window = filters.windowSeconds ?? 3 * 60 * 60;
  const search = filters.q ?? "";

  const hasFilters =
    !!search ||
    !!status ||
    !!lane ||
    !!ownerType ||
    window !== 3 * 60 * 60;

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 rounded-xl border border-white/5 bg-slate-900/40 p-3",
        className,
      )}
      data-testid={selectors.operationsCenter.filterBar}
    >
      <div className="relative flex-1 min-w-[200px]">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500"
          aria-hidden
        />
        <Input
          type="text"
          value={search}
          onChange={(e) => onFiltersChange({ q: e.target.value })}
          placeholder="Search owner, run id, title..."
          aria-label="Search activities"
          className="pl-9"
          data-testid={selectors.operationsCenter.searchInput}
        />
      </div>
      <Select
        variant="filter"
        withChevron
        wrapperClassName="min-w-[150px]"
        value={status}
        onChange={(e) =>
          onFiltersChange({ statuses: e.target.value ? [e.target.value] : [] })
        }
        aria-label="Status filter"
        data-testid={selectors.operationsCenter.statusSelect}
      >
        {STATUS_OPTIONS.map((opt) => (
          <option key={opt.value || "all"} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </Select>
      <Select
        variant="filter"
        withChevron
        wrapperClassName="min-w-[150px]"
        value={lane}
        onChange={(e) =>
          onFiltersChange({ lanes: e.target.value ? [e.target.value] : [] })
        }
        aria-label="Lane filter"
        data-testid={selectors.operationsCenter.laneSelect}
      >
        <option value="">All lanes</option>
        {OPERATIONS_LANES.map((value) => (
          <option key={value} value={value}>
            {laneLabel(value)}
          </option>
        ))}
      </Select>
      <Select
        variant="filter"
        withChevron
        wrapperClassName="min-w-[150px]"
        value={ownerType}
        onChange={(e) =>
          onFiltersChange({
            ownerTypes: e.target.value ? [e.target.value] : [],
          })
        }
        aria-label="Owner type filter"
        data-testid={selectors.operationsCenter.ownerTypeSelect}
      >
        {OWNER_TYPE_OPTIONS.map((opt) => (
          <option key={opt.value || "all"} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </Select>
      <Select
        variant="filter"
        withChevron
        wrapperClassName="min-w-[120px]"
        value={String(window)}
        onChange={(e) => onFiltersChange({ windowSeconds: Number(e.target.value) })}
        aria-label="Time window"
        data-testid={selectors.operationsCenter.windowSelect}
      >
        {WINDOW_OPTIONS.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </Select>
      {hasFilters && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onReset}
          data-testid={selectors.operationsCenter.resetFilters}
          aria-label="Reset filters"
        >
          <X className="mr-1 h-3.5 w-3.5" aria-hidden />
          Reset
        </Button>
      )}
    </div>
  );
}
