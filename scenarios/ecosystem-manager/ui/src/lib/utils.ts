import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

const BUILT_IN_PHASE_LABELS: Record<string, string> = {
  progress: 'Progress',
  ux: 'UX',
  refactor: 'Refactor',
  test: 'Test',
  explore: 'Explore',
  polish: 'Polish',
  performance: 'Performance',
  security: 'Security',
};

/**
 * Formats a phase name (e.g., "progress-mode" -> "Progress Mode")
 */
export function formatPhaseName(name: string): string {
  return name
    .split(/[-_]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function normalizeSteerMode(mode?: string): string {
  return (mode ?? '').trim().toLowerCase();
}

export function getPhaseDisplayName(
  phaseId?: string,
  phases: Array<{ id: string; name: string }> = []
): string | undefined {
  if (!phaseId) return undefined;
  const normalized = normalizeSteerMode(phaseId);
  const match = phases.find((phase) => normalizeSteerMode(phase.id) === normalized);
  return match?.name ?? BUILT_IN_PHASE_LABELS[normalized] ?? formatPhaseName(phaseId);
}

export function getApiErrorMessage(error: unknown): string {
  if (!error) return 'Unknown error';

  if (error instanceof Error) {
    try {
      const match = error.message.match(/API Error \(\d+\): (.+)/);
      if (match) {
        const parsed = JSON.parse(match[1]);
        if (typeof parsed === 'string') {
          return parsed;
        }
        if (parsed && typeof parsed === 'object') {
          return parsed.message || parsed.error || error.message;
        }
      }
    } catch {
      // Fall through to raw message
    }
    return error.message;
  }

  if (typeof error === 'object' && error !== null) {
    const maybeError = error as { message?: string; error?: string };
    return maybeError.message || maybeError.error || 'Unknown error';
  }

  return String(error);
}
