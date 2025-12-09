import React, { useEffect } from 'react';
import { useDashboardStore, RecentWorkflow, FavoriteWorkflow } from '../../stores/dashboardStore';

interface RecentWorkflowsWidgetProps {
  onNavigate: (projectId: string, workflowId: string) => void;
  onViewAll: () => void;
}

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
};

export const RecentWorkflowsWidget: React.FC<RecentWorkflowsWidgetProps> = ({ onNavigate, onViewAll }) => {
  const recentWorkflows = useDashboardStore((state) => state.recentWorkflows);
  const favoriteWorkflows = useDashboardStore((state) => state.favoriteWorkflows);
  const isLoading = useDashboardStore((state) => state.isLoadingRecent);
  const fetchRecentWorkflows = useDashboardStore((state) => state.fetchRecentWorkflows);
  const addFavorite = useDashboardStore((state) => state.addFavorite);
  const removeFavorite = useDashboardStore((state) => state.removeFavorite);
  const isFavorite = useDashboardStore((state) => state.isFavorite);

  useEffect(() => {
    fetchRecentWorkflows();
  }, [fetchRecentWorkflows]);

  const handleToggleFavorite = (e: React.MouseEvent, workflow: RecentWorkflow) => {
    e.stopPropagation();
    if (isFavorite(workflow.id)) {
      removeFavorite(workflow.id);
    } else {
      const favorite: FavoriteWorkflow = {
        id: workflow.id,
        name: workflow.name,
        projectId: workflow.projectId,
        projectName: workflow.projectName,
        addedAt: new Date(),
      };
      addFavorite(favorite);
    }
  };

  // Combine favorites and recent, with favorites first
  const favoriteIds = new Set(favoriteWorkflows.map(f => f.id));
  const recentNotFavorite = recentWorkflows.filter(w => !favoriteIds.has(w.id));
  const displayWorkflows = [
    ...favoriteWorkflows.map(f => ({
      ...f,
      updatedAt: f.addedAt,
      folderPath: '/',
      isFavorite: true,
    })),
    ...recentNotFavorite.slice(0, 5 - favoriteWorkflows.length).map(w => ({
      ...w,
      isFavorite: false,
    })),
  ].slice(0, 5);

  if (isLoading) {
    return (
      <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-300">Recent Workflows</h3>
        </div>
        <div className="space-y-2">
          {[1, 2, 3].map((i) => (
            <div key={i} className="animate-pulse flex items-center gap-3 p-2">
              <div className="w-8 h-8 bg-gray-700 rounded" />
              <div className="flex-1">
                <div className="h-4 bg-gray-700 rounded w-3/4 mb-1" />
                <div className="h-3 bg-gray-700/50 rounded w-1/2" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (displayWorkflows.length === 0) {
    return (
      <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-gray-300">Recent Workflows</h3>
        </div>
        <div className="text-center py-6 text-gray-500 text-sm">
          <svg className="w-8 h-8 mx-auto mb-2 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
          No workflows yet
        </div>
      </div>
    );
  }

  return (
    <div className="bg-gray-800/50 border border-gray-700/50 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-300">Recent Workflows</h3>
        <button
          onClick={onViewAll}
          className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
        >
          View all
        </button>
      </div>
      <div className="space-y-1">
        {displayWorkflows.map((workflow) => (
          <div
            key={workflow.id}
            onClick={() => onNavigate(workflow.projectId, workflow.id)}
            className="group flex items-center gap-3 p-2 rounded-md hover:bg-gray-700/50 cursor-pointer transition-colors"
          >
            <div className="flex items-center justify-center w-8 h-8 bg-gray-700/50 rounded text-gray-400 group-hover:bg-gray-600/50 transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <div className="flex-1 min-w-0">
              <div className="text-sm text-surface truncate">{workflow.name}</div>
              <div className="text-xs text-gray-500 truncate">
                {workflow.projectName} &middot; {formatRelativeTime(new Date(workflow.updatedAt))}
              </div>
            </div>
            <button
              onClick={(e) => handleToggleFavorite(e, workflow as RecentWorkflow)}
              className={`p-1 rounded transition-colors ${
                isFavorite(workflow.id)
                  ? 'text-yellow-400 hover:text-yellow-300'
                  : 'text-gray-500 hover:text-yellow-400 opacity-0 group-hover:opacity-100'
              }`}
              title={isFavorite(workflow.id) ? 'Remove from favorites' : 'Add to favorites'}
            >
              <svg className="w-4 h-4" fill={isFavorite(workflow.id) ? 'currentColor' : 'none'} stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};
