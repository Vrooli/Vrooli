import { useState, useCallback, useId, useEffect } from "react";
import {
  X,
  Circle,
  Sparkles,
  FolderTree,
  FolderPlus,
  ChevronRight,
  ChevronLeft,
  Loader,
} from "lucide-react";
import { useProjectStore, type Project, buildProjectFolderPath } from "@/domains/projects";
import { ResponsiveDialog } from "@shared/layout";

export type WorkflowCreationType = "record" | "ai" | "visual";

interface WorkflowCreationDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectType: (type: WorkflowCreationType, project: Project) => void;
  onCreateProject: () => void;
  /** When provided, skips project selection and uses this project directly */
  preSelectedProject?: Project | null;
}

interface ProjectSelectorProps {
  projects: Project[];
  selectedProject: Project | null;
  onSelect: (project: Project) => void;
  onCreateNew: () => void;
}

function ProjectSelector({
  projects,
  selectedProject,
  onSelect,
  onCreateNew,
}: ProjectSelectorProps) {
  return (
    <div className="space-y-3">
      <div className="text-sm text-gray-400 mb-4">
        Select a project for your new workflow, or create a new one.
      </div>
      <div className="grid gap-2 max-h-64 overflow-y-auto">
        {projects.map((project) => (
          <button
            key={project.id}
            type="button"
            onClick={() => onSelect(project)}
            className={`
              flex items-center gap-3 w-full p-3 rounded-xl border-2 text-left transition-all
              ${
                selectedProject?.id === project.id
                  ? "border-flow-accent bg-flow-accent/10"
                  : "border-gray-700 bg-gray-800/50 hover:border-gray-600"
              }
            `}
          >
            <div
              className={`w-8 h-8 rounded-lg flex items-center justify-center ${
                selectedProject?.id === project.id
                  ? "bg-flow-accent/20 text-flow-accent"
                  : "bg-gray-700 text-gray-400"
              }`}
            >
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
                <ChevronRight size={12} className="text-white" />
              </div>
            )}
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={onCreateNew}
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
    </div>
  );
}

interface WorkflowTypeSelectorProps {
  onSelect: (type: WorkflowCreationType) => void;
}

function WorkflowTypeSelector({ onSelect }: WorkflowTypeSelectorProps) {
  return (
    <div className="space-y-3">
      <div className="text-sm text-gray-400 mb-4">
        Choose how you want to create your workflow.
      </div>
      <div className="space-y-3">
        {/* Record - Primary method */}
        <button
          type="button"
          onClick={() => onSelect("record")}
          className="flex items-start gap-4 w-full p-4 rounded-xl border-2 border-red-500/30 bg-gray-800/50 hover:border-red-500/60 hover:bg-gray-800 transition-all text-left group"
        >
          <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-red-500/20 text-red-400 group-hover:bg-red-500/30 transition-colors flex-shrink-0">
            <Circle size={20} className="fill-red-400" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="font-semibold text-white mb-1">Record</div>
            <div className="text-sm text-gray-400">
              Record your browser actions and convert them into a reusable workflow.
            </div>
          </div>
        </button>

        {/* AI-Assisted */}
        <button
          type="button"
          onClick={() => onSelect("ai")}
          className="flex items-start gap-4 w-full p-4 rounded-xl border-2 border-gray-700 bg-gray-800/50 hover:border-purple-500/60 hover:bg-gray-800 transition-all text-left group"
        >
          <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-purple-500/20 text-purple-400 group-hover:bg-purple-500/30 transition-colors flex-shrink-0">
            <Sparkles size={20} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="font-semibold text-white mb-1">AI-Assisted</div>
            <div className="text-sm text-gray-400">
              Describe your automation in plain language and let AI generate the workflow.
            </div>
          </div>
        </button>

        {/* Visual Builder */}
        <button
          type="button"
          onClick={() => onSelect("visual")}
          className="flex items-start gap-4 w-full p-4 rounded-xl border-2 border-gray-700 bg-gray-800/50 hover:border-blue-500/60 hover:bg-gray-800 transition-all text-left group"
        >
          <div className="w-10 h-10 rounded-lg flex items-center justify-center bg-blue-500/20 text-blue-400 group-hover:bg-blue-500/30 transition-colors flex-shrink-0">
            <FolderTree size={20} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="font-semibold text-white mb-1">Visual Builder</div>
            <div className="text-sm text-gray-400">
              Use the drag-and-drop interface to build workflows step by step.
            </div>
          </div>
        </button>
      </div>
    </div>
  );
}

export function WorkflowCreationDialog({
  isOpen,
  onClose,
  onSelectType,
  onCreateProject,
  preSelectedProject,
}: WorkflowCreationDialogProps) {
  const titleId = useId();
  const { projects, isLoading: isLoadingProjects } = useProjectStore();

  // Step: "project" (select project) or "type" (select workflow type)
  const [step, setStep] = useState<"project" | "type">("project");
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);

  // Reset state when dialog opens
  useEffect(() => {
    if (isOpen) {
      // If we have a pre-selected project, skip to type selection
      if (preSelectedProject) {
        setStep("type");
        setSelectedProject(preSelectedProject);
      } else {
        setStep(projects.length > 0 ? "project" : "type");
        setSelectedProject(null);
      }
    }
  }, [isOpen, projects.length, preSelectedProject]);

  const handleProjectSelect = useCallback((project: Project) => {
    setSelectedProject(project);
    setStep("type");
  }, []);

  const handleCreateNewProject = useCallback(() => {
    onClose();
    onCreateProject();
  }, [onClose, onCreateProject]);

  const handleTypeSelect = useCallback(
    async (type: WorkflowCreationType) => {
      if (selectedProject) {
        // We have a project, proceed with workflow creation
        onSelectType(type, selectedProject);
        onClose();
      } else if (projects.length === 0) {
        // No projects exist - auto-create one with default name
        try {
          const newProject = await useProjectStore.getState().createProject({
            name: "My Automations",
            description: "Automated browser workflows",
            folder_path: buildProjectFolderPath("my-automations"),
          });
          onSelectType(type, newProject);
          onClose();
        } catch (error) {
          console.error("Failed to create project:", error);
        }
      }
    },
    [selectedProject, projects.length, onSelectType, onClose]
  );

  const handleBack = useCallback(() => {
    setStep("project");
    setSelectedProject(null);
  }, []);

  if (!isOpen) return null;

  const stepTitle =
    step === "project" ? "Select Project" : "Create New Workflow";
  const stepSubtitle =
    step === "project"
      ? "Choose where to save your workflow"
      : selectedProject
        ? `in ${selectedProject.name}`
        : "Choose how to build your automation";

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
              {step === "project" ? (
                <FolderTree size={28} className="text-flow-accent" />
              ) : (
                <Sparkles size={28} className="text-flow-accent" />
              )}
            </div>
            <h2
              id={titleId}
              className="text-xl font-bold text-white tracking-tight"
            >
              {stepTitle}
            </h2>
            <p className="text-sm text-gray-400 mt-1">{stepSubtitle}</p>
          </div>
        </div>

        {/* Content */}
        <div className="px-6 pb-6">
          {isLoadingProjects ? (
            <div className="flex items-center justify-center py-8">
              <Loader size={24} className="animate-spin text-gray-400" />
            </div>
          ) : step === "project" && projects.length > 0 ? (
            <ProjectSelector
              projects={projects}
              selectedProject={selectedProject}
              onSelect={handleProjectSelect}
              onCreateNew={handleCreateNewProject}
            />
          ) : (
            <WorkflowTypeSelector onSelect={handleTypeSelect} />
          )}
        </div>

        {/* Footer with back button - only show when user can go back to project selection */}
        {step === "type" && projects.length > 0 && !preSelectedProject && (
          <div className="px-6 pb-6 pt-2 border-t border-gray-800">
            <button
              onClick={handleBack}
              className="flex items-center gap-2 text-sm text-gray-400 hover:text-white transition-colors"
            >
              <ChevronLeft size={16} />
              Back to project selection
            </button>
          </div>
        )}
      </div>
    </ResponsiveDialog>
  );
}

export default WorkflowCreationDialog;
