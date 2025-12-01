import { memo, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AlertCircle, Loader2 } from 'lucide-react';
import { type GeneratedScenario, type PreviewLinks } from '../lib/api';
import { useFullscreenMode } from '../hooks/useFullscreenMode';
import { useIframeBridge } from '../hooks/useIframeBridge';
import { LandingPreviewToolbar, type PreviewViewType } from './LandingPreviewToolbar';
import { ScenarioInfoPanel } from './ScenarioInfoPanel';
import { type LifecycleControlConfig } from './types';

export interface LandingPreviewViewProps {
  scenario: GeneratedScenario;
  isRunning: boolean;
  onClose: () => void;
  onCustomize?: (scenario: GeneratedScenario) => void;
  onStartScenario: (scenarioId: string) => void;
  previewLinks?: PreviewLinks;
  initialView?: PreviewViewType;
  onStopScenario?: (scenarioId: string) => void;
  onRestartScenario?: (scenarioId: string) => void;
  lifecycleLoading?: boolean;
  enableInfoPanel?: boolean;
  scenarioLogs?: string;
  logsLoading?: boolean;
  onLoadLogs?: (scenarioId: string, options?: { force?: boolean; tail?: number }) => void;
}

/**
 * Build the proxy URL for viewing a generated scenario's landing page.
 * This routes through the landing-manager UI server which proxies to the running scenario.
 */
function buildProxyPreviewUrl(scenarioId: string, viewType: PreviewViewType): string {
  const path = viewType === 'admin' ? '/admin' : '/';
  return `/landing/${encodeURIComponent(scenarioId)}/proxy${path}`;
}

function resolvePreviewUrl(
  scenarioId: string,
  viewType: PreviewViewType,
  isRunning: boolean,
  links?: PreviewLinks,
): string | null {
  if (!isRunning) return null;

  const linkSet = links?.links;
  if (linkSet) {
    if (viewType === 'admin') {
      if (linkSet.admin) return linkSet.admin;
      if (linkSet.admin_login) return linkSet.admin_login;
      if (linkSet.public) return `${linkSet.public.replace(/\/$/, '')}/admin`;
      return buildProxyPreviewUrl(scenarioId, viewType);
    }
    return linkSet.public ?? linkSet.admin ?? buildProxyPreviewUrl(scenarioId, viewType);
  }

  return buildProxyPreviewUrl(scenarioId, viewType);
}

