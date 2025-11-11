import { resolveApiBase } from "@vrooli/api-base";
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function prettyNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1
  }).format(value);
}

export function getApiBaseUrl(options?: { appendSuffix?: boolean }): string {
  return resolveApiBase({ appendSuffix: options?.appendSuffix ?? true });
}
