import { useState, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, BookOpen } from "lucide-react";
import { fetchGlossary } from "../../lib/api";
import { useDebouncedValue } from "../../hooks/useDebouncedValue";
import { SearchInput, type SearchInputHandle } from "../ui/SearchInput";

export function GlossaryPanel() {
  const [searchTerm, setSearchTerm] = useState("");
  const searchRef = useRef<SearchInputHandle>(null);
  const { debounced: debouncedSearch, isPending: debouncing } = useDebouncedValue(searchTerm, 300);

  const { data, isLoading } = useQuery({
    queryKey: ["glossary", debouncedSearch],
    queryFn: () => fetchGlossary(debouncedSearch || undefined),
  });

  const entries = data?.entries ?? [];

  return (
    <div data-testid="glossary-panel">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-xl font-semibold sm:text-2xl">Glossary</h1>
          <p className="mt-1 text-sm text-slate-300">
            Look up Vrooli terms and concepts
          </p>
        </div>
      </div>

      {/* Search */}
      <div className="mt-4">
        <SearchInput
          ref={searchRef}
          value={searchTerm}
          onChange={setSearchTerm}
          placeholder="Search terms..."
          ariaLabel="Search glossary terms"
          testId="glossary-search"
          clearTestId="glossary-clear-search"
          busy={debouncing}
          busyTestId="glossary-debounce-indicator"
        />
      </div>

      {/* Results count */}
      {!isLoading && entries.length > 0 && (
        <p className="mt-3 text-xs text-slate-300" aria-live="polite" data-testid="glossary-count">
          {entries.length} term{entries.length !== 1 ? "s" : ""}{searchTerm ? ` matching "${searchTerm}"` : ""}
        </p>
      )}

      {/* Content */}
      <div className={entries.length > 0 ? "mt-2" : "mt-6"}>
        {isLoading ? (
          <div data-testid="glossary-loading" className="flex items-center justify-center py-12 text-slate-300" role="status">
            <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
            <span className="ml-2 text-sm">Loading glossary...</span>
          </div>
        ) : entries.length === 0 ? (
          <div data-testid="glossary-empty" className="flex flex-col items-center justify-center py-12 text-slate-300">
            <BookOpen className="h-8 w-8" aria-hidden="true" />
            <p className="mt-3 text-sm font-medium">No matching terms found</p>
            {searchTerm ? (
              <p className="mt-1 text-xs text-slate-300">
                Try a different search term or{" "}
                <button
                  type="button"
                  onClick={() => { setSearchTerm(""); searchRef.current?.focus(); }}
                  className="text-emerald-400 underline underline-offset-2 hover:text-emerald-300 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-emerald-500"
                >
                  clear the search
                </button>
                .
              </p>
            ) : (
              <p className="mt-1 text-xs text-slate-300">
                Complete the setup wizard to populate the glossary.
              </p>
            )}
          </div>
        ) : (
          <dl data-testid="glossary-list" className="space-y-1">
            {entries.map((entry) => (
              <div
                key={entry.term}
                data-testid={`glossary-entry-${entry.term}`}
                className="rounded-lg border border-white/5 bg-white/[0.02] p-4 transition-colors hover:bg-white/5"
              >
                <dt className="flex items-baseline gap-2">
                  <span className="font-medium text-slate-100">{entry.term}</span>
                  <span className="rounded bg-white/10 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wider text-slate-300">
                    {entry.category}
                  </span>
                </dt>
                <dd className="mt-1.5 text-sm leading-relaxed text-slate-300">
                  {entry.description}
                </dd>
              </div>
            ))}
          </dl>
        )}
      </div>
    </div>
  );
}
