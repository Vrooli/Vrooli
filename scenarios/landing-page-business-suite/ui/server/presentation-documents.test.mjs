import test from 'node:test';
import assert from 'node:assert/strict';
import { once } from 'node:events';
import { JSDOM } from 'jsdom';
import { canonicalBase, createPresentationDocuments, injectPresentationHead, presentationHead, presentationVisitor } from './presentation-documents.mjs';
import { createLandingServer } from '../server.js';
import { productMarkPaths } from '../src/surfaces/public-landing/presentation/productMarks.js';

const template = '<!doctype html><html lang="en"><head><title>Legacy</title><meta name="description" content="Legacy"><meta property="og:title" content="Legacy"><meta property="og:image" content="legacy.png"><meta name="twitter:title" content="Legacy"><link rel="canonical" href="https://wrong.test/"><meta charset="utf-8"><script src="/assets/app.js"></script></head><body><div id="root"></div></body></html>';
function fixture(route = '/', revision = 'a'.repeat(64), variant = '') {
  return { presentation: {
    mode: route === '/' ? 'single_app' : 'app_detail',
    page: { title: 'Aquila — a place for your agents', description: 'Voice, files & sessions.', locale: 'en', display: { shell: { brandName: 'Aquila' } }, theme: { background: '#faf8f2' } },
    diagnostics: { requestedRoute: route, resolvedRoute: route, requestedVariant: variant, resolvedVariant: variant || 'control', resolvedRevision: revision, blockDigest: 'b'.repeat(64), locale: 'en' },
  } };
}
const branding = { branding: { canonicalBaseUrl: 'https://suite.example/base/' } };
function responseRecorder() {
  return { headers: {}, statusCode: 200, body: '', setHeader(name, value) { this.headers[name] = value; }, status(value) { this.statusCode = value; return this; }, type(value) { this.contentType = value; return this; }, send(value) { this.body = value; return this; } };
}
function harness(respond = (_url, init) => fixture(JSON.parse(init.body).route)) {
  const calls = [];
  const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => template,
    fetchImplementation: async (url, init) => { calls.push({ url, init }); return new Response(JSON.stringify(url.includes('GetPublicBranding') ? branding : respond(url, init)), { headers: { 'content-type': 'application/json' } }); },
  });
  return { calls, async request(url, method = 'GET') { const res = responseRecorder(); let next = false; await middleware({ url, originalUrl: url, method }, res, () => { next = true; }); return { ...res, next }; } };
}

test('LP-PRES-011: root and app metadata follow the public revision before JavaScript', async () => {
  const run = harness();
  for (const route of ['/', '/apps/aquila']) {
    const result = await run.request(route);
    assert.equal(result.statusCode, 200); assert.equal(result.headers['Cache-Control'], 'private, no-store');
    const doc = new JSDOM(result.body).window.document;
    assert.equal(doc.title, 'Aquila — a place for your agents');
    assert.equal(doc.querySelectorAll('title').length, 1);
    assert.equal(doc.querySelectorAll('meta[property="og:title"]').length, 1);
    assert.equal(doc.querySelector('meta[property="og:image"]'), null);
    assert.equal(doc.querySelector('link[rel="canonical"]').href, `https://suite.example/base${route}`);
    const boot = JSON.parse(doc.querySelector('#lpbs-presentation-bootstrap').textContent);
    assert.equal(boot.request.route, route); assert.equal(boot.config.presentation.diagnostics.resolvedRevision, 'a'.repeat(64));
    assert.equal(boot.canonicalBaseUrl, 'https://suite.example/base');
  }
  for (const { url, init } of run.calls) {
    assert.equal(new URL(url).hostname, '127.0.0.1'); assert.equal(init.cache, 'no-store');
    assert.equal(init.headers.authorization, undefined); assert.equal(init.headers.cookie, undefined);
    if (url.includes('GetLandingConfig')) assert.match(JSON.parse(init.body).visitorId, /^[A-Za-z0-9_-]{1,128}$/);
  }
});

test('LP-PRES-009: admin and unknown paths never request a public assignment', async () => {
  const run = harness();
  for (const route of ['/admin/presentation/control', '/login', '/checkout', '/feedback', '/not-a-page']) {
    const result = await run.request(route);
    assert.equal(result.statusCode, route === '/not-a-page' ? 404 : 200);
    assert.equal(result.headers['X-Robots-Tag'], 'noindex, nofollow');
    assert.doesNotMatch(result.body, /lpbs-presentation-bootstrap|rel="canonical"|Legacy/);
  }
  assert.equal(run.calls.length, 0);
  for (const route of ['/assets/app.js', '/public/icon.svg', '/api/health', '/health', '/config', '/embedded/app']) assert.equal((await run.request(route)).next, true);
  assert.equal((await run.request('/', 'POST')).next, true);
  assert.equal(run.calls.length, 0);
});

