import { useState, useEffect, useCallback, lazy, Suspense } from "react";
import {
  Plus,
  WifiOff,
  Search,
  Settings,
  HelpCircle,
  Command,
  Circle,
} from "lucide-react";
import { useProjectStore, Project } from "@stores/projectStore";
import { useDashboardStore } from "@stores/dashboardStore";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { selectors } from "@constants/selectors";
import { getModifierKey } from "@hooks/useKeyboardShortcuts";
import { GlobalSearchModal } from "./GlobalSearchModal";
import { TabNavigation, type DashboardTab } from "./DashboardTabs";
import { HomeTab } from "./HomeSection";
import { RunningExecutionsBadge } from "./RunningExecutionsBadge";
import { WelcomeHero } from "./WelcomeHero";

const ExecutionsTab = lazy(() =>
  import("@/domains/executions/history/ExecutionsTab").then((mod) => ({ default: mod.ExecutionsTab })),
);
const ExportsTab = lazy(() =>
  import("@/domains/exports/ExportsTab").then((mod) => ({ default: mod.ExportsTab })),
);
const ProjectsTab = lazy(() =>
  import("@/domains/projects/ProjectsTab").then((mod) => ({ default: mod.ProjectsTab })),
);
const SchedulesTab = lazy(() =>
  import("./SchedulesSection").then((mod) => ({ default: mod.SchedulesTab })),
);

interface DashboardProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
  onCreateFirstWorkflow?: () => void;
  onStartRecording?: () => void;
  onOpenSettings?: () => void;
  onOpenHelp?: () => void;
  onOpenTutorial?: () => void;
  onNavigateToWorkflow?: (projectId: string, workflowId: string) => void;
  onViewExecution?: (executionId: string, workflowId: string) => void;
  onAIGenerateWorkflow?: (prompt: string) => void;
  onRunWorkflow?: (workflowId: string) => void;
  onViewAllWorkflows?: () => void;
  onViewAllExecutions?: () => void;
  activeTab?: DashboardTab;
  onTabChange?: (tab: DashboardTab) => void;
  isGeneratingWorkflow?: boolean;
}

