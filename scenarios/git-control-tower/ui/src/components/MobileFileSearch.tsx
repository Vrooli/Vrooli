import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { Search, X, FileCode, Loader2, AlertCircle, Clock, ChevronDown, ChevronUp } from "lucide-react";
import { BottomSheet } from "./ui/bottom-sheet";
import { useFileSearch } from "../lib/hooks";
import type { FileInfo } from "../lib/api";

interface MobileFileSearchProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectFile: (path: string) => void;
}

// localStorage key for file search history
const FILE_HISTORY_KEY = "git-control-tower:file-search-history";
const MAX_HISTORY_ITEMS = 20;

// Helper to get history from localStorage
function getFileHistory(): string[] {
  try {
    const stored = localStorage.getItem(FILE_HISTORY_KEY);
    if (!stored) return [];
    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is string => typeof item === "string");
  } catch {
    return [];
  }
}

// Helper to add a file to history
function addToFileHistory(path: string): void {
  try {
    const history = getFileHistory();
    // Remove if already exists (will re-add at front)
    const filtered = history.filter((p) => p !== path);
    // Add to front, limit size
    const updated = [path, ...filtered].slice(0, MAX_HISTORY_ITEMS);
    localStorage.setItem(FILE_HISTORY_KEY, JSON.stringify(updated));
  } catch {
    // Ignore localStorage errors
  }
}

// Simple fuzzy matching for file paths
function fuzzyMatch(path: string, query: string): boolean {
  if (!query) return true;
  const lowerPath = path.toLowerCase();
  const lowerQuery = query.toLowerCase();

  // Check if all query characters appear in order
  let pathIdx = 0;
  for (let i = 0; i < lowerQuery.length; i++) {
    const char = lowerQuery[i];
    const found = lowerPath.indexOf(char, pathIdx);
    if (found === -1) return false;
    pathIdx = found + 1;
  }
  return true;
}

// Score fuzzy match for sorting (higher is better)
function fuzzyScore(path: string, query: string): number {
  if (!query) return 0;
  const lowerPath = path.toLowerCase();
  const lowerQuery = query.toLowerCase();

  let score = 0;
  let pathIdx = 0;
  let consecutive = 0;

  for (let i = 0; i < lowerQuery.length; i++) {
    const char = lowerQuery[i];
    const found = lowerPath.indexOf(char, pathIdx);
    if (found === -1) return -1;

    // Bonus for consecutive matches
    if (found === pathIdx) {
      consecutive++;
      score += consecutive * 2;
    } else {
      consecutive = 0;
    }

    // Bonus for matching at start of path segments
    if (found === 0 || lowerPath[found - 1] === "/") {
      score += 5;
    }

    // Penalty for distance
    score -= (found - pathIdx) * 0.5;

    pathIdx = found + 1;
  }

  // Bonus for shorter paths
  score -= path.length * 0.1;

  // Bonus if query matches filename directly
  const filename = path.split("/").pop() || "";
  if (filename.toLowerCase().includes(lowerQuery)) {
    score += 10;
  }

  return score;
}

// Get file icon color based on language
function getFileIconColor(language?: string): string {
  switch (language) {
    case "typescript":
      return "text-blue-400";
    case "javascript":
      return "text-yellow-400";
    case "go":
      return "text-cyan-400";
    case "python":
      return "text-green-400";
    case "rust":
      return "text-orange-400";
    case "java":
    case "kotlin":
      return "text-red-400";
    case "css":
    case "scss":
    case "less":
      return "text-pink-400";
    case "html":
      return "text-orange-300";
    case "json":
    case "yaml":
    case "toml":
      return "text-purple-400";
    case "markdown":
      return "text-slate-400";
    default:
      return "text-slate-500";
  }
}

