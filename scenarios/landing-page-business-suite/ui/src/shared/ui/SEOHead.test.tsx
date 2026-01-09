import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor } from '@testing-library/react';
import { SEOHead } from './SEOHead';
import type { LandingBranding, VariantSEOConfig } from '../api';

describe('SEOHead', () => {
  const originalTitle = document.title;
  let originalHead: string;

  beforeEach(() => {
    originalHead = document.head.innerHTML;
    document.title = 'Initial Title';
    // Clear any meta tags we'll be testing
    document.querySelectorAll('meta[name="description"]').forEach(el => el.remove());
    document.querySelectorAll('meta[property^="og:"]').forEach(el => el.remove());
    document.querySelectorAll('meta[name^="twitter:"]').forEach(el => el.remove());
    document.querySelectorAll('link[rel="icon"]').forEach(el => el.remove());
    document.querySelectorAll('link[rel="canonical"]').forEach(el => el.remove());
    document.querySelectorAll('meta[name="theme-color"]').forEach(el => el.remove());
  });

  afterEach(() => {
    document.title = originalTitle;
    document.head.innerHTML = originalHead;
  });

  it('renders nothing to the DOM', () => {
    const { container } = render(<SEOHead />);
    expect(container.firstChild).toBeNull();
  });

  it('updates document title from branding', async () => {
    const branding: LandingBranding = {
      site_name: 'Test Brand',
    };

    render(<SEOHead branding={branding} />);

    await waitFor(() => {
      expect(document.title).toBe('Test Brand');
    });
  });

  it('updates meta description', async () => {
    const seoConfig: VariantSEOConfig = {
      title: 'SEO Title',
      description: 'This is a test description',
    };

    render(<SEOHead seoConfig={seoConfig} />);

    await waitFor(() => {
      const meta = document.querySelector('meta[name="description"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('This is a test description');
    });
  });

  it('updates Open Graph meta tags', async () => {
    const seoConfig: VariantSEOConfig = {
      title: 'OG Test',
      og_title: 'Custom OG Title',
      og_description: 'Custom OG Description',
      og_image_url: 'https://example.com/og-image.jpg',
    };

    render(<SEOHead seoConfig={seoConfig} />);

    await waitFor(() => {
      expect(document.querySelector('meta[property="og:title"]')?.getAttribute('content')).toBe('Custom OG Title');
      expect(document.querySelector('meta[property="og:description"]')?.getAttribute('content')).toBe('Custom OG Description');
      expect(document.querySelector('meta[property="og:image"]')?.getAttribute('content')).toBe('https://example.com/og-image.jpg');
    });
  });

  it('updates Twitter card meta tags', async () => {
    const seoConfig: VariantSEOConfig = {
      title: 'Twitter Test',
      twitter_card: 'summary',
    };

    render(<SEOHead seoConfig={seoConfig} />);

    await waitFor(() => {
      expect(document.querySelector('meta[name="twitter:card"]')?.getAttribute('content')).toBe('summary');
      expect(document.querySelector('meta[name="twitter:title"]')?.getAttribute('content')).toBe('Twitter Test');
    });
  });

  it('sets favicon from branding', async () => {
    const branding: LandingBranding = {
      site_name: 'Test',
      favicon_url: 'https://example.com/favicon.ico',
    };

    render(<SEOHead branding={branding} />);

    await waitFor(() => {
      const icon = document.querySelector('link[rel="icon"]');
      expect(icon).toBeTruthy();
      expect(icon?.getAttribute('href')).toBe('https://example.com/favicon.ico');
    });
  });

  it('sets theme color from branding', async () => {
    const branding: LandingBranding = {
      site_name: 'Test',
      theme_primary_color: '#6366f1',
    };

    render(<SEOHead branding={branding} />);

    await waitFor(() => {
      const meta = document.querySelector('meta[name="theme-color"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('#6366f1');
    });
  });

  it('sets canonical URL', async () => {
    const seoConfig: VariantSEOConfig = {
      canonical_path: '/landing',
    };

    render(<SEOHead seoConfig={seoConfig} baseUrl="https://example.com" />);

    await waitFor(() => {
      const link = document.querySelector('link[rel="canonical"]');
      expect(link).toBeTruthy();
      expect(link?.getAttribute('href')).toBe('https://example.com/landing');
    });
  });

  it('variant SEO config overrides branding defaults', async () => {
    const branding: LandingBranding = {
      site_name: 'Default Site Name',
    };

    const seoConfig: VariantSEOConfig = {
      title: 'Custom Variant Title',
    };

    render(<SEOHead branding={branding} seoConfig={seoConfig} />);

    await waitFor(() => {
      expect(document.title).toBe('Custom Variant Title');
    });
  });

  it('falls back to branding when SEO config is empty', async () => {
    const branding: LandingBranding = {
      site_name: 'Fallback Site',
    };

    render(<SEOHead branding={branding} seoConfig={{}} />);

    await waitFor(() => {
      expect(document.title).toBe('Fallback Site');
    });
  });

  it('handles noindex directive', async () => {
    const seoConfig: VariantSEOConfig = {
      noindex: true,
    };

    render(<SEOHead seoConfig={seoConfig} />);

    await waitFor(() => {
      const meta = document.querySelector('meta[name="robots"]');
      expect(meta).toBeTruthy();
      expect(meta?.getAttribute('content')).toBe('noindex, nofollow');
    });
  });

  it('removes robots meta when noindex is false', async () => {
    // First add a robots meta
    const robotsMeta = document.createElement('meta');
    robotsMeta.name = 'robots';
    robotsMeta.content = 'noindex';
    document.head.appendChild(robotsMeta);

    const seoConfig: VariantSEOConfig = {
      noindex: false,
    };

    render(<SEOHead seoConfig={seoConfig} />);

    await waitFor(() => {
      const meta = document.querySelector('meta[name="robots"]');
      expect(meta).toBeNull();
    });
  });

  it('handles null/undefined gracefully', async () => {
    const { rerender } = render(<SEOHead branding={null} seoConfig={null} />);

    // Should not throw
    expect(document.title).toBe('Initial Title');

    rerender(<SEOHead branding={undefined} seoConfig={undefined} />);

    // Still should not throw
    expect(document.title).toBe('Initial Title');
  });
});
