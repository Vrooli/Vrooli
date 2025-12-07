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

interface FolderItem {
  path: string;
  name: string;
  children?: FolderItem[];
  workflows?: WorkflowWithStats[];
  expanded?: boolean;
}

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
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const statsButtonRef = useRef<HTMLButtonElement | null>(null);
  const statsPopoverRef = useRef<HTMLDivElement | null>(null);
  const viewModeButtonRef = useRef<HTMLButtonElement | null>(null);
  const viewModeDropdownRef = useRef<HTMLDivElement | null>(null);
  const moreMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const moreMenuRef = useRef<HTMLDivElement | null>(null);
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
  const { deleteProject } = useProjectStore();
  const { bulkDeleteWorkflows } = useWorkflowStore();
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const isExecutionViewerOpen = Boolean(currentExecution);

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

  // Build folder structure from workflows - memoized to prevent unnecessary recalculation
  const buildFolderStructure = useCallback(
    (workflowList: WorkflowWithStats[]): FolderItem[] => {
      const folderMap = new Map<string, FolderItem>();

      workflowList.forEach((workflow) => {
        const pathParts =
          (workflow.folder_path as string | undefined)
            ?.split("/")
            .filter(Boolean) || [];
        let currentPath = "";
        let parent: FolderItem | null = null;

        pathParts.forEach((part: string) => {
          currentPath += "/" + part;

          if (!folderMap.has(currentPath)) {
            const folder: FolderItem = {
              path: currentPath,
              name: part,
              children: [],
              workflows: [],
            };
            folderMap.set(currentPath, folder);

            if (parent) {
              if (!parent.children) parent.children = [];
              if (!parent.children.find((c) => c.path === currentPath)) {
                parent.children.push(folder);
              }
            }
          }

          parent = folderMap.get(currentPath) || null;
        });

        // Add workflow to the appropriate folder (only if we have a parent folder)
        if (parent) {
          const parentFolder = parent as FolderItem;
          if (!parentFolder.workflows) {
            parentFolder.workflows = [];
          }
          parentFolder.workflows.push(workflow);
        } else if (pathParts.length === 0) {
          // Workflow is at root level, create a default folder
          const defaultFolder: FolderItem = {
            path: "/workflows",
            name: "Workflows",
            workflows: [workflow],
          };
          if (!folderMap.has("/workflows")) {
            folderMap.set("/workflows", defaultFolder);
          } else {
            const folder = folderMap.get("/workflows");
            if (folder && folder.workflows) {
              folder.workflows.push(workflow);
            }
          }
        }
      });

      // Get root folders
      const rootFolders: FolderItem[] = [];
      folderMap.forEach((folder, path) => {
        if (path.split("/").filter(Boolean).length === 1) {
          rootFolders.push(folder);
        }
      });

      // If no folder structure exists, create a default one
      if (rootFolders.length === 0 && workflowList.length > 0) {
        rootFolders.push({
          path: "/workflows",
          name: "All Workflows",
          workflows: workflowList,
        });
      }

      return rootFolders;
    },
    [],
  );

  // Memoize folder structure to avoid rebuilding on every render
  const memoizedFolderStructure = useMemo(() => {
    return buildFolderStructure(workflows);
  }, [workflows, buildFolderStructure]);

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
  }, [project.id, buildFolderStructure]);

  useEffect(() => {
    fetchWorkflows();
  }, [fetchWorkflows]);

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

      const confirmed = window.confirm(
        `Delete "${workflowName}"? This action cannot be undone.`
      );
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
    [project.id, bulkDeleteWorkflows]
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

    const confirmed = window.confirm(
      `Delete ${selectedWorkflows.size} workflow${selectedWorkflows.size === 1 ? "" : "s"}? This action cannot be undone.`,
    );
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
    buildFolderStructure,
  ]);

  const handleDeleteProject = useCallback(async () => {
    const confirmed = window.confirm(
      "Delete this project and all associated workflows? This action cannot be undone.",
    );
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
  }, [project.id, deleteProject, onBack]);

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

  interface TreeEntry {
    kind: "folder" | "workflow";
    folder?: FolderItem;
    workflow?: WorkflowWithStats;
  }

  // Tree View Component
  const FolderTreeItem = ({
    item,
    prefixParts = [],
  }: {
    item: FolderItem;
    prefixParts?: boolean[];
  }) => {
    const isExpanded = expandedFolders.has(item.path);
    const childFolders = item.children ?? [];
    const workflowItems = item.workflows ?? [];
    const hasChildren = childFolders.length > 0 || workflowItems.length > 0;

    const entries: TreeEntry[] = [
      ...childFolders.map((child) => ({
        kind: "folder" as const,
        folder: child,
      })),
      ...workflowItems.map((workflow) => ({
        kind: "workflow" as const,
        workflow,
      })),
    ];

    return (
      <div className="select-none">
        <div
          className="flex items-center gap-1.5 px-2 py-1.5 hover:bg-flow-node rounded cursor-pointer transition-colors"
          onClick={() => toggleFolder(item.path)}
        >
          {renderTreePrefix(prefixParts)}
          {hasChildren ? (
            isExpanded ? (
              <ChevronDown size={14} className="text-gray-400" />
            ) : (
              <ChevronRight size={14} className="text-gray-400" />
            )
          ) : (
            <span className="inline-block w-3.5" aria-hidden="true" />
          )}
          <FolderOpen size={14} className="text-yellow-500" />
          <span className="text-sm text-gray-300">{item.name}</span>
          {workflowItems.length > 0 && (
            <span className="ml-auto text-xs text-gray-500 mr-2">
              {workflowItems.length} workflow
              {workflowItems.length !== 1 ? "s" : ""}
            </span>
          )}
        </div>

        {isExpanded &&
          entries.map((entry, index) => {
            const isLastChild = index === entries.length - 1;
            const childPrefix = [...prefixParts, !isLastChild];

            if (entry.kind === "folder" && entry.folder) {
              return (
                <FolderTreeItem
                  key={entry.folder.path}
                  item={entry.folder}
                  prefixParts={childPrefix}
                />
              );
            }

            if (!entry.workflow) {
              return null;
            }

            const workflow = entry.workflow;
            const isSelected = selectedWorkflows.has(workflow.id);

            const handleRowClick = async () => {
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
                onClick={handleRowClick}
                className={`flex items-center gap-1.5 px-2 py-1.5 rounded cursor-pointer transition-colors group ${
                  selectionMode
                    ? isSelected
                      ? "bg-flow-node/80 border border-flow-accent"
                      : "hover:bg-flow-node border border-transparent"
                    : "hover:bg-flow-node"
                }`}
              >
                {renderTreePrefix(childPrefix)}
                {selectionMode ? (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      toggleWorkflowSelection(workflow.id);
                    }}
                    className="text-gray-300 hover:text-white"
                    title={isSelected ? "Deselect workflow" : "Select workflow"}
                    aria-label={
                      isSelected ? "Deselect workflow" : "Select workflow"
                    }
                  >
                    {isSelected ? (
                      <CheckSquare size={14} />
                    ) : (
                      <Square size={14} />
                    )}
                  </button>
                ) : (
                  <FileCode size={14} className="text-green-400" />
                )}
                <span
                  className={`text-sm ${selectionMode && isSelected ? "text-white" : "text-gray-400 group-hover:text-white"}`}
                >
                  {workflow.name}
                </span>
                {!selectionMode && (
                  <div className="ml-auto flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      data-testid={selectors.workflowBuilder.executeButton}
                      onClick={(e) => handleExecuteWorkflow(e, workflow.id)}
                      disabled={executionInProgress[workflow.id]}
                      className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded transition-colors"
                      title="Execute Workflow"
                      aria-label="Execute Workflow"
                    >
                      {executionInProgress[workflow.id] ? (
                        <Loader size={14} className="animate-spin" />
                      ) : (
                        <PlayCircle size={14} />
                      )}
                    </button>
                  </div>
                )}
              </div>
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
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
              >
                <ArrowLeft size={20} />
              </button>
              <div>
                <div className="flex flex-col md:flex-row md:items-center gap-2 mb-1">
                  <h1 className="text-2xl font-bold text-white">
                    {project.name}
                  </h1>
                  <div className="flex items-center gap-2">
                    {/* Info Button */}
                    <div className="relative">
                      <button
                        ref={statsButtonRef}
                        type="button"
                        onClick={() => setShowStatsPopover((prev) => !prev)}
                        className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
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
                            <h3 className="text-sm font-semibold text-white">
                              Project Details
                            </h3>
                            <button
                              type="button"
                              onClick={() => setShowStatsPopover(false)}
                              className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
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
                              <dd className="text-sm font-medium text-white">
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
                                <dd className="font-medium text-white">
                                  {workflowCount}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">
                                  Total executions
                                </dt>
                                <dd className="font-medium text-white">
                                  {totalExecutions}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">
                                  Last execution
                                </dt>
                                <dd className="font-medium text-white">
                                  {lastExecutionLabel}
                                </dd>
                              </div>
                              <div className="flex items-center justify-between">
                                <dt className="text-gray-400">Last updated</dt>
                                <dd className="font-medium text-white">
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
                      className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
                      title="Edit project details"
                    >
                      <PencilLine size={16} />
                    </button>

                    {/* More Menu Button */}
                    <div className="relative">
                      <button
                        ref={moreMenuButtonRef}
                        onClick={() => setShowMoreMenu((prev) => !prev)}
                        className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
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
                            className="w-full flex items-center gap-3 px-4 py-3 text-gray-300 hover:bg-flow-node-hover hover:text-white transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-left"
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
                              className="w-full flex items-center gap-3 px-4 py-3 text-gray-300 hover:bg-flow-node-hover hover:text-white transition-colors text-left"
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
                            className="w-full flex items-center gap-3 px-4 py-3 text-gray-300 hover:bg-flow-node-hover hover:text-white transition-colors disabled:opacity-40 disabled:cursor-not-allowed text-left"
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
                    <button
                      data-testid={selectors.workflows.newButton}
                      onClick={onCreateWorkflow}
                      className="hidden md:flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors md:ml-auto"
                    >
                      <Plus size={16} />
                      <span>New Workflow</span>
                    </button>
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
                    className="flex items-center gap-2 px-3 py-1.5 md:rounded-lg rounded-full border border-gray-700 text-gray-300 hover:border-flow-accent hover:text-white transition-colors"
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
                    ? "border-flow-accent text-white"
                    : "border-transparent text-gray-400 hover:text-white"
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
                    ? "border-flow-accent text-white"
                    : "border-transparent text-gray-400 hover:text-white"
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
                    className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
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
                        : "text-gray-400 hover:text-white"
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
                        : "text-gray-400 hover:text-white"
                    }`}
                    title="Tree View"
                  >
                    <FolderTree size={16} />
                    <span className="text-sm">Tree</span>
                  </button>
                </div>
                {/* Mobile: Icon button with popover */}
                <div className="md:hidden relative">
                  <button
                    ref={viewModeButtonRef}
                    onClick={() =>
                      setShowViewModeDropdown(!showViewModeDropdown)
                    }
                    className="p-2 bg-flow-node border border-gray-700 rounded-lg text-gray-300 hover:border-flow-accent hover:text-white transition-colors"
                    title={viewMode === "card" ? "Card View" : "Tree View"}
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
                            : "text-gray-300 hover:bg-flow-node-hover hover:text-white"
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
                            : "text-gray-300 hover:bg-flow-node-hover hover:text-white"
                        }`}
                      >
                        <FolderTree size={16} />
                        <span className="text-sm">Tree View</span>
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
                    <h3 className="text-xl font-semibold text-white mb-2">
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
                            <h4 className="font-medium text-white">AI-Assisted</h4>
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
                            <h4 className="font-medium text-white">Visual Builder</h4>
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
                    <h3 className="text-lg font-semibold text-white mb-2">
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
                  {memoizedFolderStructure.length === 0 ? (
                    <div className="text-center text-gray-400 py-8">
                      <FileCode size={48} className="mx-auto mb-4 opacity-50" />
                      <p>No folder structure available</p>
                    </div>
                  ) : (
                    <div className="space-y-1">
                      {memoizedFolderStructure.map((folder, index) => {
                        const hasNextRoot =
                          index < memoizedFolderStructure.length - 1;
                        const rootPrefix =
                          memoizedFolderStructure.length > 1
                            ? [hasNextRoot]
                            : [];
                        return (
                          <FolderTreeItem
                            key={folder.path}
                            item={folder}
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
                                className="font-semibold text-white truncate"
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
                              className="flex-shrink-0 p-2 text-gray-300 hover:text-white transition-colors"
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
                                className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
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
                                      className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-white transition-colors disabled:opacity-50"
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
                              <div className="text-lg font-semibold text-white">
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
