import { useCallback, useEffect, useState, lazy, Suspense, useMemo } from "react";

// Shared layout components
import ProjectModal from "@features/projects/ProjectModal";
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
const Header = lazy(() => import("@shared/layout/Header"));
const Sidebar = lazy(() => import("@shared/layout/Sidebar"));
const ResponsiveDialog = lazy(() => import("@shared/layout/ResponsiveDialog"));
const WorkflowBuilder = lazy(() => import("@features/workflows/builder/WorkflowBuilder"));
const ExecutionViewer = lazy(() => import("@features/execution/ExecutionViewer"));
const AIPromptModal = lazy(() => import("@features/ai/AIPromptModal"));
const Dashboard = lazy(() => import("@features/projects/Dashboard"));
const ProjectDetail = lazy(() => import("@features/projects/ProjectDetail"));
const SettingsPage = lazy(() => import("@features/settings/SettingsPage"));
const GlobalWorkflowsView = lazy(() => import("@features/dashboard/GlobalWorkflowsView").then(m => ({ default: m.GlobalWorkflowsView })));
const GlobalExecutionsView = lazy(() => import("@features/dashboard/GlobalExecutionsView").then(m => ({ default: m.GlobalExecutionsView })));
const RecordModePage = lazy(() => import("@features/record-mode/RecordModePage").then(m => ({ default: m.RecordModePage })));
import { type DashboardTab } from "@features/dashboard";

// Stores and hooks
import { useExecutionStore } from "@stores/executionStore";
import { useProjectStore, type Project, buildProjectFolderPath } from "@stores/projectStore";
import { useScenarioStore } from "@stores/scenarioStore";
import { useDashboardStore, type RecentWorkflow } from "@stores/dashboardStore";
import { useMediaQuery } from "@hooks/useMediaQuery";
import { useEntitlementInit } from "@hooks/useEntitlement";
import { useScheduleNotifications } from "@hooks/useScheduleNotifications";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import { selectors } from "@constants/selectors";
import { LoadingSpinner } from "@shared/ui";

interface NormalizedWorkflow extends Partial<Record<string, unknown>> {
  id: string;
  name: string;
  folderPath: string;
  createdAt: Date;
  updatedAt: Date;
  projectId?: string;
}

type AppView = "dashboard" | "project-detail" | "project-workflow" | "settings" | "all-workflows" | "all-executions" | "record-mode";

const EXECUTION_MIN_WIDTH = 360;
const EXECUTION_MAX_WIDTH = 720;
const EXECUTION_DEFAULT_WIDTH = 440;

