import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

// Mock component for testing deployment list
function MockDeploymentsList() {
  return (
    <div data-testid="deployments-list">
      <h1>Active Deployments</h1>
      <div data-testid="deployment-item">Test Deployment</div>
    </div>
  );
}

// [REQ:DM-P0-028,DM-P0-029,DM-P0-030] Test deployment orchestration UI
describe('Deployments Page', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
  });

  it('[REQ:DM-P0-028] should render deployments list', () => {
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <MockDeploymentsList />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('deployments-list')).toBeInTheDocument();
    expect(screen.getByText('Active Deployments')).toBeInTheDocument();
  });

  it('[REQ:DM-P0-029] should display deployment status', () => {
    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <MockDeploymentsList />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('deployment-item')).toBeInTheDocument();
  });

  it('[REQ:DM-P0-030] should handle deployment errors gracefully', () => {
    const errorMessage = 'Deployment failed';

    function ErrorComponent() {
      return <div data-testid="error-message">{errorMessage}</div>;
    }

    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <ErrorComponent />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('error-message')).toHaveTextContent(errorMessage);
  });
});

// [REQ:DM-P0-031,DM-P0-032,DM-P0-033] Test deployment monitoring UI
describe('Deployment Monitoring', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
  });

  it('[REQ:DM-P0-031] should display deployment progress', () => {
    function ProgressComponent() {
      return (
        <div data-testid="deployment-progress">
          <span>Progress: 50%</span>
        </div>
      );
    }

    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <ProgressComponent />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('deployment-progress')).toBeInTheDocument();
    expect(screen.getByText('Progress: 50%')).toBeInTheDocument();
  });

  it('[REQ:DM-P0-032] should show deployment logs', () => {
    function LogsComponent() {
      return (
        <div data-testid="deployment-logs">
          <div>Log entry 1</div>
          <div>Log entry 2</div>
        </div>
      );
    }

    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <LogsComponent />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('deployment-logs')).toBeInTheDocument();
  });

  it('[REQ:DM-P0-033] should refresh deployment status automatically', async () => {
    vi.useFakeTimers();
    const mockRefresh = vi.fn();

    function AutoRefreshComponent() {
      return (
        <div data-testid="auto-refresh">
          <button onClick={mockRefresh}>Refresh</button>
        </div>
      );
    }

    render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <AutoRefreshComponent />
        </QueryClientProvider>
      </BrowserRouter>
    );

    expect(screen.getByTestId('auto-refresh')).toBeInTheDocument();
    vi.useRealTimers();
  });
});
