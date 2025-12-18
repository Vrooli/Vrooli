import { useState, useEffect, useId, useMemo, useRef } from "react";
import { logger } from "@utils/logger";
import {
  X,
  FolderOpen,
  AlertCircle,
  Sparkles,
  FolderPlus,
  Settings2,
  ChevronDown,
  Check,
  Layers,
  FileBox,
  Workflow,
  Image,
} from "lucide-react";
import { useProjectStore, type Project } from "./store";
import { ResponsiveDialog } from "@shared/layout";
import { selectors } from "@constants/selectors";

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
  isOpen?: boolean;
  onClose: () => void;
  project?: Project | null; // null for create, project for edit
  onSuccess?: (project: Project) => void;
}

type PresetType = "recommended" | "empty" | "custom";

interface PresetCardProps {
  type: PresetType;
  selected: boolean;
  onSelect: () => void;
  disabled?: boolean;
}

function PresetCard({ type, selected, onSelect, disabled }: PresetCardProps) {
  const configs: Record<
    PresetType,
    {
      icon: React.ReactNode;
      title: string;
      description: string;
      badge?: string;
      folders?: { name: string; icon: React.ReactNode; description: string }[];
    }
  > = {
    recommended: {
      icon: <Sparkles size={20} />,
      title: "Standard",
      description: "Best for testing & automation",
      badge: "Recommended",
      folders: [
        {
          name: "actions",
          icon: <Layers size={12} />,
          description: "Reusable steps",
        },
        {
          name: "flows",
          icon: <Workflow size={12} />,
          description: "Multi-step sequences",
        },
        {
          name: "cases",
          icon: <FileBox size={12} />,
          description: "Test scenarios",
        },
        {
          name: "assets",
          icon: <Image size={12} />,
          description: "Screenshots & data",
        },
      ],
    },
    empty: {
      icon: <FolderPlus size={20} />,
      title: "Empty",
      description: "Start with a blank slate",
    },
    custom: {
      icon: <Settings2 size={20} />,
      title: "Custom",
      description: "Define your own structure",
    },
  };

  const config = configs[type];

  return (
    <button
      type="button"
      onClick={onSelect}
      disabled={disabled}
      className={`
        relative flex flex-col p-4 rounded-xl border-2 transition-all duration-200 text-left w-full
        ${
          selected
            ? "border-flow-accent bg-flow-accent/10 shadow-lg shadow-flow-accent/20"
            : "border-gray-700 bg-gray-800/50 hover:border-gray-600 hover:bg-gray-800"
        }
        ${disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}
      `}
    >
      {/* Selection indicator */}
      {selected && (
        <div className="absolute top-3 right-3 w-5 h-5 bg-flow-accent rounded-full flex items-center justify-center">
          <Check size={12} className="text-white" />
        </div>
      )}

      {/* Badge */}
      {config.badge && (
        <span className="absolute -top-2 left-3 px-2 py-0.5 bg-gradient-to-r from-amber-500 to-orange-500 text-white text-[10px] font-semibold rounded-full uppercase tracking-wide">
          {config.badge}
        </span>
      )}

      {/* Icon & Title */}
      <div className="flex items-center gap-3 mb-2">
        <div
          className={`w-10 h-10 rounded-lg flex items-center justify-center ${
            selected
              ? "bg-flow-accent/20 text-flow-accent"
              : "bg-gray-700 text-gray-400"
          }`}
        >
          {config.icon}
        </div>
        <div>
          <h3
            className={`font-semibold ${selected ? "text-white" : "text-gray-200"}`}
          >
            {config.title}
          </h3>
          <p className="text-xs text-gray-500">{config.description}</p>
        </div>
      </div>

      {/* Folder preview for recommended */}
      {type === "recommended" && config.folders && (
        <div className="mt-3 pt-3 border-t border-gray-700/50">
          <div className="grid grid-cols-2 gap-2">
            {config.folders.map((folder) => (
              <div
                key={folder.name}
                className="flex items-center gap-2 text-xs text-gray-400"
              >
                <span
                  className={`${selected ? "text-flow-accent/70" : "text-gray-500"}`}
                >
                  {folder.icon}
                </span>
                <span className="font-mono">{folder.name}/</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </button>
  );
}

function ProjectModal({
  isOpen = true,
  onClose,
  project,
  onSuccess,
}: ProjectModalProps) {
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
  const [showAdvanced, setShowAdvanced] = useState(false);
  const titleId = useId();
  const nameInputRef = useRef<HTMLInputElement>(null);

  const isEditing = !!project;
  const [preset, setPreset] = useState<PresetType>("recommended");
  const [presetPathsText, setPresetPathsText] = useState("");

  // Focus name input when modal opens
  useEffect(() => {
    if (isOpen && nameInputRef.current) {
      // Small delay to ensure modal animation completes
      const timer = setTimeout(() => {
        nameInputRef.current?.focus();
      }, 100);
      return () => clearTimeout(timer);
    }
  }, [isOpen]);

  useEffect(() => {
    if (project) {
      setFormData({
        name: project.name,
        description: project.description || "",
        folder_path: project.folder_path,
      });
      setIsFolderPathDirty(true);
      setShowAdvanced(true); // Show advanced when editing
    } else {
      setIsFolderPathDirty(false);
      setPreset("recommended");
      setPresetPathsText("");
      setShowAdvanced(false);
    }
  }, [project]);

  const parsePresetPaths = (
    raw: string,
  ): { ok: true; paths: string[] } | { ok: false; error: string } => {
    const normalized = raw.replace(/\r\n/g, "\n");
    const lines = normalized
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
    const paths = lines.map((line) =>
      trimLeadingSlash(trimTrailingSlash(normalizePath(line))),
    );
    for (const relPath of paths) {
      if (!relPath) {
        return { ok: false, error: "Preset paths contain empty entries." };
      }
      const parts = relPath.split("/");
      for (const part of parts) {
        const segment = part.trim();
        if (!segment || segment === "." || segment === "..") {
          return {
            ok: false,
            error: `Invalid preset path segment in "${relPath}".`,
          };
        }
      }
    }
    return { ok: true, paths };
  };

  const presetPreview = useMemo(() => {
    if (isEditing) {
      return [];
    }
    if (preset === "empty") {
      return [];
    }
    if (preset === "recommended") {
      return ["actions", "flows", "cases", "assets"];
    }
    const parsed = parsePresetPaths(presetPathsText);
    if (!parsed.ok) {
      return [];
    }
    return parsed.paths;
  }, [isEditing, preset, presetPathsText]);

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

    if (!isEditing && preset === "custom") {
      const parsed = parsePresetPaths(presetPathsText);
      if (!parsed.ok) {
        errors.preset_paths = parsed.error;
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
        onClose();
        await new Promise((resolve) => setTimeout(resolve, 100));
        onSuccess?.(updated);
      } else {
        const presetPaths =
          preset === "custom" ? parsePresetPaths(presetPathsText) : null;
        const newProject = await createProject({
          ...formData,
          preset,
          preset_paths:
            preset === "custom" && presetPaths && presetPaths.ok
              ? presetPaths.paths
              : undefined,
        });
        onClose();
        await new Promise((resolve) => setTimeout(resolve, 100));
        onSuccess?.(newProject);
      }
    } catch (err) {
      logger.error(
        "Failed to save project",
        { component: "ProjectModal", action: "handleSave" },
        err,
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <ResponsiveDialog
      isOpen={isOpen}
      onDismiss={onClose}
      ariaLabelledBy={titleId}
      size="default"
      overlayClassName="bg-black/70 backdrop-blur-sm"
      className="bg-gray-900 border border-gray-700/50 shadow-2xl rounded-2xl overflow-hidden"
    >
      <div data-testid={selectors.dialogs.project.root}>
        {/* Header */}
        <div className="relative px-6 pt-6 pb-4">
          {/* Close button */}
          <button
            data-testid={selectors.dialogs.base.closeButton}
            onClick={() => {
              logger.debug("ProjectModal X button clicked", {
                component: "ProjectModal",
              });
              onClose();
            }}
            className="absolute top-4 right-4 p-2 text-gray-500 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            aria-label="Close project modal"
          >
            <X size={18} />
          </button>

          {/* Icon & Title */}
          <div className="flex flex-col items-center text-center mb-2">
            <div className="w-14 h-14 bg-gradient-to-br from-flow-accent/30 to-purple-500/20 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-flow-accent/10">
              <FolderOpen size={28} className="text-flow-accent" />
            </div>
            <h2
              id={titleId}
              className="text-xl font-bold text-white tracking-tight"
            >
              {isEditing ? "Edit Project" : "Create New Project"}
            </h2>
            <p className="text-sm text-gray-400 mt-1">
              {isEditing
                ? "Update your project settings"
                : "Organize your automations in one place"}
            </p>
          </div>
        </div>

        {/* Form */}
        <form
          data-testid={selectors.dialogs.project.form}
          onSubmit={handleSubmit}
          className="px-6 pb-6"
        >
          {/* Error Message */}
          {error && (
            <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-xl flex items-center gap-3">
              <div className="w-8 h-8 bg-red-500/20 rounded-lg flex items-center justify-center flex-shrink-0">
                <AlertCircle size={16} className="text-red-400" />
              </div>
              <span className="text-red-300 text-sm">{error}</span>
            </div>
          )}

          {/* Project Name */}
          <div className="mb-5">
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Project Name
            </label>
            <input
              ref={nameInputRef}
              type="text"
              data-testid={selectors.dialogs.project.nameInput}
              value={formData.name}
              onChange={(e) =>
                setFormData((prev) => ({ ...prev, name: e.target.value }))
              }
              className={`w-full px-4 py-3 bg-gray-800/50 border-2 rounded-xl text-white placeholder-gray-500 focus:outline-none transition-colors ${
                validationErrors.name
                  ? "border-red-500/50 focus:border-red-500"
                  : "border-gray-700/50 focus:border-flow-accent"
              }`}
              placeholder="e.g., Visited Tracker Tests"
            />
            {validationErrors.name && (
              <p
                className="mt-2 text-red-400 text-xs flex items-center gap-1"
                data-testid={selectors.dialogs.project.nameError}
              >
                <AlertCircle size={12} />
                {validationErrors.name}
              </p>
            )}
          </div>

          {/* Preset Selection - Only for create mode */}
          {!isEditing && (
            <div className="mb-5">
              <label className="block text-sm font-medium text-gray-300 mb-3">
                Project Structure
              </label>
              <div className="grid grid-cols-1 gap-3">
                <PresetCard
                  type="recommended"
                  selected={preset === "recommended"}
                  onSelect={() => setPreset("recommended")}
                />
                <div className="grid grid-cols-2 gap-3">
                  <PresetCard
                    type="empty"
                    selected={preset === "empty"}
                    onSelect={() => setPreset("empty")}
                  />
                  <PresetCard
                    type="custom"
                    selected={preset === "custom"}
                    onSelect={() => setPreset("custom")}
                  />
                </div>
              </div>

              {/* Custom folders input */}
              {preset === "custom" && (
                <div className="mt-3">
                  <textarea
                    value={presetPathsText}
                    onChange={(e) => setPresetPathsText(e.target.value)}
                    rows={3}
                    className={`w-full px-4 py-3 bg-gray-800/50 border-2 rounded-xl text-white placeholder-gray-500 focus:outline-none font-mono text-sm resize-none transition-colors ${
                      validationErrors.preset_paths
                        ? "border-red-500/50 focus:border-red-500"
                        : "border-gray-700/50 focus:border-flow-accent"
                    }`}
                    placeholder={"tests\nscripts\ndata\nreports"}
                  />
                  {validationErrors.preset_paths ? (
                    <p className="mt-2 text-red-400 text-xs flex items-center gap-1">
                      <AlertCircle size={12} />
                      {validationErrors.preset_paths}
                    </p>
                  ) : (
                    <p className="mt-2 text-xs text-gray-500">
                      One folder name per line
                    </p>
                  )}
                </div>
              )}

              {/* Preview of what will be created */}
              {presetPreview.length > 0 && preset !== "recommended" && (
                <p className="mt-3 text-xs text-gray-500">
                  Will create:{" "}
                  <span className="text-gray-400 font-mono">
                    {presetPreview.map((p) => `${p}/`).join(" ")}
                  </span>
                </p>
              )}
            </div>
          )}

          {/* Advanced Options Accordion */}
          <div className="mb-5">
            <button
              type="button"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-2 text-sm text-gray-400 hover:text-gray-300 transition-colors"
            >
              <ChevronDown
                size={16}
                className={`transition-transform duration-200 ${showAdvanced ? "rotate-180" : ""}`}
              />
              <span>Advanced options</span>
            </button>

            {showAdvanced && (
              <div className="mt-4 space-y-4 pl-1">
                {/* Description */}
                <div>
                  <label className="block text-sm font-medium text-gray-400 mb-2">
                    Description{" "}
                    <span className="text-gray-600 font-normal">(optional)</span>
                  </label>
                  <textarea
                    data-testid={selectors.dialogs.project.descriptionInput}
                    value={formData.description}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        description: e.target.value,
                      }))
                    }
                    rows={2}
                    className="w-full px-4 py-3 bg-gray-800/50 border-2 border-gray-700/50 rounded-xl text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent resize-none transition-colors"
                    placeholder="Describe what this project is for..."
                  />
                </div>

                {/* Folder Path */}
                <div>
                  <label className="block text-sm font-medium text-gray-400 mb-2">
                    Storage Location
                  </label>
                  <input
                    type="text"
                    data-testid={selectors.dialogs.project.folderPathInput}
                    value={formData.folder_path}
                    onChange={(e) => {
                      setIsFolderPathDirty(true);
                      setFormData((prev) => ({
                        ...prev,
                        folder_path: e.target.value,
                      }));
                    }}
                    className={`w-full px-4 py-3 bg-gray-800/50 border-2 rounded-xl text-white placeholder-gray-500 focus:outline-none font-mono text-sm transition-colors ${
                      validationErrors.folder_path
                        ? "border-red-500/50 focus:border-red-500"
                        : "border-gray-700/50 focus:border-flow-accent"
                    }`}
                    placeholder="/path/to/project/folder"
                  />
                  {validationErrors.folder_path ? (
                    <p className="mt-2 text-red-400 text-xs flex items-center gap-1">
                      <AlertCircle size={12} />
                      {validationErrors.folder_path}
                    </p>
                  ) : (
                    <p className="mt-2 text-xs text-gray-500">
                      Auto-generated from project name. Edit to customize.
                    </p>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-2">
            <button
              type="button"
              data-testid={selectors.dialogs.project.cancelButton}
              onClick={() => {
                logger.debug("ProjectModal Cancel button clicked", {
                  component: "ProjectModal",
                });
                onClose();
              }}
              className="px-5 py-2.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-xl transition-colors"
              disabled={isSubmitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              data-testid={selectors.dialogs.project.submitButton}
              disabled={isSubmitting || isLoading}
              className="px-5 py-2.5 bg-gradient-to-r from-flow-accent to-blue-600 text-white font-medium rounded-xl hover:from-blue-500 hover:to-blue-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-flow-accent/20 hover:shadow-flow-accent/30"
            >
              {isSubmitting || isLoading ? (
                <span className="flex items-center gap-2">
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  {isEditing ? "Updating..." : "Creating..."}
                </span>
              ) : (
                <span className="flex items-center gap-2">
                  {isEditing ? "Update Project" : "Create Project"}
                  {!isEditing && <Sparkles size={14} />}
                </span>
              )}
            </button>
          </div>
        </form>
      </div>
    </ResponsiveDialog>
  );
}

export default ProjectModal;
