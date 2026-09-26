import { injectBaseTag } from '@vrooli/api-base/server';
import { randomUUID } from 'node:crypto';
import { getProductMarkPath, productMarkDrawing } from '../src/surfaces/public-landing/presentation/productMarks.js';

const LANDING = '/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig';
const BRANDING = '/landing_page_business_suite.v1.BrandingService/GetPublicBranding';
const MAX_RESPONSE_BYTES = 8 * 1024 * 1024;
const detailRoute = /^\/apps\/[a-z0-9][a-z0-9-]*$/;
const downloadRoute = /^\/apps\/[a-z0-9][a-z0-9-]*\/download$/;
const privateRoute = /^\/(?:admin(?:\/|$)|login$|auth(?:\/|$)|checkout$|feedback$)/;
const field = (value, camel, snake) => value?.[camel] ?? value?.[snake];

// The API's canonical development routing marker is the only caller header
// allowed across this boundary. Never turn an isolated request into a live
// read by dropping it. The API remains responsible for mode/lease admission.
function routingHeaders(req) {
  return req.headers?.['x-vrooli-test-mode'] === '1' ? { 'x-vrooli-test-mode': '1' } : {};
}

// This anonymous analytics identity is not authentication. Keep it separate
// from all other cookies and accept no executable/structured cookie payload.
export function presentationVisitor(req) {
  const values = String(req.headers?.cookie || '').split(';').map(part => part.trim()).filter(part => part.startsWith('metrics_visitor_id='));
  if (values.length === 1) {
    const value = values[0].slice('metrics_visitor_id='.length);
    if (/^[A-Za-z0-9_-]{1,128}$/.test(value)) return { id: value, fresh: false };
  }
  return { id: randomUUID(), fresh: true };
}

export function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

export function canonicalBase(value) {
  if (typeof value !== 'string' || !value || /[\s\\]/.test(value)) return '';
  try {
    const url = new URL(value);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.search || url.hash) return '';
    return url.origin + url.pathname.replace(/\/+$/, '');
  } catch { return ''; }
}

function scriptJSON(value) {
  return JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, c => `\\u${c.charCodeAt(0).toString(16).padStart(4, '0')}`);
}

async function boundedBody(response, maximum) {
  if (Number(response.headers.get('content-length')) > maximum) { await response.body?.cancel(); throw new Error('Public response too large'); }
  const reader = response.body?.getReader();
  if (!reader) throw new Error('Missing public response');
  const chunks = []; let size = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > maximum) { await reader.cancel(); throw new Error('Public response too large'); }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  return Buffer.concat(chunks).toString('utf8');
}

/** Bounded public Connect JSON only; never forward cookies or private auth. */
async function publicRPC(apiBase, procedure, request, fetchImplementation, routing = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 5000);
  try {
    const response = await fetchImplementation(apiBase + procedure, {
      method: 'POST', headers: { 'content-type': 'application/json', 'connect-protocol-version': '1', ...routing },
      body: JSON.stringify(request), signal: controller.signal, cache: 'no-store', redirect: 'error',
    });
    if (!response.ok) {
      const error = new Error('Public presentation owner unavailable');
      error.status = response.status === 404 ? 404 : 503;
      await response.body?.cancel();
      throw error;
    }
    return JSON.parse(await boundedBody(response, MAX_RESPONSE_BYTES));
  } finally { clearTimeout(timeout); }
}

/** Node's runtime cannot load generated TypeScript; validate only the public
 * head projection here. The browser still strictly decodes the whole generated
 * wire document. No hand-written marketing fallback or variant-wide cache. */
export function presentationHead(config, request, branding) {
  const p = config?.presentation; const page = p?.page; const d = p?.diagnostics;
  if (!page?.title || !page.description || !page.locale || !d || d.preview || config.fallback ||
      field(d, 'resolvedRoute', 'resolved_route') !== request.route ||
      field(d, 'requestedRoute', 'requested_route') !== request.route ||
      (field(d, 'requestedVariant', 'requested_variant') ?? '') !== request.variant ||
      !field(d, 'resolvedVariant', 'resolved_variant') || !field(d, 'resolvedRevision', 'resolved_revision') ||
      !field(d, 'blockDigest', 'block_digest') || d.locale !== page.locale ||
      (request.route !== '/' && p.mode !== 'app_detail')) throw new Error('Unqualified public page metadata');
  const base = canonicalBase(field(branding, 'canonicalBaseUrl', 'canonical_base_url'));
  return {
    title: page.title, description: page.description, locale: page.locale,
    siteName: field(page.display?.shell, 'brandName', 'brand_name') || '',
    brandMark: field(page.display?.shell, 'brandMark', 'brand_mark') || '',
    primary: page.theme?.primary || '',
    background: page.theme?.background || '', canonicalBaseUrl: base,
    canonical: base ? base + request.route : '', noindex: Boolean(d.noindex || request.variant),
    revision: field(d, 'resolvedRevision', 'resolved_revision'),
    variant: field(d, 'resolvedVariant', 'resolved_variant'),
  };
}

