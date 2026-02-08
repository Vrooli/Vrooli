import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import { usePreviewWorkspaceStore } from '../state/previewWorkspaceStore';
import type { App } from '@/types';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { isRunningStatus, locateAppByIdentifier } from '@/utils/appPreview';
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
  const paneViewState = usePreviewWorkspaceStore((state) => state.paneViewState[paneId]);
  const setPaneViewState = usePreviewWorkspaceStore((state) => state.setPaneViewState);
  const resetPaneViewState = usePreviewWorkspaceStore((state) => state.resetPaneViewState);
  const [currentApp, setCurrentApp] = useState<App | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>('Select an app to preview.');
  const [loading, setLoading] = useState(false);
  const [isLogsVisible, setIsLogsVisible] = useState(() => paneViewState?.isLogsVisible ?? false);
  const [isDetailsOpen, setIsDetailsOpen] = useState(false);
  const [isFullView, setIsFullView] = useState(false);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const [paneSurfaceNode, setPaneSurfaceNode] = useState<HTMLDivElement | null>(null);
  const [previewContainerNode, setPreviewContainerNode] = useState<HTMLDivElement | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const paneRef = useRef<HTMLDivElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);

  const activeAppIdentifier = useMemo(() => {
    if (!appId) {
      return null;
    }
    const trimmed = appId.trim();
    return trimmed.length > 0 ? trimmed : null;
  }, [appId]);

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
    clearNavigationSession,
    history,
  } = navigationSession;
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

  const deviceEmulation = useDeviceEmulation({ container: previewContainerNode });
  const {
    isActive: isDeviceEmulationActive,
    toggleActive: toggleDeviceEmulation,
    toolbar: deviceToolbar,
    viewport: deviceViewport,
  } = deviceEmulation;

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

  usePreviewBridgeComplianceCheck({
    enabled: reportDialogOpen,
    runComplianceCheck,
    onSuccess: setBridgeCompliance,
    onError: (error) => {
      logger.debug('[preview-pane] Bridge compliance check unavailable', error);
      setBridgeCompliance(null);
    },
  });

  useEffect(() => {
    if (!activeAppIdentifier) {
      setCurrentApp(null);
      setLoading(false);
      setStatusMessage('Select an app to preview.');
      setIsLogsVisible(false);
      resetPaneViewState(paneId);
      setIframeLoadError(null);
      setReportDialogOpen(false);
      resetReportDraftState();
      clearNavigationSession();
      resetPreviewState({ force: true });
      resetBridgeState();
      return;
    }

    const localMatch = locateAppByIdentifier(apps, activeAppIdentifier);
    if (localMatch) {
      setCurrentApp(localMatch);
      return;
    }

    setLoading(true);
    setStatusMessage('Loading app metadata...');

    let cancelled = false;
    appService.getApp(activeAppIdentifier)
      .then((fetched) => {
        if (cancelled) {
          return;
        }
        if (!fetched) {
          setCurrentApp(null);
          setStatusMessage('App not found.');
          setLoading(false);
          return;
        }
        setCurrentApp(fetched);
      })
      .catch((error) => {
        if (!cancelled) {
          logger.warn('[preview-pane] Failed to load app', error);
          setCurrentApp(null);
          setStatusMessage('Failed to load app metadata.');
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    activeAppIdentifier,
    apps,
    clearNavigationSession,
    paneId,
    resetBridgeState,
    resetPaneViewState,
    resetPreviewState,
    resetReportDraftState,
  ]);

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
    if (!currentApp) {
      return;
    }

    const { hasPreviewCandidate } = syncPreviewUrl({
      appForPreview: currentApp,
    });
    if (!hasPreviewCandidate) {
      setLoading(false);
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
    setIframeLoadError(null);
    if (previewUrl) {
      setLoading(true);
    }
  }, [previewUrl, previewReloadToken]);

  const logsState = useAppLogs({
    app: currentApp,
    appId: currentApp?.id ?? activeAppIdentifier,
    active: isLogsVisible,
  });

  const onRefresh = useCallback(() => {
    if (!activeAppIdentifier) {
      return;
    }

    setLoading(true);
    setStatusMessage('Refreshing preview...');
    setIframeLoadError(null);
    resetBridgeState();
    setPreviewReloadToken((value) => value + 1);

    appService.getApp(activeAppIdentifier)
      .then((fetched) => {
        if (fetched) {
          setCurrentApp(fetched);
        }
      })
      .catch((error) => {
        logger.warn('[preview-pane] Failed to refresh app', error);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [activeAppIdentifier, resetBridgeState]);

  const {
    openPreviewTarget,
    urlSuggestions,
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
    currentAppIdentifier: activeAppIdentifier,
    iframeRef,
    previewViewRef: paneRef,
    previewViewNode: paneSurfaceNode,
    onCaptureAdd: handleInspectorCaptureAdded,
    onViewReportRequest: handleOpenReportDialog,
  });

  const onIframeLoad = useCallback(() => {
    setLoading(false);
    setIframeLoadError(null);
    setStatusMessage(null);
  }, []);

  const onIframeError = useCallback(() => {
    setLoading(false);
    setIframeLoadError('Preview iframe failed to load.');
  }, []);

  usePreviewIframeReadinessFallback({
    iframeRef,
    enabled: loading && Boolean(previewUrl) && !iframeLoadError,
    onReady: () => {
      setLoading(false);
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

  return (
    <div
      className={clsx(
        'preview-pane',
        isFocused && 'preview-pane--focused',
        isFullView && 'preview-pane--full-view',
        isBeingDragged && 'preview-pane--dragging',
      )}
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
        isRefreshing={loading}
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
        hasCurrentApp={Boolean(currentApp)}
        isAppRunning={isAppRunning}
        pendingAction={lifecycle.pendingAction}
        actionInProgress={actionInProgress}
        toggleActionLabel={toggleActionLabel}
        onToggleApp={() => {
          void lifecycle.handleToggleCurrentApp();
        }}
        restartActionLabel={restartActionLabel}
        onRestartApp={() => {
          void lifecycle.handleRestartCurrentApp();
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
        urlSuggestions={urlSuggestions}
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
                  loading="lazy"
                  onLoad={onIframeLoad}
                  onError={onIframeError}
                />
              </DeviceEmulationViewport>
            ) : (
              <iframe
                key={previewReloadToken}
                ref={iframeRef}
                src={previewUrl}
                title={`${currentApp?.name ?? 'Application'} preview pane`}
                className="preview-pane__iframe"
                loading="lazy"
                onLoad={onIframeLoad}
                onError={onIframeError}
              />
            )}
            {loading && !iframeLoadError && (
              <div className="preview-pane__overlay" role="status">Loading preview...</div>
            )}
            {iframeLoadError && (
              <div className="preview-pane__overlay preview-pane__overlay--error" role="status">{iframeLoadError}</div>
            )}
          </div>
        ) : (
          <div className="preview-pane__empty" role="status">
            {statusMessage ?? 'Select an app to start this pane.'}
          </div>
        )}
      </div>

      {currentApp && (
        <AppModal
          app={currentApp}
          isOpen={isDetailsOpen}
          onClose={() => setIsDetailsOpen(false)}
          onAction={async (currentAppId, action) => {
            await lifecycle.runAction(currentAppId, action);
          }}
          onViewLogs={() => {
            setIsLogsVisible(true);
            setIsDetailsOpen(false);
          }}
          previewUrl={previewUrl}
        />
      )}

      {reportDialogOpen && (
        <ErrorBoundary fallback={SectionErrorFallback}>
          <ReportIssueDialog
            isOpen={reportDialogOpen}
            onClose={() => setReportDialogOpen(false)}
            appId={currentApp?.id ?? activeAppIdentifier ?? undefined}
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
