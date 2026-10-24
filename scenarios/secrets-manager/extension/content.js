const documentId = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`;
const fields = new Map();
let nextFieldID = 0;

function visible(input) {
  if (!(input instanceof HTMLInputElement)) return false;
  if (input.disabled || input.readOnly || input.type === 'hidden') return false;
  const style = window.getComputedStyle(input);
  return style.display !== 'none' && style.visibility !== 'hidden' && input.getClientRects().length > 0;
}

function classify(input) {
  const hints = `${input.type} ${input.name} ${input.id} ${input.autocomplete} ${input.placeholder}`.toLowerCase();
  if (input.autocomplete === 'one-time-code' || hints.includes('totp') || hints.includes('otp')) return 'totp';
  if (input.type === 'password' || hints.includes('password')) return 'password';
  if (input.type === 'email' || hints.includes('email') || hints.includes('user') || hints.includes('login')) return 'username';
  return null;
}

function discover() {
  fields.clear();
  nextFieldID = 0;
  const result = [];
  for (const input of document.querySelectorAll('input')) {
    if (!visible(input)) continue;
    const type = classify(input);
    if (!type) continue;
    const id = `field-${++nextFieldID}`;
    fields.set(id, input);
    result.push({ id, type, label: input.getAttribute('aria-label') || input.name || input.id || type, autocomplete: input.autocomplete || '' });
  }
  return { documentId, fields: result };
}

function applyFill(message) {
  if (message.origin !== location.origin || message.documentId !== documentId) return { filled: false, error: 'origin or document changed' };
  const input = fields.get(message.fieldId);
  if (!input || !visible(input) || !['password', 'username', 'totp'].includes(message.field)) return { filled: false, error: 'field is stale or not fillable' };
  if (typeof message.credential !== 'string' || message.credential.length > 32768) return { filled: false, error: 'credential is invalid' };
  const descriptor = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value');
  descriptor?.set?.call(input, message.credential);
  input.dispatchEvent(new Event('input', { bubbles: true, composed: true }));
  input.dispatchEvent(new Event('change', { bubbles: true, composed: true }));
  return { filled: true };
}

function collectLogin() {
  discover();
  const values = {};
  for (const [id, input] of fields) {
    const type = classify(input);
    if (typeof input.value !== 'string' || input.value.length > 32768) continue;
    if (type === 'username' && !values.username) values.username = input.value;
    if (type === 'password' && !values.password) values.password = input.value;
    if (type === 'totp' && !values.totp) values.totp = input.value;
  }
  if (!values.password) return { documentId, values, error: 'Enter a password before saving' };
  return { documentId, values };
}

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message?.type === 'DISCOVER_FIELDS') sendResponse(discover());
  else if (message?.type === 'COLLECT_LOGIN') sendResponse(collectLogin());
  else if (message?.type === 'APPLY_FILL') sendResponse(applyFill(message));
  return false;
});

document.addEventListener('submit', (event) => {
  if (!(event.target instanceof HTMLFormElement)) return;
  const hasLoginField = Array.from(event.target.querySelectorAll('input')).some((input) => classify(input) === 'password');
  if (hasLoginField) chrome.runtime.sendMessage({ type: 'FORM_SUBMITTED', documentId });
}, true);
