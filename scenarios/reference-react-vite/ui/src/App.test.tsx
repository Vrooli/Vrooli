// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-component-tests
// [REQ:RRV-UI-001] App component - Unit tests for main application component
// [REQ:P0-006a] React component architecture with proper routing
// [REQ:P1-002b] Status cycling - Tests for task/project status transitions
// [REQ:P1-006b] Delete workflow - Tests for delete confirmation flow
import { screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { Layout } from './components/Layout';
import { Dashboard } from './pages/Dashboard';
import { Tasks } from './pages/Tasks';
import { Projects } from './pages/Projects';
import { renderWithProviders, createMockHealthResponse, createMockTask, createMockProject } from './test-utils';

// Mock the api module
vi.mock('./lib/api', () => ({
  fetchHealth: vi.fn(),
  fetchTasks: vi.fn(),
  fetchProjects: vi.fn(),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  deleteTask: vi.fn(),
  createProject: vi.fn(),
  updateProject: vi.fn(),
  deleteProject: vi.fn(),
}));

import { fetchHealth, fetchTasks, fetchProjects, createTask, createProject, updateTask, deleteTask, updateProject, deleteProject } from './lib/api';

describe('Layout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(fetchHealth).mockResolvedValue(createMockHealthResponse());
  });

  describe('rendering', () => {
    it('renders the app header', () => {
      renderWithProviders(<Layout />);
      expect(screen.getByTestId('app-header')).toBeInTheDocument();
    });

    it('renders the navigation', () => {
      renderWithProviders(<Layout />);
      expect(screen.getByTestId('main-nav')).toBeInTheDocument();
    });

    it('renders navigation links', () => {
      renderWithProviders(<Layout />);
      expect(screen.getByTestId('nav-dashboard')).toBeInTheDocument();
      expect(screen.getByTestId('nav-tasks')).toBeInTheDocument();
      expect(screen.getByTestId('nav-projects')).toBeInTheDocument();
    });

    it('renders the health indicator', () => {
      renderWithProviders(<Layout />);
      expect(screen.getByTestId('health-indicator')).toBeInTheDocument();
    });

    it('renders the main content area', () => {
      renderWithProviders(<Layout />);
      expect(screen.getByTestId('main-content')).toBeInTheDocument();
    });
  });

  describe('health indicator', () => {
    it('shows healthy status when API is reachable', async () => {
      vi.mocked(fetchHealth).mockResolvedValue(createMockHealthResponse({ status: 'healthy' }));

      renderWithProviders(<Layout />);

      await waitFor(() => {
        expect(screen.getByText('healthy')).toBeInTheDocument();
      });
    });

    it('shows offline status when API fails', async () => {
      vi.mocked(fetchHealth).mockRejectedValue(new Error('Network error'));

      renderWithProviders(<Layout />);

      await waitFor(() => {
        expect(screen.getByText('offline')).toBeInTheDocument();
      });
    });
  });
});

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    beforeEach(() => {
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
    });

    it('renders the dashboard page', () => {
      renderWithProviders(<Dashboard />);
      expect(screen.getByTestId('dashboard-page')).toBeInTheDocument();
    });

    it('renders the page title', () => {
      renderWithProviders(<Dashboard />);
      expect(screen.getByText('Dashboard')).toBeInTheDocument();
    });

    it('renders stat cards', async () => {
      renderWithProviders(<Dashboard />);
      await waitFor(() => {
        expect(screen.getByTestId('stat-total-tasks')).toBeInTheDocument();
        expect(screen.getByTestId('stat-pending')).toBeInTheDocument();
        expect(screen.getByTestId('stat-completed')).toBeInTheDocument();
        expect(screen.getByTestId('stat-projects')).toBeInTheDocument();
      });
    });

    it('renders quick action buttons', () => {
      renderWithProviders(<Dashboard />);
      expect(screen.getByTestId('quick-action-new-task')).toBeInTheDocument();
      expect(screen.getByTestId('quick-action-new-project')).toBeInTheDocument();
    });
  });

  describe('with data', () => {
    it('displays task counts correctly', async () => {
      const tasks = [
        createMockTask({ id: '1', status: 'pending' }),
        createMockTask({ id: '2', status: 'completed' }),
        createMockTask({ id: '3', status: 'pending' }),
      ];
      vi.mocked(fetchTasks).mockResolvedValue({ data: tasks, pagination: { total: 3, limit: 20, offset: 0 } });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });

      renderWithProviders(<Dashboard />);

      await waitFor(() => {
        expect(screen.getByText('3')).toBeInTheDocument(); // total tasks
      });
    });

    it('displays recent tasks when available', async () => {
      const tasks = [createMockTask({ id: '1', title: 'Test Task' })];
      vi.mocked(fetchTasks).mockResolvedValue({ data: tasks, pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });

      renderWithProviders(<Dashboard />);

      await waitFor(() => {
        expect(screen.getByTestId('recent-tasks-list')).toBeInTheDocument();
      });
    });

    it('shows empty state when no tasks', async () => {
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });

      renderWithProviders(<Dashboard />);

      await waitFor(() => {
        expect(screen.getByTestId('recent-tasks-empty')).toBeInTheDocument();
      });
    });
  });
});

