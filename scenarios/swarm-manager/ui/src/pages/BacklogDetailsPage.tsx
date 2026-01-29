/**
 * Backlog Details Page
 *
 * Displays detailed information about a single backlog item including:
 * - Metadata (title, description, status, priority, tags)
 * - File tree view showing all files in the backlog folder
 * - File preview when a file is selected
 * - Drag-and-drop file upload
 * - Navigation back to the backlog list
 * - Contextual actions for each backlog kind
 *
 * Experience Architecture (Phase 29):
 * - Enhanced breadcrumb navigation shows current location context
 * - Reduces cognitive load for returning users navigating hierarchies
 *
 * [REQ:REQ-P0-004] Backlog Details UI Page with file tree view, preview, and upload
 */

import { useState, useCallback, useMemo } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ChevronRight, Edit, Trash2, Upload, Play, Loader2, Sparkles, Search, Wrench, ArrowRight } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorState } from "../components/ui/error-state";
import { FileTree } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { FileUpload } from "../components/ui/file-upload";
import { TagList } from "../components/ui/tag-list";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { BacklogAgentDialog } from "../components/backlog/backlog-agent-dialog";
import { IdeaClarifyPanel } from "../components/backlog/idea-clarify-panel";
import { IdeaSuggestionsPanel } from "../components/backlog/idea-suggestions-panel";
import {
  buildClarifyQuestionsContent,
  buildSuggestionsContent,
  defaultQueryOptions,
  findBacklogFileByPath,
  formatRelativeTime,
  IDEA_AGENT_FILE_PATHS,
  parseClarifyQuestionsFile,
  parseSuggestionsFile,
} from "../lib";
import { backlogService } from "../services";
import { selectors } from "../consts/selectors";
import { BACKLOG_STATUS_COLORS, formatBacklogStatus } from "../types";
import type {
  BacklogFile,
  BacklogKind,
  BacklogResearchTarget,
  BacklogStatus,
  IdeaAgentMode,
  IdeaClarificationQuestion,
  IdeaSuggestion,
} from "../types";
import { useBacklogStore } from "../stores";

const BACKLOG_KINDS: BacklogKind[] = ["idea", "research", "fix", "execute"];

const KIND_LABELS: Record<BacklogKind, string> = {
  idea: "Idea",
  research: "Research",
  fix: "Fix",
  execute: "Execute",
};

const RESEARCH_TARGET_LABELS: Record<BacklogResearchTarget, string> = {
  idea: "Idea",
  fix: "Fix",
  execute: "Execute",
  unspecified: "Unspecified",
};

const QUEUEABLE_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready"];

const buildFollowupPrompt = (mode: IdeaAgentMode): string => {
  if (mode === "suggest") {
    return "Use clarify/questions.json (with answers) to generate actionable suggestions for this idea. Append new suggestions without deleting prior ones.";
  }
  return "Use clarify/questions.json answers to refine the idea and produce an enhanced plan. If suggestions exist, apply accepted ones and ignore rejected ones.";
};

