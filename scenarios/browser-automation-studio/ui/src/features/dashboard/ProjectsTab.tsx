import React, { useState, useCallback, useEffect } from 'react';
import {
  Plus,
  FolderOpen,
  Play,
  Clock,
  Search,
  X,
  Trash2,
  MoreVertical,
  Loader,
  PlayCircle,
  Star,
  StarOff,
  Lightbulb,
} from 'lucide-react';
import { useProjectStore, type Project } from '@stores/projectStore';
import { useDashboardStore, type FavoriteWorkflow } from '@stores/dashboardStore';
import { formatDistanceToNow } from 'date-fns';
import toast from 'react-hot-toast';

interface ProjectsTabProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
  onNavigateToWorkflow: (projectId: string, workflowId: string) => void;
  onRunWorkflow: (workflowId: string) => void;
}

interface ProjectWorkflow {
  id: string;
  name: string;
  folder_path?: string;
  folderPath?: string;
  updated_at?: string;
  updatedAt?: string;
  project_id?: string;
  projectId?: string;
}

export const ProjectsTab: React.FC<ProjectsTabProps> = ({
  onProjectSelect,
  onCreateProject,
  onNavigateToWorkflow,
  onRunWorkflow,
}) => {
  const {
    projects,
    isLoading,
    bulkExecutionInProgress,
    deleteProject,
    executeAllWorkflows,
  } = useProjectStore();
  const { addFavorite, removeFavorite, isFavorite } = useDashboardStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [deletingProjectId, setDeletingProjectId] = useState<string | null>(null);
  const [showActionsFor, setShowActionsFor] = useState<string | null>(null);
  const [projectWorkflows, setProjectWorkflows] = useState<Record<string, ProjectWorkflow[]>>({});
  const [loadingWorkflows, setLoadingWorkflows] = useState<string | null>(null);

  // Determine if we're in single-project mode (0 or 1 project)
  const isSingleProjectMode = projects.length <= 1;
  const singleProject = projects[0];

  // Filter projects based on search
  const filteredProjects = projects.filter((project) => {
    const searchLower = searchTerm.toLowerCase();
    return (
      project.name?.toLowerCase().includes(searchLower) ||
      project.description?.toLowerCase().includes(searchLower)
    );
  });

  // Load workflows for expanded project
  const loadProjectWorkflows = useCallback(async (projectId: string) => {
    if (projectWorkflows[projectId]) return;

    setLoadingWorkflows(projectId);
    try {
      const response = await fetch(`/api/v1/projects/${projectId}/workflows`);
      if (response.ok) {
        const data = await response.json();
        const workflows = Array.isArray(data.workflows) ? data.workflows : [];
        setProjectWorkflows((prev) => ({ ...prev, [projectId]: workflows }));
      }
    } catch (error) {
      console.error('Failed to load project workflows', error);
    } finally {
      setLoadingWorkflows(null);
    }
  }, [projectWorkflows]);

  // Auto-load workflows in single project mode
  useEffect(() => {
    if (isSingleProjectMode && singleProject && !projectWorkflows[singleProject.id]) {
      void loadProjectWorkflows(singleProject.id);
    }
  }, [isSingleProjectMode, singleProject, projectWorkflows, loadProjectWorkflows]);

  const handleDeleteProject = useCallback(async (e: React.MouseEvent, projectId: string, projectName: string) => {
    e.stopPropagation();
    setShowActionsFor(null);

    const confirmed = window.confirm(
      `Delete "${projectName}" and all its workflows? This action cannot be undone.`
    );
    if (!confirmed) return;

    setDeletingProjectId(projectId);
    try {
      await deleteProject(projectId);
      toast.success(`Project "${projectName}" deleted`);
    } catch (error) {
      toast.error('Failed to delete project');
    } finally {
      setDeletingProjectId(null);
    }
  }, [deleteProject]);

  const handleRunAllWorkflows = async (e: React.MouseEvent, projectId: string) => {
    e.stopPropagation();
    setShowActionsFor(null);

    try {
      const result = await executeAllWorkflows(projectId);
      const successCount = result.executions.filter((exec) => exec.status !== 'failed').length;
      toast.success(`Started ${successCount} workflow(s)`);
    } catch (error) {
      toast.error('Failed to execute workflows');
    }
  };

  const handleToggleFavorite = useCallback((workflow: ProjectWorkflow, projectName: string) => {
    if (isFavorite(workflow.id)) {
      removeFavorite(workflow.id);
    } else {
      const favorite: FavoriteWorkflow = {
        id: workflow.id,
        name: workflow.name,
        projectId: workflow.project_id ?? workflow.projectId ?? '',
        projectName,
        addedAt: new Date(),
      };
      addFavorite(favorite);
    }
  }, [addFavorite, removeFavorite, isFavorite]);

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never';
    return formatDistanceToNow(new Date(dateString), { addSuffix: true });
  };

  // Render single-project view (shows workflows directly)
  const renderSingleProjectView = () => {
    const workflows = singleProject ? (projectWorkflows[singleProject.id] ?? []) : [];
    const isLoadingWfs = singleProject && loadingWorkflows === singleProject.id;

    return (
      <div className="space-y-6">
        {/* Project Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-gradient-to-br from-flow-accent/20 to-blue-500/10 rounded-xl flex items-center justify-center border border-flow-accent/20">
              <FolderOpen size={18} className="text-flow-accent" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">
                {singleProject?.name ?? 'My Automations'}
              </h2>
              <p className="text-sm text-gray-400">
                {workflows.length} workflow{workflows.length !== 1 ? 's' : ''}
              </p>
            </div>
          </div>
          <button
            onClick={onCreateProject}
            className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors"
          >
            <Plus size={14} />
            New Project
          </button>
        </div>

        {/* Search */}
        {workflows.length > 5 && (
          <div className="relative">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
            <input
              type="text"
              placeholder="Search workflows..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-10 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
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
        )}

        {/* Workflows List */}
        {isLoadingWfs ? (
          <div className="flex items-center justify-center py-12">
            <Loader size={24} className="animate-spin text-gray-500" />
          </div>
        ) : workflows.length === 0 ? (
          <div className="text-center py-12 border-2 border-dashed border-gray-700 rounded-xl">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
              <FolderOpen size={24} className="text-gray-600" />
            </div>
            <h4 className="text-lg font-medium text-white mb-2">No workflows yet</h4>
            <p className="text-gray-400 text-sm mb-4">
              Create your first automation to get started
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {workflows
              .filter((wf) => {
                if (!searchTerm) return true;
                return wf.name.toLowerCase().includes(searchTerm.toLowerCase());
              })
              .map((workflow) => {
                const starred = isFavorite(workflow.id);
                return (
                  <div
                    key={workflow.id}
                    className="flex items-center gap-3 p-3 bg-gray-800/50 hover:bg-gray-800 border border-gray-700/50 rounded-lg group transition-colors"
                  >
                    <button
                      onClick={() => handleToggleFavorite(workflow, singleProject?.name ?? 'My Automations')}
                      className={`transition-colors ${
                        starred
                          ? 'text-amber-400 hover:text-amber-300'
                          : 'text-gray-600 hover:text-amber-400'
                      }`}
                    >
                      {starred ? <Star size={16} fill="currentColor" /> : <StarOff size={16} />}
                    </button>
                    <button
                      onClick={() => onNavigateToWorkflow(singleProject?.id ?? '', workflow.id)}
                      className="flex-1 min-w-0 text-left"
                    >
                      <div className="font-medium text-white truncate group-hover:text-blue-300 transition-colors">
                        {workflow.name}
                      </div>
                      <div className="text-xs text-gray-500">
                        Updated {formatDate(workflow.updated_at ?? workflow.updatedAt)}
                      </div>
                    </button>
                    <button
                      onClick={() => onRunWorkflow(workflow.id)}
                      className="p-2 text-gray-500 hover:text-green-400 hover:bg-green-500/10 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
                      title="Run workflow"
                    >
                      <Play size={14} />
                    </button>
                  </div>
                );
              })}
          </div>
        )}

        {/* Tip for multi-project */}
        <div className="flex items-center gap-3 p-4 bg-blue-900/10 border border-blue-500/20 rounded-lg">
          <Lightbulb size={18} className="text-blue-400 flex-shrink-0" />
          <p className="text-sm text-blue-300/80">
            <span className="font-medium">Tip:</span> Create additional projects to organize workflows by purpose (e.g., "E2E Tests", "Marketing Replays")
          </p>
        </div>
      </div>
    );
  };

  // Render multi-project view (grid of projects)
  const renderMultiProjectView = () => {
    return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-white">Your Projects</h2>
            <p className="text-sm text-gray-400">{projects.length} projects</p>
          </div>
          <button
            onClick={onCreateProject}
            className="flex items-center gap-2 px-4 py-2 bg-flow-accent hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <Plus size={16} />
            New Project
          </button>
        </div>

        {/* Search */}
        <div className="relative">
          <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
          <input
            type="text"
            placeholder="Search projects..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-10 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
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

        {/* Projects Grid */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="animate-pulse bg-gray-800/50 border border-gray-700 rounded-xl p-5 h-40" />
            ))}
          </div>
        ) : filteredProjects.length === 0 ? (
          <div className="text-center py-12">
            <Search size={32} className="mx-auto mb-4 text-gray-600" />
            <p className="text-gray-400">No projects match "{searchTerm}"</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredProjects.map((project) => {
              const isDeleting = deletingProjectId === project.id;
              const isActionsOpen = showActionsFor === project.id;
              const workflowCount = project.stats?.workflow_count ?? 0;
              const executionCount = project.stats?.execution_count ?? 0;

              return (
                <div
                  key={project.id}
                  onClick={() => !isDeleting && onProjectSelect(project)}
                  className={`group relative bg-gray-800/50 border border-gray-700 rounded-xl p-5 cursor-pointer hover:border-flow-accent/60 hover:shadow-lg hover:shadow-blue-500/10 transition-all ${
                    isDeleting ? 'opacity-50 pointer-events-none' : ''
                  }`}
                >
                  {/* Deleting Overlay */}
                  {isDeleting && (
                    <div className="absolute inset-0 bg-gray-900/80 rounded-xl flex items-center justify-center z-10">
                      <Loader size={24} className="animate-spin text-red-400" />
                    </div>
                  )}

                  {/* Project Header */}
                  <div className="flex items-start justify-between gap-3 mb-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="flex-shrink-0 w-10 h-10 bg-gradient-to-br from-flow-accent/20 to-blue-500/10 rounded-xl flex items-center justify-center border border-flow-accent/20">
                        <FolderOpen size={18} className="text-flow-accent" />
                      </div>
                      <div className="min-w-0">
                        <h3 className="font-semibold text-white truncate">{project.name}</h3>
                        <div className="text-xs text-gray-500">
                          {workflowCount} workflow{workflowCount !== 1 ? 's' : ''}
                        </div>
                      </div>
                    </div>

                    {/* Actions Menu */}
                    <div className="relative flex-shrink-0">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setShowActionsFor(isActionsOpen ? null : project.id);
                        }}
                        className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
                      >
                        <MoreVertical size={16} />
                      </button>

                      {isActionsOpen && (
                        <>
                          <div
                            className="fixed inset-0 z-20"
                            onClick={(e) => {
                              e.stopPropagation();
                              setShowActionsFor(null);
                            }}
                          />
                          <div className="absolute right-0 top-full mt-1 z-30 w-48 bg-gray-800 border border-gray-700 rounded-lg shadow-xl overflow-hidden">
                            <button
                              onClick={(e) => handleRunAllWorkflows(e, project.id)}
                              disabled={bulkExecutionInProgress[project.id]}
                              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700 transition-colors"
                            >
                              <PlayCircle size={14} />
                              Run All Workflows
                            </button>
                            <div className="border-t border-gray-700" />
                            <button
                              onClick={(e) => handleDeleteProject(e, project.id, project.name)}
                              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-red-400 hover:bg-red-500/10 transition-colors"
                            >
                              <Trash2 size={14} />
                              Delete Project
                            </button>
                          </div>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Description */}
                  {project.description ? (
                    <p className="text-sm text-gray-400 line-clamp-2 mb-3">{project.description}</p>
                  ) : (
                    <p className="text-sm text-gray-600 italic mb-3">No description</p>
                  )}

                  {/* Quick Actions - Always visible at bottom */}
                  <div className="flex items-center gap-2 pt-3 border-t border-gray-700/50">
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        onProjectSelect(project);
                        // The onCreateProject would need to be called after navigating
                        // For now, just navigate to project which has "New Workflow" button
                      }}
                      className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium text-blue-300 hover:text-blue-200 bg-blue-900/30 hover:bg-blue-900/50 border border-blue-500/30 hover:border-blue-500/50 rounded-lg transition-colors"
                      title="Create new workflow in this project"
                    >
                      <Plus size={12} />
                      New Workflow
                    </button>
                    {workflowCount > 0 && (
                      <button
                        onClick={(e) => handleRunAllWorkflows(e, project.id)}
                        disabled={bulkExecutionInProgress[project.id]}
                        className="flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium text-green-300 hover:text-green-200 bg-green-900/30 hover:bg-green-900/50 border border-green-500/30 hover:border-green-500/50 rounded-lg transition-colors disabled:opacity-50"
                        title="Run all workflows in this project"
                      >
                        {bulkExecutionInProgress[project.id] ? (
                          <Loader size={12} className="animate-spin" />
                        ) : (
                          <PlayCircle size={12} />
                        )}
                        Run All
                      </button>
                    )}
                  </div>

                  {/* Stats Row */}
                  <div className="flex items-center justify-between text-xs text-gray-500 pt-2">
                    <span className="flex items-center gap-1">
                      <Clock size={10} />
                      {formatDate(project.updated_at)}
                    </span>
                    {executionCount > 0 && (
                      <span className="flex items-center gap-1 text-green-500/80">
                        <Play size={10} />
                        {executionCount} runs
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    );
  };

  return isSingleProjectMode ? renderSingleProjectView() : renderMultiProjectView();
};

export default ProjectsTab;
