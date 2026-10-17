/**
 * Map syslog priorities (0=emerg .. 7=debug) to short labels and a
 * tonal palette for the badge UI. -1 means "unknown" (parser fallback).
 */
export const priorityLabels: Record<number, string> = {
  0: 'EMERG',
  1: 'ALERT',
  2: 'CRIT',
  3: 'ERR',
  4: 'WARN',
  5: 'NOTICE',
  6: 'INFO',
  7: 'DEBUG',
};

export type PriorityTone = 'critical' | 'error' | 'warning' | 'notice' | 'info' | 'debug' | 'unknown';

export function priorityLabel(priority: number): string {
  return priorityLabels[priority] ?? 'UNK';
}

export function priorityTone(priority: number): PriorityTone {
  if (priority >= 0 && priority <= 2) return 'critical';
  if (priority === 3) return 'error';
  if (priority === 4) return 'warning';
  if (priority === 5) return 'notice';
  if (priority === 6) return 'info';
  if (priority === 7) return 'debug';
  return 'unknown';
}

export function priorityColor(priority: number): string {
  switch (priorityTone(priority)) {
    case 'critical':
    case 'error':
      return 'var(--color-error)';
    case 'warning':
      return 'var(--color-warning)';
    case 'notice':
    case 'info':
      return 'var(--color-info)';
    case 'debug':
      return 'var(--color-muted)';
    case 'unknown':
      return 'var(--color-muted)';
  }
}
