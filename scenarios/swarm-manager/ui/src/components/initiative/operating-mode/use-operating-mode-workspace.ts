import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { initiativeModeService, initiativeService } from "../../../services";
import type { Initiative, InitiativeOperatingMode } from "../../../types";
import type { OperatingModeRound } from "../../../types/operating-mode";
import { activeRound } from "./utils";

export function useOperatingModeWorkspace({
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
  const modeCatalogQuery = useQuery({
    queryKey: ["operating-mode-catalog"],
    queryFn: () => initiativeModeService.catalog(),
  });

  const invalidateWorkspace = () => {
    void queryClient.invalidateQueries({ queryKey: ["initiative-operating-mode", initiative.name] });
  };

  const modeMutation = useMutation({
    mutationFn: (mode: InitiativeOperatingMode) => initiativeModeService.switchMode(initiative.name, {
      mode,
      cancelActiveItemExecutions: confirmItemCancellation,
    }),
    onSuccess: () => {
      setConfirmItemCancellation(false);
      onInitiativeUpdated();
      invalidateWorkspace();
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
      invalidateWorkspace();
    },
  });

  const refreshMutation = useMutation({
    mutationFn: (round: OperatingModeRound) => initiativeModeService.refreshRound(initiative.name, round.mode, round.round),
    onSuccess: invalidateWorkspace,
  });

  const cancelMutation = useMutation({
    mutationFn: (round: OperatingModeRound) => initiativeModeService.cancelRound(initiative.name, round.mode, round.round),
    onSuccess: invalidateWorkspace,
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
      invalidateWorkspace();
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
      invalidateWorkspace();
    },
  });

  const workspace = workspaceQuery.data;
  const currentMode = initiative.mode ?? "item-level";
  const switchingAwayFromItemLevel = currentMode === "item-level" && selectedMode !== "item-level";
  const runningRound = useMemo(() => activeRound(workspace?.rounds ?? []), [workspace?.rounds]);
  const phaseBusy = startMutation.isPending ||
    refreshMutation.isPending ||
    cancelMutation.isPending ||
    completeItemsMutation.isPending ||
    applyBacklogSyncMutation.isPending;
  const canRunPhases = currentMode !== "item-level" && Boolean(workspace) && !runningRound;

  return {
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
  };
}
