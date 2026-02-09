import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import clsx from 'clsx';
import { GripVertical, Trash2 } from 'lucide-react';
import ErrorBoundary, { SectionErrorFallback } from '@/components/ErrorBoundary';
import AppModal from '@/components/AppModal';
import AppPreviewToolbar from '@/components/AppPreviewToolbar';
import DeviceEmulationToolbar from '@/components/device-emulation/DeviceEmulationToolbar';
import DeviceEmulationViewport from '@/components/device-emulation/DeviceEmulationViewport';
import DeviceVisionFilterDefs from '@/components/device-emulation/DeviceVisionFilterDefs';
import AppLogsPanel from '@/components/logs/AppLogsPanel';
import ReportIssueDialog from '@/components/report/ReportIssueDialog';
import PreviewInspectorPanel from '@/components/views/PreviewInspectorPanel';
import usePreviewInspector from '@/components/views/usePreviewInspector';
import { useAppLogs } from '@/hooks/useAppLogs';
import { useDeviceEmulation } from '@/hooks/useDeviceEmulation';
import { usePreviewAppLifecycle } from '@/hooks/usePreviewAppLifecycle';
import { usePreviewBridgeComplianceCheck } from '@/hooks/usePreviewBridgeComplianceCheck';
import { usePreviewIframeReadinessFallback } from '@/hooks/usePreviewIframeReadinessFallback';
import { usePreviewNavigationSession } from '@/hooks/usePreviewNavigationSession';
import type { PreviewNavigationSessionSnapshot } from '@/hooks/usePreviewNavigationSession';
import { usePreviewReportSession } from '@/hooks/usePreviewReportSession';
import { usePreviewToolbarSession } from '@/hooks/usePreviewToolbarSession';
import { usePreviewUrlOrchestration } from '@/hooks/usePreviewUrlOrchestration';
import { usePreviewOverlay } from '@/hooks/usePreviewOverlay';
import { useAppInsights } from '@/hooks/useAppInsights';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useKeyboardScope } from '@/hooks/useKeyboardScopes';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import { useAppsStore } from '@/state/appsStore';
import { usePreviewWorkspaceStore } from '../state/previewWorkspaceStore';
import type { App } from '@/types';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { isRunningStatus, locateAppByIdentifier, resolveAppIdentifier } from '@/utils/appPreview';
import { parseScenarioProxyPreviewTarget } from '@/utils/previewUrl';
import PreviewFallbackState from '@/components/preview/PreviewFallbackState';
import { usePaneMetadata } from './usePaneMetadata';
import './PreviewPane.css';

export interface PreviewPaneProps {
  paneId: string;
  appId: string | null;
  apps: App[];
  isFocused: boolean;
  isArrangeMode: boolean;
  isBeingDragged: boolean;
  canRemove: boolean;
  onFocus: (paneId: string) => void;
  onRemove: (paneId: string) => void;
  onArrangeDragStart: (paneId: string, event: ReactPointerEvent<HTMLButtonElement>) => void;
}

