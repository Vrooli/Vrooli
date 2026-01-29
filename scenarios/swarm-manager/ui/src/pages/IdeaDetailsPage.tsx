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

import { useState, useCallback, useMemo } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ChevronRight, Edit, Trash2, Upload, Play, Loader2, Lightbulb, Sparkles } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { FileTree } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { FileUpload } from "../components/ui/file-upload";
import { TagList } from "../components/ui/tag-list";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { IdeaFormDialog } from "../components/ideas/idea-form-dialog";
import { IdeaAgentDialog } from "../components/ideas/idea-agent-dialog";
import { IdeaClarifyPanel } from "../components/ideas/idea-clarify-panel";
import { IdeaSuggestionsPanel } from "../components/ideas/idea-suggestions-panel";
import {
  buildClarifyQuestionsContent,
  buildSuggestionsContent,
  defaultQueryOptions,
  findIdeaFileByPath,
  formatRelativeTime,
  IDEA_AGENT_FILE_PATHS,
  parseClarifyQuestionsFile,
  parseSuggestionsFile,
} from "../lib";
import { ideasService } from "../services";
import { selectors } from "../consts/selectors";
import { IDEA_STATUS_COLORS, formatIdeaStatus } from "../types";
import type {
  IdeaAgentMode,
  IdeaClarificationQuestion,
  IdeaFile,
  IdeaStatus,
  IdeaSuggestion,
} from "../types";
import { useIdeasStore } from "../stores";

