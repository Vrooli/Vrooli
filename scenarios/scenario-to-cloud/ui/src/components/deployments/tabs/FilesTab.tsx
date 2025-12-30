import { useState } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  FolderTree,
  Package,
  FileText,
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

export function FilesTab({ deploymentId }: FilesTabProps) {
  const [currentPath, setCurrentPath] = useState<string | undefined>(undefined);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [activeSection, setActiveSection] = useState<Section>("explorer");

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

          {/* File Explorer and Viewer */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* File Tree */}
            <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4 max-h-[600px] overflow-auto">
              <h3 className="text-sm font-medium text-slate-400 mb-3">Directory Tree</h3>
              <FileTree
                entries={filesData?.entries || []}
                currentPath={filesData?.path || "/"}
                onNavigate={handleNavigate}
                onSelectFile={handleSelectFile}
                selectedFile={selectedFile}
              />
            </div>

            {/* File Viewer */}
            <div className="border border-white/10 rounded-lg bg-slate-900/50 p-4 max-h-[600px] overflow-auto">
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
      ) : (
        <BundleInventory deploymentId={deploymentId} />
      )}
    </div>
  );
}
