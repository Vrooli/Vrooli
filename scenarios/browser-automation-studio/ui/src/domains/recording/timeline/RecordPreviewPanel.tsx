/**
 * RecordPreviewPanel - Content-only live preview for recording mode
 *
 * This component renders ONLY the content (PlaywrightView or StartRecordingState).
 * It is completely agnostic to presentation styling - that is handled by PreviewContainer.
 *
 * The component:
 * - Renders PlaywrightView when a session and URL are active
 * - Renders StartRecordingState when no session/URL is active
 * - Receives viewport from parent (PreviewContainer manages viewport state)
 * - Exposes metadata (pageTitle, frameStats) via callbacks for the parent to use
 * - Fills available space (h-full w-full)
 *
 * ARCHITECTURAL NOTE:
 * Viewport management has been moved to PreviewContainer. This component now
 * receives the viewport as a prop instead of managing its own ViewportSyncManager.
 * This ensures the browser viewport stays stable during replay style toggles.
 */

import { useCallback, useMemo, useState, useEffect } from 'react';
import { Globe, Loader2 } from 'lucide-react';
import { loadHistory, type HistoryEntry } from '../capture/BrowserUrlBar';
import { useLinkPreviewsBatch, type LinkPreviewData } from '../hooks/useLinkPreview';
import type { RecordedAction } from '../types/types';
import { PlaywrightView, type FrameStats, type PageMetadata, type StreamConnectionStatus } from '../capture/PlaywrightView';
import { useStreamSettings } from '../capture/StreamSettings';
import { useViewportOptional } from '../context';

interface RecordPreviewPanelProps {
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  sessionId?: string | null;
  /** Active page ID for multi-tab sessions */
  activePageId?: string | null;
  /**
   * Browser viewport dimensions (managed by PreviewContainer).
   * This is the viewport that Playwright uses - stable during replay style toggles.
   */
  viewport?: { width: number; height: number } | null;
  /** Callback when stream connection status changes */
  onConnectionStatusChange?: (status: StreamConnectionStatus) => void;
  /** Whether to hide the in-preview connection indicator (shown in header instead) */
  hideConnectionIndicator?: boolean;
  /** Callback when page title changes (for BrowserChrome display) */
  onPageTitleChange?: (title: string) => void;
  /** Callback when frame stats update (for BrowserChrome display) */
  onFrameStatsChange?: (stats: FrameStats | null) => void;
  /** Refresh token - increment to trigger a page refresh */
  refreshToken?: number;
  /** Whether the viewport is being resized (show resize indicator) */
  isResizing?: boolean;
  /** Whether viewport sync is in progress */
  isViewportSyncing?: boolean;
}

