export type ClearAllTarget = 'cookies' | 'localStorage' | 'history' | 'tabs';

interface ClearAllConfirmationModalProps {
  target: ClearAllTarget;
  deleting: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

const TARGET_LABELS: Record<ClearAllTarget, { title: string; description: string }> = {
  cookies: {
    title: 'Cookies',
    description: 'cookies',
  },
  localStorage: {
    title: 'LocalStorage',
    description: 'localStorage data',
  },
  history: {
    title: 'History',
    description: 'navigation history',
  },
  tabs: {
    title: 'Tabs',
    description: 'saved tabs',
  },
};

export function ClearAllConfirmationModal({
  target,
  deleting,
  onConfirm,
  onCancel,
}: ClearAllConfirmationModalProps) {
  const { title, description } = TARGET_LABELS[target];

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/50"
      onClick={onCancel}
    >
      <div
        className="bg-white dark:bg-gray-900 rounded-lg shadow-xl p-6 max-w-sm mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
          Clear All {title}?
        </h3>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
          This will permanently delete all {description} from this session. This action cannot be
          undone.
        </p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={deleting}
            className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            {deleting ? 'Deleting...' : 'Clear All'}
          </button>
        </div>
      </div>
    </div>
  );
}
