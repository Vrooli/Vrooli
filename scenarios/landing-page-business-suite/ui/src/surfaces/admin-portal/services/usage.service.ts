import type { AdminUsageSummary, UsageRecord } from '../../../shared/api';
import { getAdminUsageSummary as apiGetAdminUsageSummary, formatCredits } from '../../../shared/api';

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
