import type { FormEvent } from "react";
import { Search, Loader2, AlertCircle } from "lucide-react";
import { Button } from "../../../shared/ui/button";
import { selectors } from "../../../consts/selectors";
import type { SearchResultView } from "../../../shared/controllers/knowledgeController";

// AI_CHECK: REACT_STABILITY=1 | LAST: 2026-01-25

const EMPTY_RESULTS: SearchResultView[] = [];
const EMPTY_SAMPLE_QUERIES: string[] = [];
const DEFAULT_ERROR_MESSAGE = "Search failed. Please try again.";
const DEFAULT_DISPLAY_QUERY = "your query";

function safeJsonStringify(value: unknown) {
  try {
    return JSON.stringify(value, null, 2);
  } catch (error) {
    console.warn("[knowledge-observatory] Unable to render metadata", error);
    return "Unable to render metadata.";
  }
}

function normalizeSamples(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return EMPTY_SAMPLE_QUERIES;
  }

  return value.filter((entry): entry is string => typeof entry === "string" && entry.trim().length > 0);
}

function normalizeResults(value: unknown): SearchResultView[] {
  if (!Array.isArray(value)) {
    return EMPTY_RESULTS;
  }

  return value.filter((entry): entry is SearchResultView => Boolean(entry) && typeof entry === "object");
}

export type SearchPanelProps = {
  query: string;
  onQueryChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onClear: () => void;
  onSampleClick: (value: string) => void;
  sampleQueries: string[];
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
  results: SearchResultView[];
};

export function SearchPanel({
  query,
  onQueryChange,
  onSubmit,
  onClear,
  onSampleClick,
  sampleQueries,
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
}: SearchPanelProps) {
  const safeSampleQueries = normalizeSamples(sampleQueries);
  const safeResults = normalizeResults(results);
  const safeDisplayQuery =
    typeof displayQuery === "string" && displayQuery.trim().length > 0
      ? displayQuery
      : DEFAULT_DISPLAY_QUERY;
  const safeErrorMessage =
    typeof errorMessage === "string" && errorMessage.trim().length > 0
      ? errorMessage
      : DEFAULT_ERROR_MESSAGE;
  const safeTotalResults = Number.isFinite(totalResults) ? totalResults : safeResults.length;
  const safeTookMsLabel =
    typeof tookMsLabel === "string" && tookMsLabel.trim().length > 0 ? tookMsLabel : "?ms";
  const resolvedHasResults = hasResults || safeResults.length > 0;
  const handleQueryChange = (value: string) => {
    if (typeof onQueryChange === "function") {
      onQueryChange(value);
    }
  };
  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (typeof onSubmit === "function") {
      onSubmit(event);
    }
  };
  const handleClear = () => {
    if (typeof onClear === "function") {
      onClear();
    }
  };
  const handleSampleClick = (value: string) => {
    if (typeof onSampleClick === "function") {
      onSampleClick(value);
    }
  };

  return (
    <div className="ko-stack-sm">
      <form onSubmit={handleSubmit} className="flex flex-col md:flex-row gap-3" data-testid={selectors.search.form}>
        <div className="flex-1">
          <input
            type="text"
            value={query}
            onChange={(e) => handleQueryChange(e.target.value)}
            placeholder="Ask a question or describe the concept you want to find..."
            className="ko-input"
            data-testid={selectors.search.input}
            aria-label="Semantic search query"
          />
          <p className="ko-input-help">
            Use natural language, e.g. “knowledge health status” or “semantic drift detector”.
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            type="submit"
            disabled={isSubmitDisabled}
            variant="primary"
            data-testid={selectors.search.submit}
          >
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Search className="h-4 w-4" />
            )}
            <span className="ml-2">Search</span>
          </Button>
          <Button
            type="button"
            onClick={handleClear}
            variant="secondary"
            data-testid={selectors.search.clear}
            disabled={isClearDisabled}
          >
            Clear
          </Button>
        </div>
      </form>

      <div className="flex flex-wrap items-center gap-2" data-testid={selectors.search.sampleGroup}>
        <span className="ko-meta">Try:</span>
        {safeSampleQueries.map((sample, index) => (
          <Button
            key={`${sample}-${index}`}
            type="button"
            onClick={() => handleSampleClick(sample)}
            variant="secondary"
            size="compact"
            data-testid={selectors.search.sampleButton}
            data-query={sample}
          >
            {sample}
          </Button>
        ))}
      </div>

      {hasError && (
        <div className="ko-alert ko-alert-danger" data-testid={selectors.search.error}>
          <AlertCircle className="h-5 w-5 ko-text-danger flex-shrink-0 mt-0.5" />
          <div>
            <p className="ko-alert-title ko-text-danger-strong">Search Error</p>
            <p className="ko-text-sm ko-text-danger-muted mt-1">{safeErrorMessage}</p>
          </div>
        </div>
      )}

      {hasData && (
        <div className="ko-stack-sm">
          <div className="flex flex-wrap items-center justify-between ko-text-sm ko-muted" data-testid={selectors.search.resultsSummary}>
            <span>Found {safeTotalResults} results</span>
            <span>Took {safeTookMsLabel}</span>
          </div>

          {!resolvedHasResults ? (
            <div
              className="ko-panel p-6 text-center"
              data-testid={selectors.search.emptyState}
            >
              <p className="ko-muted">No results found for "{safeDisplayQuery}"</p>
              <p className="ko-text-sm ko-subtle mt-1">Try a different phrasing or a broader concept.</p>
            </div>
          ) : (
            <div className="ko-stack-xs" data-testid={selectors.search.resultsList}>
              {safeResults.map((result, index) => {
                const resultId = typeof result.id === "string" && result.id.trim().length > 0 ? result.id : `result-${index + 1}`;
                const content =
                  typeof result.content === "string" && result.content.trim().length > 0
                    ? result.content
                    : "No content available";
                const metadata = result.metadata && typeof result.metadata === "object" ? result.metadata : {};
                const hasMetadata = result.hasMetadata && Object.keys(metadata).length > 0;

                return (
                  <div key={resultId} className="ko-card ko-card-interactive p-4">
                    <div className="flex items-start justify-between gap-4 mb-2">
                      <span className="ko-text-xs ko-subtle font-mono">ID: {resultId}</span>
                      <span className="ko-text-xs ko-text-strong font-semibold">
                        Score: {result.scoreLabel}
                      </span>
                    </div>
                    <p className="ko-text-sm ko-text-primary">{content}</p>
                    {hasMetadata && (
                      <div className="mt-2 pt-2 ko-border-subtle border-t">
                        <details className="ko-text-xs ko-subtle">
                          <summary className="ko-link">Metadata</summary>
                          <pre className="mt-2 p-2 ko-surface-strong rounded overflow-x-auto ko-text-primary font-mono">
                            {safeJsonStringify(metadata)}
                          </pre>
                        </details>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {!hasData && !hasError && !isLoading && (
        <div className="ko-panel p-8 text-center" data-testid={selectors.search.emptyState}>
          <Search className="h-12 w-12 ko-icon-strong mx-auto mb-3" />
          <p className="ko-muted">Enter a query to search the knowledge base</p>
          <p className="ko-text-sm ko-subtle mt-1">Uses semantic embeddings to find relevant content</p>
        </div>
      )}
    </div>
  );
}
