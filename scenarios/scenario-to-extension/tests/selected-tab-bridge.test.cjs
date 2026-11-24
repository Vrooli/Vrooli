const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const workerPath = path.join(__dirname, '..', 'templates', 'selected-tab-bridge', 'service-worker.js');
const workerSource = fs.readFileSync(workerPath, 'utf8');

function deferred() {
  let resolve;
  const promise = new Promise(done => { resolve = done; });
  return { promise, resolve };
}

class MockSocket {
  static OPEN = 1;

  constructor(url, protocols) {
    this.url = url;
    this.protocols = protocols;
    this.readyState = 0;
    this.listeners = new Map();
    this.sent = [];
    setImmediate(() => {
      this.readyState = MockSocket.OPEN;
      this.emit('open', {});
    });
  }

  addEventListener(type, listener) {
    const listeners = this.listeners.get(type) || new Set();
    listeners.add(listener);
    this.listeners.set(type, listeners);
  }

  removeEventListener(type, listener) {
    this.listeners.get(type)?.delete(listener);
  }

  emit(type, event) {
    for (const listener of this.listeners.get(type) || []) listener(event);
  }

  send(raw) {
    const message = JSON.parse(raw);
    this.sent.push(message);
    if (message.type === 'lease') {
      setImmediate(() => this.emit('message', { data: JSON.stringify({ type: 'lease_accepted' }) }));
    }
  }

  close() {
    if (this.readyState === 3) return;
    this.readyState = 3;
    this.emit('close', {});
  }

  serverSend(message) {
    this.emit('message', { data: JSON.stringify(message) });
  }
}

