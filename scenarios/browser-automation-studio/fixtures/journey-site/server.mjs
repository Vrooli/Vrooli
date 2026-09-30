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

/**
 * Start the BAS journey fixture on an ephemeral loopback port. It owns the
 * independent oracle for journey effects: every effect, held-request release
 * and observed field value is recorded here, never by BAS itself.
 */
export async function startJourneySite() {
  const effects = [];
  const released = [];
  const inputs = [];
  const counts = { effect: () => effects.length, release: () => released.length, input: () => inputs.length };
  const waiters = new Set();
  const snapshot = () => ({ effects, released, inputs });
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
