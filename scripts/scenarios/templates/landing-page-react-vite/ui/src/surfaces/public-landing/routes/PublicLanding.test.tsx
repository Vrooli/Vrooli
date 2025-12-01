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
        id: 1,
        label: 'macOS',
        platform: 'mac',
        requires_entitlement: false,
      },
    ],
    fallback: false,
    pricing: null,
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
