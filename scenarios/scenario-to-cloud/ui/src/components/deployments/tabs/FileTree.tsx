import {
  Folder,
  FolderOpen,
  File,
  FileJson,
  FileCode,
  FileText,
  FileArchive,
  Link2,
  ChevronUp,
} from "lucide-react";
import type { FileEntry } from "../../../lib/api";
import { cn } from "../../../lib/utils";
import { formatBytes } from "../../../hooks/useLiveState";

interface FileTreeProps {
  entries: FileEntry[];
  currentPath: string;
  onNavigate: (path: string) => void;
  onSelectFile: (path: string) => void;
  selectedFile: string | null;
}

export function FileTree({
  entries,
  currentPath,
  onNavigate,
  onSelectFile,
  selectedFile,
}: FileTreeProps) {
  // Get parent path for navigation
  const parentPath = getParentPath(currentPath);
  const canGoUp = currentPath !== "/" && currentPath.includes("/");

  // Sort entries: directories first, then files
  const sortedEntries = [...entries].sort((a, b) => {
    if (a.type === "directory" && b.type !== "directory") return -1;
    if (a.type !== "directory" && b.type === "directory") return 1;
    return a.name.localeCompare(b.name);
  });

  if (entries.length === 0) {
    return (
      <div className="text-slate-500 text-sm italic">
        Empty directory
      </div>
    );
  }

  return (
    <div className="space-y-0.5">
      {/* Go up button */}
      {canGoUp && (
        <button
          onClick={() => onNavigate(parentPath)}
          className={cn(
            "w-full flex items-center gap-2 px-2 py-1.5 rounded text-left",
            "text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
          )}
        >
          <ChevronUp className="h-4 w-4" />
          <span className="text-sm">..</span>
        </button>
      )}

      {/* File entries */}
      {sortedEntries.map((entry) => (
        <FileEntryRow
          key={entry.name}
          entry={entry}
          currentPath={currentPath}
          onNavigate={onNavigate}
          onSelectFile={onSelectFile}
          isSelected={selectedFile === getFullPath(currentPath, entry.name)}
        />
      ))}
    </div>
  );
}

interface FileEntryRowProps {
  entry: FileEntry;
  currentPath: string;
  onNavigate: (path: string) => void;
  onSelectFile: (path: string) => void;
  isSelected: boolean;
}

function FileEntryRow({
  entry,
  currentPath,
  onNavigate,
  onSelectFile,
  isSelected,
}: FileEntryRowProps) {
  const fullPath = getFullPath(currentPath, entry.name);
  const Icon = getFileIcon(entry);

  const handleClick = () => {
    if (entry.type === "directory") {
      onNavigate(fullPath);
    } else {
      onSelectFile(fullPath);
    }
  };

  return (
    <button
      onClick={handleClick}
      className={cn(
        "w-full flex items-center gap-2 px-2 py-1.5 rounded text-left group",
        "transition-colors",
        isSelected
          ? "bg-blue-500/20 text-blue-400"
          : "text-slate-300 hover:text-white hover:bg-slate-800"
      )}
    >
      <Icon className={cn(
        "h-4 w-4 flex-shrink-0",
        entry.type === "directory" ? "text-amber-400" : "text-slate-500",
        isSelected && "text-blue-400"
      )} />

      <span className="text-sm truncate flex-1">{entry.name}</span>

      <span className="text-xs text-slate-500 group-hover:text-slate-400 flex-shrink-0">
        {entry.type === "directory" ? "" : formatBytes(entry.size_bytes)}
      </span>
    </button>
  );
}

// Helper functions

function getParentPath(path: string): string {
  const parts = path.split("/").filter(Boolean);
  parts.pop();
  return "/" + parts.join("/");
}

function getFullPath(currentPath: string, name: string): string {
  if (currentPath.endsWith("/")) {
    return currentPath + name;
  }
  return currentPath + "/" + name;
}

function getFileIcon(entry: FileEntry) {
  if (entry.type === "directory") {
    return Folder;
  }
  if (entry.type === "symlink") {
    return Link2;
  }

  const name = entry.name.toLowerCase();

  // JSON files
  if (name.endsWith(".json")) {
    return FileJson;
  }

  // Code files
  if (
    name.endsWith(".ts") ||
    name.endsWith(".tsx") ||
    name.endsWith(".js") ||
    name.endsWith(".jsx") ||
    name.endsWith(".go") ||
    name.endsWith(".py") ||
    name.endsWith(".rs") ||
    name.endsWith(".sh") ||
    name.endsWith(".bash")
  ) {
    return FileCode;
  }

  // Archive files
  if (
    name.endsWith(".tar.gz") ||
    name.endsWith(".tgz") ||
    name.endsWith(".zip") ||
    name.endsWith(".gz")
  ) {
    return FileArchive;
  }

  // Text files
  if (
    name.endsWith(".txt") ||
    name.endsWith(".md") ||
    name.endsWith(".log") ||
    name === "readme" ||
    name === "license"
  ) {
    return FileText;
  }

  return File;
}
