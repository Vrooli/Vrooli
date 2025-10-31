import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function formatDate(value?: string | number | Date | null): string {
  if (!value) {
    return '—';
  }
  try {
    const date = value instanceof Date ? value : new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '—';
    }
    return date.toLocaleString();
  } catch (error) {
    console.warn('[data-structurer] Failed to format date', error);
    return '—';
  }
}

export function safeJsonParse<T>(value: string): T | undefined {
  try {
    return JSON.parse(value) as T;
  } catch (error) {
    console.warn('[data-structurer] Failed to parse JSON', error);
    return undefined;
  }
}

export function stringify(value: unknown): string {
  try {
    if (typeof value === 'string') {
      return value;
    }
    return JSON.stringify(value, null, 2);
  } catch (error) {
    console.warn('[data-structurer] Failed to stringify value', error);
    return '';
  }
}

export function computeConfidenceVariant(score?: number | null): 'low' | 'medium' | 'high' {
  if (typeof score !== 'number') {
    return 'low';
  }
  if (score >= 0.75) {
    return 'high';
  }
  if (score >= 0.45) {
    return 'medium';
  }
  return 'low';
}

export function percent(score?: number | null, fallback = '—'): string {
  if (typeof score !== 'number' || Number.isNaN(score)) {
    return fallback;
  }
  return `${(score * 100).toFixed(1)}%`;
}
