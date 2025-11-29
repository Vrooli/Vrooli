/**
 * Navigation hook for the Browser Automation Studio.
 * Extracts navigation/routing logic from App.tsx for better separation of concerns.
 */
import { useCallback, useEffect, useState } from "react";
import { useProjectStore, Project } from "../stores/projectStore";
import { useWorkflowStore, Workflow } from "../stores/workflowStore";
import { logger } from "../utils/logger";
import toast from "react-hot-toast";

export type AppView = "dashboard" | "project-detail" | "project-workflow";

export interface NormalizedWorkflow extends Partial<Workflow> {
  id: string;
  name: string;
  folderPath: string;
  createdAt: Date;
  updatedAt: Date;
  projectId?: string;
}

interface RawWorkflow {
  id?: string;
  name?: string;
  folder_path?: string;
  folderPath?: string;
  created_at?: string;
  createdAt?: Date;
  updated_at?: string;
  updatedAt?: Date;
  project_id?: string;
  projectId?: string;
  [key: string]: unknown;
}

export interface NavigationState {
  currentView: AppView | null;
  currentProject: Project | null;
  selectedWorkflow: NormalizedWorkflow | null;
  selectedFolder: string;
}

export interface NavigationActions {
  navigateToDashboard: (replace?: boolean) => void;
  openProject: (project: Project, options?: { replace?: boolean }) => void;
  openWorkflow: (
    project: Project,
    workflowId: string | undefined,
    options?: { replace?: boolean; workflowData?: Record<string, unknown> }
  ) => Promise<void>;
  setSelectedFolder: (folder: string) => void;
}

/**
 * Transform raw workflow data (from API) to normalized format
 */
export const transformWorkflow = (
  workflow: RawWorkflow | null | undefined
): NormalizedWorkflow | null => {
  if (!workflow || !workflow.id || !workflow.name) return null;
  return {
    ...workflow,
    id: workflow.id,
    name: workflow.name,
    folderPath: workflow.folder_path ?? workflow.folderPath ?? "/",
    createdAt: workflow.created_at
      ? new Date(workflow.created_at)
      : workflow.createdAt
        ? new Date(workflow.createdAt)
        : new Date(),
    updatedAt: workflow.updated_at
      ? new Date(workflow.updated_at)
      : workflow.updatedAt
        ? new Date(workflow.updatedAt)
        : new Date(),
    projectId: workflow.project_id ?? workflow.projectId,
  };
};

/**
 * Safe wrapper for history API that handles sandboxed iframes
 */
const safeNavigate = (
  state: Record<string, unknown>,
  url: string,
  replace = false
) => {
  try {
    if (replace) {
      window.history.replaceState(state, "", url);
    } else {
      window.history.pushState(state, "", url);
    }
  } catch (error) {
    // Some embedded hosts sandbox history APIs; log but continue rendering.
    logger.warn(
      "Failed to update history state",
      { component: "useAppNavigation", action: "safeNavigate" },
      error
    );
  }
};

/**
 * Custom hook for managing app navigation state and actions.
 * Centralizes routing logic that was previously scattered in App.tsx.
 */
