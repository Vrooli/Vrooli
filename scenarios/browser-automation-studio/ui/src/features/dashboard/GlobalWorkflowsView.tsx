import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Search, X, ChevronLeft, FolderOpen, Play, Star, Clock, Filter } from 'lucide-react';
import { getConfig } from '../../config';
import { logger } from '../../utils/logger';
import { useDashboardStore, FavoriteWorkflow } from '../../stores/dashboardStore';
import toast from 'react-hot-toast';

interface WorkflowItem {
  id: string;
  name: string;
  projectId: string;
  projectName: string;
  folderPath: string;
  updatedAt: Date;
  executionCount?: number;
  lastExecution?: Date;
}

interface ProjectGroup {
  projectId: string;
  projectName: string;
  workflows: WorkflowItem[];
}

interface GlobalWorkflowsViewProps {
  onBack: () => void;
  onNavigateToWorkflow: (projectId: string, workflowId: string) => void;
  onRunWorkflow?: (workflowId: string) => void;
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

type SortOption = 'recent' | 'name' | 'project';
type GroupOption = 'none' | 'project';

export const GlobalWorkflowsView: React.FC<GlobalWorkflowsViewProps> = ({
  onBack,
  onNavigateToWorkflow,
  onRunWorkflow,
}) => {
  const [workflows, setWorkflows] = useState<WorkflowItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [sortBy, setSortBy] = useState<SortOption>('recent');
  const [groupBy, setGroupBy] = useState<GroupOption>('project');
  const [showFilters, setShowFilters] = useState(false);

  const favoriteWorkflows = useDashboardStore((state) => state.favoriteWorkflows);
  const addFavorite = useDashboardStore((state) => state.addFavorite);
  const removeFavorite = useDashboardStore((state) => state.removeFavorite);
  const isFavorite = useDashboardStore((state) => state.isFavorite);

  const fetchAllWorkflows = useCallback(async () => {
    setIsLoading(true);
    try {
      const config = await getConfig();

      // Fetch projects first
      const projectsResponse = await fetch(`${config.API_URL}/projects`);
      const projectsData = await projectsResponse.json();
      const projectsMap = new Map<string, string>();
      if (Array.isArray(projectsData.projects)) {
        projectsData.projects.forEach((p: { id: string; name: string }) => {
          projectsMap.set(p.id, p.name);
        });
      }

      // Fetch all workflows
      const response = await fetch(`${config.API_URL}/workflows?limit=500`);
      if (!response.ok) {
        throw new Error(`Failed to fetch workflows: ${response.status}`);
      }
      const data = await response.json();

      const workflowItems: WorkflowItem[] = Array.isArray(data.workflows)
        ? data.workflows.map((w: Record<string, unknown>) => {
            const projectId = String(w.project_id ?? w.projectId ?? '');
            return {
              id: String(w.id ?? ''),
              name: String(w.name ?? 'Untitled'),
              projectId,
              projectName: projectsMap.get(projectId) ?? 'Unknown Project',
              folderPath: String(w.folder_path ?? w.folderPath ?? '/'),
              updatedAt: new Date(String(w.updated_at ?? w.updatedAt ?? new Date().toISOString())),
              executionCount: typeof w.execution_count === 'number' ? w.execution_count : undefined,
            };
          })
        : [];

      setWorkflows(workflowItems);
    } catch (error) {
      logger.error('Failed to fetch all workflows', { component: 'GlobalWorkflowsView', action: 'fetchAllWorkflows' }, error);
      toast.error('Failed to load workflows');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchAllWorkflows();
  }, [fetchAllWorkflows]);

  const handleToggleFavorite = (e: React.MouseEvent, workflow: WorkflowItem) => {
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

  const handleRunWorkflow = (e: React.MouseEvent, workflowId: string) => {
    e.stopPropagation();
    if (onRunWorkflow) {
      onRunWorkflow(workflowId);
    }
  };

  // Filter and sort workflows
  const filteredWorkflows = useMemo(() => {
    let result = workflows;

    // Filter by search term
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      result = result.filter(
        (w) =>
          w.name.toLowerCase().includes(term) ||
          w.projectName.toLowerCase().includes(term) ||
          w.folderPath.toLowerCase().includes(term)
      );
    }

    // Sort
    switch (sortBy) {
      case 'recent':
        result = [...result].sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime());
        break;
      case 'name':
        result = [...result].sort((a, b) => a.name.localeCompare(b.name));
        break;
      case 'project':
        result = [...result].sort((a, b) => a.projectName.localeCompare(b.projectName));
        break;
    }

    return result;
  }, [workflows, searchTerm, sortBy]);

  // Group workflows by project if needed
  const groupedWorkflows = useMemo((): ProjectGroup[] | null => {
    if (groupBy === 'none') return null;

    const groups = new Map<string, ProjectGroup>();

    // Add favorites first as a special group
    const favoriteIds = new Set(favoriteWorkflows.map(f => f.id));
    const favoriteItems = filteredWorkflows.filter(w => favoriteIds.has(w.id));

    if (favoriteItems.length > 0) {
      groups.set('__favorites__', {
        projectId: '__favorites__',
        projectName: 'Favorites',
        workflows: favoriteItems,
      });
    }

    // Group remaining by project
    filteredWorkflows
      .filter(w => !favoriteIds.has(w.id))
      .forEach((workflow) => {
        const existing = groups.get(workflow.projectId);
        if (existing) {
          existing.workflows.push(workflow);
        } else {
          groups.set(workflow.projectId, {
            projectId: workflow.projectId,
            projectName: workflow.projectName,
            workflows: [workflow],
          });
        }
      });

    return Array.from(groups.values());
  }, [filteredWorkflows, groupBy, favoriteWorkflows]);

  const renderWorkflowItem = (workflow: WorkflowItem, showProject: boolean = false) => (
    <div
      key={workflow.id}
      onClick={() => onNavigateToWorkflow(workflow.projectId, workflow.id)}
      className="group flex items-center gap-3 p-3 rounded-lg hover:bg-gray-700/50 cursor-pointer transition-colors border border-transparent hover:border-gray-600"
    >
      <div className="flex items-center justify-center w-10 h-10 bg-gray-700/50 rounded-lg text-gray-400 group-hover:bg-gray-600/50 transition-colors">
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
      </div>
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-white truncate">{workflow.name}</div>
        <div className="text-xs text-gray-500 truncate">
          {showProject && <span>{workflow.projectName} &middot; </span>}
          {workflow.folderPath !== '/' && <span>{workflow.folderPath} &middot; </span>}
          <Clock className="inline w-3 h-3 mr-1" />
          {formatRelativeTime(workflow.updatedAt)}
        </div>
      </div>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={(e) => handleToggleFavorite(e, workflow)}
          className={`p-1.5 rounded-md transition-colors ${
            isFavorite(workflow.id)
              ? 'text-yellow-400 hover:text-yellow-300'
              : 'text-gray-500 hover:text-yellow-400'
          }`}
          title={isFavorite(workflow.id) ? 'Remove from favorites' : 'Add to favorites'}
        >
          <Star className="w-4 h-4" fill={isFavorite(workflow.id) ? 'currentColor' : 'none'} />
        </button>
        {onRunWorkflow && (
          <button
            onClick={(e) => handleRunWorkflow(e, workflow.id)}
            className="p-1.5 text-gray-500 hover:text-green-400 rounded-md transition-colors"
            title="Run workflow"
          >
            <Play className="w-4 h-4" />
          </button>
        )}
      </div>
    </div>
  );

  return (
    <div className="flex-1 flex flex-col min-h-[100svh] overflow-hidden bg-flow-bg">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center gap-4">
            <button
              onClick={onBack}
              className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
              title="Back to Dashboard"
            >
              <ChevronLeft size={20} />
            </button>
            <div>
              <h1 className="text-xl font-bold text-white">All Workflows</h1>
              <p className="text-sm text-gray-400">
                {isLoading ? 'Loading...' : `${filteredWorkflows.length} workflow${filteredWorkflows.length !== 1 ? 's' : ''}`}
              </p>
            </div>
          </div>

          {/* Search and Filters */}
          <div className="mt-4 flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
              <input
                type="text"
                placeholder="Search workflows..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/50"
              />
              {searchTerm && (
                <button
                  onClick={() => setSearchTerm('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
                >
                  <X size={16} />
                </button>
              )}
            </div>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`flex items-center gap-2 px-3 py-2 rounded-lg border transition-colors ${
                showFilters
                  ? 'bg-flow-accent/20 border-flow-accent text-flow-accent'
                  : 'bg-gray-800/50 border-gray-700 text-gray-400 hover:text-white'
              }`}
            >
              <Filter size={16} />
              <span className="text-sm">Filters</span>
            </button>
          </div>

          {/* Filter Options */}
          {showFilters && (
            <div className="mt-3 flex flex-wrap gap-4 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
              <div>
                <label className="block text-xs text-gray-400 mb-1">Sort by</label>
                <select
                  value={sortBy}
                  onChange={(e) => setSortBy(e.target.value as SortOption)}
                  className="px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-white focus:outline-none focus:border-flow-accent"
                >
                  <option value="recent">Most Recent</option>
                  <option value="name">Name</option>
                  <option value="project">Project</option>
                </select>
              </div>
              <div>
                <label className="block text-xs text-gray-400 mb-1">Group by</label>
                <select
                  value={groupBy}
                  onChange={(e) => setGroupBy(e.target.value as GroupOption)}
                  className="px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-white focus:outline-none focus:border-flow-accent"
                >
                  <option value="project">Project</option>
                  <option value="none">None</option>
                </select>
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-auto px-4 py-6 sm:px-6">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="animate-pulse flex items-center gap-3 p-3 bg-gray-800/30 rounded-lg">
                <div className="w-10 h-10 bg-gray-700 rounded-lg" />
                <div className="flex-1">
                  <div className="h-4 bg-gray-700 rounded w-1/3 mb-2" />
                  <div className="h-3 bg-gray-700/50 rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : filteredWorkflows.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
              <Search size={28} className="text-gray-500" />
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              {searchTerm ? 'No workflows found' : 'No workflows yet'}
            </h3>
            <p className="text-gray-400">
              {searchTerm
                ? `No workflows match "${searchTerm}"`
                : 'Create your first workflow to get started'}
            </p>
            {searchTerm && (
              <button
                onClick={() => setSearchTerm('')}
                className="mt-4 text-flow-accent hover:text-blue-400 text-sm"
              >
                Clear search
              </button>
            )}
          </div>
        ) : groupBy === 'none' ? (
          <div className="space-y-1">
            {filteredWorkflows.map((workflow) => renderWorkflowItem(workflow, true))}
          </div>
        ) : (
          <div className="space-y-6">
            {groupedWorkflows?.map((group) => (
              <div key={group.projectId}>
                <div className="flex items-center gap-2 mb-2 sticky top-0 bg-flow-bg py-2">
                  {group.projectId === '__favorites__' ? (
                    <Star size={16} className="text-yellow-400" fill="currentColor" />
                  ) : (
                    <FolderOpen size={16} className="text-flow-accent" />
                  )}
                  <h2 className="text-sm font-medium text-gray-300">{group.projectName}</h2>
                  <span className="text-xs text-gray-500">({group.workflows.length})</span>
                </div>
                <div className="space-y-1 ml-6">
                  {group.workflows.map((workflow) => renderWorkflowItem(workflow, false))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
