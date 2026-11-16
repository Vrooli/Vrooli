#!/usr/bin/env node
// Exercise native sign-in and remote AI consent in the exact packaged BAS
// Electron artifact. The script owns its temporary UI proxy, xdg-open shim,
// Electron helper, and all child-process cleanup. LPBS, BAS, and Gateway
// servers plus the AppImage are supplied by the outer Go integration test.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { createReadStream } from 'node:fs';
import { chmod, mkdir, mkdtemp, readFile, rm, stat, writeFile } from 'node:fs/promises';
import http from 'node:http';
import { isIP } from 'node:net';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '../../../../..');
const s2dAPIDir = path.join(repoRoot, 'scenarios/scenario-to-desktop/api');
const appImage = required(process.env.A15_APPIMAGE_PATH, 'A15_APPIMAGE_PATH');
const expectedDigest = normalizeDigest(required(process.env.A15_ARTIFACT_DIGEST, 'A15_ARTIFACT_DIGEST'));
const lpbsURL = loopbackURL(required(process.env.A15_LPBS_URL, 'A15_LPBS_URL'), 'A15_LPBS_URL');
const basURL = loopbackURL(required(process.env.A15_BAS_API_URL, 'A15_BAS_API_URL'), 'A15_BAS_API_URL');
const gatewayURL = loopbackURL(required(process.env.CONSENT_GATEWAY_URL, 'CONSENT_GATEWAY_URL'), 'CONSENT_GATEWAY_URL');
const aliceToken = required(process.env.CONSENT_ALICE_TOKEN, 'CONSENT_ALICE_TOKEN');
const initialDispatches = Number(required(process.env.CONSENT_FINAL_DISPATCHES, 'CONSENT_FINAL_DISPATCHES'));
assert.ok(Number.isInteger(initialDispatches) && initialDispatches >= 0, 'CONSENT_FINAL_DISPATCHES must be a nonnegative integer');
const uiDistDir = path.resolve(required(process.env.A15_UI_DIST_DIR, 'A15_UI_DIST_DIR'));
const evidencePath = process.env.A15_EVIDENCE_PATH ? path.resolve(process.env.A15_EVIDENCE_PATH) : '';
const screenshotPath = process.env.A15_SCREENSHOT_PATH ? path.resolve(process.env.A15_SCREENSHOT_PATH) : '';
const requestedFront = process.env.CONSENT_FRONT_URL ? loopbackURL(process.env.CONSENT_FRONT_URL, 'CONSENT_FRONT_URL') : null;
const timeoutMs = Number(process.env.A15_PACKAGED_TIMEOUT_MS || 90000);

