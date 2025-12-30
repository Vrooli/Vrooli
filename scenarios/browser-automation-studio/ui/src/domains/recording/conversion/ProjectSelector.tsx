/**
 * ProjectSelector Component
 *
 * A reusable component for selecting a project. Shows a button that displays
 * the currently selected project (or a placeholder) and opens a ProjectPickerModal
 * when clicked. Optionally supports creating new projects.
 *
 * Used in:
 * - WorkflowCreationForm (with create new project option)
 * - WorkflowPickerModal (without create new project option)
 */

import { useCallback, useState } from 'react';
import { FolderTree, ChevronRight, ChevronDown } from 'lucide-react';
import { type Project } from '@/domains/projects';
import ProjectModal from '@/domains/projects/ProjectModal';
import { ProjectPickerModal } from './ProjectPickerModal';

interface ProjectSelectorProps {
  /** Currently selected project */
  selectedProject: Project | null;
  /** Callback when a project is selected */
  onSelectProject: (project: Project) => void;
  /** Allow creating new projects (shows "Create New Project" option in picker) */
  allowCreate?: boolean;
  /** Placeholder text when no project is selected */
  placeholder?: string;
  /** Visual variant */
  variant?: 'button' | 'dropdown';
  /** Additional class names for the button */
  className?: string;
  /** Whether the selector is disabled */
  disabled?: boolean;
}

export function ProjectSelector({
  selectedProject,
  onSelectProject,
  allowCreate = false,
  placeholder = 'Select a project...',
  variant = 'button',
  className = '',
  disabled = false,
}: ProjectSelectorProps) {
  const [isPickerOpen, setIsPickerOpen] = useState(false);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  const handleOpenPicker = useCallback(() => {
    if (disabled) return;
    setIsPickerOpen(true);
  }, [disabled]);

  const handleClosePicker = useCallback(() => {
    setIsPickerOpen(false);
  }, []);

  const handleSelectProject = useCallback(
    (project: Project) => {
      onSelectProject(project);
      setIsPickerOpen(false);
    },
    [onSelectProject]
  );

  const handleCreateNew = useCallback(() => {
    setIsPickerOpen(false);
    setIsCreateModalOpen(true);
  }, []);

  const handleProjectCreated = useCallback(
    (project: Project) => {
      onSelectProject(project);
      setIsCreateModalOpen(false);
    },
    [onSelectProject]
  );

  const handleCloseCreateModal = useCallback(() => {
    setIsCreateModalOpen(false);
  }, []);

  // Determine icon based on variant
  const TrailingIcon = variant === 'dropdown' ? ChevronDown : ChevronRight;

  // Base button styles
  const baseStyles = variant === 'dropdown'
    ? 'w-full flex items-center justify-between gap-2 px-3 py-2 rounded-lg border border-gray-700 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left'
    : 'w-full flex items-center gap-3 px-3 py-2.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-left hover:bg-gray-50 dark:hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-colors';

  const disabledStyles = disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer';

  return (
    <>
      <button
        type="button"
        onClick={handleOpenPicker}
        disabled={disabled}
        className={`${baseStyles} ${disabledStyles} ${className}`}
      >
        {variant === 'dropdown' ? (
          // Dropdown variant (compact, for modals)
          <>
            <div className="flex items-center gap-2 min-w-0">
              <FolderTree size={16} className="text-gray-400 flex-shrink-0" />
              <span className="text-sm text-white truncate">
                {selectedProject?.name ?? placeholder}
              </span>
            </div>
            <TrailingIcon size={16} className="text-gray-400 flex-shrink-0" />
          </>
        ) : (
          // Button variant (more detail, for forms)
          <>
            <div
              className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${
                selectedProject
                  ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                  : 'bg-gray-100 dark:bg-gray-700 text-gray-400'
              }`}
            >
              <FolderTree size={16} />
            </div>
            <div className="flex-1 min-w-0">
              {selectedProject ? (
                <>
                  <div className="font-medium text-gray-900 dark:text-white truncate">
                    {selectedProject.name}
                  </div>
                  {selectedProject.description && (
                    <div className="text-xs text-gray-500 dark:text-gray-400 truncate">
                      {selectedProject.description}
                    </div>
                  )}
                </>
              ) : (
                <div className="text-gray-400 dark:text-gray-500">{placeholder}</div>
              )}
            </div>
            <TrailingIcon size={16} className="text-gray-400 flex-shrink-0" />
          </>
        )}
      </button>

      {/* Project Picker Modal */}
      <ProjectPickerModal
        isOpen={isPickerOpen}
        onClose={handleClosePicker}
        onSelect={handleSelectProject}
        onCreateNew={allowCreate ? handleCreateNew : undefined}
        selectedProject={selectedProject}
        hideCreateNew={!allowCreate}
      />

      {/* Project Creation Modal (only rendered when allowCreate is true) */}
      {allowCreate && (
        <ProjectModal
          isOpen={isCreateModalOpen}
          onClose={handleCloseCreateModal}
          onSuccess={handleProjectCreated}
        />
      )}
    </>
  );
}
