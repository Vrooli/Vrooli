import { describe, expect, it, vi } from 'vitest';
import type { ReactNode } from 'react';
import App from './App';
import { renderWithProviders } from './test-utils/renderWithProviders';
import { expectNoA11yViolations } from './test-utils/a11y';

// This test audits the composed shell, not variant-loading behavior (covered by
// LandingVariantProvider tests). A stable provider keeps the accessibility run
// free of unrelated network state transitions.
vi.mock('./app/providers/LandingVariantProvider', () => ({
  LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control' },
    config: { sections: [], downloads: [], fallback: false },
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: null,
    lastUpdated: 0,
    refresh: vi.fn(),
  }),
}));

describe('application shell accessibility', () => {
  it('has no detectable Axe violations on its initial route', async () => {
    const { container } = renderWithProviders(<App />);
    expect(container).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
