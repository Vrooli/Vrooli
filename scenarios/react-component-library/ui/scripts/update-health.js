#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { performance } = require('perf_hooks');

const WINDOW_PROXY_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const ENV_PROXY_KEYS = ['APP_MONITOR_PROXY_INFO', 'APP_MONITOR_PROXY_INDEX'];
const OUTPUT_FILES = ['health', 'health.json'];

const controller = new AbortController();
const timeout = setTimeout(() => controller.abort(), 3000);

const toTrimmedString = value => {
  if (typeof value !== 'string') {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
};

const joinUrl = (base, segment) => {
  if (!base) {
    return segment;
  }
  const normalizedBase = base.replace(/\/+$/, '');
  const normalizedSegment = segment.replace(/^\/+/, '');
  return `${normalizedBase}/${normalizedSegment}`;
};

const safeParseJson = (payload, source) => {
  if (!payload) {
    return undefined;
  }
  try {
    return JSON.parse(payload);
  } catch (error) {
    console.warn(`[ReactComponentLibrary] Failed to parse proxy metadata from ${source}`, error);
    return undefined;
  }
};

const collectProxyCandidates = (input, seen = new Set()) => {
  if (!input) {
    return [];
  }

  const queue = [input];
  const candidates = [];

  while (queue.length > 0) {
    const value = queue.shift();
    if (!value) {
      continue;
    }

    if (typeof value === 'string') {
      const trimmed = toTrimmedString(value);
      if (trimmed && !seen.has(trimmed)) {
        seen.add(trimmed);
        candidates.push(trimmed);
      }
      continue;
    }

    if (Array.isArray(value)) {
      queue.push(...value);
      continue;
    }

    if (typeof value === 'object') {
      const record = value;
      const possibleKeys = ['url', 'target', 'href', 'origin', 'path', 'basePath'];
      for (const key of possibleKeys) {
        if (key in record) {
          queue.push(record[key]);
        }
      }
      if (record.primary) {
        queue.push(record.primary);
      }
      if (Array.isArray(record.endpoints)) {
        queue.push(...record.endpoints);
      }
    }
  }

  return candidates;
};

const toAbsoluteUrl = candidate => {
  const trimmed = toTrimmedString(candidate);
  if (!trimmed) {
    return undefined;
  }

  if (/^https?:\/\//i.test(trimmed)) {
    return trimmed.replace(/\/+$/, '');
  }

  const proxyOrigin = toTrimmedString(process.env.APP_MONITOR_PROXY_ORIGIN);
  if (proxyOrigin && trimmed.startsWith('/')) {
    return joinUrl(proxyOrigin, trimmed);
  }

  if (typeof window !== 'undefined') {
    const origin = toTrimmedString(window.location?.origin);
    if (origin && trimmed.startsWith('/')) {
      return joinUrl(origin, trimmed);
    }
  }

  return undefined;
};

const resolveProxyInfo = () => {
  if (typeof window !== 'undefined') {
    for (const key of WINDOW_PROXY_KEYS) {
      try {
        const value = window[key];
        if (value) {
          return value;
        }
      } catch (error) {
        console.warn('[ReactComponentLibrary] Unable to read proxy metadata from window', error);
      }
    }
  }

  for (const key of ENV_PROXY_KEYS) {
    const parsed = safeParseJson(process.env[key], `process.env.${key}`);
    if (parsed) {
      return parsed;
    }
  }

  return undefined;
};

const resolveProxyBase = info => {
  const candidates = collectProxyCandidates(info);
  for (const candidate of candidates) {
    const absolute = toAbsoluteUrl(candidate);
    if (absolute) {
      return absolute;
    }
  }
  return undefined;
};

const resolveApiOrigin = () => {
  const explicit = toTrimmedString(process.env.API_URL);
  if (explicit) {
    return explicit;
  }

  const proxyInfo = resolveProxyInfo();
  const proxyBase = resolveProxyBase(proxyInfo);
  if (proxyBase) {
    return proxyBase;
  }

  const port = Number.parseInt(process.env.API_PORT ?? process.env.DEFAULT_API_PORT ?? '15092', 10);
  const safePort = Number.isFinite(port) && port > 0 ? port : 15092;
  const protocol = toTrimmedString(process.env.API_PROTOCOL)?.toLowerCase() === 'https' ? 'https' : 'http';
  return `${protocol}://localhost:${safePort}`;
};

const API_ORIGIN = resolveApiOrigin();
const HEALTH_URL = joinUrl(API_ORIGIN, '/health');
const outputPaths = OUTPUT_FILES.map(file => path.join(__dirname, '..', 'public', file));

async function checkApiHealth() {
  const started = performance.now();
  try {
    const response = await fetch(HEALTH_URL, {
      method: 'GET',
      headers: { Accept: 'application/json' },
      signal: controller.signal,
    });

    const latencyMs = Math.round(performance.now() - started);

    if (!response.ok) {
      return {
        connected: false,
        latency_ms: null,
        error: {
          code: `HTTP_${response.status}`,
          message: `API health responded with status ${response.status}`,
          category: 'network',
          retryable: response.status >= 500,
        },
      };
    }

    return {
      connected: true,
      latency_ms: latencyMs,
      error: null,
    };
  } catch (error) {
    const latencyMs = Math.round(performance.now() - started);
    const code = error.name === 'AbortError' ? 'TIMEOUT' : (error.code || 'CONNECTION_ERROR');
    return {
      connected: false,
      latency_ms: null,
      error: {
        code,
        message: error.message || 'Failed to reach API health endpoint',
        category: 'network',
        retryable: true,
        details: { latency_ms: latencyMs },
      },
    };
  } finally {
    clearTimeout(timeout);
  }
}

(async function generateHealthPayload() {
  const nowIso = new Date().toISOString();
  const connectivity = await checkApiHealth();

  const payload = {
    status: connectivity.connected ? 'healthy' : 'degraded',
    service: 'react-component-library-ui',
    timestamp: nowIso,
    readiness: true,
    api_connectivity: {
      connected: connectivity.connected,
      api_url: API_ORIGIN,
      last_check: nowIso,
      latency_ms: connectivity.latency_ms,
      error: connectivity.error,
    },
  };

  const serialized = `${JSON.stringify(payload, null, 2)}\n`;
  for (const filePath of outputPaths) {
    fs.writeFileSync(filePath, serialized, 'utf8');
  }
})();