function App() {
  const [currentView, setCurrentView] = useState<AppView | null>(null);
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [showDocs, setShowDocs] = useState(false);
  const [docsInitialTab, setDocsInitialTab] = useState<
    "getting-started" | "node-reference" | "schema-reference" | "shortcuts"
  >("getting-started");
  const [selectedFolder, setSelectedFolder] = useState<string>("/");
  const [selectedWorkflow, setSelectedWorkflow] =
    useState<NormalizedWorkflow | null>(null);
  const [executionPaneWidth, setExecutionPaneWidth] = useState(
    EXECUTION_DEFAULT_WIDTH,
  );
  const [isResizingExecution, setIsResizingExecution] = useState(false);
  const [recordingSessionId, setRecordingSessionId] = useState<string | null>(null);
  const [dashboardTab, setDashboardTab] = useState<DashboardTab>("home");

  const { projects, currentProject, setCurrentProject } = useProjectStore();
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

  // Initialize entitlement state on app mount
  useEntitlementInit();

  // Initialize schedule notifications (WebSocket + desktop tray updates)
  useScheduleNotifications();

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
    (options?: { replace?: boolean; tab?: DashboardTab } | boolean) => {
      const replace = typeof options === "boolean" ? options : options?.replace ?? false;
      const targetTab =
        typeof options === "object" && options?.tab ? options.tab : dashboardTab;
      const search = targetTab !== "home" ? `?tab=${targetTab}` : "";
      const url = `/${search}`;
      const state = { view: "dashboard", tab: targetTab };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setSelectedWorkflow(null);
      setSelectedFolder("/");
      setCurrentProject(null);
      setDashboardTab(targetTab);
      setCurrentView("dashboard");
    },
    [dashboardTab, safeNavigate, setCurrentProject],
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

  const navigateToRecordMode = useCallback(
    (sessionId?: string | null, replace = false) => {
      const isNewSession = !sessionId;
      const url = isNewSession ? "/record/new" : `/record/${sessionId}`;
      const state = { view: "record-mode", sessionId: sessionId ?? null, isNew: isNewSession };
      safeNavigate(state, url, replace);
      setShowAIModal(false);
      setShowProjectModal(false);
      setRecordingSessionId(sessionId ?? null);
      setCurrentView("record-mode");
    },
    [safeNavigate],
  );

  const closeRecordMode = useCallback(async () => {
    if (recordingSessionId) {
      try {
        // Close the session on the server
        const config = await import("./config").then(m => m.getConfig());
        await fetch(`${config.API_URL}/recordings/live/session/${recordingSessionId}/close`, {
          method: "POST",
        });
      } catch (error) {
        logger.warn("Failed to close recording session", { component: "App", action: "closeRecordMode" }, error);
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
        const workflowStore = await import("@stores/workflowStore");
        if (options?.workflowData) {
          // When workflowData is provided (e.g., from createWorkflow), use it directly
          // to avoid redundant API call that could fail and cause navigation away
          const normalized = transformWorkflow(options.workflowData);
          if (normalized) {
            setSelectedWorkflow(normalized);
            setSelectedFolder(
              normalized.folderPath || project.folder_path || "/",
            );
          }
          // Verify the store is populated (createWorkflow should have done this)
          const storeWorkflow = workflowStore.useWorkflowStore.getState().currentWorkflow;
          if (!storeWorkflow || storeWorkflow.id !== workflowId) {
            await workflowStore.useWorkflowStore.getState().loadWorkflow(workflowId);
          }
        } else {
          // No workflowData provided, load from API (e.g., direct URL navigation)
          await workflowStore.useWorkflowStore.getState().loadWorkflow(workflowId);
          const loadedWorkflow = workflowStore.useWorkflowStore.getState().currentWorkflow;
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
      } catch (error) {
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
        openProject(project, { replace: options?.replace ?? false });
      }
    },
    [
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

  const handleWorkflowSelect = async (workflow: Record<string, unknown> & { id?: string }) => {
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
          folder_path: buildProjectFolderPath('my-automations'),
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

  const handleDashboardTabChange = useCallback((tab: DashboardTab) => {
    navigateToDashboard({ tab });
  }, [navigateToDashboard]);

  // Handler for "Create Your First Workflow" button on welcome screen
  // Auto-creates a project and workflow, then navigates to the workflow builder
  const handleCreateFirstWorkflow = useCallback(async () => {
    try {
      // Create a default project
      const project = await useProjectStore.getState().createProject({
        name: 'My Automations',
        description: 'Automated browser workflows',
        folder_path: buildProjectFolderPath('my-automations'),
      });

      // Create an empty workflow in the project
      const workflowName = `my-first-workflow`;
      const workflowStore = await import("@stores/workflowStore");
      const workflow = await workflowStore.useWorkflowStore.getState().createWorkflow(
        workflowName,
        '/',
        project.id
      );

      if (workflow?.id) {
        // Navigate directly to the workflow builder
        await openWorkflow(project, workflow.id, {
          workflowData: workflow as Record<string, unknown>,
        });
        toast.success('Welcome! Start building your first automation.');
      } else {
        throw new Error('Failed to create workflow');
      }
    } catch (error) {
      logger.error('Failed to create first workflow', { component: 'App', action: 'handleCreateFirstWorkflow' }, error);
      toast.error('Failed to create workflow. Please try again.');
    }
  }, [openWorkflow]);

  // Handler for when a workflow is generated from recording
  const handleRecordingWorkflowGenerated = useCallback(async (workflowId: string, projectId: string) => {
    // Close the recording session
    if (recordingSessionId) {
      try {
        const config = await import("./config").then(m => m.getConfig());
        await fetch(`${config.API_URL}/recordings/live/session/${recordingSessionId}/close`, {
          method: "POST",
        });
      } catch (error) {
        logger.warn("Failed to close recording session", { component: "App", action: "handleRecordingWorkflowGenerated" }, error);
      }
      setRecordingSessionId(null);
    }

    // Navigate to the generated workflow
    toast.success("Workflow generated from recording!");
    await handleNavigateToWorkflow(projectId, workflowId);
  }, [recordingSessionId, handleNavigateToWorkflow]);

  const handleRecordingSessionReady = useCallback((sessionId: string) => {
    setRecordingSessionId(sessionId);
    navigateToRecordMode(sessionId, true);
  }, [navigateToRecordMode]);

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

  const handleStartRecording = useCallback(() => {
    navigateToRecordMode(null);
  }, [navigateToRecordMode]);

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
      const workflowStore = await import("@stores/workflowStore");
      const workflow = await workflowStore.useWorkflowStore
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
    if (!currentProject?.id) {
      toast.error("Select a project before using the manual builder.");
      return;
    }

    // Close modal immediately BEFORE async operation
    // This ensures the modal closes instantly for the user and tests don't fail
    // waiting for the async workflow creation to complete
    setShowAIModal(false);

    // Create a new empty workflow and switch to the manual builder
    const workflowName = `new-workflow-${Date.now()}`;
    const folderPath = selectedFolder || "/";

    try {
      const workflowStore = await import("@stores/workflowStore");
      const workflow = await workflowStore.useWorkflowStore
        .getState()
        .createWorkflow(workflowName, folderPath, currentProject.id);

      if (currentProject && workflow?.id) {
        await openWorkflow(currentProject, workflow.id, {
          workflowData: workflow as Record<string, unknown>,
        });
      } else {
        toast.error("Failed to create workflow.");
      }
    } catch (error) {
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

  const handleWorkflowGenerated = async (workflow: Record<string, unknown> & { id?: string }) => {
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
    if (showAIModal || showProjectModal || showDocs || showTour) {
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
      case 'record-mode':
        return 'modal'; // Record mode uses modal context for focused recording experience
      default:
        return 'dashboard';
    }
  }, [currentView, showAIModal, showProjectModal, showDocs, showTour]);

  // Set the active shortcut context
  useShortcutContext(shortcutContext);

  // Register all shortcut actions
  const shortcutActions = useMemo(
    () => ({
      // Global shortcuts
      'show-shortcuts': () => {
        setDocsInitialTab('shortcuts');
        setShowDocs(true);
      },
      'close-modal': () => {
        if (showDocs) {
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
      'start-recording': () => {
        void handleStartRecording();
      },
    }),
    [
      showDocs,
      showAIModal,
      showProjectModal,
      currentView,
      currentProject,
      openProject,
      navigateToDashboard,
      navigateToSettings,
      openTour,
      handleStartRecording,
    ]
  );

  useRegisterShortcuts(shortcutActions);

  useEffect(() => {
    const parseDashboardTab = (value: string | null): DashboardTab => {
      if (value === "projects" || value === "executions" || value === "exports" || value === "schedules") {
        return value;
      }
      return "home";
    };

    const resolvePath = async (path: string, replace = false) => {
      const normalized = path.replace(/\/+/g, "/").replace(/\/$/, "") || "/";
      const searchParams = new URLSearchParams(window.location.search);
      const requestedTab = parseDashboardTab(searchParams.get("tab"));

      if (normalized === "/") {
        navigateToDashboard({ replace, tab: requestedTab });
        return;
      }

      if (normalized === "/schedules") {
        navigateToDashboard({ replace, tab: "schedules" });
        return;
      }

      const segments = normalized.split("/").filter(Boolean);

      // Handle /settings route
      if (segments[0] === "settings") {
        navigateToSettings(replace);
        return;
      }

      // Handle /workflows route (global workflows view)
      if (segments[0] === "workflows" && segments.length === 1) {
        navigateToAllWorkflows(replace);
        return;
      }

      // Handle /executions route (global executions view)
      if (segments[0] === "executions" && segments.length === 1) {
        navigateToAllExecutions(replace);
        return;
      }

      // Handle /record/{sessionId} route (record mode)
      if (segments[0] === "record" && segments[1]) {
        const sessionId = segments[1] === "new" ? null : segments[1];
        navigateToRecordMode(sessionId, replace);
        return;
      }

      if (segments[0] !== "projects" || !segments[1]) {
        navigateToDashboard(replace);
        return;
      }

      const projectId = segments[1];
      const projectState = useProjectStore.getState();
      let project: Project | null | undefined = projectState.projects.find(
        (p) => p.id === projectId,
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
  }, [navigateToDashboard, navigateToSettings, navigateToAllWorkflows, navigateToAllExecutions, navigateToRecordMode, openProject, openWorkflow]);

  // Show loading while determining initial route
  if (currentView === null) {
    return (
      <div className="h-screen flex items-center justify-center bg-flow-bg">
        <LoadingSpinner variant="branded" size={32} message="Loading Vrooli Ascension..." />
      </div>
    );
  }

  // Dashboard View
  const docsModal = (
    <DocsModal
      isOpen={showDocs}
      initialTab={docsInitialTab}
      onClose={() => setShowDocs(false)}
      onOpenTutorial={openTour}
    />
  );

  if (currentView === "dashboard") {
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
            activeTab={dashboardTab}
            onTabChange={handleDashboardTabChange}
            onProjectSelect={handleProjectSelect}
            onCreateProject={handleCreateProject}
            onCreateFirstWorkflow={handleCreateFirstWorkflow}
          onStartRecording={handleStartRecording}
          onOpenSettings={navigateToSettings}
          onOpenHelp={() => {
            setDocsInitialTab('getting-started');
            setShowDocs(true);
          }}
          onOpenTutorial={openTour}
          onNavigateToWorkflow={handleNavigateToWorkflow}
          onViewExecution={handleViewExecution}
          onAIGenerateWorkflow={handleDashboardAIGenerate}
          onRunWorkflow={handleRunWorkflow}
          onViewAllWorkflows={navigateToAllWorkflows}
            onViewAllExecutions={navigateToAllExecutions}
          />
        </Suspense>

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />

        <ProjectModal
          isOpen={showProjectModal}
          onClose={() => {
            setShowProjectModal(false);
          }}
          onSuccess={handleProjectCreated}
        />

        {docsModal}
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

        {docsModal}

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

        {docsModal}

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

        {docsModal}

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // Record Mode View
  if (currentView === "record-mode") {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <LoadingSpinner variant="branded" size={24} message="Loading record mode..." />
            </div>
          }
        >
          <RecordModePage
            sessionId={recordingSessionId}
            onWorkflowGenerated={handleRecordingWorkflowGenerated}
            onSessionReady={handleRecordingSessionReady}
            onClose={closeRecordMode}
          />
        </Suspense>

        {docsModal}

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
            onStartRecording={handleStartRecording}
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

        {docsModal}

        <GuidedTour
          isOpen={showTour}
          onClose={closeTour}
        />
      </div>
    );
  }

  // Project Workflow View (fallback)
  return (
    <div
      className="h-screen flex flex-col bg-flow-bg"
      data-testid={selectors.app.shell.ready}
    >
      <Suspense
        fallback={
          <div className="h-[72px] border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90" />
        }
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
          onOpenHelp={() => {
            setDocsInitialTab('shortcuts');
            setShowDocs(true);
          }}
        />
      </Suspense>

      <div className="flex-1 flex overflow-hidden min-h-0">
        <Suspense
          fallback={
            <div className="hidden lg:block w-64 border-r border-gray-800 bg-flow-bg" />
          }
        >
          <Sidebar
            selectedFolder={selectedFolder}
            onFolderSelect={setSelectedFolder}
            projectId={currentProject?.id}
          />
        </Suspense>

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
              <WorkflowBuilder
                projectId={currentProject?.id}
                onStartRecording={handleStartRecording}
              />
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
          <Suspense
            fallback={
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                <LoadingSpinner variant="minimal" size={20} />
              </div>
            }
          >
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
          </Suspense>
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

      {docsModal}

      <GuidedTour
        isOpen={showTour}
        onClose={closeTour}
      />
    </div>
  );
}

export default App;
