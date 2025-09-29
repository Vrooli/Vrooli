import type { App } from '@/types';

const resolveLocalOrigin = (): string => {
  const fallback = 'http://127.0.0.1';

  try {
    const override =
      typeof import.meta !== 'undefined' &&
      import.meta.env &&
      typeof import.meta.env.VITE_APP_MONITOR_LOCAL_ORIGIN === 'string'
        ? import.meta.env.VITE_APP_MONITOR_LOCAL_ORIGIN.trim()
        : '';

    if (!override) {
      return fallback;
    }

    const candidate = new URL(override.includes('://') ? override : `http://${override}`);
    return candidate.origin;
  } catch (error) {
    if (typeof console !== 'undefined' && typeof console.warn === 'function') {
      console.warn('Invalid VITE_APP_MONITOR_LOCAL_ORIGIN override; falling back to 127.0.0.1', error);
    }
    return fallback;
  }
};

const LOCAL_ORIGIN = resolveLocalOrigin();
const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '::1', '[::1]']);

const buildFromLocalOrigin = (path: string): string => {
  try {
    const base = new URL(LOCAL_ORIGIN);
    const normalizedPath = path.replace(/^\//, '');
    const joined = new URL(normalizedPath, base);
    return joined.toString();
  } catch {
    const normalizedPath = path.replace(/^\//, '');
    return `${LOCAL_ORIGIN.replace(/\/$/, '')}/${normalizedPath}`;
  }
};

const buildFromLocalPort = (port: number): string => {
  try {
    const base = new URL(LOCAL_ORIGIN);
    if (port) {
      base.port = String(port);
    }
    return base.toString();
  } catch {
    return `${LOCAL_ORIGIN.replace(/:\d+$/, '')}:${port}`;
  }
};

const coerceAbsoluteToLocal = (absoluteUrl: string, fallbackPort?: number | null): string => {
  try {
    const parsed = new URL(absoluteUrl);
    if (LOOPBACK_HOSTS.has(parsed.hostname)) {
      return parsed.toString();
    }

    const localBase = new URL(LOCAL_ORIGIN);
    localBase.pathname = parsed.pathname;
    localBase.search = parsed.search;
    localBase.hash = parsed.hash;

    if (parsed.port) {
      localBase.port = parsed.port;
    } else if (fallbackPort) {
      localBase.port = String(fallbackPort);
    }

    return localBase.toString();
  } catch {
    if (fallbackPort) {
      return buildFromLocalPort(fallbackPort);
    }
    return buildFromLocalOrigin(absoluteUrl);
  }
};

export const normalizeStatus = (status?: string | null) => (status ?? '').toLowerCase();

export const isRunningStatus = (status?: string | null) => {
  const normalized = normalizeStatus(status);
  return normalized === 'running' || normalized === 'healthy' || normalized === 'degraded' || normalized === 'unhealthy';
};

export const isStoppedStatus = (status?: string | null) => normalizeStatus(status) === 'stopped';

const parsePort = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (!trimmed) return null;
    if (/^\d+$/.test(trimmed)) {
      const numeric = Number(trimmed);
      return Number.isFinite(numeric) ? numeric : null;
    }

    try {
      const url = new URL(trimmed, trimmed.startsWith('http') ? undefined : 'http://placeholder');
      const portValue = url.port ? Number(url.port) : null;
      if (portValue !== null && Number.isFinite(portValue)) {
        return portValue;
      }
    } catch (error) {
      const match = trimmed.match(/:(\d+)(?!.*:\d+)/);
      if (match) {
        const candidate = Number(match[1]);
        if (Number.isFinite(candidate)) {
          return candidate;
        }
      }
    }
  }

  return null;
};

const unwrapValue = (value: unknown, depth = 0): unknown => {
  if (depth > 3) {
    return value;
  }

  if (Array.isArray(value) && value.length > 0) {
    return unwrapValue(value[0], depth + 1);
  }

  if (value && typeof value === 'object') {
    const record = value as Record<string, unknown>;
    if ('value' in record) {
      return unwrapValue(record.value, depth + 1);
    }
    if ('port' in record) {
      return unwrapValue(record.port, depth + 1);
    }
  }

  return value;
};

const candidatePortKeys = [
  'ui_port',
  'ui',
  'app_port',
  'web_port',
  'client_port',
  'frontend_port',
  'preview_port',
  'vite_port',
  'http_port',
  'http',
  'port',
];

const getPortFromMappings = (portMappings: Record<string, unknown>, key: string): number | null => {
  const normalizedKey = key.toLowerCase();
  for (const [label, value] of Object.entries(portMappings)) {
    if (label.toLowerCase() === normalizedKey) {
      const parsed = parsePort(unwrapValue(value));
      if (parsed !== null) {
        return parsed;
      }
    }
  }

  return null;
};

