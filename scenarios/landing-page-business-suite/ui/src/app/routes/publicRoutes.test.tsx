import { describe, expect, it, vi } from 'vitest';
import { renderWithProviders as render } from '../../test-utils/renderWithProviders';
import { screen } from '@testing-library/react';
import { PublicRouteGuard } from './publicRoutes';
import type { useLandingVariant } from '../providers/useLandingVariant';
import type { LandingVariantContextType } from '../providers/LandingVariantContext';
import type { LandingConfigResponse } from '../../shared/api';

const useLandingVariantMock = vi.fn<() => ReturnType<typeof useLandingVariant>>();
const createLandingVariantContext = (
  overrides: Partial<LandingVariantContextType>
): LandingVariantContextType => ({
  variant: null,
  config: null,
  loading: false,
  error: null,
  resolution: 'unknown',
  statusNote: null,
  lastUpdated: null,
  refresh: vi.fn(),
  ...overrides,
});

vi.mock('../providers/useLandingVariant', () => ({
  useLandingVariant: () => useLandingVariantMock(),
}));

describe('PublicRouteGuard experience readiness', () => {
  it('exposes the loading state until landing configuration resolves', () => {
    useLandingVariantMock.mockReturnValue(createLandingVariantContext({ config: null, loading: true }));

    render(<PublicRouteGuard><div>landing content</div></PublicRouteGuard>);

    expect(screen.getByLabelText('Preparing Silent Founder OS')).toHaveAttribute('data-experience-surface', 'public-landing');
    expect(screen.getByLabelText('Preparing Silent Founder OS')).toHaveAttribute('data-experience-state', 'loading');
    expect(screen.getByLabelText('Preparing Silent Founder OS')).toHaveAttribute('data-testid', 'landing-experience-surface');
  });

  it('exposes a terminal ready state once configuration resolves', () => {
    useLandingVariantMock.mockReturnValue(createLandingVariantContext({
      config: { branding: { coming_soon_enabled: false } } as LandingConfigResponse,
      loading: false,
    }));

    render(<PublicRouteGuard><div>landing content</div></PublicRouteGuard>);

    expect(screen.getByText('landing content').parentElement).toHaveAttribute('data-experience-surface', 'public-landing');
    expect(screen.getByText('landing content').parentElement).toHaveAttribute('data-experience-state', 'ready');
  });

  it('reports an error state when configuration cannot be resolved', () => {
    useLandingVariantMock.mockReturnValue(createLandingVariantContext({ config: null, loading: false }));

    render(<PublicRouteGuard><div>landing content</div></PublicRouteGuard>);

    expect(screen.getByText('landing content').parentElement).toHaveAttribute('data-experience-state', 'error');
  });
});
