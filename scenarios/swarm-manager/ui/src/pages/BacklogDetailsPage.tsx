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

import { useState, useCallback, useEffect, useMemo, useRef, type MouseEvent, type PointerEvent } from "react";
import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { useParams, Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  AlertTriangle,
  ArrowRight,
  ArrowRightLeft,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Copy,
  Edit,
  FileText,
  Files,
  Info,
  Lock,
  Loader2,
  MoreHorizontal,
  Play,
  Search,
  Sparkles,
  Square,
  Tags,
  Trash2,
  Upload,
  Wrench,
  X,
} from "lucide-react";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { Dialog } from "../components/ui/dialog";
import { Popover } from "../components/ui/popover";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ErrorBoundary } from "../components/ui/error-boundary";
import { ErrorState } from "../components/ui/error-state";
import { Input } from "../components/ui/input";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { FileTree, type TreeFile } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { FileUpload } from "../components/ui/file-upload";
import { TagList } from "../components/ui/tag-list";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { BacklogAgentDialog } from "../components/backlog/backlog-agent-dialog";
import { IdeaClarifyPanel } from "../components/backlog/idea-clarify-panel";
import { IdeaSuggestionsPanel } from "../components/backlog/idea-suggestions-panel";
import { OperationalTargetsPanel } from "../components/backlog/operational-targets-panel";
import { RequirementFormDialog } from "../components/backlog/requirement-form-dialog";
import { TargetFormDialog } from "../components/backlog/target-form-dialog";
import { QuestionFormDialog } from "../components/backlog/question-form-dialog";
import { SuggestionFormDialog } from "../components/backlog/suggestion-form-dialog";
import { ModuleFormDialog } from "../components/backlog/module-form-dialog";
import {
  buildClarifyQuestionsContent,
  buildSuggestionsContent,
  cn,
  defaultQueryOptions,
  findBacklogFileByPath,
  formatRelativeTime,
  getBacklogNotQueueableReason,
  IDEA_AGENT_FILE_PATHS,
  isBacklogQueueable,
  parseClarifyQuestionsFile,
  parseSuggestionsFile,
} from "../lib";
import { backlogService } from "../services";
import type { QueueResponse } from "../services";
import { selectors } from "../consts/selectors";
import {
  BACKLOG_KIND_LABELS,
  BACKLOG_KINDS,
  BACKLOG_RESEARCH_TARGET_LABELS,
  BACKLOG_STATUS_COLORS,
  formatBacklogStatus,
} from "../types";
import type {
  ArchiveRequirement,
  ArchiveRequirementRecord,
  ArchiveTarget,
  ArchiveTargetFormValues,
  ArchiveTargetsResponse,
  BacklogFile,
  BacklogKind,
  BacklogResearchTarget,
  BacklogStatus,
  IdeaAgentMode,
  ClarifyQuestionFormValues,
  IdeaClarificationQuestion,
  IdeaSuggestion,
  ModuleFormValues,
  ResearchResponse,
  SuggestionFormValues,
} from "../types";
import { selectLatestRunForBacklog, useAgentRunsStore, useBacklogStore } from "../stores";

const RECENT_FILES_LIMIT = 5;
const DEFAULT_PREVIEW_FILE_PATH = "spec.json";
const MIN_FILES_PANEL_WIDTH = 240;
const MAX_FILES_PANEL_WIDTH = 520;
const MIN_PREVIEW_WIDTH = 320;
const RESIZE_HANDLE_WIDTH = 8;
type MobileView = "info" | "files";
type FileActionType = "rename" | "move" | "copy" | "delete";

interface FileActionTarget {
  action: FileActionType;
  target: BacklogFile;
}

interface FileActionMenuState {
  x: number;
  y: number;
  target: BacklogFile;
}

const collectMatchingFiles = (entries: BacklogFile[], query: string): BacklogFile[] => {
  const normalized = query.trim().toLowerCase();
  if (!normalized) return [];

  const matches: BacklogFile[] = [];
  const walk = (items: BacklogFile[]) => {
    items.forEach((item) => {
      if (item.type === "file") {
        const haystack = `${item.name} ${item.path}`.toLowerCase();
        if (haystack.includes(normalized)) {
          matches.push(item);
        }
      }
      if (item.children && item.children.length > 0) {
        walk(item.children);
      }
    });
  };

  walk(entries);
  return matches;
};

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

const buildFollowupPrompt = (mode: IdeaAgentMode): string => {
  if (mode === "suggest") {
    return "Use clarify/questions.json (with answers) to generate actionable suggestions for this idea. Append new suggestions without deleting prior ones.";
  }
  return "Use clarify/questions.json answers to refine the idea and produce an enhanced plan. If suggestions exist, apply accepted ones and ignore rejected ones.";
};

const formatDuration = (seconds?: number): string => {
  if (!seconds || seconds <= 0) return "Unknown";
  if (seconds < 60) return `${Math.round(seconds)}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m`;
};

const getParentPath = (path: string): string => {
  const slashIndex = path.lastIndexOf("/");
  return slashIndex > -1 ? path.slice(0, slashIndex) : "";
};

const getBaseName = (path: string): string => {
  const slashIndex = path.lastIndexOf("/");
  return slashIndex > -1 ? path.slice(slashIndex + 1) : path;
};

const joinPath = (parent: string, name: string): string => (parent ? `${parent}/${name}` : name);

const normalizeDestinationPath = (value: string): string => {
  return value.trim().replace(/\\/g, "/").replace(/^\/+/, "").replace(/\/+$/, "");
};

const remapSelectedPath = (
  currentPath: string,
  target: BacklogFile,
  destinationPath: string
): string | null => {
  if (target.type === "file") {
    return currentPath === target.path ? destinationPath : currentPath;
  }
  const prefix = `${target.path}/`;
  if (currentPath === target.path) return destinationPath;
  if (currentPath.startsWith(prefix)) {
    return `${destinationPath}/${currentPath.slice(prefix.length)}`;
  }
  return currentPath;
};

