/**
 * File search input, recent-files list, and inline search results.
 *
 * Shared by the entity file browser on both the backlog and goal detail
 * pages. Presentation only — all state lives in EntityFileBrowser.
 */

import { FileText, Search, X } from "lucide-react";
import { Input } from "../ui/input";
import type { BacklogFile } from "../../types";

export interface FileSearchInputProps {
  fileSearch: string;
  onFileSearchChange: (value: string) => void;
}

/** Search box for the browser toolbar. */
export function FileSearchInput({ fileSearch, onFileSearchChange }: FileSearchInputProps) {
  const hasSearch = fileSearch.trim().length > 0;

  return (
    <Input
      type="text"
      size="sm"
      value={fileSearch}
      onChange={(event) => onFileSearchChange(event.target.value)}
      placeholder="Search files"
      aria-label="Search files"
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
  );
}

export interface RecentFilesListProps {
  recentFiles: BacklogFile[];
  onFileSelect: (file: BacklogFile) => void;
}

/** Recently opened files, shown above the tree when no search is active. */
export function RecentFilesList({ recentFiles, onFileSelect }: RecentFilesListProps) {
  if (recentFiles.length === 0) return null;

  return (
    <div className="space-y-1" data-testid="file-recent-list">
      <p className="text-xs uppercase tracking-wider text-slate-500">Recent files</p>
      {recentFiles.map((file) => (
        <button
          key={file.path}
          type="button"
          onClick={() => onFileSelect(file)}
          className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-slate-800/40 px-3 py-2 text-left text-sm text-slate-200 hover:border-cyan-500/50 hover:bg-slate-800/70"
        >
          <FileText className="h-4 w-4 shrink-0 text-slate-400" />
          <span className="truncate">{file.name}</span>
        </button>
      ))}
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
            <FileText className="h-4 w-4 shrink-0 text-slate-400" />
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
