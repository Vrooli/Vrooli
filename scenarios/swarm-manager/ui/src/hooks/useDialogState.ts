/**
 * useDialogState – manages the open/mode/editing state triple for CRUD dialogs.
 *
 * Replaces the repeated pattern of three separate useState calls
 * (open boolean + "create"|"edit" mode + editing entity) with a single
 * hook that guarantees atomic state transitions.
 */

import { useState, useCallback } from "react";

export interface UseDialogStateReturn<TEditing> {
  /** Whether the dialog is currently visible. */
  isOpen: boolean;
  /** Whether we are creating a new entity or editing an existing one. */
  mode: "create" | "edit";
  /** The entity being edited, or null when creating. */
  editing: TEditing | null;
  /** Open the dialog in "create" mode (editing = null). */
  openCreate: () => void;
  /** Open the dialog in "edit" mode with the given entity. */
  openEdit: (item: TEditing) => void;
  /** Close the dialog and clear editing state. */
  close: () => void;
}

interface DialogInternalState<T> {
  isOpen: boolean;
  mode: "create" | "edit";
  editing: T | null;
}

const CLOSED_STATE: DialogInternalState<never> = { isOpen: false, mode: "create", editing: null };

export function useDialogState<TEditing = unknown>(): UseDialogStateReturn<TEditing> {
  const [state, setState] = useState<DialogInternalState<TEditing>>(
    CLOSED_STATE as DialogInternalState<TEditing>,
  );

  const openCreate = useCallback(() => {
    setState({ isOpen: true, mode: "create", editing: null });
  }, []);

  const openEdit = useCallback((item: TEditing) => {
    setState({ isOpen: true, mode: "edit", editing: item });
  }, []);

  const close = useCallback(() => {
    setState(CLOSED_STATE as DialogInternalState<TEditing>);
  }, []);

  return {
    isOpen: state.isOpen,
    mode: state.mode,
    editing: state.editing,
    openCreate,
    openEdit,
    close,
  };
}
