import { useMemo, useState } from 'react';
import type { RecordedAction } from './types';
import type { PreviewMode } from './hooks/usePreviewMode';
import type { SnapshotPreviewState } from './hooks/useSnapshotPreview';

interface RecordPreviewPanelProps {
  mode: PreviewMode;
  onModeChange: (mode: PreviewMode) => void;
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  snapshot: SnapshotPreviewState;
  onLiveBlocked: () => void;
  actions: RecordedAction[];
}

export function RecordPreviewPanel({
  mode,
  onModeChange,
  previewUrl,
  onPreviewUrlChange,
  snapshot,
  onLiveBlocked,
  actions,
}: RecordPreviewPanelProps) {
  const [iframeKey, setIframeKey] = useState(0);

  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  const effectiveUrl = previewUrl || lastUrl;

  return (
    <div className="flex flex-col h-full">
      {/* Header with mode toggle and URL input */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-3 flex items-center gap-3">
        <div className="flex items-center rounded-lg bg-gray-100 dark:bg-gray-800 p-1 text-sm">
          <button
            className={`px-3 py-1.5 rounded-md transition-colors ${mode === 'live' ? 'bg-white dark:bg-gray-700 shadow-sm' : 'text-gray-500'}`}
            onClick={() => onModeChange('live')}
          >
            Live iframe
          </button>
          <button
            className={`px-3 py-1.5 rounded-md transition-colors ${mode === 'snapshot' ? 'bg-white dark:bg-gray-700 shadow-sm' : 'text-gray-500'}`}
            onClick={() => onModeChange('snapshot')}
          >
            Snapshot
          </button>
        </div>

        <div className="flex-1 flex items-center gap-2">
          <input
            className="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800"
            placeholder={lastUrl || 'Enter a URL to preview'}
            value={previewUrl}
            onChange={(e) => onPreviewUrlChange(e.target.value)}
          />
          <button
            className="px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
            onClick={() => {
              if (mode === 'snapshot') {
                void snapshot.refresh();
              } else {
                // Force iframe reload by bumping key
                setIframeKey((k) => k + 1);
              }
            }}
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-hidden">
        {mode === 'live' ? (
          effectiveUrl ? (
            <iframe
              key={iframeKey}
              src={effectiveUrl}
              className="w-full h-full bg-white"
              sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-pointer-lock allow-modals"
              onError={onLiveBlocked}
              title="Live preview"
            />
          ) : (
            <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the page directly. Some sites may block embedding." />
          )
        ) : (
          <SnapshotView snapshot={snapshot} url={effectiveUrl} />
        )}
      </div>
    </div>
  );
}

function SnapshotView({ snapshot, url }: { snapshot: SnapshotPreviewState; url: string | null | undefined }) {
  if (!url) {
    return (
      <EmptyState
        title="No URL available"
        subtitle="Enter a URL or record a navigation to capture a snapshot."
      />
    );
  }

  if (snapshot.isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-gray-500">
        <svg className="w-6 h-6 animate-spin mb-2" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <p className="text-sm">Capturing snapshot...</p>
      </div>
    );
  }

  if (snapshot.error) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-center px-6">
        <p className="text-sm font-medium text-red-600 dark:text-red-400">Snapshot failed</p>
        <p className="text-xs text-gray-500 mt-1">{snapshot.error}</p>
        <button
          className="mt-3 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600"
          onClick={() => snapshot.refresh()}
        >
          Retry
        </button>
      </div>
    );
  }

  if (!snapshot.imageUrl) {
    return (
      <EmptyState
        title="No snapshot yet"
        subtitle="Click refresh to capture the current page state."
        actionLabel="Capture"
        onAction={() => snapshot.refresh()}
      />
    );
  }

  return (
    <div className="h-full w-full overflow-auto bg-gray-100 dark:bg-gray-900">
      <img src={snapshot.imageUrl} alt="Snapshot preview" className="block w-full h-auto" />
      <div className="px-4 py-2 text-xs text-gray-500 flex items-center justify-between bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700">
        <span>Last updated: {snapshot.lastUpdated ? new Date(snapshot.lastUpdated).toLocaleTimeString() : '—'}</span>
        <button
          className="text-blue-600 dark:text-blue-400 hover:underline"
          onClick={() => snapshot.refresh()}
        >
          Refresh
        </button>
      </div>
    </div>
  );
}

function EmptyState({ title, subtitle, actionLabel, onAction }: { title: string; subtitle?: string; actionLabel?: string; onAction?: () => void }) {
  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      <svg className="w-10 h-10 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2" />
      </svg>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-200">{title}</p>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
      {actionLabel && onAction && (
        <button
          className="mt-3 px-3 py-1.5 text-xs font-medium text-white bg-blue-500 rounded-lg hover:bg-blue-600"
          onClick={onAction}
        >
          {actionLabel}
        </button>
      )}
    </div>
  );
}
