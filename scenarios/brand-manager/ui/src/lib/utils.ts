import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format an ISO date string as a short locale date (e.g. "3/26/2026").
 * Centralises the repeated `new Date(x).toLocaleDateString()` calls.
 */
export function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString();
}

/**
 * Format an ISO date string as a full locale date-time (e.g. "3/26/2026, 4:18:27 AM").
 * Centralises the repeated `new Date(x).toLocaleString()` calls.
 */
export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString();
}

/**
 * Format a contrast ratio to one decimal place with the ":1" suffix.
 * E.g. `formatContrastRatio(4.523)` → `"4.5:1"`.
 */
export function formatContrastRatio(ratio: number): string {
  return `${ratio.toFixed(1)}:1`;
}
