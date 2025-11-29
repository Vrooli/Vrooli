import React from 'react';
import { useDashboardStore } from '../../stores/dashboardStore';

interface ContinueEditingBannerProps {
  onNavigate: (projectId: string, workflowId: string) => void;
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

export const ContinueEditingBanner: React.FC<ContinueEditingBannerProps> = ({ onNavigate }) => {
  const lastEditedWorkflow = useDashboardStore((state) => state.lastEditedWorkflow);
  const clearLastEdited = useDashboardStore((state) => state.clearLastEdited);

  if (!lastEditedWorkflow) {
    return null;
  }

  const handleClick = () => {
    onNavigate(lastEditedWorkflow.projectId, lastEditedWorkflow.id);
  };

  const handleDismiss = (e: React.MouseEvent) => {
    e.stopPropagation();
    clearLastEdited();
  };

  return (
    <div
      onClick={handleClick}
      className="group relative flex items-center justify-between p-4 bg-gradient-to-r from-blue-600/20 to-purple-600/20 border border-blue-500/30 rounded-lg cursor-pointer hover:from-blue-600/30 hover:to-purple-600/30 hover:border-blue-500/50 transition-all duration-200"
    >
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-10 h-10 bg-blue-500/20 rounded-lg">
          <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
        </div>
        <div>
          <div className="text-sm text-gray-400">Continue editing</div>
          <div className="text-white font-medium">{lastEditedWorkflow.name}</div>
          <div className="text-xs text-gray-500">
            in {lastEditedWorkflow.projectName} &middot; {formatRelativeTime(new Date(lastEditedWorkflow.updatedAt))}
          </div>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <button
          onClick={handleDismiss}
          className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700/50 rounded transition-colors opacity-0 group-hover:opacity-100"
          title="Dismiss"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
        <svg className="w-5 h-5 text-gray-400 group-hover:text-white transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </div>
  );
};
