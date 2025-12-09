import { useEffect, useMemo, useRef, useState } from 'react';
import type { RecordedAction } from './types';
import type { PreviewMode } from './hooks/usePreviewMode';
import type { SnapshotPreviewState } from './hooks/useSnapshotPreview';
import { PlaywrightView } from './components/PlaywrightView';

interface RecordPreviewPanelProps {
  mode: PreviewMode;
  onModeChange: (mode: PreviewMode) => void;
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  snapshot: SnapshotPreviewState;
  onLiveBlocked: () => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  onViewportChange?: (size: { width: number; height: number }) => void;
}

export function RecordPreviewPanel({
  mode,
  onModeChange,
  previewUrl,
  onPreviewUrlChange,
  snapshot,
  onLiveBlocked,
  actions,
  sessionId,
  onViewportChange,
}: RecordPreviewPanelProps) {
  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  const effectiveUrl = previewUrl || lastUrl;

  const [urlInput, setUrlInput] = useState(previewUrl);
  const [liveRefreshToken, setLiveRefreshToken] = useState(0);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const lastReportedViewportRef = useRef<{ width: number; height: number } | null>(null);

  // Keep input in sync with upstream changes
  useEffect(() => {
    setUrlInput(previewUrl);
  }, [previewUrl]);

  // Debounce URL changes before notifying parent
  useEffect(() => {
    const debounce = setTimeout(() => {
      if (urlInput !== previewUrl) {
        onPreviewUrlChange(urlInput);
      }
    }, 400);
    return () => clearTimeout(debounce);
  }, [urlInput, previewUrl, onPreviewUrlChange]);

  // Observe preview container size and notify parent for viewport sizing.
  useEffect(() => {
    const node = previewContainerRef.current;
    if (!node) return;

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      const width = entry.contentRect.width;
      const height = entry.contentRect.height;
      if (width <= 0 || height <= 0) return;

      const size = { width, height };
      const prev = lastReportedViewportRef.current;
      if (prev && Math.abs(prev.width - width) < 1 && Math.abs(prev.height - height) < 1) {
        return;
      }
      lastReportedViewportRef.current = size;
      if (onViewportChange) {
        onViewportChange(size);
      }
    });

    observer.observe(node);
    return () => observer.disconnect();
  }, [previewContainerRef, onViewportChange]);

  return (
    <div className="flex flex-col h-full">
      {/* Header with mode toggle and URL input */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-3 flex items-center gap-3">
        <div className="flex items-center rounded-lg bg-gray-100 dark:bg-gray-800 p-1 text-sm">
          <button
            className={`px-3 py-1.5 rounded-md transition-colors ${mode === 'live' ? 'bg-white dark:bg-gray-700 shadow-sm' : 'text-gray-500'}`}
            onClick={() => onModeChange('live')}
          >
            Live browser
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
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
          />
          <button
            className="px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
            onClick={() => {
              if (mode === 'snapshot') {
                void snapshot.refresh();
              } else {
                setLiveRefreshToken((t) => t + 1);
              }
            }}
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Body */}
      {sessionId && (
        <div className="px-4 py-2 text-[11px] text-gray-500 dark:text-gray-400 border-b border-gray-100 dark:border-gray-800">
          Recording session is synced to this URL. Interactions are executed in the Playwright browser session.
        </div>
      )}
      <div className="flex-1 overflow-hidden" ref={previewContainerRef}>
        {mode === 'live' ? (
          sessionId ? (
            effectiveUrl ? (
              <PlaywrightView
                sessionId={sessionId}
                refreshToken={liveRefreshToken}
                onStreamError={() => onLiveBlocked()}
              />
            ) : (
              <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the actual Playwright session." />
            )
          ) : (
            <EmptyState title="Start a recording session" subtitle="Create or resume a recording session to view the live browser." />
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
      <div className="h-full flex flex-col items-center justify-center text-center px-6 gap-3 bg-gradient-to-b from-gray-50 to-white dark:from-gray-900 dark:to-gray-950">
        <div className="relative">
          <div className="w-14 h-14 rounded-full border-4 border-blue-100 dark:border-blue-900/30" />
          <div
            className="absolute inset-1 rounded-full border-4 border-blue-500/60 border-t-transparent animate-spin"
            style={{ animationDuration: '1.1s' }}
          />
          <div
            className="absolute inset-2 rounded-full border-4 border-blue-200/80 dark:border-blue-800/60 border-t-transparent animate-spin"
            style={{ animationDuration: '1.6s' }}
          />
        </div>
        <div className="space-y-1">
          <p className="text-sm font-medium text-gray-800 dark:text-gray-100">Capturing snapshot…</p>
          <p className="text-xs text-gray-500 dark:text-gray-400">May take a few seconds while the page loads and renders.</p>
        </div>
        <div className="flex items-center gap-2 text-[11px] text-gray-500 dark:text-gray-400">
          <span className="w-2 h-2 bg-blue-400 rounded-full animate-pulse" />
          <span>Waiting for a fresh preview</span>
        </div>
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
