/**
 * useBacklogCRUDHandlers
 *
 * Create, update, delete, status change, agent, and workshop handlers for
 * BacklogDetailsPage. Extracted from useBacklogHandlers for modularity.
 */

import { useCallback } from "react";
import { useBacklogDetailUIStore } from "../stores";
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
    (mode: "workshop" | "finalize", prompt: string) => {
      if (!backlogKind || !name) return;
      handleAgentSubmit({ mode, prompt });
    },
    [backlogKind, name, handleAgentSubmit],
  );

  const handleRunWorkshop = useCallback(() => {
    startWorkshopMode("workshop", "Run the next workshop round for this backlog item.");
  }, [startWorkshopMode]);

  const handleFinalizeWorkshop = useCallback(() => {
    startWorkshopMode(
      "finalize",
      `Finalize the latest workshop answers into the ${deliverableLabelLower} for this backlog item.`,
    );
  }, [deliverableLabelLower, startWorkshopMode]);

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
  } as const;
}
