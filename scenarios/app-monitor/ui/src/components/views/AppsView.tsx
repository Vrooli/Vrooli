import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import clsx from 'clsx';
import { appService } from '@/services/api';
import type { App, AppViewMode } from '@/types';
import AppCard from '../AppCard';
import AppModal from '../AppModal';
import './AppsView.css';

interface AppsViewProps {
  apps: App[];
  setApps: React.Dispatch<React.SetStateAction<App[]>>;
}

export default function AppsView({ apps, setApps }: AppsViewProps) {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [viewMode, setViewMode] = useState<AppViewMode>('grid');
  const [selectedApp, setSelectedApp] = useState<App | null>(null);
  const [modalOpen, setModalOpen] = useState(false);

  // Fetch apps on mount
  useEffect(() => {
    fetchApps();
  }, []);

  const fetchApps = async () => {
    setLoading(true);
    try {
      const fetchedApps = await appService.getApps();
      setApps(fetchedApps);
    } catch (error) {
      console.error('Failed to fetch apps:', error);
    } finally {
      setLoading(false);
    }
  };

  // Filter apps based on search
  const filteredApps = apps.filter(app =>
    app.name.toLowerCase().includes(search.toLowerCase())
  );

  // Handle app control actions
  const handleAppAction = async (appId: string, action: 'start' | 'stop' | 'restart') => {
    try {
      const success = await appService.controlApp(appId, action);
      if (success) {
        // Update local state
        setApps(prev => prev.map(app => {
          if (app.id === appId) {
            return {
              ...app,
              status: action === 'stop' ? 'stopped' : 'running'
            };
          }
          return app;
        }));
      }
    } catch (error) {
      console.error(`Failed to ${action} app ${appId}:`, error);
    }
  };

  // Handle app click
  const handleAppClick = useCallback((app: App) => {
    setSelectedApp(app);
    setModalOpen(true);
  }, []);

  // Handle view logs
  const handleViewLogs = (appId: string) => {
    setModalOpen(false);
    navigate(`/logs/${appId}`);
  };

  // Toggle view mode
  const toggleViewMode = () => {
    setViewMode(prev => prev === 'grid' ? 'list' : 'grid');
  };

  return (
    <div className="apps-view">
      <div className="panel-header">
        <h2>ACTIVE APPLICATIONS</h2>
        <div className="panel-controls">
          <input
            type="text"
            className="search-box"
            placeholder="SEARCH..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <button
            className="control-btn view-toggle"
            onClick={toggleViewMode}
            title="Toggle View"
          >
            {viewMode === 'grid' ? '☰' : '⊞'}
          </button>
          <button className="control-btn" onClick={fetchApps} disabled={loading}>
            ⟳ REFRESH
          </button>
        </div>
      </div>

      {loading ? (
        <div className="loading-container">
          <div className="spinner"></div>
          <p>Loading applications...</p>
        </div>
      ) : filteredApps.length === 0 ? (
        <div className="empty-state">
          <p>No applications found</p>
        </div>
      ) : viewMode === 'grid' ? (
        <div className="apps-grid">
          {filteredApps.map(app => (
            <AppCard
              key={app.id}
              app={app}
              onClick={() => handleAppClick(app)}
              onAction={handleAppAction}
            />
          ))}
        </div>
      ) : (
        <div className="apps-list">
          <table className="apps-table">
            <thead>
              <tr className="table-header">
                <th>NAME</th>
                <th>STATUS</th>
                <th>PORT</th>
                <th>ACTIONS</th>
              </tr>
            </thead>
            <tbody>
              {filteredApps.map(app => (
                <tr key={app.id} className="app-row" onClick={() => handleAppClick(app)}>
                  <td className="app-name-cell">{app.name}</td>
                  <td className="status-cell">
                    <span className={clsx('app-status', app.status)}>
                      {app.status.toUpperCase()}
                    </span>
                  </td>
                  <td className="port-cell">
                    {app.port_mappings && Object.keys(app.port_mappings).length > 0 ? (
                      Object.entries(app.port_mappings).map(([name, port]) => (
                        <span key={name} className="port-tag" title={name}>
                          {port}
                        </span>
                      ))
                    ) : '-'}
                  </td>
                  <td className="actions-cell" onClick={(e) => e.stopPropagation()}>
                    <button
                      className="app-btn small"
                      onClick={() => handleAppAction(app.id, 'start')}
                      disabled={app.status === 'running'}
                      title="Start"
                    >
                      ▶
                    </button>
                    <button
                      className="app-btn small"
                      onClick={() => handleAppAction(app.id, 'stop')}
                      disabled={app.status === 'stopped'}
                      title="Stop"
                    >
                      ■
                    </button>
                    <button
                      className="app-btn small"
                      onClick={() => handleAppAction(app.id, 'restart')}
                      title="Restart"
                    >
                      ⟲
                    </button>
                    <button
                      className="app-btn small"
                      onClick={() => navigate(`/logs/${app.id}`)}
                      title="Logs"
                    >
                      📜
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {modalOpen && selectedApp && (
        <AppModal
          app={selectedApp}
          isOpen={modalOpen}
          onClose={() => setModalOpen(false)}
          onAction={handleAppAction}
          onViewLogs={handleViewLogs}
        />
      )}
    </div>
  );
}