test('LP-PRES-009: missing/private app remains a real noindex 404, outages a 503', async () => {
  for (const status of [404, 500, 503]) {
    const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => template, fetchImplementation: async () => new Response('{}', { status }) });
    const res = responseRecorder(); await middleware({ method: 'GET', url: '/apps/private' }, res, () => assert.fail('unexpected next'));
    assert.equal(res.statusCode, status === 404 ? 404 : 503);
    assert.equal(res.headers['X-Robots-Tag'], 'noindex, nofollow');
    assert.doesNotMatch(res.body, /lpbs-presentation-bootstrap|rel="canonical"|Legacy/);
  }
});

test('LP-PRES-011: rejects stale scope, preview, fallback and incomplete diagnostics', async () => {
  const invalid = [
    p => { p.presentation.diagnostics.preview = true; }, p => { p.fallback = true; },
    p => { p.presentation.diagnostics.resolvedRoute = '/apps/other'; }, p => { p.presentation.diagnostics.requestedRoute = '/'; },
    p => { p.presentation.diagnostics.requestedVariant = 'wrong'; }, p => { p.presentation.diagnostics.resolvedRevision = ''; },
    p => { p.presentation.diagnostics.blockDigest = ''; }, p => { p.presentation.page.locale = 'fr'; },
    p => { p.presentation.mode = 'bundle'; }, p => { delete p.presentation; },
  ];
  for (const mutate of invalid) {
    const config = fixture('/apps/aquila'); mutate(config);
    const result = await harness(() => config).request('/apps/aquila');
    assert.equal(result.statusCode, 503); assert.doesNotMatch(result.body, /lpbs-presentation-bootstrap/);
  }
});

test('LP-PRES-011: no shared metadata cache and no query-variant indexing', async () => {
  let revision = 'a'.repeat(64);
  const run = harness((_url, init) => { const req = JSON.parse(init.body); return fixture(req.route, revision, req.variantSlug); });
  const first = await run.request('/'); revision = 'c'.repeat(64);
  const second = await run.request('/?variant=alternate');
  assert.match(first.body, new RegExp('content="' + 'a'.repeat(64) + '"'));
  assert.match(second.body, new RegExp('content="' + 'c'.repeat(64) + '"'));
  assert.equal(second.headers['X-Robots-Tag'], 'noindex, nofollow');
  const doc = new JSDOM(second.body).window.document;
  assert.equal(doc.querySelector('link[rel="canonical"]').href, 'https://suite.example/base/');
});

test('LP-PRES-011: canonical base is configured authority, not executable or request input', () => {
  for (const value of ['', 'javascript:alert(1)', '//evil.test', 'https://user:pass@test/', 'https://test/?x=1', 'https://test/#hash', 'https://test/\\evil']) assert.equal(canonicalBase(value), '');
  assert.equal(canonicalBase('https://suite.example/base/'), 'https://suite.example/base');
  const head = presentationHead(fixture(), { route: '/', variant: '', locale: '' }, {});
  assert.equal(head.canonical, '');
  assert.throws(() => createPresentationDocuments({ apiBase: 'https://attacker.test', readIndex: async () => template }));
});

test('LP-PRES-011: untrusted copy cannot escape head or bootstrap script context', () => {
  const attack = '</script><script>globalThis.compromised=true</script> & " \u2028';
  const head = { title: attack, description: attack, locale: 'en', noindex: true };
  const html = injectPresentationHead(template, head, { text: attack });
  const doc = new JSDOM(html).window.document;
  assert.equal(doc.title, attack);
  assert.equal(doc.querySelectorAll('script').length, 2); // existing app script + inert JSON
  assert.equal(JSON.parse(doc.querySelector('#lpbs-presentation-bootstrap').textContent).text, attack);
  assert.equal(doc.querySelector('meta[name="description"]').content, attack);
});

