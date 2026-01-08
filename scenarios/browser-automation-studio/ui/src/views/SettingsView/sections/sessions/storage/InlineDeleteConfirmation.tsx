interface InlineDeleteConfirmationProps {
  /** The confirmation message to display */
  message: string;
  /** Label for the confirm button (default: "Delete") */
  confirmLabel?: string;
  /** Label shown while the action is in progress (default: "Deleting...") */
  loadingLabel?: string;
  /** Whether the delete action is in progress */
  deleting?: boolean;
  /** Called when the user confirms the action */
  onConfirm: () => void;
  /** Called when the user cancels */
  onCancel: () => void;
}

/**
 * Inline confirmation bar for delete/remove actions.
 * Displayed inside a card or list item when the user clicks a delete button.
 */
export function InlineDeleteConfirmation({
  message,
  confirmLabel = 'Delete',
  loadingLabel = 'Deleting...',
  deleting,
  onConfirm,
  onCancel,
}: InlineDeleteConfirmationProps) {
  return (
    <div className="px-4 py-3 bg-red-50 dark:bg-red-900/30 border-t border-red-200 dark:border-red-800">
      <div className="flex items-center justify-between">
        <p className="text-sm text-red-700 dark:text-red-300">{message}</p>
        <div className="flex items-center gap-2">
          <button
            onClick={onCancel}
            className="px-3 py-1 text-xs font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={deleting}
            className="px-3 py-1 text-xs font-medium text-white bg-red-600 rounded hover:bg-red-700 disabled:opacity-50"
          >
            {deleting ? loadingLabel : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