export function PreviewPane({
  paneId,
  appId,
  apps,
  isFocused,
  isArrangeMode,
  isBeingDragged,
  canRemove,
  onFocus,
  onRemove,
  onArrangeDragStart,
}: PreviewPaneProps) {
  const { openOverlay } = useOverlayRouter();
  const setAppsState = useAppsStore((state) => state.setAppsState);
  const paneViewState = usePreviewWorkspaceStore((state) => state.paneViewState[paneId]);
  const workspaceZoom = usePreviewWorkspaceStore((state) => state.workspaceZoom);
  const setPaneViewState = usePreviewWorkspaceStore((state) => state.setPaneViewState);
  const [statusMessage, setStatusMessage] = useState<string | null>('Select an app to preview.');
  const [isIframeLoading, setIsIframeLoading] = useState(false);
  const [isLogsVisible, setIsLogsVisible] = useState(() => paneViewState?.isLogsVisible ?? false);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);
  const [isFullView, setIsFullView] = useState(false);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [iframeLoadedAt, setIframeLoadedAt] = useState<number | null>(null);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const [paneSurfaceNode, setPaneSurfaceNode] = useState<HTMLDivElement | null>(null);
  const [previewContainerNode, setPreviewContainerNode] = useState<HTMLDivElement | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const paneRef = useRef<HTMLDivElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const lastAssignedAppIdentifierRef = useRef<string | null>(null);

  const activeAppIdentifier = useMemo(() => {
    if (!appId) {
      return null;
    }
    const trimmed = appId.trim();
    return trimmed.length > 0 ? trimmed : null;
  }, [appId]);

  const shouldPreferExistingApp = useCallback((existing: App | null, incoming: App): boolean => {
    if (!existing) {
      return false;
    }
    const existingIdentifier = resolveAppIdentifier(existing);
    const incomingIdentifier = resolveAppIdentifier(incoming);
    if (!existingIdentifier || !incomingIdentifier || existingIdentifier !== incomingIdentifier) {
      return false;
    }

    const existingIsRich = !existing.is_partial && existing.status && existing.status !== 'unknown';
    const incomingIsPoor = Boolean(incoming.is_partial || !incoming.status || incoming.status === 'unknown');
    return existingIsRich && incomingIsPoor;
  }, []);

  const navigationSession = usePreviewNavigationSession({
    iframeRef,
    setStatusMessage,
    initialState: paneViewState,
    onStateChange: useCallback((nextState: PreviewNavigationSessionSnapshot) => {
      setPaneViewState(paneId, nextState);
    }, [paneId, setPaneViewState]),
  });

  const {
    bridge,
    previewUrl,
    previewUrlInput,
    hasCustomPreviewUrl,
    canGoBack,
    canGoForward,
    handleUrlInputChange,
    handleUrlInputKeyDown,
    handleUrlInputBlur,
    handleGoBack,
    handleGoForward,
    applyDefaultPreviewUrl,
    applyPreviewUrlValue,
    resetPreviewState,
    setPreviewUrl,
    initialPreviewUrlRef,
    history,
    historyIndex,
    clearNavigationSession,
  } = navigationSession;
  const scenarioTargetFromUrl = useMemo(
    () => parseScenarioProxyPreviewTarget(previewUrlInput),
    [previewUrlInput],
  );
  const resolvedAppIdentifier = useMemo(() => {
    const scenarioIdentifierFromUrl = scenarioTargetFromUrl?.scenarioIdentifier ?? null;
    if (!scenarioIdentifierFromUrl) {
      return activeAppIdentifier ?? null;
    }
    if (!activeAppIdentifier) {
      return scenarioIdentifierFromUrl;
    }
    if (!hasCustomPreviewUrl) {
      return activeAppIdentifier;
    }

    return scenarioIdentifierFromUrl.toLowerCase() === activeAppIdentifier.toLowerCase()
      ? activeAppIdentifier
      : scenarioIdentifierFromUrl;
  }, [activeAppIdentifier, hasCustomPreviewUrl, scenarioTargetFromUrl?.scenarioIdentifier]);
  const {
    state: bridgeState,
    runComplianceCheck,
    resetState: resetBridgeState,
    requestScreenshot,
    logState,
    configureLogs,
    getRecentLogs,
    requestLogBatch,
    networkState,
    configureNetwork,
    getRecentNetworkEvents,
    requestNetworkBatch,
    inspectState,
    startInspect,
    stopInspect,
    setInspectTargetIndex,
    shiftInspectTarget,
  } = bridge;
  const syncPreviewUrl = usePreviewUrlOrchestration({
    hasCustomPreviewUrl,
    previewUrl,
    applyDefaultPreviewUrl,
    resetPreviewState,
    setPreviewUrl,
    initialPreviewUrlRef,
  });
  const activePreviewUrl = useMemo(
    () => bridgeState.href || previewUrl || '',
    [bridgeState.href, previewUrl],
  );
  const { setPreviewOverlay, fallbackState } = usePreviewOverlay({
    previewUrl,
    previewReloadToken,
    loading: isIframeLoading,
    statusMessage,
    defaultEmptyMessage: 'Select an app to start this pane.',
    bridgeIsReady: bridgeState.isReady,
    iframeLoadedAt,
    iframeLoadError,
  });
  const {
    reportElementCaptures,
    setHasPrimaryCaptureDraft,
    stagedCaptureCount,
    canCaptureScreenshot,
    bridgeSupportsScreenshot,
    isPreviewSameOrigin,
    handleInspectorCaptureAdded,
    handleElementCaptureNoteChange,
    handleRemoveElementCapture,
    resetElementCaptures: handleResetElementCaptures,
    resetReportDraftState,
  } = usePreviewReportSession({
    activePreviewUrl,
    bridgeState: {
      isSupported: bridgeState.isSupported,
      caps: bridgeState.caps,
    },
    logPrefix: '[preview-pane]',
  });

  const deviceEmulation = useDeviceEmulation({
    container: previewContainerNode,
    storageNamespace: paneId,
  });
  const {
    isActive: isDeviceEmulationActive,
    toggleActive: toggleDeviceEmulation,
    toolbar: deviceToolbar,
    viewport: deviceViewport,
  } = deviceEmulation;
  const standardPreviewZoom = isDeviceEmulationActive ? 1 : workspaceZoom;
  const standardPreviewStyle = useMemo<CSSProperties | undefined>(() => {
    if (standardPreviewZoom === 1) {
      return undefined;
    }
    const percent = 100 / standardPreviewZoom;
    return {
      width: `${percent}%`,
      height: `${percent}%`,
      transform: `scale(${standardPreviewZoom})`,
      transformOrigin: 'top left',
    };
  }, [standardPreviewZoom]);

  const handleOpenReportDialog = useCallback(() => {
    setReportDialogOpen(true);
  }, []);

  useEffect(() => {
    if (!isFullView || typeof document === 'undefined') {
      return;
    }
    document.body.classList.add('preview-pane-fullscreen-active');
    return () => {
      document.body.classList.remove('preview-pane-fullscreen-active');
    };
  }, [isFullView]);

  useKeyboardScope({
    id: `preview-pane-full-view-escape-${paneId}`,
    priority: 850,
    enabled: isFullView,
    onKeyDown: (event) => {
      if (event.key !== 'Escape' || typeof document === 'undefined') {
        return false;
      }
      // If browser fullscreen is active, let the higher-priority shell handler
      // handle exiting that first.
      if (document.fullscreenElement) {
        return false;
      }
      event.preventDefault();
      setIsFullView(false);
      return true;
    },
  });

  usePreviewBridgeComplianceCheck({
    enabled: reportDialogOpen,
    runComplianceCheck,
    onSuccess: setBridgeCompliance,
    onError: (error) => {
      logger.debug('[preview-pane] Bridge compliance check unavailable', error);
      setBridgeCompliance(null);
    },
  });
  const {
    currentApp,
    setCurrentApp,
    isMetadataLoading: metadataLoadingFromHydration,
  } = usePaneMetadata({
    paneId,
    apps,
    resolvedAppIdentifier,
    scenarioIdentifierFromUrl: scenarioTargetFromUrl?.scenarioIdentifier ?? null,
    shouldPreferExistingApp,
    setAppsState,
    getApp: appService.getApp,
    setStatusMessage,
    onResetForMissingIdentifier: () => {
      setIsIframeLoading(false);
      setStatusMessage('Select an app to preview.');
      setIsLogsVisible(false);
      setIframeLoadError(null);
      setReportDialogOpen(false);
      resetReportDraftState();
    },
  });

  useEffect(() => {
    setPaneViewState(paneId, { isLogsVisible });
  }, [isLogsVisible, paneId, setPaneViewState]);

  useEffect(() => {
    if (!paneViewState) {
      return;
    }
    setIsLogsVisible((current) => (
      current === paneViewState.isLogsVisible ? current : paneViewState.isLogsVisible
    ));
  }, [paneViewState]);

  useEffect(() => {
    const normalize = (value: string | null): string | null => {
      if (!value) {
        return null;
      }
      const trimmed = value.trim().toLowerCase();
      return trimmed.length > 0 ? trimmed : null;
    };

    const previous = normalize(lastAssignedAppIdentifierRef.current);
    const next = normalize(activeAppIdentifier);
    lastAssignedAppIdentifierRef.current = activeAppIdentifier;

    if (previous === null || previous === next) {
      return;
    }

    // Changing the pane's assigned app should reset stale per-pane URL state so
    // a prior custom/invalid proxy target cannot keep the pane stuck.
    clearNavigationSession();
    resetBridgeState();
    setIsIframeLoading(false);
    setIframeLoadedAt(null);
    setIframeLoadError(null);
    setStatusMessage('Loading app metadata...');
    setPreviewOverlay(null);
    setReportDialogOpen(false);
    resetReportDraftState();
  }, [
    activeAppIdentifier,
    clearNavigationSession,
    resetBridgeState,
    resetReportDraftState,
    setPreviewOverlay,
  ]);

  useEffect(() => {
    if (!currentApp) {
      return;
    }

    const { hasPreviewCandidate } = syncPreviewUrl({
      appForPreview: currentApp,
    });
    if (!hasPreviewCandidate) {
      setIsIframeLoading(false);
      setStatusMessage('Preview unavailable for this app (no UI port found).');
      return;
    }

    if (!isRunningStatus(currentApp.status)) {
      setStatusMessage('App is not running. Start it to load preview.');
    } else {
      setStatusMessage(null);
    }
  }, [currentApp, syncPreviewUrl]);

  useEffect(() => {
    setPreviewOverlay(null);
    setIframeLoadError(null);
    setIframeLoadedAt(null);
    setIsIframeLoading(Boolean(previewUrl));
  }, [previewReloadToken, previewUrl, setPreviewOverlay]);

  const logsState = useAppLogs({
    app: currentApp,
    appId: currentApp?.id ?? resolvedAppIdentifier,
    active: isLogsVisible,
  });

  const onRefresh = useCallback(() => {
    if (!resolvedAppIdentifier) {
      return;
    }

    const historyTarget = historyIndex >= 0 ? (history[historyIndex] ?? null) : null;
    const refreshTarget = bridgeState.href || historyTarget || previewUrl;

    if (refreshTarget && refreshTarget !== previewUrl) {
      setPreviewUrl(refreshTarget);
    }

    setIsIframeLoading(Boolean(refreshTarget || previewUrl));
    setStatusMessage('Refreshing preview...');
    setPreviewOverlay(null);
    setIframeLoadError(null);
    setIframeLoadedAt(null);
    resetBridgeState();
    setPreviewReloadToken((value) => value + 1);

    appService.getApp(resolvedAppIdentifier)
      .then((fetched) => {
        if (fetched) {
          setCurrentApp(fetched);
        }
      })
      .catch((error) => {
        logger.warn('[preview-pane] Failed to refresh app', error);
      });
  }, [bridgeState.href, history, historyIndex, previewUrl, resolvedAppIdentifier, resetBridgeState, setCurrentApp, setPreviewOverlay, setPreviewUrl]);

  const {
    openPreviewTarget,
    buildUrlSuggestionSections,
    handleOpenScenarioSelector,
    handleOpenPreviewInNewTab: onOpenInNewTab,
  } = usePreviewToolbarSession({
    bridgeHref: bridgeState.href,
    previewUrl,
    history,
    apps,
    openOverlay,
    appOpenMode: 'replace-focused',
    onBeforeOpenScenarioSelector: () => onFocus(paneId),
  });

  const inspector = usePreviewInspector({
    inspectState,
    startInspect,
    stopInspect,
    setInspectTargetIndex,
    shiftInspectTarget,
    requestScreenshot,
    previewUrl,
    currentAppIdentifier: resolvedAppIdentifier,
    iframeRef,
    previewViewRef: paneRef,
    previewViewNode: paneSurfaceNode,
    onCaptureAdd: handleInspectorCaptureAdded,
    onViewReportRequest: handleOpenReportDialog,
  });

  const onIframeLoad = useCallback(() => {
    setIsIframeLoading(false);
    setIframeLoadedAt(Date.now());
    setIframeLoadError(null);
    setStatusMessage(null);
    setPreviewOverlay(null);
  }, [setPreviewOverlay]);

  const onIframeError = useCallback(() => {
    setIsIframeLoading(false);
    setIframeLoadedAt(null);
    setIframeLoadError('Preview iframe failed to load.');
  }, []);

  usePreviewIframeReadinessFallback({
    iframeRef,
    enabled: isIframeLoading && Boolean(previewUrl) && !iframeLoadError,
    onReady: () => {
      setIsIframeLoading(false);
      setIframeLoadedAt(Date.now());
      setIframeLoadError(null);
      setStatusMessage(null);
    },
  });

  const lifecycle = usePreviewAppLifecycle({
    currentApp,
    setStatusMessage,
    failureMessageForAction: (action) => `Unable to ${action} this application. Check logs for details.`,
    toggleLabels: {
      start: 'Start app',
      stop: 'Stop app',
    },
    restartLabel: 'Restart app',
    onSuccess: async ({ appId: currentAppId, action }) => {
      const refreshed = await appService.getApp(currentAppId);
      if (refreshed) {
        setCurrentApp(refreshed);
      } else if (action === 'stop') {
        setCurrentApp((previous) => (
          previous ? { ...previous, status: 'stopped', updated_at: new Date().toISOString() } : previous
        ));
      }

      if (action === 'stop') {
        setStatusMessage('Application stopped. Start it again to load preview.');
      } else if (action === 'start') {
        setStatusMessage('Application started. Refreshing preview...');
        setPreviewReloadToken((value) => value + 1);
      } else {
        setStatusMessage('Restart command sent. Refreshing preview...');
        setPreviewReloadToken((value) => value + 1);
      }
    },
  });

  const paneActions = (
    <>
      {isArrangeMode && (
        <button
          type="button"
          className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--secondary')}
          onPointerDown={(event) => onArrangeDragStart(paneId, event)}
          aria-label="Drag pane"
          title="Drag pane"
        >
          <GripVertical size={18} aria-hidden />
        </button>
      )}
      {canRemove && (
        <button
          type="button"
          className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--danger')}
          onClick={() => onRemove(paneId)}
          aria-label="Remove pane"
          title="Remove pane"
        >
          <Trash2 size={18} aria-hidden />
        </button>
      )}
    </>
  );

  const isAppRunning = lifecycle.isAppRunning;
  const toggleActionLabel = lifecycle.toggleActionLabel;
  const restartActionLabel = lifecycle.restartActionLabel;
  const actionInProgress = lifecycle.actionInProgress;
  const {
    diagnostics,
    diagnosticsLoading,
    diagnosticsError,
    lighthouseHistory,
    lighthouseLoading,
    lighthouseError,
    completeness,
    completenessLoading,
    completenessError,
    proxyMetadata,
    refetchDiagnostics,
    refetchLighthouse,
    refetchCompleteness,
  } = useAppInsights(
    currentApp?.id ?? resolvedAppIdentifier ?? null,
    { preload: true },
  );
  const scenarioActionIdentifier = resolvedAppIdentifier;
  const modalFallbackApp = useMemo(() => {
    if (!resolvedAppIdentifier) {
      return null;
    }
    return locateAppByIdentifier(apps, resolvedAppIdentifier);
  }, [resolvedAppIdentifier, apps]);
  const appForModal = currentApp ?? modalFallbackApp;
  const hasScenarioContext = Boolean(currentApp || resolvedAppIdentifier || scenarioActionIdentifier);

  return (
    <div
      className={clsx(
        'preview-pane',
        isFocused && 'preview-pane--focused',
        isFullView && 'preview-pane--full-view',
        isBeingDragged && 'preview-pane--dragging',
      )}
      data-preview-pane-id={paneId}
      onMouseDown={() => onFocus(paneId)}
      aria-label={`Preview pane ${paneId}`}
      ref={(node) => {
        paneRef.current = node;
        setPaneSurfaceNode(node);
      }}
    >
      <DeviceVisionFilterDefs />
      <AppPreviewToolbar
        canGoBack={canGoBack}
        canGoForward={canGoForward}
        onGoBack={handleGoBack}
        onGoForward={handleGoForward}
        onRefresh={onRefresh}
        isRefreshing={metadataLoadingFromHydration || isIframeLoading}
        onOpenDetails={() => setIsDetailsOpen(true)}
        previewUrlInput={previewUrlInput}
        onPreviewUrlInputChange={handleUrlInputChange}
        onPreviewUrlInputBlur={handleUrlInputBlur}
        onPreviewUrlInputKeyDown={handleUrlInputKeyDown}
        onOpenInNewTab={onOpenInNewTab}
        openPreviewTarget={openPreviewTarget}
        urlStatusClass={lifecycle.urlStatusClass}
        urlStatusTitle={statusMessage ?? lifecycle.appStatusLabel}
        hasDetailsWarning={false}
        hasCurrentApp={hasScenarioContext}
        isAppRunning={isAppRunning}
        pendingAction={lifecycle.pendingAction}
        actionInProgress={actionInProgress}
        toggleActionLabel={toggleActionLabel}
        onToggleApp={() => {
          if (currentApp) {
            void lifecycle.handleToggleCurrentApp();
            return;
          }
          if (scenarioActionIdentifier) {
            void lifecycle.runAction(scenarioActionIdentifier, 'start');
          }
        }}
        restartActionLabel={restartActionLabel}
        onRestartApp={() => {
          if (currentApp) {
            void lifecycle.handleRestartCurrentApp();
            return;
          }
          if (scenarioActionIdentifier) {
            void lifecycle.runAction(scenarioActionIdentifier, 'restart');
          }
        }}
        onToggleLogs={() => setIsLogsVisible((value) => !value)}
        areLogsVisible={isLogsVisible}
        onReportIssue={handleOpenReportDialog}
        appStatusLabel={lifecycle.appStatusLabel}
        isFullView={isFullView}
        onToggleFullView={() => setIsFullView((value) => !value)}
        isDeviceEmulationActive={isDeviceEmulationActive}
        onToggleDeviceEmulation={toggleDeviceEmulation}
        canInspect={inspectState.supported}
        isInspecting={inspectState.active}
        onToggleInspect={inspector.handleToggleInspectMode}
        menuPortalContainer={paneSurfaceNode}
        canOpenTabsOverlay={false}
        previewInteractionSignal={previewReloadToken}
        issueCaptureCount={stagedCaptureCount}
        showDetailsButton={true}
        showLifecycleMenu={true}
        showDevMenu={true}
        rightInlineActions={paneActions}
        buildUrlSuggestionSections={buildUrlSuggestionSections}
        onSelectUrlSuggestion={applyPreviewUrlValue}
        onOpenScenarioSelector={handleOpenScenarioSelector}
        scenarioSelectorLabel="Replace this pane from scenarios"
      />

      {isDeviceEmulationActive && !isLogsVisible && (
        <div className="preview-pane__emulation-toolbar-wrap">
          <DeviceEmulationToolbar {...deviceToolbar} />
        </div>
      )}

      <ErrorBoundary fallback={SectionErrorFallback}>
        <PreviewInspectorPanel
          inspectState={inspectState}
          previewUrl={previewUrl}
          inspector={inspector}
        />
      </ErrorBoundary>

      <div className="preview-pane__body">
        {isLogsVisible ? (
          <AppLogsPanel
            app={currentApp}
            onClose={() => setIsLogsVisible(false)}
            {...logsState}
          />
        ) : previewUrl ? (
          <div
            className={clsx(
              'preview-pane__iframe-shell',
              isDeviceEmulationActive && 'preview-pane__iframe-shell--emulated',
            )}
            ref={(node) => {
              previewContainerRef.current = node;
              setPreviewContainerNode(node);
            }}
          >
            {isDeviceEmulationActive ? (
              <DeviceEmulationViewport {...deviceViewport}>
                <iframe
                  key={previewReloadToken}
                  ref={iframeRef}
                  src={previewUrl}
                  title={`${currentApp?.name ?? 'Application'} preview pane`}
                  className="preview-pane__iframe"
                  loading="eager"
                  onLoad={onIframeLoad}
                  onError={onIframeError}
                />
              </DeviceEmulationViewport>
            ) : (
              <div
                className={clsx(
                  'preview-pane__iframe-scale',
                  standardPreviewZoom !== 1 && 'preview-pane__iframe-scale--zoomed',
                )}
                style={standardPreviewStyle}
              >
                <iframe
                  key={previewReloadToken}
                  ref={iframeRef}
                  src={previewUrl}
                  title={`${currentApp?.name ?? 'Application'} preview pane`}
                  className="preview-pane__iframe"
                  loading="eager"
                  onLoad={onIframeLoad}
                  onError={onIframeError}
                />
              </div>
            )}
            {fallbackState && <PreviewFallbackState state={fallbackState} variant="overlay" />}
          </div>
        ) : (
          fallbackState ? (
            <PreviewFallbackState state={fallbackState} variant="panel" />
          ) : (
            <div className="preview-pane__empty" role="status">
              {statusMessage ?? 'Select an app to start this pane.'}
            </div>
          )
        )}
      </div>

      {isDetailsOpen && appForModal && (
        <AppModal
          app={appForModal}
          isOpen={isDetailsOpen}
          onClose={() => setIsDetailsOpen(false)}
          onAction={async (currentAppId, action) => {
            await lifecycle.runAction(currentAppId, action);
          }}
          onViewLogs={() => {
            setIsLogsVisible(true);
            setIsDetailsOpen(false);
          }}
          proxyMetadata={proxyMetadata}
          previewUrl={previewUrl}
          preloadedDiagnostics={diagnostics}
          diagnosticsLoading={diagnosticsLoading}
          diagnosticsError={diagnosticsError}
          onRefetchDiagnostics={refetchDiagnostics}
          preloadedLighthouseHistory={lighthouseHistory}
          lighthouseLoading={lighthouseLoading}
          lighthouseError={lighthouseError}
          onRefetchLighthouse={refetchLighthouse}
          preloadedCompleteness={completeness}
          completenessLoading={completenessLoading}
          completenessError={completenessError}
          onRefetchCompleteness={refetchCompleteness}
        />
      )}

      {reportDialogOpen && (
        <ErrorBoundary fallback={SectionErrorFallback}>
          <ReportIssueDialog
            isOpen={reportDialogOpen}
            onClose={() => setReportDialogOpen(false)}
            appId={currentApp?.id ?? resolvedAppIdentifier ?? undefined}
            app={currentApp}
            activePreviewUrl={openPreviewTarget || null}
            canCaptureScreenshot={canCaptureScreenshot}
            previewContainerRef={previewContainerRef}
            iframeRef={iframeRef}
            isPreviewSameOrigin={isPreviewSameOrigin}
            bridgeSupportsScreenshot={bridgeSupportsScreenshot}
            requestScreenshot={requestScreenshot}
            bridgeState={bridgeState}
            logState={logState}
            configureLogs={configureLogs}
            getRecentLogs={getRecentLogs}
            requestLogBatch={requestLogBatch}
            networkState={networkState}
            configureNetwork={configureNetwork}
            getRecentNetworkEvents={getRecentNetworkEvents}
            requestNetworkBatch={requestNetworkBatch}
            bridgeCompliance={bridgeCompliance}
            elementCaptures={reportElementCaptures}
            onElementCaptureNoteChange={handleElementCaptureNoteChange}
            onElementCaptureRemove={handleRemoveElementCapture}
            onElementCapturesReset={handleResetElementCaptures}
            onPrimaryCaptureDraftChange={setHasPrimaryCaptureDraft}
          />
        </ErrorBoundary>
      )}
    </div>
  );
}

export default PreviewPane;
