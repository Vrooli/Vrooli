import { useState, useEffect } from "react";
import { logger } from "@utils/logger";
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
} from "lucide-react";
import { useProjectStore, Project } from "@stores/projectStore";
import { openCalendar } from "@utils/vrooli";
import { selectors } from "@constants/selectors";

interface DashboardProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
}

function Dashboard({ onProjectSelect, onCreateProject }: DashboardProps) {
  const {
    projects,
    isLoading,
    error,
    bulkExecutionInProgress,
    fetchProjects,
    executeAllWorkflows,
    clearError,
  } = useProjectStore();
  const [searchTerm, setSearchTerm] = useState("");

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

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
            <button
              data-testid={selectors.dashboard.newProjectButton}
              onClick={onCreateProject}
              className="hidden md:inline-flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
            >
              <Plus size={16} />
              New Project
            </button>
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
          <div className="flex items-center justify-center h-64">
            <div className="text-gray-400">Loading projects...</div>
          </div>
        ) : filteredProjects.length === 0 && searchTerm === "" ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="mb-4 flex items-center justify-center text-gray-600">
                {error ? <WifiOff size={48} /> : <FolderOpen size={48} />}
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">
                {error ? "Unable to Load Projects" : "No Projects Yet"}
              </h3>
              <p className="text-gray-400 mb-6">
                {error
                  ? "There was an issue connecting to the API. You can still use the interface when the connection is restored."
                  : "Create your first project to get started with browser automation"}
              </p>
              {!error && (
                <button
                  data-testid={selectors.dashboard.newProjectButton}
                  onClick={onCreateProject}
                  className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
                >
                  <Plus size={16} />
                  Create Project
                </button>
              )}
            </div>
          </div>
        ) : filteredProjects.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="mb-4 flex items-center justify-center text-gray-600">
                <FolderOpen size={48} />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">
                No Projects Found
              </h3>
              <p className="text-gray-400">
                No projects match your search criteria
              </p>
            </div>
          </div>
        ) : (
          <div
            id="projects-content"
            className="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
            data-testid={selectors.projects.grid}
            role="region"
            aria-label="Projects list"
          >
            {filteredProjects.map((project) => (
              <div
                key={project.id}
                onClick={() => onProjectSelect(project)}
                className="bg-flow-node border border-gray-700 rounded-lg p-5 sm:p-6 cursor-pointer hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/20 transition-all"
                data-testid={selectors.projects.card}
                data-project-id={project.id}
                data-project-name={project.name}
              >
                {/* Project Header */}
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 sm:w-8 sm:h-8 bg-flow-accent/20 rounded-lg flex items-center justify-center">
                      <FolderOpen size={16} className="text-flow-accent" />
                    </div>
                    <div>
                      <h3
                        className="font-semibold text-white truncate max-w-[12rem] sm:max-w-32"
                        title={project.name}
                        data-testid={selectors.projects.cardTitle}
                        data-project-id={project.id}
                        data-project-name={project.name}
                      >
                        {project.name}
                      </h3>
                    </div>
                  </div>

                  {/* Action Buttons */}
                  <div className="hidden sm:flex items-center gap-1">
                    {/* Schedule Button */}
                    <button
                      onClick={(e) => handleScheduleClick(e)}
                      className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                      title="Open Calendar"
                      aria-label="Open Calendar for project"
                    >
                      <Calendar size={14} />
                    </button>

                    {/* Run All Workflows Button */}
                    <button
                      onClick={(e) => handleRunAllWorkflows(e, project.id)}
                      disabled={bulkExecutionInProgress[project.id]}
                      className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      title={
                        bulkExecutionInProgress[project.id]
                          ? "Running workflows..."
                          : "Run All Workflows"
                      }
                      aria-label={
                        bulkExecutionInProgress[project.id]
                          ? "Running workflows"
                          : "Run all workflows in project"
                      }
                    >
                      {bulkExecutionInProgress[project.id] ? (
                        <Loader size={14} className="animate-spin" />
                      ) : (
                        <PlayCircle size={14} />
                      )}
                    </button>
                  </div>
                </div>

                {/* Mobile Actions */}
                <div className="flex flex-col gap-2 sm:hidden">
                  <button
                    onClick={(e) => handleRunAllWorkflows(e, project.id)}
                    disabled={bulkExecutionInProgress[project.id]}
                    className="flex items-center justify-center gap-2 rounded-md bg-flow-accent/80 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-flow-accent disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {bulkExecutionInProgress[project.id] ? (
                      <Loader size={14} className="animate-spin" />
                    ) : (
                      <PlayCircle size={14} />
                    )}
                    Run Workflows
                  </button>
                  <button
                    onClick={(e) => handleScheduleClick(e)}
                    className="flex items-center justify-center gap-2 rounded-md border border-gray-700 px-3 py-2 text-sm font-medium text-gray-200 transition-colors hover:bg-gray-700/70"
                  >
                    <Calendar size={14} />
                    Open Calendar
                  </button>
                </div>

                {/* Project Description */}
                {project.description && (
                  <p className="text-gray-400 text-sm mb-4 line-clamp-3">
                    {project.description}
                  </p>
                )}

                {/* Last Activity */}
                <div className="flex items-center justify-between text-xs text-gray-500 pt-4 border-t border-gray-700">
                  <div className="flex items-center gap-1">
                    <Clock size={12} />
                    <span>Updated {formatDate(project.updated_at)}</span>
                  </div>
                  {project.stats?.last_execution && (
                    <div className="flex items-center gap-1">
                      <Play size={12} />
                      <span>
                        Last run {formatDate(project.stats.last_execution)}
                      </span>
                    </div>
                  )}
                </div>

                {/* Folder Path */}
                <div
                  className="mt-2 text-xs text-gray-600 truncate"
                  title={project.folder_path}
                >
                  {project.folder_path}
                </div>
              </div>
            ))}
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
    </div>
  );
}

export default Dashboard;
