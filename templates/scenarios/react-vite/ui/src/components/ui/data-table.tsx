/**
 * @vrooliComponentSource react-component-library:DataTable
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption template:react-vite:data-table
 * @vrooliComponentAppliedAt 2026-07-07T00:00:00Z
 * @vrooliComponentSourceSha256 bcb59d91151425aa99db0e396e6e9f1c3192554180b4fa32672c4db9756ecdec
 * @vrooliComponentDriftHash bcb59d91151425aa99db0e396e6e9f1c3192554180b4fa32672c4db9756ecdec
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { ArrowDown, ArrowUp, ChevronsUpDown, Search } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";

export interface DataTableColumn<Row> {
  id: string;
  header: string;
  accessor: (row: Row) => ReactNode;
  sortValue?: (row: Row) => string | number;
  searchValue?: (row: Row) => string;
  className?: string;
}

export interface DataTableFilter<Row> {
  id: string;
  label: string;
  predicate: (row: Row) => boolean;
}

export interface DataTableProps<Row> {
  rows: Row[];
  columns: Array<DataTableColumn<Row>>;
  getRowKey: (row: Row, index: number) => string;
  caption: string;
  searchLabel?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  filters?: Array<DataTableFilter<Row>>;
  className?: string;
  tableTestId?: string;
}

type SortDirection = "asc" | "desc";

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

function compareValues(a: string | number, b: string | number) {
  if (typeof a === "number" && typeof b === "number") {
    return a - b;
  }
  return String(a).localeCompare(String(b), undefined, { numeric: true, sensitivity: "base" });
}

export function DataTable<Row>({
  rows,
  columns,
  getRowKey,
  caption,
  searchLabel = "Search rows",
  searchPlaceholder = "Search",
  emptyMessage = "No rows",
  filters = [],
  className,
  tableTestId,
}: DataTableProps<Row>) {
  const firstSortable = columns.find((column) => column.sortValue);
  const [query, setQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState(filters[0]?.id ?? "");
  const [sortColumn, setSortColumn] = useState(firstSortable?.id ?? "");
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");

  const filteredRows = useMemo(() => {
    const normalizedQuery = query.trim().toLowerCase();
    const filter = filters.find((entry) => entry.id === activeFilter);
    const searched = rows.filter((row) => {
      if (filter && !filter.predicate(row)) {
        return false;
      }
      if (!normalizedQuery) {
        return true;
      }
      return columns.some((column) => {
        const value = column.searchValue ? column.searchValue(row) : String(column.accessor(row) ?? "");
        return value.toLowerCase().includes(normalizedQuery);
      });
    });
    const sortable = columns.find((column) => column.id === sortColumn && column.sortValue);
    if (!sortable?.sortValue) {
      return searched;
    }
    return [...searched].sort((a, b) => {
      const result = compareValues(sortable.sortValue?.(a) ?? "", sortable.sortValue?.(b) ?? "");
      return sortDirection === "asc" ? result : -result;
    });
  }, [activeFilter, columns, filters, query, rows, sortColumn, sortDirection]);

  const toggleSort = (column: DataTableColumn<Row>) => {
    if (!column.sortValue) {
      return;
    }
    if (sortColumn === column.id) {
      setSortDirection((current) => (current === "asc" ? "desc" : "asc"));
      return;
    }
    setSortColumn(column.id);
    setSortDirection("asc");
  };

  return (
    <div className={joinClasses("min-w-0 rounded-panel border border-app-border bg-app-surface", className)}>
      <div className="flex flex-col gap-3 border-b border-app-border p-3 md:flex-row md:items-center md:justify-between">
        <label className="relative min-w-0 flex-1">
          <span className="sr-only">{searchLabel}</span>
          <Search aria-hidden className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-app-muted-foreground" />
          <input
            type="search"
            value={query}
            placeholder={searchPlaceholder}
            className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-9 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 md:text-sm"
            onChange={(event) => setQuery(event.target.value)}
          />
        </label>
        {filters.length > 0 && (
          <div className="flex flex-wrap gap-2" role="group" aria-label="Table filters">
            {filters.map((filter) => (
              <button
                key={filter.id}
                type="button"
                className={joinClasses(
                  "min-h-9 rounded-control border px-3 text-sm font-medium transition",
                  activeFilter === filter.id
                    ? "border-app-primary bg-app-primary text-app-primary-foreground"
                    : "border-app-border text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                )}
                onClick={() => setActiveFilter(filter.id)}
              >
                {filter.label}
              </button>
            ))}
          </div>
        )}
      </div>
      <div className="max-w-full overflow-x-auto">
        <table data-testid={tableTestId} className="w-full min-w-max border-collapse text-left text-sm">
          <caption className="sr-only">{caption}</caption>
          <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
            <tr>
              {columns.map((column) => {
                const active = sortColumn === column.id;
                const SortIcon = !column.sortValue ? null : active ? (sortDirection === "asc" ? ArrowUp : ArrowDown) : ChevronsUpDown;
                return (
                  <th key={column.id} scope="col" className={joinClasses("px-3 py-3 font-semibold", column.className)}>
                    {column.sortValue ? (
                      <button
                        type="button"
                        className="inline-flex min-h-9 items-center gap-1 rounded-control text-left hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                        onClick={() => toggleSort(column)}
                      >
                        <span>{column.header}</span>
                        {SortIcon && <SortIcon aria-hidden className="h-4 w-4" />}
                      </button>
                    ) : (
                      column.header
                    )}
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {filteredRows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-3 py-6 text-center text-app-muted-foreground">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              filteredRows.map((row, index) => (
                <tr key={getRowKey(row, index)} className="border-t border-app-border">
                  {columns.map((column) => (
                    <td key={column.id} className={joinClasses("px-3 py-3 align-middle text-app-foreground", column.className)}>
                      {column.accessor(row)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
