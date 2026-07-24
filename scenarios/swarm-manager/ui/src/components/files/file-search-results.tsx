/**
 * FileSearchResults
 *
 * Search results list and recent-files section,
 * extracted from BacklogFileBrowser.
 */

import { FileText, Search, X } from "lucide-react";
import { Input } from "../ui/input";
import type { BacklogFile } from "../../types";

export interface FileSearchResultsProps {
  fileSearch: string;
  onFileSearchChange: (value: string) => void;
  searchResults: BacklogFile[];
  recentFiles: BacklogFile[];
  onFileSelect: (file: BacklogFile) => void;
}

export function FileSearchResults({
  fileSearch,
  onFileSearchChange,
  searchResults: _searchResults,
  recentFiles,
  onFileSelect,
}: FileSearchResultsProps) {
  const hasSearch = fileSearch.trim().length > 0;

  return (
    <div className="space-y-3 lg:hidden">
      <Input
        type="text"
        value={fileSearch}
        onChange={(event) => onFileSearchChange(event.target.value)}
        placeholder="Search files"
        leftIcon={<Search className="h-4 w-4" />}
        rightSlot={
          hasSearch ? (
            <button
              type="button"
              onClick={() => onFileSearchChange("")}
              className="rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
              aria-label="Clear search"
            >
              <X className="h-4 w-4" />
            </button>
          ) : null
        }
      />
      {recentFiles.length > 0 && !hasSearch && (
        <div className="space-y-2">
          <p className="text-xs uppercase tracking-wider text-slate-500">Recent files</p>
          <div className="space-y-1">
            {recentFiles.map((file) => (
              <button
                key={file.path}
                type="button"
                onClick={() => onFileSelect(file)}
                className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/50 hover:bg-slate-800/70"
              >
                <FileText className="h-4 w-4 text-slate-400" />
                <span className="truncate">{file.name}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

/** Inline search results when a query is active. */
export function FileSearchResultsList({
  searchResults,
  fileSearch,
  onFileSelect,
}: {
  searchResults: BacklogFile[];
  fileSearch: string;
  onFileSelect: (file: BacklogFile) => void;
}) {
  if (searchResults.length > 0) {
    return (
      <div className="space-y-1">
        {searchResults.map((file) => (
          <button
            key={file.path}
            type="button"
            onClick={() => onFileSelect(file)}
            className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/50 hover:bg-slate-800/70"
          >
            <FileText className="h-4 w-4 text-slate-400" />
            <div className="flex min-w-0 flex-1 flex-col">
              <span className="truncate">{file.name}</span>
              <span className="truncate text-xs text-slate-500">{file.path}</span>
            </div>
          </button>
        ))}
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center text-sm text-slate-500">
      No files match &quot;{fileSearch.trim()}&quot;.
    </div>
  );
}
