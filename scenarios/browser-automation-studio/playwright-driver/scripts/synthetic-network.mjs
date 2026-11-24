import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

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
  // A populated, fully invented workspace for product-review captures. The
  // fixture makes the panes, unread cue and transcript observable while every
  // API request remains inside the synthetic route boundary. It contains no
  // host identity, real session id, file path, account data or credentials.
  'web-console-showcase': Object.freeze({
    apiPrefixes: ['/api/', '/health', '/vrooli.web_console.v1.'],
    health: { status: 'healthy', service: 'web-console', timestamp: '2026-01-01T00:00:00Z', readiness: true, version: 'synthetic', uptime_seconds: 1, dependencies: {} },
    defaultJson: {},
    responses: Object.freeze({
      '/api/v1/coding-backup': Object.freeze({
        decision: 'subscription',
        limit: Object.freeze({ harness: 'codex', kind: 'subscription' }),
        consent: null,
        on_credits: false,
        revocation_pending: false,
      }),
      '/vrooli.web_console.v1.sessions.SessionsService/ListArchived': Object.freeze({ sessions: Object.freeze([]), total: 0 }),
      '/vrooli.web_console.v1.sessions.SessionsService/List': Object.freeze({
        sessions: Object.freeze([
          Object.freeze({ id: 'synthetic-session-0001', shell: '/bin/bash', createdAt: '2026-01-01T00:00:00Z', cols: 100, rows: 30, backend: 'standard', survivesRestart: false, policy: Object.freeze({ mode: 'never' }), origin: 'SESSION_ORIGIN_UI', displayLabel: 'Research notes' }),
          Object.freeze({ id: 'synthetic-session-0002', shell: '/bin/bash', createdAt: '2026-01-01T00:01:00Z', cols: 100, rows: 30, backend: 'standard', survivesRestart: false, policy: Object.freeze({ mode: 'never' }), origin: 'SESSION_ORIGIN_UI', displayLabel: 'Launch draft' }),
          Object.freeze({ id: 'synthetic-session-0003', shell: '/bin/bash', createdAt: '2026-01-01T00:02:00Z', cols: 100, rows: 30, backend: 'standard', survivesRestart: false, policy: Object.freeze({ mode: 'never' }), origin: 'SESSION_ORIGIN_UI', displayLabel: 'Review output' }),
        ]),
        recovery: Object.freeze({ inProgress: false, total: 0, recovered: 0, awaitingRecovery: 0, adopted: 0 }),
        supportsStableCreateIdentity: false,
      }),
      '/vrooli.web_console.v1.workspace.WorkspaceService/GetLayout': Object.freeze({
        activePane: 'synthetic-session-0001',
        panes: Object.freeze([
          Object.freeze({ sessionId: 'synthetic-session-0001', name: 'Research notes', sortOrder: 0, groupId: 'synthetic-group-0001', headerColor: '#38bdf8', supportsMessagesView: true, manuallyUnread: false }),
          Object.freeze({ sessionId: 'synthetic-session-0002', name: 'Launch draft', sortOrder: 1, groupId: 'synthetic-group-0001', headerColor: '#a78bfa', supportsMessagesView: true, manuallyUnread: true }),
          Object.freeze({ sessionId: 'synthetic-session-0003', name: 'Review output', sortOrder: 2, groupId: 'synthetic-group-0002', headerColor: '#34d399', supportsMessagesView: true, manuallyUnread: false }),
        ]),
        groups: Object.freeze([
          Object.freeze({ id: 'synthetic-group-0001', name: 'Workbench', color: '#14b8a6', sortOrder: 0, isCollapsed: false }),
          Object.freeze({ id: 'synthetic-group-0002', name: 'Review', color: '#8b5cf6', sortOrder: 1, isCollapsed: true }),
        ]),
        roles: Object.freeze([]),
      }),
      '/vrooli.web_console.v1.conversation.ConversationService/Get': Object.freeze({
        sessionId: 'synthetic-session-0001',
        events: Object.freeze([
          Object.freeze({ id: 'synthetic-event-0001', sessionId: 'synthetic-session-0001', source: 'synthetic', role: 'user', text: 'Compare the product notes and summarize the next decision.', createdAt: '2026-01-01T00:03:00Z', sequence: 1, deliveryState: 'received', ttsState: 'idle', consumptionState: 'seen' }),
          Object.freeze({ id: 'synthetic-event-0002', sessionId: 'synthetic-session-0001', source: 'synthetic', role: 'assistant', text: 'The notes are organized by opportunity. The next step is to review the evidence and choose which draft to refine.', createdAt: '2026-01-01T00:03:15Z', sequence: 2, deliveryState: 'received', ttsState: 'idle', consumptionState: 'seen' }),
        ]),
        cursor: Object.freeze({ lastSeenSequence: 2, lastListenedSequence: 0 }),
        hasMore: false,
        oldestSequence: 1,
        newestSequence: 2,
        totalCount: 2,
        capture: Object.freeze({ state: 'MESSAGE_CAPTURE_STATE_CAPTURING', summary: 'Synthetic transcript for private product review.' }),
      }),
      '/vrooli.web_console.v1.conversation.ConversationService/UpdateCursor': Object.freeze({
        cursor: Object.freeze({ lastSeenSequence: 2, lastListenedSequence: 0 }),
      }),
      '/vrooli.web_console.v1.targets.TargetCatalogService/List': Object.freeze({
        state: 'CATALOG_STATE_READY',
        targets: Object.freeze([Object.freeze({ id: 'local', kind: 'local', label: 'This machine', os: 'linux', arch: 'amd64', online: true, dispatchable: true, state: 'TARGET_STATE_DISPATCHABLE', readiness: Object.freeze([
          Object.freeze({ key: 'capability:codex', label: 'Codex', passed: true, detail: 'Synthetic capture fixture', state: 'ready', version: 'fixture' }),
          Object.freeze({ key: 'capability:claude', label: 'Claude', passed: true, detail: 'Synthetic capture fixture', state: 'ready', version: 'fixture' }),
          Object.freeze({ key: 'capability:opencode', label: 'OpenCode', passed: true, detail: 'Synthetic capture fixture', state: 'ready', version: 'fixture' }),
        ]) })]),
      }),
      '/vrooli.web_console.v1.capabilities.CapabilitiesService/Get': Object.freeze({ capabilities: Object.freeze([]), timestamp: '2026-01-01T00:00:00Z', sessionBackends: Object.freeze([{ id: 'standard', displayName: 'Terminal', available: true }]), defaultBackend: 'standard' }),
      '/vrooli.web_console.v1.capabilities.CapabilitiesService/Liveness': Object.freeze({ capabilities: Object.freeze([]), timestamp: '2026-01-01T00:00:00Z', sessionBackends: Object.freeze([{ id: 'standard', displayName: 'Terminal', available: true }]), defaultBackend: 'standard' }),
      '/vrooli.web_console.v1.capabilities.CapabilitiesService/ListHarnesses': Object.freeze({ targetId: 'local', harnesses: Object.freeze([
        Object.freeze({ agentId: 'codex', resource: 'codex', displayName: 'Codex', usageWindows: Object.freeze([]), supportsContextLimit: false, supportsUsageMode: false, supportsModelSelection: false, supportsProviderSelection: false }),
        Object.freeze({ agentId: 'claude', resource: 'claude', displayName: 'Claude', usageWindows: Object.freeze([]), supportsContextLimit: false, supportsUsageMode: false, supportsModelSelection: false, supportsProviderSelection: false }),
        Object.freeze({ agentId: 'opencode', resource: 'opencode', displayName: 'OpenCode', usageWindows: Object.freeze([]), supportsContextLimit: false, supportsUsageMode: false, supportsModelSelection: false, supportsProviderSelection: false }),
      ]) }),
    }),
  }),
});

