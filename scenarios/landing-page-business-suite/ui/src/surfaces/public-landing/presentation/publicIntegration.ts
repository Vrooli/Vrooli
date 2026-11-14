import type { ResolvedProductPresentation } from '@vrooli/proto-types/landing-page-business-suite/v1/shared/product_presentation_pb';
import { decodeProductPresentation } from './decode';
import { actionKey, type Action, type Block, type Presentation, type ResolvedActions } from './types';
import { safeHref } from './links';
import type { DownloadApp } from '../../../shared/api/types';

/** Canonical authority is configured, never inferred from the browser Host. */
export function canonicalPresentationHref(authority: string | undefined, route: string): string | undefined {
  if (!authority) return undefined;
  try {
    const url = new URL(authority);
    if (['https:', 'http:'].includes(url.protocol) && !url.username && !url.password && !url.search && !url.hash) return url.origin + url.pathname.replace(/\/$/, '') + route;
  } catch { /* malformed authority cannot supply metadata */ }
  return undefined;
}

export function publicHref(value: string, basename: string): string {
  const safe = safeHref(value);
  if (!safe) throw new Error('Unsafe presentation link');
  if (safe.startsWith('/')) {
    const segments = safe.split(/[?#]/)[0]?.split('/') ?? [];
    if (segments.some(segment => { const decoded = decodeURIComponent(segment); return decoded === '.' || decoded === '..' || /[\\/]/.test(decoded); })) throw new Error('Unsafe presentation path');
  }
  const base = basename.replace(/\/+$/, '');
  // Root-relative URLs are app paths; do not double-prefix already based owner URLs.
  return safe.startsWith('/') && base && safe !== base && !safe.startsWith(`${base}/`) ? `${base}${safe}` : safe;
}

/** Only known internal marketing routes inherit explicit presentation scope. */
export function scopedPresentationHref(value: string, basename: string, request?: { locale: string; variant: string }): string {
  const href = publicHref(value, basename);
  if (!request || (!request.locale && !request.variant) || !href.startsWith('/')) return href;
  const url = new URL(href, 'https://presentation.invalid');
  const base = basename.replace(/\/+$/, '');
  const path = base && (url.pathname === base || url.pathname.startsWith(`${base}/`)) ? url.pathname.slice(base.length) || '/' : url.pathname;
  if (path !== '/' && !/^\/apps\/[a-z0-9][a-z0-9-]*(?:\/download)?$/.test(path)) return href;
  if (request.locale) url.searchParams.set('locale', request.locale);
  if (request.variant) url.searchParams.set('variant', request.variant);
  return url.pathname + url.search + url.hash;
}

/** Owner navigation accepts HTTPS or app-absolute paths, never browser-normalized tricks. */
export function safeOwnerHref(value: unknown, basename: string): string | undefined {
  const hasControlOrSpace = (text: string) => {
    for (let index = 0; index < text.length; index++) {
      if (text.charCodeAt(index) <= 32 || text.charCodeAt(index) === 127) return true;
    }
    return false;
  };
  if (typeof value !== 'string' || !safeHref(value) || value.startsWith('#') || hasControlOrSpace(value) || value.includes('\\')) return undefined;
  try {
    const absolute = /^https:\/\//i.test(value);
    if (!absolute && !value.startsWith('/')) return undefined;
    const path = (absolute ? value.replace(/^https:\/\/[^/?#]*/i, '') : value).split(/[?#]/)[0] || '/';
    // Validate before URL parsing can remove dot segments. Query tokens remain opaque.
    publicHref(path, '');
    if (path.split('/').some(segment => hasControlOrSpace(decodeURIComponent(segment)))) return undefined;
    if (absolute) {
      const url = new URL(value);
      if (url.protocol !== 'https:' || !url.hostname || url.username || url.password) return undefined;
      return url.href;
    }
    return publicHref(value, basename);
  } catch { return undefined; }
}

export const actionsOf = (p: Presentation): Action[] => [
  ...(p.page.display.shell.header_action ? [p.page.display.shell.header_action] : []),
  ...p.page.blocks.flatMap(b => 'actions' in b.content ? b.content.actions : []),
];

/** Owner results are observations; never derive price IDs, installers or checkout URLs. */
export function resolvePublicActions(wire: ResolvedProductPresentation, p: Presentation, basename: string, downloads: readonly DownloadApp[] = []): ResolvedActions {
  const configured = new Map(actionsOf(p).map(action => [actionKey(action), action]));
  const result: Record<string, ResolvedActions[string]> = {};
  for (const owner of wire.actions) {
    const action = configured.get(owner.key);
    if (!action) continue;
    if (Object.prototype.hasOwnProperty.call(result, owner.key)) throw new Error('Duplicate owner action');
    const href = action.kind === 'anchor' ? safeHref(owner.href) : safeOwnerHref(owner.href, '');
    const path = href ? new URL(href, 'https://presentation.invalid').pathname : '';
    const isDetailDownload = action.kind === 'download' && (/\/apps\/[^/]+\/?$/.test(path) || href === wire.diagnostics?.resolvedRoute);
    let launchMatches = true;
    if (action.kind === 'open' && action.app_key) {
      const bundle = wire.diagnostics?.bundleKey;
      const matches = downloads.filter(app => app.app_key === action.app_key && app.bundle_key === bundle);
      const app = matches.length === 1 ? matches[0] : undefined;
      const launch = safeOwnerHref(app?.metadata?.web_url, '');
      const launchPath = launch ? new URL(launch, 'https://presentation.invalid').pathname : '';
      const marketingPath = (launch?.startsWith('/') && launchPath === '/') || /\/apps\/[^/]+(?:\/download)?\/?$/.test(launchPath);
      launchMatches = Boolean(bundle && wire.diagnostics?.eligibleAppKeys.includes(action.app_key) && app && app.metadata?.enabled !== false && launch && !marketingPath && launch === href);
    }
    result[owner.key] = owner.status === 'ready' && href && !isDetailDownload && launchMatches
      ? { status: 'ready', href: publicHref(href, basename) }
      : { status: 'unavailable', reason: owner.reason || p.page.display.shell.unavailable_reason };
  }
  return result;
}

/** Preserve canonical identity and content. Only local use-site URLs gain router base. */
export function withPresentationBase(p: Presentation, basename: string, request?: { locale: string; variant: string }): Presentation {
  const link = (value: string) => scopedPresentationHref(value, basename, request);
  const navigation = (n: { label: string; accessible_label: string; target: string }) => ({ ...n, target: link(n.target) });
  // Action target identity is unchanged; local action URLs get resolved via the map.
  const blocks: Block[] = p.page.blocks.map(b => b.kind === 'footer' ? { ...b, content: { ...b.content, links: b.content.links.map(navigation) } } : b);
  return {
    ...p, spotlights: p.spotlights?.map(a => ({ ...a, detail_route: link(a.detail_route) })),
    assets: p.assets?.map(a => ({ ...a, public_url: a.public_url ? publicHref(a.public_url, basename) : a.public_url })),
    page: { ...p.page, blocks, navigation: { ...p.page.navigation, items: p.page.navigation.items.map(navigation) },
      footer: { ...p.page.footer, links: p.page.footer.links.map(navigation) },
      display: { ...p.page.display, shell: { ...p.page.display.shell, brand_target: link(p.page.display.shell.brand_target), footer_brand_target: link(p.page.display.shell.footer_brand_target) } },
    },
  };
}

export function resolveNavigationActions(p: Presentation, basename: string): ResolvedActions {
  return Object.fromEntries(actionsOf(p).filter(action => (action.kind === 'anchor' || action.kind === 'app-detail') && action.target).map(action => [actionKey(action), { status: 'ready', href: publicHref(action.target ?? '', basename) }]));
}

export function resolvePublicPresentation(wire: ResolvedProductPresentation, request: { route: string; locale: string; variant: string }, basename: string, downloads: readonly DownloadApp[] = []) {
  const d = wire.diagnostics;
  if (!d || d.preview || d.requestedRoute !== request.route || d.resolvedRoute !== request.route || !d.resolvedRevision || !d.blockDigest || !d.resolvedVariant || d.requestedVariant !== request.variant || d.locale !== wire.page?.locale) throw new Error('Unqualified public presentation diagnostics');
  if (request.route !== '/' && wire.mode !== 'app_detail') throw new Error('Detail route did not resolve an app');
  const presentation = decodeProductPresentation(wire);
  if (request.locale && d.locale !== request.locale && !d.fallback) throw new Error('Unexplained presentation locale mismatch');
  for (const asset of presentation.assets ?? []) {
    if (!asset.release_ref || !/^[0-9a-f]{64}$/i.test(asset.content_hash) || !d.assetReleaseRefs.includes(asset.release_ref)) throw new Error('Unqualified public asset');
  }
  const actions: Record<string, ResolvedActions[string]> = { ...resolvePublicActions(wire, presentation, basename, downloads) };
  for (const action of actionsOf(presentation)) {
    const key = actionKey(action);
    if (!Object.prototype.hasOwnProperty.call(actions, key) && (action.kind === 'anchor' || action.kind === 'app-detail') && action.target) actions[key] = { status: 'ready', href: publicHref(action.target, basename) };
  }
  for (const action of actionsOf(presentation)) {
    const key = actionKey(action); const owner = actions[key];
    // Launch and commerce URLs remain opaque even if an owner uses a local path.
    if (owner?.status === 'ready' && owner.href && ['anchor', 'app-detail', 'download'].includes(action.kind)) actions[key] = { ...owner, href: scopedPresentationHref(owner.href, basename, request) };
  }
  return { presentation: withPresentationBase(presentation, basename, request), resolvedActions: actions };
}