export function useAppNavigation(): NavigationState & NavigationActions {
  const [currentView, setCurrentView] = useState<AppView | null>(null);
  const [selectedWorkflow, setSelectedWorkflow] =
    useState<NormalizedWorkflow | null>(null);
  const [selectedFolder, setSelectedFolder] = useState<string>("/");

  const { currentProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow } = useWorkflowStore();

  const navigateToDashboard = useCallback(
    (replace = false) => {
      const url = "/";
      const state = { view: "dashboard" };
      safeNavigate(state, url, replace);
      setSelectedWorkflow(null);
      setSelectedFolder("/");
      setCurrentProject(null);
      setCurrentView("dashboard");
    },
    [setCurrentProject]
  );

  const openProject = useCallback(
    (project: Project, options?: { replace?: boolean }) => {
      if (!project) {
        navigateToDashboard(options?.replace ?? false);
        return;
      }

      const url = `/projects/${project.id}`;
      const state = { view: "project-detail", projectId: project.id };
      safeNavigate(state, url, options?.replace ?? false);

      setCurrentProject(project);
      setSelectedFolder(project.folder_path ?? "/");
      setSelectedWorkflow(null);
      setCurrentView("project-detail");
    },
    [navigateToDashboard, setCurrentProject]
  );

  const openWorkflow = useCallback(
    async (
      project: Project,
      workflowId: string | undefined,
      options?: { replace?: boolean; workflowData?: Record<string, unknown> }
    ) => {
      if (!project || !workflowId) {
        navigateToDashboard(options?.replace ?? false);
        return;
      }

      const url = `/projects/${project.id}/workflows/${workflowId}`;
      const state = {
        view: "project-workflow",
        projectId: project.id,
        workflowId,
      };
      safeNavigate(state, url, options?.replace ?? false);

      setCurrentProject(project);

      const initialWorkflow = options?.workflowData
        ? transformWorkflow(options.workflowData)
        : null;
      if (initialWorkflow) {
        setSelectedWorkflow(initialWorkflow);
        setSelectedFolder(
          initialWorkflow.folderPath || project.folder_path || "/"
        );
      } else {
        setSelectedWorkflow(null);
        setSelectedFolder(project.folder_path || "/");
      }

      // Set view immediately so WorkflowBuilder renders even if loading fails
      setCurrentView("project-workflow");

      try {
        if (options?.workflowData) {
          // When workflowData is provided (e.g., from createWorkflow), use it directly
          const normalized = transformWorkflow(options.workflowData);
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolder(
              normalized.folderPath || project.folder_path || "/"
            );
          }
          // Verify the store is populated (createWorkflow should have done this)
          const storeWorkflow = useWorkflowStore.getState().currentWorkflow;
          if (!storeWorkflow || storeWorkflow.id !== workflowId) {
            await loadWorkflow(workflowId);
          }
        } else {
          // No workflowData provided, load from API (e.g., direct URL navigation)
          await loadWorkflow(workflowId);
          const loadedWorkflow = useWorkflowStore.getState().currentWorkflow;
          if (!loadedWorkflow) {
            throw new Error("Workflow data not loaded");
          }
          const normalized = transformWorkflow(loadedWorkflow);
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolder(
              normalized.folderPath || project.folder_path || "/"
            );
          }
        }
      } catch (error) {
        logger.error(
          "Failed to load workflow",
          {
            component: "useAppNavigation",
            action: "openWorkflow",
            workflowId,
            projectId: project.id,
            errorMessage:
              error instanceof Error ? error.message : String(error),
          },
          error
        );
        toast.error(
          `Failed to load workflow: ${error instanceof Error ? error.message : "Unknown error"}`
        );
        openProject(project, { replace: options?.replace ?? false });
      }
    },
    [loadWorkflow, navigateToDashboard, openProject, setCurrentProject]
  );

  // Handle initial route resolution and popstate events
  useEffect(() => {
    const resolvePath = async (path: string, replace = false) => {
      const normalized = path.replace(/\/+/g, "/").replace(/\/$/, "") || "/";

      if (normalized === "/") {
        navigateToDashboard(replace);
        return;
      }

      const segments = normalized.split("/").filter(Boolean);

      if (segments[0] !== "projects" || !segments[1]) {
        navigateToDashboard(replace);
        return;
      }

      const projectId = segments[1];
      const projectState = useProjectStore.getState();
      let project: Project | null | undefined = projectState.projects.find(
        (p) => p.id === projectId
      );
      if (!project) {
        project = (await projectState.getProject(projectId)) as Project | null;
        if (project) {
          useProjectStore.setState((state) => ({
            projects: state.projects.some((p) => p.id === project!.id)
              ? state.projects
              : [project!, ...state.projects],
          }));
        }
      }

      if (!project) {
        navigateToDashboard(replace);
        return;
      }

      if (segments.length >= 4 && segments[2] === "workflows") {
        const workflowId = segments[3];
        await openWorkflow(project, workflowId, { replace });
        return;
      }

      openProject(project, { replace });
    };

    resolvePath(window.location.pathname, true).catch((error) => {
      logger.warn(
        "Failed to resolve initial route",
        { component: "useAppNavigation", action: "resolvePath" },
        error
      );
    });

    const popHandler = () => {
      resolvePath(window.location.pathname, true).catch((error) => {
        logger.warn(
          "Failed to resolve popstate route",
          { component: "useAppNavigation", action: "handlePopState" },
          error
        );
      });
    };

    window.addEventListener("popstate", popHandler);
    return () => window.removeEventListener("popstate", popHandler);
  }, [navigateToDashboard, openProject, openWorkflow]);

  return {
    currentView,
    currentProject,
    selectedWorkflow,
    selectedFolder,
    navigateToDashboard,
    openProject,
    openWorkflow,
    setSelectedFolder,
  };
}

export default useAppNavigation;
