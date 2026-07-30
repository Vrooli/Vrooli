import { act, fireEvent, screen, within } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { HeroSection } from './HeroSection';

const trackCTAClick = vi.fn();
vi.mock('../../../shared/hooks/useMetricsHook', () => ({ useMetrics: () => ({ trackCTAClick }) }));

describe('HeroSection', () => {
  afterEach(() => { vi.clearAllMocks(); vi.useRealTimers(); });

  it('uses configured CTA content, records engagement, and renders the linked secondary action', () => {
    render(<HeroSection content={{ title: 'Build faster', subtitle: 'Ship safely', cta_text: 'Start now', cta_url: '#plans', secondary_cta_text: 'See demo', secondary_cta_url: '#demo' }} />);
    expect(screen.getByRole('heading', { name: 'Build faster' })).toBeInTheDocument();
    expect(screen.getByText('Ship safely')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'See demo' })).toHaveAttribute('href', '#demo');
    fireEvent.click(screen.getByTestId('hero-cta'));
    expect(trackCTAClick).toHaveBeenCalledWith('hero-cta', { cta_text: 'Start now', cta_url: '#plans' });
  });

  it('supports manual preview navigation, pause behavior, and animated preview progress', () => {
    vi.useFakeTimers();
    render(<HeroSection content={{}} />);
    expect(screen.getByText('record-session')).toBeInTheDocument();
    const showcase = screen.getByLabelText('Go to preview 1').parentElement?.parentElement;
    expect(showcase).not.toBeNull();
    if (!showcase) throw new Error('preview showcase is missing');
    fireEvent.mouseEnter(showcase);
    expect(screen.getByText('Paused')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Go to preview 2'));
    act(() => { vi.advanceTimersByTime(250); });
    expect(screen.getByText('workflow-builder.tsx')).toBeInTheDocument();
    fireEvent.mouseLeave(showcase);
    act(() => { vi.advanceTimersByTime(300 + 400 * 5); });
    expect(screen.getByText('Workflow executing...')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Go to preview 3'));
    act(() => { vi.advanceTimersByTime(250 + 400 + 600); });
    expect(screen.getByText('execution-monitor')).toBeInTheDocument();
    expect(screen.getByText('Running workflow...')).toBeInTheDocument();
  });

  it('uses safe default conversion copy and resets an inactive preview before returning to it', () => {
    vi.useFakeTimers();
    render(<HeroSection content={{}} />);

    expect(screen.getByRole('heading', { name: 'Record once. Automate forever' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Watch video' })).toHaveAttribute('href', '#video');
    fireEvent.click(screen.getByTestId('hero-cta'));
    expect(trackCTAClick).toHaveBeenCalledWith('hero-cta', { cta_text: 'Start free', cta_url: '#pricing' });

    const showcase = screen.getByLabelText('Go to preview 1').parentElement?.parentElement;
    expect(showcase).not.toBeNull();
    if (!showcase) throw new Error('preview showcase is missing');
    act(() => { vi.advanceTimersByTime(600 + 800); });
    expect(screen.getByText('amazon.com')).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText('Go to preview 2'));
    act(() => { vi.advanceTimersByTime(250); });
    fireEvent.click(screen.getByLabelText('Go to preview 1'));
    act(() => { vi.advanceTimersByTime(250); });
    expect(within(showcase).getByText('0:00')).toBeInTheDocument();
    expect(screen.getByText('Recording in progress...')).toBeInTheDocument();
  });

  it('shows the complete recording timeline before the automatic showcase rotation begins', () => {
    vi.useFakeTimers();
    render(<HeroSection content={{}} />);

    act(() => { vi.advanceTimersByTime(600 + 800 * 5); });

    expect(screen.getByText('amazon.com')).toBeInTheDocument();
    expect(screen.getByText('Search button')).toBeInTheDocument();
    expect(screen.getByText('First result')).toBeInTheDocument();
    expect(screen.getAllByText('click')).toHaveLength(3);
  });

});
