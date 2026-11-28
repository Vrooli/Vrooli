/**
 * useBacklogCRUDHandlers
 *
 * Create, update, delete, and status-change handlers for
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
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

export function useBacklogCRUDHandlers(opts: UseBacklogCRUDHandlersOptions) {
  const { data, closeDetail } = opts;
  const { _mutations } = data;

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
    handleArchiveItem,
    handleUnarchiveItem,
  } as const;
}
