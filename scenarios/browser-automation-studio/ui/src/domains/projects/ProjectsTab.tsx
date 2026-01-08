import React, { useState, useCallback } from 'react';
import {
  Plus,
  FolderOpen,
  Search,
  Trash2,
  MoreVertical,
  Loader,
  PlayCircle,
  Clock,
  X,
  Play,
  Sparkles,
  Lightbulb,
  Upload,
} from 'lucide-react';
import { useProjectStore, type Project } from './store';
import { formatDistanceToNow } from 'date-fns';
import toast from 'react-hot-toast';
import { selectors } from '@constants/selectors';
import { TabEmptyState, ProjectsEmptyPreview } from '@/views/DashboardView/previews';
import ProjectImportModal from './ProjectImportModal';

interface ProjectsTabProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
  onCreateWorkflow?: () => void;
  onNavigateToWorkflow: (projectId: string, workflowId: string) => void;
  onRunWorkflow: (workflowId: string) => void;
  onTryDemo?: () => void;
}

export const ProjectsTab: React.FC<ProjectsTabProps> = ({
  onProjectSelect,
  onCreateProject,
  onCreateWorkflow,
  onTryDemo,
}) => {
  const {
    projects,
    isLoading,
    bulkExecutionInProgress,
    deleteProject,
    executeAllWorkflows,
  } = useProjectStore();
  const [searchTerm, setSearchTerm] = useState('');
  const [deletingProjectId, setDeletingProjectId] = useState<string | null>(null);
  const [showActionsFor, setShowActionsFor] = useState<string | null>(null);
  const [showImportModal, setShowImportModal] = useState(false);

  const handleImportSuccess = useCallback((project: Project) => {
    onProjectSelect(project);
  }, [onProjectSelect]);

  // Filter projects based on search
  const filteredProjects = projects.filter((project) => {
    const searchLower = searchTerm.toLowerCase();
    return (
      project.name?.toLowerCase().includes(searchLower) ||
      project.description?.toLowerCase().includes(searchLower)
    );
  });

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

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Never';
    return formatDistanceToNow(new Date(dateString), { addSuffix: true });
  };

  const renderHero = () => (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between bg-gradient-to-r from-gray-900/80 via-gray-900 to-blue-900/20 border border-gray-800/80 rounded-2xl p-4">
      <div className="space-y-2">
        <div className="text-xs text-gray-400">Projects</div>
        <div className="text-xl font-semibold text-white">
          Organize, launch, and share your automations
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-gray-300">
          <span className="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-blue-500/15 border border-blue-500/30 text-blue-100">
            <FolderOpen size={12} /> {projects.length} project{projects.length !== 1 ? 's' : ''}
          </span>
        </div>
      </div>
      <div className="flex flex-col sm:flex-row gap-2">
        <button
          onClick={onCreateProject}
          data-testid={selectors.dashboard.newProjectButton}
          className="hero-button-primary w-full sm:w-auto justify-center"
        >
          New project
          <div className="hero-button-glow" />
        </button>
        <button
          onClick={() => setShowImportModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-600 rounded-lg transition-colors w-full sm:w-auto justify-center"
        >
          <Upload size={16} />
          Import
        </button>
      </div>
    </div>
  );
  return (
    <div className="space-y-6">
      {renderHero()}
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-white">Your Projects</h2>
          <p className="text-sm text-gray-400">{projects.length} projects</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowImportModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-600 rounded-lg transition-colors"
          >
            <Upload size={16} />
            Import
          </button>
          <button
            onClick={onCreateProject}
            data-testid={selectors.dashboard.newProjectButton}
            className="flex items-center gap-2 px-4 py-2 bg-flow-accent hover:bg-blue-600 text-white rounded-lg transition-colors"
          >
            <Plus size={16} />
            New Project
          </button>
        </div>
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
          data-testid={selectors.projects.search.input}
        />
        {searchTerm && (
          <button
            onClick={() => setSearchTerm('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
            data-testid={selectors.projects.search.clearButton}
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
        projects.length === 0 && !searchTerm ? (
          <TabEmptyState
            icon={<FolderOpen size={22} />}
            title="Start your automation portfolio"
            subtitle="Group workflows by customer or use case, then run and export with confidence."
            preview={<ProjectsEmptyPreview />}
            variant="polished"
            primaryCta={{ label: 'Create your first project', onClick: onCreateProject }}
            secondaryCta={onTryDemo ? { label: 'Try demo workflow', onClick: onTryDemo } : undefined}
            progressPath={[
              { label: 'Create project', active: true },
              { label: 'Add workflows' },
              { label: 'Run & export' },
            ]}
            features={[
              {
                title: 'Templates & AI',
                description: 'Launch from guided templates or describe what to automate.',
                icon: <Sparkles size={16} />,
              },
              {
                title: 'Portfolio view',
                description: 'See workflows, executions, and exports at a glance per project.',
                icon: <Lightbulb size={16} />,
              },
            ]}
          />
        ) : (
          <div className="text-center py-12">
            <Search size={32} className="mx-auto mb-4 text-gray-600" />
            <p className="text-gray-400">No projects match "{searchTerm}"</p>
          </div>
        )
      ) : (
        <div
          className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
          data-testid={selectors.projects.grid}
        >
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
                data-testid={selectors.projects.card}
                data-project-id={project.id}
                data-project-name={project.name}
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
                      <h3
                        className="font-semibold text-white truncate"
                        data-testid={selectors.projects.cardTitle}
                      >
                        {project.name}
                      </h3>
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
                      if (onCreateWorkflow) {
                        onCreateWorkflow();
                      } else {
                        onProjectSelect(project);
                      }
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

      {/* Import Modal */}
      <ProjectImportModal
        isOpen={showImportModal}
        onClose={() => setShowImportModal(false)}
        onSuccess={handleImportSuccess}
      />
    </div>
  );
};

export default ProjectsTab;
