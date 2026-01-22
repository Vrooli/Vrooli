import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { FeedbackRequest } from '../../../shared/api';
import {
  TYPE_CONFIG,
  STATUS_CONFIG,
  getTypeConfig,
  getStatusConfig,
  formatFeedbackDate,
  filterByStatus,
  filterByType,
  filterFeedback,
  countByStatus,
  buildReplyUrl,
  toggleSelection,
  toggleSelectAll,
  removeFromList,
  removeMultipleFromList,
  updateStatusInList,
  removeFromSelection,
} from './feedback.service';

const createMockFeedback = (overrides: Partial<FeedbackRequest> = {}): FeedbackRequest => ({
  id: 1,
  email: 'test@example.com',
  subject: 'Test Subject',
  message: 'Test message',
  type: 'general',
  status: 'pending',
  created_at: '2024-01-15T10:30:00Z',
  updated_at: '2024-01-15T10:30:00Z',
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

    it('each type has label and color', () => {
      Object.values(TYPE_CONFIG).forEach((config) => {
        expect(config.label).toBeTruthy();
        expect(config.color).toBeTruthy();
      });
    });
  });

  describe('STATUS_CONFIG', () => {
    it('has configuration for all statuses', () => {
      expect(STATUS_CONFIG.pending).toBeDefined();
      expect(STATUS_CONFIG.in_progress).toBeDefined();
      expect(STATUS_CONFIG.resolved).toBeDefined();
      expect(STATUS_CONFIG.rejected).toBeDefined();
    });

    it('each status has label and color', () => {
      Object.values(STATUS_CONFIG).forEach((config) => {
        expect(config.label).toBeTruthy();
        expect(config.color).toBeTruthy();
      });
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

  describe('formatFeedbackDate', () => {
    it('formats date string to locale string', () => {
      const result = formatFeedbackDate('2024-01-15T10:30:00Z');
      expect(typeof result).toBe('string');
      expect(result.length).toBeGreaterThan(0);
    });
  });

  describe('filterByStatus', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1, status: 'pending' }),
      createMockFeedback({ id: 2, status: 'resolved' }),
      createMockFeedback({ id: 3, status: 'pending' }),
    ];

    it('returns all when filter is "all"', () => {
      const result = filterByStatus(feedbackList, 'all');
      expect(result).toHaveLength(3);
    });

    it('filters by status', () => {
      const result = filterByStatus(feedbackList, 'pending');
      expect(result).toHaveLength(2);
      expect(result.every((f) => f.status === 'pending')).toBe(true);
    });
  });

  describe('filterByType', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1, type: 'bug' }),
      createMockFeedback({ id: 2, type: 'feature' }),
      createMockFeedback({ id: 3, type: 'bug' }),
    ];

    it('returns all when filter is "all"', () => {
      const result = filterByType(feedbackList, 'all');
      expect(result).toHaveLength(3);
    });

    it('filters by type', () => {
      const result = filterByType(feedbackList, 'bug');
      expect(result).toHaveLength(2);
      expect(result.every((f) => f.type === 'bug')).toBe(true);
    });
  });

  describe('filterFeedback', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1, status: 'pending', type: 'bug' }),
      createMockFeedback({ id: 2, status: 'resolved', type: 'bug' }),
      createMockFeedback({ id: 3, status: 'pending', type: 'feature' }),
    ];

    it('returns all when both filters are "all"', () => {
      const result = filterFeedback(feedbackList, 'all', 'all');
      expect(result).toHaveLength(3);
    });

    it('filters by status only', () => {
      const result = filterFeedback(feedbackList, 'pending', 'all');
      expect(result).toHaveLength(2);
    });

    it('filters by type only', () => {
      const result = filterFeedback(feedbackList, 'all', 'bug');
      expect(result).toHaveLength(2);
    });

    it('filters by both status and type', () => {
      const result = filterFeedback(feedbackList, 'pending', 'bug');
      expect(result).toHaveLength(1);
      expect(result[0].id).toBe(1);
    });
  });

  describe('countByStatus', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1, status: 'pending' }),
      createMockFeedback({ id: 2, status: 'pending' }),
      createMockFeedback({ id: 3, status: 'resolved' }),
      createMockFeedback({ id: 4, status: 'in_progress' }),
    ];

    it('counts feedback by status', () => {
      expect(countByStatus(feedbackList, 'pending')).toBe(2);
      expect(countByStatus(feedbackList, 'resolved')).toBe(1);
      expect(countByStatus(feedbackList, 'in_progress')).toBe(1);
      expect(countByStatus(feedbackList, 'rejected')).toBe(0);
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
  });

  describe('toggleSelection', () => {
    it('adds id to selection', () => {
      const selected = new Set([1, 2]);
      const result = toggleSelection(selected, 3);
      expect(result.has(3)).toBe(true);
      expect(result.size).toBe(3);
    });

    it('removes id from selection', () => {
      const selected = new Set([1, 2, 3]);
      const result = toggleSelection(selected, 2);
      expect(result.has(2)).toBe(false);
      expect(result.size).toBe(2);
    });

    it('does not mutate original set', () => {
      const selected = new Set([1, 2]);
      toggleSelection(selected, 3);
      expect(selected.size).toBe(2);
    });
  });

  describe('toggleSelectAll', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1 }),
      createMockFeedback({ id: 2 }),
      createMockFeedback({ id: 3 }),
    ];

    it('selects all when none selected', () => {
      const result = toggleSelectAll(new Set(), feedbackList);
      expect(result.size).toBe(3);
      expect(result.has(1)).toBe(true);
      expect(result.has(2)).toBe(true);
      expect(result.has(3)).toBe(true);
    });

    it('selects all when some selected', () => {
      const result = toggleSelectAll(new Set([1]), feedbackList);
      expect(result.size).toBe(3);
    });

    it('deselects all when all selected', () => {
      const result = toggleSelectAll(new Set([1, 2, 3]), feedbackList);
      expect(result.size).toBe(0);
    });
  });

  describe('removeFromList', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1 }),
      createMockFeedback({ id: 2 }),
      createMockFeedback({ id: 3 }),
    ];

    it('removes item by id', () => {
      const result = removeFromList(feedbackList, 2);
      expect(result).toHaveLength(2);
      expect(result.find((f) => f.id === 2)).toBeUndefined();
    });

    it('does not mutate original array', () => {
      removeFromList(feedbackList, 2);
      expect(feedbackList).toHaveLength(3);
    });

    it('returns same items if id not found', () => {
      const result = removeFromList(feedbackList, 999);
      expect(result).toHaveLength(3);
    });
  });

  describe('removeMultipleFromList', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1 }),
      createMockFeedback({ id: 2 }),
      createMockFeedback({ id: 3 }),
      createMockFeedback({ id: 4 }),
    ];

    it('removes multiple items by ids', () => {
      const result = removeMultipleFromList(feedbackList, new Set([2, 4]));
      expect(result).toHaveLength(2);
      expect(result.map((f) => f.id)).toEqual([1, 3]);
    });

    it('handles empty set', () => {
      const result = removeMultipleFromList(feedbackList, new Set());
      expect(result).toHaveLength(4);
    });
  });

  describe('updateStatusInList', () => {
    const feedbackList: FeedbackRequest[] = [
      createMockFeedback({ id: 1, status: 'pending' }),
      createMockFeedback({ id: 2, status: 'pending' }),
    ];

    it('updates feedback item in list', () => {
      const updated = createMockFeedback({ id: 1, status: 'resolved' });
      const result = updateStatusInList(feedbackList, 1, updated);
      expect(result[0].status).toBe('resolved');
      expect(result[1].status).toBe('pending');
    });

    it('does not mutate original array', () => {
      const updated = createMockFeedback({ id: 1, status: 'resolved' });
      updateStatusInList(feedbackList, 1, updated);
      expect(feedbackList[0].status).toBe('pending');
    });
  });

  describe('removeFromSelection', () => {
    it('removes id from selection', () => {
      const selected = new Set([1, 2, 3]);
      const result = removeFromSelection(selected, 2);
      expect(result.has(2)).toBe(false);
      expect(result.size).toBe(2);
    });

    it('does not mutate original set', () => {
      const selected = new Set([1, 2, 3]);
      removeFromSelection(selected, 2);
      expect(selected.size).toBe(3);
    });
  });
});
