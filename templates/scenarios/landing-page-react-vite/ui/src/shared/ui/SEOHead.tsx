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
    const title = seoConfig?.title || branding?.siteName || undefined;
    const description = seoConfig?.description || undefined;

    const config: SEOConfig = {
      title,
      description,
      ogTitle: seoConfig?.ogTitle || title,
      ogDescription: seoConfig?.ogDescription || description,
      ogImage: seoConfig?.ogImageUrl || undefined,
      twitterCard: seoConfig?.twitterCard === 'summary' ? 'summary' : 'summary_large_image',
      noindex: seoConfig?.noindex || false,
    };

    // Build canonical URL
    if (baseUrl && seoConfig?.canonicalPath) {
      config.canonical = baseUrl.replace(/\/$/, '') + seoConfig.canonicalPath;
    }

    updateMetaTags(config);

    // Update favicon from branding
    if (branding?.faviconUrl) {
      updateFavicon(branding.faviconUrl);
    }

    // Update theme color from branding
    if (branding?.themePrimaryColor) {
      updateThemeColor(branding.themePrimaryColor);
    }
  }, [branding, seoConfig, baseUrl]);

  // This component renders nothing - it just updates the document head
  return null;
}
