import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Activity, CheckCircle2, CheckSquare, FileText, GitBranch, Play, RefreshCw, Save, Square, Workflow } from "lucide-react";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { Textarea } from "../ui/textarea";
import { ErrorState } from "../ui/error-state";
import { PageLoadingState } from "../ui/loading-states";
import { initiativeModeService, initiativeService } from "../../services";
import { formatDisplayText } from "../../lib/format-utils";
import { formatRelativeTime } from "../../lib";
import { selectors } from "../../consts/selectors";
import type { Initiative, InitiativeOperatingMode } from "../../types";
import type { ProposalMutation } from "../../types/feedback";
import type { OperatingModeBacklogSyncResult, OperatingModeRound, OperatingModeWorkspacePhase } from "../../types/operating-mode";

const MODES: Array<{ value: InitiativeOperatingMode; label: string }> = [
  { value: "item-level", label: "Item Level" },
  { value: "holistic-loop", label: "Holistic Loop" },
  { value: "phased-plan-drain", label: "Phased Plan Drain" },
];

function activeRound(rounds: OperatingModeRound[]): OperatingModeRound | undefined {
  return rounds.find((round) => round.status === "reserved" || round.status === "agent_running");
}

function phaseLabel(phase: string): string {
  return formatDisplayText(phase.replace(/_/g, " "));
}

function statusClasses(status: string): string {
  switch (status) {
    case "completed":
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-300";
    case "failed":
      return "border-red-500/30 bg-red-500/10 text-red-300";
    case "canceled":
      return "border-slate-600 bg-slate-800/70 text-slate-300";
    case "agent_running":
      return "border-cyan-500/30 bg-cyan-500/10 text-cyan-300";
    default:
      return "border-amber-500/30 bg-amber-500/10 text-amber-300";
  }
}

function backlogSyncCompletedItems(round: OperatingModeRound): string[] {
  if (round.payload?.backlog_sync) {
    return [];
  }
  const plan = round.payload?.backlog_sync_plan;
  if (!plan || typeof plan !== "object" || Array.isArray(plan)) {
    return [];
  }
  const completedItems = (plan as { completed_items?: unknown; completedItems?: unknown }).completed_items ??
    (plan as { completed_items?: unknown; completedItems?: unknown }).completedItems;
  return Array.isArray(completedItems)
    ? completedItems.filter((item): item is string => typeof item === "string")
    : [];
}

function backlogSyncProposal(round: OperatingModeRound) {
  if (round.payload?.backlog_sync) {
    return undefined;
  }
  const plan = round.payload?.backlog_sync_plan;
  if (!plan || typeof plan !== "object" || Array.isArray(plan)) {
    return undefined;
  }
  const proposal = (plan as { proposal?: unknown }).proposal;
  if (!proposal || typeof proposal !== "object" || Array.isArray(proposal)) {
    return undefined;
  }
  const typedProposal = proposal as { form?: unknown; mutations?: unknown; rationale?: unknown };
  if (typedProposal.form !== "mutation_list" || !Array.isArray(typedProposal.mutations)) {
    return undefined;
  }
  return {
    form: "mutation_list" as const,
    rationale: typeof typedProposal.rationale === "string" ? typedProposal.rationale : undefined,
    mutations: typedProposal.mutations.filter((mutation): mutation is ProposalMutation => {
      return Boolean(mutation) &&
        typeof mutation === "object" &&
        !Array.isArray(mutation) &&
        typeof (mutation as { id?: unknown }).id === "string" &&
        typeof (mutation as { op?: unknown }).op === "string";
    }),
  };
}

function appliedBacklogSync(round: OperatingModeRound): OperatingModeBacklogSyncResult | undefined {
  const raw = round.payload?.backlog_sync;
  return raw && typeof raw === "object" && !Array.isArray(raw)
    ? raw as OperatingModeBacklogSyncResult
    : undefined;
}

function mutationSummary(mutation: ProposalMutation): string {
  const bits: string[] = [];
  if (mutation.target) {
    bits.push(mutation.target);
  }
  if (mutation.item) {
    bits.push(`${mutation.item.kind}/${mutation.item.name}`);
    if (mutation.item.title) {
      bits.push(mutation.item.title);
    }
  }
  if (mutation.patch) {
    const fields = Object.keys(mutation.patch);
    if (fields.length > 0) {
      bits.push(`patch: ${fields.join(", ")}`);
    }
  }
  if (mutation.status) {
    bits.push(`status: ${mutation.status}`);
  }
  if (mutation.priority != null) {
    bits.push(`priority: ${mutation.priority}`);
  }
  if (mutation.from && mutation.to) {
    bits.push(`${mutation.from} -> ${mutation.to}`);
  }
  return bits.join(" | ");
}

