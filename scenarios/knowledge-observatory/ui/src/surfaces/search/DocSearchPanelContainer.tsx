// DOC: docs/reference/api-endpoints.md#documentation-search
import { useEffect, useRef } from "react";
import { DocSearchPanel } from "./components/DocSearchPanel";
import { useDocSearchController, type DocSearchMode } from "../../shared/hooks/docSearchHooks";

export type DocSearchPanelContainerProps = {
  mode: DocSearchMode;
  prefillValue?: string | null;
  autoRun?: boolean;
};

export function DocSearchPanelContainer({ mode, prefillValue, autoRun }: DocSearchPanelContainerProps) {
  const appliedRef = useRef(false);
  const {
    pattern,
    setPattern,
    query,
    setQuery,
    scope,
    setScope,
    scenario,
    setScenario,
    basePath,
    setBasePath,
    includeContent,
    setIncludeContent,
    fileTypes,
    setFileTypes,
    caseSensitive,
    setCaseSensitive,
    contextLines,
    setContextLines,
    useSemantic,
    setUseSemantic,
    isLoading,
    hasError,
    errorMessage,
    hasData,
    isSubmitDisabled,
    isClearDisabled,
    submit,
    clear,
    viewModel,
  } = useDocSearchController(mode);

  useEffect(() => {
    if (!prefillValue || appliedRef.current) return;
    appliedRef.current = true;
    if (mode === "files") {
      setPattern(prefillValue);
      if (autoRun) {
        void submit({ pattern: prefillValue });
      }
      return;
    }
    setQuery(prefillValue);
    if (autoRun) {
      void submit({ query: prefillValue });
    }
  }, [mode, prefillValue, autoRun, setPattern, setQuery, submit]);

  return (
    <DocSearchPanel
      mode={mode}
      pattern={pattern}
      query={query}
      scope={scope}
      scenario={scenario}
      basePath={basePath}
      includeContent={includeContent}
      fileTypes={fileTypes}
      caseSensitive={caseSensitive}
      contextLines={contextLines}
      useSemantic={useSemantic}
      isLoading={isLoading}
      hasError={hasError}
      errorMessage={errorMessage}
      hasData={hasData}
      hasResults={viewModel.hasResults}
      isSubmitDisabled={isSubmitDisabled}
      isClearDisabled={isClearDisabled}
      displayQuery={viewModel.displayQuery}
      totalResults={viewModel.totalResults}
      tookMsLabel={viewModel.tookMsLabel}
      results={viewModel.results}
      onPatternChange={setPattern}
      onQueryChange={setQuery}
      onScopeChange={setScope}
      onScenarioChange={setScenario}
      onBasePathChange={setBasePath}
      onIncludeContentChange={setIncludeContent}
      onFileTypesChange={setFileTypes}
      onCaseSensitiveChange={setCaseSensitive}
      onContextLinesChange={setContextLines}
      onUseSemanticChange={setUseSemantic}
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
      onClear={clear}
    />
  );
}
