/**
 * Scenario Details Page
 *
 * Displays detailed information about a single scenario including:
 * - Scenario metadata (name, description, status, priority, tags)
 * - Editable metadata toggle (greenfield)
 * - Deletion with confirmation dialog and archive option
 * - Navigation back to the scenarios list
 *
 * Experience Architecture (Phase 29):
 * - Enhanced breadcrumb navigation shows current location context
 * - Reduces cognitive load for returning users navigating hierarchies
 *
 * [REQ:REQ-P0-007] Scenario Metadata Management UI
 * [REQ:REQ-P0-008] Scenario Deletion Workflow with Safeguards
 */

import { useState, useEffect } from "react";

import {
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronUp,
  Circle,
  Loader2,
  MoreHorizontal,
  Package,
  Play,
  RefreshCw,
  Settings2,
  Square,
  Terminal,
  Trash2,
  XCircle,
} from "lucide-react";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { TagList } from "../components/ui/tag-list";
import type { FileSelectionResult } from "../components/scenarios/file-selection-dialog";
import { ScenarioDeleteDialog } from "../components/scenarios/ScenarioDeleteDialog";
import { capitalize } from "../lib";
import { selectors } from "../consts/selectors";
import { SCENARIO_STATUS_COLORS, SCENARIO_STATUS_ICONS } from "../types";
import { useDetailSelectionStore } from "../stores/detail-selection-store";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailSection } from "../components/detail/DetailSection";
import { SCENARIO_LENSES } from "../components/detail/lens-options";
import { selectionToNodeId } from "../stores/detail-selection-store";
import { useDetailNavigation } from "../hooks/useDetailNavigation";
import { useScenarioDetailData } from "../hooks/useScenarioDetailData";
import { useArchivePreferences } from "../hooks/useArchivePreferences";

