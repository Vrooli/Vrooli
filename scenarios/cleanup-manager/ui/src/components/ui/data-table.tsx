/**
 * @vrooliComponentSource react-component-library:DataTable
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption template:react-vite:data-table
 * @vrooliComponentAppliedAt 2026-07-15T03:05:52Z
 * @vrooliComponentSourceSha256 8f743927ac49c317d429a0e74f856ea6ede6746b87c75d5d8a55b7c2703fad1f
 * @vrooliComponentDriftHash 8f743927ac49c317d429a0e74f856ea6ede6746b87c75d5d8a55b7c2703fad1f
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { ArrowDown, ArrowUp, ChevronsUpDown, Search } from "lucide-react";
import { isValidElement, type ReactNode, useMemo, useState } from "react";

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
  filterLabel?: string;
  filters?: Array<DataTableFilter<Row>>;
  className?: string;
  // Applied to the rendered <table> as data-testid. A generic test hook that
  // adopters use to target the table without depending on markup structure.
  tableTestId?: string;
  // aria-label for the filter button group. Overrides filterLabel when set.
  // Restored in 1.1.2 after 1.1.x dropped it (adopter a11y regression).
  filterGroupLabel?: string;
  // Formats the aria-label of each sortable column's sort button. Restored in
  // 1.1.2 after 1.1.x dropped it (adopter a11y regression).
  sortLabel?: (header: string) => string;
}

type SortDirection = "asc" | "desc";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

// searchableText recursively extracts the visible text from a rendered accessor
// node so a column with no explicit searchValue is still searchable by its
// content. Restored in 1.1.1 after 1.1.0 dropped the fallback entirely.
const searchableText = (value: ReactNode): string => {
  if (value === null || value === undefined || typeof value === "boolean") {
    return "";
  }
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.map(searchableText).join(" ");
  }
  if (isValidElement<{ children?: ReactNode }>(value)) {
    return searchableText(value.props.children);
  }
  return "";
};

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
  filterLabel = "Table filters",
  filters = [],
  className,
  tableTestId,
  filterGroupLabel,
  sortLabel = (header) => `Sort by ${header}`,
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
        const value = column.searchValue ? column.searchValue(row) : searchableText(column.accessor(row));
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
    <div className={cn("min-w-0 rounded-panel border border-app-border bg-app-surface", className)}>
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
          <div className="flex flex-wrap gap-2" role="group" aria-label={filterGroupLabel ?? filterLabel}>
            {filters.map((filter) => (
              <button
                key={filter.id}
                type="button"
                className={cn(
                  "min-h-11 rounded-control border px-3 text-sm font-medium transition",
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
        <table data-testid={tableTestId} className="w-full table-fixed border-collapse text-left text-sm">
          <caption className="sr-only">{caption}</caption>
          <thead className="bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
            <tr>
              {columns.map((column) => {
                const active = sortColumn === column.id;
                const SortIcon = !column.sortValue ? null : active ? (sortDirection === "asc" ? ArrowUp : ArrowDown) : ChevronsUpDown;
                return (
                  <th key={column.id} scope="col" className={cn("px-3 py-3 font-semibold", column.className)}>
                    {column.sortValue ? (
                      <button
                        type="button"
                        aria-label={sortLabel(column.header)}
                        className="inline-flex min-h-11 items-center gap-1 rounded-control text-left hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
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
                    <td key={column.id} className={cn("break-words px-3 py-3 align-middle text-app-foreground", column.className)}>
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