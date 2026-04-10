import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  TYPE_CONFIG,
  STATUS_CONFIG,
  getTypeConfig,
  getStatusConfig,
  getTypeColor,
  getStatusColor,
  buildReplyUrl,
  openEmailReply,
  fetchFeedbackList,
  updateFeedbackStatus,
  deleteFeedback,
  deleteFeedbackBulk,
} from './feedback.service';
import * as feedbackApi from '../../../shared/api/feedback';
import { createWindowOpenMock } from '../../../shared/test-utils/api-mocks';
import type { FeedbackRequest } from '../../../shared/api';

vi.mock('../../../shared/api/feedback', () => ({
  fetchFeedbackList: vi.fn(),
  updateFeedbackStatus: vi.fn(),
  deleteFeedback: vi.fn(),
  deleteFeedbackBulk: vi.fn(),
}));

const createMockFeedback = (overrides: Partial<FeedbackRequest> = {}): FeedbackRequest => ({
  id: 1,
  type: 'bug',
  email: 'user@example.com',
  subject: 'Test Subject',
  message: 'Test message',
  status: 'pending',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('feedback.service', () => {
  describe('TYPE_CONFIG', () => {
    it('has configuration for all feedback types', () => {
      expect(TYPE_CONFIG.refund).toBeDefined();
      expect(TYPE_CONFIG.bug).toBeDefined();
      expect(TYPE_CONFIG.feature).toBeDefined();
      expect(TYPE_CONFIG.general).toBeDefined();
    });

    it('each type has label and variant', () => {
      Object.values(TYPE_CONFIG).forEach((config) => {
        expect(config.label).toBeTruthy();
        expect(config.variant).toBeTruthy();
      });
    });

    it('maps types to correct semantic variants', () => {
      expect(TYPE_CONFIG.refund.variant).toBe('warning');
      expect(TYPE_CONFIG.bug.variant).toBe('danger');
      expect(TYPE_CONFIG.feature.variant).toBe('info');
      expect(TYPE_CONFIG.general.variant).toBe('primary');
    });
  });

  describe('STATUS_CONFIG', () => {
    it('has configuration for all statuses', () => {
      expect(STATUS_CONFIG.pending).toBeDefined();
      expect(STATUS_CONFIG.in_progress).toBeDefined();
      expect(STATUS_CONFIG.resolved).toBeDefined();
      expect(STATUS_CONFIG.rejected).toBeDefined();
    });

    it('each status has label and variant', () => {
      Object.values(STATUS_CONFIG).forEach((config) => {
        expect(config.label).toBeTruthy();
        expect(config.variant).toBeTruthy();
      });
    });

    it('maps statuses to correct semantic variants', () => {
      expect(STATUS_CONFIG.pending.variant).toBe('neutral');
      expect(STATUS_CONFIG.in_progress.variant).toBe('info');
      expect(STATUS_CONFIG.resolved.variant).toBe('success');
      expect(STATUS_CONFIG.rejected.variant).toBe('danger');
    });
  });

  describe('getTypeColor', () => {
    it('returns Tailwind classes for known types', () => {
      const bugColor = getTypeColor('bug');
      expect(bugColor).toContain('text-');
      expect(bugColor).toContain('bg-');
      expect(bugColor).toContain('border-');
    });

    it('returns primary variant classes for unknown types', () => {
      const unknownColor = getTypeColor('unknown');
      expect(unknownColor).toContain('text-purple');
    });
  });

  describe('getStatusColor', () => {
    it('returns Tailwind classes for known statuses', () => {
      const resolvedColor = getStatusColor('resolved');
      expect(resolvedColor).toContain('text-green');
    });

    it('returns neutral variant classes for unknown statuses', () => {
      const unknownColor = getStatusColor('unknown');
      expect(unknownColor).toContain('text-slate');
    });
  });

  describe('getTypeConfig', () => {
    it('returns config for known types', () => {
      expect(getTypeConfig('bug')).toBe(TYPE_CONFIG.bug);
      expect(getTypeConfig('feature')).toBe(TYPE_CONFIG.feature);
    });

    it('returns general config for unknown types', () => {
      expect(getTypeConfig('unknown')).toBe(TYPE_CONFIG.general);
    });
  });

  describe('getStatusConfig', () => {
    it('returns config for known statuses', () => {
      expect(getStatusConfig('pending')).toBe(STATUS_CONFIG.pending);
      expect(getStatusConfig('resolved')).toBe(STATUS_CONFIG.resolved);
    });

    it('returns pending config for unknown statuses', () => {
      expect(getStatusConfig('unknown')).toBe(STATUS_CONFIG.pending);
    });
  });

  describe('buildReplyUrl', () => {
    it('builds mailto URL with encoded subject', () => {
      const url = buildReplyUrl('user@example.com', 'Test Subject');
      expect(url).toBe('mailto:user@example.com?subject=Re: Test%20Subject');
    });

    it('handles special characters in subject', () => {
      const url = buildReplyUrl('user@example.com', 'Test & Special <chars>');
      expect(url).toContain('mailto:user@example.com');
      expect(url).toContain('subject=Re:');
    });

    it('handles empty subject', () => {
      const url = buildReplyUrl('user@example.com', '');
      expect(url).toBe('mailto:user@example.com?subject=Re: ');
    });

    it('handles unicode characters', () => {
      const url = buildReplyUrl('user@example.com', 'Test 你好');
      expect(url).toContain('mailto:user@example.com');
      expect(url).toContain('subject=Re:');
    });
  });

  describe('openEmailReply', () => {
    let windowMock: ReturnType<typeof createWindowOpenMock>;

    beforeEach(() => {
      windowMock = createWindowOpenMock();
    });

    afterEach(() => {
      windowMock.restore();
    });

    it('opens mailto URL in new window', () => {
      openEmailReply('user@example.com', 'Test Subject');
      expect(windowMock.mock).toHaveBeenCalledWith(
        'mailto:user@example.com?subject=Re: Test%20Subject',
        '_blank'
      );
    });
  });

  describe('fetchFeedbackList', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('returns feedback list from API', async () => {
      const mockFeedback = [createMockFeedback({ id: 1 }), createMockFeedback({ id: 2 })];
      vi.mocked(feedbackApi.fetchFeedbackList).mockResolvedValue(mockFeedback);

      const result = await fetchFeedbackList();

      expect(result).toEqual(mockFeedback);
      expect(feedbackApi.fetchFeedbackList).toHaveBeenCalledTimes(1);
    });

    it('propagates API errors', async () => {
      vi.mocked(feedbackApi.fetchFeedbackList).mockRejectedValue(new Error('Network error'));

      await expect(fetchFeedbackList()).rejects.toThrow('Network error');
    });

    it('returns empty array when API returns empty', async () => {
      vi.mocked(feedbackApi.fetchFeedbackList).mockResolvedValue([]);

      const result = await fetchFeedbackList();

      expect(result).toEqual([]);
    });
  });

  describe('updateFeedbackStatus', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('updates status and returns updated feedback', async () => {
      const updatedFeedback = createMockFeedback({ id: 1, status: 'resolved' });
      vi.mocked(feedbackApi.updateFeedbackStatus).mockResolvedValue(updatedFeedback);

      const result = await updateFeedbackStatus(1, 'resolved');

      expect(result).toEqual(updatedFeedback);
      expect(feedbackApi.updateFeedbackStatus).toHaveBeenCalledWith(1, 'resolved');
    });

    it('propagates API errors', async () => {
      vi.mocked(feedbackApi.updateFeedbackStatus).mockRejectedValue(new Error('Not found'));

      await expect(updateFeedbackStatus(999, 'resolved')).rejects.toThrow('Not found');
    });
  });

  describe('deleteFeedback', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('deletes feedback and returns success', async () => {
      vi.mocked(feedbackApi.deleteFeedback).mockResolvedValue({ success: true, id: 1 });

      const result = await deleteFeedback(1);

      expect(result).toEqual({ success: true, id: 1 });
      expect(feedbackApi.deleteFeedback).toHaveBeenCalledWith(1);
    });

    it('propagates API errors', async () => {
      vi.mocked(feedbackApi.deleteFeedback).mockRejectedValue(new Error('Forbidden'));

      await expect(deleteFeedback(1)).rejects.toThrow('Forbidden');
    });
  });

  describe('deleteFeedbackBulk', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('deletes multiple feedback items', async () => {
      vi.mocked(feedbackApi.deleteFeedbackBulk).mockResolvedValue({ success: true, deleted: 3 });

      const result = await deleteFeedbackBulk([1, 2, 3]);

      expect(result).toEqual({ success: true, deleted: 3 });
      expect(feedbackApi.deleteFeedbackBulk).toHaveBeenCalledWith([1, 2, 3]);
    });

    it('handles empty array', async () => {
      vi.mocked(feedbackApi.deleteFeedbackBulk).mockResolvedValue({ success: true, deleted: 0 });

      const result = await deleteFeedbackBulk([]);

      expect(result).toEqual({ success: true, deleted: 0 });
    });

    it('propagates API errors', async () => {
      vi.mocked(feedbackApi.deleteFeedbackBulk).mockRejectedValue(new Error('Server error'));

      await expect(deleteFeedbackBulk([1, 2])).rejects.toThrow('Server error');
    });
  });
});
