// DOC: docs/reference/api-endpoints.md#documentation-search
import type { FormEvent } from "react";
import { AlertCircle, FileSearch, Loader2, Search } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import { Button } from "../../../shared/ui/button";
import type { DocSearchMode } from "../../../shared/hooks/docSearchHooks";
import type { DocSearchResultView } from "../../../shared/controllers/documentationController";

export type DocSearchPanelProps = {
  mode: DocSearchMode;
  pattern: string;
  query: string;
  scope: string;
  scenario: string;
  basePath: string;
  includeContent: boolean;
  fileTypes: string;
  caseSensitive: boolean;
  contextLines: number;
  useSemantic: boolean;
  isLoading: boolean;
  hasError: boolean;
  errorMessage: string;
  hasData: boolean;
  hasResults: boolean;
  isSubmitDisabled: boolean;
  isClearDisabled: boolean;
  displayQuery: string;
  totalResults: number;
  tookMsLabel: string;
  results: DocSearchResultView[];
  onPatternChange: (value: string) => void;
  onQueryChange: (value: string) => void;
  onScopeChange: (value: string) => void;
  onScenarioChange: (value: string) => void;
  onBasePathChange: (value: string) => void;
  onIncludeContentChange: (value: boolean) => void;
  onFileTypesChange: (value: string) => void;
  onCaseSensitiveChange: (value: boolean) => void;
  onContextLinesChange: (value: number) => void;
  onUseSemanticChange: (value: boolean) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onClear: () => void;
};

const MODE_LABELS: Record<DocSearchMode, string> = {
  files: "File Search",
  text: "Text Search",
  unified: "Unified Documentation Search",
};

const MODE_PLACEHOLDERS: Record<DocSearchMode, string> = {
  files: "**/README.md",
  text: "health score",
  unified: "search documentation by topic or filename",
};

