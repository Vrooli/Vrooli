// DOC: docs/internal/SEAMS.md#operating-mode-panel

import { useState } from "react";
import { Activity, FileBox, FileText, History, Workflow } from "lucide-react";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../types";
import type { OperatingModeEvidenceRecord } from "../../types/operating-mode";
import { useUrlState } from "../../hooks/use-url-state";
import { AcceptanceCriteriaEditor } from "./operating-mode/acceptance-criteria-editor";
import { ArtifactList } from "./operating-mode/artifact-list";
import { ItemLevelEmptyState } from "./operating-mode/item-level-empty-state";
import { ModePickerDialog } from "./operating-mode/mode-picker-dialog";
import { OperatingModeHero } from "./operating-mode/operating-mode-hero";
import { PhaseComposer } from "./operating-mode/phase-composer";
import { RoundTimeline } from "./operating-mode/round-timeline";
import { useOperatingModeWorkspace } from "./operating-mode/use-operating-mode-workspace";
import { HowToChooseDialog } from "./operating-mode/how-to-choose-dialog";
import { OrientationBanner } from "./operating-mode/orientation-banner";
import { useTransientHighlight } from "../../hooks/useTransientHighlight";

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
  const [howToChooseOpen, setHowToChooseOpen] = useState(false);
  const [pendingSelectedMode, setPendingSelectedMode] = useState<Initiative["mode"] | null>(null);
  const [orientation, setOrientation] = useState<{
    title: string;
    description: string;
    targetSelector: string;
  } | null>(null);

  // Highlight the orientation target whenever a banner is set. The hook
  // looks the element up by attribute so it survives component refactors;
  // it scrolls into view + applies the transient highlight class for the
  // configured duration.
  useTransientHighlight({
    targetSelector: orientation?.targetSelector ?? null,
    highlightClass: "ring-2 ring-cyan-400/60 transition-shadow",
  });

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
  const evidenceByRun = (workspace?.executions ?? []).reduce<Record<string, OperatingModeEvidenceRecord[]>>((byRun, execution) => {
    for (const record of execution.evidence ?? []) {
      if (!record.runId) continue;
      (byRun[record.runId] ??= []).push(record);
    }
    return byRun;
  }, {});

  return (
    <div className="space-y-2" data-testid={selectors.initiativeDetails.modePanel}>
      <OperatingModeHero
        currentMode={currentMode}
        catalogEntry={currentModeEntry}
        runningRound={runningRound}
        onSwitchClick={openPicker}
      />

      {orientation && (
        <OrientationBanner
          title={orientation.title}
          description={orientation.description}
          onDismiss={() => setOrientation(null)}
        />
      )}

      {workspaceQuery.isLoading && <PageLoadingState label="Loading mode workspace..." />}
      {workspaceQuery.error && (
        <ErrorState
          title="Failed to load mode workspace"
          error={workspaceQuery.error}
          onRetry={() => void workspaceQuery.refetch()}
        />
      )}

      {capabilities?.requiresAcceptanceCriteria && (
        <div data-orientation-target="acceptance-criteria" className="rounded-lg">
          <DetailSection title="Acceptance Criteria" icon={FileText}>
            <AcceptanceCriteriaEditor
              value={criteriaText}
              saved={initiative.acceptanceCriteria ?? []}
              isPending={criteriaMutation.isPending}
              onChange={setCriteriaText}
              onSave={() => criteriaMutation.mutate()}
            />
          </DetailSection>
        </div>
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
        <div data-orientation-target="phase-composer" className="rounded-lg">
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
        </div>
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
            evidenceByRun={evidenceByRun}
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
        onOpenHowToChoose={() => setHowToChooseOpen(true)}
        pendingSelectedMode={pendingSelectedMode}
        onConfirm={(mode, cancelActiveItemExecutions) => {
          modeMutation.mutate(
            { mode, cancelActiveItemExecutions },
            {
              onSuccess: () => {
                closePicker();
                const targetEntry = catalogModes.find((entry) => entry.mode === mode);
                const requiresCriteria = Boolean(
                  targetEntry?.capabilities.requiresAcceptanceCriteria,
                );
                const hasCriteria = (initiative.acceptanceCriteria ?? []).length > 0;
                if (requiresCriteria && !hasCriteria) {
                  setOrientation({
                    title: `You're now in ${targetEntry?.label ?? mode}`,
                    description:
                      "This mode reviews against acceptance criteria. Set them next, then start the first phase.",
                    targetSelector: "[data-orientation-target='acceptance-criteria']",
                  });
                } else if (targetEntry?.capabilities.supportsPhases) {
                  setOrientation({
                    title: `You're now in ${targetEntry?.label ?? mode}`,
                    description: "Start the first phase from the composer below.",
                    targetSelector: "[data-orientation-target='phase-composer']",
                  });
                }
              },
            },
          );
        }}
      />
      <HowToChooseDialog
        isOpen={howToChooseOpen}
        onClose={() => setHowToChooseOpen(false)}
        catalog={catalogModes}
        onPickRecommendation={(mode) => {
          // Land the operator on the recommended mode in the picker, then
          // close how-to-choose. The picker stays open if it already was.
          setPendingSelectedMode(mode);
          if (!isPickerOpen) {
            openPicker();
          }
        }}
      />
    </div>
  );
}
