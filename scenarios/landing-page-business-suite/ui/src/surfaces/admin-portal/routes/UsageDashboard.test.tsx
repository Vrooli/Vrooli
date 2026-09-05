import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { UsageDashboard } from './UsageDashboard';
import * as usageHook from '../hooks/useUsageDashboard';

vi.mock('../hooks/useUsageDashboard');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));

function dashboardState(overrides: Record<string, unknown> = {}) {
  return {
    summary: null, totalUsage: 0, sortedAppTotals: [], topUsers: [], recentRecords: [], formattedPeriod: 'January 2026', isCurrentPeriod: true, loading: false, fetchSummary: vi.fn(), navigateMonth: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof usageHook.useUsageDashboard>;
}

describe('UsageDashboard', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('shows loading and no-data states without presenting stale consumption information', () => {
    vi.mocked(usageHook.useUsageDashboard).mockReturnValue(dashboardState({ loading: true }));
    const { rerender } = render(<UsageDashboard />);
    expect(screen.getByText('Loading usage data...')).toBeInTheDocument();
    vi.mocked(usageHook.useUsageDashboard).mockReturnValue(dashboardState());
    rerender(<UsageDashboard />);
    expect(screen.getByText('No usage data available')).toBeInTheDocument();
  });

  it('renders usage, customer, and audit details and invokes refresh/month navigation', () => {
    const state = dashboardState({
      summary: { total_users: 2, total_records: 3 }, totalUsage: 1500000,
      sortedAppTotals: [{ app: 'Automation', usage: 1000000, percentage: 66.666 }], topUsers: [{ user: 'customer@example.com', usage: 1000000 }],
      recentRecords: [{ id: 'usage-1', user_identity: 'customer@example.com', limit_key: 'api_calls', app_bundle_key: 'business_suite', usage_amount: 500000, last_operation_at: '2026-01-02T03:04:05Z' }],
      isCurrentPeriod: false,
    });
    vi.mocked(usageHook.useUsageDashboard).mockReturnValue(state);
    render(<UsageDashboard />);
    expect(screen.getByText('Active Users')).toBeInTheDocument();
    expect(screen.getByText('Automation')).toBeInTheDocument();
    expect(screen.getAllByText('customer@example.com')).toHaveLength(2);
    expect(screen.getByText('api_calls')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    fireEvent.click(screen.getByRole('button', { name: 'Previous month' }));
    fireEvent.click(screen.getByRole('button', { name: 'Next month' }));
    expect(state.fetchSummary).toHaveBeenCalledOnce();
    expect(state.navigateMonth).toHaveBeenCalledWith(-1);
    expect(state.navigateMonth).toHaveBeenCalledWith(1);
  });

  it('disables forward navigation and refresh while the current reporting period is loading', () => {
    vi.mocked(usageHook.useUsageDashboard).mockReturnValue(dashboardState({ loading: true, isCurrentPeriod: true }));
    render(<UsageDashboard />);
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Next month' })).toBeDisabled();
  });
});
