/**
 * ProjectPickerModal Component
 *
 * A modal for selecting an existing project. When "Create New Project" is clicked,
 * it delegates to the parent to open the full ProjectModal.
 */

import { useCallback, useId } from 'react';
import { X, FolderTree, FolderPlus, Check } from 'lucide-react';
import { useProjectStore, type Project } from '@/domains/projects';
import { ResponsiveDialog } from '@shared/layout';

interface ProjectPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelect: (project: Project) => void;
  onCreateNew: () => void;
  selectedProject: Project | null;
}

export function ProjectPickerModal({
  isOpen,
  onClose,
  onSelect,
  onCreateNew,
  selectedProject,
}: ProjectPickerModalProps) {
  const titleId = useId();
  const { projects, isLoading } = useProjectStore();

  const handleProjectClick = useCallback((project: Project) => {
    onSelect(project);
    onClose();
  }, [onSelect, onClose]);

  const handleCreateNew = useCallback(() => {
    onClose();
    onCreateNew();
  }, [onClose, onCreateNew]);

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          <button
            onClick={onClose}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close dialog"
          >
            <X size={18} />
          </button>

          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-flow-accent/30 to-purple-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-flow-accent/10">
              <FolderTree size={28} className="text-flow-accent" />
            </div>
            <h2 id={titleId} className="text-xl font-bold text-white tracking-tight">
              Select Project
            </h2>
            <p className="text-sm text-gray-400 mt-1">Choose where to save your workflow</p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6">
          <div className="space-y-3">
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="w-6 h-6 border-2 border-gray-400 border-t-transparent rounded-full animate-spin" />
              </div>
            ) : (
              <>
                <div className="grid gap-2 max-h-64 overflow-y-auto">
                  {projects.map((project) => (
                    <button
                      key={project.id}
                      type="button"
                      onClick={() => handleProjectClick(project)}
                      className={`
                        flex items-center gap-3 w-full p-3 rounded-xl border-2 text-left transition-all
                        ${selectedProject?.id === project.id
                          ? 'border-flow-accent bg-flow-accent/10'
                          : 'border-gray-700 bg-gray-800/50 hover:border-gray-600'}
                      `}
                    >
                      <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                        selectedProject?.id === project.id
                          ? 'bg-flow-accent/20 text-flow-accent'
                          : 'bg-gray-700 text-gray-400'
                      }`}>
                        <FolderTree size={16} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-white truncate">{project.name}</div>
                        {project.description && (
                          <div className="text-xs text-gray-500 truncate">
                            {project.description}
                          </div>
                        )}
                      </div>
                      {selectedProject?.id === project.id && (
                        <div className="w-5 h-5 bg-flow-accent rounded-full flex items-center justify-center">
                          <Check size={12} className="text-white" />
                        </div>
                      )}
                    </button>
                  ))}
                </div>

                {/* Create New Button */}
                <button
                  type="button"
                  onClick={handleCreateNew}
                  className="flex items-center gap-3 w-full p-3 rounded-xl border-2 border-dashed border-gray-600 bg-gray-800/30 hover:border-gray-500 hover:bg-gray-800/50 transition-all text-left"
                >
                  <div className="w-8 h-8 rounded-lg flex items-center justify-center bg-green-500/20 text-green-400">
                    <FolderPlus size={16} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium text-white">Create New Project</div>
                    <div className="text-xs text-gray-500">
                      Start fresh with a new project
                    </div>
                  </div>
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