export function ScenarioDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);
  const name = selection?.name;
  const nodeId = selectionToNodeId(selection);
  const { closeDetail } = useDetailNavigation();

  const {
    scenario,
    isLoading,
    error,
    refetch,
    localGreenfield,
    handleGreenfieldToggle,
    updateMutation,
    actionMutation,
    deleteMutationInternal,
    specSyncPhase,
    specSyncError,
    isSpecSyncInProgress,
    triggerSpecSyncArchive,
    handleSpecSyncCancel,
    resetSpecSync,
    scenarioFiles,
    filesLoading,
    loadScenarioFiles,
  } = useScenarioDetailData(name, closeDetail);

  // Delete dialog state
  // [REQ:REQ-P0-008] Delete confirmation dialog state
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [archiveOnDelete, setArchiveOnDelete] = useState(true);
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const [mobileDangerExpanded, setMobileDangerExpanded] = useState(false);
  const [showFileSelectionDialog, setShowFileSelectionDialog] = useState(false);

  const {
    archivePreset,
    setArchivePreset,
    archiveMode,
    setArchiveMode,
    customPaths,
    setCustomPaths,
    specSyncEnabled,
    setSpecSyncEnabled,
    fileTreeLoaded,
    setFileTreeLoaded,
    effectiveArchiveMode,
    previewPaths,
    previewList,
  } = useArchivePreferences(scenarioFiles, archiveOnDelete);

  useEffect(() => {
    setShowActionsSheet(false);
    setMobileDangerExpanded(false);
  }, [name]);

  // Delete handlers
  const handleDeleteClick = () => {
    setShowDeleteDialog(true);
    setFileTreeLoaded(false);
    resetSpecSync();
    void loadScenarioFiles().finally(() => setFileTreeLoaded(true));
  };

  const handleDeleteConfirm = () => {
    if (archiveOnDelete && specSyncEnabled) {
      triggerSpecSyncArchive(effectiveArchiveMode, customPaths, archivePreset);
    } else {
      deleteMutationInternal.mutate({ archiveOnDelete, effectiveArchiveMode, customPaths, archivePreset });
    }
  };

  const handleSpecSyncRetry = () => {
    resetSpecSync();
    handleDeleteConfirm();
  };

  const handleArchiveWithoutSync = () => {
    resetSpecSync();
    setSpecSyncEnabled(false);
    deleteMutationInternal.mutate({ archiveOnDelete, effectiveArchiveMode, customPaths, archivePreset });
  };

  const handleDeleteCancel = () => {
    if (specSyncPhase === "syncing") {
      handleSpecSyncCancel();
      setShowDeleteDialog(false);
      setArchiveOnDelete(true);
      return;
    }
    setShowDeleteDialog(false);
    setArchiveOnDelete(true);
    resetSpecSync();
  };

  const handleCustomizeFilesClick = () => {
    void loadScenarioFiles().finally(() => setFileTreeLoaded(true));
    setShowFileSelectionDialog(true);
  };

  const handleFileSelectionConfirm = (selection: FileSelectionResult) => {
    setArchiveMode(selection.preset ? "preset" : "custom");
    if (selection.preset) {
      setArchivePreset(selection.preset);
      setCustomPaths([]);
    } else {
      setCustomPaths(selection.paths);
    }
    setShowFileSelectionDialog(false);
  };

  const handleFileSelectionCancel = () => {
    setShowFileSelectionDialog(false);
  };

  // Get status icon
  const StatusIcon = scenario ? (SCENARIO_STATUS_ICONS[scenario.status] || Circle) : Circle;
  const isPageLoading = isLoading && !scenario;
  const isRunning = scenario?.status === "running";
  const isStopped = scenario?.status === "stopped";
  const actionInFlight = actionMutation.isPending ? actionMutation.variables : null;
  const actionError = actionMutation.isError
    ? `Failed to ${actionMutation.variables ?? "run action"}. Please try again.`
    : null;

  if (!name) {
    return (
      <div className="space-y-6" data-testid={selectors.scenarioDetails.page}>
        <ErrorState
          error={new Error("Scenario name is required")}
          title="Invalid URL"
        />
      </div>
    );
  }

  const renderActionButtons = (closeOnAction = false) => {
    const runAction = (action: () => void) => {
      if (closeOnAction) {
        setShowActionsSheet(false);
      }
      action();
    };
    const rowButtonClass =
      "h-10 w-full justify-start rounded-lg border-slate-700/80 bg-slate-900/40 px-3 text-sm text-slate-100 hover:bg-slate-800/70";

    return (
      <div className="space-y-2">
        <Button
          variant={isRunning ? "outline" : "default"}
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("start"))}
          disabled={actionMutation.isPending || isRunning}
        >
          {actionInFlight === "start" ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Play className="mr-2 h-4 w-4" />
          )}
          Start
        </Button>
        <Button
          variant={isStopped ? "outline" : "default"}
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("stop"))}
          disabled={actionMutation.isPending || isStopped}
        >
          {actionInFlight === "stop" ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Square className="mr-2 h-4 w-4" />
          )}
          Stop
        </Button>
        <Button
          variant="outline"
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("restart"))}
          disabled={actionMutation.isPending}
        >
          {actionInFlight === "restart" ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Restart
        </Button>
      </div>
    );
  };

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="scenario"
          title={scenario?.displayName || name || "Unknown"}
          status={scenario?.status}
          nodeId={nodeId}
          lenses={SCENARIO_LENSES}
          actions={scenario ? renderActionButtons() : undefined}
        />
      }
      mobileActions={scenario ? renderActionButtons() : undefined}
      mobileActionsTitle="Scenario Actions"
    >
    <div className="space-y-6" data-testid={selectors.scenarioDetails.page}>
      {/* Loading state */}
      {isPageLoading && (
        <PageLoadingState
          label="Loading scenario details..."
          variant="detail"
          testId="scenario-details-loading-state"
        />
      )}

      {/* Error state */}
      {error && (
        <ErrorState
          error={error}
          title="Unable to load scenario"
          onRetry={() => refetch()}
        />
      )}

      {/* Scenario details */}
      {scenario && !error && (
        <>
          <div className="flex min-h-[100dvh] flex-col lg:hidden">
            <div className="sticky top-0 z-30 flex items-center gap-2 border-b border-slate-800 bg-slate-950/95 px-3 py-2 backdrop-blur">
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
                onClick={closeDetail}
                aria-label="Close scenario details"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-slate-100">{scenario.displayName}</p>
                <p className="truncate text-xs text-slate-400">
                  {capitalize(scenario.status)} · P{scenario.priority}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-9 rounded-lg border-slate-700/80 bg-slate-900/45 px-3 text-xs font-medium text-slate-100 hover:bg-slate-800/70"
                onClick={() => setShowActionsSheet(true)}
                aria-label="Open scenario actions"
              >
                Actions
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
                onClick={() => setMobileDangerExpanded((prev) => !prev)}
                aria-label={mobileDangerExpanded ? "Hide danger section" : "Show danger section"}
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </div>

            <div className="flex-1 space-y-0 overflow-y-auto pb-6">
              {actionError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                  {actionError}
                </div>
              )}

              <DetailSection title="Overview" hideDivider>
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <StatusIcon className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`} />
                    <span className="text-xs uppercase tracking-wider text-slate-500">
                      {capitalize(scenario.status)}
                    </span>
                    <span className="rounded-full bg-slate-700 px-2.5 py-0.5 text-xs text-slate-300">P{scenario.priority}</span>
                    {localGreenfield && (
                      <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-[11px] text-cyan-300">
                        Greenfield
                      </span>
                    )}
                  </div>
                  <p className="text-sm leading-relaxed text-slate-300">
                    {scenario.description || "No description provided"}
                  </p>
                  {scenario.completenessScore !== undefined && (
                    <div className="flex items-center gap-2">
                      <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-700">
                        <div
                          className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                          style={{ width: `${scenario.completenessScore}%` }}
                        />
                      </div>
                      <span className="text-xs text-slate-400">{scenario.completenessScore}%</span>
                    </div>
                  )}
                  {scenario.tags.length > 0 && <TagList tags={scenario.tags} maxTags={10} />}
                </div>
              </DetailSection>

              <DetailSection title="Scenario Settings" icon={Settings2}>
                <div className="space-y-3">
                  {updateMutation.isPending && (
                    <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
                  )}

                  <div className="rounded-lg bg-slate-700/30 p-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-slate-200">Greenfield Mode</span>
                        {localGreenfield ? (
                          <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                        ) : (
                          <XCircle className="h-4 w-4 text-slate-500" />
                        )}
                      </div>
                      <p className="text-xs text-slate-400">
                        Treat this scenario as a new project without existing code base
                      </p>
                    </div>
                    <Button
                      variant={localGreenfield ? "default" : "outline"}
                      size="sm"
                      className="mt-3 w-full"
                      onClick={handleGreenfieldToggle}
                      disabled={updateMutation.isPending}
                    >
                      {localGreenfield ? "Enabled" : "Disabled"}
                    </Button>
                  </div>

                  {updateMutation.isError && (
                    <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                      Failed to update settings. Please try again.
                    </div>
                  )}
                </div>
              </DetailSection>

              <DetailSection title="Quick Actions (CLI)" icon={Terminal} data-testid={selectors.scenarioDetails.cliHint}>
                <div className="space-y-3">
                  <p className="text-sm text-slate-400">
                    Common operations for this scenario are also available via the command line.
                  </p>
                  <div className="space-y-2">
                    <div className="rounded-lg bg-slate-700/30 p-3">
                      <span className="text-xs font-medium text-slate-300">View Logs</span>
                      <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                        vrooli scenario logs {name}
                      </code>
                    </div>
                    <div className="rounded-lg bg-slate-700/30 p-3">
                      <span className="text-xs font-medium text-slate-300">Check Status</span>
                      <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                        vrooli scenario status {name}
                      </code>
                    </div>
                    <div className="rounded-lg bg-slate-700/30 p-3">
                      <span className="text-xs font-medium text-slate-300">Run Tests</span>
                      <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                        vrooli scenario test {name}
                      </code>
                    </div>
                    <div className="rounded-lg bg-slate-700/30 p-3">
                      <span className="text-xs font-medium text-slate-300">Restart Scenario</span>
                      <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                        vrooli scenario restart {name}
                      </code>
                    </div>
                  </div>
                </div>
              </DetailSection>

              <section className="mt-4 border-t border-slate-800 pt-4">
                <button
                  type="button"
                  className="flex w-full items-center justify-between pt-3 pb-2 text-left"
                  onClick={() => setMobileDangerExpanded((prev) => !prev)}
                >
                  <span className="flex items-center gap-2 text-base font-semibold text-red-300">
                    <Trash2 className="h-4 w-4" />
                    Danger Zone
                  </span>
                  {mobileDangerExpanded ? (
                    <ChevronUp className="h-4 w-4 text-red-300" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-red-300" />
                  )}
                </button>
                {mobileDangerExpanded && (
                  <div className="space-y-3 pb-3">
                    <p className="text-sm text-slate-400">
                      Permanently remove this scenario from the catalog. This action cannot be undone.
                    </p>
                    <Button
                      variant="destructive"
                      size="sm"
                      className="w-full"
                      onClick={handleDeleteClick}
                      disabled={deleteMutationInternal.isPending}
                    >
                      {deleteMutationInternal.isPending ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Deleting...
                        </>
                      ) : (
                        <>
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete Scenario
                        </>
                      )}
                    </Button>
                    {deleteMutationInternal.isError && (
                      <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                        Failed to delete scenario. Please try again.
                      </div>
                    )}
                  </div>
                )}
              </section>
            </div>
          </div>

          <div className="hidden space-y-0 lg:block">
            {/* Scenario header */}
            <DetailSection title="Overview" hideDivider data-testid={selectors.scenarioDetails.header}>
              <div className="flex items-start justify-between">
                <div className="space-y-2">
                  <div className="flex items-center gap-3">
                    <StatusIcon
                      className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`}
                      data-testid={selectors.scenarioDetails.status}
                    />
                    <span className="text-sm uppercase tracking-wider text-slate-500">
                      {capitalize(scenario.status)}
                    </span>
                    <span
                      className="rounded-full bg-slate-700 px-3 py-1 text-sm text-slate-300"
                      data-testid={selectors.scenarioDetails.priority}
                    >
                      P{scenario.priority}
                    </span>
                    {localGreenfield && (
                      <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-400">
                        Greenfield
                      </span>
                    )}
                  </div>
                  <h1
                    className="text-2xl font-bold text-slate-100"
                    data-testid={selectors.scenarioDetails.title}
                  >
                    {scenario.displayName}
                  </h1>
                  <p
                    className="text-slate-400"
                    data-testid={selectors.scenarioDetails.description}
                  >
                    {scenario.description || "No description provided"}
                  </p>
                  <TagList
                    tags={scenario.tags}
                    maxTags={10}
                    className="mt-4"
                    data-testid={selectors.scenarioDetails.tags}
                  />
                </div>

                <div className="flex flex-col items-end gap-2">
                  <Package className="h-12 w-12 text-slate-600" />
                  {scenario.completenessScore !== undefined && (
                    <div className="flex items-center gap-2">
                      <div className="h-2 w-24 overflow-hidden rounded-full bg-slate-700">
                        <div
                          className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                          style={{ width: `${scenario.completenessScore}%` }}
                        />
                      </div>
                      <span className="text-sm text-slate-400">{scenario.completenessScore}%</span>
                    </div>
                  )}
                  <div className="flex flex-wrap items-center justify-end gap-2" data-testid={selectors.scenarioDetails.actionsSection}>
                    <Button
                      variant={isRunning ? "outline" : "default"}
                      size="sm"
                      onClick={() => actionMutation.mutate("start")}
                      disabled={actionMutation.isPending || isRunning}
                      data-testid={selectors.scenarioDetails.startButton}
                    >
                      {actionInFlight === "start" ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Play className="mr-2 h-4 w-4" />
                      )}
                      Start
                    </Button>
                    <Button
                      variant={isStopped ? "outline" : "default"}
                      size="sm"
                      onClick={() => actionMutation.mutate("stop")}
                      disabled={actionMutation.isPending || isStopped}
                      data-testid={selectors.scenarioDetails.stopButton}
                    >
                      {actionInFlight === "stop" ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <Square className="mr-2 h-4 w-4" />
                      )}
                      Stop
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => actionMutation.mutate("restart")}
                      disabled={actionMutation.isPending}
                      data-testid={selectors.scenarioDetails.restartButton}
                    >
                      {actionInFlight === "restart" ? (
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <RefreshCw className="mr-2 h-4 w-4" />
                      )}
                      Restart
                    </Button>
                  </div>
                  {actionError && (
                    <div
                      className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-1 text-xs text-red-400"
                      data-testid={selectors.scenarioDetails.actionError}
                    >
                      {actionError}
                    </div>
                  )}
                </div>
              </div>
            </DetailSection>

            {/* Metadata management section */}
            <DetailSection title="Scenario Settings" icon={Settings2} data-testid={selectors.scenarioDetails.metadataSection}>
              {updateMutation.isPending && (
                <Loader2 className="mb-2 h-4 w-4 animate-spin text-cyan-400" />
              )}

              <div className="space-y-4">
                {/* Greenfield toggle */}
                <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-slate-200">Greenfield Mode</span>
                      {localGreenfield ? (
                        <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                      ) : (
                        <XCircle className="h-4 w-4 text-slate-500" />
                      )}
                    </div>
                    <p className="text-sm text-slate-400">
                      Treat this scenario as a new project without existing code base
                    </p>
                  </div>
                  <Button
                    variant={localGreenfield ? "default" : "outline"}
                    size="sm"
                    onClick={handleGreenfieldToggle}
                    disabled={updateMutation.isPending}
                    data-testid={selectors.scenarioDetails.greenfieldToggle}
                  >
                    {localGreenfield ? "Enabled" : "Disabled"}
                  </Button>
                </div>
              </div>

              {/* Error feedback */}
              {updateMutation.isError && (
                <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                  Failed to update settings. Please try again.
                </div>
              )}
            </DetailSection>

            {/* CLI Quick Actions - Helps ops users access common operations (Phase 29 Iteration 5) */}
            <DetailSection title="Quick Actions (CLI)" icon={Terminal} data-testid={selectors.scenarioDetails.cliHint}>
              <p className="mb-4 text-sm text-slate-400">
                Common operations for this scenario are also available via the command line.
              </p>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-lg bg-slate-700/30 p-3">
                  <span className="text-xs font-medium text-slate-300">View Logs</span>
                  <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                    vrooli scenario logs {name}
                  </code>
                </div>
                <div className="rounded-lg bg-slate-700/30 p-3">
                  <span className="text-xs font-medium text-slate-300">Check Status</span>
                  <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                    vrooli scenario status {name}
                  </code>
                </div>
                <div className="rounded-lg bg-slate-700/30 p-3">
                  <span className="text-xs font-medium text-slate-300">Run Tests</span>
                  <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                    vrooli scenario test {name}
                  </code>
                </div>
                <div className="rounded-lg bg-slate-700/30 p-3">
                  <span className="text-xs font-medium text-slate-300">Restart Scenario</span>
                  <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                    vrooli scenario restart {name}
                  </code>
                </div>
              </div>
            </DetailSection>

            {/* Danger zone - Delete scenario */}
            {/* [REQ:REQ-P0-008] Scenario deletion with safeguards */}
            <DetailSection title="Danger Zone" className="text-red-300">
              <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
                <div className="space-y-1">
                  <span className="font-medium text-slate-200">Delete Scenario</span>
                  <p className="text-sm text-slate-400">
                    Permanently remove this scenario from the catalog. This action cannot be undone.
                  </p>
                </div>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={handleDeleteClick}
                  disabled={deleteMutationInternal.isPending}
                  data-testid={selectors.scenarioDetails.deleteButton}
                >
                  {deleteMutationInternal.isPending ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Deleting...
                    </>
                  ) : (
                    <>
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </>
                  )}
                </Button>
              </div>

              {/* Delete error feedback */}
              {deleteMutationInternal.isError && (
                <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                  Failed to delete scenario. Please try again.
                </div>
              )}
            </DetailSection>
          </div>

          <BottomSheet
            isOpen={showActionsSheet}
            onClose={() => setShowActionsSheet(false)}
            title="Scenario Actions"
            description="Manage scenario lifecycle"
            data-testid="scenario-mobile-actions-sheet"
          >
            {renderActionButtons(true)}
          </BottomSheet>
        </>
      )}

      {/* Delete confirmation dialog */}
      <ScenarioDeleteDialog
        isOpen={showDeleteDialog}
        scenarioDisplayName={scenario?.displayName || name || ""}
        scenarioName={scenario?.name}
        isDeleteLoading={deleteMutationInternal.isPending}
        archiveOnDelete={archiveOnDelete}
        onArchiveOnDeleteChange={(checked) => {
          setArchiveOnDelete(checked);
          if (checked) {
            void loadScenarioFiles().finally(() => setFileTreeLoaded(true));
          }
        }}
        archivePreset={archivePreset}
        onArchivePresetChange={setArchivePreset}
        effectiveArchiveMode={effectiveArchiveMode}
        customPaths={customPaths}
        onCustomPathsChange={setCustomPaths}
        onArchiveModeChange={setArchiveMode}
        previewPaths={previewPaths}
        previewList={previewList}
        filesLoading={filesLoading}
        specSyncEnabled={specSyncEnabled}
        onSpecSyncEnabledChange={setSpecSyncEnabled}
        specSyncPhase={specSyncPhase}
        specSyncError={specSyncError}
        isSpecSyncInProgress={isSpecSyncInProgress}
        onSpecSyncRetry={handleSpecSyncRetry}
        onArchiveWithoutSync={handleArchiveWithoutSync}
        onSpecSyncCancel={handleSpecSyncCancel}
        showFileSelectionDialog={showFileSelectionDialog}
        onCustomizeFilesClick={handleCustomizeFilesClick}
        onFileSelectionConfirm={handleFileSelectionConfirm}
        onFileSelectionCancel={handleFileSelectionCancel}
        scenarioFiles={scenarioFiles}
        onClose={handleDeleteCancel}
        onConfirm={handleDeleteConfirm}
      />
    </div>
    </DetailPageLayout>
  );
}
