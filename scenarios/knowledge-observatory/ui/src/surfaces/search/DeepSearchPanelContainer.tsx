// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import { useEffect, useRef } from "react";
import { DeepSearchPanel } from "./components/DeepSearchPanel";
import { useDeepSearchController } from "../../shared/hooks/deepSearchHooks";

export type DeepSearchPanelContainerProps = {
  prefillQuery?: string | null;
  autoRun?: boolean;
};

export function DeepSearchPanelContainer({ prefillQuery, autoRun }: DeepSearchPanelContainerProps) {
  const appliedRef = useRef(false);
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

  useEffect(() => {
    if (!prefillQuery || appliedRef.current) return;
    appliedRef.current = true;
    setQuery(prefillQuery);
    if (autoRun) {
      void submit({ query: prefillQuery });
    }
  }, [prefillQuery, autoRun, setQuery, submit]);

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