function required(value, name) {
  assert.ok(value, `${name} is required`);
  return value;
}
function normalizeDigest(value) {
  const digest = value.startsWith('sha256:') ? value : `sha256:${value}`;
  assert.match(digest, /^sha256:[a-f0-9]{64}$/i, 'A15_ARTIFACT_DIGEST must be a SHA-256 digest');
  return digest.toLowerCase();
}
function isLoopbackHost(host) {
  const value = host.replace(/^\[|\]$/g, '').toLowerCase();
  if (value === 'localhost' || value === '::1') return true;
  return isIP(value) === 4 && value.startsWith('127.');
}
function loopbackURL(raw, name) {
  const value = new URL(raw);
  assert.ok(value.protocol === 'http:' || value.protocol === 'https:', `${name} must use HTTP(S)`);
  assert.ok(isLoopbackHost(value.hostname), `${name} must be a loopback URL: ${value.origin}`);
  assert.equal(value.username, '', `${name} must not contain credentials`);
  assert.equal(value.password, '', `${name} must not contain credentials`);
  return value;
}
function mimeType(file) {
  return ({ '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.mjs': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.json': 'application/json; charset=utf-8', '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.woff': 'font/woff', '.woff2': 'font/woff2', '.ico': 'image/x-icon' })[path.extname(file).toLowerCase()] || 'application/octet-stream';
}
async function digestFile(file) {
  const hash = createHash('sha256');
  for await (const chunk of createReadStream(file)) hash.update(chunk);
  return `sha256:${hash.digest('hex')}`;
}
async function until(check, timeout, message, interval = 100) {
  const end = Date.now() + timeout;
  let lastError;
  while (Date.now() < end) {
    try { const result = await check(); if (result) return result; } catch (error) { lastError = error; }
    await delay(interval);
  }
  throw new Error(`${message}${lastError ? ` (${lastError.message})` : ''}`);
}
function cdpClient(ws) {
  const pending = new Map();
  let id = 0;
  ws.addEventListener('message', event => {
    const packet = JSON.parse(typeof event.data === 'string' ? event.data : Buffer.from(event.data).toString('utf8'));
    const entry = pending.get(packet.id);
    if (!entry) return;
    pending.delete(packet.id);
    clearTimeout(entry.timer);
    packet.error ? entry.reject(new Error(packet.error.message)) : entry.resolve(packet.result);
  });
  return {
    call(method, params = {}, timeout = 15000) {
      const requestID = ++id;
      return new Promise((resolve, reject) => {
        const timer = setTimeout(() => { pending.delete(requestID); reject(new Error(`CDP timeout: ${method}`)); }, timeout);
        pending.set(requestID, { resolve, reject, timer });
        ws.send(JSON.stringify({ id: requestID, method, params }));
      });
    },
    async evaluate(expression) {
      const response = await this.call('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true, userGesture: true });
      if (response.exceptionDetails) throw new Error(response.exceptionDetails.exception?.description || response.exceptionDetails.text || 'renderer script failed');
      return response.result?.value;
    },
  };
}
function rendererExpression(script) { return `(${script})()`; }
async function stopChild(child, timeoutMs = 12000) {
  if (!child || child.exitCode !== null || child.signalCode !== null) return;
  try { process.kill(-child.pid, 'SIGTERM'); } catch { try { child.kill('SIGTERM'); } catch {} }
  await Promise.race([once(child, 'close').catch(() => {}), delay(timeoutMs)]);
  if (child.exitCode === null && child.signalCode === null) {
    try { process.kill(-child.pid, 'SIGKILL'); } catch { try { child.kill('SIGKILL'); } catch {} }
    await Promise.race([once(child, 'close').catch(() => {}), delay(3000)]);
  }
}
function appendProxyHeaders(headers, url) {
  const result = { ...headers, host: url.host };
  delete result.connection;
  delete result['content-length'];
  return result;
}
function startProxy(port, host) {
  const gatewayPrefix = '/api/v1/ai/privacy-consent';
  const server = http.createServer((req, res) => {
    const requestPath = new URL(req.url || '/', `http://${req.headers.host || '127.0.0.1'}`).pathname;
    const upstreamBase = requestPath === gatewayPrefix || requestPath.startsWith(`${gatewayPrefix}/`) || requestPath === '/__fixture/stats'
      ? gatewayURL : basURL;
    if (requestPath === '/__fixture/page') {
      res.writeHead(200, { 'content-type': 'text/html; charset=utf-8' });
      res.end('<!doctype html><title>A15 local page</title><main><h1>Local fixture page</h1><p>Packaged consent test target.</p></main>');
      return;
    }
    if (requestPath.startsWith('/__vite') || requestPath.startsWith('/@vite') || requestPath.startsWith('/src/')) {
      res.writeHead(404); res.end('Vite development routes are not available in the packaged fixture.'); return;
    }
    if (requestPath.startsWith('/api/') || requestPath.includes('.v1.')) {
      const targetURL = new URL(req.url || '/', upstreamBase);
      const upstream = http.request(targetURL, { method: req.method, headers: appendProxyHeaders(req.headers, targetURL) }, upstreamResponse => {
        res.writeHead(upstreamResponse.statusCode || 502, upstreamResponse.headers);
        upstreamResponse.pipe(res);
      });
      upstream.on('error', error => { if (!res.headersSent) res.writeHead(502); res.end(error.message); });
      req.pipe(upstream);
      return;
    }
    void serveStatic(req, res, requestPath);
  });
  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, host, () => resolve(server));
  });
}
async function serveStatic(req, res, requestPath) {
  if (req.method !== 'GET' && req.method !== 'HEAD') { res.writeHead(405); res.end(); return; }
  let pathname;
  try { pathname = decodeURIComponent(requestPath); } catch { res.writeHead(400); res.end(); return; }
  const root = path.resolve(uiDistDir);
  let file = pathname === '/' ? path.join(root, 'index.html') : path.resolve(root, `.${pathname}`);
  if (!file.startsWith(`${root}${path.sep}`) && file !== path.join(root, 'index.html')) { res.writeHead(403); res.end(); return; }
  try { await stat(file); } catch { file = path.join(root, 'index.html'); }
  try {
    const info = await stat(file);
    if (!info.isFile()) throw new Error('not a file');
    res.writeHead(200, { 'content-type': mimeType(file), 'cache-control': path.basename(file) === 'index.html' ? 'no-store' : 'public, max-age=300' });
    if (req.method === 'HEAD') { res.end(); return; }
    createReadStream(file).on('error', () => { if (!res.headersSent) res.writeHead(404); res.end(); }).pipe(res);
  } catch { res.writeHead(404); res.end('UI artifact not found.'); }
}
async function startFront() {
  const index = path.join(uiDistDir, 'index.html');
  await stat(index);
  if (requestedFront) {
    const port = requestedFront.port ? Number(requestedFront.port) : requestedFront.protocol === 'https:' ? 443 : 80;
    const host = requestedFront.hostname;
    const server = await startProxy(port, host);
    return { server, url: new URL('/', requestedFront.origin) };
  }
  const server = await startProxy(0, '127.0.0.1');
  const address = server.address();
  assert.ok(address && typeof address === 'object', 'UI proxy failed to expose its ephemeral port');
  return { server, url: new URL(`http://127.0.0.1:${address.port}/`) };
}
async function waitForReadyLine(child, stderr) {
  let buffer = '';
  return await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error(`timed out waiting for Electron helper readiness; stderr=${stderr()}`)), timeoutMs);
    child.stdout.setEncoding('utf8');
    child.stdout.on('data', chunk => {
      buffer += chunk;
      const lines = buffer.split(/\r?\n/);
      buffer = lines.pop() || '';
      for (const line of lines) {
        if (!line.trim()) continue;
        try { const parsed = JSON.parse(line); clearTimeout(timer); resolve(parsed); return; } catch { /* Go tool output before helper readiness */ }
      }
    });
    child.once('error', error => { clearTimeout(timer); reject(error); });
    child.once('exit', code => { clearTimeout(timer); reject(new Error(`Electron helper exited before readiness (${code}); stderr=${stderr()}`)); });
  });
}
function makeAuthShim(tempDir, captureFile) {
  const shimDir = path.join(tempDir, 'xdg-bin');
  const script = path.join(shimDir, 'xdg-open');
  return mkdir(shimDir, { recursive: true }).then(async () => {
    await writeFile(script, `#!/bin/sh\nset -eu\nprintf '%s\\n' "$1" >> "$A15_XDG_CAPTURE_FILE"\n`, { mode: 0o700 });
    await chmod(script, 0o700);
    return shimDir;
  });
}
async function readCapturedAuthURL(file, before) {
  return await until(async () => {
    let content;
    try { content = await readFile(file, 'utf8'); } catch { return null; }
    const values = content.split(/\r?\n/).map(value => value.trim()).filter(Boolean);
    const candidate = values.at(-1);
    return candidate && candidate !== before ? candidate : null;
  }, 15000, 'Electron did not call xdg-open with a new auth URL');
}
function validateCapturedAuth(raw) {
  const auth = loopbackURL(raw, 'captured xdg-open authorization URL');
  assert.equal(auth.origin, lpbsURL.origin, 'xdg-open URL did not point to the task-owned fake LPBS');
  assert.equal(auth.pathname, '/auth/login', 'xdg-open URL path must be /auth/login');
  const callback = loopbackURL(required(auth.searchParams.get('redirect_uri'), 'native redirect_uri'), 'native redirect_uri');
  assert.equal(callback.pathname, '/callback');
  assert.ok(auth.searchParams.get('state'), 'native auth URL omitted state');
  assert.ok(auth.searchParams.get('code_challenge'), 'native auth URL omitted PKCE challenge');
  assert.equal(auth.searchParams.get('code_challenge_method'), 'S256');
  return { auth, callback };
}
async function completeFakeLogin(auth, callback, state) {
  const hops = [];
  let current = auth;
  for (let i = 0; i < 8; i += 1) {
    assert.ok(isLoopbackHost(current.hostname), `fake auth flow tried non-loopback ${current.href}`);
    assert.ok(current.origin === lpbsURL.origin || current.origin === callback.origin, `fake auth flow tried unexpected origin ${current.origin}`);
    const response = await fetch(current, { redirect: 'manual', headers: { 'User-Agent': 'a15-packaged-consent-runner' } });
    hops.push({ url: current.href, status: response.status });
    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get('location');
      assert.ok(location, `fake LPBS redirect had no Location at ${current.href}`);
      current = new URL(location, current);
      if (current.origin === callback.origin) {
        assert.equal(current.pathname, callback.pathname, 'fake LPBS redirected to an unexpected callback path');
        assert.equal(current.searchParams.get('state'), state, 'fake LPBS callback state mismatch');
        assert.ok(current.searchParams.get('code'), 'fake LPBS callback omitted code');
      }
      continue;
    }
    assert.ok(response.ok, `fake LPBS returned HTTP ${response.status} from ${current.href}`);
    if (current.origin === callback.origin) {
      assert.equal(current.pathname, callback.pathname, 'fake LPBS callback path mismatch');
      assert.equal(current.searchParams.get('state'), state, 'fake LPBS callback state mismatch');
      assert.ok(current.searchParams.get('code'), 'fake LPBS callback omitted code');
      return { hops, callbackURL: current.href };
    }
    throw new Error(`fake LPBS returned HTTP ${response.status} without redirecting to the native callback`);
  }
  throw new Error('fake LPBS login exceeded eight redirects');
}
async function readStats() {
  const url = new URL('/__fixture/stats', gatewayURL);
  const response = await fetch(url, { headers: { Authorization: `Bearer ${aliceToken}` } });
  assert.equal(response.status, 200, `Gateway dispatch stats returned ${response.status}`);
  const body = await response.json();
  assert.ok(Number.isInteger(body.dispatches), 'Gateway dispatch stats omitted dispatches');
  return body.dispatches;
}
async function main() {
  assert.ok(path.isAbsolute(appImage), 'A15_APPIMAGE_PATH must be absolute');
  assert.equal(await digestFile(appImage), expectedDigest, 'exact AppImage SHA-256 did not match A15_ARTIFACT_DIGEST');
  const tempDir = await mkdtemp(path.join(process.env.TMPDIR || '/tmp', 'a15-packaged-electron-'));
  const captureFile = path.join(tempDir, 'xdg-open-capture.txt');
  await writeFile(captureFile, '', { mode: 0o600 });
  const shimDir = await makeAuthShim(tempDir, captureFile);
  let proxy;
  let helper;
  let ws;
  let cdp;
  let evidence;
  try {
    const { server, url: frontURL } = await startFront();
    proxy = server;
    const frontHealth = await fetch(frontURL);
    assert.equal(frontHealth.status, 200, `UI proxy failed to serve ${frontURL.href}`);
    const helperEnv = {
      ...process.env,
      A15_APPIMAGE_PATH: appImage,
      A15_ARTIFACT_DIGEST: expectedDigest,
      A15_RENDERER_URL: frontURL.href,
      A15_API_URL: frontURL.origin,
      A15_LPBS_URL: lpbsURL.origin,
      A15_XDG_OPEN_DIR: shimDir,
      A15_XDG_CAPTURE_FILE: captureFile,
      PATH: `${shimDir}${path.delimiter}${process.env.PATH || ''}`,
      HOME: tempDir,
      XDG_CONFIG_HOME: path.join(tempDir, '.config'),
      XDG_DATA_HOME: path.join(tempDir, '.local/share'),
      XDG_STATE_HOME: path.join(tempDir, '.local/state'),
      XDG_CACHE_HOME: path.join(tempDir, '.cache'),
    };
    helper = spawn('go', ['run', './cmd/a15-electron-session'], { cwd: s2dAPIDir, env: helperEnv, stdio: ['ignore', 'pipe', 'pipe'], detached: true });
    let helperStderr = '';
    helper.stderr.setEncoding('utf8');
    helper.stderr.on('data', chunk => { helperStderr = `${helperStderr}${chunk}`.slice(-12000); });
    const ready = await waitForReadyLine(helper, () => helperStderr);
    assert.equal(path.resolve(ready.artifact), path.resolve(appImage), 'Electron helper launched a different AppImage');
    assert.equal(normalizeDigest(ready.digest), expectedDigest, 'Electron helper readiness digest mismatch');
    assert.ok(Number.isInteger(ready.pid) && ready.pid > 1, 'Electron helper readiness omitted application PID');
    assert.equal(new URL(ready.renderer).origin, frontURL.origin, 'Electron helper selected an unexpected renderer origin');
    const endpoint = loopbackURL(ready.endpoint, 'Electron helper CDP endpoint');
    const listResponse = await fetch(new URL('/json/list', endpoint));
    assert.equal(listResponse.status, 200, 'S2D CDP renderer discovery failed');
    const targets = await listResponse.json();
    const candidates = targets.filter(target => target.type === 'page' && target.url?.startsWith(frontURL.origin) && target.webSocketDebuggerUrl);
    assert.equal(candidates.length, 1, `expected exactly one packaged renderer, found ${candidates.length}`);
    const renderer = candidates[0];
    const wsURL = new URL(renderer.webSocketDebuggerUrl);
    assert.ok(isLoopbackHost(wsURL.hostname), 'Electron renderer CDP websocket must be loopback');
    ws = new WebSocket(wsURL);
    await new Promise((resolve, reject) => {
      ws.addEventListener('open', resolve, { once: true });
      ws.addEventListener('error', () => reject(new Error('failed to attach to packaged BAS renderer CDP')), { once: true });
    });
    cdp = cdpClient(ws);
    await cdp.call('Page.enable');
    await cdp.call('Runtime.enable');
    await cdp.call('Network.enable');
    await cdp.call('Page.navigate', { url: new URL('/settings?tab=subscription', frontURL).href });
    await until(() => cdp.evaluate(rendererExpression(`()=>[...document.querySelectorAll('button')].some(button=>/sign in/i.test(button.textContent||''))`)), 30000, 'packaged subscription sign-in control did not render');
    const beforeCapture = await readFile(captureFile, 'utf8');
    const signIn = await cdp.evaluate(rendererExpression(`async()=>{if(!window.desktop?.auth?.signIn)throw new Error('window.desktop.auth.signIn is unavailable');await window.desktop.auth.signIn();return true}`));
    assert.equal(signIn, true, 'native Electron sign-in bridge was not called');
    const capturedURL = await readCapturedAuthURL(captureFile, beforeCapture);
    const { auth: authURL, callback } = validateCapturedAuth(capturedURL);
    const login = await completeFakeLogin(authURL, callback, authURL.searchParams.get('state'));
    await until(() => cdp.evaluate(rendererExpression(`()=>window.desktop?.auth?.isAuthenticated?.()`)), 25000, 'Electron auth manager did not accept the PKCE callback and tokens');
    const nativeAuth = await cdp.evaluate(rendererExpression(`async()=>({token:Boolean(await window.desktop.auth.getAccessToken()),buttons:[...document.querySelectorAll('button')].map(button=>button.textContent?.trim()).filter(Boolean),body:document.body?.innerText||''})`));
    assert.equal(nativeAuth.token, true, 'native access token is missing after sign-in');
    const entitlement = await cdp.evaluate(rendererExpression(`async()=>{const response=await fetch('/browser_automation_studio.v1.entitlement.EntitlementService/GetStatus',{method:'POST',headers:{'Content-Type':'application/json','Connect-Protocol-Version':'1','Authorization':'Bearer '+await window.desktop.auth.getAccessToken()},body:'{}'});return {status:response.status,body:await response.text()}}`));
    assert.equal(entitlement.status, 200, `native session did not resolve BAS entitlement: ${entitlement.body}`);
    const entitlementStatus = JSON.parse(entitlement.body).status;
    assert.ok(entitlementStatus && entitlementStatus.canUseAi === true, `LPBS app-scoped entitlement did not grant BAS AI: ${entitlement.body}`);

    await cdp.call('Page.navigate', { url: new URL('/settings?tab=apikeys', frontURL).href });
    await until(() => cdp.evaluate(rendererExpression(`()=>Boolean(document.querySelector('[data-testid="ai-privacy-consent"]'))`)), 30000, 'packaged AI settings consent panel did not render');
    await cdp.evaluate(rendererExpression(`()=>{const select=document.querySelector('[data-testid="ai-privacy-consent-source"]');if(!select)throw new Error('source selector not found');Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype,'value').set.call(select,'own-key');select.dispatchEvent(new Event('change',{bubbles:true}));document.querySelector('[data-testid="ai-privacy-consent-allow"]')?.click();return true}`));
    await until(() => cdp.evaluate(rendererExpression(`()=>Boolean(sessionStorage.getItem('bas.ai.remote-consent-receipt')) && Boolean(document.querySelector('[data-testid="ai-privacy-consent"] [role="status"]')?.textContent.includes('receipt is active'))`)), 15000, 'packaged UI did not save a remote consent receipt');
    const grant = await cdp.evaluate(rendererExpression(`async()=>({receipt:sessionStorage.getItem('bas.ai.remote-consent-receipt'),source:sessionStorage.getItem('bas.ai.remote-consent-source'),token:await window.desktop.auth.getAccessToken(),revoke:[...document.querySelectorAll('[data-testid^="ai-privacy-consent-revoke-"]')].map(button=>button.dataset.testid)})`));
    assert.ok(grant.receipt, 'consent UI returned no receipt');
    assert.equal(grant.source, 'own-key');
    assert.ok(grant.token, 'AI consent operations require native account token');

    await cdp.call('Page.reload');
    await until(() => cdp.evaluate(rendererExpression(`()=>Boolean(document.querySelector('[data-testid="ai-privacy-consent"]'))`)), 30000, 'packaged consent UI did not render after reload');
    await until(() => cdp.evaluate(rendererExpression(`()=>sessionStorage.getItem('bas.ai.remote-consent-receipt')===${JSON.stringify(grant.receipt)}`)), 10000, 'active consent receipt did not survive packaged renderer reload');
    await until(() => cdp.evaluate(rendererExpression(`()=>[...document.querySelectorAll('[data-testid^="ai-privacy-consent-revoke-"]')].some(button=>button.closest('li')?.textContent.includes('your configured OpenRouter account'))`)), 15000, 'granted consent did not persist in BAS list after reload');
    if (screenshotPath) {
      const screenshot = await cdp.call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: false });
      await mkdir(path.dirname(screenshotPath), { recursive: true });
      await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'), { mode: 0o600 });
    }

    const requestAI = async receipt => await cdp.evaluate(rendererExpression(`async()=>{const response=await fetch('/browser_automation_studio.v1.ai.AIService/AIAnalyzeElements',{method:'POST',headers:{'Content-Type':'application/json','Connect-Protocol-Version':'1','Authorization':'Bearer '+await window.desktop.auth.getAccessToken(),'X-Vrooli-AI-Consent-Source':'own-key','X-Vrooli-AI-Consent-Grant':${JSON.stringify(receipt)},'Idempotency-Key':crypto.randomUUID().replaceAll('-','')},body:JSON.stringify({url:${JSON.stringify(new URL('/__fixture/page', frontURL).href)},intent:'identify the main action button'})});return {status:response.status,body:(await response.text()).slice(0,1200)}}`));
    const beforeValid = await readStats();
    assert.equal(beforeValid, initialDispatches, `Gateway dispatch count changed before consented request: expected ${initialDispatches}, got ${beforeValid}`);
    const grantedResponse = await requestAI(grant.receipt);
    assert.equal(grantedResponse.status, 200, `valid consented BAS AIService call failed: ${grantedResponse.body}`);
    const grantDispatches = await until(async () => { const count = await readStats(); return count === beforeValid + 1 ? count : null; }, 20000, 'valid consent did not dispatch exactly once to fake Gateway');

    const revokeSelector = await cdp.evaluate(rendererExpression(`()=>[...document.querySelectorAll('[data-testid^="ai-privacy-consent-revoke-"]')].find(button=>button.closest('li')?.textContent.includes('your configured OpenRouter account'))?.dataset.testid||''`));
    assert.ok(revokeSelector, 'persisted own-key grant has no revoke control');
    await cdp.evaluate(rendererExpression(`()=>{const button=document.querySelector('[data-testid=${JSON.stringify(revokeSelector)}]');if(!button)throw new Error('revoke control disappeared');button.click();return true}`));
    await until(() => cdp.evaluate(rendererExpression(`()=>!sessionStorage.getItem('bas.ai.remote-consent-receipt') && Boolean(document.querySelector('[data-testid="ai-privacy-consent"] [role="status"]')?.textContent.includes('No consent receipt'))`)), 15000, 'UI revoke did not clear its active receipt');
    const revokedResponse = await requestAI(grant.receipt);
    assert.ok(revokedResponse.status >= 400, `revoked receipt was accepted: ${revokedResponse.status} ${revokedResponse.body}`);
    const revokedDispatches = await readStats();
    assert.equal(revokedDispatches, grantDispatches, 'revoked receipt reached fake provider dispatch');

    const forgedResponse = await requestAI('a15-forged-consent-receipt');
    assert.ok(forgedResponse.status >= 400, `forged receipt was accepted: ${forgedResponse.status} ${forgedResponse.body}`);
    const forgedDispatches = await readStats();
    assert.equal(forgedDispatches, revokedDispatches, 'forged receipt reached fake provider dispatch');
    assert.equal(forgedDispatches, initialDispatches + 1, 'only the authorized packaged consent request may dispatch');

    evidence = {
      artifact_digest: expectedDigest,
      artifact_path: appImage,
      electron_pid: ready.pid,
      renderer_url: renderer.url,
      auth: { pkce_verified: true, token_found: nativeAuth.token, callback_url: login.callbackURL, xdg_open_url: authURL.origin + authURL.pathname },
      entitlement: { app_key: 'browser-automation-studio', feature: 'ai', signed_can_use_ai: entitlementStatus.canUseAi, legacy_global_features: entitlementStatus.features || [], verified_by: 'BAS EntitlementService/GetStatus Connect handler' },
      grant_dispatches: grantDispatches,
      revoked_dispatches: revokedDispatches,
      forged_dispatches: forgedDispatches,
      revoked_refused: revokedResponse.status >= 400,
      forged_refused: forgedResponse.status >= 400,
      consent_grant_id: revokeSelector.slice('ai-privacy-consent-revoke-'.length),
      consent_source: grant.source,
      consent_persisted_across_reload: true,
      auth_redirects: login.hops,
      ai_responses: { grant: grantedResponse.status, revoked: revokedResponse.status, forged: forgedResponse.status },
      services: { lpbs_origin: lpbsURL.origin, bas_origin: basURL.origin, gateway_origin: gatewayURL.origin, renderer_origin: frontURL.origin },
      completed_at: new Date().toISOString(),
    };
    if (evidencePath) {
      await mkdir(path.dirname(evidencePath), { recursive: true });
      await writeFile(evidencePath, `${JSON.stringify(evidence, null, 2)}\n`, { mode: 0o600 });
    }
    process.stdout.write(`${JSON.stringify(evidence)}\n`);
  } finally {
    try { ws?.close(); } catch {}
    await stopChild(helper);
    if (proxy) await new Promise(resolve => proxy.close(resolve));
    await rm(tempDir, { recursive: true, force: true, maxRetries: 3, retryDelay: 100 });
  }
}

main().catch(error => {
  process.stderr.write(`${error instanceof Error ? error.stack || error.message : String(error)}\n`);
  process.exitCode = 1;
});
