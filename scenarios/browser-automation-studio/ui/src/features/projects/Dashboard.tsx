import { useState, useEffect, useCallback } from "react";
import {
  Plus,
  WifiOff,
  Wifi,
  Search,
  Keyboard,
  Settings,
  HelpCircle,
  BookOpen,
  Command,
} from "lucide-react";
import { useProjectStore, Project } from "@stores/projectStore";
import { useDashboardStore } from "@stores/dashboardStore";
import { useAICapabilityStore } from "@stores/aiCapabilityStore";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { selectors } from "@constants/selectors";
import { getModifierKey } from "@hooks/useKeyboardShortcuts";
import { GlobalSearchModal } from "@features/dashboard";
import { TabNavigation, type DashboardTab } from "@features/dashboard/TabNavigation";
import { HomeTab } from "@features/dashboard/HomeTab";
import { ExecutionsTab } from "@features/dashboard/ExecutionsTab";
import { ExportsTab } from "@features/dashboard/ExportsTab";
import { ProjectsTab } from "@features/dashboard/ProjectsTab";
import { RunningExecutionsBadge } from "@features/dashboard/RunningExecutionsBadge";
import { WelcomeHero } from "@features/dashboard/WelcomeHero";

interface DashboardProps {
  onProjectSelect: (project: Project) => void;
  onCreateProject: () => void;
  onCreateFirstWorkflow?: () => void;
  onShowKeyboardShortcuts?: () => void;
  onOpenSettings?: () => void;
  onOpenTutorial?: () => void;
  onOpenDocs?: () => void;
  onNavigateToWorkflow?: (projectId: string, workflowId: string) => void;
  onViewExecution?: (executionId: string, workflowId: string) => void;
  onAIGenerateWorkflow?: (prompt: string) => void;
  onRunWorkflow?: (workflowId: string) => void;
  onViewAllWorkflows?: () => void;
  onViewAllExecutions?: () => void;
  onTryDemo?: () => void;
  isGeneratingWorkflow?: boolean;
}

function Dashboard({
  onProjectSelect,
  onCreateProject,
  onCreateFirstWorkflow,
  onShowKeyboardShortcuts,
  onOpenSettings,
  onOpenTutorial,
  onOpenDocs,
  onNavigateToWorkflow,
  onViewExecution,
  onAIGenerateWorkflow,
  onRunWorkflow,
  onTryDemo,
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
  const { checkCapability: checkAICapability } = useAICapabilityStore();
  const { isConnected: isWebSocketConnected } = useWebSocket();
  const [activeTab, setActiveTab] = useState<DashboardTab>('home');
  const [isSearchModalOpen, setIsSearchModalOpen] = useState(false);

  useEffect(() => {
    fetchProjects();
    fetchRecentWorkflows();
    fetchRecentExecutions();
    fetchRunningExecutions();
    void checkAICapability();
  }, [fetchProjects, fetchRecentWorkflows, fetchRecentExecutions, fetchRunningExecutions, checkAICapability]);

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
    };
    window.addEventListener('navigate-to-exports', handleNavigateToExports);
    return () => window.removeEventListener('navigate-to-exports', handleNavigateToExports);
  }, []);

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
  }, []);

  // Handle tab change
  const handleTabChange = useCallback((tab: DashboardTab) => {
    setActiveTab(tab);
  }, []);

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
            isGenerating={isGeneratingWorkflow}
          />
        );
      case 'executions':
        return (
          <ExecutionsTab
            onViewExecution={handleViewExecution}
            onNavigateToHome={() => setActiveTab('home')}
            onCreateWorkflow={onCreateProject}
          />
        );
      case 'exports':
        return (
          <ExportsTab
            onViewExecution={handleViewExecution}
            onNavigateToWorkflow={handleNavigateToWorkflow}
            onNavigateToExecutions={() => setActiveTab('executions')}
            onNavigateToHome={() => setActiveTab('home')}
            onCreateWorkflow={onCreateProject}
          />
        );
      case 'projects':
        return (
          <ProjectsTab
            onProjectSelect={onProjectSelect}
            onCreateProject={onCreateProject}
            onNavigateToWorkflow={handleNavigateToWorkflow}
            onRunWorkflow={handleRunWorkflow}
            onTryDemo={onTryDemo}
          />
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
              <h1 className="text-xl font-bold text-white">
                Vrooli Ascension
              </h1>
            </div>
            <div className="flex items-center gap-2">
              {/* WebSocket Connection Indicator */}
              <div
                className={`hidden sm:flex items-center gap-1.5 px-2 py-1 rounded text-xs ${
                  isWebSocketConnected
                    ? 'text-green-400 bg-green-500/10'
                    : 'text-gray-500 bg-gray-800/50'
                }`}
                title={isWebSocketConnected ? 'Real-time updates active' : 'Connecting to real-time updates...'}
              >
                {isWebSocketConnected ? (
                  <Wifi size={12} className="text-green-400" />
                ) : (
                  <WifiOff size={12} className="text-gray-500 animate-pulse" />
                )}
                <span className="hidden lg:inline">
                  {isWebSocketConnected ? 'Live' : 'Connecting'}
                </span>
              </div>

              {/* Running Executions Badge */}
              <RunningExecutionsBadge
                onViewExecution={handleViewExecution}
                onViewAllExecutions={handleViewAllExecutions}
              />

              {/* Global Search Button */}
              <button
                onClick={() => setIsSearchModalOpen(true)}
                className="hidden sm:flex items-center gap-2 px-3 py-1.5 text-gray-400 hover:text-white bg-gray-800/50 hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors"
                title={`Search (${getModifierKey()}+K)`}
                aria-label="Open search"
              >
                <Search size={14} />
                <span className="text-sm hidden md:inline">Search</span>
                <kbd className="hidden lg:inline-flex items-center gap-0.5 px-1.5 py-0.5 text-xs text-gray-500 bg-gray-800 border border-gray-700 rounded">
                  <Command size={10} />K
                </kbd>
              </button>

              {/* Mobile search button */}
              <button
                onClick={() => setIsSearchModalOpen(true)}
                className="sm:hidden p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                title="Search"
                aria-label="Open search"
              >
                <Search size={18} />
              </button>

              {onOpenDocs && (
                <button
                  onClick={onOpenDocs}
                  className="hidden sm:flex p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title="Documentation"
                  aria-label="Open documentation"
                  data-testid={selectors.dashboard.docsButton}
                >
                  <BookOpen size={18} />
                </button>
              )}
              {onOpenTutorial && (
                <button
                  onClick={onOpenTutorial}
                  className="hidden sm:flex p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Tutorial (${getModifierKey()}+Shift+T)`}
                  aria-label="Open tutorial"
                >
                  <HelpCircle size={18} />
                </button>
              )}
              {onShowKeyboardShortcuts && (
                <button
                  onClick={onShowKeyboardShortcuts}
                  className="hidden sm:flex p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                  title={`Keyboard shortcuts (${getModifierKey()}+?)`}
                  aria-label="Show keyboard shortcuts"
                >
                  <Keyboard size={18} />
                </button>
              )}
              {onOpenSettings && (
                <button
                  onClick={onOpenSettings}
                  className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
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