export const LandingPreviewView = memo(function LandingPreviewView({
  scenario,
  isRunning,
  onClose,
  onCustomize,
  onStartScenario,
  previewLinks,
  initialView = 'public',
  onStopScenario,
  onRestartScenario,
  lifecycleLoading = false,
  enableInfoPanel = false,
  scenarioLogs,
  logsLoading,
  onLoadLogs,
}: LandingPreviewViewProps) {
  const [viewType, setViewType] = useState<PreviewViewType>(initialView);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const lastInitialViewRef = useRef<PreviewViewType>(initialView);
  const [isInfoPanelOpen, setIsInfoPanelOpen] = useState(false);

  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);

  // Fullscreen mode
  const {
    isFullscreen,
    isLayoutFullscreen,
    handleToggleFullscreen,
  } = useFullscreenMode({ previewViewRef: previewContainerRef });

  const isAnyFullscreen = isFullscreen || isLayoutFullscreen;

  // Build preview URL
  const previewUrl = useMemo(() => {
    return resolvePreviewUrl(scenario.scenario_id, viewType, isRunning, previewLinks);
  }, [scenario.scenario_id, viewType, isRunning, previewLinks]);

  useEffect(() => {
    if (!enableInfoPanel) {
      setIsInfoPanelOpen(false);
    }
  }, [enableInfoPanel, scenario.scenario_id]);

  const lifecycleControls: LifecycleControlConfig | null = useMemo(() => {
    if (!enableInfoPanel) return null;
    return {
      running: isRunning,
      loading: lifecycleLoading,
      onStart: () => onStartScenario(scenario.scenario_id),
      onStop: onStopScenario ? () => onStopScenario(scenario.scenario_id) : undefined,
      onRestart: onRestartScenario ? () => onRestartScenario(scenario.scenario_id) : undefined,
    };
  }, [enableInfoPanel, isRunning, lifecycleLoading, onStartScenario, onStopScenario, onRestartScenario, scenario.scenario_id]);

  // Iframe bridge for navigation
  const {
    state: bridgeState,
    sendNav,
    resetState: resetBridgeState,
  } = useIframeBridge({
    iframeRef,
    previewUrl,
  });

  // Handle iframe load
  const handleIframeLoad = useCallback(() => {
    setIsLoading(false);
    setLoadError(null);
  }, []);

  useEffect(() => {
    if (initialView !== lastInitialViewRef.current && initialView) {
      lastInitialViewRef.current = initialView;
      setViewType(initialView);
    }
  }, [initialView]);

  // Handle iframe error
  const handleIframeError = useCallback(() => {
    setIsLoading(false);
    setLoadError('Failed to load preview. Make sure the scenario is running.');
  }, []);

  // Refresh preview
  const handleRefresh = useCallback(() => {
    setIsLoading(true);
    setLoadError(null);
    resetBridgeState();
    setReloadToken(prev => prev + 1);
  }, [resetBridgeState]);

  // Handle view type change
  const handleViewTypeChange = useCallback((newViewType: PreviewViewType) => {
    if (newViewType !== viewType) {
      setIsLoading(true);
      setLoadError(null);
      setViewType(newViewType);
    }
  }, [viewType]);

  // Handle open external
  const handleOpenExternal = useCallback(() => {
    if (previewUrl) {
      window.open(previewUrl, '_blank', 'noopener,noreferrer');
    }
  }, [previewUrl]);

  // Handle customize
  const handleCustomize = useCallback(() => {
    onCustomize?.(scenario);
  }, [onCustomize, scenario]);

  // Navigation handlers
  const handleGoBack = useCallback(() => {
    sendNav('BACK');
  }, [sendNav]);

  const handleGoForward = useCallback(() => {
    sendNav('FWD');
  }, [sendNav]);

  // Reset loading state when preview URL changes
  useEffect(() => {
    if (previewUrl) {
      setIsLoading(true);
      setLoadError(null);
    }
  }, [previewUrl]);

  // Handle escape key to close preview (when not in fullscreen)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isAnyFullscreen) {
        onClose();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isAnyFullscreen, onClose]);

  return (
    <div
      ref={previewContainerRef}
      className={`landing-preview-view ${isAnyFullscreen ? 'landing-preview-view--fullscreen' : ''} ${enableInfoPanel ? 'landing-preview-view--immersive' : ''}`}
      data-testid="landing-preview-view"
    >
      <LandingPreviewToolbar
        scenarioName={scenario.name}
        viewType={viewType}
        onViewTypeChange={handleViewTypeChange}
        onRefresh={handleRefresh}
        isRefreshing={isLoading}
        onCustomize={handleCustomize}
        onOpenExternal={handleOpenExternal}
        externalUrl={previewUrl}
        onClose={onClose}
        isFullscreen={isAnyFullscreen}
        onToggleFullscreen={handleToggleFullscreen}
        canGoBack={bridgeState.canGoBack}
        canGoForward={bridgeState.canGoForward}
        onGoBack={handleGoBack}
        onGoForward={handleGoForward}
        showInfoButton={enableInfoPanel}
        infoPanelOpen={isInfoPanelOpen}
        onToggleInfoPanel={enableInfoPanel ? () => setIsInfoPanelOpen((prev) => !prev) : undefined}
        lifecycleControls={lifecycleControls}
      />

      <div className="landing-preview-view__content">
        {!isRunning ? (
          <div className="landing-preview-view__placeholder">
            <div className="landing-preview-view__placeholder-content">
              <AlertCircle className="h-12 w-12 text-amber-400 mb-4" />
              <h3 className="text-lg font-semibold text-slate-100 mb-2">
                Scenario Not Running
              </h3>
              <p className="text-sm text-slate-400 mb-4 max-w-md text-center">
                Start the scenario to preview your landing page. The preview will load automatically once the scenario is running.
              </p>
              <button
                type="button"
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-emerald-500/40 bg-emerald-500/10 text-emerald-300 hover:bg-emerald-500/20 hover:border-emerald-400 transition-all"
                onClick={() => onStartScenario(scenario.scenario_id)}
              >
                Start Scenario
              </button>
            </div>
          </div>
        ) : loadError ? (
          <div className="landing-preview-view__placeholder">
            <div className="landing-preview-view__placeholder-content">
              <AlertCircle className="h-12 w-12 text-red-400 mb-4" />
              <h3 className="text-lg font-semibold text-slate-100 mb-2">
                Preview Error
              </h3>
              <p className="text-sm text-slate-400 mb-4 max-w-md text-center">
                {loadError}
              </p>
              <button
                type="button"
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-blue-500/40 bg-blue-500/10 text-blue-300 hover:bg-blue-500/20 hover:border-blue-400 transition-all"
                onClick={handleRefresh}
              >
                Try Again
              </button>
            </div>
          </div>
        ) : (
          <>
            {isLoading && (
              <div className="landing-preview-view__loading">
                <Loader2 className="h-8 w-8 animate-spin text-emerald-400" />
                <span className="text-sm text-slate-400 mt-2">Loading preview...</span>
              </div>
            )}
            {previewUrl && (
              <iframe
                key={`${scenario.scenario_id}-${viewType}-${reloadToken}`}
                ref={iframeRef}
                src={previewUrl}
                title={`${scenario.name} preview - ${viewType}`}
                className={`landing-preview-view__iframe ${isLoading ? 'landing-preview-view__iframe--loading' : ''}`}
                onLoad={handleIframeLoad}
                onError={handleIframeError}
                loading="lazy"
                sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-popups-to-escape-sandbox"
              />
            )}
          </>
        )}
      </div>
      {enableInfoPanel && (
        <ScenarioInfoPanel
          open={isInfoPanelOpen}
          onClose={() => setIsInfoPanelOpen(false)}
          scenario={scenario}
          previewLinks={previewLinks}
          lifecycleControls={lifecycleControls}
          logs={scenarioLogs}
          logsLoading={logsLoading}
          onLoadLogs={onLoadLogs}
        />
      )}
    </div>
  );
});

export default LandingPreviewView;
