/**
 * Idea Details Page
 *
 * Displays detailed information about a single idea including:
 * - Idea metadata (title, description, status, priority, tags)
 * - File tree view showing all files in the idea folder
 * - File preview when a file is selected
 * - Drag-and-drop file upload
 * - Navigation back to the ideas list
 *
 * Experience Architecture (Phase 29):
 * - Enhanced breadcrumb navigation shows current location context
 * - Reduces cognitive load for returning users navigating hierarchies
 *
 * [REQ:REQ-P0-004] Idea Details UI Page with file tree view, preview, and upload
 */

import { useState, useCallback } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { useParams, Link } from "react-router-dom";
import { ChevronRight, Edit, Trash2, Upload, Play, Loader2, Lightbulb } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { FileTree } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { FileUpload } from "../components/ui/file-upload";
import { TagList } from "../components/ui/tag-list";
import { defaultQueryOptions, formatRelativeTime } from "../lib";
import { ideasService } from "../services";
import { selectors } from "../consts/selectors";
import { IDEA_STATUS_COLORS, formatIdeaStatus } from "../types";
import type { IdeaFile } from "../types";

export function IdeaDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const queryClient = useQueryClient();
  const [selectedFile, setSelectedFile] = useState<IdeaFile | null>(null);
  const [showUpload, setShowUpload] = useState(false);

  // Fetch idea details
  // Note: queryFn is only called when enabled is true (i.e., name exists)
  const {
    data: idea,
    isLoading: isLoadingIdea,
    error: ideaError,
    refetch: refetchIdea,
  } = useQuery({
    queryKey: ["ideas", name],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.get(name);
    },
    enabled: !!name,
    ...defaultQueryOptions,
  });

  // Fetch idea files
  // Note: queryFn is only called when enabled is true (i.e., name exists)
  const {
    data: files,
    isLoading: isLoadingFiles,
    error: filesError,
    refetch: refetchFiles,
  } = useQuery({
    queryKey: ["ideas", name, "files"],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.getFiles(name);
    },
    enabled: !!name,
    ...defaultQueryOptions,
  });

  const isLoading = isLoadingIdea || isLoadingFiles;
  const error = ideaError ?? filesError;

  // Handle file selection
  const handleFileSelect = useCallback((file: IdeaFile) => {
    // Only select files, not directories
    if (file.type === "file") {
      setSelectedFile(file);
    }
  }, []);

  // Handle upload completion
  const handleUploadComplete = useCallback(() => {
    // Refresh the file list
    queryClient.invalidateQueries({ queryKey: ["ideas", name, "files"] });
  }, [queryClient, name]);

  // Queue mutation for processing ideas
  // [REQ:REQ-P0-005] Queue idea for processing via ecosystem-manager
  const queueMutation = useMutation({
    mutationFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.queue(name);
    },
    onSuccess: () => {
      // Refresh the idea to get updated status
      queryClient.invalidateQueries({ queryKey: ["ideas", name] });
      queryClient.invalidateQueries({ queryKey: ["ideas"] });
    },
  });

  // Check if idea can be queued (only from backlog, researching, or ready)
  const canQueue = idea && ["backlog", "researching", "ready"].includes(idea.status);

  if (!name) {
    return (
      <div className="space-y-6" data-testid={selectors.ideaDetails.page}>
        <ErrorState
          error={new Error("Idea name is required")}
          title="Invalid URL"
        />
      </div>
    );
  }

  return (
    <div className="space-y-6" data-testid={selectors.ideaDetails.page}>
      {/* Breadcrumb navigation - shows context and allows quick navigation (Phase 29) */}
      <nav className="flex items-center gap-2 text-sm" data-testid={selectors.ideaDetails.breadcrumb}>
        <Link
          to="/ideas"
          className="flex items-center gap-1.5 text-slate-400 hover:text-slate-200 transition-colors"
          data-testid={selectors.ideaDetails.backButton}
        >
          <Lightbulb className="h-4 w-4" />
          <span>Ideas</span>
        </Link>
        <ChevronRight className="h-4 w-4 text-slate-600" />
        <span className="text-slate-200 truncate max-w-[200px]" title={idea?.title || name}>
          {idea?.title || name}
        </span>
      </nav>

      {/* Loading state */}
      {isLoading && (
        <Card padding="lg" centered>
          <p className="text-slate-400">Loading idea details...</p>
        </Card>
      )}

      {/* Error state */}
      {error && (
        <ErrorState
          error={error}
          title="Unable to load idea"
          onRetry={() => {
            refetchIdea();
            refetchFiles();
          }}
        />
      )}

      {/* Idea details */}
      {idea && !error && (
        <>
          {/* Idea header */}
          <Card data-testid={selectors.ideaDetails.header}>
            <div className="flex items-start justify-between">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <span
                    className={`inline-block h-3 w-3 rounded-full ${IDEA_STATUS_COLORS[idea.status] ?? "bg-slate-500"}`}
                  />
                  <span className="text-sm uppercase tracking-wider text-slate-500">
                    {formatIdeaStatus(idea.status)}
                  </span>
                  <span className="rounded-full bg-slate-700 px-3 py-1 text-sm text-slate-300">
                    Priority {idea.priority}
                  </span>
                </div>
                <h1
                  className="text-2xl font-bold text-slate-100"
                  data-testid={selectors.ideaDetails.title}
                >
                  {idea.title}
                </h1>
                <p
                  className="text-slate-400"
                  data-testid={selectors.ideaDetails.description}
                >
                  {idea.description || "No description provided"}
                </p>
                <TagList tags={idea.tags} maxTags={10} className="mt-4" />
              </div>

              {/* Action buttons */}
              <div className="flex gap-2">
                {canQueue && (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => queueMutation.mutate()}
                    disabled={queueMutation.isPending}
                    data-testid={selectors.ideaDetails.queueButton}
                  >
                    {queueMutation.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="mr-2 h-4 w-4" />
                    )}
                    {queueMutation.isPending ? "Queueing..." : "Queue for Processing"}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  disabled
                  title="Edit functionality coming soon"
                  data-testid={selectors.ideaDetails.editButton}
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled
                  title="Delete functionality coming soon"
                  data-testid={selectors.ideaDetails.deleteButton}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </Button>
              </div>
            </div>

            {/* Timestamps - relative format with tooltip for exact date */}
            <div className="mt-4 flex gap-6 border-t border-white/10 pt-4 text-sm text-slate-500">
              <span title={new Date(idea.created).toLocaleString()}>Created {formatRelativeTime(idea.created)}</span>
              <span title={new Date(idea.updated).toLocaleString()}>Updated {formatRelativeTime(idea.updated)}</span>
            </div>
          </Card>

          {/* Files section with two-column layout */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Left column: File tree and upload */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium text-slate-200">Files</h2>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowUpload(!showUpload)}
                  data-testid="toggle-upload"
                >
                  <Upload className="mr-2 h-4 w-4" />
                  {showUpload ? "Hide Upload" : "Upload Files"}
                </Button>
              </div>

              {/* Upload area (collapsible) */}
              {showUpload && (
                <FileUpload
                  ideaName={name}
                  onUploadComplete={handleUploadComplete}
                  data-testid={selectors.ideaDetails.fileUpload}
                />
              )}

              {/* File tree */}
              {isLoadingFiles ? (
                <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center">
                  <p className="text-slate-400">Loading files...</p>
                </div>
              ) : (
                <FileTree
                  files={files ?? []}
                  onFileSelect={handleFileSelect}
                  selectedPath={selectedFile?.path}
                  data-testid={selectors.ideaDetails.fileTree}
                />
              )}
            </div>

            {/* Right column: File preview */}
            <div className="space-y-4">
              <h2 className="text-lg font-medium text-slate-200">
                {selectedFile ? "Preview" : "Select a file to preview"}
              </h2>

              {selectedFile ? (
                <FilePreview
                  ideaName={name}
                  filePath={selectedFile.path}
                  fileName={selectedFile.name}
                  data-testid={selectors.ideaDetails.filePreview}
                />
              ) : (
                <div className="rounded-lg border border-white/10 bg-slate-800/30 p-8 text-center min-h-[200px] flex items-center justify-center">
                  <p className="text-slate-500">
                    Click on a file in the tree to preview its contents
                  </p>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
