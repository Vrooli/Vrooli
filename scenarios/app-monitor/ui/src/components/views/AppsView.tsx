import React, { useState, useEffect, useCallback, useMemo, memo } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { App, AppViewMode } from '@/types';
import AppCard from '../AppCard';
import AppModal from '../AppModal';
import { AppsGridSkeleton } from '../LoadingSkeleton';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus } from '@/utils/appPreview';
import './AppsView.css';

interface AppsViewProps {
  apps: App[];
  setApps: React.Dispatch<React.SetStateAction<App[]>>;
}

// Memoized search input component
const SearchInput = memo(({ value, onChange }: { value: string; onChange: (value: string) => void }) => (
  <input
    type="text"
    className="search-box"
    placeholder="SEARCH..."
    value={value}
    onChange={(e) => onChange(e.target.value)}
  />
));
SearchInput.displayName = 'SearchInput';

// Memoized view toggle button
const ViewToggle = memo(({ viewMode, onClick }: { viewMode: AppViewMode; onClick: () => void }) => (
  <button
    className="control-btn view-toggle"
    onClick={onClick}
    title="Toggle View"
  >
    {viewMode === 'grid' ? '☰' : '⊞'}
  </button>
));
ViewToggle.displayName = 'ViewToggle';

// Virtual scrolling component for large lists
const VirtualAppList = memo(({ 
  apps, 
  viewMode, 
  onPreview, 
  onDetails,
  onAppAction,
}: {
  apps: App[];
  viewMode: AppViewMode;
  onPreview: (app: App) => void;
  onDetails: (app: App) => void;
  onAppAction: (appId: string, action: 'start' | 'stop') => void;
}) => {
  // For lists with many items, implement virtual scrolling
  const ITEMS_PER_PAGE = 50;
  const [visibleRange, setVisibleRange] = useState({ start: 0, end: ITEMS_PER_PAGE });
  
  useEffect(() => {
    const handleScroll = () => {
      const scrollTop = window.scrollY;
      const itemHeight = viewMode === 'grid' ? 200 : 100; // Approximate heights
      const start = Math.floor(scrollTop / itemHeight);
      const end = start + Math.ceil(window.innerHeight / itemHeight) + 10; // Buffer
      
      setVisibleRange({ start, end });
    };
    
    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [viewMode]);
  
  // Only render visible items for performance
  const visibleApps = apps.slice(visibleRange.start, visibleRange.end);
  
  return (
    <div className={clsx('apps-grid', viewMode)}>
      {/* Spacer for virtual scrolling */}
      {visibleRange.start > 0 && (
        <div style={{ height: `${visibleRange.start * (viewMode === 'grid' ? 200 : 100)}px` }} />
      )}
      
      {visibleApps.map(app => (
        <AppCard
          key={app.id}
          app={app}
          viewMode={viewMode}
          onCardClick={onPreview}
          onDetails={onDetails}
          onStart={(id) => onAppAction(id, 'start')}
          onStop={(id) => onAppAction(id, 'stop')}
        />
      ))}
      
      {/* Spacer for virtual scrolling */}
      {visibleRange.end < apps.length && (
        <div style={{ height: `${(apps.length - visibleRange.end) * (viewMode === 'grid' ? 200 : 100)}px` }} />
      )}
    </div>
  );
});
VirtualAppList.displayName = 'VirtualAppList';

// Main AppsView component with optimizations
const AppsView = memo<AppsViewProps>(({ apps, setApps }) => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<AppViewMode>('grid');
  const [selectedApp, setSelectedApp] = useState<App | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [infoMessage, setInfoMessage] = useState<string | null>(null);

  const fetchApps = useCallback(async () => {
    setLoading(true);
    logger.time('fetchApps');

    try {
      const fetchedApps = await appService.getApps();
      setApps(fetchedApps);
      logger.info(`Fetched ${fetchedApps.length} apps`);
    } catch (error) {
      logger.error('Failed to fetch apps', error);
    } finally {
      setLoading(false);
      logger.timeEnd('fetchApps');
    }
  }, [setApps]);

  useEffect(() => {
    fetchApps();
  }, [fetchApps]);

  const filteredApps = useMemo(() => {
    const statusPriority: Record<string, number> = {
      running: 0,
      healthy: 0,
      degraded: 1,
      unhealthy: 1,
      error: 2,
      unknown: 3,
      stopped: 4,
    };

    const matchesSearch = (app: App) => {
      if (!search) return true;
      const searchLower = search.toLowerCase();
      return (
        app.name.toLowerCase().includes(searchLower) ||
        app.status.toLowerCase().includes(searchLower)
      );
    };

    return apps
      .filter(matchesSearch)
      .slice()
      .sort((a, b) => {
        const priorityA = statusPriority[a.status] ?? 5;
        const priorityB = statusPriority[b.status] ?? 5;
        if (priorityA !== priorityB) {
          return priorityA - priorityB;
        }
        return a.name.localeCompare(b.name);
      });
  }, [apps, search]);

  const handleAppAction = useCallback(async (appId: string, action: 'start' | 'stop' | 'restart') => {
    logger.info(`App action: ${action} for ${appId}`);

    try {
      const success = await appService.controlApp(appId, action);
      if (success) {
        setApps(prev => prev.map(app => {
          if (app.id === appId) {
            return {
              ...app,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString()
            };
          }
          return app;
        }));
        logger.info(`App ${appId} ${action} successful`);
      }
    } catch (error) {
      logger.error(`Failed to ${action} app ${appId}`, error);
    }
  }, [setApps]);

  const handleAppDetails = useCallback((app: App) => {
    setSelectedApp(app);
    setModalOpen(true);
  }, []);

  const handleAppPreview = useCallback((app: App) => {
    if (!isRunningStatus(app.status) || isStoppedStatus(app.status)) {
      setInfoMessage(null);
      setSelectedApp(app);
      setModalOpen(true);
      return;
    }

    const url = buildPreviewUrl(app);
    if (!url) {
      setInfoMessage('This application does not expose a UI port. Opening details instead.');
      setSelectedApp(app);
      setModalOpen(true);
      return;
    }

    setInfoMessage(null);
    navigate(`/apps/${encodeURIComponent(app.id)}/preview`);
  }, [navigate]);

  const handleViewLogs = useCallback((appId: string) => {
    setModalOpen(false);
    setSelectedApp(null);
    navigate(`/logs/${appId}`);
  }, [navigate]);

  const toggleViewMode = useCallback(() => {
    setViewMode(prev => prev === 'grid' ? 'list' : 'grid');
  }, []);

  const handleSearchChange = useCallback((value: string) => {
    setSearch(value);
  }, []);

  const dismissInfoMessage = useCallback(() => {
    setInfoMessage(null);
  }, []);

  const useVirtualScrolling = filteredApps.length > 100;

  return (
    <div className="apps-view">
      <div className="panel-header">
        <h2>ACTIVE APPLICATIONS</h2>
        <div className="panel-controls">
          <SearchInput value={search} onChange={handleSearchChange} />
          <ViewToggle viewMode={viewMode} onClick={toggleViewMode} />
          <button
            className="control-btn refresh"
            onClick={fetchApps}
            disabled={loading}
            title="Refresh"
          >
            {loading ? '⟳' : '⟲'}
          </button>
        </div>
      </div>

      {loading && filteredApps.length === 0 ? (
        <AppsGridSkeleton count={6} viewMode={viewMode} />
      ) : filteredApps.length === 0 ? (
        <div className="empty-state">
          <p>NO APPLICATIONS FOUND</p>
          {search && <p className="hint">Try adjusting your search</p>}
        </div>
      ) : useVirtualScrolling ? (
        <VirtualAppList
          apps={filteredApps}
          viewMode={viewMode}
          onPreview={handleAppPreview}
          onDetails={handleAppDetails}
          onAppAction={handleAppAction}
        />
      ) : (
        <div className={clsx('apps-grid', viewMode)}>
          {filteredApps.map(app => (
            <AppCard
              key={app.id}
              app={app}
              viewMode={viewMode}
              onCardClick={handleAppPreview}
              onDetails={handleAppDetails}
              onStart={(id) => handleAppAction(id, 'start')}
              onStop={(id) => handleAppAction(id, 'stop')}
            />
          ))}
        </div>
      )}

      {infoMessage && (
        <div className="apps-view-message">
          <span>{infoMessage}</span>
          <button className="apps-view-message__dismiss" onClick={dismissInfoMessage}>
            DISMISS
          </button>
        </div>
      )}

      {modalOpen && selectedApp && (
        <AppModal
          app={selectedApp}
          isOpen={modalOpen}
          onClose={() => {
            setModalOpen(false);
            setSelectedApp(null);
          }}
          onAction={handleAppAction}
          onViewLogs={handleViewLogs}
        />
      )}
    </div>
  );
});

AppsView.displayName = 'AppsView';

export default AppsView;
