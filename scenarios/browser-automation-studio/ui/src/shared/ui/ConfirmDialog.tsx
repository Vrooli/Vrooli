import { X } from "lucide-react";
import type { ConfirmDialogState } from "@hooks/useConfirmDialog";
import ResponsiveDialog from "@shared/layout/ResponsiveDialog";

interface ConfirmDialogProps {
  /** Dialog state from useConfirmDialog hook */
  state: ConfirmDialogState | null;
  /** Called when dialog should close with result */
  onClose: (result: boolean) => void;
}

/**
 * Reusable confirmation dialog component.
 * Use with the useConfirmDialog hook.
 *
 * @example
 * ```tsx
 * const { dialogState, confirm, close } = useConfirmDialog();
 *
 * return (
 *   <>
 *     <button onClick={() => confirm({ title: "Delete?" })}>Delete</button>
 *     <ConfirmDialog state={dialogState} onClose={close} />
 *   </>
 * );
 * ```
 */
export function ConfirmDialog({ state, onClose }: ConfirmDialogProps) {
  if (!state) return null;

  return (
    <ResponsiveDialog
      isOpen={state.isOpen}
      onDismiss={() => onClose(false)}
      ariaLabel={state.title}
      role="alertdialog"
    >
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-surface">{state.title}</h2>
        <button
          type="button"
          onClick={() => onClose(false)}
          className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
          aria-label="Close"
        >
          <X size={16} />
        </button>
      </div>
      {state.message && (
        <p className="text-sm text-gray-200">{state.message}</p>
      )}
      <div className="mt-6 flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={() => onClose(false)}
          className="px-4 py-2 rounded-lg border border-gray-700 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
        >
          {state.cancelLabel ?? "Cancel"}
        </button>
        <button
          type="button"
          onClick={() => onClose(true)}
          className={`px-4 py-2 rounded-lg text-white transition-colors ${
            state.danger
              ? "bg-red-600 hover:bg-red-500"
              : "bg-flow-accent hover:bg-blue-600"
          }`}
        >
          {state.confirmLabel ?? "Confirm"}
        </button>
      </div>
    </ResponsiveDialog>
  );
}

export default ConfirmDialog;
