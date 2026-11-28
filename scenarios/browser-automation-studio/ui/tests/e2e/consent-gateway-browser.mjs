#!/usr/bin/env node
// Browser-rendered integration against the real AI Gateway consent HTTP and
// routing Connect handlers. Gateway URL and signed principals come only from
// the fixture-only Go integration test.
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import http from 'node:http';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const repoRoot = path.resolve(uiRoot, '../../..');
const gatewayURL = process.env.CONSENT_GATEWAY_URL;
const basAPIURL = process.env.CONSENT_BAS_API_URL || '';
const basCaller = process.env.CONSENT_BAS_CALLER || (basAPIURL ? 'bas' : '');
const aliceToken = process.env.CONSENT_ALICE_TOKEN;
const bobToken = process.env.CONSENT_BOB_TOKEN;
const chromeBin = process.env.CHROME_BIN || '/usr/bin/google-chrome';
const evidenceDir = process.env.CONSENT_BROWSER_EVIDENCE_DIR || path.resolve(repoRoot, '../.vrooli/research/topics/web-console-monetization/runs/models-monetization-music-20261115');
assert.ok(gatewayURL && aliceToken && bobToken, 'fixture Gateway URL and fixture-only signed tokens are required');

function port() { return new Promise((resolve, reject) => { const s = http.createServer(); s.once('error', reject); s.listen(0, '127.0.0.1', () => { const p = s.address().port; s.close(() => resolve(p)); }); }); }
async function until(check, timeout, description) { const end = Date.now() + timeout; while (Date.now() < end) { try { if (await check()) return; } catch {} await delay(100); } throw new Error(description); }
function cdpClient(ws) {
  const pending = new Map(); let nextID = 0; const events = [];
  ws.addEventListener('message', event => { const packet = JSON.parse(event.data); if (packet.id && pending.has(packet.id)) { const job = pending.get(packet.id); pending.delete(packet.id); packet.error ? job.reject(new Error(packet.error.message)) : job.resolve(packet.result); } else if (packet.method && events.length < 100) events.push(packet); });
  return { events, call(method, params = {}) { const id = ++nextID; return new Promise((resolve, reject) => { const timer = setTimeout(() => { pending.delete(id); reject(new Error(`CDP timeout: ${method}`)); }, 15000); pending.set(id, { resolve: value => { clearTimeout(timer); resolve(value); }, reject }); ws.send(JSON.stringify({ id, method, params })); }); }, async eval(expression) { const r = await this.call('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true, userGesture: true }); if (r.exceptionDetails) throw new Error(r.exceptionDetails.text); return r.result?.value; } };
}
async function stopGroup(child) { try { process.kill(-child.pid, 'SIGTERM'); } catch { child.kill('SIGTERM'); } await Promise.race([once(child, 'close').catch(() => {}), delay(3000)]); }
const vitePort = await port(); const frontPort = await port();
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', String(vitePort), '--strictPort'], { cwd: uiRoot, stdio: 'ignore', detached: true });
const proxy = http.createServer((req, res) => {
  const toBAS = basAPIURL && (req.url.includes('browser_automation_studio.v1.ai.AIService/') || req.url.includes('browser_automation_studio.v1.ai.VisionNavigationService/') || req.url.startsWith('/__fixture/bas/'));
  const toGateway = req.url.startsWith('/api/v1/ai/privacy-consent') || req.url.startsWith('/__fixture/') || req.url.includes('ai_gateway.v1.routing.RoutingService/');
  const upstreamURL = toBAS ? new URL(req.url, basAPIURL) : toGateway ? new URL(req.url, gatewayURL) : new URL(req.url, `http://127.0.0.1:${vitePort}`);
  const upstream = http.request(upstreamURL, { method: req.method, headers: { ...req.headers, host: upstreamURL.host } }, response => { res.writeHead(response.statusCode || 500, response.headers); response.pipe(res); });
  upstream.on('error', error => { res.writeHead(502); res.end(error.message); }); req.pipe(upstream);
});
proxy.listen(frontPort, '127.0.0.1'); await once(proxy, 'listening');
let chrome; let profile; let tab;
try {
  await until(async () => (await fetch(`http://127.0.0.1:${vitePort}/`)).ok, 30000, 'BAS Vite did not start');
  const fixtureTransform = await fetch(`http://127.0.0.1:${vitePort}/tests/e2e/consent-gateway-fixture.tsx`);
  if (!fixtureTransform.ok) throw new Error(`Vite failed to transform browser fixture (${fixtureTransform.status}): ${(await fixtureTransform.text()).slice(0, 4000)}`);
  profile = await mkdtemp(path.join(tmpdir(), 'consent-gateway-chrome-'));
  chrome = spawn(chromeBin, ['--headless=new', '--no-sandbox', '--disable-dev-shm-usage', '--remote-debugging-port=0', `--user-data-dir=${profile}`, 'about:blank'], { stdio: 'ignore', detached: true });
  const active = path.join(profile, 'DevToolsActivePort'); let browserPort;
  await until(async () => { try { browserPort = Number((await readFile(active, 'utf8')).split('\n')[0]); return Boolean(browserPort); } catch { return false; } }, 15000, 'Chrome failed to start');
  await until(async () => (await fetch(`http://127.0.0.1:${browserPort}/json/version`)).ok, 5000, 'Chrome DevTools endpoint unavailable');
  const target = await (await fetch(`http://127.0.0.1:${browserPort}/json/new?about:blank`, { method: 'PUT' })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl); await new Promise((resolve, reject) => { ws.addEventListener('open', resolve, { once: true }); ws.addEventListener('error', reject, { once: true }); });
  const cdp = cdpClient(ws);
  await cdp.call('Page.enable'); await cdp.call('Runtime.enable'); await cdp.call('Log.enable'); await cdp.call('Network.enable');
  const callerQuery = basAPIURL ? `&caller=${encodeURIComponent(basCaller)}` : '';
  await cdp.call('Page.navigate', { url: `http://127.0.0.1:${frontPort}/tests/e2e/consent-gateway-fixture.html?fixtureToken=${encodeURIComponent(aliceToken)}${callerQuery}` });
  try { await until(async () => await cdp.eval(`Boolean(document.querySelector('[data-testid="ai-privacy-consent"]'))`), 30000, 'real BAS consent UI did not render'); }
  catch (error) { throw new Error(`${error.message}; browser state=${await cdp.eval(`JSON.stringify({url:location.href,title:document.title,body:document.body?.innerText,errors:window.__errors||[],scripts:[...document.scripts].map(s=>s.src)})`)}; CDP=${JSON.stringify(cdp.events.filter(e => ['Runtime.exceptionThrown','Log.entryAdded'].includes(e.method) || (e.method === 'Network.responseReceived' && e.params.response.status >= 400)))}`); }
  if (basCaller === 'vision') await until(async () => await cdp.eval(`Boolean(document.querySelector('[data-testid="send-ai"]') && !document.querySelector('[data-testid="send-ai"]').disabled)`), 60000, 'fixture Playwright session was not ready');
  const click = async id => { assert.equal(await cdp.eval(`(()=>{const el=document.querySelector('[data-testid="${id}"]');if(!el)return false;el.click();return true})()`), true, `missing UI control ${id}`); };
  const stats = async () => await (await fetch(`http://127.0.0.1:${frontPort}/__fixture/stats`, { headers: { Authorization: `Bearer ${aliceToken}` } })).json();
  const setToken = token => cdp.eval(`sessionStorage.setItem('vrooli.web.access-token',JSON.stringify({accessToken:${JSON.stringify(token)},expiresAt:new Date(Date.now()+3600000).toISOString()}))`);
  const setReceipt = (receipt, source = 'own-key') => cdp.eval(`sessionStorage.setItem('bas.ai.remote-consent-receipt',${JSON.stringify(receipt)});sessionStorage.setItem('bas.ai.remote-consent-source',${JSON.stringify(source)})`);
  const chooseConsentSource = async source => cdp.eval(`(()=>{const el=document.querySelector('[data-testid="ai-privacy-consent-source"]');if(!el)return false;const setter=Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype,'value').set;setter.call(el,${JSON.stringify(source)});el.dispatchEvent(new Event('change',{bubbles:true}));return true})()`);
  const attempt = async (name, token, receipt, source = 'own-key', privacy = 'confidential') => {
    if (token) await setToken(token); await setReceipt(receipt, source);
    const before = await stats(); await click('send-ai');
    try {
      await until(async () => /^(Rejected|Dispatched|Navigation)/.test(String(await cdp.eval(`document.querySelector('[data-testid="request-result"]')?.textContent||''`))), 45000, `${name}: route response not shown`);
    } catch (error) {
      const state = await cdp.eval(`JSON.stringify({result:document.querySelector('[data-testid="request-result"]')?.textContent,buttonDisabled:document.querySelector('[data-testid="send-ai"]')?.disabled,session:sessionStorage.getItem('a15.fixture.vision-session'),body:document.body?.innerText})`);
      throw new Error(`${error.message}; state=${state}; network=${JSON.stringify(cdp.events.filter(e => e.method === 'Network.responseReceived' && e.params.response.status >= 400).map(e => ({url:e.params.response.url,status:e.params.response.status})))}`);
    }
    const text = await cdp.eval(`document.querySelector('[data-testid="request-result"]')?.textContent||''`);
    if (basCaller === 'vision' && name === 'valid-own-key') {
      try {
        await until(async () => (await stats()).dispatches === before.dispatches + 1, 60000, `${name}: sidecar did not dispatch the selected fake provider`);
      } catch (error) {
        throw new Error(`${error.message}; stats=${JSON.stringify(await stats())}; navigation=${text}`);
      }
    } else if (basCaller === 'vision' && privacy === 'confidential') {
      // The sidecar performs the Gateway request asynchronously after the API
      // has returned its navigation ID. Allow the local refusal path to settle.
      await delay(1200);
    }
    const after = await stats();
    const validVisionDispatch = basCaller === 'vision' && name === 'valid-own-key';
    assert.equal(after.dispatches, before.dispatches + (text.startsWith('Dispatched:') || validVisionDispatch ? 1 : 0), `${name}: dispatch counter changed unexpectedly`);
    if (privacy === 'confidential' && name !== 'valid-own-key') assert.equal(after.dispatches, before.dispatches, `${name}: must be refused before provider dispatch`);
    return { name, text, provider: text.match(/ via ([a-z0-9-]+)/i)?.[1]?.toLowerCase() || '', before: before.dispatches, after: after.dispatches };
  };
  const results = [];
  results.push(await attempt('missing', aliceToken, ''));
  await click('ai-privacy-consent-allow');
  await until(async () => Boolean(await cdp.eval(`sessionStorage.getItem('bas.ai.remote-consent-receipt')`)), 8000, 'grant receipt was not stored');
  const ownReceipt = await cdp.eval(`sessionStorage.getItem('bas.ai.remote-consent-receipt')`);
  await chooseConsentSource('vrooli-credits'); await click('ai-privacy-consent-allow');
  await until(async () => Boolean(await cdp.eval(`sessionStorage.getItem('bas.ai.remote-consent-receipt')`)) && String(await cdp.eval(`sessionStorage.getItem('bas.ai.remote-consent-source')`)) === 'vrooli-credits', 8000, 'credits grant was not stored');
  const creditsReceipt = await cdp.eval(`sessionStorage.getItem('bas.ai.remote-consent-receipt')`);
  await chooseConsentSource('own-key'); await setReceipt(ownReceipt, 'own-key');
  await cdp.call('Page.reload'); await until(async () => await cdp.eval(`Boolean(document.querySelector('[data-testid="ai-privacy-consent"]'))`), 20000, 'UI did not reload');
  await until(async () => String(await cdp.eval(`document.querySelector('[data-testid="ai-privacy-consent"] [role="status"]')?.textContent||''`)).includes('receipt is active'), 8000, 'active receipt did not survive browser reload');
  const expired = await (await fetch(`http://127.0.0.1:${frontPort}/__fixture/expired`, { method: 'POST', headers: { Authorization: `Bearer ${aliceToken}` } })).json();
  results.push(await attempt('forged', aliceToken, 'fixture-forged-receipt'));
  results.push(await attempt('wrong-user', bobToken, ownReceipt));
  results.push(await attempt('wrong-source', aliceToken, creditsReceipt, 'own-key'));
  results.push(await attempt('expired', aliceToken, expired.receipt));
  results.push(await attempt('valid-own-key', aliceToken, ownReceipt));
  if (basCaller === 'vision') {
    assert.equal(results.at(-1).after, results.at(-1).before + 1, `valid exact-source vision request did not dispatch; fixture stats=${JSON.stringify(await stats())}; result=${results.at(-1).text}`);
  } else {
    assert.match(results.at(-1).text, /Dispatched: succeeded/, `valid exact-source request refused; fixture stats=${JSON.stringify(await stats())}; result=${results.at(-1).text}`);
  }
  const beforeSecret = await stats();
  await mkdir(evidenceDir, { recursive: true });
  const screenshot = await cdp.call('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true });
  const screenshotStem = basCaller === 'vision' ? 'browser-consent-bas-playwright-sidecar' : basAPIURL ? 'browser-consent-bas-production-caller' : 'browser-consent-gateway-real';
  const screenshotPath = path.join(evidenceDir, `${screenshotStem}.png`);
  await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'));
  await cdp.eval(`document.querySelector('[data-testid="secret-toggle"]').click()`);
  results.push(await attempt('secret', aliceToken, ownReceipt, 'own-key', 'secret'));
  assert.equal((await stats()).dispatches, beforeSecret.dispatches, 'SECRET must not dispatch');
  await cdp.eval(`document.querySelector('[data-testid="secret-toggle"]').click()`);
  await setToken(aliceToken); await setReceipt(ownReceipt);
  try { await until(async () => await cdp.eval(`Boolean(document.querySelector('[data-testid^="ai-privacy-consent-revoke-"]'))`), 10000, 'persisted grant did not reload into the consent list'); }
  catch (error) { const list = await fetch(`http://127.0.0.1:${frontPort}/api/v1/ai/privacy-consent`, { headers: { Authorization: `Bearer ${aliceToken}` } }).then(r => r.text()); throw new Error(`${error.message}; panel=${await cdp.eval(`document.querySelector('[data-testid="ai-privacy-consent"]')?.innerText`)}; list=${list}`); }
  const persistedGrants = await fetch(`http://127.0.0.1:${frontPort}/api/v1/ai/privacy-consent`, { headers: { Authorization: `Bearer ${aliceToken}` } }).then(r => r.json());
  const ownGrant = persistedGrants.grants.find(grant => grant.source === 'own-key' && !grant.revoked_at && Date.parse(grant.expires_at) > Date.now());
  assert.ok(ownGrant, 'own-key grant must persist after browser reload');
  await until(async () => await cdp.eval(`Boolean([...document.querySelectorAll('[data-testid^="ai-privacy-consent-revoke-"]')].find(el=>el.closest('li')?.textContent.includes('your configured OpenRouter account')))`), 10000, 'persisted own-key grant must render a revoke action after reload');
  const revokeButton = await cdp.eval(`[...document.querySelectorAll('[data-testid^="ai-privacy-consent-revoke-"]')].find(el=>el.closest('li')?.textContent.includes('your configured OpenRouter account'))?.dataset.testid||''`);
  const grantID = revokeButton.slice('ai-privacy-consent-revoke-'.length);
  assert.ok(grantID, 'own-key revoke control should identify its grant');
  await click(revokeButton); await until(async () => String(await cdp.eval(`document.querySelector('[data-testid="ai-privacy-consent"] [role="status"]')?.textContent||''`)).includes('No consent receipt'), 10000, 'revoke did not clear receipt');
  results.push(await attempt('revoked', aliceToken, ownReceipt));
  await writeFile(path.join(evidenceDir, `${screenshotStem}-results.json`), `${JSON.stringify({ provider: 'fake OpenRouter only', gateway: 'real HTTP and Connect handlers; fixture SQLite', basAPI: basAPIURL ? `real BAS ${basCaller} + production Gateway client; opt-in loopback fixture server` : null, principal: 'locally signed synthetic fixture token', grantID, screenshot: screenshotPath, results, finalStats: await stats() }, null, 2)}\n`);
  console.log(JSON.stringify({ grantID, results, finalStats: await stats() }, null, 2));
  ws.close();
  await fetch(`http://127.0.0.1:${browserPort}/json/close/${target.id}`, { method: 'PUT' }).catch(() => {});
} finally {
  if (chrome) await stopGroup(chrome);
  await stopGroup(vite);
  await new Promise(resolve => proxy.close(resolve));
  if (profile) await rm(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 }).catch(() => {});
}
