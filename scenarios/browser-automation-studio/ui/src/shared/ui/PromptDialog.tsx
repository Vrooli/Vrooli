import { FormEvent } from "react";
import { X } from "lucide-react";
import type { PromptDialogState } from "@hooks/usePromptDialog";
import ResponsiveDialog from "@shared/layout/ResponsiveDialog";

interface PromptDialogProps {
  /** Dialog state from usePromptDialog hook */
  state: PromptDialogState | null;
  /** Called when input value changes */
  onValueChange: (value: string) => void;
  /** Called when dialog should close (null = cancelled) */
  onClose: (result: string | null) => void;
  /** Called when form is submitted */
  onSubmit: () => void;
}

/**
 * Reusable prompt dialog component with validation support.
 * Use with the usePromptDialog hook.
 *
 * @example
 * ```tsx
 * const { dialogState, prompt, setValue, close, submit } = usePromptDialog();
 *
 * return (
 *   <>
 *     <button onClick={() => prompt({ title: "Name", label: "Enter name" })}>
 *       Rename
 *     </button>
 *     <PromptDialog
 *       state={dialogState}
 *       onValueChange={setValue}
 *       onClose={close}
 *       onSubmit={submit}
 *     />
 *   </>
 * );
 * ```
 */
export function PromptDialog({
  state,
  onValueChange,
  onClose,
  onSubmit,
}: PromptDialogProps) {
  if (!state) return null;

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault();
    onSubmit();
  };

  return (
    <ResponsiveDialog
      isOpen={state.isOpen}
      onDismiss={() => onClose(null)}
      ariaLabel={state.title}
    >
      <form onSubmit={handleSubmit}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface">{state.title}</h2>
          <button
            type="button"
            onClick={() => onClose(null)}
            className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
            aria-label="Close"
          >
            <X size={16} />
          </button>
        </div>
        {state.message && (
          <p className="mb-3 text-sm text-gray-200">{state.message}</p>
        )}
        <label className="block text-sm text-gray-300 mb-2">
          {state.label}
        </label>
        <input
          autoFocus
          value={state.value}
          onChange={(e) => onValueChange(e.target.value)}
          placeholder={state.placeholder}
          className="w-full px-3 py-2 rounded-lg bg-gray-900 border border-gray-700 text-surface placeholder:text-gray-500 focus:outline-none focus:ring-2 focus:ring-flow-accent/60"
        />
        {state.error && (
          <p className="mt-2 text-sm text-red-300">{state.error}</p>
        )}
        <div className="mt-6 flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={() => onClose(null)}
            className="px-4 py-2 rounded-lg border border-gray-700 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
          >
            {state.cancelLabel ?? "Cancel"}
          </button>
          <button
            type="submit"
            className="px-4 py-2 rounded-lg bg-flow-accent text-white hover:bg-blue-600 transition-colors"
          >
            {state.submitLabel ?? "OK"}
          </button>
        </div>
      </form>
    </ResponsiveDialog>
  );
}

export default PromptDialog;
