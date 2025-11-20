import { useState, useEffect, useId, useMemo, useRef } from "react";
import { logger } from "../utils/logger";
import { X, FolderOpen, AlertCircle } from "lucide-react";
import { useProjectStore, Project } from "../stores/projectStore";
import ResponsiveDialog from "./ResponsiveDialog";
import { selectors } from "../consts/selectors";

const normalizePath = (value: string) => value.replace(/\\/g, "/");
const trimTrailingSlash = (value: string) => value.replace(/\/+$/, "");
const trimLeadingSlash = (value: string) => value.replace(/^\/+/, "");
const parentDirectory = (value: string) => {
  const normalized = trimTrailingSlash(normalizePath(value));
  const idx = normalized.lastIndexOf("/");
  return idx > 0 ? normalized.slice(0, idx) : normalized;
};
const slugifySegment = (value: string) =>
  value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
const deriveWorkspaceRoot = (folderPath?: string) => {
  if (!folderPath) {
    return "";
  }
  const normalized = normalizePath(folderPath);
  const marker = "/scenarios/";
  const idx = normalized.indexOf(marker);
  if (idx > 0) {
    return normalized.slice(0, idx);
  }
  return parentDirectory(normalized);
};
const deriveProjectsHomeDir = (folderPath?: string) => {
  if (!folderPath) {
    return "";
  }
  return parentDirectory(folderPath);
};
const buildAutoFolderPath = (
  baseDir: string,
  projectName: string,
  fallbackSegment: string,
) => {
  if (!baseDir) {
    return "";
  }
  const base = trimTrailingSlash(normalizePath(baseDir));
  const slug = slugifySegment(projectName);
  const segment = slug || fallbackSegment;
  return `${base}/${trimLeadingSlash(segment)}`;
};

interface ProjectModalProps {
  onClose: () => void;
  project?: Project | null; // null for create, project for edit
  onSuccess?: (project: Project) => void;
}

