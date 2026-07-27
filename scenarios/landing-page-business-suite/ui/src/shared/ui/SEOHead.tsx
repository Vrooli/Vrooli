import { useEffect } from 'react';
import { updateMetaTags, updateFavicon, updateThemeColor, type SEOConfig } from '../lib/seo';
import type { LandingBranding, VariantSEOConfig } from '../api';

interface SEOHeadProps {
  /** Site-wide branding defaults */
  branding?: LandingBranding | null;
  /** Per-variant SEO overrides */
  seoConfig?: VariantSEOConfig | null;
  /** Base URL for canonical links */
  baseUrl?: string;
}

/**
 * SEOHead updates document meta tags based on branding and variant SEO config.
 *
 * Usage:
 * ```tsx
 * <SEOHead branding={landingConfig.branding} seoConfig={variantSeo} />
 * ```
 *
 * Note: Server-side meta injection handles initial page load for crawlers.
 * This component handles client-side updates during SPA navigation.
 */
export function SEOHead({ branding, seoConfig, baseUrl }: SEOHeadProps) {
  useEffect(() => {
    const existingDescription = document
      .querySelector('meta[name="description"]')
      ?.getAttribute('content')
      ?? undefined;
    const existingOgTitle = document
      .querySelector('meta[property="og:title"]')
      ?.getAttribute('content')
      ?? undefined;
    const existingOgDescription = document
      .querySelector('meta[property="og:description"]')
      ?.getAttribute('content')
      ?? undefined;
    const existingOgImage = document
      .querySelector('meta[property="og:image"]')
      ?.getAttribute('content')
      ?? undefined;
    const existingTwitterCard = document
      .querySelector('meta[name="twitter:card"]')
      ?.getAttribute('content') as SEOConfig['twitterCard'] | undefined;

    // Merge SEO config (variant overrides branding defaults)
    const title = seoConfig?.title || branding?.site_name || document.title || undefined;
    const description = seoConfig?.description || branding?.tagline || existingDescription || undefined;

    const config: SEOConfig = {
      title,
      description,
      ogTitle: seoConfig?.og_title || title || existingOgTitle,
      ogDescription: seoConfig?.og_description || description || existingOgDescription,
      ogImage: seoConfig?.og_image_url || existingOgImage || undefined,
      twitterCard: seoConfig?.twitter_card || existingTwitterCard || 'summary_large_image',
      noindex: seoConfig?.noindex || false,
    };

    // Build canonical URL
    const resolvedBaseUrl = baseUrl || (typeof window !== 'undefined' ? window.location.origin : undefined);
    const canonicalPath = seoConfig?.canonical_path || (typeof window !== 'undefined' ? window.location.pathname : '');
    if (resolvedBaseUrl && canonicalPath) {
      config.canonical = resolvedBaseUrl.replace(/\/$/, '') + canonicalPath;
    }

    updateMetaTags(config);

    // Update favicon from branding
    if (branding?.favicon_url) {
      updateFavicon(branding.favicon_url);
    }

    // Browser chrome should match the page background, not the interactive accent.
    // A primary color can be vivid while the application surface remains dark.
    updateThemeColor(branding?.theme_background_color || '#07090F');
  }, [branding, seoConfig, baseUrl]);

  // This component renders nothing - it just updates the document head
  return null;
}