function createHarness(options = {}) {
  let now = Date.now();
  let runtimeListener;
  let updatedListener;
  let activatedListener;
  let queryOverride = null;
  let scriptCalls = 0;
  let managementCalls = 0;
  let tabUpdates = 0;
  let tabCreateCalls = 0;
  let tabRemoveCalls = 0;
  let cookieWriteCalls = 0;
  const scriptTargets = [];
  const sockets = [];
  let grantNumber = 0;
  const tab = { id: 9, url: 'https://example.test/public/start' };
  const otherTab = { id: 10, url: 'https://mail.example.test/inbox' };
  const signInState = { sessionCookie: 'session-cookie-untouched', localStorageToken: 'local-token-untouched' };
  const grantLifetime = options.grantLifetime ?? 60_000;
  const grant = () => {
    grantNumber += 1;
    return {
      session_id: `session-${grantNumber}`,
      websocket_url: `ws://localhost:9000/browser/bridge/socket?ticket=${grantNumber}`,
      lease_receipt: `lease-${grantNumber}`,
      expires_at: new Date(now + grantLifetime).toISOString(),
      permission_profile: {
        effect: 'act',
        url_allow: ['https://example.test/public/*'],
        url_deny: [],
        view: true,
        control: true,
        extension_access: Boolean(options.extensionAccess),
        extension_ids: options.extensionIDs || [],
      },
    };
  };

  const chrome = {
    runtime: {
      id: 'bridge-extension',
      onMessage: { addListener(listener) { runtimeListener = listener; } },
    },
    tabs: {
      onUpdated: { addListener(listener) { updatedListener = listener; } },
      onActivated: { addListener(listener) { activatedListener = listener; } },
      async query() {
        if (queryOverride) return queryOverride();
        return [{ ...tab }];
      },
      async update(tabId, change) {
        tabUpdates += 1;
        if (tabId !== tab.id) throw new Error(`unexpected tab update target ${tabId}`);
        tab.url = change.url;
      },
      async create() { tabCreateCalls += 1; return { ...otherTab }; },
      async remove(tabId) {
        tabRemoveCalls += 1;
        if (tabId === otherTab.id) throw new Error('fixture must preserve the unrelated tab');
      },
    },
    scripting: {
      async executeScript(options) {
        scriptCalls += 1;
        scriptTargets.push(options.target?.tabId);
        return [{ result: { ok: true, title: 'Public page', url: tab.url, text: 'ok' } }];
      },
    },
    management: { async getAll() {
      managementCalls += 1;
      return typeof options.inventory === 'function' ? options.inventory() : (options.inventory || []);
    } },
    permissions: { async request() { return true; } },
    cookies: {
      async set() { cookieWriteCalls += 1; return { value: signInState.sessionCookie }; },
      async remove() { cookieWriteCalls += 1; return { value: signInState.sessionCookie }; },
    },
  };
  class TrackedSocket extends MockSocket {
    constructor(url, protocols) {
      super(url, protocols);
      sockets.push(this);
    }
  }
  class MockDate extends Date {
    static now() { return now; }
  }
  const sandbox = {
    chrome,
    WebSocket: Object.assign(TrackedSocket, { OPEN: MockSocket.OPEN }),
    URL,
    Date: MockDate,
    fetch: async () => ({ ok: true, json: async () => grant() }),
    setTimeout,
    clearTimeout,
    setInterval,
    clearInterval,
    console,
  };
  vm.runInNewContext(workerSource, sandbox, { filename: workerPath });

  async function message(payload) {
    return new Promise(resolve => runtimeListener(payload, {}, resolve));
  }

  async function pair() {
    const result = await message({
      type: 'PAIR_SELECTED_TAB', origin: 'http://localhost:9000', pairingCode: 'one-time-code',
      tabId: tab.id, tabURL: tab.url, tabTitle: 'Public page',
    });
    assert.equal(result.error, undefined, result.error);
    return sockets.at(-1);
  }

  return {
    chrome,
    get sockets() { return sockets; },
    get grantCount() { return grantNumber; },
    get scriptCalls() { return scriptCalls; },
    get tabUpdates() { return tabUpdates; },
    get tabCreateCalls() { return tabCreateCalls; },
    get tabRemoveCalls() { return tabRemoveCalls; },
    get cookieWriteCalls() { return cookieWriteCalls; },
    get scriptTargets() { return scriptTargets; },
    get tabState() { return { ...tab }; },
    get otherTabState() { return { ...otherTab }; },
    get signInState() { return { ...signInState }; },
    get managementCalls() { return managementCalls; },
    get now() { return now; },
    set now(value) { now = value; },
    setQueryOverride(callback) { queryOverride = callback; },
    pair,
    async revoke() { return message({ type: 'REVOKE_SELECTED_TAB' }); },
    async compete() { return message({ type: 'PAIR_SELECTED_TAB', origin: 'http://localhost:9000', pairingCode: 'second-run', tabId: tab.id, tabURL: tab.url, tabTitle: 'Public page' }); },
    sendScopeEscape() { updatedListener(tab.id, { url: 'https://example.test/private/secret' }); },
    sendActivation(tabId) { activatedListener({ tabId }); },
  };
}

test('competing run cannot revoke or replace the active selected-tab lease', async () => {
  const h = createHarness();
  const first = await h.pair();
  const denied = await h.compete();
  assert.match(denied.error, /Another tab share is active/);
  assert.equal(h.sockets.length, 1);
  assert.equal(first.readyState, MockSocket.OPEN);
  await h.revoke();
});

test('temporary socket loss reconnects the same lease within its expiry', async () => {
  const h = createHarness();
  const original = await h.pair();
  original.close();
  const deadline = Date.now() + 3000;
  while (h.sockets.length < 2 && Date.now() < deadline) {
    await new Promise(resolve => setTimeout(resolve, 10));
  }
  assert.equal(h.sockets.length, 2);
  const reconnected = h.sockets[1];
  while (reconnected.readyState !== MockSocket.OPEN && Date.now() < deadline) {
    await new Promise(resolve => setTimeout(resolve, 10));
  }
  assert.equal(reconnected.readyState, MockSocket.OPEN);
  assert.equal(h.grantCount, 1, 'reconnect must not mint a second browser grant');
  assert.deepEqual(reconnected.sent[0], { type: 'lease', lease_receipt: 'lease-1' });
  await h.revoke();
  assert.equal(reconnected.readyState, 3);
});

