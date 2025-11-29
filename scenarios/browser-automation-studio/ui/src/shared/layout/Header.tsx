import {
  Play,
  Save,
  Bug,
  ArrowLeft,
  Edit2,
  Check,
  X,
  Info,
  History,
  Loader2,
  AlertTriangle,
  CheckCircle2,
  Clock,
  Keyboard,
} from "lucide-react";
import { getModifierKey } from "@hooks/useKeyboardShortcuts";
import { useState, useRef, useEffect } from "react";
import { formatDistanceToNow } from "date-fns";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useExecutionStore } from "@stores/executionStore";
import { useProjectStore, type Project } from "@stores/projectStore";
import { AIEditModal } from "@features/ai";
import toast from "react-hot-toast";
import { usePopoverPosition } from "@hooks/usePopoverPosition";
import ResponsiveDialog from "./ResponsiveDialog";
import { selectors } from "@constants/selectors";

type HeaderWorkflow = Pick<
  Workflow,
  | "id"
  | "name"
  | "description"
  | "folderPath"
  | "createdAt"
  | "updatedAt"
  | "projectId"
> & {
  version?: Workflow["version"];
  lastChangeDescription?: Workflow["lastChangeDescription"];
  nodes?: Workflow["nodes"];
  edges?: Workflow["edges"];
};

interface HeaderProps {
  onNewWorkflow: () => void;
  onBackToDashboard?: () => void;
  currentProject?: Project | null;
  currentWorkflow?: HeaderWorkflow | null;
  showBackToProject?: boolean;
  onShowKeyboardShortcuts?: () => void;
}