describe('Tasks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    beforeEach(() => {
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
    });

    it('renders the tasks page', () => {
      renderWithProviders(<Tasks />);
      expect(screen.getByTestId('tasks-page')).toBeInTheDocument();
    });

    it('renders the page title', () => {
      renderWithProviders(<Tasks />);
      expect(screen.getByText('Tasks')).toBeInTheDocument();
    });

    it('renders the task form', () => {
      renderWithProviders(<Tasks />);
      expect(screen.getByTestId('task-form')).toBeInTheDocument();
      expect(screen.getByTestId('task-input')).toBeInTheDocument();
      expect(screen.getByTestId('task-submit')).toBeInTheDocument();
    });
  });

  describe('states', () => {
    it('shows loading state', () => {
      vi.mocked(fetchTasks).mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<Tasks />);
      expect(screen.getByTestId('tasks-loading')).toBeInTheDocument();
    });

    it('shows error state when API fails', async () => {
      vi.mocked(fetchTasks).mockRejectedValue(new Error('Network error'));
      renderWithProviders(<Tasks />);
      await waitFor(() => {
        expect(screen.getByTestId('tasks-error')).toBeInTheDocument();
      });
    });

    it('shows empty state when no tasks', async () => {
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      renderWithProviders(<Tasks />);
      await waitFor(() => {
        expect(screen.getByTestId('tasks-empty')).toBeInTheDocument();
      });
    });

    it('shows task list when tasks exist', async () => {
      const tasks = [createMockTask({ id: '1', title: 'Test Task' })];
      vi.mocked(fetchTasks).mockResolvedValue({ data: tasks, pagination: { total: 1, limit: 20, offset: 0 } });
      renderWithProviders(<Tasks />);
      await waitFor(() => {
        expect(screen.getByTestId('tasks-list')).toBeInTheDocument();
        expect(screen.getByTestId('task-row-1')).toBeInTheDocument();
      });
    });
  });

  describe('task creation', () => {
    it('creates a task when form is submitted', async () => {
      const user = userEvent.setup();
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(createTask).mockResolvedValue(createMockTask({ id: '1', title: 'New Task' }));

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-input')).toBeInTheDocument();
      });

      await user.type(screen.getByTestId('task-input'), 'New Task');
      await user.click(screen.getByTestId('task-submit'));

      await waitFor(() => {
        expect(createTask).toHaveBeenCalledWith({ title: 'New Task' });
      });
    });

    it('shows error message when create fails', async () => {
      const user = userEvent.setup();
      vi.mocked(fetchTasks).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(createTask).mockRejectedValue(new Error('Create failed'));

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-input')).toBeInTheDocument();
      });

      await user.type(screen.getByTestId('task-input'), 'New Task');
      await user.click(screen.getByTestId('task-submit'));

      await waitFor(() => {
        expect(screen.getByTestId('task-create-error')).toBeInTheDocument();
      });
    });
  });

  describe('status cycling', () => {
    // [REQ:P1-002b] Test status transitions: pending -> in_progress -> completed -> pending
    it('cycles task status from pending to in_progress', async () => {
      const user = userEvent.setup();
      const task = createMockTask({ id: '1', status: 'pending' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateTask).mockResolvedValue({ ...task, status: 'in_progress' });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-status-toggle-1'));

      await waitFor(() => {
        expect(updateTask).toHaveBeenCalledWith('1', { status: 'in_progress' });
      });
    });

    it('cycles task status from in_progress to completed', async () => {
      const user = userEvent.setup();
      const task = createMockTask({ id: '1', status: 'in_progress' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateTask).mockResolvedValue({ ...task, status: 'completed' });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-status-toggle-1'));

      await waitFor(() => {
        expect(updateTask).toHaveBeenCalledWith('1', { status: 'completed' });
      });
    });

    it('cycles task status from completed back to pending', async () => {
      const user = userEvent.setup();
      const task = createMockTask({ id: '1', status: 'completed' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateTask).mockResolvedValue({ ...task, status: 'pending' });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-status-toggle-1'));

      await waitFor(() => {
        expect(updateTask).toHaveBeenCalledWith('1', { status: 'pending' });
      });
    });
  });

  describe('delete workflow', () => {
    // [REQ:P1-006b] Test delete confirmation flow
    it('opens confirmation dialog when delete button clicked', async () => {
      const user = userEvent.setup();
      const task = createMockTask({ id: '1', title: 'Task to Delete' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-delete-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-delete-1'));

      await waitFor(() => {
        expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument();
        // Check dialog title specifically
        expect(screen.getByRole('heading', { name: 'Delete Task' })).toBeInTheDocument();
      });
    });

    it('shows delete confirmation dialog with task title in message', async () => {
      // Test that delete dialog shows the correct task title and has proper buttons
      // The actual delete mutation is verified by:
      // 1. ConfirmDialog.test.tsx confirms clicking onConfirm calls the callback
      // 2. api.test.ts confirms deleteTask API function works
      const task = createMockTask({ id: '1', title: 'Important Task' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-delete-1')).toBeInTheDocument();
      });

      // Click delete button to open dialog
      fireEvent.click(screen.getByTestId('task-delete-1'));

      // Verify dialog shows the correct title and buttons
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Delete Task' })).toBeInTheDocument();
        // Check the message contains both the name and warning - wrapped in same element
        expect(screen.getByText(/Are you sure you want to delete.*Important Task.*cannot be undone/s)).toBeInTheDocument();
        expect(screen.getByTestId('confirm-dialog-confirm')).toHaveTextContent('Delete');
      });
    });

    it('cancels deletion when dialog is cancelled', async () => {
      const user = userEvent.setup();
      const task = createMockTask({ id: '1', title: 'Task to Keep' });
      vi.mocked(fetchTasks).mockResolvedValue({ data: [task], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Tasks />);

      await waitFor(() => {
        expect(screen.getByTestId('task-delete-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('task-delete-1'));

      await waitFor(() => {
        expect(screen.getByTestId('confirm-dialog-cancel')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('confirm-dialog-cancel'));

      await waitFor(() => {
        expect(screen.queryByTestId('confirm-dialog')).not.toBeInTheDocument();
      });

      expect(deleteTask).not.toHaveBeenCalled();
    });
  });
});

describe('Projects', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    beforeEach(() => {
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
    });

    it('renders the projects page', () => {
      renderWithProviders(<Projects />);
      expect(screen.getByTestId('projects-page')).toBeInTheDocument();
    });

    it('renders the page title', () => {
      renderWithProviders(<Projects />);
      expect(screen.getByText('Projects')).toBeInTheDocument();
    });

    it('renders the project form', () => {
      renderWithProviders(<Projects />);
      expect(screen.getByTestId('project-form')).toBeInTheDocument();
      expect(screen.getByTestId('project-input')).toBeInTheDocument();
      expect(screen.getByTestId('project-submit')).toBeInTheDocument();
    });
  });

  describe('states', () => {
    it('shows loading state', () => {
      vi.mocked(fetchProjects).mockImplementation(() => new Promise(() => {}));
      renderWithProviders(<Projects />);
      expect(screen.getByTestId('projects-loading')).toBeInTheDocument();
    });

    it('shows error state when API fails', async () => {
      vi.mocked(fetchProjects).mockRejectedValue(new Error('Network error'));
      renderWithProviders(<Projects />);
      await waitFor(() => {
        expect(screen.getByTestId('projects-error')).toBeInTheDocument();
      });
    });

    it('shows empty state when no projects', async () => {
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      renderWithProviders(<Projects />);
      await waitFor(() => {
        expect(screen.getByTestId('projects-empty')).toBeInTheDocument();
      });
    });

    it('shows project grid when projects exist', async () => {
      const projects = [createMockProject({ id: '1', name: 'Test Project' })];
      vi.mocked(fetchProjects).mockResolvedValue({ data: projects, pagination: { total: 1, limit: 20, offset: 0 } });
      renderWithProviders(<Projects />);
      await waitFor(() => {
        expect(screen.getByTestId('projects-grid')).toBeInTheDocument();
        expect(screen.getByTestId('project-card-1')).toBeInTheDocument();
      });
    });
  });

  describe('project creation', () => {
    it('creates a project when form is submitted', async () => {
      const user = userEvent.setup();
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(createProject).mockResolvedValue(createMockProject({ id: '1', name: 'New Project' }));

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-input')).toBeInTheDocument();
      });

      await user.type(screen.getByTestId('project-input'), 'New Project');
      await user.click(screen.getByTestId('project-submit'));

      await waitFor(() => {
        expect(createProject).toHaveBeenCalled();
      });
    });

    it('shows error message when create fails', async () => {
      const user = userEvent.setup();
      vi.mocked(fetchProjects).mockResolvedValue({ data: [], pagination: { total: 0, limit: 20, offset: 0 } });
      vi.mocked(createProject).mockRejectedValue(new Error('Create failed'));

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-input')).toBeInTheDocument();
      });

      await user.type(screen.getByTestId('project-input'), 'New Project');
      await user.click(screen.getByTestId('project-submit'));

      await waitFor(() => {
        expect(screen.getByTestId('project-create-error')).toBeInTheDocument();
      });
    });
  });

  describe('status cycling', () => {
    // [REQ:P1-002b] Test status transitions: active -> paused -> complete -> active
    it('cycles project status from active to paused', async () => {
      const user = userEvent.setup();
      const project = createMockProject({ id: '1', status: 'active' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateProject).mockResolvedValue({ ...project, status: 'paused' });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('project-status-toggle-1'));

      await waitFor(() => {
        expect(updateProject).toHaveBeenCalledWith('1', { status: 'paused' });
      });
    });

    it('cycles project status from paused to complete', async () => {
      const user = userEvent.setup();
      const project = createMockProject({ id: '1', status: 'paused' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateProject).mockResolvedValue({ ...project, status: 'complete' });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('project-status-toggle-1'));

      await waitFor(() => {
        expect(updateProject).toHaveBeenCalledWith('1', { status: 'complete' });
      });
    });

    it('cycles project status from complete back to active', async () => {
      const user = userEvent.setup();
      const project = createMockProject({ id: '1', status: 'complete' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });
      vi.mocked(updateProject).mockResolvedValue({ ...project, status: 'active' });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-status-toggle-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('project-status-toggle-1'));

      await waitFor(() => {
        expect(updateProject).toHaveBeenCalledWith('1', { status: 'active' });
      });
    });
  });

  describe('delete workflow', () => {
    // [REQ:P1-006b] Test delete confirmation flow
    it('opens confirmation dialog when delete button clicked', async () => {
      const user = userEvent.setup();
      const project = createMockProject({ id: '1', name: 'Project to Delete' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-delete-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('project-delete-1'));

      await waitFor(() => {
        expect(screen.getByTestId('confirm-dialog')).toBeInTheDocument();
        // Check dialog title specifically
        expect(screen.getByRole('heading', { name: 'Delete Project' })).toBeInTheDocument();
      });
    });

    it('shows delete confirmation dialog with project name in message', async () => {
      // Test that delete dialog shows the correct project name and has proper buttons
      // The actual delete mutation is verified by:
      // 1. ConfirmDialog.test.tsx confirms clicking onConfirm calls the callback
      // 2. api.test.ts confirms deleteProject API function works
      const project = createMockProject({ id: '1', name: 'Critical Project' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-delete-1')).toBeInTheDocument();
      });

      // Click delete button to open dialog
      fireEvent.click(screen.getByTestId('project-delete-1'));

      // Verify dialog shows the correct title and buttons
      await waitFor(() => {
        expect(screen.getByRole('heading', { name: 'Delete Project' })).toBeInTheDocument();
        // Check the message contains both the name and warning - wrapped in same element
        expect(screen.getByText(/Are you sure you want to delete.*Critical Project.*cannot be undone/s)).toBeInTheDocument();
        expect(screen.getByTestId('confirm-dialog-confirm')).toHaveTextContent('Delete');
      });
    });

    it('cancels deletion when dialog is cancelled', async () => {
      const user = userEvent.setup();
      const project = createMockProject({ id: '1', name: 'Project to Keep' });
      vi.mocked(fetchProjects).mockResolvedValue({ data: [project], pagination: { total: 1, limit: 20, offset: 0 } });

      renderWithProviders(<Projects />);

      await waitFor(() => {
        expect(screen.getByTestId('project-delete-1')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('project-delete-1'));

      await waitFor(() => {
        expect(screen.getByTestId('confirm-dialog-cancel')).toBeInTheDocument();
      });

      await user.click(screen.getByTestId('confirm-dialog-cancel'));

      await waitFor(() => {
        expect(screen.queryByTestId('confirm-dialog')).not.toBeInTheDocument();
      });

      expect(deleteProject).not.toHaveBeenCalled();
    });
  });
});