test('local revoke cancels an admitted command before it reaches the page', async () => {
  const h = createHarness();
  const socket = await h.pair();
  const queryReached = deferred();
  const finishQuery = deferred();
  h.setQueryOverride(() => { queryReached.resolve(); return finishQuery.promise; });
  socket.serverSend({ type: 'act', request_id: 'act-1', action: { type: 'click', selector: '#continue' } });
  await queryReached.promise;
  await h.revoke();
  finishQuery.resolve([{ id: 9, url: 'https://example.test/public/start' }]);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.scriptCalls, 0);
  assert.equal(socket.readyState, 3);
});

test('scope escape revokes a queued action before page script dispatch', async () => {
  const h = createHarness();
  const socket = await h.pair();
  const queryReached = deferred();
  const finishQuery = deferred();
  h.setQueryOverride(() => { queryReached.resolve(); return finishQuery.promise; });
  socket.serverSend({ type: 'act', request_id: 'act-2', action: { type: 'click', selector: '#continue' } });
  await queryReached.promise;
  h.sendScopeEscape();
  finishQuery.resolve([{ id: 9, url: 'https://example.test/private/secret' }]);
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.scriptCalls, 0);
  assert.equal(socket.readyState, 3);
});

test('expired selected-tab lease cannot dispatch an action', async () => {
  const h = createHarness({ grantLifetime: 1000 });
  const socket = await h.pair();
  h.now += 1000;
  socket.serverSend({ type: 'act', request_id: 'act-3', action: { type: 'click', selector: '#continue' } });
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.scriptCalls, 0);
  assert.equal(socket.readyState, 3);
});

test('reports only allowlisted extensions with explicitly sourced active permissions', async () => {
  const h = createHarness({
    extensionAccess: true,
    extensionIDs: ['allowed-extension', 'disabled-extension'],
    inventory: [
      { id: 'allowed-extension', type: 'extension', name: 'Allowed', version: '2.0', enabled: true, permissions: ['storage', 'activeTab'], hostPermissions: ['https://example.test/*'] },
      { id: 'disabled-extension', type: 'extension', name: 'Disabled', version: '1.5', enabled: false, permissions: ['tabs'], hostPermissions: ['<all_urls>'] },
      { id: 'other-extension', type: 'extension', name: 'Other', version: '1.0', enabled: true, permissions: ['tabs'], hostPermissions: ['<all_urls>'] },
      { id: 'bridge-extension', type: 'extension', name: 'Bridge', version: '1.0', enabled: true, permissions: ['management'], hostPermissions: [] },
    ],
  });
  const socket = await h.pair();
  assert.equal(socket.sent.some(message => message.type === 'extension_inventory'), false, 'pairing must not cache a permission snapshot');
  socket.serverSend({ type: 'extension_inventory', request_id: 'inventory-1' });
  await new Promise(resolve => setTimeout(resolve, 0));
  const inventoryMessage = socket.sent.find(message => message.type === 'extension_inventory');
  assert.ok(inventoryMessage, 'inventory should be returned for an explicitly authorized profile');
  assert.equal(inventoryMessage.request_id, 'inventory-1');
  assert.ok(Number.isFinite(Date.parse(inventoryMessage.observed_at)), 'snapshot should carry its browser observation time');
  assert.deepEqual(JSON.parse(JSON.stringify(inventoryMessage.extensions)), [{
    id: 'allowed-extension', name: 'Allowed', version: '2.0', enabled: true,
    permissions: ['storage', 'activeTab'], hostPermissions: ['https://example.test/*'],
    permission_evidence: { status: 'effective', source: 'chrome.management.getAll', scope: 'grant_set_only', per_tab_site_access: 'not_measured' },
  }, {
    id: 'disabled-extension', name: 'Disabled', version: '1.5', enabled: false,
    permissions: [], hostPermissions: [],
    permission_evidence: { status: 'unavailable', source: 'chrome.management.getAll', scope: 'unavailable', per_tab_site_access: 'not_measured' },
  }]);
  await h.revoke();
});

