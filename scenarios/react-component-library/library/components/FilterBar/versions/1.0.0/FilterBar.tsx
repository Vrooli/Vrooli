/** @vrooliComponentSource react-component-library:FilterBar */
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

const panel = {
  border: "1px solid var(--color-border, #cbd5e1)",
  borderRadius: "var(--radius-panel, .75rem)",
  background: "var(--color-surface, #fff)",
  color: "var(--color-foreground, #0f172a)",
  padding: "var(--space-md, 24px)",
  boxShadow: "var(--elev-raised, 0 1px 3px rgb(15 23 42 / .08))",
};

const muted = { color: "var(--color-muted-foreground, #64748b)" };

const filterBarStyles = `
.rcl-filter-bar__controls { display: flex; align-items: end; flex-wrap: wrap; gap: var(--space-sm, 12px); }
.rcl-filter-bar__query { flex: 1 1 280px; min-width: 0; }
.rcl-filter-bar__actions { display: flex; flex: 0 0 auto; gap: var(--space-2xs, 8px); }
@media (max-width: 480px) {
  .rcl-filter-bar__controls { display: grid; grid-template-columns: minmax(0, 1fr); }
  .rcl-filter-bar__actions { display: grid; grid-template-columns: minmax(0, 1fr); width: 100%; }
}
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
  const [localFilterIds, setLocalFilterIds] = useLocalState(
    defaultActiveFilterIds,
  );
  const resolvedQuery = isQueryControlled ? query : localQuery;
  const resolvedFilterIds = isFiltersControlled
    ? activeFilterIds
    : localFilterIds;

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
        style={panel}
      >
        <div className="rcl-filter-bar__controls">
          <div className="rcl-filter-bar__query">
            <SearchInput
              aria-label={queryLabel}
              placeholder="Search records"
              value={resolvedQuery}
              onChange={(event) => setQuery(event.target.value)}
              style={{ width: "100%" }}
            />
          </div>
          <div className="rcl-filter-bar__actions">
            <button type="submit" style={primaryButton}>
              {applyLabel}
            </button>
            <button type="button" onClick={reset} style={quietButton}>
              {resetLabel}
            </button>
          </div>
        </div>
        {options.length > 0 && (
          <fieldset
            style={{
              border: 0,
              margin: "var(--space-md, 24px) 0 0",
              padding: 0,
            }}
          >
            <legend style={{ ...muted, fontSize: 12, fontWeight: 700 }}>
              Filter by status
            </legend>
            <Cluster gap="xs" aria-label="Available filters" role="group">
              {options.map((option) => (
                <Chip
                  key={option.id}
                  selected={resolvedFilterIds.includes(option.id)}
                  aria-label={
                    option.count === undefined
                      ? option.label
                      : `${option.label} · ${option.count}`
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
        <p
          aria-live="polite"
          style={{
            ...muted,
            margin: "var(--space-sm, 12px) 0 0",
            fontSize: 13,
          }}
        >
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

const primaryButton = {
  minHeight: 44,
  border: 0,
  borderRadius: "var(--radius-control, .5rem)",
  background: "var(--color-primary, #2563eb)",
  color: "var(--color-primary-foreground, #fff)",
  paddingInline: "var(--space-xs, 8px)",
  font: "inherit",
  fontSize: "var(--font-size-sm, 14px)",
  fontWeight: 700,
  whiteSpace: "nowrap",
};

const quietButton = {
  ...primaryButton,
  border: "1px solid var(--color-border, #cbd5e1)",
  background: "transparent",
  color: "var(--color-foreground, #0f172a)",
};
