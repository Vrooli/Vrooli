import { useCallback, useEffect, lazy, Suspense } from "react";

// Shared layout components
import { ProjectModal } from "@/domains/projects";
import { WorkflowCreationDialog, type WorkflowCreationType } from "@/domains/workflows";
import { GuidedTour, useGuidedTour } from "@shared/onboarding";
import { DocsModal } from "@/domains/docs";

// Extracted feature modules
import { ModalProvider, useModals } from "@shared/modals";
import { useAppShortcuts } from "@shared/hooks/useAppShortcuts";
import { ExecutionPaneLayout } from "@/domains/executions";

// Navigation - SINGLE SOURCE OF TRUTH
import { useAppNavigation, type DashboardTab } from "@/routing";

// Lazy load heavy components for better initial load performance
const Header = lazy(() => import("@shared/layout/Header"));
const Sidebar = lazy(() => import("@shared/layout/Sidebar"));
const WorkflowBuilder = lazy(() => import("@/domains/workflows/builder/WorkflowBuilder"));
const AIPromptModal = lazy(() => import("@/domains/ai/AIPromptModal"));
const Dashboard = lazy(() => import("@/views/DashboardView/DashboardView"));
const ProjectDetail = lazy(() => import("@/domains/projects/ProjectDetail"));
const SettingsPage = lazy(() => import("@/views/SettingsView/SettingsView"));
const GlobalWorkflowsView = lazy(() => import("@/views/AllWorkflowsView/GlobalWorkflowsView").then(m => ({ default: m.GlobalWorkflowsView })));
const GlobalExecutionsView = lazy(() => import("@/views/AllExecutionsView/GlobalExecutionsView").then(m => ({ default: m.GlobalExecutionsView })));
const RecordModePage = lazy(() => import("@/domains/recording/RecordingSession").then(m => ({ default: m.RecordModePage })));

// Stores and hooks
import { useExecutionStore } from "@/domains/executions";
import { useProjectStore, type Project, buildProjectFolderPath } from "@/domains/projects";
import { useScenarioStore } from "@stores/scenarioStore";
import { useMediaQuery } from "@hooks/useMediaQuery";
import { useEntitlementInit } from "@hooks/useEntitlement";
import { useScheduleNotifications } from "@hooks/useScheduleNotifications";
import { logger } from "@utils/logger";
import toast from "react-hot-toast";
import { selectors } from "@constants/selectors";
import { LoadingSpinner } from "@shared/ui";

/**
 * App - Root component that provides the ModalProvider context
 */
function App() {
  // Navigation state from centralized hook (SINGLE SOURCE OF TRUTH)
  const { currentView } = useAppNavigation();

  // Show loading while determining initial route
  if (currentView === null) {
    return (
      <div className="h-screen flex items-center justify-center bg-flow-bg">
        <LoadingSpinner variant="branded" size={32} message="Loading Vrooli Ascension..." />
      </div>
    );
  }

  return (
    <ModalProvider currentView={currentView}>
      <AppContent />
    </ModalProvider>
  );
}

/**
 * AppContent - Main application content that uses modal context
 */
