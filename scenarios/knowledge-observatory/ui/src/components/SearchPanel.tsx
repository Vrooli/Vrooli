import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Loader2, AlertCircle } from "lucide-react";
import { Button } from "./ui/button";
import { searchKnowledge, type SearchResult } from "../lib/api";
import { selectors } from "../consts/selectors";

export function SearchPanel() {
  const [query, setQuery] = useState("");
  const [searchTrigger, setSearchTrigger] = useState<string | null>(null);
  const sampleQueries = [
    "recent knowledge health status",
    "semantic drift detection playbooks",
    "qdrant collections overview",
  ];

  const { data, isLoading, error } = useQuery({
    queryKey: ["search", searchTrigger],
    queryFn: () => searchKnowledge({ query: searchTrigger!, limit: 10 }),
    enabled: !!searchTrigger,
  });

  const runSearch = (nextQuery: string) => {
    const trimmed = nextQuery.trim();
    if (!trimmed) return;
    setQuery(trimmed);
    setSearchTrigger(trimmed);
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    runSearch(query);
  };

  const handleClear = () => {
    setQuery("");
    setSearchTrigger(null);
  };

  return (
    <div className="ko-stack-sm">
      <form onSubmit={handleSearch} className="flex flex-col md:flex-row gap-3" data-testid={selectors.search.form}>
        <div className="flex-1">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
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
            disabled={isLoading || !query.trim()}
            className="ko-button-primary"
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
            className="ko-button-secondary"
            data-testid={selectors.search.clear}
            disabled={!query && !data}
          >
            Clear
          </Button>
        </div>
      </form>

      <div className="flex flex-wrap items-center gap-2" data-testid={selectors.search.sampleGroup}>
        <span className="ko-meta">Try:</span>
        {sampleQueries.map((sample) => (
          <Button
            key={sample}
            type="button"
            onClick={() => runSearch(sample)}
            className="ko-button-secondary ko-button-compact"
            data-testid={selectors.search.sampleButton}
            data-query={sample}
          >
            {sample}
          </Button>
        ))}
      </div>

      {error && (
        <div
          className="ko-alert ko-alert-danger"
          data-testid={selectors.search.error}
        >
          <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-red-300 ko-alert-title">Search Error</p>
            <p className="ko-text-sm text-red-600 mt-1">{(error as Error).message}</p>
          </div>
        </div>
      )}

      {data && (
        <div className="ko-stack-sm">
          <div className="flex flex-wrap items-center justify-between ko-text-sm ko-muted" data-testid={selectors.search.resultsSummary}>
            <span>Found {data.results.length} results</span>
            <span>Took {data.took_ms}ms</span>
          </div>

          {data.results.length === 0 ? (
            <div
              className="ko-panel p-6 text-center"
              data-testid={selectors.search.emptyState}
            >
              <p className="ko-muted">No results found for "{data.query}"</p>
              <p className="ko-text-sm ko-subtle mt-1">Try a different phrasing or a broader concept.</p>
            </div>
          ) : (
            <div className="ko-stack-xs" data-testid={selectors.search.resultsList}>
              {data.results.map((result: SearchResult) => (
                <div
                  key={result.id}
                  className="ko-card p-4 hover:border-green-500/70 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4 mb-2">
                    <span className="ko-text-xs text-green-500 font-mono">ID: {result.id}</span>
                    <span className="ko-text-xs text-green-300 font-semibold">
                      Score: {(result.score * 100).toFixed(1)}%
                    </span>
                  </div>
                  <p className="ko-text-sm text-green-200">{result.content || "No content available"}</p>
                  {Object.keys(result.metadata).length > 0 && (
                    <div className="mt-2 pt-2 border-t border-green-800/40">
                      <details className="ko-text-xs text-green-500">
                        <summary className="cursor-pointer hover:text-green-300">Metadata</summary>
                        <pre className="mt-2 p-2 bg-black/70 rounded overflow-x-auto text-green-200 font-mono">
                          {JSON.stringify(result.metadata, null, 2)}
                        </pre>
                      </details>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {!data && !error && !isLoading && (
        <div
          className="ko-panel p-8 text-center"
          data-testid={selectors.search.emptyState}
        >
          <Search className="h-12 w-12 text-green-600 mx-auto mb-3" />
          <p className="ko-muted">Enter a query to search the knowledge base</p>
          <p className="ko-text-sm ko-subtle mt-1">Uses semantic embeddings to find relevant content</p>
        </div>
      )}
    </div>
  );
}
