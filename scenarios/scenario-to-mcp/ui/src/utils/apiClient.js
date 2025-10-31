import axios from 'axios';

const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const API_SUFFIX = '/api/v1';
const LOOPBACK_HOST = '127.0.0.1';
const RAW_ENV_API_BASE = (import.meta.env.VITE_API_BASE_URL ?? '').toString().trim();
const RAW_ENV_API_PORT = (import.meta.env.VITE_API_PORT ?? '').toString().trim();
const FALLBACK_API_PORT = RAW_ENV_API_PORT || '18000';

const stripTrailingSlashes = (value = '') => value.replace(/\/+$/, '');
const stripLeadingSlashes = (value = '') => value.replace(/^\/+/, '');

const ensureApiSuffix = (base) => {
  const sanitized = stripTrailingSlashes(base || '');
  if (!sanitized) {
    return API_SUFFIX;
  }

  const lower = sanitized.toLowerCase();
  if (lower.endsWith(API_SUFFIX)) {
    return sanitized;
  }

  if (lower.endsWith('/api')) {
    return `${sanitized}${API_SUFFIX.slice(4)}`;
  }

  return `${sanitized}${API_SUFFIX}`;
};

const collectCandidates = (value, acc) => {
  if (!value) {
    return;
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (trimmed) {
      acc.push(trimmed);
    }
    return;
  }

  if (Array.isArray(value)) {
    value.forEach((item) => collectCandidates(item, acc));
    return;
  }

  if (typeof value !== 'object') {
    return;
  }

  const fields = ['apiBase', 'api', 'apiUrl', 'apiURL', 'base', 'baseUrl', 'baseURL', 'url', 'target', 'href', 'path', 'proxy'];
  fields.forEach((field) => collectCandidates(value[field], acc));

  if (typeof value.port === 'number' || (typeof value.port === 'string' && value.port.trim())) {
    const port = typeof value.port === 'number' ? value.port.toString() : value.port.trim();
    const host = typeof value.host === 'string' && value.host.trim() ? value.host.trim() : LOOPBACK_HOST;
    acc.push(`${host}:${port}`);
  }

  collectCandidates(value.endpoints, acc);
  collectCandidates(value.ports, acc);
};

const normalizeCandidate = (candidate) => {
  if (typeof candidate !== 'string') {
    return undefined;
  }

  const trimmed = candidate.trim();
  if (!trimmed) {
    return undefined;
  }

  if (/^https?:\/\//i.test(trimmed)) {
    return stripTrailingSlashes(trimmed);
  }

  if (trimmed.startsWith('//')) {
    const protocol = typeof window !== 'undefined' && window.location?.protocol ? window.location.protocol : 'https:';
    return stripTrailingSlashes(`${protocol}${trimmed}`);
  }

  if (trimmed.startsWith('/')) {
    if (typeof window !== 'undefined' && window.location?.origin) {
      return stripTrailingSlashes(`${window.location.origin}${trimmed}`);
    }
    return stripTrailingSlashes(trimmed);
  }

  if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)(:\d+)?/i.test(trimmed)) {
    return stripTrailingSlashes(`http://${trimmed}`);
  }

  if (/^\d{2,5}$/.test(trimmed)) {
    return stripTrailingSlashes(`http://${LOOPBACK_HOST}:${trimmed}`);
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    return stripTrailingSlashes(`${window.location.origin}/${trimmed.replace(/^\/+/, '')}`);
  }

  return undefined;
};

const resolveProxyBaseFromWindow = () => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const candidates = [];
  for (const key of PROXY_INFO_KEYS) {
    if (typeof window[key] !== 'undefined') {
      collectCandidates(window[key], candidates);
    }
  }

  for (const candidate of candidates) {
    const normalized = normalizeCandidate(candidate);
    if (normalized) {
      return normalized;
    }
  }

  return undefined;
};

const deriveProxyBaseFromLocation = () => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const { origin, pathname } = window.location || {};
  if (!origin || !pathname || !pathname.includes('/proxy')) {
    return undefined;
  }

  const segment = '/proxy';
  const index = pathname.indexOf(segment);
  if (index === -1) {
    return undefined;
  }

  return stripTrailingSlashes(`${origin}${pathname.slice(0, index + segment.length)}`);
};

const resolveApiBase = () => {
  if (RAW_ENV_API_BASE) {
    return ensureApiSuffix(RAW_ENV_API_BASE);
  }

  const proxyBase = resolveProxyBaseFromWindow();
  if (proxyBase) {
    return ensureApiSuffix(proxyBase);
  }

  const derivedProxy = deriveProxyBaseFromLocation();
  if (derivedProxy) {
    return ensureApiSuffix(derivedProxy);
  }

  if (typeof window === 'undefined') {
    return ensureApiSuffix(`http://${LOOPBACK_HOST}:${FALLBACK_API_PORT}`);
  }

  if (RAW_ENV_API_PORT) {
    return ensureApiSuffix(`http://${LOOPBACK_HOST}:${RAW_ENV_API_PORT}`);
  }

  return ensureApiSuffix('');
};

const API_BASE_URL = resolveApiBase();

const joinUrl = (base, path = '') => {
  const sanitizedBase = stripTrailingSlashes(base || '');
  const sanitizedPath = stripLeadingSlashes(path || '');

  if (!sanitizedPath) {
    return sanitizedBase || '/';
  }

  if (!sanitizedBase) {
    return `/${sanitizedPath}`;
  }

  return `${sanitizedBase}/${sanitizedPath}`;
};

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    Accept: 'application/json',
    'Content-Type': 'application/json'
  }
});

export const getApiBaseUrl = () => API_BASE_URL;

export const buildApiUrl = (path = '') => joinUrl(API_BASE_URL, path);
