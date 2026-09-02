import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  Search,
  X,
  FileCode,
  Loader2,
  AlertCircle,
  Clock,
  ChevronDown,
  ChevronUp,
  FileText,
  CaseSensitive,
  WholeWord,
  Regex,
  ChevronRight
} from "lucide-react";
import { BottomSheet } from "./ui/bottom-sheet";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { useFileSearch, useContentSearch } from "../lib/hooks";
import type { FileInfo, ContentSearchMatch, ContentSearchRequest } from "../lib/api";
import {
  type SearchMode,
  getFileHistory,
  addToFileHistory,
  getSavedSearchMode,
  saveSearchMode,
  fuzzyMatch,
  fuzzyScore,
  getFileIconColor,
  groupMatchesByFile,
  highlightContent,
} from "../lib/fileSearchUtils";

interface MobileFileSearchProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectFile: (path: string, lineNumber?: number) => void;
  repoId?: string | null;
}

export function MobileFileSearch({
  isOpen,
  onClose,
  onSelectFile,
  repoId
}: MobileFileSearchProps) {
  const [searchMode, setSearchMode] = useState<SearchMode>(getSavedSearchMode);
  const [query, setQuery] = useState("");
  const [isDeepSearch, setIsDeepSearch] = useState(false);
  const [recentFiles, setRecentFiles] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  // Content search options
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [caseSensitive, setCaseSensitive] = useState(false);
  const [wholeWord, setWholeWord] = useState(false);
  const [regex, setRegex] = useState(false);
  const [include, setInclude] = useState("");
  const [exclude, setExclude] = useState("");

  // Expanded file groups for content search results
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());

  // Debounce the search query
  const [debouncedQuery, setDebouncedQuery] = useState("");
  useEffect(() => {
    const delay = searchMode === "content" ? 300 : 150;
    const timer = setTimeout(() => {
      setDebouncedQuery(query);
    }, delay);
    return () => clearTimeout(timer);
  }, [query, searchMode]);

  // File search
  const {
    data: fileData,
    isLoading: fileLoading,
    error: fileError
  } = useFileSearch(
    searchMode === "files" ? debouncedQuery || undefined : undefined,
    isDeepSearch,
    isOpen && searchMode === "files",
    repoId
  );

  // Content search options
  const contentSearchOptions: Omit<ContentSearchRequest, "query"> = useMemo(() => ({
    case_sensitive: caseSensitive,
    whole_word: wholeWord,
    regex: regex,
    include: include || undefined,
    exclude: exclude || undefined
  }), [caseSensitive, wholeWord, regex, include, exclude]);

  // Content search
  const {
    data: contentData,
    isLoading: contentLoading,
    error: contentError
  } = useContentSearch(
    searchMode === "content" ? debouncedQuery : "",
    contentSearchOptions,
    isOpen && searchMode === "content" && debouncedQuery.length >= 2,
    repoId
  );

  // Filter and sort files based on query
  const filteredFiles = useMemo(() => {
    if (searchMode !== "files" || !fileData?.files) return [];

    const matched = fileData.files
      .filter((file) => fuzzyMatch(file.path, debouncedQuery))
      .map((file) => ({
        file,
        score: fuzzyScore(file.path, debouncedQuery)
      }))
      .sort((a, b) => b.score - a.score)
      .slice(0, 100)
      .map((item) => item.file);

    return matched;
  }, [fileData?.files, debouncedQuery, searchMode]);

  // Group content search results
  const groupedContentResults = useMemo(() => {
    if (searchMode !== "content" || !contentData?.matches) return [];
    return groupMatchesByFile(contentData.matches);
  }, [contentData?.matches, searchMode]);

  // Focus input and load history when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setDebouncedQuery("");
      setRecentFiles(getFileHistory());
      setExpandedFiles(new Set());
      setTimeout(() => {
        inputRef.current?.focus();
      }, 100);
    }
  }, [isOpen]);

  // Handle mode switch
  const handleModeSwitch = useCallback((mode: SearchMode) => {
    setSearchMode(mode);
    saveSearchMode(mode);
    setQuery("");
    setDebouncedQuery("");
    setTimeout(() => inputRef.current?.focus(), 50);
  }, []);

  // Handle file selection
  const handleSelectFile = useCallback((file: FileInfo) => {
    addToFileHistory(file.path);
    onSelectFile(file.path);
    onClose();
  }, [onSelectFile, onClose]);

  // Handle selecting a recent file
  const handleSelectRecentFile = useCallback((path: string) => {
    addToFileHistory(path);
    onSelectFile(path);
    onClose();
  }, [onSelectFile, onClose]);

  // Handle content match selection
  const handleSelectMatch = useCallback((match: ContentSearchMatch) => {
    addToFileHistory(match.path);
    onSelectFile(match.path, match.line_number);
    onClose();
  }, [onSelectFile, onClose]);

  // Toggle file expansion in content results
  const toggleFileExpansion = useCallback((path: string) => {
    setExpandedFiles((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  // Handle Enter key to select first result
  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    if (event.key === "Enter") {
      event.preventDefault();
      if (searchMode === "files") {
        if (query && filteredFiles.length > 0) {
          const firstFile = filteredFiles[0];
          if (firstFile) {
            handleSelectFile(firstFile);
          }
        } else if (!query && recentFiles.length > 0) {
          const recent = recentFiles[0];
          if (recent) {
            handleSelectRecentFile(recent);
          }
        }
      } else if (searchMode === "content") {
        const firstGroup = groupedContentResults[0];
        const firstMatch = firstGroup?.matches[0];
        if (firstMatch) {
          handleSelectMatch(firstMatch);
        }
      }
    }
  }, [query, searchMode, filteredFiles, recentFiles, groupedContentResults, handleSelectFile, handleSelectRecentFile, handleSelectMatch]);

  const isLoading = searchMode === "files" ? fileLoading : contentLoading;
  const error = searchMode === "files" ? fileError : contentError;

  return (
    <BottomSheet
      isOpen={isOpen}
      onClose={onClose}
      height="full"
    >
      {/* Sticky header */}
      <div className="sticky top-0 -mx-4 -mt-4 px-4 pt-4 pb-3 bg-slate-950 border-b border-slate-800 z-10">
        {/* Tabs */}
        <div className="mb-3 rounded-lg bg-slate-900 px-2">
          <Tabs
            density="compact"
            items={[
              { id: "files", label: "Files", icon: <FileCode aria-hidden="true" /> },
              { id: "content", label: "Content", icon: <FileText aria-hidden="true" /> },
            ]}
            active={searchMode}
            onChange={(next) => handleModeSwitch(next as SearchMode)}
            ariaLabel="Search mode"
          />
        </div>

        {/* Search Input */}
        <div className="flex items-center gap-3 bg-slate-900 rounded-xl px-4 py-3 border border-slate-800 focus-within:border-slate-700">
          <Search className="h-5 w-5 text-slate-500 flex-shrink-0" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={searchMode === "files" ? "Search files..." : "Search in files..."}
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

        {/* Content search options */}
        {searchMode === "content" && (
          <div className="mt-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setCaseSensitive(!caseSensitive)}
                  title="Case sensitive"
                  className={`p-2 rounded-lg transition-colors ${
                    caseSensitive
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "bg-slate-800 text-slate-500 border border-slate-700"
                  }`}
                >
                  <CaseSensitive className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setWholeWord(!wholeWord)}
                  title="Whole word"
                  className={`p-2 rounded-lg transition-colors ${
                    wholeWord
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "bg-slate-800 text-slate-500 border border-slate-700"
                  }`}
                >
                  <WholeWord className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setRegex(!regex)}
                  title="Regular expression"
                  className={`p-2 rounded-lg transition-colors ${
                    regex
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "bg-slate-800 text-slate-500 border border-slate-700"
                  }`}
                >
                  <Regex className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className={`flex items-center gap-1 px-3 py-2 rounded-lg text-sm transition-colors ${
                    showAdvanced
                      ? "bg-slate-800 text-slate-300"
                      : "bg-slate-800 text-slate-500"
                  }`}
                >
                  {showAdvanced ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  Filters
                </button>
              </div>
              <div className="text-xs text-slate-500">
                {contentData?.total ?? 0} {(contentData?.total ?? 0) === 1 ? "match" : "matches"}
              </div>
            </div>

            {showAdvanced && (
              <div className="flex gap-3 mt-3">
                <div className="flex-1">
                  <label className="block text-xs text-slate-500 mb-1">Include</label>
                  <input
                    type="text"
                    value={include}
                    onChange={(e) => setInclude(e.target.value)}
                    placeholder="*.go, *.ts"
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 placeholder-slate-500 outline-none focus:border-slate-600"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-slate-500 mb-1">Exclude</label>
                  <input
                    type="text"
                    value={exclude}
                    onChange={(e) => setExclude(e.target.value)}
                    placeholder="*.test.ts"
                    className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded-lg text-sm text-slate-300 placeholder-slate-500 outline-none focus:border-slate-600"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* File search deep toggle */}
        {searchMode === "files" && (
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
              {fileData?.truncated && "(truncated) "}
              {fileData?.cancelled && "(timed out) "}
              {filteredFiles.length} {filteredFiles.length === 1 ? "file" : "files"}
            </div>
          </div>
        )}
        {searchMode === "files" && isDeepSearch && (
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
            <span className="text-sm">
              {searchMode === "files" ? "Failed to load files" : "Search failed"}: {error.message}
            </span>
          </div>
        )}

        {/* File Search Results */}
        {searchMode === "files" && !error && (
          <>
            {/* Recent files section */}
            {!query && recentFiles.length > 0 && (
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
            {!fileLoading && filteredFiles.length === 0 && query && (
              <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                <FileCode className="h-10 w-10 mb-3 opacity-50" />
                <p className="text-sm">No matching files found</p>
              </div>
            )}

            {/* Empty state - no files and no history */}
            {!fileLoading && filteredFiles.length === 0 && !query && recentFiles.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                <FileCode className="h-10 w-10 mb-3 opacity-50" />
                <p className="text-sm">No files in repository</p>
              </div>
            )}

            {/* File results */}
            {filteredFiles.length > 0 && (
              <div>
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
          </>
        )}

        {/* Content Search Results */}
        {searchMode === "content" && !error && (
          <>
            {debouncedQuery.length < 2 && (
              <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                <Search className="h-10 w-10 mb-3 opacity-50" />
                <p className="text-sm">Type at least 2 characters to search</p>
              </div>
            )}

            {debouncedQuery.length >= 2 && !contentLoading && groupedContentResults.length === 0 && (
              <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                <FileText className="h-10 w-10 mb-3 opacity-50" />
                <p className="text-sm">No matches found</p>
              </div>
            )}

            {groupedContentResults.length > 0 && (
              <div className="space-y-2">
                {groupedContentResults.map((group) => {
                  const isExpanded = expandedFiles.has(group.path) || groupedContentResults.length <= 5;

                  return (
                    <div key={group.path} className="bg-slate-900/50 rounded-xl overflow-hidden">
                      {/* File header */}
                      <button
                        type="button"
                        onClick={() => toggleFileExpansion(group.path)}
                        className="w-full flex items-center gap-2 px-4 py-3 text-left transition-colors text-slate-300 active:bg-slate-800"
                      >
                        <ChevronRight
                          className={`h-4 w-4 text-slate-500 transition-transform ${
                            isExpanded ? "rotate-90" : ""
                          }`}
                        />
                        <FileCode className="h-4 w-4 flex-shrink-0 text-slate-500" />
                        <span className="font-mono text-sm truncate flex-1">{group.path}</span>
                        <span className="text-xs text-slate-500">
                          ({group.matches.length})
                        </span>
                      </button>

                      {/* Matches */}
                      {isExpanded && (
                        <div className="border-t border-slate-800">
                          {group.matches.map((match) => (
                            <button
                              key={`${match.path}:${match.line_number}`}
                              type="button"
                              onClick={() => handleSelectMatch(match)}
                              className="w-full flex items-start gap-3 px-4 py-2.5 text-left transition-colors text-slate-400 active:bg-slate-800 border-b border-slate-800/50 last:border-b-0"
                            >
                              <span className="text-xs text-slate-500 font-mono min-w-[2.5rem] text-right pt-0.5">
                                {match.line_number}
                              </span>
                              <span className="font-mono text-sm flex-1 break-all">
                                {highlightContent(match.content, debouncedQuery, regex, caseSensitive)}
                              </span>
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </>
        )}
      </div>
    </BottomSheet>
  );
}