function AppContent() {
  // Navigation state from centralized hook (SINGLE SOURCE OF TRUTH)
  const {
    currentView,
    currentProject,
    selectedWorkflow,
    selectedFolder,
    dashboardTab,
    recordingSessionId,
    navigateToDashboard,
    openProject,
    openWorkflow,
    navigateToSettings,
    navigateToAllWorkflows,
    navigateToAllExecutions,
    navigateToRecordMode,
    closeRecordMode,
  } = useAppNavigation();

  // Modal state from context (replaces local useState)
  const {
    showAIModal,
    showProjectModal,
    showWorkflowCreationModal,
    showDocs,
    docsInitialTab,
    openAIModal,
    closeAIModal,
    openProjectModal,
    closeProjectModal,
    openWorkflowCreationModal,
    closeWorkflowCreationModal,
    openDocs,
    closeDocs,
  } = useModals();

  // Store hooks
  const { projects, setCurrentProject } = useProjectStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const viewerWorkflowId = useExecutionStore((state) => state.viewerWorkflowId);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const loadExecution = useExecutionStore((state) => state.loadExecution);
  const startExecution = useExecutionStore((state) => state.startExecution);
  const { fetchScenarios } = useScenarioStore();
  const isLargeScreen = useMediaQuery("(min-width: 1024px)");

  // Guided tour state
  const { showTour, openTour, closeTour } = useGuidedTour();

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
        openAIModal();
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
      openAIModal();
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
    // Close the recording session on the server
    if (recordingSessionId) {
      try {
        const config = await import("./config").then(m => m.getConfig());
        await fetch(`${config.API_URL}/recordings/live/session/${recordingSessionId}/close`, {
          method: "POST",
        });
      } catch (error) {
        logger.warn("Failed to close recording session", { component: "App", action: "handleRecordingWorkflowGenerated" }, error);
      }
      // Note: recordingSessionId will be cleared when we navigate away from record-mode
    }

    // Navigate to the generated workflow
    toast.success("Workflow generated from recording!");
    await handleNavigateToWorkflow(projectId, workflowId);
  }, [recordingSessionId, handleNavigateToWorkflow]);

  const handleRecordingSessionReady = useCallback((sessionId: string) => {
    // navigateToRecordMode handles setting recordingSessionId in the hook
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
    openProjectModal();
  };

  const handleProjectCreated = (project: Project) => {
    // Show success toast
    toast.success(`Project "${project.name}" created successfully`);
    // Automatically select the newly created project
    openProject(project);
  };

  const handleCreateWorkflow = () => {
    openWorkflowCreationModal();
  };

  // Handler for workflow type selection from WorkflowCreationDialog
  const handleWorkflowTypeSelected = useCallback(async (type: WorkflowCreationType, project: Project) => {
    // Set the current project
    setCurrentProject(project);

    if (type === "record") {
      // Start recording mode
      navigateToRecordMode(null);
    } else if (type === "ai") {
      // Open AI modal
      openProject(project);
      openAIModal();
    } else if (type === "visual") {
      // Create an empty workflow and navigate to builder
      const workflowName = `new-workflow-${Date.now()}`;
      const folderPath = "/";

      try {
        const workflowStore = await import("@stores/workflowStore");
        const workflow = await workflowStore.useWorkflowStore
          .getState()
          .createWorkflow(workflowName, folderPath, project.id);

        if (workflow?.id) {
          await openWorkflow(project, workflow.id, {
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
            action: "handleWorkflowTypeSelected",
            projectId: project.id,
          },
          error,
        );
        toast.error("Failed to create workflow. Please try again.");
      }
    }
  }, [setCurrentProject, navigateToRecordMode, openProject, openAIModal, openWorkflow]);

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
    closeAIModal();

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
  // Keyboard Shortcuts - Centralized System (extracted to useAppShortcuts)
  // =========================================================================
  useAppShortcuts({
    currentView,
    showAIModal,
    showProjectModal,
    showDocs,
    showTour,
    openDocs,
    closeDocs,
    closeAIModal,
    closeProjectModal,
    navigateToDashboard,
    navigateToSettings,
    openProject,
    openProjectModal,
    openAIModal,
    openTour,
    handleStartRecording,
    currentProject,
  });

  // Dashboard View
  const docsModal = (
    <DocsModal
      isOpen={showDocs}
      initialTab={docsInitialTab}
      onClose={closeDocs}
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
            onCreateWorkflow={handleCreateWorkflow}
            onCreateFirstWorkflow={handleCreateFirstWorkflow}
          onStartRecording={handleStartRecording}
          onOpenSettings={navigateToSettings}
          onOpenHelp={() => openDocs('getting-started')}
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
            closeProjectModal();
          }}
          onSuccess={handleProjectCreated}
        />

        <WorkflowCreationDialog
          isOpen={showWorkflowCreationModal}
          onClose={closeWorkflowCreationModal}
          onSelectType={handleWorkflowTypeSelected}
          onCreateProject={openProjectModal}
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
              onClose={() => closeAIModal()}
              folder={selectedFolder}
              projectId={currentProject.id}
              onSwitchToManual={handleSwitchToManualBuilder}
              onSuccess={handleWorkflowGenerated}
            />
          </Suspense>
        )}

        <WorkflowCreationDialog
          isOpen={showWorkflowCreationModal}
          onClose={closeWorkflowCreationModal}
          onSelectType={handleWorkflowTypeSelected}
          onCreateProject={openProjectModal}
          preSelectedProject={currentProject}
        />

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
          onNewWorkflow={() => openAIModal()}
          onBackToDashboard={handleBackToDashboard}
          onBackToProject={
            currentView === "project-workflow" && currentProject
              ? handleBackToProjectDetail
              : undefined
          }
          currentProject={currentProject}
          currentWorkflow={selectedWorkflow}
          showBackToProject={currentView === "project-workflow"}
          onOpenHelp={() => openDocs('shortcuts')}
        />
      </Suspense>

      <div className="flex-1 flex overflow-hidden min-h-0">
        <Suspense
          fallback={
            <div className="hidden lg:block w-64 border-r border-gray-800 bg-flow-bg" />
          }
        >
          <Sidebar />
        </Suspense>

        <ExecutionPaneLayout
          isOpen={isExecutionViewerOpen}
          workflowId={activeExecutionWorkflowId}
          execution={currentExecution}
          isLargeScreen={isLargeScreen}
          onClose={closeExecutionViewer}
        >
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
        </ExecutionPaneLayout>
      </div>

      {showAIModal && (
        <Suspense
          fallback={
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
              <LoadingSpinner variant="branded" size={24} message="Loading AI assistant..." />
            </div>
          }
        >
          <AIPromptModal
            onClose={() => closeAIModal()}
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
