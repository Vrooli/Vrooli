/**
 * @libraryId react-component-library:FilterBar
 * @displayName FilterBar
 * @description A responsive query surface with search, active-filter chips, reset, and explicit apply semantics.
 * @version 1.1.3
 * @tags ["data-display","form","token-bound","responsive","keyboard"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:FilterBar */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { useState, type FormEvent } from "react";
import { Chip } from "../../../Chip/versions/1.0.0/Chip";
import { SearchInput } from "../../../SearchInput/versions/1.0.0/SearchInput";
import { Cluster } from "../../../../primitives/Cluster/versions/1.0.0/Cluster";

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
  applyLabel?: string;
  resetLabel?: string;
}

const filterBarStyles = `
.rcl-filter-bar { display: grid; gap: var(--space-md); box-sizing: border-box; padding: clamp(var(--space-md), 3vw, var(--space-lg)); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: linear-gradient(145deg, color-mix(in srgb, var(--color-surface-raised) 96%, var(--color-primary)), var(--color-surface)); color: var(--color-foreground); box-shadow: var(--elev-raised); }
.rcl-filter-bar__controls { display: flex; align-items: end; flex-wrap: wrap; gap: var(--space-sm); }
.rcl-filter-bar__query { flex: 1 1 280px; min-width: 0; }
.rcl-filter-bar__actions { display: flex; flex: 0 0 auto; gap: var(--space-2xs); }
.rcl-filter-bar__button { min-block-size: var(--tap-target-min); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); padding-inline: var(--space-sm); font: var(--text-label); cursor: pointer; transition: transform 160ms ease, box-shadow 160ms ease, background-color 160ms ease, border-color 160ms ease; }
.rcl-filter-bar__button:hover { transform: translateY(-1px); box-shadow: var(--elev-raised); }
.rcl-filter-bar__button:active { transform: translateY(0); box-shadow: none; }
.rcl-filter-bar__button:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
.rcl-filter-bar__button--primary { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-primary-foreground); }
.rcl-filter-bar__button--quiet { background: var(--color-surface); color: var(--color-foreground); }
.rcl-filter-bar__legend { margin-block-end: var(--space-2xs); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .06em; text-transform: uppercase; }
.rcl-filter-bar__summary { margin: 0; color: var(--color-muted-foreground); font: var(--text-caption); }
@media (max-width: 480px) {
  .rcl-filter-bar__controls { display: grid; grid-template-columns: minmax(0, 1fr); }
  .rcl-filter-bar__actions { display: grid; grid-template-columns: minmax(0, 1fr); width: 100%; }
  .rcl-filter-bar__button { width: 100%; }
}
@media (prefers-reduced-motion: reduce) { .rcl-filter-bar__button { transition-duration: .01ms; } }
@media (forced-colors: active) { .rcl-filter-bar, .rcl-filter-bar__button { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } .rcl-filter-bar__button--primary { background: Highlight; color: HighlightText; } }
`;

export function FilterBar({
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
  applyLabel = "Apply filters",
  resetLabel = "Reset",
}: FilterBarProps) {
  const isQueryControlled = query !== undefined;
  const isFiltersControlled = activeFilterIds !== undefined;
  const [localQuery, setLocalQuery] = useLocalState(defaultQuery);
  const [localFilterIds, setLocalFilterIds] = useLocalState(defaultActiveFilterIds);
  const resolvedQuery = isQueryControlled ? query : localQuery;
  const resolvedFilterIds = isFiltersControlled ? activeFilterIds : localFilterIds;

  const setQuery = (value: string) => {
    if (!isQueryControlled) setLocalQuery(value);
    onQueryChange?.(value);
  };

  const setFilters = (ids: string[]) => {
    if (!isFiltersControlled) setLocalFilterIds(ids);
    onActiveFilterIdsChange?.(ids);
  };

  const toggleFilter = (id: string) => {
    setFilters(
      resolvedFilterIds.includes(id)
        ? resolvedFilterIds.filter((activeId) => activeId !== id)
        : [...resolvedFilterIds, id],
    );
  };

  const reset = () => {
    setQuery("");
    setFilters([]);
    onReset?.();
  };

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onApply?.({ query: resolvedQuery, activeFilterIds: resolvedFilterIds });
  };

  return (
    <>
      <style data-rcl-filter-bar-styles>{filterBarStyles}</style>
      <form
        role="search"
        aria-label={queryLabel}
        onSubmit={submit}
        className="rcl-filter-bar"
        data-rcl-filter-bar
      >
        <div className="rcl-filter-bar__controls">
          <div className="rcl-filter-bar__query">
            <SearchInput
              aria-label={queryLabel}
              placeholder={translate("data-display.filter-bar.placeholder.1", "Search records")}
              value={resolvedQuery}
              onChange={(event) => setQuery(event.target.value)}
              style={{ width: "100%" }}
            />
          </div>
          <div className="rcl-filter-bar__actions">
            <button
              data-testid="data-display.filter-bar"
              type="submit"
              className="rcl-filter-bar__button rcl-filter-bar__button--primary"
            >
              {applyLabel}
            </button>
            <button
              data-testid="data-display.filter-bar"
              type="button"
              className="rcl-filter-bar__button rcl-filter-bar__button--quiet"
              onClick={reset}
            >
              {resetLabel}
            </button>
          </div>
        </div>
        {options.length > 0 && (
          <fieldset style={{ border: 0, margin: 0, padding: 0 }}>
            <legend className="rcl-filter-bar__legend">
              {translate("data-display.filter-bar.text.1", "Filter by status")}
            </legend>
            <Cluster
              gap="xs"
              aria-label={translate("data-display.filter-bar.aria-label.2", "Available filters")}
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
        <p aria-live="polite" className="rcl-filter-bar__summary">
          {resolvedFilterIds.length === 0
            ? "Showing all results"
            : `${resolvedFilterIds.length} filter${resolvedFilterIds.length === 1 ? "" : "s"} selected`}
        </p>
      </form>
    </>
  );
}

function useLocalState<T>(initialValue: T): [T, (next: T) => void] {
  const [value, setValue] = useState(initialValue);
  return [value, setValue];
}