// This follow-on profile adds only invented linked artifacts to the existing
// showcase transcript. Resolve, text and blob replies stay inside the same
// strict synthetic route boundary; request bodies are used transiently and are
// never copied into the network receipt.
const scenarioRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const mediaBase = SYNTHETIC_PROFILES['web-console-showcase'];
const mediaEvents = Object.freeze([
  ...mediaBase.responses['/vrooli.web_console.v1.conversation.ConversationService/Get'].events,
  Object.freeze({
    id: 'synthetic-event-0003', sessionId: 'synthetic-session-0001', source: 'synthetic', role: 'assistant',
    text: 'Here are the review artifacts: [HTML preview](/showcase/overview.html), [workbook preview](/showcase/operations.xlsx), [image preview](/showcase/diagram.svg), [audio preview](/showcase/notification.wav), and [video preview](/showcase/product-loop.mp4).',
    createdAt: '2026-01-01T00:03:30Z', sequence: 3, deliveryState: 'received', ttsState: 'idle', consumptionState: 'seen',
  }),
]);
const mediaFixtures = Object.freeze({
  '/showcase/overview.html': Object.freeze({
    id: 'synthetic-preview-html', basename: 'overview.html', kind: 'PREVIEW_KIND_TEXT', mimeType: 'text/html', textContentAvailable: true,
    text: '<!doctype html><html><head><meta charset="utf-8"><style>body{font:16px system-ui;color:#eef4ff;background:#111827;padding:28px}main{max-width:680px;margin:auto}h1{color:#67e8f9}section{border:1px solid #334155;border-radius:12px;padding:20px}</style></head><body><main><h1>Release overview</h1><section><strong>Workspace handoff</strong><p>Build checks are complete. Review the summary and choose the next safe step.</p></section></main></body></html>',
  }),
  '/showcase/operations.xlsx': Object.freeze({
    id: 'synthetic-preview-workbook', basename: 'operations.xlsx', sizeBytes: 37324, kind: 'PREVIEW_KIND_SPREADSHEET', mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    asset: path.join(scenarioRoot, 'web-console/fixtures/file-preview/operations-workbook.xlsx'),
  }),
  '/showcase/diagram.svg': Object.freeze({
    id: 'synthetic-preview-image', basename: 'diagram.svg', kind: 'PREVIEW_KIND_SVG', mimeType: 'image/svg+xml', textContentAvailable: true,
    text: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 450"><defs><linearGradient id="g"><stop stop-color="#0f172a"/><stop offset="1" stop-color="#172554"/></linearGradient></defs><rect width="800" height="450" rx="24" fill="url(#g)"/><circle cx="170" cy="225" r="56" fill="#14b8a6"/><path d="M226 225h165m0 0-24-24m24 24-24 24" fill="none" stroke="#67e8f9" stroke-width="8" stroke-linecap="round"/><rect x="400" y="165" width="230" height="120" rx="18" fill="#1e293b" stroke="#475569" stroke-width="3"/><text x="446" y="232" fill="#e2e8f0" font-family="sans-serif" font-size="24">Review ready</text></svg>',
  }),
  '/showcase/notification.wav': Object.freeze({
    id: 'synthetic-preview-audio', basename: 'notification.wav', kind: 'PREVIEW_KIND_AUDIO', mimeType: 'audio/wav',
    base64: 'UklGRuYPAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgATElTVBoAAABJTkZPSVNGVA4AAABMYXZmNjAuMTYuMTAwAGRhdGGgDwAAgIeNj46IgHhycHF3foeNj46JgXlzcHF2fYaMj46KgnpzcHB1fYWLj4+Kg3t0cHB0fISLj4+LhHx1cHB0e4OKj4+MhX11cXBzeoKJjo+Mhn52cXByeYGIjo+Nh393cXByeICIjY+NiIB4cnBxd3+HjY+OiIF5cnBxdn6GjI+OiYJ6c3BxdX2FjI+PioN7dHBwdXyEi4+Pi4R8dHBwdHuDio+PjIV9dXBwc3qCio6PjIZ9dnFwc3mBiY6PjYd+d3FwcniAiI6PjYeAeHJwcXd/h42PjoiBeHJwcXZ+hoyPjomCeXNwcXV9hYyPj4qDenRwcHV8hIuPj4uDe3RwcHR7g4qPj4uEfHVwcHN6goqOj4yFfXZxcHN5gYmOj42GfndxcHJ4gIiOj42Hf3dycHJ3f4eNj46IgHhycHF3foaNj46JgXlzcHF2fYWMj46KgnpzcHB1fISLj4+Kg3t0cHB0e4OLj4+LhHx1cHB0eoKKj4+MhX12cXBzeYKJjo+Mhn52cXByeIGIjo+Nh393cXByeICHjY+OiIB4cnBxd36HjY+OiYF5c3Bxdn2GjI+OioJ6c3BwdX2Fi4+PioN7dHBwdHyEi4+Pi4R8dXBwdHuDio+PjIV9dXFwc3qCiY6PjIZ+dnFwcnmBiI6PjYd/d3FwcniAiI2PjYiAeHJwcXd/h42PjoiBeXJwcXZ+hoyPjomCenNwcXV9hYyPj4qDe3RwcHV8hIuPj4uEfHRwcHR7g4qPj4yFfXVwcHN6goqOj4yGfXZxcHN5gYmOj42HfndxcHJ4gIiOj42HgHhycHF3f4eNj46IgXhycHF2foaMj46JgnlzcHF1fYWMj4+Kg3p0cHB1fISLj4+Lg3t0cHB0e4OKj4+LhHx1cHBzeoKKjo+MhX12cXBzeYGJjo+Nhn53cXByeICIjo+Nh393cnByd3+HjY+OiIB4cnBxd36GjY+OiYF5c3Bxdn2FjI+OioJ6c3BwdXyEi4+PioN7dHBwdHuDi4+Pi4R8dXBwdHqCio+PjIV9dnFwc3mCiY6PjIZ+dnFwcniBiI6PjYd/d3FwcniAh42PjoiAeHJwcXd+h42PjomBeXNwcXZ9hoyPjoqCenNwcHV9hYuPj4qDe3RwcHR8hIuPj4uEfHVwcHR7g4qPj4yFfXVxcHN6gomOj4yGfnZxcHJ5gYiOj42Hf3dxcHJ4gIiNj42IgHhycHF3f4eNj46IgXlycHF2foaMj46JgnpzcHF1fYWMj4+Kg3t0cHB1fISLj4+LhHx0cHB0e4OKj4+MhX11cHBzeoKKjo+Mhn12cXBzeYGJjo+Nh353cXByeICIjo+Nh4B4cnBxd3+HjY+OiIF4cnBxdn6GjI+OiYJ5c3BxdX2FjI+PioN6dHBwdXyEi4+Pi4N7dHBwdHuDio+Pi4R8dXBwc3qCio6PjIV9dnFwc3mBiY6PjYZ+d3FwcniAiI6PjYd/d3Jwcnd/h42PjoiAeHJwcXd+ho2PjomBeXNwcXZ9hYyPjoqCenNwcHV8hIuPj4qDe3RwcHR7g4uPj4uEfHVwcHR6goqPj4yFfXZxcHN5gomOj4yGfnZxcHJ4gYiOj42Hf3dxcHJ4gIeNj46IgHhycHF3foeNj46JgXlzcHF2fYaMj46KgnpzcHB1fYWLj4+Kg3t0cHB0fISLj4+LhHx1cHB0e4OKj4+MhX11cXBzeoKJjo+Mhn52cXByeYGIjo+Nh393cXByeICIjY+NiIB4cnBxd3+HjY+OiIF5cnBxdn6GjI+OiYJ6c3BxdX2FjI+PioN7dHBwdXyEi4+Pi4R8dHBwdHuDio+PjIV9dXBwc3qCio6PjIZ9dnFwc3mBiY6PjYd+d3FwcniAiI6PjYeAeHJwcXd/h42PjoiBeHJwcXZ+hoyPjomCeXNwcXV9hYyPj4qDenRwcHV8hIuPj4uDe3RwcHR7g4qPj4uEfHVwcHN6goqOj4yFfXZxcHN5gYmOj42GfndxcHJ4gIiOj42Hf3dycHJ3f4eNj46IgHhycHF3foaNj46JgXlzcHF2fYWMj46KgnpzcHB1fISLj4+Kg3t0cHB0e4OLj4+LhHx1cHB0eoKKj4+MhX12cXBzeYKJjo+Mhn52cXByeIGIjo+Nh393cXByeICHjY+OiIB4cnBxd36HjY+OiYF5c3Bxdn2GjI+OioJ6c3BwdX2Fi4+PioN7dHBwdHyEi4+Pi4R8dXBwdHuDio+PjIV9dXFwc3qCiY6PjIZ+dnFwcnmBiI6PjYd/d3FwcniAiI2PjYiAeHJwcXd/h42PjoiBeXJwcXZ+hoyPjomCenNwcXV9hYyPj4qDe3RwcHV8hIuPj4uEfHRwcHR7g4qPj4yFfXVwcHN6goqOj4yGfXZxcHN5gYmOj42HfndxcHJ4gIiOj42HgHhycHF3f4eNj46IgXhycHF2foaMj46JgnlzcHF1fYWMj4+Kg3p0cHB1fISLj4+Lg3t0cHB0e4OKj4+LhHx1cHBzeoKKjo+MhX12cXBzeYGJjo+Nhn53cXByeICIjo+Nh393cnByd3+HjY+OiIB4cnBxd36GjY+OiYF5c3Bxdn2FjI+OioJ6c3BwdXyEi4+PioN7dHBwdHuDi4+Pi4R8dXBwdHqCio+PjIV9dnFwc3mCiY6PjIZ+dnFwcniBiI6PjYd/d3FwcniAh42PjoiAeHJwcXd+h42PjomBeXNwcXZ9hoyPjoqCenNwcHV9hYuPj4qDe3RwcHR8hIuPj4uEfHVwcHR7g4qPj4yFfXVxcHN6gomOj4yGfnZxcHJ5gYiOj42Hf3dxcHJ4gIiNj42IgHhycHF3f4eNj46IgXlycHF2foaMj46JgnpzcHF1fYWMj4+Kg3t0cHB1fISLj4+LhHx0cHB0e4OKj4+MhX11cHBzeoKKjo+Mhn12cXBzeYGJjo+Nh353cXByeICIjo+Nh4B4cnBxd3+HjY+OiIF4cnBxdn6GjI+OiYJ5c3BxdX2FjI+PioN6dHBwdXyEi4+Pi4N7dHBwdHuDio+Pi4R8dXBwc3qCio6PjIV9dnFwc3mBiY6PjYZ+d3FwcniAiI6PjYd/d3Jwcnd/h42PjoiAeHJwcXd+ho2PjomBeXNwcXZ9hYyPjoqCenNwcHV8hIuPj4qDe3RwcHR7g4uPj4uEfHVwcHR6goqPj4yFfXZxcHN5gomOj4yGfnZxcHJ4gYiOj42Hf3dxcHJ4gIeNj46IgHhycHF3foeNj46JgXlzcHF2fYaMj46KgnpzcHB1fYWLj4+Kg3t0cHB0fISLj4+LhHx1cHB0e4OKj4+MhX11cXBzeoKJjo+Mhn52cXByeYGIjo+Nh393cXByeICIjY+NiIB4cnBxd3+HjY+OiIF5cnBxdn6GjI+OiYJ6c3BxdX2FjI+PioN7dHBwdXyEi4+Pi4R8dHBwdHuDio+PjIV9dXBwc3qCio6PjIZ9dnFwc3mBiY6PjYd+d3FwcniAiI6PjYeAeHJwcXd/h42PjoiBeHJwcXZ+hoyPjomCeXNwcXV9hYyPj4qDenRwcHV8hIuPj4uDe3RwcHR7g4qPj4uEfHVwcHN6goqOj4yFfXZxcHN5gYmOj42GfndxcHJ4gIiOj42Hf3dycHJ3f4eNj46IgHhycHF3foaNj46JgXlzcHF2fYWMj46KgnpzcHB1fISLj4+Kg3t0cHB0e4OLj4+LhHx1cHB0eoKKj4+MhX12cXBzeYKJjo+Mhn52cXByeIGIjo+Nh393cXByeICHjY+OiIB4cnBxd36HjY+OiYF5c3Bxdn2GjI+OioJ6c3BwdX2Fi4+PioN7dHBwdHyEi4+Pi4R8dXBwdHuDio+PjIV9dXFwc3qCiY6PjIZ+dnFwcnmBiI6PjYd/d3FwcniAiI2PjYiAeHJwcXd/h42PjoiBeXJwcXZ+hoyPjomCenNwcXV9hYyPj4qDe3RwcHV8hIuPj4uEfHRwcHR7g4qPj4yFfXVwcHN6goqOj4yGfXZxcHN5gYmOj42HfndxcHJ4gIiOj42HgHhycHF3f4eNj46IgXhycHF2foaMj46JgnlzcHF1fYWMj4+Kg3p0cHB1fISLj4+Lg3t0cHB0e4OKj4+LhHx1cHBzeoKKjo+MhX12cXBzeYGJjo+Nhn53cXByeICIjo+Nh393cnByd3+HjY+OiIB4cnBxd36GjY+OiYF5c3Bxdn2FjI+OioJ6c3BwdXyEi4+PioN7dHBwdHuDi4+Pi4R8dXBwdHqCio+PjIV9dnFwc3mCiY6PjIZ+dnFwcniBiI6PjYd/d3FwcniAh42PjoiAeHJwcXd+h42PjomBeXNwcXZ9hoyPjoqCenNwcHV9hYuPj4qDe3RwcHR8hIuPj4uEfHVwcHR7g4qPj4yFfXVxcHN6gomOj4yGfnZxcHJ5gYiOj42Hf3dxcHJ4gIiNj42IgHhycHF3f4eNj46IgXlycHF2foaMj46JgnpzcHF1fYWMj4+Kg3t0cHB1fISLj4+LhHx0cHB0e4OKj4+MhX11cHBzeoKKjo+Mhn12cXBzeYGJjo+Nh353cXByeICIjo+Nh4B4cnBxd3+HjY+OiIF4cnBxdn6GjI+OiYJ5c3BxdX2FjI+PioN6dHBwdXyEi4+Pi4N7dHBwdHuDio+Pi4R8dXBwc3qCio6PjIV9dnFwc3mBiY6PjYZ+d3FwcniAiI6PjYd/d3Jwcnd/h42PjoiAeHJwcXd+ho2PjomBeXNwcXZ9hYyPjoqCenNwcHV8hIuPj4qDe3RwcHR7g4uPj4uEfHVwcHR6goqPj4yFfXZxcHN5gomOj4yGfnZxcHJ4gYiOj42Hf3dxcHJ4gIeNj46IgHhycHF3foeNj46JgXlzcHF2fYaMj46KgnpzcHB1fYWLj4+Kg3t0cHB0fISLj4+LhHx1cHB0e4OKj4+MhX11cXBzeoKJjo+Mhn52cXByeYGIjo+Nh393cXByeICIjY+NiIB4cnBxd3+HjY+OiIF5cnBxdn6GjI+OiYJ6c3BxdX2FjI+PioN7dHBwdXyEi4+Pi4R8dHBwdHuDio+PjIV9dXBwc3qCio6PjIZ9dnFwc3mBiY6PjYd+d3FwcniAiI6PjYeAeHJwcXd/h42PjoiBeHJwcXZ+hoyPjomCeXNwcXV9hYyPj4qDenRwcHV8hIuPj4uDe3RwcHR7g4qPj4uEfHVwcHN6goqOj4yFfXZxcHN5gYmOj42GfndxcHJ4gIiOj42Hf3dycHJ3f4eNj46IgHhycHF3foaNj46JgXlzcHF2fYWMj46KgnpzcHB1fISLj4+Kg3t0cHB0e4OLj4+LhHx1cHB0eoKKj4+MhX12cXBzeYKJjo+Mhn52cXByeIGIjo+Nh393cXByeA==',
  }),
  '/showcase/product-loop.mp4': Object.freeze({
    id: 'synthetic-preview-video', basename: 'product-loop.mp4', kind: 'PREVIEW_KIND_VIDEO', mimeType: 'video/mp4',
    base64: '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uuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuuvAAAAAdBmiAqgBzMAAAAB0GaQCqAHMwAAAAHQZpgLoAczAAAAAdBmoAugBzMAAAAB0GaoC6AHMwAAAAHQZrALoAczAAAAAdBmuAugBzMAAAAB0GbAC6AHMwAAAAHQZsgLoAczAAAAAdBm0AugBzMAAAAB0GbYC6AHMwAAAAHQZuALoAczAAAAAdBm6AugBzMAAAAB0GbwC6AHMwAAAAHQZvgLoAczAAAAAdBmgAugBzMAAAAB0GaIC6AHMwAAAAHQZpALoAczAAAAAdBmmAugBzMAAAAB0GagC6AHMwAAAAHQZqgLoAczAAAAAdBmsAugBzMAAAAB0Ga4C6AHMwAAAAHQZsALoAczAAAAAdBmyAugBzMAAAAB0GbQC6AHMwAAAAHQZtgLoAczAAAAAdBm4AugBzMAAAAB0GboC6AHMwAAAAHQZvALoAczAAAAAdBm+AugBzMAAAAB0GaAC6AHMwAAAAHQZogLoAczAAAAAdBmkAugBzMAAAAB0GaYC6AHMwAAAAHQZqALoAczAAAAAdBmqAugBzMAAAAB0GawC6AHMwAAAAHQZrgLoAczAAAAAdBmwAugBzMAAAAB0GbIC6AHMwAAAAHQZtALoAczAAAAAdBm2AugBzMAAAAB0GbgC6AHMwAAAAHQZugLoAczAAAAAdBm8AugBzMAAAAB0Gb4C6AHMw=',
  }),
});

