import type { App } from '@/types';

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
    const parsed = Number(trimmed);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return null;
};

const candidatePortKeys = [
  'ui_port',
  'ui',
  'app_port',
  'web_port',
  'client_port',
  'http_port',
  'http',
  'port',
];

const getPortFromMappings = (portMappings: Record<string, unknown>, key: string): number | null => {
  const normalizedKey = key.toLowerCase();
  for (const [label, value] of Object.entries(portMappings)) {
    if (label.toLowerCase() === normalizedKey) {
      const parsed = parsePort(value);
      if (parsed !== null) {
        return parsed;
      }
    }
  }

  return null;
};

const computeAppUIPort = (app: App): number | null => {
  const portMappings = (app.port_mappings ?? {}) as Record<string, unknown>;
  const config = app.config ?? {};

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
  }

  for (const value of Object.values(portMappings)) {
    const mappedPort = parsePort(value);
    if (mappedPort !== null) {
      return mappedPort;
    }
  }

  const fallbackPort = parsePort(app.port);
  if (fallbackPort !== null) {
    return fallbackPort;
  }

  return null;
};

export const buildPreviewUrl = (app: App): string | null => {
  const config = app.config ?? {};

  if (typeof config.ui_url === 'string' && config.ui_url.trim()) {
    const trimmed = config.ui_url.trim();
    if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
      return trimmed;
    }

    if (typeof window !== 'undefined') {
      const origin = window.location.origin.replace(/\/$/, '');
      return `${origin}/${trimmed.replace(/^\//, '')}`;
    }

    return `http://localhost/${trimmed.replace(/^\//, '')}`;
  }

  const uiPort = computeAppUIPort(app);
  if (uiPort === null) {
    return null;
  }

  if (typeof window === 'undefined') {
    return `http://localhost:${uiPort}`;
  }

  const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:';
  const hostname = window.location.hostname || 'localhost';
  return `${protocol}//${hostname}:${uiPort}`;
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
