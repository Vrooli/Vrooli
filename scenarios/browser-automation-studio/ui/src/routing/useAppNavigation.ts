/**
 * Navigation hook for the Vrooli Ascension.
 * Extracts navigation/routing logic from App.tsx for better separation of concerns.
 *
 * This is the SINGLE SOURCE OF TRUTH for app navigation.
 * App.tsx should use this hook instead of managing its own navigation state.
 */
import { useCallback, useEffect, useState } from "react";
import { useProjectStore, type Project } from "@/domains/projects";
import { useWorkflowStore, Workflow } from "../stores/workflowStore";
import { useDashboardStore, type RecentWorkflow } from "../stores/dashboardStore";
import { logger } from "../utils/logger";
import { getConfig } from "../config";
import toast from "react-hot-toast";

export type AppView =
  | "dashboard"
  | "project-detail"
  | "project-workflow"
  | "settings"
  | "all-workflows"
  | "all-executions"
  | "record-mode";

export type DashboardTab = "home" | "executions" | "exports" | "projects" | "schedules";

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
  dashboardTab: DashboardTab;
  recordingSessionId: string | null;
}

export interface DashboardNavigationOptions {
  replace?: boolean;
  tab?: DashboardTab;
}

export interface NavigationActions {
  // Core navigation
  navigateToDashboard: (options?: DashboardNavigationOptions | boolean) => void;
  openProject: (project: Project, options?: { replace?: boolean }) => void;
  openWorkflow: (
    project: Project,
    workflowId: string | undefined,
    options?: { replace?: boolean; workflowData?: Record<string, unknown> }
  ) => Promise<void>;

  // Additional views
  navigateToSettings: (replace?: boolean) => void;
  navigateToAllWorkflows: (replace?: boolean) => void;
  navigateToAllExecutions: (replace?: boolean) => void;
  navigateToRecordMode: (sessionId?: string | null, replace?: boolean) => void;
  closeRecordMode: () => Promise<void>;

  // State setters
  setSelectedFolder: (folder: string) => void;
  setDashboardTab: (tab: DashboardTab) => void;
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
 *
 * This is the SINGLE SOURCE OF TRUTH for navigation.
 */
export function useAppNavigation(): NavigationState & NavigationActions {
  const [currentView, setCurrentView] = useState<AppView | null>(null);
  const [selectedWorkflow, setSelectedWorkflow] =
    useState<NormalizedWorkflow | null>(null);
  const [selectedFolder, setSelectedFolderState] = useState<string>("/");
  const [dashboardTab, setDashboardTabState] = useState<DashboardTab>("home");
  const [recordingSessionId, setRecordingSessionId] = useState<string | null>(null);

  const { currentProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow } = useWorkflowStore();
  const { setLastEditedWorkflow } = useDashboardStore();

  const setSelectedFolder = useCallback((folder: string) => {
    setSelectedFolderState(folder);
  }, []);

  const setDashboardTab = useCallback((tab: DashboardTab) => {
    setDashboardTabState(tab);
  }, []);

  const navigateToDashboard = useCallback(
    (options?: DashboardNavigationOptions | boolean) => {
      const replace = typeof options === "boolean" ? options : options?.replace ?? false;
      const targetTab =
        typeof options === "object" && options?.tab ? options.tab : dashboardTab;
      const search = targetTab !== "home" ? `?tab=${targetTab}` : "";
      const url = `/${search}`;
      const state = { view: "dashboard", tab: targetTab };
      safeNavigate(state, url, replace);
      setSelectedWorkflow(null);
      setSelectedFolderState("/");
      setCurrentProject(null);
      setDashboardTabState(targetTab);
      setCurrentView("dashboard");
    },
    [dashboardTab, setCurrentProject]
  );

  const navigateToSettings = useCallback(
    (replace = false) => {
      const url = "/settings";
      const state = { view: "settings" };
      safeNavigate(state, url, replace);
      setCurrentView("settings");
    },
    []
  );

  const navigateToAllWorkflows = useCallback(
    (replace = false) => {
      const url = "/workflows";
      const state = { view: "all-workflows" };
      safeNavigate(state, url, replace);
      setCurrentView("all-workflows");
    },
    []
  );

  const navigateToAllExecutions = useCallback(
    (replace = false) => {
      const url = "/executions";
      const state = { view: "all-executions" };
      safeNavigate(state, url, replace);
      setCurrentView("all-executions");
    },
    []
  );

  const navigateToRecordMode = useCallback(
    (sessionId?: string | null, replace = false) => {
      const isNewSession = !sessionId;
      const url = isNewSession ? "/record/new" : `/record/${sessionId}`;
      const state = { view: "record-mode", sessionId: sessionId ?? null, isNew: isNewSession };
      safeNavigate(state, url, replace);
      setRecordingSessionId(sessionId ?? null);
      setCurrentView("record-mode");
    },
    []
  );

  const closeRecordMode = useCallback(async () => {
    if (recordingSessionId) {
      try {
        const config = await getConfig();
        await fetch(`${config.API_URL}/recordings/live/session/${recordingSessionId}/close`, {
          method: "POST",
        });
      } catch (error) {
        logger.warn("Failed to close recording session", { component: "useAppNavigation", action: "closeRecordMode" }, error);
      }
    }
    setRecordingSessionId(null);
    navigateToDashboard();
  }, [recordingSessionId, navigateToDashboard]);

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
        setSelectedFolderState(
          initialWorkflow.folderPath || project.folder_path || "/"
        );
      } else {
        setSelectedWorkflow(null);
        setSelectedFolderState(project.folder_path || "/");
      }

