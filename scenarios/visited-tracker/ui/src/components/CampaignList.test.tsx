import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CampaignList } from './CampaignList';
import * as api from '../lib/api';

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
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('CampaignList', () => {
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

  it('should render stats cards', async () => {
    vi.mocked(api.fetchCampaigns).mockResolvedValue({
      campaigns: [
        {
          id: '1',
          name: 'Test Campaign',
          from_agent: 'test-agent',
          status: 'active',
          location: '/test',
          pattern: '**/*.ts',
          tags: ['test'],
          total_files: 10,
          visited_files: 5,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    });

    render(
      <CampaignList onViewCampaign={vi.fn()} onCreateClick={vi.fn()} />,
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(screen.getByText('Total Campaigns')).toBeInTheDocument();
      expect(screen.getByText('Active Campaigns')).toBeInTheDocument();
      expect(screen.getByText('Tracked Files')).toBeInTheDocument();
    });
  });

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
});
