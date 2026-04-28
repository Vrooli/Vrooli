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
import { useParams } from "react-router-dom";

import { Circle, Package } from "lucide-react";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import type { FileSelectionResult } from "../components/scenarios/file-selection-dialog";
import { ScenarioDeleteDialog } from "../components/scenarios/ScenarioDeleteDialog";
import { ScenarioOverviewSection } from "../components/scenarios/ScenarioOverviewSection";
import { ScenarioCoverageSection } from "../components/scenarios/ScenarioCoverageSection";
import { ScenarioSettingsSection } from "../components/scenarios/ScenarioSettingsSection";
import { ScenarioCliHints } from "../components/scenarios/ScenarioCliHints";
import { ScenarioDangerZone } from "../components/scenarios/ScenarioDangerZone";
import { ScenarioLifecycleActions } from "../components/scenarios/ScenarioLifecycleActions";
import { ScenarioMobileView } from "../components/scenarios/ScenarioMobileView";
import { selectors } from "../consts/selectors";
import { SCENARIO_STATUS_ICONS } from "../types";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { SCENARIO_LENSES } from "../components/detail/lens-options";
import { useScenarioDetailData } from "../hooks/useScenarioDetailData";
import { useArchivePreferences } from "../hooks/useArchivePreferences";
import { routeTargetToNodeId } from "../app/routes/route-paths";
import { useAppBack } from "../app/routes/useAppBack";

export function ScenarioDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const nodeId = routeTargetToNodeId({ entityType: "scenario", name });
  const closeDetail = useAppBack();

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
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [archiveOnDelete, setArchiveOnDelete] = useState(true);
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const [showFileSelectionDialog, setShowFileSelectionDialog] = useState(false);

  const {
    archivePreset,
    setArchivePreset,
    archiveMode: _archiveMode,
    setArchiveMode,
    customPaths,
    setCustomPaths,
    specSyncEnabled,
    setSpecSyncEnabled,
    fileTreeLoaded: _fileTreeLoaded,
    setFileTreeLoaded,
    effectiveArchiveMode,
    previewPaths,
    previewList,
  } = useArchivePreferences(scenarioFiles, archiveOnDelete);

  useEffect(() => {
    setShowActionsSheet(false);
  }, [name]);

  // --- Delete handlers ---
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

  // --- Derived ---
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

  const lifecycleActions = scenario ? (
    <ScenarioLifecycleActions
      isRunning={isRunning}
      isStopped={isStopped}
      actionPending={actionMutation.isPending}
      actionInFlight={actionInFlight}
      onAction={(action) => actionMutation.mutate(action)}
    />
  ) : undefined;

  const mobileLifecycleActions = scenario ? (
    <ScenarioLifecycleActions
      isRunning={isRunning}
      isStopped={isStopped}
      actionPending={actionMutation.isPending}
      actionInFlight={actionInFlight}
      onAction={(action) => actionMutation.mutate(action)}
      mobile
    />
  ) : undefined;

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="scenario"
          entityIcon={Package}
          title={scenario?.displayName || name || "Unknown"}
          status={scenario?.status}
          nodeId={nodeId}
          lenses={SCENARIO_LENSES}
          actions={lifecycleActions}
        />
      }
      mobileActions={mobileLifecycleActions}
      mobileActionsTitle="Scenario Actions"
    >
    <div className="space-y-6" data-testid={selectors.scenarioDetails.page}>
      {isPageLoading && (
        <PageLoadingState
          label="Loading scenario details..."
          variant="detail"
          testId="scenario-details-loading-state"
        />
      )}

      {error && (
        <ErrorState
          error={error}
          title="Unable to load scenario"
          onRetry={() => refetch()}
        />
      )}

      {scenario && !error && (
        <>
          <ScenarioMobileView
            scenario={scenario}
            name={name}
            StatusIcon={StatusIcon}
            localGreenfield={localGreenfield}
            onClose={closeDetail}
            onShowActionsSheet={() => setShowActionsSheet(true)}
            actionError={actionError}
            onGreenfieldToggle={handleGreenfieldToggle}
            updatePending={updateMutation.isPending}
            updateError={updateMutation.isError}
            onDeleteClick={handleDeleteClick}
            deletePending={deleteMutationInternal.isPending}
            deleteError={deleteMutationInternal.isError}
          />

          <ScenarioCoverageSection scenarioName={name} />

          <div className="hidden space-y-0 lg:block">
            <ScenarioOverviewSection
              scenario={scenario}
              StatusIcon={StatusIcon}
              localGreenfield={localGreenfield}
              actionButtons={lifecycleActions}
              actionError={actionError}
            />

            <ScenarioSettingsSection
              localGreenfield={localGreenfield}
              onGreenfieldToggle={handleGreenfieldToggle}
              updatePending={updateMutation.isPending}
              updateError={updateMutation.isError}
            />

            <ScenarioCliHints name={name} variant="desktop" />

            <ScenarioDangerZone
              onDeleteClick={handleDeleteClick}
              deletePending={deleteMutationInternal.isPending}
              deleteError={deleteMutationInternal.isError}
            />
          </div>

          <BottomSheet
            isOpen={showActionsSheet}
            onClose={() => setShowActionsSheet(false)}
            title="Scenario Actions"
            description="Manage scenario lifecycle"
            data-testid="scenario-mobile-actions-sheet"
          >
            <ScenarioLifecycleActions
              isRunning={isRunning}
              isStopped={isStopped}
              actionPending={actionMutation.isPending}
              actionInFlight={actionInFlight}
              onAction={(action) => actionMutation.mutate(action)}
              mobile
              onCloseSheet={() => setShowActionsSheet(false)}
            />
          </BottomSheet>
        </>
      )}

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
