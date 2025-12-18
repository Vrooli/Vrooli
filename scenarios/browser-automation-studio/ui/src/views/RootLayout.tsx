/**
 * RootLayout - Provides shared context and UI for all routes.
 *
 * This component wraps all views with:
 * - ModalProvider for modal state management
 * - GuidedTour for onboarding
 * - DocsModal for help documentation
 * - Global keyboard shortcuts
 */
import { Suspense, useEffect } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { Toaster } from 'react-hot-toast';

// Modal and tour providers
import { ModalProvider, useModals } from '@shared/modals';
import { GuidedTour, useGuidedTour } from '@shared/onboarding';
import { DocsModal } from '@/domains/docs';
import { ProjectModal } from '@/domains/projects';

// Hooks
import { useAppShortcuts } from '@shared/hooks/useAppShortcuts';
import { useEntitlementInit } from '@hooks/useEntitlement';
import { useScheduleNotifications } from '@hooks/useScheduleNotifications';
import { useScenarioStore } from '@stores/scenarioStore';
import { useProjectStore, type Project } from '@/domains/projects';

// Shared UI
import { LoadingSpinner } from '@shared/ui';

// Types
import type { AppView } from '@/routing';

// Map route paths to view names for keyboard shortcuts context
function getViewFromPath(pathname: string): AppView {
  if (pathname === '/') return 'dashboard';
  if (pathname.startsWith('/settings')) return 'settings';
  if (pathname.startsWith('/record')) return 'record-mode';
  if (pathname.startsWith('/workflows')) return 'all-workflows';
  if (pathname.startsWith('/executions')) return 'all-executions';
  if (pathname.match(/^\/projects\/[^/]+\/workflows\//)) return 'project-workflow';
  if (pathname.match(/^\/projects\/[^/]+$/)) return 'project-detail';
  return 'dashboard';
}

/**
 * Inner layout that uses modal context
 */
function RootLayoutContent() {
  const location = useLocation();
  const navigate = useNavigate();
  const currentView = getViewFromPath(location.pathname);

  // Modal state from context
  const {
    showAIModal,
    showProjectModal,
    showDocs,
    docsInitialTab,
    closeAIModal,
    closeProjectModal,
    openDocs,
    closeDocs,
    openProjectModal,
    openAIModal,
  } = useModals();

  // Guided tour state
  const { showTour, openTour, closeTour } = useGuidedTour();

  // Project state
  const { currentProject, setCurrentProject } = useProjectStore();

  // Pre-fetch scenarios on mount for faster loading in NavigateNode
  const { fetchScenarios } = useScenarioStore();
  useEffect(() => {
    void fetchScenarios();
  }, [fetchScenarios]);

  // Initialize entitlement state
  useEntitlementInit();

  // Initialize schedule notifications
  useScheduleNotifications();

  // Navigation handlers using React Router
  const navigateToDashboard = () => navigate('/');
  const navigateToSettings = () => navigate('/settings');
  const handleStartRecording = () => navigate('/record/new');
  const openProject = (project: Project) => {
    setCurrentProject(project);
    navigate(`/projects/${project.id}`);
  };

  // Set up global keyboard shortcuts
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
    openProjectModal,
    openAIModal,
    openTour,
    navigateToDashboard,
    navigateToSettings,
    openProject,
    handleStartRecording,
    currentProject,
  });

  return (
    <div className="h-screen flex flex-col bg-flow-bg">
      <Suspense
        fallback={
          <div className="h-screen flex items-center justify-center bg-flow-bg">
            <LoadingSpinner variant="branded" size={32} message="Loading..." />
          </div>
        }
      >
        <Outlet />
      </Suspense>

      {/* Global modals rendered at root level */}
      <GuidedTour isOpen={showTour} onClose={closeTour} />

      <ProjectModal
        isOpen={showProjectModal}
        onClose={closeProjectModal}
        onSuccess={() => {
          closeProjectModal();
          // Navigation handled by the calling component
        }}
      />

      <DocsModal
        isOpen={showDocs}
        initialTab={docsInitialTab}
        onClose={closeDocs}
        onOpenTutorial={openTour}
      />

      <Toaster position="bottom-right" />
    </div>
  );
}

/**
 * Root layout wrapper that provides modal context
 */
export default function RootLayout() {
  const location = useLocation();
  const currentView = getViewFromPath(location.pathname);

  return (
    <ModalProvider currentView={currentView}>
      <RootLayoutContent />
    </ModalProvider>
  );
}
