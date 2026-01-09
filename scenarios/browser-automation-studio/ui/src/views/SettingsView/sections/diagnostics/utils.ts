/**
 * Diagnostics Tab Utilities
 *
 * Shared constants, types, and helper functions for the diagnostics tab.
 */

import { CheckCircle2, AlertTriangle, AlertCircle, Loader2 } from 'lucide-react';
import type { ComponentStatus } from '@/domains/observability';

/**
 * Status color key - includes both API status values and UI states.
 * API returns 'ok' | 'degraded' | 'error' for overall status,
 * and 'healthy' | 'degraded' | 'error' for component status.
 */
export type StatusColorKey = ComponentStatus | 'ok' | 'loading';

/**
 * Status indicator color and icon configuration.
 */
export const STATUS_COLORS: Record<StatusColorKey, { bg: string; text: string; icon: typeof CheckCircle2 }> = {
  ok: { bg: 'bg-emerald-500/20', text: 'text-emerald-400', icon: CheckCircle2 },
  healthy: { bg: 'bg-emerald-500/20', text: 'text-emerald-400', icon: CheckCircle2 },
  degraded: { bg: 'bg-amber-500/20', text: 'text-amber-400', icon: AlertTriangle },
  error: { bg: 'bg-red-500/20', text: 'text-red-400', icon: AlertCircle },
  loading: { bg: 'bg-gray-500/20', text: 'text-gray-400', icon: Loader2 },
};

/**
 * Format uptime duration in human-readable format.
 */
export function formatUptime(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${days}d ${hours % 24}h`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes % 60}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  }
  return `${seconds}s`;
}

/**
 * Format a timestamp as relative time (e.g., "5m ago").
 */
export function formatRelativeTime(dateString: string | undefined): string {
  if (!dateString) return 'Never';
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);

  if (diffSeconds < 60) return `${diffSeconds}s ago`;
  const diffMinutes = Math.floor(diffSeconds / 60);
  if (diffMinutes < 60) return `${diffMinutes}m ago`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;
  return date.toLocaleDateString();
}
