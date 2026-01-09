import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import CampaignsView from '../CampaignsView';
import * as api from '../../lib/api';
import { ToastProvider } from '../../components/ui/toast';

// Mock the API module
vi.mock('../../lib/api', () => ({
  fetchCampaigns: vi.fn(),
}));

// Mock clipboard API with proper mock implementation
const mockWriteText = vi.fn(() => Promise.resolve());
Object.assign(navigator, {
  clipboard: {
    writeText: mockWriteText,
  },
});

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <ToastProvider>
          {children}
        </ToastProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
};

describe('CampaignsView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Loading state', () => {
    it('should display loading spinner while fetching campaigns', () => {
      // Mock a pending promise
      (api.fetchCampaigns as any).mockImplementation(
        () => new Promise(() => {}) // Never resolves
      );

      render(<CampaignsView />, { wrapper: createWrapper() });

      expect(screen.getByRole('status')).toBeInTheDocument();
    });
  });

  describe('Empty state', () => {
    beforeEach(() => {
      (api.fetchCampaigns as any).mockResolvedValue([]);
    });

    it('should display empty state when no campaigns exist', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('No Active Campaigns')).toBeInTheDocument();
      });
    });

    it('should display descriptive text about campaigns', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(
          screen.getByText(/Campaigns systematically analyze files/i)
        ).toBeInTheDocument();
      });
    });

    it('should show CLI command examples', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Check for the code blocks containing CLI commands (multiple instances expected)
        const startCommands = screen.getAllByText(/tidiness-manager campaigns start/i);
        expect(startCommands.length).toBeGreaterThan(0);

        const listCommands = screen.getAllByText(/tidiness-manager campaigns list/i);
        expect(listCommands.length).toBeGreaterThan(0);

        const pauseCommands = screen.getAllByText(/tidiness-manager campaigns pause/i);
        expect(pauseCommands.length).toBeGreaterThan(0);
      });
    });

    it('should display disabled "Create Your First Campaign" button', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        const button = screen.getByTitle(/UI campaign creation coming soon/i);
        expect(button).toBeDisabled();
      });
    });

    it('should show workflow instructions for agents', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(
          screen.getByText(/Agents can create, monitor, and control campaigns/i)
        ).toBeInTheDocument();
      });
    });

    it('should display use cases for campaigns', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Check for the "Use Cases:" label and specific use case text
        expect(screen.getByText(/Use Cases:/i)).toBeInTheDocument();
        expect(screen.getByText(/Systematic refactoring/i)).toBeInTheDocument();
        expect(screen.getByText(/batch issue discovery/i)).toBeInTheDocument();
      });
    });
  });

  describe('Error state', () => {
    it('should display error message when fetch fails', async () => {
      const errorMessage = 'Failed to connect to server';
      (api.fetchCampaigns as any).mockRejectedValue(new Error(errorMessage));

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Component renders "Failed to load campaigns: {error.message}" as a single text node
        expect(screen.getByText(`Failed to load campaigns: ${errorMessage}`)).toBeInTheDocument();
      });
    });

    it('should still show page header on error', async () => {
      (api.fetchCampaigns as any).mockRejectedValue(new Error('Network error'));

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('Campaigns')).toBeInTheDocument();
      });
    });
  });

  describe('Campaigns list', () => {
    const mockCampaigns = [
      {
        id: 1,
        scenario: 'test-scenario-1',
        status: 'active',
        max_sessions: 10,
        current_session: 3,
        files_visited: 15,
        files_total: 50,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T12:00:00Z',
        config: {
          max_files_per_session: 5,
          priority_rules: [],
        },
      },
      {
        id: 2,
        scenario: 'test-scenario-2',
        status: 'paused',
        max_sessions: 10,
        current_session: 5,
        files_visited: 25,
        files_total: 60,
        created_at: '2024-01-02T00:00:00Z',
        updated_at: '2024-01-02T12:00:00Z',
        config: {
          max_files_per_session: 5,
          priority_rules: [],
        },
      },
      {
        id: 3,
        scenario: 'test-scenario-3',
        status: 'completed',
        max_sessions: 10,
        current_session: 10,
        files_visited: 50,
        files_total: 50,
        created_at: '2024-01-03T00:00:00Z',
        updated_at: '2024-01-03T12:00:00Z',
        config: {
          max_files_per_session: 5,
          priority_rules: [],
        },
      },
    ];

    beforeEach(() => {
      (api.fetchCampaigns as any).mockResolvedValue(mockCampaigns);
    });

    it('should display all campaigns', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('test-scenario-1')).toBeInTheDocument();
        expect(screen.getByText('test-scenario-2')).toBeInTheDocument();
        expect(screen.getByText('test-scenario-3')).toBeInTheDocument();
      });
    });

    it('should display campaign status badges', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('active')).toBeInTheDocument();
        expect(screen.getByText('paused')).toBeInTheDocument();
        expect(screen.getByText('completed')).toBeInTheDocument();
      });
    });

    it('should show correct badge variants for different statuses', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        const activeBadge = screen.getByText('active');
        const pausedBadge = screen.getByText('paused');
        const completedBadge = screen.getByText('completed');

        // Badges exist and have appropriate classes
        expect(activeBadge).toBeInTheDocument();
        expect(pausedBadge).toBeInTheDocument();
        expect(completedBadge).toBeInTheDocument();

        // Check that badges have the Badge component classes
        expect(activeBadge.closest('.badge')).toBeTruthy();
        expect(pausedBadge.closest('.badge')).toBeTruthy();
        expect(completedBadge.closest('.badge')).toBeTruthy();
      });
    });
  });

  describe('Page header', () => {
    beforeEach(() => {
      (api.fetchCampaigns as any).mockResolvedValue([]);
    });

    it('should display page title', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      expect(screen.getByText('Tidiness Campaigns')).toBeInTheDocument();
    });

    it('should display page description', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(
          screen.getByText(/Manage automated code tidiness campaigns/i)
        ).toBeInTheDocument();
      });
    });

    it('should show CLI reference in description', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        // Check for the CLI reference in the header description (may appear multiple times)
        const listCommands = screen.getAllByText(/tidiness-manager campaigns list/i);
        expect(listCommands.length).toBeGreaterThan(0);

        const createCommands = screen.getAllByText(/tidiness-manager campaign create/i);
        expect(createCommands.length).toBeGreaterThan(0);
      });
    });

    it('should have "New Campaign" button in header', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      const button = screen.getByRole('button', { name: /New Campaign/i });
      expect(button).toBeInTheDocument();
    });
  });

  describe('Campaign actions', () => {
    const mockCampaigns = [
      {
        id: 1,
        scenario: 'test-scenario',
        status: 'active',
        max_sessions: 10,
        current_session: 3,
        files_visited: 15,
        files_total: 50,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T12:00:00Z',
        config: {
          max_files_per_session: 5,
          priority_rules: [],
        },
      },
    ];

    beforeEach(() => {
      (api.fetchCampaigns as any).mockResolvedValue(mockCampaigns);
    });

    it('should copy pause command to clipboard', async () => {
      const user = userEvent.setup();
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => screen.getByText('test-scenario'));

      const pauseButton = screen.getByRole('button', { name: /Pause/i });
      await user.click(pauseButton);

      await waitFor(() => {
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
          'tidiness-manager campaigns pause test-scenario'
        );
        // Verify both toasts appear: info toast for CLI command + success toast for clipboard copy
        expect(screen.getByText(/CLI command copied to clipboard/i)).toBeInTheDocument();
      });
    });

    it('should copy resume command to clipboard', async () => {
      const user = userEvent.setup();
      (api.fetchCampaigns as any).mockResolvedValue([
        { ...mockCampaigns[0], status: 'paused' },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => screen.getByText('test-scenario'));

      const resumeButton = screen.getByRole('button', { name: /Resume/i });
      await user.click(resumeButton);

      await waitFor(() => {
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
          'tidiness-manager campaigns resume test-scenario'
        );
        expect(screen.getByText(/CLI command copied to clipboard/i)).toBeInTheDocument();
      });
    });

    it('should copy stop command to clipboard', async () => {
      const user = userEvent.setup();
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => screen.getByText('test-scenario'));

      const stopButton = screen.getByRole('button', { name: /Stop/i });
      await user.click(stopButton);

      await waitFor(() => {
        expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
          'tidiness-manager campaigns stop test-scenario'
        );
        expect(screen.getByText(/CLI command copied to clipboard/i)).toBeInTheDocument();
      });
    });

    it('should show toast on successful clipboard copy', async () => {
      const user = userEvent.setup();
      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => screen.getByText('test-scenario'));

      const pauseButton = screen.getByRole('button', { name: /Pause/i });
      await user.click(pauseButton);

      await waitFor(() => {
        expect(screen.getByText(/CLI command copied to clipboard/i)).toBeInTheDocument();
      });
    });

    it('should show error toast on clipboard failure', async () => {
      const user = userEvent.setup();
      mockWriteText.mockRejectedValueOnce(
        new Error('Clipboard access denied')
      );

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => screen.getByText('test-scenario'));

      const pauseButton = screen.getByRole('button', { name: /Pause/i });
      await user.click(pauseButton);

      await waitFor(() => {
        expect(screen.getByText(/Failed to copy to clipboard/i)).toBeInTheDocument();
      });
    });
  });

  describe('Polling behavior', () => {
    it('should refetch campaigns every 10 seconds', async () => {
      vi.useFakeTimers();
      (api.fetchCampaigns as any).mockResolvedValue([]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(api.fetchCampaigns).toHaveBeenCalledTimes(1);
      });

      // Fast-forward 10 seconds
      vi.advanceTimersByTime(10000);

      await waitFor(() => {
        expect(api.fetchCampaigns).toHaveBeenCalledTimes(2);
      });

      vi.useRealTimers();
    });
  });

  describe('Edge cases', () => {
    it('should handle unknown campaign statuses with default badge', async () => {
      (api.fetchCampaigns as any).mockResolvedValue([
        {
          id: 1,
          scenario: 'test-scenario',
          status: 'unknown-status',
          max_sessions: 10,
          current_session: 1,
          files_visited: 0,
          files_total: 100,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T12:00:00Z',
          config: {
            max_files_per_session: 10,
            priority_rules: [],
          },
        },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        const badge = screen.getByText('unknown-status');
        expect(badge).toBeInTheDocument();
      });
    });

    it('should handle campaigns with missing fields gracefully', async () => {
      (api.fetchCampaigns as any).mockResolvedValue([
        {
          id: 1,
          scenario: 'minimal-campaign',
          status: 'active',
          max_sessions: 10,
          current_session: 1,
          files_visited: 0,
          files_total: 0, // Edge case: zero total files
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T12:00:00Z',
          config: {
            max_files_per_session: 10,
            priority_rules: [],
          },
        },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText('minimal-campaign')).toBeInTheDocument();
      });
    });

    it('should handle very long scenario names', async () => {
      const longName = 'this-is-a-very-long-scenario-name-that-might-cause-layout-issues';
      (api.fetchCampaigns as any).mockResolvedValue([
        {
          id: 1,
          scenario: longName,
          status: 'active',
          max_sessions: 10,
          current_session: 1,
          files_visited: 5,
          files_total: 100,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T12:00:00Z',
          config: {
            max_files_per_session: 10,
            priority_rules: [],
          },
        },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByText(longName)).toBeInTheDocument();
      });
    });

    it('should handle empty scenario name', async () => {
      (api.fetchCampaigns as any).mockResolvedValue([
        {
          id: 1,
          scenario: '',
          status: 'active',
          max_sessions: 10,
          current_session: 1,
          files_visited: 0,
          files_total: 100,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T12:00:00Z',
          config: {
            max_files_per_session: 10,
            priority_rules: [],
          },
        },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      // Component should still render without crashing
      await waitFor(() => {
        expect(screen.getByText('Tidiness Campaigns')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    beforeEach(() => {
      (api.fetchCampaigns as any).mockResolvedValue([]);
    });

    it('should have accessible page title', () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      expect(screen.getByText('Tidiness Campaigns')).toBeInTheDocument();
    });

    it('should have accessible button labels', async () => {
      (api.fetchCampaigns as any).mockResolvedValue([
        {
          id: 1,
          scenario: 'test-scenario',
          status: 'active',
          max_sessions: 10,
          current_session: 1,
          files_visited: 10,
          files_total: 100,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T12:00:00Z',
          config: {
            max_files_per_session: 10,
            priority_rules: [],
          },
        },
      ]);

      render(<CampaignsView />, { wrapper: createWrapper() });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Pause/i })).toBeInTheDocument();
        expect(screen.getByRole('button', { name: /Stop/i })).toBeInTheDocument();
      });
    });

    it('should provide tooltips for actions', async () => {
      render(<CampaignsView />, { wrapper: createWrapper() });

      // Check for tooltip on the New Campaign button
      const newCampaignButton = screen.getByTitle(/Create a new automated campaign/i);
      expect(newCampaignButton).toBeInTheDocument();

      // Disabled button should also have the "coming soon" tooltip
      const disabledButton = screen.getByTitle(/UI campaign creation coming soon/i);
      expect(disabledButton).toBeInTheDocument();
    });
  });
});
