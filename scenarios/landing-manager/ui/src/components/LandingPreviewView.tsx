import { LOOPBACK_HOSTS, getProxyInfo, isProxyContext } from '@vrooli/api-base';
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
  onPromoteScenario?: (scenario: GeneratedScenario) => void;
}

const UI_LABEL_PATTERN = /(ui|front|preview|web|client|app)/i;

interface PreviewRoutingEnvironment {
  viewerIsLocal: boolean;
  forceProxyRouting: boolean;
}

function getViewerEnvironment(): PreviewRoutingEnvironment {
  if (typeof window === 'undefined') {
    return {
      viewerIsLocal: true,
      forceProxyRouting: false,
    };
  }

  const hostname = window.location.hostname?.toLowerCase() ?? '';
  const viewerIsLocal = hostname ? LOOPBACK_HOSTS.has(hostname) : false;
  const forceProxyRouting = isProxyContext() || !viewerIsLocal;

  return {
    viewerIsLocal,
    forceProxyRouting,
  };
}

function isLocalPreviewUrl(candidate: string): boolean {
  if (!candidate) {
    return false;
  }

  try {
    const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost';
    const parsed = new URL(candidate, candidate.startsWith('http') ? undefined : base);
    const hostname = parsed.hostname?.toLowerCase() ?? '';
    if (!hostname) {
      return false;
    }
    return LOOPBACK_HOSTS.has(hostname);
  } catch {
    return false;
  }
}

function selectUiProxyPath(): string | null {
  const info = getProxyInfo();
  if (!info) {
    return null;
  }

  const ports = Array.isArray(info.ports) ? info.ports : [];
  const matchesPattern = (value?: string | null) => {
    if (!value) return false;
    return UI_LABEL_PATTERN.test(value);
  };

  const byLabel = ports.find((port) => matchesPattern(port.normalizedLabel) || matchesPattern(port.label));
  if (byLabel?.path) {
    return byLabel.path;
  }

  const byAlias = ports.find((port) =>
    Array.isArray(port.aliases) && port.aliases.some((alias) => matchesPattern(alias?.toLowerCase())),
  );
  if (byAlias?.path) {
    return byAlias.path;
  }

  const primaryPort = ports.find((port) => port.isPrimary && typeof port.path === 'string' && port.path.trim());
  if (primaryPort?.path) {
    return primaryPort.path;
  }

  if (info.basePath) {
    return info.basePath;
  }

  if (info.primary?.path) {
    return info.primary.path;
  }

  return null;
}

function deriveScenarioBaseFromLocation(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const origin = window.location.origin ?? '';
  const pathname = window.location.pathname ?? '';
  if (!origin) {
    return '';
  }

  const proxyMarker = '/proxy';
  const proxyIndex = pathname.indexOf(proxyMarker);
  if (proxyIndex === -1) {
    return origin.replace(/\/+$/, '');
  }

  const basePath = pathname.slice(0, proxyIndex + proxyMarker.length);
  return `${origin}${basePath}`.replace(/\/+$/, '');
}

function normalizeBasePath(value: string): string {
  return value.replace(/\/+$/, '');
}

function getScenarioBaseUrl(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const origin = window.location.origin ?? '';
  const proxyPath = selectUiProxyPath();
  if (proxyPath) {
    if (proxyPath.startsWith('http://') || proxyPath.startsWith('https://')) {
      return normalizeBasePath(proxyPath);
    }
    if (proxyPath.startsWith('/') && origin) {
      return normalizeBasePath(`${origin}${proxyPath}`);
    }
  }

  const fallback = deriveScenarioBaseFromLocation();
  if (fallback) {
    return normalizeBasePath(fallback);
  }

  return normalizeBasePath(origin);
}

function withScenarioBase(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  const base = getScenarioBaseUrl();
  if (!base) {
    return normalizedPath;
  }
  return `${base}${normalizedPath}`;
}

function selectDirectPreviewUrl(
  candidates: Array<string | undefined>,
  env: PreviewRoutingEnvironment,
): string | null {
  if (env.forceProxyRouting) {
    return null;
  }

  for (const candidate of candidates) {
    if (!candidate) {
      continue;
    }
    const trimmed = candidate.trim();
    if (!trimmed) {
      continue;
    }
    if (!env.viewerIsLocal && isLocalPreviewUrl(trimmed)) {
      continue;
    }
    return trimmed;
  }

  return null;
}

