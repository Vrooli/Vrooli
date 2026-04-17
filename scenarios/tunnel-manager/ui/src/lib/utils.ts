import { useCallback, useMemo, useState } from "react";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export type SortDir = "asc" | "desc";

export const MOBILE_PAGE_SIZE = 20;
export const DESKTOP_PAGE_SIZE = 25;

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format a timestamp as a human-friendly relative time string.
 * Falls back to localized date string for times older than 24 hours.
 */
export function timeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = Date.now();
  const diffMs = now - date.getTime();

  if (diffMs < 0) return "just now";

  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 60) return "just now";

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;

  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;

  return date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

export function useSort<T, F extends string>(
  data: T[] | undefined,
  defaultField: F,
  compareFn: (a: T, b: T, field: F) => number,
  defaultDir: SortDir = "asc",
) {
  const [sortField, setSortField] = useState<F>(defaultField);
  const [sortDir, setSortDir] = useState<SortDir>(defaultDir);

  const sorted = useMemo(() => {
    if (!data) return [];
    return [...data].sort((a, b) => {
      const cmp = compareFn(a, b, sortField);
      return sortDir === "desc" ? -cmp : cmp;
    });
  }, [data, sortField, sortDir, compareFn]);

  const toggleSort = useCallback((field: F) => {
    setSortField((prev) => {
      if (prev === field) {
        setSortDir((dir) => (dir === "asc" ? "desc" : "asc"));
        return prev;
      }
      setSortDir("asc");
      return field;
    });
  }, []);

  return { sorted, sortField, sortDir, toggleSort };
}

type StatusVariant = "success" | "error" | "warning" | "neutral" | "info";

export function statusToVariant(status: string): StatusVariant {
  switch (status) {
    case "healthy":
    case "up":
    case "compliant":
    case "success":
    case "enabled":
      return "success";
    case "unhealthy":
    case "down":
    case "error":
    case "mismatch":
    case "failure":
      return "error";
    case "degraded":
    case "timeout":
    case "skipped":
    case "missing_scenario":
    case "missing_port":
      return "warning";
    default:
      return "neutral";
  }
}

export type { StatusVariant };
