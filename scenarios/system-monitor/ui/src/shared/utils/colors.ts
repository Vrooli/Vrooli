/**
 * Shared color mapping utilities for the system-monitor UI.
 *
 * Extracted from metricHelpers, expansion components, and InfrastructureMonitor
 * to eliminate duplicated ternary chains.
 */

/** Map a risk level string to a CSS color variable. */
export const getRiskLevelColor = (level?: string): string => {
  switch (level) {
    case 'high':
      return 'var(--color-error)';
    case 'medium':
      return 'var(--color-warning)';
    default:
      return 'var(--color-success)';
  }
};

/** Return a CSS color variable based on healthy/unhealthy status. */
export const getHealthColor = (healthy: boolean): string =>
  healthy ? 'var(--color-success)' : 'var(--color-error)';

/** Map a leak-risk or service status string to a CSS color variable. */
export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'critical':
    case 'high':
    case 'unhealthy':
    case 'error':
      return 'var(--color-error)';
    case 'medium':
    case 'degraded':
    case 'warning':
      return 'var(--color-warning)';
    default:
      return 'var(--color-success)';
  }
};
