import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { HeroSection } from './HeroSection';

const mockTrackCTAClick = vi.fn();
vi.mock('../../../shared/hooks/useMetrics', () => ({
  useMetrics: () => ({ trackCTAClick: mockTrackCTAClick }),
}));

const originalLocation = window.location;
beforeEach(() => {
  vi.clearAllMocks();
  Object.defineProperty(window, 'location', { configurable: true, writable: true, value: { href: '' } });
});
afterEach(() => {
  Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
});

describe('HeroSection', () => {
  it('renders title, subtitle, and a product image when provided', () => {
    render(
      <HeroSection
        content={{ title: 'Launch faster', subtitle: 'The best way', cta_text: 'Start', cta_url: '/signup', image_url: 'https://cdn/hero.png' }}
      />,
    );
    expect(screen.getByText('Launch faster')).toBeInTheDocument();
    expect(screen.getByRole('img', { name: /product preview/i })).toBeInTheDocument();
  });

  it('tracks and navigates on CTA click when a url is set', () => {
    render(<HeroSection content={{ title: 'Go', cta_text: 'Start', cta_url: 'https://app/signup' }} />);
    fireEvent.click(screen.getByText('Start'));
    expect(mockTrackCTAClick).toHaveBeenCalledWith('hero-cta', expect.objectContaining({ cta_url: 'https://app/signup' }));
    expect(window.location.href).toBe('https://app/signup');
  });

  it('does not navigate when the CTA has no url and renders without an image', () => {
    render(<HeroSection content={{ title: 'No CTA', cta_text: 'Start' }} />);
    fireEvent.click(screen.getByText('Start'));
    expect(mockTrackCTAClick).not.toHaveBeenCalled();
    expect(screen.queryByRole('img', { name: /product preview/i })).not.toBeInTheDocument();
  });
});
