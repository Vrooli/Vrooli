import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { Search, X, FileCode, Loader2, AlertCircle, ChevronDown, ChevronUp } from "lucide-react";
import { useFileSearch } from "../lib/hooks";
import type { FileInfo } from "../lib/api";

interface FileSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectFile: (path: string) => void;
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

export function FileSearchModal({ isOpen, onClose, onSelectFile }: FileSearchModalProps) {
  const [query, setQuery] = useState("");
  const [isDeepSearch, setIsDeepSearch] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Debounce the search query
  const [debouncedQuery, setDebouncedQuery] = useState("");
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, 150);
    return () => clearTimeout(timer);
  }, [query]);

  // Fetch files - pass query to backend for server-side filtering before limit is applied
  // This ensures files matching the query aren't excluded due to the 1000-file limit
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

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [filteredFiles.length]);

  // Focus input when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setSelectedIndex(0);
      setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
    }
  }, [isOpen]);

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const selectedElement = listRef.current.querySelector(`[data-index="${selectedIndex}"]`);
    if (selectedElement) {
      selectedElement.scrollIntoView({ block: "nearest" });
    }
  }, [selectedIndex]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        setSelectedIndex((prev) => Math.min(prev + 1, filteredFiles.length - 1));
        break;
      case "ArrowUp":
        event.preventDefault();
        setSelectedIndex((prev) => Math.max(prev - 1, 0));
        break;
      case "Enter":
        event.preventDefault();
        if (filteredFiles[selectedIndex]) {
          onSelectFile(filteredFiles[selectedIndex].path);
          onClose();
        }
        break;
      case "Escape":
        event.preventDefault();
        onClose();
        break;
    }
  }, [filteredFiles, selectedIndex, onSelectFile, onClose]);

  // Handle click outside
  const handleBackdropClick = useCallback((event: React.MouseEvent) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }, [onClose]);

  // Handle file selection
  const handleSelectFile = useCallback((file: FileInfo) => {
    onSelectFile(file.path);
    onClose();
  }, [onSelectFile, onClose]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-start justify-center pt-[15vh]"
      onClick={handleBackdropClick}
      data-testid="file-search-modal"
    >
      <div
        className="bg-slate-900 border border-slate-700 rounded-lg shadow-2xl w-full max-w-2xl max-h-[60vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-700">
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
            spellCheck={false}
            data-testid="file-search-input"
          />
          {isLoading && <Loader2 className="h-5 w-5 text-slate-500 animate-spin" />}
          <button
            type="button"
            onClick={onClose}
            className="p-1 text-slate-500 hover:text-slate-300 transition-colors"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Deep Search Toggle */}
        <div className="flex items-center justify-between px-4 py-2 border-b border-slate-800 bg-slate-800/30">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setIsDeepSearch(!isDeepSearch)}
              className={`flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors ${
                isDeepSearch
                  ? "bg-amber-600/20 text-amber-400 border border-amber-600/30"
                  : "bg-slate-700/50 text-slate-400 border border-slate-600/50 hover:bg-slate-700"
              }`}
            >
              {isDeepSearch ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
              Deep Search
            </button>
            {isDeepSearch && (
              <span className="text-xs text-amber-500">
                Includes all files (may be slow)
              </span>
            )}
          </div>
          <div className="text-xs text-slate-500">
            {data?.truncated && "(truncated) "}
            {data?.cancelled && "(timed out) "}
            {filteredFiles.length} {filteredFiles.length === 1 ? "file" : "files"}
          </div>
        </div>

        {/* Results */}
        <div ref={listRef} className="flex-1 overflow-y-auto min-h-0">
          {error && (
            <div className="flex items-center gap-2 px-4 py-6 text-red-400">
              <AlertCircle className="h-5 w-5" />
              <span className="text-sm">Failed to load files: {error.message}</span>
            </div>
          )}

          {!error && !isLoading && filteredFiles.length === 0 && (
            <div className="flex flex-col items-center justify-center py-12 text-slate-500">
              <FileCode className="h-10 w-10 mb-3 opacity-50" />
              <p className="text-sm">
                {query ? "No matching files found" : "No files in repository"}
              </p>
            </div>
          )}

          {!error && filteredFiles.length > 0 && (
            <div className="py-1" role="listbox">
              {filteredFiles.map((file, index) => (
                <button
                  key={file.path}
                  type="button"
                  data-index={index}
                  role="option"
                  aria-selected={index === selectedIndex}
                  onClick={() => handleSelectFile(file)}
                  onMouseEnter={() => setSelectedIndex(index)}
                  className={`w-full flex items-center gap-3 px-4 py-2 text-left transition-colors ${
                    index === selectedIndex
                      ? "bg-blue-600/20 text-slate-100"
                      : "text-slate-300 hover:bg-slate-800/50"
                  }`}
                >
                  <FileCode className={`h-4 w-4 flex-shrink-0 ${getFileIconColor(file.language)}`} />
                  <div className="flex-1 min-w-0 flex items-baseline gap-2">
                    <span className="font-mono text-sm font-medium truncate">
                      {file.path.split("/").pop()}
                    </span>
                    {file.path.includes("/") && (
                      <span className="text-xs text-slate-500 truncate">
                        {file.path.substring(0, file.path.lastIndexOf("/"))}
                      </span>
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
          )}
        </div>

        {/* Footer with keyboard hints */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-slate-700 bg-slate-800/30 text-xs text-slate-500">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 bg-slate-700 rounded text-slate-400">↑↓</kbd>
              navigate
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 bg-slate-700 rounded text-slate-400">↵</kbd>
              select
            </span>
            <span className="flex items-center gap-1">
              <kbd className="px-1.5 py-0.5 bg-slate-700 rounded text-slate-400">esc</kbd>
              close
            </span>
          </div>
          <span>
            <kbd className="px-1.5 py-0.5 bg-slate-700 rounded text-slate-400">⌘K</kbd>
            to open
          </span>
        </div>
      </div>
    </div>
  );
}
