import { useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Play, Square, Info, Loader2, RefreshCw } from 'lucide-react';
import { ResourcesGridSkeleton } from '../LoadingSkeleton';
import './ResourcesView.css';
import { useResourcesStore } from '@/state/resourcesStore';
import { useResourcesCatalog } from '@/hooks/useResourcesCatalog';

export default function ResourcesView() {
  const {
    resources,
    sortedResources,
    loading,
    error: storeError,
    loadResources,
    startResource,
    stopResource,
    refreshResource,
  } = useResourcesCatalog();
  const clearError = useResourcesStore(state => state.clearError);
  const hasInitialized = useResourcesStore(state => state.hasInitialized);

  type ResourceAction = 'start' | 'stop' | 'refresh';

  const navigate = useNavigate();

  const [pendingActions, setPendingActions] = useState<Record<string, ResourceAction | null>>({});
  const [actionFeedback, setActionFeedback] = useState<Record<string, { type: 'success' | 'error'; message: string }>>({});
  const [globalFeedback, setGlobalFeedback] = useState<{ type: 'success' | 'error'; message: string } | null>(null);
  const emptyReloadRef = useRef(false);

  useEffect(() => {
    void loadResources();
  }, [loadResources]);

  useEffect(() => {
    if (hasInitialized && !loading && resources.length === 0) {
      if (!emptyReloadRef.current) {
        emptyReloadRef.current = true;
        void loadResources({ force: true });
      }
    } else if (resources.length > 0) {
      emptyReloadRef.current = false;
    }
  }, [hasInitialized, loadResources, loading, resources.length]);

  useEffect(() => {
    if (storeError) {
      setGlobalFeedback({ type: 'error', message: storeError });
    } else {
      setGlobalFeedback(null);
    }
  }, [storeError]);

  useEffect(() => () => {
    clearError();
  }, [clearError]);

  const getPendingAction = (id: string): ResourceAction | null => pendingActions[id] ?? null;

  const isActionDisabled = (id: string, action: ResourceAction, status: string): boolean => {
    const pending = getPendingAction(id);
    if (pending !== null) {
      return true;
    }

    if (action === 'start') {
      return ['online'].includes(status);
    }
    if (action === 'stop') {
      return ['offline', 'stopped', 'unknown', 'unregistered'].includes(status);
    }
    return false;
  };

  const handleResourceAction = async (id: string, action: ResourceAction) => {
    setPendingActions(prev => ({ ...prev, [id]: action }));
    setActionFeedback(prev => {
      if (!(id in prev)) {
        return prev;
      }
      const next = { ...prev };
      delete next[id];
      return next;
    });
    setGlobalFeedback(null);

    try {
      if (action === 'start') {
        const response = await startResource(id);
        if (response?.success) {
          if (response.warning) {
            const warningMessage = response.warning ?? 'Command succeeded but reported a warning.';
            setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message: warningMessage } }));
            setGlobalFeedback({ type: 'error', message: warningMessage });
          } else {
            setActionFeedback(prev => ({ ...prev, [id]: { type: 'success', message: 'Resource start requested.' } }));
          }
        } else if (response?.error) {
          const message = response.error ?? 'Failed to start resource.';
          setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message } }));
          setGlobalFeedback({ type: 'error', message });
        }
      } else if (action === 'stop') {
        const response = await stopResource(id);
        if (response?.success) {
          if (response.warning) {
            const warningMessage = response.warning ?? 'Command succeeded but reported a warning.';
            setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message: warningMessage } }));
            setGlobalFeedback({ type: 'error', message: warningMessage });
          } else {
            setActionFeedback(prev => ({ ...prev, [id]: { type: 'success', message: 'Resource stop requested.' } }));
          }
        } else if (response?.error) {
          const message = response.error ?? 'Failed to stop resource.';
          setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message } }));
          setGlobalFeedback({ type: 'error', message });
        }
      } else {
        const refreshed = await refreshResource(id);
        if (refreshed) {
          setActionFeedback(prev => ({ ...prev, [id]: { type: 'success', message: 'Status refreshed.' } }));
        } else {
          setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message: 'Failed to refresh status.' } }));
          setGlobalFeedback({ type: 'error', message: 'Failed to refresh status.' });
        }
      }
    } catch (error) {
      console.error('Resource action failed', error);
      setActionFeedback(prev => ({ ...prev, [id]: { type: 'error', message: 'Unexpected failure.' } }));
      setGlobalFeedback({ type: 'error', message: 'Unexpected failure.' });
    } finally {
      setPendingActions(prev => {
        if (!(id in prev)) {
          return prev;
        }
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  };

  const getResourceIcon = (type: string): string => {
    const icons: Record<string, string> = {
      database: '🗄️',
      cache: '💾',
      queue: '📬',
      storage: '💿',
      api: '🔌',
      service: '⚙️',
      default: '📦',
    };
    return icons[type.toLowerCase()] || icons.default;
  };

  return (
    <div className="resources-view">
      <div className="panel-header">
        <div className="panel-title-spacer" aria-hidden="true" />
        <div className="panel-controls">
          <button className="control-btn" onClick={() => { void loadResources({ force: true }); }} disabled={loading}>
            ⟳ REFRESH
          </button>
        </div>
      </div>

      {globalFeedback && (
        <div className={`resource-feedback ${globalFeedback.type}`}>
          {globalFeedback.message}
        </div>
      )}
      
      {loading ? (
        <ResourcesGridSkeleton count={4} />
      ) : resources.length === 0 ? (
        <div className="empty-state">
          <p>No resources configured</p>
        </div>
      ) : (
        <div className="resources-grid">
          {sortedResources.map(resource => (
            <div key={resource.id} className="resource-card">
              <div className="resource-icon">
                {resource.icon || getResourceIcon(resource.type)}
              </div>
              <div className="resource-name">{resource.name}</div>
              <div className="resource-type">{resource.type.toUpperCase()}</div>
              <div className={`resource-status ${resource.status}`}>
                {resource.status.toUpperCase()}
              </div>
              {resource.description && (
                <div className="resource-description">
                  {resource.description}
                </div>
              )}
              <div className="resource-actions">
                <button
                  className="resource-action-btn start"
                  disabled={isActionDisabled(resource.id, 'start', resource.status)}
                  data-pending={getPendingAction(resource.id) === 'start'}
                  aria-label={`Start ${resource.name}`}
                  title="Start resource"
                  onClick={() => { void handleResourceAction(resource.id, 'start'); }}
                >
                  {getPendingAction(resource.id) === 'start'
                    ? <Loader2 className="resource-action-icon spinning" strokeWidth={2} />
                    : <Play className="resource-action-icon" strokeWidth={2} />}
                </button>
                <button
                  className="resource-action-btn stop"
                  disabled={isActionDisabled(resource.id, 'stop', resource.status)}
                  data-pending={getPendingAction(resource.id) === 'stop'}
                  aria-label={`Stop ${resource.name}`}
                  title="Stop resource"
                  onClick={() => { void handleResourceAction(resource.id, 'stop'); }}
                >
                  {getPendingAction(resource.id) === 'stop'
                    ? <Loader2 className="resource-action-icon spinning" strokeWidth={2} />
                    : <Square className="resource-action-icon" strokeWidth={2} />}
                </button>
                <button
                  className="resource-action-btn refresh"
                  disabled={getPendingAction(resource.id) === 'refresh'}
                  data-pending={getPendingAction(resource.id) === 'refresh'}
                  aria-label={`Refresh ${resource.name} status`}
                  title="Refresh status"
                  onClick={() => { void handleResourceAction(resource.id, 'refresh'); }}
                >
                  {getPendingAction(resource.id) === 'refresh'
                    ? <Loader2 className="resource-action-icon spinning" strokeWidth={2} />
                    : <RefreshCw className="resource-action-icon" strokeWidth={2} />}
                </button>
                <button
                  className="resource-action-btn detail"
                  aria-label={`View ${resource.name} details`}
                  title="View details"
                  onClick={() => navigate(`/resources/${encodeURIComponent(resource.id)}`)}
                >
                  <Info className="resource-action-icon" strokeWidth={2} />
                </button>
              </div>
              {actionFeedback[resource.id] && (
                <div className={`resource-feedback ${actionFeedback[resource.id]?.type}`}>
                  {actionFeedback[resource.id]?.message}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