/**
 * Build the proxy URL for viewing a generated scenario's landing page.
 * This routes through the landing-manager UI server which proxies to the running scenario.
 */
function buildProxyPreviewUrl(scenarioId: string, viewType: PreviewViewType): string {
  const path = viewType === 'admin' ? '/admin' : '/';
  return withScenarioBase(`/landing/${encodeURIComponent(scenarioId)}/proxy${path}`);
}

function resolvePreviewUrl(
  scenarioId: string,
  viewType: PreviewViewType,
  isRunning: boolean,
  links?: PreviewLinks,
  env?: PreviewRoutingEnvironment,
): string | null {
  if (!isRunning) return null;

  const linkSet = links?.links;
  if (linkSet) {
    if (viewType === 'admin') {
      const directAdmin = selectDirectPreviewUrl(
        [
          linkSet.admin,
          linkSet.admin_login,
          linkSet.public ? `${linkSet.public.replace(/\/$/, '')}/admin` : undefined,
        ],
        env ?? getViewerEnvironment(),
      );
      if (directAdmin) {
        return directAdmin;
      }
      return buildProxyPreviewUrl(scenarioId, viewType);
    }

    const directPublic = selectDirectPreviewUrl(
      [linkSet.public, linkSet.admin],
      env ?? getViewerEnvironment(),
    );
    if (directPublic) {
      return directPublic;
    }
    return buildProxyPreviewUrl(scenarioId, viewType);
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
  onPromoteScenario,
}: LandingPreviewViewProps) {
  const [viewType, setViewType] = useState<PreviewViewType>(initialView);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const lastInitialViewRef = useRef<PreviewViewType>(initialView);
  const [isInfoPanelOpen, setIsInfoPanelOpen] = useState(false);

  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const viewerEnvironment = useMemo(() => getViewerEnvironment(), []);
  const isProxyRestrictedPreview = viewerEnvironment.forceProxyRouting && !viewerEnvironment.viewerIsLocal;

  // Fullscreen mode
  const {
    isFullscreen,
    isLayoutFullscreen,
    handleToggleFullscreen,
  } = useFullscreenMode({ previewViewRef: previewContainerRef });

  const isAnyFullscreen = isFullscreen || isLayoutFullscreen;

  // Build preview URL
  const previewUrl = useMemo(() => {
    if (isProxyRestrictedPreview) {
      return null;
    }
    return resolvePreviewUrl(
      scenario.scenario_id,
      viewType,
      isRunning,
      previewLinks,
      viewerEnvironment,
    );
  }, [scenario.scenario_id, viewType, isRunning, previewLinks, viewerEnvironment, isProxyRestrictedPreview]);

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
        ) : isProxyRestrictedPreview ? (
          <div className="landing-preview-view__placeholder">
            <div className="landing-preview-view__placeholder-content">
              <AlertCircle className="h-12 w-12 text-amber-400 mb-4" />
              <h3 className="text-lg font-semibold text-slate-100 mb-2">
                Preview Requires Local Access
              </h3>
              <p className="text-sm text-slate-400 mb-4 max-w-md text-center">
                Generated landing pages can only be embedded when viewing Landing Manager locally. Promote this scenario to convert it into a managed app, or open Landing Manager directly on localhost to continue staging previews.
              </p>
              <div className="flex flex-col gap-3 sm:flex-row">
                {onPromoteScenario && (
                  <button
                    type="button"
                    className="inline-flex justify-center items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-purple-400/40 bg-purple-500/10 text-purple-100 hover:bg-purple-500/20 hover:border-purple-300 transition-all"
                    onClick={() => onPromoteScenario(scenario)}
                  >
                    Promote Scenario
                  </button>
                )}
                <button
                  type="button"
                  className="inline-flex justify-center items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg border border-slate-500/40 bg-slate-500/10 text-slate-200 hover:bg-slate-500/20 hover:border-slate-300 transition-all"
                  onClick={onClose}
                >
                  Close Preview
                </button>
              </div>
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
