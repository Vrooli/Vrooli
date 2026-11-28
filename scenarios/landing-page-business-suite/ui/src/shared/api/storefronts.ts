export type SupportedStorefront = 'app_store' | 'play_store';

export interface NormalizedStorefrontLink {
  store: SupportedStorefront;
  label: string;
  url: string;
  badge?: string;
}

/** Validate and normalize an owner-provided App Store or Google Play listing. */
export function normalizeStorefrontLink(
  value: { store: string; label: string; url: string; badge?: string },
): NormalizedStorefrontLink | null {
  const store = value.store.trim();
  const label = value.label.trim();
  const url = value.url.trim();
  const badge = value.badge?.trim() || undefined;
  if ((store !== 'app_store' && store !== 'play_store') || !isOfficialStorefrontLink(store, url)) return null;
  return { store, label, url, ...(badge ? { badge } : {}) };
}

/** Only count official HTTPS app-listing URLs as storefront configuration. */
export function isOfficialStorefrontLink(store: string, rawUrl: string): boolean {
  if (store !== 'app_store' && store !== 'play_store') return false;
  try {
    const normalizedUrl = rawUrl.trim();
    const authority = normalizedUrl.match(/^https:\/\/([^/?#]*)/i)?.[1] ?? '';
    if (authority.includes('@') || normalizedUrl.includes('#')) return false;
    const url = new URL(normalizedUrl);
    if (url.protocol !== 'https:' || url.username || url.password || url.hash || (url.port && url.port !== '443')) return false;
    const host = url.hostname.toLowerCase().replace(/\.$/, '');
    if (store === 'app_store') {
      return (host === 'apps.apple.com' || host === 'itunes.apple.com') && isAppleListingPath(url.pathname);
    }
    return host === 'play.google.com' && url.pathname === '/store/apps/details' && Boolean(url.searchParams.get('id')?.trim());
  } catch {
    return false;
  }
}

function isAppleListingPath(path: string): boolean {
  const segments = path.replace(/^\/+|\/+$/g, '').split('/');
  const appIndex = segments.indexOf('app');
  if (appIndex < 0 || appIndex > 1 || segments.some((segment) => !segment)) return false;
  if (appIndex === 1 && !/^[a-z]{2}$/i.test(segments[0]!)) return false;
  const suffix = segments.slice(appIndex + 1);
  return (suffix.length === 1 || suffix.length === 2) && /^id\d+$/i.test(suffix[suffix.length - 1]!);
}
