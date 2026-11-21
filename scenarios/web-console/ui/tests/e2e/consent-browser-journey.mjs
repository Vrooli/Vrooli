#!/usr/bin/env node
// Browser-rendered consent/revocation journeys for Web Console and BAS.
// Uses the checked-in UI components and caller transports, headless Chrome,
// and a local in-memory consent authority/fake provider. No live service is used.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import http from 'node:http';
import { createSign, createVerify, generateKeyPairSync } from 'node:crypto';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../..');
const chromeBin = process.env.CHROME_BIN || '/usr/bin/google-chrome';
const evidenceDir = process.env.CONSENT_BROWSER_EVIDENCE_DIR || path.resolve(repoRoot, '../.vrooli/research/topics/web-console-monetization/runs/models-monetization-music-20261115');
const cases = [
  { name: 'web-console', ui: path.join(repoRoot, 'scenarios/web-console/ui'), fixture: '/tests/e2e/consent-fixture.html', receiptKey: 'web-console.ai.remote-consent-receipt', sourceKey: 'web-console.ai.remote-consent-source', panel: 'remote-ai-consent', grant: 'remote-ai-consent-grant', source: 'remote-ai-consent-source', revokePrefix: 'remote-ai-consent-revoke-' },
  { name: 'browser-automation-studio', ui: path.join(repoRoot, 'scenarios/browser-automation-studio/ui'), fixture: '/tests/e2e/consent-fixture.html', receiptKey: 'bas.ai.remote-consent-receipt', sourceKey: 'bas.ai.remote-consent-source', panel: 'ai-privacy-consent', grant: 'ai-privacy-consent-allow', source: 'ai-privacy-consent-source', revokePrefix: 'ai-privacy-consent-revoke-' },
];

