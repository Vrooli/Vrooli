import type { FeedbackRequest } from '../../../shared/api';
import {
  fetchFeedbackList as apiFetchFeedbackList,
  updateFeedbackStatus as apiUpdateFeedbackStatus,
  deleteFeedback as apiDeleteFeedback,
  deleteFeedbackBulk as apiDeleteFeedbackBulk,
} from '../../../shared/api';

export type FeedbackType = 'refund' | 'bug' | 'feature' | 'general';
export type FeedbackStatus = 'pending' | 'in_progress' | 'resolved' | 'rejected';

/**
 * Configuration for feedback type display
 */
export interface TypeConfig {
  label: string;
  color: string;
}

/**
 * Configuration for feedback status display
 */
export interface StatusConfig {
  label: string;
  color: string;
}

/**
 * Type configurations for display
 */
export const TYPE_CONFIG: Record<FeedbackType, TypeConfig> = {
  refund: {
    label: 'Refund Request',
    color: 'text-amber-400 bg-amber-500/10 border-amber-500/20',
  },
  bug: {
    label: 'Bug Report',
    color: 'text-red-400 bg-red-500/10 border-red-500/20',
  },
  feature: {
    label: 'Feature Request',
    color: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  },
  general: {
    label: 'General Feedback',
    color: 'text-purple-400 bg-purple-500/10 border-purple-500/20',
  },
};

/**
 * Status configurations for display
 */
export const STATUS_CONFIG: Record<FeedbackStatus, StatusConfig> = {
  pending: {
    label: 'Pending',
    color: 'text-slate-400 bg-slate-500/10 border-slate-500/20',
  },
  in_progress: {
    label: 'In Progress',
    color: 'text-blue-400 bg-blue-500/10 border-blue-500/20',
  },
  resolved: {
    label: 'Resolved',
    color: 'text-green-400 bg-green-500/10 border-green-500/20',
  },
  rejected: {
    label: 'Rejected',
    color: 'text-red-400 bg-red-500/10 border-red-500/20',
  },
};

/**
 * Get type configuration for feedback
 */
export function getTypeConfig(type: string): TypeConfig {
  return TYPE_CONFIG[type as FeedbackType] || TYPE_CONFIG.general;
}

/**
 * Get status configuration for feedback
 */
export function getStatusConfig(status: string): StatusConfig {
  return STATUS_CONFIG[status as FeedbackStatus] || STATUS_CONFIG.pending;
}

/**
 * Format date for display
 */
export function formatFeedbackDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

/**
 * Filter feedback by status
 */
export function filterByStatus(
  feedbackList: FeedbackRequest[],
  statusFilter: string
): FeedbackRequest[] {
  if (statusFilter === 'all') {
    return feedbackList;
  }
  return feedbackList.filter((f) => f.status === statusFilter);
}

/**
 * Filter feedback by type
 */
export function filterByType(
  feedbackList: FeedbackRequest[],
  typeFilter: string
): FeedbackRequest[] {
  if (typeFilter === 'all') {
    return feedbackList;
  }
  return feedbackList.filter((f) => f.type === typeFilter);
}

/**
 * Apply both filters to feedback list
 */
export function filterFeedback(
  feedbackList: FeedbackRequest[],
  statusFilter: string,
  typeFilter: string
): FeedbackRequest[] {
  return feedbackList.filter((f) => {
    if (statusFilter !== 'all' && f.status !== statusFilter) return false;
    if (typeFilter !== 'all' && f.type !== typeFilter) return false;
    return true;
  });
}

/**
 * Count feedback by status
 */
export function countByStatus(
  feedbackList: FeedbackRequest[],
  status: FeedbackStatus
): number {
  return feedbackList.filter((f) => f.status === status).length;
}

/**
 * Build mailto URL for replying to feedback
 */
export function buildReplyUrl(email: string, subject: string): string {
  return `mailto:${email}?subject=Re: ${encodeURIComponent(subject)}`;
}

/**
 * Open email reply in new window
 */
export function openEmailReply(email: string, subject: string): void {
  window.open(buildReplyUrl(email, subject), '_blank');
}

/**
 * Toggle selection of feedback item
 */
export function toggleSelection(
  selectedIds: Set<number>,
  id: number
): Set<number> {
  const next = new Set(selectedIds);
  if (next.has(id)) {
    next.delete(id);
  } else {
    next.add(id);
  }
  return next;
}

/**
 * Select all or deselect all items
 */
export function toggleSelectAll(
  selectedIds: Set<number>,
  feedbackList: FeedbackRequest[]
): Set<number> {
  if (selectedIds.size === feedbackList.length) {
    return new Set();
  }
  return new Set(feedbackList.map((f) => f.id));
}

/**
 * Remove item from feedback list (for optimistic updates)
 */
export function removeFromList(
  feedbackList: FeedbackRequest[],
  id: number
): FeedbackRequest[] {
  return feedbackList.filter((f) => f.id !== id);
}

/**
 * Remove multiple items from feedback list (for bulk delete)
 */
export function removeMultipleFromList(
  feedbackList: FeedbackRequest[],
  ids: Set<number>
): FeedbackRequest[] {
  return feedbackList.filter((f) => !ids.has(f.id));
}

/**
 * Update feedback status in list (for optimistic updates)
 */
export function updateStatusInList(
  feedbackList: FeedbackRequest[],
  id: number,
  updated: FeedbackRequest
): FeedbackRequest[] {
  return feedbackList.map((f) => (f.id === id ? updated : f));
}

/**
 * Remove id from selection set
 */
export function removeFromSelection(
  selectedIds: Set<number>,
  id: number
): Set<number> {
  const next = new Set(selectedIds);
  next.delete(id);
  return next;
}

// API wrapper functions

export async function fetchFeedbackList(): Promise<FeedbackRequest[]> {
  return apiFetchFeedbackList();
}

export async function updateFeedbackStatus(
  id: number,
  status: FeedbackStatus
): Promise<FeedbackRequest> {
  return apiUpdateFeedbackStatus(id, status);
}

export async function deleteFeedback(id: number): Promise<{ success: boolean; id: number }> {
  return apiDeleteFeedback(id);
}

export async function deleteFeedbackBulk(ids: number[]): Promise<{ success: boolean; deleted: number }> {
  return apiDeleteFeedbackBulk(ids);
}
