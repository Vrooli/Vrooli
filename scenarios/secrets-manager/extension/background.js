const NATIVE_HOST = 'com.vrooli.secrets-manager';
const PROTOCOL = 'vrooli.secrets.native/1';
const API_ENDPOINT = '{{API_ENDPOINT}}';
const EXTENSION_ID = chrome.runtime.id;

let nativePort = null;
let ownerToken = '';
let workspaceId = 'local';
let enrolledOrigin = '';
let nativeSession = null;
let pendingSave = null;

function randomID(prefix) {
  const suffix = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  return `${prefix}-${suffix}`;
}

function originFromURL(raw) {
  const parsed = new URL(raw);
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') throw new Error('Only HTTP(S) origins are supported');
  return parsed.origin;
}

function nativeScope(tab, documentId, itemId = 'metadata', revision = 0) {
  return {
    origin: originFromURL(tab.url),
    tab_id: String(tab.id),
    frame_id: 'frame-0',
    document_id: documentId || 'extension-control',
    item_id: itemId,
    item_revision: revision
  };
}

function ensureNativePort() {
  if (nativePort) return nativePort;
  nativePort = chrome.runtime.connectNative(NATIVE_HOST);
  nativePort.onDisconnect.addListener(() => {
    nativePort = null;
    nativeSession = null;
  });
  return nativePort;
}

function nativeRequest(operation, scope, payload) {
  return new Promise((resolve, reject) => {
    const port = ensureNativePort();
    const requestId = randomID('request');
    const listener = (response) => {
      if (response?.request_id !== requestId) return;
      port.onMessage.removeListener(listener);
      port.onDisconnect.removeListener(disconnect);
      if (!response.ok) reject(new Error(response.message || response.code || 'Native host rejected the request'));
      else resolve(response);
    };
    const disconnect = () => {
      port.onMessage.removeListener(listener);
      reject(new Error('Native messaging host disconnected'));
    };
    port.onMessage.addListener(listener);
    port.onDisconnect.addListener(disconnect);
    try {
      port.postMessage({
        protocol: PROTOCOL,
        request_id: requestId,
        operation,
        extension_id: EXTENSION_ID,
        session_id: nativeSession?.sessionId || '',
        nonce: randomID('nonce'),
        expires_at: Math.floor(Date.now() / 1000) + 90,
        scope,
        payload
      });
    } catch (error) {
      port.onMessage.removeListener(listener);
      port.onDisconnect.removeListener(disconnect);
      reject(error);
    }
  });
}

async function approvedOrigins() {
  const stored = await chrome.storage.local.get({ approvedOrigins: [] });
  return Array.isArray(stored.approvedOrigins) ? stored.approvedOrigins : [];
}

async function isApproved(origin) {
  return (await approvedOrigins()).includes(origin);
}

async function currentTab() {
  const tabs = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tabs[0]?.id || !tabs[0].url) throw new Error('No supported active tab is available');
  originFromURL(tabs[0].url);
  return tabs[0];
}

async function discover(tab) {
  return chrome.tabs.sendMessage(tab.id, { type: 'DISCOVER_FIELDS' });
}

async function collectLogin(tab) {
  const response = await chrome.tabs.sendMessage(tab.id, { type: 'COLLECT_LOGIN' });
  if (!response?.documentId) throw new Error('The current page did not provide a login form');
  return response;
}

async function discoverAccounts(tab) {
  if (!(await isApproved(originFromURL(tab.url)))) throw new Error('Approve this origin before account discovery');
  if (!ownerToken) throw new Error('Connect the owner session before account discovery');
  const response = await nativeRequest('metadata', nativeScope(tab, 'extension-control'), {
    workspace_id: workspaceId,
    session_token: ownerToken
  });
  return { items: Array.isArray(response.metadata) ? response.metadata : [] };
}

async function enroll(tab, token) {
  if (!token) throw new Error('Enter the owner session token for this browser session');
  const scope = nativeScope(tab, 'extension-control');
  await nativeRequest('enroll', scope, { workspace_id: workspaceId, enrollment_token: token });
  ownerToken = token;
  enrolledOrigin = scope.origin;
  return { origin: enrolledOrigin, workspaceId };
}

async function fill(tab, input) {
  const origin = originFromURL(tab.url);
  if (origin !== input.origin || !(await isApproved(origin))) throw new Error('This origin is not approved for filling');
  if (!ownerToken) throw new Error('Connect the owner session before filling');
  const candidates = await discover(tab);
  const candidate = candidates?.fields?.find((field) => field.id === input.fieldId);
  if (!candidate) throw new Error('The selected field is stale; rediscover the current page');
  const scope = nativeScope(tab, candidates.documentId, input.itemId, input.revision);
  const payload = {
    workspace_id: workspaceId,
    grant_id: input.grantId,
    field: input.field,
    session_token: ownerToken
  };
  // A new fill starts a fresh authority session for the current document.
  nativeSession = null;
  const unlockResponse = await nativeRequest('unlock', scope, payload);
  nativeSession = { sessionId: unlockResponse.session_id, scope };
  const response = await nativeRequest('fill', scope, { field: input.field });
  const credential = response.payload?.credential;
  if (!credential) throw new Error('The authority returned no credential');
  let applied = false;
  try {
    const result = await chrome.tabs.sendMessage(tab.id, {
      type: 'APPLY_FILL',
      origin,
      documentId: candidates.documentId,
      fieldId: input.fieldId,
      field: input.field,
      credential
    });
    if (!result?.filled) throw new Error(result?.error || 'The page rejected the selected field');
    applied = true;
    return { filled: true, origin, documentId: candidates.documentId };
  } finally {
    // Keep the opaque session in memory so the user can explicitly revoke it.
    // Failed or rejected fills are immediately discarded and remain fail-closed.
    if (!applied) nativeSession = null;
  }
}

