import { useCallback, useEffect, useState, lazy, Suspense, useMemo } from "react";
import { ReactFlowProvider } from "reactflow";
import "reactflow/dist/style.css";

// Shared layout components
import { ResponsiveDialog, Sidebar, Header, KeyboardShortcutsModal } from "@shared/layout";
import { ProjectModal } from "@features/projects";
import { GuidedTour, useGuidedTour } from "@features/onboarding";
import { DocsModal } from "@features/docs";

// Keyboard shortcuts - new centralized system
import {
  useKeyboardShortcutHandler,
  useRegisterShortcuts,
  useShortcutContext,
} from "@hooks/useKeyboardShortcuts";
import { type ShortcutContext } from "@stores/keyboardShortcutsStore";

// Lazy load heavy components for better initial load performance
const WorkflowBuilder = lazy(() => import("@features/workflows/builder/WorkflowBuilder"));
const ExecutionViewer = lazy(() => import("@features/execution/ExecutionViewer"));
const AIPromptModal = lazy(() => import("@features/ai/AIPromptModal"));
const Dashboard = lazy(() => import("@features/projects/Dashboard"));
const ProjectDetail = lazy(() => import("@features/projects/ProjectDetail"));
const SettingsPage = lazy(() => import("@features/settings/SettingsPage"));
const GlobalWorkflowsView = lazy(() => import("@features/dashboard/GlobalWorkflowsView").then(m => ({ default: m.GlobalWorkflowsView })));
const GlobalExecutionsView = lazy(() => import("@features/dashboard/GlobalExecutionsView").then(m => ({ default: m.GlobalExecutionsView })));

// Stores and hooks
import { useExecutionStore } from "@stores/executionStore";
import { useProjectStore, type Project } from "@stores/projectStore";
import { useWorkflowStore, type Workflow } from "@stores/workflowStore";
import { useScenarioStore } from "@stores/scenarioStore";
import { useDashboardStore, type RecentWorkflow } from "@stores/dashboardStore";
import { useMediaQuery } from "@hooks/useMediaQuery";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import { selectors } from "@constants/selectors";
import { LoadingSpinner } from "@shared/ui";

interface NormalizedWorkflow extends Partial<Workflow> {
  id: string;
  name: string;
  folderPath: string;
  createdAt: Date;
  updatedAt: Date;
  projectId?: string;
}

type AppView = "dashboard" | "project-detail" | "project-workflow" | "settings" | "all-workflows" | "all-executions";

const EXECUTION_MIN_WIDTH = 360;
const EXECUTION_MAX_WIDTH = 720;
const EXECUTION_DEFAULT_WIDTH = 440;

