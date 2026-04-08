import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  Search,
  X,
  FileCode,
  Loader2,
  AlertCircle,
  ChevronDown,
  ChevronUp,
  Clock,
  FileText,
  CaseSensitive,
  WholeWord,
  Regex,
  ChevronRight
} from "lucide-react";
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

interface FileSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectFile: (path: string, lineNumber?: number) => void;
  repoId?: string | null;
}

export function FileSearchModal({
  isOpen,
  onClose,
  onSelectFile,
  repoId
}: FileSearchModalProps) {
  const [searchMode, setSearchMode] = useState<SearchMode>(getSavedSearchMode);
  const [query, setQuery] = useState("");
  const [isDeepSearch, setIsDeepSearch] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [recentFiles, setRecentFiles] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

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
      .slice(0, 100) // Limit to 100 results for performance
      .map((item) => item.file);

    return matched;
  }, [fileData?.files, debouncedQuery, searchMode]);

  // Group content search results
  const groupedContentResults = useMemo(() => {
    if (searchMode !== "content" || !contentData?.matches) return [];
    return groupMatchesByFile(contentData.matches);
  }, [contentData?.matches, searchMode]);

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [filteredFiles.length, groupedContentResults.length]);

  // Focus input and load history when modal opens
  useEffect(() => {
    if (isOpen) {
      setQuery("");
      setDebouncedQuery("");
      setSelectedIndex(0);
      setRecentFiles(getFileHistory());
      setExpandedFiles(new Set());
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

  // Calculate navigation bounds
  const recentFilesCount = searchMode === "files" && !query ? Math.min(recentFiles.length, 5) : 0;

  // For content search, count all visible match items
  const contentItemCount = useMemo(() => {
    let count = 0;
    for (const group of groupedContentResults) {
      count++; // File header
      if (expandedFiles.has(group.path) || groupedContentResults.length <= 5) {
        count += group.matches.length;
      }
    }
    return count;
  }, [groupedContentResults, expandedFiles]);

  const totalNavigableItems = searchMode === "files"
    ? recentFilesCount + filteredFiles.length
    : contentItemCount;

  // Handle mode switch
  const handleModeSwitch = useCallback((mode: SearchMode) => {
    setSearchMode(mode);
    saveSearchMode(mode);
    setSelectedIndex(0);
    setQuery("");
    setDebouncedQuery("");
    setTimeout(() => inputRef.current?.focus(), 50);
  }, []);

  // Handle Enter key selection
  const handleEnterKey = useCallback(() => {
    if (searchMode === "files") {
      // File search mode
      if (selectedIndex < recentFilesCount) {
        const path = recentFiles[selectedIndex];
        if (path) {
          addToFileHistory(path);
          onSelectFile(path);
          onClose();
        }
      } else {
        const fileIndex = selectedIndex - recentFilesCount;
        if (filteredFiles[fileIndex]) {
          addToFileHistory(filteredFiles[fileIndex].path);
          onSelectFile(filteredFiles[fileIndex].path);
          onClose();
        }
      }
    } else {
      // Content search mode - find the item at selectedIndex
      let idx = 0;
      for (const group of groupedContentResults) {
        if (idx === selectedIndex) {
          // Selected a file header - toggle expansion
          setExpandedFiles((prev) => {
            const next = new Set(prev);
            if (next.has(group.path)) {
              next.delete(group.path);
            } else {
              next.add(group.path);
            }
            return next;
          });
          return;
        }
        idx++;
        const isExpanded = expandedFiles.has(group.path) || groupedContentResults.length <= 5;
        if (isExpanded) {
          for (const match of group.matches) {
            if (idx === selectedIndex) {
              // Selected a match - navigate to it
              addToFileHistory(match.path);
              onSelectFile(match.path, match.line_number);
              onClose();
              return;
            }
            idx++;
          }
        }
      }
    }
  }, [
    searchMode,
    selectedIndex,
    recentFilesCount,
    recentFiles,
    filteredFiles,
    groupedContentResults,
    expandedFiles,
    onSelectFile,
    onClose
  ]);

  // Handle keyboard navigation
  const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
    // Tab to switch modes
    if (event.key === "Tab" && !event.shiftKey && !event.ctrlKey && !event.metaKey) {
      event.preventDefault();
      handleModeSwitch(searchMode === "files" ? "content" : "files");
      return;
    }

    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        setSelectedIndex((prev) => Math.min(prev + 1, totalNavigableItems - 1));
        break;
      case "ArrowUp":
        event.preventDefault();
        setSelectedIndex((prev) => Math.max(prev - 1, 0));
        break;
      case "Enter":
        event.preventDefault();
        handleEnterKey();
        break;
      case "Escape":
        event.preventDefault();
        onClose();
        break;
    }
  }, [searchMode, totalNavigableItems, handleModeSwitch, handleEnterKey, onClose]);

  // Handle click outside
  const handleBackdropClick = useCallback((event: React.MouseEvent) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }, [onClose]);

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

  if (!isOpen) return null;

  const isLoading = searchMode === "files" ? fileLoading : contentLoading;
  const error = searchMode === "files" ? fileError : contentError;

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
        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          <button
            type="button"
            onClick={() => handleModeSwitch("files")}
            className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              searchMode === "files"
                ? "text-blue-400 border-b-2 border-blue-400 -mb-px"
                : "text-slate-400 hover:text-slate-300"
            }`}
          >
            <FileCode className="h-4 w-4" />
            Files
          </button>
          <button
            type="button"
            onClick={() => handleModeSwitch("content")}
            className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              searchMode === "content"
                ? "text-blue-400 border-b-2 border-blue-400 -mb-px"
                : "text-slate-400 hover:text-slate-300"
            }`}
          >
            <FileText className="h-4 w-4" />
            Content
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-3 py-2 text-slate-500 hover:text-slate-300 transition-colors"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-700">
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
            spellCheck={false}
            data-testid="file-search-input"
          />
          {isLoading && <Loader2 className="h-5 w-5 text-slate-500 animate-spin" />}
        </div>

        {/* Advanced Options (Content mode only) */}
        {searchMode === "content" && (
          <div className="px-4 py-2 border-b border-slate-800 bg-slate-800/30">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setCaseSensitive(!caseSensitive)}
                  title="Case sensitive"
                  className={`p-1.5 rounded transition-colors ${
                    caseSensitive
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "text-slate-500 hover:text-slate-400 hover:bg-slate-700/50"
                  }`}
                >
                  <CaseSensitive className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setWholeWord(!wholeWord)}
                  title="Whole word"
                  className={`p-1.5 rounded transition-colors ${
                    wholeWord
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "text-slate-500 hover:text-slate-400 hover:bg-slate-700/50"
                  }`}
                >
                  <WholeWord className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setRegex(!regex)}
                  title="Regular expression"
                  className={`p-1.5 rounded transition-colors ${
                    regex
                      ? "bg-blue-600/20 text-blue-400 border border-blue-600/30"
                      : "text-slate-500 hover:text-slate-400 hover:bg-slate-700/50"
                  }`}
                >
                  <Regex className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setShowAdvanced(!showAdvanced)}
                  className="flex items-center gap-1 px-2 py-1 text-xs text-slate-500 hover:text-slate-400 transition-colors"
                >
                  {showAdvanced ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                  Filters
                </button>
              </div>
              <div className="text-xs text-slate-500">
                {contentData?.truncated && "(truncated) "}
                {contentData?.cancelled && "(timed out) "}
                {contentData?.total ?? 0} {(contentData?.total ?? 0) === 1 ? "match" : "matches"}
              </div>
            </div>

            {showAdvanced && (
              <div className="flex gap-3 mt-2">
                <div className="flex-1">
                  <label className="block text-xs text-slate-500 mb-1">Include</label>
                  <input
                    type="text"
                    value={include}
                    onChange={(e) => setInclude(e.target.value)}
                    placeholder="*.go, *.ts"
                    className="w-full px-2 py-1 bg-slate-800 border border-slate-700 rounded text-sm text-slate-300 placeholder-slate-500 outline-none focus:border-slate-600"
                  />
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-slate-500 mb-1">Exclude</label>
                  <input
                    type="text"
                    value={exclude}
                    onChange={(e) => setExclude(e.target.value)}
                    placeholder="*.test.ts"
                    className="w-full px-2 py-1 bg-slate-800 border border-slate-700 rounded text-sm text-slate-300 placeholder-slate-500 outline-none focus:border-slate-600"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Deep Search Toggle (Files mode only) */}
        {searchMode === "files" && (
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
              {fileData?.truncated && "(truncated) "}
              {fileData?.cancelled && "(timed out) "}
              {filteredFiles.length} {filteredFiles.length === 1 ? "file" : "files"}
            </div>
          </div>
        )}

        {/* Results */}
        <div ref={listRef} className="flex-1 overflow-y-auto min-h-0">
          {error && (
            <div className="flex items-center gap-2 px-4 py-6 text-red-400">
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
                <div className="border-b border-slate-800">
                  <div className="flex items-center gap-2 px-4 py-2 text-xs text-slate-500">
                    <Clock className="h-3 w-3" />
                    Recent Files
                  </div>
                  <div className="pb-1" role="listbox">
                    {recentFiles.slice(0, 5).map((path, index) => (
                      <button
                        key={`recent-${path}`}
                        type="button"
                        data-index={index}
                        role="option"
                        aria-selected={index === selectedIndex}
                        onClick={() => handleSelectRecentFile(path)}
                        onMouseEnter={() => setSelectedIndex(index)}
                        className={`w-full flex items-center gap-3 px-4 py-2 text-left transition-colors ${
                          index === selectedIndex
                            ? "bg-blue-600/20 text-slate-100"
                            : "text-slate-300 hover:bg-slate-800/50"
                        }`}
                      >
                        <Clock className="h-4 w-4 flex-shrink-0 text-slate-500" />
                        <div className="flex-1 min-w-0 flex items-baseline gap-2">
                          <span className="font-mono text-sm font-medium truncate">
                            {path.split("/").pop()}
                          </span>
                          {path.includes("/") && (
                            <span className="text-xs text-slate-500 truncate">
                              {path.substring(0, path.lastIndexOf("/"))}
                            </span>
                          )}
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {!isLoading && filteredFiles.length === 0 && query && (
                <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                  <FileCode className="h-10 w-10 mb-3 opacity-50" />
                  <p className="text-sm">No matching files found</p>
                </div>
              )}

              {!isLoading && filteredFiles.length === 0 && !query && recentFiles.length === 0 && (
                <div className="flex flex-col items-center justify-center py-12 text-slate-500">
                  <FileCode className="h-10 w-10 mb-3 opacity-50" />
                  <p className="text-sm">No files in repository</p>
                </div>
              )}

              {filteredFiles.length > 0 && (
                <div className="py-1" role="listbox">
                  {!query && recentFiles.length > 0 && (
                    <div className="flex items-center gap-2 px-4 py-2 text-xs text-slate-500">
                      <FileCode className="h-3 w-3" />
                      All Files
                    </div>
                  )}
                  {filteredFiles.map((file, index) => {
                    const displayIndex = !query && recentFiles.length > 0
                      ? index + Math.min(recentFiles.length, 5)
                      : index;
                    return (
                      <button
                        key={file.path}
                        type="button"
                        data-index={displayIndex}
                        role="option"
                        aria-selected={displayIndex === selectedIndex}
                        onClick={() => handleSelectFile(file)}
                        onMouseEnter={() => setSelectedIndex(displayIndex)}
                        className={`w-full flex items-center gap-3 px-4 py-2 text-left transition-colors ${
                          displayIndex === selectedIndex
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
                    );
                  })}
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
                <div className="py-1">
                  {(() => {
                    let navIndex = 0;
                    return groupedContentResults.map((group) => {
                      const fileHeaderIndex = navIndex;
                      navIndex++;
                      const isExpanded = expandedFiles.has(group.path) || groupedContentResults.length <= 5;

                      return (
                        <div key={group.path} className="border-b border-slate-800 last:border-b-0">
                          {/* File header */}
                          <button
                            type="button"
                            data-index={fileHeaderIndex}
                            onClick={() => toggleFileExpansion(group.path)}
                            onMouseEnter={() => setSelectedIndex(fileHeaderIndex)}
                            className={`w-full flex items-center gap-2 px-4 py-2 text-left transition-colors ${
                              fileHeaderIndex === selectedIndex
                                ? "bg-blue-600/20 text-slate-100"
                                : "text-slate-300 hover:bg-slate-800/50"
                            }`}
                          >
                            <ChevronRight
                              className={`h-4 w-4 text-slate-500 transition-transform ${
                                isExpanded ? "rotate-90" : ""
                              }`}
                            />
                            <FileCode className="h-4 w-4 flex-shrink-0 text-slate-500" />
                            <span className="font-mono text-sm truncate">{group.path}</span>
                            <span className="text-xs text-slate-500 ml-auto">
                              ({group.matches.length} {group.matches.length === 1 ? "match" : "matches"})
                            </span>
                          </button>

                          {/* Matches */}
                          {isExpanded && (
                            <div className="pl-8 pb-1">
                              {group.matches.map((match) => {
                                const matchIndex = navIndex;
                                navIndex++;
                                return (
                                  <button
                                    key={`${match.path}:${match.line_number}`}
                                    type="button"
                                    data-index={matchIndex}
                                    onClick={() => handleSelectMatch(match)}
                                    onMouseEnter={() => setSelectedIndex(matchIndex)}
                                    className={`w-full flex items-start gap-3 px-4 py-1.5 text-left transition-colors ${
                                      matchIndex === selectedIndex
                                        ? "bg-blue-600/20 text-slate-100"
                                        : "text-slate-400 hover:bg-slate-800/50"
                                    }`}
                                  >
                                    <span className="text-xs text-slate-500 font-mono min-w-[3rem] text-right">
                                      {match.line_number}
                                    </span>
                                    <span className="font-mono text-sm truncate flex-1">
                                      {highlightContent(match.content, debouncedQuery, regex, caseSensitive)}
                                    </span>
                                  </button>
                                );
                              })}
                            </div>
                          )}
                        </div>
                      );
                    });
                  })()}
                </div>
              )}
            </>
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
              <kbd className="px-1.5 py-0.5 bg-slate-700 rounded text-slate-400">tab</kbd>
              switch
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
