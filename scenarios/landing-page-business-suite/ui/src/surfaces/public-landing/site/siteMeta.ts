import { useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import { updateMetaTags, updateThemeColor } from '../../../shared/lib/seo';
import { canonicalPresentationHref } from '../presentation/publicIntegration';
import type { SiteIdentity } from './useSiteIdentity';

export interface SitePageMeta {
  /** Page name; the brand is appended. */
  title: string;
  description: string;
  /** Crawlers index site pages unless the page opts out. */
  noindex?: boolean;
}

export const SITE_THEME_COLOR = '#0b1728';
export const SITE_OG_IMAGE = '/public/og-image.jpg';

/** Social crawlers need absolute image URLs; a malformed configured base falls back to this origin. */
export function socialImageUrl(base?: string): string {
  try { return new URL(SITE_OG_IMAGE, base || window.location.origin).href; } catch { return new URL(SITE_OG_IMAGE, window.location.origin).href; }
}

/** Keeps document metadata in step with the site page being shown. */
export function useSitePageMeta({ title, description, noindex = false }: SitePageMeta, identity: SiteIdentity) {
  const { pathname } = useLocation();
  useEffect(() => {
    const fullTitle = identity.brandName && !title.includes(identity.brandName) ? `${title} · ${identity.brandName}` : title;
    const canonical = noindex ? undefined : canonicalPresentationHref(identity.website, pathname);
    if (!canonical) document.querySelector('link[rel="canonical"]')?.remove();
    updateMetaTags({ title: fullTitle, description, ogTitle: fullTitle, ogImage: socialImageUrl(identity.website), twitterCard: 'summary_large_image', canonical, noindex });
    let ogUrl = document.querySelector<HTMLMetaElement>('meta[property="og:url"]');
    if (canonical) {
      if (!ogUrl) { ogUrl = document.createElement('meta'); ogUrl.setAttribute('property', 'og:url'); document.head.append(ogUrl); }
      ogUrl.content = canonical;
    } else ogUrl?.remove();
    updateThemeColor(SITE_THEME_COLOR);
  }, [title, description, noindex, identity.brandName, identity.website, pathname]);
}