const getPortFromEnvironment = (environment: Record<string, unknown>, key: string): number | null => {
  const normalizedKey = key.toLowerCase();
  for (const [label, rawValue] of Object.entries(environment)) {
    const normalizedLabel = label.toLowerCase();
    if (
      normalizedLabel === normalizedKey ||
      normalizedLabel.endsWith(`_${normalizedKey}`) ||
      normalizedLabel.endsWith(`-${normalizedKey}`) ||
      normalizedLabel.includes(normalizedKey)
    ) {
      const parsed = parsePort(unwrapValue(rawValue));
      if (parsed !== null) {
        return parsed;
      }
    }
  }
  return null;
};

const findPortInEnvironment = (environment: Record<string, unknown>): number | null => {
  let fallbackPort: number | null = null;

  for (const [label, rawValue] of Object.entries(environment)) {
    const normalizedLabel = label.toLowerCase();
    const value = unwrapValue(rawValue);
    const parsed = parsePort(value);

    if (parsed === null) {
      continue;
    }

    if (normalizedLabel.includes('ui') && normalizedLabel.includes('port')) {
      return parsed;
    }

    if (!fallbackPort) {
      if (normalizedLabel.includes('url') && normalizedLabel.includes('ui')) {
        return parsed;
      }

      if (
        normalizedLabel.includes('web_port') ||
        normalizedLabel.includes('frontend') ||
        normalizedLabel.includes('preview')
      ) {
        fallbackPort = parsed;
      } else if (normalizedLabel.includes('port')) {
        fallbackPort = fallbackPort ?? parsed;
      }
    }
  }

  return fallbackPort;
};

const computeAppUIPort = (app: App): number | null => {
  const portMappings = (app.port_mappings ?? {}) as Record<string, unknown>;
  const config = app.config ?? {};
  const environment = (app.environment ?? {}) as Record<string, unknown>;

  const primaryPort = parsePort(config.primary_port);
  if (primaryPort !== null) {
    return primaryPort;
  }

  if (typeof config.primary_port_label === 'string') {
    const labeledPort = getPortFromMappings(portMappings, config.primary_port_label);
    if (labeledPort !== null) {
      return labeledPort;
    }
  }

  for (const key of candidatePortKeys) {
    const mappedPort = getPortFromMappings(portMappings, key);
    if (mappedPort !== null) {
      return mappedPort;
    }

    const envPort = getPortFromEnvironment(environment, key);
    if (envPort !== null) {
      return envPort;
    }
  }

  for (const value of Object.values(portMappings)) {
    const mappedPort = parsePort(unwrapValue(value));
    if (mappedPort !== null) {
      return mappedPort;
    }
  }

  const environmentPort = findPortInEnvironment(environment);
  if (environmentPort !== null) {
    return environmentPort;
  }

  const fallbackPort = parsePort(app.port);
  if (fallbackPort !== null) {
    return fallbackPort;
  }

  return null;
};

export const buildPreviewUrl = (app: App): string | null => {
  const config = app.config ?? {};
  const environment = (app.environment ?? {}) as Record<string, unknown>;
  const uiPort = computeAppUIPort(app);

  const resolveCandidate = (value: string): string => {
    const trimmed = value.trim();
    if (!trimmed) {
      return trimmed;
    }

    if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
      return coerceAbsoluteToLocal(trimmed, uiPort);
    }

    return buildFromLocalOrigin(trimmed);
  };

  if (typeof config.ui_url === 'string' && config.ui_url.trim()) {
    return resolveCandidate(config.ui_url);
  }

  for (const [label, rawValue] of Object.entries(environment)) {
    const normalizedLabel = label.toLowerCase();
    if (!normalizedLabel.includes('url')) {
      continue;
    }

    const candidate = unwrapValue(rawValue);
    if (typeof candidate !== 'string') {
      continue;
    }

    const resolved = resolveCandidate(candidate);
    if (resolved) {
      return resolved;
    }
  }

  if (uiPort === null) {
    return null;
  }

  return buildFromLocalPort(uiPort);
};

export const locateAppByIdentifier = (apps: App[], identifier: string): App | null => {
  return (
    apps.find(app => app.id === identifier || app.name === identifier || app.scenario_name === identifier) ??
    null
  );
};

export interface PortMetric {
  label: string;
  value: string;
}

export const orderedPortMetrics = (app: App): PortMetric[] => {
  const entries = Object.entries(app.port_mappings || {})
    .map(([label, value]) => ({
      label: label.toUpperCase(),
      value: typeof value === 'number' ? String(value) : String(value ?? ''),
    }))
    .filter(({ value }) => value !== '');

  const priorityOrder = ['UI_PORT', 'API_PORT'];
  const prioritized = entries
    .filter((entry) => priorityOrder.includes(entry.label))
    .sort((a, b) => priorityOrder.indexOf(a.label) - priorityOrder.indexOf(b.label));

  const remaining = entries
    .filter((entry) => !priorityOrder.includes(entry.label))
    .sort((a, b) => a.label.localeCompare(b.label));

  return [...prioritized, ...remaining];
};
