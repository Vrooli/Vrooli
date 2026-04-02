/**
 * BacklogNotesPanel
 *
 * Displays readiness information, workshop state, locked/blocked warnings,
 * and the workshop panel for a backlog item.
 *
 * Reads shared state (isLocked, isTerminal, etc.) from BacklogDetailContext.
 */

import { useState } from "react";
import { ReadinessDetailsPanel } from "./readiness-details-panel";
import { WorkshopPanel } from "./workshop-panel";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { formatBacklogStatus } from "../../types";
import type { WorkshopRound } from "../../types/domain";
import type { ReadinessIndicatorData } from "../../lib/maturity";
import { useDetailSelectionStore } from "../../stores";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";

export interface BacklogNotesPanelProps {
  readinessData: ReadinessIndicatorData | null;
  workshopRounds: WorkshopRound[];
  isSavingWorkshop: boolean;
  isDeletingRound: boolean;
  onStartRun: () => void;
  onSaveRound: (roundNumber: number, content: string) => void;
  onRunWorkshop: () => void;
  onFinalizeWorkshop: () => void;
  onInitializeWorkshop: () => void;
  onDeleteRound: (roundNumber: number) => void;
}

export function BacklogNotesPanel({
  readinessData,
  workshopRounds,
  isSavingWorkshop,
  isDeletingRound,
  onStartRun,
  onSaveRound,
  onRunWorkshop,
  onFinalizeWorkshop,
  onInitializeWorkshop,
  onDeleteRound,
}: BacklogNotesPanelProps) {
  const {
    backlogKind, item, itemActions, isLocked, isTerminal,
    agentRunningLabel, workshopActionLabel, deliverableLabel,
    isWorkshopFinalized, workshopBlockedDeps, isRunningAgent,
    agentRunIsActive,
  } = useBacklogDetail();

  const [showForceWorkshopConfirm, setShowForceWorkshopConfirm] = useState(false);
  const canRun = !!itemActions?.canRun;
  const canFinalize = !!itemActions?.canFinalize;
  const finalizeDisabled = !!itemActions?.finalizeDisabled;
  const blocked = !!itemActions?.blocked;

  return (
    <div className="space-y-3 mt-4 border-t border-slate-800 pt-4">
      {readinessData && !isTerminal && (
        <ReadinessDetailsPanel
          data={readinessData}
          kind={backlogKind}
          onRun={canRun ? onStartRun : undefined}
        />
      )}
      {isLocked && (
        <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-2 text-sm text-amber-300">
          This item is {item?.status ? formatBacklogStatus(item.status) : "locked"} and cannot be edited.
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
          onInitializeWorkshop();
        }}
      />
      <WorkshopPanel
        rounds={workshopRounds}
        backlogKind={backlogKind}
        backlogName={item?.name ?? ""}
        disabled={isLocked || isTerminal}
        isSaving={isSavingWorkshop}
        isRunningWorkshop={isRunningAgent || agentRunIsActive}
        onSaveRound={onSaveRound}
        primaryActionLabel={(!isTerminal && !blocked && (canFinalize || finalizeDisabled)) ? `Finalize ${deliverableLabel}` : undefined}
        onPrimaryAction={(!isTerminal && !blocked && (canFinalize || finalizeDisabled)) ? onFinalizeWorkshop : undefined}
        onRunWorkshop={!isTerminal && !blocked ? onRunWorkshop : undefined}
        workshopActionLabel={workshopActionLabel}
        onDeleteRound={isTerminal ? undefined : onDeleteRound}
        isDeletingRound={isDeletingRound}
        isFinalized={isWorkshopFinalized}
        deliverableLabel={deliverableLabel}
        runningLabel={agentRunningLabel}
      />
    </div>
  );
}
