import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { WaitlistManagement } from './WaitlistManagement';
import * as waitlistHook from '../hooks/useWaitlistForm';

vi.mock('../hooks/useWaitlistForm');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));

function formState(overrides: Record<string, unknown> = {}) {
  return { emails: [], comingSoonEnabled: false, stats: { totalSignups: 0, comingSoonCount: 0 }, loading: false, error: null, deleting: null, togglingComingSoon: false, loadData: vi.fn(), handleDelete: vi.fn(), handleToggleComingSoon: vi.fn(), handleExport: vi.fn(), ...overrides } as unknown as ReturnType<typeof waitlistHook.useWaitlistForm>;
}

describe('WaitlistManagement', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('shows loading, empty collection guidance, and an actionable coming-soon toggle', () => {
    const loading = formState({ loading: true, comingSoonEnabled: true });
    vi.mocked(waitlistHook.useWaitlistForm).mockReturnValue(loading);
    const { rerender } = render(<WaitlistManagement />);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
    const state = formState({ comingSoonEnabled: true });
    vi.mocked(waitlistHook.useWaitlistForm).mockReturnValue(state);
    rerender(<WaitlistManagement />);
    expect(screen.getByText('No emails collected yet')).toBeInTheDocument();
    expect(screen.getByText('Coming soon mode is active')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('switch', { name: 'Toggle coming soon mode' }));
    expect(state.handleToggleComingSoon).toHaveBeenCalledOnce();
  });

  it('shows collected customers and invokes export, refresh, and accessible deletion actions', () => {
    const state = formState({ emails: [{ id: 7, email: 'early@example.com', source: 'coming_soon', created_at: '2026-01-01T00:00:00Z' }], stats: { totalSignups: 3, comingSoonCount: 2 }, error: 'A previous export failed' });
    vi.mocked(waitlistHook.useWaitlistForm).mockReturnValue(state);
    render(<WaitlistManagement />);
    expect(screen.getByText('early@example.com')).toBeInTheDocument();
    expect(screen.getByText('A previous export failed')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Export CSV' }));
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete waitlist signup for early@example.com' }));
    expect(state.handleExport).toHaveBeenCalledOnce();
    expect(state.loadData).toHaveBeenCalledOnce();
    expect(state.handleDelete).toHaveBeenCalledWith(7);
  });
});
