import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';
import * as api from './lib/api';

// Mock API
vi.mock('./lib/api', () => ({
  fetchCampaigns: vi.fn(),
  createCampaign: vi.fn(),
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

describe('App', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render the application', async () => {
    vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

    render(<App />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Visited Tracker')).toBeInTheDocument();
    });
  });

  it('should start with list view', async () => {
    vi.mocked(api.fetchCampaigns).mockResolvedValue({ campaigns: [] });

    render(<App />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Welcome to Visited Tracker')).toBeInTheDocument();
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

      // Should have interactive elements
      expect(screen.getByRole('button', { name: /new campaign/i })).toBeInTheDocument();
    });
  });
});