function RoundCard({
  round,
  onRefresh,
  onCancel,
  onCompleteItems,
  onApplyBacklogSync,
  busy,
}: {
  round: OperatingModeRound;
  onRefresh: (round: OperatingModeRound) => void;
  onCancel: (round: OperatingModeRound) => void;
  onCompleteItems: (round: OperatingModeRound, itemRefs: string[]) => void;
  onApplyBacklogSync: (round: OperatingModeRound, mutationIds: string[]) => void;
  busy: boolean;
}) {
  const isActive = round.status === "reserved" || round.status === "agent_running";
  const summary = typeof round.payload?.agent_summary === "string" ? round.payload.agent_summary : "";
  const pendingCompletedItems = backlogSyncCompletedItems(round);
  const proposal = useMemo(() => backlogSyncProposal(round), [round]);
  const appliedSync = useMemo(() => appliedBacklogSync(round), [round]);
  const proposalMutationIds = useMemo(() => (proposal?.mutations ?? []).map((mutation) => mutation.id), [proposal]);
  const [selectedMutationIds, setSelectedMutationIds] = useState<Set<string>>(
    () => new Set(proposalMutationIds),
  );
  useEffect(() => {
    setSelectedMutationIds(new Set(proposalMutationIds));
  }, [proposalMutationIds]);
  const canCompleteItems = round.status === "completed" && Boolean(round.runId) && pendingCompletedItems.length > 0;
  const canApplyProposal = round.status === "completed" && Boolean(round.runId) && Boolean(proposal?.mutations.length) && selectedMutationIds.size > 0;
  const toggleMutation = (id: string) => {
    setSelectedMutationIds((previous) => {
      const next = new Set(previous);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  return (
    <div
      className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3"
      data-testid={selectors.initiativeDetails.roundCard}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-semibold text-slate-100">Round {round.round}</span>
            <span className={`rounded-full border px-2 py-0.5 text-[11px] ${statusClasses(round.status)}`}>
              {phaseLabel(round.status)}
            </span>
            <span className="rounded-full border border-slate-700/80 bg-slate-900/80 px-2 py-0.5 text-[11px] text-slate-400">
              {phaseLabel(round.phase)}
            </span>
          </div>
          <p className="mt-1 break-all text-[11px] text-slate-500">
            {round.agentProfileKey}
            {round.runId ? ` • ${round.runId}` : ""}
          </p>
        </div>
        <div className="flex gap-1.5">
          <Button variant="ghost" size="icon" onClick={() => onRefresh(round)} disabled={busy} title="Refresh round">
            <RefreshCw className="h-4 w-4" />
          </Button>
          {isActive && (
            <Button variant="ghost" size="icon" onClick={() => onCancel(round)} disabled={busy} title="Cancel round">
              <Square className="h-4 w-4" />
            </Button>
          )}
          {canCompleteItems && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onCompleteItems(round, pendingCompletedItems)}
              disabled={busy}
              title="Mark planned items complete"
              data-testid={selectors.initiativeDetails.completeItems}
            >
              <CheckSquare className="h-4 w-4" />
            </Button>
          )}
        </div>
      </div>
      {round.error && <p className="mt-2 text-sm text-red-300">{round.error}</p>}
      {summary && <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed text-slate-300">{summary}</p>}
      {round.handoffs && round.handoffs.length > 0 && (
        <div className="mt-3 space-y-2">
          {round.handoffs.map((handoff, index) => (
            <div key={`${round.round}-handoff-${index}`} className="rounded-lg border border-slate-800 bg-slate-950/50 p-2">
              <p className="text-xs font-medium text-slate-300">Handoff {index + 1}</p>
              {handoff.summary && <p className="mt-1 text-sm text-slate-400">{handoff.summary}</p>}
              {handoff.nextStep && <p className="mt-1 text-xs text-cyan-300">Next: {handoff.nextStep}</p>}
            </div>
          ))}
        </div>
      )}
      {pendingCompletedItems.length > 0 && (
        <div className="mt-3 rounded-lg border border-slate-800 bg-slate-950/50 p-2">
          <p className="text-xs font-medium text-slate-300">Backlog sync plan</p>
          <p className="mt-1 break-words text-sm text-slate-400">{pendingCompletedItems.join(", ")}</p>
        </div>
      )}
      {proposal && proposal.mutations.length > 0 && (
        <div
          className="mt-3 space-y-2 rounded-lg border border-slate-800 bg-slate-950/50 p-2"
          data-testid={selectors.initiativeDetails.backlogProposal}
        >
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-xs font-medium text-slate-300">Backlog proposal</p>
              {proposal.rationale && <p className="mt-1 text-xs text-slate-500">{proposal.rationale}</p>}
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onApplyBacklogSync(round, Array.from(selectedMutationIds))}
              disabled={busy || !canApplyProposal}
              data-testid={selectors.initiativeDetails.applyBacklogSync}
            >
              <CheckCircle2 className="mr-1.5 h-4 w-4" />
              Apply {selectedMutationIds.size || 0} of {proposal.mutations.length}
            </Button>
          </div>
          <ul className="space-y-1.5">
            {proposal.mutations.map((mutation) => (
              <li
                key={mutation.id}
                className="rounded-md border border-slate-800 bg-slate-900/70 p-2"
                data-testid={selectors.initiativeDetails.backlogProposalMutation}
              >
                <label className="flex items-start gap-2">
                  <input
                    type="checkbox"
                    checked={selectedMutationIds.has(mutation.id)}
                    onChange={() => toggleMutation(mutation.id)}
                    disabled={busy}
                    className="mt-1 h-3.5 w-3.5 shrink-0 accent-cyan-500"
                    data-testid={selectors.initiativeDetails.backlogProposalMutationToggle}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="flex flex-wrap items-center gap-2 text-xs text-slate-300">
                      <GitBranch className="h-3.5 w-3.5 text-cyan-400" />
                      <span className="font-semibold">{phaseLabel(mutation.op)}</span>
                      <code className="rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-400">{mutation.id}</code>
                    </span>
                    {mutation.rationale && <span className="mt-1 block text-xs text-slate-500">{mutation.rationale}</span>}
                    {mutationSummary(mutation) && <span className="mt-1 block text-[11px] font-mono text-slate-500">{mutationSummary(mutation)}</span>}
                  </span>
                </label>
              </li>
            ))}
          </ul>
        </div>
      )}
      {appliedSync?.proposalResult && (
        <div className="mt-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-2 text-xs text-emerald-200">
          Applied backlog proposal: {appliedSync.proposalResult.applied} applied, {appliedSync.proposalResult.skipped} skipped, {appliedSync.proposalResult.failed} failed.
        </div>
      )}
      {round.generatedAt && <p className="mt-2 text-[11px] text-slate-500">{formatRelativeTime(round.generatedAt)}</p>}
    </div>
  );
}

export function OperatingModePanel({
  initiative,
  onInitiativeUpdated,
}: {
  initiative: Initiative;
  onInitiativeUpdated: () => void;
}) {
  const queryClient = useQueryClient();
  const [selectedMode, setSelectedMode] = useState<InitiativeOperatingMode>(initiative.mode ?? "item-level");
  const [criteriaText, setCriteriaText] = useState((initiative.acceptanceCriteria ?? []).join("\n"));
  const [phaseNote, setPhaseNote] = useState("");
  const [confirmItemCancellation, setConfirmItemCancellation] = useState(false);

  useEffect(() => {
    setSelectedMode(initiative.mode ?? "item-level");
    setCriteriaText((initiative.acceptanceCriteria ?? []).join("\n"));
    setConfirmItemCancellation(false);
  }, [initiative.acceptanceCriteria, initiative.mode]);

  const workspaceQuery = useQuery({
    queryKey: ["initiative-operating-mode", initiative.name],
    queryFn: () => initiativeModeService.workspace(initiative.name),
  });

  const modeMutation = useMutation({
    mutationFn: (mode: InitiativeOperatingMode) => initiativeModeService.switchMode(initiative.name, {
      mode,
      cancelActiveItemExecutions: confirmItemCancellation,
    }),
    onSuccess: () => {
      setConfirmItemCancellation(false);
      onInitiativeUpdated();
      void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] });
    },
  });

  const criteriaMutation = useMutation({
    mutationFn: () => initiativeService.updateMetadata(initiative.name, {
      acceptanceCriteria: criteriaText
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean),
    }),
    onSuccess: onInitiativeUpdated,
  });

  const startMutation = useMutation({
    mutationFn: (phase: string) => initiativeModeService.startPhase(initiative.name, phase, { note: phaseNote }),
    onSuccess: () => {
      setPhaseNote("");
      void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] });
    },
  });

  const refreshMutation = useMutation({
    mutationFn: (round: OperatingModeRound) => initiativeModeService.refreshRound(initiative.name, round.mode, round.round),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] }),
  });

  const cancelMutation = useMutation({
    mutationFn: (round: OperatingModeRound) => initiativeModeService.cancelRound(initiative.name, round.mode, round.round),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] }),
  });

  const completeItemsMutation = useMutation({
    mutationFn: ({ round, itemRefs }: { round: OperatingModeRound; itemRefs: string[] }) => initiativeModeService.completeItems(initiative.name, {
      mode: round.mode,
      round: round.round,
      runId: round.runId ?? "",
      itemRefs,
    }),
    onSuccess: () => {
      onInitiativeUpdated();
      void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] });
    },
  });

  const applyBacklogSyncMutation = useMutation({
    mutationFn: ({ round, mutationIds }: { round: OperatingModeRound; mutationIds: string[] }) => initiativeModeService.applyBacklogSync(initiative.name, {
      mode: round.mode,
      round: round.round,
      runId: round.runId ?? "",
      acceptedMutationIds: mutationIds,
    }),
    onSuccess: () => {
      onInitiativeUpdated();
      void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] });
    },
  });

  const workspace = workspaceQuery.data;
  const currentMode = initiative.mode ?? "item-level";
  const switchingAwayFromItemLevel = currentMode === "item-level" && selectedMode !== "item-level";
  const runningRound = useMemo(() => activeRound(workspace?.rounds ?? []), [workspace?.rounds]);
  const phaseBusy = startMutation.isPending || refreshMutation.isPending || cancelMutation.isPending || completeItemsMutation.isPending || applyBacklogSyncMutation.isPending;
  const canRunPhases = currentMode !== "item-level" && workspace && !runningRound;

  return (
    <div className="space-y-5" data-testid={selectors.initiativeDetails.modePanel}>
      <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
              <Workflow className="h-3.5 w-3.5" />
              Operating Mode
            </div>
            <p className="mt-2 text-sm text-slate-400">
              Current mode: <span className="font-medium text-slate-100">{MODES.find((mode) => mode.value === currentMode)?.label ?? currentMode}</span>
            </p>
          </div>
          <div className="flex min-w-0 flex-col gap-2 sm:w-72">
            <Select
              value={selectedMode}
              onChange={(event) => {
                setSelectedMode(event.target.value as InitiativeOperatingMode);
                setConfirmItemCancellation(false);
              }}
              withChevron
              data-testid={selectors.initiativeDetails.modeSelect}
            >
              {MODES.map((mode) => <option key={mode.value} value={mode.value}>{mode.label}</option>)}
            </Select>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                if (switchingAwayFromItemLevel && !confirmItemCancellation) {
                  setConfirmItemCancellation(true);
                  return;
                }
                modeMutation.mutate(selectedMode);
              }}
              disabled={selectedMode === currentMode || modeMutation.isPending}
              data-testid={selectors.initiativeDetails.modeSave}
            >
              <Save className="mr-1.5 h-4 w-4" />
              {modeMutation.isPending ? "Saving..." : confirmItemCancellation ? "Confirm Switch" : "Save Mode"}
            </Button>
          </div>
        </div>
        {confirmItemCancellation && (
          <p className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
            Switching from item-level mode can cancel active member item executions. Click Confirm Switch to cancel any active item executions and change modes.
          </p>
        )}
        {modeMutation.error && (
          <p className="mt-3 text-sm text-red-300">{modeMutation.error instanceof Error ? modeMutation.error.message : "Failed to save mode."}</p>
        )}
      </div>

      <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4">
        <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
          <FileText className="h-3.5 w-3.5" />
          Acceptance Criteria
        </div>
        <Textarea
          className="mt-3 min-h-28"
          value={criteriaText}
          onChange={(event) => setCriteriaText(event.target.value)}
          placeholder="One acceptance criterion per line."
          data-testid={selectors.initiativeDetails.criteriaInput}
        />
        <div className="mt-3 flex justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={() => criteriaMutation.mutate()}
            disabled={criteriaMutation.isPending}
            data-testid={selectors.initiativeDetails.criteriaSave}
          >
            <Save className="mr-1.5 h-4 w-4" />
            {criteriaMutation.isPending ? "Saving..." : "Save Criteria"}
          </Button>
        </div>
      </div>

      {workspaceQuery.isLoading && <PageLoadingState label="Loading mode workspace..." />}
      {workspaceQuery.error && (
        <ErrorState
          title="Failed to load mode workspace"
          error={workspaceQuery.error as Error}
          onRetry={() => void workspaceQuery.refetch()}
        />
      )}

      {workspace && currentMode === "item-level" && (
        <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-400">
          Item-level mode uses the existing backlog item execution and review flow. Switch to a non-default mode to run initiative-scoped phases.
        </div>
      )}

      {workspace && currentMode !== "item-level" && (
        <>
          <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4">
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
              onChange={(event) => setPhaseNote(event.target.value)}
              placeholder="Optional operator note for the next phase run."
            />
            <div className="mt-3 grid gap-2 sm:grid-cols-2">
              {workspace.definition.phases.map((phase: OperatingModeWorkspacePhase) => (
                <Button
                  key={phase.phase}
                  variant="outline"
                  size="sm"
                  onClick={() => startMutation.mutate(phase.phase)}
                  disabled={!canRunPhases || phaseBusy}
                  data-testid={selectors.initiativeDetails.phaseStart}
                  title={`${phase.profileKey}${phase.writesRepo ? " • writes repo" : ""}`}
                >
                  <Play className="mr-1.5 h-4 w-4" />
                  {phaseLabel(phase.phase)}
                </Button>
              ))}
            </div>
            {startMutation.error && (
              <p className="mt-3 text-sm text-red-300">{startMutation.error instanceof Error ? startMutation.error.message : "Failed to start phase."}</p>
            )}
            {completeItemsMutation.error && (
              <p className="mt-3 text-sm text-red-300">{completeItemsMutation.error instanceof Error ? completeItemsMutation.error.message : "Failed to complete backlog items."}</p>
            )}
            {applyBacklogSyncMutation.error && (
              <p className="mt-3 text-sm text-red-300">{applyBacklogSyncMutation.error instanceof Error ? applyBacklogSyncMutation.error.message : "Failed to apply backlog proposal."}</p>
            )}
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-100">Artifacts</h3>
            {workspace.artifacts.length === 0 ? (
              <p className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500">No declared artifacts for this mode.</p>
            ) : (
              workspace.artifacts.map((artifact) => (
                <div
                  key={artifact.path}
                  className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3"
                  data-testid={selectors.initiativeDetails.artifactCard}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="break-all text-sm font-medium text-slate-100">{artifact.path}</p>
                    <span className="text-[11px] text-slate-500">{artifact.sizeBytes ? `${artifact.sizeBytes} bytes` : artifact.required ? "required" : "optional"}</span>
                  </div>
                  {artifact.content ? (
                    <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap rounded-lg bg-slate-950/70 p-3 text-xs leading-relaxed text-slate-300">
                      {artifact.content}
                    </pre>
                  ) : (
                    <p className="mt-2 text-sm text-slate-500">Not created yet.</p>
                  )}
                </div>
              ))
            )}
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-100">Rounds</h3>
            {workspace.rounds.length === 0 ? (
              <p className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-4 text-sm text-slate-500">No operating-mode rounds have run yet.</p>
            ) : (
              [...workspace.rounds].reverse().map((round) => (
                <RoundCard
                  key={`${round.mode}-${round.round}`}
                  round={round}
                  busy={phaseBusy}
                  onRefresh={(target) => refreshMutation.mutate(target)}
                  onCancel={(target) => cancelMutation.mutate(target)}
                  onCompleteItems={(target, itemRefs) => completeItemsMutation.mutate({ round: target, itemRefs })}
                  onApplyBacklogSync={(target, mutationIds) => applyBacklogSyncMutation.mutate({ round: target, mutationIds })}
                />
              ))
            )}
          </div>
        </>
      )}
    </div>
  );
}