function presentationIcon(head) {
  const path = getProductMarkPath(head.brandMark);
  if (!path || !/^#[a-f0-9]{6}$/i.test(head.primary) || !/^#[a-f0-9]{6}$/i.test(head.background)) return '';
  const drawing = productMarkDrawing;
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${drawing.viewBox}"><rect width="32" height="32" rx="8" fill="${head.background}"/><path d="${path}" fill="${drawing.fill}" stroke="${head.primary}" stroke-width="${drawing.strokeWidth}" stroke-linecap="${drawing.strokeLinecap}" stroke-linejoin="${drawing.strokeLinejoin}"/></svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

export function injectPresentationHead(html, head, bootstrap) {
  const tags = [
    `<title>${escapeHtml(head.title)}</title>`,
    `<meta name="description" content="${escapeHtml(head.description)}">`,
    `<meta name="robots" content="${head.noindex ? 'noindex, nofollow' : 'index, follow'}">`,
    '<meta property="og:type" content="website">',
    `<meta property="og:title" content="${escapeHtml(head.title)}">`,
    `<meta property="og:description" content="${escapeHtml(head.description)}">`,
    '<meta name="twitter:card" content="summary">',
    `<meta name="twitter:title" content="${escapeHtml(head.title)}">`,
    `<meta name="twitter:description" content="${escapeHtml(head.description)}">`,
  ];
  if (head.siteName) tags.push(`<meta property="og:site_name" content="${escapeHtml(head.siteName)}">`, `<meta name="apple-mobile-web-app-title" content="${escapeHtml(head.siteName)}">`);
  if (head.canonical) tags.push(`<link rel="canonical" href="${escapeHtml(head.canonical)}">`, `<meta property="og:url" content="${escapeHtml(head.canonical)}">`);
  if (head.canonicalBaseUrl) tags.push(`<meta name="presentation-canonical-base" content="${escapeHtml(head.canonicalBaseUrl)}">`);
  if (head.background) tags.push(`<meta name="theme-color" content="${escapeHtml(head.background)}">`);
  const icon = presentationIcon(head);
  if (icon) tags.push(`<link rel="icon" type="image/svg+xml" href="${escapeHtml(icon)}">`);
  if (head.revision) tags.push(`<meta name="presentation-revision" content="${escapeHtml(head.revision)}">`, `<meta name="presentation-variant" content="${escapeHtml(head.variant)}">`);
  if (bootstrap) tags.push(`<script type="application/json" id="lpbs-presentation-bootstrap">${scriptJSON(bootstrap)}</script>`);
  const clean = html.replace(/<title\b[^>]*>[\s\S]*?<\/title>/gi, '')
    .replace(/<meta\b[^>]*\b(?:name|property)\s*=\s*["'](?:description|robots|og:[^"']*|twitter:[^"']*|theme-color|(?:apple-)?mobile-web-app-[^"']*|presentation-[^"']*)["'][^>]*>/gi, '')
    // A per-app document must not advertise the static first-app install
    // manifest/icon. The finite configured mark owns its browser identity.
    .replace(/<link\b[^>]*\brel\s*=\s*["'](?:canonical|icon|shortcut icon|apple-touch-icon|manifest)["'][^>]*>/gi, '')
    .replace(/<script\b[^>]*\bid=["']lpbs-presentation-bootstrap["'][^>]*>[\s\S]*?<\/script>/gi, '');
  return injectBaseTag(clean, '/', { skipIfExists: true })
    .replace(/<html\b[^>]*>/i, `<html lang="${escapeHtml(head.locale || 'en')}">`)
    .replace(/<head\b[^>]*>/i, match => match + '\n' + tags.join('\n'));
}

/** Documents are read per request; shared static middleware still owns assets. */
export function createPresentationDocuments({ apiBase, readIndex, fetchImplementation = fetch }) {
  const owner = new URL(apiBase);
  if (!['localhost', '127.0.0.1', '[::1]'].includes(owner.hostname) || owner.username || owner.password || owner.search || owner.hash || owner.pathname !== '/') throw new Error('Public owner must be the configured local API');
  return async (req, res, next) => {
    if (!['GET', 'HEAD'].includes(req.method)) return next();
    const url = new URL(req.originalUrl || req.url, 'http://document.invalid');
    const path = url.pathname;
    const routing = routingHeaders(req);
    if (path === '/index.html') {
      res.setHeader('Cache-Control', 'no-store');
      res.setHeader('Location', './' + url.search);
      return res.status(308).type('text/plain').send('Use the canonical page route');
    }
    if (path === '/sitemap.xml' || path === '/robots.txt') {
      const controller = new AbortController(); const timeout = setTimeout(() => controller.abort(), 5000);
      res.setHeader('Cache-Control', 'no-store');
      try {
        // These exact read-only owner paths bypass SPA/static routing. Host,
        // cookies, authorization and caller query parameters are not forwarded.
        const response = await fetchImplementation(apiBase + path, { method: 'GET', signal: controller.signal, redirect: 'error', cache: 'no-store', ...(Object.keys(routing).length ? { headers: routing } : {}) });
        if (!response.ok) { await response.body?.cancel(); throw new Error('Crawler owner unavailable'); }
        const body = await boundedBody(response, 1024 * 1024);
        return res.status(200).type(path === '/sitemap.xml' ? 'application/xml' : 'text/plain').send(body);
      } catch {
        res.setHeader('X-Robots-Tag', 'noindex, nofollow');
        return res.status(503).type('text/plain').send(path === '/robots.txt' ? 'User-agent: *\nDisallow: /\n' : 'Sitemap temporarily unavailable');
      } finally { clearTimeout(timeout); }
    }
    if (/^\/(?:api|assets|public|embedded)(?:\/|$)/.test(path) || ['health', 'config'].includes(path.slice(1)) || /\.[^/]+$/.test(path)) return next();
    // A download chooser consumes the app's exact presentation revision. It is
    // a noindex workflow, not a second marketing page or a different app scope.
    const isDownload = downloadRoute.test(path);
    const route = isDownload ? path.slice(0, -9) : path;
    const request = { route, locale: url.searchParams.get('locale') || '', variant: url.searchParams.get('variant') || url.searchParams.get('variant_slug') || '' };
    let status = 200; let bootstrap;
    let head = { title: 'Private page', description: '', locale: 'en', noindex: true };
    if (path === '/' || detailRoute.test(path) || isDownload) {
      try {
        const visitor = presentationVisitor(req);
        if (visitor.fresh) {
          const secure = req.secure || req.headers?.['x-forwarded-proto'] === 'https';
          res.setHeader('Set-Cookie', `metrics_visitor_id=${visitor.id}; Path=/; SameSite=Lax; Max-Age=31536000${secure ? '; Secure' : ''}`);
        }
        const [config, publicBranding] = await Promise.all([
          publicRPC(apiBase, LANDING, { route, locale: request.locale, variantSlug: request.variant, visitorId: visitor.id }, fetchImplementation, routing),
          publicRPC(apiBase, BRANDING, {}, fetchImplementation, routing),
        ]);
        head = presentationHead(config, request, publicBranding.branding);
        if (isDownload) head.noindex = true;
        bootstrap = { request, config, canonicalBaseUrl: head.canonicalBaseUrl, visitorId: visitor.id };
      } catch (error) {
        status = error.status === 404 ? 404 : 503;
        head = { title: status === 404 ? 'Page not found' : 'Page temporarily unavailable', description: '', locale: 'en', noindex: true };
      }
    } else if (!privateRoute.test(path)) {
      status = 404; head.title = 'Page not found';
    }
    res.setHeader('Cache-Control', 'private, no-store');
    if (head.noindex) res.setHeader('X-Robots-Tag', 'noindex, nofollow');
    try { return res.status(status).type('html').send(injectPresentationHead(await readIndex(), head, bootstrap)); }
    catch { res.setHeader('X-Robots-Tag', 'noindex, nofollow'); return res.status(503).type('text').send('Page temporarily unavailable'); }
  };
}
