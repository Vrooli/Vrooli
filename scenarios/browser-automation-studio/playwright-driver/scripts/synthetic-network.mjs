/**
 * Network boundary for served UI inventories. Application API traffic is
 * always fulfilled from synthetic fixtures; only same-origin static assets
 * and the requested document may reach the owner-started scenario.
 */
export const SYNTHETIC_PROFILES = Object.freeze({
  'web-console': Object.freeze({
    apiPrefixes: ['/api/', '/health', '/vrooli.web_console.v1.'],
    health: { status: 'healthy', service: 'web-console', timestamp: '2026-01-01T00:00:00Z', readiness: true, version: 'synthetic', uptime_seconds: 1, dependencies: {} },
    defaultJson: {},
  }),
  // Opt-in fixture for served Web Console states that require a hydrated
  // workspace. Every value is invented here; no session is created and no
  // data is read from the running API. All other API calls still receive the
  // empty generic response and are recorded as synthetic by the route owner.
  'web-console-session': Object.freeze({
    apiPrefixes: ['/api/', '/health', '/vrooli.web_console.v1.'],
    health: { status: 'healthy', service: 'web-console', timestamp: '2026-01-01T00:00:00Z', readiness: true, version: 'synthetic', uptime_seconds: 1, dependencies: {} },
    defaultJson: {},
    responses: Object.freeze({
      // The workspace's optional coding-backup state is read whenever a
      // session is active. `null` is its real absent-state contract; `{}` would
      // look like a partial record and crash while reading `limit.harness`.
      '/api/v1/coding-backup': null,
      '/vrooli.web_console.v1.sessions.SessionsService/ListArchived': Object.freeze({ sessions: Object.freeze([]), total: 0 }),
      '/vrooli.web_console.v1.sessions.SessionsService/List': Object.freeze({
        sessions: Object.freeze([Object.freeze({
          id: 'synthetic-session-0001',
          shell: '/bin/bash',
          createdAt: '2026-01-01T00:00:00Z',
          cols: 80,
          rows: 24,
          backend: 'standard',
          survivesRestart: false,
          policy: Object.freeze({ mode: 'never' }),
          origin: 'SESSION_ORIGIN_UI',
          displayLabel: 'Synthetic session',
        })]),
        recovery: Object.freeze({ inProgress: false, total: 0, recovered: 0, awaitingRecovery: 0, adopted: 0 }),
        supportsStableCreateIdentity: false,
      }),
      '/vrooli.web_console.v1.workspace.WorkspaceService/GetLayout': Object.freeze({
        activePane: 'synthetic-session-0001',
        panes: Object.freeze([Object.freeze({
          sessionId: 'synthetic-session-0001',
          name: 'Synthetic session',
          sortOrder: 0,
          supportsMessagesView: false,
          manuallyUnread: false,
        })]),
        groups: Object.freeze([]),
        roles: Object.freeze([]),
      }),
    }),
  }),
});

export function isApplicationApi(url, resourceType, profile) {
  const pathname = new URL(url).pathname;
  return profile.apiPrefixes.some((prefix) => pathname.startsWith(prefix))
    || resourceType === 'fetch'
    || resourceType === 'xhr';
}

export function syntheticResponse(url, profile, resourceType = '') {
  if (resourceType === 'eventsource') {
    return { status: 200, contentType: 'text/event-stream; charset=utf-8', body: 'event: synthetic-ready\ndata: {}\n\n' };
  }
  const pathname = new URL(url).pathname;
  const body = pathname === '/health'
    ? profile.health
    : (Object.hasOwn(profile.responses ?? {}, pathname) ? profile.responses[pathname] : profile.defaultJson);
  return { status: 200, contentType: 'application/json; charset=utf-8', body: JSON.stringify(body) };
}

export function createSyntheticRouteHandler({ origin, profile, onRecord = () => {} }) {
  return async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const type = request.resourceType();
    if (url.origin === origin && type === 'document' && request.isNavigationRequest()) {
      return route.continue();
    }
    if (isApplicationApi(url.href, type, profile)) {
      onRecord({ url: url.href, method: request.method(), resourceType: type, outcome: 'synthetic' });
      return route.fulfill(syntheticResponse(url.href, profile, type));
    }
    const staticAsset = ['script', 'stylesheet', 'image', 'font'].includes(type)
      && request.method() === 'GET' && url.origin === origin
      && !url.pathname.startsWith('/api/')
      && !url.pathname.startsWith('/vrooli.web_console.v1.');
    if (staticAsset) return route.continue();
    onRecord({ url: url.href, method: request.method(), resourceType: type, outcome: 'blocked' });
    return route.abort('blockedbyclient');
  };
}
