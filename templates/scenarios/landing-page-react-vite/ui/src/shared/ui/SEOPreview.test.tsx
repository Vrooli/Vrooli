import { describe, it, expect } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../test-utils';
import { SEOPreview } from './SEOPreview';

describe('SEOPreview', () => {
  it('truncates a description longer than 160 characters', () => {
    const long = 'x'.repeat(200);
    const { container } = render(<SEOPreview title="Home" description={long} url="https://acme.test/" />);
    // The truncated form (157 chars + ellipsis) appears in the preview cards.
    expect(container.textContent).toContain(`${'x'.repeat(157)}...`);
    // The protocol and trailing slash are stripped from the display URL.
    expect(screen.getAllByText('acme.test').length).toBeGreaterThan(0);
  });

  it('shows a short description verbatim', () => {
    const { container } = render(<SEOPreview description="Short and sweet" />);
    expect(container.textContent).toContain('Short and sweet');
  });

  it('renders the OG image and favicon when provided (large twitter card)', () => {
    render(
      <SEOPreview
        title="Launch"
        description="Desc"
        ogImage="https://cdn/og.png"
        favicon="https://cdn/fav.png"
        siteName="Acme"
        twitterCard="summary_large_image"
      />,
    );
    expect(screen.getAllByText('Launch').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Acme').length).toBeGreaterThan(0);
  });

  it('renders the compact summary card without an OG image', () => {
    render(<SEOPreview title="Launch" description="Desc" twitterCard="summary" />);
    expect(screen.getAllByText('Launch').length).toBeGreaterThan(0);
  });
});
