/**
 * AINavigationView Component
 *
 * Combines the browser preview with the AI navigation panel.
 * Shows the AI controls on the right and the browser on the left.
 *
 * Note: AI steps are now shown in the main timeline (left sidebar) instead of
 * a separate section here. This reduces duplication and provides a unified view.
 */

import { useCallback, useState, useEffect, useRef } from 'react';
import { AINavigationErrorBoundary } from './AINavigationErrorBoundary';
import { AINavigationPanel } from './AINavigationPanel';
import { HumanInterventionOverlay } from './HumanInterventionOverlay';
import type { RecordedAction } from '../types/types';
import { PlaywrightView, type FrameStats, type PageMetadata } from '../capture/PlaywrightView';
import { BrowserUrlBar } from '../capture/BrowserUrlBar';
import type { StreamSettingsValues } from '../capture/StreamSettings';
import type { AINavigationState, VisionModelSpec } from './types';

interface AINavigationViewProps {
  sessionId: string | null;
  previewUrl: string;
  onPreviewUrlChange: (url: string) => void;
  actions: RecordedAction[];
  streamSettings?: StreamSettingsValues;
  onStatsUpdate?: (stats: FrameStats) => void;
  onPageMetadataChange?: (metadata: PageMetadata) => void;
  pageTitle?: string;
  /** Initial prompt to pre-fill and optionally auto-start (from template) */
  initialPrompt?: string;
  /** Model to use for auto-start (from template) */
  initialModel?: string;
  /** Max steps for auto-start (from template) */
  initialMaxSteps?: number;
  /** Whether to automatically start AI navigation with the initial prompt */
  autoStart?: boolean;
  /** AI navigation state (lifted from parent) */
  aiState?: AINavigationState;
  /** Available vision models */
  aiModels?: VisionModelSpec[];
  /** Whether AI navigation is in progress */
  aiIsNavigating?: boolean;
  /** Start AI navigation */
  onAIStart?: (prompt: string, model: string, maxSteps: number) => Promise<void>;
  /** Abort AI navigation */
  onAIAbort?: () => Promise<void>;
  /** Resume AI navigation after human intervention */
  onAIResume?: () => Promise<void>;
  /** Reset AI state */
  onAIReset?: () => void;
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
  initialPrompt,
  initialModel,
  initialMaxSteps,
  autoStart,
  aiState,
  aiModels,
  aiIsNavigating,
  onAIStart,
  onAIAbort,
  onAIResume,
  onAIReset,
}: AINavigationViewProps) {
  // Use provided state/handlers or defaults
  const state = aiState ?? {
    isNavigating: false,
    navigationId: null,
    prompt: '',
    model: 'qwen3-vl-30b',
    steps: [],
    status: 'idle' as const,
    totalTokens: 0,
    error: null,
    humanIntervention: null,
  };
  const availableModels = aiModels ?? [];
  const isNavigating = aiIsNavigating ?? false;

  const [refreshToken, setRefreshToken] = useState(0);
  const autoStartTriggeredRef = useRef(false);

  // Auto-start AI navigation when session is ready and autoStart is true
  useEffect(() => {
    if (
      autoStart &&
      sessionId &&
      previewUrl &&
      initialPrompt &&
      !autoStartTriggeredRef.current &&
      !isNavigating &&
      state.status === 'idle' &&
      onAIStart
    ) {
      autoStartTriggeredRef.current = true;

      // Use the first available model if initialModel is not specified or not available
      const modelToUse = initialModel && availableModels.some((m) => m.id === initialModel)
        ? initialModel
        : availableModels[0]?.id;

      if (modelToUse) {
        const maxSteps = initialMaxSteps ?? 20;
        void onAIStart(initialPrompt, modelToUse, maxSteps);
      }
    }
  }, [
    autoStart,
    sessionId,
    previewUrl,
    initialPrompt,
    initialModel,
    initialMaxSteps,
    isNavigating,
    state.status,
    availableModels,
    onAIStart,
  ]);

  const handleRefresh = useCallback(() => {
    setRefreshToken((t) => t + 1);
  }, []);

  const handleStart = useCallback(
    (prompt: string, model: string, maxSteps: number) => {
      if (onAIStart) {
        void onAIStart(prompt, model, maxSteps);
      }
    },
    [onAIStart]
  );

  const handleAbort = useCallback(() => {
    if (onAIAbort) {
      void onAIAbort();
    }
  }, [onAIAbort]);

  const handleResume = useCallback(() => {
    if (onAIResume) {
      void onAIResume();
    }
  }, [onAIResume]);

  const handleReset = useCallback(() => {
    if (onAIReset) {
      onAIReset();
    }
  }, [onAIReset]);

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
        <div className="flex-1 overflow-hidden relative">
          {sessionId ? (
            effectiveUrl ? (
              <>
                <PlaywrightView
                  sessionId={sessionId}
                  refreshToken={refreshToken}
                  quality={streamSettings?.quality}
                  fps={streamSettings?.fps}
                  onStatsUpdate={onStatsUpdate}
                  onPageMetadataChange={onPageMetadataChange}
                />
                {/* Human Intervention Overlay */}
                {state.humanIntervention && (
                  <HumanInterventionOverlay
                    intervention={state.humanIntervention}
                    onComplete={handleResume}
                    onAbort={handleAbort}
                  />
                )}
              </>
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

      {/* Right: AI Navigation Panel */}
      <div className="w-96 flex flex-col border-l border-gray-200 dark:border-gray-700">
        <AINavigationErrorBoundary onReset={handleReset}>
          {/* AI Navigation Controls - now takes full height since steps are in main timeline */}
          <div className="flex-1 overflow-hidden">
            <AINavigationPanel
              models={availableModels}
              isNavigating={isNavigating}
              steps={state.steps}
              totalTokens={state.totalTokens}
              status={state.status}
              error={state.error}
              onStart={handleStart}
              onAbort={handleAbort}
              onReset={handleReset}
              disabled={!sessionId}
              initialPrompt={initialPrompt}
            />
          </div>
          {/* Note: AI steps are now shown in the main timeline (left sidebar) with AI badge */}
        </AINavigationErrorBoundary>
      </div>
    </div>
  );
}
