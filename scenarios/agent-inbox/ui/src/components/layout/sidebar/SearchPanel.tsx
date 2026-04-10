import { forwardRef } from "react";
import { Search, X, Loader2 } from "lucide-react";
import { Tooltip } from "../../ui/tooltip";
import type { ChatSearchMode } from "./types";

export interface ContentSearchOptions {
  caseSensitive: boolean;
  wholeWord: boolean;
  regex: boolean;
}

interface SearchPanelProps {
  query: string;
  setQuery: (query: string) => void;
  clear: () => void;
  isSearching: boolean;
  searchMode: ChatSearchMode;
  onSearchModeChange: (mode: ChatSearchMode) => void;
  contentSearchOptions: ContentSearchOptions;
  onContentSearchOptionsChange: (options: ContentSearchOptions) => void;
  testIds: {
    searchInput: string;
    clearSearchButton: string;
    searchModeToggle: string;
    searchModeQuick: string;
    searchModeContent: string;
  };
}

export const SearchPanel = forwardRef<HTMLInputElement, SearchPanelProps>(function SearchPanel(
  {
    query,
    setQuery,
    clear,
    isSearching,
    searchMode,
    onSearchModeChange,
    contentSearchOptions,
    onContentSearchOptionsChange,
    testIds,
  },
  ref
) {
  return (
    <div className="px-3 py-2 shrink-0">
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
        <input
          ref={ref}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={searchMode === "quick" ? "Filter chats... (/ or Ctrl+K)" : "Search messages... (/ or Ctrl+K)"}
          className="w-full bg-white/5 border border-white/10 rounded-lg pl-9 pr-8 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
          data-testid={testIds.searchInput}
        />
        {query && (
          <button
            onClick={clear}
            className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white"
            data-testid={testIds.clearSearchButton}
          >
            <X className="h-3 w-3" />
          </button>
        )}
        {isSearching && (
          <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 h-3 w-3 animate-spin text-slate-400" />
        )}
      </div>
      {/* Search mode toggle - visible when query has text */}
      {query && (
        <div className="flex items-center gap-1 mt-2" data-testid={testIds.searchModeToggle}>
          <button
            type="button"
            onClick={() => onSearchModeChange("quick")}
            className={`px-2 py-1 text-[10px] rounded border transition-colors ${
              searchMode === "quick"
                ? "bg-indigo-500/20 text-indigo-400 border-indigo-500/40"
                : "text-slate-400 border-white/10 hover:text-white hover:bg-white/5"
            }`}
            data-testid={testIds.searchModeQuick}
          >
            Quick
          </button>
          <button
            type="button"
            onClick={() => onSearchModeChange("content")}
            className={`px-2 py-1 text-[10px] rounded border transition-colors ${
              searchMode === "content"
                ? "bg-indigo-500/20 text-indigo-400 border-indigo-500/40"
                : "text-slate-400 border-white/10 hover:text-white hover:bg-white/5"
            }`}
            data-testid={testIds.searchModeContent}
          >
            Content
          </button>
          {/* Content search option toggles */}
          {searchMode === "content" && (
            <>
              <div className="w-px h-4 bg-white/10 mx-0.5" />
              <Tooltip content="Case sensitive">
                <button
                  type="button"
                  onClick={() =>
                    onContentSearchOptionsChange({
                      ...contentSearchOptions,
                      caseSensitive: !contentSearchOptions.caseSensitive,
                    })
                  }
                  className={`px-1.5 py-1 text-[10px] rounded border font-mono transition-colors ${
                    contentSearchOptions.caseSensitive
                      ? "bg-indigo-500/20 text-indigo-400 border-indigo-500/40"
                      : "text-slate-400 border-white/10 hover:text-white hover:bg-white/5"
                  }`}
                  data-testid="search-opt-case"
                >
                  Aa
                </button>
              </Tooltip>
              <Tooltip content="Whole word">
                <button
                  type="button"
                  onClick={() =>
                    onContentSearchOptionsChange({
                      ...contentSearchOptions,
                      wholeWord: !contentSearchOptions.wholeWord,
                    })
                  }
                  className={`px-1.5 py-1 text-[10px] rounded border font-mono transition-colors ${
                    contentSearchOptions.wholeWord
                      ? "bg-indigo-500/20 text-indigo-400 border-indigo-500/40"
                      : "text-slate-400 border-white/10 hover:text-white hover:bg-white/5"
                  }`}
                  data-testid="search-opt-word"
                >
                  W
                </button>
              </Tooltip>
              <Tooltip content="Regex">
                <button
                  type="button"
                  onClick={() =>
                    onContentSearchOptionsChange({
                      ...contentSearchOptions,
                      regex: !contentSearchOptions.regex,
                    })
                  }
                  className={`px-1.5 py-1 text-[10px] rounded border font-mono transition-colors ${
                    contentSearchOptions.regex
                      ? "bg-indigo-500/20 text-indigo-400 border-indigo-500/40"
                      : "text-slate-400 border-white/10 hover:text-white hover:bg-white/5"
                  }`}
                  data-testid="search-opt-regex"
                >
                  .*
                </button>
              </Tooltip>
            </>
          )}
        </div>
      )}
      <p className="mt-2 text-[11px] text-slate-500">
        Tip: <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">/</kbd> or{" "}
        <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">Ctrl+K</kbd> focuses search.
      </p>
    </div>
  );
});
