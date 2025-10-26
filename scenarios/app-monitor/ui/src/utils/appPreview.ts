import type { App } from '@/types';

/**
 * Normalizes an identifier string for comparison purposes.
 * Returns lowercase, trimmed string or null if invalid.
 */
export const normalizeIdentifier = (value?: string | null): string | null => {
  if (!value) {
    return null;
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed.toLowerCase() : null;
};

/**
 * Collects all valid app identifiers (id, scenario_name, name) as normalized strings.
 * Returns array of lowercase, trimmed identifiers.
 */
export const collectAppIdentifiers = (app: App): string[] => {
  return [app.id, app.scenario_name, app.name]
    .map(normalizeIdentifier)
    .filter((value): value is string => Boolean(value));
};

/**
 * Resolves the primary identifier for an app, preserving original case.
 * Priority: id > scenario_name > name
 * Returns trimmed string or null if no valid identifier found.
 */
export const resolveAppIdentifier = (app: App): string | null => {
  const candidates: Array<string | undefined> = [app.id, app.scenario_name, app.name];
  const match = candidates.find(value => typeof value === 'string' && value.trim().length > 0);
  return match ? match.trim() : null;
};

/**
 * Derives a unique key for an app for use in Map/Set structures.
 * Unlike resolveAppIdentifier, this ALWAYS returns a non-null string,
 * falling back to a generated UUID if no identifier is available.
 * Preserves original case for display purposes.
 */
export const deriveAppKey = (app: App): string => {
  const resolved = resolveAppIdentifier(app);
  if (resolved) {
    return resolved;
  }

  // Fallback to generated identifier
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }

  return `app-${Math.random().toString(36).slice(2)}`;
};

export const parseTimestampValue = (value?: string | null): number | null => {
  if (!value) {
    return null;
  }

  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return null;
  }

  return parsed;
};

const APP_PROXY_PREFIX = '/apps';

export const buildProxyPreviewUrl = (identifier: string): string => {
  return `${APP_PROXY_PREFIX}/${encodeURIComponent(identifier)}/proxy/`;
};

export const normalizeStatus = (status?: string | null) => (status ?? '').toLowerCase();

export const isRunningStatus = (status?: string | null) => {
  const normalized = normalizeStatus(status);
  return normalized === 'running' || normalized === 'healthy' || normalized === 'degraded' || normalized === 'unhealthy';
};

export const isStoppedStatus = (status?: string | null) => normalizeStatus(status) === 'stopped';

/**
 * Checks if a scenario is explicitly stopped (not running, not partial data).
 * A scenario is explicitly stopped when:
 * - It has complete data (not is_partial)
 * - Its status is "stopped"
 */
export const isScenarioExplicitlyStopped = (app: App | null): boolean => {
  return Boolean(app && !app.is_partial && isStoppedStatus(app.status));
};

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

  if (uiPort !== null && typeof app.id === 'string' && app.id.trim()) {
    return buildProxyPreviewUrl(app.id);
  }

  if (typeof config.ui_url === 'string' && config.ui_url.trim()) {
    return config.ui_url.trim();
  }

  for (const [label, rawValue] of Object.entries(environment)) {
    const normalizedLabel = label.toLowerCase();
    if (!normalizedLabel.includes('url')) {
      continue;
    }

    const candidate = unwrapValue(rawValue);
    if (typeof candidate === 'string' && candidate.trim()) {
      return candidate.trim();
    }
  }

  if (typeof app.port === 'number' && Number.isFinite(app.port)) {
    return `http://127.0.0.1:${app.port}`;
  }

  if (typeof app.port === 'string' && app.port.trim()) {
    return app.port.trim();
  }

  return null;
};

export const locateAppByIdentifier = (apps: App[], identifier: string): App | null => {
  if (!identifier) {
    return null;
  }

  const normalized = identifier.trim().toLowerCase();
  if (!normalized) {
    return null;
  }

  return (
    apps.find(app => {
      const candidates = [app.id, app.scenario_name, app.name]
        .map(value => (typeof value === 'string' ? value.trim().toLowerCase() : ''))
        .filter(value => value.length > 0);
      return candidates.includes(normalized);
    }) ?? null
  );
};

export const matchesAppIdentifier = (app: App, identifier?: string | null): boolean => {
  if (!identifier) {
    return false;
  }

  const normalized = normalizeIdentifier(identifier);
  if (!normalized) {
    return false;
  }

  const candidates = collectAppIdentifiers(app);
  return candidates.includes(normalized);
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
