const API_BASE_PATH = '/api/v1';

type ProxyEndpoint = {
  path?: string;
  url?: string;
  target?: string;
};

type AppMonitorProxyInfo = {
  primary?: ProxyEndpoint;
  endpoints?: ProxyEndpoint[];
};

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo;
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo;
  }
}

const stripTrailingSlash = (value: string): string => {
  let result = value;
  while (result.endsWith('/') && !/^https?:\/\/$/iu.test(result)) {
    result = result.slice(0, -1);
  }
  return result;
};

const ensureLeadingSlash = (value: string): string => (value.startsWith('/') ? value : `/${value}`);

const joinUrlSegments = (base: string, segment: string): string => {
  const normalizedBase = stripTrailingSlash(base);
  const normalizedSegment = ensureLeadingSlash(segment);
  if (!normalizedBase) {
    return normalizedSegment;
  }
  return `${normalizedBase}${normalizedSegment}`;
};

const ensureApiSuffix = (base: string): string => {
  const normalized = stripTrailingSlash(base);
  if (normalized.toLowerCase().endsWith(API_BASE_PATH)) {
    return normalized;
  }
  return stripTrailingSlash(joinUrlSegments(normalized, API_BASE_PATH));
};

const pickProxyCandidate = (info?: AppMonitorProxyInfo): string | undefined => {
  if (!info) {
    return undefined;
  }

  const inspectEndpoint = (endpoint?: ProxyEndpoint): string | undefined => {
    if (!endpoint) {
      return undefined;
    }
    return endpoint.url ?? endpoint.path ?? endpoint.target;
  };

  const primaryCandidate = inspectEndpoint(info.primary);
  if (primaryCandidate) {
    return primaryCandidate;
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      const candidate = inspectEndpoint(endpoint);
      if (candidate) {
        return candidate;
      }
    }
  }

  return undefined;
};

const resolveProxyApiBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const globalWindow = window as Window & {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo;
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo;
  };

  const proxyInfo =
    globalWindow.__APP_MONITOR_PROXY_INFO__ ?? globalWindow.__APP_MONITOR_PROXY_INDEX__;

  const candidate = pickProxyCandidate(proxyInfo);
  if (!candidate) {
    return undefined;
  }

  if (/^https?:\/\//iu.test(candidate)) {
    return ensureApiSuffix(candidate);
  }

  if (candidate.startsWith('//')) {
    const protocol = typeof window.location?.protocol === 'string' ? window.location.protocol : 'http:';
    return ensureApiSuffix(`${protocol}${candidate}`);
  }

  const base = typeof window.location?.origin === 'string' ? window.location.origin : '';
  return ensureApiSuffix(joinUrlSegments(base, candidate));
};

const resolveExplicitApiBase = (): string | undefined => {
  const metaEnv = (import.meta as unknown as { env?: Record<string, unknown> }).env ?? {};
  const envUrl = metaEnv.VITE_API_URL ?? metaEnv.API_URL;
  if (typeof envUrl === 'string' && envUrl.trim().length > 0) {
    return ensureApiSuffix(envUrl.trim());
  }

  if (typeof __API_URL__ !== 'undefined' && __API_URL__) {
    return ensureApiSuffix(__API_URL__);
  }

  return undefined;
};

const resolveFallbackProtocol = (): 'http' | 'https' => {
  if (typeof window !== 'undefined' && window.location?.protocol === 'https:') {
    return 'https';
  }

  const metaEnv = (import.meta as unknown as { env?: Record<string, unknown> }).env ?? {};
  const envProtocol = metaEnv.VITE_API_PROTOCOL ?? metaEnv.API_PROTOCOL;
  if (typeof envProtocol === 'string' && envProtocol.toLowerCase() === 'https') {
    return 'https';
  }

  return 'http';
};

const resolveFallbackHost = (): string => {
  const metaEnv = (import.meta as unknown as { env?: Record<string, unknown> }).env ?? {};
  const envHost = metaEnv.VITE_API_HOST ?? metaEnv.API_HOST;
  if (typeof envHost === 'string' && envHost.trim().length > 0) {
    return envHost.trim();
  }

  if (typeof window !== 'undefined' && window.location?.hostname) {
    return window.location.hostname;
  }

  return '127.0.0.1';
};

const resolveFallbackPort = (): string => {
  const metaEnv = (import.meta as unknown as { env?: Record<string, unknown> }).env ?? {};
  const envPort = metaEnv.VITE_API_PORT ?? metaEnv.API_PORT;
  if (typeof envPort === 'string' && envPort.trim().length > 0) {
    return envPort.trim();
  }

  if (typeof window !== 'undefined' && window.location?.port) {
    const parsed = Number.parseInt(window.location.port, 10);
    if (Number.isFinite(parsed) && parsed > 0) {
      if (parsed >= 36000 && parsed <= 39999) {
        return String(parsed - 17700);
      }
      if (parsed >= 3000 && parsed < 4000) {
        return String(parsed + 15000);
      }
    }
  }

  return '30400';
};

const resolveFallbackApiBase = (): string => {
  const protocol = resolveFallbackProtocol();
  const host = resolveFallbackHost();
  const port = resolveFallbackPort();
  const authority = port ? `${host}:${port}` : host;
  return ensureApiSuffix(`${protocol}://${authority}`);
};

const runtimeApiUrl = ((): string => {
  const proxyBase = resolveProxyApiBase();
  if (proxyBase) {
    return proxyBase;
  }

  const explicitBase = resolveExplicitApiBase();
  if (explicitBase) {
    return explicitBase;
  }

  return resolveFallbackApiBase();
})();

export const API_BASE_URL = runtimeApiUrl;

export const apiFetch = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
    ...init,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed: ${response.status}`);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
};

declare const __API_URL__: string | undefined;

declare global {
  interface ImportMetaEnv {
    readonly VITE_API_PORT?: string;
    readonly VITE_API_URL?: string;
    readonly VITE_API_HOST?: string;
    readonly VITE_API_PROTOCOL?: string;
    readonly API_PORT?: string;
    readonly API_URL?: string;
    readonly API_HOST?: string;
    readonly API_PROTOCOL?: string;
  }
}