test('scope escape during permission inventory cannot release stale browser data', async () => {
  const reached = deferred();
  let finish;
  const pending = new Promise(resolve => { finish = resolve; });
  const h = createHarness({
    extensionAccess: true,
    extensionIDs: ['allowed-extension'],
    inventory: () => { reached.resolve(); return pending; },
  });
  const socket = await h.pair();
  socket.serverSend({ type: 'extension_inventory', request_id: 'inventory-scope-loss' });
  await reached.promise;
  h.sendScopeEscape();
  finish([{ id: 'allowed-extension', type: 'extension', name: 'Allowed', version: '2.0', enabled: true, permissions: ['storage'], hostPermissions: [] }]);
  await new Promise(resolve => setTimeout(resolve, 0));
  assert.equal(socket.sent.some(message => message.type === 'extension_inventory'), false);
  assert.equal(socket.readyState, 3);
  assert.equal(h.managementCalls, 1);
});

test('expired lease cannot answer a permission inventory request', async () => {
  const h = createHarness({ grantLifetime: 1000, extensionAccess: true, extensionIDs: ['allowed-extension'] });
  const socket = await h.pair();
  h.now += 1000;
  socket.serverSend({ type: 'extension_inventory', request_id: 'inventory-expired' });
  await new Promise(resolve => setTimeout(resolve, 0));
  assert.equal(h.managementCalls, 0);
  assert.equal(socket.sent.some(message => message.type === 'extension_inventory'), false);
  assert.equal(socket.readyState, 3);
});

test('handoff and broker revoke block later input while preserving other tabs and sign-in state', async () => {
  const h = createHarness();
  const socket = await h.pair();
  const originalSignInState = h.signInState;
  const originalOtherTab = h.otherTabState;

  socket.serverSend({ type: 'act', request_id: 'act-before-handoff', action: { type: 'type', selector: '#search', value: 'query' } });
  const actionDeadline = Date.now() + 1000;
  while (h.scriptCalls === 0 && Date.now() < actionDeadline) await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.scriptCalls, 1, 'a scoped selected-tab action should work before handoff');
  assert.deepEqual(h.scriptTargets, [9], 'page script must target only the consented tab');

  socket.serverSend({ type: 'handoff', request_id: 'handoff-1' });
  const handoffDeadline = Date.now() + 1000;
  while (!socket.sent.some(message => message.type === 'handoff_complete') && Date.now() < handoffDeadline) {
    await new Promise(resolve => setImmediate(resolve));
  }
  assert.ok(socket.sent.some(message => message.type === 'handoff_complete'), 'handoff should complete');
  socket.serverSend({ type: 'act', request_id: 'act-after-handoff', action: { type: 'type', selector: '#search', value: 'blocked' } });
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(h.scriptCalls, 1, 'observe-only handoff must deny subsequent input');

  socket.serverSend({ type: 'revoke' });
  const revokeDeadline = Date.now() + 1000;
  while (socket.readyState !== 3 && Date.now() < revokeDeadline) await new Promise(resolve => setImmediate(resolve));
  assert.equal(socket.readyState, 3, 'broker revoke should close the selected-tab lease');
  socket.serverSend({ type: 'act', request_id: 'act-after-revoke', action: { type: 'type', selector: '#search', value: 'blocked' } });
  await new Promise(resolve => setImmediate(resolve));

  assert.equal(h.scriptCalls, 1, 'revoked lease must not dispatch later input');
  assert.equal(h.tabCreateCalls, 0);
  assert.equal(h.tabRemoveCalls, 0);
  assert.equal(h.cookieWriteCalls, 0);
  assert.deepEqual(h.otherTabState, originalOtherTab, 'unrelated tab state must remain intact');
  assert.deepEqual(h.signInState, originalSignInState, 'sign-in state must remain intact');
});
