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
  ArrowRight,
  ArrowRightLeft,
  ArrowUpRight,
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
import { Select } from "../components/ui/select";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { FileTree, type TreeFile } from "../components/ui/file-tree";
import { FilePreview } from "../components/ui/file-preview";
import { FileUpload } from "../components/ui/file-upload";
import { TagList } from "../components/ui/tag-list";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { BacklogAgentDialog } from "../components/backlog/backlog-agent-dialog";
import { WorkshopPanel } from "../components/backlog/workshop-panel";
import { ReadinessDetailsPanel } from "../components/backlog/readiness-details-panel";
import { OperationalTargetsPanel } from "../components/backlog/operational-targets-panel";
import { RequirementFormDialog } from "../components/backlog/requirement-form-dialog";
import { TargetFormDialog } from "../components/backlog/target-form-dialog";
import { ModuleFormDialog } from "../components/backlog/module-form-dialog";
import { RunBacklogModal } from "../components/backlog/run-backlog-modal";
import {
  cn,
  defaultQueryOptions,
  formatRelativeTime,
  getBacklogNotQueueableReason,
  isBacklogQueueable,
} from "../lib";
import { parseWorkshopRound, WORKSHOP_FILE_PATHS, findBacklogFileByPath } from "../lib/workshop-files";
import { buildReadinessData } from "../lib/maturity";
import type { ReadinessIndicatorData } from "../lib/maturity";
import { backlogService, executionService } from "../services";
import { selectors } from "../consts/selectors";
import {
  BACKLOG_KIND_LABELS,
  BACKLOG_KINDS,
  BACKLOG_RESEARCH_TARGET_LABELS,
  BACKLOG_STATUS_COLORS,
  EXECUTION_STATUS_COLORS,
  formatBacklogStatus,
  formatExecutionStatus,
} from "../types";
import type {
  ArchiveRequirement,
  ArchiveRequirementRecord,
  ArchiveTarget,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogResearchTarget,
  BacklogStatus,
  ExecutionRecord,
  ExecutionStatus,
  ModuleFormValues,
  ResearchResponse,
} from "../types";
import type { WorkshopRound } from "../types/domain";
import { selectLatestRunForBacklog, useAgentRunsStore, useBacklogStore } from "../stores";

/** Statuses that users can manually set via the status dropdown. "queued" and "in_progress" are execution-system-only. */
const USER_SETTABLE_STATUSES: BacklogStatus[] = ["backlog", "researching", "ready", "failed", "completed", "archived"];

