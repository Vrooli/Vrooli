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

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      {
        index: true,
        element: <DashboardViewWrapper />,
      },
      {
        path: 'projects/:projectId',
        element: <ProjectDetailView />,
      },
      {
        path: 'projects/:projectId/workflows/:workflowId',
        element: <WorkflowEditorView />,
      },
      {
        path: 'record',
        element: <RecordModeView />,
      },
      {
        path: 'record/new',
        element: <RecordModeView />,
      },
      {
        path: 'record/:sessionId',
        element: <RecordModeView />,
      },
      {
        path: 'settings',
        element: <SettingsView />,
      },
      {
        path: 'workflows',
        element: <AllWorkflowsView />,
      },
      {
        path: 'executions',
        element: <AllExecutionsView />,
      },
      // Legacy route: redirect schedules tab to dashboard
      {
        path: 'schedules',
        element: <DashboardViewWrapper initialTab="schedules" />,
      },
    ],
  },
]);

export default router;