export function MobileFileSearch({ isOpen, onClose, onSelectFile }: MobileFileSearchProps) {
  const [query, setQuery] = useState("");
  const [isDeepSearch, setIsDeepSearch] = useState(false);
  const [recentFiles, setRecentFiles] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  // Debounce the search query
  const [debouncedQuery, setDebouncedQuery] = useState("");
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 150);
    return () => clearTimeout(timer);
  }, [query]);

  // Fetch files - pass query to backend for server-side filtering before limit is applied
  const { data, isLoading, error } = useFileSearch(debouncedQuery || undefined, isDeepSearch, isOpen);

  // Filter and sort files based on query
  const filteredFiles = useMemo(() => {
    if (!data?.files) return [];

    const matched = data.files
      .filter((file) => fuzzyMatch(file.path, debouncedQuery))
      .map((file) => ({
        file,
        score: fuzzyScore(file.path, debouncedQuery)
      }))
      .sort((a, b) => b.score - a.score)
      .slice(0, 100) // Limit to 100 results for performance
      .map((item) => item.file);

    return matched;
  }, [data?.files, debouncedQuery]);

  // Focus input and load history when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setRecentFiles(getFileHistory());
      // Delay focus to ensure input is mounted and BottomSheet animation is started
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [isOpen]);

  // Handle file selection
  const handleSelectFile = useCallback((file: FileInfo) => {
    addToFileHistory(file.path);
    onSelectFile(file.path);
    onClose();
  }, [onSelectFile, onClose]);

  // Handle selecting a recent file (just path, not FileInfo)
  const handleSelectRecentFile = useCallback((path: string) => {
    addToFileHistory(path);
    onSelectFile(path);
    onClose();
  }, [onSelectFile, onClose]);

  // Handle Enter key to select first result
  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (event.key === "Enter") {
      event.preventDefault();
      // If there's a query and results, select the first one
      if (query && filteredFiles.length > 0) {
        handleSelectFile(filteredFiles[0]);
      } else if (!query && recentFiles.length > 0) {
        // If no query but has recent files, select the first recent
        handleSelectRecentFile(recentFiles[0]);
      }
    }
  }, [query, filteredFiles, recentFiles, handleSelectFile, handleSelectRecentFile]);

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={onClose}
      height="full"
    >
      {/* Search Input - Sticky at top */}
      <div className="sticky top-0 -mx-4 -mt-4 px-4 pt-4 pb-3 bg-slate-950 border-b border-slate-800 z-10">
        <div className="flex items-center gap-3 bg-slate-900 rounded-xl px-4 py-3 border border-slate-800 focus-within:border-slate-700">
          <Search className="h-5 w-5 text-slate-500 flex-shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search files..."
            className="flex-1 bg-transparent text-slate-200 placeholder-slate-500 outline-none text-base"
            autoComplete="off"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            enterKeyHint="go"
            data-testid="mobile-file-search-input"
          />
          {isLoading && <Loader2 className="h-5 w-5 text-slate-500 animate-spin flex-shrink-0" />}
          {query && !isLoading && (
            <button
              type="button"
              onClick={() => setQuery("")}
              className="p-1 text-slate-500 hover:text-slate-300 transition-colors"
              aria-label="Clear search"
            >
              <X className="h-5 w-5" />
            </button>
          )}
        </div>

        {/* Deep Search Toggle */}
        <div className="flex items-center justify-between mt-3">
          <button
            type="button"
            onClick={() => setIsDeepSearch(!isDeepSearch)}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors ${
              isDeepSearch
                ? "bg-amber-600/20 text-amber-400 border border-amber-600/30"
                : "bg-slate-800 text-slate-400 border border-slate-700 active:bg-slate-700"
            }`}
          >
            {isDeepSearch ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            Deep Search
          </button>
          <div className="text-xs text-slate-500">
            {data?.truncated && "(truncated) "}
            {data?.cancelled && "(timed out) "}
            {filteredFiles.length} {filteredFiles.length === 1 ? "file" : "files"}
          </div>
        </div>
        {isDeepSearch && (
          <p className="text-xs text-amber-500 mt-2">
            Includes all files (may be slow)
          </p>
        )}
      </div>

      {/* Results */}
      <div className="pt-4">
        {error && (
          <div className="flex items-center gap-2 py-6 text-red-400">
            <AlertCircle className="h-5 w-5" />
            <span className="text-sm">Failed to load files: {error.message}</span>
          </div>
        )}

        {/* Recent files section - shown when no query and has history */}
        {!error && !query && recentFiles.length > 0 && (
          <div className="mb-6">
            <div className="flex items-center gap-2 mb-2 text-xs text-slate-500">
              <Clock className="h-3.5 w-3.5" />
              Recent Files
            </div>
            <div className="space-y-1">
              {recentFiles.slice(0, 5).map((path) => (
                <button
                  key={`recent-${path}`}
                  type="button"
                  onClick={() => handleSelectRecentFile(path)}
                  className="w-full flex items-center gap-3 px-3 py-3 text-left rounded-xl transition-colors text-slate-300 hover:bg-slate-800/50 active:bg-slate-800"
                >
                  <Clock className="h-4 w-4 flex-shrink-0 text-slate-500" />
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-sm font-medium truncate">
                      {path.split("/").pop()}
                    </div>
                    {path.includes("/") && (
                      <div className="text-xs text-slate-500 truncate mt-0.5">
                        {path.substring(0, path.lastIndexOf("/"))}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Empty state - no results */}
        {!error && !isLoading && filteredFiles.length === 0 && query && (
          <div className="flex flex-col items-center justify-center py-12 text-slate-500">
            <FileCode className="h-10 w-10 mb-3 opacity-50" />
            <p className="text-sm">No matching files found</p>
          </div>
        )}

        {/* Empty state - no files and no history */}
        {!error && !isLoading && filteredFiles.length === 0 && !query && recentFiles.length === 0 && (
          <div className="flex flex-col items-center justify-center py-12 text-slate-500">
            <FileCode className="h-10 w-10 mb-3 opacity-50" />
            <p className="text-sm">No files in repository</p>
          </div>
        )}

        {/* File results */}
        {!error && filteredFiles.length > 0 && (
          <div>
            {/* Section header when showing all files below recent */}
            {!query && recentFiles.length > 0 && (
              <div className="flex items-center gap-2 mb-2 text-xs text-slate-500">
                <FileCode className="h-3.5 w-3.5" />
                All Files
              </div>
            )}
            <div className="space-y-1">
              {filteredFiles.map((file) => (
                <button
                  key={file.path}
                  type="button"
                  onClick={() => handleSelectFile(file)}
                  className="w-full flex items-center gap-3 px-3 py-3 text-left rounded-xl transition-colors text-slate-300 hover:bg-slate-800/50 active:bg-slate-800"
                >
                  <FileCode className={`h-4 w-4 flex-shrink-0 ${getFileIconColor(file.language)}`} />
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-sm font-medium truncate">
                      {file.path.split("/").pop()}
                    </div>
                    {file.path.includes("/") && (
                      <div className="text-xs text-slate-500 truncate mt-0.5">
                        {file.path.substring(0, file.path.lastIndexOf("/"))}
                      </div>
                    )}
                  </div>
                  {file.status === "untracked" && (
                    <span className="text-xs text-emerald-500 px-1.5 py-0.5 bg-emerald-500/10 rounded flex-shrink-0">
                      new
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </BottomSheet>
  );
}