test('LP-PRES-011: real scenario server handles root before static index and nested routes identically', async t => {
  const priorEnvironment = process.env.NODE_ENV;
  process.env.NODE_ENV = 'test'; // api-base's explicit isolated-test entry, not a live lifecycle bypass.
  t.after(() => { if (priorEnvironment === undefined) delete process.env.NODE_ENV; else process.env.NODE_ENV = priorEnvironment; });
  const app = createLandingServer({ uiPort: '9875', apiPort: '9876', fetchImplementation: async (url, init) => new Response(JSON.stringify(url.includes('GetPublicBranding') ? branding : fixture(JSON.parse(init.body).route))) });
  const server = app.listen(0, '127.0.0.1'); await once(server, 'listening');
  t.after(() => new Promise(resolve => { server.closeAllConnections(); server.close(resolve); }));
  for (const route of ['/', '/apps/aquila']) {
    const response = await fetch(`http://127.0.0.1:${server.address().port}${route}`);
    assert.equal(response.status, 200); assert.equal(response.headers.get('cache-control'), 'private, no-store');
    const doc = new JSDOM(await response.text()).window.document;
    assert.equal(doc.title, 'Aquila — a place for your agents');
    assert.equal(doc.querySelector('link[rel="canonical"]').href, `https://suite.example/base${route}`);
  }
});

test('LP-PRES-011: crawler documents reach exact owner paths, never the SPA or request Host', async () => {
  const calls = [];
  const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => assert.fail('crawler requested SPA'),
    fetchImplementation: async (url, init) => { calls.push({ url, init }); return new Response(url.endsWith('/robots.txt') ? 'User-agent: *\nDisallow: /admin/\n' : '<?xml version="1.0"?><urlset/>'); },
  });
  for (const route of ['/robots.txt', '/sitemap.xml']) {
    const res = responseRecorder();
    await middleware({ method: 'GET', url: `${route}?host=https://attacker.test`, headers: { host: 'attacker.test', authorization: 'private-token' } }, res, () => assert.fail('crawler passed through'));
    assert.equal(res.statusCode, 200); assert.equal(res.headers['Cache-Control'], 'no-store');
    assert.equal(calls.at(-1).url, `http://127.0.0.1:9876${route}`);
    assert.equal(calls.at(-1).init.method, 'GET'); assert.equal(calls.at(-1).init.headers, undefined);
  }
});

test('LP-PRES-011: oversized owner payload and crawler outage fail closed', async () => {
  const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => template,
    fetchImplementation: async () => new Response('{}', { headers: { 'content-length': String(9 * 1024 * 1024) } }),
  });
  for (const route of ['/', '/robots.txt', '/sitemap.xml']) {
    const res = responseRecorder(); await middleware({ method: 'GET', url: route }, res, () => assert.fail('unexpected next'));
    assert.equal(res.statusCode, 503); assert.equal(res.headers['X-Robots-Tag'], 'noindex, nofollow');
    if (route === '/robots.txt') assert.equal(res.body, 'User-agent: *\nDisallow: /\n');
  }
});

test('LP-PRES-011: index.html cannot expose the static first-app metadata', async () => {
  const run = harness(); const response = await run.request('/index.html?variant=control');
  assert.equal(response.statusCode, 308); assert.equal(response.headers.Location, './?variant=control');
  assert.equal(run.calls.length, 0); assert.equal(response.next, false);
});

test('LP-PRES-009: direct download workflow pins its app detail and remains noindex', async () => {
  const run = harness((_url, init) => {
    const request = JSON.parse(init.body);
    assert.equal(request.route, '/apps/aquila');
    return fixture(request.route, 'a'.repeat(64), request.variantSlug);
  });
  const response = await run.request('/apps/aquila/download?locale=en&variant=control');
  assert.equal(response.statusCode, 200);
  assert.equal(response.headers['X-Robots-Tag'], 'noindex, nofollow');
  const doc = new JSDOM(response.body).window.document;
  assert.equal(doc.querySelector('link[rel="canonical"]').href, 'https://suite.example/base/apps/aquila');
  const bootstrap = JSON.parse(doc.querySelector('#lpbs-presentation-bootstrap').textContent);
  assert.deepEqual(bootstrap.request, { route: '/apps/aquila', locale: 'en', variant: 'control' });
  assert.equal(bootstrap.config.presentation.mode, 'app_detail');
  assert.equal((await run.request('/apps/aquila/download/unknown')).statusCode, 404);
});

test('LP-PRES-009: SSR and crawlers preserve only the exact isolated-routing marker', async () => {
  const calls = [];
  const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => template,
    fetchImplementation: async (url, init) => {
      calls.push(init);
      if (url.endsWith('/robots.txt')) return new Response('User-agent: *\nDisallow: /\n');
      return new Response(JSON.stringify(url.includes('GetPublicBranding') ? branding : fixture()));
    },
  });
  for (const marker of ['1', 'true', '0']) {
    for (const url of ['/', '/robots.txt']) {
      calls.length = 0;
      const res = responseRecorder();
      await middleware({ method: 'GET', url, headers: { 'x-vrooli-test-mode': marker, authorization: 'private', cookie: 'secret=value', host: 'attacker.test' } }, res, () => assert.fail('unexpected pass-through'));
      assert.equal(res.statusCode, 200);
      for (const call of calls) {
        assert.equal(call.headers?.['x-vrooli-test-mode'], marker === '1' ? '1' : undefined);
        for (const privateHeader of ['authorization', 'cookie', 'host']) assert.equal(call.headers?.[privateHeader], undefined);
      }
    }
  }
});

