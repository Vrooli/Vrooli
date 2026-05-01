import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { selectors } from "../../consts/selectors";
import type { Initiative } from "../../types";
import { AcceptanceCriteriaEditor } from "./operating-mode/acceptance-criteria-editor";
import { ArtifactList } from "./operating-mode/artifact-list";
import { ModeSwitchControl } from "./operating-mode/mode-switch-control";
import { PhaseControls } from "./operating-mode/phase-controls";
import { RoundTimeline } from "./operating-mode/round-timeline";
import { useOperatingModeWorkspace } from "./operating-mode/use-operating-mode-workspace";

export function OperatingModePanel({
  initiative,
  onInitiativeUpdated,
}: {
  initiative: Initiative;
  onInitiativeUpdated: () => void;
}) {
  const workspaceState = useOperatingModeWorkspace({ initiative, onInitiativeUpdated });
  const {
    selectedMode,
    setSelectedMode,
    criteriaText,
    setCriteriaText,
    phaseNote,
    setPhaseNote,
    confirmItemCancellation,
    setConfirmItemCancellation,
    workspaceQuery,
    modeCatalogQuery,
    workspace,
    currentMode,
    switchingAwayFromItemLevel,
    runningRound,
    phaseBusy,
    canRunPhases,
    modeMutation,
    criteriaMutation,
    startMutation,
    refreshMutation,
    cancelMutation,
    completeItemsMutation,
    applyBacklogSyncMutation,
  } = workspaceState;

  return (
    <div className="space-y-5" data-testid={selectors.initiativeDetails.modePanel}>
      <ModeSwitchControl
        currentMode={currentMode}
        selectedMode={selectedMode}
        confirmItemCancellation={confirmItemCancellation}
        switchingAwayFromItemLevel={switchingAwayFromItemLevel}
        isPending={modeMutation.isPending}
        error={modeMutation.error}
        catalogModes={modeCatalogQuery.data?.modes ?? []}
        catalogLoading={modeCatalogQuery.isLoading}
        catalogError={modeCatalogQuery.error}
        onSelectedModeChange={setSelectedMode}
        onConfirmItemCancellationChange={setConfirmItemCancellation}
        onSave={() => modeMutation.mutate(selectedMode)}
      />

      <AcceptanceCriteriaEditor
        value={criteriaText}
        isPending={criteriaMutation.isPending}
        onChange={setCriteriaText}
        onSave={() => criteriaMutation.mutate()}
      />

      {workspaceQuery.isLoading && <PageLoadingState label="Loading mode workspace..." />}
      {workspaceQuery.error && (
        <ErrorState
          title="Failed to load mode workspace"
          error={workspaceQuery.error}
          onRetry={() => void workspaceQuery.refetch()}
        />
      )}

      {workspace && !workspace.definition.capabilities.supportsPhases && (
        <div className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-400">
          This mode uses the existing backlog item execution and review flow. Switch to a phase-capable mode to run initiative-scoped phases.
        </div>
      )}

      {workspace && workspace.definition.capabilities.supportsPhases && (
        <>
          <PhaseControls
            workspace={workspace}
            runningRound={runningRound}
            phaseNote={phaseNote}
            canRunPhases={canRunPhases}
            phaseBusy={phaseBusy}
            startError={startMutation.error}
            completeItemsError={completeItemsMutation.error}
            applyBacklogSyncError={applyBacklogSyncMutation.error}
            onPhaseNoteChange={setPhaseNote}
            onStartPhase={(phase) => startMutation.mutate(phase)}
          />

          <ArtifactList artifacts={workspace.artifacts} />

          <RoundTimeline
            rounds={workspace.rounds}
            capabilities={workspace.definition.capabilities}
            busy={phaseBusy}
            onRefresh={(target) => refreshMutation.mutate(target)}
            onCancel={(target) => cancelMutation.mutate(target)}
            onCompleteItems={(target, itemRefs) => completeItemsMutation.mutate({ round: target, itemRefs })}
            onApplyBacklogSync={(target, mutationIds) => applyBacklogSyncMutation.mutate({ round: target, mutationIds })}
          />
        </>
      )}
    </div>
  );
}
