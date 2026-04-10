import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// ─────────────────────────────────────────────────────────────────────────────
// Date Formatting Utilities
// ─────────────────────────────────────────────────────────────────────────────
// Centralized date formatting to ensure consistent presentation across the UI.

export interface FormatDateOptions {
  /** Include time in the output */
  includeTime?: boolean;
  /** Use relative format ("Updated 2 days ago") when appropriate */
  relative?: boolean;
}

/**
 * Formats a date string for display.
 * @param dateString - ISO date string to format
 * @param options - Formatting options
 * @returns Formatted date string
 */
export function formatDate(dateString: string, options: FormatDateOptions = {}): string {
  const { includeTime = false } = options;

  const date = new Date(dateString);

  if (includeTime) {
    return date.toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit"
    });
  }

  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric"
  });
}

/**
 * Formats a date as a relative time string (e.g., "2 days ago").
 * Falls back to absolute date for times > 30 days ago.
 */
export function formatRelativeDate(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffDays === 0) {
    return "today";
  } else if (diffDays === 1) {
    return "yesterday";
  } else if (diffDays < 30) {
    return `${diffDays} days ago`;
  }

  return formatDate(dateString);
}
