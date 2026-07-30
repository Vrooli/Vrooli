import { fireEvent, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { CTASection } from './CTASection';

const trackCTAClick = vi.fn();
vi.mock('../../../shared/hooks/useMetricsHook', () => ({ useMetrics: () => ({ trackCTAClick }) }));

describe('CTASection', () => {
  afterEach(() => { vi.clearAllMocks(); });

  it('uses configured conversion copy and records a configured destination', () => {
    const location = window.location;
    Object.defineProperty(window, 'location', { configurable: true, value: { href: '' } });

    render(<CTASection content={{ title: 'Ready to ship', subtitle: 'A safe next step', cta_text: 'Open plans', cta_url: '#pricing' }} />);
    expect(screen.getByRole('heading', { name: 'Ready to ship' })).toBeInTheDocument();
    expect(screen.getByText('A safe next step')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('cta-button'));
    expect(trackCTAClick).toHaveBeenCalledWith('cta-section', { cta_text: 'Open plans', cta_url: '#pricing' });
    expect(window.location.href).toBe('#pricing');

    Object.defineProperty(window, 'location', { configurable: true, value: location });
  });

  it('uses safe defaults and does not emit conversion analytics without a destination', () => {
    render(<CTASection content={{}} />);
    expect(screen.getByRole('heading', { name: 'See your ops and marketing run themselves' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Start with Vrooli Ascension' })).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('cta-button'));
    expect(trackCTAClick).not.toHaveBeenCalled();
  });
});
