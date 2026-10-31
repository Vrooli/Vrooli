/**
 * Client-side SEO utilities for dynamic meta tag management.
 *
 * Note: Critical SEO meta tags are server-rendered via server.js for crawlers.
 * These utilities handle client-side updates during SPA navigation.
 */

export interface SEOConfig {
  title?: string;
  description?: string;
  ogTitle?: string;
  ogDescription?: string;
  ogImage?: string;
  twitterCard?: 'summary' | 'summary_large_image';
  canonical?: string;
  noindex?: boolean;
}

/**
 * Updates document meta tags. Call this when route/variant changes.
 */
export function updateMetaTags(config: SEOConfig): void {
  // Update title
  if (config.title) {
    document.title = config.title;
  }

  // Update or create meta tags
  setMetaTag('name', 'description', config.description);
  setMetaTag('property', 'og:title', config.ogTitle || config.title);
  setMetaTag('property', 'og:description', config.ogDescription || config.description);
  setMetaTag('property', 'og:image', config.ogImage);
  setMetaTag('name', 'twitter:title', config.ogTitle || config.title);
  setMetaTag('name', 'twitter:description', config.ogDescription || config.description);
  setMetaTag('name', 'twitter:image', config.ogImage);
  setMetaTag('name', 'twitter:card', config.twitterCard || 'summary_large_image');

  // Handle robots meta
  if (config.noindex) {
    setMetaTag('name', 'robots', 'noindex, nofollow');
  } else {
    removeMetaTag('name', 'robots');
  }

  // Update canonical link
  if (config.canonical) {
    let link = document.querySelector('link[rel="canonical"]') as HTMLLinkElement | null;
    if (!link) {
      link = document.createElement('link');
      link.rel = 'canonical';
      document.head.appendChild(link);
    }
    link.href = config.canonical;
  }
}

/**
 * Sets or updates a meta tag. Removes if value is empty.
 */
function setMetaTag(attrName: 'name' | 'property', attrValue: string, content?: string): void {
  const selector = `meta[${attrName}="${attrValue}"]`;
  let element = document.querySelector(selector) as HTMLMetaElement | null;

  if (!content) {
    // Remove tag if content is empty
    if (element) {
      element.remove();
    }
    return;
  }

  if (!element) {
    element = document.createElement('meta');
    element.setAttribute(attrName, attrValue);
    document.head.appendChild(element);
  }

  element.content = content;
}

/**
 * Removes a meta tag by selector.
 */
function removeMetaTag(attrName: 'name' | 'property', attrValue: string): void {
  const selector = `meta[${attrName}="${attrValue}"]`;
  const element = document.querySelector(selector);
  if (element) {
    element.remove();
  }
}

/**
 * Updates favicon dynamically.
 */
export function updateFavicon(url: string): void {
  let link = document.querySelector('link[rel="icon"]') as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement('link');
    link.rel = 'icon';
    document.head.appendChild(link);
  }
  link.href = url;

  // Also update shortcut icon
  let shortcut = document.querySelector('link[rel="shortcut icon"]') as HTMLLinkElement | null;
  if (!shortcut) {
    shortcut = document.createElement('link');
    shortcut.rel = 'shortcut icon';
    document.head.appendChild(shortcut);
  }
  shortcut.href = url;
}

/**
 * Updates theme color meta tag (for mobile browsers).
 */
export function updateThemeColor(color: string): void {
  setMetaTag('name', 'theme-color', color);
}
