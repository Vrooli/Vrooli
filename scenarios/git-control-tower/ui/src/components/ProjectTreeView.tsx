import { useState, useCallback, memo, useEffect, useMemo } from "react";
import {
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  File,
  Loader2,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { useDirectoryContents, queryKeys } from "../lib/hooks";
import { fetchDirectoryContents, type DirEntry, type DirListResponse } from "../lib/api";
import type { FileCategory } from "../lib/fileTypes";
import { getFileTypeInfo } from "../lib/fileTypes";

// Color mapping for file categories
const categoryColorMap: Record<FileCategory, string> = {
  code: "text-blue-400",
  markdown: "text-purple-400",
  image: "text-pink-400",
  pdf: "text-red-400",
  binary: "text-gray-400",
  text: "text-slate-400",
};

interface ProjectTreeViewProps {
  onSelectFile: (path: string) => void;
  selectedFile?: string;
  gitStatuses?: Record<string, string>;
}

interface LazyTreeNode {
  name: string;
  path: string;
  isDir: boolean;
  language?: string;
  children?: LazyTreeNode[]; // undefined = not fetched, [] = empty folder
  isLoading?: boolean;
}

const statusStyleMap: Record<string, string> = {
  D: "text-red-400 border-red-500/40 bg-red-500/10",
  M: "text-amber-300 border-amber-500/40 bg-amber-500/10",
  A: "text-emerald-300 border-emerald-500/40 bg-emerald-500/10",
  R: "text-cyan-300 border-cyan-500/40 bg-cyan-500/10",
  U: "text-red-300 border-red-500/40 bg-red-500/10",
  "?": "text-slate-300 border-slate-500/40 bg-slate-500/10",
};

function getStatusBadge(code: string | undefined): {
  label: string;
  style: string;
} | null {
  if (!code) return null;
  const normalized = code.toUpperCase();
  if (normalized.includes("D"))
    return { label: "D", style: statusStyleMap.D };
  if (normalized.includes("M"))
    return { label: "M", style: statusStyleMap.M };
  if (normalized.includes("A"))
    return { label: "A", style: statusStyleMap.A };
  if (normalized.includes("R"))
    return { label: "R", style: statusStyleMap.R };
  if (normalized.includes("U"))
    return { label: "U", style: statusStyleMap.U };
  if (normalized.includes("?"))
    return { label: "?", style: statusStyleMap["?"] };
  return null;
}

// Convert API DirEntry to LazyTreeNode
function entryToNode(entry: DirEntry): LazyTreeNode {
  return {
    name: entry.name,
    path: entry.path,
    isDir: entry.is_dir,
    language: entry.language,
    children: entry.is_dir ? undefined : undefined, // folders start with undefined children (not fetched)
  };
}

// Storage key for expanded folders
const EXPANDED_FOLDERS_KEY = "gct.projectTree.expandedFolders";

function loadExpandedFolders(): Set<string> {
  try {
    const stored = localStorage.getItem(EXPANDED_FOLDERS_KEY);
    if (stored) {
      return new Set(JSON.parse(stored) as string[]);
    }
  } catch {
    // Ignore parse errors
  }
  return new Set();
}

function saveExpandedFolders(expanded: Set<string>) {
  try {
    localStorage.setItem(EXPANDED_FOLDERS_KEY, JSON.stringify([...expanded]));
  } catch {
    // Ignore storage errors
  }
}

interface TreeNodeComponentProps {
  node: LazyTreeNode;
  depth: number;
  expanded: Set<string>;
  loadingPaths: Set<string>;
  onToggle: (path: string) => void;
  onSelect: (path: string) => void;
  selectedPath?: string;
  gitStatuses?: Record<string, string>;
  fetchedDirs: Map<string, LazyTreeNode[]>;
}

const TreeNodeComponent = memo(function TreeNodeComponent({
  node,
  depth,
  expanded,
  loadingPaths,
  onToggle,
  onSelect,
  selectedPath,
  gitStatuses,
  fetchedDirs,
}: TreeNodeComponentProps) {
  const isExpanded = expanded.has(node.path);
  const isLoading = loadingPaths.has(node.path);
  const isSelected = selectedPath === node.path;
  const gitStatus = gitStatuses?.[node.path];
  const badge = node.isDir ? null : getStatusBadge(gitStatus);
  const fileTypeInfo = node.isDir ? null : getFileTypeInfo(node.path);

  // Check if file is untracked (status contains "?")
  const isUntracked = gitStatus?.toUpperCase().includes("?") ?? false;

  // Get children from fetchedDirs if this folder is expanded and has been fetched
  const children = node.isDir && isExpanded ? fetchedDirs.get(node.path) : undefined;

  const handleClick = useCallback(() => {
    if (node.isDir) {
      onToggle(node.path);
    } else {
      onSelect(node.path);
    }
  }, [node.isDir, node.path, onToggle, onSelect]);

  const paddingLeft = depth * 16 + 8;

  return (
    <>
      <div
        className={`flex items-center gap-1.5 py-1 px-2 cursor-pointer transition-colors rounded select-none ${
          isSelected
            ? "bg-slate-700/50 text-slate-100"
            : "hover:bg-slate-800/50 text-slate-300"
        }`}
        style={{ paddingLeft }}
        onClick={handleClick}
        data-testid={`tree-node-${node.path}`}
      >
        {node.isDir ? (
          <>
            {isLoading ? (
              <Loader2 className="h-3 w-3 text-slate-500 flex-shrink-0 animate-spin" />
            ) : isExpanded ? (
              <ChevronDown className="h-3 w-3 text-slate-500 flex-shrink-0" />
            ) : (
              <ChevronRight className="h-3 w-3 text-slate-500 flex-shrink-0" />
            )}
            {isExpanded ? (
              <FolderOpen className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />
            ) : (
              <Folder className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />
            )}
          </>
        ) : (
          <>
            <span className="w-3" /> {/* Spacer for alignment */}
            <File
              className={`h-3.5 w-3.5 flex-shrink-0 ${
                isUntracked
                  ? "text-slate-600"
                  : fileTypeInfo
                    ? categoryColorMap[fileTypeInfo.category]
                    : "text-slate-500"
              }`}
            />
          </>
        )}

        <span
          className={`font-mono text-xs truncate flex-1 ${
            isUntracked ? "text-slate-500 italic" : ""
          }`}
          title={node.path}
        >
          {node.name}
        </span>

        {badge && (
          <span
            className={`flex items-center justify-center rounded border font-bold h-4 w-4 text-[9px] flex-shrink-0 ${badge.style}`}
            title={`Status: ${badge.label}`}
          >
            {badge.label}
          </span>
        )}
      </div>

      {node.isDir && isExpanded && children && (
        <div>
          {children.map((child) => (
            <TreeNodeComponent
              key={child.path}
              node={child}
              depth={depth + 1}
              expanded={expanded}
              loadingPaths={loadingPaths}
              onToggle={onToggle}
              onSelect={onSelect}
              selectedPath={selectedPath}
              gitStatuses={gitStatuses}
              fetchedDirs={fetchedDirs}
            />
          ))}
          {children.length === 0 && (
            <div
              className="text-xs text-slate-500 italic py-1"
              style={{ paddingLeft: paddingLeft + 16 }}
            >
              Empty folder
            </div>
          )}
        </div>
      )}
    </>
  );
});

export const ProjectTreeView = memo(function ProjectTreeView({
  onSelectFile,
  selectedFile,
  gitStatuses,
}: ProjectTreeViewProps) {
  const queryClient = useQueryClient();

  // Fetch root directory
  const rootQuery = useDirectoryContents("", true);

  // Track expanded folders
  const [expanded, setExpanded] = useState<Set<string>>(() =>
    loadExpandedFolders()
  );

  // Track folders that are currently loading
  const [loadingPaths, setLoadingPaths] = useState<Set<string>>(new Set());

  // Store fetched directory contents - key is path, value is children nodes
  const [fetchedDirs, setFetchedDirs] = useState<Map<string, LazyTreeNode[]>>(
    new Map()
  );

  // Convert root entries to tree nodes
  const rootNodes = useMemo(() => {
    if (!rootQuery.data?.entries) return [];
    return rootQuery.data.entries.map(entryToNode);
  }, [rootQuery.data?.entries]);

  // Save expanded state when it changes
  useEffect(() => {
    saveExpandedFolders(expanded);
  }, [expanded]);

  // Load previously expanded folders on mount
  useEffect(() => {
    const loadPreviouslyExpandedFolders = async () => {
      const savedExpanded = loadExpandedFolders();
      if (savedExpanded.size === 0) return;

      // Sort paths by depth so we load parent folders first
      const sortedPaths = Array.from(savedExpanded).sort(
        (a, b) => a.split("/").length - b.split("/").length
      );

      // Load each expanded folder's contents
      for (const path of sortedPaths) {
        // Check if parent folder is also expanded (it should be for this to be visible)
        const parentPath = path.includes("/")
          ? path.substring(0, path.lastIndexOf("/"))
          : "";

        // Skip if parent isn't expanded (unless it's root level)
        if (parentPath && !savedExpanded.has(parentPath)) {
          continue;
        }

        try {
          // Check if already in cache
          const cached = queryClient.getQueryData<DirListResponse>(
            queryKeys.directoryContents(path)
          );

          if (cached) {
            setFetchedDirs((prev) => {
              const next = new Map(prev);
              next.set(path, cached.entries.map(entryToNode));
              return next;
            });
          } else {
            // Fetch and cache
            const data = await fetchDirectoryContents(path);
            queryClient.setQueryData(queryKeys.directoryContents(path), data);
            setFetchedDirs((prev) => {
              const next = new Map(prev);
              next.set(path, data.entries.map(entryToNode));
              return next;
            });
          }
        } catch {
          // Remove from expanded if fetch failed
          setExpanded((prev) => {
            const next = new Set(prev);
            next.delete(path);
            return next;
          });
        }
      }
    };

    // Only run once after root is loaded
    if (rootQuery.data) {
      loadPreviouslyExpandedFolders();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rootQuery.data ? 1 : 0]); // Only run when rootQuery.data becomes available

  const handleToggle = useCallback(
    async (path: string) => {
      const isCurrentlyExpanded = expanded.has(path);

      if (isCurrentlyExpanded) {
        // Collapse - just update expanded state
        setExpanded((prev) => {
          const next = new Set(prev);
          next.delete(path);
          return next;
        });
      } else {
        // Expand - check if we need to fetch
        const hasFetched = fetchedDirs.has(path);

        if (!hasFetched) {
          // Fetch directory contents
          setLoadingPaths((prev) => new Set(prev).add(path));

          try {
            // Check React Query cache first
            let data = queryClient.getQueryData<DirListResponse>(
              queryKeys.directoryContents(path)
            );

            if (!data) {
              // Fetch from API
              data = await fetchDirectoryContents(path);
              // Cache the result
              queryClient.setQueryData(queryKeys.directoryContents(path), data);
            }

            // Store the children
            setFetchedDirs((prev) => {
              const next = new Map(prev);
              next.set(path, data.entries.map(entryToNode));
              return next;
            });
          } catch (err) {
            console.error("Failed to fetch directory contents:", err);
            // Don't expand if fetch failed
            setLoadingPaths((prev) => {
              const next = new Set(prev);
              next.delete(path);
              return next;
            });
            return;
          }

          setLoadingPaths((prev) => {
            const next = new Set(prev);
            next.delete(path);
            return next;
          });
        }

        // Mark as expanded
        setExpanded((prev) => {
          const next = new Set(prev);
          next.add(path);
          return next;
        });
      }
    },
    [expanded, fetchedDirs, queryClient]
  );

  const handleSelect = useCallback(
    (path: string) => {
      onSelectFile(path);
    },
    [onSelectFile]
  );

  if (rootQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
        <span className="ml-2 text-sm text-slate-500">Loading files...</span>
      </div>
    );
  }

  if (rootQuery.error) {
    return (
      <div className="flex items-center justify-center py-8 text-red-400 text-sm">
        Failed to load file tree
      </div>
    );
  }

  if (rootNodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Folder className="h-8 w-8 text-slate-700 mb-3" />
        <p className="text-sm text-slate-500">No files found</p>
      </div>
    );
  }

  return (
    <div className="py-2" data-testid="project-tree-view">
      {rootNodes.map((node) => (
        <TreeNodeComponent
          key={node.path}
          node={node}
          depth={0}
          expanded={expanded}
          loadingPaths={loadingPaths}
          onToggle={handleToggle}
          onSelect={handleSelect}
          selectedPath={selectedFile}
          gitStatuses={gitStatuses}
          fetchedDirs={fetchedDirs}
        />
      ))}
    </div>
  );
});
