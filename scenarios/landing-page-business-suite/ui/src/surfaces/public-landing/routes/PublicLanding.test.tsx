import { beforeEach, describe, it, expect, vi } from 'vitest';
import { renderWithProviders as render } from "../../../test-utils/renderWithProviders";
import { fireEvent, screen, within } from "@testing-library/react";
import type { ReactNode } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { PublicLanding } from './PublicLanding';

const useLandingVariantMock = vi.hoisted(() => vi.fn());

vi.mock('../../../app/providers/LandingVariantProvider', () => {
  const mockConfig = {
    variant: { id: 1, slug: 'control', name: 'Control' },
    branding: {
      site_name: 'Acme Launchpad',
      tagline: 'Automation that ships',
      logo_url: 'https://cdn.example.com/logo.svg',
      logo_icon_url: 'https://cdn.example.com/icon.png',
    },
    sections: [
      {
        id: 1,
        section_type: 'hero',
        order: 1,
        enabled: true,
        content: {
          title: 'Test Hero',
          subtitle: 'Subtitle',
          cta_text: 'Start Trial',
          cta_url: '/signup',
        },
      },
    ],
    downloads: [
      {
        bundle_key: 'bundle',
        app_key: 'desktop',
        name: 'Desktop Suite',
        tagline: 'mac + windows',
        description: '',
        install_overview: '',
        install_steps: [],
        storefronts: [],
        display_order: 0,
        platforms: [
          {
            id: 1,
            bundle_key: 'bundle',
            app_key: 'desktop',
            platform: 'mac',
            artifact_url: 'https://example.com/app.dmg',
            release_version: '1.0.0',
            requires_entitlement: false,
          },
        ],
      },
    ],
    fallback: false,
    pricing: null,
    header: {
      branding: { mode: 'logo_and_name', label: 'Acme Launchpad', mobile_preference: 'auto' },
      nav: { links: [] },
      ctas: {
        primary: { mode: 'inherit_hero', variant: 'solid' },
        secondary: { mode: 'downloads', variant: 'ghost' },
      },
      behavior: { sticky: true, hide_on_scroll: false },
    },
  };

  return {
    useLandingVariant: () => ({
      variant: { slug: 'control', name: 'Control' },
      config: mockConfig,
      loading: false,
      error: null,
      resolution: 'api_select',
      statusNote: null,
      lastUpdated: Date.now(),
      refresh: vi.fn(),
    }),
    LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  };
});

// Also mock the separate useLandingVariant hook file (PublicLanding imports from here)
vi.mock('../../../app/providers/useLandingVariant', () => {
  const mockConfig = {
    variant: { id: 1, slug: 'control', name: 'Control' },
    branding: {
      site_name: 'Acme Launchpad',
      tagline: 'Automation that ships',
      logo_url: 'https://cdn.example.com/logo.svg',
      logo_icon_url: 'https://cdn.example.com/icon.png',
    },
    sections: [
      {
        id: 1,
        section_type: 'hero',
        order: 1,
        enabled: true,
        content: {
          title: 'Test Hero',
          subtitle: 'Subtitle',
          cta_text: 'Start Trial',
          cta_url: '/signup',
        },
      },
    ],
    downloads: [
      {
        bundle_key: 'bundle',
        app_key: 'desktop',
        name: 'Desktop Suite',
        tagline: 'mac + windows',
        description: '',
        install_overview: '',
        install_steps: [],
        storefronts: [],
        display_order: 0,
        platforms: [
          {
            id: 1,
            bundle_key: 'bundle',
            app_key: 'desktop',
            platform: 'mac',
            artifact_url: 'https://example.com/app.dmg',
            release_version: '1.0.0',
            requires_entitlement: false,
          },
        ],
      },
    ],
    fallback: false,
    pricing: null,
    header: {
      branding: { mode: 'logo_and_name', label: 'Acme Launchpad', mobile_preference: 'auto' },
      nav: { links: [] },
      ctas: {
        primary: { mode: 'inherit_hero', variant: 'solid' },
        secondary: { mode: 'downloads', variant: 'ghost' },
      },
      behavior: { sticky: true, hide_on_scroll: false },
    },
  };

  return { useLandingVariant: useLandingVariantMock };
});

vi.mock('../../../shared/hooks/useMetricsHook', () => ({
  useMetrics: () => ({
    trackCTAClick: vi.fn(),
    trackDownload: vi.fn(),
  }),
}));

