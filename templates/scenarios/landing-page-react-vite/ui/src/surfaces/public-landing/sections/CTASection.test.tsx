import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, fireEvent } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { CTASection } from './CTASection';

const mockTrackCTAClick = vi.fn();
vi.mock('../../../shared/hooks/useMetrics', () => ({
  useMetrics: () => ({ trackCTAClick: mockTrackCTAClick }),
}));

const originalLocation = window.location;

beforeEach(() => {
  vi.clearAllMocks();
  Object.defineProperty(window, 'location', {
    configurable: true,
    writable: true,
    value: { href: '' },
  });
});

afterEach(() => {
  Object.defineProperty(window, 'location', { configurable: true, value: originalLocation });
});

describe('CTASection', () => {
  it('renders default copy and hides the button without cta_text', () => {
    render(<CTASection content={{}} />);
    expect(screen.getByText('Book a live review of your variant stack')).toBeInTheDocument();
    expect(screen.queryByTestId('cta-button')).not.toBeInTheDocument();
  });

  it('renders the CTA button when cta_text is provided', () => {
    render(<CTASection content={{ title: 'Go', cta_text: 'Get started' }} />);
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByTestId('cta-button')).toHaveTextContent('Get started');
  });

  it('tracks the click and navigates when a cta_url is set', () => {
    render(<CTASection content={{ cta_text: 'Book now', cta_url: 'https://cal.com/demo' }} />);
    fireEvent.click(screen.getByTestId('cta-button'));
    expect(mockTrackCTAClick).toHaveBeenCalledWith('cta-section', {
      cta_text: 'Book now',
      cta_url: 'https://cal.com/demo',
    });
    expect(window.location.href).toBe('https://cal.com/demo');
  });

  it('does not track or navigate when the CTA has no url', () => {
    render(<CTASection content={{ cta_text: 'Book now' }} />);
    fireEvent.click(screen.getByTestId('cta-button'));
    expect(mockTrackCTAClick).not.toHaveBeenCalled();
    expect(window.location.href).toBe('');
  });
});
