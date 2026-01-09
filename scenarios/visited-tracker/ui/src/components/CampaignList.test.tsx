import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CampaignList } from './CampaignList';
import { ToastProvider } from './ui/toast';
import * as api from '../lib/api';

// [REQ:VT-REQ-009] Web interface dashboard with campaign management

// Mock API
vi.mock('../lib/api', () => ({
  fetchCampaigns: vi.fn(),
  deleteCampaign: vi.fn(),
}));

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        {children}
      </ToastProvider>
    </QueryClientProvider>
  );
};

const mockCampaigns = [
  {
    id: '1',
    name: 'TypeScript Files',
    from_agent: 'test-agent',
    status: 'active',
    location: '/test',
    pattern: '**/*.ts',
    tags: ['typescript'],
    total_files: 10,
    visited_files: 5,
    created_at: new Date('2025-01-01').toISOString(),
    updated_at: new Date('2025-01-02').toISOString(),
  },
  {
    id: '2',
    name: 'React Components',
    from_agent: 'ui-agent',
    status: 'completed',
    location: '/src',
    pattern: '**/*.tsx',
    tags: ['react'],
    total_files: 20,
    visited_files: 20,
    created_at: new Date('2025-01-03').toISOString(),
    updated_at: new Date('2025-01-04').toISOString(),
  },
  {
    id: '3',
    name: 'API Routes',
    from_agent: 'test-agent',
    status: 'active',
    location: '/api',
    pattern: '**/*.go',
    description: 'Track API route files',
    tags: ['api', 'golang'],
    total_files: 15,
    visited_files: 3,
    created_at: new Date('2025-01-05').toISOString(),
    updated_at: new Date('2025-01-06').toISOString(),
  },
];

