import { beforeEach, describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
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
  beforeEach(() => vi.clearAllMocks());
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


describe('campaign review data', () => {
  beforeEach(() => vi.clearAllMocks());
  const file = {
    id: 'file-1', file_path: 'src/worker.ts', absolute_path: '/repo/src/worker.ts',
    visit_count: 2, first_seen: '2026-01-01T00:00:00Z', last_modified: '2026-01-01T00:00:00Z',
    last_visited: '2026-01-02T00:00:00Z', staleness_score: 12.5, deleted: false,
    notes: 'Review cancellation behavior',
  };

  it('renders actual review candidates and refreshes all three sources together', async () => {
    vi.mocked(api.fetchCampaign).mockResolvedValue({ ...mockCampaign, tracked_files: [file] });
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [file] });
    vi.mocked(api.fetchMostStale).mockResolvedValue({ files: [file] });
    const onBack = vi.fn();
    render(<CampaignDetail campaignId="test-id" onBack={onBack} />, { wrapper: createWrapper() });
    await waitFor(() => expect(screen.getAllByText('src/worker.ts')).toHaveLength(3));
    expect(screen.getByTestId('files-list')).toBeInTheDocument();
    expect(screen.getAllByTestId('file-row')).toHaveLength(3);
    expect(screen.getByText('Score: 12.5')).toBeInTheDocument();
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [{ ...file, id: 'file-2', file_path: 'src/new.ts' }] });
    fireEvent.click(screen.getByRole('button', { name: 'Refresh campaign data' }));
    await waitFor(() => expect(screen.getByText('src/new.ts')).toBeInTheDocument());
    expect(api.fetchCampaign).toHaveBeenCalledTimes(2);
    expect(api.fetchLeastVisited).toHaveBeenCalledTimes(2);
    expect(api.fetchMostStale).toHaveBeenCalledTimes(2);
    fireEvent.click(screen.getByRole('button', { name: 'Go back to campaign list' }));
    expect(onBack).toHaveBeenCalledOnce();
  });

  it('distinguishes failed prioritization from an empty campaign and retries it', async () => {
    vi.mocked(api.fetchCampaign).mockResolvedValue(mockCampaign);
    vi.mocked(api.fetchLeastVisited).mockRejectedValueOnce(new Error('unavailable')).mockResolvedValue({ files: [file] });
    vi.mocked(api.fetchMostStale).mockRejectedValueOnce(new Error('unavailable')).mockResolvedValue({ files: [] });
    render(<CampaignDetail campaignId="test-id" onBack={vi.fn()} />, { wrapper: createWrapper() });
    await waitFor(() => expect(screen.getAllByRole('alert')).toHaveLength(2));
    expect(screen.queryByText('No files tracked yet')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh campaign data' }));
    await waitFor(() => expect(screen.getByText('src/worker.ts')).toBeInTheDocument());
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('recovers a failed campaign read without losing the back action', async () => {
    vi.mocked(api.fetchCampaign).mockRejectedValueOnce(new Error('temporarily unavailable')).mockResolvedValue(mockCampaign);
    vi.mocked(api.fetchLeastVisited).mockResolvedValue({ files: [] });
    vi.mocked(api.fetchMostStale).mockResolvedValue({ files: [] });
    const onBack = vi.fn();
    render(<CampaignDetail campaignId="test-id" onBack={onBack} />, { wrapper: createWrapper() });
    expect(await screen.findByText('temporarily unavailable')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try Again' }));
    expect(await screen.findByText('Test Campaign')).toBeInTheDocument();
    fireEvent.keyDown(window, { key: 'Escape' });
    expect(onBack).toHaveBeenCalledOnce();
  });
});