const RECENT_FILES_LIMIT = 5;
const DEFAULT_PREVIEW_FILE_PATH = "spec.json";
/** How often to poll agent-manager for active run status updates (ms). */
const AGENT_RUN_REFRESH_MS = 6000;
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

  // Sync the agentRunsStore with the execution service's canonical run ID.
  //
  // The execution service is the single source of truth for which run is
  // associated with a backlog item. When an execution is retried, the
  // execution record gets a new RunID, but the agentRunsStore (localStorage)
  // still points to the old run. This effect detects the mismatch and
  // updates the store so the active run banner shows the correct run.
  useEffect(() => {
    if (!backlogKind || !name) return;
    let cancelled = false;
    const sync = async () => {
      try {
        const executions = await executionService.list({
          backlogKind: backlogKind as BacklogKind,
          backlogName: name,
        });
        if (cancelled || executions.length === 0) return;
        // Find the most recent execution with a run ID.
        const latest = executions
          .filter((e) => e.runId)
          .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())[0];
        if (!latest?.runId) return;
        const currentStoreRun = useAgentRunsStore.getState().runs.find(
          (r) => r.backlogKind === backlogKind && r.backlogName === name,
        );
        // If the execution service has a different (newer) run ID, update the store.
        if (!currentStoreRun || currentStoreRun.runId !== latest.runId) {
          upsertSpawnedRun({
            runId: latest.runId,
            taskId: latest.taskId ?? "",
            baseUrl: currentStoreRun?.baseUrl ?? "",
            createdAt: latest.createdAt,
            backlogKind: backlogKind as BacklogKind,
            backlogName: name,
          });
          // Immediately refresh the new run to get its current status.
          void refreshRun(latest.runId);
        }
      } catch {
        // Execution service unavailable — keep existing store data.
      }
    };
    void sync();
    return () => { cancelled = true; };
  }, [backlogKind, name, upsertSpawnedRun, refreshRun]);

  // Auto-refresh the latest agent run while it's in an active status.
  // Without this, the BacklogDetailsPage shows stale localStorage data
  // while the ExecutionPage (which polls independently) shows the real status.
  const agentRunId = latestAgentRun?.runId;
  const agentRunIsActive = latestAgentRun
    ? ["pending", "starting", "running", "needs_review"].includes(latestAgentRun.status)
    : false;
  useEffect(() => {
    if (!agentRunId || !agentRunIsActive) return;
    // Refresh immediately on mount/when run becomes active.
    void refreshRun(agentRunId);
    const interval = window.setInterval(() => {
      void refreshRun(agentRunId);
    }, AGENT_RUN_REFRESH_MS);
    return () => window.clearInterval(interval);
  }, [agentRunId, agentRunIsActive, refreshRun]);

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
  const [showRunModal, setShowRunModal] = useState(false);
  const [previewResetKey, setPreviewResetKey] = useState(0);
  const [detailsExpanded, setDetailsExpanded] = useState(true);
  const [execSectionOpen, setExecSectionOpen] = useState(false);
  const [expandedExecIds, setExpandedExecIds] = useState<Set<string>>(new Set());
  const [selectedTargetIds, setSelectedTargetIds] = useState<Set<string>>(new Set());
  const [selectedRequirementIds, setSelectedRequirementIds] = useState<Set<string>>(new Set());
  const [reqDialogOpen, setReqDialogOpen] = useState(false);
  const [reqDialogMode, setReqDialogMode] = useState<"create" | "edit">("create");
  const [editingReq, setEditingReq] = useState<{ groupId: string; req?: ArchiveRequirementRecord } | null>(null);
  const [moduleDialogOpen, setModuleDialogOpen] = useState(false);
  const [moduleDialogMode, setModuleDialogMode] = useState<"create" | "edit">("create");
  const [editingModuleId, setEditingModuleId] = useState<string | null>(null);
  const [targetDialogOpen, setTargetDialogOpen] = useState(false);
  const [targetDialogMode, setTargetDialogMode] = useState<"create" | "edit">("create");
  const [editingTarget, setEditingTarget] = useState<ArchiveTarget | null>(null);

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

  // Fetch execution history for this backlog item.
  const { data: executionHistory } = useQuery({
    queryKey: ["executions", backlogKind, name],
    queryFn: () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return executionService.list({ backlogKind: backlogKind as BacklogKind, backlogName: name });
    },
    enabled: !!backlogKind && !!name,
    refetchInterval: 10_000,
  });

  // Workshop round files — find workshop/ directory, then load each round file
  const workshopDir = useMemo(
    () => findBacklogFileByPath(files ?? [], WORKSHOP_FILE_PATHS.workshopDir.replace(/\/$/, "")),
    [files],
  );
  const workshopRoundPaths = useMemo(() => {
    if (!workshopDir?.children) return [];
    return workshopDir.children
      .filter((f) => f.type === "file" && /^round-\d+\.json$/.test(f.name))
      .sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true }))
      .map((f) => f.path);
  }, [workshopDir]);

  const {
    data: workshopRoundContents,
    refetch: refetchWorkshopRounds,
  } = useQuery({
    queryKey: ["backlog", backlogKind, name, "workshop-rounds", workshopRoundPaths],
    queryFn: async () => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const contents = await Promise.all(
        workshopRoundPaths.map((p) => backlogService.getFileContent(backlogKind, name, p)),
      );
      return contents;
    },
    enabled: !!backlogKind && !!name && workshopRoundPaths.length > 0,
    ...defaultQueryOptions,
  });

  const workshopRounds = useMemo(() => {
    if (!workshopRoundContents) return [];
    return workshopRoundContents
      .map((content) => parseWorkshopRound(content))
      .filter((r): r is { round: WorkshopRound; error?: string } => r.round !== null)
      .map((r) => r.round!);
  }, [workshopRoundContents]);

  // Maturity / readiness data from the maturity-summary endpoint
  const { data: maturitySummaryData } = useQuery({
    queryKey: ["backlog-maturity-summary"],
    queryFn: () => backlogService.getMaturitySummary(),
    ...defaultQueryOptions,
  });

  const readinessData = useMemo<ReadinessIndicatorData | null>(() => {
    if (!maturitySummaryData || !backlogKind || !name) return null;
    const match = maturitySummaryData.items.find(
      (i) => i.kind === backlogKind && i.name === name,
    );
    return match ? buildReadinessData(match) : null;
  }, [maturitySummaryData, backlogKind, name]);

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

  const LOCKED_STATUSES = new Set(["queued", "in_progress"]);
  const isLocked = Boolean(item && LOCKED_STATUSES.has(item.status));

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
      mode?: string;
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

  // Workshop save mutation — saves user answers/decisions back to a round file
  const workshopSaveMutation = useMutation({
    mutationFn: async ({ roundNumber, content }: { roundNumber: number; content: string }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      const filePath = `workshop/round-${roundNumber}.json`;
      await backlogService.saveFileContent(backlogKind, name, filePath, content, "application/json");
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "workshop-rounds"] });
      void refetchFiles();
      void refetchWorkshopRounds();
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
    const tmp = reqs[idx] as typeof reqs[number];
    reqs[idx] = reqs[swapIdx] as typeof reqs[number];
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

  // --- Workshop handlers ---
  const handleSaveRound = useCallback((roundNumber: number, content: string) => {
    workshopSaveMutation.mutate({ roundNumber, content });
  }, [workshopSaveMutation]);

  const handleRunWorkshop = useCallback(() => {
    if (!backlogKind || !name) return;
    agentMutation.mutate({
      mode: "workshop",
      prompt: "Run the next workshop round for this backlog item.",
    });
  }, [backlogKind, name, agentMutation]);

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
  const workshopSaveError = workshopSaveMutation.isError
    ? workshopSaveMutation.error instanceof Error ? workshopSaveMutation.error.message : "Failed to save workshop round."
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

  const agentLabel = item?.kind === "idea" ? "Idea Agent" : item?.kind === "research" ? "Research Agent" : "Research";
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
          disabled={isLocked}
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

  const notesPanel = (
    <div className="space-y-4">
      {readinessData && <ReadinessDetailsPanel data={readinessData} />}
      {isLocked && (
        <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-2 text-sm text-amber-300">
          This item is {item ? formatBacklogStatus(item.status) : "locked"} and cannot be edited.
        </div>
      )}
      <WorkshopPanel
        rounds={workshopRounds}
        backlogKind={backlogKind as BacklogKind}
        backlogName={name ?? ""}
        disabled={isLocked}
        isSaving={workshopSaveMutation.isPending}
        isRunningWorkshop={agentMutation.isPending}
        onSaveRound={handleSaveRound}
        onRunWorkshop={handleRunWorkshop}
      />
    </div>
  );

  const toggleExecExpand = useCallback((id: string) => {
    setExpandedExecIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const activeRunBanner = agentRunIsActive && latestAgentRun ? (
    <div
      className="flex items-center gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/10 px-3 py-1.5"
      data-testid={selectors.backlogDetails.activeRunBanner}
    >
      <span className="relative flex h-2 w-2 shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
      </span>
      <span className="text-xs font-medium capitalize text-cyan-200">
        {latestAgentRun.status.replace("_", " ")}
      </span>
      <span className="text-xs text-slate-400">
        {formatRelativeTime(latestAgentRun.createdAt)}
      </span>
      <div className="ml-auto flex items-center gap-1.5">
        <Button
          variant="outline"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => void stopRun(latestAgentRun.runId)}
          disabled={latestAgentRun.isStopping}
        >
          <Square className="mr-1 h-3 w-3" />
          {latestAgentRun.isStopping ? "Stopping..." : "Stop"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={() => navigate(`/execution?backlog=${encodeURIComponent(`${backlogKind}/${name}`)}`)}
          aria-label="View execution"
        >
          <ArrowUpRight className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  ) : null;

  const executionHistorySection = executionHistory && executionHistory.length > 0 ? (
    <Card padding="sm" data-testid={selectors.backlogDetails.executionHistory}>
      <button
        type="button"
        onClick={() => setExecSectionOpen(!execSectionOpen)}
        className="flex w-full items-center gap-2 text-left"
      >
        {execSectionOpen ? (
          <ChevronDown className="h-4 w-4 text-slate-400" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-400" />
        )}
        <span className="flex-1 text-sm font-semibold text-slate-100">
          Execution History
        </span>
        {!execSectionOpen && executionHistory[0] && (
          <span className="flex items-center gap-1.5 text-xs text-slate-500">
            <span
              className={`inline-block h-1.5 w-1.5 rounded-full ${EXECUTION_STATUS_COLORS[executionHistory[0].status as ExecutionStatus] ?? "bg-slate-500"}`}
            />
            {formatExecutionStatus(executionHistory[0].status as ExecutionStatus)}
            {executionHistory[0].operation && (
              <span className="rounded bg-slate-700/60 px-1 py-0.5 text-[10px]">
                {executionHistory[0].operation}
              </span>
            )}
          </span>
        )}
        <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-400">
          {executionHistory.length}
        </span>
      </button>
      {execSectionOpen && (
        <div className="mt-2 space-y-1.5">
          {executionHistory.slice(0, 5).map((exec: ExecutionRecord) => {
            const isExpanded = expandedExecIds.has(exec.executionId);
            const isActiveExecRun = !!(exec.runId && latestAgentRun && exec.runId === latestAgentRun.runId && agentRunIsActive);
            const duration = exec.startedAt && exec.finishedAt
              ? (new Date(exec.finishedAt).getTime() - new Date(exec.startedAt).getTime()) / 1000
              : undefined;
            return (
              <div key={exec.executionId} className="rounded-md bg-slate-800/40">
                <button
                  type="button"
                  onClick={() => toggleExecExpand(exec.executionId)}
                  className="flex w-full items-center gap-2 px-2.5 py-1.5 text-left"
                >
                  {isExpanded ? (
                    <ChevronDown className="h-3 w-3 shrink-0 text-slate-500" />
                  ) : (
                    <ChevronRight className="h-3 w-3 shrink-0 text-slate-500" />
                  )}
                  <span
                    className={`inline-block h-2 w-2 shrink-0 rounded-full ${EXECUTION_STATUS_COLORS[exec.status as ExecutionStatus] ?? "bg-slate-500"}`}
                  />
                  <span className="text-xs font-medium text-slate-200">
                    {formatExecutionStatus(exec.status as ExecutionStatus)}
                  </span>
                  {exec.operation && (
                    <span className="rounded bg-slate-700/60 px-1 py-0.5 text-[10px] text-slate-400">
                      {exec.operation}
                    </span>
                  )}
                  <span className="ml-auto text-[10px] text-slate-500">
                    {formatRelativeTime(exec.createdAt)}
                  </span>
                </button>
                {isExpanded && (
                  <div className="space-y-2 border-t border-slate-700/40 px-2.5 py-2">
                    {exec.failureReason && (
                      <p className="rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200">
                        {exec.failureReason}
                      </p>
                    )}
                    <div className="space-y-1 text-xs text-slate-400">
                      {duration !== undefined && (
                        <p>Duration: {formatDuration(duration)}</p>
                      )}
                      {exec.startedAt && (
                        <p>Started: {formatRelativeTime(exec.startedAt)}</p>
                      )}
                      {exec.finishedAt && (
                        <p>Finished: {formatRelativeTime(exec.finishedAt)}</p>
                      )}
                      <p className="font-mono text-[11px] text-slate-500">
                        ID: {exec.executionId}
                      </p>
                      {exec.runId && (
                        <p className="font-mono text-[11px] text-slate-500">
                          Run: {exec.runId}
                        </p>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 px-2 text-xs"
                        onClick={() => navigate(`/execution?backlog=${encodeURIComponent(`${backlogKind}/${name}`)}`)}
                      >
                        <ArrowUpRight className="mr-1 h-3 w-3" />
                        View
                      </Button>
                      {isActiveExecRun && (
                        <Button
                          variant="outline"
                          size="sm"
                          className="h-7 px-2 text-xs"
                          onClick={() => void stopRun(latestAgentRun.runId)}
                          disabled={latestAgentRun.isStopping}
                        >
                          <Square className="mr-1 h-3 w-3" />
                          {latestAgentRun.isStopping ? "Stopping..." : "Stop"}
                        </Button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
          {executionHistory.length > 5 && (
            <Button
              variant="outline"
              size="sm"
              className="w-full border-transparent text-xs text-slate-400 hover:text-slate-200"
              onClick={() => navigate(`/execution?backlog=${encodeURIComponent(`${backlogKind}/${name}`)}`)}
            >
              View all {executionHistory.length} executions
            </Button>
          )}
        </div>
      )}
    </Card>
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
          <Button
            variant="default"
            size="sm"
            className={primaryRowButtonClass}
            onClick={() => runAction(() => setShowRunModal(true))}
          >
            <Play className="mr-2 h-4 w-4" />
            Run
          </Button>
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
          disabled={isLocked}
        >
          <Sparkles className="mr-2 h-4 w-4" />
          {agentLabel}
        </Button>
        {!isLocked && item && (
          <div className="space-y-1">
            <label htmlFor="action-status-select" className="text-xs text-slate-400">
              Status
            </label>
            <Select
              id="action-status-select"
              variant="filter"
              withChevron
              value={item.status}
              onChange={(e) => {
                const newStatus = e.target.value as BacklogStatus;
                if (newStatus !== item.status) {
                  updateMutation.mutate({
                    title: item.title,
                    description: item.description,
                    status: newStatus,
                    priority: item.priority,
                    tags: item.tags,
                    researchTarget: item.researchTarget,
                  });
                  if (closeOnAction) setShowActionsSheet(false);
                }
              }}
              data-testid={selectors.backlogDetails.statusSelect}
            >
              {USER_SETTABLE_STATUSES.map((s) => (
                <option key={s} value={s}>
                  {formatBacklogStatus(s)}
                </option>
              ))}
            </Select>
          </div>
        )}
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
      {(deleteError || convertError) && (
        <Card padding="sm" className="space-y-2 rounded-lg border-slate-700/60 bg-slate-900/45">
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
      {activeRunBanner}
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
      {executionHistorySection}
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
                  {isLocked ? (
                    <>
                      <span
                        className={`inline-block h-3 w-3 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                      />
                      <span className="text-xs uppercase tracking-wider text-slate-500 sm:text-sm">
                        {formatBacklogStatus(item.status)}
                      </span>
                    </>
                  ) : (
                    <div className="flex items-center gap-1.5">
                      <span
                        className={`inline-block h-2.5 w-2.5 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                      />
                      <Select
                        variant="compact"
                        withChevron
                        value={item.status}
                        onChange={(e) => {
                          const newStatus = e.target.value as BacklogStatus;
                          if (newStatus !== item.status) {
                            updateMutation.mutate({
                              title: item.title,
                              description: item.description,
                              status: newStatus,
                              priority: item.priority,
                              tags: item.tags,
                              researchTarget: item.researchTarget,
                            });
                          }
                        }}
                        data-testid={selectors.backlogDetails.statusSelect}
                        className="w-auto uppercase tracking-wider"
                      >
                        {USER_SETTABLE_STATUSES.map((s) => (
                          <option key={s} value={s}>
                            {formatBacklogStatus(s)}
                          </option>
                        ))}
                      </Select>
                    </div>
                  )}
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
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => setShowRunModal(true)}
                    data-testid={selectors.backlogDetails.queueButton}
                  >
                    <Play className="mr-2 h-4 w-4" />
                    Run
                  </Button>
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
                  disabled={isLocked}
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

            {(deleteError || convertError) && (
              <div className="mt-4 space-y-2">
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
          </Card>

            {activeRunBanner}
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
            {executionHistorySection}
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

      <RunBacklogModal
        isOpen={showRunModal}
        onClose={() => setShowRunModal(false)}
        target={backlogKind && name ? { kind: backlogKind, name, title: item?.title } : undefined}
        readinessData={readinessData}
        onSuccess={(result) => {
          if (result.item) upsertItem(result.item);
          if (backlogKind && name) {
            queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
          }
          setShowRunModal(false);
        }}
      />

      <BacklogAgentDialog
        isOpen={showAgentDialog}
        isSubmitting={agentMutation.isPending}
        backlogKind={backlogKind}
        backlogTitle={item?.title ?? name ?? ""}
        itemStatus={item?.status}
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

    </div>
  );
}
