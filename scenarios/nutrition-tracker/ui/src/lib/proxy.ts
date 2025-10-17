export interface EndpointBases {
  apiBase: string;
  n8nBase: string;
}

const DEFAULT_API_BASE = 'http://localhost:8081/api';
const DEFAULT_N8N_BASE = 'http://localhost:5678/webhook';

interface ProxyEntry {
  url?: string;
  target?: string;
  path?: string;
  label?: string;
  normalizedLabel?: string;
  slug?: string;
  aliases?: string[];
  priority?: number;
  port?: number;
}

type ProxyIndex = {
  aliasMap?: Map<string, ProxyEntry>;
};

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INDEX__?: ProxyIndex;
    __APP_MONITOR_PROXY_INFO__?: {
      primary?: ProxyEntry;
      ports?: ProxyEntry[];
    };
  }
}

export function resolveEndpointBases(): EndpointBases {
  if (typeof window === 'undefined') {
    return { apiBase: DEFAULT_API_BASE, n8nBase: DEFAULT_N8N_BASE };
  }

  const apiProxyBase =
    resolveProxyEntry(
      [
        'api',
        'api_port',
        'api-port',
        'api_url',
        'api-url',
        'backend',
        'service-api',
        'nutrition-tracker-api'
      ],
      'api'
    ) ?? undefined;

  const uiProxyBase =
    resolveProxyEntry(
      ['ui', 'ui_port', 'ui-port', 'frontend', 'app', 'primary', 'default'],
      'ui'
    ) ?? resolveUiBaseFromLocation();

  const apiBase = apiProxyBase ? joinUrl(apiProxyBase, '/api') : DEFAULT_API_BASE;
  const n8nBase = uiProxyBase ? joinUrl(uiProxyBase, '/api') : DEFAULT_N8N_BASE;

  if (!apiProxyBase && window.location?.hostname && !window.location.hostname.includes('localhost')) {
    console.warn('[NutritionTracker] Falling back to localhost API base; proxy metadata unavailable.');
  }

  return { apiBase, n8nBase };
}

function resolveUiBaseFromLocation(): string | undefined {
  try {
    const baseUrl = new URL('.', window.location.href);
    return stripTrailingSlash(baseUrl.toString());
  } catch (error) {
    console.warn('[NutritionTracker] Unable to derive UI base from location', error);
    return undefined;
  }
}

function resolveProxyEntry(preferredAliases: string[] = [], fallbackKeyword?: string): string | undefined {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const normalizedPreferred = Array.from(
    new Set(
      preferredAliases
        .map((alias) => (typeof alias === 'string' ? alias.trim().toLowerCase() : ''))
        .filter(Boolean)
    )
  );

  const index = window.__APP_MONITOR_PROXY_INDEX__;
  if (index?.aliasMap instanceof Map) {
    for (const alias of normalizedPreferred) {
      const entry = index.aliasMap.get(alias);
      const base = buildEntryBaseUrl(entry);
      if (base) {
        return base;
      }
    }
  }

  const proxyEntries = gatherProxyEntries();
  for (const entry of proxyEntries) {
    const aliases = Array.isArray(entry?.aliases) ? entry.aliases : [];
    for (const alias of aliases) {
      const normalized = typeof alias === 'string' ? alias.trim().toLowerCase() : '';
      if (normalized && normalizedPreferred.includes(normalized)) {
        const base = buildEntryBaseUrl(entry);
        if (base) {
          return base;
        }
      }
    }
  }

  if (fallbackKeyword) {
    const keyword = fallbackKeyword.toLowerCase();
    const fallbackEntry = proxyEntries
      .filter((entry) => entryMatchesKeyword(entry, keyword))
      .sort((a, b) => (Number(b?.priority) || 0) - (Number(a?.priority) || 0))
      .find((entry) => Boolean(buildEntryBaseUrl(entry)));

    if (fallbackEntry) {
      return buildEntryBaseUrl(fallbackEntry);
    }
  }

  return undefined;
}

function gatherProxyEntries(): ProxyEntry[] {
  if (typeof window === 'undefined') {
    return [];
  }

  const entries: ProxyEntry[] = [];
  const seen = new Set<string>();

  const pushEntry = (entry?: ProxyEntry) => {
    if (!entry || typeof entry !== 'object') {
      return;
    }
    const key = entry.path ?? `${entry.port ?? ''}:${entry.slug ?? ''}`;
    if (key && !seen.has(key)) {
      seen.add(key);
      entries.push(entry);
    }
  };

  const info = window.__APP_MONITOR_PROXY_INFO__;
  if (info?.primary) {
    pushEntry(info.primary);
  }
  if (Array.isArray(info?.ports)) {
    info.ports.forEach((entry) => pushEntry(entry));
  }

  const index = window.__APP_MONITOR_PROXY_INDEX__;
  if (index?.aliasMap instanceof Map) {
    index.aliasMap.forEach((entry) => pushEntry(entry));
  }

  return entries;
}

function entryMatchesKeyword(entry: ProxyEntry | undefined, keyword: string): boolean {
  if (!entry || !keyword) {
    return false;
  }

  const haystack: string[] = [];
  if (typeof entry.label === 'string') {
    haystack.push(entry.label);
  }
  if (typeof entry.normalizedLabel === 'string') {
    haystack.push(entry.normalizedLabel);
  }
  if (typeof entry.slug === 'string') {
    haystack.push(entry.slug);
  }
  if (Array.isArray(entry.aliases)) {
    haystack.push(...entry.aliases);
  }
  if (typeof entry.port === 'number') {
    haystack.push(String(entry.port));
  }

  return haystack.some((value) => value?.toLowerCase().includes(keyword));
}

function buildEntryBaseUrl(entry?: ProxyEntry): string | undefined {
  if (!entry) {
    return undefined;
  }

  if (typeof entry.url === 'string' && entry.url.trim()) {
    return stripTrailingSlash(entry.url.trim());
  }
  if (typeof entry.target === 'string' && entry.target.trim()) {
    return stripTrailingSlash(entry.target.trim());
  }
  if (typeof entry.path === 'string' && entry.path.trim()) {
    return toAbsoluteProxyUrl(entry.path.trim());
  }
  return undefined;
}

function toAbsoluteProxyUrl(path: string): string | undefined {
  try {
    const normalized = path.startsWith('/') ? path : `/${path}`;
    return stripTrailingSlash(new URL(normalized, window.location.origin).toString());
  } catch (error) {
    console.warn('[NutritionTracker] Unable to normalize proxy path', path, error);
    return undefined;
  }
}

export function joinUrl(base: string, segment: string): string {
  const normalizedBase = stripTrailingSlash(base);
  const normalizedSegment = segment?.startsWith('/') ? segment : `/${segment}`;
  return `${normalizedBase}${normalizedSegment}`;
}

export function stripTrailingSlash(value?: string): string {
  if (typeof value !== 'string' || value.length === 0) {
    return '';
  }
  if (/^https?:\/\/$/i.test(value)) {
    return value;
  }
  return value.replace(/\/+$/, '');
}
