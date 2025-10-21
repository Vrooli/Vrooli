export interface EndpointBases {
  apiBase: string;
  n8nBase: string;
}

const LOOPBACK_HOST = '127.0.0.1';
const API_PORT = 8081;
const N8N_PORT = 5678;
const API_SUFFIX = '/api';
const N8N_SUFFIX = '/webhook';
const LOCAL_HOST_PATTERN = /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i;

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
  const loopbackApiBase = buildLoopbackBase(API_PORT, API_SUFFIX);
  const loopbackN8nBase = buildLoopbackBase(N8N_PORT, N8N_SUFFIX);

  if (typeof window === 'undefined') {
    return { apiBase: loopbackApiBase, n8nBase: loopbackN8nBase };
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

  const n8nProxyBase =
    resolveProxyEntry(
      [
        'n8n',
        'automation',
        'workflows',
        'workflow',
        'webhook',
        'hooks',
        'nutrition-tracker-n8n'
      ],
      'n8n'
    ) ?? undefined;

  const uiProxyBase =
    resolveProxyEntry(
      ['ui', 'ui_port', 'ui-port', 'frontend', 'app', 'primary', 'default'],
      'ui'
    ) ?? undefined;

  const locationBase = resolveUiBaseFromLocation();

  const apiBaseSource = apiProxyBase ?? locationBase;
  const apiBase = apiBaseSource ? joinUrl(apiBaseSource, API_SUFFIX) : loopbackApiBase;

  const n8nBaseSource = n8nProxyBase ?? uiProxyBase ?? locationBase;
  const n8nSuffix = n8nProxyBase ? N8N_SUFFIX : API_SUFFIX;
  const n8nBase = n8nBaseSource ? joinUrl(n8nBaseSource, n8nSuffix) : loopbackN8nBase;

  const isRemoteHost = isRemoteHostname(window.location?.hostname);
  if (!apiProxyBase && !locationBase && isRemoteHost) {
    console.warn('[NutritionTracker] Falling back to loopback API base; proxy metadata unavailable.');
  }
  if (!n8nProxyBase && !uiProxyBase && !locationBase && isRemoteHost) {
    console.warn('[NutritionTracker] Falling back to loopback automation base; proxy metadata unavailable.');
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

function buildLoopbackBase(port: number, suffix: string): string {
  const normalizedSuffix = suffix.startsWith('/') ? suffix : `/${suffix}`;
  return stripTrailingSlash(`http://${LOOPBACK_HOST}:${port}${normalizedSuffix}`);
}

function isRemoteHostname(hostname?: string | null): boolean {
  if (!hostname) {
    return false;
  }

  return !LOCAL_HOST_PATTERN.test(hostname);
}
