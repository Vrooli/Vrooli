import type { CSSProperties } from 'react';
import type { App, Resource } from '@/types';

export const normalizeSearchValue = (value: string): string => value.trim().toLowerCase();

export const matchesSearch = (value: string | null | undefined, query: string): boolean => (
  Boolean(value && query && value.toLowerCase().includes(query))
);

export const matchesResourceSearch = (resource: Resource, query: string): boolean => {
  if (!query) {
    return true;
  }
  const haystacks = [
    resource.name,
    resource.type,
    resource.description,
    resource.status,
    resource.id,
  ];
  return haystacks.some(entry => matchesSearch(entry, query));
};

export const formatViewCount = (value?: number | string | null): string | null => {
  if (value == null) {
    return null;
  }
  const numeric = typeof value === 'number' ? value : Number(value);
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return null;
  }
  if (numeric >= 1000) {
    return `${(numeric / 1000).toFixed(1)}k`;
  }
  return String(Math.round(numeric));
};

export const buildThumbStyle = (dataUrl?: string | null): CSSProperties | undefined => (
  dataUrl
    ? {
        backgroundImage: `url(${dataUrl})`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
        backgroundRepeat: 'no-repeat',
      }
    : undefined
);

export const getFallbackInitial = (value?: string | null): string => {
  const base = (value ?? '').trim();
  if (!base) {
    return '?';
  }
  const alphanumeric = base.replace(/[^a-zA-Z0-9]/g, '');
  const candidate = alphanumeric.charAt(0) || base.charAt(0);
  return candidate.toUpperCase();
};

export const getAppDisplayName = (app: App): string => app.scenario_name ?? app.name ?? app.id ?? 'Untitled scenario';

export const getStatusClassName = (status?: string | null): string => {
  const source = (status ?? 'unknown').toLowerCase();
  const normalized = source.replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
  return `status-${normalized || 'unknown'}`;
};

export const getCompletenessLevel = (score?: number | null): 'none' | 'critical' | 'low' | 'medium' | 'high' => {
  if (typeof score !== 'number' || score < 0) {
    return 'none';
  }
  if (score >= 81) return 'high';     // 81-100: nearly_ready, production_ready
  if (score >= 61) return 'medium';   // 61-80: mostly_complete
  if (score >= 41) return 'low';      // 41-60: functional_incomplete
  return 'critical';                  // 0-40: early_stage, foundation_laid
};
