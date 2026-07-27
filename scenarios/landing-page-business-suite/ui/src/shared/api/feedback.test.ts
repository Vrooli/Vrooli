import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as feedback from './feedback';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('feedback API transport', () => {
  beforeEach(() => { vi.clearAllMocks(); mockApiCall.mockResolvedValue({} as never); });

  it('uses public feedback creation and all scoped administration endpoints', async () => {
    await feedback.createFeedback({ type: 'bug', email: 'customer@example.com', subject: 'Problem', message: 'Details' });
    await feedback.fetchFeedbackList();
    await feedback.fetchFeedbackList('in progress');
    await feedback.fetchFeedbackById(7);
    await feedback.updateFeedbackStatus(7, 'resolved');
    await feedback.deleteFeedback(7);
    await feedback.deleteFeedbackBulk([7, 8]);
    expect(mockApiCall).toHaveBeenCalledWith('/feedback', {
      method: 'POST',
      body: JSON.stringify({ type: 'bug', email: 'customer@example.com', subject: 'Problem', message: 'Details' }),
    });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback?status=in%20progress');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback/7');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback/7/status', expect.objectContaining({ method: 'PATCH', body: JSON.stringify({ status: 'resolved' }) }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback/7', { method: 'DELETE' });
    expect(mockApiCall).toHaveBeenCalledWith('/admin/feedback/bulk-delete', expect.objectContaining({ method: 'POST', body: JSON.stringify({ ids: [7, 8] }) }));
  });
});
