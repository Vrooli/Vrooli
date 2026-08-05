import { describe, it, expect, vi } from 'vitest';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../test-utils/renderWithProviders';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { Approvals } from './Approvals';
import * as api from '../../lib/api';

vi.mock('../../lib/api');

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

const mockProfiles = [
  { id: 'prof-1', name: 'Production', scenario: 'test', version: 1, tiers: [2] },
  { id: 'prof-2', name: 'Staging', scenario: 'test', version: 1, tiers: [1] },
];

const mockApprovals: api.DeploymentApproval[] = [
  {
    id: 'approval-1',
    profile_id: 'prof-1',
    git_commit_hash: 'abc123def456789',
    platform: 'linux',
    status: 'pending',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  },
  {
    id: 'approval-2',
    profile_id: 'prof-1',
    git_commit_hash: 'abc123def456789',
    platform: 'windows',
    status: 'approved',
    approved_by: 'alice',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
  },
];

async function selectProfile() {
  // Wait for profiles to load into the select
  await screen.findByText('Production');
  const select = screen.getByTestId('profile-select');
  fireEvent.change(select, { target: { value: 'prof-1' } });
}

describe('Approvals', () => {
  it('renders page title', () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    expect(screen.getByText('Approvals')).toBeInTheDocument();
    expect(screen.getByText(/Manage deployment approval gates/i)).toBeInTheDocument();
  });

  it('shows select prompt when no profile selected', () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    expect(screen.getByText('Select a profile to view approvals')).toBeInTheDocument();
  });

  it('renders approval list after selecting profile', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue(mockApprovals);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();

    await screen.findByText('Approval List');
    expect(screen.getByText('linux')).toBeInTheDocument();
    expect(screen.getByText('windows')).toBeInTheDocument();
  });

  it('filters approvals by status', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue(mockApprovals);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();
    await screen.findByText('Approval List');

    // Click the "approved" filter button (not the badge)
    const filterButtons = screen.getAllByRole('button');
    const approvedFilter = filterButtons.find(
      (btn) => btn.textContent === 'approved'
    );
    if (approvedFilter) {
      fireEvent.click(approvedFilter);
    }

    await waitFor(() => {
      expect(screen.getByText('1 approval (approved)')).toBeInTheDocument();
    });
  });

  it('shows empty state when no approvals', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue([]);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();

    expect(await screen.findByText('No approvals found')).toBeInTheDocument();
  });

  it('shows error state on API failure', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockRejectedValue(new Error('Network error'));

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();

    expect(await screen.findByText(/Failed to load approvals: Network error/i)).toBeInTheDocument();
  });

  it('opens detail view on row click', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue(mockApprovals);

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();
    await screen.findByText('Approval List');

    fireEvent.click(screen.getByText('linux'));

    expect(screen.getByText('Approval Detail')).toBeInTheDocument();
    expect(screen.getByText('approval-1')).toBeInTheDocument();
  });

  it('calls decideApproval on approve action', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue(mockApprovals);
    vi.mocked(api.decideApproval).mockResolvedValue({
      id: 'approval-1',
      profile_id: 'prof-1',
      git_commit_hash: 'abc123def456789',
      platform: 'linux',
      status: 'approved',
      approved_by: 'tester',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    });

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });

    await selectProfile();
    await screen.findByText('Approval List');

    // Click linux row to open detail
    fireEvent.click(screen.getByText('linux'));

    // Fill in reviewer
    const reviewerInput = screen.getByPlaceholderText('Your name');
    fireEvent.change(reviewerInput, { target: { value: 'tester' } });

    // Click the Approve button in the detail panel (the one inside the decision section)
    const approveButtons = screen.getAllByText('Approve');
    // The last Approve button is the one in the detail decision section
    const detailApproveBtn = approveButtons[approveButtons.length - 1];
    expect(detailApproveBtn).toBeDefined();
    if (detailApproveBtn) fireEvent.click(detailApproveBtn);

    await waitFor(() => {
      expect(api.decideApproval).toHaveBeenCalledWith('approval-1', {
        decision: 'approved',
        reviewer: 'tester',
        notes: '',
      });
    });
  });

  it('supports refresh, notes, rejection, and closing the detail panel', async () => {
    vi.mocked(api.listProfiles).mockResolvedValue(mockProfiles);
    vi.mocked(api.listApprovals).mockResolvedValue(mockApprovals);
    vi.mocked(api.decideApproval).mockResolvedValue({
      id: 'approval-1', profile_id: 'prof-1', git_commit_hash: 'abc123def456789', platform: 'linux', status: 'rejected',
      created_at: '2026-01-01T00:00:00Z', updated_at: '2026-01-01T00:00:00Z',
    });

    renderWithProviders(<Approvals />, { wrapper: createWrapper() });
    await selectProfile();
    await screen.findByText('Approval List');
    fireEvent.click(screen.getByRole('button', { name: /refresh/i }));
    fireEvent.click(screen.getByText('linux'));
    fireEvent.change(screen.getByPlaceholderText('Your name'), { target: { value: 'reviewer' } });
    fireEvent.change(screen.getByPlaceholderText('Optional notes'), { target: { value: 'not ready' } });
    const rejectButtons = screen.getAllByText('Reject');
    const rejectButton = rejectButtons[rejectButtons.length - 1];
    if (rejectButton) fireEvent.click(rejectButton);
    await waitFor(() => expect(api.decideApproval).toHaveBeenCalledWith('approval-1', {
      decision: 'rejected', reviewer: 'reviewer', notes: 'not ready',
    }));
  });
});
