import { useRef, useEffect, useCallback, useMemo } from "react";
import {
  Plus,
  FileCode,
  FolderOpen,
  Trash2,
  CheckSquare,
  Square,
  ListChecks,
  PencilLine,
  UploadCloud,
  Info,
  MoreVertical,
  X,
  Circle,
  RefreshCw,
  Loader,
} from "lucide-react";
import { usePopoverPosition } from "@hooks/usePopoverPosition";
import { usePromptDialog } from "@hooks/usePromptDialog";
import { useConfirmDialog } from "@hooks/useConfirmDialog";
import { selectors } from "@constants/selectors";
import { Breadcrumbs } from "@shared/layout";
import { ConfirmDialog, PromptDialog } from "@shared/ui";
import type { Project } from "@stores/projectStore";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import toast from "react-hot-toast";
import {
  useProjectDetailStore,
  useFilteredWorkflows,
} from "./hooks/useProjectDetailStore";
import { useFileTreeOperations } from "./hooks/useFileTreeOperations";

interface ProjectDetailHeaderProps {
  project: Project;
  onBack: () => void;
  onCreateWorkflow: () => void;
  onStartRecording?: () => void;
  onDeleteProject: () => Promise<void>;
  onImportRecording: (file: File) => Promise<void>;
  onWorkflowSelect: (workflow: Workflow) => Promise<void>;
}

/**
 * Header section for ProjectDetail including breadcrumbs, project info,
 * action menus, and selection controls
 */
