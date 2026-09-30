const send = (type, data = {}) => new Promise((resolve) => {
  chrome.runtime.sendMessage({ type, ...data }, (response) => resolve(response || { ok: false, error: chrome.runtime.lastError?.message || 'No response' }));
});

const status = document.getElementById('status');
const origin = document.getElementById('origin');
const fieldSelect = document.getElementById('field-select');
const accountSelect = document.getElementById('account-select');
const candidate = document.getElementById('candidate');
let currentOrigin = '';

function show(message) { status.textContent = message; }

async function refresh() {
  const response = await send('STATUS');
  if (!response.ok) return show(response.error);
  currentOrigin = response.origin;
  origin.textContent = `${response.origin} · ${response.approved ? 'approved' : 'not approved'} · ${response.connected ? 'connected' : 'not connected'}`;
  candidate.textContent = response.saveCandidate ? 'A login form was submitted. Review the visible fields, then choose Save or Update.' : '';
}

document.getElementById('workspace').addEventListener('change', async (event) => {
  const response = await send('SET_WORKSPACE', { workspaceId: event.target.value.trim() });
  show(response.ok ? `Workspace set to ${response.workspaceId}` : response.error);
});

document.getElementById('connect').addEventListener('click', async () => {
  const token = document.getElementById('owner-token').value;
  const response = await send('ENROLL', { ownerToken: token });
  document.getElementById('owner-token').value = '';
  show(response.ok ? `Enrolled for ${response.origin}; token retained only in memory` : response.error);
  await refresh();
});

document.getElementById('approve').addEventListener('click', async () => {
  const response = await send('APPROVE_ORIGIN', { origin: currentOrigin });
  show(response.ok ? `Approved ${response.origin}` : response.error);
  await refresh();
});

document.getElementById('discover').addEventListener('click', async () => {
  const response = await send('DISCOVER');
  fieldSelect.replaceChildren();
  if (!response.ok) return show(response.error);
  for (const field of response.fields) {
    const option = document.createElement('option');
    option.value = field.id;
    option.textContent = `${field.label} (${field.type})`;
    fieldSelect.append(option);
  }
  show(`${response.fields.length} visible candidate fields found`);
});

document.getElementById('discover-accounts').addEventListener('click', async () => {
  const response = await send('DISCOVER_ACCOUNTS');
  accountSelect.replaceChildren();
  if (!response.ok) return show(response.error);
  for (const item of response.items) {
    const option = document.createElement('option');
    option.value = JSON.stringify(item);
    option.textContent = `${item.name}${item.username ? ` · ${item.username}` : ''}`;
    accountSelect.append(option);
  }
  show(`${response.items.length} approved account${response.items.length === 1 ? '' : 's'} found`);
});

accountSelect.addEventListener('change', () => {
  const item = JSON.parse(accountSelect.selectedOptions[0]?.value || '{}');
  document.getElementById('grant-id').value = item.grant_id || '';
  document.getElementById('item-id').value = item.item_id || '';
  document.getElementById('revision').value = item.revision || 0;
});

document.getElementById('fill').addEventListener('click', async () => {
  const selected = fieldSelect.selectedOptions[0];
  const response = await send('FILL', {
    origin: currentOrigin,
    fieldId: selected?.value,
    grantId: document.getElementById('grant-id').value.trim(),
    itemId: document.getElementById('item-id').value.trim(),
    revision: Number(document.getElementById('revision').value),
    field: document.getElementById('authority-field').value
  });
  show(response.ok ? 'Selected field filled; no form was submitted' : response.error);
});

document.getElementById('generate').addEventListener('click', async () => {
  const response = await send('GENERATE');
  show(response.ok ? 'Generated password filled into the selected visible field; review it before saving' : response.error);
});

document.getElementById('save').addEventListener('click', async () => {
  const response = await send('SAVE', {
    vaultId: document.getElementById('vault-id').value.trim() || 'personal',
    name: document.getElementById('save-name').value.trim()
  });
  show(response.ok ? `Saved ${response.item?.name || 'credential'}; secrets were sent only to the authority` : response.error);
  if (response.ok) await refresh();
});

document.getElementById('update').addEventListener('click', async () => {
  const item = JSON.parse(accountSelect.selectedOptions[0]?.value || '{}');
  if (!item.item_id) return show('Select an approved account before updating');
  const response = await send('UPDATE', {
    vaultId: item.vault_id || document.getElementById('vault-id').value.trim() || 'personal',
    name: document.getElementById('save-name').value.trim() || item.name,
    itemId: item.item_id,
    revision: Number(item.revision)
  });
  show(response.ok ? `Updated ${response.item?.name || item.name}; revision conflict protection applied` : response.error);
  if (response.ok) await refresh();
});

document.getElementById('revoke').addEventListener('click', async () => {
  const response = await send('REVOKE', { revokeEnrollment: true });
  show(response.ok ? 'Native session revoked' : response.error);
  await refresh();
});

refresh();
