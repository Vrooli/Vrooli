import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";
import { EmptyState } from "./EmptyState";
import { Skeleton } from "./Skeleton";

export type SortDirection = "asc" | "desc" | null;

export interface ColumnDef<Row> {
  key: string;
  header: React.ReactNode;
  cell: (row: Row, index: number) => React.ReactNode;
  sortable?: boolean;
  width?: string;
  align?: "left" | "right" | "center";
}

export interface TableProps<Row> {
  columns: ColumnDef<Row>[];
  rows: Row[];
  loading?: boolean;
  error?: string | null;
  emptyTitle?: string;
  emptyDescription?: React.ReactNode;
  rowKey: (row: Row, index: number) => string;
  onRowClick?: (row: Row) => void;
  selectedKey?: string | null;
  sort?: { key: string; direction: SortDirection };
  onSortChange?: (next: { key: string; direction: SortDirection }) => void;
  caption?: string;
  "data-testid"?: string;
}

const HEADER_CELL = "border-b border-app-border bg-app-surface-muted px-3 py-2 text-left text-xs font-medium uppercase tracking-wide text-app-muted-foreground";
const BODY_CELL = "border-b border-app-border px-3 py-2 text-sm text-app-foreground";

export function Table<Row>({
  columns,
  rows,
  loading,
  error,
  emptyTitle = "No data",
  emptyDescription,
  rowKey,
  onRowClick,
  selectedKey,
  sort,
  onSortChange,
  caption,
  "data-testid": testId,
}: TableProps<Row>) {
  if (loading) {
    return (
      <div data-testid={testId} className="flex flex-col gap-2 p-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }
  if (error) {
    return (
      <div role="alert" data-testid={testId} className="rounded-panel border border-app-danger/40 bg-app-danger/10 p-4 text-sm text-app-danger">
        {error}
      </div>
    );
  }
  if (rows.length === 0) {
    return <EmptyState title={emptyTitle} description={emptyDescription} data-testid={testId} />;
  }

  const handleSort = (column: ColumnDef<Row>) => {
    if (!column.sortable || !onSortChange) return;
    const current = sort?.key === column.key ? sort.direction : null;
    const next: SortDirection = current === "asc" ? "desc" : current === "desc" ? null : "asc";
    onSortChange({ key: column.key, direction: next });
  };

  return (
    <div className="relative overflow-x-auto rounded-panel border border-app-border bg-app-surface">
      <table data-testid={testId} className="w-full border-collapse text-sm">
        {caption ? <caption className="sr-only">{caption}</caption> : null}
        <thead className="sticky top-0 z-10">
          <tr>
            {columns.map((col) => {
              const isSorted = sort?.key === col.key;
              const direction = isSorted ? sort.direction : null;
              const Icon = direction === "asc" ? ArrowUp : direction === "desc" ? ArrowDown : ArrowUpDown;
              return (
                <th
                  key={col.key}
                  scope="col"
                  style={col.width ? { width: col.width } : undefined}
                  className={cn(HEADER_CELL, col.align === "right" && "text-right", col.align === "center" && "text-center")}
                  aria-sort={
                    isSorted ? (direction === "asc" ? "ascending" : direction === "desc" ? "descending" : "none") : undefined
                  }
                >
                  {col.sortable ? (
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 text-xs font-medium uppercase tracking-wide text-app-muted-foreground hover:text-app-foreground"
                      onClick={() => handleSort(col)}
                    >
                      {col.header}
                      <Icon aria-hidden className="h-3 w-3" />
                    </button>
                  ) : (
                    col.header
                  )}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => {
            const key = rowKey(row, i);
            const selected = selectedKey === key;
            const isClickable = Boolean(onRowClick);
            return (
              <tr
                key={key}
                data-row-key={key}
                aria-selected={selected || undefined}
                className={cn(
                  "transition-colors",
                  isClickable && "cursor-pointer hover:bg-app-surface-muted",
                  selected && "bg-app-primary/10",
                )}
                onClick={isClickable ? () => onRowClick?.(row) : undefined}
                onKeyDown={
                  isClickable
                    ? (e) => {
                        if (e.key === "Enter" || e.key === " ") {
                          e.preventDefault();
                          onRowClick?.(row);
                        }
                      }
                    : undefined
                }
                tabIndex={isClickable ? 0 : undefined}
                role={isClickable ? "button" : undefined}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={cn(BODY_CELL, col.align === "right" && "text-right", col.align === "center" && "text-center")}
                  >
                    {col.cell(row, i)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