function Header({
  onNewWorkflow: _onNewWorkflow,
  onBackToDashboard,
  currentProject,
  currentWorkflow: selectedWorkflow,
  showBackToProject,
  onShowKeyboardShortcuts,
}: HeaderProps) {
  const currentWorkflow = useWorkflowStore((state) => state.currentWorkflow);
  const saveWorkflow = useWorkflowStore((state) => state.saveWorkflow);
  const updateWorkflow = useWorkflowStore((state) => state.updateWorkflow);
  const loadWorkflow = useWorkflowStore((state) => state.loadWorkflow);
  const isDirty = useWorkflowStore((state) => state.isDirty);
  const isSaving = useWorkflowStore((state) => state.isSaving);
  const lastSavedAt = useWorkflowStore((state) => state.lastSavedAt);
  const lastSaveError = useWorkflowStore((state) => state.lastSaveError);
  const hasVersionConflict = useWorkflowStore(
    (state) => state.hasVersionConflict,
  );
  const conflictWorkflow = useWorkflowStore((state) => state.conflictWorkflow);
  const conflictMetadata = useWorkflowStore((state) => state.conflictMetadata);
  const refreshConflictWorkflow = useWorkflowStore(
    (state) => state.refreshConflictWorkflow,
  );
  const resolveConflictWithReload = useWorkflowStore(
    (state) => state.resolveConflictWithReload,
  );
  const forceSaveWorkflow = useWorkflowStore(
    (state) => state.forceSaveWorkflow,
  );
  const acknowledgeSaveError = useWorkflowStore(
    (state) => state.acknowledgeSaveError,
  );
  const loadWorkflowVersions = useWorkflowStore(
    (state) => state.loadWorkflowVersions,
  );
  const versionHistory = useWorkflowStore((state) => state.versionHistory);
  const isVersionHistoryLoading = useWorkflowStore(
    (state) => state.isVersionHistoryLoading,
  );
  const versionHistoryError = useWorkflowStore(
    (state) => state.versionHistoryError,
  );
  const restoreWorkflowVersion = useWorkflowStore(
    (state) => state.restoreWorkflowVersion,
  );
  const restoringVersion = useWorkflowStore((state) => state.restoringVersion);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const { isConnected, error } = useProjectStore();
  const [showAIEditModal, setShowAIEditModal] = useState(false);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const titleInputRef = useRef<HTMLInputElement>(null);
  const infoButtonRef = useRef<HTMLButtonElement | null>(null);
  const infoPopoverRef = useRef<HTMLDivElement | null>(null);
  const [showWorkflowInfo, setShowWorkflowInfo] = useState(false);
  const [showVersionHistory, setShowVersionHistory] = useState(false);
  const [showSaveErrorDetails, setShowSaveErrorDetails] = useState(false);
  const [isExecuting, setIsExecuting] = useState(false);

  const { floatingStyles: workflowInfoStyles } = usePopoverPosition(
    infoButtonRef,
    infoPopoverRef,
    {
      isOpen: showWorkflowInfo,
      placementPriority: ["bottom-end", "bottom-start", "top-end", "top-start"],
    },
  );

  useEffect(() => {
    if (hasVersionConflict && !conflictMetadata && currentWorkflow) {
      refreshConflictWorkflow().catch(() => {});
    }
  }, [
    hasVersionConflict,
    conflictMetadata,
    currentWorkflow?.id,
    refreshConflictWorkflow,
  ]);

  useEffect(() => {
    if (!lastSaveError && showSaveErrorDetails) {
      setShowSaveErrorDetails(false);
    }
  }, [lastSaveError, showSaveErrorDetails]);

  // Use the workflow from props if provided, otherwise use the one from store
  const displayWorkflow = selectedWorkflow || currentWorkflow;

  const formatDetailDate = (value?: Date | string) => {
    if (!value) {
      return "Not available";
    }
    const date = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(date.getTime())) {
      return "Not available";
    }
    return date.toLocaleString();
  };

  const workflowDescription = displayWorkflow?.description?.trim();

  const openVersionHistory = async () => {
    if (!currentWorkflow) {
      toast.error("No workflow loaded");
      return;
    }
    setShowVersionHistory(true);
    try {
      await loadWorkflowVersions(currentWorkflow.id);
    } catch (error) {
      toast.error("Failed to load version history");
    }
  };

  const handleHistoryButtonClick = async () => {
    if (!currentWorkflow) {
      toast.error("No workflow loaded");
      return;
    }

    if (isDirty) {
      try {
        await saveWorkflow({
          source: "manual",
          changeDescription: "Manual save before viewing history",
        });
        toast.success("Workflow saved successfully");
      } catch (error) {
        toast.error("Failed to save workflow");
        return;
      }
    }

    await openVersionHistory();
  };

  const openHistoryFromIndicator = () => {
    void handleHistoryButtonClick();
  };

  const handleAutosaveIndicatorClick = (
    event: React.MouseEvent<HTMLDivElement>,
  ) => {
    event.stopPropagation();
    openHistoryFromIndicator();
  };

  const handleAutosaveIndicatorKeyDown = (
    event: React.KeyboardEvent<HTMLDivElement>,
  ) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      event.stopPropagation();
      openHistoryFromIndicator();
    }
  };

  const handleRestoreVersion = async (versionNumber: number) => {
    if (!currentWorkflow) {
      toast.error("No workflow loaded");
      return;
    }
    const confirmed = window.confirm(
      `Restore workflow to version ${versionNumber}? This will create a new revision.`,
    );
    if (!confirmed) {
      return;
    }
    try {
      await restoreWorkflowVersion(
        currentWorkflow.id,
        versionNumber,
        `Restored from version ${versionNumber}`,
      );
      toast.success(`Workflow restored to version ${versionNumber}`);
    } catch (error) {
      toast.error("Failed to restore workflow version");
    }
  };

  const handleReloadWorkflow = async () => {
    if (!currentWorkflow) {
      return;
    }
    try {
      if (hasVersionConflict) {
        await resolveConflictWithReload();
      } else {
        await loadWorkflow(currentWorkflow.id);
      }
      toast.success("Workflow reloaded");
    } catch (error) {
      toast.error("Failed to reload workflow");
    }
  };

  const handleForceSave = async () => {
    if (!currentWorkflow) {
      return;
    }
    try {
      await forceSaveWorkflow({
        changeDescription: "Force save after conflict",
        source: "manual-force-save",
      });
      toast.success("Local changes saved");
    } catch (error) {
      toast.error("Failed to force save workflow");
    }
  };

  const handleRetrySave = async () => {
    if (!currentWorkflow) {
      toast.error("No workflow to save");
      return;
    }
    try {
      await saveWorkflow({
        source: "manual-retry",
        changeDescription: "Retry after failed autosave",
      });
      acknowledgeSaveError();
      toast.success("Workflow saved");
    } catch (error) {
      toast.error("Retry failed");
    }
  };

  const handleRefreshConflict = async () => {
    try {
      await refreshConflictWorkflow();
      toast.success("Conflict snapshot refreshed");
    } catch (error) {
      toast.error("Failed to refresh conflict snapshot");
    }
  };

  const lastSavedLabel = lastSavedAt
    ? formatDistanceToNow(lastSavedAt, { addSuffix: true })
    : null;
  const baseIndicatorClass =
    "flex flex-wrap items-center gap-2 rounded-lg border px-3 py-1.5 text-xs shadow-sm backdrop-blur-sm";
  const historyButtonDisabled =
    !currentWorkflow || isSaving || restoringVersion !== null;
  const remoteVersionLabel = conflictMetadata
    ? `v${conflictMetadata.remoteVersion}`
    : "server version";
  const remoteSavedLabel = conflictMetadata
    ? formatDistanceToNow(conflictMetadata.remoteUpdatedAt, { addSuffix: true })
    : null;
  const remoteSourceLabel =
    conflictMetadata?.changeSource ?? conflictWorkflow?.lastChangeSource ?? "";

  let saveStatusNode: React.ReactNode;
  if (!currentWorkflow) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-gray-800 bg-gray-900/70 text-gray-400`}
      >
        <Info size={14} className="text-gray-400" />
        <span>No workflow loaded</span>
      </div>
    );
  } else if (restoringVersion !== null) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-blue-500/40 bg-blue-500/10 text-blue-100`}
      >
        <Loader2 size={14} className="animate-spin" />
        <span>Restoring version {restoringVersion}…</span>
      </div>
    );
  } else if (hasVersionConflict) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-amber-500/50 bg-amber-500/10 text-amber-100`}
      >
        <AlertTriangle size={14} className="text-amber-200" />
        <span className="font-medium">
          Version conflict with {remoteVersionLabel}
          {remoteSavedLabel ? ` · saved ${remoteSavedLabel}` : ""}
          {remoteSourceLabel ? ` · ${remoteSourceLabel}` : ""}
        </span>
        <div className="flex flex-wrap items-center gap-1 sm:gap-2">
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              if (historyButtonDisabled) {
                return;
              }
              void openVersionHistory();
            }}
            disabled={historyButtonDisabled}
            className="rounded border border-amber-400/40 bg-amber-500/10 px-2 py-1 text-amber-100 transition-colors hover:bg-amber-500/20 disabled:cursor-not-allowed disabled:opacity-60"
            data-testid={selectors.header.buttons.versionConflictDetails}
          >
            Details
          </button>
        </div>
      </div>
    );
  } else if (lastSaveError) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-rose-500/60 bg-rose-500/10 text-rose-100`}
      >
        <AlertTriangle size={14} className="text-rose-200" />
        <span className="font-medium">Failed to save</span>
        <div className="flex items-center gap-1 sm:gap-2">
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              setShowSaveErrorDetails(true);
            }}
            className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-rose-400/40 bg-rose-500/20 text-rose-50 transition-colors hover:bg-rose-500/30 focus:outline-none focus:ring-2 focus:ring-rose-300/60 focus:ring-offset-2 focus:ring-offset-gray-900"
            title="View autosave error details"
            aria-label="View autosave error details"
            data-testid={selectors.header.buttons.saveErrorDetails}
          >
            <Info size={14} />
          </button>
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              handleRetrySave();
            }}
            disabled={isSaving}
            className="rounded border border-rose-400/40 bg-rose-500/20 px-2 py-1 text-xs font-medium text-rose-50 transition-colors hover:bg-rose-500/30 focus:outline-none focus:ring-2 focus:ring-rose-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-60"
            data-testid={selectors.header.buttons.saveRetry}
          >
            Retry
          </button>
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              acknowledgeSaveError();
            }}
            className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-rose-400/40 bg-rose-500/20 text-rose-50 transition-colors hover:bg-rose-500/30 focus:outline-none focus:ring-2 focus:ring-rose-300/60 focus:ring-offset-2 focus:ring-offset-gray-900"
            title="Dismiss autosave error"
            aria-label="Dismiss autosave error"
            data-testid={selectors.header.buttons.saveErrorDismiss}
          >
            <X size={14} />
          </button>
        </div>
      </div>
    );
  } else if (isSaving) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-blue-500/40 bg-blue-500/10 text-blue-100`}
      >
        <Loader2 size={14} className="animate-spin" />
        <span>Saving…</span>
      </div>
    );
  } else if (isDirty) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-yellow-500/50 bg-yellow-500/10 text-yellow-100`}
      >
        <span className="font-medium">Unsaved changes</span>
        <button
          type="button"
          onClick={(event) => {
            event.stopPropagation();
            void handleHistoryButtonClick();
          }}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-yellow-400/60 bg-yellow-500/20 text-yellow-50 transition-colors hover:bg-yellow-500/30 focus:outline-none focus:ring-2 focus:ring-yellow-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Save now and open history"
          aria-label="Save now and open history"
        >
          <Save size={14} />
        </button>
      </div>
    );
  } else if (lastSavedLabel) {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-emerald-500/40 bg-emerald-500/10 text-emerald-100`}
      >
        <CheckCircle2 size={14} className="text-emerald-200" />
        <span>Saved {lastSavedLabel}</span>
        <button
          type="button"
          data-testid={selectors.header.buttons.versionHistory}
          onClick={(event) => {
            event.stopPropagation();
            void handleHistoryButtonClick();
          }}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-emerald-400/60 bg-emerald-500/20 text-emerald-50 transition-colors hover:bg-emerald-500/30 focus:outline-none focus:ring-2 focus:ring-emerald-300/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Open version history"
          aria-label="Open version history"
        >
          <History size={14} />
        </button>
      </div>
    );
  } else {
    saveStatusNode = (
      <div
        className={`${baseIndicatorClass} border-gray-700 bg-gray-900/80 text-gray-300`}
      >
        <Clock size={14} className="text-gray-400" />
        <span>Not saved yet</span>
        <button
          type="button"
          data-testid={selectors.header.buttons.versionHistory}
          onClick={(event) => {
            event.stopPropagation();
            void handleHistoryButtonClick();
          }}
          disabled={historyButtonDisabled}
          className="inline-flex h-7 w-7 items-center justify-center rounded-full border border-gray-600 bg-gray-800/80 text-gray-200 transition-colors hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-blue-400/60 focus:ring-offset-2 focus:ring-offset-gray-900 disabled:cursor-not-allowed disabled:opacity-50"
          title="Open version history"
          aria-label="Open version history"
        >
          <History size={14} />
        </button>
      </div>
    );
  }

  const autosaveIndicator = (
    <div
      role="button"
      tabIndex={0}
      aria-label="Open version history"
      onClick={handleAutosaveIndicatorClick}
      onKeyDown={handleAutosaveIndicatorKeyDown}
      className="inline-flex rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-flow-accent/60 focus-visible:ring-offset-2 focus-visible:ring-offset-gray-900 cursor-pointer"
      data-testid={selectors.header.saveStatus.indicator}
    >
      {saveStatusNode}
    </div>
  );

  const handleExecute = async () => {
    if (!currentWorkflow) {
      toast.error("No workflow to execute");
      return;
    }
    if (isExecuting) {
      return;
    }

    try {
      setIsExecuting(true);
      await startExecution(currentWorkflow.id, async () => {
        if (!isDirty) {
          return;
        }
        await saveWorkflow({
          source: "execute",
          changeDescription: "Autosave before execution",
        });
      });
    } catch (error) {
      toast.error("Failed to start execution");
    } finally {
      setIsExecuting(false);
    }
  };

  const handleDebug = () => {
    if (!currentWorkflow) {
      toast.error("No workflow loaded to edit");
      return;
    }
    setShowAIEditModal(true);
  };

  const handleStartEditTitle = () => {
    if (!displayWorkflow) return;
    setEditTitle(displayWorkflow.name);
    setIsEditingTitle(true);
  };

  const handleSaveTitle = async () => {
    if (!displayWorkflow || !editTitle.trim()) {
      setIsEditingTitle(false);
      return;
    }

    try {
      await updateWorkflow({ ...displayWorkflow, name: editTitle.trim() });
      setIsEditingTitle(false);
      toast.success("Workflow title updated");
    } catch (error) {
      toast.error("Failed to update workflow title");
    }
  };

  const handleCancelEditTitle = () => {
    setIsEditingTitle(false);
    setEditTitle("");
  };

  const handleTitleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSaveTitle();
    } else if (e.key === "Escape") {
      handleCancelEditTitle();
    }
  };

  // Focus input when editing starts
  useEffect(() => {
    if (isEditingTitle && titleInputRef.current) {
      titleInputRef.current.focus();
      titleInputRef.current.select();
    }
  }, [isEditingTitle]);

  useEffect(() => {
    if (!showWorkflowInfo) {
      return;
    }

    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Node;
      if (
        infoPopoverRef.current &&
        !infoPopoverRef.current.contains(target) &&
        infoButtonRef.current &&
        !infoButtonRef.current.contains(target)
      ) {
        setShowWorkflowInfo(false);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setShowWorkflowInfo(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [showWorkflowInfo]);

  useEffect(() => {
    setShowWorkflowInfo(false);
  }, [displayWorkflow?.id, currentProject?.id]);

  useEffect(() => {
    if (!currentWorkflow) {
      setShowVersionHistory(false);
    }
  }, [currentWorkflow?.id]);

  return (
    <>
      <header
        className="bg-flow-node border-b border-gray-800 px-4 py-3"
        data-testid={selectors.header.root}
      >
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              {onBackToDashboard && (
                <button
                  onClick={onBackToDashboard}
                  className="toolbar-button flex items-center gap-2"
                  title={
                    showBackToProject ? "Back to Project" : "Back to Dashboard"
                  }
                  data-testid={
                    showBackToProject
                      ? selectors.header.buttons.backToProject
                      : selectors.header.buttons.backToDashboard
                  }
                >
                  <ArrowLeft size={16} />
                </button>
              )}

              <h1 className="text-xl font-bold text-white flex items-center gap-2">
                {displayWorkflow ? (
                  <div className="flex items-center gap-2">
                    {isEditingTitle ? (
                      <div className="flex items-center gap-2">
                        <input
                          ref={titleInputRef}
                          type="text"
                          value={editTitle}
                          onChange={(e) => setEditTitle(e.target.value)}
                          onKeyDown={handleTitleKeyDown}
                          onBlur={handleSaveTitle}
                          className="bg-gray-800 text-white px-2 py-1 rounded border border-gray-600 focus:border-flow-accent focus:outline-none"
                          placeholder="Workflow name..."
                          data-testid={selectors.header.title.input}
                        />
                        <button
                          onClick={handleSaveTitle}
                          className="text-green-400 hover:text-green-300 p-1"
                          title="Save title"
                          data-testid={selectors.header.title.saveButton}
                        >
                          <Check size={14} />
                        </button>
                        <button
                          onClick={handleCancelEditTitle}
                          className="text-red-400 hover:text-red-300 p-1"
                          title="Cancel editing"
                          data-testid={selectors.header.title.cancelButton}
                        >
                          <X size={14} />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-2 group">
                        <span
                          className="cursor-pointer"
                          onClick={handleStartEditTitle}
                          data-testid={selectors.header.title.text}
                        >
                          {displayWorkflow.name}
                        </span>
                        <button
                          onClick={handleStartEditTitle}
                          className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-white transition-opacity p-1"
                          title="Edit workflow name"
                          data-testid={selectors.header.title.editButton}
                        >
                          <Edit2 size={14} />
                        </button>
                      </div>
                    )}
                  </div>
                ) : currentProject ? (
                  currentProject.name
                ) : (
                  "Browser Automation Studio"
                )}
              </h1>
              {(currentProject || displayWorkflow) && (
                <div className="relative">
                  <button
                    ref={infoButtonRef}
                    type="button"
                    onClick={() => setShowWorkflowInfo((prev) => !prev)}
                    className="p-1.5 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
                    title="Workflow details"
                    aria-label="Workflow details"
                    aria-expanded={showWorkflowInfo}
                    data-testid={selectors.header.buttons.info}
                  >
                    <Info size={16} />
                  </button>
                  {showWorkflowInfo && (
                    <div
                      ref={infoPopoverRef}
                      style={workflowInfoStyles}
                      className="z-30 w-80 max-h-96 overflow-y-auto rounded-lg border border-gray-700 bg-flow-node p-4 shadow-lg"
                      data-testid={selectors.header.info.popover}
                    >
                      <div className="flex items-center justify-between mb-3">
                        <h3 className="text-sm font-semibold text-white">
                          Workflow Details
                        </h3>
                        <button
                          type="button"
                          onClick={() => setShowWorkflowInfo(false)}
                          className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
                          aria-label="Close workflow details"
                          data-testid={selectors.header.buttons.infoClose}
                        >
                          <X size={14} />
                        </button>
                      </div>

                      {displayWorkflow && (
                        <div className="mb-4 pb-4 border-b border-gray-700">
                          <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500 mb-2">
                            Workflow
                          </h4>
                          <dl className="space-y-2 text-sm text-gray-300">
                            <div>
                              <dt className="text-xs text-gray-500">Name</dt>
                              <dd className="text-sm font-medium text-white">
                                {displayWorkflow.name}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">
                                Description
                              </dt>
                              <dd className="text-sm text-gray-300">
                                {workflowDescription ||
                                  "No description provided"}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">Folder</dt>
                              <dd className="text-sm text-gray-300 font-mono break-all">
                                {displayWorkflow.folderPath || "/"}
                              </dd>
                            </div>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                              <div>
                                <dt className="text-xs text-gray-500">
                                  Created
                                </dt>
                                <dd className="text-xs text-gray-300">
                                  {formatDetailDate(displayWorkflow.createdAt)}
                                </dd>
                              </div>
                              <div>
                                <dt className="text-xs text-gray-500">
                                  Last Updated
                                </dt>
                                <dd className="text-xs text-gray-300">
                                  {formatDetailDate(displayWorkflow.updatedAt)}
                                </dd>
                              </div>
                            </div>
                          </dl>
                        </div>
                      )}

                      {currentProject && (
                        <div className="space-y-2 text-sm text-gray-300">
                          <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                            Project
                          </h4>
                          <dl className="space-y-2">
                            <div>
                              <dt className="text-xs text-gray-500">Name</dt>
                              <dd className="text-sm font-medium text-white">
                                {currentProject.name}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">
                                Description
                              </dt>
                              <dd className="text-sm text-gray-300">
                                {currentProject.description?.trim() ||
                                  "No description provided"}
                              </dd>
                            </div>
                            <div>
                              <dt className="text-xs text-gray-500">
                                Save Path
                              </dt>
                              <dd className="text-sm text-gray-300 font-mono break-all">
                                {currentProject.folder_path}
                              </dd>
                            </div>
                          </dl>
                        </div>
                      )}

                      <div className="mt-4 pt-4 border-t border-gray-800 space-y-2 text-sm text-gray-300">
                        <h4 className="text-xs font-semibold uppercase tracking-wide text-gray-500">
                          Connection
                        </h4>
                        <p
                          className={
                            isConnected
                              ? "text-xs text-green-300"
                              : "text-xs text-rose-300"
                          }
                        >
                          {isConnected
                            ? "Connected to project services"
                            : error
                              ? `Disconnected: ${error}`
                              : "Disconnected from project services"}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            <div className="flex flex-col sm:flex-row sm:items-center sm:gap-3">
              <div className="mt-1 sm:mt-0 flex items-center gap-2">
                {autosaveIndicator}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-1 sm:gap-2">
            {onShowKeyboardShortcuts && (
              <button
                onClick={onShowKeyboardShortcuts}
                className="toolbar-button flex items-center gap-0 sm:gap-2"
                title={`Keyboard shortcuts (${getModifierKey()}+?)`}
                aria-label="Show keyboard shortcuts"
              >
                <Keyboard size={16} />
              </button>
            )}
            <button
              onClick={handleDebug}
              className="toolbar-button flex items-center gap-0 sm:gap-2"
              title="Edit with AI"
              aria-label="Edit with AI"
              data-testid={selectors.header.buttons.debug}
            >
              <Bug size={16} />
              <span className="hidden text-sm sm:inline">Debug</span>
            </button>

            <button
              onClick={() => void handleExecute()}
              disabled={isExecuting}
              className="bg-flow-accent hover:bg-blue-600 disabled:bg-flow-accent/60 disabled:cursor-not-allowed text-white px-3 sm:px-4 py-2 rounded-lg flex items-center gap-0 sm:gap-2 transition-colors"
              title="Execute Workflow"
              aria-label="Execute Workflow"
              data-testid={selectors.header.buttons.execute}
            >
              {isExecuting ? <Loader2 size={16} className="animate-spin" /> : <Play size={16} />}
              <span className="hidden text-sm font-medium sm:inline">
                {isExecuting ? "Executing..." : "Execute"}
              </span>
            </button>
          </div>
        </div>
      </header>

      {showAIEditModal && (
        <AIEditModal onClose={() => setShowAIEditModal(false)} />
      )}

      <ResponsiveDialog
        isOpen={showSaveErrorDetails && Boolean(lastSaveError)}
        onDismiss={() => setShowSaveErrorDetails(false)}
        ariaLabel="Autosave error details"
        data-testid={selectors.header.saveError.dialog}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Autosave error</h2>
          <button
            type="button"
            onClick={() => setShowSaveErrorDetails(false)}
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
            data-testid={selectors.header.saveError.dialogCloseButton}
          >
            <X size={16} />
          </button>
        </div>
        <div className="space-y-3 text-sm text-gray-200">
          <p>
            {lastSaveError?.message ?? "Unknown error occurred while saving."}
          </p>
          {typeof lastSaveError?.status === "number" && (
            <p className="text-xs text-gray-400">
              HTTP status: {lastSaveError.status}
            </p>
          )}
          <p className="text-xs text-gray-400">
            Retry the save or dismiss the error once resolved. Autosave will
            resume automatically once the issue clears.
          </p>
        </div>
      </ResponsiveDialog>

      <ResponsiveDialog
        isOpen={showVersionHistory}
        onDismiss={() => setShowVersionHistory(false)}
        ariaLabel="Workflow Versions"
        size="wide"
        data-testid={selectors.header.versionHistory.dialog}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">
            Workflow Versions
          </h2>
          <button
            type="button"
            onClick={() => setShowVersionHistory(false)}
            className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded-full transition-colors"
            data-testid={selectors.header.versionHistory.closeButton}
          >
            <X size={16} />
          </button>
        </div>

        {hasVersionConflict && currentWorkflow && (
          <div className="mb-5 space-y-4 rounded-lg border border-amber-500/40 bg-amber-500/10 p-4">
            <div>
              <h3 className="text-sm font-semibold text-amber-100">
                Resolve version conflict
              </h3>
              <p className="mt-1 text-xs text-amber-200">
                The server copy ({remoteVersionLabel}) differs from your draft.
                Review both snapshots and choose how to proceed.
              </p>
            </div>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-lg border border-gray-800 bg-gray-900/70 p-3">
                <h4 className="text-sm font-semibold text-white">
                  Local draft
                </h4>
                <ul className="mt-2 space-y-1 text-xs text-gray-300">
                  <li>Version {currentWorkflow.version}</li>
                  <li>Nodes: {currentWorkflow.nodes?.length ?? 0}</li>
                  <li>Edges: {currentWorkflow.edges?.length ?? 0}</li>
                  {currentWorkflow.lastChangeDescription && (
                    <li>Change: {currentWorkflow.lastChangeDescription}</li>
                  )}
                </ul>
              </div>
              <div className="rounded-lg border border-amber-500/60 bg-gray-900/70 p-3">
                <h4 className="text-sm font-semibold text-amber-100">
                  Server snapshot
                </h4>
                <ul className="mt-2 space-y-1 text-xs text-amber-200">
                  <li>{remoteVersionLabel}</li>
                  {remoteSavedLabel && <li>Saved {remoteSavedLabel}</li>}
                  <li>Source: {remoteSourceLabel}</li>
                  <li>
                    Nodes:{" "}
                    {conflictMetadata?.nodeCount ??
                      conflictWorkflow?.nodes?.length ??
                      0}
                  </li>
                  <li>
                    Edges:{" "}
                    {conflictMetadata?.edgeCount ??
                      conflictWorkflow?.edges?.length ??
                      0}
                  </li>
                  {conflictMetadata?.changeDescription && (
                    <li>Change: {conflictMetadata.changeDescription}</li>
                  )}
                </ul>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <button
                type="button"
                onClick={() => void handleRefreshConflict()}
                className="rounded border border-amber-400/40 bg-amber-500/10 px-3 py-1.5 text-xs text-amber-100 transition-colors hover:bg-amber-500/20"
                data-testid={selectors.header.buttons.versionConflictRefresh}
              >
                Refresh snapshot
              </button>
              <button
                type="button"
                onClick={() => void handleReloadWorkflow()}
                className="rounded border border-amber-400/60 bg-amber-500/10 px-3 py-1.5 text-xs text-amber-100 transition-colors hover:bg-amber-500/25"
                data-testid={selectors.header.buttons.versionConflictReload}
              >
                Reload remote
              </button>
              <button
                type="button"
                onClick={() => void handleForceSave()}
                disabled={isSaving}
                className="rounded bg-amber-500/20 px-3 py-1.5 text-xs text-amber-50 transition-colors hover:bg-amber-500/30 disabled:cursor-not-allowed disabled:opacity-60"
                data-testid={selectors.header.buttons.versionConflictForceSave}
              >
                Force save local
              </button>
            </div>
          </div>
        )}

        {!currentWorkflow ? (
          <div className="text-sm text-gray-400">
            Load a workflow to view version history.
          </div>
        ) : isVersionHistoryLoading ? (
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <Loader2 size={16} className="animate-spin" />
            Loading version history…
          </div>
        ) : versionHistory.length === 0 ? (
          <div className="text-sm text-gray-400">
            No saved versions yet. Save a workflow to create history.
          </div>
        ) : (
          <div className="space-y-3">
            {versionHistory.map((version) => {
              const isCurrent = currentWorkflow?.version === version.version;
              const createdLabel = formatDistanceToNow(version.createdAt, {
                addSuffix: true,
              });
              return (
                <div
                  key={version.version}
                  data-testid={selectors.header.versionHistory.item({
                    version: version.version.toString(),
                  })}
                  className={`rounded-lg border p-4 bg-gray-900/70 ${
                    isCurrent
                      ? "border-blue-500/60 shadow-inner shadow-blue-500/20"
                      : "border-gray-800"
                  }`}
                >
                  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                    <div>
                      <div className="flex items-center gap-2 text-sm font-semibold text-white">
                        Version {version.version}
                        {isCurrent && (
                          <span className="text-xs text-blue-300 uppercase tracking-wide">
                            Current
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        Saved {createdLabel} by {version.createdBy || "Unknown"}
                      </div>
                    </div>
                    <div className="flex items-center gap-3 text-xs text-gray-400">
                      <span>Nodes: {version.nodeCount}</span>
                      <span>Edges: {version.edgeCount}</span>
                      <span className="hidden sm:inline text-gray-500">
                        Hash
                      </span>
                      <span className="font-mono text-gray-500 sm:text-xs break-all">
                        {version.definitionHash
                          ? version.definitionHash.slice(0, 12)
                          : "—"}
                      </span>
                    </div>
                  </div>
                  {version.changeDescription && (
                    <div className="mt-3 text-sm text-gray-300">
                      {version.changeDescription}
                    </div>
                  )}
                  <div className="mt-3 flex items-center justify-end gap-2">
                    <button
                      type="button"
                      data-testid={selectors.header.versionHistory.restoreButton({
                        version: version.version.toString(),
                      })}
                      onClick={() => handleRestoreVersion(version.version)}
                      disabled={
                        isCurrent || restoringVersion === version.version
                      }
                      className="px-3 py-1.5 text-xs bg-gray-800 hover:bg-gray-700 rounded disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                    >
                      {restoringVersion === version.version ? (
                        <Loader2 size={14} className="animate-spin" />
                      ) : (
                        <History size={14} />
                      )}
                      Restore
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {versionHistoryError && (
          <div className="mt-4 text-xs text-rose-300">
            {versionHistoryError}
          </div>
        )}
      </ResponsiveDialog>
    </>
  );
}

export default Header;
