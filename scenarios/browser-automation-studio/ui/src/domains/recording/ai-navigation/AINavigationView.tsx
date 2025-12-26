/**
 * AINavigationView Component
 *
 * Combines the browser preview with the AI navigation panel.
 * Shows the AI controls on the right and the browser on the left.
 */

import { useCallback, useState } from 'react';
import { AINavigationErrorBoundary } from './AINavigationErrorBoundary';
import { AINavigationPanel } from './AINavigationPanel';
import { AINavigationTimeline } from './AINavigationStepCard';
import { useAINavigation } from './useAINavigation';
import type { RecordedAction } from '../types/types';
import { PlaywrightView, type FrameStats, type PageMetadata } from '../capture/PlaywrightView';
import { BrowserUrlBar } from '../capture/BrowserUrlBar';
import type { StreamSettingsValues } from '../capture/StreamSettings';

interface AINavigationViewProps {
  sessionId: string | null;
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  streamSettings?: StreamSettingsValues;
  onStatsUpdate?: (stats: FrameStats) => void;
  onPageMetadataChange?: (metadata: PageMetadata) => void;
  pageTitle?: string;
}

export function AINavigationView({
  sessionId,
  previewUrl,
  onPreviewUrlChange,
  actions,
  streamSettings,
  onStatsUpdate,
  onPageMetadataChange,
  pageTitle: externalPageTitle,
}: AINavigationViewProps) {
  const {
    state,
    startNavigation,
    abortNavigation,
    reset,
    availableModels,
    isNavigating,
  } = useAINavigation({ sessionId });

  const [refreshToken, setRefreshToken] = useState(0);

  const handleRefresh = useCallback(() => {
    setRefreshToken((t) => t + 1);
  }, []);

  const handleStart = useCallback(
    (prompt: string, model: string, maxSteps: number) => {
      void startNavigation(prompt, model, maxSteps);
    },
    [startNavigation]
  );

  const handleAbort = useCallback(() => {
    void abortNavigation();
  }, [abortNavigation]);

  const lastUrl = actions.length > 0 ? actions[actions.length - 1]?.url ?? '' : '';
  const effectiveUrl = previewUrl || lastUrl;

  return (
    <div className="flex h-full">
      {/* Left: Browser Preview */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Browser URL Bar */}
        <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-2.5">
          <BrowserUrlBar
            value={previewUrl}
            onChange={onPreviewUrlChange}
            onNavigate={onPreviewUrlChange}
            onRefresh={handleRefresh}
            placeholder={lastUrl || 'Search or enter URL'}
            pageTitle={externalPageTitle}
            disabled={isNavigating}
          />
        </div>

        {/* Browser View */}
        <div className="flex-1 overflow-hidden">
          {sessionId ? (
            effectiveUrl ? (
              <PlaywrightView
                sessionId={sessionId}
                refreshToken={refreshToken}
                quality={streamSettings?.quality}
                fps={streamSettings?.fps}
                onStatsUpdate={onStatsUpdate}
                onPageMetadataChange={onPageMetadataChange}
              />
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-center px-6">
                <svg className="w-10 h-10 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                </svg>
                <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Add a URL to start</p>
                <p className="text-xs text-gray-500 mt-1">Enter a URL above, then tell the AI what to do</p>
              </div>
            )
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-center px-6">
              <svg className="w-10 h-10 text-gray-300 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Start a session first</p>
              <p className="text-xs text-gray-500 mt-1">Create a browser session to use AI navigation</p>
            </div>
          )}
        </div>
      </div>

      {/* Right: AI Navigation Panel & Steps */}
      <div className="w-96 flex flex-col border-l border-gray-200 dark:border-gray-700">
        <AINavigationErrorBoundary onReset={reset}>
          {/* AI Navigation Controls */}
          <div className="flex-shrink-0" style={{ maxHeight: '60%' }}>
            <AINavigationPanel
              models={availableModels}
              isNavigating={isNavigating}
              steps={state.steps}
              totalTokens={state.totalTokens}
              status={state.status}
              error={state.error}
              onStart={handleStart}
              onAbort={handleAbort}
              onReset={reset}
              disabled={!sessionId}
            />
          </div>

          {/* AI Navigation Steps Timeline */}
          <div className="flex-1 min-h-0 overflow-hidden flex flex-col border-t border-gray-200 dark:border-gray-700">
            <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  AI Steps
                </span>
                {state.steps.length > 0 && (
                  <span className="px-1.5 py-0.5 text-[10px] font-medium bg-purple-100 dark:bg-purple-900/50 text-purple-700 dark:text-purple-300 rounded">
                    {state.steps.length}
                  </span>
                )}
              </div>
            </div>
            <div className="flex-1 overflow-y-auto">
              <AINavigationTimeline steps={state.steps} isNavigating={isNavigating} />
            </div>
          </div>
        </AINavigationErrorBoundary>
      </div>
    </div>
  );
}