export function BacklogDetailsPage() {
  const { kind, name } = useParams<{ kind: string; name: string }>();
  const backlogKind = BACKLOG_KINDS.includes(kind as BacklogKind) ? (kind as BacklogKind) : null;
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const removeItem = useBacklogStore((state) => state.removeItem);
  const upsertSpawnedRun = useAgentRunsStore((state) => state.upsertSpawnedRun);
  const refreshRun = useAgentRunsStore((state) => state.refreshRun);
  const stopRun = useAgentRunsStore((state) => state.stopRun);
  const latestAgentRun = useAgentRunsStore((state) => {
    if (!backlogKind || !name) return null;
    return selectLatestRunForBacklog(state, backlogKind, name);
  });
  const workspaceRef = useRef<HTMLDivElement | null>(null);
  const headerFileActionsRef = useRef<HTMLDivElement | null>(null);
  const [filesPanelWidth, setFilesPanelWidth] = useState(320);
  const [isResizing, setIsResizing] = useState(false);
  const [mobileView, setMobileView] = useState<MobileView>("info");
  const [showFilesSheet, setShowFilesSheet] = useState(false);
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const [fileSearch, setFileSearch] = useState("");
  const [recentFiles, setRecentFiles] = useState<BacklogFile[]>([]);
  const [selectedFile, setSelectedFile] = useState<BacklogFile | null>(null);
  const [showFileActionsMenu, setShowFileActionsMenu] = useState(false);
  const [fileContextMenu, setFileContextMenu] = useState<FileActionMenuState | null>(null);
  const [activeFileAction, setActiveFileAction] = useState<FileActionTarget | null>(null);
  const [fileActionInput, setFileActionInput] = useState("");
  const [fileActionError, setFileActionError] = useState<string | null>(null);
  const [showUpload, setShowUpload] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [showAgentDialog, setShowAgentDialog] = useState(false);
  const [scheduleDelaySeconds, setScheduleDelaySeconds] = useState(300);
  const [previewResetKey, setPreviewResetKey] = useState(0);
  const [detailsExpanded, setDetailsExpanded] = useState(true);
  const [agentRunExpanded, setAgentRunExpanded] = useState(true);
  const [selectedTargetIds, setSelectedTargetIds] = useState<Set<string>>(new Set());
  const [selectedRequirementIds, setSelectedRequirementIds] = useState<Set<string>>(new Set());
  const [queueBlockedResult, setQueueBlockedResult] = useState<QueueResponse | null>(null);
  const [reqDialogOpen, setReqDialogOpen] = useState(false);
  const [reqDialogMode, setReqDialogMode] = useState<"create" | "edit">("create");
  const [editingReq, setEditingReq] = useState<{ groupId: string; req?: ArchiveRequirementRecord } | null>(null);
  const [moduleDialogOpen, setModuleDialogOpen] = useState(false);
  const [moduleDialogMode, setModuleDialogMode] = useState<"create" | "edit">("create");
  const [editingModuleId, setEditingModuleId] = useState<string | null>(null);
  const [targetDialogOpen, setTargetDialogOpen] = useState(false);
  const [targetDialogMode, setTargetDialogMode] = useState<"create" | "edit">("create");
  const [editingTarget, setEditingTarget] = useState<ArchiveTarget | null>(null);
  const [questionDialogOpen, setQuestionDialogOpen] = useState(false);
  const [questionDialogMode, setQuestionDialogMode] = useState<"create" | "edit">("create");
  const [editingQuestion, setEditingQuestion] = useState<IdeaClarificationQuestion | null>(null);
  const [questionSubmitting, setQuestionSubmitting] = useState(false);
  const [questionSubmitError, setQuestionSubmitError] = useState<string | null>(null);
  const [suggestionDialogOpen, setSuggestionDialogOpen] = useState(false);
  const [suggestionDialogMode, setSuggestionDialogMode] = useState<"create" | "edit">("create");
  const [editingSuggestion, setEditingSuggestion] = useState<IdeaSuggestion | null>(null);
  const [suggestionSubmitting, setSuggestionSubmitting] = useState(false);
  const [suggestionSubmitError, setSuggestionSubmitError] = useState<string | null>(null);

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
    error: filesQueryError,
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

  const {
    data: archiveTargets,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "archive-targets"],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.getArchiveTargets(backlogKind, name);
    },
    enabled: !!backlogKind && !!name,
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

  const isPageLoading = isLoadingItem && !item;
  const pageError = itemError;
  const filesError = filesQueryError instanceof Error ? filesQueryError : null;

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "file") {
      setSelectedFile(file);
      setMobileView("files");
      setShowFilesSheet(false);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("file", file.path);
        return next;
      }, { replace: true });
      setRecentFiles((prev) => {
        const next = [file, ...prev.filter((entry) => entry.path !== file.path)];
        return next.slice(0, RECENT_FILES_LIMIT);
      });
    }
  }, [setSearchParams]);

  const handleUploadComplete = useCallback(() => {
    if (!backlogKind || !name) return;
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
  }, [queryClient, backlogKind, name]);

  const queueMutation = useMutation({
    onMutate: () => setQueueBlockedResult(null),
    mutationFn: ({ mode, delaySeconds }: { mode: "manual" | "scheduled" | "yolo"; delaySeconds?: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.queue(backlogKind, name, {
        mode,
        delaySeconds,
        startedBy: "swarm-manager-ui",
        confirm: true,
      });
    },
    onSuccess: (result) => {
      if (!backlogKind || !name) return;
      if (result.dryRun && result.blockingReasons.length > 0) {
        setQueueBlockedResult(result);
        return;
      }
      setQueueBlockedResult(null);
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
    mutationFn: ({ mode, prompt, targetKind, contextPaths, contextTargetIds, contextRequirementIds }: {
      mode?: IdeaAgentMode;
      prompt: string;
      targetKind?: BacklogResearchTarget;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.research(backlogKind, name, {
        mode,
        prompt,
        targetKind,
        contextPaths,
        contextTargetIds,
        contextRequirementIds,
      });
    },
    onSuccess: (result: ResearchResponse, variables) => {
      setShowAgentDialog(false);
      if (!backlogKind || !name) return;
      upsertSpawnedRun({
        runId: result.runId,
        taskId: result.taskId,
        baseUrl: result.baseUrl,
        createdAt: result.created,
        backlogKind,
        backlogName: name,
        backlogTitle: item?.title ?? name,
        mode: variables.mode ?? "research",
      });
      void refreshRun(result.runId);
    },
  });

  const clarifyMutation = useMutation({
    mutationFn: async ({
      questions,
      nextMode,
    }: {
      questions: IdeaClarificationQuestion[];
      nextMode: IdeaAgentMode | "none";
    }): Promise<ResearchResponse | null> => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const content = buildClarifyQuestionsContent(clarifyParsed.raw, questions);
      await backlogService.saveFileContent(
        backlogKind,
        name,
        IDEA_AGENT_FILE_PATHS.clarify,
        content,
        "application/json"
      );
      if (nextMode !== "none") {
        return backlogService.research(backlogKind, name, {
          mode: nextMode,
          prompt: buildFollowupPrompt(nextMode),
        });
      }
      return null;
    },
    onSuccess: (result, variables) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify],
      });
      void refetchFiles();
      void refetchClarifyContent();
      if (result) {
        upsertSpawnedRun({
          runId: result.runId,
          taskId: result.taskId,
          baseUrl: result.baseUrl,
          createdAt: result.created,
          backlogKind,
          backlogName: name,
          backlogTitle: item?.title ?? name,
          mode: variables.nextMode,
        });
        void refreshRun(result.runId);
      }
    },
  });

  const suggestionsMutation = useMutation({
    mutationFn: async (updatedSuggestions: IdeaSuggestion[]): Promise<ResearchResponse> => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const content = buildSuggestionsContent(suggestionsParsed.raw, updatedSuggestions);
      await backlogService.saveFileContent(
        backlogKind,
        name,
        IDEA_AGENT_FILE_PATHS.suggest,
        content,
        "application/json"
      );
      return backlogService.research(backlogKind, name, {
        mode: "enhance",
        prompt:
          "Use suggest/suggestions.json decisions to enhance this idea. Apply accepted suggestions, ignore rejected ones, and reference clarify/questions.json answers if available.",
      });
    },
    onSuccess: (result) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({
        queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest],
      });
      void refetchFiles();
      void refetchSuggestionsContent();
      upsertSpawnedRun({
        runId: result.runId,
        taskId: result.taskId,
        baseUrl: result.baseUrl,
        createdAt: result.created,
        backlogKind,
        backlogName: name,
        backlogTitle: item?.title ?? name,
        mode: "enhance",
      });
      void refreshRun(result.runId);
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

  const archiveTargetsQueryKey = ["backlog", backlogKind, name, "archive-targets"];

  const updateReqsMutation = useMutation({
    mutationFn: ({ moduleId, requirements }: { moduleId: string; requirements: ArchiveRequirementRecord[] }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleRequirements(backlogKind, name, moduleId, requirements);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const createModuleMutation = useMutation({
    mutationFn: (payload: ModuleFormValues & { position?: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createModule(backlogKind, name, payload);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const updateModuleMetaMutation = useMutation({
    mutationFn: ({ moduleId, payload }: { moduleId: string; payload: { title: string; description: string } }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateModuleMeta(backlogKind, name, moduleId, payload);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const deleteModuleMutation = useMutation({
    mutationFn: (moduleId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteModule(backlogKind, name, moduleId);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const createTargetMutation = useMutation({
    mutationFn: (target: ArchiveTargetFormValues) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.createArchiveTarget(backlogKind, name, target);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const updateTargetMutation = useMutation({
    mutationFn: ({ targetId, target }: { targetId: string; target: ArchiveTargetFormValues }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.updateArchiveTarget(backlogKind, name, targetId, target);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const deleteTargetMutation = useMutation({
    mutationFn: (targetId: string) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.deleteArchiveTarget(backlogKind, name, targetId);
    },
    onSuccess: () => { queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey }); },
  });

  const handleCreateTarget = useCallback(() => {
    setEditingTarget(null);
    setTargetDialogMode("create");
    setTargetDialogOpen(true);
  }, []);

  const handleEditTarget = useCallback((target: ArchiveTarget) => {
    setEditingTarget(target);
    setTargetDialogMode("edit");
    setTargetDialogOpen(true);
  }, []);

  const handleDeleteTarget = useCallback((targetId: string) => {
    if (!window.confirm(`Delete target "${targetId}"?`)) return;
    deleteTargetMutation.mutate(targetId);
  }, [deleteTargetMutation]);

  const handleTargetDialogSubmit = useCallback((values: ArchiveTargetFormValues) => {
    if (targetDialogMode === "create") {
      createTargetMutation.mutate(values, {
        onSuccess: () => { setTargetDialogOpen(false); setEditingTarget(null); },
      });
    } else if (editingTarget) {
      updateTargetMutation.mutate({ targetId: editingTarget.id, target: values }, {
        onSuccess: () => { setTargetDialogOpen(false); setEditingTarget(null); },
      });
    }
  }, [targetDialogMode, editingTarget, createTargetMutation, updateTargetMutation]);

  const handleCreateRequirement = useCallback((groupId: string) => {
    setEditingReq({ groupId });
    setReqDialogMode("create");
    setReqDialogOpen(true);
  }, []);

  const handleEditRequirement = useCallback((groupId: string, requirement: ArchiveRequirement) => {
    setEditingReq({ groupId, req: requirement as ArchiveRequirementRecord });
    setReqDialogMode("edit");
    setReqDialogOpen(true);
  }, []);

  const handleDeleteRequirement = useCallback((groupId: string, requirementId: string) => {
    if (!window.confirm(`Delete requirement "${requirementId}"?`)) return;
    if (!archiveTargets) return;
    const findGroup = (groups: typeof archiveTargets.requirements): typeof archiveTargets.requirements[0] | undefined => {
      for (const g of groups) {
        if (g.id === groupId) return g;
        const found = findGroup(g.children);
        if (found) return found;
      }
      return undefined;
    };
    const group = findGroup(archiveTargets.requirements);
    if (!group) return;
    const updated = group.requirements.filter((r) => r.id !== requirementId) as ArchiveRequirementRecord[];
    updateReqsMutation.mutate({ moduleId: groupId, requirements: updated });
  }, [archiveTargets, updateReqsMutation]);

  const handleReorderRequirement = useCallback((groupId: string, requirementId: string, direction: "up" | "down") => {
    if (!archiveTargets) return;
    const findGroup = (groups: typeof archiveTargets.requirements): typeof archiveTargets.requirements[0] | undefined => {
      for (const g of groups) {
        if (g.id === groupId) return g;
        const found = findGroup(g.children);
        if (found) return found;
      }
      return undefined;
    };
    const group = findGroup(archiveTargets.requirements);
    if (!group) return;
    const reqs = [...group.requirements] as ArchiveRequirementRecord[];
    const idx = reqs.findIndex((r) => r.id === requirementId);
    if (idx < 0) return;
    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= reqs.length) return;
    const tmp = reqs[idx]!;
    reqs[idx] = reqs[swapIdx]!;
    reqs[swapIdx] = tmp;
    updateReqsMutation.mutate({ moduleId: groupId, requirements: reqs });
  }, [archiveTargets, updateReqsMutation]);

  const handleCreateModule = useCallback(() => {
    setEditingModuleId(null);
    setModuleDialogMode("create");
    setModuleDialogOpen(true);
  }, []);

  const handleEditModule = useCallback((groupId: string) => {
    setEditingModuleId(groupId);
    setModuleDialogMode("edit");
    setModuleDialogOpen(true);
  }, []);

  const handleDeleteModule = useCallback((groupId: string) => {
    if (!window.confirm(`Delete module "${groupId}" and all its requirements?`)) return;
    deleteModuleMutation.mutate(groupId);
  }, [deleteModuleMutation]);

  const handleReqDialogSubmit = useCallback((values: ArchiveRequirementRecord) => {
    if (!editingReq || !archiveTargets) return;
    const { groupId } = editingReq;
    const findGroup = (groups: typeof archiveTargets.requirements): typeof archiveTargets.requirements[0] | undefined => {
      for (const g of groups) {
        if (g.id === groupId) return g;
        const found = findGroup(g.children);
        if (found) return found;
      }
      return undefined;
    };
    const group = findGroup(archiveTargets.requirements);
    if (!group) return;
    let updated: ArchiveRequirementRecord[];
    if (reqDialogMode === "edit") {
      updated = group.requirements.map((r) => r.id === values.id ? values : r) as ArchiveRequirementRecord[];
    } else {
      updated = [...group.requirements as ArchiveRequirementRecord[], values];
    }
    updateReqsMutation.mutate({ moduleId: groupId, requirements: updated }, {
      onSuccess: () => { setReqDialogOpen(false); setEditingReq(null); },
    });
  }, [editingReq, archiveTargets, reqDialogMode, updateReqsMutation]);

  const handleModuleDialogSubmit = useCallback((values: ModuleFormValues) => {
    if (moduleDialogMode === "create") {
      createModuleMutation.mutate(values, {
        onSuccess: () => { setModuleDialogOpen(false); },
      });
    } else if (editingModuleId) {
      updateModuleMetaMutation.mutate({
        moduleId: editingModuleId,
        payload: { title: values.title, description: values.description },
      }, {
        onSuccess: () => { setModuleDialogOpen(false); setEditingModuleId(null); },
      });
    }
  }, [moduleDialogMode, editingModuleId, createModuleMutation, updateModuleMetaMutation]);

  // --- Question CRUD handlers ---
  const saveQuestions = useCallback(async (questions: IdeaClarificationQuestion[]) => {
    if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
    const content = buildClarifyQuestionsContent(clarifyParsed.raw, questions);
    await backlogService.saveFileContent(backlogKind, name, IDEA_AGENT_FILE_PATHS.clarify, content, "application/json");
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.clarify] });
    void refetchFiles();
    void refetchClarifyContent();
  }, [backlogKind, name, clarifyParsed.raw, queryClient, refetchFiles, refetchClarifyContent]);

  const handleAddQuestion = useCallback(() => {
    setEditingQuestion(null);
    setQuestionDialogMode("create");
    setQuestionSubmitError(null);
    setQuestionDialogOpen(true);
  }, []);

  const handleEditQuestion = useCallback((question: IdeaClarificationQuestion) => {
    setEditingQuestion(question);
    setQuestionDialogMode("edit");
    setQuestionSubmitError(null);
    setQuestionDialogOpen(true);
  }, []);

  const handleDeleteQuestion = useCallback(async (questionId: string) => {
    if (!window.confirm(`Delete this question?`)) return;
    const updated = clarifyParsed.questions.filter((q) => q.id !== questionId);
    try {
      await saveQuestions(updated);
    } catch { /* error handled by query invalidation */ }
  }, [clarifyParsed.questions, saveQuestions]);

  const handleQuestionDialogSubmit = useCallback(async (values: ClarifyQuestionFormValues) => {
    setQuestionSubmitting(true);
    setQuestionSubmitError(null);
    try {
      let updated: IdeaClarificationQuestion[];
      if (questionDialogMode === "edit" && editingQuestion) {
        updated = clarifyParsed.questions.map((q) =>
          q.id === editingQuestion.id ? { ...q, ...values } : q
        );
      } else {
        const newId = `q-${Date.now().toString(36)}`;
        updated = [...clarifyParsed.questions, { id: newId, ...values }];
      }
      await saveQuestions(updated);
      setQuestionDialogOpen(false);
      setEditingQuestion(null);
    } catch (err) {
      setQuestionSubmitError(err instanceof Error ? err.message : "Failed to save question.");
    } finally {
      setQuestionSubmitting(false);
    }
  }, [questionDialogMode, editingQuestion, clarifyParsed.questions, saveQuestions]);

  // --- Suggestion CRUD handlers ---
  const saveSuggestions = useCallback(async (suggestions: IdeaSuggestion[]) => {
    if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
    const content = buildSuggestionsContent(suggestionsParsed.raw, suggestions);
    await backlogService.saveFileContent(backlogKind, name, IDEA_AGENT_FILE_PATHS.suggest, content, "application/json");
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
    queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "agent-file", IDEA_AGENT_FILE_PATHS.suggest] });
    void refetchFiles();
    void refetchSuggestionsContent();
  }, [backlogKind, name, suggestionsParsed.raw, queryClient, refetchFiles, refetchSuggestionsContent]);

  const handleAddSuggestion = useCallback(() => {
    setEditingSuggestion(null);
    setSuggestionDialogMode("create");
    setSuggestionSubmitError(null);
    setSuggestionDialogOpen(true);
  }, []);

  const handleEditSuggestion = useCallback((suggestion: IdeaSuggestion) => {
    setEditingSuggestion(suggestion);
    setSuggestionDialogMode("edit");
    setSuggestionSubmitError(null);
    setSuggestionDialogOpen(true);
  }, []);

  const handleDeleteSuggestion = useCallback(async (suggestionId: string) => {
    if (!window.confirm(`Delete this suggestion?`)) return;
    const updated = suggestionsParsed.suggestions.filter((s) => s.id !== suggestionId);
    try {
      await saveSuggestions(updated);
    } catch { /* error handled by query invalidation */ }
  }, [suggestionsParsed.suggestions, saveSuggestions]);

  const handleSuggestionDialogSubmit = useCallback(async (values: SuggestionFormValues) => {
    setSuggestionSubmitting(true);
    setSuggestionSubmitError(null);
    try {
      let updated: IdeaSuggestion[];
      if (suggestionDialogMode === "edit" && editingSuggestion) {
        updated = suggestionsParsed.suggestions.map((s) =>
          s.id === editingSuggestion.id ? { ...s, ...values } : s
        );
      } else {
        const newId = `s-${Date.now().toString(36)}`;
        updated = [...suggestionsParsed.suggestions, { id: newId, ...values }];
      }
      await saveSuggestions(updated);
      setSuggestionDialogOpen(false);
      setEditingSuggestion(null);
    } catch (err) {
      setSuggestionSubmitError(err instanceof Error ? err.message : "Failed to save suggestion.");
    } finally {
      setSuggestionSubmitting(false);
    }
  }, [suggestionDialogMode, editingSuggestion, suggestionsParsed.suggestions, saveSuggestions]);

  const fileActionMutation = useMutation({
    mutationFn: async ({ action, target, destinationPath }: { action: FileActionType; target: BacklogFile; destinationPath?: string }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      if (action === "rename") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.renameFile(backlogKind, name, target.path, destinationPath);
      }
      if (action === "move") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.moveFile(backlogKind, name, target.path, destinationPath);
      }
      if (action === "copy") {
        if (!destinationPath) throw new Error("Destination path is required");
        return backlogService.copyFile(backlogKind, name, target.path, destinationPath);
      }
      return backlogService.deleteFile(backlogKind, name, target.path);
    },
    onSuccess: (_result, variables) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      setFileActionError(null);
      setActiveFileAction(null);
      setFileContextMenu(null);
      setShowFileActionsMenu(false);

      const currentSelectedPath = selectedFile?.path;
      if (!currentSelectedPath) return;

      if (variables.action === "delete") {
        const affectedPath =
          currentSelectedPath === variables.target.path ||
          (variables.target.type === "directory" && currentSelectedPath.startsWith(`${variables.target.path}/`));
        if (affectedPath) {
          setSelectedFile(null);
          setSearchParams((prev) => {
            const next = new URLSearchParams(prev);
            next.delete("file");
            return next;
          }, { replace: true });
        }
        return;
      }

      if (!variables.destinationPath) return;
      const remapped = remapSelectedPath(currentSelectedPath, variables.target, variables.destinationPath);
      if (!remapped || remapped === currentSelectedPath) return;
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("file", remapped);
        return next;
      }, { replace: true });
    },
    onError: (error) => {
      const message = error instanceof Error ? error.message : "Failed to apply file action.";
      setFileActionError(message);
    },
  });

  const updateError = updateMutation.isError
    ? updateMutation.error instanceof Error ? updateMutation.error.message : "Failed to update backlog item. Please try again."
    : null;
  const deleteError = deleteMutation.isError
    ? deleteMutation.error instanceof Error ? deleteMutation.error.message : "Failed to delete backlog item. Please try again."
    : null;
  const agentError = agentMutation.isError
    ? agentMutation.error instanceof Error ? agentMutation.error.message : "Failed to start the agent. Make sure agent-manager is running."
    : null;
  const clarifyError = clarifyMutation.isError
    ? clarifyMutation.error instanceof Error ? clarifyMutation.error.message : "Failed to save answers or start the next agent."
    : null;
  const suggestionsError = suggestionsMutation.isError
    ? suggestionsMutation.error instanceof Error ? suggestionsMutation.error.message : "Failed to save suggestions or start the Enhance agent."
    : null;
  const queueError = queueMutation.isError
    ? queueMutation.error instanceof Error
      ? queueMutation.error.message
      : "Failed to queue backlog item. Please try again."
    : null;
  const convertError = convertMutation.isError
    ? convertMutation.error instanceof Error ? convertMutation.error.message : "Failed to convert backlog item. Please try again."
    : null;

  const canQueue = Boolean(item && isBacklogQueueable(item));
  const queueBlockedReason = item ? getBacklogNotQueueableReason(item) : null;

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
  const hasNotes = Boolean(item?.kind === "idea" && (clarifyFile || suggestionsFile));
  const searchResults = useMemo(
    () => collectMatchingFiles(files ?? [], fileSearch),
    [files, fileSearch]
  );
  const selectedFileParam = searchParams.get("file");

  const handleResizeStart = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    setIsResizing(true);
  }, []);

  const handlePreviewRetry = useCallback(() => {
    setPreviewResetKey((prev) => prev + 1);
  }, []);

  useEffect(() => {
    if (!isResizing) return;

    const handlePointerMove = (event: globalThis.PointerEvent) => {
      if (!workspaceRef.current) return;
      const bounds = workspaceRef.current.getBoundingClientRect();
      const maxWidth = Math.max(
        MIN_FILES_PANEL_WIDTH,
        Math.min(
          MAX_FILES_PANEL_WIDTH,
          bounds.width - MIN_PREVIEW_WIDTH - RESIZE_HANDLE_WIDTH
        )
      );
      const nextWidth = clamp(event.clientX - bounds.left, MIN_FILES_PANEL_WIDTH, maxWidth);
      setFilesPanelWidth(nextWidth);
    };

    const handlePointerUp = () => {
      setIsResizing(false);
    };

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", handlePointerMove);
    window.addEventListener("pointerup", handlePointerUp);

    return () => {
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener("pointermove", handlePointerMove);
      window.removeEventListener("pointerup", handlePointerUp);
    };
  }, [isResizing]);

  useEffect(() => {
    if (!files || files.length === 0) return;

    const requestedPath = selectedFileParam || DEFAULT_PREVIEW_FILE_PATH;
    const resolvedFile = findBacklogFileByPath(files, requestedPath);

    if (resolvedFile) {
      setSelectedFile((prev) => (prev?.path === resolvedFile.path ? prev : resolvedFile));
      if (!selectedFileParam) {
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          next.set("file", resolvedFile.path);
          return next;
        }, { replace: true });
      }
      return;
    }

    if (selectedFileParam) {
      const fallbackFile = findBacklogFileByPath(files, DEFAULT_PREVIEW_FILE_PATH);
      if (fallbackFile) {
        setSelectedFile((prev) => (prev?.path === fallbackFile.path ? prev : fallbackFile));
        setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          next.set("file", fallbackFile.path);
          return next;
        }, { replace: true });
        return;
      }
    }

    setSelectedFile(null);
  }, [files, selectedFileParam, setSearchParams]);

  useEffect(() => {
    setMobileView("info");
    setShowFilesSheet(false);
  }, [backlogKind, name]);

  useEffect(() => {
    if (mobileView === "info" && showFilesSheet) {
      setShowFilesSheet(false);
    }
  }, [mobileView, showFilesSheet]);

  useEffect(() => {
    setShowFileActionsMenu(false);
    setFileContextMenu(null);
    setActiveFileAction(null);
    setFileActionInput("");
    setFileActionError(null);
  }, [selectedFile?.path, backlogKind, name]);

  useEffect(() => {
    if (!showFileActionsMenu) return;
    const onMouseDown = (event: globalThis.MouseEvent) => {
      if (headerFileActionsRef.current && !headerFileActionsRef.current.contains(event.target as Node)) {
        setShowFileActionsMenu(false);
      }
    };
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [showFileActionsMenu]);

  const openFileActionDialog = useCallback((action: FileActionType, target: BacklogFile) => {
    setActiveFileAction({ action, target });
    setFileActionError(null);
    if (action === "rename") {
      setFileActionInput(getBaseName(target.path));
      return;
    }
    if (action === "move") {
      setFileActionInput(target.path);
      return;
    }
    if (action === "copy") {
      const suffix = target.type === "directory" ? "-copy" : "-copy";
      setFileActionInput(joinPath(getParentPath(target.path), `${getBaseName(target.path)}${suffix}`));
      return;
    }
    setFileActionInput("");
  }, []);

  const handleFileContextMenu = useCallback((file: TreeFile, event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    setShowFileActionsMenu(false);
    setFileContextMenu({
      x: event.clientX,
      y: event.clientY,
      target: file as BacklogFile,
    });
  }, []);

  const handleOpenHeaderMenu = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    setFileContextMenu(null);
    setShowFileActionsMenu((prev) => !prev);
  }, []);

  const handleFileActionConfirm = useCallback(() => {
    if (!activeFileAction) return;
    const { action, target } = activeFileAction;
    if (action === "delete") {
      fileActionMutation.mutate({ action, target });
      return;
    }

    if (action === "rename") {
      const nextName = fileActionInput.trim();
      if (!nextName || nextName.includes("/")) {
        setFileActionError("Rename requires a file or folder name without slashes.");
        return;
      }
      const destinationPath = joinPath(getParentPath(target.path), nextName);
      fileActionMutation.mutate({ action, target, destinationPath });
      return;
    }

    const destinationPath = normalizeDestinationPath(fileActionInput);
    if (!destinationPath) {
      setFileActionError("Destination path is required.");
      return;
    }
    fileActionMutation.mutate({ action, target, destinationPath });
  }, [activeFileAction, fileActionInput, fileActionMutation]);

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
  const scheduleDelayValue = Number.isFinite(scheduleDelaySeconds) && scheduleDelaySeconds >= 0 ? scheduleDelaySeconds : 0;
  const isProtectedSelectedFile = selectedFile?.path === "spec.json";

  const renderFileActionItems = (target: BacklogFile, closeMenu: () => void) => {
    const isProtected = target.path === "spec.json";
    const rowClass = "flex w-full items-center justify-start gap-2 px-3 py-2 text-sm text-slate-100 hover:bg-slate-800/80";
    return (
      <div className="py-1" data-testid="backlog-file-actions-menu">
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("rename", target);
          }}
        >
          <Edit className="h-4 w-4 text-slate-300" />
          Rename
        </button>
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("move", target);
          }}
        >
          <ArrowRightLeft className="h-4 w-4 text-slate-300" />
          Move
        </button>
        <button
          type="button"
          className={rowClass}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("copy", target);
          }}
        >
          <Copy className="h-4 w-4 text-slate-300" />
          Copy
        </button>
        <button
          type="button"
          className={cn(rowClass, "text-red-300 hover:bg-red-500/20")}
          disabled={isProtected}
          onClick={() => {
            closeMenu();
            openFileActionDialog("delete", target);
          }}
        >
          <Trash2 className="h-4 w-4 text-red-300" />
          Delete
        </button>
        {isProtected && (
          <p className="flex items-center gap-2 px-3 py-2 text-xs text-slate-400">
            <Lock className="h-3.5 w-3.5" />
            `spec.json` is protected.
          </p>
        )}
      </div>
    );
  };

  const filesButton = (
    <Button
      variant="outline"
      size="sm"
      className="lg:hidden"
      onClick={() => setShowFilesSheet(true)}
    >
      <Files className="mr-2 h-4 w-4" />
      Files
    </Button>
  );
  const fileHeaderActions = (
    <>
      {filesButton}
      {selectedFile && (
        <div className="relative" ref={headerFileActionsRef}>
          <Button
            variant="outline"
            size="sm"
            onClick={handleOpenHeaderMenu}
            aria-label="File actions"
            title="File actions"
            className="h-8 w-8 p-0"
            data-testid="file-header-actions-trigger"
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
          {showFileActionsMenu && (
            <div
              className="absolute right-0 top-10 z-30 min-w-[180px] overflow-visible rounded-md border border-white/10 bg-slate-900 shadow-lg"
              data-testid="file-header-actions-popover"
            >
              {renderFileActionItems(selectedFile, () => setShowFileActionsMenu(false))}
            </div>
          )}
        </div>
      )}
      {selectedFile && (
        <span className={cn("text-xs text-slate-500", isProtectedSelectedFile && "text-amber-300")}>
          {isProtectedSelectedFile ? "Protected file" : ""}
        </span>
      )}
    </>
  );
  const fileBrowserContent = (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-end border-b border-white/10 px-3 py-2">
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
      <div className="flex-1 space-y-4 overflow-y-auto px-3 pb-4 pt-4">
        <div className="space-y-3 lg:hidden">
          <Input
            type="text"
            value={fileSearch}
            onChange={(event) => setFileSearch(event.target.value)}
            placeholder="Search files"
            leftIcon={<Search className="h-4 w-4" />}
            rightSlot={
              fileSearch.trim().length > 0 ? (
                <button
                  type="button"
                  onClick={() => setFileSearch("")}
                  className="rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                  aria-label="Clear search"
                >
                  <X className="h-4 w-4" />
                </button>
              ) : null
            }
          />
          {recentFiles.length > 0 && fileSearch.trim().length === 0 && (
            <div className="space-y-2">
              <p className="text-xs uppercase tracking-wider text-slate-500">Recent files</p>
              <div className="space-y-1">
                {recentFiles.map((file) => (
                  <button
                    key={file.path}
                    type="button"
                    onClick={() => handleFileSelect(file)}
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
            <InlineLoadingIndicator
              label="Loading files..."
              className="border-transparent bg-transparent px-0 text-slate-400"
              testId="backlog-files-loading"
            />
          </div>
        ) : filesError ? (
          <ErrorState
            error={filesError}
            title="Unable to load files"
            message="Try again to reload the file tree."
            onRetry={() => {
              void refetchFiles();
            }}
          />
        ) : fileSearch.trim().length > 0 ? (
          searchResults.length > 0 ? (
            <div className="space-y-1">
              {searchResults.map((file) => (
                <button
                  key={file.path}
                  type="button"
                  onClick={() => handleFileSelect(file)}
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
          ) : (
            <div className="rounded-lg border border-white/10 bg-slate-800/30 p-6 text-center text-sm text-slate-500">
              No files match "{fileSearch.trim()}".
            </div>
          )
        ) : (
          <FileTree
            files={files ?? []}
            onFileSelect={handleFileSelect}
            onItemContextMenu={handleFileContextMenu}
            selectedPath={selectedFile?.path}
            className="lg:rounded-none lg:border-0 lg:bg-transparent lg:py-0"
            data-testid={selectors.backlogDetails.fileTree}
          />
        )}
        <Popover
          isOpen={Boolean(fileContextMenu)}
          onClose={() => setFileContextMenu(null)}
          x={fileContextMenu?.x}
          y={fileContextMenu?.y}
          delayClickOutside
          testId="file-tree-context-popover"
        >
          {fileContextMenu
            ? renderFileActionItems(fileContextMenu.target, () => setFileContextMenu(null))
            : null}
        </Popover>
      </div>
    </div>
  );

  const handleTargetToggle = useCallback((id: string) => {
    setSelectedTargetIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const handleRequirementToggle = useCallback((id: string) => {
    setSelectedRequirementIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const detailsPanel = item ? (
    <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
      <div className="space-y-4">
        <button
          type="button"
          onClick={() => setDetailsExpanded(!detailsExpanded)}
          className="flex w-full items-center gap-2 border-b border-slate-800 pb-2 text-left"
        >
          {detailsExpanded ? (
            <ChevronDown className="h-4 w-4 text-slate-400" />
          ) : (
            <ChevronRight className="h-4 w-4 text-slate-400" />
          )}
          <Info className="h-4 w-4 text-slate-400" />
          <h2 className="text-base font-semibold text-slate-100">Details</h2>
        </button>
        {detailsExpanded && (
          <>
            <p
              className="text-sm leading-relaxed text-slate-300"
              data-testid={selectors.backlogDetails.description}
            >
              {item.description || "No description provided"}
            </p>
            {item.tags.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
                  <Tags className="h-3.5 w-3.5" />
                  Tags
                </div>
                <TagList tags={item.tags} maxTags={10} />
              </div>
            )}
            <div className="grid grid-cols-2 gap-3 border-t border-slate-800 pt-3">
              <div className="space-y-1">
                <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Created</p>
                <p className="text-sm text-slate-300" title={new Date(item.created).toLocaleString()}>
                  {formatRelativeTime(item.created)}
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">Updated</p>
                <p className="text-sm text-slate-300" title={new Date(item.updated).toLocaleString()}>
                  {formatRelativeTime(item.updated)}
                </p>
              </div>
            </div>
          </>
        )}
      </div>
    </Card>
  ) : null;

  const notesPanel = hasNotes ? (
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
          onAdd={handleAddQuestion}
          onEdit={handleEditQuestion}
          onDelete={handleDeleteQuestion}
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
          onAdd={handleAddSuggestion}
          onEdit={handleEditSuggestion}
          onDelete={handleDeleteSuggestion}
        />
      )}
    </div>
  ) : null;

  const renderActionButtons = (closeOnAction = false) => {
    const runAction = (action: () => void) => {
      if (closeOnAction) {
        setShowActionsSheet(false);
      }
      action();
    };

    const rowButtonClass =
      "h-10 w-full justify-start rounded-lg border-slate-700/80 bg-slate-900/40 px-3 text-sm text-slate-100 hover:bg-slate-800/70";
    const primaryRowButtonClass =
      "h-10 w-full justify-start rounded-lg border-transparent bg-slate-100 px-3 text-sm text-slate-900 hover:bg-white";
    const destructiveRowButtonClass =
      "h-10 w-full justify-start rounded-lg border-red-500/30 bg-red-500/10 px-3 text-sm text-red-200 hover:bg-red-500/20";

    return (
      <div className="space-y-2">
        {canQueue && (
          <div className="space-y-2">
            <Button
              variant="outline"
              size="sm"
              className={rowButtonClass}
              onClick={() => runAction(() => queueMutation.mutate({ mode: "manual" }))}
              disabled={queueMutation.isPending}
            >
              <HeaderIcon className="mr-2 h-4 w-4" />
              Queue
            </Button>
            <Button
              variant="default"
              size="sm"
              className={primaryRowButtonClass}
              onClick={() => runAction(() => queueMutation.mutate({ mode: "yolo" }))}
              disabled={queueMutation.isPending}
            >
              <Play className="mr-2 h-4 w-4" />
              Start Now
            </Button>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={0}
                step={1}
                value={scheduleDelayValue}
                onChange={(event) => setScheduleDelaySeconds(Number(event.target.value || 0))}
                className="h-10 flex-1 rounded-lg border-slate-700/80 bg-slate-900/40"
                aria-label="Schedule delay seconds"
              />
              <Button
                variant="outline"
                size="sm"
                className="h-10 shrink-0 rounded-lg border-slate-700/80 bg-slate-900/40 px-3 text-slate-100 hover:bg-slate-800/70"
                onClick={() =>
                  runAction(() => queueMutation.mutate({ mode: "scheduled", delaySeconds: scheduleDelayValue }))
                }
                disabled={queueMutation.isPending}
              >
                Schedule
              </Button>
            </div>
          </div>
        )}
        {!canQueue && queueBlockedReason ? (
          <p className="text-xs text-slate-500">{queueBlockedReason}</p>
        ) : null}
        {canConvert && convertTarget && (
          <Button
            variant="default"
            size="sm"
            className={primaryRowButtonClass}
            onClick={() => runAction(() => convertMutation.mutate(convertTarget))}
            disabled={convertMutation.isPending}
          >
            {convertMutation.isPending ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            ) : (
              <ArrowRight className="mr-2 h-4 w-4" />
            )}
            {convertMutation.isPending
              ? "Converting..."
              : `Convert to ${BACKLOG_KIND_LABELS[convertTarget]}`}
          </Button>
        )}
        <Button
          variant="outline"
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => setShowEdit(true))}
        >
          <Edit className="mr-2 h-4 w-4" />
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => setShowAgentDialog(true))}
        >
          <Sparkles className="mr-2 h-4 w-4" />
          {agentLabel}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className={destructiveRowButtonClass}
          onClick={() => runAction(() => setShowDelete(true))}
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Delete
        </Button>
      </div>
    );
  };

  const fileWorkspace = (
    <div className="flex-1 min-h-0">
      <div className="h-full overflow-hidden lg:rounded-xl lg:border lg:border-white/10 lg:bg-slate-900/30">
        <div
          ref={workspaceRef}
          className={cn(
            "flex h-full flex-1 flex-col lg:flex-row min-h-[calc(100dvh-6rem)] lg:min-h-[calc(100dvh-16rem)]",
            isResizing && "select-none"
          )}
        >
          <div className="hidden lg:flex flex-col" style={{ width: filesPanelWidth }}>
            {fileBrowserContent}
          </div>
          <div
            className="hidden lg:flex w-2 items-center justify-center bg-slate-900/40 border-x border-white/10 cursor-col-resize"
            onPointerDown={handleResizeStart}
            role="separator"
            aria-orientation="vertical"
            aria-valuenow={Math.round(filesPanelWidth)}
            aria-valuemin={MIN_FILES_PANEL_WIDTH}
            aria-valuemax={MAX_FILES_PANEL_WIDTH}
          >
            <div className="h-10 w-1 rounded-full bg-slate-700/80" />
          </div>
          <div className="flex flex-1 flex-col min-w-0">
            {selectedFile ? (
              <ErrorBoundary
                key={`${selectedFile.path}-${previewResetKey}`}
                fallback={
                  <div className="flex flex-1 items-center justify-center p-6">
                    <ErrorState
                      title="Unable to render file preview"
                      message="Try reloading the preview or choose another file."
                      onRetry={handlePreviewRetry}
                    />
                  </div>
                }
              >
                <FilePreview
                  backlogKind={backlogKind}
                  backlogName={name}
                  filePath={selectedFile.path}
                  fileName={selectedFile.name}
                  compactHeader
                  stickyHeader
                  headerActions={fileHeaderActions}
                  className="flex-1 min-h-0 border-0 rounded-none bg-transparent"
                  contentClassName="flex-1 max-h-none min-h-0"
                  data-testid={selectors.backlogDetails.filePreview}
                />
              </ErrorBoundary>
            ) : (
              <>
                <div className="flex items-center justify-between border-b border-white/10 bg-slate-800/50 px-3 py-2 sm:px-4 sm:py-3">
                  <span className="text-sm font-medium text-slate-300">No file selected</span>
                  {filesButton}
                </div>
                <div className="flex flex-1 items-center justify-center p-8 text-center text-slate-500">
                  Select a file to preview its contents
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      <Dialog
        isOpen={showFilesSheet}
        onClose={() => setShowFilesSheet(false)}
        title="Files"
        maxWidth="max-w-md"
      >
        <div className="max-h-[60vh] overflow-y-auto -mx-2">
          {fileBrowserContent}
        </div>
      </Dialog>
    </div>
  );

  const mobileInfoView = (
    <div className="flex-1 space-y-3 overflow-y-auto px-3 py-3 pb-4">
      {(queueError || queueBlockedResult || deleteError || convertError) && (
        <Card padding="sm" className="space-y-2 rounded-lg border-slate-700/60 bg-slate-900/45">
          {queueBlockedResult && (
            <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-200">
              <div className="flex items-center gap-2 font-medium">
                <AlertTriangle className="h-4 w-4 shrink-0 text-amber-400" />
                {queueBlockedResult.message}
              </div>
              {queueBlockedResult.blockingReasons.length > 0 && (
                <ul className="mt-1.5 list-disc space-y-0.5 pl-6 text-amber-300/90">
                  {queueBlockedResult.blockingReasons.map((reason, i) => (
                    <li key={i}>{reason}</li>
                  ))}
                </ul>
              )}
              {queueBlockedResult.unansweredQuestions > 0 && (
                <p className="mt-1 text-xs text-amber-300/70">
                  {queueBlockedResult.unansweredQuestions} unanswered question{queueBlockedResult.unansweredQuestions !== 1 ? "s" : ""}
                </p>
              )}
            </div>
          )}
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
        </Card>
      )}
      {latestAgentRun && (
        <Card padding="sm" className="rounded-lg border-cyan-500/30 bg-cyan-500/10">
          <button
            type="button"
            onClick={() => setAgentRunExpanded(!agentRunExpanded)}
            className="flex w-full items-center gap-2 text-left"
          >
            {agentRunExpanded ? (
              <ChevronDown className="h-4 w-4 text-slate-400" />
            ) : (
              <ChevronRight className="h-4 w-4 text-slate-400" />
            )}
            <span className="flex-1 text-sm font-semibold text-slate-100">Last Agent Run</span>
            <span className="rounded-full bg-slate-900/70 px-2 py-0.5 text-xs text-cyan-200">
              {latestAgentRun.status.replace("_", " ")}
            </span>
          </button>
          {agentRunExpanded && (
            <div className="mt-2 space-y-2">
              <p className="font-mono text-xs text-cyan-300">{latestAgentRun.runId}</p>
              <p className="text-xs text-slate-300">Spawned {formatRelativeTime(latestAgentRun.createdAt)}</p>
              <p className="text-xs text-slate-300">Duration {formatDuration(latestAgentRun.durationSeconds)}</p>
              {latestAgentRun.errorMessage ? (
                <p className="rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200">
                  {latestAgentRun.errorMessage}
                </p>
              ) : null}
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void refreshRun(latestAgentRun.runId)}
                >
                  Refresh
                </Button>
                {latestAgentRun.active && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => void stopRun(latestAgentRun.runId)}
                    disabled={latestAgentRun.isStopping}
                  >
                    <Square className="mr-2 h-3.5 w-3.5" />
                    {latestAgentRun.isStopping ? "Stopping..." : "Stop"}
                  </Button>
                )}
              </div>
            </div>
          )}
        </Card>
      )}
      {detailsPanel}
      {notesPanel}
      {archiveTargets?.has_archive && (
        <OperationalTargetsPanel
          targets={archiveTargets.targets}
          requirements={archiveTargets.requirements}
          selectedTargetIds={selectedTargetIds}
          selectedRequirementIds={selectedRequirementIds}
          onTargetToggle={handleTargetToggle}
          onRequirementToggle={handleRequirementToggle}
          editable
          onCreateRequirement={handleCreateRequirement}
          onEditRequirement={handleEditRequirement}
          onDeleteRequirement={handleDeleteRequirement}
          onReorderRequirement={handleReorderRequirement}
          onCreateModule={handleCreateModule}
          onEditModule={handleEditModule}
          onDeleteModule={handleDeleteModule}
          onCreateTarget={handleCreateTarget}
          onEditTarget={handleEditTarget}
          onDeleteTarget={handleDeleteTarget}
        />
      )}
    </div>
  );

  return (
    <div className="space-y-0 lg:space-y-6" data-testid={selectors.backlogDetails.page}>

      {isPageLoading && (
        <PageLoadingState
          label="Loading backlog details..."
          variant="detail"
          testId="backlog-details-loading-state"
        />
      )}

      {pageError && (
        <ErrorState
          error={pageError}
          title="Unable to load backlog item"
          onRetry={() => {
            void refetchItem();
          }}
        />
      )}

      {item && !pageError && (
        <>
          <div className="flex h-dvh flex-col overflow-hidden lg:hidden">
            <div className="sticky top-0 z-30 flex items-center gap-2 border-b border-slate-800 bg-slate-950/95 px-3 py-2 backdrop-blur">
              <Button
                asChild
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
              >
                <Link
                  to="/backlog"
                  data-testid={selectors.backlogDetails.backButton}
                  aria-label="Back to backlog"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Link>
              </Button>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-slate-100" data-testid={selectors.backlogDetails.title}>
                  {item.title}
                </p>
                <p className="truncate text-xs text-slate-400">
                  {BACKLOG_KIND_LABELS[item.kind]} · {formatBacklogStatus(item.status)}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-9 rounded-lg border-slate-700/80 bg-slate-900/45 px-3 text-xs font-medium text-slate-100 hover:bg-slate-800/70"
                onClick={() => setMobileView((prev) => (prev === "info" ? "files" : "info"))}
              >
                {mobileView === "info" ? (
                  <>
                    <Files className="mr-1.5 h-4 w-4" />
                    Files
                  </>
                ) : (
                  <>
                    <CircleHelp className="mr-1.5 h-4 w-4" />
                    Info
                  </>
                )}
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
                onClick={() => setShowActionsSheet(true)}
                aria-label="More actions"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </div>
            {mobileView === "info" ? mobileInfoView : fileWorkspace}
          </div>

          <div className="hidden space-y-6 lg:block">
            <Card data-testid={selectors.backlogDetails.header}>
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={`inline-block h-3 w-3 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                  />
                  <span className="text-xs uppercase tracking-wider text-slate-500 sm:text-sm">
                    {formatBacklogStatus(item.status)}
                  </span>
                  <span className="rounded-full bg-slate-700 px-3 py-1 text-xs text-slate-300 sm:text-sm">
                    Priority {item.priority}
                  </span>
                  {item.kind === "research" && (
                    <span className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-300">
                      Target: {item.researchTarget ? BACKLOG_RESEARCH_TARGET_LABELS[item.researchTarget] : "Unspecified"}
                    </span>
                  )}
                </div>
                <h1
                  className="text-xl font-bold text-slate-100 sm:text-2xl"
                  data-testid={selectors.backlogDetails.title}
                >
                  {item.title}
                </h1>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                {canQueue && (
                  <>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => queueMutation.mutate({ mode: "manual" })}
                      disabled={queueMutation.isPending}
                      data-testid={selectors.backlogDetails.queueButton}
                    >
                      {queueMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <HeaderIcon className="mr-2 h-4 w-4" />}
                      Queue
                    </Button>
                    <Button
                      variant="default"
                      size="sm"
                      onClick={() => queueMutation.mutate({ mode: "yolo" })}
                      disabled={queueMutation.isPending}
                    >
                      {queueMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Play className="mr-2 h-4 w-4" />}
                      Start Now
                    </Button>
                    <Input
                      type="number"
                      min={0}
                      step={1}
                      value={scheduleDelayValue}
                      onChange={(event) => setScheduleDelaySeconds(Number(event.target.value || 0))}
                      className="h-9 w-24"
                      aria-label="Schedule delay seconds"
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => queueMutation.mutate({ mode: "scheduled", delaySeconds: scheduleDelayValue })}
                      disabled={queueMutation.isPending}
                    >
                      Schedule
                    </Button>
                  </>
                )}
                {!canQueue && queueBlockedReason ? (
                  <span className="max-w-xs text-xs text-slate-500">{queueBlockedReason}</span>
                ) : null}
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
                      : `Convert to ${BACKLOG_KIND_LABELS[convertTarget]}`}
                  </Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  className="hidden lg:inline-flex"
                  data-testid={selectors.backlogDetails.editButton}
                  onClick={() => setShowEdit(true)}
                >
                  <Edit className="mr-2 h-4 w-4" />
                  Edit
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="hidden lg:inline-flex"
                  data-testid={selectors.backlogDetails.deleteButton}
                  onClick={() => setShowDelete(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="hidden lg:inline-flex"
                  onClick={() => setShowAgentDialog(true)}
                  data-testid={selectors.backlogDetails.agentButton}
                >
                  <Sparkles className="mr-2 h-4 w-4" />
                  {agentLabel}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70 lg:hidden"
                  onClick={() => setShowActionsSheet(true)}
                  aria-label="More actions"
                >
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </div>
            </div>

            {(queueError || queueBlockedResult || deleteError || convertError) && (
              <div className="mt-4 space-y-2">
                {queueBlockedResult && (
                  <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-200">
                    <div className="flex items-center gap-2 font-medium">
                      <AlertTriangle className="h-4 w-4 shrink-0 text-amber-400" />
                      {queueBlockedResult.message}
                    </div>
                    {queueBlockedResult.blockingReasons.length > 0 && (
                      <ul className="mt-1.5 list-disc space-y-0.5 pl-6 text-amber-300/90">
                        {queueBlockedResult.blockingReasons.map((reason, i) => (
                          <li key={i}>{reason}</li>
                        ))}
                      </ul>
                    )}
                    {queueBlockedResult.unansweredQuestions > 0 && (
                      <p className="mt-1 text-xs text-amber-300/70">
                        {queueBlockedResult.unansweredQuestions} unanswered question{queueBlockedResult.unansweredQuestions !== 1 ? "s" : ""}
                      </p>
                    )}
                  </div>
                )}
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
            {latestAgentRun && (
              <div className="mt-4 rounded-lg border border-cyan-500/30 bg-cyan-500/10 px-3 py-2">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-sm font-semibold text-slate-100">Last Agent Run</p>
                  <span className="rounded-full bg-slate-900/70 px-2 py-0.5 text-xs text-cyan-200">
                    {latestAgentRun.status.replace("_", " ")}
                  </span>
                </div>
                <p className="mt-1 font-mono text-xs text-cyan-300">{latestAgentRun.runId}</p>
                <p className="text-xs text-slate-300">Spawned {formatRelativeTime(latestAgentRun.createdAt)}</p>
                <p className="text-xs text-slate-300">Duration {formatDuration(latestAgentRun.durationSeconds)}</p>
                {latestAgentRun.errorMessage ? (
                  <p className="mt-2 rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200">
                    {latestAgentRun.errorMessage}
                  </p>
                ) : null}
                <div className="mt-2 flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => void refreshRun(latestAgentRun.runId)}
                  >
                    Refresh
                  </Button>
                  {latestAgentRun.active && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => void stopRun(latestAgentRun.runId)}
                      disabled={latestAgentRun.isStopping}
                    >
                      <Square className="mr-2 h-3.5 w-3.5" />
                      {latestAgentRun.isStopping ? "Stopping..." : "Stop"}
                    </Button>
                  )}
                </div>
              </div>
            )}
          </Card>

            {detailsPanel}
            {notesPanel}
            {archiveTargets?.has_archive && (
              <OperationalTargetsPanel
                targets={archiveTargets.targets}
                requirements={archiveTargets.requirements}
                editable
                onCreateRequirement={handleCreateRequirement}
                onEditRequirement={handleEditRequirement}
                onDeleteRequirement={handleDeleteRequirement}
                onReorderRequirement={handleReorderRequirement}
                onCreateModule={handleCreateModule}
                onEditModule={handleEditModule}
                onDeleteModule={handleDeleteModule}
                onCreateTarget={handleCreateTarget}
                onEditTarget={handleEditTarget}
                onDeleteTarget={handleDeleteTarget}
              />
            )}
            {fileWorkspace}
          </div>

          <BottomSheet
            isOpen={showActionsSheet}
            onClose={() => setShowActionsSheet(false)}
            title="Actions"
            description="Quick actions for this backlog item"
          >
            {renderActionButtons(true)}
          </BottomSheet>
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

      <Dialog
        isOpen={Boolean(activeFileAction && activeFileAction.action !== "delete")}
        onClose={() => {
          setActiveFileAction(null);
          setFileActionError(null);
        }}
        title={
          activeFileAction?.action === "rename"
            ? `Rename ${activeFileAction.target.type}`
            : activeFileAction?.action === "move"
              ? `Move ${activeFileAction.target.type}`
              : activeFileAction?.action === "copy"
                ? `Copy ${activeFileAction.target.type}`
                : "File Action"
        }
        maxWidth="max-w-md"
      >
        {activeFileAction && activeFileAction.action !== "delete" && (
          <div className="space-y-4">
            <div className="text-sm text-slate-300">
              <p className="text-xs uppercase tracking-wide text-slate-500">Source</p>
              <p className="mt-1 break-all rounded-lg bg-slate-800/60 px-3 py-2">{activeFileAction.target.path}</p>
            </div>
            <div className="space-y-2">
              <label className="text-xs uppercase tracking-wide text-slate-500">
                {activeFileAction.action === "rename" ? "New name" : "Destination path"}
              </label>
              <Input
                value={fileActionInput}
                onChange={(event) => setFileActionInput(event.target.value)}
                placeholder={activeFileAction.action === "rename" ? "new-name.ext" : "path/to/target"}
              />
            </div>
            {fileActionError && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                {fileActionError}
              </div>
            )}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setActiveFileAction(null);
                  setFileActionError(null);
                }}
                disabled={fileActionMutation.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={handleFileActionConfirm}
                disabled={fileActionMutation.isPending}
                data-testid="confirm-file-action"
              >
                {fileActionMutation.isPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : null}
                Apply
              </Button>
            </div>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        isOpen={Boolean(activeFileAction && activeFileAction.action === "delete")}
        onClose={() => {
          setActiveFileAction(null);
          setFileActionError(null);
        }}
        onConfirm={handleFileActionConfirm}
        title={`Delete ${activeFileAction?.target.type ?? "file"}`}
        description={`Delete "${activeFileAction?.target.path ?? ""}" from this backlog item? This cannot be undone.`}
        confirmLabel="Delete"
        isLoading={fileActionMutation.isPending}
      />

      <BacklogAgentDialog
        isOpen={showAgentDialog}
        isSubmitting={agentMutation.isPending}
        backlogKind={backlogKind}
        backlogTitle={item?.title ?? name ?? ""}
        researchTarget={item?.researchTarget}
        errorMessage={agentError}
        files={files}
        archiveTargets={archiveTargets}
        initialSelectedTargetIds={selectedTargetIds}
        initialSelectedRequirementIds={selectedRequirementIds}
        onClose={() => {
          setShowAgentDialog(false);
          agentMutation.reset();
        }}
        onSubmit={(payload) => agentMutation.mutate(payload)}
      />

      <RequirementFormDialog
        isOpen={reqDialogOpen}
        mode={reqDialogMode}
        initialValues={editingReq?.req}
        isSubmitting={updateReqsMutation.isPending}
        submitError={updateReqsMutation.error instanceof Error ? updateReqsMutation.error.message : null}
        onClose={() => { setReqDialogOpen(false); setEditingReq(null); updateReqsMutation.reset(); }}
        onSubmit={handleReqDialogSubmit}
      />

      <ModuleFormDialog
        isOpen={moduleDialogOpen}
        mode={moduleDialogMode}
        initialValues={
          editingModuleId && archiveTargets
            ? (() => {
                const findGroup = (groups: typeof archiveTargets.requirements): typeof archiveTargets.requirements[0] | undefined => {
                  for (const g of groups) {
                    if (g.id === editingModuleId) return g;
                    const found = findGroup(g.children);
                    if (found) return found;
                  }
                  return undefined;
                };
                const g = findGroup(archiveTargets.requirements);
                return g ? { id: g.id, title: g.name, description: "" } : undefined;
              })()
            : undefined
        }
        isSubmitting={createModuleMutation.isPending || updateModuleMetaMutation.isPending}
        submitError={
          (createModuleMutation.error instanceof Error ? createModuleMutation.error.message : null)
          ?? (updateModuleMetaMutation.error instanceof Error ? updateModuleMetaMutation.error.message : null)
        }
        onClose={() => { setModuleDialogOpen(false); setEditingModuleId(null); createModuleMutation.reset(); updateModuleMetaMutation.reset(); }}
        onSubmit={handleModuleDialogSubmit}
      />

      <TargetFormDialog
        isOpen={targetDialogOpen}
        mode={targetDialogMode}
        initialValues={editingTarget ?? undefined}
        isSubmitting={createTargetMutation.isPending || updateTargetMutation.isPending}
        submitError={
          (createTargetMutation.error instanceof Error ? createTargetMutation.error.message : null)
          ?? (updateTargetMutation.error instanceof Error ? updateTargetMutation.error.message : null)
        }
        onClose={() => { setTargetDialogOpen(false); setEditingTarget(null); createTargetMutation.reset(); updateTargetMutation.reset(); }}
        onSubmit={handleTargetDialogSubmit}
      />

      <QuestionFormDialog
        isOpen={questionDialogOpen}
        mode={questionDialogMode}
        initialValues={editingQuestion ? {
          question: editingQuestion.question,
          options: editingQuestion.options,
          answer: editingQuestion.answer,
        } : undefined}
        isSubmitting={questionSubmitting}
        submitError={questionSubmitError}
        onClose={() => { setQuestionDialogOpen(false); setEditingQuestion(null); setQuestionSubmitError(null); }}
        onSubmit={handleQuestionDialogSubmit}
      />

      <SuggestionFormDialog
        isOpen={suggestionDialogOpen}
        mode={suggestionDialogMode}
        initialValues={editingSuggestion ? {
          suggestion: editingSuggestion.suggestion,
          details: editingSuggestion.details,
          status: editingSuggestion.status,
        } : undefined}
        isSubmitting={suggestionSubmitting}
        submitError={suggestionSubmitError}
        onClose={() => { setSuggestionDialogOpen(false); setEditingSuggestion(null); setSuggestionSubmitError(null); }}
        onSubmit={handleSuggestionDialogSubmit}
      />
    </div>
  );
}
