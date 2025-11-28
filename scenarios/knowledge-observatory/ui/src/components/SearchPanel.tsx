import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, Loader2, AlertCircle } from "lucide-react";
import { Button } from "./ui/button";
import { searchKnowledge, type SearchResult } from "../lib/api";

export function SearchPanel() {
  const [query, setQuery] = useState("");
  const [searchTrigger, setSearchTrigger] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["search", searchTrigger],
    queryFn: () => searchKnowledge({ query: searchTrigger!, limit: 10 }),
    enabled: !!searchTrigger,
  });

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      setSearchTrigger(query);
    }
  };

  return (
    <div className="space-y-4">
      <form onSubmit={handleSearch} className="flex gap-2">
        <div className="flex-1 relative">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter your semantic query..."
            className="w-full px-4 py-2 bg-black border border-green-900/50 rounded text-green-400 placeholder-green-700 focus:outline-none focus:border-green-600"
          />
        </div>
        <Button
          type="submit"
          disabled={isLoading || !query.trim()}
          className="bg-green-900 hover:bg-green-800 text-green-100 border border-green-700"
        >
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Search className="h-4 w-4" />
          )}
        </Button>
      </form>

      {error && (
        <div className="p-4 border border-red-900/50 bg-red-950/20 rounded flex items-start gap-2">
          <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0 mt-0.5" />
          <div>
            <p className="text-red-400 font-semibold">Search Error</p>
            <p className="text-sm text-red-600 mt-1">{(error as Error).message}</p>
          </div>
        </div>
      )}

      {data && (
        <div className="space-y-3">
          <div className="flex items-center justify-between text-sm text-green-600">
            <span>Found {data.results.length} results</span>
            <span>Took {data.took_ms}ms</span>
          </div>

          {data.results.length === 0 ? (
            <div className="p-6 text-center border border-green-900/50 bg-green-950/10 rounded">
              <p className="text-green-600">No results found for "{data.query}"</p>
              <p className="text-sm text-green-700 mt-1">Try a different query or lower the threshold</p>
            </div>
          ) : (
            <div className="space-y-2">
              {data.results.map((result: SearchResult) => (
                <div
                  key={result.id}
                  className="p-4 border border-green-900/50 bg-black/40 rounded hover:border-green-700 transition-colors"
                >
                  <div className="flex items-start justify-between gap-4 mb-2">
                    <span className="text-xs text-green-700 font-mono">ID: {result.id}</span>
                    <span className="text-xs text-green-500 font-semibold">
                      Score: {(result.score * 100).toFixed(1)}%
                    </span>
                  </div>
                  <p className="text-sm text-green-300">{result.content || "No content available"}</p>
                  {Object.keys(result.metadata).length > 0 && (
                    <div className="mt-2 pt-2 border-t border-green-900/30">
                      <details className="text-xs text-green-700">
                        <summary className="cursor-pointer hover:text-green-500">Metadata</summary>
                        <pre className="mt-2 p-2 bg-black/60 rounded overflow-x-auto">
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
        <div className="p-8 text-center border border-green-900/50 bg-green-950/10 rounded">
          <Search className="h-12 w-12 text-green-700 mx-auto mb-3" />
          <p className="text-green-600">Enter a query to search the knowledge base</p>
          <p className="text-sm text-green-700 mt-1">Uses semantic embeddings to find relevant content</p>
        </div>
      )}
    </div>
  );
}
