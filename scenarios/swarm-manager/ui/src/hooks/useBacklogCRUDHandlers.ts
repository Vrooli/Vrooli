/**
 * useBacklogCRUDHandlers
 *
 * Create, update, delete, status change, agent, and workshop handlers for
 * BacklogDetailsPage. Extracted from useBacklogHandlers for modularity.
 */

import { useCallback } from "react";
import { useBacklogDetailUIStore, useBacklogStore } from "../stores";
import type { useBacklogDetailData } from "./useBacklogDetailData";
import type { BacklogKind, BacklogStatus } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** The subset of useBacklogDetailData return used by CRUD handlers. */
type BacklogDetailData = ReturnType<typeof useBacklogDetailData>;

export interface UseBacklogCRUDHandlersOptions {
  data: BacklogDetailData;
  backlogKind: BacklogKind | null;
  name: string | undefined;
  closeDetail: () => void;
  refreshActivities: (force: boolean) => Promise<void>;
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useBacklogCRUDHandlers(opts: UseBacklogCRUDHandlersOptions) {
  const { data, backlogKind, name, closeDetail, refreshActivities } = opts;
  const { _mutations, deliverableLabelLower } = data;

  // --- Item CRUD ---

  const handleUpdateItem = useCallback(
    (values: {
      title: string;
      description: string;
      status: BacklogStatus;
      priority: number;
      tags: string[];
    }) => {
      _mutations.update.mutate(values, {
        onSuccess: () => {
          useBacklogDetailUIStore.getState().closeEdit();
        },
      });
    },
    [_mutations.update],
  );

  const handleAcceptanceGlobSave = useCallback(
    (allow: string[], deny: string[]) => {
      _mutations.acceptanceGlob.mutate(
        { acceptanceAllow: allow, acceptanceDeny: deny },
        {
          onSuccess: () => {
            useBacklogDetailUIStore.getState().closeGlob();
          },
        },
      );
    },
    [_mutations.acceptanceGlob],
  );

  const handleDeleteConfirm = useCallback(() => {
    _mutations.delete.mutate(undefined, {
      onSuccess: () => {
        closeDetail();
      },
    });
  }, [_mutations.delete, closeDetail]);

  // --- Agent ---

  const handleAgentSubmit = useCallback(
    (payload: {
      mode?: string;
      prompt: string;
      contextPaths?: string[];
      contextTargetIds?: string[];
      contextRequirementIds?: string[];
      confirm?: boolean;
      force?: boolean;
    }) => {
      _mutations.agent.mutate(payload, {
        onSuccess: () => {
          useBacklogDetailUIStore.getState().closeAgent();
          void refreshActivities(true);
        },
      });
    },
    [_mutations.agent, refreshActivities],
  );

  // --- Workshop ---

  const handleSaveRound = useCallback(
    (roundNumber: number, content: string) => {
      _mutations.workshopSave.mutate(
        { roundNumber, content },
        {
          onSuccess: (result) => {
            if (result.autoAdvance?.triggered && result.autoAdvance?.runId) {
              void refreshActivities(true);
            }
          },
        },
      );
    },
    [_mutations.workshopSave, refreshActivities],
  );

  const handleDeleteWorkshopRound = useCallback(() => {
    const roundToDelete = useBacklogDetailUIStore.getState().roundToDelete;
    if (roundToDelete !== null) {
      _mutations.workshopDeleteRound.mutate(
        { roundNumber: roundToDelete },
        {
          onSuccess: () => {
            useBacklogDetailUIStore.getState().setRoundToDelete(null);
          },
        },
      );
    }
  }, [_mutations.workshopDeleteRound]);

  const handleWorkshopResetConfirm = useCallback(() => {
    _mutations.workshopReset.mutate(undefined, {
      onSuccess: () => {
        useBacklogDetailUIStore.getState().closeWorkshopReset();
      },
    });
  }, [_mutations.workshopReset]);

  const startWorkshopMode = useCallback(
    (mode: "workshop" | "finalize", prompt: string, force?: boolean) => {
      if (!backlogKind || !name) return;
      handleAgentSubmit({ mode, prompt, confirm: true, force });
    },
    [backlogKind, name, handleAgentSubmit],
  );

  const handleRunWorkshop = useCallback(() => {
    // Check if item is blocked — if so, show confirmation dialog
    const blockingMap = useBacklogStore.getState().blockingMap;
    const key = `${backlogKind}/${name}`;
    const info = blockingMap[key];
    if (info?.blocked) {
      useBacklogDetailUIStore.getState().openWorkshopBlockingConfirm("workshop");
      return;
    }
    startWorkshopMode("workshop", "Run the next workshop round for this backlog item.");
  }, [backlogKind, name, startWorkshopMode]);

  const handleFinalizeWorkshop = useCallback(() => {
    // Check if item is blocked — if so, show confirmation dialog
    const blockingMap = useBacklogStore.getState().blockingMap;
    const key = `${backlogKind}/${name}`;
    const info = blockingMap[key];
    if (info?.blocked) {
      useBacklogDetailUIStore.getState().openWorkshopBlockingConfirm("finalize");
      return;
    }
    startWorkshopMode(
      "finalize",
      `Finalize the latest workshop answers into the ${deliverableLabelLower} for this backlog item.`,
    );
  }, [backlogKind, name, deliverableLabelLower, startWorkshopMode]);

  const handleWorkshopBlockingOverride = useCallback(() => {
    const { workshopBlockingConfirm } = useBacklogDetailUIStore.getState();
    const mode = workshopBlockingConfirm.mode;
    const prompt = mode === "finalize"
      ? `Finalize the latest workshop answers into the ${deliverableLabelLower} for this backlog item.`
      : "Run the next workshop round for this backlog item.";
    useBacklogDetailUIStore.getState().closeWorkshopBlockingConfirm();
    startWorkshopMode(mode, prompt, true);
  }, [deliverableLabelLower, startWorkshopMode]);

  const handleArchiveItem = useCallback(() => {
    _mutations.archiveMutation.mutate();
  }, [_mutations.archiveMutation]);

  const handleUnarchiveItem = useCallback(() => {
    _mutations.unarchiveMutation.mutate();
  }, [_mutations.unarchiveMutation]);

  return {
    handleUpdateItem,
    handleAcceptanceGlobSave,
    handleDeleteConfirm,
    handleAgentSubmit,
    handleSaveRound,
    handleDeleteWorkshopRound,
    handleWorkshopResetConfirm,
    handleRunWorkshop,
    handleFinalizeWorkshop,
    handleWorkshopBlockingOverride,
    handleArchiveItem,
    handleUnarchiveItem,
  } as const;
}
