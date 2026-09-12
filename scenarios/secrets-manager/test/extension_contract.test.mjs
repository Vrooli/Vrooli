import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..', 'extension');
const read = (name) => readFileSync(resolve(root, name), 'utf8');

test('extension source is syntactically valid JavaScript', () => {
  for (const file of ['background.js', 'content.js', 'popup.js']) {
    execFileSync(process.execPath, ['--check', resolve(root, file)]);
  }
});

test('manifest requires native messaging and avoids automatic submission', () => {
  const manifest = JSON.parse(read('manifest.json'));
  assert.equal(manifest.manifest_version, 3);
  assert.ok(manifest.permissions.includes('nativeMessaging'));
  assert.ok(manifest.background?.service_worker);
  assert.ok(manifest.content_scripts?.length);
  assert.ok(!read('content.js').includes('.submit('));
});

test('secret material is not persisted or logged by extension source', () => {
  const background = read('background.js');
  const content = read('content.js');
  assert.ok(!/storage\.(local|sync|session)\.set\([^\n]*(ownerToken|credential)/.test(background));
  assert.ok(!/console\.(log|debug|info|error)\([^\n]*(credential|ownerToken|session_token)/.test(background + content));
  assert.match(content, /message\.origin !== location\.origin/);
  assert.match(content, /message\.documentId !== documentId/);
});

test('fill is explicit and field scoped', () => {
  const background = read('background.js');
  const popup = read('popup.js');
  assert.match(background, /APPROVE_ORIGIN/);
  assert.match(background, /nativeRequest\('unlock'/);
  assert.match(background, /nativeRequest\('fill'/);
  assert.match(popup, /Selected field filled; no form was submitted/);
});

test('account discovery is credential-free and one-time-code filling is field scoped', () => {
  const background = read('background.js');
  const content = read('content.js');
  assert.match(background, /nativeRequest\('metadata'/);
  assert.match(background, /response\.metadata/);
  assert.match(content, /one-time-code/);
  assert.match(content, /'totp'/);
});

test('save and update require an explicit popup action and remain revision bound', () => {
  const background = read('background.js');
  const popup = read('popup.js');
  assert.match(background, /nativeRequest\(update \? 'update' : 'save'/);
  assert.match(background, /update \? Number\(input\.revision\) : 0/);
  assert.match(popup, /send\('SAVE'/);
  assert.match(popup, /send\('UPDATE'/);
  assert.match(read('content.js'), /COLLECT_LOGIN/);
  assert.match(background, /FORM_SUBMITTED/);
  assert.match(background, /GENERATE/);
  assert.match(popup, /send\('GENERATE'/);
});
