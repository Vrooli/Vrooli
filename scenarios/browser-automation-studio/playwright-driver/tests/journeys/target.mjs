const requiredEndpointKeys = [
  'BAS_JOURNEY_API_URL',
  'BAS_JOURNEY_UI_URL',
  'BAS_JOURNEY_DRIVER_URL',
];
const loopbackHosts = new Set(['127.0.0.1', 'localhost', '[::1]']);

function parseLocalEndpoint(value, key) {
  let endpoint;
  try {
    endpoint = new URL(value);
  } catch {
    throw new Error(`${key} must be an absolute local HTTP(S) URL`);
  }
  if (!['http:', 'https:'].includes(endpoint.protocol) || !loopbackHosts.has(endpoint.hostname)) {
    throw new Error(`${key} must use a loopback HTTP(S) URL`);
  }
  if (endpoint.username || endpoint.password) {
    throw new Error(`${key} must not include credentials`);
  }
  if (!endpoint.port || endpoint.pathname !== '/' || endpoint.search || endpoint.hash) {
    throw new Error(`${key} must include an explicit port and no path, query, or fragment`);
  }
  return endpoint;
}

export function resolveJourneyTarget(env, getPorts) {
  const configured = requiredEndpointKeys.map((key) => (env[key] ?? '').trim());
  const anyConfigured = configured.some(Boolean);
  const allConfigured = configured.every(Boolean);
  const instance = (env.BAS_JOURNEY_INSTANCE ?? 'shadow').trim();
  if (!instance || ['live', 'default'].includes(instance.toLowerCase())) {
    throw new Error('BAS_JOURNEY_INSTANCE must name an isolated non-live instance');
  }

  if (anyConfigured) {
    if (!allConfigured) {
      throw new Error(`explicit journey target requires all of ${requiredEndpointKeys.join(', ')}`);
    }
    const [apiEndpoint, uiEndpoint, driverEndpoint] = configured.map((value, index) =>
      parseLocalEndpoint(value, requiredEndpointKeys[index])
    );
    if (new Set([apiEndpoint.port, uiEndpoint.port, driverEndpoint.port]).size !== 3) {
      throw new Error('explicit journey API, UI, and driver endpoints must use distinct ports');
    }
    return {
      instance,
      api: apiEndpoint.origin,
      ui: uiEndpoint.origin,
      driver: driverEndpoint.origin,
      source: 'explicit-loopback',
    };
  }

  const ports = getPorts(instance);
  if (!ports.API_PORT || !ports.UI_PORT || !ports.PLAYWRIGHT_DRIVER_PORT) {
    throw new Error(`scenario port lookup did not return all BAS ports for ${instance}`);
  }
  if (new Set([ports.API_PORT, ports.UI_PORT, ports.PLAYWRIGHT_DRIVER_PORT]).size !== 3) {
    throw new Error(`scenario port lookup returned duplicate BAS ports for ${instance}`);
  }
  return {
    instance,
    api: `http://127.0.0.1:${ports.API_PORT}`,
    ui: `http://127.0.0.1:${ports.UI_PORT}`,
    driver: `http://127.0.0.1:${ports.PLAYWRIGHT_DRIVER_PORT}`,
    source: 'scenario-port',
  };
}
