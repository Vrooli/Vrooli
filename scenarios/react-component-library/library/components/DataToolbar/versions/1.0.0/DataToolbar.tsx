/** @vrooliComponentSource data-display.data-toolbar */
import { useId, useState, type CSSProperties } from "react";
import {
  FilterBar,
  type FilterOption,
} from "../../../FilterBar/versions/1.0.0/FilterBar";
import {
  Toolbar,
  type ToolbarItem,
} from "../../../Toolbar/versions/1.0.0/Toolbar";

export interface DataToolbarView {
  id: string;
  label: string;
  count?: number;
}

export interface DataToolbarSortOption {
  id: string;
  label: string;
}

export type DataToolbarDensity = "comfortable" | "compact";
export type DataToolbarStatus = "idle" | "refreshing" | "stale" | "offline";

export interface DataToolbarProps {
  query?: string;
  defaultQuery?: string;
  filterOptions?: FilterOption[];
  activeFilterIds?: string[];
  defaultActiveFilterIds?: string[];
  views?: DataToolbarView[];
  activeViewId?: string;
  defaultViewId?: string;
  sortOptions?: DataToolbarSortOption[];
  sortId?: string;
  defaultSortId?: string;
  density?: DataToolbarDensity;
  defaultDensity?: DataToolbarDensity;
  status?: DataToolbarStatus;
  statusMessage?: string;
  lastUpdatedLabel?: string;
  queryLabel?: string;
  applyLabel?: string;
  resetLabel?: string;
  refreshLabel?: string;
  exportLabel?: string;
  columnsLabel?: string;
  onQueryChange?: (query: string) => void;
  onFilterChange?: (ids: string[]) => void;
  onApply?: (state: { query: string; activeFilterIds: string[] }) => void;
  onReset?: () => void;
  onViewChange?: (id: string) => void;
  onSortChange?: (id: string) => void;
  onDensityChange?: (density: DataToolbarDensity) => void;
  onRefresh?: () => void | Promise<void>;
  onExport?: () => void;
  onColumns?: () => void;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-data-toolbar] { display: grid; gap: var(--space-sm); min-inline-size: 0; }
