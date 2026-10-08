import { readFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { randomUUID } from 'node:crypto';

const layout = JSON.parse(readFileSync(new URL('./layout.json', import.meta.url), 'utf8'));
const box = (name) => Object.entries(layout[name]).map(([key, value]) => `${key}:${value}px`).join(';');
const html = (body) => `<!doctype html><html lang="en"><meta charset="utf-8"><title>BAS journey fixture</title><body>${body}</body></html>`;
const readBody = (request) => new Promise((resolve) => {
  let body = '';
  request.setEncoding('utf8');
  request.on('data', (chunk) => { body += chunk; });
  request.on('end', () => resolve(body));
});

const FORMS = html(`<h1>Journey form</h1><form>
  <label>Name <input name="name"></label>
  <label>Notes <textarea name="notes"></textarea></label>
</form>`);

const SAME = html('<button id="same" onclick="document.body.dataset.clicked=location.pathname">same selector</button>');
const FRAME_HOST = html('<h1>Frames</h1><iframe id="fixture-frame" src="/frame-child"></iframe>');
const GESTURES = html(`<h1>Gestures</h1><input id="shortcut" value="start"><div id="drag-source" draggable="true" style="position:absolute;left:20px;top:80px;width:100px;height:40px;background:#ddd">source</div><div id="drag-target" style="position:absolute;left:180px;top:80px;width:120px;height:40px;background:#eee" ondragover="event.preventDefault()" ondrop="this.dataset.dropped='yes';fetch('/effect',{method:'POST'})">target</div><div style="height:1600px"></div><div id="scroll-end">end</div><script>document.querySelector('#shortcut').addEventListener('input',event=>fetch('/input-observation',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({type:'field',value:event.target.value})}));window.addEventListener('scroll',()=>fetch('/scroll-observation',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({y:window.scrollY})}));</script>`);
const COMPLEX = html(`<h1>Browser behaviors</h1><button id="spa" onclick="history.pushState({},'', '/spa/next');document.querySelector('h1').textContent='SPA navigated'">SPA</button><button id="popup" onclick="window.open('/same-popup','fixture-popup')">Popup</button><div id="shadow-host"></div><script>const root=document.querySelector('#shadow-host').attachShadow({mode:'open'});const button=document.createElement('button');button.id='shadow-button';button.textContent='Shadow';button.addEventListener('click',()=>button.dataset.clicked='yes');root.append(button);</script>`);
const SERVICE_WORKER = `self.addEventListener('install', event => event.waitUntil(self.skipWaiting())); self.addEventListener('activate', event => event.waitUntil(self.clients.claim())); self.addEventListener('fetch', event => { if (new URL(event.request.url).pathname === '/worker-probe') event.respondWith(new Response('worker-ok')); });`;
const SW_PAGE = html(`<h1>Service worker fixture</h1><output id="worker-result">pending</output><script>navigator.serviceWorker.register('/service-worker.js').then(async () => { await navigator.serviceWorker.ready; if (!navigator.serviceWorker.controller) await new Promise(resolve => navigator.serviceWorker.addEventListener('controllerchange', resolve, {once:true})); const response = await fetch('/worker-probe'); const output=document.querySelector('#worker-result'); output.textContent=await response.text(); output.classList.add(output.textContent); }).catch(error => document.querySelector('#worker-result').textContent = String(error));</script>`);

const HOME = html(`<h1>BAS journey fixture</h1>
  <input id="fixture-input" value="initial" aria-label="Recordable text" style="position:absolute;${box('input')}">
  <button id="paste" type="button" style="position:absolute;${box('paste')}">Paste fixture text</button>
  <button id="compose" type="button" style="position:absolute;${box('compose')}">Compose fixture text</button>
  <button id="counter" type="button" style="position:absolute;${box('counter')}" onclick="fetch('/effect',{method:'POST'}).then(r=>r.json()).then(r=>this.textContent='Effect '+r.count)">Click fixture</button>
  <script>
    const input = document.querySelector('#fixture-input');
    const report = (value) => fetch('/input-observation', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({type:'field', value})});
    input.addEventListener('input', () => { void report(input.value); });
    const setValue = (value, events) => { input.focus(); input.value = value; for (const event of events) input.dispatchEvent(event); };
    document.querySelector('#paste').addEventListener('click', () => setValue('pasted value', [
      new InputEvent('input', {bubbles:true, data:'pasted value', inputType:'insertFromPaste'})]));
    document.querySelector('#compose').addEventListener('click', () => setValue('東京', [
      new CompositionEvent('compositionstart', {bubbles:true, data:''}),
      new CompositionEvent('compositionupdate', {bubbles:true, data:'東京'}),
      new InputEvent('input', {bubbles:true, data:'東京', inputType:'insertCompositionText', isComposing:true}),
      new CompositionEvent('compositionend', {bubbles:true, data:'東京'}),
      new InputEvent('input', {bubbles:true, data:'東京', inputType:'insertFromComposition', isComposing:false})]));
  </script>`);

// Fixture-owned fingerprint observation contract:
// - webdriver: navigator.webdriver as a boolean
// - language/languages: the active browser's navigator language values
// - timezone: Intl resolved timezone identifier
// - hardwareConcurrency: positive integer exposed by navigator
// These values come from the active browser, not an external fingerprint service
// or browser/version allowlist. The journey validates observable types and shape.
const FINGERPRINT = html(`<h1>Local browser observations</h1><output id="fingerprint" aria-label="Browser observations"></output><script>
  const observations = {
    webdriver: navigator.webdriver,
    language: navigator.language,
    languages: [...navigator.languages],
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    hardwareConcurrency: navigator.hardwareConcurrency,
  };
  document.querySelector('#fingerprint').textContent = JSON.stringify(observations);
  void fetch('/fingerprint-observation', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(observations)});
</script>`);

/**
 * Start the BAS journey fixture on an ephemeral loopback port. It owns the
 * independent oracle for journey effects: every effect, held-request release
 * and observed field value is recorded here, never by BAS itself.
 */
export async function startJourneySite() {
  const effects = [];
  const released = [];
  const inputs = [];
  const scrolls = [];
  const fingerprints = [];
  let retryAttempts = 0;
  const counts = { effect: () => effects.length, release: () => released.length, input: () => inputs.length, scroll: () => scrolls.length, fingerprint: () => fingerprints.length };
  const waiters = new Set();
  const snapshot = () => ({ effects, released, inputs, scrolls, fingerprints, retryAttempts });
  const notify = () => { for (const wake of waiters) wake(); };
  const record = (entry) => { effects.push({ sequence: effects.length + 1, ...entry }); notify(); return effects.length; };
  const json = (response, status, body) => response.writeHead(status, { 'Content-Type': 'application/json' }).end(JSON.stringify(body));

  const server = createServer(async (request, response) => {
    const url = new URL(request.url ?? '/', 'http://127.0.0.1');
    const context = request.headers.cookie ?? '';
    switch (`${request.method} ${url.pathname}`) {
      case 'GET /effect/hold': {
        // Held open until the browser drops it; the release is the cancellation oracle.
        const sequence = record({ context, held: true });
        response.on('close', () => { if (!response.writableEnded) { released.push({ sequence }); notify(); } });
        return;
      }
      case 'POST /effect':
        json(response, 200, { count: record({ context }) });
        return;
      case 'POST /input-observation':
        inputs.push({ ...JSON.parse(await readBody(request)), context });
        notify();
        response.writeHead(204).end();
        return;
      case 'POST /scroll-observation':
        scrolls.push(JSON.parse(await readBody(request)));
        notify();
        response.writeHead(204).end();
        return;
      case 'POST /fingerprint-observation':
        fingerprints.push(JSON.parse(await readBody(request)));
        notify();
        response.writeHead(204).end();
        return;
      case 'GET /fingerprint-state':
        json(response, 200, fingerprints);
        return;
      case 'GET /journey-state':
        json(response, 200, snapshot());
        return;
      case 'GET /journey-wait': {
        // Long-poll until more than `after` records of `kind` exist, so journeys never sleep.
        const count = counts[url.searchParams.get('kind')];
        const after = Number(url.searchParams.get('after') ?? 0);
        if (!count || !Number.isSafeInteger(after) || after < 0) return json(response, 400, snapshot());
        const timeoutMs = Math.max(1, Math.min(10000, Number(url.searchParams.get('timeout_ms') ?? 5000)));
        if (count() > after) return json(response, 200, snapshot());
        const wake = () => {
          if (count() <= after) return;
          clearTimeout(timer);
          waiters.delete(wake);
          json(response, 200, snapshot());
        };
        const timer = setTimeout(() => { waiters.delete(wake); json(response, 408, snapshot()); }, timeoutMs);
        waiters.add(wake);
        return;
      }
      case 'GET /forms':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(FORMS);
        return;
      case 'GET /same': case 'GET /same-popup': case 'GET /redirect-target':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(SAME);
        return;
      case 'GET /frames':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(FRAME_HOST);
        return;
      case 'GET /frame-child':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(SAME);
        return;
      case 'GET /gestures':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(GESTURES);
        return;
      case 'GET /complex':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(COMPLEX);
        return;
      case 'GET /service-worker':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(SW_PAGE);
        return;
      case 'GET /fingerprint':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(FINGERPRINT);
        return;
      case 'GET /redirect':
        response.writeHead(302, { Location: '/redirect-target' }).end();
        return;
      case 'GET /service-worker.js':
        response.writeHead(200, { 'Content-Type': 'text/javascript', 'Service-Worker-Allowed': '/' }).end(SERVICE_WORKER);
        return;
      case 'GET /retry-page':
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(html(`<h1>Retry fixture</h1><script>setTimeout(() => { const button=document.createElement('button'); button.id='retry-target'; button.textContent='Ready'; button.onclick=()=>fetch('/effect',{method:'POST'}); document.body.append(button); }, 3000);</script>`));
        return;
      case 'GET /retry-once':
        retryAttempts += 1;
        if (retryAttempts === 1) { response.socket.destroy(); return; }
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' }).end(FORMS);
        return;
      default: {
        // Each fresh browser context gets its own identity, so replay contexts are distinguishable.
        const headers = { 'Content-Type': 'text/html; charset=utf-8' };
        if (!/(^|;\s*)fixture_context=/.test(context)) headers['Set-Cookie'] = `fixture_context=${randomUUID()}; Path=/; SameSite=Lax`;
        response.writeHead(200, headers).end(HOME);
      }
    }
  });

  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('Journey site did not bind to a TCP port');
  return {
    origin: `http://127.0.0.1:${address.port}`,
    close: () => new Promise((resolve, reject) => {
      server.closeAllConnections();
      server.close((error) => error ? reject(error) : resolve());
    }),
  };
}
