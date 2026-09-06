/**
 * @libraryId react-component-library:FilterBar
 * @displayName FilterBar
 * @description The query-control surface coordinating search, structured filters, active-filter chips, presets, reset, responsive overflow, and URL synchronization.
 * @version 1.3.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:FilterBar */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useState, type FormEvent } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Chip } from "@vrooli/react-component-library/Chip/1";
import { SearchInput } from "@vrooli/react-component-library/SearchInput/1";
import { Cluster } from "@vrooli/react-component-library/Cluster/1";

export interface FilterOption {
  id: string;
  label: string;
  count?: number;
}

export interface FilterBarProps {
  query?: string;
  defaultQuery?: string;
  onQueryChange?: (value: string) => void;
  options?: FilterOption[];
  activeFilterIds?: string[];
  defaultActiveFilterIds?: string[];
  onActiveFilterIdsChange?: (ids: string[]) => void;
  onApply?: (state: { query: string; activeFilterIds: string[] }) => void;
  onReset?: () => void;
  queryLabel?: string;
  queryPlaceholder?: string;
  /** Compact spacing with side-by-side actions, including narrow viewports. */
  density?: "comfortable" | "compact";
  /** Inline controls for an existing collection surface. */
  presentation?: "panel" | "inline";
  /** Immediate mode applies edits without a redundant submit control. */
  applyMode?: "submit" | "immediate";
  applyLabel?: string;
  resetLabel?: string;
}

const filterBarStyles = `
.rcl-filter-bar { display: grid; gap: var(--space-md); box-sizing: border-box; padding: clamp(var(--space-md), 3vw, var(--space-lg)); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: linear-gradient(145deg, color-mix(in srgb, var(--color-surface-raised) 96%, var(--color-primary)), var(--color-surface)); color: var(--color-foreground); box-shadow: var(--elev-raised); }
.rcl-filter-bar__controls { display: flex; align-items: end; flex-wrap: wrap; gap: var(--space-sm); }
.rcl-filter-bar__query { flex: 1 1 280px; min-width: 0; }
.rcl-filter-bar__actions { display: flex; flex: 0 0 auto; gap: var(--space-2xs); }
.rcl-filter-bar__legend { margin-block-end: var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .06em; text-transform: uppercase; }
.rcl-filter-bar__summary { margin: 0; color: var(--color-muted-foreground); font: var(--text-caption); }
@media (max-width: 480px) {
  .rcl-filter-bar__controls { display: grid; grid-template-columns: minmax(0, 1fr); }
  .rcl-filter-bar__actions { display: grid; grid-template-columns: minmax(0, 1fr); width: 100%; }
  .rcl-filter-bar__button { width: 100%; }
}
.rcl-filter-bar[data-density="compact"] { padding: var(--space-sm, 12px); gap: var(--space-xs, 8px); }
.rcl-filter-bar[data-density="compact"] .rcl-filter-bar__actions { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); }
.rcl-filter-bar[data-density="compact"] .rcl-filter-bar__button { min-inline-size: 0; overflow-wrap: anywhere; white-space: normal; }

.rcl-filter-bar[data-presentation="inline"] { padding: 0; border: 0; border-radius: 0; background: transparent; box-shadow: none; }
.rcl-filter-bar[data-presentation="inline"] .rcl-filter-bar__controls { display: flex; flex-wrap: nowrap; align-items: center; }
.rcl-filter-bar[data-presentation="inline"] .rcl-filter-bar__query { flex: 1 1 0; }
.rcl-filter-bar[data-presentation="inline"] .rcl-filter-bar__actions { display: flex; width: auto; }
.rcl-filter-bar[data-presentation="inline"] [data-rcl-search-input-label] { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
`;

