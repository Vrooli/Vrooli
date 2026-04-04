import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { ProfileDetail } from './ProfileDetail';
import * as api from '../lib/api';

vi.mock('../lib/api');

// Mock useParams to return a profile ID
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ id: 'prof-1' }),
  };
});

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    </MemoryRouter>
  );
};

const mockProfile: api.DeploymentProfile = {
  id: 'prof-1',
  name: 'Production',
  scenario: 'test-scenario',
  version: 3,
  tiers: [2, 4],
  created_at: '2026-01-01T00:00:00Z',
};

describe('ProfileDetail - Release Gate', () => {
  it('renders release gate card', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: [] });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Release Gate');
    expect(screen.getByPlaceholderText('Enter git commit hash...')).toBeInTheDocument();
  });

  it('shows Ready badge when gate is ready', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: ['linux'] });
    vi.mocked(api.checkReleaseGate).mockResolvedValue({
      profile_id: 'prof-1',
      git_commit_hash: 'abc123def456789',
      ready: true,
      platforms: [
        { platform: 'linux', required: true, status: 'approved' },
      ],
    });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Release Gate');

    const input = screen.getByPlaceholderText('Enter git commit hash...');
    fireEvent.change(input, { target: { value: 'abc123def456789' } });

    await screen.findByText('Ready');
  });

  it('shows Blocked badge with platform breakdown when not ready', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: ['linux', 'windows'] });
    vi.mocked(api.checkReleaseGate).mockResolvedValue({
      profile_id: 'prof-1',
      git_commit_hash: 'abc123def456789',
      ready: false,
      platforms: [
        { platform: 'linux', required: true, status: 'approved' },
        { platform: 'windows', required: true, status: 'pending' },
      ],
    });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Release Gate');

    const input = screen.getByPlaceholderText('Enter git commit hash...');
    fireEvent.change(input, { target: { value: 'abc123def456789' } });

    await screen.findByText('Blocked');
    // "linux" and "windows" appear in both checkboxes and gate status
    expect(screen.getAllByText('linux').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('windows').length).toBeGreaterThanOrEqual(1);
    // Check gate-specific badges
    expect(screen.getByText('pending')).toBeInTheDocument();
  });
});

describe('ProfileDetail - Required Platforms', () => {
  it('renders required platforms checkboxes', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: ['linux'] });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Required Platforms');
    expect(screen.getByText('windows')).toBeInTheDocument();
    expect(screen.getByText('macos')).toBeInTheDocument();
    expect(screen.getByText('linux')).toBeInTheDocument();
  });

  it('shows save button after changing selection', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: ['linux'] });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Required Platforms');

    // Check the windows checkbox
    const windowsCheckbox = screen.getAllByRole('checkbox')[0];
    expect(windowsCheckbox).toBeDefined();
    // windows is first in ALL_PLATFORMS
    if (windowsCheckbox) fireEvent.click(windowsCheckbox);

    expect(screen.getByText('Save Platforms')).toBeInTheDocument();
  });

  it('calls setRequiredPlatforms on save', async () => {
    vi.mocked(api.getProfile).mockResolvedValue(mockProfile);
    vi.mocked(api.getRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: [] });
    vi.mocked(api.setRequiredPlatforms).mockResolvedValue({ profile_id: 'prof-1', platforms: ['windows'] });

    render(<ProfileDetail />, { wrapper: createWrapper() });

    await screen.findByText('Required Platforms');

    // Check windows checkbox
    const firstCheckbox = screen.getAllByRole('checkbox')[0];
    expect(firstCheckbox).toBeDefined();
    if (firstCheckbox) fireEvent.click(firstCheckbox);

    // Click save
    fireEvent.click(screen.getByText('Save Platforms'));

    await waitFor(() => {
      expect(api.setRequiredPlatforms).toHaveBeenCalledWith('prof-1', ['windows']);
    });
  });
});
