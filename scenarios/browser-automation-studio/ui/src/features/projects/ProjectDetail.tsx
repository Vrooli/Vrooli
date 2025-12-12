import { useState, useEffect, useRef, useCallback, useMemo } from "react";
import { logger } from "@utils/logger";
import {
  ArrowLeft,
  Plus,
  FileCode,
  Play,
  Clock,
  PlayCircle,
  Loader,
  WifiOff,
  X,
  FolderOpen,
  LayoutGrid,
  FolderTree,
  ChevronRight,
  ChevronDown,
  Trash2,
  CheckSquare,
  Square,
  ListChecks,
  PencilLine,
  UploadCloud,
  History,
  Info,
  MoreVertical,
  Search,
  Circle,
  RefreshCw,
  FileText,
} from "lucide-react";
import { Project, useProjectStore } from "@stores/projectStore";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useExecutionStore } from "@stores/executionStore";
import { getConfig } from "@/config";
import toast from "react-hot-toast";
import ProjectModal from "./ProjectModal";
import { ExecutionHistory, ExecutionViewer } from "@features/execution";
import { usePopoverPosition } from "@hooks/usePopoverPosition";
import { selectors } from "@constants/selectors";
import ResponsiveDialog from "@shared/layout/ResponsiveDialog";

// Extended Workflow interface with API response fields
interface WorkflowWithStats extends Workflow {
  folder_path?: string;
  created_at?: string;
  updated_at?: string;
  project_id?: string;
  stats?: {
    execution_count: number;
    last_execution?: string;
    success_rate?: number;
  };
}

interface ProjectDetailProps {
  project: Project;
  onBack: () => void;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
  onCreateWorkflow: () => void;
  onCreateWorkflowDirect?: () => void;
  onStartRecording?: () => void;
}

type ProjectEntryKind = "folder" | "workflow_file" | "asset_file";

interface ProjectEntry {
  id: string;
  project_id: string;
  path: string;
  kind: ProjectEntryKind;
  workflow_id?: string;
  metadata?: Record<string, unknown>;
}

type FileTreeNodeKind = "folder" | "workflow_file" | "asset_file";

interface FileTreeNode {
  kind: FileTreeNodeKind;
  path: string;
  name: string;
  children?: FileTreeNode[];
  workflowId?: string;
  metadata?: Record<string, unknown>;
}

type FileTreeDragPayload = {
  path: string;
  kind: FileTreeNodeKind;
};

type ConfirmDialogState = {
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
};

type PromptDialogState = {
  title: string;
  message?: string;
  label: string;
  defaultValue?: string;
  placeholder?: string;
  submitLabel?: string;
  cancelLabel?: string;
};