export function DocSearchPanel({
  mode,
  pattern,
  query,
  scope,
  scenario,
  basePath,
  includeContent,
  fileTypes,
  caseSensitive,
  contextLines,
  useSemantic,
  isLoading,
  hasError,
  errorMessage,
  hasData,
  hasResults,
  isSubmitDisabled,
  isClearDisabled,
  displayQuery,
  totalResults,
  tookMsLabel,
  results,
  onPatternChange,
  onQueryChange,
  onScopeChange,
  onScenarioChange,
  onBasePathChange,
  onIncludeContentChange,
  onFileTypesChange,
  onCaseSensitiveChange,
  onContextLinesChange,
  onUseSemanticChange,
  onSubmit,
  onClear,
}: DocSearchPanelProps) {
  const showScopeScenario = scope === "scenario";
  const showScopePath = scope === "path";
  const showQueryInput = mode !== "files";
  const showPatternInput = mode !== "text";
  const showFileTypes = mode !== "files";
  const showTextOptions = mode !== "files";

  return (
    <div className="ko-stack-sm">
      <div className="ko-text-sm ko-muted">
        <strong className="ko-text-strong">{MODE_LABELS[mode]}</strong> ·{" "}
        {mode === "files"
          ? "Find documentation by name or location patterns."
          : mode === "text"
          ? "Scan documentation content with regex support."
          : "Blend file, text, and semantic results together."}
      </div>

      <form
        onSubmit={onSubmit}
        className="ko-stack-sm"
        data-testid={selectors.search.docSearchForm}
      >
        <div className="grid gap-3 md:grid-cols-2">
          {showQueryInput && (
            <label className="ko-stack-xs">
              <span className="ko-meta">Query</span>
              <input
                type="text"
                value={query}
                onChange={(event) => onQueryChange(event.target.value)}
                placeholder={MODE_PLACEHOLDERS[mode]}
                className="ko-input"
                data-testid={selectors.search.docSearchQuery}
              />
            </label>
          )}
          {showPatternInput && (
            <label className="ko-stack-xs">
              <span className="ko-meta">Pattern</span>
              <input
                type="text"
                value={pattern}
                onChange={(event) => onPatternChange(event.target.value)}
                placeholder="**/README.md"
                className="ko-input"
                data-testid={selectors.search.docSearchPattern}
              />
            </label>
          )}
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <label className="ko-stack-xs">
            <span className="ko-meta">Scope</span>
            <select
              className="ko-input"
              value={scope}
              onChange={(event) => onScopeChange(event.target.value)}
              data-testid={selectors.search.docSearchScope}
            >
              <option value="global">Global</option>
              <option value="scenario">Scenario</option>
              <option value="path">Path</option>
            </select>
          </label>
          {showScopeScenario && (
            <label className="ko-stack-xs">
              <span className="ko-meta">Scenario</span>
              <input
                type="text"
                value={scenario}
                onChange={(event) => onScenarioChange(event.target.value)}
                placeholder="knowledge-observatory"
                className="ko-input"
                data-testid={selectors.search.docSearchScenario}
              />
            </label>
          )}
          {showScopePath && (
            <label className="ko-stack-xs">
              <span className="ko-meta">Base Path</span>
              <input
                type="text"
                value={basePath}
                onChange={(event) => onBasePathChange(event.target.value)}
                placeholder="scenarios/knowledge-observatory/docs"
                className="ko-input"
                data-testid={selectors.search.docSearchBasePath}
              />
            </label>
          )}
        </div>

        {showFileTypes && (
          <label className="ko-stack-xs">
            <span className="ko-meta">File Types</span>
            <input
              type="text"
              value={fileTypes}
              onChange={(event) => onFileTypesChange(event.target.value)}
              placeholder="md,txt,json"
              className="ko-input"
              data-testid={selectors.search.docSearchFileTypes}
            />
          </label>
        )}

        {showTextOptions && (
          <div className="grid gap-3 md:grid-cols-3">
            <label className="ko-stack-xs">
              <span className="ko-meta">Context Lines</span>
              <input
                type="number"
                min={0}
                value={Number.isFinite(contextLines) ? contextLines : 0}
                onChange={(event) => onContextLinesChange(Number(event.target.value))}
                className="ko-input"
                data-testid={selectors.search.docSearchContextLines}
              />
            </label>
            <label className="ko-checkbox-row">
              <input
                type="checkbox"
                checked={caseSensitive}
                onChange={(event) => onCaseSensitiveChange(event.target.checked)}
                data-testid={selectors.search.docSearchCaseSensitive}
              />
              <span>Case sensitive</span>
            </label>
            {mode === "unified" && (
              <label className="ko-checkbox-row">
                <input
                  type="checkbox"
                  checked={useSemantic}
                  onChange={(event) => onUseSemanticChange(event.target.checked)}
                  data-testid={selectors.search.docSearchUseSemantic}
                />
                <span>Include semantic matches</span>
              </label>
            )}
          </div>
        )}

        {mode === "files" && (
          <label className="ko-checkbox-row">
            <input
              type="checkbox"
              checked={includeContent}
              onChange={(event) => onIncludeContentChange(event.target.checked)}
              data-testid={selectors.search.docSearchIncludeContent}
            />
            <span>Include content previews</span>
          </label>
        )}

        <div className="flex flex-wrap gap-2">
          <Button type="submit" variant="primary" disabled={isSubmitDisabled}>
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : mode === "files" ? (
              <FileSearch className="h-4 w-4" />
            ) : (
              <Search className="h-4 w-4" />
            )}
            <span className="ml-2">{isLoading ? "Searching" : "Search"}</span>
          </Button>
          <Button type="button" variant="secondary" onClick={onClear} disabled={isClearDisabled}>
            Clear
          </Button>
        </div>
      </form>

      {hasError && (
        <div className="ko-alert ko-alert-danger" data-testid={selectors.search.docSearchError}>
          <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
          <div>
            <p className="ko-alert-title ko-text-danger-strong">Search Error</p>
            <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          </div>
        </div>
      )}

      {hasData && (
        <div className="ko-stack-sm">
          <div
            className="flex flex-wrap items-center justify-between ko-text-sm ko-muted"
            data-testid={selectors.search.docSearchSummary}
          >
            <span>Found {totalResults} results</span>
            <span>Took {tookMsLabel}</span>
          </div>

          {!hasResults ? (
            <div className="ko-panel p-6 text-center" data-testid={selectors.search.docSearchEmpty}>
              <p className="ko-muted">No results found for "{displayQuery}"</p>
              <p className="ko-text-sm ko-subtle mt-1">Try a broader scope or different pattern.</p>
            </div>
          ) : (
            <div className="ko-stack-xs" data-testid={selectors.search.docSearchResults}>
              {results.map((result) => (
                <div key={result.id} className="ko-card ko-card-interactive p-4">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <p className="ko-text-sm font-semibold">{result.title}</p>
                      <p className="ko-text-xs ko-subtle mt-1">{result.path}</p>
                    </div>
                    <div className="flex flex-wrap items-center gap-2">
                      {result.scoreLabel && (
                        <span className="ko-pill ko-pill-muted">Score {result.scoreLabel}</span>
                      )}
                      {result.sourceLabel && (
                        <span className="ko-pill">{result.sourceLabel}</span>
                      )}
                    </div>
                  </div>
                  {result.meta.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-2">
                      {result.meta.map((entry, index) => (
                        <span key={`${result.id}-meta-${index}`} className="ko-pill ko-pill-muted">
                          {entry}
                        </span>
                      ))}
                    </div>
                  )}
                  {result.snippet && (
                    <pre className="ko-snippet mt-3">{result.snippet}</pre>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {!hasData && !hasError && !isLoading && (
        <div className="ko-panel p-8 text-center" data-testid={selectors.search.docSearchEmpty}>
          <Search className="h-12 w-12 ko-icon-strong mx-auto mb-3" />
          <p className="ko-muted">Enter a query to search documentation</p>
          <p className="ko-text-sm ko-subtle mt-1">Use scope filters to limit the search space</p>
        </div>
      )}
    </div>
  );
}
