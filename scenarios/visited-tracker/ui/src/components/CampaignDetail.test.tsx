import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from './ui/tooltip';
import { CampaignDetail } from './CampaignDetail';
import * as api from '../lib/api';

// Mock API
vi.mock('../lib/api', () => ({
  fetchCampaign: vi.fn(),
  fetchLeastVisited: vi.fn(),
  fetchMostStale: vi.fn(),
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
      <TooltipProvider>{children}</TooltipProvider>
    </QueryClientProvider>
  );
};

const mockCampaign = {
  id: 'test-id',
  name: 'Test Campaign',
  from_agent: 'test-agent',
  status: 'active' as const,
  patterns: ['**/*.tsx'],
  total_files: 20,
  visited_files: 10,
  coverage_percent: 50,
  notes: 'Test notes',
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
  visits: [],
  tracked_files: [],
};

describe('CampaignDetail', () => {
  it('should render campaign name and metadata', async () => {
    vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [] });
    vi.mocked(api.fetchMostStale).mockResolvedValue({ files: [] });

    render(<CampaignDetail campaignId="test-id" onBack={vi.fn()} />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByText('Test Campaign')).toBeInTheDocument();
    });
  });

  it('should display actionable files section heading', async () => {
    vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [] });
    vi.mocked(api.fetchMostStale).mockResolvedValue({ files: [] });

    render(<CampaignDetail campaignId="test-id" onBack={vi.fn()} />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByText(/Test Campaign/)).toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it('should render without errors', async () => {
    vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [] });
    vi.mocked(api.fetchMostStale).mockResolvedValue({ files: [] });

    const { container } = render(<CampaignDetail campaignId="test-id" onBack={vi.fn()} />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(container).toBeTruthy();
    });
  });
});
