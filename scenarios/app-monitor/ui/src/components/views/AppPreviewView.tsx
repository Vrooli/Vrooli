import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, KeyboardEvent, MouseEvent } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { ArrowLeft, ArrowRight, ExternalLink, Info, RefreshCw, ScrollText } from 'lucide-react';
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
      }
    } catch (error) {
      logger.error(`Failed to ${action} app ${appToControl}`, error);
      setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
    }
  }, [setApps]);

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
            className="preview-toolbar__icon-btn"
            onClick={handleViewLogs}
            disabled={!currentApp}
            aria-label="View logs"
            title="View logs"
          >
            <ScrollText aria-hidden size={18} />
          </button>
        </div>
      </div>

      {statusMessage && (
        <div className="preview-status">
          {statusMessage}
        </div>
      )}

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
    </div>
  );
};

export default AppPreviewView;
