import { describe, it, expect, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import type { ReactNode } from 'react';
import { BrowserRouter } from 'react-router-dom';
import { PublicLanding } from './PublicLanding';

vi.mock('../../../app/providers/LandingVariantProvider', () => {
  const mockConfig = {
    variant: { id: 1, slug: 'control', name: 'Control' },
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
      branding: { mode: 'logo_and_name', label: 'Control', mobile_preference: 'auto' },
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

describe('PublicLanding header rails', () => {
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
});