export function RecordPreviewPanel({
  previewUrl,
  onPreviewUrlChange,
  actions,
  sessionId,
  activePageId,
  viewport,
  onConnectionStatusChange,
  hideConnectionIndicator = false,
  onPageTitleChange,
  onFrameStatsChange,
  refreshToken = 0,
  isResizing = false,
  isViewportSyncing = false,
}: RecordPreviewPanelProps) {
  const lastUrl = useMemo(() => {
    if (actions.length === 0) return '';
    return actions[actions.length - 1]?.url ?? '';
  }, [actions]);

  // Check if URL is navigable (not empty, about:blank, or chrome-error://)
  const isNavigableUrl = (url: string) => {
    if (!url) return false;
    const lowerUrl = url.toLowerCase();
    return !lowerUrl.startsWith('about:') && !lowerUrl.startsWith('chrome-error://');
  };

  const rawEffectiveUrl = previewUrl || lastUrl;
  const effectiveUrl = isNavigableUrl(rawEffectiveUrl) ? rawEffectiveUrl : '';

  // Stream settings (for quality/fps)
  const { settings: streamSettings } = useStreamSettings();

  // Get viewport context if available (for recording mode with ViewportProvider)
  const viewportContext = useViewportOptional();

  // Use context values when available, fall back to props
  const effectiveViewport = viewportContext?.browserViewport ?? viewport;
  const effectiveIsResizing = viewportContext?.syncState.isResizing ?? isResizing;
  const effectiveIsSyncing = viewportContext?.syncState.isSyncing ?? isViewportSyncing;

  // Frame statistics from PlaywrightView
  const handleStatsUpdate = useCallback((stats: FrameStats) => {
    onFrameStatsChange?.(stats);
  }, [onFrameStatsChange]);

  // Page metadata (title, url) from PlaywrightView
  const handlePageMetadataChange = useCallback((metadata: PageMetadata) => {
    onPageTitleChange?.(metadata.title);
  }, [onPageTitleChange]);

  // Convert viewport to the format PlaywrightView expects
  const viewportForPlaywright = effectiveViewport ?? undefined;

  return (
    <div className="h-full w-full">
      {sessionId && effectiveUrl ? (
        <PlaywrightView
          sessionId={sessionId}
          pageId={activePageId ?? undefined}
          refreshToken={refreshToken}
          viewport={viewportForPlaywright}
          quality={streamSettings.quality}
          fps={streamSettings.fps}
          onStatsUpdate={handleStatsUpdate}
          onPageMetadataChange={handlePageMetadataChange}
          onConnectionStatusChange={onConnectionStatusChange}
          hideConnectionIndicator={hideConnectionIndicator}
          isResizing={effectiveIsResizing}
          isViewportSyncing={effectiveIsSyncing}
        />
      ) : (
        <StartRecordingState onNavigate={onPreviewUrlChange} />
      )}
    </div>
  );
}

/** Extract domain from URL for display */
function extractDomain(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.hostname.replace(/^www\./, '');
  } catch {
    return url;
  }
}

/** Get unique recent domains from history, grouped by domain */
function getRecentDomains(history: HistoryEntry[], maxDomains = 6): Array<{ domain: string; url: string; title?: string; favicon?: string }> {
  const domainMap = new Map<string, { url: string; title?: string; lastVisited: number }>();

  for (const entry of history) {
    const domain = extractDomain(entry.url);
    const existing = domainMap.get(domain);

    // Keep the most recent entry for each domain
    if (!existing || entry.lastVisited > existing.lastVisited) {
      domainMap.set(domain, {
        url: entry.url,
        title: entry.title,
        lastVisited: entry.lastVisited,
      });
    }
  }

  // Sort by recency and take top N
  return Array.from(domainMap.entries())
    .sort((a, b) => b[1].lastVisited - a[1].lastVisited)
    .slice(0, maxDomains)
    .map(([domain, data]) => ({
      domain,
      url: data.url,
      title: data.title,
    }));
}

/** Get first letter of domain for favicon placeholder */
function getDomainInitial(domain: string): string {
  return domain.charAt(0).toUpperCase();
}

/** Generate a consistent color gradient based on domain name */
function getDomainGradient(domain: string): string {
  const gradients = [
    'from-blue-500 to-indigo-600',
    'from-purple-500 to-pink-600',
    'from-green-500 to-teal-600',
    'from-orange-500 to-red-600',
    'from-cyan-500 to-blue-600',
    'from-rose-500 to-purple-600',
  ];
  const hash = domain.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
  return gradients[hash % gradients.length];
}

