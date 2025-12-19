import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";
import { resolveApiBase } from "@vrooli/api-base";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getApiBaseUrl(): string {
  // Use api-base resolution which handles localhost/proxy scenarios correctly
  // The UI server proxies /api/* to the actual API server
  return resolveApiBase({ appendSuffix: true });
}

export function formatDate(date: string | Date | undefined): string {
  if (!date) return "N/A";
  const d = typeof date === "string" ? new Date(date) : date;
  return d.toLocaleString();
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms + "ms";
  if (ms < 60000) return (ms / 1000).toFixed(1) + "s";
  if (ms < 3600000) return Math.floor(ms / 60000) + "m " + Math.floor((ms % 60000) / 1000) + "s";
  return Math.floor(ms / 3600000) + "h " + Math.floor((ms % 3600000) / 60000) + "m";
}

export function formatRelativeTime(date: string | Date | undefined): string {
  if (!date) return "N/A";
  const d = typeof date === "string" ? new Date(date) : date;
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  
  if (diffMs < 60000) return "just now";
  if (diffMs < 3600000) return Math.floor(diffMs / 60000) + "m ago";
  if (diffMs < 86400000) return Math.floor(diffMs / 3600000) + "h ago";
  return Math.floor(diffMs / 86400000) + "d ago";
}

export function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 3) + "...";
}
