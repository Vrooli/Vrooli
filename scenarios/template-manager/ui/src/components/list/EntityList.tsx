import { ChevronLeft, ChevronRight, Search } from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export interface EntityListProps<T> {
  rows: T[];
  getRowKey: (row: T) => string;
  getRowHref: (row: T) => string;
  getRowTestId: (row: T) => string;
  getRowLabel: (row: T) => string;
  renderRow: (row: T) => ReactNode;
  searchText: (row: T) => string;
  searchLabel: string;
  searchPlaceholder: string;
  emptyLabel: string;
  /** Rows per page. Kept small so interactive rows stay near the viewport. */
  pageSize?: number;
  tableTestId?: string;
}

/**
 * A compact, paginated, searchable list of entity rows. Only the current page's
 * rows are rendered to the DOM, so long ledgers never strand dozens of
 * interactive elements far below the fold (a ui-health runtime-render gate).
 * Each row is a router link to the entity's detail view.
 */
export function EntityList<T>({
  rows,
  getRowKey,
  getRowHref,
  getRowTestId,
  getRowLabel,
  renderRow,
  searchText,
  searchLabel,
  searchPlaceholder,
  emptyLabel,
  pageSize = 6,
  tableTestId,
}: EntityListProps<T>) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(0);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return rows;
    return rows.filter((row) => searchText(row).toLowerCase().includes(needle));
  }, [rows, query, searchText]);

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  // Clamp the active page whenever filtering shrinks the result set.
  useEffect(() => {
    if (page > pageCount - 1) {
      setPage(pageCount - 1);
    }
  }, [page, pageCount]);

  const safePage = Math.min(page, pageCount - 1);
  const pageRows = filtered.slice(safePage * pageSize, safePage * pageSize + pageSize);

  return (
    <div className="flex flex-col gap-3">
      <label className="relative min-w-0">
        <span className="sr-only">{searchLabel}</span>
        <Search aria-hidden className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-app-muted-foreground" />
        <input
          type="search"
          value={query}
          placeholder={searchPlaceholder}
          onChange={(event) => setQuery(event.target.value)}
          className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-9 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 md:text-sm"
        />
      </label>

      {pageRows.length === 0 ? (
        <p
          data-testid={tableTestId}
          className="rounded-panel border border-dashed border-app-border px-3 py-6 text-center text-sm text-app-muted-foreground"
        >
          {emptyLabel}
        </p>
      ) : (
        <ul data-testid={tableTestId} className="grid gap-2">
          {pageRows.map((row) => (
            <li key={getRowKey(row)} className="min-w-0">
              <Link
                to={getRowHref(row)}
                data-testid={getRowTestId(row)}
                aria-label={getRowLabel(row)}
                className="grid min-h-11 min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-panel border border-app-border px-3 py-2 transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
              >
                {renderRow(row)}
              </Link>
            </li>
          ))}
        </ul>
      )}

      {pageCount > 1 && (
        <nav className="flex items-center justify-between gap-3" aria-label={t(strings.list.pagination.label)}>
          <button
            type="button"
            onClick={() => setPage((current) => Math.max(0, current - 1))}
            disabled={safePage === 0}
            className="inline-flex min-h-11 items-center gap-1 rounded-control border border-app-border px-3 text-sm font-medium transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-50"
          >
            <ChevronLeft aria-hidden className="h-4 w-4" />
            {t(strings.list.pagination.prev)}
          </button>
          <span className="text-sm tabular-nums text-app-muted-foreground">
            {t(strings.list.pagination.status, { page: safePage + 1, total: pageCount })}
          </span>
          <button
            type="button"
            onClick={() => setPage((current) => Math.min(pageCount - 1, current + 1))}
            disabled={safePage >= pageCount - 1}
            className="inline-flex min-h-11 items-center gap-1 rounded-control border border-app-border px-3 text-sm font-medium transition-colors hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-50"
          >
            {t(strings.list.pagination.next)}
            <ChevronRight aria-hidden className="h-4 w-4" />
          </button>
        </nav>
      )}
    </div>
  );
}

/**
 * The standard two-part row body: a primary/secondary text block plus a
 * trailing node (usually a status badge). Used by the entity list pages.
 */
export function EntityRowBody({ primary, secondary, trailing }: { primary: string; secondary: string; trailing: ReactNode }) {
  return (
    <>
      <span className="min-w-0">
        <span className="block truncate text-sm font-medium">{primary}</span>
        <span className="block truncate text-xs text-app-muted-foreground">{secondary}</span>
      </span>
      {trailing}
    </>
  );
}
