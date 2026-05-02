// DOC: docs/internal/SEAMS.md#operating-mode-panel

import { Activity, FileBox, FileText, History, Workflow } from "lucide-react";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../types";
import { useUrlState } from "../../hooks/use-url-state";
import { AcceptanceCriteriaEditor } from "./operating-mode/acceptance-criteria-editor";
import { ArtifactList } from "./operating-mode/artifact-list";
import { ItemLevelEmptyState } from "./operating-mode/item-level-empty-state";
import { ModePickerDialog } from "./operating-mode/mode-picker-dialog";
import { OperatingModeHero } from "./operating-mode/operating-mode-hero";
import { PhaseComposer } from "./operating-mode/phase-composer";
import { RoundTimeline } from "./operating-mode/round-timeline";
import { useOperatingModeWorkspace } from "./operating-mode/use-operating-mode-workspace";

type PickerState = "open" | "closed";

const PICKER_PARAM_VALIDATE = (v: string): v is PickerState => v === "open" || v === "closed";

export function OperatingModePanel({
  initiative,
  rollup,
  onInitiativeUpdated,
}: {
  initiative: Initiative;
  rollup?: InitiativeRollup;
  onInitiativeUpdated: () => void;
}) {
  const [pickerState, setPickerState] = useUrlState<PickerState>("modePicker", "closed", {
    validate: PICKER_PARAM_VALIDATE,
  });
  const isPickerOpen = pickerState === "open";
  const openPicker = () => setPickerState("open");
  const closePicker = () => setPickerState("closed");

  const ws = useOperatingModeWorkspace({ initiative, onInitiativeUpdated });
  const {
    criteriaText,
    setCriteriaText,
    pendingPhase,
    setPendingPhase,
    selectedActions,
    setSelectedActions,
    selectedItems,
    setSelectedItems,
    pickerOpen: itemPickerOpen,
    setPickerOpen: setItemPickerOpen,
    composerNote,
    setComposerNote,
    workspaceQuery,
    modeCatalogQuery,
    refetchCatalog,
    workspace,
    currentMode,
    currentModeEntry,
    catalogModes,
    runningRound,
    phaseBusy,
    canRunPhases,
    phaseStartDisabledReason,
    modeMutation,
    criteriaMutation,
    startMutation,
    refreshMutation,
    cancelMutation,
    completeItemsMutation,
    applyBacklogSyncMutation,
  } = ws;

  const capabilities = workspace?.definition.capabilities;

  const items = (initiative.items ?? []).map((ref) => ({ ref, title: ref }));

  return (
    <div className="space-y-2" data-testid={selectors.initiativeDetails.modePanel}>
      <OperatingModeHero
        currentMode={currentMode}
        catalogEntry={currentModeEntry}
        runningRound={runningRound}
        onSwitchClick={openPicker}
      />

      {workspaceQuery.isLoading && <PageLoadingState label="Loading mode workspace..." />}
      {workspaceQuery.error && (
        <ErrorState
          title="Failed to load mode workspace"
          error={workspaceQuery.error}
          onRetry={() => void workspaceQuery.refetch()}
        />
      )}

      {capabilities?.requiresAcceptanceCriteria && (
        <DetailSection title="Acceptance Criteria" icon={FileText}>
          <AcceptanceCriteriaEditor
            value={criteriaText}
            saved={initiative.acceptanceCriteria ?? []}
            isPending={criteriaMutation.isPending}
            onChange={setCriteriaText}
            onSave={() => criteriaMutation.mutate()}
          />
        </DetailSection>
      )}

      {capabilities?.usesItemExecutionFlow && (
        <DetailSection title="How Item-Level Works" icon={Workflow} hideDivider>
          <ItemLevelEmptyState
            initiative={initiative}
            rollup={rollup}
            workspace={workspace}
            onSwitchClick={openPicker}
          />
        </DetailSection>
      )}

      {capabilities?.supportsPhases && currentModeEntry && workspace && (
        <DetailSection title="Start a Phase" icon={Activity} hideDivider>
          <PhaseComposer
            catalogEntry={currentModeEntry}
            workspace={workspace}
            runningRound={runningRound}
            items={items}
            pendingPhase={pendingPhase}
            onPendingPhaseChange={setPendingPhase}
            selectedActions={selectedActions}
            onSelectedActionsChange={setSelectedActions}
            selectedItems={selectedItems}
            onSelectedItemsChange={setSelectedItems}
            pickerOpen={itemPickerOpen}
            onPickerOpenChange={setItemPickerOpen}
            note={composerNote}
            onNoteChange={setComposerNote}
            phaseBusy={phaseBusy}
            canRunPhases={canRunPhases}
            phaseStartDisabledReason={phaseStartDisabledReason}
            startError={startMutation.error}
            onStart={(phase, note) => startMutation.mutate({ phase, note })}
          />
        </DetailSection>
      )}

      {capabilities?.supportsArtifacts && capabilities?.supportsPhases && workspace && (
        <DetailSection title="Artifacts" icon={FileBox}>
          <ArtifactList artifacts={workspace.artifacts} />
        </DetailSection>
      )}

      {capabilities?.supportsPhases && workspace && (
        <DetailSection title="Rounds" icon={History}>
          <RoundTimeline
            rounds={workspace.rounds}
            capabilities={capabilities}
            busy={phaseBusy}
            onRefresh={(target) => refreshMutation.mutate(target)}
            onCancel={(target) => cancelMutation.mutate(target)}
            onCompleteItems={(target, itemRefs) =>
              completeItemsMutation.mutate({ round: target, itemRefs })
            }
            onApplyBacklogSync={(target, mutationIds) =>
              applyBacklogSyncMutation.mutate({ round: target, mutationIds })
            }
          />
        </DetailSection>
      )}

      <ModePickerDialog
        isOpen={isPickerOpen}
        onClose={closePicker}
        currentMode={currentMode}
        catalog={catalogModes}
        catalogLoading={modeCatalogQuery.isLoading}
        catalogError={modeCatalogQuery.error}
        catalogFetching={modeCatalogQuery.isFetching}
        onRetryCatalog={() => void refetchCatalog()}
        isMutating={modeMutation.isPending}
        mutationError={modeMutation.error}
        onConfirm={(mode, cancelActiveItemExecutions) => {
          modeMutation.mutate(
            { mode, cancelActiveItemExecutions },
            {
              onSuccess: () => {
                closePicker();
              },
            },
          );
        }}
      />
    </div>
  );
}
