/**
 * SidebarEmptyState — Canonical empty-state composite for every sidebar tab.
 *
 * Pass an icon, the "no results" title, and (optionally) the active search
 * query plus a clear-search callback. When `query` is set the title flips to
 * `No matches for "<query>"` and a Clear-search button appears so the user
 * can recover from a stale query left over after a tab switch.
 *
 * Tabs must use this primitive — bespoke empty-state JSX has been removed
 * from every *Tab.tsx by design (sidebar empty-state seam, see SEAMS.md).
 */

import type { ComponentType } from "react";
import { X } from "lucide-react";
import { Button } from "../../../../components/ui/button";
import { selectors } from "../../../../consts/selectors";

export interface SidebarEmptyStateProps {
  icon: ComponentType<{ className?: string }>;
  title: string;
  hint?: string;
  query?: string;
  onClearSearch?: () => void;
}

export function SidebarEmptyState({
  icon: Icon,
  title,
  hint,
  query,
  onClearSearch,
}: SidebarEmptyStateProps) {
  const trimmedQuery = query?.trim() ?? "";
  const showClear = trimmedQuery.length > 0 && Boolean(onClearSearch);
  const effectiveTitle = trimmedQuery.length > 0 ? `No matches for "${trimmedQuery}"` : title;

  return (
    <div
      className="flex flex-col items-center justify-center gap-2 py-12 text-center text-slate-500"
      data-testid={selectors.sidebar.emptyState}
    >
      <Icon className="h-8 w-8" />
      <p
        className="text-sm text-slate-400"
        data-testid={selectors.sidebar.emptyStateTitle}
      >
        {effectiveTitle}
      </p>
      {hint && trimmedQuery.length === 0 && (
        <p className="max-w-xs text-xs leading-snug text-slate-500">{hint}</p>
      )}
      {showClear && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="mt-1"
          onClick={onClearSearch}
          data-testid={selectors.sidebar.emptyStateClear}
        >
          <X className="mr-1.5 h-3.5 w-3.5" />
          Clear search
        </Button>
      )}
    </div>
  );
}
