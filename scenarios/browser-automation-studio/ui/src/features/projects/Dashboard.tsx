import { useState, useEffect, useCallback } from "react";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import {
  Plus,
  FolderOpen,
  Play,
  Clock,
  Calendar,
  PlayCircle,
  Loader,
  WifiOff,
  Search,
  X,
  Keyboard,
  Sparkles,
  Zap,
  FileCode,
  Bot,
  Settings,
  HelpCircle,
  Trash2,
  MoreVertical,
  Command,
} from "lucide-react";
import { useProjectStore, Project } from "@stores/projectStore";
import { useDashboardStore } from "@stores/dashboardStore";
import { openCalendar } from "@utils/vrooli";
import { selectors } from "@constants/selectors";
import { getModifierKey } from "@hooks/useKeyboardShortcuts";
import {
  ContinueEditingBanner,
  RecentWorkflowsWidget,
  RecentExecutionsWidget,
  QuickStartWidget,
  GlobalSearchModal,
  RunningIndicator,
  TemplatesGallery,
} from "@features/dashboard";

interface DashboardProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
  onShowKeyboardShortcuts?: () => void;
  onOpenSettings?: () => void;
  onOpenTutorial?: () => void;
  onNavigateToWorkflow?: (projectId: string, workflowId: string) => void;
  onViewExecution?: (executionId: string, workflowId: string) => void;
  onAIGenerateWorkflow?: (prompt: string) => void;
  onRunWorkflow?: (workflowId: string) => void;
  isGeneratingWorkflow?: boolean;
}