function App() {
  const [currentView, setCurrentView] = useState<AppView | null>(null);
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [showKeyboardShortcuts, setShowKeyboardShortcuts] = useState(false);
  const [showDocs, setShowDocs] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string>("/");
  const [selectedWorkflow, setSelectedWorkflow] =
    useState<NormalizedWorkflow | null>(null);
  const [executionPaneWidth, setExecutionPaneWidth] = useState(
    EXECUTION_DEFAULT_WIDTH,
  );
  const [isResizingExecution, setIsResizingExecution] = useState(false);

  const { projects, currentProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow } = useWorkflowStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const viewerWorkflowId = useExecutionStore((state) => state.viewerWorkflowId);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const { fetchScenarios } = useScenarioStore();
  const { setLastEditedWorkflow } = useDashboardStore();
  const isLargeScreen = useMediaQuery("(min-width: 1024px)");

  // Guided tour state
  const { showTour, openTour, closeTour } = useGuidedTour();

  // Debug logging for modal state
  useEffect(() => {
    console.log("[DEBUG] App: showProjectModal changed to:", showProjectModal);
  }, [showProjectModal]);

  const clampExecutionWidth = useCallback(
    (value: number) =>
      Math.min(EXECUTION_MAX_WIDTH, Math.max(EXECUTION_MIN_WIDTH, value)),
    [],
  );

  const handleExecutionResizeMouseDown = useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      if (!isLargeScreen) {
        return;
      }

      event.preventDefault();
      event.stopPropagation();

      const startX = event.clientX;
      const startWidth = executionPaneWidth;

      setIsResizingExecution(true);
      document.body.style.userSelect = "none";
      document.body.style.cursor = "col-resize";

      const onMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const nextWidth = clampExecutionWidth(startWidth - deltaX);
        setExecutionPaneWidth(nextWidth);
      };

      const onMouseUp = () => {
        setIsResizingExecution(false);
        document.body.style.userSelect = "";
        document.body.style.cursor = "";
        window.removeEventListener("mousemove", onMouseMove);
        window.removeEventListener("mouseup", onMouseUp);
      };

      window.addEventListener("mousemove", onMouseMove);
      window.addEventListener("mouseup", onMouseUp);
    },
    [clampExecutionWidth, executionPaneWidth, isLargeScreen],
  );

  // Pre-fetch scenarios on app mount for faster loading in NavigateNode
  useEffect(() => {
    void fetchScenarios();
  }, [fetchScenarios]);

  const isExecutionViewerOpen = Boolean(viewerWorkflowId);
  const activeExecutionWorkflowId =
    viewerWorkflowId ??
    currentExecution?.workflowId ??
    selectedWorkflow?.id ??
    null;

  useEffect(() => {
    if (!isExecutionViewerOpen) {
      setIsResizingExecution(false);
      document.body.style.userSelect = "";
      document.body.style.cursor = "";
    }
  }, [isExecutionViewerOpen]);

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

  const transformWorkflow = useCallback(
    (workflow: RawWorkflow | null | undefined): NormalizedWorkflow | null => {
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
    },
    [],
  );

  const safeNavigate = useCallback(
    (state: Record<string, unknown>, url: string, replace = false) => {
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
          { component: "App", action: "safeNavigate" },
          error,
        );
      }
    },
    [],
  );

  const navigateToDashboard = useCallback(
    (replace = false) => {
      const url = "/";
      const state = { view: "dashboard" };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setSelectedWorkflow(null);
      setSelectedFolder("/");
      setCurrentProject(null);
      setCurrentView("dashboard");
    },
    [safeNavigate, setCurrentProject],
  );

  const navigateToSettings = useCallback(
    (replace = false) => {
      const url = "/settings";
      const state = { view: "settings" };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setCurrentView("settings");
    },
    [safeNavigate],
  );

  const navigateToAllWorkflows = useCallback(
    (replace = false) => {
      const url = "/workflows";
      const state = { view: "all-workflows" };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setCurrentView("all-workflows");
    },
    [safeNavigate],
  );

  const navigateToAllExecutions = useCallback(
    (replace = false) => {
      const url = "/executions";
      const state = { view: "all-executions" };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setCurrentView("all-executions");
    },
    [safeNavigate],
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

      setShowAIModal(false);
      setShowProjectModal(false);
      setCurrentProject(project);
      setSelectedFolder(project.folder_path ?? "/");
      setSelectedWorkflow(null);
      setCurrentView("project-detail");
    },
    [navigateToDashboard, safeNavigate, setCurrentProject],
  );

  const openWorkflow = useCallback(
    async (
      project: Project,
      workflowId: string | undefined,
      options?: { replace?: boolean; workflowData?: Record<string, unknown> },
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

      setShowAIModal(false);
      setShowProjectModal(false);
      setCurrentProject(project);

      const initialWorkflow = options?.workflowData
        ? transformWorkflow(options.workflowData)
        : null;
      if (initialWorkflow) {
        setSelectedWorkflow(initialWorkflow);
        setSelectedFolder(
          initialWorkflow.folderPath || project.folder_path || "/",
        );
      } else {
        setSelectedWorkflow(null);
        setSelectedFolder(project.folder_path || "/");
      }

      // Set view immediately so WorkflowBuilder renders even if loading fails
      setCurrentView("project-workflow");

      // Update last edited workflow for "Continue Editing" feature
      const lastEdited: RecentWorkflow = {
        id: workflowId,
        name: options?.workflowData?.name as string ?? 'Untitled',
        projectId: project.id,
        projectName: project.name,
        updatedAt: new Date(),
        folderPath: options?.workflowData?.folder_path as string ?? options?.workflowData?.folderPath as string ?? '/',
      };
      setLastEditedWorkflow(lastEdited);

      try {
        console.log("[DEBUG] openWorkflow called", {
          workflowId,
          projectId: project.id,
          hasWorkflowData: !!options?.workflowData,
        });

        if (options?.workflowData) {
          console.log("[DEBUG] Using provided workflowData", { workflowId });
          // When workflowData is provided (e.g., from createWorkflow), use it directly
          // to avoid redundant API call that could fail and cause navigation away
          const normalized = transformWorkflow(options.workflowData);
          console.log("[DEBUG] Normalized workflow from workflowData", {
            hasNormalized: !!normalized,
            normalizedId: normalized?.id,
          });
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolder(
              normalized.folderPath || project.folder_path || "/",
            );
          }
          // Verify the store is populated (createWorkflow should have done this)
          const storeWorkflow = useWorkflowStore.getState().currentWorkflow;
          console.log("[DEBUG] Checking workflow store", {
            hasStoreWorkflow: !!storeWorkflow,
            storeWorkflowId: storeWorkflow?.id,
            expectedWorkflowId: workflowId,
          });
          if (!storeWorkflow || storeWorkflow.id !== workflowId) {
            console.warn("[DEBUG] Store not populated, loading from API", { workflowId });
            await loadWorkflow(workflowId);
          } else {
            console.log("[DEBUG] Store already populated, skipping API call", { workflowId });
          }
        } else {
          console.log("[DEBUG] No workflowData, loading from API", { workflowId });
          // No workflowData provided, load from API (e.g., direct URL navigation)
          await loadWorkflow(workflowId);
          const loadedWorkflow = useWorkflowStore.getState().currentWorkflow;
          if (!loadedWorkflow) {
            throw new Error('Workflow data not loaded');
          }
          const normalized = transformWorkflow(loadedWorkflow);
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolder(
              normalized.folderPath || project.folder_path || "/",
            );
            // Update last edited workflow with actual name from API
            setLastEditedWorkflow({
              id: normalized.id,
              name: normalized.name,
              projectId: project.id,
              projectName: project.name,
              updatedAt: normalized.updatedAt,
              folderPath: normalized.folderPath,
            });
          }
        }
        console.log("[DEBUG] openWorkflow completed successfully", { workflowId });
      } catch (error) {
        console.error("[DEBUG] openWorkflow FAILED:", error);
        logger.error(
          "Failed to load workflow",
          {
            component: "App",
            action: "openWorkflow",
            workflowId,
            projectId: project.id,
            errorMessage: error instanceof Error ? error.message : String(error),
          },
          error,
        );
        toast.error(`Failed to load workflow: ${error instanceof Error ? error.message : 'Unknown error'}`);
        console.warn("[DEBUG] Navigating back to project due to error", { workflowId });
        openProject(project, { replace: options?.replace ?? false });
      }
    },
    [
      loadWorkflow,
      navigateToDashboard,
      openProject,
      safeNavigate,
      setCurrentProject,
      transformWorkflow,
    ],
  );

  const handleProjectSelect = (project: Project) => {
    openProject(project);
  };

  const handleWorkflowSelect = async (workflow: Workflow) => {
    if (!currentProject) return;
    await openWorkflow(currentProject, workflow.id, {
      workflowData: workflow as Record<string, unknown>,
    });
  };

  // Handler for navigating directly to a workflow from dashboard widgets
  const handleNavigateToWorkflow = useCallback(async (projectId: string, workflowId: string) => {
    const projectState = useProjectStore.getState();
    let project: Project | undefined = projectState.projects.find(p => p.id === projectId);
    if (!project) {
      const fetchedProject = await projectState.getProject(projectId);
      project = fetchedProject ?? undefined;
    }
    if (project) {
      await openWorkflow(project, workflowId);
    }
  }, [openWorkflow]);

  // Handler for viewing an execution from dashboard widgets
  const handleViewExecution = useCallback(async (executionId: string, workflowId: string) => {
    // Load the execution details
    await loadExecution(executionId);

    // Find the workflow's project and navigate to the workflow view
    const workflowsResponse = await fetch(`/api/v1/workflows/${workflowId}`);
    if (workflowsResponse.ok) {
      const workflowData = await workflowsResponse.json();
      const projectId = workflowData.project_id ?? workflowData.projectId;
      if (projectId) {
        await handleNavigateToWorkflow(projectId, workflowId);
      }
    }
  }, [loadExecution, handleNavigateToWorkflow]);

  // Handler for AI workflow generation from dashboard
  const handleDashboardAIGenerate = useCallback(async (prompt: string) => {
    // If no project exists, create one first
    if (projects.length === 0) {
      try {
        const project = await useProjectStore.getState().createProject({
          name: 'My Automations',
          description: 'Automated workflows',
          folder_path: '/automations',
        });
        setCurrentProject(project);
        // Then open AI modal with the prompt
        setShowAIModal(true);
        // Store the prompt in session storage for the AI modal to use
        sessionStorage.setItem('pendingAIPrompt', prompt);
      } catch (error) {
        logger.error('Failed to create default project', { component: 'App', action: 'handleDashboardAIGenerate' }, error);
        toast.error('Failed to create project. Please try again.');
      }
    } else {
      // Use first project or current project
      const targetProject = currentProject ?? projects[0];
      setCurrentProject(targetProject);
      openProject(targetProject);
      // Store the prompt and open AI modal after navigation
      sessionStorage.setItem('pendingAIPrompt', prompt);
      setShowAIModal(true);
    }
  }, [projects, currentProject, setCurrentProject, openProject]);

  // Handler for running a workflow from dashboard
  const handleRunWorkflow = useCallback(async (workflowId: string) => {
    try {
      await startExecution(workflowId);
      toast.success('Workflow execution started');
    } catch (error) {
      logger.error('Failed to start workflow execution', { component: 'App', action: 'handleRunWorkflow', workflowId }, error);
      toast.error('Failed to start workflow');
    }
  }, [startExecution]);

  // Handler for trying the demo workflow from welcome state
  const handleTryDemo = useCallback(async () => {
    try {
      // Fetch the first workflow (demo workflow is seeded first)
      const response = await fetch('/api/v1/workflows?limit=1');
      if (!response.ok) {
        throw new Error('Failed to fetch workflows');
      }
      const data = await response.json();
      const workflows = data.workflows || [];

      if (workflows.length > 0) {
        const demoWorkflow = workflows[0];
        const projectId = demoWorkflow.project_id ?? demoWorkflow.projectId;
        const workflowId = demoWorkflow.id;

        if (projectId && workflowId) {
          await handleNavigateToWorkflow(projectId, workflowId);
          toast.success('Loading demo workflow...');
        } else {
          toast.error('Demo workflow not properly configured');
        }
      } else {
        toast.error('No demo workflow available. Create a project first!');
      }
    } catch (error) {
      logger.error('Failed to load demo workflow', { component: 'App', action: 'handleTryDemo' }, error);
      toast.error('Failed to load demo workflow');
    }
  }, [handleNavigateToWorkflow]);

  const handleBackToDashboard = () => {
    navigateToDashboard();
  };

  const handleBackToProjectDetail = () => {
    if (currentProject) {
      openProject(currentProject);
    } else {
      navigateToDashboard();
    }
  };

  const handleCreateProject = () => {
    setShowProjectModal(true);
  };

  const handleProjectCreated = (project: Project) => {
    // Show success toast
    toast.success(`Project "${project.name}" created successfully`);
    // Automatically select the newly created project
    openProject(project);
  };

  const handleCreateWorkflow = () => {
    setShowAIModal(true);
  };

  const handleCreateWorkflowDirect = async () => {
    if (!currentProject?.id) {
      toast.error("Select a project before creating a workflow.");
      return;
    }

    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || "/";

    try {
      const workflow = await useWorkflowStore
        .getState()
        .createWorkflow(workflowName, folderPath, currentProject.id);

      if (currentProject && workflow?.id) {
        await openWorkflow(currentProject, workflow.id, {
          workflowData: workflow as Record<string, unknown>,
        });
      }
    } catch (error) {
      logger.error(
        "Failed to create workflow",
        {
          component: "App",
          action: "handleCreateWorkflowDirect",
          projectId: currentProject?.id,
        },
        error,
      );
      toast.error("Failed to create workflow. Please try again.");
    }
  };

  const handleSwitchToManualBuilder = async () => {
    console.log("[DEBUG] handleSwitchToManualBuilder called", {
      hasCurrentProject: !!currentProject,
      currentProjectId: currentProject?.id,
    });

    if (!currentProject?.id) {
      console.error("[DEBUG] No current project, cannot switch to manual builder");
      toast.error("Select a project before using the manual builder.");
      return;
    }

    // Close modal immediately BEFORE async operation
    // This ensures the modal closes instantly for the user and tests don't fail
    // waiting for the async workflow creation to complete
    console.log("[DEBUG] Closing AI modal immediately");
    setShowAIModal(false);

    // Create a new empty workflow and switch to the manual builder
    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || "/";
    console.log("[DEBUG] Creating new workflow", { workflowName, folderPath });

    try {
      const workflow = await useWorkflowStore
        .getState()
        .createWorkflow(workflowName, folderPath, currentProject.id);
      console.log("[DEBUG] Workflow created", {
        hasWorkflow: !!workflow,
        workflowId: workflow?.id,
      });

      if (currentProject && workflow?.id) {
        console.log("[DEBUG] Calling openWorkflow with workflowData");
        await openWorkflow(currentProject, workflow.id, {
          workflowData: workflow as Record<string, unknown>,
        });
        console.log("[DEBUG] openWorkflow completed");
      } else {
        console.error("[DEBUG] Missing currentProject or workflow.id", {
          hasCurrentProject: !!currentProject,
          hasWorkflowId: !!workflow?.id,
        });
      }
    } catch (error) {
      console.error("[DEBUG] handleSwitchToManualBuilder FAILED:", error);
      logger.error(
        "Failed to create workflow",
        {
          component: "App",
          action: "handleCreateWorkflow",
          projectId: currentProject?.id,
        },
        error,
      );
      toast.error("Failed to open the manual builder. Please try again.");
    }
  };

  const handleWorkflowGenerated = async (workflow: Workflow) => {
    if (!workflow?.id || !currentProject) {
      return;
    }

    try {
      await openWorkflow(currentProject, workflow.id, {
        workflowData: workflow as Record<string, unknown>,
      });
    } catch (error) {
      logger.error(
        "Failed to open generated workflow",
        {
          component: "App",
          action: "handleWorkflowGenerated",
          workflowId: workflow.id,
          projectId: currentProject.id,
        },
        error,
      );
    }
  };

  // =========================================================================
  // Keyboard Shortcuts - Centralized System
  // =========================================================================

  // Set up the global keyboard shortcut handler (must be called once at root)
  useKeyboardShortcutHandler();

  // Map current view to shortcut context
  const shortcutContext = useMemo<ShortcutContext>(() => {
    // Check for open modals first
    if (showKeyboardShortcuts || showAIModal || showProjectModal || showDocs || showTour) {
      return 'modal';
    }
    switch (currentView) {
      case 'dashboard':
      case 'all-workflows':
      case 'all-executions':
        return 'dashboard';
      case 'project-detail':
        return 'project-detail';
      case 'project-workflow':
        return 'workflow-builder';
      case 'settings':
        return 'settings';
      default:
        return 'dashboard';
    }
  }, [currentView, showKeyboardShortcuts, showAIModal, showProjectModal, showDocs, showTour]);

  // Set the active shortcut context
  useShortcutContext(shortcutContext);

  // Register all shortcut actions
  const shortcutActions = useMemo(
    () => ({
      // Global shortcuts
      'show-shortcuts': () => setShowKeyboardShortcuts(true),
      'close-modal': () => {
        if (showKeyboardShortcuts) {
          setShowKeyboardShortcuts(false);
        } else if (showDocs) {
          setShowDocs(false);
        } else if (showAIModal) {
          setShowAIModal(false);
        } else if (showProjectModal) {
          setShowProjectModal(false);
        } else if (currentView === "project-workflow" && currentProject) {
          openProject(currentProject);
        } else if (currentView === "project-detail" || currentView === "settings") {
          navigateToDashboard();
        }
      },
      'global-search': () => {
        // In workflow builder context, focus the node palette search
        // Otherwise, open the dashboard global search modal
        const nodePaletteSearch = document.querySelector<HTMLInputElement>(
          '[data-testid="node-palette-search-input"]'
        );
        if (nodePaletteSearch && shortcutContext === 'workflow-builder') {
          nodePaletteSearch.focus();
        } else {
          // Dispatch event for Dashboard to open search modal
          window.dispatchEvent(new CustomEvent('open-global-search'));
        }
      },
      'open-settings': () => navigateToSettings(),

      // Dashboard shortcuts
      'new-project': () => setShowProjectModal(true),
      'go-home': () => navigateToDashboard(),
      'open-tutorial': () => openTour(),

      // Project detail shortcuts
      'new-workflow': () => setShowAIModal(true),
    }),
    [
      showKeyboardShortcuts,
      showDocs,
      showAIModal,
      showProjectModal,
      currentView,
      currentProject,
      openProject,
      navigateToDashboard,
      navigateToSettings,
      openTour,
    ]
  );

  useRegisterShortcuts(shortcutActions);

  useEffect(() => {
    const resolvePath = async (path: string, replace = false) => {
      console.log("[DEBUG] resolvePath called", { path, replace });
      const normalized = path.replace(/\/+/g, "/").replace(/\/$/, "") || "/";

      if (normalized === "/") {
        console.log("[DEBUG] resolvePath: navigating to dashboard");
        navigateToDashboard(replace);
        return;
      }

      const segments = normalized.split("/").filter(Boolean);
      console.log("[DEBUG] resolvePath segments", { segments });

      // Handle /settings route
      if (segments[0] === "settings") {
        console.log("[DEBUG] resolvePath: navigating to settings");
        navigateToSettings(replace);
        return;
      }

      // Handle /workflows route (global workflows view)
      if (segments[0] === "workflows" && segments.length === 1) {
        console.log("[DEBUG] resolvePath: navigating to all workflows");
        navigateToAllWorkflows(replace);
        return;
      }

      // Handle /executions route (global executions view)
      if (segments[0] === "executions" && segments.length === 1) {
        console.log("[DEBUG] resolvePath: navigating to all executions");
        navigateToAllExecutions(replace);
        return;
      }

      if (segments[0] !== "projects" || !segments[1]) {
        console.log("[DEBUG] resolvePath: invalid path, navigating to dashboard");
        navigateToDashboard(replace);
        return;
      }

      const projectId = segments[1];
      console.log("[DEBUG] resolvePath: loading project", { projectId });
      const projectState = useProjectStore.getState();
      let project: Project | null | undefined = projectState.projects.find(
        (p) => p.id === projectId,
      );
      if (!project) {
        console.log("[DEBUG] resolvePath: project not in cache, fetching from API");
        project = (await projectState.getProject(projectId)) as Project | null;
        if (project) {
          console.log("[DEBUG] resolvePath: project fetched, adding to store");
          useProjectStore.setState((state) => ({
            projects: state.projects.some((p) => p.id === project!.id)
              ? state.projects
              : [project!, ...state.projects],
          }));
        }
      } else {
        console.log("[DEBUG] resolvePath: project found in cache");
      }

      if (!project) {
        console.error("[DEBUG] resolvePath: project not found, navigating to dashboard");
        navigateToDashboard(replace);
        return;
      }

      if (segments.length >= 4 && segments[2] === "workflows") {
        const workflowId = segments[3];
        console.log("[DEBUG] resolvePath: calling openWorkflow", { workflowId, hasProject: !!project });
        await openWorkflow(project, workflowId, { replace });
        console.log("[DEBUG] resolvePath: openWorkflow completed");
        return;
      }

      console.log("[DEBUG] resolvePath: opening project view");
      openProject(project, { replace });
    };

    resolvePath(window.location.pathname, true).catch((error) => {
      logger.warn(
        "Failed to resolve initial route",
        { component: "App", action: "resolvePath" },
        error,
      );
    });

    const popHandler = () => {
      resolvePath(window.location.pathname, true).catch((error) => {
        logger.warn(
          "Failed to resolve popstate route",
          { component: "App", action: "handlePopState" },
          error,
        );
      });
    };

    window.addEventListener("popstate", popHandler);
    return () => window.removeEventListener("popstate", popHandler);
  }, [navigateToDashboard, navigateToSettings, navigateToAllWorkflows, navigateToAllExecutions, openProject, openWorkflow]);

  // Show loading while determining initial route
  if (currentView === null) {
    return (
      <div className="h-screen flex items-center justify-center bg-flow-bg">
        <LoadingSpinner variant="branded" size={32} message="Loading Browser Automation Studio..." />
      </div>
    );
  }

  // Dashboard View
  if (currentView === "dashboard") {
    console.log("[DEBUG] App rendering Dashboard view, showProjectModal:", showProjectModal);
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="branded" size={28} message="Loading dashboard..." />
            </div>
          }
        >
          <Dashboard
            onProjectSelect={handleProjectSelect}
            onCreateProject={handleCreateProject}
            onShowKeyboardShortcuts={() => setShowKeyboardShortcuts(true)}
            onOpenSettings={navigateToSettings}
            onOpenTutorial={openTour}
            onOpenDocs={() => setShowDocs(true)}
            onNavigateToWorkflow={handleNavigateToWorkflow}
            onViewExecution={handleViewExecution}
            onAIGenerateWorkflow={handleDashboardAIGenerate}
            onRunWorkflow={handleRunWorkflow}
            onViewAllWorkflows={navigateToAllWorkflows}
            onViewAllExecutions={navigateToAllExecutions}
            onTryDemo={handleTryDemo}
          />
        </Suspense>

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />

        <ProjectModal
          isOpen={showProjectModal}
          onClose={() => {
            console.log("[DEBUG] App: setShowProjectModal(false) called, current value:", showProjectModal);
            setShowProjectModal(false);
            console.log("[DEBUG] App: setShowProjectModal(false) completed");
          }}
          onSuccess={handleProjectCreated}
        />

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <DocsModal
          isOpen={showDocs}
          onClose={() => setShowDocs(false)}
        />
      </div>
    );
  }

  // Settings View
  if (currentView === "settings") {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="default" size={24} message="Loading settings..." />
            </div>
          }
        >
          <SettingsPage onBack={navigateToDashboard} />
        </Suspense>

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // All Workflows View (global)
  if (currentView === "all-workflows") {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="default" size={24} message="Loading workflows..." />
            </div>
          }
        >
          <GlobalWorkflowsView
            onBack={navigateToDashboard}
            onNavigateToWorkflow={handleNavigateToWorkflow}
            onRunWorkflow={handleRunWorkflow}
          />
        </Suspense>

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // All Executions View (global)
  if (currentView === "all-executions") {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="default" size={24} message="Loading executions..." />
            </div>
          }
        >
          <GlobalExecutionsView
            onBack={navigateToDashboard}
            onViewExecution={handleViewExecution}
          />
        </Suspense>

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // Project Detail View
  if (currentView === "project-detail" && currentProject) {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="default" size={24} message="Loading project..." />
            </div>
          }
        >
          <ProjectDetail
            project={currentProject}
            onBack={handleBackToDashboard}
            onWorkflowSelect={handleWorkflowSelect}
            onCreateWorkflow={handleCreateWorkflow}
            onCreateWorkflowDirect={handleCreateWorkflowDirect}
          />
        </Suspense>

        {showAIModal && (
          <Suspense
            fallback={
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                <LoadingSpinner variant="branded" size={24} message="Loading AI assistant..." />
              </div>
            }
          >
            <AIPromptModal
              onClose={() => setShowAIModal(false)}
              folder={selectedFolder}
              projectId={currentProject.id}
              onSwitchToManual={handleSwitchToManualBuilder}
              onSuccess={handleWorkflowGenerated}
            />
          </Suspense>
        )}

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // Project Workflow View (fallback)
  return (
    <ReactFlowProvider>
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Header
          onNewWorkflow={() => setShowAIModal(true)}
          onBackToDashboard={
            currentView === "project-workflow"
              ? handleBackToProjectDetail
              : handleBackToDashboard
          }
          currentProject={currentProject}
          currentWorkflow={selectedWorkflow}
          showBackToProject={currentView === "project-workflow"}
          onShowKeyboardShortcuts={() => setShowKeyboardShortcuts(true)}
          onOpenTutorial={openTour}
        />

        <div className="flex-1 flex overflow-hidden min-h-0">
          <Sidebar
            selectedFolder={selectedFolder}
            onFolderSelect={setSelectedFolder}
            projectId={currentProject?.id}
          />

          <div
            className={`flex-1 flex min-h-0 ${isResizingExecution ? "select-none" : ""}`}
          >
            <div className="flex-1 flex flex-col min-h-0">
              <Suspense
                fallback={
                  <div className="h-full flex items-center justify-center">
                    <LoadingSpinner variant="default" size={24} message="Loading workflow builder..." />
                  </div>
                }
              >
                <WorkflowBuilder projectId={currentProject?.id} />
              </Suspense>
            </div>

            {isExecutionViewerOpen &&
              activeExecutionWorkflowId &&
              isLargeScreen && (
                <>
                  <div
                    role="separator"
                    aria-orientation="vertical"
                    className={`w-1 cursor-col-resize transition-colors ${
                      isResizingExecution
                        ? "bg-flow-accent/50"
                        : "bg-transparent hover:bg-flow-accent/40"
                    }`}
                    onMouseDown={handleExecutionResizeMouseDown}
                    aria-label="Resize execution viewer pane"
                  />
                  <div
                    className="border-l border-gray-800 flex flex-col min-h-0"
                    style={{
                      width: executionPaneWidth,
                      minWidth: EXECUTION_MIN_WIDTH,
                    }}
                  >
                    <Suspense
                      fallback={
                        <div className="h-full flex items-center justify-center">
                          <LoadingSpinner variant="minimal" size={20} />
                        </div>
                      }
                    >
                      <ExecutionViewer
                        workflowId={activeExecutionWorkflowId}
                        execution={currentExecution}
                        onClose={closeExecutionViewer}
                        showExecutionSwitcher
                      />
                    </Suspense>
                  </div>
                </>
              )}
          </div>
        </div>

        {isExecutionViewerOpen &&
          activeExecutionWorkflowId &&
          !isLargeScreen && (
            <ResponsiveDialog
              isOpen={true}
              onDismiss={closeExecutionViewer}
              ariaLabel="Execution Viewer"
              size="xl"
            >
              <Suspense
                fallback={
                  <div className="h-full flex items-center justify-center">
                    <LoadingSpinner variant="minimal" size={20} />
                  </div>
                }
              >
                <ExecutionViewer
                  workflowId={activeExecutionWorkflowId}
                  execution={currentExecution}
                  onClose={closeExecutionViewer}
                  showExecutionSwitcher
                />
              </Suspense>
            </ResponsiveDialog>
          )}

        {showAIModal && (
          <Suspense
            fallback={
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                <LoadingSpinner variant="branded" size={24} message="Loading AI assistant..." />
              </div>
            }
          >
            <AIPromptModal
              onClose={() => setShowAIModal(false)}
              folder={selectedFolder}
              projectId={currentProject?.id}
              onSwitchToManual={handleSwitchToManualBuilder}
              onSuccess={handleWorkflowGenerated}
            />
          </Suspense>
        )}

        <KeyboardShortcutsModal
          isOpen={showKeyboardShortcuts}
          onClose={() => setShowKeyboardShortcuts(false)}
                  />

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    </ReactFlowProvider>
  );
}

export default App;
