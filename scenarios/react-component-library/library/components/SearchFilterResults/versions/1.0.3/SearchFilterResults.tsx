/**
 * @libraryId react-component-library:SearchFilterResults
 * @displayName SearchFilterResults
 * @description A URL-ready search and filter composition that separates no-match emptiness from loading and error states.
 * @version 1.0.3
 * @tags ["pattern","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource patterns.search-filter-results */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { useCallback, useRef, useState } from "react";
import { FilterBar } from "@vrooli/react-component-library/FilterBar/1.0.0";
import {
  SearchResults,
  type SearchResultsState,
} from "@vrooli/react-component-library/SearchResults/1.0.4";
import { useAbortableTask } from "@vrooli/react-component-library/useAbortableTask/1.0.0";

export interface SearchFilterResultsProps {
  query?: string;
  items?: string[];
  state?: SearchResultsState;
  errorMessage?: string;
  onRetry?: () => void;
}

const styles = `
[data-rcl-search-filter-results] { display: grid; gap: var(--space-lg); min-inline-size: 0; }
[data-rcl-search-filter-results-summary] { color: var(--color-muted-foreground); font: var(--text-caption); }
`;

export const SearchFilterResults = withClassName(function SearchFilterResults({
  query = "",
  items = [],
  state: initialState = "default",
  errorMessage,
  onRetry,
}: SearchFilterResultsProps) {
  const [draftQuery, setDraftQuery] = useState(query);
  const [appliedQuery, setAppliedQuery] = useState(query);
  const [state, setState] = useState<SearchResultsState>(initialState);
  const pendingQuery = useRef(query);
  const applyTask = useAbortableTask(
    useCallback(async (signal: AbortSignal) => {
      await new Promise((resolve) => setTimeout(resolve, 160));
      if (!signal.aborted) {
        setAppliedQuery(pendingQuery.current);
        setState("default");
      }
    }, []),
  );

  const apply = ({ query: nextQuery }: { query: string }) => {
    pendingQuery.current = nextQuery;
    setState("loading");
    void applyTask.run();
  };

  const retry = () => {
    onRetry?.();
    setState("loading");
    void applyTask.run();
  };

  return (
    <div data-testid="patterns.search-filter-results" data-rcl-search-filter-results>
      <StyleSheet name="search-filter-results-1-0-3" css={styles} />
      <FilterBar
        query={draftQuery}
        onQueryChange={setDraftQuery}
        onApply={apply}
        onReset={() => {
          pendingQuery.current = "";
          setDraftQuery("");
          setAppliedQuery("");
          setState("default");
        }}
        queryLabel="Filter results"
        applyLabel="Search"
      />
      <span data-rcl-search-filter-results-summary aria-live="polite">
        {state === "loading"
          ? "Updating results…"
          : `Search context: ${appliedQuery || "all results"}`}
      </span>
      <SearchResults
        query={appliedQuery}
        items={items}
        state={state}
        errorMessage={errorMessage}
        onRetry={retry}
      />
    </div>
  );
});
