import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FeedbackManagement } from './FeedbackManagement';
import * as feedbackHook from '../hooks/useFeedbackManagement';

vi.mock('../hooks/useFeedbackManagement');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title, actions }: { title: string; actions?: React.ReactNode }) => <><h1>{title}</h1>{actions}</> }));

function managementState(overrides: Record<string, unknown> = {}) {
  return { feedbackList: [], filteredFeedback: [], pendingCount: 0, inProgressCount: 0, statusFilter: 'all', typeFilter: 'all', selectedIds: new Set(), expandedId: null, loading: false, error: null, actionLoading: null, bulkActionLoading: false, setStatusFilter: vi.fn(), setTypeFilter: vi.fn(), handleToggleSelect: vi.fn(), handleToggleSelectAll: vi.fn(), setExpandedId: vi.fn(), loadFeedback: vi.fn(), handleStatusChange: vi.fn(), handleDelete: vi.fn(), handleBulkDelete: vi.fn(), handleReply: vi.fn(), ...overrides } as unknown as ReturnType<typeof feedbackHook.useFeedbackManagement>;
}

describe('FeedbackManagement', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('confirm', vi.fn(() => true)); });

  it('shows loading and error recovery states without rendering stale feedback actions', () => {
    vi.mocked(feedbackHook.useFeedbackManagement).mockReturnValue(managementState({ loading: true }));
    const { rerender } = render(<FeedbackManagement />);
    expect(screen.getByText('Loading feedback...')).toBeInTheDocument();
    const failed = managementState({ error: 'Feedback service unavailable' });
    vi.mocked(feedbackHook.useFeedbackManagement).mockReturnValue(failed);
    rerender(<FeedbackManagement />);
    expect(screen.getByText('Error: Feedback service unavailable')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(failed.loadFeedback).toHaveBeenCalledOnce();
  });

  it('shows filtered empty-state guidance and refreshes the feedback queue', () => {
    const state = managementState({ statusFilter: 'pending' });
    vi.mocked(feedbackHook.useFeedbackManagement).mockReturnValue(state);
    render(<FeedbackManagement />);
    expect(screen.getByText('No feedback found')).toBeInTheDocument();
    expect(screen.getByText('Try adjusting your filters')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(state.loadFeedback).toHaveBeenCalledOnce();
  });

  it('expands feedback for reply and deletion, and supports selected-item bulk deletion', async () => {
    const feedback = { id: 12, type: 'bug', status: 'pending', subject: 'Checkout failed', email: 'customer@example.com', created_at: '2026-01-01T00:00:00Z', message: 'The payment page returned an error.', order_id: 'order-1' };
    const state = managementState({ feedbackList: [feedback], filteredFeedback: [feedback], pendingCount: 1, selectedIds: new Set([12]), expandedId: 12 });
    vi.mocked(feedbackHook.useFeedbackManagement).mockReturnValue(state);
    render(<FeedbackManagement />);
    expect(screen.getByText('The payment page returned an error.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Reply via Email' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Delete Selected' }));
    await waitFor(() => { expect(state.handleDelete).toHaveBeenCalledWith(12); });
    expect(state.handleReply).toHaveBeenCalledWith('customer@example.com', 'Checkout failed');
    expect(state.handleBulkDelete).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: 'Deselect' }));
    fireEvent.click(screen.getByRole('button', { name: 'Deselect all' }));
    expect(state.handleToggleSelect).toHaveBeenCalledWith(12);
    expect(state.handleToggleSelectAll).toHaveBeenCalledOnce();
  });
});
