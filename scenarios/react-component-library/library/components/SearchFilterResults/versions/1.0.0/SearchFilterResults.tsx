/** @vrooliComponentSource patterns.search-filter-results */
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

export function SearchFilterResults({
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
      <style
        data-rcl-search-filter-results-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
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
}
