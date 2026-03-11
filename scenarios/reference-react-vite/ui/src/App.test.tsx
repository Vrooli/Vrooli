// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md#ui-component-tests
// [REQ:RRV-UI-001] App component - Unit tests for main application component
// [REQ:P0-006a] React component architecture with proper routing
import { screen, waitFor } from '@testing-library/react';
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

import { fetchHealth, fetchTasks, fetchProjects, createTask, createProject } from './lib/api';

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
  });
});
