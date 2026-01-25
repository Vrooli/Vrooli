import { useState, useMemo, useCallback, memo, useEffect } from "react";
import {
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  File,
  Loader2,
} from "lucide-react";
import { useFileSearch } from "../lib/hooks";
import type { FileInfo } from "../lib/api";
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

interface TreeNode {
  name: string;
  path: string;
  isFolder: boolean;
  children: TreeNode[];
  language?: string;
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

function buildProjectTree(files: FileInfo[]): TreeNode[] {
  const root: Map<string, TreeNode> = new Map();

  // Sort files to ensure consistent ordering
  const sortedFiles = [...files].sort((a, b) => a.path.localeCompare(b.path));

  for (const file of sortedFiles) {
    const parts = file.path.split("/");
    let currentMap = root;
    let currentPath = "";

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      currentPath = currentPath ? `${currentPath}/${part}` : part;
      const isLast = i === parts.length - 1;

      if (!currentMap.has(part)) {
        const node: TreeNode = {
          name: part,
          path: currentPath,
          isFolder: !isLast,
          children: [],
          language: isLast ? file.language : undefined,
        };
        currentMap.set(part, node);
      }

      const node = currentMap.get(part)!;
      if (!isLast) {
        // Ensure folder has a children map
        if (!node.children) {
          node.children = [];
        }
        // Convert children array to map for next iteration
        const childMap = new Map<string, TreeNode>();
        for (const child of node.children) {
          childMap.set(child.name, child);
        }
        currentMap = childMap;
        // Store children back as array
        node.children = Array.from(childMap.values());
      }
    }
  }

  // Convert root map to sorted array (folders first, then alphabetically)
  function mapToSortedArray(map: Map<string, TreeNode>): TreeNode[] {
    const nodes = Array.from(map.values());
    // Recursively sort children
    for (const node of nodes) {
      if (node.isFolder && node.children.length > 0) {
        const childMap = new Map<string, TreeNode>();
        for (const child of node.children) {
          childMap.set(child.name, child);
        }
        node.children = mapToSortedArray(childMap);
      }
    }
    // Sort: folders first, then alphabetically
    return nodes.sort((a, b) => {
      if (a.isFolder && !b.isFolder) return -1;
      if (!a.isFolder && b.isFolder) return 1;
      return a.name.localeCompare(b.name);
    });
  }

  return mapToSortedArray(root);
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
  node: TreeNode;
  depth: number;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  onSelect: (path: string) => void;
  selectedPath?: string;
  gitStatuses?: Record<string, string>;
}

const TreeNodeComponent = memo(function TreeNodeComponent({
  node,
  depth,
  expanded,
  onToggle,
  onSelect,
  selectedPath,
  gitStatuses,
}: TreeNodeComponentProps) {
  const isExpanded = expanded.has(node.path);
  const isSelected = selectedPath === node.path;
  const badge = node.isFolder ? null : getStatusBadge(gitStatuses?.[node.path]);
  const fileTypeInfo = node.isFolder ? null : getFileTypeInfo(node.path);

  const handleClick = useCallback(() => {
    if (node.isFolder) {
      onToggle(node.path);
    } else {
      onSelect(node.path);
    }
  }, [node.isFolder, node.path, onToggle, onSelect]);

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
        {node.isFolder ? (
          <>
            {isExpanded ? (
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
                fileTypeInfo ? categoryColorMap[fileTypeInfo.category] : "text-slate-500"
              }`}
            />
          </>
        )}

        <span className="font-mono text-xs truncate flex-1" title={node.path}>
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

      {node.isFolder && isExpanded && (
        <div>
          {node.children.map((child) => (
            <TreeNodeComponent
              key={child.path}
              node={child}
              depth={depth + 1}
              expanded={expanded}
              onToggle={onToggle}
              onSelect={onSelect}
              selectedPath={selectedPath}
              gitStatuses={gitStatuses}
            />
          ))}
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
  const filesQuery = useFileSearch(undefined, true, true);
  const [expanded, setExpanded] = useState<Set<string>>(() =>
    loadExpandedFolders()
  );

  // Save expanded state when it changes
  useEffect(() => {
    saveExpandedFolders(expanded);
  }, [expanded]);

  const tree = useMemo(() => {
    if (!filesQuery.data?.files) return [];
    return buildProjectTree(filesQuery.data.files);
  }, [filesQuery.data?.files]);

  const handleToggle = useCallback((path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) {
        next.delete(path);
      } else {
        next.add(path);
      }
      return next;
    });
  }, []);

  const handleSelect = useCallback(
    (path: string) => {
      onSelectFile(path);
    },
    [onSelectFile]
  );

  if (filesQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
        <span className="ml-2 text-sm text-slate-500">Loading files...</span>
      </div>
    );
  }

  if (filesQuery.error) {
    return (
      <div className="flex items-center justify-center py-8 text-red-400 text-sm">
        Failed to load file tree
      </div>
    );
  }

  if (tree.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-8 text-center">
        <Folder className="h-8 w-8 text-slate-700 mb-3" />
        <p className="text-sm text-slate-500">No files found</p>
      </div>
    );
  }

  return (
    <div className="py-2" data-testid="project-tree-view">
      {tree.map((node) => (
        <TreeNodeComponent
          key={node.path}
          node={node}
          depth={0}
          expanded={expanded}
          onToggle={handleToggle}
          onSelect={handleSelect}
          selectedPath={selectedFile}
          gitStatuses={gitStatuses}
        />
      ))}
      {filesQuery.data?.truncated && (
        <div className="px-2 py-2 text-xs text-amber-400/80 text-center">
          File list truncated. Some files may not be shown.
        </div>
      )}
    </div>
  );
});
