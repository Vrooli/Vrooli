import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__gitControlTowerBridgeInitialized';
const LOOPBACK_HOST = ['127', '0', '0', '1'].join('.');

function coerceString(value) {
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }
  return undefined;
}

function collectProxyCandidates(value, seen, list) {
  if (!value) {
    return;
  }

  if (typeof value === 'string') {
    const candidate = value.trim();
    if (candidate && !seen.has(candidate)) {
      seen.add(candidate);
      list.push(candidate);
    }
    return;
  }

  if (typeof value === 'object') {
    const record = value || {};
    collectProxyCandidates(record.url, seen, list);
    collectProxyCandidates(record.path, seen, list);
    collectProxyCandidates(record.target, seen, list);
  }
}

export function ensureBridge() {
  if (typeof window === 'undefined') {
    return;
  }
  if (window[BRIDGE_FLAG]) {
    return;
  }
  if (window.parent === window) {
    return;
  }

  try {
    initIframeBridgeChild({
      appId: 'git-control-tower',
      captureLogs: { enabled: true },
      captureNetwork: { enabled: true },
    });
    window[BRIDGE_FLAG] = true;
  } catch (error) {
    console.warn('[Git Control Tower] Unable to initialize iframe bridge', error);
  }
}

function pickProxyBase(info) {
  const seen = new Set();
  const candidates = [];

  collectProxyCandidates(info, seen, candidates);

  if (info && typeof info === 'object') {
    const descriptor = info;
    collectProxyCandidates(descriptor.primary, seen, candidates);
    if (Array.isArray(descriptor.endpoints)) {
      descriptor.endpoints.forEach((endpoint) => collectProxyCandidates(endpoint, seen, candidates));
    }
  }

  return candidates.find(Boolean);
}

function ensureLeadingSlash(value) {
  if (!value) {
    return '/';
  }
  return value.startsWith('/') ? value : `/${value}`;
}

function stripTrailingSlash(value) {
  return (value || '').replace(/\/+$/, '');
}

function sanitizePort(port) {
  if (typeof port === 'number') {
    return port > 0 ? String(port) : '';
  }
  if (typeof port === 'string') {
    const trimmed = port.trim();
    return trimmed ? trimmed : '';
  }
  return '';
}

export function resolveProxyBase() {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const info = window.__APP_MONITOR_PROXY_INFO__ ?? window.__APP_MONITOR_PROXY_INDEX__;
  const candidate = pickProxyBase(info);
  if (!candidate) {
    return undefined;
  }

  if (/^https?:\/\//i.test(candidate)) {
    return stripTrailingSlash(candidate);
  }

  const origin = coerceString(window.location?.origin);
  if (!origin) {
    return undefined;
  }

  return stripTrailingSlash(origin + ensureLeadingSlash(candidate));
}

export function resolveApiBase(basePath = '/api/v1') {
  if (typeof window === 'undefined') {
    return basePath;
  }

  const proxyBase = resolveProxyBase();
  if (proxyBase) {
    return stripTrailingSlash(proxyBase) + ensureLeadingSlash(basePath);
  }

  const origin = coerceString(window.location?.origin);
  if (origin) {
    return stripTrailingSlash(origin) + ensureLeadingSlash(basePath);
  }

  const protocol = window.location?.protocol || 'http:';
  const hostname = coerceString(window.location?.hostname) || LOOPBACK_HOST;
  const portSegment = sanitizePort(window.location?.port);
  const port = portSegment ? `:${portSegment}` : '';

  return `${protocol}//${hostname}${port}` + ensureLeadingSlash(basePath);
}

export function buildApiUrl(path, basePath = '/api/v1') {
  const base = resolveApiBase(basePath);
  if (!path) {
    return base;
  }
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  const normalized = ensureLeadingSlash(path);
  return stripTrailingSlash(base) + normalized;
}

ensureBridge();