export const FilterBar = withClassName(function FilterBar({
  query,
  defaultQuery = "",
  onQueryChange,
  options = [],
  activeFilterIds,
  defaultActiveFilterIds = [],
  onActiveFilterIdsChange,
  onApply,
  onReset,
  queryLabel = "Filter results",
  queryPlaceholder,
  density = "comfortable",
  presentation = "panel",
  applyMode = "submit",
  applyLabel = "Apply filters",
  resetLabel = "Reset",
}: FilterBarProps) {
  const strings = useStrings();
  const isQueryControlled = query !== undefined;
  const isFiltersControlled = activeFilterIds !== undefined;
  const [localQuery, setLocalQuery] = useLocalState(defaultQuery);
  const [localFilterIds, setLocalFilterIds] = useLocalState(defaultActiveFilterIds);
  const resolvedQuery = isQueryControlled ? query : localQuery;
  const resolvedFilterIds = isFiltersControlled ? activeFilterIds : localFilterIds;

  const setQuery = (value: string) => {
    if (!isQueryControlled) setLocalQuery(value);
    onQueryChange?.(value);
    if (applyMode === "immediate") onApply?.({ query: value, activeFilterIds: resolvedFilterIds });
  };

  const setFilters = (ids: string[]) => {
    if (!isFiltersControlled) setLocalFilterIds(ids);
    onActiveFilterIdsChange?.(ids);
    if (applyMode === "immediate") onApply?.({ query: resolvedQuery, activeFilterIds: ids });
  };

  const toggleFilter = (id: string) => {
    setFilters(
      resolvedFilterIds.includes(id)
        ? resolvedFilterIds.filter((activeId) => activeId !== id)
        : [...resolvedFilterIds, id],
    );
  };

  const reset = () => {
    if (!isQueryControlled) setLocalQuery("");
    if (!isFiltersControlled) setLocalFilterIds([]);
    onQueryChange?.("");
    onActiveFilterIdsChange?.([]);
    if (applyMode === "immediate") onApply?.({ query: "", activeFilterIds: [] });
    onReset?.();
  };

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (applyMode === "submit")
      onApply?.({ query: resolvedQuery, activeFilterIds: resolvedFilterIds });
  };

  return (
    <>
      <StyleSheet name="filterbar-1-2" css={filterBarStyles} />
      <form
        role="search"
        aria-label={queryLabel}
        onSubmit={submit}
        className="rcl-filter-bar"
        data-rcl-filter-bar
        data-density={density}
        data-presentation={presentation}
      >
        <div className="rcl-filter-bar__controls">
          <div className="rcl-filter-bar__query">
            <SearchInput
              aria-label={queryLabel}
              placeholder={
                queryPlaceholder ??
                strings("data-display.filter-bar.search-records", "Search records")
              }
              value={resolvedQuery}
              onChange={(event) => setQuery(event.target.value)}
              style={{ width: "100%" }}
            />
          </div>
          {(applyMode === "submit" || resolvedQuery || resolvedFilterIds.length > 0) && (
            <div className="rcl-filter-bar__actions">
              {applyMode === "submit" && (
                <Button
                  data-testid="data-display.filter-bar"
                  type="submit"
                  className="rcl-filter-bar__button rcl-filter-bar__button--primary"
                >
                  {applyLabel}
                </Button>
              )}
              <Button
                variant="ghost"
                data-testid="data-display.filter-bar"
                type="button"
                className="rcl-filter-bar__button rcl-filter-bar__button--quiet"
                onClick={reset}
              >
                {resetLabel}
              </Button>
            </div>
          )}
        </div>
        {options.length > 0 && (
          <fieldset style={{ border: 0, margin: 0, padding: 0 }}>
            <legend className="rcl-filter-bar__legend">
              {strings("data-display.filter-bar.filter-by-status", "Filter by status")}
            </legend>
            <Cluster
              gap="xs"
              aria-label={strings("data-display.filter-bar.available-filters", "Available filters")}
              role="group"
            >
              {options.map((option) => (
                <Chip
                  key={option.id}
                  selected={resolvedFilterIds.includes(option.id)}
                  aria-label={
                    option.count === undefined ? option.label : `${option.label} · ${option.count}`
                  }
                  onClick={() => toggleFilter(option.id)}
                >
                  {option.label}
                  {option.count !== undefined ? ` · ${option.count}` : ""}
                </Chip>
              ))}
            </Cluster>
          </fieldset>
        )}
        {(options.length > 0 || resolvedFilterIds.length > 0) && (
          <p aria-live="polite" className="rcl-filter-bar__summary">
            {strings("data-display.filter-bar.selected-filter-options", "Selected filter options")}:{" "}
            {resolvedFilterIds.length}
          </p>
        )}
      </form>
    </>
  );
});

function useLocalState<T>(initialValue: T): [T, (next: T) => void] {
  const [value, setValue] = useState(initialValue);
  return [value, setValue];
}