      // Set view immediately so WorkflowBuilder renders even if loading fails
      setCurrentView("project-workflow");

      // Update last edited workflow for "Continue Editing" feature
      const lastEdited: RecentWorkflow = {
        id: workflowId,
        name: (options?.workflowData?.name as string) ?? "Untitled",
        projectId: project.id,
        projectName: project.name,
        updatedAt: new Date(),
        folderPath:
          (options?.workflowData?.folder_path as string) ??
          (options?.workflowData?.folderPath as string) ??
          "/",
      };
      setLastEditedWorkflow(lastEdited);

      try {
        if (options?.workflowData) {
          // When workflowData is provided (e.g., from createWorkflow), use it directly
          const normalized = transformWorkflow(options.workflowData);
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolderState(
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
            setSelectedFolderState(
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
    [loadWorkflow, navigateToDashboard, openProject, setCurrentProject, setLastEditedWorkflow]
  );

  // Handle initial route resolution and popstate events
  useEffect(() => {
    const resolvePath = async (path: string, replace = false) => {
      const normalized = path.replace(/\/+/g, "/").replace(/\/$/, "") || "/";

      // Parse query string for dashboard tab
      const url = new URL(window.location.href);
      const tabParam = url.searchParams.get("tab") as DashboardTab | null;

      if (normalized === "/") {
        navigateToDashboard({ replace, tab: tabParam ?? "home" });
        return;
      }

      const segments = normalized.split("/").filter(Boolean);

      // Handle /schedules (legacy route, redirect to dashboard with schedules tab)
      if (segments[0] === "schedules") {
        navigateToDashboard({ replace, tab: "schedules" });
        return;
      }

      // Handle /settings
      if (segments[0] === "settings") {
        navigateToSettings(replace);
        return;
      }

      // Handle /workflows
      if (segments[0] === "workflows") {
        navigateToAllWorkflows(replace);
        return;
      }

      // Handle /executions
      if (segments[0] === "executions") {
        navigateToAllExecutions(replace);
        return;
      }

      // Handle /record/new or /record/:sessionId
      if (segments[0] === "record") {
        const sessionId = segments[1] === "new" ? null : segments[1] ?? null;
        navigateToRecordMode(sessionId, replace);
        return;
      }

      // Handle /projects/:projectId and /projects/:projectId/workflows/:workflowId
      if (segments[0] !== "projects" || !segments[1]) {
        navigateToDashboard({ replace });
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
        navigateToDashboard({ replace });
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
      // Fallback to dashboard if route resolution fails
      setCurrentView("dashboard");
    });

    const popHandler = () => {
      resolvePath(window.location.pathname, true).catch((error) => {
        logger.warn(
          "Failed to resolve popstate route",
          { component: "useAppNavigation", action: "handlePopState" },
          error
        );
        // Fallback to dashboard if route resolution fails
        setCurrentView("dashboard");
      });
    };

    window.addEventListener("popstate", popHandler);
    return () => window.removeEventListener("popstate", popHandler);
  }, [
    navigateToDashboard,
    navigateToSettings,
    navigateToAllWorkflows,
    navigateToAllExecutions,
    navigateToRecordMode,
    openProject,
    openWorkflow,
  ]);

  return {
    // State
    currentView,
    currentProject,
    selectedWorkflow,
    selectedFolder,
    dashboardTab,
    recordingSessionId,

    // Core navigation
    navigateToDashboard,
    openProject,
    openWorkflow,

    // Additional views
    navigateToSettings,
    navigateToAllWorkflows,
    navigateToAllExecutions,
    navigateToRecordMode,
    closeRecordMode,

    // State setters
    setSelectedFolder,
    setDashboardTab,
  };
}

export default useAppNavigation;