async function writeItem(tab, input, update) {
  const origin = originFromURL(tab.url);
  if (!(await isApproved(origin))) throw new Error('Approve this origin before saving credentials');
  if (!ownerToken) throw new Error('Connect the owner session before saving credentials');
  const login = await collectLogin(tab);
  const selected = input.itemId || 'new-item';
  const scope = nativeScope(tab, login.documentId, selected, update ? Number(input.revision) : 0);
  nativeSession = null;
  const response = await nativeRequest(update ? 'update' : 'save', scope, {
    workspace_id: workspaceId,
    vault_id: input.vaultId || 'personal',
    name: input.name || new URL(tab.url).hostname,
    username: login.values.username || '',
    uri: origin,
    password: login.values.password || '',
    totp: login.values.totp || '',
    session_token: ownerToken
  });
  pendingSave = null;
  return { item: response.metadata?.[0] || null, documentId: login.documentId };
}

function generatedPassword(length = 24) {
  const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*';
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (value) => alphabet[value % alphabet.length]).join('');
}

async function generateAndFill(tab) {
  const origin = originFromURL(tab.url);
  if (!(await isApproved(origin))) throw new Error('Approve this origin before generating a password');
  const candidates = await discover(tab);
  const passwordField = candidates?.fields?.find((field) => field.type === 'password');
  if (!passwordField) throw new Error('No visible password field was found');
  const credential = generatedPassword();
  const result = await chrome.tabs.sendMessage(tab.id, {
    type: 'APPLY_FILL', origin, documentId: candidates.documentId, fieldId: passwordField.id, field: 'password', credential
  });
  if (!result?.filled) throw new Error(result?.error || 'The page rejected the generated password');
  return { generated: true, documentId: candidates.documentId };
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  (async () => {
    try {
      switch (message?.type) {
        case 'STATUS': {
          const tab = await currentTab();
          const origin = originFromURL(tab.url);
          return { ok: true, origin, approved: await isApproved(origin), connected: Boolean(ownerToken), saveCandidate: Boolean(pendingSave?.tabId === tab.id && pendingSave.origin === origin), apiEndpoint: API_ENDPOINT };
        }
        case 'SET_WORKSPACE':
          if (typeof message.workspaceId !== 'string' || !/^[A-Za-z0-9._-]{1,128}$/.test(message.workspaceId)) throw new Error('Workspace ID is invalid');
          workspaceId = message.workspaceId;
          return { ok: true, workspaceId };
        case 'ENROLL': {
          const tab = await currentTab();
          return { ok: true, ...(await enroll(tab, message.ownerToken)) };
        }
        case 'APPROVE_ORIGIN': {
          const origin = originFromURL(message.origin);
          const origins = await approvedOrigins();
          if (!origins.includes(origin)) await chrome.storage.local.set({ approvedOrigins: [...origins, origin] });
          return { ok: true, origin };
        }
        case 'DISCOVER': {
          const tab = await currentTab();
          return { ok: true, ...(await discover(tab)) };
        }
        case 'DISCOVER_ACCOUNTS': {
          const tab = await currentTab();
          return { ok: true, ...(await discoverAccounts(tab)) };
        }
        case 'FILL': {
          const tab = await currentTab();
          return { ok: true, ...(await fill(tab, message)) };
        }
        case 'SAVE': {
          const tab = await currentTab();
          return { ok: true, ...(await writeItem(tab, message, false)) };
        }
        case 'UPDATE': {
          const tab = await currentTab();
          return { ok: true, ...(await writeItem(tab, message, true)) };
        }
        case 'GENERATE': {
          const tab = await currentTab();
          return { ok: true, ...(await generateAndFill(tab)) };
        }
        case 'FORM_SUBMITTED':
          if (sender.tab?.id && sender.tab.url) {
            pendingSave = { tabId: sender.tab.id, origin: originFromURL(sender.tab.url), documentId: message.documentId };
          }
          return { ok: true, candidate: true };
        case 'REVOKE': {
          if (nativeSession) {
            await nativeRequest('revoke', nativeSession.scope, { revoke_enrollment: message.revokeEnrollment ? 'true' : 'false' });
            nativeSession = null;
          }
          ownerToken = '';
          enrolledOrigin = '';
          return { ok: true };
        }
        default:
          throw new Error('Unsupported extension operation');
      }
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : String(error) };
    }
  })().then(sendResponse);
  return true;
});

chrome.runtime.onSuspend.addListener(() => {
  ownerToken = '';
  nativeSession = null;
  pendingSave = null;
  nativePort?.disconnect();
  nativePort = null;
});
