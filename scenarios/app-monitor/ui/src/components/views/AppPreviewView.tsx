import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MouseEvent } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { ArrowLeft, ArrowRight, Bug, ExternalLink, Info, Loader2, Power, RefreshCw, RotateCcw, ScrollText, X } from 'lucide-react';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { App } from '@/types';
import AppModal from '../AppModal';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus, locateAppByIdentifier } from '@/utils/appPreview';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import './AppPreviewView.css';

interface AppPreviewViewProps {
  apps: App[];
  setApps: React.Dispatch<React.SetStateAction<App[]>>;
}

const AppPreviewView = ({ apps, setApps }: AppPreviewViewProps) => {
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
  const [currentApp, setCurrentApp] = useState<App | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewUrlInput, setPreviewUrlInput] = useState('');
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>('Loading application preview...');
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [fetchAttempted, setFetchAttempted] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [pendingAction, setPendingAction] = useState<null | 'start' | 'stop' | 'restart'>(null);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [reportMessage, setReportMessage] = useState('');
  const [reportIncludeScreenshot, setReportIncludeScreenshot] = useState(false);
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [reportResult, setReportResult] = useState<{ issueId?: string; message?: string } | null>(null);
  const [reportScreenshotData, setReportScreenshotData] = useState<string | null>(null);
  const [reportScreenshotLoading, setReportScreenshotLoading] = useState(false);
  const [reportScreenshotError, setReportScreenshotError] = useState<string | null>(null);
  const [reportScreenshotRequestId, setReportScreenshotRequestId] = useState(0);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const complianceRunRef = useRef(false);
  const initialPreviewUrlRef = useRef<string | null>(null);

  const handleBridgeLocation = useCallback((message: { href: string; title?: string | null }) => {
    if (message.href) {
      setPreviewUrlInput(message.href);
      if (!initialPreviewUrlRef.current) {
        initialPreviewUrlRef.current = message.href;
      }
      setHasCustomPreviewUrl(prev => {
        if (prev) {
          return prev;
        }
        const base = initialPreviewUrlRef.current;
        if (!base) {
          return prev;
        }
        const normalize = (value: string) => value.replace(/\/$/, '');
        return normalize(message.href) !== normalize(base);
      });
    }
    setStatusMessage(null);
  }, []);

  const { state: bridgeState, childOrigin, sendNav: sendBridgeNav, runComplianceCheck, resetState } = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
  });

  const activePreviewUrl = useMemo(() => bridgeState.href || previewUrl || '', [bridgeState.href, previewUrl]);
  const canCaptureScreenshot = useMemo(() => Boolean(activePreviewUrl), [activePreviewUrl]);
  const targetAppIdentifier = useMemo(() => currentApp?.id ?? appId ?? '', [appId, currentApp]);

  useEffect(() => {
    if (!appId) {
      navigate('/apps', { replace: true });
    }
  }, [appId, navigate]);

  useEffect(() => {
    setFetchAttempted(false);
  }, [appId]);

  useEffect(() => {
    setHasCustomPreviewUrl(false);
    setHistory([]);
    setHistoryIndex(-1);
    complianceRunRef.current = false;
    setBridgeCompliance(null);
    resetState();
  }, [appId]);

  useEffect(() => {
    complianceRunRef.current = false;
    setBridgeCompliance(null);
  }, [previewUrl]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const located = locateAppByIdentifier(apps, appId);
    if (located) {
      setCurrentApp(located);
      if (!located.is_partial) {
        setLoading(false);
        return;
      }

      setStatusMessage('Loading application details...');
      setLoading(true);
    }

    if (fetchAttempted) {
      return;
    }

    setFetchAttempted(true);
    const fetchApp = async () => {
      try {
        const fetched = await appService.getApp(appId);
        if (fetched) {
          setCurrentApp(fetched);
          setApps(prev => {
            const index = prev.findIndex(app => app.id === fetched.id);
            if (index === -1) {
              return [...prev, fetched];
            }

            const updated = [...prev];
            updated[index] = fetched;
            return updated;
          });
          setStatusMessage(null);
        } else {
          setStatusMessage('Application not found.');
        }
      } catch (error) {
        logger.error('Failed to load application for preview', error);
        setStatusMessage('Failed to load application details.');
      } finally {
        setLoading(false);
      }
    };

    fetchApp().catch((error) => {
      logger.error('Preview fetch failed', error);
    });
  }, [appId, apps, fetchAttempted, setApps]);

  useEffect(() => {
    if (!currentApp) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      return;
    }

    if (currentApp.is_partial) {
      setStatusMessage('Loading application details...');
      setLoading(true);
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      return;
    }

    setLoading(false);

    if (!isRunningStatus(currentApp.status) || isStoppedStatus(currentApp.status)) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      setStatusMessage('Application is not running. Start it from the Applications view to access the UI preview.');
      return;
    }

    const url = buildPreviewUrl(currentApp);
    if (!url) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      setStatusMessage('This application does not expose a UI endpoint to preview.');
      return;
    }

    setStatusMessage(null);
    if (!hasCustomPreviewUrl) {
      initialPreviewUrlRef.current = url;
      setPreviewUrl(url);
      setPreviewUrlInput(url);
      setHistory([url]);
      setHistoryIndex(0);
    } else if (previewUrl === null) {
      initialPreviewUrlRef.current = url;
      setPreviewUrl(url);
    }
  }, [currentApp, hasCustomPreviewUrl, previewUrl]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const match = locateAppByIdentifier(apps, appId);
    if (match) {
      setCurrentApp(match);
    }
  }, [apps, appId]);

  useEffect(() => {
    if (!bridgeState.isSupported || !bridgeState.isReady || !bridgeState.href) {
      return;
    }
    if (complianceRunRef.current) {
      return;
    }

    let cancelled = false;
    complianceRunRef.current = true;
    runComplianceCheck()
      .then(result => {
        if (!cancelled) {
          setBridgeCompliance(result);
        }
      })
      .catch(error => {
        logger.warn('Bridge compliance check failed', error);
        if (!cancelled) {
          setBridgeCompliance({ ok: false, failures: ['CHECK_FAILED'], checkedAt: Date.now() });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [bridgeState.href, bridgeState.isReady, bridgeState.isSupported, runComplianceCheck]);

  const handleAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    setPendingAction(action);
    const actionInProgressMessage = action === 'stop'
      ? 'Stopping application...'
      : action === 'start'
        ? 'Starting application...'
        : 'Restarting application...';
    setStatusMessage(actionInProgressMessage);

    try {
      const success = await appService.controlApp(appToControl, action);
      if (success) {
        setApps(prev => prev.map(app => {
          if (app.id === appToControl) {
            return {
              ...app,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString(),
            };
          }
          return app;
        }));
        setCurrentApp(prev => {
          if (prev && prev.id === appToControl) {
            return {
              ...prev,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString(),
            };
          }
          return prev;
        });
        setStatusMessage(action === 'stop'
          ? 'Application stopped. Start it again to relaunch the UI preview.'
          : action === 'start'
            ? 'Application started. Preview will refresh automatically.'
            : 'Application restarted. Preview will refresh automatically.');
      } else {
        setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
      }
    } catch (error) {
      logger.error(`Failed to ${action} app ${appToControl}`, error);
      setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
    } finally {
      setPendingAction(null);
    }
  }, [setApps]);

  const handleToggleApp = useCallback(() => {
    if (!currentApp || pendingAction) {
      return;
    }

    const action: 'start' | 'stop' = isRunningStatus(currentApp.status) ? 'stop' : 'start';
    handleAppAction(currentApp.id, action);
  }, [currentApp, handleAppAction, pendingAction]);

  const handleRestartApp = useCallback(() => {
    if (!currentApp || pendingAction || !isRunningStatus(currentApp.status)) {
      return;
    }

    handleAppAction(currentApp.id, 'restart');
  }, [currentApp, handleAppAction, pendingAction]);

  const applyPreviewUrlInput = useCallback(() => {
    const trimmed = previewUrlInput.trim();

    if (!trimmed) {
      if (previewUrlInput !== '') {
        setPreviewUrlInput('');
      }
      setHasCustomPreviewUrl(false);
      return;
    }

    if (trimmed !== previewUrlInput) {
      setPreviewUrlInput(trimmed);
    }

    if (bridgeState.isSupported) {
      try {
        const reference = bridgeState.href || previewUrl || window.location.href;
        const resolved = new URL(trimmed, reference);
        if (!childOrigin || resolved.origin === childOrigin) {
          const sent = sendBridgeNav('GO', resolved.href);
          if (sent) {
            setStatusMessage(null);
            return;
          }
        }
      } catch (error) {
        logger.warn('Bridge navigation failed to parse URL', error);
      }
    }

    setHasCustomPreviewUrl(true);
    setPreviewUrl(trimmed);
    initialPreviewUrlRef.current = trimmed;
    resetState();
    setStatusMessage(null);
    const baseHistory = historyIndex >= 0 ? history.slice(0, historyIndex + 1) : [];
    if (baseHistory[baseHistory.length - 1] === trimmed) {
      setHistory(baseHistory);
      setHistoryIndex(baseHistory.length - 1);
    } else {
      const updatedHistory = [...baseHistory, trimmed];
      setHistory(updatedHistory);
      setHistoryIndex(updatedHistory.length - 1);
    }
  }, [bridgeState.href, bridgeState.isSupported, childOrigin, history, historyIndex, previewUrl, previewUrlInput, resetState, sendBridgeNav]);

  const handleUrlInputChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setPreviewUrlInput(event.target.value);
  }, []);

  const handleUrlInputKeyDown = useCallback((event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      applyPreviewUrlInput();
    }
  }, [applyPreviewUrlInput]);

  const handleUrlInputBlur = useCallback(() => {
    applyPreviewUrlInput();
  }, [applyPreviewUrlInput]);

  const handleRefresh = useCallback(() => {
    if (!appId) {
      return;
    }

    setLoading(true);
    setStatusMessage('Refreshing application status...');

    appService.getApp(appId)
      .then(fetched => {
        if (fetched) {
          setCurrentApp(fetched);
          setApps(prev => {
            const index = prev.findIndex(app => app.id === fetched.id);
            if (index === -1) {
              return [...prev, fetched];
            }

            const updated = [...prev];
            updated[index] = fetched;
            return updated;
          });
          if (!hasCustomPreviewUrl) {
            setStatusMessage(null);
          }
        } else {
          setStatusMessage('Application not found.');
        }
      })
      .catch(error => {
        logger.error('Failed to refresh application preview', error);
        setStatusMessage('Failed to refresh application preview.');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, hasCustomPreviewUrl, setApps]);

  const handleGoBack = useCallback(() => {
    if (bridgeState.isSupported) {
      sendBridgeNav('BACK');
      return;
    }

    if (historyIndex <= 0) {
      return;
    }

    const targetIndex = historyIndex - 1;
    const targetUrl = history[targetIndex];
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl);
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav]);

  const handleGoForward = useCallback(() => {
    if (bridgeState.isSupported) {
      sendBridgeNav('FWD');
      return;
    }

    if (historyIndex === -1 || historyIndex >= history.length - 1) {
      return;
    }

    const targetIndex = historyIndex + 1;
    const targetUrl = history[targetIndex];
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl);
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav]);

  const handleOpenPreviewInNewTab = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    const target = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;
    if (!target) {
      return;
    }

    event.preventDefault();
    window.open(target, '_blank', 'noopener,noreferrer');
  }, [bridgeState.href, bridgeState.isSupported, previewUrl]);

  const handleViewLogs = useCallback(() => {
    if (currentApp) {
      navigate(`/logs/${currentApp.id}`);
    }
  }, [currentApp, navigate]);

  const handleOpenReportDialog = useCallback(() => {
    setReportDialogOpen(true);
    setReportMessage('');
    setReportError(null);
    setReportResult(null);
    setReportScreenshotData(null);
    setReportScreenshotError(null);
    setReportScreenshotLoading(false);
    setReportIncludeScreenshot(canCaptureScreenshot);
    setReportScreenshotRequestId(prev => prev + 1);
  }, [canCaptureScreenshot]);

  const handleCloseReportDialog = useCallback(() => {
    setReportDialogOpen(false);
    setReportSubmitting(false);
    setReportError(null);
    setReportResult(null);
    setReportMessage('');
    setReportIncludeScreenshot(canCaptureScreenshot);
    setReportScreenshotData(null);
    setReportScreenshotError(null);
    setReportScreenshotLoading(false);
    setReportScreenshotRequestId(0);
  }, [canCaptureScreenshot]);

  const handleReportMessageChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setReportMessage(event.target.value);
    if (reportError) {
      setReportError(null);
    }
  }, [reportError]);

  const handleReportIncludeScreenshotChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = event.target.checked;
    setReportIncludeScreenshot(nextValue);
    if (nextValue) {
      setReportScreenshotRequestId(prev => prev + 1);
    } else {
      setReportScreenshotData(null);
      setReportScreenshotError(null);
      setReportScreenshotLoading(false);
    }
  }, []);

  const handleRetryScreenshotCapture = useCallback(() => {
    setReportScreenshotRequestId(prev => prev + 1);
  }, []);

  const handleSubmitReport = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmed = reportMessage.trim();
    if (!trimmed) {
      setReportError('Please add a short description of the issue.');
      return;
    }

    const targetAppId = currentApp?.id ?? appId ?? '';
    if (!targetAppId) {
      setReportError('Unable to determine which application to report.');
      return;
    }

    setReportSubmitting(true);
    setReportError(null);

    try {
      const includeScreenshot = reportIncludeScreenshot && canCaptureScreenshot;
      const previewContextUrl = activePreviewUrl || null;

      const response = await appService.reportAppIssue(targetAppId, {
        message: trimmed,
        includeScreenshot,
        previewUrl: previewContextUrl,
        appName: currentApp?.name ?? null,
        scenarioName: currentApp?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? reportScreenshotData ?? null : null,
      });

      const issueId = response.data?.issue_id;
      setReportResult({
        issueId,
        message: response.message ?? 'Issue report sent successfully.',
      });
      setReportMessage('');
    } catch (error: unknown) {
      const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to send issue report.';
      setReportError(fallbackMessage);
    } finally {
      setReportSubmitting(false);
    }
  }, [activePreviewUrl, appId, canCaptureScreenshot, currentApp, reportIncludeScreenshot, reportMessage]);

  useEffect(() => {
    if (!reportDialogOpen) {
      return;
    }

    const keyListener = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape' && !reportSubmitting) {
        event.preventDefault();
        handleCloseReportDialog();
      }
    };

    window.addEventListener('keydown', keyListener);
    return () => {
      window.removeEventListener('keydown', keyListener);
    };
  }, [handleCloseReportDialog, reportDialogOpen, reportSubmitting]);

  useEffect(() => {
    if (!reportDialogOpen) {
      return;
    }

    if (!reportIncludeScreenshot) {
      setReportScreenshotLoading(false);
      setReportScreenshotError(null);
      return;
    }

    if (!canCaptureScreenshot || !activePreviewUrl || !targetAppIdentifier) {
      setReportScreenshotLoading(false);
      if (!canCaptureScreenshot) {
        setReportScreenshotError(null);
      }
      return;
    }

    let cancelled = false;
    setReportScreenshotLoading(true);
    setReportScreenshotError(null);

    appService.fetchReportScreenshot(targetAppIdentifier, activePreviewUrl)
      .then((response) => {
        if (cancelled) {
          return;
        }
        const screenshot = response.data?.screenshot ?? null;
        setReportScreenshotData(screenshot ?? null);
        if (!screenshot) {
          setReportScreenshotError('Screenshot capture returned no data.');
        }
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return;
        }
        const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to capture screenshot.';
        setReportScreenshotError(fallbackMessage);
        setReportScreenshotData(null);
      })
      .finally(() => {
        if (!cancelled) {
          setReportScreenshotLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [reportDialogOpen, reportIncludeScreenshot, canCaptureScreenshot, activePreviewUrl, targetAppIdentifier, reportScreenshotRequestId]);

  const urlStatusClass = useMemo(() => {
    if (!currentApp) {
      return 'unknown';
    }
    return currentApp.status?.toLowerCase() || 'unknown';
  }, [currentApp]);

  const urlStatusTitle = useMemo(() => {
    if (!currentApp) {
      return 'Status: Unknown';
    }
    const status = currentApp.status ?? 'Unknown';
    return `Status: ${status}`;
  }, [currentApp]);

  const isAppRunning = useMemo(() => (currentApp ? isRunningStatus(currentApp.status) : false), [currentApp]);
  const scenarioDisplayName = useMemo(() => {
    if (!currentApp) {
      return 'application';
    }
    return currentApp.name || currentApp.scenario_name || currentApp.id || 'application';
  }, [currentApp]);

  const toggleActionLabel = isAppRunning ? 'Stop scenario' : 'Start scenario';
  const restartActionLabel = `Restart ${scenarioDisplayName}`;
  const toggleTooltip = `${toggleActionLabel}${currentApp ? ` (${currentApp.id})` : ''}`;
  const actionInProgress = pendingAction !== null;

  const bridgeIssueMessage = useMemo(() => {
    if (!bridgeState.isSupported || !bridgeCompliance || bridgeCompliance.ok) {
      return null;
    }
    const detail = bridgeCompliance.failures.join(', ');
    return `Preview bridge diagnostics failed (${detail}). History syncing may be unreliable.`;
  }, [bridgeCompliance, bridgeState.isSupported]);

  const canGoBack = bridgeState.isSupported ? bridgeState.canGoBack : historyIndex > 0;
  const canGoForward = bridgeState.isSupported ? bridgeState.canGoForward : (historyIndex >= 0 && historyIndex < history.length - 1);
  const openPreviewTarget = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;

  return (
    <div className="app-preview-view">
      <div className="preview-toolbar">
        <div className="preview-toolbar__group preview-toolbar__group--left">
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleGoBack}
            disabled={!canGoBack}
            aria-label={canGoBack ? 'Go back' : 'No previous page'}
            title={canGoBack ? 'Go back' : 'No previous page'}
          >
            <ArrowLeft aria-hidden size={18} />
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleGoForward}
            disabled={!canGoForward}
            aria-label={canGoForward ? 'Go forward' : 'No forward page'}
            title={canGoForward ? 'Go forward' : 'No forward page'}
          >
            <ArrowRight aria-hidden size={18} />
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleRefresh}
            disabled={loading}
            aria-label={loading ? 'Refreshing application status' : 'Refresh application'}
            title={loading ? 'Refreshing…' : 'Refresh'}
          >
            <RefreshCw aria-hidden size={18} className={clsx({ spinning: loading })} />
          </button>
          <div className="preview-toolbar__title">
            <div
              className={clsx('preview-toolbar__url-wrapper', urlStatusClass)}
              title={urlStatusTitle}
            >
              <button
                type="button"
                className="preview-toolbar__url-action-btn"
                onClick={() => setModalOpen(true)}
                disabled={!currentApp}
                aria-label="Application details"
                title="Application details"
              >
                <Info aria-hidden size={16} />
              </button>
              <input
                type="text"
                className="preview-toolbar__url-input"
                value={previewUrlInput}
                onChange={handleUrlInputChange}
                onBlur={handleUrlInputBlur}
                onKeyDown={handleUrlInputKeyDown}
                placeholder="Enter preview URL"
                aria-label="Preview URL"
                autoComplete="off"
                spellCheck={false}
                inputMode="url"
              />
              <button
                type="button"
                className="preview-toolbar__url-action-btn"
                onClick={handleOpenPreviewInNewTab}
                disabled={!openPreviewTarget}
                aria-label={openPreviewTarget ? 'Open preview in new tab' : 'Preview unavailable'}
                title={openPreviewTarget ? 'Open in new tab' : 'Preview unavailable'}
              >
                <ExternalLink aria-hidden size={16} />
              </button>
            </div>
          </div>
        </div>
        <div className="preview-toolbar__group preview-toolbar__group--right">
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              isAppRunning && 'preview-toolbar__icon-btn--danger',
              (pendingAction === 'start' || pendingAction === 'stop') && 'preview-toolbar__icon-btn--waiting',
            )}
            onClick={handleToggleApp}
            disabled={!currentApp || actionInProgress}
            aria-label={toggleTooltip}
            title={toggleTooltip}
          >
            {(pendingAction === 'start' || pendingAction === 'stop') ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <Power aria-hidden size={18} />
            )}
          </button>
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              'preview-toolbar__icon-btn--secondary',
              pendingAction === 'restart' && 'preview-toolbar__icon-btn--waiting',
            )}
            onClick={handleRestartApp}
            disabled={!currentApp || !isAppRunning || actionInProgress}
            aria-label={restartActionLabel}
            title={restartActionLabel}
          >
            {pendingAction === 'restart' ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <RotateCcw aria-hidden size={18} />
            )}
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleViewLogs}
            disabled={!currentApp}
            aria-label="View logs"
            title="View logs"
          >
            <ScrollText aria-hidden size={18} />
          </button>
          <button
            type="button"
            className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--report')}
            onClick={handleOpenReportDialog}
            disabled={!currentApp}
            aria-label="Report an issue"
            title="Report an issue"
          >
            <Bug aria-hidden size={18} />
          </button>
        </div>
      </div>

      {bridgeIssueMessage && (
        <div className="preview-status">
          {bridgeIssueMessage}
        </div>
      )}

      {previewUrl ? (
        <div className="preview-iframe-container">
          <iframe
            src={previewUrl}
            title={`${currentApp?.name ?? 'Application'} preview`}
            className="preview-iframe"
            loading="lazy"
            ref={iframeRef}
          />
        </div>
      ) : (
        <div className="preview-placeholder">
          {loading ? 'Fetching application details…' : statusMessage ?? 'Preview unavailable.'}
        </div>
      )}

      {modalOpen && currentApp && (
        <AppModal
          app={currentApp}
          isOpen={modalOpen}
          onClose={() => setModalOpen(false)}
          onAction={handleAppAction}
          onViewLogs={(appIdentifier) => {
            setModalOpen(false);
            navigate(`/logs/${appIdentifier}`);
          }}
        />
      )}

      {reportDialogOpen && (
        <div
          className="report-dialog__overlay"
          role="presentation"
          onClick={() => {
            if (!reportSubmitting) {
              handleCloseReportDialog();
            }
          }}
        >
          <div
            className="report-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="app-report-dialog-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="report-dialog__header">
              <h2 id="app-report-dialog-title">Report an Issue</h2>
              <button
                type="button"
                className="report-dialog__close"
                onClick={handleCloseReportDialog}
                disabled={reportSubmitting}
                aria-label="Close report dialog"
              >
                <X aria-hidden size={16} />
              </button>
            </div>

            {reportResult ? (
              <div className="report-dialog__state">
                <p className="report-dialog__success">
                  {reportResult.message ?? 'Issue report sent successfully.'}
                </p>
                {reportResult.issueId && (
                  <p className="report-dialog__success-id">
                    Tracking ID: <span>{reportResult.issueId}</span>
                  </p>
                )}
                <div className="report-dialog__actions">
                  <button
                    type="button"
                    className="report-dialog__button report-dialog__button--primary"
                    onClick={handleCloseReportDialog}
                  >
                    Close
                  </button>
                </div>
              </div>
            ) : (
              <form className="report-dialog__form" onSubmit={handleSubmitReport}>
                <label htmlFor="app-report-message" className="report-dialog__label">
                  Describe the issue
                </label>
                <textarea
                  id="app-report-message"
                  className="report-dialog__textarea"
                  value={reportMessage}
                  onChange={handleReportMessageChange}
                  rows={6}
                  placeholder="What are you seeing? Include steps to reproduce if possible."
                  disabled={reportSubmitting}
                  required
                />

                <label className="report-dialog__checkbox">
                  <input
                    type="checkbox"
                    checked={reportIncludeScreenshot && canCaptureScreenshot}
                    onChange={handleReportIncludeScreenshotChange}
                    disabled={!canCaptureScreenshot || reportSubmitting}
                  />
                  <span>Include screenshot of the current preview</span>
                </label>
                {!canCaptureScreenshot && (
                  <p className="report-dialog__hint">Load the preview to capture a screenshot.</p>
                )}
                {reportIncludeScreenshot && canCaptureScreenshot && (
                  <div className="report-dialog__preview" aria-live="polite">
                    {reportScreenshotLoading ? (
                      <div className="report-dialog__preview-loading">
                        <Loader2 aria-hidden size={18} className="spinning" />
                        <span>Capturing screenshot…</span>
                      </div>
                    ) : reportScreenshotError ? (
                      <div className="report-dialog__preview-error">
                        <p>{reportScreenshotError}</p>
                        <button
                          type="button"
                          className="report-dialog__button report-dialog__button--ghost"
                          onClick={handleRetryScreenshotCapture}
                          disabled={reportScreenshotLoading}
                        >
                          Retry capture
                        </button>
                      </div>
                    ) : reportScreenshotData ? (
                      <div className="report-dialog__preview-image">
                        <img
                          src={`data:image/png;base64,${reportScreenshotData}`}
                          alt="Preview screenshot"
                          loading="lazy"
                        />
                      </div>
                    ) : (
                      <p className="report-dialog__preview-hint">Ready to capture the current preview.</p>
                    )}
                  </div>
                )}

                {reportError && (
                  <p className="report-dialog__error" role="alert">
                    {reportError}
                  </p>
                )}

                <div className="report-dialog__actions">
                  <button
                    type="button"
                    className="report-dialog__button"
                    onClick={handleCloseReportDialog}
                    disabled={reportSubmitting}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="report-dialog__button report-dialog__button--primary"
                    disabled={reportSubmitting}
                  >
                    {reportSubmitting ? (
                      <>
                        <Loader2 aria-hidden size={16} className="spinning" />
                        Sending…
                      </>
                    ) : (
                      'Send Report'
                    )}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AppPreviewView;
