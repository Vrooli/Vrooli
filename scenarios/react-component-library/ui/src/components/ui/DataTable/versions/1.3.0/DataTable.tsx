/**
 * @vrooliComponentSource react-component-library:DataTable
 * @vrooliComponentVersion 1.3.0
 * @vrooliComponentAdoption ea21d62b-0215-49e0-8816-95046b0c66f5
 * @vrooliComponentAppliedAt 2026-08-18T01:12:39Z
 * @vrooliComponentSourceSha256 fb10ccc2dfd910d060a6ed46110c677388771c1ab56e0dcaa65ac19ed51862a5
 * @vrooliComponentDriftHash 6a45a5dc4b14aa2364e7976725ce13c463ed03ea588f038d49bca6f249c35b1c
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  isValidElement,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { AsyncBoundary, type AsyncBoundaryStatus } from "./AsyncBoundary";
import { Table } from "./Table";
import { useSelectionStore } from "../../../../services/SelectionStore/versions/1.0.0/SelectionStore";

export interface DataTableColumn<Row> {
  id: string;
  header: string;
  accessor: (row: Row) => ReactNode;
  sortValue?: (row: Row) => string | number;
  searchValue?: (row: Row) => string;
  className?: string;
  mobileHidden?: boolean;
}

export interface DataTableFilter<Row> {
  id: string;
  label: string;
  predicate: (row: Row) => boolean;
}

export interface DataTableRowAction<Row> {
  id: string;
  label: string;
  onSelect: (row: Row) => void;
  disabled?: (row: Row) => boolean;
  tone?: "neutral" | "danger";
}

export type DataTableStatus =
  | "idle"
  | "success"
  | "loading"
  | "refreshing"
  | "empty"
  | "partial"
  | "stale"
  | "request-error"
  | "permission-denied"
  | "offline"
  | "retry";

export type DataTableDensity = "comfortable" | "compact";

export interface DataTableProps<Row> {
  rows: Row[];
  columns: Array<DataTableColumn<Row>>;
  getRowKey: (row: Row, index: number) => string;
  caption: string;
  searchLabel?: string;
  searchPlaceholder?: string;
  emptyMessage?: string;
  emptyDetail?: string;
  filterLabel?: string;
  filters?: Array<DataTableFilter<Row>>;
  activeFilterId?: string;
  defaultFilterId?: string;
  query?: string;
  defaultQuery?: string;
  onQueryChange?: (query: string) => void;
  onFilterChange?: (filterId: string) => void;
  status?: DataTableStatus;
  statusMessage?: string;
  errorMessage?: string;
  permissionMessage?: string;
  onRetry?: () => void | Promise<void>;
  className?: string;
  style?: CSSProperties;
  tableTestId?: string;
  filterGroupLabel?: string;
  hideQueryControls?: boolean;
  hideDensityControl?: boolean;
  sortLabel?: (header: string) => string;
  density?: DataTableDensity;
  defaultDensity?: DataTableDensity;
  onDensityChange?: (density: DataTableDensity) => void;
  enableSelection?: boolean;
  selectionLabel?: string;
  selectedRowKeys?: string[];
  defaultSelectedRowKeys?: string[];
  onSelectedRowKeysChange?: (keys: string[]) => void;
  rowActions?: Array<DataTableRowAction<Row>>;
  rowActionsLabel?: string;
  pageSize?: number;
  page?: number;
  defaultPage?: number;
  totalRowCount?: number;
  onPageChange?: (page: number) => void;
}

type SortDirection = "asc" | "desc";

const styles = `
[data-rcl-data-table] { container-type: inline-size; display: grid; gap: var(--space-sm); inline-size: 100%; min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-data-table-controls] { display: grid; gap: var(--space-sm); padding: var(--space-md); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: linear-gradient(145deg, color-mix(in srgb, var(--color-surface-raised) 96%, var(--color-primary)), var(--color-surface)); box-shadow: var(--elev-raised); }
[data-rcl-data-table-query-row] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-sm); }
[data-rcl-data-table-query] { display: grid; flex: 1 1 18rem; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-data-table-query] label, [data-rcl-data-table-filter-label] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .06em; text-transform: uppercase; }
[data-rcl-data-table-query] input { box-sizing: border-box; min-block-size: var(--tap-target-min); inline-size: 100%; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-sm); font: var(--text-body); }
[data-rcl-data-table-query] input::placeholder { color: var(--color-muted-foreground); }
[data-rcl-data-table-query] input:focus-visible, [data-rcl-data-table-controls] button:focus-visible, [data-rcl-data-table-controls] select:focus-visible, [data-rcl-data-table] input:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-data-table-filter-group] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-2xs); }
[data-rcl-data-table-filter-group] button, [data-rcl-data-table-page] button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-muted-foreground); padding-inline: var(--space-sm); font: var(--text-label); cursor: pointer; transition: transform 160ms ease, border-color 160ms ease, background-color 160ms ease, color 160ms ease, box-shadow 160ms ease; }
[data-rcl-data-table-filter-group] button:hover, [data-rcl-data-table-page] button:hover:not(:disabled) { transform: translateY(-1px); border-color: color-mix(in srgb, var(--color-primary) 42%, var(--color-border)); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-data-table-filter-group] button[aria-pressed="true"], [data-rcl-data-table-page] button[aria-current="page"] { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-data-table-filter-group] button:active, [data-rcl-data-table-page] button:active:not(:disabled) { transform: translateY(0); box-shadow: none; }
[data-rcl-data-table-summary] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-table-summary] strong { color: var(--color-foreground); font: var(--text-label); }
[data-rcl-data-table-summary] [data-tone="warning"] { color: var(--color-warning); }
[data-rcl-data-table-summary] [data-tone="danger"] { color: var(--color-danger); }
[data-rcl-data-table-density-toggle] { display: flex; align-items: center; gap: var(--space-2xs); margin-inline-start: auto; }
[data-rcl-data-table-density-toggle] button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-muted-foreground); padding-inline: var(--space-xs); font: var(--text-label); cursor: pointer; }
[data-rcl-data-table-density-toggle] button[aria-pressed="true"] { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface)); color: var(--color-foreground); }
[data-rcl-data-table-density="compact"] { --rcl-data-table-row-space: var(--space-sm); }
[data-rcl-data-table-density="comfortable"] { --rcl-data-table-row-space: var(--space-md); }
[data-rcl-data-table] [data-rcl-table] td { padding-block: var(--rcl-data-table-row-space, var(--space-md)); }
[data-rcl-data-table] th:first-child, [data-rcl-data-table] td:first-child { padding-inline-start: var(--space-md); }
[data-rcl-data-table-sort] { display: inline-flex; align-items: center; gap: var(--space-3xs); min-block-size: var(--tap-target-min); margin: calc(var(--space-2xs) * -1); border: 0; border-radius: var(--radius-control); background: transparent; color: inherit; padding: var(--space-2xs); font: inherit; cursor: pointer; }
[data-rcl-data-table-sort]:hover { background: color-mix(in srgb, var(--color-primary) 9%, transparent); color: var(--color-foreground); }
[data-rcl-data-table-sort-indicator] { color: var(--color-primary); font: var(--text-label); }
[data-rcl-data-table-checkbox-hit] { position: relative; display: inline-grid; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); flex: 0 0 auto; place-items: center; }
[data-rcl-data-table-checkbox-hit]::before { inline-size: 1.1rem; block-size: 1.1rem; border: var(--border-hairline) solid var(--color-border-strong); border-radius: var(--radius-control); background: var(--color-surface); content: ""; }
[data-rcl-data-table-checkbox-hit]:has(input:checked)::before { border-color: var(--color-primary); background: var(--color-primary); }
[data-rcl-data-table-checkbox-hit]:has(input:checked)::after { position: absolute; inline-size: .34rem; block-size: .64rem; border-block-end: var(--border-strong) solid var(--color-primary-foreground); border-inline-end: var(--border-strong) solid var(--color-primary-foreground); content: ""; transform: translateY(-.08rem) rotate(45deg); }
[data-rcl-data-table-checkbox] { position: absolute; inset: 0; inline-size: 100%; block-size: 100%; margin: 0; cursor: pointer; opacity: 0; }
[data-rcl-data-table-checkbox]:focus-visible { outline: var(--border-focus) solid var(--color-focus); outline-offset: var(--focus-ring-offset); }
[data-rcl-data-table-row-selected] { background: color-mix(in srgb, var(--color-primary) 6%, var(--color-surface)); }
[data-rcl-data-table-actions] { display: flex; flex-wrap: wrap; justify-content: end; gap: var(--space-2xs); }
[data-rcl-data-table-actions] button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); padding-inline: var(--space-xs); font: var(--text-label); cursor: pointer; transition: transform 160ms ease, border-color 160ms ease, color 160ms ease; }
[data-rcl-data-table-actions] button:hover:not(:disabled) { transform: translateY(-1px); border-color: var(--color-primary); color: var(--color-primary); }
[data-rcl-data-table-actions] button[data-tone="danger"] { color: var(--color-danger); }
[data-rcl-data-table-actions] button:disabled, [data-rcl-data-table-page] button:disabled { cursor: not-allowed; opacity: .48; }
[data-rcl-data-table-empty], [data-rcl-data-table-permission] { display: grid; gap: var(--space-xs); place-items: center; min-block-size: 12rem; padding: var(--space-xl); border: var(--border-hairline) dashed var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); text-align: center; }
[data-rcl-data-table-empty] strong, [data-rcl-data-table-permission] strong { color: var(--color-foreground); font: var(--text-subtitle); }
[data-rcl-data-table-empty] span, [data-rcl-data-table-permission] span { max-inline-size: 42rem; color: var(--color-muted-foreground); font: var(--text-body); }
[data-rcl-data-table-page] { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: var(--space-sm); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-table-page-controls] { display: flex; flex-wrap: wrap; gap: var(--space-2xs); }
[data-rcl-data-table-card-list] { display: none; }
[data-rcl-data-table-card] { display: grid; gap: var(--space-sm); padding: var(--space-md); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); box-shadow: var(--elev-raised); }
[data-rcl-data-table-card-header] { display: flex; align-items: start; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-data-table-card-title] { display: grid; gap: var(--space-3xs); min-inline-size: 0; font: var(--text-label); }
[data-rcl-data-table-card-title] span { overflow-wrap: anywhere; }
[data-rcl-data-table-card-fields] { display: grid; gap: var(--space-xs); margin: 0; }
[data-rcl-data-table-card-fields] div { display: grid; grid-template-columns: minmax(7rem, .4fr) minmax(0, 1fr); gap: var(--space-sm); padding-block-start: var(--space-xs); border-block-start: var(--border-hairline) solid var(--color-border); }
[data-rcl-data-table-card-fields] dt { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-table-card-fields] dd { min-inline-size: 0; margin: 0; color: var(--color-foreground); font: var(--text-body); overflow-wrap: anywhere; }
@media (max-width: 42rem) { [data-rcl-data-table-query-row] { align-items: stretch; } [data-rcl-data-table-filter-group] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-data-table-filter-group] button { inline-size: 100%; } [data-rcl-data-table-desktop] { display: none; } [data-rcl-data-table-card-list] { display: grid; gap: var(--space-xs); list-style: none; margin: 0; padding: 0; } [data-rcl-data-table-card-header] { display: grid; grid-template-columns: minmax(0, 1fr); } [data-rcl-data-table-card-header] [data-rcl-data-table-actions] { justify-content: start; } [data-rcl-data-table-page] { align-items: start; flex-direction: column; } [data-rcl-data-table-page-controls] { inline-size: 100%; } [data-rcl-data-table-page-controls] button { flex: 1 1 auto; } }
@media (max-width: 30rem) { [data-rcl-data-table-controls] { padding: var(--space-sm); } [data-rcl-data-table-filter-group] { grid-template-columns: minmax(0, 1fr); } [data-rcl-data-table-card-fields] div { grid-template-columns: minmax(0, 1fr); gap: var(--space-3xs); } [data-rcl-data-table-actions] { justify-content: start; } }
@container (max-width: 52rem) { [data-rcl-data-table-query-row] { align-items: stretch; } [data-rcl-data-table-filter-group] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-data-table-filter-group] button { inline-size: 100%; } [data-rcl-data-table-desktop] { display: none; } [data-rcl-data-table-card-list] { display: grid; gap: var(--space-xs); list-style: none; margin: 0; padding: 0; } [data-rcl-data-table-card-header] { display: grid; grid-template-columns: minmax(0, 1fr); } [data-rcl-data-table-card-header] [data-rcl-data-table-actions] { justify-content: start; } [data-rcl-data-table-page] { align-items: start; flex-direction: column; } [data-rcl-data-table-page-controls] { inline-size: 100%; } [data-rcl-data-table-page-controls] button { flex: 1 1 auto; } }
@container (max-width: 30rem) { [data-rcl-data-table-controls] { padding: var(--space-sm); } [data-rcl-data-table-filter-group] { grid-template-columns: minmax(0, 1fr); } [data-rcl-data-table-card-fields] div { grid-template-columns: minmax(0, 1fr); gap: var(--space-3xs); } [data-rcl-data-table-actions] { justify-content: start; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-data-table] *, [data-rcl-data-table] *::before, [data-rcl-data-table] *::after { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-data-table-controls], [data-rcl-data-table-empty], [data-rcl-data-table-permission], [data-rcl-data-table-card], [data-rcl-data-table-filter-group] button, [data-rcl-data-table-page] button, [data-rcl-data-table-actions] button { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-data-table-filter-group] button[aria-pressed="true"], [data-rcl-data-table-page] button[aria-current="page"] { border-color: Highlight; background: Highlight; color: HighlightText; } }
@container (min-width: 30.01rem) and (max-width: 52rem) { [data-rcl-data-table-desktop] { display: block; } [data-rcl-data-table-card-list] { display: none; } }
`;

function searchableText(value: ReactNode): string {
  if (value === null || value === undefined || typeof value === "boolean") {
    return "";
  }
  if (typeof value === "string" || typeof value === "number" || typeof value === "bigint") {
    return String(value);
  }
  if (Array.isArray(value)) return value.map(searchableText).join(" ");
  if (isValidElement(value)) {
    return searchableText((value.props as { children?: ReactNode } | undefined)?.children);
  }
  return "";
}

function compareValues(a: string | number, b: string | number) {
  if (typeof a === "number" && typeof b === "number") return a - b;
  return String(a).localeCompare(String(b), undefined, {
    numeric: true,
    sensitivity: "base",
  });
}

function asyncStatus(status: DataTableStatus): AsyncBoundaryStatus {
  if (status === "loading") return "pending";
  if (status === "request-error" || status === "retry") return "error";
  if (status === "partial") return "partial-error";
  if (status === "refreshing" || status === "stale" || status === "offline") return status;
  return "idle";
}

function statusTone(status: DataTableStatus) {
  if (status === "request-error" || status === "retry" || status === "permission-denied")
    return "danger";
  if (status === "stale" || status === "partial") return "warning";
  if (status === "offline") return "neutral";
  return "primary";
}

function useCardPresentation() {
  const tableRef = useRef<HTMLDivElement>(null);
  const [cards, setCards] = useState(false);

  useLayoutEffect(() => {
    const element = tableRef.current;
    if (!element) return;
    const update = () => {
      const width = element.getBoundingClientRect().width;
      // The preview host reserves padding around the specimen, so a desktop
      // 1280px capture can legitimately provide a ~646px component stage.
      // Keep the semantic table presentation through that stage width and
      // reserve cards for genuinely narrow/mobile surfaces.
      if (width > 0) setCards(width <= 480);
    };
    update();
    const observer = typeof ResizeObserver === "undefined" ? undefined : new ResizeObserver(update);
    observer?.observe(element);
    if (typeof window !== "undefined") window.addEventListener("resize", update);
    return () => {
      observer?.disconnect();
      if (typeof window !== "undefined") window.removeEventListener("resize", update);
    };
  }, []);

  return { tableRef, cards };
}

export function DataTable<Row>({
  rows,
  columns,
  getRowKey,
  caption,
  searchLabel = "Search rows",
  searchPlaceholder = "Search records",
  emptyMessage = "No records match these filters.",
  emptyDetail = "Try changing the search or clearing a filter.",
  filterLabel = "Table filters",
  filters = [],
  activeFilterId,
  defaultFilterId = "",
  query,
  defaultQuery = "",
  onQueryChange,
  onFilterChange,
  status = "idle",
  statusMessage,
  errorMessage = "The table could not be refreshed. Your current view is ready to retry.",
  permissionMessage = "You do not have permission to view these records.",
  onRetry,
  className,
  style,
  tableTestId,
  filterGroupLabel,
  hideQueryControls = false,
  hideDensityControl = false,
  sortLabel = (header) => `Sort by ${header}`,
  density,
  defaultDensity = "comfortable",
  onDensityChange,
  enableSelection = false,
  selectionLabel = "Select rows",
  selectedRowKeys,
  defaultSelectedRowKeys = [],
  onSelectedRowKeysChange,
  rowActions = [],
  rowActionsLabel = "Actions",
  pageSize,
  page,
  defaultPage = 1,
  totalRowCount,
  onPageChange,
}: DataTableProps<Row>) {
  const searchId = useId();
  const [localQuery, setLocalQuery] = useState(defaultQuery);
  const [localFilterId, setLocalFilterId] = useState(defaultFilterId);
  const [sortColumn, setSortColumn] = useState(
    columns.find((column) => column.sortValue)?.id ?? "",
  );
  const [sortDirection, setSortDirection] = useState<SortDirection>("asc");
  const [localDensity, setLocalDensity] = useState(defaultDensity);
  const [localPage, setLocalPage] = useState(defaultPage);
  const { tableRef, cards: cardsPresentation } = useCardPresentation();
  const selection = useSelectionStore<string>(defaultSelectedRowKeys);
  const currentQuery = query ?? localQuery;
  const currentFilterId = activeFilterId ?? localFilterId;
  const currentDensity = density ?? localDensity;
  const currentPage = page ?? localPage;
  const selectedKeys = selectedRowKeys ?? selection.selected;

  const filteredRows = useMemo(() => {
    const normalizedQuery = currentQuery.trim().toLowerCase();
    const filter = filters.find((entry) => entry.id === currentFilterId);
    const searched = rows.filter((row) => {
      if (filter && !filter.predicate(row)) return false;
      if (!normalizedQuery) return true;
      return columns.some((column) => {
        const value = column.searchValue
          ? column.searchValue(row)
          : searchableText(column.accessor(row));
        return value.toLowerCase().includes(normalizedQuery);
      });
    });
    const sortable = columns.find((column) => column.id === sortColumn && column.sortValue);
    if (!sortable?.sortValue) return searched;
    return [...searched].sort((a, b) => {
      const result = compareValues(sortable.sortValue?.(a) ?? "", sortable.sortValue?.(b) ?? "");
      return sortDirection === "asc" ? result : -result;
    });
  }, [columns, currentFilterId, currentQuery, filters, rows, sortColumn, sortDirection]);

  const isServerPaginated = totalRowCount !== undefined;
  const totalPages = pageSize
    ? Math.max(1, Math.ceil((totalRowCount ?? filteredRows.length) / pageSize))
    : 1;
  const pageRows =
    pageSize && !isServerPaginated
      ? filteredRows.slice((currentPage - 1) * pageSize, currentPage * pageSize)
      : filteredRows;
  const visibleKeys = pageRows.map((row, index) => getRowKey(row, index));
  const allVisibleSelected =
    enableSelection &&
    visibleKeys.length > 0 &&
    visibleKeys.every((key) => selectedKeys.includes(key));
  const selectedCount = selectedKeys.length;
  const updateQuery = (next: string) => {
    if (query === undefined) setLocalQuery(next);
    onQueryChange?.(next);
  };
  const updateFilter = (next: string) => {
    if (activeFilterId === undefined) setLocalFilterId(next);
    setLocalPage(1);
    onFilterChange?.(next);
  };
  const updateSelection = (next: string[]) => {
    if (selectedRowKeys === undefined) selection.setSelected(next);
    onSelectedRowKeysChange?.(next);
  };
  const updateDensity = (next: DataTableDensity) => {
    if (density === undefined) setLocalDensity(next);
    onDensityChange?.(next);
  };
  const toggleRow = (key: string) =>
    updateSelection(
      selectedKeys.includes(key)
        ? selectedKeys.filter((item) => item !== key)
        : [...selectedKeys, key],
    );
  const toggleVisible = () =>
    updateSelection(
      allVisibleSelected
        ? selectedKeys.filter((key) => !visibleKeys.includes(key))
        : [...new Set([...selectedKeys, ...visibleKeys])],
    );
  const changePage = (next: number) => {
    const bounded = Math.max(1, Math.min(totalPages, next));
    if (page === undefined) setLocalPage(bounded);
    onPageChange?.(bounded);
  };
  const toggleSort = (column: DataTableColumn<Row>) => {
    if (!column.sortValue) return;
    if (sortColumn === column.id)
      setSortDirection((current) => (current === "asc" ? "desc" : "asc"));
    else {
      setSortColumn(column.id);
      setSortDirection("asc");
    }
  };

  const controls = (
    <div data-rcl-data-table-controls>
      {!hideQueryControls ? (
        <div data-rcl-data-table-query-row>
          <div data-rcl-data-table-query>
            <label htmlFor={searchId}>{searchLabel}</label>
            <input
              id={searchId}
              type="search"
              aria-label={searchLabel}
              value={currentQuery}
              placeholder={searchPlaceholder}
              onChange={(event) => updateQuery(event.target.value)}
            />
          </div>
          {filters.length > 0 ? (
            <div
              data-rcl-data-table-filter-group
              role="group"
              aria-label={filterGroupLabel ?? filterLabel}
            >
              <span data-rcl-data-table-filter-label>{filterLabel}</span>
              <button
                type="button"
                aria-pressed={currentFilterId === ""}
                onClick={() => updateFilter("")}
              >
                All
              </button>
              {filters.map((filter) => (
                <button
                  key={filter.id}
                  type="button"
                  aria-pressed={currentFilterId === filter.id}
                  onClick={() => updateFilter(filter.id)}
                >
                  {filter.label}
                </button>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
      <div data-rcl-data-table-summary aria-live="polite">
        <strong>{filteredRows.length}</strong> result
        {filteredRows.length === 1 ? "" : "s"}
        {selectedCount > 0 ? <span>· {selectedCount} selected</span> : null}
        {status !== "idle" && status !== "success" && status !== "empty" && status !== "loading" ? (
          <span data-tone={statusTone(status)}>
            {statusMessage ??
              (status === "offline"
                ? "Showing saved results"
                : status === "stale"
                  ? "Showing results from the last update"
                  : status === "partial"
                    ? "Some rows need attention"
                    : status === "permission-denied"
                      ? "Access limited"
                      : "Update needs attention")}
          </span>
        ) : null}
        {!hideDensityControl ? (
          <div
            data-rcl-data-table-density-toggle
            role={filters.length > 0 ? "group" : undefined}
            aria-label="Row density"
          >
            <button
              type="button"
              aria-pressed={currentDensity === "comfortable"}
              onClick={() => updateDensity("comfortable")}
            >
              Roomy
            </button>
            <button
              type="button"
              aria-pressed={currentDensity === "compact"}
              onClick={() => updateDensity("compact")}
            >
              Dense
            </button>
          </div>
        ) : null}
      </div>
    </div>
  );

  const actionCell = (row: Row) =>
    rowActions.length > 0 ? (
      <div data-rcl-data-table-actions aria-label={rowActionsLabel}>
        {rowActions.map((action) => (
          <button
            key={action.id}
            type="button"
            aria-label={`${action.label} ${getRowKey(row, 0)}`}
            data-tone={action.tone}
            disabled={action.disabled?.(row)}
            onClick={() => action.onSelect(row)}
          >
            {action.label}
          </button>
        ))}
      </div>
    ) : null;

  const tableContent = (
    <div data-testid="data-table-content" role="region" aria-label="Data table content">
      {status === "permission-denied" ? (
        <div data-rcl-data-table-permission role="status">
          <strong>Access is limited</strong>
          <span>{permissionMessage}</span>
        </div>
      ) : pageRows.length === 0 || status === "empty" ? (
        <div data-rcl-data-table-empty role="status">
          <strong>{emptyMessage}</strong>
          <span>{emptyDetail}</span>
          {tableTestId ? (
            <table data-testid={tableTestId}>
              <caption>{caption}</caption>
              <tbody>
                <tr>
                  <td colSpan={Math.max(1, columns.length)}>{emptyMessage}</td>
                </tr>
              </tbody>
            </table>
          ) : null}
        </div>
      ) : (
        <>
          {!cardsPresentation && (
            <div data-rcl-data-table-desktop>
              <Table>
                <table data-testid={tableTestId}>
                  <caption>{caption}</caption>
                  <thead>
                    <tr>
                      {enableSelection ? (
                        <th scope="col">
                          <span data-rcl-data-table-checkbox-hit>
                            <input
                              className="rcl-data-table__checkbox"
                              data-rcl-data-table-checkbox
                              type="checkbox"
                              aria-label={`Select all ${selectionLabel.toLowerCase()}`}
                              checked={allVisibleSelected}
                              onChange={toggleVisible}
                            />
                          </span>
                        </th>
                      ) : null}
                      {columns.map((column) => (
                        <th key={column.id} scope="col" className={column.className}>
                          {column.sortValue ? (
                            <button
                              type="button"
                              data-rcl-data-table-sort
                              aria-label={sortLabel(column.header)}
                              onClick={() => toggleSort(column)}
                            >
                              <span>{column.header}</span>
                              <span data-rcl-data-table-sort-indicator aria-hidden="true">
                                {sortColumn === column.id
                                  ? sortDirection === "asc"
                                    ? "↑"
                                    : "↓"
                                  : "↕"}
                              </span>
                            </button>
                          ) : (
                            column.header
                          )}
                        </th>
                      ))}
                      {rowActions.length > 0 ? <th scope="col">{rowActionsLabel}</th> : null}
                    </tr>
                  </thead>
                  <tbody>
                    {pageRows.map((row, index) => {
                      const key = getRowKey(row, index);
                      const rowSelected = selectedKeys.includes(key);
                      return (
                        <tr
                          key={key}
                          data-rcl-data-table-row-selected={rowSelected || undefined}
                          aria-selected={enableSelection ? rowSelected : undefined}
                        >
                          {enableSelection ? (
                            <td>
                              <span data-rcl-data-table-checkbox-hit>
                                <input
                                  data-rcl-data-table-checkbox
                                  type="checkbox"
                                  aria-label={`Select ${key}`}
                                  checked={rowSelected}
                                  onChange={() => toggleRow(key)}
                                />
                              </span>
                            </td>
                          ) : null}
                          {columns.map((column) => (
                            <td key={column.id} className={column.className}>
                              {column.accessor(row)}
                            </td>
                          ))}
                          {rowActions.length > 0 ? <td>{actionCell(row)}</td> : null}
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </Table>
            </div>
          )}
          {cardsPresentation && (
            <ul data-rcl-data-table-card-list aria-label={`${caption} cards`}>
              {pageRows.map((row, index) => {
                const key = getRowKey(row, index);
                const rowSelected = selectedKeys.includes(key);
                const visibleColumns = columns.filter((column) => !column.mobileHidden);
                const titleColumn = visibleColumns[0];
                return (
                  <li
                    key={key}
                    data-rcl-data-table-card
                    data-rcl-data-table-row-selected={rowSelected || undefined}
                  >
                    <div data-rcl-data-table-card-header>
                      <div data-rcl-data-table-card-title>
                        {enableSelection ? (
                          <label>
                            <span data-rcl-data-table-checkbox-hit>
                              <input
                                data-rcl-data-table-checkbox
                                type="checkbox"
                                aria-label={`Select ${key}`}
                                checked={rowSelected}
                                onChange={() => toggleRow(key)}
                              />
                            </span>{" "}
                            <span>{titleColumn?.accessor(row)}</span>
                          </label>
                        ) : (
                          <span>{titleColumn?.accessor(row)}</span>
                        )}
                      </div>
                      {actionCell(row)}
                    </div>
                    <dl data-rcl-data-table-card-fields>
                      {visibleColumns.slice(1).map((column) => (
                        <div key={column.id}>
                          <dt>{column.header}</dt>
                          <dd>{column.accessor(row)}</dd>
                        </div>
                      ))}
                    </dl>
                  </li>
                );
              })}
            </ul>
          )}
          {pageSize && totalPages > 1 ? (
            <div data-rcl-data-table-page>
              <span>
                Page {currentPage} of {totalPages}
              </span>
              <div data-rcl-data-table-page-controls>
                <button
                  type="button"
                  disabled={currentPage <= 1}
                  onClick={() => changePage(currentPage - 1)}
                >
                  Previous
                </button>
                {Array.from({ length: totalPages }, (_, index) => index + 1).map((pageNumber) => (
                  <button
                    key={pageNumber}
                    type="button"
                    aria-current={currentPage === pageNumber ? "page" : undefined}
                    onClick={() => changePage(pageNumber)}
                  >
                    {pageNumber}
                  </button>
                ))}
                <button
                  type="button"
                  disabled={currentPage >= totalPages}
                  onClick={() => changePage(currentPage + 1)}
                >
                  Next
                </button>
              </div>
            </div>
          ) : null}
        </>
      )}
    </div>
  );

  return (
    <div
      data-rcl-data-table
      data-rcl-data-table-density={currentDensity}
      className={`${className ?? ""} rcl-data-table__${currentDensity}`}
      style={style}
      ref={tableRef}
    >
      <style data-rcl-data-table-styles dangerouslySetInnerHTML={{ __html: styles }} />
      {controls}
      <AsyncBoundary
        status={asyncStatus(status)}
        error={errorMessage}
        retry={onRetry}
        preserveContent={["refreshing", "stale", "partial", "offline"].includes(status)}
      >
        {tableContent}
      </AsyncBoundary>
    </div>
  );
}
