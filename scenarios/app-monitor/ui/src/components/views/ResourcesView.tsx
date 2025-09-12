import { useState, useEffect } from 'react';
import { resourceService } from '@/services/api';
import type { Resource } from '@/types';
import { ResourcesGridSkeleton } from '../LoadingSkeleton';
import './ResourcesView.css';

export default function ResourcesView() {
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchResources();
  }, []);

  const fetchResources = async () => {
    setLoading(true);
    try {
      const data = await resourceService.getResources();
      setResources(data);
    } catch (error) {
      console.error('Failed to fetch resources:', error);
    } finally {
      setLoading(false);
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
        <h2>RESOURCE STATUS</h2>
        <div className="panel-controls">
          <button className="control-btn" onClick={fetchResources} disabled={loading}>
            ⟳ REFRESH
          </button>
        </div>
      </div>
      
      {loading ? (
        <ResourcesGridSkeleton count={4} />
      ) : resources.length === 0 ? (
        <div className="empty-state">
          <p>No resources configured</p>
        </div>
      ) : (
        <div className="resources-grid">
          {resources.map(resource => (
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
            </div>
          ))}
        </div>
      )}
    </div>
  );
}