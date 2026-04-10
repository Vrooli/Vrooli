import { useCallback, useRef, useState } from "react";

export interface DeleteProjectDialogOptions {
  projectName: string;
}

export interface DeleteProjectDialogState extends DeleteProjectDialogOptions {
  isOpen: boolean;
  deleteFiles: boolean;
}

export interface DeleteProjectResult {
  confirmed: boolean;
  deleteFiles: boolean;
}

export interface UseDeleteProjectDialogReturn {
  /** Current dialog state - pass to DeleteProjectDialog component */
  dialogState: DeleteProjectDialogState | null;
  /** Request deletion confirmation - returns promise with result */
  confirm: (options: DeleteProjectDialogOptions) => Promise<DeleteProjectResult>;
  /** Update the deleteFiles checkbox state */
  setDeleteFiles: (value: boolean) => void;
  /** Close the dialog with a result */
  close: (confirmed: boolean) => void;
}

/**
 * Hook for managing the delete project dialog with file deletion option.
 *
 * @example
 * ```tsx
 * const { dialogState, confirm, setDeleteFiles, close } = useDeleteProjectDialog();
 *
 * const handleDelete = async () => {
 *   const result = await confirm({ projectName: project.name });
 *   if (result.confirmed) {
 *     await deleteProject(project.id, result.deleteFiles);
 *   }
 * };
 *
 * return (
 *   <>
 *     <button onClick={handleDelete}>Delete</button>
 *     <DeleteProjectDialog
 *       state={dialogState}
 *       onDeleteFilesChange={setDeleteFiles}
 *       onClose={close}
 *     />
 *   </>
 * );
 * ```
 */
export function useDeleteProjectDialog(): UseDeleteProjectDialogReturn {
  const [dialogState, setDialogState] = useState<DeleteProjectDialogState | null>(null);
  const resolveRef = useRef<((value: DeleteProjectResult) => void) | null>(null);

  const confirm = useCallback(
    async (options: DeleteProjectDialogOptions): Promise<DeleteProjectResult> => {
      // If there's a pending dialog, resolve it as cancelled
      if (resolveRef.current) {
        resolveRef.current({ confirmed: false, deleteFiles: false });
        resolveRef.current = null;
      }

      return new Promise<DeleteProjectResult>((resolve) => {
        resolveRef.current = resolve;
        setDialogState({ ...options, isOpen: true, deleteFiles: false });
      });
    },
    []
  );

  const setDeleteFiles = useCallback((value: boolean) => {
    setDialogState((prev) => (prev ? { ...prev, deleteFiles: value } : null));
  }, []);

  const close = useCallback((confirmed: boolean) => {
    const resolve = resolveRef.current;
    const deleteFiles = dialogState?.deleteFiles ?? false;
    resolveRef.current = null;
    setDialogState(null);
    resolve?.({ confirmed, deleteFiles });
  }, [dialogState?.deleteFiles]);

  return { dialogState, confirm, setDeleteFiles, close };
}

export default useDeleteProjectDialog;