function Dashboard({
  onProjectSelect,
  onCreateProject,
  onShowKeyboardShortcuts,
  onOpenSettings,
  onOpenTutorial,
  onNavigateToWorkflow,
  onViewExecution,
  onAIGenerateWorkflow,
  onRunWorkflow,
  isGeneratingWorkflow = false,
}: DashboardProps) {
  const {
    projects,
    isLoading,
    error,
    bulkExecutionInProgress,
    fetchProjects,
    executeAllWorkflows,
    clearError,
    deleteProject,
  } = useProjectStore();
  const { fetchRecentWorkflows, fetchRecentExecutions } = useDashboardStore();
  const [searchTerm, setSearchTerm] = useState("");
  const [deletingProjectId, setDeletingProjectId] = useState<string | null>(null);
  const [showActionsFor, setShowActionsFor] = useState<string | null>(null);
  const [isSearchModalOpen, setIsSearchModalOpen] = useState(false);

  useEffect(() => {
    fetchProjects();
    fetchRecentWorkflows();
    fetchRecentExecutions();
  }, [fetchProjects, fetchRecentWorkflows, fetchRecentExecutions]);

  // Global keyboard shortcut for Cmd+K search
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setIsSearchModalOpen(true);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleNavigateToWorkflow = useCallback((projectId: string, workflowId: string) => {
    if (onNavigateToWorkflow) {
      onNavigateToWorkflow(projectId, workflowId);
    }
  }, [onNavigateToWorkflow]);

  const handleViewExecution = useCallback((executionId: string, workflowId: string) => {
    if (onViewExecution) {
      onViewExecution(executionId, workflowId);
    }
  }, [onViewExecution]);

  const handleAIGenerate = useCallback((prompt: string) => {
    if (onAIGenerateWorkflow) {
      onAIGenerateWorkflow(prompt);
    }
  }, [onAIGenerateWorkflow]);

  const handleUseTemplate = useCallback((prompt: string, _templateName: string) => {
    if (onAIGenerateWorkflow) {
      onAIGenerateWorkflow(prompt);
    }
  }, [onAIGenerateWorkflow]);

  const handleRunWorkflow = useCallback((workflowId: string) => {
    if (onRunWorkflow) {
      onRunWorkflow(workflowId);
    }
  }, [onRunWorkflow]);

  const handleViewAllWorkflows = useCallback(() => {
    // For now, just select the first project or create one
    if (projects.length > 0) {
      onProjectSelect(projects[0]);
    } else {
      onCreateProject();
    }
  }, [projects, onProjectSelect, onCreateProject]);

  const normalizedSearch = searchTerm.toLowerCase();
  const filteredProjects = projects.filter((project) => {
    const nameMatch =
      project.name?.toLowerCase().includes(normalizedSearch) ?? false;
    const descriptionMatch = project.description
      ? project.description.toLowerCase().includes(normalizedSearch)
      : false;
    return nameMatch || descriptionMatch;
  });

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const handleScheduleClick = async (e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent project selection
    try {
      await openCalendar();
    } catch (error) {
      logger.error(
        "Failed to open calendar",
        { component: "Dashboard", action: "handleOpenCalendar" },
        error,
      );
      alert(
        "Failed to open calendar. Make sure the calendar scenario is running.",
      );
    }
  };

  const handleRunAllWorkflows = async (
    e: React.MouseEvent,
    projectId: string,
  ) => {
    e.stopPropagation(); // Prevent project selection
    try {
      const result = await executeAllWorkflows(projectId);

      // Show success message with details
      const successCount = result.executions.filter(
        (exec) => exec.status !== "failed",
      ).length;
      const failureCount = result.executions.length - successCount;

      let message = `Started ${successCount} workflow(s)`;
      if (failureCount > 0) {
        message += `, ${failureCount} failed to start`;
      }

      alert(message);
    } catch (error) {
      logger.error(
        "Failed to execute all workflows",
        { component: "Dashboard", action: "handleExecuteAll" },
        error,
      );
      alert("Failed to execute workflows. Please try again.");
    }
  };

  const handleDeleteProject = useCallback(async (e: React.MouseEvent, projectId: string, projectName: string) => {
    e.stopPropagation();
    setShowActionsFor(null);

    const confirmed = window.confirm(
      `Delete "${projectName}" and all its workflows? This action cannot be undone.`
    );
    if (!confirmed) {
      return;
    }

    setDeletingProjectId(projectId);
    try {
      await deleteProject(projectId);
      toast.success(`Project "${projectName}" deleted`);
    } catch (error) {
      logger.error(
        "Failed to delete project",
        { component: "Dashboard", action: "handleDeleteProject", projectId },
        error
      );
      toast.error("Failed to delete project");
    } finally {
      setDeletingProjectId(null);
    }
  }, [deleteProject]);

  // Status bar for API connection issues
  const StatusBar = () => {
    if (!error) return null;

    return (
      <div className="px-4 sm:px-6 mt-4">
        <div className="bg-red-900/20 border-l-4 border-red-500 p-4 rounded-lg">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-3">
              <WifiOff size={20} className="text-red-400" />
              <div>
                <div className="text-red-400 font-medium">
                  API Connection Failed
                </div>
                <div className="text-red-300/80 text-sm">{error}</div>
              </div>
            </div>
            <button
              onClick={() => {
                clearError();
                fetchProjects();
              }}
              className="inline-flex items-center justify-center px-3 py-2 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="flex-1 flex flex-col min-h-[100svh] overflow-hidden">
      {/* Skip to content link for keyboard navigation */}
      <a
        href="#projects-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-flow-accent focus:text-white focus:rounded-md"
      >
        Skip to projects
      </a>

      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90">
        <div className="px-4 py-4 sm:px-6 sm:py-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <h1 className="text-2xl font-bold text-white">
                Browser Automation Studio
              </h1>
              <p className="text-gray-400 text-sm sm:text-base">
                Manage your automation projects and workflows
              </p>
            </div>
            <div className="hidden md:flex items-center gap-2">
              {/* Running Executions Indicator */}
              <RunningIndicator onViewExecution={handleViewExecution} />

              {/* Global Search Button */}
              <button
                onClick={() => setIsSearchModalOpen(true)}
                className="flex items-center gap-2 px-3 py-1.5 text-gray-400 hover:text-white bg-gray-800/50 hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors"
                title={`Search (${getModifierKey()}+K)`}
                aria-label="Open search"
              >
                <Search size={14} />
                <span className="text-sm">Search</span>
                <kbd className="hidden lg:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-xs text-gray-500 bg-gray-800 border border-gray-700 rounded">
                  <Command size={10} />K
                </kbd>
              </button>

              {onOpenTutorial && (
                <button
                  onClick={onOpenTutorial}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Tutorial (${getModifierKey()}+Shift+T)`}
                  aria-label="Open tutorial"
                >
                  <HelpCircle size={18} />
                </button>
              )}
              {onShowKeyboardShortcuts && (
                <button
                  onClick={onShowKeyboardShortcuts}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Keyboard shortcuts (${getModifierKey()}+?)`}
                  aria-label="Show keyboard shortcuts"
                >
                  <Keyboard size={18} />
                </button>
              )}
              {onOpenSettings && (
                <button
                  onClick={onOpenSettings}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Settings (${getModifierKey()}+,)`}
                  aria-label="Open settings"
                  data-testid={selectors.dashboard.settingsButton}
                >
                  <Settings size={18} />
                </button>
              )}
              <button
                data-testid={selectors.dashboard.newProjectButton}
                onClick={onCreateProject}
                className="inline-flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
              >
                <Plus size={16} />
                New Project
              </button>
            </div>
          </div>

          {/* Search */}
          <div className="relative mt-4 sm:mt-6">
            <label htmlFor="project-search" className="sr-only">
              Search projects
            </label>
            <Search
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500"
              aria-hidden="true"
            />
            <input
              id="project-search"
              type="text"
              placeholder="Search projects..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/50"
              aria-label="Search for projects by name or description"
              data-testid={selectors.projects.search.input}
            />
            {searchTerm && (
              <button
                type="button"
                onClick={() => setSearchTerm("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300 transition-colors"
                aria-label="Clear project search"
                data-testid={selectors.projects.search.clearButton}
              >
                <X size={16} aria-hidden="true" />
              </button>
            )}
          </div>
        </div>
      </header>

      {/* Status Bar for API errors */}
      <StatusBar />

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-auto px-4 py-6 sm:px-6">
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6 animate-pulse">
            {Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="bg-flow-node border border-gray-700 rounded-lg p-6"
              >
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 bg-gray-700 rounded-lg" />
                  <div className="flex-1">
                    <div className="h-5 bg-gray-700 rounded w-3/4 mb-2" />
                    <div className="h-3 bg-gray-700 rounded w-1/2" />
                  </div>
                </div>
                <div className="h-4 bg-gray-700 rounded w-full mb-2" />
                <div className="h-4 bg-gray-700 rounded w-2/3 mb-4" />
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div className="text-center">
                    <div className="h-6 bg-gray-700 rounded w-12 mx-auto mb-1" />
                    <div className="h-3 bg-gray-700 rounded w-16 mx-auto" />
                  </div>
                  <div className="text-center">
                    <div className="h-6 bg-gray-700 rounded w-12 mx-auto mb-1" />
                    <div className="h-3 bg-gray-700 rounded w-16 mx-auto" />
                  </div>
                </div>
                <div className="flex justify-between pt-4 border-t border-gray-700">
                  <div className="h-3 bg-gray-700 rounded w-20" />
                  <div className="h-3 bg-gray-700 rounded w-16" />
                </div>
              </div>
            ))}
          </div>
        ) : filteredProjects.length === 0 && searchTerm === "" ? (
          <div className="max-w-4xl mx-auto">
            {/* Welcome Section */}
            <div className="text-center mb-10">
              <div className="mb-6 flex items-center justify-center">
                <div className="p-4 bg-flow-accent/20 rounded-2xl">
                  {error ? <WifiOff size={48} className="text-red-400" /> : <Bot size={48} className="text-flow-accent" />}
                </div>
              </div>
              <h2 className="text-2xl font-bold text-white mb-3">
                {error ? "Unable to Load Projects" : "Welcome to Browser Automation Studio"}
              </h2>
              <p className="text-gray-400 max-w-lg mx-auto">
                {error
                  ? "There was an issue connecting to the API. You can still use the interface when the connection is restored."
                  : "Create visual workflows to automate browser tasks, test UIs, and extract data from websites with AI assistance."}
              </p>
            </div>

            {!error && (
              <>
                {/* Getting Started Steps */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
                  <div className="bg-flow-node border border-gray-700 rounded-xl p-5 hover:border-flow-accent/50 transition-colors">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center text-blue-400 font-bold text-sm">
                        1
                      </div>
                      <FolderOpen size={20} className="text-flow-accent" />
                    </div>
                    <h3 className="font-semibold text-white mb-2">Create a Project</h3>
                    <p className="text-sm text-gray-400">
                      Organize your workflows into projects for easy management.
                    </p>
                  </div>

                  <div className="bg-flow-node border border-gray-700 rounded-xl p-5 hover:border-flow-accent/50 transition-colors">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center text-blue-400 font-bold text-sm">
                        2
                      </div>
                      <Sparkles size={20} className="text-amber-400" />
                    </div>
                    <h3 className="font-semibold text-white mb-2">Build with AI</h3>
                    <p className="text-sm text-gray-400">
                      Describe what you want to automate and let AI create the workflow.
                    </p>
                  </div>

                  <div className="bg-flow-node border border-gray-700 rounded-xl p-5 hover:border-flow-accent/50 transition-colors">
                    <div className="flex items-center gap-3 mb-3">
                      <div className="w-8 h-8 bg-blue-500/20 rounded-lg flex items-center justify-center text-blue-400 font-bold text-sm">
                        3
                      </div>
                      <Zap size={20} className="text-green-400" />
                    </div>
                    <h3 className="font-semibold text-white mb-2">Execute & Monitor</h3>
                    <p className="text-sm text-gray-400">
                      Run workflows and watch real-time screenshots of automation progress.
                    </p>
                  </div>
                </div>

                {/* CTA and Tips */}
                <div className="flex flex-col items-center gap-6">
                  <button
                    data-testid={selectors.dashboard.newProjectButton}
                    onClick={onCreateProject}
                    className="flex items-center gap-2 px-6 py-3 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors text-lg font-medium"
                  >
                    <Plus size={20} />
                    Create Your First Project
                  </button>

                  <div className="flex flex-wrap items-center justify-center gap-4 sm:gap-6 text-sm text-gray-500">
                    {onOpenTutorial && (
                      <button
                        onClick={onOpenTutorial}
                        className="flex items-center gap-1.5 hover:text-gray-300 transition-colors"
                      >
                        <HelpCircle size={14} />
                        <span>View tutorial</span>
                      </button>
                    )}
                    <span className="flex items-center gap-1.5">
                      <Keyboard size={14} />
                      Press <kbd className="px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">{getModifierKey()}+?</kbd> for shortcuts
                    </span>
                    <span className="hidden sm:flex items-center gap-1.5">
                      <FileCode size={14} />
                      Visual workflow builder included
                    </span>
                  </div>
                </div>
              </>
            )}
          </div>
        ) : filteredProjects.length === 0 ? (
          <div className="flex items-center justify-center h-64 animate-fade-in">
            <div className="text-center max-w-sm">
              <div className="mb-4 flex items-center justify-center">
                <div className="w-16 h-16 rounded-full bg-gray-800 flex items-center justify-center">
                  <Search size={28} className="text-gray-500" />
                </div>
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">
                No Projects Found
              </h3>
              <p className="text-gray-400 mb-4">
                No projects match &ldquo;<span className="text-gray-300">{searchTerm}</span>&rdquo;
              </p>
              <button
                onClick={() => setSearchTerm("")}
                className="text-flow-accent hover:text-blue-400 text-sm font-medium transition-colors"
              >
                Clear search
              </button>
            </div>
          </div>
        ) : (
          <div id="projects-content" role="region" aria-label="Dashboard">
            {/* Dashboard Widgets Section */}
            <div className="space-y-6 mb-8">
              {/* Continue Editing Banner */}
              <ContinueEditingBanner onNavigate={handleNavigateToWorkflow} />

              {/* Quick Start - AI Workflow Generation */}
              {onAIGenerateWorkflow && (
                <QuickStartWidget
                  onAIGenerate={handleAIGenerate}
                  onCreateManual={onCreateProject}
                  isGenerating={isGeneratingWorkflow}
                />
              )}

              {/* Recent Items Row */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <RecentWorkflowsWidget
                  onNavigate={handleNavigateToWorkflow}
                  onViewAll={handleViewAllWorkflows}
                />
                <RecentExecutionsWidget
                  onViewExecution={handleViewExecution}
                  onViewAll={handleViewAllWorkflows}
                />
              </div>

              {/* Templates Gallery */}
              {onAIGenerateWorkflow && (
                <TemplatesGallery onUseTemplate={handleUseTemplate} />
              )}
            </div>

            {/* Projects Section Header */}
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-white">Your Projects</h2>
              <span className="text-sm text-gray-500">{filteredProjects.length} project{filteredProjects.length !== 1 ? 's' : ''}</span>
            </div>

            {/* Projects Grid */}
            <div
              className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
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
                  className={`group relative bg-flow-node border border-gray-700 rounded-xl p-5 cursor-pointer hover:border-flow-accent/60 hover:shadow-lg hover:shadow-blue-500/10 transition-all ${
                    isDeleting ? "opacity-50 pointer-events-none" : ""
                  }`}
                  data-testid={selectors.projects.card}
                  data-project-id={project.id}
                  data-project-name={project.name}
                >
                  {/* Deleting Overlay */}
                  {isDeleting && (
                    <div className="absolute inset-0 bg-flow-node/80 rounded-xl flex items-center justify-center z-10">
                      <Loader size={24} className="animate-spin text-red-400" />
                    </div>
                  )}

                  {/* Project Header */}
                  <div className="flex items-start justify-between gap-3 mb-4">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="flex-shrink-0 w-10 h-10 bg-gradient-to-br from-flow-accent/20 to-blue-500/10 rounded-xl flex items-center justify-center border border-flow-accent/20">
                        <FolderOpen size={18} className="text-flow-accent" />
                      </div>
                      <div className="min-w-0">
                        <h3
                          className="font-semibold text-white truncate"
                          title={project.name}
                          data-testid={selectors.projects.cardTitle}
                          data-project-id={project.id}
                          data-project-name={project.name}
                        >
                          {project.name}
                        </h3>
                        <div className="flex items-center gap-2 text-xs text-gray-500 mt-0.5">
                          <span>{workflowCount} workflow{workflowCount !== 1 ? 's' : ''}</span>
                          {executionCount > 0 && (
                            <>
                              <span>•</span>
                              <span>{executionCount} run{executionCount !== 1 ? 's' : ''}</span>
                            </>
                          )}
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
                        className="p-1.5 text-gray-500 hover:text-white hover:bg-gray-700 rounded-lg transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
                        aria-label="Project actions"
                        aria-expanded={isActionsOpen}
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
                          <div className="absolute right-0 top-full mt-1 z-30 w-48 bg-flow-node border border-gray-700 rounded-lg shadow-xl overflow-hidden animate-fade-in">
                            <button
                              onClick={(e) => handleRunAllWorkflows(e, project.id)}
                              disabled={bulkExecutionInProgress[project.id]}
                              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-white transition-colors disabled:opacity-50"
                            >
                              {bulkExecutionInProgress[project.id] ? (
                                <Loader size={14} className="animate-spin" />
                              ) : (
                                <PlayCircle size={14} />
                              )}
                              Run All Workflows
                            </button>
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                setShowActionsFor(null);
                                handleScheduleClick(e);
                              }}
                              className="w-full flex items-center gap-3 px-4 py-2.5 text-sm text-gray-300 hover:bg-gray-700/50 hover:text-white transition-colors"
                            >
                              <Calendar size={14} />
                              Schedule
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

                  {/* Project Description */}
                  {project.description ? (
                    <p className="text-gray-400 text-sm mb-4 line-clamp-2">
                      {project.description}
                    </p>
                  ) : (
                    <p className="text-gray-600 text-sm mb-4 italic">
                      No description
                    </p>
                  )}

                  {/* Quick Stats */}
                  {(workflowCount > 0 || executionCount > 0) && (
                    <div className="grid grid-cols-2 gap-3 mb-4">
                      <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                        <div className="text-lg font-semibold text-white">{workflowCount}</div>
                        <div className="text-xs text-gray-500">Workflows</div>
                      </div>
                      <div className="bg-gray-800/50 rounded-lg px-3 py-2 text-center">
                        <div className="text-lg font-semibold text-white">{executionCount}</div>
                        <div className="text-xs text-gray-500">Executions</div>
                      </div>
                    </div>
                  )}

                  {/* Footer */}
                  <div className="flex items-center justify-between text-xs text-gray-500 pt-3 border-t border-gray-700/50">
                    <div className="flex items-center gap-1.5">
                      <Clock size={12} />
                      <span>Updated {formatDate(project.updated_at)}</span>
                    </div>
                    {project.stats?.last_execution && (
                      <div className="flex items-center gap-1.5 text-green-500/80">
                        <Play size={10} />
                        <span>{formatDate(project.stats.last_execution)}</span>
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
            </div>
          </div>
        )}
      </div>

      {/* Floating Action Button - Mobile */}
      <button
        type="button"
        onClick={onCreateProject}
        className="md:hidden fixed bottom-6 right-6 z-40 w-14 h-14 bg-flow-accent text-white rounded-full shadow-lg hover:bg-blue-600 transition-all hover:shadow-xl flex items-center justify-center"
        aria-label="Create new project"
      >
        <Plus size={24} />
      </button>

      {/* Global Search Modal */}
      <GlobalSearchModal
        isOpen={isSearchModalOpen}
        onClose={() => setIsSearchModalOpen(false)}
        onSelectWorkflow={handleNavigateToWorkflow}
        onSelectProject={(projectId) => {
          const project = projects.find(p => p.id === projectId);
          if (project) onProjectSelect(project);
        }}
        onSelectExecution={handleViewExecution}
        onRunWorkflow={handleRunWorkflow}
      />
    </div>
  );
}

export default Dashboard;
