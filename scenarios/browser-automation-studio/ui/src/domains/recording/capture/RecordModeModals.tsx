import type { ReplayPreviewResponse } from '../types/types';

export function ClearActionsModal({
  open,
  actionCount,
  onCancel,
  onConfirm,
}: {
  open: boolean;
  actionCount: number;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-sm mx-4">
        <div className="p-6">
          <div className="flex items-center justify-center w-12 h-12 mx-auto bg-red-100 dark:bg-red-900/30 rounded-full mb-4">
            <svg className="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </div>
          <h3 className="text-lg font-semibold text-center text-gray-900 dark:text-white mb-2">
            Clear all actions?
          </h3>
          <p className="text-sm text-center text-gray-600 dark:text-gray-400 mb-6">
            This will permanently delete all {actionCount} recorded actions. This action cannot be undone.
          </p>
          <div className="flex gap-3">
            <button
              onClick={onCancel}
              className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={onConfirm}
              className="flex-1 px-4 py-2 text-sm font-medium text-white bg-red-500 rounded-lg hover:bg-red-600 transition-colors"
            >
              Clear All
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function GenerateWorkflowModal({
  open,
  workflowName,
  onWorkflowNameChange,
  hasUnstableSelectors,
  actionCount,
  generateError,
  isLoading,
  onCancel,
  onGenerate,
}: {
  open: boolean;
  workflowName: string;
  onWorkflowNameChange: (name: string) => void;
  hasUnstableSelectors: boolean;
  actionCount: number;
  generateError: string | null;
  isLoading: boolean;
  onCancel: () => void;
  onGenerate: () => void;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Generate Workflow
          </h2>
        </div>
        <div className="px-6 py-4">
          {hasUnstableSelectors && (
            <div className="mb-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
              <p className="text-xs text-yellow-700 dark:text-yellow-300">
                <strong>Note:</strong> Some actions have unstable selectors. The workflow may fail on replay.
                Consider editing selectors before generating.
              </p>
            </div>
          )}

          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Create a reusable workflow from {actionCount} recorded action{actionCount !== 1 ? 's' : ''}.
          </p>

          <label className="block">
            <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
              Workflow Name
            </span>
            <input
              type="text"
              value={workflowName}
              onChange={(e) => onWorkflowNameChange(e.target.value)}
              placeholder="e.g., Login Flow, Submit Form, etc."
              className="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              autoFocus
            />
          </label>

          {generateError && (
            <div className="mt-3 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
              <p className="text-xs text-red-700 dark:text-red-300">{generateError}</p>
            </div>
          )}
        </div>
        <div className="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onGenerate}
            disabled={!workflowName.trim() || isLoading}
            className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? (
              <>
                <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                Generating...
              </>
            ) : (
              'Generate'
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

export function ReplayResultsModal({
  open,
  isReplaying,
  replayResults,
  replayError,
  onClose,
}: {
  open: boolean;
  isReplaying: boolean;
  replayResults: ReplayPreviewResponse | null;
  replayError: string | null;
  onClose: () => void;
}) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-white">
            Test Results
          </h2>
          {replayResults && (
            <span
              className={`px-2 py-1 text-xs font-medium rounded-full ${
                replayResults.success
                  ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300'
                  : 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300'
              }`}
            >
              {replayResults.success ? 'All Passed' : 'Failed'}
            </span>
          )}
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4">
          {isReplaying && !replayResults && (
            <div className="flex flex-col items-center justify-center py-8">
              <svg className="w-8 h-8 animate-spin text-blue-500 mb-3" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
              </svg>
              <p className="text-sm text-gray-600 dark:text-gray-400">Running test...</p>
              <p className="text-xs text-gray-500 dark:text-gray-500 mt-1">Replaying recorded actions</p>
            </div>
          )}

          {replayError && (
            <div className="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
              <div className="flex items-start gap-3">
                <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    fillRule="evenodd"
                    d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
                <div>
                  <p className="text-sm font-medium text-red-800 dark:text-red-200">Test Failed</p>
                  <p className="text-xs text-red-600 dark:text-red-400 mt-1">{replayError}</p>
                </div>
              </div>
            </div>
          )}

          {replayResults && (
            <div className="space-y-4">
              <div className="flex items-center gap-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
                <div className="text-center">
                  <p className="text-2xl font-bold text-green-600 dark:text-green-400">{replayResults.passed_actions}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Passed</p>
                </div>
                <div className="text-center">
                  <p className="text-2xl font-bold text-red-600 dark:text-red-400">{replayResults.failed_actions}</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">Failed</p>
                </div>
                <div className="flex-1 text-right">
                  <p className="text-sm text-gray-600 dark:text-gray-400">
                    {(replayResults.total_duration_ms / 1000).toFixed(1)}s total
                  </p>
                  {replayResults.stopped_early && (
                    <p className="text-xs text-yellow-600 dark:text-yellow-400">Stopped on first failure</p>
                  )}
                </div>
              </div>

              <div className="space-y-2">
                {replayResults.results.map((result, index) => (
                  <div
                    key={result.action_id}
                    className={`p-3 rounded-lg border ${
                      result.success
                        ? 'bg-green-50 dark:bg-green-900/10 border-green-200 dark:border-green-800'
                        : 'bg-red-50 dark:bg-red-900/10 border-red-200 dark:border-red-800'
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      {result.success ? (
                        <svg className="w-4 h-4 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                          <path
                            fillRule="evenodd"
                            d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                            clipRule="evenodd"
                          />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                          <path
                            fillRule="evenodd"
                            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                      <span className="text-sm font-medium text-gray-900 dark:text-white">
                        {index + 1}. {result.action_type}
                      </span>
                      <span className="text-xs text-gray-500 dark:text-gray-400 ml-auto">
                        {result.duration_ms}ms
                      </span>
                    </div>
                    {result.error && (
                      <div className="mt-2 pl-6">
                        <p className="text-xs text-red-700 dark:text-red-300">{result.error.message}</p>
                        {result.error.selector && (
                          <code className="block mt-1 text-xs font-mono text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30 px-2 py-1 rounded truncate">
                            {result.error.selector}
                          </code>
                        )}
                        {result.error.code === 'SELECTOR_NOT_FOUND' && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            💡 Try editing the selector to use a more stable attribute
                          </p>
                        )}
                        {result.error.code === 'SELECTOR_AMBIGUOUS' && result.error.match_count && (
                          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                            💡 Found {result.error.match_count} elements. Make the selector more specific.
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="px-6 py-4 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