export function IdeaDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const upsertIdea = useIdeasStore((state) => state.upsertIdea);
  const removeIdea = useIdeasStore((state) => state.removeIdea);
  const [selectedFile, setSelectedFile] = useState<IdeaFile | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [showAgentDialog, setShowAgentDialog] = useState(false);

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

  const clarifyFile = useMemo(
    () => findIdeaFileByPath(files ?? [], IDEA_AGENT_FILE_PATHS.clarify),
    [files]
  );
  const suggestionsFile = useMemo(
    () => findIdeaFileByPath(files ?? [], IDEA_AGENT_FILE_PATHS.suggest),
    [files]
  );

  const {
    data: clarifyContent,
    error: clarifyContentError,
    refetch: refetchClarifyContent,
  } = useQuery({
    queryKey: ["ideas", name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.getFileContent(name, IDEA_AGENT_FILE_PATHS.clarify);
    },
    enabled: !!name && !!clarifyFile,
    ...defaultQueryOptions,
  });

  const {
    data: suggestionsContent,
    error: suggestionsContentError,
    refetch: refetchSuggestionsContent,
  } = useQuery({
    queryKey: ["ideas", name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.getFileContent(name, IDEA_AGENT_FILE_PATHS.suggest);
    },
    enabled: !!name && !!suggestionsFile,
    ...defaultQueryOptions,
  });

  const clarifyParsed = useMemo(
    () => parseClarifyQuestionsFile(clarifyContent),
    [clarifyContent]
  );
  const suggestionsParsed = useMemo(
    () => parseSuggestionsFile(suggestionsContent),
    [suggestionsContent]
  );
  const clarifyErrorMessage =
    clarifyContentError instanceof Error
      ? clarifyContentError.message
      : clarifyContentError
        ? "Unable to load clarify questions."
        : clarifyParsed.error;
  const suggestionsErrorMessage =
    suggestionsContentError instanceof Error
      ? suggestionsContentError.message
      : suggestionsContentError
        ? "Unable to load suggestions."
        : suggestionsParsed.error;

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
    onSuccess: (result) => {
      // Refresh the idea to get updated status
      queryClient.invalidateQueries({ queryKey: ["ideas", name] });
      if (result?.idea) {
        upsertIdea(result.idea);
      }
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: { title: string; description: string; status: IdeaStatus; priority: number; tags: string[] }) => {
      if (!name) throw new Error("Name is required");
      return ideasService.update(name, {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        tags: values.tags,
      });
    },
    onSuccess: (updatedIdea) => {
      queryClient.invalidateQueries({ queryKey: ["ideas", name] });
      upsertIdea(updatedIdea);
      setShowEdit(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!name) throw new Error("Name is required");
      return ideasService.delete(name);
    },
    onSuccess: () => {
      if (name) {
        removeIdea(name);
      }
      navigate("/ideas");
    },
  });

  const agentMutation = useMutation({
    mutationFn: ({ mode, prompt }: { mode: IdeaAgentMode; prompt: string }) => {
      if (!name) throw new Error("Name is required");
      return ideasService.research(name, { mode, prompt });
    },
    onSuccess: () => {
      setShowAgentDialog(false);
    },
  });

  const buildFollowupPrompt = (mode: IdeaAgentMode) => {
    if (mode === "suggest") {
      return "Use clarify/questions.json (with answers) to generate actionable suggestions for this idea. Append new suggestions without deleting prior ones.";
    }
    return "Use clarify/questions.json answers to refine the idea and produce an enhanced plan. If suggestions exist, apply accepted ones and ignore rejected ones.";
  };

  const clarifyMutation = useMutation({
    mutationFn: async ({ questions, nextMode }: { questions: IdeaClarificationQuestion[]; nextMode: IdeaAgentMode }) => {
      if (!name) throw new Error("Name is required");
      const content = buildClarifyQuestionsContent(clarifyParsed.raw, questions);
      await ideasService.saveFileContent(
        name,
        IDEA_AGENT_FILE_PATHS.clarify,
        content,
        "application/json"
      );
      await ideasService.research(name, {
        mode: nextMode,
        prompt: buildFollowupPrompt(nextMode),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ideas", name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["ideas", name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify],
      });
      void refetchFiles();
      void refetchClarifyContent();
    },
  });

  const suggestionsMutation = useMutation({
    mutationFn: async (updatedSuggestions: IdeaSuggestion[]) => {
      if (!name) throw new Error("Name is required");
      const content = buildSuggestionsContent(suggestionsParsed.raw, updatedSuggestions);
      await ideasService.saveFileContent(
        name,
        IDEA_AGENT_FILE_PATHS.suggest,
        content,
        "application/json"
      );
      await ideasService.research(name, {
        mode: "enhance",
        prompt:
          "Use suggest/suggestions.json decisions to enhance this idea. Apply accepted suggestions, ignore rejected ones, and reference clarify/questions.json answers if available.",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["ideas", name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["ideas", name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest],
      });
      void refetchFiles();
      void refetchSuggestionsContent();
    },
  });

  const updateError = updateMutation.isError ? "Failed to update idea. Please try again." : null;
  const deleteError = deleteMutation.isError ? "Failed to delete idea. Please try again." : null;
  const agentError = agentMutation.isError
    ? "Failed to start the agent. Make sure agent-manager is running."
    : null;
  const clarifyError = clarifyMutation.isError
    ? "Failed to save answers or start the next agent."
    : null;
  const suggestionsError = suggestionsMutation.isError
    ? "Failed to save suggestions or start the Enhance agent."
    : null;
  const queueError = queueMutation.isError ? "Failed to queue idea. Please try again." : null;

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
                  data-testid={selectors.ideaDetails.editButton}
                  onClick={() => setShowEdit(true)}
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  data-testid={selectors.ideaDetails.deleteButton}
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowAgentDialog(true)}
                  data-testid={selectors.ideaDetails.agentButton}
                >
                  <Sparkles className="mr-2 h-4 w-4" />
                  Idea Agent
                </Button>
              </div>
            </div>

            {(queueError || deleteError) && (
              <div className="mt-4 space-y-2">
                {queueError && (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                    {queueError}
                  </div>
                )}
                {deleteError && (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                    {deleteError}
                  </div>
                )}
              </div>
            )}

            {/* Timestamps - relative format with tooltip for exact date */}
            <div className="mt-4 flex gap-6 border-t border-white/10 pt-4 text-sm text-slate-500">
              <span title={new Date(idea.created).toLocaleString()}>Created {formatRelativeTime(idea.created)}</span>
              <span title={new Date(idea.updated).toLocaleString()}>Updated {formatRelativeTime(idea.updated)}</span>
            </div>
          </Card>

          {(clarifyFile || suggestionsFile) && (
            <div className="space-y-4">
              {clarifyFile && (
                <IdeaClarifyPanel
                  questions={clarifyParsed.questions}
                  filePath={IDEA_AGENT_FILE_PATHS.clarify}
                  parseError={clarifyErrorMessage}
                  isSubmitting={clarifyMutation.isPending}
                  submitError={clarifyError}
                  onSubmit={({ questions, nextMode }) =>
                    clarifyMutation.mutate({ questions, nextMode })
                  }
                />
              )}
              {suggestionsFile && (
                <IdeaSuggestionsPanel
                  suggestions={suggestionsParsed.suggestions}
                  filePath={IDEA_AGENT_FILE_PATHS.suggest}
                  parseError={suggestionsErrorMessage}
                  isSubmitting={suggestionsMutation.isPending}
                  submitError={suggestionsError}
                  onSubmit={(updatedSuggestions) => suggestionsMutation.mutate(updatedSuggestions)}
                />
              )}
            </div>
          )}

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

      {idea && (
        <IdeaFormDialog
          isOpen={showEdit}
          mode="edit"
          initialValues={{
            name: idea.name,
            title: idea.title,
            description: idea.description,
            status: idea.status,
            priority: idea.priority,
            tags: idea.tags,
          }}
          isSubmitting={updateMutation.isPending}
          submitError={updateError}
          onClose={() => {
            setShowEdit(false);
            updateMutation.reset();
          }}
          onSubmit={(values) => updateMutation.mutate(values)}
        />
      )}

      <ConfirmDialog
        isOpen={showDelete}
        onClose={() => {
          setShowDelete(false);
          deleteMutation.reset();
        }}
        onConfirm={() => deleteMutation.mutate()}
        title="Delete Idea"
        description={`Are you sure you want to delete "${idea?.title || name}"? This will remove the idea folder permanently.`}
        confirmationText={idea?.name}
        confirmLabel="Delete Idea"
        isLoading={deleteMutation.isPending}
        testIds={{
          dialog: selectors.ideaDetails.deleteDialog,
          confirmButton: selectors.ideaDetails.deleteConfirmButton,
          cancelButton: selectors.ideaDetails.deleteCancelButton,
        }}
      />

      <IdeaAgentDialog
        isOpen={showAgentDialog}
        isSubmitting={agentMutation.isPending}
        ideaTitle={idea?.title ?? name ?? ""}
        errorMessage={agentError}
        onClose={() => {
          setShowAgentDialog(false);
          agentMutation.reset();
        }}
        onSubmit={(payload) => agentMutation.mutate(payload)}
      />
    </div>
  );
}