export function ProjectDetailHeader({
  project,
  onBack,
  onCreateWorkflow,
  onStartRecording,
  onDeleteProject,
  onImportRecording,
  onWorkflowSelect,
}: ProjectDetailHeaderProps) {
  // Store state
  const viewMode = useProjectDetailStore((s) => s.viewMode);
  const selectionMode = useProjectDetailStore((s) => s.selectionMode);
  const selectedWorkflows = useProjectDetailStore((s) => s.selectedWorkflows);
  const workflows = useProjectDetailStore((s) => s.workflows);
  const showStatsPopover = useProjectDetailStore((s) => s.showStatsPopover);
  const showMoreMenu = useProjectDetailStore((s) => s.showMoreMenu);
  const showNewFileMenu = useProjectDetailStore((s) => s.showNewFileMenu);
  const isBulkDeleting = useProjectDetailStore((s) => s.isBulkDeleting);
  const isDeletingProject = useProjectDetailStore((s) => s.isDeletingProject);
  const isImportingRecording = useProjectDetailStore((s) => s.isImportingRecording);
  const selectedTreeFolder = useProjectDetailStore((s) => s.selectedTreeFolder);

  // Store actions
  const toggleSelectionMode = useProjectDetailStore((s) => s.toggleSelectionMode);
  const clearSelection = useProjectDetailStore((s) => s.clearSelection);
  const selectAll = useProjectDetailStore((s) => s.selectAll);
  const setShowStatsPopover = useProjectDetailStore((s) => s.setShowStatsPopover);
  const setShowMoreMenu = useProjectDetailStore((s) => s.setShowMoreMenu);
  const setShowNewFileMenu = useProjectDetailStore((s) => s.setShowNewFileMenu);
  const setShowEditProjectModal = useProjectDetailStore((s) => s.setShowEditProjectModal);
  const setBulkDeleting = useProjectDetailStore((s) => s.setBulkDeleting);
  const setIsImportingRecording = useProjectDetailStore((s) => s.setIsImportingRecording);
  const removeWorkflows = useProjectDetailStore((s) => s.removeWorkflows);
  const setSelectionMode = useProjectDetailStore((s) => s.setSelectionMode);
  const fetchProjectEntries = useProjectDetailStore((s) => s.fetchProjectEntries);
  const fetchWorkflows = useProjectDetailStore((s) => s.fetchWorkflows);

  // File operations
  const fileOps = useFileTreeOperations(project.id);

  // Workflow store
  const { bulkDeleteWorkflows } = useWorkflowStore();
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);

  // Filtered workflows
  const filteredWorkflows = useFilteredWorkflows();

  // Dialog hooks
  const {
    dialogState: confirmDialogState,
    confirm: requestConfirm,
    close: closeConfirmDialog,
  } = useConfirmDialog();
  const {
    dialogState: promptDialogState,
    prompt: requestPrompt,
    setValue: setPromptValue,
    close: closePromptDialog,
    submit: submitPrompt,
  } = usePromptDialog();

  // Refs for popovers
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const statsButtonRef = useRef<HTMLButtonElement | null>(null);
  const statsPopoverRef = useRef<HTMLDivElement | null>(null);
  const moreMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const moreMenuRef = useRef<HTMLDivElement | null>(null);
  const newFileMenuButtonRef = useRef<HTMLButtonElement | null>(null);
  const newFileMenuRef = useRef<HTMLDivElement | null>(null);

  // Popover positioning
  const { floatingStyles: statsPopoverStyles } = usePopoverPosition(
    statsButtonRef,
    statsPopoverRef,
    {
      isOpen: showStatsPopover,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
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

  // Computed values
  const workflowCount = useMemo(
    () => project.stats?.workflow_count ?? workflows.length,
    [project.stats?.workflow_count, workflows.length],
  );

  const totalExecutions = useMemo(
    () => project.stats?.execution_count ?? 0,
    [project.stats?.execution_count],
  );

  const formatDate = useCallback((dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }, []);

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

  // Click outside handlers
  useEffect(() => {
    if (!showStatsPopover) return;

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
  }, [showStatsPopover, setShowStatsPopover]);

  useEffect(() => {
    if (!showMoreMenu) return;

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
  }, [showMoreMenu, setShowMoreMenu]);

  useEffect(() => {
    if (!showNewFileMenu) return;

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
  }, [showNewFileMenu, setShowNewFileMenu]);

  // Handlers
  const handleRecordingImportClick = useCallback(() => {
    if (isImportingRecording) return;
    fileInputRef.current?.click();
  }, [isImportingRecording]);

  const handleRecordingImport = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const files = event.target.files;
      if (!files || files.length === 0) return;

      const file = files[0];
      setIsImportingRecording(true);

      try {
        await onImportRecording(file);
      } finally {
        setIsImportingRecording(false);
        if (event.target) {
          event.target.value = "";
        }
      }
    },
    [onImportRecording, setIsImportingRecording],
  );

  const handleSelectAll = useCallback(() => {
    if (selectedWorkflows.size === filteredWorkflows.length) {
      clearSelection();
    } else {
      selectAll(filteredWorkflows.map((w) => w.id));
    }
  }, [selectedWorkflows.size, filteredWorkflows, clearSelection, selectAll]);

  const handleBulkDeleteSelected = useCallback(async () => {
    if (selectedWorkflows.size === 0) return;

    const confirmed = await requestConfirm({
      title: "Delete workflows?",
      message: `Delete ${selectedWorkflows.size} workflow${selectedWorkflows.size === 1 ? "" : "s"}? This cannot be undone.`,
      confirmLabel: "Delete",
      cancelLabel: "Cancel",
      danger: true,
    });
    if (!confirmed) return;

    setBulkDeleting(true);
    try {
      const workflowIds = Array.from(selectedWorkflows);
      const deletedIds = await bulkDeleteWorkflows(project.id, workflowIds);
      removeWorkflows(deletedIds);
      toast.success(
        `Deleted ${deletedIds.length} workflow${deletedIds.length === 1 ? "" : "s"}`,
      );
      clearSelection();
      setSelectionMode(false);
    } catch (error) {
      toast.error("Failed to delete selected workflows");
    } finally {
      setBulkDeleting(false);
    }
  }, [
    selectedWorkflows,
    project.id,
    bulkDeleteWorkflows,
    requestConfirm,
    setBulkDeleting,
    removeWorkflows,
    clearSelection,
    setSelectionMode,
  ]);

  const handleCreateFolder = useCallback(async () => {
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
          const normalized = fileOps.normalizeProjectRelPath(value);
          if (!normalized.ok) return normalized.error;
          return null;
        },
        normalize: (value) => {
          const normalized = fileOps.normalizeProjectRelPath(value);
          return normalized.ok ? normalized.path : value.trim();
        },
      },
    );
    if (!relPath) return;
    try {
      await fileOps.createFolder(relPath);
      toast.success("Folder created");
      await fetchProjectEntries(project.id);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to create folder");
    }
  }, [selectedTreeFolder, requestPrompt, fileOps, fetchProjectEntries, project.id, setShowNewFileMenu]);

  const handleCreateWorkflowFile = useCallback(
    async (type: "action" | "flow" | "case") => {
      setShowNewFileMenu(false);
      const suffix = `.${type}.json`;
      const defaultDir =
        selectedTreeFolder !== null
          ? selectedTreeFolder
          : type === "action"
            ? "actions"
            : type === "case"
              ? "cases"
              : "flows";
      const suggested =
        defaultDir === ""
          ? `${type}-example${suffix}`
          : `${defaultDir}/${type}-example${suffix}`;
      const relPath = await requestPrompt(
        {
          title: `New ${type.charAt(0).toUpperCase() + type.slice(1)}`,
          label: `File path (relative to project root)`,
          defaultValue: suggested,
          submitLabel: "Create",
          cancelLabel: "Cancel",
        },
        {
          validate: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            if (!normalized.ok) return normalized.error;
            const lower = normalized.path.toLowerCase();
            const typedSuffixes = [".action.json", ".flow.json", ".case.json"];
            const hasTypedSuffix = typedSuffixes.some((s) => lower.endsWith(s));
            if (hasTypedSuffix && !lower.endsWith(suffix)) {
              return `File extension must be ${suffix}`;
            }
            return null;
          },
          normalize: (value) => {
            const normalized = fileOps.normalizeProjectRelPath(value);
            if (!normalized.ok) return value.trim();
            const lower = normalized.path.toLowerCase();
            if (lower.endsWith(suffix)) return normalized.path;
            if (
              lower.endsWith(".json") &&
              !lower.endsWith(".action.json") &&
              !lower.endsWith(".flow.json") &&
              !lower.endsWith(".case.json")
            ) {
              return `${normalized.path.slice(0, -".json".length)}${suffix}`;
            }
            return `${normalized.path}${suffix}`;
          },
        },
      );
      if (!relPath) return;
      try {
        const workflowId = await fileOps.createWorkflowFile(relPath, type);
        toast.success(`New ${type} created`);
        await fetchProjectEntries(project.id);
        await fetchWorkflows(project.id);
        if (workflowId) {
          await loadWorkflow(workflowId);
          const wf = useWorkflowStore.getState().currentWorkflow;
          if (wf) {
            await onWorkflowSelect(wf);
          }
        }
      } catch (err) {
        toast.error(err instanceof Error ? err.message : `Failed to create ${type}`);
      }
    },
    [selectedTreeFolder, requestPrompt, fileOps, fetchProjectEntries, fetchWorkflows, project.id, loadWorkflow, onWorkflowSelect, setShowNewFileMenu],
  );

  const handleResyncFiles = useCallback(async () => {
    setShowNewFileMenu(false);
    const confirmed = await requestConfirm({
      title: "Resync from disk?",
      message:
        "This will rebuild the project file index from disk and may repair workflow files (id/type/version).",
      confirmLabel: "Resync",
      cancelLabel: "Cancel",
    });
    if (!confirmed) return;
    await fileOps.resyncFiles();
  }, [requestConfirm, fileOps, setShowNewFileMenu]);

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        accept=".zip"
        className="hidden"
        onChange={handleRecordingImport}
      />

      <div className="relative px-6 pt-6 border-b border-gray-800 space-y-3 md:space-y-6">
        {/* Breadcrumbs */}
        <Breadcrumbs
          items={[
            { label: "Dashboard", onClick: onBack },
            { label: project.name, current: true },
          ]}
          className="mb-2"
        />

        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-start gap-4">
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
                      onClick={() => setShowStatsPopover(!showStatsPopover)}
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
                            <dt className="text-xs text-gray-400 mb-1">Project Name</dt>
                            <dd className="text-sm font-medium text-surface">{project.name}</dd>
                          </div>
                          <div className="mb-3">
                            <dt className="text-xs text-gray-400 mb-1">Description</dt>
                            <dd className="text-sm text-gray-300">
                              {project.description?.trim() ? project.description : "No description"}
                            </dd>
                          </div>
                          <div>
                            <dt className="text-xs text-gray-400 mb-1">Save Path</dt>
                            <dd className="text-sm text-gray-300 font-mono break-all">
                              {project.folder_path}
                            </dd>
                          </div>
                        </div>

                        {/* Metrics Section */}
                        <div className="mb-3">
                          <h4 className="text-xs font-semibold text-gray-400 mb-2">Metrics</h4>
                          <dl className="space-y-2 text-sm text-gray-300">
                            <div className="flex items-center justify-between">
                              <dt className="text-gray-400">Workflows</dt>
                              <dd className="font-medium text-surface">{workflowCount}</dd>
                            </div>
                            <div className="flex items-center justify-between">
                              <dt className="text-gray-400">Total executions</dt>
                              <dd className="font-medium text-surface">{totalExecutions}</dd>
                            </div>
                            <div className="flex items-center justify-between">
                              <dt className="text-gray-400">Last execution</dt>
                              <dd className="font-medium text-surface">{lastExecutionLabel}</dd>
                            </div>
                            <div className="flex items-center justify-between">
                              <dt className="text-gray-400">Last updated</dt>
                              <dd className="font-medium text-surface">{lastUpdatedLabel}</dd>
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
                      onClick={() => setShowMoreMenu(!showMoreMenu)}
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
                            {isImportingRecording ? "Importing..." : "Import Recording"}
                          </span>
                        </button>
                        <button
                          onClick={() => {
                            onDeleteProject();
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
                            {isDeletingProject ? "Deleting..." : "Delete Project"}
                          </span>
                        </button>
                      </div>
                    )}
                  </div>

                  {/* New File Menu (tree mode) or New Workflow Button (card mode) */}
                  {viewMode === "tree" ? (
                    <div className="relative md:ml-auto">
                      <button
                        ref={newFileMenuButtonRef}
                        data-testid={selectors.workflows.newButton}
                        onClick={() => setShowNewFileMenu(!showNewFileMenu)}
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
                            onClick={handleCreateFolder}
                            className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left"
                          >
                            <FolderOpen size={16} />
                            <span className="text-sm">New Folder</span>
                          </button>
                          {(["action", "flow", "case"] as const).map((type) => (
                            <button
                              key={type}
                              onClick={() => handleCreateWorkflowFile(type)}
                              className="w-full flex items-center gap-3 px-4 py-3 text-subtle hover:bg-flow-node-hover hover:text-surface transition-colors text-left"
                            >
                              <FileCode size={16} />
                              <span className="text-sm">
                                New {type.charAt(0).toUpperCase() + type.slice(1)}
                              </span>
                            </button>
                          ))}
                          <button
                            onClick={handleResyncFiles}
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

          {/* Selection Controls */}
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
      </div>

      <ConfirmDialog state={confirmDialogState} onClose={closeConfirmDialog} />
      <PromptDialog
        state={promptDialogState}
        onValueChange={setPromptValue}
        onClose={closePromptDialog}
        onSubmit={submitPrompt}
      />
    </>
  );
}
