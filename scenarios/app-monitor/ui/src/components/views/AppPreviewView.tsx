import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { App } from '@/types';
import AppModal from '../AppModal';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus, locateAppByIdentifier } from '@/utils/appPreview';
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
  const [statusMessage, setStatusMessage] = useState<string | null>('Loading application preview...');
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [fetchAttempted, setFetchAttempted] = useState(false);

  useEffect(() => {
    if (!appId) {
      navigate('/apps', { replace: true });
    }
  }, [appId, navigate]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const located = locateAppByIdentifier(apps, appId);
    if (located) {
      setCurrentApp(located);
      setLoading(false);
      return;
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
      return;
    }

    setLoading(false);

    if (!isRunningStatus(currentApp.status) || isStoppedStatus(currentApp.status)) {
      setPreviewUrl(null);
      setStatusMessage('Application is not running. Start it from the Applications view to access the UI preview.');
      return;
    }

    const url = buildPreviewUrl(currentApp);
    if (!url) {
      setPreviewUrl(null);
      setStatusMessage('This application does not expose a UI endpoint to preview.');
      return;
    }

    setPreviewUrl(url);
    setStatusMessage(null);
  }, [currentApp]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const match = locateAppByIdentifier(apps, appId);
    if (match) {
      setCurrentApp(match);
    }
  }, [apps, appId]);

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
          setStatusMessage(null);
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
  }, [appId, setApps]);

  const handleBack = useCallback(() => {
    navigate('/apps');
  }, [navigate]);

  const handleViewLogs = useCallback(() => {
    if (currentApp) {
      navigate(`/logs/${currentApp.id}`);
    }
  }, [currentApp, navigate]);

  const statusBadgeClass = useMemo(() => {
    if (!currentApp) {
      return 'status-badge unknown';
    }
    return clsx('status-badge', currentApp.status?.toLowerCase() || 'unknown');
  }, [currentApp]);

  return (
    <div className="app-preview-view">
      <div className="preview-toolbar">
        <div className="preview-toolbar__left">
          <button className="control-btn" onClick={handleBack}>
            ← APPS
          </button>
        </div>
        <div className="preview-toolbar__center">
          <h2>{currentApp?.name ?? 'Loading Application...'}</h2>
          {currentApp && (
            <span className={statusBadgeClass}>{currentApp.status.toUpperCase()}</span>
          )}
        </div>
        <div className="preview-toolbar__actions">
          <button className="control-btn" onClick={handleRefresh} disabled={loading}>
            {loading ? 'REFRESHING' : 'REFRESH'}
          </button>
          {previewUrl && (
            <a
              href={previewUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="control-btn"
              title="Open in new tab"
            >
              ↗
            </a>
          )}
          <button
            className="control-btn"
            onClick={() => setModalOpen(true)}
            disabled={!currentApp}
          >
            DETAILS
          </button>
          <button
            className="control-btn"
            onClick={handleViewLogs}
            disabled={!currentApp}
          >
            LOGS
          </button>
        </div>
      </div>

      {statusMessage && (
        <div className="preview-status">
          {statusMessage}
        </div>
      )}

      {previewUrl ? (
        <div className="preview-iframe-container">
          <iframe
            src={previewUrl}
            title={`${currentApp?.name ?? 'Application'} preview`}
            className="preview-iframe"
            loading="lazy"
          />
        </div>
      ) : (
        <div className="preview-placeholder">
          {loading ? 'Loading preview...' : 'Preview unavailable.'}
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
