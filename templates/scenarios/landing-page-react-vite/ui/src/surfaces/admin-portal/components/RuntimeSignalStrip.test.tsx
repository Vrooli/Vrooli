import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RuntimeSignalStrip } from './RuntimeSignalStrip';

const mockUseLandingVariant = vi.fn();

vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => mockUseLandingVariant(),
}));

const buildContext = (overrides?: Record<string, unknown>) => ({
  variant: { slug: 'control', name: 'Control' },
  config: { fallback: false },
  loading: false,
  error: null,
  resolution: 'api_select',
  statusNote: 'Variant selected via weighted API',
  lastUpdated: Date.now(),
  refresh: vi.fn(),
  ...overrides,
});

describe('RuntimeSignalStrip [SIGNAL]', () => {
  beforeEach(() => {
    mockUseLandingVariant.mockReturnValue(buildContext());
  });

  it('renders live config status and variant info', () => {
    render(<RuntimeSignalStrip />);

    expect(screen.getByTestId('runtime-signal-strip')).toBeInTheDocument();
    expect(screen.getByText(/Control/)).toBeInTheDocument();
    expect(screen.getByText(/Weighted API selection/)).toBeInTheDocument();
  });

  it('surfaces fallback state and note', () => {
    mockUseLandingVariant.mockReturnValue(
      buildContext({
        config: { fallback: true },
        statusNote: 'API unavailable: timeout',
        resolution: 'fallback',
      })
    );

    render(<RuntimeSignalStrip />);

    expect(screen.getByText(/Fallback copy active/)).toBeInTheDocument();
    expect(screen.getByText(/API unavailable: timeout/)).toBeInTheDocument();
  });

  it('shows error card when landing config cannot load', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    mockUseLandingVariant.mockReturnValue(
      buildContext({
        error: 'Network failure',
        refresh,
      })
    );

    const user = userEvent.setup();
    render(<RuntimeSignalStrip />);

    expect(screen.getByTestId('runtime-signal-error')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /retry sync/i }));
    expect(refresh).toHaveBeenCalled();
  });

  it('invokes refresh when action button clicked', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    mockUseLandingVariant.mockReturnValue(buildContext({ refresh }));

    const user = userEvent.setup();
    render(<RuntimeSignalStrip />);

    await user.click(screen.getByTestId('runtime-refresh'));
    expect(refresh).toHaveBeenCalled();
  });
});