function ProjectDetail({
  project,
  onBack,
  onWorkflowSelect,
  onCreateWorkflow,
  onCreateWorkflowDirect,
  onStartRecording,
}: ProjectDetailProps) {
  const [workflows, setWorkflows] = useState<WorkflowWithStats[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [executionInProgress, setExecutionInProgress] = useState<
    Record<string, boolean>
  >({});
  const [viewMode, setViewMode] = useState<"card" | "tree">("card");
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(
    new Set(),
  );
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedWorkflows, setSelectedWorkflows] = useState<Set<string>>(
    new Set(),
  );
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);
  const [isDeletingProject, setIsDeletingProject] = useState(false);
  const [deletingWorkflowId, setDeletingWorkflowId] = useState<string | null>(null);
  const [showWorkflowActionsFor, setShowWorkflowActionsFor] = useState<string | null>(null);
  const [showEditProjectModal, setShowEditProjectModal] = useState(false);
  const [isImportingRecording, setIsImportingRecording] = useState(false);
  const [activeTab, setActiveTab] = useState<"workflows" | "executions">(
    "workflows",
  );
  const [showStatsPopover, setShowStatsPopover] = useState(false);
  const [showViewModeDropdown, setShowViewModeDropdown] = useState(false);
  const [showMoreMenu, setShowMoreMenu] = useState(false);

  const [selectedTreeFolder, setSelectedTreeFolder] = useState<string | null>(
    null,
  );
  const [dragSourcePath, setDragSourcePath] = useState<string | null>(null);
  const [dropTargetFolder, setDropTargetFolder] = useState<string | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState | null>(
    null,
  );
  const confirmResolveRef = useRef<((value: boolean) => void) | null>(null);

  const [promptDialog, setPromptDialog] = useState<PromptDialogState | null>(null);
  const promptResolveRef = useRef<((value: string | null) => void) | null>(null);
  const promptValidateRef = useRef<((value: string) => string | null) | null>(null);
  const promptNormalizeRef = useRef<((value: string) => string) | null>(null);
  const [promptValue, setPromptValue] = useState("");
  const [promptError, setPromptError] = useState<string | null>(null);
  const [showNewFileMenu, setShowNewFileMenu] = useState(false);
  const [projectEntries, setProjectEntries] = useState<ProjectEntry[]>([]);
  const [projectEntriesLoading, setProjectEntriesLoading] = useState(false);
  const [projectEntriesError, setProjectEntriesError] = useState<string | null>(
    null,
  );
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const statsButtonRef = useRef<HTMLButtonElement | null>(null);
  const statsPopoverRef = useRef<HTMLDivElement | null>(null);
  const viewModeButtonRef = useRef<HTMLButtonElement | null>(null);
  const viewModeDropdownRef = useRef<HTMLDivElement | null>(null);
  const moreMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const moreMenuRef = useRef<HTMLDivElement | null>(null);
  const newFileMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const newFileMenuRef = useRef<HTMLDivElement | null>(null);
  const { floatingStyles: statsPopoverStyles } = usePopoverPosition(
    statsButtonRef,
    statsPopoverRef,
    {
      isOpen: showStatsPopover,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
    },
  );
  const { floatingStyles: viewModeDropdownStyles } = usePopoverPosition(
    viewModeButtonRef,
    viewModeDropdownRef,
    {
      isOpen: showViewModeDropdown,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
      matchReferenceWidth: true,
    },
  );
  const { floatingStyles: moreMenuStyles } = usePopoverPosition(
    moreMenuButtonRef,
    moreMenuRef,
    {
      isOpen: showMoreMenu,
      placementPriority: ["bottom-end", "top-end", "bottom-start", "top-start"],
    },
  );
  const { floatingStyles: newFileMenuStyles } = usePopoverPosition(
    newFileMenuButtonRef,
    newFileMenuRef,
    {
      isOpen: showNewFileMenu,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
    },
  );
  const { deleteProject } = useProjectStore();
  const { bulkDeleteWorkflows } = useWorkflowStore();
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const isExecutionViewerOpen = Boolean(currentExecution);

  const normalizeProjectRelPathClient = useCallback((raw: string) => {
    const trimmed = raw.trim().replace(/\\/g, "/");
    const withoutLeading = trimmed.replace(/^\/+/, "").replace(/\/+$/, "");
    if (!withoutLeading) {
      return { ok: false as const, error: "Path is required." };
    }
    const parts: string[] = withoutLeading.split("/");
    for (const part of parts) {
      const segment = part.trim();
      if (!segment || segment === "." || segment === "..") {
        return { ok: false as const, error: "Path contains invalid segments." };
      }
    }
    return {
      ok: true as const,
      path: parts.map((segment: string) => segment.trim()).join("/"),
    };
  }, []);

  const fileBasename = useCallback((relPath: string) => {
    const normalized = relPath.replace(/\\/g, "/").replace(/^\/+/, "").replace(/\/+$/, "");
    const parts = normalized.split("/").filter(Boolean);
    return parts.length > 0 ? parts[parts.length - 1] : "";
  }, []);

  const parseDragPayload = useCallback(
    (dt: DataTransfer | null): FileTreeDragPayload | null => {
      if (!dt) return null;
      try {
        const raw = dt.getData("application/x-bas-project-entry");
        if (!raw) return null;
        const parsed = JSON.parse(raw) as Partial<FileTreeDragPayload>;
        if (!parsed || typeof parsed.path !== "string" || typeof parsed.kind !== "string") {
          return null;
        }
        if (
          parsed.kind !== "folder" &&
          parsed.kind !== "workflow_file" &&
          parsed.kind !== "asset_file"
        ) {
          return null;
        }
        return { path: parsed.path, kind: parsed.kind };
      } catch {
        return null;
      }
    },
    [],
  );

  const requestConfirm = useCallback(async (opts: ConfirmDialogState) => {
    if (confirmResolveRef.current) {
      confirmResolveRef.current(false);
      confirmResolveRef.current = null;
    }
    return await new Promise<boolean>((resolve) => {
      confirmResolveRef.current = resolve;
      setConfirmDialog(opts);
    });
  }, []);

  const closeConfirmDialog = useCallback((result: boolean) => {
    const resolve = confirmResolveRef.current;
    confirmResolveRef.current = null;
    setConfirmDialog(null);
    resolve?.(result);
  }, []);

  const requestPrompt = useCallback(
    async (
      opts: PromptDialogState,
      config?: {
        validate?: (value: string) => string | null;
        normalize?: (value: string) => string;
      },
    ) => {
      if (promptResolveRef.current) {
        promptResolveRef.current(null);
        promptResolveRef.current = null;
      }

      promptValidateRef.current = config?.validate ?? null;
      promptNormalizeRef.current = config?.normalize ?? null;

      setPromptError(null);
      setPromptValue(opts.defaultValue ?? "");
      return await new Promise<string | null>((resolve) => {
        promptResolveRef.current = resolve;
        setPromptDialog(opts);
      });
    },
    [],
  );

  const closePromptDialog = useCallback((result: string | null) => {
    const resolve = promptResolveRef.current;
    promptResolveRef.current = null;
    promptValidateRef.current = null;
    promptNormalizeRef.current = null;
    setPromptDialog(null);
    setPromptError(null);
    resolve?.(result);
  }, []);

  // Memoize filtered workflows to prevent recalculation on every render
  const filteredWorkflows = useMemo(() => {
    return workflows.filter(
      (workflow) =>
        workflow.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (workflow.description as string | undefined)
          ?.toLowerCase()
          .includes(searchTerm.toLowerCase()),
    );
  }, [workflows, searchTerm]);

  const fetchWorkflows = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/workflows`,
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch workflows: ${response.status}`);
      }
      const data = await response.json();
      const workflowList = data.workflows || [];
      setWorkflows(workflowList);
      setSelectedWorkflows(new Set());
      setSelectionMode(false);
    } catch (error) {
      logger.error(
        "Failed to fetch workflows",
        {
          component: "ProjectDetail",
          action: "loadWorkflows",
          projectId: project.id,
        },
        error,
      );
      setError(
        error instanceof Error ? error.message : "Failed to fetch workflows",
      );
    } finally {
      setIsLoading(false);
    }
  }, [project.id]);

  const fetchProjectEntries = useCallback(async () => {
    setProjectEntriesLoading(true);
    setProjectEntriesError(null);
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/tree`,
      );
      if (!response.ok) {
        throw new Error(`Failed to fetch project files: ${response.status}`);
      }
      const payload = (await response.json()) as { entries?: ProjectEntry[] };
      setProjectEntries(Array.isArray(payload.entries) ? payload.entries : []);
    } catch (error) {
      logger.error(
        "Failed to fetch project file tree",
        {
          component: "ProjectDetail",
          action: "fetchProjectEntries",
          projectId: project.id,
        },
        error,
      );
      setProjectEntriesError(
        error instanceof Error ? error.message : "Failed to fetch project files",
      );
    } finally {
      setProjectEntriesLoading(false);
    }
  }, [project.id]);

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

  useEffect(() => {
    fetchProjectEntries();
  }, [fetchProjectEntries]);

  useEffect(() => {
    if (!showStatsPopover) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        statsPopoverRef.current &&
        !statsPopoverRef.current.contains(target) &&
        statsButtonRef.current &&
        !statsButtonRef.current.contains(target)
      ) {
        setShowStatsPopover(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowStatsPopover(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showStatsPopover]);

  useEffect(() => {
    if (!showViewModeDropdown) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        viewModeDropdownRef.current &&
        !viewModeDropdownRef.current.contains(target) &&
        viewModeButtonRef.current &&
        !viewModeButtonRef.current.contains(target)
      ) {
        setShowViewModeDropdown(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowViewModeDropdown(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showViewModeDropdown]);

  useEffect(() => {
    if (!showNewFileMenu) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        newFileMenuRef.current &&
        !newFileMenuRef.current.contains(target) &&
        newFileMenuButtonRef.current &&
        !newFileMenuButtonRef.current.contains(target)
      ) {
        setShowNewFileMenu(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowNewFileMenu(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showNewFileMenu]);

  useEffect(() => {
    if (!showMoreMenu) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        moreMenuRef.current &&
        !moreMenuRef.current.contains(target) &&
        moreMenuButtonRef.current &&
        !moreMenuButtonRef.current.contains(target)
      ) {
        setShowMoreMenu(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowMoreMenu(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showMoreMenu]);

  // Memoize formatDate to prevent recreation on every render
  const formatDate = useCallback((dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }, []);

  const handleRecordingImportClick = useCallback(() => {
    if (isImportingRecording) {
      return;
    }
    fileInputRef.current?.click();
  }, [isImportingRecording]);

  const handleRecordingImport = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = event.target.files;
      if (!files || files.length === 0) {
        return;
      }

      const file = files[0];
      setIsImportingRecording(true);

      try {
        const config = await getConfig();
        const formData = new FormData();
        formData.append("file", file);
        formData.append("project_id", project.id);
        if (project.name) {
          formData.append("project_name", project.name);
        }

        const response = await fetch(`${config.API_URL}/recordings/import`, {
          method: "POST",
          body: formData,
        });

        if (!response.ok) {
          const text = await response.text();
          try {
            const payload = JSON.parse(text);
            const message =
              payload.message || payload.error || "Failed to import recording";
            throw new Error(message);
          } catch (error) {
            throw new Error(text || "Failed to import recording");
          }
        }

        const payload = await response.json();
        const executionId = payload.execution_id || payload.executionId;
        toast.success(
          `Recording imported${executionId ? ` (execution ${executionId})` : ""}.`,
        );
      } catch (error) {
        const message =
          error instanceof Error ? error.message : "Failed to import recording";
        toast.error(message);
      } finally {
        setIsImportingRecording(false);
        if (event.target) {
          event.target.value = "";
        }
      }
    },
    [project.id, project.name],
  );

  const handleExecuteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string) => {
      e.stopPropagation(); // Prevent workflow selection
      setExecutionInProgress((prev) => ({ ...prev, [workflowId]: true }));

      try {
        await startExecution(workflowId);
        setActiveTab("executions");
        logger.info("Workflow execution started", {
          component: "ProjectDetail",
          action: "handleExecuteWorkflow",
          workflowId,
        });
      } catch (error) {
        logger.error(
          "Failed to execute workflow",
          {
            component: "ProjectDetail",
            action: "handleExecuteWorkflow",
            workflowId,
          },
          error,
        );
        alert("Failed to execute workflow. Please try again.");
      } finally {
        setExecutionInProgress((prev) => ({ ...prev, [workflowId]: false }));
      }
    },
    [startExecution],
  );

  const handleDeleteWorkflow = useCallback(
    async (e: React.MouseEvent, workflowId: string, workflowName: string) => {
      e.stopPropagation();
      setShowWorkflowActionsFor(null);

      const confirmed = await requestConfirm({
        title: "Delete workflow?",
        message: `Delete "${workflowName}"? This cannot be undone.`,
        confirmLabel: "Delete",
        cancelLabel: "Cancel",
        danger: true,
      });
      if (!confirmed) {
        return;
      }

      setDeletingWorkflowId(workflowId);
      try {
        const deletedIds = await bulkDeleteWorkflows(project.id, [workflowId]);
        if (deletedIds.includes(workflowId)) {
          setWorkflows((prev) => prev.filter((w) => w.id !== workflowId));
          toast.success(`Workflow "${workflowName}" deleted`);
        } else {
          throw new Error("Workflow was not deleted");
        }
      } catch (error) {
        logger.error(
          "Failed to delete workflow",
          {
            component: "ProjectDetail",
            action: "handleDeleteWorkflow",
            workflowId,
          },
          error
        );
        toast.error("Failed to delete workflow");
      } finally {
        setDeletingWorkflowId(null);
      }
    },
    [project.id, bulkDeleteWorkflows, requestConfirm]
  );

  const toggleSelectionMode = useCallback(() => {
    setSelectionMode((prev) => {
      if (prev) {
        setSelectedWorkflows(new Set());
      }
      return !prev;
    });
  }, []);

  const toggleWorkflowSelection = useCallback((workflowId: string) => {
    setSelectedWorkflows((prev) => {
      const next = new Set(prev);
      if (next.has(workflowId)) {
        next.delete(workflowId);
      } else {
        next.add(workflowId);
      }
      return next;
    });
  }, []);

  const handleSelectAll = useCallback(() => {
    if (selectedWorkflows.size === filteredWorkflows.length) {
      setSelectedWorkflows(new Set());
      return;
    }
    setSelectedWorkflows(
      new Set(filteredWorkflows.map((workflow) => workflow.id)),
    );
  }, [selectedWorkflows.size, filteredWorkflows]);

  const handleBulkDeleteSelected = useCallback(async () => {
    if (selectedWorkflows.size === 0) {
      return;
    }

    const confirmed = await requestConfirm({
      title: "Delete workflows?",
      message: `Delete ${selectedWorkflows.size} workflow${selectedWorkflows.size === 1 ? "" : "s"}? This cannot be undone.`,
      confirmLabel: "Delete",
      cancelLabel: "Cancel",
      danger: true,
    });
    if (!confirmed) {
      return;
    }

    setIsBulkDeleting(true);
    try {
      const workflowIds = Array.from(selectedWorkflows);
      const deletedIds = await bulkDeleteWorkflows(project.id, workflowIds);
      const deletedSet = new Set(deletedIds);
      const remainingWorkflows = workflows.filter(
        (workflow) => !deletedSet.has(workflow.id),
      );
      setWorkflows(remainingWorkflows);
      toast.success(
        `Deleted ${deletedSet.size} workflow${deletedSet.size === 1 ? "" : "s"}`,
      );
      setSelectedWorkflows(new Set());
      setSelectionMode(false);
    } catch (error) {
      logger.error(
        "Failed to delete workflows",
        {
          component: "ProjectDetail",
          action: "handleBulkDelete",
          projectId: project.id,
        },
        error,
      );
      toast.error("Failed to delete selected workflows");
    } finally {
      setIsBulkDeleting(false);
    }
  }, [
    selectedWorkflows,
    project.id,
    bulkDeleteWorkflows,
    workflows,
    requestConfirm,
  ]);

  const handleDeleteProject = useCallback(async () => {
    const confirmed = await requestConfirm({
      title: "Delete project?",
      message:
        "Delete this project and all associated workflows? This cannot be undone.",
      confirmLabel: "Delete Project",
      cancelLabel: "Cancel",
      danger: true,
    });
    if (!confirmed) {
      return;
    }

    setIsDeletingProject(true);
    try {
      await deleteProject(project.id);
      toast.success("Project deleted successfully");
      onBack();
    } catch (error) {
      logger.error(
        "Failed to delete project",
        {
          component: "ProjectDetail",
          action: "handleDeleteProject",
          projectId: project.id,
        },
        error,
      );
      toast.error("Failed to delete project");
    } finally {
      setIsDeletingProject(false);
    }
  }, [project.id, deleteProject, onBack, requestConfirm]);

  const handleSelectExecution = useCallback(
    async (execution: { id: string }) => {
      try {
        setActiveTab("executions");
        await loadExecution(execution.id);
      } catch (error) {
        logger.error(
          "Failed to load execution details",
          {
            component: "ProjectDetail",
            action: "handleSelectExecution",
            executionId: execution.id,
          },
          error,
        );
        toast.error("Failed to load execution details");
      }
    },
    [loadExecution],
  );

  const handleCloseExecutionViewer = useCallback(() => {
    closeExecutionViewer();
  }, [closeExecutionViewer]);

  const toggleFolder = useCallback(
    (path: string) => {
      const newExpanded = new Set(expandedFolders);
      if (newExpanded.has(path)) {
        newExpanded.delete(path);
      } else {
        newExpanded.add(path);
      }
      setExpandedFolders(newExpanded);
    },
    [expandedFolders],
  );

  // Memoize computed values
  const workflowCount = useMemo(
    () => project.stats?.workflow_count ?? workflows.length,
    [project.stats?.workflow_count, workflows.length],
  );

  const totalExecutions = useMemo(
    () => project.stats?.execution_count ?? 0,
    [project.stats?.execution_count],
  );

  const lastExecutionLabel = useMemo(
    () =>
      project.stats?.last_execution
        ? formatDate(project.stats.last_execution)
        : "Never",
    [project.stats?.last_execution, formatDate],
  );

  const lastUpdatedLabel = useMemo(
    () => (project.updated_at ? formatDate(project.updated_at) : "Unknown"),
    [project.updated_at, formatDate],
  );

  const renderTreePrefix = (prefixParts: boolean[]) => {
    if (prefixParts.length === 0) {
      return null;
    }

    const prefix = prefixParts
      .map((hasSibling, index) => {
        const isLast = index === prefixParts.length - 1;
        if (isLast) {
          return hasSibling ? "├── " : "└── ";
        }
        return hasSibling ? "│   " : "    ";
      })
      .join("");

    return (
      <span
        aria-hidden="true"
        className="font-mono text-[11px] leading-4 text-gray-500 whitespace-pre pointer-events-none select-none"
      >
        {prefix}
      </span>
    );
  };

  const fileTypeLabelFromPath = (relPath: string): string | null => {
    const normalized = relPath.toLowerCase();
    if (normalized.endsWith(".action.json")) return "Action";
    if (normalized.endsWith(".flow.json")) return "Flow";
    if (normalized.endsWith(".case.json")) return "Case";
    return null;
  };

  const fileKindIcon = (node: FileTreeNode) => {
    if (node.kind === "folder") {
      return <FolderOpen size={14} className="text-yellow-500" />;
    }
    if (node.kind === "workflow_file") {
      return <FileCode size={14} className="text-green-400" />;
    }
    return <FileText size={14} className="text-gray-400" />;
  };

  const buildFileTree = useCallback((entries: ProjectEntry[]): FileTreeNode[] => {
    const folderMap = new Map<string, FileTreeNode>();

    const ensureFolder = (folderPath: string): FileTreeNode => {
      if (folderMap.has(folderPath)) {
        return folderMap.get(folderPath)!;
      }
      const name = folderPath === "" ? "root" : folderPath.split("/").pop()!;
      const node: FileTreeNode = {
        kind: "folder",
        path: folderPath,
        name,
        children: [],
      };
      folderMap.set(folderPath, node);
      return node;
    };

    // Always create root
    ensureFolder("");

    for (const entry of entries) {
      if (!entry?.path || typeof entry.path !== "string") continue;
      const relPath = entry.path.replace(/^\/+/, "");
      const parts = relPath.split("/").filter(Boolean);
      if (parts.length === 0) continue;
      const isDir = entry.kind === "folder";

      let current = "";
      for (let i = 0; i < parts.length - (isDir ? 0 : 1); i++) {
        current = current ? `${current}/${parts[i]}` : parts[i];
        ensureFolder(current);
      }

      if (isDir) {
        ensureFolder(relPath);
      } else {
        const parentPath = parts.length > 1 ? parts.slice(0, -1).join("/") : "";
        const parent = ensureFolder(parentPath);
        parent.children = parent.children ?? [];
        parent.children.push({
          kind: entry.kind,
          path: relPath,
          name: parts[parts.length - 1],
          workflowId: entry.workflow_id,
          metadata: entry.metadata,
        });
      }
    }

    // Attach folder children to parents
    const folders = Array.from(folderMap.values());
    folders.sort((a, b) => a.path.localeCompare(b.path));
    for (const folder of folders) {
      if (folder.path === "") continue;
      const parentPath = folder.path.includes("/")
        ? folder.path.split("/").slice(0, -1).join("/")
        : "";
      const parent = ensureFolder(parentPath);
      parent.children = parent.children ?? [];
      if (!parent.children.find((c) => c.kind === "folder" && c.path === folder.path)) {
        parent.children.push(folder);
      }
    }

    const sortChildren = (node: FileTreeNode) => {
      if (!node.children) return;
      node.children.sort((a, b) => {
        if (a.kind !== b.kind) {
          if (a.kind === "folder") return -1;
          if (b.kind === "folder") return 1;
          if (a.kind === "workflow_file") return -1;
          if (b.kind === "workflow_file") return 1;
        }
        return a.name.localeCompare(b.name);
      });
      node.children.forEach(sortChildren);
    };

    const root = ensureFolder("");
    sortChildren(root);
    return root.children ?? [];
  }, []);

  const memoizedFileTree = useMemo(() => {
    return buildFileTree(projectEntries);
  }, [buildFileTree, projectEntries]);

  const filteredFileTree = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    if (!term) {
      return memoizedFileTree;
    }

    const filterNodes = (nodes: FileTreeNode[]): FileTreeNode[] => {
      const out: FileTreeNode[] = [];
      for (const node of nodes) {
        const matches =
          node.name.toLowerCase().includes(term) ||
          node.path.toLowerCase().includes(term);
        if (node.kind === "folder") {
          const children = filterNodes(node.children ?? []);
          if (matches || children.length > 0) {
            out.push({ ...node, children });
          }
        } else if (matches) {
          out.push(node);
        }
      }
      return out;
    };

    return filterNodes(memoizedFileTree);
  }, [memoizedFileTree, searchTerm]);

  const resyncProjectFiles = useCallback(async () => {
    try {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/resync`,
        { method: "POST" },
      );
      if (!response.ok) {
        throw new Error(`Resync failed: ${response.status}`);
      }
      toast.success("Project files resynced");
      await fetchProjectEntries();
      await fetchWorkflows();
    } catch (error) {
      logger.error(
        "Failed to resync project files",
        { component: "ProjectDetail", action: "resyncProjectFiles", projectId: project.id },
        error,
      );
      toast.error(error instanceof Error ? error.message : "Failed to resync");
    }
  }, [project.id, fetchProjectEntries, fetchWorkflows]);

  const mkdirProjectPath = useCallback(
    async (relPath: string) => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/mkdir`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: relPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to create folder: ${response.status}`);
      }
    },
    [project.id],
  );

  const writeProjectWorkflowFile = useCallback(
    async (relPath: string, type: "action" | "flow" | "case", name?: string) => {
      const config = await getConfig();
      const inferredName =
        name?.trim() ||
        relPath
          .split("/")
          .pop()
          ?.replace(/\.action\.json$/i, "")
          .replace(/\.flow\.json$/i, "")
          .replace(/\.case\.json$/i, "") ||
        "workflow";
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/write`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            path: relPath,
            workflow: {
              name: inferredName,
              type,
              flow_definition: { nodes: [], edges: [] },
            },
          }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to create workflow file: ${response.status}`);
      }
      const payload = (await response.json()) as {
        workflowId?: string;
        warnings?: string[];
      };
      if (payload.warnings && payload.warnings.length > 0) {
        toast(payload.warnings[0] ?? "Created with warnings");
      }
      return payload.workflowId;
    },
    [project.id],
  );

  const moveProjectFile = useCallback(
    async (fromPath: string, toPath: string) => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/move`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ from_path: fromPath, to_path: toPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to move: ${response.status}`);
      }
    },
    [project.id],
  );

  const deleteProjectFile = useCallback(
    async (relPath: string) => {
      const config = await getConfig();
      const response = await fetch(
        `${config.API_URL}/projects/${project.id}/files/delete`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ path: relPath }),
        },
      );
      if (!response.ok) {
        const msg = await response.text();
        throw new Error(msg || `Failed to delete: ${response.status}`);
      }
    },
    [project.id],
  );

  const handleDropMove = useCallback(
    async (e: React.DragEvent, targetFolderPath: string) => {
      e.preventDefault();
      e.stopPropagation();
      const payload = parseDragPayload(e.dataTransfer);
      setDropTargetFolder(null);
      setDragSourcePath(null);
      if (!payload) return;

      const sourcePath = payload.path;
      const baseName = fileBasename(sourcePath);
      if (!baseName) return;

      const destPath = targetFolderPath ? `${targetFolderPath}/${baseName}` : baseName;
      if (destPath === sourcePath) return;

      if (payload.kind === "folder") {
        if (targetFolderPath === sourcePath || targetFolderPath.startsWith(`${sourcePath}/`)) {
          toast.error("Cannot move a folder into itself.");
          return;
        }
      }

      try {
        await moveProjectFile(sourcePath, destPath);
        toast.success("Moved");
        await fetchProjectEntries();
        await fetchWorkflows();
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Move failed");
      }
    },
    [fetchProjectEntries, fetchWorkflows, fileBasename, moveProjectFile, parseDragPayload],
  );

  const FileTreeItem = ({
    node,
    prefixParts = [],
  }: {
    node: FileTreeNode;
    prefixParts?: boolean[];
  }) => {
    const isFolder = node.kind === "folder";
    const isExpanded = isFolder && expandedFolders.has(node.path);
    const children = node.children ?? [];
    const hasChildren = isFolder && children.length > 0;
    const typeLabel = node.kind === "workflow_file" ? fileTypeLabelFromPath(node.path) : null;
    const isDropTarget = Boolean(
      dragSourcePath !== null && dropTargetFolder !== null && dropTargetFolder === node.path,
    );

    const handleToggle = () => {
      if (!isFolder) return;
      toggleFolder(node.path);
    };

    const handleOpenWorkflowFile = async () => {
      if (node.kind !== "workflow_file" || !node.workflowId) {
        return;
      }
      try {
        await loadWorkflow(node.workflowId);
        const wf = useWorkflowStore.getState().currentWorkflow;
        if (wf) {
          await onWorkflowSelect(wf);
        }
      } catch (err) {
        toast.error("Failed to open workflow");
      }
    };

    const handleDeleteNode = async (e: React.MouseEvent) => {
      e.stopPropagation();
      const confirmed = await requestConfirm({
        title: "Delete?",
        message: `Delete "${node.path}"? This cannot be undone.`,
        confirmLabel: "Delete",
        cancelLabel: "Cancel",
        danger: true,
      });
      if (!confirmed) return;
      try {
        await deleteProjectFile(node.path);
        toast.success("Deleted");
        await fetchProjectEntries();
        await fetchWorkflows();
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Delete failed");
      }
    };

    const handleRenameNode = async (e: React.MouseEvent) => {
      e.stopPropagation();
      const next = await requestPrompt(
        {
          title: "Move / Rename",
          label: "New path (relative to project root)",
          defaultValue: node.path,
          submitLabel: "Move",
          cancelLabel: "Cancel",
        },
        {
          validate: (value) => {
            const normalized = normalizeProjectRelPathClient(value);
            if (!normalized.ok) return normalized.error;
            if (normalized.path === node.path) return "Path must be different.";
            return null;
          },
          normalize: (value) => {
            const normalized = normalizeProjectRelPathClient(value);
            return normalized.ok ? normalized.path : value.trim();
          },
        },
      );
      if (!next) return;
      try {
        await moveProjectFile(node.path, next);
        toast.success("Moved");
        await fetchProjectEntries();
        await fetchWorkflows();
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Move failed");
      }
    };

    const handleExecuteFromTree = async (e: React.MouseEvent) => {
      if (node.kind !== "workflow_file" || !node.workflowId) {
        e.stopPropagation();
        return;
      }
      await handleExecuteWorkflow(e, node.workflowId);
    };

    return (
      <div className="select-none">
        <div
          draggable={node.path !== ""}
          onDragStart={(e) => {
            e.stopPropagation();
            e.dataTransfer.effectAllowed = "move";
            e.dataTransfer.setData(
              "application/x-bas-project-entry",
              JSON.stringify({ path: node.path, kind: node.kind } as FileTreeDragPayload),
            );
            setDragSourcePath(node.path);
          }}
          onDragEnd={() => {
            setDragSourcePath(null);
            setDropTargetFolder(null);
          }}
          onDragOver={(e) => {
            if (!isFolder) return;
            e.preventDefault();
            e.stopPropagation();
            e.dataTransfer.dropEffect = "move";
            setDropTargetFolder(node.path);
          }}
          onDrop={(e) => {
            if (!isFolder) return;
            void handleDropMove(e, node.path);
          }}
          className={`flex items-center gap-1.5 px-2 py-1.5 rounded cursor-pointer transition-colors group ${
            isFolder && selectedTreeFolder === node.path ? "bg-flow-node" : "hover:bg-flow-node"
          } ${isDropTarget ? "ring-1 ring-flow-accent bg-flow-node" : ""}`}
          onClick={() => {
            if (isFolder) {
              setSelectedTreeFolder(node.path);
              handleToggle();
              return;
            }
            if (node.kind === "workflow_file") {
              void handleOpenWorkflowFile();
              return;
            }
            toast("Assets are read-only in v1");
          }}
        >
          {renderTreePrefix(prefixParts)}
          {isFolder ? (
            hasChildren ? (
              isExpanded ? (
                <ChevronDown size={14} className="text-gray-400" />
              ) : (
                <ChevronRight size={14} className="text-gray-400" />
              )
            ) : (
              <span className="inline-block w-3.5" aria-hidden="true" />
            )
          ) : (
            <span className="inline-block w-3.5" aria-hidden="true" />
          )}
          {fileKindIcon(node)}
          <span className="text-sm text-gray-300">
            {node.name}
            {typeLabel ? (
              <span className="ml-2 text-[11px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
                {typeLabel}
              </span>
            ) : null}
          </span>

          <div className="ml-auto flex items-center gap-2 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity">
            {node.kind === "workflow_file" && node.workflowId ? (
              <button
                onClick={handleExecuteFromTree}
                className="text-subtle hover:text-surface"
                title={typeLabel === "Case" ? "Test" : "Run"}
                aria-label={typeLabel === "Case" ? "Test workflow" : "Run workflow"}
              >
                {typeLabel === "Case" ? <ListChecks size={14} /> : <Play size={14} />}
              </button>
            ) : null}
            <button
              onClick={handleRenameNode}
              className="text-subtle hover:text-surface"
              title="Rename / Move"
              aria-label="Rename / Move"
            >
              <PencilLine size={14} />
            </button>
            <button
              onClick={handleDeleteNode}
              className="text-subtle hover:text-red-300"
              title="Delete"
              aria-label="Delete"
            >
              <Trash2 size={14} />
            </button>
          </div>
        </div>

        {isFolder &&
          isExpanded &&
          children.map((child, index) => {
            const isLastChild = index === children.length - 1;
            const childPrefix = [...prefixParts, !isLastChild];
            return (
              <FileTreeItem
                key={`${child.kind}:${child.path}`}
                node={child}
                prefixParts={childPrefix}
              />
            );
          })}
      </div>
    );
  };

  // Status bar for API connection issues
  const StatusBar = () => {
    if (!error) return null;

    return (
      <div className="bg-red-900/20 border-l-4 border-red-500 p-4 mx-6 mb-4 rounded-r-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <WifiOff size={20} className="text-red-400" />
            <div>
              <div className="text-red-400 font-medium">
                API Connection Failed
              </div>
              <div className="text-red-300/80 text-sm">{error}</div>
            </div>
          </div>
          <button
            onClick={() => {
              setError(null);
              fetchWorkflows();
            }}
            className="px-3 py-1 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  };

  return (
    <>
      <div className="flex-1 flex flex-col h-full overflow-hidden min-h-0">
        <input
          ref={fileInputRef}
          type="file"
          accept=".zip"
          className="hidden"
          onChange={handleRecordingImport}
        />
        {/* Header */}
        <div className="relative px-6 pt-6 border-b border-gray-800 space-y-3 md:space-y-6">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="flex items-start gap-4">
              <button
                onClick={onBack}
                className="p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
              >
                <ArrowLeft size={20} />
              </button>
              <div>
              <div className="flex flex-col md:flex-row md:items-center gap-2 mb-1">
                  <h1 className="text-2xl font-bold text-surface">
                    {project.name}
                  </h1>
                  <div className="flex items-center gap-2">
                    {/* Info Button */}
                    <div className="relative">
                      <button
                        ref={statsButtonRef}
                        type="button"
                        onClick={() => setShowStatsPopover((prev) => !prev)}
                        className="p-1.5 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
                        aria-label="Project details"
                        aria-expanded={showStatsPopover}
                      >
                        <Info size={16} />
                      </button>
                      {showStatsPopover && (
                        <div
                          ref={statsPopoverRef}
                          style={statsPopoverStyles}
                          className="z-30 w-80 rounded-lg border border-gray-700 bg-flow-node p-4 shadow-lg"
                        >
                          <div className="flex items-center justify-between mb-3">
                            <h3 className="text-sm font-semibold text-surface">
                              Project Details
                            </h3>
                            <button
                              type="button"
                              onClick={() => setShowStatsPopover(false)}
                              className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
                              aria-label="Close project details"
                            >
                              <X size={14} />
                            </button>
                          </div>

                          {/* Project Info Section */}
                          <div className="mb-4 pb-4 border-b border-gray-700">
                            <div className="mb-3">
                              <dt className="text-xs text-gray-400 mb-1">
                                Project Name
                              </dt>
                              <dd className="text-sm font-medium text-surface">
                                {project.name}
                              </dd>
                            </div>
                            <div className="mb-3">
                              <dt className="text-xs text-gray-400 mb-1">
                                Description
                              </dt>
                              <dd className="text-sm text-gray-300">
                                {project.description?.trim()
                                  ? project.description
                                  : "No description"}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-400 mb-1">
                                Save Path
                              </dt>
                              <dd className="text-sm text-gray-300 font-mono break-all">
                                {project.folder_path}
                              </dd>
                            </div>
                          </div>

                          {/* Metrics Section */}
                          <div className="mb-3">
                            <h4 className="text-xs font-semibold text-gray-400 mb-2">
                              Metrics
                            </h4>
                            <dl className="space-y-2 text-sm text-gray-300">
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">Workflows</dt>
                                <dd className="font-medium text-surface">
                                  {workflowCount}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">
                                  Total executions
                                </dt>
                                <dd className="font-medium text-surface">
                                  {totalExecutions}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">
                                  Last execution
                                </dt>
                                <dd className="font-medium text-surface">
                                  {lastExecutionLabel}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">Last updated</dt>
                                <dd className="font-medium text-surface">
                                  {lastUpdatedLabel}
                                </dd>
                              </div>
                            </dl>
                          </div>
                          <p className="text-xs text-gray-500">
                            Metrics refresh automatically as your workflows run.
                          </p>
                        </div>
                      )}
                    </div>

                    {/* Edit Button */}
                    <button
                      data-testid={selectors.projects.editButton}
                      onClick={() => setShowEditProjectModal(true)}
                      className="p-1.5 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
                      title="Edit project details"
                    >
                      <PencilLine size={16} />
                    </button>

                    {/* More Menu Button */}
                    <div className="relative">
                      <button
                        ref={moreMenuButtonRef}
                        onClick={() => setShowMoreMenu((prev) => !prev)}
                        className="p-1.5 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
                        aria-label="More options"
                        aria-expanded={showMoreMenu}
                      >
                        <MoreVertical size={16} />
                      </button>
                      {showMoreMenu && (
                        <div
                          ref={moreMenuRef}
                          style={moreMenuStyles}
                          className="z-30 w-56 rounded-lg border border-gray-700 bg-flow-node shadow-lg overflow-hidden"
                        >
                          <button
                            onClick={() => {
                              toggleSelectionMode();
                              setShowMoreMenu(false);
                            }}
                            disabled={workflows.length === 0}
                            className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-left"
                          >
                            <ListChecks size={16} />
                            <span className="text-sm">Manage Workflows</span>
                          </button>
                          {onStartRecording && (
                            <button
                              onClick={() => {
                                onStartRecording();
                                setShowMoreMenu(false);
                              }}
                              className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left"
                            >
                              <Circle size={16} className="text-red-500 fill-red-500" />
                              <span className="text-sm">Record Actions</span>
                            </button>
                          )}
                          <button
                            onClick={() => {
                              handleRecordingImportClick();
                              setShowMoreMenu(false);
                            }}
                            disabled={isImportingRecording}
                            className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-left"
                          >
                            {isImportingRecording ? (
                              <Loader size={16} className="animate-spin" />
                            ) : (
                              <UploadCloud size={16} />
                            )}
                            <span className="text-sm">
                              {isImportingRecording
                                ? "Importing..."
                                : "Import Recording"}
                            </span>
                          </button>
                          <button
                            onClick={() => {
                              handleDeleteProject();
                              setShowMoreMenu(false);
                            }}
                            disabled={isDeletingProject}
                            className="w-full flex items-center gap-3 px-4 py-3 text-red-300 hover:bg-red-500/10 transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-left border-t border-gray-700"
                          >
                            {isDeletingProject ? (
                              <Loader size={16} className="animate-spin" />
                            ) : (
                              <Trash2 size={16} />
                            )}
                            <span className="text-sm">
                              {isDeletingProject
                                ? "Deleting..."
                                : "Delete Project"}
                            </span>
                          </button>
                        </div>
                      )}
                    </div>
                    {viewMode === "tree" ? (
                      <div className="relative md:ml-auto">
                        <button
                          ref={newFileMenuButtonRef}
                          data-testid={selectors.workflows.newButton}
                          onClick={() => setShowNewFileMenu((prev) => !prev)}
                          className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
                        >
                          <Plus size={16} />
                          <span className="hidden sm:inline">New</span>
                        </button>
                        {showNewFileMenu && (
                          <div
                            ref={newFileMenuRef}
                            style={newFileMenuStyles}
                            className="z-30 w-56 rounded-lg border border-gray-700 bg-flow-node shadow-lg overflow-hidden mt-2"
                          >
                            <button
                              onClick={async () => {
                                setShowNewFileMenu(false);
                                const suggested =
                                  selectedTreeFolder === null
                                    ? "actions"
                                    : selectedTreeFolder === ""
                                      ? "new-folder"
                                      : `${selectedTreeFolder}/new-folder`;
                                const relPath = await requestPrompt(
                                  {
                                    title: "New Folder",
                                    label: "Folder path (relative to project root)",
                                    defaultValue: suggested,
                                    submitLabel: "Create Folder",
                                    cancelLabel: "Cancel",
                                  },
                                  {
                                    validate: (value) => {
                                      const normalized =
                                        normalizeProjectRelPathClient(value);
                                      if (!normalized.ok) return normalized.error;
                                      return null;
                                    },
                                    normalize: (value) => {
                                      const normalized =
                                        normalizeProjectRelPathClient(value);
                                      return normalized.ok
                                        ? normalized.path
                                        : value.trim();
                                    },
                                  },
                                );
                                if (!relPath) return;
                                try {
                                  await mkdirProjectPath(relPath);
                                  toast.success("Folder created");
                                  await fetchProjectEntries();
                                } catch (err) {
                                  toast.error(
                                    err instanceof Error
                                      ? err.message
                                      : "Failed to create folder",
                                  );
                                }
                              }}
                              className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left"
                            >
                              <FolderOpen size={16} />
                              <span className="text-sm">New Folder</span>
                            </button>
                            {[
                              {
                                label: "New Action",
                                type: "action",
                                suffix: ".action.json",
                              },
                              { label: "New Flow", type: "flow", suffix: ".flow.json" },
                              { label: "New Case", type: "case", suffix: ".case.json" },
                            ].map((opt) => (
                              <button
                                key={opt.type}
                                onClick={async () => {
                                  setShowNewFileMenu(false);
                                  const defaultDir =
                                    selectedTreeFolder !== null
                                      ? selectedTreeFolder
                                      : opt.type === "action"
                                        ? "actions"
                                        : opt.type === "case"
                                          ? "cases"
                                          : "flows";
                                  const suggested =
                                    defaultDir === ""
                                      ? `${opt.type}-example${opt.suffix}`
                                      : `${defaultDir}/${opt.type}-example${opt.suffix}`;
                                  const relPath = await requestPrompt(
                                    {
                                      title: opt.label,
                                      label: `File path (relative to project root)`,
                                      defaultValue: suggested,
                                      submitLabel: "Create",
                                      cancelLabel: "Cancel",
                                    },
                                    {
                                      validate: (value) => {
                                        const normalized =
                                          normalizeProjectRelPathClient(value);
                                        if (!normalized.ok) return normalized.error;
                                        const lower = normalized.path.toLowerCase();
                                        const typedSuffixes = [
                                          ".action.json",
                                          ".flow.json",
                                          ".case.json",
                                        ];
                                        const hasTypedSuffix = typedSuffixes.some((s) =>
                                          lower.endsWith(s),
                                        );
                                        if (hasTypedSuffix && !lower.endsWith(opt.suffix)) {
                                          return `File extension must be ${opt.suffix}`;
                                        }
                                        return null;
                                      },
                                      normalize: (value) => {
                                        const normalized =
                                          normalizeProjectRelPathClient(value);
                                        if (!normalized.ok) return value.trim();
                                        const lower = normalized.path.toLowerCase();
                                        if (lower.endsWith(opt.suffix)) return normalized.path;
                                        if (
                                          lower.endsWith(".json") &&
                                          !lower.endsWith(".action.json") &&
                                          !lower.endsWith(".flow.json") &&
                                          !lower.endsWith(".case.json")
                                        ) {
                                          return `${normalized.path.slice(0, -".json".length)}${opt.suffix}`;
                                        }
                                        return `${normalized.path}${opt.suffix}`;
                                      },
                                    },
                                  );
                                  if (!relPath) return;
                                  try {
                                    const workflowId = await writeProjectWorkflowFile(
                                      relPath,
                                      opt.type as any,
                                    );
                                    toast.success(`${opt.label} created`);
                                    await fetchProjectEntries();
                                    await fetchWorkflows();
                                    if (workflowId) {
                                      await loadWorkflow(workflowId);
                                      const wf =
                                        useWorkflowStore.getState().currentWorkflow;
                                      if (wf) {
                                        await onWorkflowSelect(wf);
                                      }
                                    }
                                  } catch (err) {
                                    toast.error(
                                      err instanceof Error
                                        ? err.message
                                        : `Failed to create ${opt.type}`,
                                    );
                                  }
                                }}
                                className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left"
                              >
                                <FileCode size={16} />
                                <span className="text-sm">{opt.label}</span>
                              </button>
                            ))}
                            <button
                              onClick={async () => {
                                setShowNewFileMenu(false);
                                const confirmed = await requestConfirm({
                                  title: "Resync from disk?",
                                  message:
                                    "This will rebuild the project file index from disk and may repair workflow files (id/type/version).",
                                  confirmLabel: "Resync",
                                  cancelLabel: "Cancel",
                                });
                                if (!confirmed) return;
                                await resyncProjectFiles();
                              }}
                              className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left border-t border-gray-700"
                            >
                              <RefreshCw size={16} />
                              <span className="text-sm">Resync From Disk</span>
                            </button>
                          </div>
                        )}
                      </div>
                    ) : (
                      <button
                        data-testid={selectors.workflows.newButton}
                        onClick={onCreateWorkflow}
                        className="hidden md:flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors md:ml-auto"
                      >
                        <Plus size={16} />
                        <span>New Workflow</span>
                      </button>
                    )}
                  </div>
                </div>
                <p className="hidden md:block text-gray-400">
                  {project.description?.trim()
                    ? project.description
                    : "Add a description to keep collaborators aligned."}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center justify-end gap-2">
              {selectionMode && (
                <>
                  <button
                    onClick={toggleSelectionMode}
                    className="flex items-center gap-2 px-3 py-1.5 md:rounded-lg rounded-full border border-flow-accent text-flow-accent transition-colors"
                    title="Done"
                  >
                    <X size={14} />
                    <span className="hidden md:inline">Done</span>
                  </button>
                  <button
                    onClick={handleSelectAll}
                    className="flex items-center gap-2 px-3 py-1.5 md:rounded-lg rounded-full border border-gray-700 text-subtle hover:border-flow-accent hover:text-surface transition-colors"
                    title={
                      selectedWorkflows.size === filteredWorkflows.length &&
                      filteredWorkflows.length > 0
                        ? "Clear Selection"
                        : "Select All"
                    }
                  >
                    {selectedWorkflows.size === filteredWorkflows.length &&
                    filteredWorkflows.length > 0 ? (
                      <CheckSquare size={14} />
                    ) : (
                      <Square size={14} />
                    )}
                    <span className="hidden md:inline">
                      {selectedWorkflows.size === filteredWorkflows.length &&
                      filteredWorkflows.length > 0
                        ? "Clear Selection"
                        : "Select All"}
                    </span>
                  </button>
                  <button
                    onClick={handleBulkDeleteSelected}
                    disabled={selectedWorkflows.size === 0 || isBulkDeleting}
                    className={`flex items-center gap-2 px-3 py-1.5 md:rounded-lg rounded-full transition-colors ${
                      selectedWorkflows.size === 0
                        ? "bg-red-500/20 text-red-300 opacity-60 cursor-not-allowed"
                        : "bg-red-500/10 text-red-300 hover:bg-red-500/20"
                    }`}
                    title="Delete Selected"
                  >
                    {isBulkDeleting ? (
                      <Loader size={14} className="animate-spin" />
                    ) : (
                      <Trash2 size={14} />
                    )}
                    <span className="hidden md:inline">Delete Selected</span>
                  </button>
                </>
              )}
            </div>
          </div>

          <div>
            <div className="flex items-center gap-3 border-b border-gray-700 pb-2">
              <button
                data-testid={selectors.workflows.tab}
                onClick={() => setActiveTab("workflows")}
                role="tab"
                aria-selected={activeTab === "workflows"}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "workflows"
                    ? "border-flow-accent text-surface"
                    : "border-transparent text-subtle hover:text-surface"
                }`}
              >
                <div className="flex items-center gap-2">
                  <FileCode size={16} />
                  <span className="whitespace-nowrap">
                    Workflows ({workflows.length})
                  </span>
                </div>
              </button>
              <button
                data-testid={selectors.projects.tabs.executions}
                onClick={() => setActiveTab("executions")}
                role="tab"
                aria-selected={activeTab === "executions"}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === "executions"
                    ? "border-flow-accent text-surface"
                    : "border-transparent text-subtle hover:text-surface"
                }`}
              >
                <div className="flex items-center gap-2">
                  <History size={16} />
                  <span className="whitespace-nowrap">Execution History</span>
                </div>
              </button>
            </div>

            {/* Search and View Mode Toggle - only show for workflows tab */}
            {activeTab === "workflows" && (
              <div className="mt-0 md:mt-4 flex items-center gap-3">
                <div className="relative flex-1">
                  <Search
                    size={16}
                    className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
                  />
                  <input
                    type="text"
                    data-testid={selectors.workflowBuilder.search.input}
                    placeholder="Search workflows..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-surface placeholder-gray-500 focus:outline-none focus:border-flow-accent"
                  />
                  {searchTerm && (
                    <button
                      type="button"
                      data-testid={selectors.workflowBuilder.search.clearButton}
                      onClick={() => setSearchTerm("")}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300 transition-colors"
                      aria-label="Clear workflow search"
                    >
                      <X size={16} />
                    </button>
                  )}
                </div>
                {/* Desktop: Toggle buttons */}
                <div className="hidden md:flex items-center gap-2 bg-flow-node border border-gray-700 rounded-lg p-1">
                  <button
                    onClick={() => setViewMode("card")}
                    className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                      viewMode === "card"
                        ? "bg-flow-accent text-white"
                        : "text-subtle hover:text-surface"
                    }`}
                    title="Card View"
                  >
                    <LayoutGrid size={16} />
                    <span className="text-sm">Cards</span>
                  </button>
                  <button
                    onClick={() => setViewMode("tree")}
                    className={`px-3 py-1.5 rounded flex items-center gap-2 transition-colors ${
                      viewMode === "tree"
                        ? "bg-flow-accent text-white"
                        : "text-subtle hover:text-surface"
                    }`}
                    title="File Tree View"
                  >
                    <FolderTree size={16} />
                    <span className="text-sm">Files</span>
                  </button>
                </div>
                {/* Mobile: Icon button with popover */}
                <div className="md:hidden relative">
                  <button
                    ref={viewModeButtonRef}
                    onClick={() =>
                      setShowViewModeDropdown(!showViewModeDropdown)
                    }
                    className="p-2 bg-flow-node border border-gray-700 rounded-lg text-subtle hover:border-flow-accent hover:text-surface transition-colors"
                    title={viewMode === "card" ? "Card View" : "File Tree"}
                  >
                    {viewMode === "card" ? (
                      <LayoutGrid size={20} />
                    ) : (
                      <FolderTree size={20} />
                    )}
                  </button>
                  {showViewModeDropdown && (
                    <div
                      ref={viewModeDropdownRef}
                      style={viewModeDropdownStyles}
                      className="z-30 w-48 rounded-lg border border-gray-700 bg-flow-node shadow-lg overflow-hidden"
                    >
                      <button
                        onClick={() => {
                          setViewMode("card");
                          setShowViewModeDropdown(false);
                        }}
                        className={`w-full flex items-center gap-3 px-4 py-3 transition-colors ${
                          viewMode === "card"
                            ? "bg-flow-accent text-white"
                            : "text-subtle hover:bg-flow-node-hover hover:text-surface"
                        }`}
                      >
                        <LayoutGrid size={16} />
                        <span className="text-sm">Card View</span>
                      </button>
                      <button
                        onClick={() => {
                          setViewMode("tree");
                          setShowViewModeDropdown(false);
                        }}
                        className={`w-full flex items-center gap-3 px-4 py-3 transition-colors ${
                          viewMode === "tree"
                            ? "bg-flow-accent text-white"
                            : "text-subtle hover:bg-flow-node-hover hover:text-surface"
                        }`}
                      >
                        <FolderTree size={16} />
                        <span className="text-sm">File Tree</span>
                      </button>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Status Bar for API errors */}
        <StatusBar />

        {/* Content */}
        <div className="flex-1 overflow-hidden min-h-0">
          {activeTab === "executions" ? (
            <div className="h-full flex flex-col md:flex-row min-h-0">
              <div
                className={`${
                  isExecutionViewerOpen
                    ? "hidden md:block md:w-1/2 md:border-r md:border-gray-800"
                    : "block md:w-full"
                } flex-1 min-h-0`}
              >
                <ExecutionHistory onSelectExecution={handleSelectExecution} />
              </div>
              {isExecutionViewerOpen && currentExecution && (
                <div className="w-full md:w-1/2 flex-1 flex flex-col min-h-0">
                  <ExecutionViewer
                    workflowId={currentExecution.workflowId}
                    execution={currentExecution}
                    onClose={handleCloseExecutionViewer}
                  />
                </div>
              )}
            </div>
          ) : (
            <div className="flex-1 overflow-auto p-6">
              {isLoading ? (
                <div className="flex items-center justify-center h-64">
                  <div className="text-gray-400">Loading workflows...</div>
                </div>
              ) : filteredWorkflows.length === 0 && searchTerm === "" ? (
                <div className="max-w-2xl mx-auto pt-8">
                  <div className="text-center mb-8">
                    <div className="mb-5 flex items-center justify-center">
                      <div className="p-4 bg-green-500/20 rounded-2xl">
                        {error ? <WifiOff size={40} className="text-red-400" /> : <FileCode size={40} className="text-green-400" />}
                      </div>
                    </div>
                    <h3 className="text-xl font-semibold text-surface mb-2">
                      {error ? "Unable to Load Workflows" : "Ready to Automate"}
                    </h3>
                    <p className="text-gray-400 max-w-md mx-auto">
                      {error
                        ? "There was an issue connecting to the API. You can still use the interface when the connection is restored."
                        : "Create your first workflow to automate browser tasks. Use AI to describe what you want, or build visually with the drag-and-drop builder."}
                    </p>
                  </div>

                  {!error && (
                    <>
                      {/* Quick Actions */}
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-8">
                        <button
                          onClick={onCreateWorkflow}
                          className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/10 transition-all group"
                        >
                          <div className="flex items-center gap-3 mb-3">
                            <div className="p-2 bg-amber-500/20 rounded-lg group-hover:bg-amber-500/30 transition-colors">
                              <Plus size={20} className="text-amber-400" />
                            </div>
                            <h4 className="font-medium text-surface">AI-Assisted</h4>
                          </div>
                          <p className="text-sm text-gray-400">
                            Describe your automation in plain language and let AI generate the workflow.
                          </p>
                        </button>

                        <button
                          onClick={onCreateWorkflowDirect ?? onCreateWorkflow}
                          className="bg-flow-node border border-gray-700 rounded-xl p-5 text-left hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/10 transition-all group"
                        >
                          <div className="flex items-center gap-3 mb-3">
                            <div className="p-2 bg-blue-500/20 rounded-lg group-hover:bg-blue-500/30 transition-colors">
                              <FolderTree size={20} className="text-blue-400" />
                            </div>
                            <h4 className="font-medium text-surface">Visual Builder</h4>
                          </div>
                          <p className="text-sm text-gray-400">
                            Use the drag-and-drop interface to build workflows step by step.
                          </p>
                        </button>
                      </div>

                      {/* Workflow ideas */}
                      <div className="text-center text-sm text-gray-500">
                        <p className="mb-2">Try automating:</p>
                        <div className="flex flex-wrap justify-center gap-2">
                          <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Form submissions</span>
                          <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">UI testing</span>
                          <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Data extraction</span>
                          <span className="px-2 py-1 bg-gray-800 rounded text-gray-400">Screenshots</span>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              ) : filteredWorkflows.length === 0 ? (
                <div className="flex items-center justify-center h-64 animate-fade-in">
                  <div className="text-center max-w-sm">
                    <div className="mb-4 flex items-center justify-center">
                      <div className="w-16 h-16 rounded-full bg-gray-800 flex items-center justify-center">
                        <Search size={28} className="text-gray-500" />
                      </div>
                    </div>
                    <h3 className="text-lg font-semibold text-surface mb-2">
                      No Workflows Found
                    </h3>
                    <p className="text-gray-400 mb-4">
                      No workflows match &ldquo;<span className="text-gray-300">{searchTerm}</span>&rdquo;
                    </p>
                    <button
                      onClick={() => setSearchTerm("")}
                      className="text-flow-accent hover:text-blue-400 text-sm font-medium transition-colors"
                    >
                      Clear search
                    </button>
                  </div>
                </div>
              ) : viewMode === "tree" ? (
                <div className="bg-flow-node border border-gray-700 rounded-lg p-4">
                  {projectEntriesLoading ? (
                    <div className="text-center text-gray-400 py-8">
                      <Loader size={24} className="mx-auto mb-3 animate-spin" />
                      <p>Loading project files…</p>
                    </div>
                  ) : projectEntriesError ? (
                    <div className="text-center text-gray-400 py-8">
                      <WifiOff size={40} className="mx-auto mb-3 opacity-50" />
                      <p className="mb-3">{projectEntriesError}</p>
                      <button
                        onClick={fetchProjectEntries}
                        className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-flow-accent text-white hover:bg-blue-600 transition-colors"
                      >
                        <RefreshCw size={16} />
                        <span>Retry</span>
                      </button>
                    </div>
                  ) : memoizedFileTree.length === 0 ? (
                    <div className="text-center text-gray-400 py-8">
                      <FolderTree size={48} className="mx-auto mb-4 opacity-50" />
                      <p>No files indexed yet.</p>
                      <p className="text-sm text-gray-500 mt-2">
                        Use “New” to create folders/files, or resync from disk.
                      </p>
                    </div>
                  ) : filteredFileTree.length === 0 ? (
                    <div className="text-center text-gray-400 py-8">
                      <FolderTree size={48} className="mx-auto mb-4 opacity-50" />
                      <p>No files match &ldquo;{searchTerm}&rdquo;.</p>
                      <button
                        onClick={() => setSearchTerm("")}
                        className="mt-3 text-flow-accent hover:text-blue-400 text-sm font-medium transition-colors"
                      >
                        Clear search
                      </button>
                    </div>
                  ) : (
                    <div className="space-y-1">
                      <div
                        className={`flex items-center gap-1.5 px-2 py-1.5 rounded cursor-pointer transition-colors ${
                          selectedTreeFolder === "" ? "bg-flow-node" : "hover:bg-flow-node"
                        } ${
                          dragSourcePath !== null && dropTargetFolder === ""
                            ? "ring-1 ring-flow-accent bg-flow-node"
                            : ""
                        }`}
                        onClick={() => setSelectedTreeFolder("")}
                        onDragOver={(e) => {
                          e.preventDefault();
                          e.stopPropagation();
                          e.dataTransfer.dropEffect = "move";
                          setDropTargetFolder("");
                        }}
                        onDrop={(e) => {
                          void handleDropMove(e, "");
                        }}
                      >
                        <FolderOpen size={14} className="text-yellow-500" />
                        <span className="text-sm text-gray-300">{project.name}</span>
                        <span className="ml-2 text-[11px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
                          root
                        </span>
                      </div>
                      {filteredFileTree.map((node, index) => {
                        const hasNextRoot = index < filteredFileTree.length - 1;
                        const rootPrefix = filteredFileTree.length > 1 ? [hasNextRoot] : [];
                        return (
                          <FileTreeItem
                            key={`${node.kind}:${node.path}`}
                            node={node}
                            prefixParts={rootPrefix}
                          />
                        );
                      })}
                    </div>
                  )}
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                  {filteredWorkflows.map((workflow: WorkflowWithStats) => {
                    const isSelected = selectedWorkflows.has(workflow.id);
                    const isDeleting = deletingWorkflowId === workflow.id;
                    const isActionsOpen = showWorkflowActionsFor === workflow.id;
                    const executionCount = workflow.stats?.execution_count || 0;
                    const successRate = workflow.stats?.success_rate;

                    const handleCardClick = async () => {
                      if (isDeleting) return;
                      if (selectionMode) {
                        toggleWorkflowSelection(workflow.id);
                      } else {
                        await onWorkflowSelect(workflow);
                      }
                    };

                    return (
                      <div
                        key={workflow.id}
                        data-testid={selectors.workflows.card}
                        data-workflow-id={workflow.id}
                        data-workflow-name={workflow.name}
                        onClick={handleCardClick}
                        className={`group relative bg-flow-node border rounded-xl p-5 cursor-pointer transition-all ${
                          isDeleting
                            ? "opacity-50 pointer-events-none"
                            : selectionMode
                              ? isSelected
                                ? "border-flow-accent shadow-lg shadow-blue-500/20"
                                : "border-gray-700 hover:border-flow-accent/60"
                              : "border-gray-700 hover:border-flow-accent/60 hover:shadow-lg hover:shadow-blue-500/10"
                        }`}
                      >
                        {/* Deleting Overlay */}
                        {isDeleting && (
                          <div className="absolute inset-0 bg-flow-node/80 rounded-xl flex items-center justify-center z-10">
                            <Loader size={24} className="animate-spin text-red-400" />
                          </div>
                        )}

                        {/* Workflow Header */}
                        <div
                          className="flex items-start justify-between gap-3 mb-3"
                          data-workflow-header
                        >
                          <div className="flex items-center gap-3 min-w-0">
                            <div
                              className={`flex-shrink-0 w-10 h-10 rounded-xl flex items-center justify-center border ${
                                selectionMode && isSelected
                                  ? "bg-flow-accent/20 border-flow-accent/30"
                                  : "bg-gradient-to-br from-green-500/20 to-emerald-500/10 border-green-500/20"
                              }`}
                            >
                              <FileCode size={18} className="text-green-400" />
                            </div>
                            <div className="min-w-0">
                              <h3
                                className="font-semibold text-surface truncate"
                                title={String(workflow.name)}
                              >
                                {String(workflow.name)}
                              </h3>
                              <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
                                <span>{executionCount} run{executionCount !== 1 ? 's' : ''}</span>
                                {successRate != null && (
                                  <>
                                    <span>•</span>
                                    <span className={successRate >= 80 ? "text-green-500" : successRate >= 50 ? "text-amber-500" : "text-red-400"}>
                                      {successRate}% success
                                    </span>
                                  </>
                                )}
                              </div>
                            </div>
                          </div>

                          {selectionMode ? (
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                toggleWorkflowSelection(workflow.id);
                              }}
                              className="flex-shrink-0 p-2 text-gray-300 hover:text-surface transition-colors"
                              title={isSelected ? "Deselect workflow" : "Select workflow"}
                            >
                              {isSelected ? (
                                <CheckSquare size={16} />
                              ) : (
                                <Square size={16} />
                              )}
                            </button>
                          ) : (
                            <div className="relative flex-shrink-0">
                              <button
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setShowWorkflowActionsFor(isActionsOpen ? null : workflow.id);
                                }}
                                className="p-1.5 text-gray-500 hover:text-surface hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
                                aria-label="Workflow actions"
                                aria-expanded={isActionsOpen}
                              >
                                <MoreVertical size={16} />
                              </button>

                              {isActionsOpen && (
                                <>
                                  <div
                                    className="fixed inset-0 z-20"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      setShowWorkflowActionsFor(null);
                                    }}
                                  />
                                  <div className="absolute right-0 top-full mt-1 z-30 w-44 bg-flow-node border border-gray-700 rounded-lg shadow-xl overflow-hidden animate-fade-in">
                                    <button
                                      data-testid={selectors.workflowBuilder.executeButton}
                                      onClick={(e) => {
                                        setShowWorkflowActionsFor(null);
                                        handleExecuteWorkflow(e, workflow.id);
                                      }}
                                      disabled={executionInProgress[workflow.id]}
                                      className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-surface transition-colors disabled:opacity-50"
                                    >
                                      {executionInProgress[workflow.id] ? (
                                        <Loader size={14} className="animate-spin" />
                                      ) : (
                                        <PlayCircle size={14} />
                                      )}
                                      Run Workflow
                                    </button>
                                    <div className="border-t border-gray-700" />
                                    <button
                                      onClick={(e) => handleDeleteWorkflow(e, workflow.id, workflow.name)}
                                      className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                                    >
                                      <Trash2 size={14} />
                                      Delete
                                    </button>
                                  </div>
                                </>
                              )}
                            </div>
                          )}
                        </div>

                        {/* Workflow Description */}
                        {workflow.description ? (
                          <p
                            className={`text-sm mb-4 line-clamp-2 ${
                              selectionMode && isSelected ? "text-gray-200" : "text-gray-400"
                            }`}
                          >
                            {(workflow.description as string | undefined) || ""}
                          </p>
                        ) : (
                          <p className="text-gray-600 text-sm mb-4 italic">
                            No description
                          </p>
                        )}

                        {/* Workflow Stats - Simplified for cleaner look */}
                        {(executionCount > 0 || successRate != null) && (
                          <div className="grid grid-cols-2 gap-3 mb-4">
                            <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                              <div className="text-lg font-semibold text-surface">
                                {executionCount}
                              </div>
                              <div className="text-xs text-gray-500">Executions</div>
                            </div>
                            <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                              <div className={`text-lg font-semibold ${
                                successRate != null
                                  ? successRate >= 80 ? "text-green-400" : successRate >= 50 ? "text-amber-400" : "text-red-400"
                                  : "text-gray-500"
                              }`}>
                                {successRate != null ? `${successRate}%` : "—"}
                              </div>
                              <div className="text-xs text-gray-500">Success</div>
                            </div>
                          </div>
                        )}

                        {/* Footer */}
                        <div
                          className={`flex items-center justify-between text-xs pt-3 border-t border-gray-700/50 ${
                            selectionMode && isSelected ? "text-gray-200" : "text-gray-500"
                          }`}
                        >
                          <div className="flex items-center gap-1.5">
                            <Clock size={12} />
                            <span>Updated {formatDate(workflow.updated_at || "")}</span>
                          </div>
                          {workflow.stats?.last_execution && (
                            <div className="flex items-center gap-1.5 text-green-500/80">
                              <Play size={10} />
                              <span>{formatDate(workflow.stats.last_execution)}</span>
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      <ResponsiveDialog
        isOpen={Boolean(confirmDialog)}
        onDismiss={() => closeConfirmDialog(false)}
        ariaLabel={confirmDialog?.title ?? "Confirm"}
        role="alertdialog"
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface">
            {confirmDialog?.title ?? "Confirm"}
          </h2>
          <button
            type="button"
            onClick={() => closeConfirmDialog(false)}
            className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
            aria-label="Close"
          >
            <X size={16} />
          </button>
        </div>
        {confirmDialog?.message ? (
          <p className="text-sm text-gray-200">{confirmDialog.message}</p>
        ) : null}
        <div className="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => closeConfirmDialog(false)}
            className="px-4 py-2 rounded-lg border border-gray-700 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
          >
            {confirmDialog?.cancelLabel ?? "Cancel"}
          </button>
          <button
            type="button"
            onClick={() => closeConfirmDialog(true)}
            className={`px-4 py-2 rounded-lg text-white transition-colors ${
              confirmDialog?.danger
                ? "bg-red-600 hover:bg-red-500"
                : "bg-flow-accent hover:bg-blue-600"
            }`}
          >
            {confirmDialog?.confirmLabel ?? "Confirm"}
          </button>
        </div>
      </ResponsiveDialog>

      <ResponsiveDialog
        isOpen={Boolean(promptDialog)}
        onDismiss={() => closePromptDialog(null)}
        ariaLabel={promptDialog?.title ?? "Input"}
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const validate = promptValidateRef.current;
            const normalize = promptNormalizeRef.current;
            const errorMessage = validate ? validate(promptValue) : null;
            if (errorMessage) {
              setPromptError(errorMessage);
              return;
            }
            const finalValue = normalize ? normalize(promptValue) : promptValue.trim();
            closePromptDialog(finalValue);
          }}
        >
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface">
              {promptDialog?.title ?? "Input"}
            </h2>
            <button
              type="button"
              onClick={() => closePromptDialog(null)}
              className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
              aria-label="Close"
            >
              <X size={16} />
            </button>
          </div>
          {promptDialog?.message ? (
            <p className="mb-3 text-sm text-gray-200">{promptDialog.message}</p>
          ) : null}
          <label className="block text-sm text-gray-300 mb-2">
            {promptDialog?.label ?? "Value"}
          </label>
          <input
            autoFocus
            value={promptValue}
            onChange={(e) => {
              setPromptValue(e.target.value);
              setPromptError(null);
            }}
            placeholder={promptDialog?.placeholder}
            className="w-full px-3 py-2 rounded-lg bg-gray-900 border border-gray-700 text-surface placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-flow-accent/60"
          />
          {promptError ? (
            <p className="mt-2 text-sm text-red-300">{promptError}</p>
          ) : null}
          <div className="mt-6 flex items-center justify-end gap-2">
            <button
              type="button"
              onClick={() => closePromptDialog(null)}
              className="px-4 py-2 rounded-lg border border-gray-700 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
            >
              {promptDialog?.cancelLabel ?? "Cancel"}
            </button>
            <button
              type="submit"
              className="px-4 py-2 rounded-lg bg-flow-accent text-white hover:bg-blue-600 transition-colors"
            >
              {promptDialog?.submitLabel ?? "OK"}
            </button>
          </div>
        </form>
      </ResponsiveDialog>

      {/* Floating Action Button (FAB) - Mobile only */}
      <button
        data-testid={selectors.workflows.newButtonFab}
        onClick={onCreateWorkflow}
        className="md:hidden fixed bottom-6 right-6 z-40 w-14 h-14 bg-flow-accent text-white rounded-full shadow-lg hover:bg-blue-600 transition-all hover:shadow-xl flex items-center justify-center"
        aria-label="New Workflow"
      >
        <Plus size={24} />
      </button>

      {showEditProjectModal && (
        <ProjectModal
          onClose={() => {
            setShowEditProjectModal(false);
          }}
          project={project}
          onSuccess={() => toast.success("Project updated successfully")}
        />
      )}
    </>
  );
}

export default ProjectDetail;