function Dashboard({
  onProjectSelect,
  onCreateProject,
  onCreateFirstWorkflow,
  onStartRecording,
  onOpenSettings,
  onOpenHelp,
  onOpenTutorial,
  onNavigateToWorkflow,
  onViewExecution,
  onAIGenerateWorkflow,
  onRunWorkflow,
  activeTab: activeTabProp,
  onTabChange,
  isGeneratingWorkflow = false,
}: DashboardProps) {
  const {
    projects,
    isLoading,
    error,
    fetchProjects,
    clearError,
  } = useProjectStore();
  const {
    fetchRecentWorkflows,
    fetchRecentExecutions,
    fetchRunningExecutions,
    runningExecutions,
  } = useDashboardStore();
  const { isConnected: isWebSocketConnected } = useWebSocket();
  const [activeTab, setActiveTab] = useState<DashboardTab>(activeTabProp ?? 'home');
  const [isSearchModalOpen, setIsSearchModalOpen] = useState(false);

  const scheduleNonCriticalWork = useCallback((work: () => void | Promise<void>) => {
    if (typeof window === 'undefined') {
      void work();
      return;
    }

    const requestIdleCallback = (window as unknown as { requestIdleCallback?: (cb: () => void, options?: { timeout: number }) => number }).requestIdleCallback;
    if (typeof requestIdleCallback === 'function') {
      requestIdleCallback(() => {
        void work();
      }, { timeout: 1500 });
      return;
    }

    window.setTimeout(() => {
      void work();
    }, 0);
  }, []);

  useEffect(() => {
    if (activeTabProp && activeTabProp !== activeTab) {
      setActiveTab(activeTabProp);
    }
  }, [activeTabProp, activeTab]);

  useEffect(() => {
    fetchProjects();
    fetchRecentWorkflows();
    fetchRecentExecutions();
    scheduleNonCriticalWork(() => {
      void fetchRunningExecutions();
    });
  }, [fetchProjects, fetchRecentWorkflows, fetchRecentExecutions, fetchRunningExecutions, scheduleNonCriticalWork]);

  // Auto-refresh running executions periodically
  useEffect(() => {
    const interval = setInterval(() => {
      void fetchRunningExecutions();
    }, 10000); // Every 10 seconds
    return () => clearInterval(interval);
  }, [fetchRunningExecutions]);

  // Listen for global search event from centralized keyboard shortcut system
  useEffect(() => {
    const handleOpenSearch = () => {
      setIsSearchModalOpen(true);
    };
    window.addEventListener('open-global-search', handleOpenSearch);
    return () => window.removeEventListener('open-global-search', handleOpenSearch);
  }, []);

  // Listen for navigate-to-exports events from ExecutionViewer
  useEffect(() => {
    const handleNavigateToExports = () => {
      setActiveTab('exports');
      if (onTabChange) {
        onTabChange('exports');
      }
    };
    window.addEventListener('navigate-to-exports', handleNavigateToExports);
    return () => window.removeEventListener('navigate-to-exports', handleNavigateToExports);
  }, [onTabChange]);

  const handleNavigateToWorkflow = useCallback((projectId: string, workflowId: string) => {
    if (onNavigateToWorkflow) {
      onNavigateToWorkflow(projectId, workflowId);
    }
  }, [onNavigateToWorkflow]);

  const handleViewExecution = useCallback((executionId: string, workflowId: string) => {
    if (onViewExecution) {
      onViewExecution(executionId, workflowId);
    }
  }, [onViewExecution]);

  const handleAIGenerate = useCallback((prompt: string) => {
    if (onAIGenerateWorkflow) {
      onAIGenerateWorkflow(prompt);
    }
  }, [onAIGenerateWorkflow]);

  const handleUseTemplate = useCallback((prompt: string, _templateName: string) => {
    if (onAIGenerateWorkflow) {
      onAIGenerateWorkflow(prompt);
    }
  }, [onAIGenerateWorkflow]);

  const handleRunWorkflow = useCallback((workflowId: string) => {
    if (onRunWorkflow) {
      onRunWorkflow(workflowId);
    }
  }, [onRunWorkflow]);

  const handleOpenSettings = useCallback(() => {
    if (onOpenSettings) {
      onOpenSettings();
    }
  }, [onOpenSettings]);

  const handleViewAllExecutions = useCallback(() => {
    setActiveTab('executions');
    if (onTabChange) {
      onTabChange('executions');
    }
  }, [onTabChange]);

  // Handle tab change
  const handleTabChange = useCallback((tab: DashboardTab) => {
    setActiveTab(tab);
    if (onTabChange) {
      onTabChange(tab);
    }
  }, [onTabChange]);

  // Status bar for API connection issues
  const StatusBar = () => {
    if (!error) return null;

    return (
      <div className="px-4 sm:px-6 py-4 bg-red-900/20 border-b border-red-500/30">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <WifiOff size={20} className="text-red-400" />
            <div>
              <div className="text-red-400 font-medium">
                API Connection Failed
              </div>
              <div className="text-red-300/80 text-sm">{error}</div>
            </div>
          </div>
          <button
            onClick={() => {
              clearError();
              fetchProjects();
            }}
            className="inline-flex items-center justify-center px-3 py-2 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  };


  // Render tab content based on active tab
  const renderTabContent = () => {
    switch (activeTab) {
      case 'home':
        return (
          <HomeTab
            onAIGenerate={handleAIGenerate}
            onCreateManual={onCreateProject}
            onNavigateToWorkflow={handleNavigateToWorkflow}
            onRunWorkflow={handleRunWorkflow}
            onViewExecution={handleViewExecution}
            onOpenSettings={handleOpenSettings}
            onUseTemplate={handleUseTemplate}
            onStartRecording={onStartRecording}
            isGenerating={isGeneratingWorkflow}
          />
        );
      case 'executions':
        return (
          <Suspense
            fallback={
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-flow-accent"></div>
              </div>
            }
          >
            <ExecutionsTab
              onViewExecution={handleViewExecution}
              onNavigateToHome={() => setActiveTab('home')}
              onCreateWorkflow={onCreateProject}
            />
          </Suspense>
        );
      case 'exports':
        return (
          <Suspense
            fallback={
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-flow-accent"></div>
              </div>
            }
          >
            <ExportsTab
              onViewExecution={handleViewExecution}
              onNavigateToWorkflow={handleNavigateToWorkflow}
              onNavigateToExecutions={() => setActiveTab('executions')}
              onNavigateToHome={() => setActiveTab('home')}
              onCreateWorkflow={onCreateProject}
              onOpenSettings={handleOpenSettings}
            />
          </Suspense>
        );
      case 'projects':
        return (
          <Suspense
            fallback={
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-flow-accent"></div>
              </div>
            }
          >
            <ProjectsTab
              onProjectSelect={onProjectSelect}
              onCreateProject={onCreateProject}
              onNavigateToWorkflow={handleNavigateToWorkflow}
              onRunWorkflow={handleRunWorkflow}
            />
          </Suspense>
        );
      case 'schedules':
        return (
          <Suspense
            fallback={
              <div className="flex items-center justify-center h-64">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-flow-accent"></div>
              </div>
            }
          >
            <SchedulesTab
              onNavigateToExecution={handleViewExecution}
              onNavigateToHome={() => setActiveTab('home')}
            />
          </Suspense>
        );
      default:
        return null;
    }
  };

  return (
    <div className="flex-1 flex flex-col min-h-[100svh] overflow-hidden bg-flow-bg">
      {/* Skip to content link for keyboard navigation */}
      <a
        href="#dashboard-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:px-4 focus:py-2 focus:bg-flow-accent focus:text-white focus:rounded-md"
      >
        Skip to content
      </a>

      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 sm:gap-4">
              <div className="flex items-center justify-center w-9 h-9 rounded-xl bg-flow-node/60 border border-flow-border/60">
                <img
                  src="/manifest-icon-192.maskable.png"
                  alt="Vrooli Ascension"
                  className="w-7 h-7"
                  loading="lazy"
                />
              </div>
              <h1 className="text-xl font-bold text-surface">
                Vrooli Ascension
              </h1>
            </div>
            <div className="flex items-center gap-2">
              {/* WebSocket Connection Indicator */}
              {!isWebSocketConnected && (
                <div
                  className="hidden sm:flex items-center gap-1.5 px-2 py-1 rounded text-xs text-amber-200 bg-amber-500/10 border border-amber-500/30"
                  title="Live updates temporarily unavailable"
                >
                  <WifiOff size={12} className="text-amber-300 animate-pulse" />
                  <span className="hidden lg:inline">Live updates offline</span>
                </div>
              )}

              {/* Running Executions Badge */}
              <RunningExecutionsBadge
                onViewExecution={handleViewExecution}
                onViewAllExecutions={handleViewAllExecutions}
              />

              {onStartRecording && (
                <button
                  onClick={onStartRecording}
                  className="hidden sm:flex items-center gap-2 px-3 py-1.5 text-[rgb(var(--flow-error))] bg-[rgba(var(--flow-error),0.12)] hover:bg-[rgba(var(--flow-error),0.2)] border border-[rgba(var(--flow-error),0.35)] rounded-lg transition-colors"
                  title="Start Record Mode"
                  aria-label="Start recording actions"
                >
                  <Circle size={12} className="text-red-400 fill-red-400" />
                  <span className="text-sm">Record</span>
                </button>
              )}

              {/* Global Search Button */}
              <button
                onClick={() => setIsSearchModalOpen(true)}
                className="hidden sm:flex items-center gap-2 px-3 py-1.5 text-subtle hover:text-surface bg-gray-800/50 hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors"
                title={`Search (${getModifierKey()}+K)`}
                aria-label="Search"
              >
                <Search size={14} />
                <span className="text-sm hidden md:inline">Search</span>
                <kbd
                  aria-hidden="true"
                  className="hidden lg:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-xs text-gray-500 bg-gray-800 border border-gray-700 rounded"
                >
                  <Command size={10} />K
                </kbd>
              </button>

              {/* Mobile search button */}
              <button
                onClick={() => setIsSearchModalOpen(true)}
                className="sm:hidden p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
                title="Search"
                aria-label="Search"
              >
                <Search size={18} />
              </button>

              {onOpenHelp && (
                <button
                  onClick={onOpenHelp}
                  className="hidden sm:flex p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
                  title="Help"
                  aria-label="Open help"
                  data-testid={selectors.dashboard.docsButton}
                >
                  <HelpCircle size={18} />
                </button>
              )}
              {onOpenSettings && (
                <button
                  onClick={onOpenSettings}
                  className="p-2 text-subtle hover:text-surface hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Settings (${getModifierKey()}+,)`}
                  aria-label="Open settings"
                  data-testid={selectors.dashboard.settingsButton}
                >
                  <Settings size={18} />
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Tab Navigation */}
        <TabNavigation
          activeTab={activeTab}
          onTabChange={handleTabChange}
          runningCount={runningExecutions.length}
        />
      </header>

      {/* Status Bar for API errors */}
      <StatusBar />

      {/* Content */}
      <div
        id="dashboard-content"
        className="flex-1 min-h-0 overflow-auto px-4 py-6 sm:px-6"
        role="region"
        aria-label="Dashboard content"
      >
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-flow-accent"></div>
          </div>
        ) : projects.length === 0 && activeTab === 'home' ? (
          <WelcomeHero
            onCreateFirstWorkflow={onCreateFirstWorkflow ?? onCreateProject}
            onOpenTutorial={onOpenTutorial}
            onStartRecording={onStartRecording}
          />
        ) : (
          renderTabContent()
        )}
      </div>

      {/* Floating Action Button - Mobile */}
      <button
        type="button"
        onClick={onCreateProject}
        className="md:hidden fixed bottom-6 right-6 z-40 w-14 h-14 bg-flow-accent text-white rounded-full shadow-lg hover:bg-blue-600 transition-all hover:shadow-xl flex items-center justify-center"
        aria-label="Create new project"
      >
        <Plus size={24} />
      </button>

      {/* Global Search Modal */}
      <GlobalSearchModal
        isOpen={isSearchModalOpen}
        onClose={() => setIsSearchModalOpen(false)}
        onSelectWorkflow={handleNavigateToWorkflow}
        onSelectProject={(projectId) => {
          const project = projects.find(p => p.id === projectId);
          if (project) onProjectSelect(project);
        }}
        onSelectExecution={handleViewExecution}
        onRunWorkflow={handleRunWorkflow}
      />
    </div>
  );
}

export default Dashboard;