function freePort() {
  return new Promise((resolve, reject) => {
    const server = http.createServer();
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const port = server.address().port;
      server.close(() => resolve(port));
    });
  });
}
function sendJSON(res, status, data) {
  res.writeHead(status, { 'content-type': 'application/json', 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,DELETE,OPTIONS' });
  res.end(JSON.stringify(data));
}
function tokenFor(subject, privateKey) {
  const now = Math.floor(Date.now() / 1000);
  const enc = value => Buffer.from(JSON.stringify(value)).toString('base64url');
  const content = `${enc({ alg: 'RS256', typ: 'JWT' })}.${enc({ sub: subject, iss: 'https://fake-lpbs.example.test', aud: 'consent-fixture', iat: now, exp: now + 3600, email: 'fixture@example.test', email_verified: true })}`;
  const signer = createSign('RSA-SHA256');
  signer.update(content);
  signer.end();
  return `${content}.${signer.sign(privateKey).toString('base64url')}`;
}
function userFrom(req, publicKey) {
  const token = String(req.headers.authorization || '').replace(/^Bearer\s+/i, '');
  try {
    const [header, payload, signature] = token.split('.');
    const claims = JSON.parse(Buffer.from(payload, 'base64url').toString());
    const verifier = createVerify('RSA-SHA256');
    verifier.update(`${header}.${payload}`);
    verifier.end();
    if (JSON.parse(Buffer.from(header, 'base64url').toString()).alg !== 'RS256' || !verifier.verify(publicKey, Buffer.from(signature, 'base64url'))) return '';
    if (claims.iss !== 'https://fake-lpbs.example.test' || claims.aud !== 'consent-fixture' || !claims.sub || claims.email_verified !== true || claims.exp <= Date.now() / 1000) return '';
    return claims.sub;
  } catch { return ''; }
}
function createFakeAuthority() {
  const { privateKey, publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
  const grants = new Map();
  const attempts = [];
  const dispatches = [];
  const expiredReceipt = 'fixture-expired-consent-receipt';
  grants.set(expiredReceipt, { id: 'expired-grant', receipt: expiredReceipt, owner: 'user-1', source: 'own-key', expires_at: new Date(Date.now() - 60_000).toISOString(), revoked_at: '' });
  async function handle(req, res) {
    if (req.method === 'OPTIONS') { res.writeHead(204, { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,DELETE,OPTIONS' }); return res.end(); }
    const url = new URL(req.url, 'http://fixture');
    const user = userFrom(req, publicKey);
    if (url.pathname === '/api/v1/ai/privacy-consent' && req.method === 'GET') {
      return sendJSON(res, 200, { grants: [...grants.values()].filter(grant => grant.owner === user).map(({ receipt, owner, ...grant }) => grant) });
    }
    if (url.pathname === '/api/v1/ai/privacy-consent' && req.method === 'POST') {
      const body = await readBody(req);
      const source = body.source;
      if (!user || !['own-key', 'vrooli-credits'].includes(source)) return sendJSON(res, 400, { error: 'invalid user or source' });
      const id = `grant-${Math.random().toString(36).slice(2)}`;
      const receipt = `fixture-receipt-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
      const grant = { id, receipt, owner: user, source, created_at: new Date().toISOString(), expires_at: new Date(Date.now() + 30 * 86400_000).toISOString(), revoked_at: '' };
      grants.set(receipt, grant);
      return sendJSON(res, 201, { grant: publicGrant(grant), receipt });
    }
    const revoke = url.pathname.match(/^\/api\/v1\/ai\/privacy-consent\/([^/]+)$/);
    if (revoke && req.method === 'DELETE') {
      const grant = [...grants.values()].find(item => item.id === decodeURIComponent(revoke[1]) && item.owner === user);
      if (!grant) return res.writeHead(404).end();
      grant.revoked_at = new Date().toISOString();
      res.writeHead(204, { 'access-control-allow-origin': '*' }); return res.end();
    }
    const isDispatch = req.method === 'POST' && (url.pathname === '/api/v1/ai/generate' || url.pathname.includes('AIService/'));
    if (isDispatch) {
      const bodyText = await readBodyText(req);
      const receipt = req.headers['x-vrooli-ai-consent-grant'] || '';
      const source = req.headers['x-vrooli-ai-consent-source'] || '';
      const grant = grants.get(receipt);
      let rejection = '';
      if (!receipt) rejection = 'missing';
      else if (!grant) rejection = 'forged';
      else if (grant.owner !== user) rejection = 'wrong-user';
      else if (grant.source !== source) rejection = 'wrong-source';
      else if (grant.revoked_at) rejection = 'revoked';
      else if (Date.parse(grant.expires_at) <= Date.now()) rejection = 'expired';
      else if (bodyText.includes('SECRET_SENTINEL')) rejection = 'secret-local-only';
      attempts.push({ user, source, receiptPresent: Boolean(receipt), rejection, body: bodyText });
      if (rejection) return sendJSON(res, 403, { error: rejection });
      dispatches.push({ user, source, provider: source === 'own-key' ? 'fake-openrouter' : 'fake-vrooli-credits' });
      if (url.pathname === '/api/v1/ai/generate') return sendJSON(res, 200, { command: 'echo fixture', provider: 'fake-openrouter' });
      return sendJSON(res, 200, { suggestions: [] });
    }
    sendJSON(res, 404, { error: 'fake authority route not found' });
  }
  return { grants, attempts, dispatches, expiredReceipt, handle, tokenFor: subject => tokenFor(subject, privateKey) };
}
function publicGrant({ receipt, owner, ...grant }) { return grant; }
function readBody(req) { return readBodyText(req).then(text => text ? JSON.parse(text) : {}); }
function readBodyText(req) { return new Promise((resolve, reject) => { let text = ''; req.setEncoding('utf8'); req.on('data', chunk => text += chunk); req.on('end', () => resolve(text)); req.on('error', reject); }); }

async function startServers(uiRoot) {
  const vitePort = await freePort();
  const frontPort = await freePort();
  const viteLogs = [];
  const frontRequests = [];
  const keepLog = stream => {
    let pending = '';
    stream.on('data', chunk => {
      pending += chunk.toString();
      const lines = pending.split(/\r?\n/);
      pending = lines.pop() || '';
      viteLogs.push(...lines.filter(Boolean));
      if (viteLogs.length > 120) viteLogs.splice(0, viteLogs.length - 120);
    });
  };
  const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', String(vitePort), '--strictPort'], { cwd: uiRoot, stdio: ['ignore', 'pipe', 'pipe'], detached: true });
  keepLog(vite.stdout);
  keepLog(vite.stderr);
  const authority = createFakeAuthority();
  const proxy = http.createServer((req, res) => {
    if (req.url.startsWith('/api/') || req.url.includes('AIService/')) return void authority.handle(req, res);
    const upstream = http.request({ hostname: '127.0.0.1', port: vitePort, path: req.url, method: req.method, headers: { ...req.headers, host: `127.0.0.1:${vitePort}` } }, upstreamRes => {
      frontRequests.push({ path: req.url, status: upstreamRes.statusCode || 500 });
      if (frontRequests.length > 200) frontRequests.shift();
      res.writeHead(upstreamRes.statusCode || 500, upstreamRes.headers); upstreamRes.pipe(res);
    });
    upstream.on('error', error => { frontRequests.push({ path: req.url, status: 502, error: error.message }); res.writeHead(502); res.end(error.message); });
    req.pipe(upstream);
  });
  proxy.listen(frontPort, '127.0.0.1');
  await once(proxy, 'listening');
  try {
    await waitUntil(async () => vite.exitCode !== null ? false : (await fetch(`http://127.0.0.1:${vitePort}/`)).ok, 30_000, `Vite did not start for ${uiRoot}`);
    const fixtureScript = uiRoot.includes('browser-automation-studio') ? '/tests/e2e/consent-fixture/bas.tsx' : '/tests/e2e/consent-fixture/web-console.tsx';
    const scriptResponse = await fetch(`http://127.0.0.1:${vitePort}${fixtureScript}`);
    if (!scriptResponse.ok) throw new Error(`Vite fixture transform returned ${scriptResponse.status}: ${(await scriptResponse.text()).slice(0, 1000)}\n${viteLogs.join('\n')}`);
  } catch (error) {
    await stopVite(vite);
    await new Promise(resolve => proxy.close(resolve));
    throw new Error(`${error.message}\nVite log:\n${viteLogs.join('\n')}`);
  }
  return { vite, proxy, vitePort, frontPort, authority, viteLogs, frontRequests };
}
async function stopVite(vite) {
  try { process.kill(-vite.pid, 'SIGTERM'); } catch { vite.kill('SIGTERM'); }
  await Promise.race([once(vite, 'close').catch(() => {}), delay(3000)]);
}
async function waitUntil(check, timeout, message) {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try { if (await check()) return; } catch { /* service or UI is still starting */ }
    await delay(100);
  }
  throw new Error(message);
}

class CDP {
  constructor(ws) { this.ws = ws; this.id = 0; this.pending = new Map(); }
  static async connect(url) {
    const ws = new WebSocket(url);
    await new Promise((resolve, reject) => { ws.addEventListener('open', resolve, { once: true }); ws.addEventListener('error', reject, { once: true }); });
    const client = new CDP(ws);
    client.events = [];
    ws.addEventListener('message', event => {
      const packet = JSON.parse(event.data);
      if (packet.id && client.pending.has(packet.id)) { const { resolve, reject } = client.pending.get(packet.id); client.pending.delete(packet.id); packet.error ? reject(new Error(packet.error.message)) : resolve(packet.result); }
      if (packet.method && client.events.length < 200) client.events.push(packet);
    });
    return client;
  }
  call(method, params = {}) {
    const id = ++this.id;
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`Chrome DevTools command timed out: ${method}`));
      }, 15_000);
      this.pending.set(id, {
        resolve: value => { clearTimeout(timeout); resolve(value); },
        reject: error => { clearTimeout(timeout); reject(error); },
      });
      this.ws.send(JSON.stringify({ id, method, params }));
    });
  }
  async evaluate(expression) {
    const result = await this.call('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true, userGesture: true });
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.text || 'browser evaluation failed');
    return result.result?.value;
  }
}
async function launchChrome() {
  const profile = await mkdtemp(path.join(tmpdir(), 'consent-chrome-'));
  const chrome = spawn(chromeBin, ['--headless=new', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank'], { stdio: 'ignore', detached: true });
  const active = path.join(profile, 'DevToolsActivePort');
  const started = Date.now();
  let port;
  while (Date.now() - started < 15_000) {
    try { port = Number((await readFile(active, 'utf8')).split('\n')[0]); break; } catch { await delay(100); }
  }
  if (!port) { chrome.kill('SIGTERM'); throw new Error('Chrome did not expose its local DevTools endpoint'); }
  await waitUntil(async () => (await fetch(`http://127.0.0.1:${port}/json/version`)).ok, 5000, 'Chrome DevTools endpoint unavailable');
  return { chrome, profile, port };
}
async function newTab(browserPort, url) {
  const response = await fetch(`http://127.0.0.1:${browserPort}/json/new?about:blank`, { method: 'PUT' });
  if (!response.ok) throw new Error(`Chrome could not open fixture tab (${response.status})`);
  const target = await response.json();
  const cdp = await CDP.connect(target.webSocketDebuggerUrl);
  try {
    await cdp.call('Page.enable');
    await cdp.call('Runtime.enable');
    await cdp.call('Log.enable');
    await cdp.call('Network.enable');
    await cdp.call('Page.addScriptToEvaluateOnNewDocument', { source: `window.__consentHarnessErrors=[];window.addEventListener('error',event=>window.__consentHarnessErrors.push({type:'error',message:String(event.message||''),source:String(event.filename||''),line:event.lineno||0}));window.addEventListener('unhandledrejection',event=>window.__consentHarnessErrors.push({type:'unhandledrejection',message:String(event.reason||'')}));` });
    cdp.events.length = 0;
    await cdp.call('Page.navigate', { url });
    await waitUntil(async () => await cdp.evaluate(`document.readyState === 'complete' && location.href !== 'about:blank'`), 15_000, `Chrome did not finish loading ${url}`);
    return { cdp, targetId: target.id };
  } catch (error) {
    try { await fetch(`http://127.0.0.1:${browserPort}/json/close/${target.id}`, { method: 'PUT' }); } catch {}
    throw error;
  }
}
async function click(cdp, testId) {
  const clicked = await cdp.evaluate(`(() => { const el=document.querySelector('[data-testid="${testId}"]'); if (!el) return false; el.click(); return true; })()`);
  assert.equal(clicked, true, `missing browser control ${testId}`);
}
async function setToken(cdp, signedToken) {
  await cdp.evaluate(`sessionStorage.setItem('vrooli.web.access-token', JSON.stringify({accessToken:${JSON.stringify(signedToken)},expiresAt:new Date(Date.now()+3600000).toISOString()}))`);
}
async function waitForResult(cdp, expected) {
  await waitUntil(async () => String(await cdp.evaluate(`document.querySelector('[data-testid="request-result"]')?.textContent || ''`)).includes(expected), 5000, `request result did not contain ${expected}`);
}
async function runJourney(browserPort, testCase) {
  const servers = await startServers(testCase.ui);
  let tab;
  try {
    const authority = servers.authority;
    const user1Token = authority.tokenFor('user-1');
    tab = await newTab(browserPort, `http://127.0.0.1:${servers.frontPort}${testCase.fixture}?fixtureToken=${encodeURIComponent(user1Token)}`);
    const { cdp } = tab;
    const outcome = { app: testCase.name, checks: [], acceptedDispatches: [], rejectedBeforeDispatch: [] };
    try {
      await waitUntil(async () => Boolean(await cdp.evaluate(`Boolean(document.querySelector('[data-testid="${testCase.panel}"]'))`)), 30_000, `${testCase.name} consent panel did not render`);
    } catch (error) {
      const pageState = await cdp.evaluate(`JSON.stringify({url:location.href,title:document.title,readyState:document.readyState,moduleScripts:[...document.querySelectorAll('script[type="module"]')].map(script=>script.src),windowErrors:window.__consentHarnessErrors||[],body:document.body?.innerText?.slice(0,1500),html:document.documentElement?.outerHTML?.slice(0,4000)})`);
      const browserErrors = cdp.events.filter(event => event.method === 'Runtime.exceptionThrown' || event.method === 'Log.entryAdded' || (event.method === 'Network.responseReceived' && event.params.response.status >= 400)).map(event => ({ method: event.method, params: event.params })).slice(-25);
      throw new Error(`${error.message}; browser page: ${pageState}\nBrowser diagnostics: ${JSON.stringify(browserErrors)}\nVite diagnostics: ${servers.viteLogs.join('\n')}\nUI requests: ${JSON.stringify(servers.frontRequests.slice(-30))}`);
    }
    const receipt = testCase.receiptKey;
    const source = testCase.sourceKey;
    const send = 'send-ai';
    const attempt = async (kind, expectedReject) => {
      const before = authority.attempts.length;
      const dispatchCountBefore = authority.dispatches.length;
      await click(cdp, send);
      try { await waitUntil(async () => authority.attempts.length > before, 5000, `${kind} request did not reach fake gateway`); }
      catch (error) {
        const result = await cdp.evaluate(`JSON.stringify({result:document.querySelector('[data-testid="request-result"]')?.textContent,resources:performance.getEntriesByType('resource').map(item=>item.name).filter(name=>name.includes('/api/')).slice(-8)})`);
        throw new Error(`${error.message}; browser: ${result}`);
      }
      await waitUntil(async () => String(await cdp.evaluate(`document.querySelector('[data-testid="request-result"]')?.textContent || ''`)).includes('Rejected:'), 5000, `${kind} request result did not show refusal`);
      const record = authority.attempts.at(-1);
      assert.equal(record.rejection, expectedReject, `${testCase.name} ${kind} refusal; gateway record=${JSON.stringify(record)}`);
      assert.equal(authority.dispatches.length, dispatchCountBefore, `${kind} must not reach fake provider`);
      outcome.rejectedBeforeDispatch.push(kind);
    };
    await attempt('missing receipt', 'missing');

    await cdp.evaluate(`(() => { const s=document.querySelector('[data-testid="${testCase.source}"]'); s.value='own-key'; s.dispatchEvent(new Event('change',{bubbles:true})); })()`);
    await click(cdp, testCase.grant);
    await waitUntil(async () => authority.grants.size > 1 && Boolean(await cdp.evaluate(`sessionStorage.getItem(${JSON.stringify(receipt)})`)), 5000, `${testCase.name} grant UI did not save receipt`);
    const issuedReceipt = await cdp.evaluate(`sessionStorage.getItem(${JSON.stringify(receipt)})`);
    assert.ok(issuedReceipt, 'grant receipt must be present in the app tab');

    await setToken(cdp, authority.tokenFor('user-2'));
    await attempt('wrong user', 'wrong-user');
    await setToken(cdp, user1Token);

    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(source)}, 'vrooli-credits')`);
    await attempt('wrong source', 'wrong-source');
    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(source)}, 'own-key')`);

    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(receipt)}, 'forged-fixture-receipt')`);
    await attempt('forged receipt', 'forged');
    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(receipt)}, ${JSON.stringify(authority.expiredReceipt)}); sessionStorage.setItem(${JSON.stringify(source)}, 'own-key')`);
    await attempt('expired receipt', 'expired');

    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(receipt)}, ${JSON.stringify(issuedReceipt)}); sessionStorage.setItem(${JSON.stringify(source)}, 'own-key')`);
    const beforeValid = authority.attempts.length;
    await click(cdp, send);
    await waitUntil(async () => authority.attempts.length > beforeValid, 5000, 'valid receipt request did not reach fake gateway');
    await waitForResult(cdp, 'Dispatched:');
    assert.equal(authority.dispatches.length, 1, 'valid selected source must dispatch exactly once');
    assert.equal(authority.dispatches[0].source, 'own-key', 'dispatch must use the selected source');
    outcome.acceptedDispatches.push(authority.dispatches[0]);

    await cdp.evaluate(`(() => { const s=document.querySelector('[data-testid="${testCase.source}"]'); s.value='vrooli-credits'; s.dispatchEvent(new Event('change',{bubbles:true})); })()`);
    await click(cdp, testCase.grant);
    await waitUntil(async () => await cdp.evaluate(`sessionStorage.getItem(${JSON.stringify(source)})`) === 'vrooli-credits', 5000, `${testCase.name} credits consent UI did not save selected source`);
    const creditsReceipt = await cdp.evaluate(`sessionStorage.getItem(${JSON.stringify(receipt)})`);
    const beforeCredits = authority.attempts.length;
    await click(cdp, send);
    await waitUntil(async () => authority.attempts.length > beforeCredits, 5000, 'valid Vrooli credits request did not reach fake gateway');
    await waitForResult(cdp, 'Dispatched:');
    assert.equal(authority.dispatches.length, 2, 'each independently selected source should dispatch once');
    assert.equal(authority.dispatches[1].source, 'vrooli-credits', 'credits consent must dispatch only to the selected credits source');
    outcome.acceptedDispatches.push(authority.dispatches[1]);
    await cdp.call('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1100, deviceScaleFactor: 1, mobile: false });
    const screenshot = await cdp.call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true });
    const screenshotPath = path.join(evidenceDir, `browser-consent-${testCase.name}.png`);
    await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'));
    outcome.screenshot = screenshotPath;

    await click(cdp, 'secret-toggle');
    await attempt('secret remains local-only', 'secret-local-only');

    const creditsGrant = [...authority.grants.values()].find(grant => grant.owner === 'user-1' && grant.source === 'vrooli-credits' && !grant.revoked_at);
    assert.ok(creditsGrant, `${testCase.name} credits grant should remain revocable`);
    const revokeId = `${testCase.revokePrefix}${creditsGrant.id}`;
    assert.equal(await cdp.evaluate(`Boolean(document.querySelector('[data-testid="${revokeId}"]'))`), true, `${testCase.name} credits grant must expose revoke control`);
    assert.ok(revokeId, `${testCase.name} grant list must expose revoke control`);
    const id = revokeId.slice(testCase.revokePrefix.length);
    await click(cdp, revokeId);
    await waitUntil(async () => Boolean([...authority.grants.values()].find(grant => grant.id === id)?.revoked_at), 5000, `${testCase.name} revoke did not reach fake authority`);
    await waitUntil(async () => String(await cdp.evaluate(`document.querySelector('[data-testid="${testCase.panel}"] [role="status"]')?.textContent || ''`)).includes('No consent receipt'), 5000, `${testCase.name} revoke UI did not finish clearing the active receipt`);
    await cdp.evaluate(`sessionStorage.setItem(${JSON.stringify(receipt)}, ${JSON.stringify(creditsReceipt)}); sessionStorage.setItem(${JSON.stringify(source)}, 'vrooli-credits')`);
    const restoredReceipt = await cdp.evaluate(`sessionStorage.getItem(${JSON.stringify(receipt)})`);
    assert.equal(restoredReceipt, creditsReceipt, `${testCase.name} harness restores the revoked receipt before retry`);
    await attempt('revoked receipt', 'revoked');
    outcome.checks = ['real consent panel rendered', 'real caller transport sent scoped receipt', 'grant/revoke round trip', 'fake provider dispatch gated by user, source and expiry', 'SECRET rejected before fake dispatch'];
    outcome.acceptedDispatches = authority.dispatches;
    outcome.fakeDispatchCount = authority.dispatches.length;
    outcome.refusalMatrix = authority.attempts.map(item => ({ rejection: item.rejection, receiptPresent: item.receiptPresent, source: item.source }));
    return outcome;
  } finally {
    if (tab) try { await fetch(`http://127.0.0.1:${browserPort}/json/close/${tab.targetId}`, { method: 'PUT' }); } catch {}
    await stopVite(servers.vite);
    await new Promise(resolve => servers.proxy.close(resolve));
  }
}

const profile = await launchChrome();
try {
  await mkdir(evidenceDir, { recursive: true });
  const results = [];
  for (const testCase of cases) results.push(await runJourney(profile.port, testCase));
  const report = { browser: 'headless Chrome', provider: 'in-memory fake only', results };
  await writeFile(path.join(evidenceDir, 'browser-consent-results.json'), `${JSON.stringify(report, null, 2)}\n`);
  console.log(JSON.stringify(report, null, 2));
} finally {
  try { process.kill(-profile.chrome.pid, 'SIGTERM'); } catch { profile.chrome.kill('SIGTERM'); }
  await Promise.race([once(profile.chrome, 'close').catch(() => {}), delay(3000)]);
  for (let retry = 0; retry < 8; retry++) {
    try { await rm(profile.profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 }); break; }
    catch { await delay(200); }
  }
}
