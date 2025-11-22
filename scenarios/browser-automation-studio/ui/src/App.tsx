import { useCallback, useEffect, useState, lazy, Suspense } from "react";
import { ReactFlowProvider } from "reactflow";
import ResponsiveDialog from "./components/ResponsiveDialog";
import Sidebar from "./components/Sidebar";
import Header from "./components/Header";
import ProjectModal from "./components/ProjectModal";

// Lazy load heavy components for better initial load performance
const WorkflowBuilder = lazy(() => import("./components/WorkflowBuilder"));
const ExecutionViewer = lazy(() => import("./components/ExecutionViewer"));
const AIPromptModal = lazy(() => import("./components/AIPromptModal"));
const Dashboard = lazy(() => import("./components/Dashboard"));
const ProjectDetail = lazy(() => import("./components/ProjectDetail"));
import { useExecutionStore } from "./stores/executionStore";
import { useProjectStore, Project } from "./stores/projectStore";
import { useWorkflowStore } from "./stores/workflowStore";
import { useScenarioStore } from "./stores/scenarioStore";
import { useMediaQuery } from "./hooks/useMediaQuery";
import type { Workflow } from "./stores/workflowStore";
import toast from "react-hot-toast";
import { selectors } from "./consts/selectors";

interface NormalizedWorkflow extends Partial<Workflow> {
  id: string;
  name: string;
  folderPath: string;
  createdAt: Date;
  updatedAt: Date;
  projectId?: string;
}
import { logger } from "./utils/logger";
import "reactflow/dist/style.css";

type AppView = "dashboard" | "project-detail" | "project-workflow";

const EXECUTION_MIN_WIDTH = 360;
const EXECUTION_MAX_WIDTH = 720;
const EXECUTION_DEFAULT_WIDTH = 440;

function App() {
  const [currentView, setCurrentView] = useState<AppView | null>(null);
  const [showAIModal, setShowAIModal] = useState(false);
  const [showProjectModal, setShowProjectModal] = useState(false);
  const [selectedFolder, setSelectedFolder] = useState<string>("/");
  const [selectedWorkflow, setSelectedWorkflow] =
    useState<NormalizedWorkflow | null>(null);
  const [executionPaneWidth, setExecutionPaneWidth] = useState(
    EXECUTION_DEFAULT_WIDTH,
  );
  const [isResizingExecution, setIsResizingExecution] = useState(false);

  const { currentProject, setCurrentProject } = useProjectStore();
  const { loadWorkflow } = useWorkflowStore();
  const currentExecution = useExecutionStore((state) => state.currentExecution);
  const viewerWorkflowId = useExecutionStore((state) => state.viewerWorkflowId);
  const closeExecutionViewer = useExecutionStore((state) => state.closeViewer);
  const { fetchScenarios } = useScenarioStore();
  const isLargeScreen = useMediaQuery("(min-width: 1024px)");

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
  }, [navigateToDashboard, openProject, openWorkflow]);

  // Show loading while determining initial route
  if (currentView === null) {
    return (
      <div className="h-screen flex items-center justify-center bg-flow-bg">
        <div className="text-gray-400">Loading...</div>
      </div>
    );
  }

  // Dashboard View
  if (currentView === "dashboard") {
    return (
      <div
        className="h-screen flex flex-col bg-flow-bg"
        data-testid={selectors.app.shell.ready}
      >
        <Suspense
          fallback={
            <div className="h-screen flex items-center justify-center bg-flow-bg">
              <div className="text-flow-text">Loading dashboard...</div>
            </div>
          }
        >
          <Dashboard
            onProjectSelect={handleProjectSelect}
            onCreateProject={handleCreateProject}
          />
        </Suspense>

        {showProjectModal && (
          <ProjectModal
            onClose={() => setShowProjectModal(false)}
            onSuccess={handleProjectCreated}
          />
        )}
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
              <div className="text-flow-text">Loading project...</div>
            </div>
          }
        >
          <ProjectDetail
            project={currentProject}
            onBack={handleBackToDashboard}
            onWorkflowSelect={handleWorkflowSelect}
            onCreateWorkflow={handleCreateWorkflow}
          />
        </Suspense>

        {showAIModal && (
          <Suspense
            fallback={<div className="text-flow-text">Loading AI modal...</div>}
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
                    <div className="text-flow-text">
                      Loading workflow builder...
                    </div>
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
                          <div className="text-flow-text">
                            Loading execution viewer...
                          </div>
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
                    <div className="text-flow-text">
                      Loading execution viewer...
                    </div>
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
            fallback={<div className="text-flow-text">Loading AI modal...</div>}
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
      </div>
    </ReactFlowProvider>
  );
}

export default App;
