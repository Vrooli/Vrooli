import { useCallback, useState, type Dispatch, type SetStateAction } from 'react';

export interface DialogState {
  open: boolean;
  openDialog: () => void;
  closeDialog: () => void;
  toggleDialog: () => void;
  setOpen: Dispatch<SetStateAction<boolean>>;
}

export function useDialogState(initialState = false): DialogState {
  const [open, setOpen] = useState(initialState);

  const openDialog = useCallback(() => {
    setOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setOpen(false);
  }, []);

  const toggleDialog = useCallback(() => {
    setOpen((current) => !current);
  }, []);

  return {
    open,
    openDialog,
    closeDialog,
    toggleDialog,
    setOpen,
  };
}
