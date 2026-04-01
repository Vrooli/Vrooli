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
import { Link, useSearchParams } from "react-router-dom";
import {
  Archive,
  ArrowRightLeft,
  ArrowUpRight,
  CheckCircle2,
  CircleHelp,
  Copy,
  Edit,
  FileText,
  Files,
  FolderOpen,
  GitBranch,
  Info,
  Lock,
  Loader2,
  MessageSquare,
  MessageSquareText,
  MoreHorizontal,
  Play,
  RefreshCw,
  Search,
  Sparkles,
  History,
  Square,
  Tags,
  Target,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { Tabs, TabsList, TabsTrigger } from "../components/ui/tabs";
import { PlanPanel } from "../components/backlog/plan-panel";
import { useUrlState } from "../hooks/use-url-state";
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
import { AcceptanceGlobDialog } from "../components/backlog/acceptance-glob-dialog";
import { BacklogAgentDialog } from "../components/backlog/backlog-agent-dialog";
import { WorkshopPanel } from "../components/backlog/workshop-panel";
import { ClarificationPanel } from "../components/backlog/clarification-panel";
import { ReadinessDetailsPanel } from "../components/backlog/readiness-details-panel";
import { FollowUpDialog } from "../components/execution/follow-up-dialog";
import { PostRunStatusBadge } from "../components/execution/post-run-status-badge";
import { Drawer } from "../components/ui/drawer";
import { ActivityTimeline } from "../components/detail/ActivityTimeline";
import { useActivityTimeline } from "../hooks/useActivityTimeline";
import { OperationalTargetsPanel } from "../components/backlog/operational-targets-panel";
import { BulkActionToolbar } from "../components/backlog/bulk-action-toolbar";
import { RequirementFormDialog } from "../components/backlog/requirement-form-dialog";
import { TargetFormDialog } from "../components/backlog/target-form-dialog";
import { ModuleFormDialog } from "../components/backlog/module-form-dialog";
import { RunBacklogModal } from "../components/backlog/run-backlog-modal";
import {
  cn,
  defaultQueryOptions,
  formatRelativeTime,
  getItemActions,
  scenariosFromGlobs,
} from "../lib";
import type { ItemActions } from "../lib/backlog-queue-utils";
import { computeDependencyRelations } from "../lib/backlog-queue-utils";
import { DependencyChipList } from "../components/backlog/dependency-chip-list";
import { parseWorkshopRound, WORKSHOP_FILE_PATHS, findBacklogFileByPath } from "../lib/workshop-files";
import { buildReadinessData } from "../lib/maturity";
import type { ReadinessIndicatorData } from "../lib/maturity";
import { backlogService, executionService } from "../services";
import { selectors } from "../consts/selectors";
import {
  BACKLOG_KIND_ICONS,
  BACKLOG_KIND_LABELS,
  BACKLOG_KINDS,
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
} from "../types";
import type {
  ArchiveRequirement,
  ArchiveRequirementRecord,
  ArchiveTarget,
  ArchiveTargetFormValues,
  BacklogFile,
  BacklogKind,
  BacklogStatus,
  ExecutionRecord,
  ModuleFormValues,
  ResearchResponse,
  ReviewAction,
  ReviewUpdate,
} from "../types";
import type { WorkshopRound } from "../types/domain";
import { selectLatestActivityForBacklog, useAgentActivitiesStore, useBacklogStore, useDetailSelectionStore } from "../stores";
import { BACKLOG_LENSES } from "../components/detail/lens-options";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { useDetailNavigation } from "../hooks/useDetailNavigation";
import { selectionToNodeId } from "../stores/detail-selection-store";


const RECENT_FILES_LIMIT = 5;
const DEFAULT_PREVIEW_FILE_PATH = "spec.json";
/** How often to poll agent-manager for active run status updates (ms). */
const AGENT_RUN_REFRESH_MS = 6000;
const MIN_FILES_PANEL_WIDTH = 240;
const MAX_FILES_PANEL_WIDTH = 520;
const MIN_PREVIEW_WIDTH = 320;
const RESIZE_HANDLE_WIDTH = 8;
type DetailsTab = "info" | "prompt" | "files";
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
  const selection = useDetailSelectionStore((s) => s.selection);
  const selectInitiative = useDetailSelectionStore((s) => s.selectInitiative);
  const selectScenario = useDetailSelectionStore((s) => s.selectScenario);
  const nodeId = selectionToNodeId(selection);
  const { closeDetail } = useDetailNavigation();
  const kind = selection?.kind;
  const name = selection?.name;
  const backlogKind = BACKLOG_KINDS.includes(kind as BacklogKind) ? (kind as BacklogKind) : null;
  const [searchParams, setSearchParams] = useSearchParams();
  const queryClient = useQueryClient();
  const upsertItem = useBacklogStore((state) => state.upsertItem);
  const allBacklogItems = useBacklogStore((state) => state.items);
  const removeItem = useBacklogStore((state) => state.removeItem);
  const cachedItem = useMemo(
    () => allBacklogItems.find((i) => i.kind === backlogKind && i.name === name),
    [allBacklogItems, backlogKind, name],
  );
  const refreshActivities = useAgentActivitiesStore((state) => state.refreshActivities);
  const stopRun = useAgentActivitiesStore((state) => state.stopRun);
  const latestAgentActivity = useAgentActivitiesStore((state) => {
    if (!backlogKind || !name) return null;
    return selectLatestActivityForBacklog(state, backlogKind, name);
  });
  useEffect(() => {
    void refreshActivities(true);
    const interval = window.setInterval(() => {
      void refreshActivities(true);
    }, AGENT_RUN_REFRESH_MS);
    return () => window.clearInterval(interval);
  }, [refreshActivities]);

  const agentRunIsActive = latestAgentActivity
    ? ["pending", "starting", "running", "needs_review"].includes(latestAgentActivity.status)
    : false;

  const workspaceRef = useRef<HTMLDivElement | null>(null);
  const headerFileActionsRef = useRef<HTMLDivElement | null>(null);
  const [filesPanelWidth, setFilesPanelWidth] = useState(320);
  const [isResizing, setIsResizing] = useState(false);
  const [activeTab, setActiveTab] = useUrlState<DetailsTab>("tab", "info", {
    validate: (v): v is DetailsTab => ["info", "prompt", "files"].includes(v),
  });
  const [showFilesSheet, setShowFilesSheet] = useState(false);
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
  const [showForceWorkshopConfirm, setShowForceWorkshopConfirm] = useState(false);
  const [previewResetKey, setPreviewResetKey] = useState(0);
  const [descExpanded, setDescExpanded] = useState(false);
  const [descOverflows, setDescOverflows] = useState(false);
  const [allowExpanded, setAllowExpanded] = useState(false);
  const [denyExpanded, setDenyExpanded] = useState(false);
  const [showGlobDialog, setShowGlobDialog] = useState(false);

  const [isTimelineOpen, setIsTimelineOpen] = useState(false);
  const [agentManagerUiUrl, setAgentManagerUiUrl] = useState<string | null>(null);
  const [followUpTarget, setFollowUpTarget] = useState<ExecutionRecord | null>(null);
  const [selectedTargetIds, setSelectedTargetIds] = useState<Set<string>>(new Set());
  const [selectedRequirementIds, setSelectedRequirementIds] = useState<Set<string>>(new Set());
  const [reviewMode, setReviewMode] = useState(false);
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
    placeholderData: cachedItem,
    ...defaultQueryOptions,
  });

  const depRelations = useMemo(
    () => item ? computeDependencyRelations(item, allBacklogItems) : { parents: [], children: [] },
    [item, allBacklogItems],
  );

  const spawnRef = item ? `${item.kind}/${item.name}` : "";
  const { data: spawnedItems } = useQuery({
    queryKey: ["backlog", "spawned-from", spawnRef],
    queryFn: () => backlogService.listBySpawnedFrom(spawnRef),
    enabled: !!spawnRef,
  });

  useEffect(() => {
    const desc = item?.description ?? "";
    // Heuristic: 3 lines of text-sm leading-relaxed is roughly 150 chars on
    // mobile, less on desktop. Any newline also signals multi-line content.
    setDescOverflows(desc.length > 120 || desc.includes("\n"));
  }, [item?.description]);

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
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
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

  // Fetch agent-manager external URL for "View Run" links in the activity timeline.
  useEffect(() => {
    let cancelled = false;
    fetch(`/embedded/${encodeURIComponent("agent-manager")}/external-url`)
      .then((res) => (res.ok ? res.json() : null))
      .then((data: { url?: string } | null) => {
        if (!cancelled && data?.url) setAgentManagerUiUrl(data.url);
      })
      .catch(() => { /* agent-manager not available */ });
    return () => { cancelled = true; };
  }, []);

  // Activity timeline (activities + executions merged) — fetches only when the drawer is open.
  const timeline = useActivityTimeline({ backlogKind: backlogKind ?? undefined, backlogName: name, enabled: isTimelineOpen, agentRunIsActive });

  // Auto-open follow-up dialog when navigated with ?action=followup
  const actionParam = searchParams.get("action");
  useEffect(() => {
    if (actionParam === "followup" && executionHistory && executionHistory.length > 0 && !followUpTarget) {
      setFollowUpTarget(executionHistory[0] ?? null);
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.delete("action");
        return next;
      }, { replace: true });
    }
  }, [actionParam, executionHistory, followUpTarget, setSearchParams]);

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
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const workshopRounds = useMemo(() => {
    if (!workshopRoundContents) return [];
    return workshopRoundContents
      .map((content) => parseWorkshopRound(content))
      .filter((r): r is { round: WorkshopRound; error?: string } => r.round !== null)
      .map((r) => r.round);
  }, [workshopRoundContents]);

  // Maturity / readiness data from the maturity-summary endpoint
  const { data: maturitySummaryData } = useQuery({
    queryKey: ["backlog-maturity-summary"],
    queryFn: () => backlogService.getMaturitySummary(),
    refetchInterval: agentRunIsActive ? AGENT_RUN_REFRESH_MS : false,
    ...defaultQueryOptions,
  });

  const readinessData = useMemo<ReadinessIndicatorData | null>(() => {
    if (!maturitySummaryData || !backlogKind || !name) return null;
    const match = (maturitySummaryData.items ?? []).find(
      (i) => i.kind === backlogKind && i.name === name,
    );
    return match ? buildReadinessData(match) : null;
  }, [maturitySummaryData, backlogKind, name]);
  const deliverableLabel = backlogKind === "research" ? "Conclusion" : "Plan";
  const deliverableLabelLower = deliverableLabel.toLowerCase();
  const workshopActionLabel = workshopRounds.length > 0 ? "Next Round" : "Workshop";
  // Finalization is complete when a finalize-mode round exists and there is no
  // pending synthesis (i.e. the deliverable is current with workshop answers).
  const isWorkshopFinalized = workshopRounds.some((r) => r.mode === "finalize")
    && !(readinessData?.pendingSynthesis ?? false);

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

  const itemActions: ItemActions | null = useMemo(() => {
    if (!item) return null;
    return getItemActions({
      item,
      allItems: allBacklogItems,
      readinessReady: readinessData ? readinessData.ready : null,
      pendingSynthesis: readinessData?.pendingSynthesis ?? false,
      agentRunning: agentRunIsActive,
      hasPendingDecisions: workshopRounds.some(
        (r) => r.items?.some((wi) => wi.type === "decision" && wi.selected == null),
      ),
      hasExecutionHistory: (executionHistory?.length ?? 0) > 0,
    });
  }, [item, allBacklogItems, readinessData, agentRunIsActive, workshopRounds, executionHistory]);
  const isLocked = itemActions?.locked ?? false;
  const isTerminal = itemActions?.terminal ?? false;
  const workshopBlockedDeps = itemActions?.blockingDepKeys ?? [];

  // Human-readable label for the active agent run mode (e.g. "Running workshop…").
  const agentRunningLabel = useMemo(() => {
    if (!agentRunIsActive || !latestAgentActivity) return "Agent running…";
    switch (latestAgentActivity.purpose) {
      case "workshop": return "Running workshop…";
      case "finalize": return "Running finalize…";
      case "research": return "Running research…";
      case "initialize": return "Initializing workshop…";
      case "process": return "Processing…";
      default: return "Agent running…";
    }
  }, [agentRunIsActive, latestAgentActivity]);

  const isPageLoading = isLoadingItem && !item;
  const pageError = itemError;
  const filesError = filesQueryError instanceof Error ? filesQueryError : null;

  const handleFileSelect = useCallback((file: BacklogFile) => {
    if (file.type === "file") {
      setSelectedFile(file);
      setActiveTab("files");
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
  }, [setActiveTab, setSearchParams]);

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
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        tags: values.tags,
      });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
      setShowEdit(false);
    },
  });

  const statusMutation = useMutation({
    mutationFn: (newStatus: BacklogStatus) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, { status: newStatus });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
    },
  });

  const depStatusMutation = useMutation({
    mutationFn: ({ kind, depName, newStatus }: { kind: string; depName: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind as BacklogKind, depName, { status: newStatus }),
    onSuccess: () => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      queryClient.invalidateQueries({ queryKey: ["backlog-list"] });
    },
  });

  const acceptanceGlobMutation = useMutation({
    mutationFn: (values: { acceptanceAllow: string[]; acceptanceDeny: string[] }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.update(backlogKind, name, {
        acceptanceAllow: values.acceptanceAllow,
        acceptanceDeny: values.acceptanceDeny,
      });
    },
    onSuccess: (updatedItem) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name] });
      upsertItem(updatedItem);
      setShowGlobDialog(false);
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
      closeDetail();
    },
  });

  const agentMutation = useMutation({
    mutationFn: ({ mode, prompt, contextPaths, contextTargetIds, contextRequirementIds }: {
      mode?: string;
      prompt: string;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
    }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.research(backlogKind, name, {
        mode,
        prompt,
        contextPaths,
        contextTargetIds,
        contextRequirementIds,
      });
    },
    onSuccess: (_result: ResearchResponse, _variables) => {
      setShowAgentDialog(false);
      if (!backlogKind || !name) return;
      void refreshActivities(true);
      void queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    },
  });

  // Workshop save mutation — saves user answers/decisions back to a round file
  // and auto-triggers the next round if the item isn't ready yet.
  const workshopSaveMutation = useMutation({
    mutationFn: async ({ roundNumber, content }: { roundNumber: number; content: string }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.workshopSave(backlogKind, name, roundNumber, content);
    },
    onSuccess: (result) => {
      if (!backlogKind || !name) return;
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "workshop-rounds"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
      void refetchFiles();
      void refetchWorkshopRounds();

      if (result.autoAdvance.triggered && result.autoAdvance.runId) {
        void refreshActivities(true);
      }
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

  const batchReviewMutation = useMutation({
    mutationFn: (items: ReviewUpdate[]) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.batchReview(backlogKind, name, items);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: archiveTargetsQueryKey });
    },
  });

  // Build module_id lookup for requirements (needed to resolve requirement → module mapping)
  const reqModuleMap = useMemo(() => {
    const map = new Map<string, string>();
    if (!archiveTargets) return map;
    const walk = (groups: typeof archiveTargets.requirements) => {
      for (const g of groups) {
        for (const r of g.requirements) {
          map.set(r.id, g.id);
        }
        walk(g.children);
      }
    };
    walk(archiveTargets.requirements);
    return map;
  }, [archiveTargets]);

  const targetIdSet = useMemo(
    () => new Set(archiveTargets?.targets.map((t) => t.id) ?? []),
    [archiveTargets],
  );

  const handleReviewAction = useCallback((id: string, _type: "target" | "requirement", action: ReviewAction) => {
    // Fire immediately — no local accumulation
    let item: ReviewUpdate;
    if (targetIdSet.has(id)) {
      item = { id, type: "target", ...action };
    } else {
      const moduleId = reqModuleMap.get(id);
      if (!moduleId) return;
      item = { id, type: "requirement", module_id: moduleId, ...action };
    }
    batchReviewMutation.mutate([item]);
  }, [targetIdSet, reqModuleMap, batchReviewMutation]);

  const handleBulkApprove = useCallback(() => {
    const items: ReviewUpdate[] = [];
    for (const id of selectedTargetIds) {
      items.push({ id, type: "target", review_status: "approved" });
    }
    for (const id of selectedRequirementIds) {
      const moduleId = reqModuleMap.get(id);
      if (moduleId) items.push({ id, type: "requirement", module_id: moduleId, review_status: "approved" });
    }
    if (items.length > 0) batchReviewMutation.mutate(items);
  }, [selectedTargetIds, selectedRequirementIds, reqModuleMap, batchReviewMutation]);

  const handleBulkFlag = useCallback(() => {
    const items: ReviewUpdate[] = [];
    for (const id of selectedTargetIds) {
      items.push({ id, type: "target", review_status: "flagged" });
    }
    for (const id of selectedRequirementIds) {
      const moduleId = reqModuleMap.get(id);
      if (moduleId) items.push({ id, type: "requirement", module_id: moduleId, review_status: "flagged" });
    }
    if (items.length > 0) batchReviewMutation.mutate(items);
  }, [selectedTargetIds, selectedRequirementIds, reqModuleMap, batchReviewMutation]);

  // Merge flagged items from archive data into agent dialog selections
  const agentDialogTargetIds = useMemo(() => {
    const merged = new Set(selectedTargetIds);
    for (const t of archiveTargets?.targets ?? []) {
      if (t.review_status === "flagged") merged.add(t.id);
    }
    return merged;
  }, [selectedTargetIds, archiveTargets]);

  const agentDialogRequirementIds = useMemo(() => {
    const merged = new Set(selectedRequirementIds);
    if (!archiveTargets) return merged;
    const walkReqs = (groups: typeof archiveTargets.requirements) => {
      for (const g of groups) {
        for (const r of g.requirements) {
          if (r.review_status === "flagged") merged.add(r.id);
        }
        walkReqs(g.children);
      }
    };
    walkReqs(archiveTargets.requirements);
    return merged;
  }, [selectedRequirementIds, archiveTargets]);

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

  // Workshop round deletion mutation.
  const [roundToDelete, setRoundToDelete] = useState<number | null>(null);
  const workshopDeleteRoundMutation = useMutation({
    mutationFn: async ({ roundNumber }: { roundNumber: number }) => {
      if (!backlogKind || !name) throw new Error("Backlog kind and name are required");
      return backlogService.workshopDeleteRound(backlogKind, name, roundNumber);
    },
    onSuccess: () => {
      if (!backlogKind || !name) return;
      setRoundToDelete(null);
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "files"] });
      queryClient.invalidateQueries({ queryKey: ["backlog", backlogKind, name, "workshop-rounds"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-maturity-summary"] });
      queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
      void refetchFiles();
      void refetchWorkshopRounds();
    },
  });

  // --- Workshop handlers ---
  const handleSaveRound = useCallback((roundNumber: number, content: string) => {
    workshopSaveMutation.mutate({ roundNumber, content });
  }, [workshopSaveMutation]);

  const startWorkshopMode = useCallback((mode: "workshop" | "finalize", prompt: string) => {
    if (!backlogKind || !name) return;
    agentMutation.mutate({
      mode,
      prompt,
    });
  }, [backlogKind, name, agentMutation]);

  const handleRunWorkshop = useCallback(() => {
    startWorkshopMode("workshop", "Run the next workshop round for this backlog item.");
  }, [startWorkshopMode]);

  const handleFinalizeWorkshop = useCallback(() => {
    startWorkshopMode(
      "finalize",
      `Finalize the latest workshop answers into the ${deliverableLabelLower} for this backlog item.`,
    );
  }, [deliverableLabelLower, startWorkshopMode]);

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
  const _workshopSaveError = workshopSaveMutation.isError
    ? workshopSaveMutation.error instanceof Error ? workshopSaveMutation.error.message : "Failed to save workshop round."
    : null;
  const searchResults = useMemo(
    () => collectMatchingFiles(files ?? [], fileSearch),
    [files, fileSearch]
  );
  const renderHeaderPrimaryAction = (className?: string) => {
    if (!itemActions) return null;

    switch (itemActions.primaryCta) {
      case "finalize":
        if (!itemActions.canFinalize && !itemActions.finalizeDisabled) return null;
        return (
          <Button
            variant="default"
            size="sm"
            className={className}
            onClick={handleFinalizeWorkshop}
            disabled={itemActions.finalizeDisabled || agentMutation.isPending}
          >
            <Sparkles className="mr-1.5 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : agentMutation.isPending ? "Starting..." : "Finalize"}
          </Button>
        );
      case "run":
        if (!itemActions.canRun && !itemActions.runDisabled) return null;
        return (
          <Button
            variant="default"
            size="sm"
            className={className}
            onClick={() => setShowRunModal(true)}
            disabled={itemActions.runDisabled}
            data-testid={selectors.backlogDetails.queueButton}
          >
            <Play className="mr-1.5 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : "Run"}
          </Button>
        );
      case "workshop":
        if (!itemActions.canWorkshop && !itemActions.workshopDisabled) return null;
        return (
          <Button
            variant="default"
            size="sm"
            className={className}
            onClick={handleRunWorkshop}
            disabled={itemActions.workshopDisabled || agentMutation.isPending}
          >
            <MessageSquareText className="mr-1.5 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : agentMutation.isPending ? "Starting..." : workshopActionLabel}
          </Button>
        );
      default:
        return null;
    }
  };
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

  const prevItemRef = useRef(`${backlogKind}/${name}`);
  useEffect(() => {
    const key = `${backlogKind}/${name}`;
    if (key !== prevItemRef.current) {
      prevItemRef.current = key;
      setActiveTab("info");
      setShowFilesSheet(false);
    }
  }, [backlogKind, name, setActiveTab]);

  useEffect(() => {
    if (activeTab === "info" && showFilesSheet) {
      setShowFilesSheet(false);
    }
  }, [activeTab, showFilesSheet]);

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

  const openTimeline = useCallback(() => {
    setIsTimelineOpen(true);
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

  const agentLabel = item?.kind === "idea" ? "Idea Agent" : "Workshop";
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
        <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
          <Info className="h-4 w-4 text-slate-400" />
          <h2 className="text-base font-semibold text-slate-100">Details</h2>
        </div>
        <div className="relative">
          <p
            className={`text-sm leading-relaxed text-slate-300 ${descExpanded ? "" : "line-clamp-3"}`}
            data-testid={selectors.backlogDetails.description}
          >
            {item.description || "No description provided"}
          </p>
          {(descOverflows || descExpanded) && (
            <button
              type="button"
              onClick={() => setDescExpanded(!descExpanded)}
              className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
            >
              {descExpanded ? "Show less" : "Show more\u2026"}
            </button>
          )}
        </div>
        {item.tags.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <Tags className="h-3.5 w-3.5" />
              Tags
            </div>
            <TagList tags={item.tags} maxTags={10} />
          </div>
        )}
        {item.initiative && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <Target className="h-3.5 w-3.5" />
              Initiative
            </div>
            <button
              type="button"
              onClick={() => item.initiative && selectInitiative(item.initiative)}
              className="inline-flex items-center rounded-full bg-sky-500/15 px-2.5 py-1 text-xs font-medium text-sky-400 transition-colors hover:bg-sky-500/25 hover:text-sky-300"
              data-testid={selectors.backlogDetails.initiativeChip}
            >
              {item.initiative}
            </button>
          </div>
        )}
        <DependencyChipList
          label="Depends on"
          items={depRelations.parents}
          icon={ArrowUpRight}
          onStatusChange={(dep, newStatus) =>
            depStatusMutation.mutate({ kind: dep.kind, depName: dep.name, newStatus })
          }
        />
        <DependencyChipList
          label="Depended on by"
          items={depRelations.children}
          icon={ArrowRightLeft}
          onStatusChange={(dep, newStatus) =>
            depStatusMutation.mutate({ kind: dep.kind, depName: dep.name, newStatus })
          }
        />
        {item.spawnedFrom && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <GitBranch className="h-3.5 w-3.5" />
              Spawned from
            </div>
            <Link
              to={`/backlog/${item.spawnedFrom}`}
              className="inline-flex items-center rounded-full bg-violet-500/15 px-2.5 py-1 text-xs font-medium text-violet-400 transition-colors hover:bg-violet-500/25 hover:text-violet-300"
            >
              {item.spawnedFrom}
            </Link>
          </div>
        )}
        {spawnedItems && spawnedItems.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <GitBranch className="h-3.5 w-3.5" />
              Spawned items
            </div>
            <div className="flex flex-wrap gap-1.5">
              {spawnedItems.map((si) => (
                <Link
                  key={`${si.kind}/${si.name}`}
                  to={`/backlog/${si.kind}/${si.name}`}
                  className="inline-flex items-center rounded-full bg-emerald-500/15 px-2.5 py-1 text-xs font-medium text-emerald-400 transition-colors hover:bg-emerald-500/25 hover:text-emerald-300"
                >
                  {si.title}
                </Link>
              ))}
            </div>
          </div>
        )}
        <div className="space-y-2 border-t border-slate-800 pt-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-slate-500">
              <FolderOpen className="h-3.5 w-3.5" />
              Acceptance Globs
            </div>
            {!isLocked && (
              <button
                type="button"
                onClick={() => setShowGlobDialog(true)}
                className="rounded p-1 text-slate-400 hover:bg-white/10 hover:text-white transition-colors"
                aria-label="Edit acceptance globs"
                data-testid="edit-acceptance-globs-btn"
              >
                <Edit className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          {(!item.acceptanceAllow || item.acceptanceAllow.length === 0) &&
           (!item.acceptanceDeny || item.acceptanceDeny.length === 0) ? (
            <button
              type="button"
              onClick={() => !isLocked && setShowGlobDialog(true)}
              disabled={isLocked}
              className="text-xs italic text-slate-500 hover:text-blue-400 transition-colors disabled:cursor-not-allowed disabled:hover:text-slate-500"
              data-testid="acceptance-globs-empty-state"
            >
              No patterns set — click to add
            </button>
          ) : (
            <>
              {item.acceptanceAllow && item.acceptanceAllow.length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-medium text-slate-500">Allow</p>
                  <div className="flex flex-wrap gap-1.5">
                    {(allowExpanded ? item.acceptanceAllow : item.acceptanceAllow.slice(0, 3)).map((glob) => (
                      <code
                        key={glob}
                        className="inline-block rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-300 font-mono"
                      >
                        {glob}
                      </code>
                    ))}
                  </div>
                  {item.acceptanceAllow.length > 3 && (
                    <button
                      type="button"
                      onClick={() => setAllowExpanded(!allowExpanded)}
                      className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                    >
                      {allowExpanded ? "Show less" : `Show more\u2026 (${item.acceptanceAllow.length - 3} more)`}
                    </button>
                  )}
                </div>
              )}
              {item.acceptanceDeny && item.acceptanceDeny.length > 0 && (
                <div className="space-y-1.5">
                  <p className="text-[11px] font-medium text-slate-500">Deny</p>
                  <div className="flex flex-wrap gap-1.5">
                    {(denyExpanded ? item.acceptanceDeny : item.acceptanceDeny.slice(0, 3)).map((glob) => (
                      <code
                        key={glob}
                        className="inline-block rounded bg-slate-800 px-2 py-0.5 text-xs text-slate-300 font-mono"
                      >
                        {glob}
                      </code>
                    ))}
                  </div>
                  {item.acceptanceDeny.length > 3 && (
                    <button
                      type="button"
                      onClick={() => setDenyExpanded(!denyExpanded)}
                      className="mt-1 text-xs font-medium text-blue-400 hover:text-blue-300"
                    >
                      {denyExpanded ? "Show less" : `Show more\u2026 (${item.acceptanceDeny.length - 3} more)`}
                    </button>
                  )}
                </div>
              )}
            </>
          )}
        </div>
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
      </div>
    </Card>
  ) : null;

  const targetScenarios = scenariosFromGlobs(item?.acceptanceAllow);

  const scenariosPanel = targetScenarios.length > 0 ? (
    <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
      <div className="space-y-3">
        <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
          <FolderOpen className="h-4 w-4 text-slate-400" />
          <h2 className="text-base font-semibold text-slate-100">Target Scenarios</h2>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {targetScenarios.map((scenarioName) => (
            <button
              key={scenarioName}
              type="button"
              onClick={() => selectScenario(scenarioName)}
              className="inline-flex items-center rounded-full bg-violet-500/15 px-2.5 py-1 text-xs font-medium text-violet-400 hover:bg-violet-500/25 transition-colors"
            >
              {scenarioName}
            </button>
          ))}
        </div>
        {(() => {
          const latestExec = executionHistory?.[0];
          if (latestExec?.finalization) {
            return (
              <div className="border-t border-slate-800 pt-2">
                <PostRunStatusBadge
                  execution={latestExec}
                  onRunChecks={async () => {
                    try {
                      await executionService.triggerReview(latestExec.executionId);
                    } catch {
                      // Will be visible on next query refetch
                    }
                  }}
                />
              </div>
            );
          }
          if (latestExec?.status === "validating") {
            return (
              <div className="border-t border-slate-800 pt-2">
                <PostRunStatusBadge
                  execution={{
                    ...latestExec,
                    finalization: {
                      eligible: true,
                      status: "running",
                      phase: "scope_detection",
                      scopeSource: "none",
                      warnings: [],
                      affectedScenarios: [],
                      aggregateClassification: "not_assessable",
                      scenarios: [],
                    },
                  }}
                />
              </div>
            );
          }
          if (!latestExec) {
            return (
              <div className="flex items-center gap-1.5 border-t border-slate-800 pt-2">
                <CheckCircle2 className="h-3.5 w-3.5 text-slate-500" />
                <span className="text-xs text-slate-400">Post-run checks will run after execution</span>
              </div>
            );
          }
          // Completed/failed execution with no finalization data — offer to run it.
          return (
            <div className="space-y-2 border-t border-slate-800 pt-2">
              <div className="flex items-center gap-1.5">
                <CheckCircle2 className="h-3.5 w-3.5 text-slate-500" />
                <span className="text-xs text-slate-400">No post-run checks yet</span>
              </div>
              <Button
                size="sm"
                variant="outline"
                className="w-full"
                onClick={async () => {
                  try {
                    await executionService.triggerReview(latestExec.executionId);
                  } catch {
                    // Error handled by query refetch showing updated state
                  }
                }}
              >
                <RefreshCw className="mr-1.5 h-3 w-3" />
                Run Post-Run Checks
              </Button>
            </div>
          );
        })()}
      </div>
    </Card>
  ) : null;

  const notesPanel = (
    <div className="space-y-4">
      {readinessData && !isTerminal && (
        <ReadinessDetailsPanel
          data={readinessData}
          kind={backlogKind as BacklogKind}
          onRun={itemActions?.canRun ? () => setShowRunModal(true) : undefined}
        />
      )}
      {isLocked && (
        <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-2 text-sm text-amber-300">
          This item is {item ? formatBacklogStatus(item.status) : "locked"} and cannot be edited.
        </div>
      )}
      {workshopBlockedDeps.length > 0 && workshopRounds.length === 0 && (
        <div className="rounded-lg border border-orange-500/30 bg-orange-500/10 p-3">
          <div className="flex items-center justify-between gap-3">
            <p className="text-sm text-orange-300">
              Workshop paused &mdash; waiting for:{" "}
              {workshopBlockedDeps.map((dep, i) => {
                const slashIdx = dep.indexOf("/");
                const depKind = slashIdx > 0 ? dep.slice(0, slashIdx) : "";
                const depName = slashIdx > 0 ? dep.slice(slashIdx + 1) : dep;
                return (
                  <span key={dep}>
                    {i > 0 && ", "}
                    <button
                      type="button"
                      onClick={() => {
                        if (depKind && depName) {
                          useDetailSelectionStore.getState().selectBacklog(depKind, depName);
                        }
                      }}
                      className="font-medium text-orange-200 underline decoration-orange-500/40 hover:text-orange-100 hover:decoration-orange-400/60"
                    >
                      {dep}
                    </button>
                  </span>
                );
              })}
            </p>
            <button
              className="shrink-0 text-xs text-orange-400 hover:text-orange-300 underline"
              onClick={() => setShowForceWorkshopConfirm(true)}
            >
              Start Anyway
            </button>
          </div>
        </div>
      )}
      <ConfirmDialog
        isOpen={showForceWorkshopConfirm}
        onClose={() => setShowForceWorkshopConfirm(false)}
        title="Start Workshop Despite Unplanned Dependencies?"
        description={`Dependencies not yet planned: ${workshopBlockedDeps.join(", ")}. Starting now may produce a ${backlogKind === "research" ? "conclusion" : "plan"} that needs revision when these dependencies are finalized.`}
        confirmLabel="Start Workshop"
        onConfirm={() => {
          setShowForceWorkshopConfirm(false);
          agentMutation.mutate({
            mode: "initialize",
            prompt: "Initialize the first workshop round for this backlog item.",
          });
        }}
      />
      <ConfirmDialog
        isOpen={roundToDelete !== null}
        onClose={() => setRoundToDelete(null)}
        title="Delete Workshop Round"
        description={`Round ${roundToDelete} and all its decisions will be permanently deleted. Subsequent rounds will be renumbered.`}
        confirmLabel="Delete Round"
        isLoading={workshopDeleteRoundMutation.isPending}
        onConfirm={() => {
          if (roundToDelete !== null) {
            workshopDeleteRoundMutation.mutate({ roundNumber: roundToDelete });
          }
        }}
      />
      <WorkshopPanel
        rounds={workshopRounds}
        backlogKind={backlogKind as BacklogKind}
        backlogName={name ?? ""}
        disabled={isLocked || isTerminal}
        isSaving={workshopSaveMutation.isPending}
        isRunningWorkshop={agentMutation.isPending || agentRunIsActive}
        onSaveRound={handleSaveRound}
        primaryActionLabel={(!isTerminal && !itemActions?.blocked && (itemActions?.canFinalize || itemActions?.finalizeDisabled)) ? `Finalize ${deliverableLabel}` : undefined}
        onPrimaryAction={(!isTerminal && !itemActions?.blocked && (itemActions?.canFinalize || itemActions?.finalizeDisabled)) ? handleFinalizeWorkshop : undefined}
        onRunWorkshop={!isTerminal && !itemActions?.blocked ? handleRunWorkshop : undefined}
        workshopActionLabel={workshopActionLabel}
        onDeleteRound={isTerminal ? undefined : setRoundToDelete}
        isDeletingRound={workshopDeleteRoundMutation.isPending}
        isFinalized={isWorkshopFinalized}
        deliverableLabel={deliverableLabel}
        runningLabel={agentRunningLabel}
      />
    </div>
  );

  const activeRunBanner = agentRunIsActive && latestAgentActivity ? (
    <div
      className="flex items-center gap-2 rounded-lg border border-cyan-500/30 bg-cyan-500/10 px-3 py-1.5"
      data-testid={selectors.backlogDetails.activeRunBanner}
    >
      <span className="relative flex h-2 w-2 shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
      <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
      </span>
      <span className="text-xs font-medium capitalize text-cyan-200">
        {latestAgentActivity.status.replace("_", " ")}
      </span>
      {latestAgentActivity.purpose && (
        <span className="rounded bg-cyan-500/20 px-1.5 py-0.5 text-[11px] font-medium text-cyan-300">
          {latestAgentActivity.purpose.replace("_", " ")}
        </span>
      )}
      <span className="text-xs text-slate-400">
        {formatRelativeTime(latestAgentActivity.requestedAt)}
      </span>
      <div className="ml-auto flex items-center gap-1.5">
        <Button
          variant="outline"
          size="sm"
          className="h-7 px-2 text-xs"
          onClick={() => void stopRun(latestAgentActivity.runId ?? "")}
          disabled={latestAgentActivity.isStopping}
        >
          <Square className="mr-1 h-3 w-3" />
          {latestAgentActivity.isStopping ? "Stopping..." : "Stop"}
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="h-7 w-7 p-0"
          onClick={() => closeDetail()}
          aria-label="View execution"
        >
          <ArrowUpRight className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  ) : null;


  const renderActionButtons = () => {
    const runAction = (action: () => void) => {
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
        {(itemActions?.canFinalize || itemActions?.finalizeDisabled) && (
          <Button
            variant="default"
            size="sm"
            className={itemActions.primaryCta === "finalize" ? primaryRowButtonClass : rowButtonClass}
            onClick={() => runAction(handleFinalizeWorkshop)}
            disabled={itemActions.finalizeDisabled || agentMutation.isPending}
          >
            <Sparkles className="mr-2 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : agentMutation.isPending ? "Starting..." : `Finalize ${deliverableLabel}`}
          </Button>
        )}
        {(itemActions?.canRun || itemActions?.runDisabled) && (
          <Button
            variant="default"
            size="sm"
            className={itemActions.primaryCta === "run" ? primaryRowButtonClass : rowButtonClass}
            onClick={() => runAction(() => setShowRunModal(true))}
            disabled={itemActions.runDisabled}
          >
            <Play className="mr-2 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : "Run"}
          </Button>
        )}
        {(itemActions?.canWorkshop || itemActions?.workshopDisabled) && (
          <Button
            variant="default"
            size="sm"
            className={itemActions.primaryCta === "workshop" ? primaryRowButtonClass : rowButtonClass}
            onClick={() => runAction(handleRunWorkshop)}
            disabled={itemActions.workshopDisabled || agentMutation.isPending}
          >
            <MessageSquareText className="mr-2 h-4 w-4" />
            {itemActions.agentRunning ? agentRunningLabel : agentMutation.isPending ? "Starting..." : workshopActionLabel}
          </Button>
        )}
        {itemActions?.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.canRun && !itemActions.runDisabled && !itemActions.canWorkshop && !itemActions.workshopDisabled && !itemActions.canFinalize && !itemActions.finalizeDisabled ? (
          <p className="text-xs text-slate-500">{itemActions.notQueueableReason}</p>
        ) : null}
        <Button
          variant="outline"
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => setShowEdit(true))}
        >
          <Edit className="mr-2 h-4 w-4" />
          Edit
        </Button>
        {itemActions?.canFollowUp ? (
          <Button
            variant="outline"
            size="sm"
            className={rowButtonClass}
            onClick={() => runAction(() => setFollowUpTarget(executionHistory?.[0] ?? null))}
          >
            <MessageSquare className="mr-2 h-4 w-4" />
            Follow Up
          </Button>
        ) : !isTerminal ? (
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
        ) : null}
        {itemActions?.canArchive && item && (
          <Button
            variant="outline"
            size="sm"
            className={rowButtonClass}
            onClick={() => runAction(() => {
              updateMutation.mutate({
                title: item.title,
                description: item.description,
                status: "archived",
                priority: item.priority,
                tags: item.tags,
              });
            })}
            disabled={updateMutation.isPending}
          >
            <Archive className="mr-2 h-4 w-4" />
            {updateMutation.isPending ? "Archiving..." : "Archive"}
          </Button>
        )}
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
                  });
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
      {deleteError && (
        <Card padding="sm" className="space-y-2 rounded-lg border-slate-700/60 bg-slate-900/45">
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {deleteError}
          </div>
        </Card>
      )}
      {activeRunBanner}
      {detailsPanel}
      {scenariosPanel}
      {notesPanel}
      {archiveTargets?.has_archive && (
        <>
          <OperationalTargetsPanel
            targets={archiveTargets.targets}
            requirements={archiveTargets.requirements}
            selectedTargetIds={selectedTargetIds}
            selectedRequirementIds={selectedRequirementIds}
            onTargetToggle={reviewMode ? undefined : handleTargetToggle}
            onRequirementToggle={reviewMode ? undefined : handleRequirementToggle}
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
            reviewMode={reviewMode}
            onToggleReviewMode={() => setReviewMode((prev) => !prev)}
            onReviewAction={handleReviewAction}
            reviewSaving={batchReviewMutation.isPending}
            reviewError={batchReviewMutation.error instanceof Error ? batchReviewMutation.error.message : null}
          />
          {!reviewMode && (
            <BulkActionToolbar
              selectedCount={selectedTargetIds.size + selectedRequirementIds.size}
              onApproveSelected={handleBulkApprove}
              onFlagSelected={handleBulkFlag}
              onSendToAgent={() => setShowAgentDialog(true)}
              onClearSelection={() => {
                setSelectedTargetIds(new Set());
                setSelectedRequirementIds(new Set());
              }}
            />
          )}
        </>
      )}

    </div>
  );

  const tabBar = item ? (
    <div className="border-t border-slate-800/50" data-testid={selectors.backlogDetails.tabRow}>
      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as DetailsTab)}>
        <TabsList className="w-full flex-nowrap justify-start gap-1 overflow-x-auto no-scrollbar rounded-none bg-transparent p-0 px-3">
          <TabsTrigger value="info" className="gap-2" data-testid={selectors.backlogDetails.tabInfo}>
            <CircleHelp className="h-4 w-4" />
            Info
          </TabsTrigger>
          <TabsTrigger value="prompt" className="gap-2" data-testid={selectors.backlogDetails.tabPrompt}>
            <Sparkles className="h-4 w-4" />
            {backlogKind === "research" ? "Conclusion" : "Plan"}
          </TabsTrigger>
          <TabsTrigger value="files" className="gap-2" data-testid={selectors.backlogDetails.tabFiles}>
            <Files className="h-4 w-4" />
            Files
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  ) : null;

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType={backlogKind ? BACKLOG_KIND_LABELS[backlogKind] : "backlog"}
          entityIcon={backlogKind ? BACKLOG_KIND_ICONS[backlogKind] : undefined}
          title={item?.title ?? name ?? "Loading..."}
          status={item?.status}
          nodeId={nodeId}
          lenses={BACKLOG_LENSES}
          actions={item ? renderHeaderPrimaryAction("shrink-0") : undefined}
          onStatusChange={!isLocked ? (newStatus) => statusMutation.mutate(newStatus) : undefined}
          statusChangePending={statusMutation.isPending}
          tabBar={tabBar}
        />
      }
      mobileActions={item ? renderActionButtons() : undefined}
      mobileActionsTitle="Backlog Actions"
    >
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
          {/* Mobile tab content */}
          <div className="lg:hidden">
            {activeTab === "info" && mobileInfoView}
            {activeTab === "prompt" && backlogKind && name && (
              <PlanPanel backlogKind={backlogKind as BacklogKind} backlogName={name} className="flex-1 overflow-y-auto" />
            )}
            {activeTab === "files" && fileWorkspace}
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
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 w-7 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
                    onClick={openTimeline}
                    aria-label="View activity timeline"
                    data-testid={selectors.backlogDetails.timelineButton}
                  >
                    <History className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <h1
                  className="text-xl font-bold text-slate-100 sm:text-2xl"
                  data-testid={selectors.backlogDetails.title}
                >
                  {item.title}
                </h1>
              </div>

              <div className="flex flex-wrap items-center gap-2">
                {renderHeaderPrimaryAction()}
                {itemActions?.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.canRun && !itemActions.runDisabled && !itemActions.canWorkshop && !itemActions.workshopDisabled && !itemActions.canFinalize && !itemActions.finalizeDisabled ? (
                  <span className="max-w-xs text-xs text-slate-500">{itemActions.notQueueableReason}</span>
                ) : null}
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
              </div>
            </div>

            {deleteError && (
              <div className="mt-4 space-y-2">
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                  {deleteError}
                </div>
              </div>
            )}
          </Card>

            <div>
              {activeTab === "info" && (
                <div className="space-y-6 pt-6">
  
                  {activeRunBanner}
                  {detailsPanel}
                  {scenariosPanel}
                  {notesPanel}
                  {archiveTargets?.has_archive && (
                    <>
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
                        reviewMode={reviewMode}
                        onToggleReviewMode={() => setReviewMode((prev) => !prev)}
                        onReviewAction={handleReviewAction}
                        reviewSaving={batchReviewMutation.isPending}
                        reviewError={batchReviewMutation.error instanceof Error ? batchReviewMutation.error.message : null}
                      />
                    </>
                  )}
            
                </div>
              )}
              {activeTab === "prompt" && backlogKind && name && (
                <PlanPanel
                  backlogKind={backlogKind as BacklogKind}
                  backlogName={name}
                  className="mt-6 min-h-[500px] rounded-lg border border-slate-800 bg-slate-900/50"
                />
              )}
              {activeTab === "files" && (
                <div className="pt-6">
                  {fileWorkspace}
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
            })
          }
        />
      )}

      {item && (
        <AcceptanceGlobDialog
          isOpen={showGlobDialog}
          onClose={() => {
            setShowGlobDialog(false);
            acceptanceGlobMutation.reset();
          }}
          initialAllow={item.acceptanceAllow ?? []}
          initialDeny={item.acceptanceDeny ?? []}
          onSave={(allow, deny) =>
            acceptanceGlobMutation.mutate({ acceptanceAllow: allow, acceptanceDeny: deny })
          }
          isSubmitting={acceptanceGlobMutation.isPending}
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
        errorMessage={agentError}
        files={files}
        archiveTargets={archiveTargets}
        initialSelectedTargetIds={agentDialogTargetIds}
        initialSelectedRequirementIds={agentDialogRequirementIds}
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

      {followUpTarget && (
        <FollowUpDialog
          isOpen={Boolean(followUpTarget)}
          onClose={() => setFollowUpTarget(null)}
          execution={followUpTarget}
          onSuccess={() => {
            setFollowUpTarget(null);
            void queryClient.invalidateQueries({ queryKey: ["execution-history"] });
          }}
        />
      )}

      <Drawer
        isOpen={isTimelineOpen}
        onClose={() => setIsTimelineOpen(false)}
        title="Activity Timeline"
        description="Executions and agent activities for this backlog item"
        testId={selectors.backlogDetails.activityTimeline}
      >
        <ActivityTimeline
          entries={timeline.entries}
          isLoading={timeline.isLoading}
          error={timeline.error}
          onViewExecution={() => { setIsTimelineOpen(false); closeDetail(); }}
          onStopRun={(runId) => void stopRun(runId)}
          onFollowUp={(exec) => { setIsTimelineOpen(false); setFollowUpTarget(exec); }}
          latestAgentActivity={latestAgentActivity ?? undefined}
          agentRunIsActive={agentRunIsActive}
          agentManagerUiUrl={agentManagerUiUrl ?? undefined}
        />
      </Drawer>

      <ClarificationPanel
        onAction={(action) => {
          if (action === "invalidate_round" || action === "remove_decision" || action === "update_decision") {
            void refetchItem();
          }
        }}
      />

    </div>
    </DetailPageLayout>
  );
}
