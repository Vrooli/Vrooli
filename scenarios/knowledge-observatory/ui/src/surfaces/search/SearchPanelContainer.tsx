// DOC: docs/reference/api-endpoints.md#search
import { useEffect, useRef } from "react";
import { SearchPanel } from "./components/SearchPanel";
import { useSearchController } from "../../shared/hooks/knowledgeHooks";

export type SearchPanelContainerProps = {
  prefillQuery?: string | null;
  autoRun?: boolean;
};

export function SearchPanelContainer({ prefillQuery, autoRun }: SearchPanelContainerProps) {
  const appliedRef = useRef(false);
  const {
    query,
    setQuery,
    sampleQueries,
    runSearch,
    handleSubmit,
    clear,
    isLoading,
    hasError,
    hasData,
    isSubmitDisabled,
    isClearDisabled,
    viewModel,
  } = useSearchController();

  useEffect(() => {
    if (!prefillQuery || appliedRef.current) return;
    appliedRef.current = true;
    setQuery(prefillQuery);
    if (autoRun) {
      runSearch(prefillQuery);
    }
  }, [prefillQuery, autoRun, runSearch, setQuery]);

  return (
    <SearchPanel
      query={query}
      onQueryChange={setQuery}
      onSubmit={handleSubmit}
      onClear={clear}
      onSampleClick={runSearch}
      sampleQueries={sampleQueries}
      isLoading={isLoading}
      hasError={hasError}
      errorMessage={viewModel.errorMessage}
      hasData={hasData}
      hasResults={viewModel.hasResults}
      isSubmitDisabled={isSubmitDisabled}
      isClearDisabled={isClearDisabled}
      displayQuery={viewModel.displayQuery}
      totalResults={viewModel.totalResults}
      tookMsLabel={viewModel.tookMsLabel}
      results={viewModel.results}
    />
  );
}
