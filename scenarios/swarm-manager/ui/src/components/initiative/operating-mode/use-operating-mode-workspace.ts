import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { initiativeModeService, initiativeService } from "../../../services";
import type { Initiative, InitiativeOperatingMode } from "../../../types";
import type { OperatingModeRound } from "../../../types/operating-mode";
import { activeRound, parseAcceptanceCriteria, serializeAcceptanceCriteria } from "./utils";
import {
  buildPhaseEnvelope,
  type PhaseQuickActionKey,
} from "./phase-composer-envelope";

export function useOperatingModeWorkspace({
  initiative,
  onInitiativeUpdated,
}: {
  initiative: Initiative;
  onInitiativeUpdated: () => void;
}) {
  const queryClient = useQueryClient();
  const [criteriaText, setCriteriaText] = useState(serializeAcceptanceCriteria(initiative.acceptanceCriteria ?? []));

  // Phase composer state. Replaces the prior plain-string `phaseNote` so
  // chip + item-picker selections compose into the envelope sent as the
  // server-side `note` field. The composer is envelope-agnostic — it
  // hands raw note text up via onStart; this hook composes the envelope.
  const [pendingPhase, setPendingPhase] = useState<string | null>(null);
  const [selectedActions, setSelectedActions] = useState<Set<PhaseQuickActionKey>>(new Set());
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [pickerOpen, setPickerOpen] = useState(false);
  const [composerNote, setComposerNote] = useState("");

  useEffect(() => {
    setCriteriaText(serializeAcceptanceCriteria(initiative.acceptanceCriteria ?? []));
  }, [initiative.acceptanceCriteria]);

  // Reset transient composer state on initiative change so a half-composed
  // start doesn't bleed into a different initiative.
  useEffect(() => {
    setPendingPhase(null);
    setSelectedActions(new Set());
    setSelectedItems(new Set());
    setPickerOpen(false);
    setComposerNote("");
  }, [initiative.name, initiative.mode]);

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
    mutationFn: ({ mode, cancelActiveItemExecutions }: { mode: InitiativeOperatingMode; cancelActiveItemExecutions: boolean }) =>
      initiativeModeService.switchMode(initiative.name, {
        mode,
        cancelActiveItemExecutions,
      }),
    onSuccess: () => {
      onInitiativeUpdated();
      invalidateWorkspace();
    },
  });

  const criteriaMutation = useMutation({
    mutationFn: () => initiativeService.updateMetadata(initiative.name, {
      acceptanceCriteria: parseAcceptanceCriteria(criteriaText),
    }),
    onSuccess: onInitiativeUpdated,
  });

  const startMutation = useMutation({
    mutationFn: ({ phase, note }: { phase: string; note: string }) => {
      // Compose the envelope here (not in the composer) so any future wire
      // changes are centralized. The skill consumes the note text — empty
      // selection / actions blocks signal "raw note only".
      const envelope = buildPhaseEnvelope({
        phase,
        items: [...selectedItems],
        actions: [...selectedActions],
        note,
      });
      return initiativeModeService.startPhase(initiative.name, phase, { note: envelope });
    },
    onSuccess: () => {
      setComposerNote("");
      setSelectedActions(new Set());
      setSelectedItems(new Set());
      setPickerOpen(false);
      setPendingPhase(null);
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
  const catalogModes = modeCatalogQuery.data?.modes ?? [];
  const currentModeEntry = catalogModes.find((mode) => mode.mode === currentMode);
  const runningRound = useMemo(() => activeRound(workspace?.rounds ?? []), [workspace?.rounds]);
  const phaseBusy = startMutation.isPending ||
    refreshMutation.isPending ||
    cancelMutation.isPending ||
    completeItemsMutation.isPending ||
    applyBacklogSyncMutation.isPending;
  const canRunPhases = Boolean(workspace?.definition.capabilities.canStartPhases) && !runningRound;

  return {
    criteriaText,
    setCriteriaText,
    pendingPhase,
    setPendingPhase,
    selectedActions,
    setSelectedActions,
    selectedItems,
    setSelectedItems,
    pickerOpen,
    setPickerOpen,
    composerNote,
    setComposerNote,
    workspaceQuery,
    modeCatalogQuery,
    workspace,
    currentMode,
    currentModeEntry,
    catalogModes,
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
