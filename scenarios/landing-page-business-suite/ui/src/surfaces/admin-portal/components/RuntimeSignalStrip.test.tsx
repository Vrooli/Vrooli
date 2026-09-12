import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { screen } from "@testing-library/react";
import userEvent from '@testing-library/user-event';
import { RuntimeSignalStrip } from './RuntimeSignalStrip';
import type { useLandingVariant } from '../../../app/providers/useLandingVariant';
import type { LandingConfigResponse } from '../../../shared/api';

const mockUseLandingVariant = vi.fn<() => ReturnType<typeof useLandingVariant>>();

vi.mock('../../../app/providers/useLandingVariant', () => ({
  useLandingVariant: () => mockUseLandingVariant(),
}));

const baseConfig: LandingConfigResponse = {
  variant: { id: 1, slug: 'control', name: 'Control' },
  sections: [],
  downloads: [],
  header: {
    branding: { mode: 'logo' },
    nav: { links: [] },
    ctas: { primary: { mode: 'inherit_hero' }, secondary: { mode: 'hidden' } },
    behavior: { sticky: true, hide_on_scroll: false },
  },
  fallback: false,
};

const buildContext = (
  overrides: Partial<ReturnType<typeof useLandingVariant>> = {}
): ReturnType<typeof useLandingVariant> => ({
  variant: { id: 1, slug: 'control', name: 'Control' },
  config: baseConfig,
  loading: false,
  error: null,
  resolution: 'api_select',
  statusNote: 'Variant selected via weighted API',
  lastUpdated: Date.now(),
  refresh: vi.fn<() => Promise<void>>(),
  ...overrides,
});

describe('RuntimeSignalStrip [SIGNAL]', () => {
  beforeEach(() => {
    mockUseLandingVariant.mockReturnValue(buildContext());
  });

  describe('full mode (default)', () => {
    it('renders live config status and variant info', () => {
      render(<RuntimeSignalStrip />);

      expect(screen.getByTestId('runtime-signal-strip')).toBeInTheDocument();
      expect(screen.getByText(/Control/)).toBeInTheDocument();
      expect(screen.getByText(/Weighted API selection/)).toBeInTheDocument();
    });

    it('surfaces fallback state and note', () => {
      mockUseLandingVariant.mockReturnValue(
        buildContext({
          config: { ...baseConfig, fallback: true },
          statusNote: 'API unavailable: timeout',
          resolution: 'fallback',
        })
      );

      render(<RuntimeSignalStrip />);

      expect(screen.getByText(/Fallback copy active/)).toBeInTheDocument();
      expect(screen.getByText(/API unavailable: timeout/)).toBeInTheDocument();
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

  describe('compact mode', () => {
    it('renders collapsed badge by default', () => {
      render(<RuntimeSignalStrip mode="compact" />);

      expect(screen.getByTestId('runtime-signal-compact')).toBeInTheDocument();
      expect(screen.getByTestId('runtime-signal-toggle')).toBeInTheDocument();
      expect(screen.queryByTestId('runtime-signal-expanded')).not.toBeInTheDocument();
    });

    it('expands when toggle is clicked', async () => {
      const user = userEvent.setup();
      render(<RuntimeSignalStrip mode="compact" />);

      await user.click(screen.getByTestId('runtime-signal-toggle'));

      expect(screen.getByTestId('runtime-signal-expanded')).toBeInTheDocument();
      expect(screen.getByText(/Weighted API selection/)).toBeInTheDocument();
    });

    it('collapses when toggle is clicked again', async () => {
      const user = userEvent.setup();
      render(<RuntimeSignalStrip mode="compact" />);

      await user.click(screen.getByTestId('runtime-signal-toggle'));
      expect(screen.getByTestId('runtime-signal-expanded')).toBeInTheDocument();

      await user.click(screen.getByTestId('runtime-signal-toggle'));
      expect(screen.queryByTestId('runtime-signal-expanded')).not.toBeInTheDocument();
    });

    it('shows variant name in collapsed state', () => {
      render(<RuntimeSignalStrip mode="compact" />);

      expect(screen.getByText('Control')).toBeInTheDocument();
    });

    it('shows refresh button in expanded state', async () => {
      const refresh = vi.fn().mockResolvedValue(undefined);
      mockUseLandingVariant.mockReturnValue(buildContext({ refresh }));

      const user = userEvent.setup();
      render(<RuntimeSignalStrip mode="compact" />);

      await user.click(screen.getByTestId('runtime-signal-toggle'));
      await user.click(screen.getByTestId('runtime-refresh'));

      expect(refresh).toHaveBeenCalled();
    });
  });

  describe('error state', () => {
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

    it('shows error card in compact mode too', () => {
      mockUseLandingVariant.mockReturnValue(
        buildContext({
          error: 'Network failure',
        })
      );

      render(<RuntimeSignalStrip mode="compact" />);

      expect(screen.getByTestId('runtime-signal-error')).toBeInTheDocument();
    });
  });
});
