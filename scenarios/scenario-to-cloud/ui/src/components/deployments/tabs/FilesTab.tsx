import { useState, useRef, useEffect, useCallback } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  FolderTree,
  Package,
  FileText,
  GripVertical,
} from "lucide-react";
import { useFiles, useFileContent } from "../../../hooks/useLiveState";
import { FileTree } from "./FileTree";
import { FileViewer } from "./FileViewer";
import { QuickAccess } from "./QuickAccess";
import { BundleInventory } from "./BundleInventory";
import { cn } from "../../../lib/utils";

interface FilesTabProps {
  deploymentId: string;
}

type Section = "explorer" | "bundles";

const STORAGE_KEY = "stc.files.treeWidth";
const MIN_TREE_WIDTH = 200;
const MAX_TREE_WIDTH = 600;
const DEFAULT_TREE_WIDTH = 280;

export function FilesTab({ deploymentId }: FilesTabProps) {
  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<Section>("explorer");

  // Resizable panel state
  const [treeWidth, setTreeWidth] = useState(() => {
    if (typeof window === "undefined") return DEFAULT_TREE_WIDTH;
    const stored = Number(localStorage.getItem(STORAGE_KEY));
    return Number.isFinite(stored) && stored >= MIN_TREE_WIDTH && stored <= MAX_TREE_WIDTH
      ? stored
      : DEFAULT_TREE_WIDTH;
  });
  const [isResizing, setIsResizing] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const resizeRef = useRef<{ startX: number; startWidth: number } | null>(null);

  const {
    data: filesData,
    isLoading,
    error,
    refetch,
    isFetching
  } = useFiles(deploymentId, currentPath);

  const {
    data: fileContent,
    isLoading: isLoadingContent,
  } = useFileContent(deploymentId, selectedFile);

  const handleNavigate = (path: string) => {
    setCurrentPath(path);
    setSelectedFile(null);
  };

  const handleSelectFile = (path: string) => {
    setSelectedFile(path);
  };

  const handleQuickAccessSelect = (path: string) => {
    setSelectedFile(path);
  };

  const handleRefresh = () => {
    refetch();
  };

  // Persist tree width to localStorage
  useEffect(() => {
    if (typeof window === "undefined") return;
    localStorage.setItem(STORAGE_KEY, String(treeWidth));
  }, [treeWidth]);

  // Handle resize start
  const handleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    resizeRef.current = { startX: e.clientX, startWidth: treeWidth };
    setIsResizing(true);
  }, [treeWidth]);

  // Handle resize move and end
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!resizeRef.current) return;
      const delta = e.clientX - resizeRef.current.startX;
      const newWidth = Math.max(
        MIN_TREE_WIDTH,
        Math.min(MAX_TREE_WIDTH, resizeRef.current.startWidth + delta)
      );
      setTreeWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      resizeRef.current = null;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);

    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
  }, [isResizing]);

  const sections = [
    { id: "explorer" as const, label: "File Explorer", icon: FolderTree },
    { id: "bundles" as const, label: "Bundles", icon: Package },
  ];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load files: {error.message}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header with refresh and path info */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <FileText className="h-4 w-4" />
          <span>Path: {filesData?.path || "/"}</span>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isFetching && "opacity-50 cursor-not-allowed"
          )}
        >
          {isFetching ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </button>
      </div>

      {/* Section tabs */}
      <div className="flex gap-1 border-b border-white/10 pb-px">
        {sections.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveSection(id)}
            className={cn(
              "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg",
              activeSection === id
                ? "bg-slate-800 text-white border-b-2 border-blue-500"
                : "text-slate-400 hover:text-white hover:bg-slate-800/50"
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Section content */}
      {activeSection === "explorer" ? (
        <div className="space-y-4">
          {/* Quick Access */}
          <QuickAccess
            deploymentId={deploymentId}
            onSelect={handleQuickAccessSelect}
            selectedPath={selectedFile}
          />

          {/* File Explorer and Viewer - Resizable */}
          <div
            ref={containerRef}
            className="flex border border-white/10 rounded-lg bg-slate-900/50 overflow-hidden"
            style={{ minHeight: "600px" }}
          >
            {/* File Tree Panel */}
            <div
              className="flex-shrink-0 overflow-auto border-r border-white/10"
              style={{ width: treeWidth }}
            >
              <div className="p-4">
                <h3 className="text-sm font-medium text-slate-400 mb-3">Directory Tree</h3>
                <FileTree
                  entries={filesData?.entries || []}
                  currentPath={filesData?.path || "/"}
                  onNavigate={handleNavigate}
                  onSelectFile={handleSelectFile}
                  selectedFile={selectedFile}
                />
              </div>
            </div>

            {/* Resize Handle */}
            <div
              className={cn(
                "flex-shrink-0 w-2 cursor-col-resize flex items-center justify-center",
                "bg-slate-800/50 hover:bg-slate-700/50 transition-colors",
                isResizing && "bg-blue-600/30"
              )}
              onMouseDown={handleResizeStart}
              role="separator"
              aria-orientation="vertical"
              aria-label="Resize panels"
            >
              <GripVertical className="h-4 w-4 text-slate-600" />
            </div>

            {/* File Viewer Panel */}
            <div className="flex-1 min-w-0 overflow-auto">
              <div className="p-4">
                <h3 className="text-sm font-medium text-slate-400 mb-3">File Content</h3>
                <FileViewer
                  path={selectedFile}
                  content={fileContent?.content}
                  isLoading={isLoadingContent}
                  truncated={fileContent?.truncated}
                  sizeBytes={fileContent?.sizeBytes}
                />
              </div>
            </div>
          </div>
        </div>
      ) : (
        <BundleInventory deploymentId={deploymentId} />
      )}
    </div>
  );
}