[data-rcl-data-toolbar-filter] { min-inline-size: 0; }
[data-rcl-data-toolbar-actions] { display: grid; gap: var(--space-sm); min-inline-size: 0; padding: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); box-shadow: var(--elev-raised); }
[data-rcl-data-toolbar-context] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-xs); min-inline-size: 0; color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-toolbar-context] strong { color: var(--color-foreground); font: var(--text-label); }
[data-rcl-data-toolbar-status] { display: inline-flex; align-items: center; gap: var(--space-3xs); min-block-size: var(--tap-target-min); padding-inline: var(--space-xs); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface)); }
[data-rcl-data-toolbar-status][data-tone="warning"] { color: var(--color-warning); }
[data-rcl-data-toolbar-status][data-tone="danger"] { color: var(--color-danger); }
[data-rcl-data-toolbar-views] { display: flex; align-items: center; flex-wrap: wrap; gap: var(--space-3xs); min-inline-size: 0; }
[data-rcl-data-toolbar-views] button { min-block-size: var(--tap-target-min); padding-inline: var(--space-xs); border: var(--border-hairline) solid transparent; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); font: var(--text-label); cursor: pointer; }
[data-rcl-data-toolbar-views] button:hover, [data-rcl-data-toolbar-views] button[aria-pressed="true"] { border-color: color-mix(in srgb, var(--color-primary) 35%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface)); color: var(--color-foreground); }
[data-rcl-data-toolbar-views] button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
[data-rcl-data-toolbar-view-count] { margin-inline-start: var(--space-3xs); color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-toolbar-controls] { display: flex; align-items: center; justify-content: start; flex-wrap: wrap; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-data-toolbar-controls] [data-control-slot="label"] { overflow: visible; text-overflow: clip; white-space: nowrap; }
[data-rcl-data-toolbar-sort] { display: inline-flex; align-items: center; gap: var(--space-3xs); min-block-size: var(--tap-target-min); padding-inline: var(--space-xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface); color: var(--color-foreground); font: var(--text-label); }
[data-rcl-data-toolbar-sort] label { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-data-toolbar-sort] select { min-block-size: calc(var(--tap-target-min) - var(--space-2xs)); max-inline-size: 12rem; border: 0; background: transparent; color: inherit; font: inherit; }
[data-rcl-data-toolbar-sort] select:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (max-width: 34rem) { [data-rcl-data-toolbar-controls] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); } [data-rcl-data-toolbar-controls] > * { min-inline-size: 0; inline-size: 100%; } [data-rcl-data-toolbar-controls] > [data-rcl-toolbar] { grid-column: 1 / -1; } [data-rcl-data-toolbar-sort] { grid-column: 1 / -1; justify-content: space-between; } [data-rcl-data-toolbar-sort] select { min-inline-size: 0; max-inline-size: 100%; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-data-toolbar] *, [data-rcl-data-toolbar] *::before, [data-rcl-data-toolbar] *::after { animation-duration: 0.01ms; transition-duration: 0.01ms; } }
@media (forced-colors: active) { [data-rcl-data-toolbar-actions], [data-rcl-data-toolbar-status], [data-rcl-data-toolbar-sort], [data-rcl-data-toolbar-views] button { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-data-toolbar-views] button[aria-pressed="true"] { border-color: Highlight; color: Highlight; } }
`;

function statusTone(status: DataToolbarStatus) {
  if (status === "offline") return "danger";
  if (status === "stale") return "warning";
  return "neutral";
}

export function DataToolbar({
  query,
  defaultQuery = "",
  filterOptions = [],
  activeFilterIds,
  defaultActiveFilterIds = [],
  views = [],
  activeViewId,
  defaultViewId,
  sortOptions = [],
  sortId,
  defaultSortId,
  density,
  defaultDensity = "comfortable",
  status = "idle",
  statusMessage,
  lastUpdatedLabel,
  queryLabel = "Filter records",
  applyLabel = "Apply filters",
  resetLabel = "Reset",
  refreshLabel = "Refresh",
  exportLabel = "Export",
  columnsLabel = "Columns",
  onQueryChange,
  onFilterChange,
  onApply,
  onReset,
  onViewChange,
  onSortChange,
  onDensityChange,
  onRefresh,
  onExport,
  onColumns,
  className,
  style,
}: DataToolbarProps) {
  const sortControlId = useId();
  const [localViewId, setLocalViewId] = useState(
    defaultViewId ?? views[0]?.id ?? "",
  );
  const [localSortId, setLocalSortId] = useState(
    defaultSortId ?? sortOptions[0]?.id ?? "",
  );
  const [localDensity, setLocalDensity] = useState(defaultDensity);
  const resolvedViewId = activeViewId ?? localViewId;
  const resolvedSortId = sortId ?? localSortId;
  const resolvedDensity = density ?? localDensity;
  const updateView = (id: string) => {
    if (activeViewId === undefined) setLocalViewId(id);
    onViewChange?.(id);
  };
  const updateSort = (id: string) => {
    if (sortId === undefined) setLocalSortId(id);
    onSortChange?.(id);
  };
  const updateDensity = (next: DataToolbarDensity) => {
    if (density === undefined) setLocalDensity(next);
    onDensityChange?.(next);
  };
  const toolbarItems: ToolbarItem[] = [
    ...(onRefresh
      ? [
          {
            id: "refresh",
            label: status === "refreshing" ? "Refreshing…" : refreshLabel,
            disabled: status === "refreshing",
            onSelect: () => {
              void onRefresh();
            },
          },
        ]
      : []),
    ...(onExport
      ? [{ id: "export", label: exportLabel, onSelect: onExport }]
      : []),
    ...(onColumns
      ? [{ id: "columns", label: columnsLabel, onSelect: onColumns }]
      : []),
    {
      id: "comfortable",
      label: "Comfortable",
      kind: "toggle",
      pressed: resolvedDensity === "comfortable",
      onPressedChange: () => updateDensity("comfortable"),
    },
    {
      id: "compact",
      label: "Compact",
      kind: "toggle",
      pressed: resolvedDensity === "compact",
      onPressedChange: () => updateDensity("compact"),
    },
  ];
  return (
    <section data-rcl-data-toolbar className={className} style={style}>
      <style
        data-rcl-data-toolbar-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <div data-rcl-data-toolbar-filter>
        <FilterBar
          query={query}
          defaultQuery={defaultQuery}
          options={filterOptions}
          activeFilterIds={activeFilterIds}
          defaultActiveFilterIds={defaultActiveFilterIds}
          onQueryChange={onQueryChange}
          onActiveFilterIdsChange={onFilterChange}
          onApply={onApply}
          onReset={onReset}
          queryLabel={queryLabel}
          applyLabel={applyLabel}
          resetLabel={resetLabel}
        />
      </div>
      <div data-rcl-data-toolbar-actions>
        <div data-rcl-data-toolbar-context>
          {views.length > 0 ? (
            <div
              data-rcl-data-toolbar-views
              role="group"
              aria-label="Saved views"
            >
              {views.map((view) => (
                <button
                  key={view.id}
                  type="button"
                  aria-pressed={resolvedViewId === view.id}
                  onClick={() => updateView(view.id)}
                >
                  {view.label}
                  {view.count !== undefined ? (
                    <span data-rcl-data-toolbar-view-count aria-hidden="true">
                      {view.count}
                    </span>
                  ) : null}
                </button>
              ))}
            </div>
          ) : null}
          {status !== "idle" ? (
            <span
              data-rcl-data-toolbar-status
              data-tone={statusTone(status)}
              role="status"
              aria-live="polite"
            >
              {statusMessage ??
                (status === "refreshing"
                  ? "Updating results…"
                  : status === "offline"
                    ? "Offline · showing saved results"
                    : "Showing results from the last update")}
            </span>
          ) : null}
          {lastUpdatedLabel ? <span>{lastUpdatedLabel}</span> : null}
        </div>
        <div data-rcl-data-toolbar-controls>
          {sortOptions.length > 0 ? (
            <span data-rcl-data-toolbar-sort>
              <label htmlFor={sortControlId}>Sort</label>
              <select
                id={sortControlId}
                value={resolvedSortId}
                onChange={(event) => updateSort(event.target.value)}
              >
                {sortOptions.map((option) => (
                  <option key={option.id} value={option.id}>
                    {option.label}
                  </option>
                ))}
              </select>
            </span>
          ) : null}
          {toolbarItems.length > 0 ? (
            <Toolbar
              items={toolbarItems}
              label="Collection actions"
              size="sm"
            />
          ) : null}
        </div>
      </div>
    </section>
  );
}