describe('PublicLanding header rails', () => {
  beforeEach(() => {
    useLandingVariantMock.mockReset();
    useLandingVariantMock.mockReturnValue({
      variant: { slug: 'control', name: 'Control', status: 'active' },
      config: {
        variant: { id: 1, slug: 'control', name: 'Control' },
        branding: {
          site_name: 'Acme Launchpad',
          tagline: 'Automation that ships',
          logo_url: 'https://cdn.example.com/logo.svg',
          logo_icon_url: 'https://cdn.example.com/icon.png',
        },
        sections: [{
          id: 1,
          section_type: 'hero',
          order: 1,
          enabled: true,
          content: { title: 'Test Hero', subtitle: 'Subtitle', cta_text: 'Start Trial', cta_url: '/signup' },
        }],
        downloads: [{
          bundle_key: 'bundle', app_key: 'desktop', name: 'Desktop Suite', tagline: 'mac + windows',
          description: '', install_overview: '', install_steps: [], storefronts: [], display_order: 0,
          platforms: [{ id: 1, bundle_key: 'bundle', app_key: 'desktop', platform: 'mac', artifact_url: 'https://example.com/app.dmg', release_version: '1.0.0', requires_entitlement: false }],
        }],
        fallback: false,
        pricing: null,
        header: {
          branding: { mode: 'logo_and_name', label: 'Acme Launchpad', mobile_preference: 'auto' },
          nav: { links: [] },
          ctas: { primary: { mode: 'inherit_hero', variant: 'solid' }, secondary: { mode: 'downloads', variant: 'ghost' } },
          behavior: { sticky: true, hide_on_scroll: false },
        },
      },
      loading: false,
      error: null,
      resolution: 'api_select',
      statusNote: null,
      lastUpdated: Date.now(),
      refresh: vi.fn(),
    });
  });

  it('surfaces CTA and download anchors in the sticky header', () => {
    render(
      <BrowserRouter>
        <PublicLanding />
      </BrowserRouter>
    );

    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
    expect(screen.getByTestId('landing-nav-cta')).toBeInTheDocument();
    const downloadButton = screen.getByTestId('landing-nav-download');
    expect(downloadButton).toBeInTheDocument();
    expect(within(downloadButton).getByText(/Download macOS/i)).toBeInTheDocument();
  });

  it('shows site branding name and logo when provided', () => {
    render(
      <BrowserRouter>
        <PublicLanding />
      </BrowserRouter>
    );

    expect(screen.getByText('Acme Launchpad')).toBeInTheDocument();
    expect(screen.getByText(/Automation that ships/)).toBeInTheDocument();
    expect(screen.getByTestId('branding-logo')).toBeInTheDocument();
    expect(screen.getByAltText(/Acme Launchpad logo/i)).toBeInTheDocument();
  });

  it('renders a loading state without attempting to resolve sections', () => {
    useLandingVariantMock.mockReturnValue({ loading: true });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByText('Loading your landing page...')).toBeInTheDocument();
    expect(screen.queryByTestId('landing-experience-header')).not.toBeInTheDocument();
  });

  it('renders a retryable error when no offline-safe fallback is available', () => {
    useLandingVariantMock.mockReturnValue({
      loading: false,
      config: null,
      variant: null,
      error: 'The configuration service is unavailable.',
      resolution: 'unknown',
      statusNote: null,
      lastUpdated: null,
    });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByRole('heading', { name: 'Failed to Load Variant' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('renders the no-content state for an otherwise valid variant', () => {
    useLandingVariantMock.mockReturnValue({
      variant: { slug: 'empty', name: 'Empty variant', status: 'active' },
      config: { sections: [], downloads: [], fallback: false },
      loading: false,
      error: null,
      resolution: 'api_select',
      statusNote: null,
      lastUpdated: null,
    });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByRole('heading', { name: 'No Content Yet' })).toBeInTheDocument();
    expect(screen.getByText(/doesn't have any sections yet/i)).toBeInTheDocument();
  });

  it('explains when no variant has been resolved instead of rendering a partial landing page', () => {
    useLandingVariantMock.mockReturnValue({
      variant: null,
      config: null,
      loading: false,
      error: null,
      resolution: 'unknown',
      statusNote: null,
      lastUpdated: null,
    });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByRole('heading', { name: 'No Variant Available' })).toBeInTheDocument();
    expect(screen.queryByTestId('landing-experience-header')).not.toBeInTheDocument();
  });

  it('omits unavailable navigation and CTAs while hiding a sticky header after a meaningful downward scroll', () => {
    useLandingVariantMock.mockReturnValue({
      variant: { slug: 'minimal', name: 'Minimal', status: 'active' },
      config: {
        variant: { id: 3, slug: 'minimal', name: 'Minimal' },
        branding: { site_name: 'Minimal' },
        sections: [{ id: 1, section_type: 'hero', order: 1, enabled: true, content: { title: 'Minimal' } }],
        downloads: [],
        fallback: false,
        pricing: null,
        header: {
          branding: { mode: 'logo', label: '', mobile_preference: 'name', logo_url: null, logo_icon_url: null },
          nav: { links: [{ id: 'downloads', type: 'downloads', label: 'Download' }, { id: 'missing-custom', type: 'custom', label: 'Missing' }] },
          ctas: { primary: { mode: 'inherit_hero', variant: 'solid' }, secondary: { mode: 'downloads', variant: 'ghost' } },
          behavior: { sticky: true, hide_on_scroll: true },
        },
      },
      loading: false,
      error: null,
      resolution: 'fallback',
      statusNote: null,
      lastUpdated: null,
    });
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 0 });
    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    const header = screen.getByTestId('landing-experience-header');
    expect(screen.queryByTestId('landing-nav-cta')).not.toBeInTheDocument();
    expect(screen.queryByTestId('landing-nav-download')).not.toBeInTheDocument();
    expect(screen.queryByText('Download')).not.toBeInTheDocument();
    expect(header).toHaveClass('sticky');

    Object.defineProperty(window, 'scrollY', { configurable: true, value: 200 });
    fireEvent.scroll(window);
    expect(header).toHaveClass('-translate-y-full');
    Object.defineProperty(window, 'scrollY', { configurable: true, value: 100 });
    fireEvent.scroll(window);
    expect(header).toHaveClass('translate-y-0');
  });

  it('renders each supported section and the offline-safe operational signals', () => {
    window.history.pushState({}, '', '/?debug=1&variant=control');
    useLandingVariantMock.mockReturnValue({
      variant: { slug: 'control', name: 'Control', status: 'active' },
      config: {
        variant: { id: 1, slug: 'control', name: 'Control' },
        branding: { site_name: 'Acme Launchpad', support_email: 'support@example.com' },
        sections: [
          { id: 1, section_type: 'hero', order: 1, enabled: true, content: { title: 'Hero', subtitle: 'Start here' } },
          { id: 2, section_type: 'features', order: 2, enabled: true, content: { title: 'Features', items: [] } },
          { id: 3, section_type: 'pricing', order: 3, enabled: true, content: { title: 'Pricing', tiers: [] } },
          { id: 4, section_type: 'cta', order: 4, enabled: true, content: { title: 'Ready?', cta_text: 'Join' } },
          { id: 5, section_type: 'testimonials', order: 5, enabled: true, content: { title: 'Customers', testimonials: [] } },
          { id: 6, section_type: 'faq', order: 6, enabled: true, content: { title: 'FAQ', items: [] } },
          { id: 7, section_type: 'footer', order: 7, enabled: true, content: { copyright: '© Acme' } },
          { id: 8, section_type: 'video', order: 8, enabled: true, content: { title: 'Watch', video_url: 'https://example.com/demo' } },
          { id: 9, section_type: 'downloads', order: 9, enabled: true, content: { title: 'Downloads' } },
        ],
        downloads: [],
        fallback: true,
        pricing: null,
        header: { branding: { mode: 'name', label: 'Acme', mobile_preference: 'auto' }, nav: { links: [] }, ctas: { primary: { mode: 'hidden', variant: 'solid' }, secondary: { mode: 'hidden', variant: 'ghost' } }, behavior: { sticky: true, hide_on_scroll: false } },
      },
      loading: false,
      error: 'Network unavailable',
      resolution: 'url_param',
      statusNote: 'Using cached configuration',
      lastUpdated: Date.UTC(2026, 0, 1, 12, 0, 0),
    });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByTestId('variant-source-banner')).toHaveTextContent('Control');
    expect(screen.getByTestId('fallback-signal-banner')).toHaveTextContent('Using cached configuration');
    expect(screen.getByText('Debug Info')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Hero' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Features' })).toBeInTheDocument();
    expect(screen.getByRole('contentinfo')).toBeInTheDocument();
    window.history.pushState({}, '', '/');
  });

  it('honors configured navigation, custom CTAs, disabled sections, and standalone downloads', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    useLandingVariantMock.mockReturnValue({
      variant: { slug: 'configured', name: 'Configured', status: 'active' },
      config: {
        variant: { id: 2, slug: 'configured', name: 'Configured' },
        branding: { site_name: 'Configured Co.' },
        sections: [
          { id: 1, section_type: 'hero', order: 2, enabled: true, content: { title: 'Welcome', cta_text: 'Begin', cta_url: '/begin' } },
          { id: 2, section_type: 'faq', order: 1, enabled: true, content: { title: 'Help', items: [] } },
          { id: 3, section_type: 'features', order: 3, enabled: false, content: { title: 'Hidden feature', items: [] } },
          { id: 4, section_type: 'unsupported', order: 4, enabled: true, content: {} },
        ],
        downloads: [{
          bundle_key: 'bundle', app_key: 'desktop', name: 'Desktop Suite', tagline: 'mac + windows', description: '', install_overview: '', install_steps: [], storefronts: [], display_order: 0,
          platforms: [{ id: 1, bundle_key: 'bundle', app_key: 'desktop', platform: 'windows', artifact_url: 'https://example.com/app.exe', release_version: '1.0.0', requires_entitlement: false }],
        }],
        fallback: false,
        pricing: null,
        header: {
          branding: { mode: 'none', mobile_preference: 'auto' },
          nav: { links: [
            { id: 'faq', type: 'section', section_id: 2, label: 'Support', visible_on: { desktop: true, mobile: false } },
            { id: 'learn', type: 'custom', label: 'Learn', href: '/learn', visible_on: { desktop: false, mobile: true } },
            { id: 'products', type: 'menu', label: 'Products', children: [
              { id: 'product-faq', section_type: 'faq', label: 'FAQ', visible_on: { desktop: false, mobile: true } },
              { id: 'product-download', type: 'downloads', label: 'Install', visible_on: { desktop: true, mobile: false } },
              { id: 'product-fallback', label: 'Fallback', visible_on: { desktop: false, mobile: false } },
            ] },
            { id: 'missing', type: 'section', label: 'Not rendered', section_type: 'pricing' },
            { id: 'download', type: 'downloads', label: 'Get the app' },
          ] },
          ctas: {
            primary: { mode: 'custom', label: 'Talk to us', href: '/contact', variant: 'solid' },
            secondary: { mode: 'downloads', label: 'Install now', variant: 'ghost' },
          },
          behavior: { sticky: false, hide_on_scroll: true },
        },
      },
      loading: false,
      error: null,
      resolution: 'local_storage',
      statusNote: null,
      lastUpdated: null,
    });

    render(<BrowserRouter><PublicLanding /></BrowserRouter>);

    expect(screen.getByText('Configured')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Talk to us' })).toHaveAttribute('href', '/contact');
    expect(screen.getByRole('link', { name: 'Install now' })).toHaveAttribute('href', '#downloads-section');
    expect(screen.getByRole('link', { name: 'Support' })).toHaveAttribute('href', '#faq-2');
    expect(screen.getByRole('link', { name: 'Learn' })).toHaveAttribute('href', '/learn');
    expect(screen.getByText('Products')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Install' })).toHaveAttribute('href', '#downloads-section');
    expect(screen.queryByRole('link', { name: 'FAQ' })).not.toBeInTheDocument();
    expect(screen.getByText('Products · FAQ')).toBeInTheDocument();
    expect(screen.queryByText('Products · Install')).not.toBeInTheDocument();
    expect(screen.getByText('Products · Fallback')).toBeInTheDocument();
    expect(screen.getAllByText('Get the app')).toHaveLength(2);
    expect(screen.queryByText('Hidden feature')).not.toBeInTheDocument();
    expect(screen.queryByText('Not rendered')).not.toBeInTheDocument();
    expect(screen.getByText('Download Vrooli Ascension')).toBeInTheDocument();
    expect(warn).toHaveBeenCalledWith('Unknown section type: unsupported');
    warn.mockRestore();
  });
});
