import { Activity, Play } from "lucide-react";
import { Button } from "../../ui/button";
import { Textarea } from "../../ui/textarea";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeRound, OperatingModeWorkspace, OperatingModeWorkspacePhase } from "../../../types/operating-mode";
import { phaseLabel } from "./utils";

export function PhaseControls({
  workspace,
  runningRound,
  phaseNote,
  canRunPhases,
  phaseBusy,
  startError,
  completeItemsError,
  applyBacklogSyncError,
  onPhaseNoteChange,
  onStartPhase,
}: {
  workspace: OperatingModeWorkspace;
  runningRound?: OperatingModeRound;
  phaseNote: string;
  canRunPhases: boolean;
  phaseBusy: boolean;
  startError: unknown;
  completeItemsError: unknown;
  applyBacklogSyncError: unknown;
  onPhaseNoteChange: (value: string) => void;
  onStartPhase: (phase: string) => void;
}) {
  const enabled = Boolean(canRunPhases);

  return (
    <div className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
            <Activity className="h-3.5 w-3.5" />
            Phases
          </div>
          <p className="mt-1 text-sm text-slate-400">{workspace.definition.label} • {phaseLabel(workspace.definition.runStrategy)}</p>
        </div>
        {runningRound && (
          <span className="rounded-full border border-cyan-500/30 bg-cyan-500/10 px-2.5 py-1 text-xs text-cyan-300">
            Round {runningRound.round} running
          </span>
        )}
      </div>
      <Textarea
        className="mt-3 min-h-20"
        value={phaseNote}
        onChange={(event) => onPhaseNoteChange(event.target.value)}
        placeholder="Optional operator note for the next phase run."
      />
      <div className="mt-3 grid gap-2 sm:grid-cols-2">
        {workspace.definition.phases.map((phase: OperatingModeWorkspacePhase) => (
          <Button
            key={phase.phase}
            variant="outline"
            size="sm"
            onClick={() => onStartPhase(phase.phase)}
            disabled={!enabled || phaseBusy || !phase.startable}
            data-testid={selectors.initiativeDetails.phaseStart}
            title={phase.startable ? `${phase.profileKey}${phase.writesRepo ? " • writes repo" : ""}` : (phase.reason ?? "Phase is not currently startable")}
          >
            <Play className="mr-1.5 h-4 w-4" />
            {phaseLabel(phase.phase)}
            {phase.next ? <span className="ml-1.5 text-[10px] uppercase text-cyan-300">Next</span> : null}
          </Button>
        ))}
      </div>
      {Boolean(startError) && (
        <p className="mt-3 text-sm text-red-300">{startError instanceof Error ? startError.message : "Failed to start phase."}</p>
      )}
      {Boolean(completeItemsError) && (
        <p className="mt-3 text-sm text-red-300">{completeItemsError instanceof Error ? completeItemsError.message : "Failed to complete backlog items."}</p>
      )}
      {Boolean(applyBacklogSyncError) && (
        <p className="mt-3 text-sm text-red-300">{applyBacklogSyncError instanceof Error ? applyBacklogSyncError.message : "Failed to apply backlog proposal."}</p>
      )}
    </div>
  );
}
