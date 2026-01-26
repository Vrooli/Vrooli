// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import { DeepSearchPanel } from "./components/DeepSearchPanel";
import { useDeepSearchController } from "../../shared/hooks/deepSearchHooks";

export function DeepSearchPanelContainer() {
  const {
    query,
    setQuery,
    scope,
    setScope,
    scenario,
    setScenario,
    basePath,
    setBasePath,
    maxResults,
    setMaxResults,
    followRefs,
    setFollowRefs,
    timeoutSeconds,
    setTimeoutSeconds,
    isSubmitting,
    isRunning,
    hasResults,
    submit,
    clear,
    viewModel,
  } = useDeepSearchController();

  return (
    <DeepSearchPanel
      query={query}
      scope={scope}
      scenario={scenario}
      basePath={basePath}
      maxResults={maxResults}
      followRefs={followRefs}
      timeoutSeconds={timeoutSeconds}
      isSubmitting={isSubmitting}
      isRunning={isRunning}
      statusLabel={viewModel.statusLabel}
      progressLabel={viewModel.progressLabel}
      errorMessage={viewModel.errorMessage}
      hasResults={hasResults}
      results={viewModel.results}
      onQueryChange={setQuery}
      onScopeChange={setScope}
      onScenarioChange={setScenario}
      onBasePathChange={setBasePath}
      onMaxResultsChange={setMaxResults}
      onFollowRefsChange={setFollowRefs}
      onTimeoutSecondsChange={setTimeoutSeconds}
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
      onClear={clear}
    />
  );
}
