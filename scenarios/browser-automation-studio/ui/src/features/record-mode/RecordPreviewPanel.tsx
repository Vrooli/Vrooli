import { useEffect, useMemo, useRef, useState } from 'react';
import type { RecordedAction } from './types';
import { PlaywrightView } from './components/PlaywrightView';

interface RecordPreviewPanelProps {
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  onViewportChange?: (size: { width: number; height: number }) => void;
}

export function RecordPreviewPanel({
  previewUrl,
  onPreviewUrlChange,
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
      {/* Header with URL input */}
      <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-3 flex items-center gap-3">
        <div className="flex-1 flex items-center gap-2">
          <input
            className="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800"
            placeholder={lastUrl || 'Enter a URL to preview'}
            value={urlInput}
            onChange={(e) => setUrlInput(e.target.value)}
          />
          <button
            className="px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700"
            onClick={() => setLiveRefreshToken((t) => t + 1)}
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
        {sessionId ? (
          effectiveUrl ? (
            <PlaywrightView
              sessionId={sessionId}
              refreshToken={liveRefreshToken}
            />
          ) : (
            <EmptyState title="Add a URL to load the live preview" subtitle="Live preview renders the actual Playwright session." />
          )
        ) : (
          <EmptyState title="Start a recording session" subtitle="Create or resume a recording session to view the live browser." />
        )}
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
