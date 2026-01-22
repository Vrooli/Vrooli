import type { AdminUsageSummary, UsageRecord } from '../../../shared/api';
import { getAdminUsageSummary as apiGetAdminUsageSummary, formatCredits } from '../../../shared/api';

/**
 * Get current billing period in YYYY-MM format
 */
export function getCurrentBillingPeriod(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Navigate to next or previous month
 */
export function navigateToBillingPeriod(
  currentPeriod: string,
  delta: number
): string {
  const [year, month] = currentPeriod.split('-').map(Number);
  const date = new Date(year, month - 1 + delta, 1);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Check if period is current month (cannot navigate forward)
 */
export function isCurrentMonth(period: string): boolean {
  return period === getCurrentBillingPeriod();
}

/**
 * Format billing period for display
 */
export function formatBillingPeriod(period: string): string {
  const [year, month] = period.split('-').map(Number);
  return new Date(year, month - 1, 1).toLocaleDateString('en-US', {
    month: 'long',
    year: 'numeric',
  });
}

/**
 * Calculate total usage from user totals
 */
export function calculateTotalUsage(
  userTotals: Record<string, number> | undefined
): number {
  if (!userTotals) return 0;
  return Object.values(userTotals).reduce((sum, val) => sum + val, 0);
}

/**
 * Calculate usage percentage for an app
 */
export function calculateUsagePercentage(
  appUsage: number,
  totalUsage: number
): number {
  if (totalUsage === 0) return 0;
  return (appUsage / totalUsage) * 100;
}

/**
 * Sort entries by usage descending
 */
export function sortByUsageDesc<T extends [string, number]>(
  entries: T[]
): T[] {
  return [...entries].sort((a, b) => b[1] - a[1]);
}

/**
 * Get top N users by usage
 */
export function getTopUsers(
  userTotals: Record<string, number>,
  limit = 10
): Array<{ user: string; usage: number }> {
  return Object.entries(userTotals)
    .sort(([, a], [, b]) => b - a)
    .slice(0, limit)
    .map(([user, usage]) => ({ user, usage }));
}

/**
 * Get app totals sorted by usage
 */
export function getSortedAppTotals(
  appTotals: Record<string, number>
): Array<{ app: string; usage: number; percentage: number }> {
  const totalUsage = Object.values(appTotals).reduce((sum, val) => sum + val, 0);

  return Object.entries(appTotals)
    .sort(([, a], [, b]) => b - a)
    .map(([app, usage]) => ({
      app,
      usage,
      percentage: calculateUsagePercentage(usage, totalUsage),
    }));
}

/**
 * Format date for activity display
 */
export function formatActivityDate(dateStr: string | null | undefined): string {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString();
}

/**
 * Get limited records for display
 */
export function getLimitedRecords(
  records: UsageRecord[],
  limit = 20
): UsageRecord[] {
  return records.slice(0, limit);
}

// API wrapper functions

/**
 * Fetch admin usage summary
 */
export async function fetchUsageSummary(
  billingPeriod: string
): Promise<AdminUsageSummary> {
  return apiGetAdminUsageSummary(billingPeriod);
}

// Re-export for convenience
export { formatCredits };
