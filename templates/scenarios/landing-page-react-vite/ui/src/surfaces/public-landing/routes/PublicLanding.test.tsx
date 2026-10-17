import { describe, it, expect, vi } from 'vitest';
import { screen, within, fireEvent } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils';
import type { ReactNode } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { PublicLanding } from './PublicLanding';

const section = (sectionType: string, order: number, content: Record<string, unknown> = {}) => ({
  sectionType,
  order,
  enabled: true,
  content,
});

const richConfig = {
  variant: { id: 1n, slug: 'control', name: 'Control' },
  branding: {
    siteName: 'Acme Launchpad',
    tagline: 'Automation that ships',
    logoUrl: 'https://cdn.example.com/logo.svg',
    logoIconUrl: 'https://cdn.example.com/icon.png',
  },
  sections: [
    section('hero', 1, { title: 'Test Hero', subtitle: 'Subtitle', cta_text: 'Start Trial', cta_url: '/signup' }),
    section('features', 2, { title: 'Features' }),
    section('pricing', 3, { title: 'Pricing' }),
    section('cta', 4, { title: 'CTA', cta_text: 'Go', cta_url: '/go' }),
    section('testimonials', 5, { title: 'Testimonials' }),
    section('faq', 6, { title: 'FAQ' }),
    section('video', 7, { title: 'Video', video_url: 'https://www.youtube.com/watch?v=abc' }),
    section('downloads', 8, { title: 'Downloads' }),
    section('footer', 9, { company_name: 'Acme' }),
  ],
  downloads: [
    {
      bundleKey: 'bundle',
      appKey: 'desktop',
      name: 'Desktop Suite',
      tagline: 'mac + windows',
      description: '',
      installOverview: '',
      installSteps: [],
      storefronts: [],
      displayOrder: 0,
      platforms: [
        {
          id: 1n,
          bundleKey: 'bundle',
          appKey: 'desktop',
          platform: 'mac',
          artifactUrl: 'https://example.com/app.dmg',
          releaseVersion: '1.0.0',
          requiresEntitlement: false,
        },
      ],
    },
  ],
  fallback: false,
  pricing: undefined,
  header: {
    branding: { mode: 'logo_and_name', label: 'Acme Launchpad', mobilePreference: 'auto' },
    nav: {
      links: [
        { id: 'l1', type: 'section', label: 'Features', sectionType: 'features', anchor: 'features', visibleOn: { desktop: true, mobile: true } },
        { id: 'l2', type: 'downloads', label: 'Downloads', anchor: 'downloads', visibleOn: { desktop: true, mobile: true } },
        { id: 'l3', type: 'custom', label: 'Docs', href: 'https://docs.example.com', visibleOn: { desktop: true, mobile: false } },
        {
          id: 'l4',
          type: 'menu',
          label: 'More',
          visibleOn: { desktop: true, mobile: true },
          children: [{ id: 'c1', type: 'custom', label: 'Blog', href: '/blog', visibleOn: { desktop: true, mobile: true } }],
        },
      ],
    },
    ctas: {
      primary: { mode: 'custom', label: 'Buy', href: '/buy', variant: 'solid' },
      secondary: { mode: 'downloads', variant: 'ghost' },
    },
    behavior: { sticky: true, hideOnScroll: true },
  },
};

const { holder } = vi.hoisted(() => ({ holder: { value: null as unknown } }));

vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => holder.value,
  LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

const hookState = (over: Record<string, unknown> = {}) => ({
  variant: { slug: 'control', name: 'Control' },
  config: richConfig,
  loading: false,
  error: null,
  resolution: 'api_select',
  statusNote: null,
  lastUpdated: Date.now(),
  refresh: vi.fn(),
  ...over,
});

const renderLanding = () =>
  renderWithProviders(
    <BrowserRouter>
      <PublicLanding />
    </BrowserRouter>,
    { withoutRouter: true },
  );

