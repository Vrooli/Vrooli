import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { UsersDashboard } from './UsersDashboard';
import { fetchFeedbackList } from '../../../shared/api/feedback';
import { getWaitlistEmails } from '../../../shared/api/waitlist';

const navigate = vi.fn();
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate }));
vi.mock('../../../shared/api/feedback', () => ({ fetchFeedbackList: vi.fn() }));
vi.mock('../../../shared/api/waitlist', () => ({ getWaitlistEmails: vi.fn() }));
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));

describe('UsersDashboard', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('aggregates pending feedback and waitlist customers, then routes triage actions correctly', async () => {
    vi.mocked(fetchFeedbackList).mockResolvedValue([{ status: 'pending' }, { status: 'in_progress' }, { status: 'resolved' }] as never);
    vi.mocked(getWaitlistEmails).mockResolvedValue([{ email: 'one@example.com' }, { email: 'two@example.com' }] as never);
    render(<UsersDashboard />);
    await waitFor(() => { expect(screen.getByText('2 pending')).toBeInTheDocument(); });
    expect(screen.getByTestId('users-stats')).toHaveTextContent('Pending feedback2');
    expect(screen.getByTestId('users-stats')).toHaveTextContent('Total feedback3');
    expect(screen.getByTestId('users-stats')).toHaveTextContent('Waitlist signups2');
    fireEvent.click(screen.getByTestId('flow-accounts'));
    fireEvent.click(screen.getByTestId('flow-feedback'));
    fireEvent.click(screen.getByTestId('flow-waitlist'));
    fireEvent.click(screen.getByTestId('users-feedback-triage'));
    expect(navigate).toHaveBeenNthCalledWith(1, '/admin/accounts');
    expect(navigate).toHaveBeenNthCalledWith(2, '/admin/feedback');
    expect(navigate).toHaveBeenNthCalledWith(3, '/admin/waitlist');
    expect(navigate).toHaveBeenLastCalledWith('/admin/feedback');
  });

  it('keeps customer operations available when either aggregate request fails and refreshes both sources', async () => {
    vi.mocked(fetchFeedbackList).mockRejectedValue(new Error('feedback unavailable'));
    vi.mocked(getWaitlistEmails).mockResolvedValue([{ email: 'waitlist@example.com' }] as never);
    render(<UsersDashboard />);
    await waitFor(() => { expect(screen.getByTestId('users-stats')).toHaveTextContent('Waitlist signups1'); });
    expect(screen.getByTestId('users-stats')).toHaveTextContent('Pending feedback0');
    expect(screen.getByTestId('users-stats')).toHaveTextContent('Waitlist signups1');
    fireEvent.click(screen.getByTestId('users-stats-refresh'));
    await waitFor(() => { expect(fetchFeedbackList).toHaveBeenCalledTimes(2); });
    expect(getWaitlistEmails).toHaveBeenCalledTimes(2);
  });
});
