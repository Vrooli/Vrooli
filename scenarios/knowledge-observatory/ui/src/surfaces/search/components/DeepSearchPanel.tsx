import type { FormEvent } from "react";
import { Loader2, Search, Sparkles } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { DeepSearchResult } from "../../../shared/services/documentationApi";

export type DeepSearchPanelProps = {
  query: string;
  scope: string;
  scenario: string;
  basePath: string;
  maxResults: number;
  followRefs: boolean;
  timeoutSeconds?: number;
  isSubmitting: boolean;
  isRunning: boolean;
  statusLabel: string;
  progressLabel: string;
  errorMessage: string;
  hasResults: boolean;
  results: DeepSearchResult[];
  onQueryChange: (value: string) => void;
  onScopeChange: (value: string) => void;
  onScenarioChange: (value: string) => void;
  onBasePathChange: (value: string) => void;
  onMaxResultsChange: (value: number) => void;
  onFollowRefsChange: (value: boolean) => void;
  onTimeoutSecondsChange: (value: number | undefined) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onClear: () => void;
};

export function DeepSearchPanel({
  query,
  scope,
  scenario,
  basePath,
  maxResults,
  followRefs,
  timeoutSeconds,
  isSubmitting,
  isRunning,
  statusLabel,
  progressLabel,
  errorMessage,
  hasResults,
  results,
  onQueryChange,
  onScopeChange,
  onScenarioChange,
  onBasePathChange,
  onMaxResultsChange,
  onFollowRefsChange,
  onTimeoutSecondsChange,
  onSubmit,
  onClear,
}: DeepSearchPanelProps) {
  const showScenario = scope === "scenario";
  const showBasePath = scope === "path";
  const resolvedStatus = progressLabel ? `${statusLabel} · ${progressLabel}` : statusLabel;

  return (
    <div className="ko-stack-sm">
      <form onSubmit={onSubmit} className="ko-stack-sm" data-testid={selectors.deepSearch.form}>
        <div className="ko-stack-xs">
          <label className="ko-text-sm ko-text-strong" htmlFor="deep-search-query">
            Deep documentation search
          </label>
          <input
            id="deep-search-query"
            type="text"
            value={query}
            onChange={(e) => onQueryChange(e.target.value)}
            placeholder="Ask a documentation question or describe what you need..."
            className="ko-input"
            data-testid={selectors.deepSearch.input}
          />
          <p className="ko-input-help">
            Uses agent-manager to scan docs, follow references, and rank the most relevant files.
          </p>
        </div>

        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="ko-text-xs ko-subtle" htmlFor="deep-search-scope">
              Scope
            </label>
            <select
              id="deep-search-scope"
              className="ko-input"
              value={scope}
              onChange={(e) => onScopeChange(e.target.value)}
            >
              <option value="global">Global</option>
              <option value="scenario">Scenario</option>
              <option value="path">Path</option>
            </select>
          </div>
          <div>
            <label className="ko-text-xs ko-subtle" htmlFor="deep-search-max-results">
              Max results
            </label>
            <input
              id="deep-search-max-results"
              type="number"
              min={1}
              max={50}
              value={Number.isFinite(maxResults) ? maxResults : 10}
              onChange={(e) => onMaxResultsChange(Number(e.target.value))}
              className="ko-input"
            />
          </div>
          <div>
            <label className="ko-text-xs ko-subtle" htmlFor="deep-search-timeout">
              Timeout (seconds)
            </label>
            <input
              id="deep-search-timeout"
              type="number"
              min={10}
              max={600}
              value={timeoutSeconds ?? ""}
              onChange={(e) => {
                const value = Number(e.target.value);
                onTimeoutSecondsChange(Number.isFinite(value) && value > 0 ? value : undefined);
              }}
              className="ko-input"
            />
          </div>
        </div>

        {showScenario && (
          <div>
            <label className="ko-text-xs ko-subtle" htmlFor="deep-search-scenario">
              Scenario name
            </label>
            <input
              id="deep-search-scenario"
              type="text"
              value={scenario}
              onChange={(e) => onScenarioChange(e.target.value)}
              placeholder="knowledge-observatory"
              className="ko-input"
            />
          </div>
        )}

        {showBasePath && (
          <div>
            <label className="ko-text-xs ko-subtle" htmlFor="deep-search-base-path">
              Base path
            </label>
            <input
              id="deep-search-base-path"
              type="text"
              value={basePath}
              onChange={(e) => onBasePathChange(e.target.value)}
              placeholder="scenarios/knowledge-observatory/docs"
              className="ko-input"
            />
          </div>
        )}

        <div className="flex flex-wrap items-center gap-3">
          <label className="flex items-center gap-2 ko-text-xs ko-subtle">
            <input
              type="checkbox"
              checked={followRefs}
              onChange={(e) => onFollowRefsChange(e.target.checked)}
            />
            Follow references
          </label>
          <div className="ml-auto flex gap-2">
            <Button
              type="submit"
              variant="primary"
              disabled={isSubmitting || query.trim().length === 0}
              data-testid={selectors.deepSearch.submit}
            >
              {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
              <span className="ml-2">Start deep search</span>
            </Button>
            <Button
              type="button"
              variant="secondary"
              onClick={onClear}
              disabled={isSubmitting && !hasResults}
              data-testid={selectors.deepSearch.clear}
            >
              Clear
            </Button>
          </div>
        </div>
      </form>

      {(isRunning || statusLabel !== "idle") && (
        <div className="ko-card p-3 flex items-center gap-2" data-testid={selectors.deepSearch.status}>
          <Sparkles className="h-4 w-4 ko-icon" />
          <span className="ko-text-sm ko-text-strong">Status:</span>
          <span className="ko-text-sm ko-text-primary">{resolvedStatus}</span>
        </div>
      )}

      {errorMessage && (
        <div className="ko-alert ko-alert-danger" data-testid={selectors.deepSearch.error}>
          <div>
            <p className="ko-alert-title ko-text-danger-strong">Deep Search Error</p>
            <p className="ko-text-sm ko-text-danger-muted mt-1">{errorMessage}</p>
          </div>
        </div>
      )}

      {hasResults && (
        <div className="ko-stack-sm" data-testid={selectors.deepSearch.results}>
          {results.map((result, index) => (
            <div key={`${result.path}-${index}`} className="ko-card p-4 ko-card-interactive">
              <div className="flex flex-wrap items-center justify-between gap-3 mb-2">
                <span className="ko-text-xs ko-subtle font-mono">{result.path}</span>
                <span className="ko-text-xs ko-text-strong">
                  Relevance: {result.relevance.toFixed(2)}
                </span>
              </div>
              <p className="ko-text-sm ko-text-primary">{result.summary}</p>
              <p className="ko-text-xs ko-text-subtle mt-2">Match: {result.match_reason}</p>
              {result.snippet && (
                <p className="ko-text-xs ko-text-subtle mt-2 font-mono">{result.snippet}</p>
              )}
              {result.references && result.references.length > 0 && (
                <div className="mt-3 ko-text-xs ko-text-subtle">
                  <p className="font-semibold">References</p>
                  <ul className="list-disc list-inside">
                    {result.references.map((ref) => (
                      <li key={ref}>{ref}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