export function BacklogDetailsPage() {
  const { kind, name } = useParams<{ kind: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const removeItem = useBacklogStore((state) => state.removeItem);
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [showAgentDialog, setShowAgentDialog] = useState(false);

  const backlogKind = BACKLOG_KINDS.includes(kind as BacklogKind) ? (kind as BacklogKind) : null;

  const {
    data: item,
    isLoading: isLoadingItem,
    error: itemError,
    refetch: refetchItem,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.get(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    ...defaultQueryOptions,
  });

  const {
    data: files,
    isLoading: isLoadingFiles,
    error: filesError,
    refetch: refetchFiles,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "files"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getFiles(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
    ...defaultQueryOptions,
  });

  const clarifyFile = useMemo(
    () => (item?.kind === "idea" ? findBacklogFileByPath(files ?? [], IDEA_AGENT_FILE_PATHS.clarify) : null),
    [files, item?.kind]
  );
  const suggestionsFile = useMemo(
    () => (item?.kind === "idea" ? findBacklogFileByPath(files ?? [], IDEA_AGENT_FILE_PATHS.suggest) : null),
    [files, item?.kind]
  );

  const {
    data: clarifyContent,
    error: clarifyContentError,
    refetch: refetchClarifyContent,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getFileContent(backlogKind, name, IDEA_AGENT_FILE_PATHS.clarify);
    },
    enabled: !!backlogKind && !!name && !!clarifyFile,
    ...defaultQueryOptions,
  });

  const {
    data: suggestionsContent,
    error: suggestionsContentError,
    refetch: refetchSuggestionsContent,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getFileContent(backlogKind, name, IDEA_AGENT_FILE_PATHS.suggest);
    },
    enabled: !!backlogKind && !!name && !!suggestionsFile,
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

  const isLoading = isLoadingItem || isLoadingFiles;
  const error = itemError ?? filesError;

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "file") {
      setSelectedFile(file);
    }
  }, []);

  const handleUploadComplete = useCallback(() => {
    if (!backlogKind || !name) return;
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
  }, [queryClient, backlogKind, name]);

  const queueMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.queue(backlogKind, name);
    },
    onSuccess: (result) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      if (result?.item) {
        upsertItem(result.item);
      }
    },
  });

  const updateMutation = useMutation({
    mutationFn: (values: {
      title: string;
      description: string;
      status: BacklogStatus;
      priority: number;
      tags: string[];
      researchTarget?: BacklogResearchTarget;
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        tags: values.tags,
        researchTarget: values.researchTarget,
      });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
      setShowEdit(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.delete(backlogKind, name);
    },
    onSuccess: () => {
      if (backlogKind && name) {
        removeItem(name, backlogKind);
      }
      navigate("/backlog");
    },
  });

  const agentMutation = useMutation({
    mutationFn: ({ mode, prompt, targetKind }: { mode?: IdeaAgentMode; prompt: string; targetKind?: BacklogResearchTarget }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.research(backlogKind, name, {
        mode,
        prompt,
        targetKind,
      });
    },
    onSuccess: () => {
      setShowAgentDialog(false);
    },
  });

  const clarifyMutation = useMutation({
    mutationFn: async ({ questions, nextMode }: { questions: IdeaClarificationQuestion[]; nextMode: IdeaAgentMode }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const content = buildClarifyQuestionsContent(clarifyParsed.raw, questions);
      await backlogService.saveFileContent(
        backlogKind,
        name,
        IDEA_AGENT_FILE_PATHS.clarify,
        content,
        "application/json"
      );
      await backlogService.research(backlogKind, name, {
        mode: nextMode,
        prompt: buildFollowupPrompt(nextMode),
      });
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify],
      });
      void refetchFiles();
      void refetchClarifyContent();
    },
  });

  const suggestionsMutation = useMutation({
    mutationFn: async (updatedSuggestions: IdeaSuggestion[]) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const content = buildSuggestionsContent(suggestionsParsed.raw, updatedSuggestions);
      await backlogService.saveFileContent(
        backlogKind,
        name,
        IDEA_AGENT_FILE_PATHS.suggest,
        content,
        "application/json"
      );
      await backlogService.research(backlogKind, name, {
        mode: "enhance",
        prompt:
          "Use suggest/suggestions.json decisions to enhance this idea. Apply accepted suggestions, ignore rejected ones, and reference clarify/questions.json answers if available.",
      });
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest],
      });
      void refetchFiles();
      void refetchSuggestionsContent();
    },
  });

  const convertMutation = useMutation({
    mutationFn: async (targetKind: BacklogKind) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.convert(backlogKind, name, { targetKind });
    },
    onSuccess: (convertedItem) => {
      if (!backlogKind || !name) return;
      removeItem(name, backlogKind);
      upsertItem(convertedItem);
      navigate(`/backlog/${convertedItem.kind}/${convertedItem.name}`);
    },
  });

  const updateError = updateMutation.isError ? "Failed to update backlog item. Please try again." : null;
  const deleteError = deleteMutation.isError ? "Failed to delete backlog item. Please try again." : null;
  const agentError = agentMutation.isError
    ? "Failed to start the agent. Make sure agent-manager is running."
    : null;
  const clarifyError = clarifyMutation.isError
    ? "Failed to save answers or start the next agent."
    : null;
  const suggestionsError = suggestionsMutation.isError
    ? "Failed to save suggestions or start the Enhance agent."
    : null;
  const queueError = queueMutation.isError ? "Failed to queue backlog item. Please try again." : null;
  const convertError = convertMutation.isError ? "Failed to convert backlog item. Please try again." : null;

  const canQueue = Boolean(item && item.kind !== "research" && QUEUEABLE_STATUSES.includes(item.status));

  const hasResearchOutput = useMemo(() => {
    if (!files || files.length === 0) return false;
    const hasNonSpecFile = (entries: BacklogFile[]): boolean => {
      return entries.some((entry) => {
        if (entry.type === "directory") {
          return entry.children ? hasNonSpecFile(entry.children) : false;
        }
        return entry.path !== "spec.json";
      });
    };
    return hasNonSpecFile(files);
  }, [files]);

  const canConvert =
    item?.kind === "research" &&
    item.researchTarget &&
    item.researchTarget !== "unspecified" &&
    hasResearchOutput;

  const convertTarget = canConvert ? (item.researchTarget as BacklogKind) : null;

  if (!backlogKind || !name) {
    return (
      <div className="space-y-6" data-testid={selectors.backlogDetails.page}>
        <ErrorState
          error={new Error("Backlog kind and name are required")}
          title="Invalid URL"
        />
      </div>
    );
  }

  const HeaderIcon = backlogKind === "research" ? Search : backlogKind === "fix" ? Wrench : Play;
  const agentLabel = item?.kind === "idea" ? "Idea Agent" : item?.kind === "research" ? "Research Agent" : "Research";
  const queueLabel = item?.kind === "fix" ? "Apply Fix" : item?.kind === "execute" ? "Execute" : "Queue for Processing";

  return (
    <div className="space-y-6" data-testid={selectors.backlogDetails.page}>
      <nav className="flex flex-wrap items-center gap-2 text-sm" data-testid={selectors.backlogDetails.breadcrumb}>
        <Link
          to="/backlog"
          className="flex items-center gap-1.5 text-slate-400 hover:text-slate-200 transition-colors"
          data-testid={selectors.backlogDetails.backButton}
        >
          <Sparkles className="h-4 w-4" />
          <span>Backlog</span>
        </Link>
        <ChevronRight className="h-4 w-4 text-slate-600" />
        <span className="text-slate-300">{KIND_LABELS[backlogKind]}</span>
        <ChevronRight className="h-4 w-4 text-slate-600" />
        <span className="text-slate-200 truncate max-w-[220px]" title={item?.title || name}>
          {item?.title || name}
        </span>
      </nav>

      {isLoading && (
        <Card padding="lg" centered>
          <p className="text-slate-400">Loading backlog details...</p>
        </Card>
      )}

      {error && (
        <ErrorState
          error={error}
          title="Unable to load backlog item"
          onRetry={() => {
            refetchItem();
            refetchFiles();
          }}
        />
      )}

      {item && !error && (
        <>
          <Card data-testid={selectors.backlogDetails.header}>
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-3">
                  <span
                    className={`inline-block h-3 w-3 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                  />
                  <span className="text-sm uppercase tracking-wider text-slate-500">
                    {formatBacklogStatus(item.status)}
                  </span>
                  <span className="rounded-full bg-slate-700 px-3 py-1 text-sm text-slate-300">
                    Priority {item.priority}
                  </span>
                  {item.kind === "research" && (
                    <span className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-300">
                      Target: {item.researchTarget ? RESEARCH_TARGET_LABELS[item.researchTarget] : "Unspecified"}
                    </span>
                  )}
                </div>
                <h1
                  className="text-2xl font-bold text-slate-100"
                  data-testid={selectors.backlogDetails.title}
                >
                  {item.title}
                </h1>
                <p
                  className="text-slate-400"
                  data-testid={selectors.backlogDetails.description}
                >
                  {item.description || "No description provided"}
                </p>
                <TagList tags={item.tags} maxTags={10} className="mt-4" />
              </div>

              <div className="flex flex-wrap gap-2">
                {canQueue && (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => queueMutation.mutate()}
                    disabled={queueMutation.isPending}
                    data-testid={selectors.backlogDetails.queueButton}
                  >
                    {queueMutation.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <HeaderIcon className="mr-2 h-4 w-4" />
                    )}
                    {queueMutation.isPending ? "Queueing..." : queueLabel}
                  </Button>
                )}
                {canConvert && convertTarget && (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => convertMutation.mutate(convertTarget)}
                    disabled={convertMutation.isPending}
                    data-testid={selectors.backlogDetails.convertButton}
                  >
                    {convertMutation.isPending ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <ArrowRight className="mr-2 h-4 w-4" />
                    )}
                    {convertMutation.isPending
                      ? "Converting..."
                      : `Convert to ${KIND_LABELS[convertTarget]}`}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  data-testid={selectors.backlogDetails.editButton}
                  onClick={() => setShowEdit(true)}
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  data-testid={selectors.backlogDetails.deleteButton}
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowAgentDialog(true)}
                  data-testid={selectors.backlogDetails.agentButton}
                >
                  <Sparkles className="mr-2 h-4 w-4" />
                  {agentLabel}
                </Button>
              </div>
            </div>

            {(queueError || deleteError || convertError) && (
              <div className="mt-4 space-y-2">
                {queueError && (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                    {queueError}
                  </div>
                )}
                {convertError && (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                    {convertError}
                  </div>
                )}
                {deleteError && (
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                    {deleteError}
                  </div>
                )}
              </div>
            )}

            <div className="mt-4 flex gap-6 border-t border-white/10 pt-4 text-sm text-slate-500">
              <span title={new Date(item.created).toLocaleString()}>Created {formatRelativeTime(item.created)}</span>
              <span title={new Date(item.updated).toLocaleString()}>Updated {formatRelativeTime(item.updated)}</span>
            </div>
          </Card>

          {(clarifyFile || suggestionsFile) && item.kind === "idea" && (
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

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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

              {showUpload && (
                <FileUpload
                  backlogKind={backlogKind}
                  backlogName={name}
                  onUploadComplete={handleUploadComplete}
                  data-testid={selectors.backlogDetails.fileUpload}
                />
              )}

              {isLoadingFiles ? (
                <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center">
                  <p className="text-slate-400">Loading files...</p>
                </div>
              ) : (
                <FileTree
                  files={files ?? []}
                  onFileSelect={handleFileSelect}
                  selectedPath={selectedFile?.path}
                  data-testid={selectors.backlogDetails.fileTree}
                />
              )}
            </div>

            <div className="space-y-4">
              <h2 className="text-lg font-medium text-slate-200">
                {selectedFile ? "Preview" : "Select a file to preview"}
              </h2>

              {selectedFile ? (
                <FilePreview
                  backlogKind={backlogKind}
                  backlogName={name}
                  filePath={selectedFile.path}
                  fileName={selectedFile.name}
                  data-testid={selectors.backlogDetails.filePreview}
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

      {item && (
        <BacklogFormDialog
          isOpen={showEdit}
          mode="edit"
          initialValues={{
            name: item.name,
            title: item.title,
            description: item.description,
            status: item.status,
            priority: item.priority,
            tags: item.tags,
            kind: item.kind,
            researchTarget: item.researchTarget,
          }}
          isSubmitting={updateMutation.isPending}
          submitError={updateError}
          onClose={() => {
            setShowEdit(false);
            updateMutation.reset();
          }}
          onSubmit={(values) =>
            updateMutation.mutate({
              title: values.title,
              description: values.description,
              status: values.status,
              priority: values.priority,
              tags: values.tags,
              researchTarget: values.researchTarget,
            })
          }
        />
      )}

      <ConfirmDialog
        isOpen={showDelete}
        onClose={() => {
          setShowDelete(false);
          deleteMutation.reset();
        }}
        onConfirm={() => deleteMutation.mutate()}
        title="Delete Backlog Item"
        description={`Are you sure you want to delete "${item?.title || name}"? This will remove the backlog folder permanently.`}
        confirmationText={item?.name}
        confirmLabel="Delete Item"
        isLoading={deleteMutation.isPending}
        testIds={{
          dialog: selectors.backlogDetails.deleteDialog,
          confirmButton: selectors.backlogDetails.deleteConfirmButton,
          cancelButton: selectors.backlogDetails.deleteCancelButton,
        }}
      />

      <BacklogAgentDialog
        isOpen={showAgentDialog}
        isSubmitting={agentMutation.isPending}
        backlogKind={backlogKind}
        backlogTitle={item?.title ?? name ?? ""}
        researchTarget={item?.researchTarget}
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
