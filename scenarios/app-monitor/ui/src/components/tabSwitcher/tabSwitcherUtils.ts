import type { CSSProperties } from 'react';
import type { BrowserTabRecord } from '@/state/browserTabsStore';
import type { App, Resource } from '@/types';

export const normalizeSearchValue = (value: string): string => value.trim().toLowerCase();

export const matchesSearch = (value: string | null | undefined, query: string): boolean => (
  Boolean(value && query && value.toLowerCase().includes(query))
);

export const matchesWebTabSearch = (tab: BrowserTabRecord, query: string): boolean => (
  !query || matchesSearch(tab.title, query) || matchesSearch(tab.url, query)
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

export const safeHostname = (value: string) => {
  try {
    return new URL(value).hostname;
  } catch (error) {
    return value;
  }
};

export const parseWebTabInput = (value: string): { url: string; title: string } | null => {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  const tryParse = (candidate: string) => {
    try {
      const normalized = new URL(candidate);
      return normalized.toString();
    } catch (error) {
      return null;
    }
  };

  const normalized = tryParse(trimmed)
    ?? (!trimmed.includes('://') ? tryParse(`https://${trimmed}`) : null);

  if (!normalized) {
    return null;
  }

  const hostname = safeHostname(normalized);
  return {
    url: normalized,
    title: hostname && hostname !== normalized ? hostname : normalized,
  };
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

export const getWebTabFallbackInitial = (tab: BrowserTabRecord): string | null => {
  const hostname = safeHostname(tab.url);
  if (hostname) {
    const alpha = hostname.replace(/[^a-zA-Z]/g, '');
    if (alpha) {
      return alpha.slice(0, 1).toUpperCase();
    }
    return hostname.slice(0, 1).toUpperCase();
  }
  const candidate = tab.title?.trim();
  return candidate && candidate.length > 0 ? candidate.slice(0, 1).toUpperCase() : null;
};
