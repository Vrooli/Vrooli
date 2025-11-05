import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../test-utils/testHelpers';
import ProjectModal from '../ProjectModal';
import { useProjectStore } from '../../stores/projectStore';

// Mock the project store
vi.mock('../../stores/projectStore', () => ({
  useProjectStore: vi.fn(),
}));

describe('ProjectModal [REQ:BAS-PROJECT-CREATE-SUCCESS] [REQ:BAS-PROJECT-CREATE-VALIDATION]', () => {
  const mockCreateProject = vi.fn();
  const mockUpdateProject = vi.fn();
  const mockOnClose = vi.fn();
  const mockOnSuccess = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();

    // Setup default store mock
    (useProjectStore as unknown as ReturnType<typeof vi.fn>).mockReturnValue({
      createProject: mockCreateProject,
      updateProject: mockUpdateProject,
      isLoading: false,
      error: null,
    });
  });

  it('renders create mode when no project is provided [REQ:BAS-PROJECT-DIALOG-OPEN]', () => {
    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    expect(screen.getByTestId('project-modal')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/visited tracker tests/i)).toHaveValue('');
  });

  it('renders edit mode when project is provided', () => {
    const existingProject = {
      id: 'test-id',
      name: 'Existing Project',
      description: 'Test Description',
      folder_path: '/test/path',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    };

    renderWithProviders(<ProjectModal onClose={mockOnClose} project={existingProject} />);

    expect(screen.getByPlaceholderText(/visited tracker tests/i)).toHaveValue('Existing Project');
    expect(screen.getByPlaceholderText(/describe what this project/i)).toHaveValue('Test Description');
  });

  it('validates required fields [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    // Try to submit without filling required fields
    const submitButton = screen.getByRole('button', { name: /create project/i });
    await user.click(submitButton);

    // Should show validation errors
    await waitFor(() => {
      expect(screen.getByText(/project name is required/i)).toBeInTheDocument();
    });

    expect(mockCreateProject).not.toHaveBeenCalled();
  });

  it('validates minimum name length [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    const nameInput = screen.getByPlaceholderText(/visited tracker tests/i);
    await user.clear(nameInput);
    await user.type(nameInput, 'ab'); // Less than 3 characters

    const submitButton = screen.getByRole('button', { name: /create project/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/must be at least 3 characters/i)).toBeInTheDocument();
    });

    expect(mockCreateProject).not.toHaveBeenCalled();
  });

  it('validates folder path format [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    const nameInput = screen.getByPlaceholderText(/visited tracker tests/i);
    const folderInput = screen.getByPlaceholderText(/\/path\/to\/project\/folder/i);

    await user.type(nameInput, 'Valid Project Name');
    await user.clear(folderInput);
    await user.type(folderInput, 'relative/path'); // Not absolute

    const submitButton = screen.getByRole('button', { name: /create project/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText(/must be an absolute path/i)).toBeInTheDocument();
    });

    expect(mockCreateProject).not.toHaveBeenCalled();
  });

  it('creates project successfully with valid data [REQ:BAS-PROJECT-CREATE-SUCCESS]', async () => {
    const user = userEvent.setup();
    const newProject = {
      id: 'new-id',
      name: 'New Test Project',
      description: 'Test Description',
      folder_path: '/test/path',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    };

    mockCreateProject.mockResolvedValue(newProject);

    renderWithProviders(<ProjectModal onClose={mockOnClose} onSuccess={mockOnSuccess} />);

    const nameInput = screen.getByPlaceholderText(/visited tracker tests/i);
    const descriptionInput = screen.getByPlaceholderText(/describe what this project/i);

    await user.type(nameInput, 'New Test Project');
    await user.type(descriptionInput, 'Test Description');

    const submitButton = screen.getByRole('button', { name: /create project/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockCreateProject).toHaveBeenCalledWith({
        name: 'New Test Project',
        description: 'Test Description',
        folder_path: expect.stringContaining('/'),
      });
    });

    expect(mockOnSuccess).toHaveBeenCalledWith(newProject);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('updates existing project successfully [REQ:BAS-PROJECT-CREATE-SUCCESS]', async () => {
    const user = userEvent.setup();
    const existingProject = {
      id: 'existing-id',
      name: 'Original Name',
      description: 'Original Description',
      folder_path: '/original/path',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    };

    const updatedProject = {
      ...existingProject,
      name: 'Updated Name',
      updated_at: '2025-01-02',
    };

    mockUpdateProject.mockResolvedValue(updatedProject);

    renderWithProviders(
      <ProjectModal onClose={mockOnClose} onSuccess={mockOnSuccess} project={existingProject} />
    );

    const nameInput = screen.getByPlaceholderText(/visited tracker tests/i);
    await user.clear(nameInput);
    await user.type(nameInput, 'Updated Name');

    const submitButton = screen.getByRole('button', { name: /update project/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockUpdateProject).toHaveBeenCalledWith('existing-id', {
        name: 'Updated Name',
        description: 'Original Description',
        folder_path: '/original/path',
      });
    });

    expect(mockOnSuccess).toHaveBeenCalledWith(updatedProject);
    expect(mockOnClose).toHaveBeenCalled();
  });

  it('closes dialog when cancel button is clicked [REQ:BAS-PROJECT-DIALOG-CLOSE]', async () => {
    const user = userEvent.setup();
    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    await user.click(cancelButton);

    expect(mockOnClose).toHaveBeenCalled();
    expect(mockCreateProject).not.toHaveBeenCalled();
  });

  it('shows loading state during submission', async () => {
    const user = userEvent.setup();

    // Make the mock return a pending promise
    let resolveCreate: (value: unknown) => void;
    const createPromise = new Promise((resolve) => {
      resolveCreate = resolve;
    });
    mockCreateProject.mockReturnValue(createPromise);

    renderWithProviders(<ProjectModal onClose={mockOnClose} />);

    const nameInput = screen.getByPlaceholderText(/visited tracker tests/i);
    await user.type(nameInput, 'Test Project');

    const submitButton = screen.getByRole('button', { name: /create project/i });
    await user.click(submitButton);

    // Button should be disabled during submission
    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });

    // Resolve the promise
    resolveCreate!({
      id: 'new-id',
      name: 'Test Project',
      folder_path: '/test/path',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    });

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalled();
    });
  });
});
