/**
 * WorkflowPickerModal Component
 *
 * A modal for selecting a workflow to execute. Shows workflows from the selected
 * project with ability to switch projects. When a workflow is selected, it returns
 * the workflow details to the parent for execution setup.
 */

import { useCallback, useEffect, useId, useMemo, useState } from 'react';
import { X, Play, Search } from 'lucide-react';
import { useProjectStore, useProjectDetailStore, WorkflowCard, type WorkflowWithStats, type Project } from '@/domains/projects';
import { ResponsiveDialog } from '@shared/layout';
import { ProjectSelector } from './ProjectSelector';

interface WorkflowPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (workflowId: string, projectId: string, workflowName: string) => void;
  /** Initial project to show (from last selection or smart default) */
  initialProjectId?: string | null;
}

export function WorkflowPickerModal({
  isOpen,
  onClose,
  onSelect,
  initialProjectId,
}: WorkflowPickerModalProps) {
  const titleId = useId();
  const { projects, getSmartDefaultProject } = useProjectStore();
  const { workflows, isLoading, fetchWorkflows } = useProjectDetailStore();

  const [selectedProject, setSelectedProject] = useState<Project | null>(() => {
    if (initialProjectId) {
      return projects.find(p => p.id === initialProjectId) ?? null;
    }
    return null;
  });
  const [searchTerm, setSearchTerm] = useState('');

  // Initialize with smart default project when modal opens
  useEffect(() => {
    if (isOpen && !selectedProject && projects.length > 0) {
      const defaultProject = getSmartDefaultProject();
      if (defaultProject) {
        setSelectedProject(defaultProject);
      } else {
        setSelectedProject(projects[0]);
      }
    }
  }, [isOpen, selectedProject, projects, getSmartDefaultProject]);

  // Fetch workflows when project changes
  useEffect(() => {
    if (selectedProject?.id && isOpen) {
      fetchWorkflows(selectedProject.id);
    }
  }, [selectedProject?.id, isOpen, fetchWorkflows]);

  // Reset state when modal closes
  useEffect(() => {
    if (!isOpen) {
      setSearchTerm('');
    }
  }, [isOpen]);

  // Filter workflows based on search term
  const filteredWorkflows = useMemo(() => {
    if (!searchTerm.trim()) return workflows;
    const lowerSearch = searchTerm.toLowerCase();
    return workflows.filter(
      (w) =>
        w.name.toLowerCase().includes(lowerSearch) ||
        (w.description && String(w.description).toLowerCase().includes(lowerSearch))
    );
  }, [workflows, searchTerm]);

  const handleProjectChange = useCallback((project: Project) => {
    setSelectedProject(project);
    setSearchTerm(''); // Clear search when switching projects
  }, []);

  const handleWorkflowClick = useCallback((workflow: WorkflowWithStats) => {
    if (!selectedProject) return;
    onSelect(workflow.id, selectedProject.id, workflow.name);
    onClose();
  }, [onSelect, selectedProject, onClose]);

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="xl"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div className="flex flex-col max-h-[85vh]">
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4 flex-shrink-0">
          <button
            onClick={onClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>

          <div className="flex flex-col items-center text-center mb-4">
            <div className="w-14 h-14 bg-gradient-to-br from-flow-accent/30 to-green-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-flow-accent/10">
              <Play size={28} className="text-flow-accent" />
            </div>
            <h2 id={titleId} className="text-xl font-bold text-white tracking-tight">
              Select Workflow
            </h2>
            <p className="text-sm text-gray-400 mt-1">Choose a workflow to execute</p>
          </div>

          {/* Project selector */}
          {projects.length > 1 && (
            <div className="mb-4">
              <ProjectSelector
                selectedProject={selectedProject}
                onSelectProject={handleProjectChange}
                variant="dropdown"
                placeholder="Select project"
              />
            </div>
          )}

          {/* Search input */}
          {workflows.length > 0 && (
            <div className="relative">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
              <input
                type="text"
                placeholder="Search workflows..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-9 pr-4 py-2 rounded-lg border border-gray-700 bg-gray-800/50 text-white placeholder-gray-500 text-sm focus:outline-none focus:border-flow-accent focus:ring-1 focus:ring-flow-accent/50 transition-colors"
              />
              {searchTerm && (
                <button
                  onClick={() => setSearchTerm('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-white transition-colors"
                  aria-label="Clear search"
                >
                  <X size={14} />
                </button>
              )}
            </div>
          )}
        </div>

        {/* Content */}
        <div className="px-6 pb-6 flex-1 overflow-y-auto min-h-0">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="w-8 h-8 border-2 border-gray-400 border-t-transparent rounded-full animate-spin" />
            </div>
          ) : workflows.length === 0 ? (
            <div className="text-center py-12">
              <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
                <Play size={28} className="text-gray-500" />
              </div>
              <p className="text-gray-400 text-sm">No workflows in this project</p>
              <p className="text-gray-500 text-xs mt-1">
                Create a workflow first by recording actions
              </p>
            </div>
          ) : filteredWorkflows.length === 0 ? (
            <div className="text-center py-12">
              <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
                <Search size={28} className="text-gray-500" />
              </div>
              <p className="text-gray-400 text-sm">No workflows match your search</p>
              <button
                onClick={() => setSearchTerm('')}
                className="text-flow-accent hover:text-blue-400 text-sm mt-2 transition-colors"
              >
                Clear search
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
              {filteredWorkflows.map((workflow) => (
                <WorkflowCard
                  key={workflow.id}
                  workflow={workflow}
                  onClick={handleWorkflowClick}
                  hideActions
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </ResponsiveDialog>
  );
}
