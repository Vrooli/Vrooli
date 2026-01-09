import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { Profiles } from './Profiles';
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

describe('Profiles', () => {
  it('renders page title and description', () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Profiles />, { wrapper: createWrapper() });

    expect(screen.getByText('Deployment Profiles')).toBeInTheDocument();
    expect(screen.getByText(/Manage your deployment configurations/i)).toBeInTheDocument();
  });

  it('shows New Profile button', () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Profiles />, { wrapper: createWrapper() });

    const newProfileButton = screen.getByText('New Profile');
    expect(newProfileButton).toBeInTheDocument();
  });

  it('displays empty state when no profiles exist', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue([]);

    render(<Profiles />, { wrapper: createWrapper() });

    const emptyMessage = await screen.findByText('No deployment profiles yet');
    expect(emptyMessage).toBeInTheDocument();
    expect(screen.getByText('Create Your First Profile')).toBeInTheDocument();
  });

  it('displays error message when API fails', async () => {
    vi.mocked(api.listProfiles).mockRejectedValue(new Error('Network error'));

    render(<Profiles />, { wrapper: createWrapper() });

    const errorMessage = await screen.findByText(/Failed to load profiles: Network error/i);
    expect(errorMessage).toBeInTheDocument();
  });

  it('renders profile cards when profiles exist', async () => {
    const mockProfiles = [
      {
        id: '1',
        name: 'Production Profile',
        scenario: 'my-scenario',
        version: 3,
        tiers: [1, 4],
        created_at: '2025-11-22T00:00:00Z',
      },
      {
        id: '2',
        name: 'Desktop Profile',
        scenario: 'desktop-app',
        version: 1,
        tiers: [2],
        created_at: '2025-11-21T00:00:00Z',
      },
    ];
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    render(<Profiles />, { wrapper: createWrapper() });

    const profile1 = await screen.findByText('Production Profile');
    expect(profile1).toBeInTheDocument();
    expect(screen.getByText('my-scenario')).toBeInTheDocument();
    expect(screen.getByText('Desktop Profile')).toBeInTheDocument();
    expect(screen.getByText('desktop-app')).toBeInTheDocument();
  });

  it('displays tier badges correctly', async () => {
    const mockProfiles = [
      {
        id: '1',
        name: 'Multi-Tier Profile',
        scenario: 'test',
        version: 1,
        tiers: [2, 3, 4],
        created_at: '2025-11-22T00:00:00Z',
      },
    ];
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    render(<Profiles />, { wrapper: createWrapper() });

    await screen.findByText('Multi-Tier Profile');
    expect(screen.getByText('Desktop')).toBeInTheDocument();
    expect(screen.getByText('Mobile')).toBeInTheDocument();
    expect(screen.getByText('SaaS/Cloud')).toBeInTheDocument();
  });

  it('displays version badges', async () => {
    const mockProfiles = [
      {
        id: '1',
        name: 'Test Profile',
        scenario: 'test',
        version: 5,
        tiers: [1],
        created_at: '2025-11-22T00:00:00Z',
      },
    ];
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    render(<Profiles />, { wrapper: createWrapper() });

    const versionBadge = await screen.findByText('v5');
    expect(versionBadge).toBeInTheDocument();
  });
});
