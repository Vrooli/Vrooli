/**
 * @libraryId react-component-library:ResourceCollection
 * @displayName Resource Collection
 * @version 1.0.9
 * @tags ["pattern","collection","resources","responsive","recovery","keyboard","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource patterns.resource-collection */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useState, type CSSProperties, type ReactNode } from "react";
import {
  DataTable,
  type DataTableColumn,
  type DataTableDensity,
  type DataTableFilter,
  type DataTableRowAction,
  type DataTableStatus,
} from "@vrooli/react-component-library/DataTable/1";
import {
  DataToolbar,
  type DataToolbarDensity,
  type DataToolbarSortOption,
  type DataToolbarStatus,
  type DataToolbarView,
} from "@vrooli/react-component-library/DataToolbar/1";

export type ResourceCollectionStatus = DataTableStatus | "submitting" | "success";

export interface ResourceCollectionProps<Row> {
  title: string;
  description?: string;
  rows: Row[];
  columns: Array<DataTableColumn<Row>>;
  getRowKey: (row: Row, index: number) => string;
  caption?: string;
  searchLabel?: string;
  emptyMessage?: string;
  emptyDetail?: string;
  filters?: Array<DataTableFilter<Row>>;
  views?: DataToolbarView[];
  sortOptions?: DataToolbarSortOption[];
  defaultSortId?: string;
  query?: string;
  defaultQuery?: string;
  onQueryChange?: (query: string) => void;
  onQueryApply?: (query: string) => void;
  status?: ResourceCollectionStatus;
  statusMessage?: string;
  errorMessage?: string;
  permissionMessage?: string;
  onRetry?: () => void | Promise<void>;
  onRefresh?: () => void | Promise<void>;
  onExport?: () => void;
  onColumns?: () => void;
  onNavigate?: (row: Row) => void;
  rowActions?: Array<DataTableRowAction<Row>>;
  enableSelection?: boolean;
  selectedRowKeys?: string[];
  defaultSelectedRowKeys?: string[];
  onSelectedRowKeysChange?: (keys: string[]) => void;
  density?: DataTableDensity;
  defaultDensity?: DataTableDensity;
  onDensityChange?: (density: DataTableDensity) => void;
  pageSize?: number;
  page?: number;
  defaultPage?: number;
  totalRowCount?: number;
  onPageChange?: (page: number) => void;
  headerAside?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-resource-collection] { container-type: inline-size; display: grid; gap: var(--space-lg); inline-size: 100%; min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-resource-collection-header] { display: flex; align-items: end; justify-content: space-between; gap: var(--space-md); min-inline-size: 0; }
[data-rcl-resource-collection-heading] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-resource-collection-eyebrow] { color: var(--color-primary); font: var(--text-overline); letter-spacing: .1em; text-transform: uppercase; }
[data-rcl-resource-collection-title] { margin: 0; color: var(--color-foreground); font: var(--text-display); overflow-wrap: anywhere; }
[data-rcl-resource-collection-description] { max-inline-size: 68ch; margin: 0; color: var(--color-muted-foreground); font: var(--text-body); }
[data-rcl-resource-collection-aside] { flex: 0 0 auto; }
[data-rcl-resource-collection-status] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs); min-block-size: var(--tap-target-min); padding: var(--space-sm) var(--space-md); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: color-mix(in srgb, var(--color-surface-raised) 92%, var(--color-primary)); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-resource-collection-status] strong { color: var(--color-foreground); font: var(--text-label); }
[data-rcl-resource-collection-status][data-tone="danger"] { border-color: color-mix(in srgb, var(--color-danger) 40%, var(--color-border)); }
[data-rcl-resource-collection-status][data-tone="warning"] { border-color: color-mix(in srgb, var(--color-warning) 40%, var(--color-border)); }
@container (max-width: 38rem) { [data-rcl-resource-collection-header] { align-items: start; flex-direction: column; } [data-rcl-resource-collection-aside] { inline-size: 100%; } }


