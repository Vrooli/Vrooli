import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Formats a phase name (e.g., "progress-mode" -> "Progress Mode")
 */
export function formatPhaseName(name: string): string {
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function normalizeSteerMode(mode?: string): string {
  return (mode ?? '').trim().toLowerCase();
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
