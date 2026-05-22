import * as React from "react";
import { cn } from "../lib/utils";
import { selectors } from "../consts/selectors";

export interface DataTableColumn<TRow> {
  /** Stable column key — appears in the dynamic selector. */
  key: string;
  /** Pre-translated header label. */
  header: string;
  /** Cell renderer. Returns a React node — primitive scalars get wrapped. */
  cell: (row: TRow) => React.ReactNode;
  /** Optional className applied to <td> and <th>. */
  className?: string;
}

export interface DataTableProps<TRow> {
  rows: ReadonlyArray<TRow>;
  /** Stable id extractor; used for keys + selectors. */
  getRowId: (row: TRow) => string;
  columns: ReadonlyArray<DataTableColumn<TRow>>;
  /** Pre-translated message shown when rows is empty. */
  emptyMessage: string;
  /** Optional caption for SR users; renders as a visually-hidden <caption>. */
  caption?: string;
  /** Click handler when a row is selected. */
  onRowClick?: (row: TRow) => void;
  className?: string;
}

export function DataTable<TRow>({
  rows,
  getRowId,
  columns,
  emptyMessage,
  caption,
  onRowClick,
  className,
}: DataTableProps<TRow>) {
  if (rows.length === 0) {
    return (
      <div
        data-testid={selectors.shared.dataTable.empty}
        className="rounded-panel border border-dashed border-app-border bg-app-surface p-6 text-center text-sm text-app-muted-foreground backdrop-blur-sm"
      >
        {emptyMessage}
      </div>
    );
  }
  return (
    <div
      data-testid={selectors.shared.dataTable.root}
      className={cn("overflow-x-auto rounded-panel border border-app-border bg-app-surface", className)}
    >
      <table className="w-full text-left text-sm">
        {caption ? (
          <caption className="sr-only">{caption}</caption>
        ) : null}
        <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
          <tr>
            {columns.map((col) => (
              <th key={col.key} scope="col" className={cn("p-3 font-semibold", col.className)}>
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => {
            const id = getRowId(row);
            return (
              <tr
                key={id}
                data-testid={selectors.shared.dataTable.row({ id })}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                className={cn(
                  "border-t border-app-border align-top",
                  onRowClick ? "cursor-pointer hover:bg-app-surface-muted" : "",
                )}
              >
                {columns.map((col) => (
                  <td
                    key={col.key}
                    data-testid={selectors.shared.dataTable.cell({ id, column: col.key })}
                    className={cn("p-3 text-app-foreground", col.className)}
                  >
                    {col.cell(row)}
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
