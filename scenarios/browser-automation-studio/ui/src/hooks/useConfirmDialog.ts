import { useCallback, useRef, useState } from "react";

export interface ConfirmDialogOptions {
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

export interface ConfirmDialogState extends ConfirmDialogOptions {
  isOpen: boolean;
}

export interface UseConfirmDialogReturn {
  /** Current dialog state - pass to ConfirmDialog component */
  dialogState: ConfirmDialogState | null;
  /** Request confirmation from user - returns promise that resolves to true/false */
  confirm: (options: ConfirmDialogOptions) => Promise<boolean>;
  /** Close the dialog with a result */
  close: (result: boolean) => void;
}

/**
 * Hook for managing confirm/cancel dialogs.
 *
 * @example
 * ```tsx
 * const { dialogState, confirm, close } = useConfirmDialog();
 *
 * const handleDelete = async () => {
 *   const confirmed = await confirm({
 *     title: "Delete item?",
 *     message: "This action cannot be undone.",
 *     confirmLabel: "Delete",
 *     danger: true,
 *   });
 *   if (confirmed) {
 *     // perform delete
 *   }
 * };
 *
 * return (
 *   <>
 *     <button onClick={handleDelete}>Delete</button>
 *     <ConfirmDialog state={dialogState} onClose={close} />
 *   </>
 * );
 * ```
 */
export function useConfirmDialog(): UseConfirmDialogReturn {
  const [dialogState, setDialogState] = useState<ConfirmDialogState | null>(null);
  const resolveRef = useRef<((value: boolean) => void) | null>(null);

  const confirm = useCallback(async (options: ConfirmDialogOptions): Promise<boolean> => {
    // If there's a pending dialog, resolve it as cancelled
    if (resolveRef.current) {
      resolveRef.current(false);
      resolveRef.current = null;
    }

    return new Promise<boolean>((resolve) => {
      resolveRef.current = resolve;
      setDialogState({ ...options, isOpen: true });
    });
  }, []);

  const close = useCallback((result: boolean) => {
    const resolve = resolveRef.current;
    resolveRef.current = null;
    setDialogState(null);
    resolve?.(result);
  }, []);

  return { dialogState, confirm, close };
}

export default useConfirmDialog;
