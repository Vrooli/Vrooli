const PAIR_PATH = '/api/v1/browser/bridge/pair';
const HEARTBEAT_MS = 20000;
let active = null;

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'PAIR_SELECTED_TAB') {
    let createdLease = null;
    pairSelectedTab(message, lease => { createdLease = lease; }).then(sendResponse).catch(async error => {
      // A competing run must not revoke the share that caused pairing to be
      // rejected. Clean up only a lease created by this pairing attempt.
      if (createdLease && active === createdLease) await revoke();
      sendResponse({ error: safeError(error) });
    });
    return true;
  }
  if (message?.type === 'REVOKE_SELECTED_TAB') {
    revoke().then(sendResponse).catch(error => sendResponse({ error: safeError(error) }));
    return true;
  }
});

chrome.tabs.onUpdated.addListener((tabId, changeInfo) => {
  if (!active || tabId !== active.tabId || !changeInfo.url) return;
  if (!isAllowed(active.profile, changeInfo.url)) {
    send({ type: 'scope_violation', tabId, observedURL: changeInfo.url });
    revoke();
  }
});

chrome.tabs.onActivated.addListener(info => {
  if (active && info.tabId !== active.tabId) revoke();
});

async function pairSelectedTab({ origin, pairingCode, tabId, tabURL, tabTitle }, onLeaseCreated = () => {}) {
  if (active) throw new Error('Another tab share is active. Revoke it before sharing a different tab.');
  if (!Number.isInteger(tabId) || !/^https?:\/\//i.test(tabURL) || !isLocalOrConfiguredOrigin(origin)) throw new Error('Invalid selected tab or bridge origin.');
  const response = await fetch(`${origin}${PAIR_PATH}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ pairing_code: pairingCode, tab: { id: tabId, url: tabURL, title: tabTitle } }),
    cache: 'no-store',
  });
  if (!response.ok) throw new Error(`Vrooli rejected the pairing request (${response.status}).`);
  const grant = await response.json();
  if (!grant.session_id || !grant.websocket_url || !grant.lease_receipt || !grant.permission_profile || !grant.expires_at) throw new Error('Vrooli returned an incomplete tab grant.');
  const expiresAt = Date.parse(grant.expires_at);
  if (!Number.isFinite(expiresAt) || expiresAt <= Date.now()) throw new Error('Vrooli returned an expired tab grant.');
  if (!['ws:', 'wss:'].includes(new URL(grant.websocket_url).protocol) || new URL(grant.websocket_url).origin !== origin.replace(/^http/, 'ws')) throw new Error('Vrooli returned a websocket outside the approved origin.');
  if (!isAllowed(grant.permission_profile, tabURL)) throw new Error('The selected tab is outside the granted URL scope.');
  active = { tabId, origin, socket: null, websocketURL: grant.websocket_url, profile: grant.permission_profile, sessionId: grant.session_id, leaseReceipt: grant.lease_receipt, expiresAt, heartbeat: null, reconnectAttempts: 0, reconnectTimer: null, revoking: false, commandQueue: Promise.resolve() };
  const lease = active;
  onLeaseCreated(lease);
  lease.expiryTimer = setTimeout(() => {
    if (active === lease && Date.now() >= lease.expiresAt) revoke();
  }, Math.max(0, lease.expiresAt - Date.now()));
  await connectActiveSocket();
  active.heartbeat = setInterval(() => send({ type: 'heartbeat', session_id: active?.sessionId }), HEARTBEAT_MS);
  return { ok: true, sessionId: active.sessionId };
}

async function handleCommand(event, lease = active) {
	if (active !== lease) return;
	if (!leaseIsLive(lease)) return revokeLease(lease);
  let command;
  try { command = JSON.parse(event.data); } catch { return send({ type: 'error', error: 'invalid_json' }); }
  if (command.type === 'lease_accepted' || command.type === 'lease_denied') return;
  if (command.type === 'revoke') return revoke();
  if (command.type === 'scope_violation') return revoke();
  if (command.type === 'handoff') {
    lease.role = 'observe';
    return send({ type: 'handoff_complete', request_id: command.request_id, role: lease.role });
  }
  if (command.type === 'scope_check') {
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    if (!leaseIsLive(lease) || tab?.id !== lease.tabId || !isAllowed(lease.profile, tab.url || '')) return revokeLease(lease);
    return send({ type: 'scope_checked', request_id: command.request_id, url: tab.url });
  }
  if (command.type === 'extension_inventory') return reportExtensionInventory(command.request_id, lease);
  if (command.type === 'observe') return observe(command.request_id, lease);
  if (command.type === 'act') return act(command, lease);
  send({ type: 'error', request_id: command.request_id, error: 'unsupported_operation' });
}

async function reportExtensionInventory(requestId, lease = active) {
  if (!leaseIsLive(lease)) return revokeLease(lease);
  const extensionIDs = lease.profile.extension_ids || [];
  if (!lease.profile.extension_access || extensionIDs.length === 0) {
    return send({ type: 'denied', request_id: requestId, reason: 'extension_inventory_not_allowed' });
  }
  const [selected] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!leaseIsLive(lease) || selected?.id !== lease.tabId || !isAllowed(lease.profile, selected.url || '')) return revokeLease(lease);
  let inventory;
  try {
    inventory = await chrome.management.getAll();
  } catch (error) {
    return send({ type: 'error', request_id: requestId, error: safeError(error) });
  }
  if (!leaseIsLive(lease)) return revokeLease(lease);
  const [rechecked] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!leaseIsLive(lease) || rechecked?.id !== lease.tabId || !isAllowed(lease.profile, rechecked.url || '')) return revokeLease(lease);
  const allowedExtensionIDs = new Set(extensionIDs);
  const extensions = inventory.filter(item => item.id !== chrome.runtime.id && item.type === 'extension' && allowedExtensionIDs.has(item.id)).map(item => ({
    id: item.id,
    name: item.name,
    version: item.version || '',
    enabled: item.enabled,
    permissions: item.enabled ? (item.permissions || []) : [],
    hostPermissions: item.enabled ? (item.hostPermissions || []) : [],
    permission_evidence: {
      status: item.enabled ? 'effective' : 'unavailable',
      source: 'chrome.management.getAll',
      scope: item.enabled ? 'grant_set_only' : 'unavailable',
      per_tab_site_access: 'not_measured',
    },
  }));
  if (!leaseIsLive(lease)) return revokeLease(lease);
  return send({ type: 'extension_inventory', request_id: requestId, observed_at: new Date().toISOString(), extensions });
}

async function observe(requestId, lease = active) {
	if (active !== lease) return;
	if (!leaseIsLive(lease)) return revokeLease(lease);
  if (!lease.profile.view) return send({ type: 'denied', request_id: requestId, reason: 'view_disabled' });
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!leaseIsLive(lease) || tab?.id !== lease.tabId || !isAllowed(lease.profile, tab.url || '')) return revokeLease(lease);
  const [{ result }] = await chrome.scripting.executeScript({ target: { tabId: lease.tabId }, func: () => ({ title: document.title, url: location.href, text: document.body?.innerText?.slice(0, 20000) || '' }) });
  if (!leaseIsLive(lease)) return;
  send({ type: 'observed', request_id: requestId, selected_tab: result });
}

async function act(command, lease = active) {
	if (active !== lease) return;
	if (!leaseIsLive(lease)) return revokeLease(lease);
  const profile = lease.profile;
  if (!profile?.control || profile.effect === 'observe' || lease.role === 'observe') return send({ type: 'denied', request_id: command.request_id, reason: 'control_disabled' });
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!leaseIsLive(lease) || tab?.id !== lease.tabId || !isAllowed(profile, tab.url || '')) return revokeLease(lease);
  const kind = command.action?.type;
  if (kind === 'navigate') {
    if (!isAllowed(profile, command.action.url)) return send({ type: 'denied', request_id: command.request_id, reason: 'url_scope' });
    if (!leaseIsLive(lease)) return revokeLease(lease);
    await chrome.tabs.update(lease.tabId, { url: command.action.url });
  } else if (kind === 'click' || kind === 'type') {
    const selector = String(command.action.selector || '');
    if (!selector || selector.length > 2048) return send({ type: 'denied', request_id: command.request_id, reason: 'selector_invalid' });
    if (!leaseIsLive(lease)) return revokeLease(lease);
    const result = await chrome.scripting.executeScript({
      target: { tabId: lease.tabId },
      func: (selector, kind, value) => {
        const element = document.querySelector(selector);
        if (!element) return { ok: false, reason: 'element_not_found' };
        if (kind === 'click') { element.click(); return { ok: true }; }
        if (element instanceof HTMLInputElement && element.type === 'password') return { ok: false, reason: 'credential_authority_required' };
        if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element.isContentEditable)) return { ok: false, reason: 'element_not_editable' };
        if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
          const setter = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(element), 'value')?.set;
          setter?.call(element, String(value ?? ''));
        } else element.textContent = String(value ?? '');
        element.dispatchEvent(new Event('input', { bubbles: true }));
        element.dispatchEvent(new Event('change', { bubbles: true }));
        return { ok: true };
      },
      args: [selector, kind, String(command.action.value ?? '').slice(0, 100000)],
    });
    if (!result[0]?.result?.ok) return send({ type: 'denied', request_id: command.request_id, reason: result[0]?.result?.reason || 'action_failed' });
  } else return send({ type: 'denied', request_id: command.request_id, reason: 'unsupported_action' });
  if (!leaseIsLive(lease)) return;
  send({ type: 'action_complete', request_id: command.request_id });
}

function leaseIsLive(lease) {
  return !!lease && active === lease && !lease.revoking && Date.now() < lease.expiresAt;
}

function revokeLease(lease) {
  if (active === lease) return revoke();
}

async function revoke() {
  if (!active) return { ok: true };
  active.revoking = true;
  send({ type: 'revoke' });
  const socket = active.socket;
  clearActive(false);
  socket?.close(1000, 'user revoked tab share');
  return { ok: true };
}

function send(value) {
  const lease = active;
  if (!lease) return false;
  if (value.type !== 'revoke' && Date.now() >= lease.expiresAt) {
    revoke();
    return false;
  }
  if (lease.socket?.readyState === WebSocket.OPEN) {
    lease.socket.send(JSON.stringify(value));
    return true;
  }
  return false;
}

function clearActive(notify = true) {
  if (!active) return;
  if (notify) send({ type: 'revoke' });
  clearInterval(active.heartbeat);
  clearTimeout(active.reconnectTimer);
  clearTimeout(active.expiryTimer);
  active = null;
}

async function connectActiveSocket() {
  const current = active;
  if (!current || current.revoking || Date.now() >= current.expiresAt) return clearActive(false);
  const [selected] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!selected || selected.id !== current.tabId || !isAllowed(current.profile, selected.url || '')) {
    await revoke();
    throw new Error('The selected tab changed or left the permitted URL scope.');
  }
  const socket = new WebSocket(current.websocketURL, ['vrooli-browser-bridge-v1']);
  current.socket = socket;
  socket.addEventListener('message', event => {
    let message;
    try { message = JSON.parse(event.data); } catch { message = null; }
    if (message?.type === 'revoke' || message?.type === 'scope_violation') {
      if (active === current) revoke();
      return;
    }
    if (active !== current) return;
    current.commandQueue = current.commandQueue.then(() => handleCommand(event, current)).catch(async () => {
      if (active === current) await revoke();
    });
  });
  socket.addEventListener('close', () => onSocketClosed(socket));
  try {
    await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timed out connecting to the Vrooli bridge socket.')), 5000);
      socket.addEventListener('open', () => { clearTimeout(timeout); resolve(); }, { once: true });
      socket.addEventListener('error', () => { clearTimeout(timeout); reject(new Error('Could not connect to the Vrooli bridge socket.')); }, { once: true });
    });
    if (active !== current || current.revoking || Date.now() >= current.expiresAt) {
      socket.close(1000, 'tab share revoked during connection');
      throw new Error('The selected-tab share was revoked before the lease could connect.');
    }
    await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Vrooli did not acknowledge the tab lease.')), 5000);
      const confirmLease = event => {
        let message;
        try { message = JSON.parse(event.data); } catch { return; }
        if (message.type === 'lease_accepted') {
          clearTimeout(timeout);
          socket.removeEventListener('message', confirmLease);
          resolve();
        } else if (message.type === 'lease_denied') {
          clearTimeout(timeout);
          socket.removeEventListener('message', confirmLease);
          reject(new Error('Vrooli denied the selected-tab lease.'));
        }
      };
      socket.addEventListener('message', confirmLease);
      socket.send(JSON.stringify({ type: 'lease', lease_receipt: current.leaseReceipt }));
    });
    if (active !== current || current.revoking || Date.now() >= current.expiresAt) {
      socket.close(1000, 'tab share revoked during lease confirmation');
      throw new Error('The selected-tab share was revoked before the lease was confirmed.');
    }
    current.reconnectAttempts = 0;
  } catch (error) {
    if (active === current && current.socket === socket) current.socket = null;
    socket.close(1000, 'bridge handshake failed');
    throw error;
  }
}

function onSocketClosed(socket) {
  if (!active || active.socket !== socket || active.revoking) return;
  active.socket = null;
  if (Date.now() >= active.expiresAt || active.reconnectAttempts >= 5) return clearActive(false);
  const delay = Math.min(1000 * 2 ** active.reconnectAttempts, 10000);
  active.reconnectAttempts++;
  active.reconnectTimer = setTimeout(() => {
    connectActiveSocket().catch(() => onReconnectFailed());
  }, delay);
}

function onReconnectFailed() {
  if (!active || active.revoking) return;
  if (active.reconnectAttempts >= 5 || Date.now() >= active.expiresAt) return clearActive(false);
  const delay = Math.min(1000 * 2 ** active.reconnectAttempts, 10000);
  active.reconnectAttempts++;
  active.reconnectTimer = setTimeout(() => connectActiveSocket().catch(onReconnectFailed), delay);
}

function isAllowed(profile, raw) {
  let candidate;
  try { candidate = new URL(raw); } catch { return false; }
  if (!['https:', 'http:'].includes(candidate.protocol) || candidate.username || candidate.password) return false;
  if ((profile.url_deny || []).some(pattern => matches(pattern, candidate))) return false;
  return (profile.url_allow || []).some(pattern => matches(pattern, candidate));
}

function matches(pattern, candidate) {
  const rule = /^(https?):\/\/([^/]+)(\/.*)?$/i.exec(pattern.trim());
  if (!rule || `${rule[1].toLowerCase()}:` !== candidate.protocol) return false;
  const hostPattern = rule[2].toLowerCase();
  const host = candidate.hostname.toLowerCase();
  if (hostPattern.startsWith('*.')) {
    const suffix = hostPattern.slice(1);
    if (!host.endsWith(suffix) || host === suffix.slice(1)) return false;
  } else if (host !== hostPattern) return false;
  const path = rule[3] || '/*';
  if (path === '/*') return true;
  return candidate.pathname.startsWith(path.endsWith('*') ? path.slice(0, -1) : path);
}

function isLocalOrConfiguredOrigin(raw) {
  try {
    const value = new URL(raw);
    return value.protocol === 'https:' || (value.protocol === 'http:' && ['localhost', '127.0.0.1', '[::1]'].includes(value.hostname));
  } catch { return false; }
}

function safeError(error) { return error instanceof Error ? error.message : 'unknown bridge error'; }
