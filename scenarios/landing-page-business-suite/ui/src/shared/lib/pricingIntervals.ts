import type { PlanOption } from '../api';

export type IntervalSlug = 'month' | 'year' | 'one_time' | 'other';

export const normalizeInterval = (value: PlanOption['billing_interval'] | string | number | null | undefined): IntervalSlug => {
  if (typeof value === 'number') {
    if (value === 1) return 'month';
    if (value === 2) return 'year';
    if (value === 3) return 'one_time';
  }
  const raw = String(value ?? '').toLowerCase();
  if (raw.includes('month')) return 'month';
  if (raw.includes('year')) return 'year';
  if (raw.includes('one_time') || raw.includes('one-time') || raw.includes('onetime')) return 'one_time';
  return 'other';
};

export const intervalLabel = (slug: IntervalSlug) => {
  switch (slug) {
    case 'month':
      return 'Monthly';
    case 'year':
      return 'Yearly';
    case 'one_time':
      return 'One-time';
    default:
      return 'Other';
  }
};
