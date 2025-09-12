import { useState, useEffect } from 'react';
import { Plus, FolderOpen, Play, Clock, Activity, Calendar, PlayCircle, Loader } from 'lucide-react';
import { useProjectStore, Project } from '../stores/projectStore';
import { openCalendar } from '../utils/vrooli';

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
    clearError 
  } = useProjectStore();
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  const filteredProjects = projects.filter(project =>
    project.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    project.description?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const handleScheduleClick = async (e: React.MouseEvent, projectId: string) => {
    e.stopPropagation(); // Prevent project selection
    try {
      await openCalendar();
    } catch (error) {
      console.error('Failed to open calendar:', error);
      alert('Failed to open calendar. Make sure the calendar scenario is running.');
    }
  };

  const handleRunAllWorkflows = async (e: React.MouseEvent, projectId: string) => {
    e.stopPropagation(); // Prevent project selection
    try {
      const result = await executeAllWorkflows(projectId);
      
      // Show success message with details
      const successCount = result.executions.filter(exec => exec.status !== 'failed').length;
      const failureCount = result.executions.length - successCount;
      
      let message = `Started ${successCount} workflow(s)`;
      if (failureCount > 0) {
        message += `, ${failureCount} failed to start`;
      }
      
      alert(message);
    } catch (error) {
      console.error('Failed to execute workflows:', error);
      alert('Failed to execute workflows. Please try again.');
    }
  };

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-400 mb-4">
            <Activity size={48} />
          </div>
          <h3 className="text-lg font-semibold text-white mb-2">Error Loading Projects</h3>
          <p className="text-gray-400 mb-4">{error}</p>
          <button
            onClick={() => {
              clearError();
              fetchProjects();
            }}
            className="px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            Try Again
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="p-6 border-b border-gray-800">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-white mb-2">Browser Automation Studio</h1>
            <p className="text-gray-400">Manage your automation projects and workflows</p>
          </div>
          <button
            onClick={onCreateProject}
            className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            <Plus size={16} />
            New Project
          </button>
        </div>

        {/* Search */}
        <div className="relative">
          <input
            type="text"
            placeholder="Search projects..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-4 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent"
          />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-gray-400">Loading projects...</div>
          </div>
        ) : filteredProjects.length === 0 && searchTerm === '' ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="text-gray-600 mb-4">
                <FolderOpen size={48} />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">No Projects Yet</h3>
              <p className="text-gray-400 mb-6">Create your first project to get started with browser automation</p>
              <button
                onClick={onCreateProject}
                className="flex items-center gap-2 px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
              >
                <Plus size={16} />
                Create Project
              </button>
            </div>
          </div>
        ) : filteredProjects.length === 0 ? (
          <div className="flex items-center justify-center h-64">
            <div className="text-center">
              <div className="text-gray-600 mb-4">
                <FolderOpen size={48} />
              </div>
              <h3 className="text-lg font-semibold text-white mb-2">No Projects Found</h3>
              <p className="text-gray-400">No projects match your search criteria</p>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
            {filteredProjects.map((project) => (
              <div
                key={project.id}
                onClick={() => onProjectSelect(project)}
                className="bg-flow-node border border-gray-700 rounded-lg p-6 cursor-pointer hover:border-flow-accent hover:shadow-lg hover:shadow-blue-500/20 transition-all"
              >
                {/* Project Header */}
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-flow-accent/20 rounded-lg flex items-center justify-center">
                      <FolderOpen size={16} className="text-flow-accent" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-white truncate max-w-32" title={project.name}>
                        {project.name}
                      </h3>
                    </div>
                  </div>
                  
                  {/* Action Buttons */}
                  <div className="flex items-center gap-1">
                    {/* Schedule Button */}
                    <button
                      onClick={(e) => handleScheduleClick(e, project.id)}
                      className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                      title="Open Calendar"
                    >
                      <Calendar size={14} />
                    </button>
                    
                    {/* Run All Workflows Button */}
                    <button
                      onClick={(e) => handleRunAllWorkflows(e, project.id)}
                      disabled={bulkExecutionInProgress[project.id]}
                      className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      title={bulkExecutionInProgress[project.id] ? 'Running workflows...' : 'Run All Workflows'}
                    >
                      {bulkExecutionInProgress[project.id] ? (
                        <Loader size={14} className="animate-spin" />
                      ) : (
                        <PlayCircle size={14} />
                      )}
                    </button>
                  </div>
                </div>

                {/* Project Description */}
                {project.description && (
                  <p className="text-gray-400 text-sm mb-4 line-clamp-2">
                    {project.description}
                  </p>
                )}

                {/* Project Stats */}
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div className="text-center">
                    <div className="text-lg font-semibold text-white">
                      {project.stats?.workflow_count || 0}
                    </div>
                    <div className="text-xs text-gray-500">Workflows</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold text-white">
                      {project.stats?.execution_count || 0}
                    </div>
                    <div className="text-xs text-gray-500">Executions</div>
                  </div>
                </div>

                {/* Last Activity */}
                <div className="flex items-center justify-between text-xs text-gray-500 pt-4 border-t border-gray-700">
                  <div className="flex items-center gap-1">
                    <Clock size={12} />
                    <span>Updated {formatDate(project.updated_at)}</span>
                  </div>
                  {project.stats?.last_execution && (
                    <div className="flex items-center gap-1">
                      <Play size={12} />
                      <span>Last run {formatDate(project.stats.last_execution)}</span>
                    </div>
                  )}
                </div>

                {/* Folder Path */}
                <div className="mt-2 text-xs text-gray-600 truncate" title={project.folder_path}>
                  {project.folder_path}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default Dashboard;