test('LP-PRES-009: SSR selects from the scoped anonymous cookie without forwarding private cookies', async () => {
  const calls = [];
  const middleware = createPresentationDocuments({ apiBase: 'http://127.0.0.1:9876', readIndex: async () => template,
    fetchImplementation: async (url, init) => { calls.push({ url, init }); return new Response(JSON.stringify(url.includes('GetPublicBranding') ? branding : fixture(JSON.parse(init.body).route))); },
  });
  for (const route of ['/', '/apps/aquila']) {
    const res = responseRecorder();
    await middleware({ method: 'GET', url: route, headers: { cookie: 'auth=private; metrics_visitor_id=visitor_existing-123; unrelated=secret' } }, res, () => assert.fail('unexpected next'));
    assert.equal(res.statusCode, 200); assert.equal(res.headers['Set-Cookie'], undefined);
    const bootstrap = JSON.parse(new JSDOM(res.body).window.document.querySelector('#lpbs-presentation-bootstrap').textContent);
    assert.equal(bootstrap.visitorId, 'visitor_existing-123');
    assert.doesNotMatch(res.body, /auth=private|unrelated=secret/);
  }
  for (const { url, init } of calls) {
    assert.equal(init.headers.cookie, undefined);
    if (url.includes('GetLandingConfig')) assert.equal(JSON.parse(init.body).visitorId, 'visitor_existing-123');
  }
  const res = responseRecorder();
  await middleware({ method: 'GET', url: '/', secure: true, headers: {} }, res, () => assert.fail('unexpected next'));
  assert.match(res.headers['Set-Cookie'], /^metrics_visitor_id=[a-f0-9-]+; Path=\/; SameSite=Lax; Max-Age=31536000; Secure$/);
  const bootstrap = JSON.parse(new JSDOM(res.body).window.document.querySelector('#lpbs-presentation-bootstrap').textContent);
  assert.match(res.headers['Set-Cookie'], new RegExp(`^metrics_visitor_id=${bootstrap.visitorId};`));
  for (const cookie of ['metrics_visitor_id=<script>', 'metrics_visitor_id=one; metrics_visitor_id=two', 'metrics_visitor_id=' + 'a'.repeat(129), 'metrics_visitor_id=%0aattack']) {
    const visitor = presentationVisitor({ headers: { cookie } });
    assert.equal(visitor.fresh, true); assert.match(visitor.id, /^[a-f0-9-]+$/);
  }
});

test('LP-PRES-003: each page uses its configured mark, never the static first-app install identity', () => {
  const legacy = template.replace('</head>', '<link rel="icon" href="aquila-only.svg"><link rel="apple-touch-icon" href="aquila-only.svg"><link rel="manifest" href="aquila-only.webmanifest"><meta name="apple-mobile-web-app-title" content="Aquila only"><meta name="apple-mobile-web-app-capable" content="yes"></head>');
  for (const brandMark of ['letter-a', 'suite', 'landscape']) {
    const config = fixture();
    config.presentation.page.display.shell.brandMark = brandMark;
    config.presentation.page.theme.primary = '#263e38';
    const head = presentationHead(config, { route: '/', locale: '', variant: '' }, branding.branding);
    const doc = new JSDOM(injectPresentationHead(legacy, head)).window.document;
    assert.equal(doc.querySelectorAll('link[rel="icon"]').length, 1);
    assert.equal(doc.querySelector('link[rel="manifest"]'), null);
    assert.equal(doc.querySelector('link[rel="apple-touch-icon"]'), null);
    assert.equal(doc.querySelector('meta[name="apple-mobile-web-app-capable"]'), null);
    const svg = decodeURIComponent(doc.querySelector('link[rel="icon"]').href.split(',')[1]);
    assert.ok(svg.includes(`d="${productMarkPaths[brandMark]}"`));
    assert.ok(svg.includes('stroke="#263e38"'));
    assert.doesNotMatch(doc.documentElement.outerHTML, /aquila-only/);
  }
  const privateDoc = new JSDOM(injectPresentationHead(legacy, { title: 'Private page', noindex: true })).window.document;
  assert.equal(privateDoc.querySelector('link[rel="icon"]'), null);
  assert.equal(privateDoc.querySelector('link[rel="manifest"]'), null);
});
