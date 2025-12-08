import { useEffect, useMemo, useRef, useState } from 'react';
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

  const [liveState, setLiveState] = useState<'idle' | 'loading' | 'loaded' | 'blocked'>('idle');
  const loadTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const [urlInput, setUrlInput] = useState(previewUrl);
  const blockedRef = useRef(false);

  useEffect(() => {
    blockedRef.current = false;
    // Start load timer when switching to live with a URL
    if (mode === 'live' && effectiveUrl) {
      setLiveState('loading');
      if (loadTimerRef.current) clearTimeout(loadTimerRef.current);
      loadTimerRef.current = setTimeout(() => {
        markBlocked();
      }, 4500);
    } else {
      setLiveState('idle');
      if (loadTimerRef.current) {
        clearTimeout(loadTimerRef.current);
        loadTimerRef.current = null;
      }
    }
    return () => {
      if (loadTimerRef.current) {
        clearTimeout(loadTimerRef.current);
        loadTimerRef.current = null;
      }
    };
  }, [mode, effectiveUrl, onLiveBlocked]);

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

  const markBlocked = () => {
    blockedRef.current = true;
    setLiveState('blocked');
    onLiveBlocked();
  };

  const inspectIframe = (): 'blocked' | 'ok' | 'unknown' => {
    const iframe = iframeRef.current;
    if (!iframe) return 'unknown';
    try {
      const doc = iframe.contentDocument;
      if (!doc || !doc.body) return 'unknown';
      const text = (doc.body.innerText || '').toLowerCase();
      const len = text.trim().length;
      const childCount = doc.body.childElementCount || 0;
      const refused = text.includes('refused to connect') || text.includes('denied') || text.includes('blocked');
      const empty = len === 0 && childCount === 0;
      if (refused || empty) return 'blocked';
      return 'ok';
    } catch {
      // Cross-origin; assume ok since we cannot inspect
      return 'ok';
    }
  };

  const handleIframeLoad = () => {
    if (loadTimerRef.current) {
      clearTimeout(loadTimerRef.current);
      loadTimerRef.current = null;
    }
    blockedRef.current = false;
    setLiveState('loading');

    const runChecks = () => {
      const status = inspectIframe();
      if (status === 'blocked') {
        markBlocked();
        return true;
      }
      if (status === 'ok') {
        setLiveState('loaded');
        return true;
      }
      return false;
    };

    if (runChecks()) return;
    setTimeout(() => {
      if (blockedRef.current) return;
      if (runChecks()) return;
      setTimeout(() => {
        if (blockedRef.current) return;
        if (!runChecks()) {
          // If still unknown after retries, assume blocked to avoid hanging on blank pages.
          markBlocked();
        }
      }, 600);
    }, 400);
  };

  const handleIframeError = () => {
    if (loadTimerRef.current) {
      clearTimeout(loadTimerRef.current);
      loadTimerRef.current = null;
    }
    markBlocked();
  };

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
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
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
            <div className="relative h-full">
              <iframe
                key={iframeKey}
                src={effectiveUrl}
                className="w-full h-full bg-white"
                sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-pointer-lock allow-modals"
                ref={iframeRef}
                onError={handleIframeError}
                onLoad={handleIframeLoad}
                title="Live preview"
              />
              {liveState === 'loading' && (
                <div className="absolute inset-0 flex items-center justify-center bg-white/70 dark:bg-gray-900/70 backdrop-blur-sm text-xs text-gray-600 dark:text-gray-300">
                  Attempting live preview… switching to snapshot if it doesn’t load.
                </div>
              )}
              {liveState === 'blocked' && (
                <div className="absolute inset-0 flex flex-col items-center justify-center bg-white/80 dark:bg-gray-900/80 backdrop-blur-sm text-center px-4 gap-2 text-sm">
                  <p className="font-medium text-gray-800 dark:text-gray-100">Live preview blocked or slow</p>
                  <p className="text-xs text-gray-500 dark:text-gray-400">We’ll use Snapshot mode instead.</p>
                </div>
              )}
            </div>
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
