/**
 * @libraryId react-component-library:SearchFilterResults
 * @displayName SearchFilterResults
 * @description A URL-ready search and filter composition that separates no-match emptiness from loading and error states.
 * @version 1.0.2
 * @tags ["pattern","data-source","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource patterns.search-filter-results */
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

import { useCallback, useRef, useState } from "react";
import { FilterBar } from "../../../FilterBar/versions/1.0.0/FilterBar";
import {
  SearchResults,
  type SearchResultsState,
} from "../../../SearchResults/versions/1.0.0/SearchResults";
import { useAbortableTask } from "../../../../hooks/useAbortableTask/versions/1.0.0/useAbortableTask";

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
    <div data-rcl-search-filter-results>
      <style data-rcl-search-filter-results-styles dangerouslySetInnerHTML={{ __html: styles }} />
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