describe('CampaignList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Empty state', () => {
    it('should render empty state for new users', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Welcome to Visited Tracker')).toBeInTheDocument();
      });
    });

    it('should display agent onboarding with CLI examples', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText(/Quick Start for Agents/i)).toBeInTheDocument();
      });
      // CLI examples are in code blocks
      const codeBlocks = screen.getAllByText(/visited-tracker/);
      expect(codeBlocks.length).toBeGreaterThan(0);
    });
  });

  describe('Campaign list display', () => {
    it('should render stats cards with correct data', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('Total Campaigns')).toBeInTheDocument();
        expect(screen.getByText('Active Campaigns')).toBeInTheDocument();
        expect(screen.getByText('Tracked Files')).toBeInTheDocument();

        // Verify stats by ID since numbers appear multiple times
        const totalStat = document.getElementById('stat-total');
        const activeStat = document.getElementById('stat-active');
        const filesStat = document.getElementById('stat-files');

        expect(totalStat).toHaveTextContent('3');
        expect(activeStat).toHaveTextContent('2');
        expect(filesStat).toHaveTextContent('45');
      });
    });

    it('should display all campaigns in the list', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
        expect(screen.getByText('React Components')).toBeInTheDocument();
        expect(screen.getByText('API Routes')).toBeInTheDocument();
      });
    });

    it('should show loading skeleton while fetching', () => {
      vi.mocked(api.fetchCampaigns).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      // Should show skeleton loaders
      const skeletons = document.querySelectorAll('[data-testid="campaign-skeleton"]');
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it('should display error state when fetch fails', async () => {
      vi.mocked(api.fetchCampaigns).mockRejectedValue(new Error('Network error'));

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText(/failed to load campaigns/i)).toBeInTheDocument();
      });
    });
  });

  describe('Search functionality', () => {
    it('should have search input with proper placeholder', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        const searchInput = screen.getByPlaceholderText('Search campaigns...');
        expect(searchInput).toBeInTheDocument();
      });
    });

    it('should filter campaigns by name', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.type(searchInput, 'TypeScript');

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
        expect(screen.queryByText('React Components')).not.toBeInTheDocument();
        expect(screen.queryByText('API Routes')).not.toBeInTheDocument();
      });
    });

    it('should filter campaigns by description', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('API Routes')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.type(searchInput, 'route files');

      await waitFor(() => {
        expect(screen.getByText('API Routes')).toBeInTheDocument();
        expect(screen.queryByText('TypeScript Files')).not.toBeInTheDocument();
      });
    });

    it('should clear search when X button is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.type(searchInput, 'TypeScript');

      await waitFor(() => {
        expect(screen.queryByText('React Components')).not.toBeInTheDocument();
      });

      // Find and click clear button
      const clearButton = screen.getByRole('button', { name: /clear search/i });
      await user.click(clearButton);

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
        expect(screen.getByText('React Components')).toBeInTheDocument();
        expect(screen.getByText('API Routes')).toBeInTheDocument();
      });
    });
  });

  describe('Filter functionality', () => {
    it('should filter campaigns by status', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      // Use the status dropdown filter
      const statusSelect = screen.getByRole('combobox', { name: /filter by status/i });
      await user.selectOptions(statusSelect, 'active');

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
        expect(screen.getByText('API Routes')).toBeInTheDocument();
        expect(screen.queryByText('React Components')).not.toBeInTheDocument();
      });
    });

    it('should filter campaigns by agent', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      // Select agent from dropdown (not badge click)
      const agentSelect = screen.getByRole('combobox', { name: /filter by agent/i });
      await user.selectOptions(agentSelect, 'ui-agent');

      await waitFor(() => {
        expect(screen.getByText('React Components')).toBeInTheDocument();
        expect(screen.queryByText('TypeScript Files')).not.toBeInTheDocument();
        expect(screen.queryByText('API Routes')).not.toBeInTheDocument();
      });
    });

    it('should clear status filter when clicked again', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const statusSelect = screen.getByRole('combobox', { name: /filter by status/i });

      // Select to filter
      await user.selectOptions(statusSelect, 'active');
      await waitFor(() => {
        expect(screen.queryByText('React Components')).not.toBeInTheDocument();
      });

      // Clear filter (select empty option)
      await user.selectOptions(statusSelect, '');
      await waitFor(() => {
        expect(screen.getByText('React Components')).toBeInTheDocument();
      });
    });

    it('should combine search and status filters', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      // Apply status filter
      const statusSelect = screen.getByRole('combobox', { name: /filter by status/i });
      await user.selectOptions(statusSelect, 'active');

      // Apply search
      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.type(searchInput, 'API');

      await waitFor(() => {
        expect(screen.getByText('API Routes')).toBeInTheDocument();
        expect(screen.queryByText('TypeScript Files')).not.toBeInTheDocument();
        expect(screen.queryByText('React Components')).not.toBeInTheDocument();
      });
    });
  });

  describe('Keyboard shortcuts', () => {
    it('should focus search input when "/" is pressed', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      expect(searchInput).not.toHaveFocus();

      act(() => {
        const event = new KeyboardEvent('keydown', { key: '/', bubbles: true });
        window.dispatchEvent(event);
      });

      await waitFor(() => {
        expect(searchInput).toHaveFocus();
      });
    });

    it('should call onCreateClick when "n" is pressed', async () => {
      const onCreateClick = vi.fn();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={onCreateClick} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      fireEvent.keyDown(window, { key: 'n' });

      expect(onCreateClick).toHaveBeenCalledTimes(1);
    });

    it('should not trigger shortcuts when typing in input', async () => {
      const user = userEvent.setup();
      const onCreateClick = vi.fn();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: mockCampaigns,
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={onCreateClick} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('Search campaigns...');
      await user.click(searchInput);
      await user.type(searchInput, 'n');

      // Should not trigger create campaign
      expect(onCreateClick).not.toHaveBeenCalled();
      // Search input should contain 'n'
      expect(searchInput).toHaveValue('n');
    });
  });

  describe('Delete functionality', () => {
    it('should open confirmation dialog when delete is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaigns[0]],
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      // Find and click delete button
      const deleteButton = screen.getByRole('button', { name: /delete/i });
      await user.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
      });
    });

    it('should delete campaign when confirmed', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaigns[0]],
      });
      vi.mocked(api.deleteCampaign).mockResolvedValue(undefined);

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const deleteButton = screen.getByRole('button', { name: /delete campaign typescript files/i });
      await user.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
      });

      const confirmButton = screen.getByRole('button', { name: /^delete campaign$/i });
      await user.click(confirmButton);

      await waitFor(() => {
        expect(api.deleteCampaign).toHaveBeenCalled();
        expect(api.deleteCampaign).toHaveBeenCalledWith('1', expect.anything());
      });
    });

    it('should cancel delete when cancel is clicked', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaigns[0]],
      });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const deleteButton = screen.getByRole('button', { name: /delete/i });
      await user.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
      });

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      await waitFor(() => {
        expect(screen.queryByText(/are you sure/i)).not.toBeInTheDocument();
      });

      expect(api.deleteCampaign).not.toHaveBeenCalled();
    });

    it('should show error toast when delete fails', async () => {
      const user = userEvent.setup();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaigns[0]],
      });
      vi.mocked(api.deleteCampaign).mockRejectedValue(new Error('Delete failed'));

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      const deleteButton = screen.getByRole('button', { name: /delete campaign typescript files/i });
      await user.click(deleteButton);

      await waitFor(() => {
        expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
      });

      const confirmButton = screen.getByRole('button', { name: /^delete campaign$/i });
      await user.click(confirmButton);

      await waitFor(() => {
        expect(screen.getByText(/failed to delete campaign/i)).toBeInTheDocument();
      });
    });
  });

  describe('UI controls', () => {
    it('should render new campaign button', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /new campaign/i })).toBeInTheDocument();
      });
    });

    it('should call onCreateClick when new campaign button is clicked', async () => {
      const user = userEvent.setup();
      const onCreateClick = vi.fn();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={onCreateClick} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        const button = screen.getByRole('button', { name: /new campaign/i });
        expect(button).toBeInTheDocument();
      });

      const button = screen.getByRole('button', { name: /new campaign/i });
      await user.click(button);

      expect(onCreateClick).toHaveBeenCalledTimes(1);
    });

    it('should render refresh button', async () => {
      vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

      render(
        <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument();
      });
    });

    it('should call onViewCampaign when campaign card is clicked', async () => {
      const user = userEvent.setup();
      const onViewCampaign = vi.fn();
      vi.mocked(api.fetchCampaigns).mockResolvedValue({
        campaigns: [mockCampaigns[0]],
      });

      render(
        <CampaignList onViewCampaign={onViewCampaign} onCreateClick={vi.fn()} />,
        { wrapper: createWrapper() }
      );

      await waitFor(() => {
        expect(screen.getByText('TypeScript Files')).toBeInTheDocument();
      });

      // Click on the campaign name/card
      const campaignCard = screen.getByText('TypeScript Files');
      await user.click(campaignCard);

      await waitFor(() => {
        expect(onViewCampaign).toHaveBeenCalledWith('1');
      });
    });
  });
});
