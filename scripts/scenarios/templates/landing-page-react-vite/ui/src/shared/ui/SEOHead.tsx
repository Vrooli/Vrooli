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
    // Merge SEO config (variant overrides branding defaults)
    const title = seoConfig?.title || branding?.site_name || undefined;
    const description = seoConfig?.description || undefined;

    const config: SEOConfig = {
      title,
      description,
      ogTitle: seoConfig?.og_title || title,
      ogDescription: seoConfig?.og_description || description,
      ogImage: seoConfig?.og_image_url || undefined,
      twitterCard: seoConfig?.twitter_card || 'summary_large_image',
      noindex: seoConfig?.noindex || false,
    };

    // Build canonical URL
    if (baseUrl && seoConfig?.canonical_path) {
      config.canonical = baseUrl.replace(/\/$/, '') + seoConfig.canonical_path;
    }

    updateMetaTags(config);

    // Update favicon from branding
    if (branding?.favicon_url) {
      updateFavicon(branding.favicon_url);
    }

    // Update theme color from branding
    if (branding?.theme_primary_color) {
      updateThemeColor(branding.theme_primary_color);
    }
  }, [branding, seoConfig, baseUrl]);

  // This component renders nothing - it just updates the document head
  return null;
}
