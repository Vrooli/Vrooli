import type { BootEntry } from '../types';

export type ShutdownClass = 'clean' | 'unclean' | 'in-progress' | 'unknown';

/**
 * Classify a boot entry's shutdown state for UI tinting.
 *
 * The current boot (index 0) is always `in-progress` per backend convention.
 * `clean` boots have a recognized shutdown marker; `unclean` boots do not
 * and have a `reason` set. Anything else (no reason, no clean flag) is
 * `unknown` — we don't surface false alarms.
 */
export function classifyShutdown(boot: BootEntry): ShutdownClass {
  if (boot.index === 0) return 'in-progress';
  if (boot.clean) return 'clean';
  if (boot.reason && boot.reason.length > 0) return 'unclean';
  return 'unknown';
}

/** Human-friendly label for a shutdown class. */
export function shutdownLabel(c: ShutdownClass): string {
  switch (c) {
    case 'clean':
      return 'Clean shutdown';
    case 'unclean':
      return 'Unclean shutdown';
    case 'in-progress':
      return 'Current boot';
    case 'unknown':
      return 'Unknown';
  }
}
