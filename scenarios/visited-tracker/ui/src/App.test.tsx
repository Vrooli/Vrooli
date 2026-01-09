import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, fireEvent, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import * as api from './lib/api';

// [REQ:VT-REQ-009] Web interface dashboard application

// Mock API
vi.mock('./lib/api', () => ({
  fetchCampaigns: vi.fn(),
  createCampaign: vi.fn(),
  deleteCampaign: vi.fn(),
  fetchCampaign: vi.fn(),
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

const mockCampaign = {
  id: '1',
  name: 'Test Campaign',
  from_agent: 'test-agent',
  status: 'active',
  location: '/test',
  pattern: '**/*.ts',
  patterns: ['**/*.ts'],
  tags: ['test'],
  total_files: 10,
  visited_files: 5,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset window location to root path for each test
    window.history.pushState({}, '', '/');
  });

  afterEach(() => {
    cleanup();
  });

  describe('Application rendering', () => {
    it('should render the application', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Visited Tracker')).toBeInTheDocument();
      });
    });

    it('should render within tooltip provider', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      const { container } = render(<App />, { wrapper: createWrapper() });

      // TooltipProvider wraps the app, so the app should render without errors
      await waitFor(() => {
        expect(container).toBeTruthy();
      });
    });

    it('should have proper document structure', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Should have heading
        expect(screen.getByRole('heading', { name: /visited tracker/i })).toBeInTheDocument();

        // Should have interactive elements (button shows "New" on small screens, "New Campaign" on larger)
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });
    });
  });

  describe('View management', () => {
    it('should start with list view', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Welcome to Visited Tracker')).toBeInTheDocument();
      });
    });

    it('should switch to detail view when campaign is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaign],
      });
      vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Test Campaign')).toBeInTheDocument();
      });

      const campaignCard = screen.getByText('Test Campaign');
      await user.click(campaignCard);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument();
      });
    });

    it('should return to list view when back button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaign],
      });
      vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Test Campaign')).toBeInTheDocument();
      });

      // Go to detail view
      const campaignCard = screen.getByText('Test Campaign');
      await user.click(campaignCard);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument();
      });

      // Go back to list view
      const backButton = screen.getByRole('button', { name: /back/i });
      await user.click(backButton);

      await waitFor(() => {
        expect(screen.getByText('Test Campaign')).toBeInTheDocument();
        expect(screen.queryByRole('button', { name: /back/i })).not.toBeInTheDocument();
      });
    });
  });

  describe('Campaign creation', () => {
    it('should open create dialog when new campaign button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });
    });

    it('should create campaign and switch to detail view on success', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });
      vi.mocked(api.createCampaign).mockResolvedValue(mockCampaign);
      vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      // Open create dialog
      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });

      // Fill in form
      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Submit form
      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      // Should switch to detail view
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument();
      });
    });

    it('should show success toast when campaign is created', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });
      vi.mocked(api.createCampaign).mockResolvedValue(mockCampaign);
      vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      // Open create dialog
      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });

      // Fill in form
      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Submit form
      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      // Should show success toast
      await waitFor(() => {
        expect(screen.getByText(/created successfully/i)).toBeInTheDocument();
      });
    });

    it('should show error toast when campaign creation fails', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });
      vi.mocked(api.createCampaign).mockRejectedValue(new Error('Creation failed'));

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      // Open create dialog
      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });

      // Fill in form
      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Submit form
      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      // Should show error toast
      await waitFor(() => {
        expect(screen.getByText(/failed to create campaign/i)).toBeInTheDocument();
      });
    });

    it('should close dialog and remain on list view on cancel', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      // Open create dialog
      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });

      // Cancel
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      await waitFor(() => {
        expect(screen.queryByText('Create New Campaign')).not.toBeInTheDocument();
      });

      // Still on list view
      expect(screen.getByText('Welcome to Visited Tracker')).toBeInTheDocument();
    });
  });

  describe('Keyboard shortcuts', () => {
    it('should open help dialog when "?" is pressed', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Welcome to Visited Tracker')).toBeInTheDocument();
      });

      fireEvent.keyDown(window, { key: '?' });

      await waitFor(() => {
        // Check for the dialog title specifically (it's a heading)
        expect(screen.getByRole('heading', { name: /keyboard shortcuts/i })).toBeInTheDocument();
      });
    });

    it('should not open help dialog when "?" is pressed in input field', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByPlaceholderText('Search campaigns...')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.click(searchInput);
      await user.type(searchInput, '?');

      // Help dialog should not open
      expect(screen.queryByText(/keyboard shortcuts/i)).not.toBeInTheDocument();
      // Search input should contain '?'
      expect(searchInput).toHaveValue('?');
    });

    it('should not open help dialog when on detail view', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaign],
      });
      vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Test Campaign')).toBeInTheDocument();
      });

      // Go to detail view
      const campaignCard = screen.getByText('Test Campaign');
      await user.click(campaignCard);

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /back/i })).toBeInTheDocument();
      });

      // Press "?" - should not open help
      fireEvent.keyDown(window, { key: '?' });

      await waitFor(() => {
        expect(screen.queryByText(/keyboard shortcuts/i)).not.toBeInTheDocument();
      });
    });
  });

  describe('Loading states', () => {
    it('should show loading indicator during campaign creation', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });
      vi.mocked(api.createCampaign).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      render(<App />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new/i })).toBeInTheDocument();
      });

      // Open create dialog
      const newButton = screen.getByRole('button', { name: /new/i });
      await user.click(newButton);

      await waitFor(() => {
        expect(screen.getByText('Create New Campaign')).toBeInTheDocument();
      });

      // Fill in form
      const nameInput = screen.getByLabelText(/campaign name/i);
      await user.type(nameInput, 'Test Campaign');

      const patternsInput = screen.getByLabelText(/file patterns/i);
      await user.type(patternsInput, '**/*.ts');

      // Submit form
      const createButton = screen.getByRole('button', { name: /create campaign/i });
      await user.click(createButton);

      // Should show loading state
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /creating/i })).toBeInTheDocument();
      });
    });
  });
});
