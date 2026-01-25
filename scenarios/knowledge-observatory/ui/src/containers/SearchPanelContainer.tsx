import { SearchPanel } from "../components/SearchPanel";
import { useSearchController } from "../hooks/knowledgeHooks";

export function SearchPanelContainer() {
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
