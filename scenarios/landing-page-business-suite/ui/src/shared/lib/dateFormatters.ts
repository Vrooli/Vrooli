/**
 * Consolidated date formatting utilities for the admin portal.
 *
 * These functions standardize date formatting across the application,
 * replacing various ad-hoc implementations in individual services.
 */

/** Milliseconds in a day - used for relative time calculations */
export const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Style options for formatDateTime
 */
export type DateTimeStyle = 'full' | 'short';

/**
 * Options for formatRelativeTime
 */
export interface RelativeTimeOptions {
  /** Label to use when date is null/undefined/invalid (default: 'Never') */
  nullLabel?: string;
  /** Label to use when date is today (default: 'Today') */
  todayLabel?: string;
  /** Label to use when date is yesterday (default: 'Yesterday') */
  yesterdayLabel?: string;
  /** Template for other days, where {days} is replaced with day count (default: '{days} days ago') */
  daysAgoTemplate?: string;
  /** Prefix to add to all labels (default: '') */
  prefix?: string;
}

/**
 * Check if a date string is valid and can be parsed
 */
export function isValidDate(dateStr: string | null | undefined): boolean {
  if (!dateStr) return false;
  const parsed = new Date(dateStr);
  return !Number.isNaN(parsed.getTime());
}

/**
 * Format a date/time for display.
 *
 * Replaces: waitlist.service.ts formatDate(), usage.service.ts formatActivityDate()
 *
 * @param dateStr - ISO date string to format
 * @param style - 'full' for complete date/time, 'short' for abbreviated
 * @returns Formatted date string, or '-' if date is invalid/missing
 *
 * @example
 * formatDateTime('2024-01-15T14:30:00Z', 'full')  // "1/15/2024, 2:30:00 PM"
 * formatDateTime('2024-01-15T14:30:00Z', 'short') // "1/15/24, 2:30 PM"
 * formatDateTime(null, 'full')                    // "-"
 */
export function formatDateTime(
  dateStr: string | null | undefined,
  style: DateTimeStyle = 'full'
): string {
  if (!dateStr) return '-';

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return '-';

  if (style === 'short') {
    return date.toLocaleString('en-US', {
      month: 'numeric',
      day: 'numeric',
      year: '2-digit',
      hour: 'numeric',
      minute: '2-digit',
    });
  }

  return date.toLocaleString();
}

/**
 * Format a date for display (date only, no time).
 *
 * Replaces: apiKeys.service.ts formatVerifiedDate()
 *
 * @param dateStr - ISO date string to format
 * @returns Formatted date string, or null if date is invalid/missing
 *
 * @example
 * formatDateOnly('2024-01-15T14:30:00Z') // "1/15/2024"
 * formatDateOnly(null)                    // null
 */
export function formatDateOnly(dateStr: string | null | undefined): string | null {
  if (!dateStr) return null;

  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return null;

  return date.toLocaleDateString();
}

/**
 * Format a billing period (YYYY-MM) for display.
 *
 * Replaces: usage.service.ts formatBillingPeriod()
 *
 * @param period - Period string in YYYY-MM format
 * @returns Formatted string like "January 2024"
 *
 * @example
 * formatMonthYear('2024-01') // "January 2024"
 * formatMonthYear('2024-12') // "December 2024"
 */
export function formatMonthYear(period: string): string {
  if (!period || !period.includes('-')) {
    return period; // Return as-is if empty or invalid format
  }

  const [year, month] = period.split('-').map(Number);
  if (Number.isNaN(year) || Number.isNaN(month)) {
    return period; // Return as-is if parsing fails
  }

  const date = new Date(year, month - 1, 1);
  return date.toLocaleDateString('en-US', {
    month: 'long',
    year: 'numeric',
  });
}

/**
 * Calculate days since a date.
 *
 * @param dateStr - ISO date string
 * @returns Number of days since the date, or null if invalid/missing
 *
 * @example
 * calculateDaysSince('2024-01-15T00:00:00Z') // 5 (if today is Jan 20)
 * calculateDaysSince(null)                   // null
 */
export function calculateDaysSince(dateStr: string | null | undefined): number | null {
  if (!dateStr) return null;

  const parsed = new Date(dateStr);
  if (Number.isNaN(parsed.getTime())) return null;

  return Math.max(0, Math.floor((Date.now() - parsed.getTime()) / DAY_MS));
}

/**
 * Format a date as relative time (e.g., "Updated today", "3 days ago").
 *
 * Replaces: adminHome.service.ts formatVariantUpdatedLabel()
 *
 * @param dateStr - ISO date string to format
 * @param options - Customization options for labels
 * @returns Human-readable relative time string
 *
 * @example
 * formatRelativeTime('2024-01-20T00:00:00Z') // "Today" (if today)
 * formatRelativeTime('2024-01-19T00:00:00Z') // "Yesterday" (if yesterday)
 * formatRelativeTime('2024-01-15T00:00:00Z') // "5 days ago"
 * formatRelativeTime(null)                    // "Never"
 *
 * // With prefix
 * formatRelativeTime('2024-01-20', { prefix: 'Updated ' }) // "Updated today"
 *
 * // With custom labels
 * formatRelativeTime(null, { nullLabel: 'Never customized' }) // "Never customized"
 */
export function formatRelativeTime(
  dateStr: string | null | undefined,
  options: RelativeTimeOptions = {}
): string {
  const {
    nullLabel = 'Never',
    todayLabel = 'Today',
    yesterdayLabel = 'Yesterday',
    daysAgoTemplate = '{days} days ago',
    prefix = '',
  } = options;

  if (!dateStr) {
    return nullLabel;
  }

  const parsed = new Date(dateStr);
  if (Number.isNaN(parsed.getTime())) {
    return nullLabel;
  }

  const diffMs = Date.now() - parsed.getTime();
  const diffDays = Math.max(0, Math.floor(diffMs / DAY_MS));

  if (diffDays === 0) {
    return prefix + todayLabel.toLowerCase();
  }
  if (diffDays === 1) {
    return prefix + yesterdayLabel.toLowerCase();
  }

  return prefix + daysAgoTemplate.replace('{days}', String(diffDays));
}

/**
 * Format a timestamp for feedback display (detailed format).
 *
 * Replaces: feedback.service.ts formatFeedbackDate()
 *
 * @param dateStr - ISO date string to format
 * @returns Formatted string like "Jan 15, 2024, 2:30 PM"
 *
 * @example
 * formatFeedbackTimestamp('2024-01-15T14:30:00Z') // "Jan 15, 2024, 2:30 PM"
 */
export function formatFeedbackTimestamp(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr; // Return as-is if parsing fails
  }

  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

/**
 * Get the current billing period in YYYY-MM format.
 *
 * Kept here for convenience alongside formatMonthYear.
 *
 * @returns Current period string like "2024-01"
 */
export function getCurrentPeriod(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Navigate to next or previous month from a billing period.
 *
 * @param currentPeriod - Period string in YYYY-MM format
 * @param delta - Number of months to navigate (+1 for next, -1 for previous)
 * @returns New period string
 *
 * @example
 * navigatePeriod('2024-01', 1)  // "2024-02"
 * navigatePeriod('2024-01', -1) // "2023-12"
 */
export function navigatePeriod(currentPeriod: string, delta: number): string {
  const [year, month] = currentPeriod.split('-').map(Number);
  const date = new Date(year, month - 1 + delta, 1);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Check if a period is the current month.
 *
 * @param period - Period string in YYYY-MM format
 * @returns true if the period is the current month
 */
export function isCurrentPeriod(period: string): boolean {
  return period === getCurrentPeriod();
}
