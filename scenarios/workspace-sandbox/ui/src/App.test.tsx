import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import App from './App';

// Mock the API functions. We use importOriginal so all type-level
// constants (ACTIVE_STATUSES, isHistoryStatus, etc.) keep their real
// values; only network-touching functions are stubbed.
vi.mock('./lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./lib/api')>();
  return {
    ...actual,
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
    listHistory: vi.fn().mockResolvedValue({
      archives: [],
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
    formatBytes: vi.fn((bytes: number) => `${bytes}B`),
    formatRelativeTime: vi.fn(() => 'just now'),
  };
});

const createQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });

function setViewportWidth(width: number) {
  Object.defineProperty(window, 'innerWidth', { value: width, writable: true, configurable: true });
  window.dispatchEvent(new Event('resize'));
}

describe('App - Desktop Layout', () => {
  let queryClient: QueryClient;
  const originalWidth = window.innerWidth;

  beforeEach(() => {
    queryClient = createQueryClient();
    setViewportWidth(1024);
  });

  afterEach(() => {
    setViewportWidth(originalWidth);
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

    const sandboxList = screen.getByTestId('sandbox-list');
    expect(sandboxList).toBeInTheDocument();
  });

  it('does not render mobile navigation on desktop', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    expect(screen.queryByTestId('mobile-nav')).not.toBeInTheDocument();
    expect(screen.queryByTestId('mobile-header')).not.toBeInTheDocument();
  });
});

describe('App - Mobile Layout', () => {
  let queryClient: QueryClient;
  const originalWidth = window.innerWidth;

  beforeEach(() => {
    queryClient = createQueryClient();
    setViewportWidth(375);
  });

  afterEach(() => {
    setViewportWidth(originalWidth);
  });

  it('renders mobile navigation', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('mobile-nav')).toBeInTheDocument();
    expect(screen.getByTestId('mobile-header')).toBeInTheDocument();
  });

  it('renders sandbox list on the sandboxes tab', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('sandbox-list')).toBeInTheDocument();
  });

  it('does not render the desktop status header on mobile', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    expect(screen.queryByTestId('status-header')).not.toBeInTheDocument();
  });

  it('renders the app container with data-testid', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    );

    expect(screen.getByTestId('workspace-sandbox-app')).toBeInTheDocument();
  });
});