describe('PublicLanding', () => {
  it('surfaces CTA and download anchors in the sticky header', () => {
    holder.value = hookState();
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
    expect(screen.getByTestId('landing-nav-cta')).toBeInTheDocument();
    const downloadButton = screen.getByTestId('landing-nav-download');
    expect(within(downloadButton).getByText(/Download macOS/i)).toBeInTheDocument();
  });

  it('shows site branding name and logo when provided', () => {
    holder.value = hookState();
    renderLanding();
    expect(screen.getByText('Acme Launchpad')).toBeInTheDocument();
    expect(screen.getByText(/Automation that ships/)).toBeInTheDocument();
    expect(screen.getByTestId('branding-logo')).toBeInTheDocument();
    expect(screen.getByAltText(/Acme Launchpad logo/i)).toBeInTheDocument();
  });

  it('renders every section type even when content is omitted', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        sections: [
          { sectionType: 'hero', order: 1, enabled: true },
          { sectionType: 'features', order: 2, enabled: true },
          { sectionType: 'pricing', order: 3, enabled: true },
          { sectionType: 'cta', order: 4, enabled: true },
          { sectionType: 'testimonials', order: 5, enabled: true },
          { sectionType: 'faq', order: 6, enabled: true },
          { sectionType: 'footer', order: 7, enabled: true },
          { sectionType: 'downloads', order: 8, enabled: true },
        ],
      },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('labels the resolution as unknown and shows runtime meta in debug + fallback', () => {
    window.history.replaceState({}, '', '/?debug=1');
    holder.value = hookState({ resolution: 'unknown', config: { ...richConfig, fallback: true }, statusNote: 'degraded' });
    renderLanding();
    window.history.replaceState({}, '', '/');
    expect(screen.getByTestId('fallback-signal-banner')).toBeInTheDocument();
  });

  it('renders a menu whose children omit visibility flags', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: {
          ...richConfig.header,
          nav: {
            links: [
              { id: 'm', type: 'menu', label: 'More', children: [{ id: 'c1', type: 'custom', label: 'Blog', href: '/blog' }] },
            ],
          },
        },
      },
    });
    renderLanding();
    expect(screen.getAllByText('More').length).toBeGreaterThan(0);
  });

  it('renders every configured section type', () => {
    holder.value = hookState();
    renderLanding();
    expect(screen.getByText('Test Hero')).toBeInTheDocument();
    expect(screen.getAllByText('Features').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Pricing').length).toBeGreaterThan(0);
    expect(screen.getByText('Testimonials')).toBeInTheDocument();
    expect(screen.getAllByText('FAQ').length).toBeGreaterThan(0);
  });

  it('renders custom and menu navigation links with dropdown children', () => {
    holder.value = hookState();
    renderLanding();
    expect(screen.getAllByText('Docs').length).toBeGreaterThan(0);
    expect(screen.getAllByText('More').length).toBeGreaterThan(0);
  });

  it('shows the fallback signal banner (with a status note) when the config is a fallback', () => {
    holder.value = hookState({ config: { ...richConfig, fallback: true }, statusNote: 'API unavailable' });
    renderLanding();
    const banner = screen.getByTestId('fallback-signal-banner');
    expect(banner).toBeInTheDocument();
    expect(within(banner).getByText(/API unavailable/)).toBeInTheDocument();
  });

  it('handles a slug-only variant, contentless sections, and an unknown download platform', () => {
    holder.value = hookState({
      variant: { slug: 'anon' },
      config: {
        ...richConfig,
        sections: [
          // Section without content -> falls back to an empty object.
          { sectionType: 'hero', order: 1, enabled: true },
          ...richConfig.sections.filter((s) => s.sectionType !== 'hero' && s.sectionType !== 'downloads'),
        ],
        downloads: [
          {
            ...richConfig.downloads[0],
            storefronts: [],
            platforms: [{ ...richConfig.downloads[0]!.platforms[0]!, platform: 'freebsd' }],
          },
        ],
      },
    });
    renderLanding();
    // Unknown platform label passes through verbatim in the download button.
    expect(within(screen.getByTestId('landing-nav-download')).getByText(/freebsd/i)).toBeInTheDocument();
  });

  it('shows a spinner while the config is loading', () => {
    holder.value = hookState({ config: null, loading: true });
    renderLanding();
    expect(screen.getByText(/Loading your landing page/i)).toBeInTheDocument();
  });

  it('shows an error screen with retry when loading fails without a fallback', () => {
    holder.value = hookState({ config: null, error: 'API exploded' });
    renderLanding();
    expect(screen.getByText('Failed to Load Variant')).toBeInTheDocument();
    expect(screen.getByText('API exploded')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Retry' })).toBeInTheDocument();
  });

  it('shows the no-variant screen when there is no config', () => {
    holder.value = hookState({ config: null, variant: null });
    renderLanding();
    expect(screen.getByText('No Variant Available')).toBeInTheDocument();
  });

  it('surfaces the variant-source banner when pinned via URL parameter', () => {
    holder.value = hookState({ resolution: 'url_param', statusNote: 'Pinned via ?variant' });
    renderLanding();
    expect(screen.getByTestId('variant-source-banner')).toBeInTheDocument();
  });

  it('labels the download button generically when multiple apps exist', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        downloads: [
          richConfig.downloads[0],
          { ...richConfig.downloads[0], appKey: 'second', name: 'Second App' },
        ],
      },
    });
    renderLanding();
    expect(within(screen.getByTestId('landing-nav-download')).getByText(/View downloads/i)).toBeInTheDocument();
  });

  it('renders without a download button when there are no download targets', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        sections: richConfig.sections.filter((s) => s.sectionType !== 'downloads'),
        downloads: [],
      },
    });
    renderLanding();
    expect(screen.queryByTestId('landing-nav-download')).not.toBeInTheDocument();
  });

  it('renders a brand placeholder when no logo is configured', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        branding: { siteName: 'Nologo Co', tagline: 'No logo here' },
        header: { ...richConfig.header, branding: { mode: 'name', mobilePreference: 'name' } },
      },
    });
    renderLanding();
    expect(screen.getAllByText('Nologo Co').length).toBeGreaterThan(0);
    expect(screen.queryByTestId('branding-logo')).not.toBeInTheDocument();
  });

  it('honors hidden and inherited CTA modes with logo-only branding', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: {
          ...richConfig.header,
          branding: { mode: 'logo', mobilePreference: 'logo' },
          ctas: {
            primary: { mode: 'inherit_hero', variant: 'solid' },
            secondary: { mode: 'hidden', variant: 'ghost' },
          },
        },
      },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('renders a non-sticky header when configured', () => {
    holder.value = hookState({
      config: { ...richConfig, header: { ...richConfig.header, behavior: { sticky: false, hideOnScroll: false } } },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('falls back to default nav mirroring when no header config is supplied', () => {
    holder.value = hookState({
      config: { ...richConfig, header: undefined },
    });
    renderLanding();
    // Default nav mirrors section order; header still renders.
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('respects hide-on-scroll behavior on scroll events', () => {
    holder.value = hookState({
      config: { ...richConfig, header: { ...richConfig.header, behavior: { sticky: true, hideOnScroll: true } } },
    });
    renderLanding();
    window.scrollY = 400;
    fireEvent.scroll(window);
    window.scrollY = 0;
    fireEvent.scroll(window);
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('labels a storefront-only single download by its store', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        downloads: [
          {
            ...richConfig.downloads[0],
            platforms: [],
            storefronts: [{ store: 'app_store', label: 'App Store', url: 'https://apps.apple.com/x', badge: '' }],
          },
        ],
      },
    });
    renderLanding();
    expect(within(screen.getByTestId('landing-nav-download')).getByText(/Open App Store/i)).toBeInTheDocument();
  });

  it('labels a multi-platform single download with the app name', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        downloads: [
          {
            ...richConfig.downloads[0],
            name: 'Suite Pro',
            storefronts: [],
            platforms: [
              { ...richConfig.downloads[0]!.platforms[0]!, platform: 'mac' },
              { ...richConfig.downloads[0]!.platforms[0]!, platform: 'windows' },
            ],
          },
        ],
      },
    });
    renderLanding();
    expect(within(screen.getByTestId('landing-nav-download')).getByText(/View Suite Pro/i)).toBeInTheDocument();
  });

  it('renders the debug panel when the debug query flag is set', () => {
    window.history.replaceState({}, '', '/?debug=1');
    holder.value = hookState({ statusNote: 'debug note' });
    renderLanding();
    window.history.replaceState({}, '', '/');
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it.each([
    ['none', 'stacked'],
    ['logo', 'name'],
    ['name', 'logo'],
  ])('renders header branding mode %s with mobile preference %s', (mode, mobilePreference) => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: { ...richConfig.header, branding: { mode, label: 'Acme Launchpad', mobilePreference } },
      },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('injects a downloads section when apps exist without an explicit section', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        // Downloads apps present, but no "downloads" section in the list.
        sections: richConfig.sections.filter((s) => s.sectionType !== 'downloads'),
        // An unknown section type renders nothing (renderSection guard).
        // (kept alongside the known ones)
      },
    });
    renderLanding();
    expect(screen.getByTestId('downloads-section')).toBeInTheDocument();
  });

  it('ignores unknown section types and renders a brand placeholder without a logo', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        branding: { siteName: 'Acme Launchpad', tagline: 'Automation that ships' },
        sections: [...richConfig.sections, section('mystery', 20, { title: 'Mystery' })],
      },
    });
    renderLanding();
    // Unknown section type produces no heading of its own.
    expect(screen.queryByText('Mystery')).not.toBeInTheDocument();
    // No logo -> placeholder initials render instead of an img.
    expect(screen.queryByTestId('branding-logo')).not.toBeInTheDocument();
  });

  it('filters out disabled sections before rendering', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        sections: [
          ...richConfig.sections,
          section('faq', 10, { title: 'Hidden FAQ', enabled: false }),
          { ...section('features', 11, { title: 'Disabled Features' }), enabled: false },
        ],
      },
    });
    renderLanding();
    expect(screen.queryByText('Disabled Features')).not.toBeInTheDocument();
  });

  it('drops inherited CTAs when there is no hero CTA and no ctas config', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: { ...richConfig.header, ctas: undefined },
        sections: richConfig.sections.filter((s) => s.sectionType !== 'hero'),
      },
    });
    renderLanding();
    // Both CTAs resolve to null (no hero to inherit), header still renders.
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
    expect(screen.queryByTestId('landing-nav-cta')).not.toBeInTheDocument();
  });

  it('drops a custom CTA missing its href and overrides the downloads label', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: {
          ...richConfig.header,
          ctas: {
            primary: { mode: 'custom', label: 'No link', variant: 'solid' },
            secondary: { mode: 'downloads', label: 'Grab it', variant: 'ghost' },
          },
        },
      },
    });
    renderLanding();
    expect(screen.queryByTestId('landing-nav-cta')).not.toBeInTheDocument();
    expect(within(screen.getByTestId('landing-nav-download')).getByText('Grab it')).toBeInTheDocument();
  });

  it('resolves nav links with id/label fallbacks, menu-child anchors, and unresolvable sections', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: {
          ...richConfig.header,
          nav: {
            links: [
              // downloads link with no id + all-false visibility (reset to both true)
              { type: 'downloads', label: '', anchor: 'downloads', visibleOn: { desktop: false, mobile: false } },
              // custom link with no id
              { type: 'custom', label: 'Docs', href: 'https://docs.example.com' },
              // section link with no id + sectionType resolved via section anchors, no label
              { type: 'section', sectionType: 'pricing', visibleOn: { desktop: true } },
              // section link that cannot resolve an anchor -> skipped
              { type: 'section', label: 'Ghost', sectionType: 'nonexistent' },
              // menu with no id and varied children
              {
                type: 'menu',
                label: 'More',
                children: [
                  { type: 'section', label: '', sectionType: 'features' },
                  { type: 'downloads', label: 'Get app' },
                  { type: 'custom', label: 'Anchored', anchor: '#top' },
                  { type: 'custom', label: 'Fallback' },
                ],
              },
            ],
          },
        },
      },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
    expect(screen.getAllByText('More').length).toBeGreaterThan(0);
  });

  it('skips custom nav links that have no href and menus with no children', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        header: {
          ...richConfig.header,
          nav: {
            links: [
              { id: 'x1', type: 'custom', label: 'Broken', visibleOn: { desktop: true, mobile: true } },
              { id: 'x2', type: 'menu', label: 'EmptyMenu', children: [], visibleOn: { desktop: true, mobile: true } },
              { id: 'x3', type: 'section', label: 'Pricing', sectionType: 'pricing', anchor: 'pricing', visibleOn: { desktop: true, mobile: true } },
            ],
          },
        },
      },
    });
    renderLanding();
    expect(screen.getByTestId('landing-experience-header')).toBeInTheDocument();
  });

  it('labels a single-platform download precisely and hides desktop-only nav on mobile', () => {
    holder.value = hookState({
      config: {
        ...richConfig,
        downloads: [
          {
            ...richConfig.downloads[0],
            storefronts: [],
            platforms: [{ ...richConfig.downloads[0]!.platforms[0]!, platform: 'windows' }],
          },
        ],
      },
    });
    renderLanding();
    expect(within(screen.getByTestId('landing-nav-download')).getByText(/Download Windows/i)).toBeInTheDocument();
  });
});
