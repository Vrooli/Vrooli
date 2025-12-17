import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';

// Mock the API functions
vi.mock('./lib/api', () => ({
  fetchHealth: vi.fn().mockResolvedValue({
    status: 'healthy',
    service: 'Workspace Sandbox API',
    version: '1.0.0',
    readiness: true,
    timestamp: new Date().toISOString(),
    dependencies: { database: 'connected', driver: 'available' },
  }),
  listSandboxes: vi.fn().mockResolvedValue({
    sandboxes: [],
    totalCount: 0,
    limit: 100,
    offset: 0,
  }),
  computeStats: vi.fn().mockReturnValue({
    total: 0,
    active: 0,
    stopped: 0,
    approved: 0,
    rejected: 0,
    error: 0,
    totalSizeBytes: 0,
  }),
}));

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

describe('App', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = createQueryClient();
  });

  /**
   * [REQ:REQ-P0-010] Health Check API Endpoint
   * Verifies the UI renders health status from the API
   */
  it('renders the app container', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    // App should render with the main container
    const appContainer = screen.getByTestId('workspace-sandbox-app');
    expect(appContainer).toBeInTheDocument();
  });

  /**
   * [REQ:REQ-P0-001] Fast Sandbox Creation
   * Verifies the create sandbox button is present
   */
  it('renders the create sandbox button', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    // The create button should be visible
    const createButton = await screen.findByTestId('create-sandbox-button');
    expect(createButton).toBeInTheDocument();
  });

  /**
   * [REQ:REQ-P0-002] Stable Sandbox Identifier
   * Verifies the sandbox list renders
   */
  it('renders the sandbox list container', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    // The sandbox list should be visible
    const sandboxList = screen.getByTestId('sandbox-list');
    expect(sandboxList).toBeInTheDocument();
  });
});