/** Individual site card with preview data */
function SitePreviewCard({
  domain,
  url,
  historyTitle,
  preview,
  isLoading,
  onClick,
}: {
  domain: string;
  url: string;
  historyTitle?: string;
  preview: LinkPreviewData | null | undefined;
  isLoading: boolean;
  onClick: () => void;
}) {
  const hasImage = preview?.image;
  const title = preview?.title || preview?.site_name || historyTitle || domain;
  const favicon = preview?.favicon;
  const gradient = getDomainGradient(domain);

  return (
    <button
      onClick={onClick}
      className="group relative flex flex-col overflow-hidden rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600 hover:shadow-lg transition-all duration-200"
      title={preview?.description || url}
    >
      {/* Image or gradient placeholder */}
      <div className="relative h-20 w-full overflow-hidden bg-gray-100 dark:bg-gray-700">
        {isLoading ? (
          <div className="absolute inset-0 flex items-center justify-center">
            <Loader2 className="w-5 h-5 text-gray-400 animate-spin" />
          </div>
        ) : hasImage ? (
          <img
            src={preview.image}
            alt=""
            className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
            onError={(e) => {
              // Hide image on error and show gradient fallback
              (e.target as HTMLImageElement).style.display = 'none';
              const parent = (e.target as HTMLImageElement).parentElement;
              if (parent) {
                parent.classList.add('bg-gradient-to-br', ...gradient.split(' '));
              }
            }}
          />
        ) : (
          <div className={`absolute inset-0 bg-gradient-to-br ${gradient} flex items-center justify-center`}>
            <span className="text-2xl font-bold text-white/90">{getDomainInitial(domain)}</span>
          </div>
        )}
        {/* Hover overlay */}
        <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors" />
      </div>

      {/* Content */}
      <div className="flex items-center gap-2 p-2.5">
        {/* Favicon */}
        {favicon ? (
          <img
            src={favicon}
            alt=""
            className="w-4 h-4 rounded flex-shrink-0"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
        ) : (
          <div className={`w-4 h-4 rounded bg-gradient-to-br ${gradient} flex items-center justify-center flex-shrink-0`}>
            <span className="text-[8px] font-bold text-white">{getDomainInitial(domain)}</span>
          </div>
        )}
        {/* Title/Domain */}
        <span className="text-xs font-medium text-gray-700 dark:text-gray-300 truncate group-hover:text-gray-900 dark:group-hover:text-gray-100 transition-colors">
          {title}
        </span>
      </div>
    </button>
  );
}

/** Start recording state with recently-navigated domains */
function StartRecordingState({ onNavigate }: { onNavigate?: (url: string) => void }) {
  const [recentDomains, setRecentDomains] = useState<Array<{ domain: string; url: string; title?: string }>>([]);

  useEffect(() => {
    const history = loadHistory();
    setRecentDomains(getRecentDomains(history));
  }, []);

  // Fetch link previews for all recent domains
  const urls = useMemo(() => recentDomains.map((d) => d.url), [recentDomains]);
  const { previews, isLoading } = useLinkPreviewsBatch(urls);

  return (
    <div className="h-full flex flex-col items-center justify-center text-center px-6">
      {/* Clock icon with full circle */}
      <svg className="w-12 h-12 text-gray-300 dark:text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="10" strokeWidth={1.5} />
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6l4 2" />
      </svg>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Start a recording session</p>
      <p className="text-xs text-gray-500 mt-1">Enter a URL above or select a recent site to begin</p>

      {/* Recent domains grid */}
      {recentDomains.length > 0 && (
        <div className="mt-6 w-full max-w-lg">
          <p className="text-xs text-gray-400 dark:text-gray-500 mb-3 uppercase tracking-wide font-medium">Recent Sites</p>
          <div className="grid grid-cols-3 gap-3">
            {recentDomains.map((item) => (
              <SitePreviewCard
                key={item.domain}
                domain={item.domain}
                url={item.url}
                historyTitle={item.title}
                preview={previews.get(item.url)}
                isLoading={isLoading && !previews.has(item.url)}
                onClick={() => onNavigate?.(item.url)}
              />
            ))}
          </div>
        </div>
      )}

      {/* Empty state when no history */}
      {recentDomains.length === 0 && (
        <div className="mt-6 flex items-center gap-2 text-xs text-gray-400 dark:text-gray-500">
          <Globe size={14} />
          <span>Your recently visited sites will appear here</span>
        </div>
      )}
    </div>
  );
}