export function getSyntheticProfile(name) {
  if (name !== 'web-console-showcase-media') return SYNTHETIC_PROFILES[name];
  return {
    ...mediaBase,
    responses: {
      ...mediaBase.responses,
      '/vrooli.web_console.v1.conversation.ConversationService/Get': {
        ...mediaBase.responses['/vrooli.web_console.v1.conversation.ConversationService/Get'],
        events: mediaEvents,
        cursor: { lastSeenSequence: 3, lastListenedSequence: 0 },
        newestSequence: 3,
        totalCount: 3,
        capture: { state: 'MESSAGE_CAPTURE_STATE_CAPTURING', summary: 'Synthetic transcript and local review artifacts.' },
      },
      '/vrooli.web_console.v1.conversation.ConversationService/UpdateCursor': { cursor: { lastSeenSequence: 3, lastListenedSequence: 0 } },
    },
    previewFixtures: mediaFixtures,
  };
}

export function isApplicationApi(url, resourceType, profile) {
  const pathname = new URL(url).pathname;
  return profile.apiPrefixes.some((prefix) => pathname.startsWith(prefix))
    || resourceType === 'fetch'
    || resourceType === 'xhr';
}

export function syntheticResponse(url, profile, resourceType = '', requestBody = undefined) {
  if (resourceType === 'eventsource') {
    return { status: 200, contentType: 'text/event-stream; charset=utf-8', body: 'event: synthetic-ready\ndata: {}\n\n' };
  }
  const pathname = new URL(url).pathname;
  if (profile.previewFixtures && pathname === '/vrooli.web_console.v1.file_preview.FilePreviewService/Resolve') {
    const fixture = profile.previewFixtures[requestBody?.path];
    if (fixture) {
      const byteSize = fixture.base64 ? Buffer.from(fixture.base64, 'base64').length : Buffer.byteLength(fixture.text ?? '');
      const media = ['PREVIEW_KIND_SPREADSHEET', 'PREVIEW_KIND_AUDIO', 'PREVIEW_KIND_VIDEO'].includes(fixture.kind);
      return {
        status: 200,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({
          previewId: fixture.id,
          inputPath: fixture.basename,
          resolvedPath: `/synthetic-workspace/${fixture.basename}`,
          basename: fixture.basename,
          hasLine: false,
          resolutionBasis: 'source_context',
          previewKind: fixture.kind,
          mimeType: fixture.mimeType,
          sizeBytes: String(fixture.sizeBytes ?? byteSize),
          canPreview: true,
          canDownload: false,
          supportsRange: media,
          textContentAvailable: Boolean(fixture.textContentAvailable),
          blobUrl: media ? `/api/v1/sessions/synthetic-session-0001/file-previews/${fixture.id}/blob` : '',
          expiresUnixNano: '4102444800000000000',
          warnings: [],
          listingAvailable: false,
          archiveMember: false,
          candidates: [],
          unresolved: false,
          searchBases: [],
        }),
      };
    }
  }
  if (profile.previewFixtures && pathname === '/vrooli.web_console.v1.file_preview.FilePreviewService/GetTextContent') {
    const fixture = Object.values(profile.previewFixtures).find((item) => item.id === requestBody?.previewId);
    if (fixture?.text !== undefined) {
      return {
        status: 200,
        contentType: 'application/json; charset=utf-8',
        body: JSON.stringify({ resolvedPath: `/synthetic-workspace/${fixture.basename}`, previewKind: fixture.kind, mimeType: fixture.mimeType, content: fixture.text, truncated: false, hasLine: false }),
      };
    }
  }
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
    if (url.origin === origin && url.pathname === '/' && type === 'document' && request.isNavigationRequest()) {
      return route.continue();
    }
    if (isApplicationApi(url.href, type, profile)) {
      onRecord({ url: url.href, method: request.method(), resourceType: type, outcome: 'synthetic' });
      const body = request.method() === 'POST' ? request.postDataJSON?.() : undefined;
      const response = syntheticResponse(url.href, profile, type, body);
      const blobMatch = /^\/api\/v1\/sessions\/synthetic-session-0001\/file-previews\/([^/]+)\/blob$/.exec(url.pathname);
      if (blobMatch && profile.previewFixtures) {
        const fixture = Object.values(profile.previewFixtures).find((item) => item.id === blobMatch[1]);
        if (fixture) {
          const bytes = fixture.base64
            ? Buffer.from(fixture.base64, 'base64')
            : fixture.asset ? await fs.readFile(fixture.asset) : undefined;
          if (bytes) return route.fulfill({ status: 200, contentType: fixture.mimeType, body: bytes, headers: { 'Accept-Ranges': 'bytes', 'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff' } });
        }
      }
      return route.fulfill(response);
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
