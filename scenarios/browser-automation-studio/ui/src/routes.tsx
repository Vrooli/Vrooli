/**
 * Route definitions for the Browser Automation Studio UI.
 *
 * This file defines all application routes using React Router.
 * Each route maps to a view component in the views/ directory.
 */
import { createBrowserRouter } from 'react-router-dom';
import { lazy } from 'react';

// Lazy load view components for better initial load performance
const DashboardViewWrapper = lazy(() =>
  import('@/views/DashboardView/DashboardViewWrapper')
);
const ProjectDetailView = lazy(() => import('@/views/ProjectDetailView'));
const WorkflowEditorView = lazy(() => import('@/views/WorkflowEditorView'));
const RecordModeView = lazy(() => import('@/views/RecordModeView'));
const SettingsView = lazy(() => import('@/views/SettingsView'));
const AllWorkflowsView = lazy(() => import('@/views/AllWorkflowsView'));
const AllExecutionsView = lazy(() => import('@/views/AllExecutionsView'));

// Import the root layout that provides shared context and UI
import RootLayout from '@/views/RootLayout';

// Import error boundary for route-level error handling
import ErrorBoundary from '@/shared/components/ErrorBoundary';
import SectionErrorBoundary from '@/shared/components/SectionErrorBoundary';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <ErrorBoundary />,
    children: [
      {
        index: true,
        element: (
          <SectionErrorBoundary title="Dashboard failed to load">
            <DashboardViewWrapper />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'projects/:projectId',
        element: (
          <SectionErrorBoundary title="Project view failed to load">
            <ProjectDetailView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'projects/:projectId/workflows/:workflowId',
        element: (
          <SectionErrorBoundary title="Workflow editor failed to load">
            <WorkflowEditorView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'record',
        element: (
          <SectionErrorBoundary title="Record mode failed to load">
            <RecordModeView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'record/new',
        element: (
          <SectionErrorBoundary title="Record mode failed to load">
            <RecordModeView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'record/:sessionId',
        element: (
          <SectionErrorBoundary title="Record mode failed to load">
            <RecordModeView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'settings',
        element: (
          <SectionErrorBoundary title="Settings failed to load">
            <SettingsView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'workflows',
        element: (
          <SectionErrorBoundary title="Workflows failed to load">
            <AllWorkflowsView />
          </SectionErrorBoundary>
        ),
      },
      {
        path: 'executions',
        element: (
          <SectionErrorBoundary title="Executions failed to load">
            <AllExecutionsView />
          </SectionErrorBoundary>
        ),
      },
      // Legacy route: redirect schedules tab to dashboard
      {
        path: 'schedules',
        element: (
          <SectionErrorBoundary title="Schedules failed to load">
            <DashboardViewWrapper initialTab="schedules" />
          </SectionErrorBoundary>
        ),
      },
    ],
  },
]);

export default router;