function ProjectModal({ onClose, project, onSuccess }: ProjectModalProps) {
  const { createProject, updateProject, isLoading, error, projects } =
    useProjectStore();
  const detectedPathSource = useMemo(() => {
    if (project?.folder_path) {
      return project.folder_path;
    }
    const candidate = projects.find((entry) => Boolean(entry.folder_path));
    return candidate?.folder_path ?? "";
  }, [project, projects]);
  const workspaceRoot = useMemo(
    () => deriveWorkspaceRoot(detectedPathSource),
    [detectedPathSource],
  );
  const projectsHomeDir = useMemo(() => {
    const home = deriveProjectsHomeDir(
      project?.folder_path ?? detectedPathSource,
    );
    if (home) {
      return home;
    }
    return workspaceRoot
      ? `${trimTrailingSlash(normalizePath(workspaceRoot))}/scenarios/browser-automation-studio/data/projects`
      : "";
  }, [project, detectedPathSource, workspaceRoot]);
  const generatedFolderSuffixRef = useRef(`project-${Date.now().toString(36)}`);
  const [formData, setFormData] = useState({
    name: "",
    description: "",
    folder_path: "",
  });
  const [validationErrors, setValidationErrors] = useState<
    Record<string, string>
  >({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isFolderPathDirty, setIsFolderPathDirty] = useState(false);
  const titleId = useId();

  const isEditing = !!project;

  useEffect(() => {
    if (project) {
      setFormData({
        name: project.name,
        description: project.description || "",
        folder_path: project.folder_path,
      });
      setIsFolderPathDirty(true);
    } else {
      setIsFolderPathDirty(false);
    }
  }, [project]);

  useEffect(() => {
    if (project) {
      return;
    }
    if (!projectsHomeDir) {
      return;
    }
    if (isFolderPathDirty) {
      return;
    }
    const nextSegment = formData.name.trim()
      ? slugifySegment(formData.name) || generatedFolderSuffixRef.current
      : generatedFolderSuffixRef.current;
    const candidate = buildAutoFolderPath(
      projectsHomeDir,
      nextSegment,
      generatedFolderSuffixRef.current,
    );
    if (candidate && candidate !== formData.folder_path) {
      setFormData((prev) => ({ ...prev, folder_path: candidate }));
    }
  }, [
    project,
    projectsHomeDir,
    formData.name,
    isFolderPathDirty,
    formData.folder_path,
  ]);

  const validateForm = () => {
    const errors: Record<string, string> = {};

    if (!formData.name.trim()) {
      errors.name = "Project name is required";
    } else if (formData.name.length < 3) {
      errors.name = "Project name must be at least 3 characters";
    } else if (formData.name.length > 100) {
      errors.name = "Project name must not exceed 100 characters";
    }

    if (!formData.folder_path.trim()) {
      errors.folder_path = "Folder path is required";
    } else if (!formData.folder_path.startsWith("/")) {
      errors.folder_path =
        "Folder path must be an absolute path starting with /";
    } else if (workspaceRoot) {
      const normalizedTarget = normalizePath(formData.folder_path);
      const normalizedRoot = trimTrailingSlash(normalizePath(workspaceRoot));
      if (!normalizedTarget.startsWith(normalizedRoot)) {
        errors.folder_path = `Folder path must stay inside ${normalizedRoot}`;
      }
    }

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      if (isEditing && project) {
        const updated = await updateProject(project.id, formData);
        onClose(); // Close modal before navigation
        // Small delay to ensure modal unmounts before navigation
        await new Promise((resolve) => setTimeout(resolve, 100));
        onSuccess?.(updated);
      } else {
        const newProject = await createProject(formData);
        onClose(); // Close modal before navigation
        // Small delay to ensure modal unmounts before navigation
        await new Promise((resolve) => setTimeout(resolve, 100));
        onSuccess?.(newProject);
      }
    } catch (error) {
      logger.error(
        "Failed to save project",
        { component: "ProjectModal", action: "handleSave" },
        error,
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const suggestedPaths = useMemo(() => {
    if (!projectsHomeDir) {
      return ["/tmp/browser-automation-studio/projects"];
    }
    const suggestions = [
      buildAutoFolderPath(
        projectsHomeDir,
        formData.name,
        generatedFolderSuffixRef.current,
      ),
      `${trimTrailingSlash(projectsHomeDir)}/recordings`,
      `${trimTrailingSlash(projectsHomeDir)}/workflow-runs`,
      `${trimTrailingSlash(projectsHomeDir)}/exports`,
    ];
    return Array.from(new Set(suggestions.filter(Boolean)));
  }, [projectsHomeDir, formData.name]);

  return (
    <ResponsiveDialog
      isOpen
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      className="bg-flow-node border border-gray-700 shadow-2xl"
    >
      <div data-testid={selectors.dialogs.project.root}>
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-700">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-flow-accent/20 rounded-lg flex items-center justify-center">
            <FolderOpen size={16} className="text-flow-accent" />
          </div>
          <h2 id={titleId} className="text-xl font-bold text-white">
            {isEditing ? "Edit Project" : "Create New Project"}
          </h2>
        </div>
        <button
          data-testid={selectors.dialogs.base.closeButton}
          onClick={onClose}
          className="text-gray-400 hover:text-white transition-colors"
          aria-label="Close project modal"
        >
          <X size={20} />
        </button>
      </div>

      {/* Form */}
      <form
        data-testid={selectors.dialogs.project.form}
        onSubmit={handleSubmit}
        className="p-6"
      >
        {/* Error Message */}
        {error && (
          <div className="mb-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg flex items-center gap-2">
            <AlertCircle size={16} className="text-red-400" />
            <span className="text-red-400 text-sm">{error}</span>
          </div>
        )}

        {/* Project Name */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Project Name *
          </label>
          <input
            type="text"
            data-testid={selectors.dialogs.project.nameInput}
            value={formData.name}
            onChange={(e) =>
              setFormData((prev) => ({ ...prev, name: e.target.value }))
            }
            className={`w-full px-3 py-2 bg-flow-bg border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent ${
              validationErrors.name ? "border-red-500" : "border-gray-700"
            }`}
            placeholder="e.g., Visited Tracker Tests"
          />
          {validationErrors.name && (
            <p
              className="mt-1 text-red-400 text-xs"
              data-testid={selectors.dialogs.project.nameError}
            >
              {validationErrors.name}
            </p>
          )}
        </div>

        {/* Description */}
        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Description
          </label>
          <textarea
            data-testid={selectors.dialogs.project.descriptionInput}
            value={formData.description}
            onChange={(e) =>
              setFormData((prev) => ({ ...prev, description: e.target.value }))
            }
            rows={3}
            className="w-full px-3 py-2 bg-flow-bg border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent resize-none"
            placeholder="Describe what this project is for..."
          />
        </div>

        {/* Folder Path */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-300 mb-2">
            Folder Path *
          </label>
          <input
            type="text"
            value={formData.folder_path}
            onChange={(e) => {
              setIsFolderPathDirty(true);
              setFormData((prev) => ({ ...prev, folder_path: e.target.value }));
            }}
            className={`w-full px-3 py-2 bg-flow-bg border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent ${
              validationErrors.folder_path
                ? "border-red-500"
                : "border-gray-700"
            }`}
            placeholder="/path/to/project/folder"
          />
          {validationErrors.folder_path && (
            <p className="mt-1 text-red-400 text-xs">
              {validationErrors.folder_path}
            </p>
          )}
          {!validationErrors.folder_path && workspaceRoot && (
            <p className="mt-1 text-xs text-gray-500">
              Detected workspace root: {workspaceRoot}
            </p>
          )}

          {/* Suggested Paths */}
          <div className="mt-2">
            <p className="text-xs text-gray-500 mb-1">Suggested paths:</p>
            <div className="flex flex-wrap gap-1">
              {suggestedPaths.map((path) => (
                <button
                  key={path}
                  type="button"
                  onClick={() => {
                    setIsFolderPathDirty(true);
                    setFormData((prev) => ({ ...prev, folder_path: path }));
                  }}
                  className="px-2 py-1 bg-gray-700 hover:bg-gray-600 text-xs text-gray-300 rounded transition-colors"
                >
                  {path}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <button
            type="button"
            data-testid={selectors.dialogs.project.cancelButton}
            onClick={onClose}
            className="px-4 py-2 text-gray-400 hover:text-white transition-colors"
            disabled={isSubmitting}
          >
            Cancel
          </button>
          <button
            type="submit"
            data-testid={selectors.dialogs.project.submitButton}
            disabled={isSubmitting || isLoading}
            className="px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting || isLoading ? (
              <span className="flex items-center gap-2">
                <div className="w-4 h-4 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                {isEditing ? "Updating..." : "Creating..."}
              </span>
            ) : isEditing ? (
              "Update Project"
            ) : (
              "Create Project"
            )}
          </button>
        </div>
      </form>
      </div>
    </ResponsiveDialog>
  );
}

export default ProjectModal;
