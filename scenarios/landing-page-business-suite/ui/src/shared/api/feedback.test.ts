import { beforeEach, describe, expect, it, vi } from 'vitest';

const feedbackClient = vi.hoisted(() => ({ createFeedback: vi.fn(), listFeedback: vi.fn(), getFeedback: vi.fn(), updateFeedbackStatus: vi.fn(), deleteFeedback: vi.fn(), deleteFeedbackBulk: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => feedbackClient) }));

import * as feedback from './feedback';

describe('feedback API transport', () => {
  beforeEach(() => vi.clearAllMocks());

  it('uses generated Connect procedures for public creation and admin lifecycle operations', async () => {
    feedbackClient.createFeedback.mockResolvedValue({ success: true, id: 7n });
    feedbackClient.listFeedback.mockResolvedValue({ feedback: [] });
    feedbackClient.getFeedback.mockResolvedValue({ feedback: { id: 7n, type: 2, email: 'customer@example.com', subject: 'Problem', message: 'Details', status: 1, createdAt: { seconds: 0n, nanos: 0 }, updatedAt: { seconds: 0n, nanos: 0 } } });
    feedbackClient.updateFeedbackStatus.mockResolvedValue({ feedback: { id: 7n, type: 2, email: 'customer@example.com', subject: 'Problem', message: 'Details', status: 3, createdAt: { seconds: 0n, nanos: 0 }, updatedAt: { seconds: 0n, nanos: 0 } } });
    feedbackClient.deleteFeedback.mockResolvedValue({ deleted: true, id: 7n });
    feedbackClient.deleteFeedbackBulk.mockResolvedValue({ deleted: 2n });

    await expect(feedback.createFeedback({ type: 'bug', email: 'customer@example.com', subject: 'Problem', message: 'Details' })).resolves.toEqual({ success: true, id: 7 });
    await expect(feedback.fetchFeedbackList('pending')).resolves.toEqual([]);
    await expect(feedback.fetchFeedbackById(7)).resolves.toMatchObject({ id: 7, type: 'bug', status: 'pending' });
    await expect(feedback.updateFeedbackStatus(7, 'resolved')).resolves.toMatchObject({ id: 7, status: 'resolved' });
    await expect(feedback.deleteFeedback(7)).resolves.toEqual({ success: true, id: 7 });
    await expect(feedback.deleteFeedbackBulk([7, 8])).resolves.toEqual({ success: true, deleted: 2 });
    expect(feedbackClient.createFeedback).toHaveBeenCalledWith({ type: 'bug', email: 'customer@example.com', subject: 'Problem', message: 'Details' }, { signal: undefined });
    expect(feedbackClient.listFeedback).toHaveBeenCalledWith({ status: 1 });
    expect(feedbackClient.updateFeedbackStatus).toHaveBeenCalledWith({ id: 7n, status: 3 });
    expect(feedbackClient.deleteFeedbackBulk).toHaveBeenCalledWith({ ids: [7n, 8n] });
  });
});
