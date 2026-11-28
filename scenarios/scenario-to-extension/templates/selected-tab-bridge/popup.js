const originInput = document.querySelector('#origin');
const codeInput = document.querySelector('#pairing-code');
const statusNode = document.querySelector('#status');
const inventoryNode = document.querySelector('#inventory');

chrome.storage.local.get(['bridgeOrigin'], values => {
  originInput.value = values.bridgeOrigin || '';
});

chrome.management.getAll(items => {
  if (chrome.runtime.lastError) {
    inventoryNode.textContent = `Extension inventory unavailable: ${chrome.runtime.lastError.message}`;
    return;
  }
  const others = items.filter(item => item.id !== chrome.runtime.id && item.type === 'extension');
  inventoryNode.replaceChildren(...others.map(item => {
    const row = document.createElement('p');
    const permissions = [...(item.permissions || []), ...(item.hostPermissions || [])];
    row.textContent = item.enabled
      ? `${item.name} · enabled · Chrome-reported active permissions: ${permissions.join(', ') || 'none'}`
      : `${item.name} · disabled · active permission state unavailable while disabled`;
    return row;
  }));
});

document.querySelector('#share').addEventListener('click', async () => {
  statusNode.textContent = 'Checking the selected tab and bridge permission…';
  const origin = normalizeOrigin(originInput.value);
  if (!origin) return setStatus('Enter an http://localhost, http://127.0.0.1, or configured HTTPS Vrooli bridge origin.');
  const permission = await chrome.permissions.request({ origins: [`${origin}/*`] });
  if (!permission) return setStatus('The bridge host permission was not granted.');
  const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
  if (!tab?.id || !tab.url || !/^https?:\/\//i.test(tab.url)) return setStatus('The selected tab must be an HTTP or HTTPS page.');
  const pairingCode = codeInput.value.trim();
  if (!pairingCode) return setStatus('Enter the one-time code shown by Vrooli.');
  chrome.storage.local.set({ bridgeOrigin: origin });
  const reply = await chrome.runtime.sendMessage({ type: 'PAIR_SELECTED_TAB', origin, pairingCode, tabId: tab.id, tabURL: tab.url, tabTitle: tab.title || '' });
  setStatus(reply?.error || (reply?.ok ? `Shared ${tab.title || tab.url}. Session ${reply.sessionId}.` : 'Bridge pairing failed.'));
  codeInput.value = '';
});

document.querySelector('#revoke').addEventListener('click', async () => {
  const reply = await chrome.runtime.sendMessage({ type: 'REVOKE_SELECTED_TAB' });
  setStatus(reply?.error || (reply?.ok ? 'Tab share revoked and the bridge disconnected.' : 'No active tab share.'));
});

function normalizeOrigin(raw) {
  try {
    const parsed = new URL(raw.trim());
    if (!['http:', 'https:'].includes(parsed.protocol) || parsed.username || parsed.password || parsed.pathname !== '/' || parsed.search || parsed.hash) return '';
    if (parsed.protocol === 'http:' && !['localhost', '127.0.0.1', '[::1]'].includes(parsed.hostname)) return '';
    return parsed.origin;
  } catch { return ''; }
}

function setStatus(message) { statusNode.textContent = message; }
