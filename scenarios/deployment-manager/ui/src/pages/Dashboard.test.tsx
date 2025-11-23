import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { Dashboard } from './Dashboard';
import * as api from '../lib/api';

// Mock the API
vi.mock('../lib/api');

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    </BrowserRouter>
  );
};

describe('Dashboard', () => {
  it('renders dashboard title and description', () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Dashboard />, { wrapper: createWrapper() });

    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText(/Manage deployment profiles and monitor your deployments/i)).toBeInTheDocument();
  });

  it('displays stat cards with correct titles', () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Dashboard />, { wrapper: createWrapper() });

    expect(screen.getByText('Total Profiles')).toBeInTheDocument();
    expect(screen.getByText('Active Deployments')).toBeInTheDocument();
    expect(screen.getByText('Failed Deployments')).toBeInTheDocument();
  });

  it('shows New Profile button', () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Dashboard />, { wrapper: createWrapper() });

    const newProfileButton = screen.getAllByText('New Profile')[0];
    expect(newProfileButton).toBeInTheDocument();
  });

  it('displays empty state when no profiles exist', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Dashboard />, { wrapper: createWrapper() });

    const emptyMessage = await screen.findByText('No deployment profiles yet');
    expect(emptyMessage).toBeInTheDocument();
  });

  it('displays profile count in stats when profiles exist', async () => {
    const mockProfiles = [
      { id: '1', name: 'Test Profile', scenario: 'test-scenario', version: 1, tiers: [1, 2] },
      { id: '2', name: 'Another Profile', scenario: 'another-scenario', version: 2, tiers: [3] },
    ];
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    render(<Dashboard />, { wrapper: createWrapper() });

    const profileCount = await screen.findByText('2');
    expect(profileCount).toBeInTheDocument();
  });

  it('renders profile cards when profiles exist', async () => {
    const mockProfiles = [
      { id: '1', name: 'Test Profile', scenario: 'test-scenario', version: 1, tiers: [1] },
    ];
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    render(<Dashboard />, { wrapper: createWrapper() });

    const profileName = await screen.findByText('Test Profile');
    expect(profileName).toBeInTheDocument();
    expect(screen.getByText('Scenario: test-scenario')).toBeInTheDocument();
  });
});