`;

function toolbarStatus(status: ResourceCollectionStatus): DataToolbarStatus {
  if (status === "offline") return "offline";
  if (status === "stale") return "stale";
  if (status === "refreshing") return "refreshing";
  return "idle";
}

function tableStatus(status: ResourceCollectionStatus): DataTableStatus {
  if (status === "submitting") return "refreshing";
  if (status === "success") return "success";
  return status;
}

function statusCopy(status: ResourceCollectionStatus) {
  switch (status) {
    case "loading":
      return "Preparing resources";
    case "refreshing":
      return "Checking for changes";
    case "stale":
      return "Showing the last known resources";
    case "partial":
      return "Some resources need attention";
    case "offline":
      return "Showing saved resources while offline";
    case "submitting":
      return "Saving collection changes";
    case "success":
      return "Collection updated";
    case "request-error":
    case "retry":
      return "Resource update needs attention";
    case "permission-denied":
      return "Access is limited";
    default:
      return "";
  }
}

export const ResourceCollection = withClassName(function ResourceCollection<Row>({
  title,
  description,
  rows,
  columns,
  getRowKey,
  caption = title,
  searchLabel = "Search resources",
  emptyMessage = "No resources match this view",
  emptyDetail = "Clear the search or choose another saved view to see more resources.",
  filters = [],
  views = [],
  sortOptions = [],
  defaultSortId,
  query,
  defaultQuery = "",
  onQueryChange,
  onQueryApply,
  status = "idle",
  statusMessage,
  errorMessage,
  permissionMessage,
  onRetry,
  onRefresh,
  onExport,
  onColumns,
  onNavigate,
  rowActions = [],
  enableSelection = false,
  selectedRowKeys,
  defaultSelectedRowKeys,
  onSelectedRowKeysChange,
  density,
  defaultDensity = "comfortable",
  onDensityChange,
  pageSize,
  page,
  defaultPage,
  totalRowCount,
  onPageChange,
  headerAside,
  className,
  style,
}: ResourceCollectionProps<Row>) {
  const libraryStrings = useStrings();
  const [draftQuery, setDraftQuery] = useState(defaultQuery);
  const [appliedQuery, setAppliedQuery] = useState(defaultQuery);
  const [densityState, setDensityState] = useState<DataToolbarDensity>(defaultDensity);
  const [activeFilterIds, setActiveFilterIds] = useState<string[]>([]);
  const resolvedDraftQuery = query ?? draftQuery;
  const resolvedAppliedQuery = query ?? appliedQuery;
  const resolvedDensity = density ?? densityState;
  const effectiveStatus = tableStatus(status);
  const visibleActions = onNavigate
    ? [
        {
          id: "open",
          label: "Open",
          onSelect: onNavigate,
        },
        ...rowActions,
      ]
    : rowActions;

  const updateQuery = (next: string) => {
    if (query === undefined) setDraftQuery(next);
    onQueryChange?.(next);
  };
  const applyQuery = ({ query: next }: { query: string }) => {
    if (query === undefined) {
      setDraftQuery(next);
      setAppliedQuery(next);
    }
    onQueryApply?.(next);
    onQueryChange?.(next);
  };
  const updateDensity = (next: DataToolbarDensity) => {
    setDensityState(next);
    onDensityChange?.(next);
  };

  return (
    <div
      data-testid="patterns.resource-collection"
      data-rcl-resource-collection
      className={className}
      style={style}
      data-rcl-resource-collection-status={status}
    >
      <StyleSheet name="resourcecollection-1-0-4-1" css={styles} />
      <header data-rcl-resource-collection-header>
        <div data-rcl-resource-collection-heading>
          <span data-rcl-resource-collection-eyebrow>
            {libraryStrings("patterns.resource-collection.resource-library", "Resource library")}
          </span>
          <h1 data-rcl-resource-collection-title>{title}</h1>
          {description ? <p data-rcl-resource-collection-description>{description}</p> : null}
        </div>
        {headerAside ? <div data-rcl-resource-collection-aside>{headerAside}</div> : null}
      </header>
      {status !== "idle" && status !== "success" ? (
        <div
          data-rcl-resource-collection-status
          data-tone={
            status === "request-error" || status === "retry"
              ? "danger"
              : status === "stale" || status === "partial"
                ? "warning"
                : "neutral"
          }
          role={status === "request-error" || status === "retry" ? "alert" : "status"}
          aria-live="polite"
        >
          <strong>{statusCopy(status)}</strong>
          <span>{statusMessage ?? "Your collection context is preserved."}</span>
        </div>
      ) : null}
      <DataToolbar
        query={resolvedDraftQuery}
        filterOptions={filters.map(({ id, label }) => ({ id, label }))}
        activeFilterIds={activeFilterIds}
        views={views}
        sortOptions={sortOptions}
        defaultSortId={defaultSortId}
        defaultDensity={resolvedDensity}
        status={toolbarStatus(status)}
        statusMessage={statusMessage}
        lastUpdatedLabel={status === "offline" ? "Saved recently" : undefined}
        queryLabel="Search resources"
        applyLabel="Apply search"
        onQueryChange={updateQuery}
        onFilterChange={setActiveFilterIds}
        onApply={applyQuery}
        onReset={() => {
          updateQuery("");
          setActiveFilterIds([]);
          if (query === undefined) setAppliedQuery("");
        }}
        onDensityChange={updateDensity}
        onRefresh={onRefresh}
        onExport={onExport}
        onColumns={onColumns}
      />
      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={getRowKey}
        caption={caption}
        searchLabel={searchLabel}
        emptyMessage={emptyMessage}
        emptyDetail={emptyDetail}
        query={resolvedAppliedQuery}
        filters={filters}
        activeFilterId={activeFilterIds[0] ?? ""}
        hideQueryControls
        hideDensityControl
        status={effectiveStatus}
        statusMessage={statusMessage}
        errorMessage={errorMessage}
        permissionMessage={permissionMessage}
        onRetry={onRetry}
        rowActions={visibleActions}
        enableSelection={enableSelection}
        selectedRowKeys={selectedRowKeys}
        defaultSelectedRowKeys={defaultSelectedRowKeys}
        onSelectedRowKeysChange={onSelectedRowKeysChange}
        density={resolvedDensity}
        onDensityChange={onDensityChange}
        pageSize={pageSize}
        page={page}
        defaultPage={defaultPage}
        totalRowCount={totalRowCount}
        onPageChange={onPageChange}
      />
    </div>
  );
});
