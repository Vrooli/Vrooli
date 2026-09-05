import type { FeedbackRequest } from '../../../shared/api';
import {
  fetchFeedbackList as apiFetchFeedbackList,
  updateFeedbackStatus as apiUpdateFeedbackStatus,
  deleteFeedback as apiDeleteFeedback,
  deleteFeedbackBulk as apiDeleteFeedbackBulk,
} from '../../../shared/api';
import {
  type FeedbackVariant,
  getTypeStyles,
  getStatusStyles,
} from '../styles/feedback.styles';

export type FeedbackType = 'refund' | 'bug' | 'feature' | 'general';
export type FeedbackStatus = 'pending' | 'in_progress' | 'resolved' | 'rejected';

/**
 * Configuration for feedback type display
 */
export interface TypeConfig {
  label: string;
  variant: FeedbackVariant;
}

/**
 * Configuration for feedback status display
 */
export interface StatusConfig {
  label: string;
  variant: FeedbackVariant;
}

/**
 * Type configurations for display (semantic variants, not Tailwind classes)
 */
export const TYPE_CONFIG: Record<FeedbackType, TypeConfig> = {
  refund: {
    label: 'Refund Request',
    variant: 'warning',
  },
  bug: {
    label: 'Bug Report',
    variant: 'danger',
  },
  feature: {
    label: 'Feature Request',
    variant: 'info',
  },
  general: {
    label: 'General Feedback',
    variant: 'primary',
  },
};

/**
 * Status configurations for display (semantic variants, not Tailwind classes)
 */
export const STATUS_CONFIG: Record<FeedbackStatus, StatusConfig> = {
  pending: {
    label: 'Pending',
    variant: 'neutral',
  },
  in_progress: {
    label: 'In Progress',
    variant: 'info',
  },
  resolved: {
    label: 'Resolved',
    variant: 'success',
  },
  rejected: {
    label: 'Rejected',
    variant: 'danger',
  },
};

function isFeedbackType(type: string): type is FeedbackType {
  return Object.prototype.hasOwnProperty.call(TYPE_CONFIG, type);
}

function isFeedbackStatus(status: string): status is FeedbackStatus {
  return Object.prototype.hasOwnProperty.call(STATUS_CONFIG, status);
}

/**
 * Get type configuration for feedback
 */
export function getTypeConfig(type: string): TypeConfig {
  return isFeedbackType(type) ? TYPE_CONFIG[type] : TYPE_CONFIG.general;
}

/**
 * Get Tailwind CSS classes for a feedback type.
 * Use this when you need the actual CSS classes for rendering.
 */
export function getTypeColor(type: string): string {
  return getTypeStyles(type);
}

/**
 * Get status configuration for feedback
 */
export function getStatusConfig(status: string): StatusConfig {
  return isFeedbackStatus(status) ? STATUS_CONFIG[status] : STATUS_CONFIG.pending;
}

/**
 * Get Tailwind CSS classes for a feedback status.
 * Use this when you need the actual CSS classes for rendering.
 */
export function getStatusColor(status: string): string {
  return getStatusStyles(status);
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
